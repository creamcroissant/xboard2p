package configcenter

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/creamcroissant/xboard/internal/agent/config"
	"github.com/creamcroissant/xboard/internal/agent/transport"
	agentv1 "github.com/creamcroissant/xboard/pkg/pb/agent/v1"
)

const (
	applyRunStatusApplying   = "applying"
	applyRunStatusSuccess    = "success"
	applyRunStatusFailed     = "failed"
	applyRunStatusRolledBack = "rolled_back"
)

// BatchClient defines gRPC operations required by AgentBatchApplier.
type BatchClient interface {
	GetApplyBatch(ctx context.Context, coreType string, currentRevision int64) (*agentv1.ApplyBatchResponse, error)
	ReportApplyRun(ctx context.Context, report *agentv1.ApplyRunReport) (*agentv1.ApplyRunResponse, error)
}

// ProtocolManager defines protocol manager capabilities required by AgentBatchApplier.
type ProtocolManager interface {
	ValidateConfigInDir(ctx context.Context, dir string) error
	ReloadServiceWithValidationDir(ctx context.Context, dir string) error
}

// AgentBatchApplier pulls apply batches, performs staged atomic switch, rollback, and run reporting.
type AgentBatchApplier struct {
	coreType        string
	legacyDir       string
	managedDir      string
	mergeOutputFile string

	client   BatchClient
	protoMgr ProtocolManager
	logger   *slog.Logger
}

