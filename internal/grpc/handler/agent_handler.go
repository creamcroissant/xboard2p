package handler

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
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
	forwardingService  service.ForwardingService
	accessLogService   service.AccessLogService
	settingsService    service.AdminSystemSettingsService
	logger             *slog.Logger
}

// NewAgentHandler 创建 Agent gRPC 处理器。
func NewAgentHandler(
	agentHostService service.AgentHostService,
	agentService service.AgentService,
	telemetryService service.ServerTelemetryService,
	serverNodeService service.ServerNodeService,
	userTrafficService service.UserTrafficService,
	forwardingService service.ForwardingService,
	accessLogService service.AccessLogService,
	settingsService service.AdminSystemSettingsService,
	logger *slog.Logger,
) *AgentHandler {
	return &AgentHandler{
		agentHostService:   agentHostService,
		agentService:       agentService,
		telemetryService:   telemetryService,
		serverNodeService:  serverNodeService,
		userTrafficService: userTrafficService,
		forwardingService:  forwardingService,
		accessLogService:   accessLogService,
		settingsService:    settingsService,
		logger:             logger,
	}
}

// Heartbeat 处理 Agent 心跳请求。
func (h *AgentHandler) Heartbeat(ctx context.Context, req *agentv1.HeartbeatRequest) (*agentv1.HeartbeatResponse, error) {
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no agent host in context")
	}

	if err := h.agentHostService.UpdateHeartbeat(ctx, agentHost.Token); err != nil {
		h.logger.Error("failed to update heartbeat",
			"agent_host_id", agentHost.ID,
			"error", err,
		)
		return nil, status.Error(codes.Internal, "failed to update heartbeat")
	}

	h.logger.Debug("heartbeat received",
		"agent_host_id", agentHost.ID,
		"agent_host_name", agentHost.Name,
	)

	return &agentv1.HeartbeatResponse{
		Success:    true,
		ServerTime: time.Now().Unix(),
	}, nil
}

// ReportStatus 处理 Agent 状态上报。
func (h *AgentHandler) ReportStatus(ctx context.Context, req *agentv1.StatusReport) (*agentv1.StatusResponse, error) {
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no agent host in context")
	}

	// 将 protobuf 数据转换为服务层指标
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
		h.logger.Error("failed to update metrics",
			"agent_host_id", agentHost.ID,
			"error", err,
		)
		return nil, status.Error(codes.Internal, "failed to update metrics")
	}

	// 处理核心能力信息
	if req.System != nil && (req.System.CoreVersion != "" || len(req.System.Capabilities) > 0 || len(req.System.BuildTags) > 0) {
		if err := h.agentHostService.UpdateCapabilities(ctx, agentHost.Token,
			req.System.CoreVersion,
			req.System.Capabilities,
			req.System.BuildTags,
		); err != nil {
			h.logger.Error("failed to update capabilities",
				"agent_host_id", agentHost.ID,
				"error", err,
			)
		} else {
			h.logger.Debug("updated capabilities",
				"agent_host_id", agentHost.ID,
				"core_version", req.System.CoreVersion,
				"capabilities", req.System.Capabilities,
				"build_tags", req.System.BuildTags,
			)
		}
	}

	// 处理协议清单
	if len(req.Protocols) > 0 {
		protocols := make([]service.ProtocolInfo, len(req.Protocols))
		for i, p := range req.Protocols {
			protocols[i] = service.ProtocolInfo{
				Name:    p.Name,
				Type:    p.Type,
				Running: p.Running,
				Details: convertProtocolDetails(p.Details),
			}
		}
		if err := h.agentHostService.UpdateProtocols(ctx, agentHost.Token, protocols); err != nil {
			h.logger.Error("failed to update protocols",
				"agent_host_id", agentHost.ID,
				"error", err,
			)
		}
	}

	// 处理订阅目录上传的客户端配置
	if req.ClientConfigs != nil && len(req.ClientConfigs.Configs) > 0 {
		clientConfigs := make([]service.ClientConfigInfo, len(req.ClientConfigs.Configs))
		for i, cfg := range req.ClientConfigs.Configs {
			rawConfigs := make(map[string]string)
			for k, v := range cfg.RawConfigs {
				rawConfigs[k] = v
			}
			clientConfigs[i] = service.ClientConfigInfo{
				Name:       cfg.Name,
				Protocol:   cfg.Protocol,
				Port:       int(cfg.Port),
				RawConfigs: rawConfigs,
			}
		}
		if err := h.agentHostService.UpdateClientConfigs(ctx, agentHost.Token, clientConfigs); err != nil {
			h.logger.Error("failed to update client configs",
				"agent_host_id", agentHost.ID,
				"error", err,
			)
		} else {
			h.logger.Debug("updated client configs",
				"agent_host_id", agentHost.ID,
				"count", len(clientConfigs),
			)
		}
	}

	h.logger.Debug("status report received",
		"agent_host_id", agentHost.ID,
		"cpu_usage", req.System.GetCpuUsage(),
		"upload_delta", req.Network.GetUploadDelta(),
		"download_delta", req.Network.GetDownloadDelta(),
	)

	// 获取配置的间隔
	var syncInterval, reportInterval int
	if h.settingsService != nil {
		if val, err := h.settingsService.Get(ctx, "server_pull_interval"); err == nil && val != "" {
			syncInterval, _ = strconv.Atoi(val)
		}
		if val, err := h.settingsService.Get(ctx, "server_push_interval"); err == nil && val != "" {
			reportInterval, _ = strconv.Atoi(val)
		}
	}

	return &agentv1.StatusResponse{
		Success:               true,
		Message:               "status updated",
		SyncIntervalSeconds:   int32(syncInterval),
		ReportIntervalSeconds: int32(reportInterval),
	}, nil
}

