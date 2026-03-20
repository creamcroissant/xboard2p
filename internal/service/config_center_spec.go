package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/creamcroissant/xboard/internal/repository"
)

// InboundSpecService manages inbound spec lifecycle and conflict checks.
type InboundSpecService interface {
	UpsertSpec(ctx context.Context, req UpsertInboundSpecRequest) (specID int64, revision int64, err error)
	ListSpecs(ctx context.Context, filter ListInboundSpecFilter) ([]*repository.InboundSpec, int64, error)
	GetSpecHistory(ctx context.Context, specID int64, limit, offset int) ([]*repository.InboundSpecRevision, error)
	ImportFromApplied(ctx context.Context, req ImportInboundSpecRequest) (createdCount int64, err error)
}

// UpsertInboundSpecRequest carries create/update fields for one inbound spec.
type UpsertInboundSpecRequest struct {
	SpecID       int64
	AgentHostID  int64
	CoreType     string
	Tag          string
	Enabled      *bool
	SemanticSpec json.RawMessage
	CoreSpecific json.RawMessage
	OperatorID   int64
	ChangeNote   string
}

// ListInboundSpecFilter constrains list query.
type ListInboundSpecFilter struct {
	AgentHostID *int64
	CoreType    *string
	Tag         *string
	Enabled     *bool
	Limit       int
	Offset      int
}

// ImportInboundSpecRequest imports inbound specs from applied semantic index.
type ImportInboundSpecRequest struct {
	AgentHostID       int64
	CoreType          string
	Source            *string
	Filename          *string
	Tag               *string
	Enabled           *bool
	OperatorID        int64
	ChangeNote        string
	OverwriteExisting bool
}

type inboundSpecService struct {
	specs          repository.InboundSpecRepository
	revisions      repository.InboundSpecRevisionRepository
	inboundIndexes repository.InboundIndexRepository
	compiler       ArtifactCompilerService
}

// NewInboundSpecService creates InboundSpecService.
func NewInboundSpecService(
	specs repository.InboundSpecRepository,
	revisions repository.InboundSpecRevisionRepository,
	inboundIndexes repository.InboundIndexRepository,
	compilers ...ArtifactCompilerService,
) InboundSpecService {
	var compiler ArtifactCompilerService
	if len(compilers) > 0 {
		compiler = compilers[0]
	}
	return &inboundSpecService{
		specs:          specs,
		revisions:      revisions,
		inboundIndexes: inboundIndexes,
		compiler:       compiler,
	}
}

func (s *inboundSpecService) UpsertSpec(ctx context.Context, req UpsertInboundSpecRequest) (int64, int64, error) {
	if s == nil || s.specs == nil || s.revisions == nil {
		return 0, 0, fmt.Errorf("inbound spec service not configured / 入站配置服务未配置")
	}
	if req.SpecID <= 0 {
		return s.createSpec(ctx, req)
	}
	return s.updateSpec(ctx, req)
}

