package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/creamcroissant/xboard/internal/grpc/client"
	"github.com/creamcroissant/xboard/internal/repository"
	"github.com/creamcroissant/xboard/internal/template"
	agentv1 "github.com/creamcroissant/xboard/pkg/pb/agent/v1"
)

const (
	switchStatusPending   = "pending"
	switchStatusRunning   = "in_progress"
	switchStatusCompleted = "completed"
	switchStatusFailed    = "failed"
)

// AgentCoreService 定义 Panel 侧的核心管理逻辑。
type AgentCoreService interface {
	GetCores(ctx context.Context, agentHostID int64) ([]*agentv1.CoreInfo, error)
	GetInstances(ctx context.Context, agentHostID int64) ([]*repository.AgentCoreInstance, error)
	CreateInstance(ctx context.Context, req CreateInstanceRequest) (*repository.AgentCoreInstance, error)
	DeleteInstance(ctx context.Context, agentHostID int64, instanceID string) error
	SwitchCore(ctx context.Context, req SwitchCoreRequest) (*SwitchResult, error)
	GetSwitchLogs(ctx context.Context, filter SwitchLogFilter) ([]*repository.AgentCoreSwitchLog, int64, error)
	ConvertConfig(ctx context.Context, req ConvertRequest) (*ConvertResult, error)
}

// CreateInstanceRequest 定义创建核心实例的请求参数。
type CreateInstanceRequest struct {
	AgentHostID      int64
	CoreType         string
	InstanceID       string
	ConfigTemplateID int64
	ConfigJSON       json.RawMessage
	OperatorID       *int64
}

// SwitchCoreRequest 定义核心切换请求参数。
type SwitchCoreRequest struct {
	AgentHostID      int64
	FromInstanceID   string
	ToCoreType       string
	ConfigTemplateID int64
	ConfigJSON       json.RawMessage
	SwitchID         string
	ListenPorts      []int
	ZeroDowntime     *bool
	OperatorID       *int64
}

// SwitchResult 返回切换结果与日志信息。
type SwitchResult struct {
	Success        bool   `json:"success"`
	NewInstanceID  string `json:"new_instance_id,omitempty"`
	Message        string `json:"message,omitempty"`
	Error          string `json:"error,omitempty"`
	SwitchLogID    int64  `json:"switch_log_id,omitempty"`
	CompletedAt    *int64 `json:"completed_at,omitempty"`
	FromInstanceID string `json:"from_instance_id,omitempty"`
	ToCoreType     string `json:"to_core_type,omitempty"`
}

// SwitchLogFilter 定义切换日志查询条件。
type SwitchLogFilter struct {
	AgentHostID int64
	Status      *string
	StartAt     *int64
	EndAt       *int64
	Limit       int
	Offset      int
}

// ConvertRequest 定义配置转换的输入参数。
type ConvertRequest struct {
	SourceCore string
	TargetCore string
	ConfigJSON json.RawMessage
}

// ConvertResult 返回转换后的配置与警告信息。
type ConvertResult struct {
	ConfigJSON json.RawMessage `json:"config_json"`
	Warnings   []string        `json:"warnings,omitempty"`
}

// agentCoreService 组合核心管理相关依赖与配置。
type agentCoreService struct {
	agentHosts     repository.AgentHostRepository
	instances      repository.AgentCoreInstanceRepository
	switchLogs     repository.AgentCoreSwitchLogRepository
	templates      repository.ConfigTemplateRepository
	converters     *template.ConverterRegistry
	logger         *slog.Logger
	grpcTLS        *client.TLSConfig
	grpcTimeout    client.TimeoutConfig
	grpcKeepalive  *client.KeepaliveConfig
	grpcPort       string
	grpcClientFunc func(cfg client.Config) (*client.AgentClient, error)
}

// NewAgentCoreService 组装核心管理服务。
func NewAgentCoreService(
	agentHosts repository.AgentHostRepository,
	instances repository.AgentCoreInstanceRepository,
	switchLogs repository.AgentCoreSwitchLogRepository,
	templates repository.ConfigTemplateRepository,
	converters *template.ConverterRegistry,
	logger *slog.Logger,
) AgentCoreService {
	return NewAgentCoreServiceWithOptions(agentHosts, instances, switchLogs, templates, converters, logger, AgentCoreServiceOptions{})
}