// NewAgentBatchApplier creates AgentBatchApplier with normalized protocol paths.
func NewAgentBatchApplier(
	cfg config.ProtocolConfig,
	coreType string,
	client BatchClient,
	protoMgr ProtocolManager,
	logger *slog.Logger,
) (*AgentBatchApplier, error) {
	if client == nil {
		return nil, fmt.Errorf("batch client is required")
	}
	if protoMgr == nil {
		return nil, fmt.Errorf("protocol manager is required")
	}

	normalizedCore := normalizeCoreType(coreType)
	if normalizedCore == "" {
		return nil, fmt.Errorf("core_type must be sing-box or xray")
	}

	legacyDirAbs, managedDirAbs, mergeOutputFile, err := resolveProtocolPaths(cfg)
	if err != nil {
		return nil, err
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &AgentBatchApplier{
		coreType:        normalizedCore,
		legacyDir:       legacyDirAbs,
		managedDir:      managedDirAbs,
		mergeOutputFile: mergeOutputFile,
		client:          client,
		protoMgr:        protoMgr,
		logger:          logger,
	}, nil
}

// SyncOnce fetches one apply batch and performs staged apply with rollback/report.
// It returns latest applied revision on success, otherwise returns currentRevision with an error.
func (a *AgentBatchApplier) SyncOnce(ctx context.Context, currentRevision int64) (int64, error) {
	if currentRevision < 0 {
		return currentRevision, fmt.Errorf("current revision must be >= 0")
	}

	batchResp, err := a.client.GetApplyBatch(ctx, a.coreType, currentRevision)
	if err != nil {
		return currentRevision, fmt.Errorf("fetch apply batch: %w", err)
	}
	if batchResp == nil {
		return currentRevision, fmt.Errorf("fetch apply batch: empty response")
	}
	if !batchResp.GetSuccess() {
		message := strings.TrimSpace(batchResp.GetErrorMessage())
		if message == "" {
			message = "panel rejected apply batch"
		}
		return currentRevision, fmt.Errorf("fetch apply batch: %s", message)
	}
	if batchResp.GetNotModified() {
		return currentRevision, nil
	}

	runID := strings.TrimSpace(batchResp.GetRunId())
	if runID == "" {
		return currentRevision, fmt.Errorf("apply batch missing run_id")
	}
	targetRevision := batchResp.GetTargetRevision()
	previousRevision := batchResp.GetPreviousRevision()
	if targetRevision <= currentRevision {
		return currentRevision, fmt.Errorf("invalid target revision: %d <= current revision: %d", targetRevision, currentRevision)
	}

	batchCore := normalizeCoreType(batchResp.GetCoreType())
	if batchCore == "" {
		batchCore = a.coreType
	}
	if batchCore != a.coreType {
		return currentRevision, fmt.Errorf("batch core type mismatch: expected %s got %s", a.coreType, batchCore)
	}

	artifacts, err := a.normalizeArtifacts(batchResp.GetArtifacts())
	if err != nil {
		reportErr := a.reportApplyResult(ctx, runID, targetRevision, false, applyRunStatusFailed, err.Error(), 0)
		if reportErr != nil {
			return currentRevision, errors.Join(err, fmt.Errorf("report failed status: %w", reportErr))
		}
		return currentRevision, err
	}

	if reportErr := a.reportApplyResult(ctx, runID, targetRevision, false, applyRunStatusApplying, "", 0); reportErr != nil {
		a.logger.Warn("failed to report applying status",
			"run_id", runID,
			"error", reportErr,
			"error_category", transport.ClassifyError(reportErr).String(),
		)
	}

	execResult, applyErr := a.applyBatch(ctx, runID, previousRevision, artifacts)
	if applyErr != nil {
		status := applyRunStatusFailed
		rollbackRevision := int64(0)
		if execResult.rolledBack {
			status = applyRunStatusRolledBack
			rollbackRevision = execResult.rollbackRevision
		}
		reportErr := a.reportApplyResult(ctx, runID, targetRevision, false, status, applyErr.Error(), rollbackRevision)
		if reportErr != nil {
			return currentRevision, errors.Join(applyErr, fmt.Errorf("report %s status: %w", status, reportErr))
		}
		return currentRevision, applyErr
	}

	reportErr := a.reportApplyResult(ctx, runID, targetRevision, true, applyRunStatusSuccess, "", 0)
	if reportErr != nil {
		return currentRevision, fmt.Errorf("apply succeeded but report success failed: %w", reportErr)
	}

	return targetRevision, nil
}

type normalizedArtifact struct {
	filename    string
	sourceTag   string
	content     []byte
	contentHash string
}

type applyExecutionResult struct {
	rolledBack       bool
	rollbackRevision int64
}

func (a *AgentBatchApplier) normalizeArtifacts(artifacts []*agentv1.ApplyArtifact) ([]normalizedArtifact, error) {
	if len(artifacts) == 0 {
		return nil, fmt.Errorf("apply batch has no artifacts")
	}

	result := make([]normalizedArtifact, 0, len(artifacts))
	seen := make(map[string]struct{}, len(artifacts))
	for _, artifact := range artifacts {
		if artifact == nil {
			continue
		}
		filename, err := sanitizeFilename(artifact.GetFilename())
		if err != nil {
			return nil, fmt.Errorf("invalid artifact filename %q: %w", artifact.GetFilename(), err)
		}
		if _, ok := seen[filename]; ok {
			return nil, fmt.Errorf("duplicate artifact filename: %s", filename)
		}
		seen[filename] = struct{}{}

		content := append([]byte(nil), artifact.GetContent()...)
		if len(content) == 0 {
			return nil, fmt.Errorf("artifact %s content is empty", filename)
		}
		if !json.Valid(content) {
			return nil, fmt.Errorf("artifact %s content is not valid JSON", filename)
		}

		expectedHash := strings.ToLower(strings.TrimSpace(artifact.GetContentHash()))
		if expectedHash != "" {
			sum := md5.Sum(content)
			actualHash := hex.EncodeToString(sum[:])
			if actualHash != expectedHash {
				return nil, fmt.Errorf("artifact %s content_hash mismatch", filename)
			}
		}

		result = append(result, normalizedArtifact{
			filename:    filename,
			sourceTag:   strings.TrimSpace(artifact.GetSourceTag()),
			content:     content,
			contentHash: expectedHash,
		})
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("apply batch has no valid artifacts")
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].filename < result[j].filename
	})
	return result, nil
}

func (a *AgentBatchApplier) applyBatch(ctx context.Context, runID string, previousRevision int64, artifacts []normalizedArtifact) (applyExecutionResult, error) {
	var result applyExecutionResult

	managedParent := filepath.Dir(a.managedDir)
	if err := os.MkdirAll(managedParent, 0o755); err != nil {
		return result, fmt.Errorf("create managed parent dir: %w", err)
	}

	stageDir, err := os.MkdirTemp(managedParent, ".xboard_apply_stage_*")
	if err != nil {
		return result, fmt.Errorf("create stage dir: %w", err)
	}
	stageDir = filepath.Clean(stageDir)

	stageMoved := false
	defer func() {
		if !stageMoved {
			_ = os.RemoveAll(stageDir)
		}
	}()

	if err := a.writeArtifactsToStage(stageDir, artifacts); err != nil {
		return result, err
	}

	if a.coreType == "xray" {
		if err := a.buildMergedOutput(stageDir); err != nil {
			return result, fmt.Errorf("build merged output: %w", err)
		}
	}

	if err := a.protoMgr.ValidateConfigInDir(ctx, stageDir); err != nil {
		return result, fmt.Errorf("validate staged batch: %w", err)
	}

	backupDir, switched, err := a.switchManagedDir(runID, stageDir)
	if err != nil {
		return result, err
	}
	stageMoved = true

	reloadErr := a.protoMgr.ReloadServiceWithValidationDir(ctx, a.managedDir)
	if reloadErr == nil {
		if switched {
			_ = os.RemoveAll(backupDir)
		}
		return result, nil
	}

	if !switched {
		return result, fmt.Errorf("reload service after apply: %w", reloadErr)
	}

	rollbackErr := a.rollbackManagedDir(backupDir)
	if rollbackErr != nil {
		return result, fmt.Errorf("reload service after apply: %w (rollback switch failed: %v)", reloadErr, rollbackErr)
	}
	result.rolledBack = true
	result.rollbackRevision = previousRevision

	reloadRollbackErr := a.protoMgr.ReloadServiceWithValidationDir(ctx, a.managedDir)
	if reloadRollbackErr != nil {
		return result, fmt.Errorf("reload service after apply: %w (rollback reload failed: %v)", reloadErr, reloadRollbackErr)
	}

	return result, fmt.Errorf("reload service after apply: %w (rolled back to previous revision)", reloadErr)
}