// GetConfig 为 Agent 获取节点配置。
func (h *AgentHandler) GetConfig(ctx context.Context, req *agentv1.ConfigRequest) (*agentv1.ConfigResponse, error) {
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no agent host in context")
	}

	h.logger.Debug("config request received",
		"agent_host_id", agentHost.ID,
		"node_id", req.NodeId,
		"etag", req.Etag,
	)

	// 从服务层获取渲染后的配置
	configJSON, err := h.agentHostService.GenerateConfig(ctx, agentHost.ID)
	if err != nil {
		h.logger.Error("failed to generate config",
			"agent_host_id", agentHost.ID,
			"error", err,
		)
		return nil, status.Error(codes.Internal, "failed to generate config")
	}

	if configJSON == nil {
		// 未分配模板或配置未变更
		return &agentv1.ConfigResponse{
			NotModified: true,
		}, nil
	}

	// 计算 ETag
	newETag := fmt.Sprintf("%x", md5.Sum(configJSON))
	if req.Etag == newETag {
		return &agentv1.ConfigResponse{
			NotModified: true,
			Etag:        newETag,
		}, nil
	}

	return &agentv1.ConfigResponse{
		Success:    true,
		ConfigJson: configJSON,
		Version:    1, // TODO: 补充版本控制
		Etag:       newETag,
	}, nil
}

// GetUsers 为 Agent 获取用户列表。
func (h *AgentHandler) GetUsers(ctx context.Context, req *agentv1.UsersRequest) (*agentv1.UsersResponse, error) {
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no agent host in context")
	}

	h.logger.Debug("users request received",
		"agent_host_id", agentHost.ID,
		"node_id", req.NodeId,
		"etag", req.Etag,
	)

	users, err := h.agentService.GetUsersForAgent(ctx, agentHost.ID)
	if err != nil {
		h.logger.Error("failed to get users for agent",
			"agent_host_id", agentHost.ID,
			"error", err,
		)
		return nil, status.Error(codes.Internal, "failed to fetch users")
	}

	// 将仓储用户转换为 protobuf 结构
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

		pbUsers[i] = &agentv1.UserInfo{
			UserId: int64(u.ID),
			Uuid:   u.UUID,
			Email:       u.Email,
			Enabled:     true,
			// Limiter 信息可在 NodeUser 可用时补充
			SpeedLimit:  speedLimit,
			DeviceLimit: deviceLimit,
		}
	}

	return &agentv1.UsersResponse{
		Success: true,
		Users:   pbUsers,
		Version: 1, // TODO: 补充版本/ETag 控制
	}, nil
}