// AgentCoreServiceOptions 定义 gRPC 客户端构造参数。
type AgentCoreServiceOptions struct {
	GRPCTLS       *client.TLSConfig
	Timeout       client.TimeoutConfig
	Keepalive     *client.KeepaliveConfig
	GRPCPort      string
	ClientFactory func(cfg client.Config) (*client.AgentClient, error)
}

// NewAgentCoreServiceWithOptions 构造可定制的核心管理服务。
func NewAgentCoreServiceWithOptions(
	agentHosts repository.AgentHostRepository,
	instances repository.AgentCoreInstanceRepository,
	switchLogs repository.AgentCoreSwitchLogRepository,
	templates repository.ConfigTemplateRepository,
	converters *template.ConverterRegistry,
	logger *slog.Logger,
	opts AgentCoreServiceOptions,
) AgentCoreService {
	if logger == nil {
		logger = slog.Default()
	}
	grpcPort := opts.GRPCPort
	if grpcPort == "" {
		grpcPort = ":19090"
	}
	factory := opts.ClientFactory
	if factory == nil {
		factory = client.NewAgentClient
	}
	return &agentCoreService{
		agentHosts:     agentHosts,
		instances:      instances,
		switchLogs:     switchLogs,
		templates:      templates,
		converters:     converters,
		logger:         logger,
		grpcTLS:        opts.GRPCTLS,
		grpcTimeout:    opts.Timeout,
		grpcKeepalive:  opts.Keepalive,
		grpcPort:       grpcPort,
		grpcClientFunc: factory,
	}
}

