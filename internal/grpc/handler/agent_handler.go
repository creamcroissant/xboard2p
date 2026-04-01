package handler

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/creamcroissant/xboard/internal/grpc/interceptor"
	"github.com/creamcroissant/xboard/internal/repository"
	"github.com/creamcroissant/xboard/internal/service"
	agentv1 "github.com/creamcroissant/xboard/pkg/pb/agent/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AgentHandler 实现 AgentServiceServer 接口。
type AgentHandler struct {
	agentv1.UnimplementedAgentServiceServer

	agentHostService   service.AgentHostService
	agentService       service.AgentService
	telemetryService   service.ServerTelemetryService
	serverNodeService  service.ServerNodeService
	userTrafficService service.UserTrafficService
	trafficDedupRepo   repository.TrafficReportDedupRepository
	forwardingService  service.ForwardingService
	accessLogService   service.AccessLogService
	settingsService    service.AdminSystemSettingsService
	inventoryIngest    service.InventoryIngestService
	applyOrchestrator  service.ApplyOrchestratorService
	coreOperations     service.CoreOperationService
	coreSnapshots      service.CoreSnapshotService
	logger             *slog.Logger
	timeNow            func() time.Time
}

// NewAgentHandler 创建 Agent gRPC 处理器。
func NewAgentHandler(
	agentHostService service.AgentHostService,
	agentService service.AgentService,
	telemetryService service.ServerTelemetryService,
	serverNodeService service.ServerNodeService,
	userTrafficService service.UserTrafficService,
	trafficDedupRepo repository.TrafficReportDedupRepository,
	forwardingService service.ForwardingService,
	accessLogService service.AccessLogService,
	settingsService service.AdminSystemSettingsService,
	inventoryIngest service.InventoryIngestService,
	applyOrchestrator service.ApplyOrchestratorService,
	logger *slog.Logger,
) *AgentHandler {
	return NewAgentHandlerWithCoreServices(
		agentHostService,
		agentService,
		telemetryService,
		serverNodeService,
		userTrafficService,
		trafficDedupRepo,
		forwardingService,
		accessLogService,
		settingsService,
		inventoryIngest,
		applyOrchestrator,
		nil,
		nil,
		logger,
	)
}

// NewAgentHandlerWithCoreServices 创建带 core operation/core snapshot 能力的 Agent gRPC 处理器。
func NewAgentHandlerWithCoreServices(
	agentHostService service.AgentHostService,
	agentService service.AgentService,
	telemetryService service.ServerTelemetryService,
	serverNodeService service.ServerNodeService,
	userTrafficService service.UserTrafficService,
	trafficDedupRepo repository.TrafficReportDedupRepository,
	forwardingService service.ForwardingService,
	accessLogService service.AccessLogService,
	settingsService service.AdminSystemSettingsService,
	inventoryIngest service.InventoryIngestService,
	applyOrchestrator service.ApplyOrchestratorService,
	coreOperations service.CoreOperationService,
	coreSnapshots service.CoreSnapshotService,
	logger *slog.Logger,
) *AgentHandler {
	return &AgentHandler{
		agentHostService:   agentHostService,
		agentService:       agentService,
		telemetryService:   telemetryService,
		serverNodeService:  serverNodeService,
		userTrafficService: userTrafficService,
		trafficDedupRepo:   trafficDedupRepo,
		forwardingService:  forwardingService,
		accessLogService:   accessLogService,
		settingsService:    settingsService,
		inventoryIngest:    inventoryIngest,
		applyOrchestrator:  applyOrchestrator,
		coreOperations:     coreOperations,
		coreSnapshots:      coreSnapshots,
		logger:             logger,
		timeNow:            time.Now,
	}
}