func (s *inboundSpecService) createSpec(ctx context.Context, req UpsertInboundSpecRequest) (int64, int64, error) {
	if req.AgentHostID <= 0 {
		validationErr := &InboundSpecValidationError{}
		validationErr.add("agent_host_id", "is required / 不能为空")
		return 0, 0, validationErr
	}

	coreType := normalizeCoreType(req.CoreType)
	tag := normalizeTag(req.Tag)
	normalizedSemantic, normalizedCoreSpecific, semantic, err := validateSpecInput(coreType, tag, req.SemanticSpec, req.CoreSpecific)
	if err != nil {
		return 0, 0, err
	}

	if err := s.ensureTagAvailable(ctx, req.AgentHostID, coreType, tag, 0); err != nil {
		return 0, 0, err
	}
	if err := s.ensureListenAvailable(ctx, req.AgentHostID, coreType, semantic.Listen, semantic.Port, 0); err != nil {
		return 0, 0, err
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	spec := &repository.InboundSpec{
		AgentHostID:     req.AgentHostID,
		CoreType:        coreType,
		Tag:             tag,
		Enabled:         enabled,
		SemanticSpec:    normalizedSemantic,
		CoreSpecific:    normalizedCoreSpecific,
		DesiredRevision: 1,
		CreatedBy:       req.OperatorID,
		UpdatedBy:       req.OperatorID,
	}
	if err := s.specs.Create(ctx, spec); err != nil {
		return 0, 0, err
	}

	cleanup := func(baseErr error) error {
		artifactErr := errors.Join(s.deleteDesiredArtifacts(ctx, spec.AgentHostID, spec.CoreType, spec.DesiredRevision))
		specErr := errors.Join(s.specs.Delete(ctx, spec.ID))
		return errors.Join(baseErr, artifactErr, specErr)
	}

	if err := s.renderDesiredArtifacts(ctx, spec.AgentHostID, spec.CoreType, spec.DesiredRevision); err != nil {
		return 0, 0, cleanup(err)
	}

	snapshot, err := buildInboundSpecSnapshot(spec)
	if err != nil {
		return 0, 0, cleanup(err)
	}
	revision := &repository.InboundSpecRevision{
		SpecID:     spec.ID,
		Revision:   spec.DesiredRevision,
		Snapshot:   snapshot,
		ChangeNote: firstNonEmpty(req.ChangeNote, "create"),
		OperatorID: req.OperatorID,
	}
	if err := s.revisions.Create(ctx, revision); err != nil {
		return 0, 0, cleanup(err)
	}
	return spec.ID, spec.DesiredRevision, nil
}

func (s *inboundSpecService) updateSpec(ctx context.Context, req UpsertInboundSpecRequest) (int64, int64, error) {
	existing, err := s.specs.FindByID(ctx, req.SpecID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return 0, 0, ErrNotFound
		}
		return 0, 0, err
	}
	previous := *existing

	agentHostID := existing.AgentHostID
	if req.AgentHostID > 0 {
		agentHostID = req.AgentHostID
	}

	coreType := existing.CoreType
	if strings.TrimSpace(req.CoreType) != "" {
		coreType = req.CoreType
	}

	tag := existing.Tag
	if strings.TrimSpace(req.Tag) != "" {
		tag = req.Tag
	}

	semanticSpec := existing.SemanticSpec
	if len(strings.TrimSpace(string(req.SemanticSpec))) > 0 {
		semanticSpec = req.SemanticSpec
	}
	coreSpecific := existing.CoreSpecific
	if len(strings.TrimSpace(string(req.CoreSpecific))) > 0 {
		coreSpecific = req.CoreSpecific
	}

	enabled := existing.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	coreType = normalizeCoreType(coreType)
	tag = normalizeTag(tag)
	normalizedSemantic, normalizedCoreSpecific, semantic, err := validateSpecInput(coreType, tag, semanticSpec, coreSpecific)
	if err != nil {
		return 0, 0, err
	}

	if err := s.ensureTagAvailable(ctx, agentHostID, coreType, tag, existing.ID); err != nil {
		return 0, 0, err
	}
	if err := s.ensureListenAvailable(ctx, agentHostID, coreType, semantic.Listen, semantic.Port, existing.ID); err != nil {
		return 0, 0, err
	}

	maxRevision, err := s.revisions.GetMaxRevision(ctx, existing.ID)
	if err != nil {
		return 0, 0, err
	}
	nextRevision := existing.DesiredRevision
	if maxRevision > nextRevision {
		nextRevision = maxRevision
	}
	nextRevision++

	existing.AgentHostID = agentHostID
	existing.CoreType = coreType
	existing.Tag = tag
	existing.Enabled = enabled
	existing.SemanticSpec = normalizedSemantic
	existing.CoreSpecific = normalizedCoreSpecific
	existing.DesiredRevision = nextRevision
	existing.UpdatedBy = req.OperatorID

	if err := s.specs.Update(ctx, existing); err != nil {
		return 0, 0, err
	}

	cleanup := func(baseErr error) error {
		rollbackErr := s.specs.Update(ctx, &previous)
		artifactErr := s.deleteDesiredArtifacts(ctx, existing.AgentHostID, existing.CoreType, nextRevision)
		return errors.Join(baseErr, rollbackErr, artifactErr)
	}

	if err := s.renderDesiredArtifacts(ctx, existing.AgentHostID, existing.CoreType, nextRevision); err != nil {
		return 0, 0, cleanup(err)
	}

	snapshot, err := buildInboundSpecSnapshot(existing)
	if err != nil {
		return 0, 0, cleanup(err)
	}
	revision := &repository.InboundSpecRevision{
		SpecID:     existing.ID,
		Revision:   nextRevision,
		Snapshot:   snapshot,
		ChangeNote: firstNonEmpty(req.ChangeNote, "update"),
		OperatorID: req.OperatorID,
	}
	if err := s.revisions.Create(ctx, revision); err != nil {
		return 0, 0, cleanup(err)
	}

	return existing.ID, nextRevision, nil
}

