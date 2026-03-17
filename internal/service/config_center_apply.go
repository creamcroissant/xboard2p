package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/creamcroissant/xboard/internal/repository"
)

var (
	ErrApplyOrchestratorInvalidRequest   = errors.New("service: invalid apply orchestrator request / 发布编排请求无效")
	ErrApplyOrchestratorNotConfigured    = errors.New("service: apply orchestrator not configured / 发布编排服务未配置")
	ErrApplyOrchestratorNotFound         = errors.New("service: apply run not found / 发布记录不存在")
	ErrApplyOrchestratorPermissionDenied = errors.New("service: apply run permission denied / 发布记录无权限")
	ErrApplyOrchestratorInvalidState     = errors.New("service: invalid apply run state / 发布状态无效")
)

const (
	applyRunStatusPending    = "pending"
	applyRunStatusApplying   = "applying"
	applyRunStatusSuccess    = "success"
	applyRunStatusFailed     = "failed"
	applyRunStatusRolledBack = "rolled_back"
)

// ApplyOrchestratorService orchestrates apply run lifecycle and batch payload assembly.
type ApplyOrchestratorService interface {
	CreateApplyRun(ctx context.Context, req CreateApplyRunRequest) (*repository.ApplyRun, error)
	ListApplyRuns(ctx context.Context, req ListApplyRunsRequest) (*ListApplyRunsResult, error)
	GetApplyBatch(ctx context.Context, req GetApplyBatchRequest) (*GetApplyBatchResult, error)
	ReportApplyResult(ctx context.Context, req ReportApplyResultRequest) error
}

// CreateApplyRunRequest defines one apply run creation.
type CreateApplyRunRequest struct {
	AgentHostID      int64
	CoreType         string
	TargetRevision   int64
	PreviousRevision int64
	OperatorID       int64
}

// ListApplyRunsRequest defines apply run list query from control plane.
type ListApplyRunsRequest struct {
	AgentHostID *int64
	CoreType    string
	Status      string
	Limit       int
	Offset      int
}

// ListApplyRunsResult returns paged apply runs.
type ListApplyRunsResult struct {
	Items []*repository.ApplyRun
	Total int64
}

// GetApplyBatchRequest queries one host/core apply batch by current revision.
type GetApplyBatchRequest struct {
	AgentHostID     int64
	CoreType        string
	CurrentRevision int64
	RunOperatorID   int64
}

// ApplyBatchArtifact is one deployable file payload.
type ApplyBatchArtifact struct {
	Filename    string
	SourceTag   string
	Content     []byte
	ContentHash string
}

// GetApplyBatchResult is the orchestration result returned to transport layer.
type GetApplyBatchResult struct {
	NotModified      bool
	RunID            string
	CoreType         string
	TargetRevision   int64
	PreviousRevision int64
	Artifacts        []ApplyBatchArtifact
}

// ReportApplyResultRequest carries one apply callback event.
type ReportApplyResultRequest struct {
	AgentHostID      int64
	RunID            string
	CoreType         string
	TargetRevision   int64
	Success          bool
	Status           string
	ErrorMessage     string
	RollbackRevision int64
	FinishedAt       int64
}

type applyOrchestratorService struct {
	artifacts repository.DesiredArtifactRepository
	applyRuns repository.ApplyRunRepository
}

// NewApplyOrchestratorService creates ApplyOrchestratorService.
func NewApplyOrchestratorService(
	artifacts repository.DesiredArtifactRepository,
	applyRuns repository.ApplyRunRepository,
) ApplyOrchestratorService {
	return &applyOrchestratorService{
		artifacts: artifacts,
		applyRuns: applyRuns,
	}
}