// Heartbeat 处理 Agent 心跳请求。
func (h *AgentHandler) Heartbeat(ctx context.Context, req *agentv1.HeartbeatRequest) (*agentv1.HeartbeatResponse, error) {
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no agent host in context")
	}
	if err := h.agentHostService.UpdateHeartbeat(ctx, agentHost.Token); err != nil {
		h.logger.Error("failed to update heartbeat", "agent_host_id", agentHost.ID, "error", err)
		return nil, status.Error(codes.Internal, "failed to update heartbeat")
	}
	return &agentv1.HeartbeatResponse{Success: true, ServerTime: time.Now().Unix()}, nil
}

// ReportStatus 处理 Agent 状态上报。
func (h *AgentHandler) ReportStatus(ctx context.Context, req *agentv1.StatusReport) (*agentv1.StatusResponse, error) {
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no agent host in context")
	}

	metrics := service.AgentHostMetricsReport{
		CPUUsed:       req.System.GetCpuUsage(),
		MemTotal:      int64(req.System.GetMemoryTotal()),
		MemUsed:       int64(req.System.GetMemoryUsed()),
		DiskTotal:     int64(req.System.GetDiskTotal()),
		DiskUsed:      int64(req.System.GetDiskUsed()),
		UploadTotal:   int64(req.Network.GetUploadBytes()),
		DownloadTotal: int64(req.Network.GetDownloadBytes()),
	}
	if err := h.agentHostService.UpdateMetrics(ctx, agentHost.Token, metrics); err != nil {
		h.logger.Error("failed to update metrics", "agent_host_id", agentHost.ID, "error", err)
		return nil, status.Error(codes.Internal, "failed to update metrics")
	}

	if req.System != nil && (req.System.CoreVersion != "" || len(req.System.Capabilities) > 0 || len(req.System.BuildTags) > 0) {
		if err := h.agentHostService.UpdateCapabilities(ctx, agentHost.Token, req.System.CoreVersion, req.System.Capabilities, req.System.BuildTags); err != nil {
			h.logger.Error("failed to update capabilities", "agent_host_id", agentHost.ID, "error", err)
		}
	}

	if len(req.Protocols) > 0 {
		protocols := make([]service.ProtocolInfo, len(req.Protocols))
		for i, p := range req.Protocols {
			protocols[i] = service.ProtocolInfo{Name: p.Name, Type: p.Type, Running: p.Running, Details: convertProtocolDetails(p.Details)}
		}
		if err := h.agentHostService.UpdateProtocols(ctx, agentHost.Token, protocols); err != nil {
			h.logger.Error("failed to update protocols", "agent_host_id", agentHost.ID, "error", err)
		}
	}

	if req.ClientConfigs != nil && len(req.ClientConfigs.Configs) > 0 {
		clientConfigs := make([]service.ClientConfigInfo, len(req.ClientConfigs.Configs))
		for i, cfg := range req.ClientConfigs.Configs {
			rawConfigs := make(map[string]string)
			for k, v := range cfg.RawConfigs {
				rawConfigs[k] = v
			}
			clientConfigs[i] = service.ClientConfigInfo{Name: cfg.Name, Protocol: cfg.Protocol, Port: int(cfg.Port), RawConfigs: rawConfigs}
		}
		if err := h.agentHostService.UpdateClientConfigs(ctx, agentHost.Token, clientConfigs); err != nil {
			h.logger.Error("failed to update client configs", "agent_host_id", agentHost.ID, "error", err)
		}
	}

	if h.coreSnapshots != nil && len(req.GetInstances()) > 0 {
		snapshots := buildCoreSnapshotsFromStatus(req)
		if err := h.coreSnapshots.ReplaceInstanceSnapshot(ctx, agentHost.ID, req.GetInstances(), snapshots); err != nil {
			h.logger.Error("failed to replace core snapshot", "agent_host_id", agentHost.ID, "error", err)
		}
	}

	h.ingestInventoryReport(ctx, agentHost, req.GetTimestamp(), req.Inventory, req.InboundIndex, "unary")

	var syncInterval, reportInterval int
	if h.settingsService != nil {
		if val, err := h.settingsService.Get(ctx, "server_pull_interval"); err == nil && val != "" {
			syncInterval, _ = strconv.Atoi(val)
		}
		if val, err := h.settingsService.Get(ctx, "server_push_interval"); err == nil && val != "" {
			reportInterval, _ = strconv.Atoi(val)
		}
	}
	return &agentv1.StatusResponse{Success: true, Message: "status updated", SyncIntervalSeconds: int32(syncInterval), ReportIntervalSeconds: int32(reportInterval)}, nil
}