func (a *AgentBatchApplier) writeArtifactsToStage(stageDir string, artifacts []normalizedArtifact) error {
	if err := os.MkdirAll(stageDir, 0o755); err != nil {
		return fmt.Errorf("create stage dir: %w", err)
	}
	for _, artifact := range artifacts {
		path := filepath.Join(stageDir, artifact.filename)
		if err := os.WriteFile(path, artifact.content, 0o644); err != nil {
			return fmt.Errorf("write stage artifact %s: %w", artifact.filename, err)
		}
	}
	return nil
}

func (a *AgentBatchApplier) switchManagedDir(runID, stageDir string) (backupDir string, switched bool, err error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		runID = "unknown"
	}
	backupDir = filepath.Join(filepath.Dir(a.managedDir), fmt.Sprintf(".xboard_apply_backup_%s_%d", runID, time.Now().UnixNano()))

	if _, statErr := os.Stat(a.managedDir); statErr == nil {
		if err := os.Rename(a.managedDir, backupDir); err != nil {
			return "", false, fmt.Errorf("move current managed dir to backup: %w", err)
		}
		switched = true
	} else if !os.IsNotExist(statErr) {
		return "", false, fmt.Errorf("stat managed dir: %w", statErr)
	}

	if err := os.Rename(stageDir, a.managedDir); err != nil {
		if switched {
			_ = os.Rename(backupDir, a.managedDir)
		}
		return "", false, fmt.Errorf("activate staged batch: %w", err)
	}
	return backupDir, switched, nil
}

func (a *AgentBatchApplier) rollbackManagedDir(backupDir string) error {
	backupDir = strings.TrimSpace(backupDir)
	if backupDir == "" {
		return fmt.Errorf("rollback backup dir is empty")
	}
	if err := os.RemoveAll(a.managedDir); err != nil {
		return fmt.Errorf("remove failed managed dir: %w", err)
	}
	if err := os.Rename(backupDir, a.managedDir); err != nil {
		return fmt.Errorf("restore backup managed dir: %w", err)
	}
	return nil
}