func (s *applyOrchestratorService) CreateApplyRun(ctx context.Context, req CreateApplyRunRequest) (*repository.ApplyRun, error) {
	if s == nil || s.applyRuns == nil {
		return nil, ErrApplyOrchestratorNotConfigured
	}
	if req.AgentHostID <= 0 {
		return nil, fmt.Errorf("%w (agent_host_id is required / 不能为空)", ErrApplyOrchestratorInvalidRequest)
	}
	if req.TargetRevision <= 0 {
		return nil, fmt.Errorf("%w (target_revision must be greater than 0 / 必须大于 0)", ErrApplyOrchestratorInvalidRequest)
	}

	coreType := normalizeCoreType(req.CoreType)
	if coreType == "" {
		return nil, fmt.Errorf("%w (core_type must be sing-box or xray / 必须是 sing-box 或 xray)", ErrApplyOrchestratorInvalidRequest)
	}

	runID, err := generateApplyRunID(req.AgentHostID)
	if err != nil {
		return nil, err
	}

	run := &repository.ApplyRun{
		RunID:            runID,
		AgentHostID:      req.AgentHostID,
		CoreType:         coreType,
		TargetRevision:   req.TargetRevision,
		Status:           applyRunStatusPending,
		ErrorMessage:     "",
		PreviousRevision: req.PreviousRevision,
		RollbackRevision: 0,
		OperatorID:       req.OperatorID,
		StartedAt:        time.Now().Unix(),
		FinishedAt:       0,
	}
	if err := s.applyRuns.Create(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *applyOrchestratorService) ListApplyRuns(ctx context.Context, req ListApplyRunsRequest) (*ListApplyRunsResult, error) {
	if s == nil || s.applyRuns == nil {
		return nil, ErrApplyOrchestratorNotConfigured
	}

	filter := repository.ApplyRunFilter{
		Limit:  req.Limit,
		Offset: req.Offset,
	}
	if req.AgentHostID != nil {
		if *req.AgentHostID <= 0 {
			return nil, fmt.Errorf("%w (agent_host_id must be greater than 0 / 必须大于 0)", ErrApplyOrchestratorInvalidRequest)
		}
		hostID := *req.AgentHostID
		filter.AgentHostID = &hostID
	}
	if coreType := normalizeCoreType(req.CoreType); req.CoreType != "" {
		if coreType == "" {
			return nil, fmt.Errorf("%w (core_type must be sing-box or xray / 必须是 sing-box 或 xray)", ErrApplyOrchestratorInvalidRequest)
		}
		filter.CoreType = &coreType
	}
	if statusValue := strings.ToLower(strings.TrimSpace(req.Status)); statusValue != "" {
		normalizedStatus, err := normalizeApplyRunStatus(false, statusValue)
		if err != nil {
			return nil, err
		}
		filter.Status = &normalizedStatus
	}

	total, err := s.applyRuns.Count(ctx, filter)
	if err != nil {
		return nil, err
	}
	items, err := s.applyRuns.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &ListApplyRunsResult{Items: items, Total: total}, nil
}

func (s *applyOrchestratorService) GetApplyBatch(ctx context.Context, req GetApplyBatchRequest) (*GetApplyBatchResult, error) {
	if s == nil || s.artifacts == nil || s.applyRuns == nil {
		return nil, ErrApplyOrchestratorNotConfigured
	}
	if req.AgentHostID <= 0 {
		return nil, fmt.Errorf("%w (agent_host_id is required / 不能为空)", ErrApplyOrchestratorInvalidRequest)
	}
	if req.CurrentRevision < 0 {
		return nil, fmt.Errorf("%w (current_revision must be greater than or equal to 0 / 必须大于等于 0)", ErrApplyOrchestratorInvalidRequest)
	}

	coreType := normalizeCoreType(req.CoreType)
	if coreType == "" {
		return nil, fmt.Errorf("%w (core_type is required / 不能为空)", ErrApplyOrchestratorInvalidRequest)
	}

	targetRevision, err := s.artifacts.GetLatestRevision(ctx, req.AgentHostID, coreType)
	if err != nil {
		return nil, err
	}
	if targetRevision == 0 {
		return &GetApplyBatchResult{
			NotModified:      true,
			RunID:            "",
			CoreType:         coreType,
			TargetRevision:   req.CurrentRevision,
			PreviousRevision: req.CurrentRevision,
			Artifacts:        nil,
		}, nil
	}
	if targetRevision <= req.CurrentRevision {
		return &GetApplyBatchResult{
			NotModified:      true,
			RunID:            "",
			CoreType:         coreType,
			TargetRevision:   targetRevision,
			PreviousRevision: req.CurrentRevision,
			Artifacts:        nil,
		}, nil
	}

	artifacts, err := s.artifacts.List(ctx, repository.DesiredArtifactFilter{
		AgentHostID:     req.AgentHostID,
		CoreType:        &coreType,
		DesiredRevision: &targetRevision,
		Limit:           1000,
		Offset:          0,
	})
	if err != nil {
		return nil, err
	}
	if len(artifacts) == 0 {
		return nil, fmt.Errorf("latest revision has no artifacts (agent_host_id=%d core_type=%s target_revision=%d)", req.AgentHostID, coreType, targetRevision)
	}

	payload := make([]ApplyBatchArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact == nil {
			continue
		}
		payload = append(payload, ApplyBatchArtifact{
			Filename:    artifact.Filename,
			SourceTag:   artifact.SourceTag,
			Content:     artifact.Content,
			ContentHash: artifact.ContentHash,
		})
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("latest revision has no valid artifacts (agent_host_id=%d core_type=%s target_revision=%d)", req.AgentHostID, coreType, targetRevision)
	}

	run, err := s.CreateApplyRun(ctx, CreateApplyRunRequest{
		AgentHostID:      req.AgentHostID,
		CoreType:         coreType,
		TargetRevision:   targetRevision,
		PreviousRevision: req.CurrentRevision,
		OperatorID:       req.RunOperatorID,
	})
	if err != nil {
		return nil, err
	}

	return &GetApplyBatchResult{
		NotModified:      false,
		RunID:            run.RunID,
		CoreType:         coreType,
		TargetRevision:   targetRevision,
		PreviousRevision: req.CurrentRevision,
		Artifacts:        payload,
	}, nil
}

func (s *applyOrchestratorService) ReportApplyResult(ctx context.Context, req ReportApplyResultRequest) error {
	if s == nil || s.applyRuns == nil {
		return ErrApplyOrchestratorNotConfigured
	}
	if req.AgentHostID <= 0 {
		return fmt.Errorf("%w (agent_host_id is required / 不能为空)", ErrApplyOrchestratorInvalidRequest)
	}
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		return fmt.Errorf("%w (run_id is required / 不能为空)", ErrApplyOrchestratorInvalidRequest)
	}

	run, err := s.applyRuns.FindByRunID(ctx, runID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrApplyOrchestratorNotFound
		}
		return err
	}
	if run.AgentHostID != req.AgentHostID {
		return ErrApplyOrchestratorPermissionDenied
	}

	if reqCoreType := strings.TrimSpace(req.CoreType); reqCoreType != "" {
		reqCoreType = normalizeCoreType(reqCoreType)
		if reqCoreType == "" {
			return fmt.Errorf("%w (core_type must be sing-box or xray / 必须是 sing-box 或 xray)", ErrApplyOrchestratorInvalidRequest)
		}
		if reqCoreType != run.CoreType {
			return fmt.Errorf("%w (core_type does not match apply run)", ErrApplyOrchestratorInvalidRequest)
		}
	}
	if req.TargetRevision > 0 && req.TargetRevision != run.TargetRevision {
		return fmt.Errorf("%w (target_revision does not match apply run)", ErrApplyOrchestratorInvalidRequest)
	}

	nextStatus, err := normalizeApplyRunStatus(req.Success, req.Status)
	if err != nil {
		return err
	}
	if err := validateApplyResultStatusConsistency(req.Success, req.Status, nextStatus); err != nil {
		return err
	}
	currentStatus, err := normalizeApplyRunStatus(false, run.Status)
	if err != nil {
		return fmt.Errorf("%w (stored status is invalid: %s)", ErrApplyOrchestratorInvalidState, run.Status)
	}
	if currentStatus == nextStatus {
		return nil
	}
	if !isApplyRunTransitionAllowed(currentStatus, nextStatus) {
		return fmt.Errorf("%w (from=%s to=%s)", ErrApplyOrchestratorInvalidState, currentStatus, nextStatus)
	}

	finishedAt := req.FinishedAt
	if isApplyRunTerminalStatus(nextStatus) {
		if finishedAt <= 0 {
			finishedAt = time.Now().Unix()
		}
	} else {
		finishedAt = 0
	}

	if err := s.applyRuns.UpdateStatus(ctx, run.RunID, nextStatus, req.ErrorMessage, req.RollbackRevision, finishedAt); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrApplyOrchestratorNotFound
		}
		return err
	}
	return nil
}