// GetConfig 为 Agent 获取节点配置。
func (h *AgentHandler) GetConfig(ctx context.Context, req *agentv1.ConfigRequest) (*agentv1.ConfigResponse, error) {
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no agent host in context")
	}
	configJSON, err := h.agentHostService.GenerateConfig(ctx, agentHost.ID)
	if err != nil {
		h.logger.Error("failed to generate config", "agent_host_id", agentHost.ID, "error", err)
		return nil, status.Error(codes.Internal, "failed to generate config")
	}
	if configJSON == nil {
		return &agentv1.ConfigResponse{NotModified: true}, nil
	}
	newETag := fmt.Sprintf("%x", md5.Sum(configJSON))
	if req.Etag == newETag {
		return &agentv1.ConfigResponse{NotModified: true, Etag: newETag}, nil
	}
	return &agentv1.ConfigResponse{Success: true, ConfigJson: configJSON, Version: 1, Etag: newETag}, nil
}

// GetUsers 为 Agent 获取用户列表。
func (h *AgentHandler) GetUsers(ctx context.Context, req *agentv1.UsersRequest) (*agentv1.UsersResponse, error) {
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no agent host in context")
	}
	users, err := h.agentService.GetUsersForAgent(ctx, agentHost.ID)
	if err != nil {
		h.logger.Error("failed to get users for agent", "agent_host_id", agentHost.ID, "error", err)
		return nil, status.Error(codes.Internal, "failed to fetch users")
	}
	pbUsers := make([]*agentv1.UserInfo, len(users))
	for i, u := range users {
		var speedLimit int64
		if u.SpeedLimit != nil {
			speedLimit = *u.SpeedLimit
		}
		var deviceLimit int32
		if u.DeviceLimit != nil {
			deviceLimit = int32(*u.DeviceLimit)
		}
		pbUsers[i] = &agentv1.UserInfo{UserId: int64(u.ID), Uuid: u.UUID, Email: u.Email, Enabled: true, SpeedLimit: speedLimit, DeviceLimit: deviceLimit}
	}
	return &agentv1.UsersResponse{Success: true, Users: pbUsers, Version: 1}, nil
}

// ReportTraffic 处理用户维度流量上报。
func (h *AgentHandler) ReportTraffic(ctx context.Context, req *agentv1.TrafficReport) (*agentv1.TrafficResponse, error) {
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no agent host in context")
	}
	reportID := strings.TrimSpace(req.GetReportId())
	if reportID != "" && h.trafficDedupRepo != nil {
		handledAt := h.timeNow().Unix()
		inserted, err := h.trafficDedupRepo.MarkHandled(ctx, agentHost.ID, reportID, handledAt)
		if err != nil {
			h.logger.Error("failed to mark traffic report dedup", "agent_host_id", agentHost.ID, "report_id", reportID, "error", err)
			return nil, status.Error(codes.Internal, "failed to process traffic batch")
		}
		if !inserted {
			h.logger.Info("traffic report deduplicated",
				"agent_host_id", agentHost.ID,
				"report_id", reportID,
				"dedup_hit", true,
				"accepted", 0,
				"skipped", len(req.UserTraffic),
			)
			return &agentv1.TrafficResponse{Success: true, AcceptedCount: 0, Message: "traffic accepted (deduplicated)"}, nil
		}
	}
	traffic := make([]service.UserTrafficDelta, 0, len(req.UserTraffic))
	skipped := 0
	for _, u := range req.UserTraffic {
		if u.UserId <= 0 {
			skipped++
			continue
		}
		upload := int64(u.UploadBytes)
		download := int64(u.DownloadBytes)
		if upload == 0 && download == 0 {
			skipped++
			continue
		}
		traffic = append(traffic, service.UserTrafficDelta{UserID: u.UserId, Upload: upload, Download: download})
	}
	acceptedCount := int32(len(traffic))
	if h.userTrafficService != nil && len(traffic) > 0 {
		result, err := h.userTrafficService.ProcessTrafficBatch(ctx, agentHost.ID, traffic)
		if err != nil {
			h.logger.Error("failed to process traffic batch", "agent_host_id", agentHost.ID, "error", err)
			return nil, status.Error(codes.Internal, "failed to process traffic batch")
		}
		if result != nil {
			acceptedCount = result.AcceptedCount
		}
	}
	h.logger.Debug("traffic report processed",
		"agent_host_id", agentHost.ID,
		"report_id", reportID,
		"dedup_hit", false,
		"received", len(req.UserTraffic),
		"accepted", acceptedCount,
		"skipped", skipped,
	)
	return &agentv1.TrafficResponse{Success: true, AcceptedCount: acceptedCount, Message: "traffic accepted"}, nil
}