func (a *AgentBatchApplier) buildMergedOutput(stageDir string) error {
	legacyBasePath := filepath.Join(a.legacyDir, a.mergeOutputFile)
	baseDocument := map[string]any{}

	if content, err := os.ReadFile(legacyBasePath); err == nil {
		if err := json.Unmarshal(content, &baseDocument); err != nil {
			return fmt.Errorf("parse legacy base config %s: %w", legacyBasePath, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read legacy base config %s: %w", legacyBasePath, err)
	}

	baseInbounds, err := extractInbounds(baseDocument)
	if err != nil {
		return fmt.Errorf("extract legacy inbounds: %w", err)
	}
	managedInbounds, err := collectManagedInbounds(stageDir, a.mergeOutputFile)
	if err != nil {
		return err
	}

	mergedInbounds, err := overlayInboundsByTag(baseInbounds, managedInbounds)
	if err != nil {
		return err
	}

	inboundsAny := make([]any, 0, len(mergedInbounds))
	for _, inbound := range mergedInbounds {
		inboundsAny = append(inboundsAny, inbound)
	}
	baseDocument["inbounds"] = inboundsAny

	mergedContent, err := json.MarshalIndent(baseDocument, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal merged output: %w", err)
	}
	mergedPath := filepath.Join(stageDir, a.mergeOutputFile)
	if err := os.WriteFile(mergedPath, mergedContent, 0o644); err != nil {
		return fmt.Errorf("write merged output %s: %w", mergedPath, err)
	}
	return nil
}

func extractInbounds(document map[string]any) ([]map[string]any, error) {
	if document == nil {
		return nil, nil
	}
	inboundsRaw, ok := document["inbounds"]
	if !ok || inboundsRaw == nil {
		return nil, nil
	}
	inboundsArray, ok := inboundsRaw.([]any)
	if !ok {
		return nil, fmt.Errorf("inbounds must be array")
	}
	result := make([]map[string]any, 0, len(inboundsArray))
	for _, item := range inboundsArray {
		inbound, ok := toMap(item)
		if !ok {
			return nil, fmt.Errorf("inbounds entry must be object")
		}
		result = append(result, inbound)
	}
	return result, nil
}

func collectManagedInbounds(stageDir, mergeOutputFile string) ([]map[string]any, error) {
	entries, err := os.ReadDir(stageDir)
	if err != nil {
		return nil, fmt.Errorf("read stage dir: %w", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			continue
		}
		if name == mergeOutputFile {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)

	result := make([]map[string]any, 0)
	for _, filename := range files {
		path := filepath.Join(stageDir, filename)
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read staged file %s: %w", filename, err)
		}
		var document map[string]any
		if err := json.Unmarshal(content, &document); err != nil {
			return nil, fmt.Errorf("parse staged file %s: %w", filename, err)
		}
		inbounds, err := extractInbounds(document)
		if err != nil {
			return nil, fmt.Errorf("extract inbounds from staged file %s: %w", filename, err)
		}
		result = append(result, inbounds...)
	}
	return result, nil
}

func overlayInboundsByTag(baseInbounds, managedInbounds []map[string]any) ([]map[string]any, error) {
	managedByTag := make(map[string]map[string]any, len(managedInbounds))
	managedNoTag := make([]map[string]any, 0)

	for _, inbound := range managedInbounds {
		tag := inboundTag(inbound)
		if tag == "" {
			managedNoTag = append(managedNoTag, inbound)
			continue
		}
		if _, exists := managedByTag[tag]; exists {
			return nil, fmt.Errorf("duplicate managed inbound tag: %s", tag)
		}
		managedByTag[tag] = inbound
	}

	result := make([]map[string]any, 0, len(baseInbounds)+len(managedInbounds))
	usedManagedTags := make(map[string]struct{}, len(managedByTag))
	for _, inbound := range baseInbounds {
		tag := inboundTag(inbound)
		if tag != "" {
			if managed, ok := managedByTag[tag]; ok {
				result = append(result, managed)
				usedManagedTags[tag] = struct{}{}
				continue
			}
		}
		result = append(result, inbound)
	}

	remainingTags := make([]string, 0)
	for tag := range managedByTag {
		if _, used := usedManagedTags[tag]; used {
			continue
		}
		remainingTags = append(remainingTags, tag)
	}
	sort.Strings(remainingTags)
	for _, tag := range remainingTags {
		result = append(result, managedByTag[tag])
	}

	result = append(result, managedNoTag...)
	return result, nil
}

func inboundTag(inbound map[string]any) string {
	if inbound == nil {
		return ""
	}
	tag, _ := inbound["tag"].(string)
	return strings.TrimSpace(tag)
}

func (a *AgentBatchApplier) reportApplyResult(
	ctx context.Context,
	runID string,
	targetRevision int64,
	success bool,
	statusValue string,
	errorMessage string,
	rollbackRevision int64,
) error {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return fmt.Errorf("report apply result: run_id is required")
	}
	report := &agentv1.ApplyRunReport{
		RunId:            runID,
		CoreType:         a.coreType,
		TargetRevision:   targetRevision,
		Success:          success,
		Status:           statusValue,
		ErrorMessage:     strings.TrimSpace(errorMessage),
		RollbackRevision: rollbackRevision,
		FinishedAt:       time.Now().Unix(),
	}
	resp, err := a.client.ReportApplyRun(ctx, report)
	if err != nil {
		return fmt.Errorf("report apply result: %w", err)
	}
	if resp != nil && !resp.GetSuccess() {
		message := strings.TrimSpace(resp.GetMessage())
		if message == "" {
			message = "panel rejected apply report"
		}
		return fmt.Errorf("report apply result: %s", message)
	}
	return nil
}

func toMap(value any) (map[string]any, bool) {
	mapped, ok := value.(map[string]any)
	if !ok || mapped == nil {
		return nil, false
	}
	return mapped, true
}