func normalizeApplyRunStatus(success bool, statusValue string) (string, error) {
	if trimmed := strings.ToLower(strings.TrimSpace(statusValue)); trimmed != "" {
		switch trimmed {
		case applyRunStatusPending, applyRunStatusApplying, applyRunStatusSuccess, applyRunStatusFailed, applyRunStatusRolledBack:
			return trimmed, nil
		default:
			return "", fmt.Errorf("%w (status must be pending/applying/success/failed/rolled_back)", ErrApplyOrchestratorInvalidRequest)
		}
	}
	if success {
		return applyRunStatusSuccess, nil
	}
	return applyRunStatusFailed, nil
}

func isApplyRunTransitionAllowed(currentStatus, nextStatus string) bool {
	switch currentStatus {
	case applyRunStatusPending:
		return nextStatus == applyRunStatusApplying || nextStatus == applyRunStatusSuccess || nextStatus == applyRunStatusFailed || nextStatus == applyRunStatusRolledBack
	case applyRunStatusApplying:
		return nextStatus == applyRunStatusSuccess || nextStatus == applyRunStatusFailed || nextStatus == applyRunStatusRolledBack
	case applyRunStatusFailed:
		return nextStatus == applyRunStatusRolledBack
	case applyRunStatusSuccess, applyRunStatusRolledBack:
		return false
	default:
		return false
	}
}

func isApplyRunTerminalStatus(statusValue string) bool {
	switch statusValue {
	case applyRunStatusSuccess, applyRunStatusFailed, applyRunStatusRolledBack:
		return true
	default:
		return false
	}
}

func validateApplyResultStatusConsistency(success bool, statusValue, nextStatus string) error {
	trimmed := strings.TrimSpace(statusValue)
	if trimmed == "" {
		return nil
	}
	expected := applyRunStatusFailed
	if success {
		expected = applyRunStatusSuccess
	}
	if nextStatus == applyRunStatusApplying || nextStatus == applyRunStatusPending {
		return nil
	}
	if nextStatus != expected {
		return fmt.Errorf("%w (status and success are inconsistent)", ErrApplyOrchestratorInvalidRequest)
	}
	return nil
}

func generateApplyRunID(agentHostID int64) (string, error) {
	randSuffix, err := randomHex(4)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("run_%d_%d_%s", agentHostID, time.Now().UnixNano(), randSuffix), nil
}