// ReportForwardingStatus 处理转发规则应用结果上报。
func (h *AgentHandler) ReportForwardingStatus(ctx context.Context, req *agentv1.ForwardingStatusReport) (*agentv1.StatusResponse, error) {
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no agent host in context")
	}
	if h.forwardingService == nil {
		return nil, status.Error(codes.FailedPrecondition, "forwarding service not available")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	detailPayload := map[string]any{"version": req.Version, "applied_at": req.AppliedAt, "success": req.Success, "error_message": req.ErrorMessage}
	payload, err := json.Marshal(detailPayload)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to marshal forwarding apply result")
	}
	if err := h.forwardingService.LogApplyResult(ctx, agentHost.ID, nil, req.Success, string(payload)); err != nil {
		return nil, status.Error(codes.Internal, "failed to log forwarding apply result")
	}
	return &agentv1.StatusResponse{Success: true}, nil
}

// GetForwardingRules 为 Agent 获取转发规则。
func (h *AgentHandler) GetForwardingRules(ctx context.Context, req *agentv1.ForwardingRulesRequest) (*agentv1.ForwardingRulesResponse, error) {
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no agent host in context")
	}
	if h.forwardingService == nil {
		return nil, status.Error(codes.FailedPrecondition, "forwarding service not available")
	}
	currentVersion, err := h.forwardingService.GetVersionForAgent(ctx, agentHost.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch forwarding rules")
	}
	if req.GetVersion() == currentVersion {
		return &agentv1.ForwardingRulesResponse{Success: true, NotModified: true, Version: currentVersion}, nil
	}
	rules, err := h.forwardingService.ListEnabledByAgent(ctx, agentHost.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch forwarding rules")
	}
	pbRules := make([]*agentv1.ForwardingRule, 0, len(rules))
	for _, rule := range rules {
		pbRules = append(pbRules, &agentv1.ForwardingRule{Id: rule.ID, Name: rule.Name, Protocol: rule.Protocol, ListenPort: int32(rule.ListenPort), TargetAddress: rule.TargetAddress, TargetPort: int32(rule.TargetPort), Priority: int32(rule.Priority), Enabled: rule.Enabled})
	}
	return &agentv1.ForwardingRulesResponse{Success: true, Rules: pbRules, Version: currentVersion}, nil
}