// ReportTraffic 处理用户维度流量上报。
func (h *AgentHandler) ReportTraffic(ctx context.Context, req *agentv1.TrafficReport) (*agentv1.TrafficResponse, error) {
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no agent host in context")
	}

	h.logger.Debug("traffic report received",
		"agent_host_id", agentHost.ID,
		"user_count", len(req.UserTraffic),
	)

	// 准备批量处理的数据
	traffic := make([]service.UserTrafficDelta, 0, len(req.UserTraffic))
	for _, u := range req.UserTraffic {
		if u.UserId <= 0 {
			continue
		}
		upload := int64(u.UploadBytes)
		download := int64(u.DownloadBytes)
		if upload == 0 && download == 0 {
			continue
		}
		traffic = append(traffic, service.UserTrafficDelta{
			UserID:   u.UserId,
			Upload:   upload,
			Download: download,
		})
	}

	// 批量处理流量，并携带 agent_host_id
	if h.userTrafficService != nil && len(traffic) > 0 {
		result, err := h.userTrafficService.ProcessTrafficBatch(ctx, agentHost.ID, traffic)
		if err != nil {
			h.logger.Error("failed to process traffic batch",
				"agent_host_id", agentHost.ID,
				"error", err,
			)
			return nil, status.Error(codes.Internal, "failed to process traffic batch")
		}
		if result != nil && len(result.ExceededUserIDs) > 0 {
			h.logger.Debug("users exceeded quota",
				"agent_host_id", agentHost.ID,
				"exceeded_users", len(result.ExceededUserIDs),
			)
		}
	} else {
		h.logger.Warn("user traffic service not available, skipping traffic processing")
	}

	return &agentv1.TrafficResponse{
		Success:       true,
		AcceptedCount: int32(len(traffic)),
		Message:       "traffic accepted",
	}, nil
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

	// 构建应用结果明细并序列化
	detailPayload := map[string]any{
		"version":       req.Version,
		"applied_at":    req.AppliedAt,
		"success":       req.Success,
		"error_message": req.ErrorMessage,
	}
	payload, err := json.Marshal(detailPayload)
	if err != nil {
		h.logger.Error("failed to marshal forwarding apply result",
			"agent_host_id", agentHost.ID,
			"error", err,
		)
		return nil, status.Error(codes.Internal, "failed to marshal forwarding apply result")
	}

	if err := h.forwardingService.LogApplyResult(ctx, agentHost.ID, nil, req.Success, string(payload)); err != nil {
		h.logger.Error("failed to log forwarding apply result",
			"agent_host_id", agentHost.ID,
			"error", err,
		)
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
		h.logger.Error("failed to get forwarding version",
			"agent_host_id", agentHost.ID,
			"error", err,
		)
		return nil, status.Error(codes.Internal, "failed to fetch forwarding rules")
	}

	// 无变化时直接返回
	if req.GetVersion() == currentVersion {
		return &agentv1.ForwardingRulesResponse{
			Success:     true,
			NotModified: true,
			Version:     currentVersion,
		}, nil
	}

	rules, err := h.forwardingService.ListEnabledByAgent(ctx, agentHost.ID)
	if err != nil {
		h.logger.Error("failed to list forwarding rules",
			"agent_host_id", agentHost.ID,
			"error", err,
		)
		return nil, status.Error(codes.Internal, "failed to fetch forwarding rules")
	}

	pbRules := make([]*agentv1.ForwardingRule, 0, len(rules))
	for _, rule := range rules {
		pbRules = append(pbRules, &agentv1.ForwardingRule{
			Id:            rule.ID,
			Name:          rule.Name,
			Protocol:      rule.Protocol,
			ListenPort:    int32(rule.ListenPort),
			TargetAddress: rule.TargetAddress,
			TargetPort:    int32(rule.TargetPort),
			Priority:      int32(rule.Priority),
			Enabled:       rule.Enabled,
		})
	}

	return &agentv1.ForwardingRulesResponse{
		Success: true,
		Rules:   pbRules,
		Version: currentVersion,
	}, nil
}

