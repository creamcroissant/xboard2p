package configcenter

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/creamcroissant/xboard/internal/agent/config"
	"github.com/creamcroissant/xboard/internal/agent/protocol"
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

// ProtocolManager defines staged apply capabilities required by AgentBatchApplier.
type ProtocolManager interface {
	ExecuteStagedApply(ctx context.Context, req protocol.StagedApplyRequest) (protocol.StagedApplyResult, error)
}

// AgentBatchApplier pulls apply batches, delegates snapshot staged apply, and reports run status.
type AgentBatchApplier struct {
	coreType string

	client   BatchClient
	protoMgr ProtocolManager
	logger   *slog.Logger
}

// NewAgentBatchApplier creates AgentBatchApplier with validated protocol paths.
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

	if _, _, _, err := resolveProtocolPaths(cfg); err != nil {
		return nil, err
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &AgentBatchApplier{
		coreType: normalizedCore,
		client:   client,
		protoMgr: protoMgr,
		logger:   logger,
	}, nil
}

// SyncOnce fetches one apply batch and performs snapshot staged apply with rollback/report.
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
	filename string
	content  []byte
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
			filename: filename,
			content:  content,
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
	result, err := a.protoMgr.ExecuteStagedApply(ctx, protocol.StagedApplyRequest{
		Mode:          protocol.StagedApplyModeSnapshot,
		RunID:         runID,
		CoreType:      a.coreType,
		SnapshotFiles: toSnapshotFiles(artifacts),
	})
	if err != nil {
		if result.RolledBack {
			return applyExecutionResult{rolledBack: true, rollbackRevision: previousRevision}, err
		}
		return applyExecutionResult{}, err
	}
	return applyExecutionResult{}, nil
}

func toSnapshotFiles(artifacts []normalizedArtifact) []protocol.StagedApplyFile {
	files := make([]protocol.StagedApplyFile, 0, len(artifacts))
	for _, artifact := range artifacts {
		files = append(files, protocol.StagedApplyFile{
			Filename: artifact.filename,
			Content:  append([]byte(nil), artifact.content...),
		})
	}
	return files
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