// GetCoreOperations 为 Agent 拉取待执行的 core operation。
func (h *AgentHandler) GetCoreOperations(ctx context.Context, req *agentv1.GetCoreOperationsRequest) (*agentv1.GetCoreOperationsResponse, error) {
	agentHost, err := getAgentHost(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	if h.coreOperations == nil {
		return nil, status.Error(codes.FailedPrecondition, "core operation service not available")
	}
	limit := req.GetLimit()
	if limit <= 0 {
		limit = 1
	}
	operations := make([]*agentv1.CoreOperation, 0, limit)
	for i := int32(0); i < limit; i++ {
		op, err := h.coreOperations.ClaimNext(ctx, service.ClaimCoreOperationRequest{AgentHostID: agentHost.ID, ClaimedBy: fmt.Sprintf("agent-%d", agentHost.ID), Statuses: req.GetStatuses()})
		if err != nil {
			if errors.Is(err, service.ErrCoreOperationNotFound) {
				break
			}
			return nil, mapCoreOperationGRPCError(err)
		}
		operations = append(operations, convertCoreOperation(op))
	}
	return &agentv1.GetCoreOperationsResponse{Success: true, Operations: operations}, nil
}

// ReportCoreOperation 处理 Agent 上报的 core operation 结果。
func (h *AgentHandler) ReportCoreOperation(ctx context.Context, req *agentv1.ReportCoreOperationRequest) (*agentv1.ReportCoreOperationResponse, error) {
	agentHost, err := getAgentHost(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	if h.coreOperations == nil {
		return nil, status.Error(codes.FailedPrecondition, "core operation service not available")
	}
	if err := h.coreOperations.ReportResult(ctx, service.ReportCoreOperationResultRequest{AgentHostID: agentHost.ID, OperationID: req.GetOperationId(), Status: req.GetStatus(), ResultPayload: append(json.RawMessage(nil), req.GetResultPayload()...), ErrorMessage: req.GetErrorMessage(), FinishedAt: req.GetFinishedAt()}); err != nil {
		return nil, mapCoreOperationGRPCError(err)
	}
	return &agentv1.ReportCoreOperationResponse{Success: true, Message: "core operation accepted"}, nil
}

// ReportAccessLogs reports access logs
func (h *AgentHandler) ReportAccessLogs(ctx context.Context, req *agentv1.AccessLogReport) (*agentv1.AccessLogResponse, error) {
	agentHost, err := getAgentHost(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	if h.accessLogService == nil {
		return nil, status.Error(codes.FailedPrecondition, "access log service not available")
	}
	logs := make([]*repository.AccessLog, 0, len(req.GetEntries()))
	for _, entry := range req.GetEntries() {
		if entry == nil {
			continue
		}
		var startAt *int64
		if entry.ConnectionStart != nil {
			t := entry.ConnectionStart.AsTime().Unix()
			startAt = &t
		}
		var endAt *int64
		if entry.ConnectionEnd != nil {
			t := entry.ConnectionEnd.AsTime().Unix()
			endAt = &t
		}
		logs = append(logs, &repository.AccessLog{UserEmail: entry.UserEmail, SourceIP: entry.SourceIp, TargetDomain: entry.TargetDomain, TargetIP: entry.TargetIp, TargetPort: int(entry.TargetPort), Protocol: entry.Protocol, Upload: entry.Upload, Download: entry.Download, ConnectionStart: startAt, ConnectionEnd: endAt})
	}
	if err := h.accessLogService.LogAccessRecords(ctx, agentHost.ID, logs); err != nil {
		return nil, status.Error(codes.Internal, "failed to log access records")
	}
	return &agentv1.AccessLogResponse{Success: true, Message: "logs accepted", AcceptedCount: int32(len(logs))}, nil
}

// GetApplyBatch 为 Agent 获取目标 revision 的发布批次。
func (h *AgentHandler) GetApplyBatch(ctx context.Context, req *agentv1.ApplyBatchRequest) (*agentv1.ApplyBatchResponse, error) {
	agentHost, err := getAgentHost(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	if h.applyOrchestrator == nil {
		return nil, status.Error(codes.FailedPrecondition, "apply orchestrator service not available")
	}
	result, err := h.applyOrchestrator.GetApplyBatch(ctx, service.GetApplyBatchRequest{AgentHostID: agentHost.ID, CoreType: req.GetCoreType(), CurrentRevision: req.GetCurrentRevision()})
	if err != nil {
		return nil, mapApplyOrchestratorGRPCError(err)
	}
	if result.NotModified {
		return &agentv1.ApplyBatchResponse{Success: true, NotModified: true, RunId: "", CoreType: result.CoreType, TargetRevision: result.TargetRevision, PreviousRevision: result.PreviousRevision}, nil
	}
	pbArtifacts := make([]*agentv1.ApplyArtifact, 0, len(result.Artifacts))
	for _, artifact := range result.Artifacts {
		pbArtifacts = append(pbArtifacts, &agentv1.ApplyArtifact{Filename: artifact.Filename, SourceTag: artifact.SourceTag, Content: artifact.Content, ContentHash: artifact.ContentHash})
	}
	return &agentv1.ApplyBatchResponse{Success: true, NotModified: false, RunId: result.RunID, CoreType: result.CoreType, TargetRevision: result.TargetRevision, PreviousRevision: result.PreviousRevision, Artifacts: pbArtifacts}, nil
}

// ReportApplyRun 处理 Agent 上报的发布回执。
func (h *AgentHandler) ReportApplyRun(ctx context.Context, req *agentv1.ApplyRunReport) (*agentv1.ApplyRunResponse, error) {
	agentHost, err := getAgentHost(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	if h.applyOrchestrator == nil {
		return nil, status.Error(codes.FailedPrecondition, "apply orchestrator service not available")
	}
	err = h.applyOrchestrator.ReportApplyResult(ctx, service.ReportApplyResultRequest{AgentHostID: agentHost.ID, RunID: req.GetRunId(), CoreType: req.GetCoreType(), TargetRevision: req.GetTargetRevision(), Success: req.GetSuccess(), Status: req.GetStatus(), ErrorMessage: req.GetErrorMessage(), RollbackRevision: req.GetRollbackRevision(), FinishedAt: req.GetFinishedAt()})
	if err != nil {
		return nil, mapApplyOrchestratorGRPCError(err)
	}
	return &agentv1.ApplyRunResponse{Success: true, Message: "apply run accepted"}, nil
}

// StatusStream 处理双向流的实时状态更新。
func (h *AgentHandler) StatusStream(stream grpc.BidiStreamingServer[agentv1.StatusReport, agentv1.StatusCommand]) error {
	ctx := stream.Context()
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "no agent host in context")
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		report, err := stream.Recv()
		if err != nil {
			return err
		}
		metrics := service.AgentHostMetricsReport{CPUUsed: report.System.GetCpuUsage(), MemTotal: int64(report.System.GetMemoryTotal()), MemUsed: int64(report.System.GetMemoryUsed()), DiskTotal: int64(report.System.GetDiskTotal()), DiskUsed: int64(report.System.GetDiskUsed()), UploadTotal: int64(report.Network.GetUploadBytes()), DownloadTotal: int64(report.Network.GetDownloadBytes())}
		if err := h.agentHostService.UpdateMetrics(ctx, agentHost.Token, metrics); err != nil {
			h.logger.Error("failed to update metrics from stream", "agent_host_id", agentHost.ID, "error", err)
		}
		if len(report.Protocols) > 0 {
			protocols := make([]service.ProtocolInfo, len(report.Protocols))
			for i, p := range report.Protocols {
				protocols[i] = service.ProtocolInfo{Name: p.Name, Type: p.Type, Running: p.Running, Details: convertProtocolDetails(p.Details)}
			}
			if err := h.agentHostService.UpdateProtocols(ctx, agentHost.Token, protocols); err != nil {
				h.logger.Error("failed to update protocols from stream", "agent_host_id", agentHost.ID, "error", err)
			}
		}
		if report.ClientConfigs != nil && len(report.ClientConfigs.Configs) > 0 {
			clientConfigs := make([]service.ClientConfigInfo, len(report.ClientConfigs.Configs))
			for i, cfg := range report.ClientConfigs.Configs {
				rawConfigs := make(map[string]string)
				for k, v := range cfg.RawConfigs {
					rawConfigs[k] = v
				}
				clientConfigs[i] = service.ClientConfigInfo{Name: cfg.Name, Protocol: cfg.Protocol, Port: int(cfg.Port), RawConfigs: rawConfigs}
			}
			if err := h.agentHostService.UpdateClientConfigs(ctx, agentHost.Token, clientConfigs); err != nil {
				h.logger.Error("failed to update client configs from stream", "agent_host_id", agentHost.ID, "error", err)
			}
		}
		h.ingestInventoryReport(ctx, agentHost, report.GetTimestamp(), report.Inventory, report.InboundIndex, "stream")
	}
}

func (h *AgentHandler) ingestInventoryReport(ctx context.Context, agentHost *repository.AgentHost, reportedAt int64, inventory []*agentv1.ConfigInventoryEntry, inboundIndex []*agentv1.InboundIndexEntry, source string) {
	if h.inventoryIngest == nil || (len(inventory) == 0 && len(inboundIndex) == 0) {
		return
	}
	inventoryEntries := convertInventoryEntries(inventory)
	inboundIndexEntries := convertInboundIndexEntries(inboundIndex)
	if len(inventoryEntries) == 0 && len(inboundIndexEntries) == 0 {
		return
	}
	if err := h.inventoryIngest.IngestReport(ctx, service.IngestInventoryReportRequest{AgentHostID: agentHost.ID, ReportedAt: reportedAt, Inventory: inventoryEntries, InboundIndex: inboundIndexEntries}); err != nil {
		h.logger.Error("failed to ingest inventory report", "source", source, "agent_host_id", agentHost.ID, "inventory_count", len(inventoryEntries), "inbound_index_count", len(inboundIndexEntries), "error", err)
	}
}

// getAgentHost 是获取 Agent 上下文的辅助方法（兼容旧调用）。
func getAgentHost(ctx context.Context) (*repository.AgentHost, error) {
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no agent host in context")
	}
	return agentHost, nil
}

func mapApplyOrchestratorGRPCError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, service.ErrApplyOrchestratorInvalidRequest):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, service.ErrApplyOrchestratorNotConfigured):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, service.ErrApplyOrchestratorNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, service.ErrApplyOrchestratorPermissionDenied):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, service.ErrApplyOrchestratorInvalidState):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, service.ErrApplyOrchestratorNoArtifacts), errors.Is(err, service.ErrApplyOrchestratorNoPayload):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, "apply orchestrator operation failed")
	}
}