// ReportAlive 处理存活用户上报。
func (h *AgentHandler) ReportAlive(ctx context.Context, req *agentv1.AliveReport) (*agentv1.AliveResponse, error) {
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no agent host in context")
	}

	h.logger.Debug("alive report received",
		"agent_host_id", agentHost.ID,
		"user_count", len(req.UserIds),
	)

	// TODO: 补充在线用户追踪
	return &agentv1.AliveResponse{
		Success: true,
	}, nil
}

// ReportAccessLogs 处理访问日志上报。
func (h *AgentHandler) ReportAccessLogs(ctx context.Context, req *agentv1.AccessLogReport) (*agentv1.AccessLogResponse, error) {
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no agent host in context")
	}

	if h.accessLogService == nil {
		return nil, status.Error(codes.FailedPrecondition, "access log service not available")
	}

	h.logger.Debug("access log report received",
		"agent_host_id", agentHost.ID,
		"count", len(req.Entries),
	)

	if len(req.Entries) == 0 {
		return &agentv1.AccessLogResponse{
			Success: true,
			Message: "no entries",
		}, nil
	}

	logs := make([]*repository.AccessLog, 0, len(req.Entries))
	for _, entry := range req.Entries {
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

		logs = append(logs, &repository.AccessLog{
			UserEmail:       entry.UserEmail,
			SourceIP:        entry.SourceIp,
			TargetDomain:    entry.TargetDomain,
			TargetIP:        entry.TargetIp,
			TargetPort:      int(entry.TargetPort),
			Protocol:        entry.Protocol,
			Upload:          entry.Upload,
			Download:        entry.Download,
			ConnectionStart: startAt,
			ConnectionEnd:   endAt,
		})
	}

	if err := h.accessLogService.LogAccessRecords(ctx, agentHost.ID, logs); err != nil {
		h.logger.Error("failed to log access records",
			"agent_host_id", agentHost.ID,
			"error", err,
		)
		return nil, status.Error(codes.Internal, "failed to log access records")
	}

	return &agentv1.AccessLogResponse{
		Success:       true,
		Message:       "logs accepted",
		AcceptedCount: int32(len(logs)),
	}, nil
}

// StatusStream 处理双向流的实时状态更新。
func (h *AgentHandler) StatusStream(stream grpc.BidiStreamingServer[agentv1.StatusReport, agentv1.StatusCommand]) error {
	ctx := stream.Context()
	agentHost, ok := interceptor.GetAgentHostFromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "no agent host in context")
	}

	h.logger.Info("status stream started",
		"agent_host_id", agentHost.ID,
		"agent_host_name", agentHost.Name,
	)

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("status stream ended",
				"agent_host_id", agentHost.ID,
				"reason", ctx.Err(),
			)
			return nil
		default:
		}

		report, err := stream.Recv()
		if err != nil {
			h.logger.Error("failed to receive status report",
				"agent_host_id", agentHost.ID,
				"error", err,
			)
			return err
		}

		// 处理上报的指标数据
		metrics := service.AgentHostMetricsReport{
			CPUUsed:       report.System.GetCpuUsage(),
			MemTotal:      int64(report.System.GetMemoryTotal()),
			MemUsed:       int64(report.System.GetMemoryUsed()),
			DiskTotal:     int64(report.System.GetDiskTotal()),
			DiskUsed:      int64(report.System.GetDiskUsed()),
			UploadTotal:   int64(report.Network.GetUploadBytes()),
			DownloadTotal: int64(report.Network.GetDownloadBytes()),
		}

		if err := h.agentHostService.UpdateMetrics(ctx, agentHost.Token, metrics); err != nil {
			h.logger.Error("failed to update metrics from stream",
				"agent_host_id", agentHost.ID,
				"error", err,
			)
		}

		// 处理流式协议清单
		if len(report.Protocols) > 0 {
			protocols := make([]service.ProtocolInfo, len(report.Protocols))
			for i, p := range report.Protocols {
				protocols[i] = service.ProtocolInfo{
					Name:    p.Name,
					Type:    p.Type,
					Running: p.Running,
					Details: convertProtocolDetails(p.Details),
				}
			}
			if err := h.agentHostService.UpdateProtocols(ctx, agentHost.Token, protocols); err != nil {
				h.logger.Error("failed to update protocols from stream",
					"agent_host_id", agentHost.ID,
					"error", err,
				)
			}
		}

		// 处理流式客户端配置
		if report.ClientConfigs != nil && len(report.ClientConfigs.Configs) > 0 {
			clientConfigs := make([]service.ClientConfigInfo, len(report.ClientConfigs.Configs))
			for i, cfg := range report.ClientConfigs.Configs {
				rawConfigs := make(map[string]string)
				for k, v := range cfg.RawConfigs {
					rawConfigs[k] = v
				}
				clientConfigs[i] = service.ClientConfigInfo{
					Name:       cfg.Name,
					Protocol:   cfg.Protocol,
					Port:       int(cfg.Port),
					RawConfigs: rawConfigs,
				}
			}
			if err := h.agentHostService.UpdateClientConfigs(ctx, agentHost.Token, clientConfigs); err != nil {
				h.logger.Error("failed to update client configs from stream",
					"agent_host_id", agentHost.ID,
					"error", err,
				)
			}
		}

		// 可选：向 Agent 回传命令
		// 当前未发送任何命令
		// 如需发送待处理命令，可在此处实现：
		// if err := stream.Send(&agentv1.StatusCommand{Command: "reload"}); err != nil {
		//     return err
		// }
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