func (s *agentCoreService) GetCores(ctx context.Context, agentHostID int64) ([]*agentv1.CoreInfo, error) {
	// 透传到 Agent 获取核心能力信息
	client, _, err := s.buildAgentClient(ctx, agentHostID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	resp, err := client.GetCores(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Cores, nil
}

func (s *agentCoreService) GetInstances(ctx context.Context, agentHostID int64) ([]*repository.AgentCoreInstance, error) {
	if s.instances == nil {
		return nil, fmt.Errorf("agent core instance repository unavailable / 核心实例仓库不可用")
	}
	// 先校验 Agent Host 是否存在，避免查询出错
	if _, err := s.agentHosts.FindByID(ctx, agentHostID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.instances.ListByAgentHostID(ctx, agentHostID)
}

func (s *agentCoreService) CreateInstance(ctx context.Context, req CreateInstanceRequest) (*repository.AgentCoreInstance, error) {
	// 创建核心实例时会触发一次 SwitchCore RPC（FromInstanceId 为空）
	if req.AgentHostID == 0 {
		return nil, fmt.Errorf("agent host id required / 需要节点 ID")
	}
	if strings.TrimSpace(req.CoreType) == "" {
		return nil, fmt.Errorf("core type required / 需要核心类型")
	}
	if strings.TrimSpace(req.InstanceID) == "" {
		return nil, fmt.Errorf("instance id required / 需要实例 ID")
	}
	configJSON, configHash, templateID, err := s.resolveConfigJSON(ctx, req.ConfigTemplateID, req.ConfigJSON)
	if err != nil {
		return nil, err
	}

	client, host, err := s.buildAgentClient(ctx, req.AgentHostID)
	if err != nil {
		s.logger.Error("failed to build agent client", "error", err)
		return nil, err
	}
	defer client.Close()

	resp, err := client.SwitchCore(ctx, &agentv1.SwitchCoreRequest{
		FromInstanceId: "",
		ToCoreType:     strings.TrimSpace(req.CoreType),
		ConfigJson:     configJSON,
		SwitchId:       fmt.Sprintf("create-%d-%s", req.AgentHostID, req.InstanceID),
		ZeroDowntime:   false,
	})
	if err != nil {
		s.logger.Error("SwitchCore RPC failed", "error", err)
		return nil, err
	}
	if !resp.Success {
		errStr := firstNonEmpty(resp.Error, resp.Message)
		s.logger.Error("SwitchCore reported failure", "error", errStr)
		return nil, fmt.Errorf("create instance failed: %s", errStr)
	}

	instance := &repository.AgentCoreInstance{
		AgentHostID:     host.ID,
		InstanceID:      resp.NewInstanceId,
		CoreType:        strings.TrimSpace(req.CoreType),
		Status:          "running",
		ConfigTemplateID: templateID,
		ConfigHash:      configHash,
		ListenPorts:     []int{},
	}
	if s.instances != nil {
		if err := s.instances.Create(ctx, instance); err != nil {
			return nil, err
		}
	}
	return instance, nil
}

func (s *agentCoreService) DeleteInstance(ctx context.Context, agentHostID int64, instanceID string) error {
	if strings.TrimSpace(instanceID) == "" {
		return fmt.Errorf("instance id required / 需要实例 ID")
	}
	if s.instances == nil {
		return fmt.Errorf("agent core instance repository unavailable / 核心实例仓库不可用")
	}
	instance, err := s.instances.FindByInstanceID(ctx, agentHostID, instanceID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return s.instances.Delete(ctx, instance.ID)
}

func (s *agentCoreService) SwitchCore(ctx context.Context, req SwitchCoreRequest) (*SwitchResult, error) {
	// 核心切换的关键步骤：参数校验 → 解析配置 → gRPC 调用 → 记录日志 → 更新实例
	if req.AgentHostID == 0 {
		return nil, fmt.Errorf("agent host id required / 需要节点 ID")
	}
	if strings.TrimSpace(req.FromInstanceID) == "" {
		return nil, fmt.Errorf("from instance id required / 需要源实例 ID")
	}
	if strings.TrimSpace(req.ToCoreType) == "" {
		return nil, fmt.Errorf("target core type required / 需要目标核心类型")
	}

	configJSON, configHash, templateID, err := s.resolveConfigJSON(ctx, req.ConfigTemplateID, req.ConfigJSON)
	if err != nil {
		return nil, err
	}

	client, host, err := s.buildAgentClient(ctx, req.AgentHostID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	switchLogID, err := s.createSwitchLog(ctx, host.ID, req, switchStatusPending)
	if err != nil {
		return nil, err
	}

	// 状态先标记为进行中，便于前端及时展示
	if err := s.switchLogs.UpdateStatus(ctx, switchLogID, switchStatusRunning, "", nil); err != nil {
		s.logger.Error("failed to update switch log status",
			"switch_log_id", switchLogID,
			"error", err,
		)
	}

	switchID := req.SwitchID
	if switchID == "" {
		switchID = fmt.Sprintf("switch-%d-%d", host.ID, time.Now().Unix())
	}

	resp, err := client.SwitchCore(ctx, &agentv1.SwitchCoreRequest{
		FromInstanceId: strings.TrimSpace(req.FromInstanceID),
		ToCoreType:     strings.TrimSpace(req.ToCoreType),
		ConfigJson:     configJSON,
		SwitchId:       switchID,
		ListenPorts:    listenPortsToInt32(req.ListenPorts),
		ZeroDowntime:   ptrToBool(req.ZeroDowntime),
	})
	if err != nil {
		now := time.Now().Unix()
		if updateErr := s.switchLogs.UpdateStatus(ctx, switchLogID, switchStatusFailed, err.Error(), &now); updateErr != nil {
			s.logger.Error("failed to update switch log status",
				"switch_log_id", switchLogID,
				"error", updateErr,
			)
		}
		return &SwitchResult{
			Success:     false,
			Error:       err.Error(),
			SwitchLogID: switchLogID,
		}, nil
	}

	result := &SwitchResult{
		Success:        resp.Success,
		NewInstanceID:  resp.NewInstanceId,
		Message:        resp.Message,
		Error:          resp.Error,
		SwitchLogID:    switchLogID,
		FromInstanceID: req.FromInstanceID,
		ToCoreType:     req.ToCoreType,
	}
	completedAt := time.Now().Unix()
	if resp.Success {
		// 成功时更新日志并同步实例状态
		if updateErr := s.switchLogs.UpdateStatus(ctx, switchLogID, switchStatusCompleted, resp.Message, &completedAt); updateErr != nil {
			s.logger.Error("failed to update switch log status",
				"switch_log_id", switchLogID,
				"error", updateErr,
			)
		}
		result.CompletedAt = &completedAt
		if s.instances != nil {
			if err := s.updateInstancesAfterSwitch(ctx, host.ID, req.FromInstanceID, resp.NewInstanceId, req.ToCoreType, templateID, configHash); err != nil {
				s.logger.Error("failed to update core instances after switch",
					"agent_host_id", host.ID,
					"error", err,
				)
			}
		}
		return result, nil
	}

	if updateErr := s.switchLogs.UpdateStatus(ctx, switchLogID, switchStatusFailed, firstNonEmpty(resp.Error, resp.Message), &completedAt); updateErr != nil {
		s.logger.Error("failed to update switch log status",
			"switch_log_id", switchLogID,
			"error", updateErr,
		)
	}
	result.CompletedAt = &completedAt
	return result, nil
}

func (s *agentCoreService) GetSwitchLogs(ctx context.Context, filter SwitchLogFilter) ([]*repository.AgentCoreSwitchLog, int64, error) {
	if s.switchLogs == nil {
		return nil, 0, fmt.Errorf("agent core switch log repository unavailable / 核心切换日志仓库不可用")
	}
	if filter.AgentHostID == 0 {
		return nil, 0, fmt.Errorf("agent host id required / 需要节点 ID")
	}
	logs, err := s.switchLogs.List(ctx, repository.AgentCoreSwitchLogFilter{
		AgentHostID: filter.AgentHostID,
		Status:      filter.Status,
		StartAt:     filter.StartAt,
		EndAt:       filter.EndAt,
		Limit:       normalizeLimit(filter.Limit, 50),
		Offset:      normalizeOffset(filter.Offset),
	})
	if err != nil {
		return nil, 0, err
	}
	count, err := s.switchLogs.Count(ctx, repository.AgentCoreSwitchLogFilter{
		AgentHostID: filter.AgentHostID,
		Status:      filter.Status,
		StartAt:     filter.StartAt,
		EndAt:       filter.EndAt,
		Limit:       normalizeLimit(filter.Limit, 50),
		Offset:      normalizeOffset(filter.Offset),
	})
	if err != nil {
		return nil, 0, err
	}
	return logs, count, nil
}

func (s *agentCoreService) ConvertConfig(ctx context.Context, req ConvertRequest) (*ConvertResult, error) {
	// 调用转换器将配置从源核心转换为目标核心格式
	if strings.TrimSpace(req.SourceCore) == "" {
		return nil, fmt.Errorf("source core required / 需要源核心")
	}
	if strings.TrimSpace(req.TargetCore) == "" {
		return nil, fmt.Errorf("target core required / 需要目标核心")
	}
	if len(req.ConfigJSON) == 0 {
		return nil, fmt.Errorf("config json required / 需要配置 JSON")
	}
	if s.converters == nil {
		return nil, fmt.Errorf("converter registry unavailable / 转换器注册表不可用")
	}

	inbounds, err := s.converters.Parse(req.ConfigJSON, req.SourceCore)
	if err != nil {
		return nil, err
	}
	converted, err := s.converters.Convert(inbounds, req.TargetCore)
	if err != nil {
		return nil, err
	}
	return &ConvertResult{ConfigJSON: converted}, nil
}

func (s *agentCoreService) buildAgentClient(ctx context.Context, agentHostID int64) (*client.AgentClient, *repository.AgentHost, error) {
	// 读取 Agent 地址与 Token，构造 gRPC 客户端
	if s.agentHosts == nil {
		return nil, nil, fmt.Errorf("agent host repository unavailable / 节点仓库不可用")
	}
	host, err := s.agentHosts.FindByID(ctx, agentHostID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	address := host.Host
	if !strings.Contains(address, ":") {
		address = address + s.grpcPort
	}

	grpcCfg := client.Config{
		Address:   address,
		Token:     host.Token,
		TLS:       s.grpcTLS,
		Keepalive: s.grpcKeepalive,
		Timeout:   s.grpcTimeout,
	}

	cli, err := s.grpcClientFunc(grpcCfg)
	if err != nil {
		return nil, nil, err
	}
	return cli, host, nil
}

func (s *agentCoreService) resolveConfigJSON(ctx context.Context, templateID int64, configJSON json.RawMessage) ([]byte, string, *int64, error) {
	// 配置优先级：显式 JSON > 模板 ID
	var payload []byte
	var tplID *int64
	if len(configJSON) > 0 {
		payload = configJSON
	} else if templateID > 0 {
		if s.templates == nil {
			return nil, "", nil, fmt.Errorf("config template repository unavailable / 配置模板仓库不可用")
		}
		tpl, err := s.templates.FindByID(ctx, templateID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, "", nil, ErrNotFound
			}
			return nil, "", nil, err
		}
		payload = []byte(tpl.Content)
		tplID = &tpl.ID
	} else {
		return nil, "", nil, fmt.Errorf("config json required / 需要配置 JSON")
	}

	if !json.Valid(payload) {
		return nil, "", nil, fmt.Errorf("config json is invalid / 配置 JSON 无效")
	}
	hash := md5.Sum(payload)
	return payload, hex.EncodeToString(hash[:]), tplID, nil
}

func (s *agentCoreService) createSwitchLog(ctx context.Context, agentHostID int64, req SwitchCoreRequest, status string) (int64, error) {
	// 初始化切换日志记录
	if s.switchLogs == nil {
		return 0, fmt.Errorf("agent core switch log repository unavailable / 核心切换日志仓库不可用")
	}
	log := &repository.AgentCoreSwitchLog{
		AgentHostID:  agentHostID,
		Status:       status,
		ToInstanceID: "",
		ToCoreType:   strings.TrimSpace(req.ToCoreType),
		OperatorID:   req.OperatorID,
	}
	if req.FromInstanceID != "" {
		value := req.FromInstanceID
		log.FromInstanceID = &value
	}
	if req.FromInstanceID != "" && req.ToCoreType != "" {
		log.FromCoreType = nil
	}

	if err := s.switchLogs.Create(ctx, log); err != nil {
		return 0, err
	}
	return log.ID, nil
}

func (s *agentCoreService) updateInstancesAfterSwitch(ctx context.Context, agentHostID int64, fromInstanceID, newInstanceID, toCoreType string, templateID *int64, configHash string) error {
	// 切换成功后：停止旧实例、创建新实例记录
	if s.instances == nil {
		return nil
	}
	if fromInstanceID != "" {
		if inst, err := s.instances.FindByInstanceID(ctx, agentHostID, fromInstanceID); err == nil {
			inst.Status = "stopped"
			inst.ErrorMessage = ""
			_ = s.instances.Update(ctx, inst)
		}
	}

	if newInstanceID == "" {
		return nil
	}
	instance := &repository.AgentCoreInstance{
		AgentHostID:     agentHostID,
		InstanceID:      newInstanceID,
		CoreType:        strings.TrimSpace(toCoreType),
		Status:          "running",
		ConfigTemplateID: templateID,
		ConfigHash:      configHash,
		ListenPorts:     []int{},
		LastHeartbeatAt: nil,
		ErrorMessage:    "",
	}
	if err := s.instances.Create(ctx, instance); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}
	return nil
}

func normalizeLimit(limit int, fallback int) int {
	// 限制分页大小并提供默认值
	if limit <= 0 {
		return fallback
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func normalizeOffset(offset int) int {
	// 负数偏移直接归零
	if offset < 0 {
		return 0
	}
	return offset
}

func listenPortsToInt32(ports []int) []int32 {
	if len(ports) == 0 {
		return nil
	}
	result := make([]int32, 0, len(ports))
	for _, port := range ports {
		if port <= 0 {
			continue
		}
		result = append(result, int32(port))
	}
	return result
}

func ptrToBool(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