func mapCoreOperationGRPCError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, service.ErrCoreOperationInvalidRequest):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, service.ErrCoreOperationNotConfigured):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, service.ErrCoreOperationForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, service.ErrCoreOperationNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "core operation failed")
	}
}

func convertCoreOperation(op *repository.CoreOperation) *agentv1.CoreOperation {
	if op == nil {
		return nil
	}
	pb := &agentv1.CoreOperation{Id: op.ID, AgentHostId: op.AgentHostID, OperationType: op.OperationType, CoreType: op.CoreType, Status: op.Status, RequestPayload: op.RequestPayload, ResultPayload: op.ResultPayload, ErrorMessage: op.ErrorMessage, CreatedAt: op.CreatedAt, UpdatedAt: op.UpdatedAt}
	if op.StartedAt != nil {
		pb.StartedAt = *op.StartedAt
	}
	if op.FinishedAt != nil {
		pb.FinishedAt = *op.FinishedAt
	}
	return pb
}

func buildCoreSnapshotsFromStatus(report *agentv1.StatusReport) []*repository.CoreStatusSnapshot {
	if report == nil {
		return nil
	}
	byType := make(map[string]*repository.CoreStatusSnapshot)
	for _, inst := range report.GetInstances() {
		if inst == nil || inst.GetCoreType() == "" {
			continue
		}
		byType[inst.GetCoreType()] = &repository.CoreStatusSnapshot{Type: inst.GetCoreType(), Version: report.GetSystem().GetCoreVersion(), Installed: true, Capabilities: append([]string(nil), report.GetSystem().GetCapabilities()...)}
	}
	result := make([]*repository.CoreStatusSnapshot, 0, len(byType))
	for _, snapshot := range byType {
		result = append(result, snapshot)
	}
	return result
}