// convertProtocolDetails 将 protobuf 的协议详情转换为服务层结构。
func convertProtocolDetails(pbDetails []*agentv1.ProtocolDetails) []service.ProtocolDetails {
	if len(pbDetails) == 0 {
		return nil
	}

	details := make([]service.ProtocolDetails, len(pbDetails))
	for i, pb := range pbDetails {
		detail := service.ProtocolDetails{
			Protocol: pb.Protocol,
			Tag:      pb.Tag,
			Listen:   pb.Listen,
			Port:     int(pb.Port),
			CoreType: pb.CoreType,
		}

		// 处理传输层参数
		if pb.Transport != nil {
			detail.Transport = &service.TransportInfo{
				Type:        pb.Transport.Type,
				Path:        pb.Transport.Path,
				Host:        pb.Transport.Host,
				ServiceName: pb.Transport.ServiceName,
			}
		}

		// 处理 TLS 参数
		if pb.Tls != nil {
			detail.TLS = &service.TLSInfo{
				Enabled:    pb.Tls.Enabled,
				ServerName: pb.Tls.ServerName,
				ALPN:       pb.Tls.Alpn,
			}
			if pb.Tls.Reality != nil {
				detail.TLS.Reality = &service.RealityInfo{
					Enabled:       pb.Tls.Reality.Enabled,
					ShortIDs:      pb.Tls.Reality.ShortIds,
					ServerName:    pb.Tls.Reality.ServerName,
					Fingerprint:   pb.Tls.Reality.Fingerprint,
					HandshakeAddr: pb.Tls.Reality.HandshakeAddr,
					HandshakePort: int(pb.Tls.Reality.HandshakePort),
					PublicKey:     pb.Tls.Reality.PublicKey,
				}
			}
		}

		// 处理多路复用参数
		if pb.Multiplex != nil {
			detail.Multiplex = &service.MultiplexInfo{
				Enabled: pb.Multiplex.Enabled,
				Padding: pb.Multiplex.Padding,
			}
			if pb.Multiplex.Brutal != nil {
				detail.Multiplex.Brutal = &service.BrutalInfo{
					Enabled:  pb.Multiplex.Brutal.Enabled,
					UpMbps:   int(pb.Multiplex.Brutal.UpMbps),
					DownMbps: int(pb.Multiplex.Brutal.DownMbps),
				}
			}
		}

		// 处理用户配置
		for _, u := range pb.Users {
			detail.Users = append(detail.Users, service.UserInfoData{
				UUID:   u.Uuid,
				Flow:   u.Flow,
				Email:  u.Email,
				Method: u.Method,
			})
		}

		details[i] = detail
	}

	return details
}