func (s *inboundSpecService) ListSpecs(ctx context.Context, filter ListInboundSpecFilter) ([]*repository.InboundSpec, int64, error) {
	if s == nil || s.specs == nil {
		return nil, 0, fmt.Errorf("inbound spec service not configured / 入站配置服务未配置")
	}

	repoFilter := repository.InboundSpecFilter{
		AgentHostID: filter.AgentHostID,
		Enabled:     filter.Enabled,
		Limit:       filter.Limit,
		Offset:      filter.Offset,
	}
	if filter.CoreType != nil {
		coreType := normalizeCoreType(*filter.CoreType)
		if coreType == "" {
			validationErr := &InboundSpecValidationError{}
			validationErr.add("core_type", "must be sing-box or xray / 必须是 sing-box 或 xray")
			return nil, 0, validationErr
		}
		repoFilter.CoreType = &coreType
	}
	if filter.Tag != nil {
		tag := normalizeTag(*filter.Tag)
		repoFilter.Tag = &tag
	}

	items, err := s.specs.List(ctx, repoFilter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.specs.Count(ctx, repoFilter)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *inboundSpecService) GetSpecHistory(ctx context.Context, specID int64, limit, offset int) ([]*repository.InboundSpecRevision, error) {
	if s == nil || s.specs == nil || s.revisions == nil {
		return nil, fmt.Errorf("inbound spec service not configured / 入站配置服务未配置")
	}
	if specID <= 0 {
		return nil, ErrNotFound
	}
	if _, err := s.specs.FindByID(ctx, specID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.revisions.ListBySpecID(ctx, specID, limit, offset)
}

func (s *inboundSpecService) ImportFromApplied(ctx context.Context, req ImportInboundSpecRequest) (int64, error) {
	if s == nil || s.specs == nil || s.revisions == nil || s.inboundIndexes == nil {
		return 0, fmt.Errorf("inbound spec service not configured / 入站配置服务未配置")
	}
	if req.AgentHostID <= 0 {
		validationErr := &InboundSpecValidationError{}
		validationErr.add("agent_host_id", "is required / 不能为空")
		return 0, validationErr
	}

	coreType := normalizeCoreType(req.CoreType)
	if coreType == "" {
		validationErr := &InboundSpecValidationError{}
		validationErr.add("core_type", "must be sing-box or xray / 必须是 sing-box 或 xray")
		return 0, validationErr
	}

	filter := repository.InboundIndexFilter{
		AgentHostID: &req.AgentHostID,
		CoreType:    &coreType,
		Limit:       200,
		Offset:      0,
	}
	if req.Source != nil {
		source := strings.TrimSpace(*req.Source)
		if source != "" {
			filter.Source = &source
		}
	}
	if req.Filename != nil {
		filename := strings.TrimSpace(*req.Filename)
		if filename != "" {
			filter.Filename = &filename
		}
	}
	if req.Tag != nil {
		tag := normalizeTag(*req.Tag)
		if tag != "" {
			filter.Tag = &tag
		}
	}

	type importCandidate struct {
		semantic json.RawMessage
		listen   string
		port     int
		protocol string
	}

	orderedTags := make([]string, 0)
	candidates := make(map[string]importCandidate)

	for {
		indexes, err := s.inboundIndexes.List(ctx, filter)
		if err != nil {
			return 0, err
		}
		if len(indexes) == 0 {
			break
		}

		for _, item := range indexes {
			if item == nil {
				continue
			}
			tag := normalizeTag(item.Tag)
			listen := normalizeListen(item.Listen)
			protocol := strings.TrimSpace(item.Protocol)
			if tag == "" || listen == "" || protocol == "" || item.Port <= 0 || item.Port > 65535 {
				continue
			}

			rawSemantic, err := buildSemanticFromInboundIndex(item, tag, listen)
			if err != nil {
				return 0, err
			}

			candidate := importCandidate{
				semantic: rawSemantic,
				listen:   listen,
				port:     item.Port,
				protocol: protocol,
			}
			if old, ok := candidates[tag]; ok {
				if old.listen != candidate.listen || old.port != candidate.port || old.protocol != candidate.protocol {
					return 0, &InboundSpecConflictError{
						Kind:  "tag",
						Field: "tag",
						Value: tag,
					}
				}
				continue
			}
			orderedTags = append(orderedTags, tag)
			candidates[tag] = candidate
		}

		if len(indexes) < filter.Limit {
			break
		}
		filter.Offset += filter.Limit
	}

	if len(candidates) == 0 {
		return 0, nil
	}

	changeNote := strings.TrimSpace(req.ChangeNote)
	if changeNote == "" {
		changeNote = "import-from-applied"
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	createdCount := int64(0)
	for _, tag := range orderedTags {
		candidate := candidates[tag]

		upsertReq := UpsertInboundSpecRequest{
			SpecID:       0,
			AgentHostID:  req.AgentHostID,
			CoreType:     coreType,
			Tag:          tag,
			Enabled:      &enabled,
			SemanticSpec: candidate.semantic,
			CoreSpecific: json.RawMessage("{}"),
			OperatorID:   req.OperatorID,
			ChangeNote:   changeNote,
		}

		existing, err := s.specs.FindByHostCoreTag(ctx, req.AgentHostID, coreType, tag)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return 0, err
		}
		if err == nil {
			if !req.OverwriteExisting {
				continue
			}
			upsertReq.SpecID = existing.ID
		}

		specID, revision, err := s.UpsertSpec(ctx, upsertReq)
		if err != nil {
			return 0, err
		}
		if err := s.renderDesiredArtifacts(ctx, req.AgentHostID, coreType, revision); err != nil {
			return 0, err
		}
		if upsertReq.SpecID == 0 && specID > 0 {
			createdCount++
		}
	}

	return createdCount, nil
}

func (s *inboundSpecService) ensureTagAvailable(ctx context.Context, agentHostID int64, coreType, tag string, excludeSpecID int64) error {
	existing, err := s.specs.FindByHostCoreTag(ctx, agentHostID, coreType, tag)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}
	if existing.ID == excludeSpecID {
		return nil
	}
	return &InboundSpecConflictError{
		Kind:           "tag",
		Field:          "tag",
		Value:          tag,
		ExistingSpecID: existing.ID,
		ExistingTag:    existing.Tag,
	}
}

func (s *inboundSpecService) ensureListenAvailable(ctx context.Context, agentHostID int64, coreType, listen string, port int, excludeSpecID int64) error {
	if listen == "" || port <= 0 {
		return nil
	}

	filter := repository.InboundSpecFilter{
		AgentHostID: &agentHostID,
		CoreType:    &coreType,
		Limit:       200,
		Offset:      0,
	}
	for {
		specs, err := s.specs.List(ctx, filter)
		if err != nil {
			return err
		}
		if len(specs) == 0 {
			return nil
		}

		for _, item := range specs {
			if item == nil || item.ID == excludeSpecID {
				continue
			}
			semantic, err := parseSemanticSpec(item.SemanticSpec)
			if err != nil {
				return fmt.Errorf("parse existing inbound semantic spec (id=%d): %w", item.ID, err)
			}
			if semantic.Listen == listen && semantic.Port == port {
				return &InboundSpecConflictError{
					Kind:           "listen",
					Field:          "semantic_spec.listen",
					Value:          fmt.Sprintf("%s:%d", listen, port),
					ExistingSpecID: item.ID,
					ExistingTag:    item.Tag,
				}
			}
		}

		if len(specs) < filter.Limit {
			return nil
		}
		filter.Offset += filter.Limit
	}
}


func (s *inboundSpecService) renderDesiredArtifacts(ctx context.Context, agentHostID int64, coreType string, desiredRevision int64) error {
	if s == nil || s.compiler == nil {
		return nil
	}
	_, err := s.compiler.RenderArtifacts(ctx, RenderArtifactsRequest{
		AgentHostID:     agentHostID,
		CoreType:        coreType,
		DesiredRevision: desiredRevision,
	})
	if err != nil {
		return fmt.Errorf("render desired artifacts: %w", err)
	}
	return nil
}

func (s *inboundSpecService) deleteDesiredArtifacts(ctx context.Context, agentHostID int64, coreType string, desiredRevision int64) error {
	if s == nil || s.compiler == nil {
		return nil
	}
	return s.compiler.DeleteArtifacts(ctx, agentHostID, coreType, desiredRevision)
}

func buildInboundSpecSnapshot(spec *repository.InboundSpec) ([]byte, error) {
	if spec == nil {
		return nil, fmt.Errorf("nil inbound spec")
	}
	type inboundSpecSnapshot struct {
		ID              int64           `json:"id"`
		AgentHostID     int64           `json:"agent_host_id"`
		CoreType        string          `json:"core_type"`
		Tag             string          `json:"tag"`
		Enabled         bool            `json:"enabled"`
		SemanticSpec    json.RawMessage `json:"semantic_spec"`
		CoreSpecific    json.RawMessage `json:"core_specific"`
		DesiredRevision int64           `json:"desired_revision"`
		CreatedBy       int64           `json:"created_by"`
		UpdatedBy       int64           `json:"updated_by"`
	}
	snapshot := inboundSpecSnapshot{
		ID:              spec.ID,
		AgentHostID:     spec.AgentHostID,
		CoreType:        spec.CoreType,
		Tag:             spec.Tag,
		Enabled:         spec.Enabled,
		SemanticSpec:    spec.SemanticSpec,
		CoreSpecific:    spec.CoreSpecific,
		DesiredRevision: spec.DesiredRevision,
		CreatedBy:       spec.CreatedBy,
		UpdatedBy:       spec.UpdatedBy,
	}
	return json.Marshal(snapshot)
}

func buildSemanticFromInboundIndex(item *repository.InboundIndex, tag, listen string) (json.RawMessage, error) {
	spec := inboundSemanticSpec{
		Tag:       tag,
		Protocol:  strings.TrimSpace(item.Protocol),
		Listen:    listen,
		Port:      item.Port,
		TLS:       item.TLS,
		Transport: item.Transport,
		Multiplex: item.Multiplex,
	}
	raw, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	return raw, nil
}