// convertProtocolDetails 将 protobuf 的协议详情转换为服务层结构。
func convertProtocolDetails(pbDetails []*agentv1.ProtocolDetails) []service.ProtocolDetails {
	if len(pbDetails) == 0 {
		return nil
	}
	details := make([]service.ProtocolDetails, len(pbDetails))
	for i, pb := range pbDetails {
		detail := service.ProtocolDetails{Protocol: pb.Protocol, Tag: pb.Tag, Listen: pb.Listen, Port: int(pb.Port), CoreType: pb.CoreType}
		if pb.Transport != nil {
			detail.Transport = &service.TransportInfo{Type: pb.Transport.Type, Path: pb.Transport.Path, Host: pb.Transport.Host, ServiceName: pb.Transport.ServiceName}
		}
		if pb.Tls != nil {
			detail.TLS = &service.TLSInfo{Enabled: pb.Tls.Enabled, ServerName: pb.Tls.ServerName, ALPN: pb.Tls.Alpn}
			if pb.Tls.Reality != nil {
				detail.TLS.Reality = &service.RealityInfo{Enabled: pb.Tls.Reality.Enabled, ShortIDs: pb.Tls.Reality.ShortIds, ServerName: pb.Tls.Reality.ServerName, Fingerprint: pb.Tls.Reality.Fingerprint, HandshakeAddr: pb.Tls.Reality.HandshakeAddr, HandshakePort: int(pb.Tls.Reality.HandshakePort), PublicKey: pb.Tls.Reality.PublicKey}
			}
		}
		if pb.Multiplex != nil {
			detail.Multiplex = &service.MultiplexInfo{Enabled: pb.Multiplex.Enabled, Padding: pb.Multiplex.Padding}
			if pb.Multiplex.Brutal != nil {
				detail.Multiplex.Brutal = &service.BrutalInfo{Enabled: pb.Multiplex.Brutal.Enabled, UpMbps: int(pb.Multiplex.Brutal.UpMbps), DownMbps: int(pb.Multiplex.Brutal.DownMbps)}
			}
		}
		for _, u := range pb.Users {
			detail.Users = append(detail.Users, service.UserInfoData{UUID: u.Uuid, Flow: u.Flow, Email: u.Email, Method: u.Method})
		}
		details[i] = detail
	}
	return details
}

func convertInventoryEntries(entries []*agentv1.ConfigInventoryEntry) []service.InventoryReportEntry {
	if len(entries) == 0 {
		return nil
	}
	result := make([]service.InventoryReportEntry, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		result = append(result, service.InventoryReportEntry{Source: entry.GetSource(), Filename: entry.GetFilename(), CoreType: entry.GetCoreType(), ContentHash: entry.GetContentHash(), ParseStatus: entry.GetParseStatus(), ParseError: entry.GetParseError(), LastSeenAt: entry.GetLastSeenAt()})
	}
	return result
}

func convertInboundIndexEntries(entries []*agentv1.InboundIndexEntry) []service.InboundIndexReportEntry {
	if len(entries) == 0 {
		return nil
	}
	result := make([]service.InboundIndexReportEntry, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		result = append(result, service.InboundIndexReportEntry{Source: entry.GetSource(), Filename: entry.GetFilename(), CoreType: entry.GetCoreType(), Tag: entry.GetTag(), Protocol: entry.GetProtocol(), Listen: entry.GetListen(), Port: int(entry.GetPort()), TLS: json.RawMessage(entry.GetTls()), Transport: json.RawMessage(entry.GetTransport()), Multiplex: json.RawMessage(entry.GetMultiplex()), LastSeenAt: entry.GetLastSeenAt()})
	}
	return result
}
