// 文件路径: internal/repository/interfaces.go
// 模块说明: 这是 internal 模块里的 interfaces 逻辑，下面的注释会用非常通俗的中文帮你理解每一步。
package repository

import "context"

// Store 暴露每个聚合根对应的仓储接口。
type Store interface {
	Users() UserRepository
	Settings() SettingRepository
	InviteCodes() InviteCodeRepository
	Plugins() PluginRepository
	Plans() PlanRepository
	LoginLogs() LoginLogRepository
	Tokens() TokenRepository
	Payments() PaymentRepository
	Servers() ServerRepository
	ServerGroups() ServerGroupRepository
	ServerRoutes() ServerRouteRepository
	StatUsers() StatUserRepository
	StatServers() StatServerRepository
	Notices() NoticeRepository
	Knowledge() KnowledgeRepository
	SubscriptionLogs() SubscriptionLogRepository
	AgentHosts() AgentHostRepository
	ConfigTemplates() ConfigTemplateRepository
	UserTraffic() UserTrafficRepository
	ShortLinks() ShortLinkRepository
	SubscriptionTemplates() SubscriptionTemplateRepository
	ForwardingRules() ForwardingRuleRepository
	UserNoticeReads() UserNoticeReadsRepository
	AgentCoreInstances() AgentCoreInstanceRepository
	AgentCoreSwitchLogs() AgentCoreSwitchLogRepository
	AccessLogs() AccessLogRepository
}

// UserRepository 定义用户相关数据访问方法。
type UserRepository interface {
	FindByID(ctx context.Context, id int64) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByToken(ctx context.Context, token string) (*User, error)
	Save(ctx context.Context, user *User) error
	Create(ctx context.Context, user *User) (*User, error)
	HasAdmin(ctx context.Context) (bool, error)
	ActiveCountByPlan(ctx context.Context, planID int64, nowUnix int64) (int64, error)
	AdjustBalance(ctx context.Context, userID int64, deltaCents int64) (bool, error)
	IncrementTraffic(ctx context.Context, userID int64, uploadDelta, downloadDelta int64) error
	ListActiveForGroups(ctx context.Context, groupIDs []int64, nowUnix int64) ([]*NodeUser, error)
	PlanCounts(ctx context.Context, planIDs []int64, nowUnix int64) (map[int64]PlanUserCount, error)
	Search(ctx context.Context, filter UserSearchFilter) ([]*User, error)
	CountFiltered(ctx context.Context, filter UserSearchFilter) (int64, error)
	Count(ctx context.Context) (int64, error)
	CountActive(ctx context.Context, nowUnix int64) (int64, error)
	CountCreatedBetween(ctx context.Context, startUnix, endUnix int64) (int64, error)
	SetTrafficExceeded(ctx context.Context, userID int64, exceeded bool) error
	GetExceededUserIDs(ctx context.Context) ([]int64, error)
	Delete(ctx context.Context, id int64) error
}

// SettingRepository 处理系统配置的存取。
type SettingRepository interface {
	Get(ctx context.Context, key string) (*Setting, error)
	Upsert(ctx context.Context, setting *Setting) error
	List(ctx context.Context) ([]Setting, error)
	ListByCategory(ctx context.Context, category string) ([]Setting, error)
}

// InviteCodeRepository 管理邀请码相关操作。
type InviteCodeRepository interface {
	IncrementPV(ctx context.Context, code string) error
	FindByCode(ctx context.Context, code string) (*InviteCode, error)
	MarkUsed(ctx context.Context, id int64) error
	CreateBatch(ctx context.Context, codes []*InviteCode) error
	CountByStatus(ctx context.Context, status int) (int64, error)
	CountByUser(ctx context.Context, userID int64) (int64, error)
	List(ctx context.Context, limit, offset int) ([]*InviteCode, error)
	CountAll(ctx context.Context) (int64, error)
}

// PluginRepository 提供插件元数据与配置访问。
type PluginRepository interface {
	FindEnabledByCode(ctx context.Context, code string) (*Plugin, error)
}

// PlanRepository 管理订阅套餐相关数据。
type PlanRepository interface {
	ListVisible(ctx context.Context) ([]*Plan, error)
	ListAll(ctx context.Context) ([]*Plan, error)
	FindByID(ctx context.Context, id int64) (*Plan, error)
	Create(ctx context.Context, plan *Plan) (*Plan, error)
	Update(ctx context.Context, plan *Plan) error
	Delete(ctx context.Context, id int64) error
	Sort(ctx context.Context, ids []int64, updatedAt int64) error
	BindGroups(ctx context.Context, planID int64, groupIDs []int64) error
	UnbindGroups(ctx context.Context, planID int64) error
	ReplaceGroups(ctx context.Context, planID int64, groupIDs []int64) error
	UpdateWithGroups(ctx context.Context, plan *Plan, groupIDs []int64) error
	GetGroups(ctx context.Context, planID int64) ([]int64, error)
}

// PaymentRepository 管理支付方式相关数据。
type PaymentRepository interface {
	ListEnabled(ctx context.Context) ([]*Payment, error)
}

// ServerRepository 管理节点相关数据。
type ServerRepository interface {
	FindAllVisible(ctx context.Context) ([]*Server, error)
	FindByGroupIDs(ctx context.Context, groupIDs []int64) ([]*Server, error)
	FindByIdentifier(ctx context.Context, identifier string, nodeType string) (*Server, error)
	FindByID(ctx context.Context, id int64) (*Server, error)
	FindByAgentHostID(ctx context.Context, agentHostID int64) ([]*Server, error)
	ListAll(ctx context.Context) ([]*Server, error)
	Create(ctx context.Context, server *Server) error
	Update(ctx context.Context, server *Server) error
	UpdateHeartbeat(ctx context.Context, id int64, heartbeatAt int64) error
	Delete(ctx context.Context, id int64) error
	Count(ctx context.Context) (int64, error)
}

// ServerGroupRepository 提供节点分组信息。
type ServerGroupRepository interface {
	List(ctx context.Context) ([]*ServerGroup, error)
}

// ServerRouteRepository 提供节点路由信息。
type ServerRouteRepository interface {
	List(ctx context.Context) ([]*ServerRoute, error)
	FindByIDs(ctx context.Context, ids []int64) ([]*ServerRoute, error)
}

// StatUserRepository 管理用户流量聚合统计。
type StatUserRepository interface {
	Upsert(ctx context.Context, record StatUserRecord) error
	ListByRecord(ctx context.Context, recordType int, recordAt int64, agentHostID *int64, limit int) ([]StatUserRecord, error)
	ListByUserSince(ctx context.Context, userID int64, since int64, limit int) ([]StatUserRecord, error)
	SumByRange(ctx context.Context, filter StatUserSumFilter) (StatUserSumResult, error)
	TopByRange(ctx context.Context, filter StatUserTopFilter) ([]StatUserAggregate, error)

	// 多节点聚合查询
	ListByAgentHost(ctx context.Context, agentHostID int64, recordType int, since int64, limit int) ([]StatUserRecord, error)
	SumByAgentHost(ctx context.Context, agentHostID int64, recordType int, startAt, endAt int64) (StatUserSumResult, error)
}

// NoticeRepository 管理站点公告数据。
type NoticeRepository interface {
	List(ctx context.Context) ([]*Notice, error)
	FindByID(ctx context.Context, id int64) (*Notice, error)
	Create(ctx context.Context, notice *Notice) (*Notice, error)
	Update(ctx context.Context, notice *Notice) error
	Delete(ctx context.Context, id int64) error
	Sort(ctx context.Context, ids []int64, updatedAt int64) error
}

// UserNoticeReadsRepository 管理用户已读公告记录。
type UserNoticeReadsRepository interface {
	// MarkRead 记录用户已读公告（幂等）
	MarkRead(ctx context.Context, userID, noticeID int64) error

	// HasRead 判断用户是否已读该公告
	HasRead(ctx context.Context, userID, noticeID int64) (bool, error)

	// GetUnreadPopupNoticeIDs 返回未读弹窗公告 ID 列表
	GetUnreadPopupNoticeIDs(ctx context.Context, userID int64) ([]int64, error)
}

// KnowledgeRepository 管理知识库条目。
type KnowledgeRepository interface {
	List(ctx context.Context) ([]*Knowledge, error)
	FindByID(ctx context.Context, id int64) (*Knowledge, error)
	Create(ctx context.Context, knowledge *Knowledge) (*Knowledge, error)
	Update(ctx context.Context, knowledge *Knowledge) error
	Delete(ctx context.Context, id int64) error
	Sort(ctx context.Context, ids []int64, updatedAt int64) error
	Categories(ctx context.Context) ([]string, error)
	ListVisible(ctx context.Context, filter KnowledgeVisibleFilter) ([]*Knowledge, error)
}

// LoginLogRepository 保存登录日志。
type LoginLogRepository interface {
	Create(ctx context.Context, log *LoginLog) error
}

// TokenRepository 管理访问/刷新令牌。
type TokenRepository interface {
	Create(ctx context.Context, token *AccessToken) (*AccessToken, error)
	FindByRefreshToken(ctx context.Context, refreshToken string) (*AccessToken, error)
	DeleteByRefreshToken(ctx context.Context, refreshToken string) error
	DeleteByUser(ctx context.Context, userID int64) error
}

// SubscriptionLogRepository 记录订阅访问日志。
type SubscriptionLogRepository interface {
	Log(ctx context.Context, log *SubscriptionLog) error
	GetRecentLogs(ctx context.Context, userID int64, limit int) ([]*SubscriptionLog, error)
}

// StatServerRepository 管理节点维度统计。
type StatServerRepository interface {
	Upsert(ctx context.Context, record StatServerRecord) error
	ListByServer(ctx context.Context, serverID int64, recordType int, since int64, limit int) ([]StatServerRecord, error)
	SumByRange(ctx context.Context, filter StatServerSumFilter) (StatServerSumResult, error)
	TopByRange(ctx context.Context, filter StatServerTopFilter) ([]StatServerAggregate, error)
}

// StatServerSumFilter 定义节点流量汇总筛选条件。
type StatServerSumFilter struct {
	ServerID   *int64 // nil = all servers
	RecordType int
	StartAt    int64
	EndAt      int64
}

// StatServerTopFilter 定义节点流量排行筛选条件。
type StatServerTopFilter struct {
	RecordType int
	StartAt    int64
	EndAt      int64
	Limit      int
}

// AgentHostRepository 管理 Agent 主机信息。
type AgentHostRepository interface {
	// CRUD 操作
	Create(ctx context.Context, host *AgentHost) error
	FindByID(ctx context.Context, id int64) (*AgentHost, error)
	FindByHost(ctx context.Context, host string) (*AgentHost, error)
	FindByToken(ctx context.Context, token string) (*AgentHost, error)
	Update(ctx context.Context, host *AgentHost) error
	Delete(ctx context.Context, id int64) error
	ListAll(ctx context.Context) ([]*AgentHost, error)

	// 状态更新
	UpdateStatus(ctx context.Context, id int64, status int, heartbeatAt int64) error
	UpdateMetrics(ctx context.Context, id int64, metrics AgentHostMetrics) error
	UpdateCapabilities(ctx context.Context, id int64, coreVersion string, capabilities, buildTags []string) error

	// 统计查询
	Count(ctx context.Context) (int64, error)
	CountOnline(ctx context.Context) (int64, error)
}

// ConfigTemplateRepository 管理配置模板数据。
type ConfigTemplateRepository interface {
	Create(ctx context.Context, tpl *ConfigTemplate) error
	Update(ctx context.Context, tpl *ConfigTemplate) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*ConfigTemplate, error)
	ListAll(ctx context.Context) ([]*ConfigTemplate, error)
}

// AgentHostMetrics contains real-time metrics reported by an agent.
type AgentHostMetrics struct {
	CPUTotal      float64
	CPUUsed       float64
	MemTotal      int64
	MemUsed       int64
	DiskTotal     int64
	DiskUsed      int64
	UploadTotal   int64
	DownloadTotal int64
}

// ServerClientConfigRepository 管理客户端订阅配置。
type ServerClientConfigRepository interface {
	// Create 插入新的客户端订阅配置
	Create(ctx context.Context, cfg *ServerClientConfig) error

	// FindByServerID 获取指定节点的全部订阅配置
	FindByServerID(ctx context.Context, serverID int64) ([]*ServerClientConfig, error)

	// FindByServerIDAndFormat 获取指定节点 + 格式的订阅配置
	FindByServerIDAndFormat(ctx context.Context, serverID int64, format string) (*ServerClientConfig, error)

	// Upsert 新增或更新订阅配置
	Upsert(ctx context.Context, cfg *ServerClientConfig) error

	// DeleteByServerID 删除指定节点的全部订阅配置
	DeleteByServerID(ctx context.Context, serverID int64) error

	// DeleteByServerIDAndFormat 删除指定节点的某种格式订阅配置
	DeleteByServerIDAndFormat(ctx context.Context, serverID int64, format string) error
}

// UserTrafficRepository 管理用户流量周期与节点选择。
type UserTrafficRepository interface {
	// 节点选择相关操作
	AddServerSelection(ctx context.Context, userID, serverID int64) error
	RemoveServerSelection(ctx context.Context, userID, serverID int64) error
	GetUserServerIDs(ctx context.Context, userID int64) ([]int64, error)
	ClearUserSelections(ctx context.Context, userID int64) error
	ReplaceUserSelections(ctx context.Context, userID int64, serverIDs []int64) error

	// 流量周期相关操作
	GetCurrentPeriod(ctx context.Context, userID int64) (*UserTrafficPeriod, error)
	CreatePeriod(ctx context.Context, period *UserTrafficPeriod) error
	IncrementPeriodTraffic(ctx context.Context, userID int64, uploadDelta, downloadDelta int64) error
	MarkPeriodExceeded(ctx context.Context, userID int64, periodStart int64) error
	GetExpiredPeriodUserIDs(ctx context.Context, nowUnix int64) ([]int64, error)

	// 查询相关操作
	GetExceededUserIDs(ctx context.Context) ([]int64, error)
	GetUserTrafficStats(ctx context.Context, userID int64) (*UserTrafficStats, error)
}

// ShortLinkRepository 管理短链接映射。
type ShortLinkRepository interface {
	// Create 插入新的短链接
	Create(ctx context.Context, link *ShortLink) error

	// FindByCode 按 code 查询短链接
	FindByCode(ctx context.Context, code string) (*ShortLink, error)

	// FindByID 按 ID 查询短链接
	FindByID(ctx context.Context, id int64) (*ShortLink, error)

	// FindByUserID 查询用户创建的所有短链接
	FindByUserID(ctx context.Context, userID int64) ([]*ShortLink, error)

	// Update 更新短链接记录
	Update(ctx context.Context, link *ShortLink) error

	// Delete 删除指定 ID 的短链接
	Delete(ctx context.Context, id int64) error

	// DeleteByUserID 删除用户所有短链接
	DeleteByUserID(ctx context.Context, userID int64) error

	// IncrementAccessCount 增加访问次数并更新最近访问时间
	IncrementAccessCount(ctx context.Context, id int64, accessTime int64) error

	// CodeExists 判断短码是否已存在
	CodeExists(ctx context.Context, code string) (bool, error)
}

// SubscriptionTemplateRepository 管理订阅模板。
type SubscriptionTemplateRepository interface {
	// Create 插入新的订阅模板
	Create(ctx context.Context, tpl *SubscriptionTemplate) error

	// FindByID 按 ID 查询订阅模板
	FindByID(ctx context.Context, id int64) (*SubscriptionTemplate, error)

	// FindDefaultByType 获取指定类型的默认模板 (clash, singbox 等)
	FindDefaultByType(ctx context.Context, templateType string) (*SubscriptionTemplate, error)

	// ListByType 获取指定类型的全部模板
	ListByType(ctx context.Context, templateType string) ([]*SubscriptionTemplate, error)

	// ListPublic 获取公开模板列表
	ListPublic(ctx context.Context) ([]*SubscriptionTemplate, error)

	// Update 更新订阅模板
	Update(ctx context.Context, tpl *SubscriptionTemplate) error

	// Delete 删除指定 ID 的订阅模板
	Delete(ctx context.Context, id int64) error

	// SetDefault 将模板设为默认
	SetDefault(ctx context.Context, id int64) error
}

// ForwardingRuleRepository 管理端口转发规则。
type ForwardingRuleRepository interface {
	// CRUD 操作
	Create(ctx context.Context, rule *ForwardingRule) error
	Update(ctx context.Context, rule *ForwardingRule) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*ForwardingRule, error)

	// 查询操作
	ListByAgentHostID(ctx context.Context, agentHostID int64) ([]*ForwardingRule, error)
	ListEnabledByAgentHostID(ctx context.Context, agentHostID int64) ([]*ForwardingRule, error)

	// 版本管理
	GetMaxVersion(ctx context.Context, agentHostID int64) (int64, error)

	// 冲突检测
	CheckPortConflict(ctx context.Context, agentHostID int64, listenPort int, protocol string, excludeID int64) (bool, error)
}

// ForwardingRuleLogFilter 定义转发规则日志筛选条件。
type ForwardingRuleLogFilter struct {
	AgentHostID int64
	RuleID      *int64
	StartAt     *int64
	EndAt       *int64
	Limit       int
	Offset      int
}

// ForwardingRuleLogRepository 管理转发规则审计日志。
type ForwardingRuleLogRepository interface {
	// 新增审计日志
	Create(ctx context.Context, log *ForwardingRuleLog) error

	// 按筛选条件返回日志列表
	List(ctx context.Context, filter ForwardingRuleLogFilter) ([]*ForwardingRuleLog, error)

	// 统计筛选条件下的日志总数
	Count(ctx context.Context, filter ForwardingRuleLogFilter) (int64, error)

	// 查询指定规则的日志列表
	ListByRuleID(ctx context.Context, ruleID int64, limit int) ([]*ForwardingRuleLog, error)
}

// AgentCoreInstanceRepository 管理核心实例记录。
type AgentCoreInstanceRepository interface {
	Create(ctx context.Context, instance *AgentCoreInstance) error
	Update(ctx context.Context, instance *AgentCoreInstance) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*AgentCoreInstance, error)
	FindByInstanceID(ctx context.Context, agentHostID int64, instanceID string) (*AgentCoreInstance, error)
	ListByAgentHostID(ctx context.Context, agentHostID int64) ([]*AgentCoreInstance, error)
	UpdateHeartbeat(ctx context.Context, agentHostID int64, instanceID string, heartbeatAt int64) error
}

// AgentCoreSwitchLogFilter 定义核心切换日志筛选条件。
type AgentCoreSwitchLogFilter struct {
	AgentHostID int64
	Status      *string
	StartAt     *int64
	EndAt       *int64
	Limit       int
	Offset      int
}

// AgentCoreSwitchLogRepository 管理核心切换审计日志。
type AgentCoreSwitchLogRepository interface {
	Create(ctx context.Context, log *AgentCoreSwitchLog) error
	UpdateStatus(ctx context.Context, id int64, status string, detail string, completedAt *int64) error
	List(ctx context.Context, filter AgentCoreSwitchLogFilter) ([]*AgentCoreSwitchLog, error)
	Count(ctx context.Context, filter AgentCoreSwitchLogFilter) (int64, error)
}

// AccessLogRepository manages access log data.
type AccessLogRepository interface {
	Create(ctx context.Context, log *AccessLog) error
	BatchCreate(ctx context.Context, logs []*AccessLog) error
	List(ctx context.Context, filter AccessLogFilter) ([]*AccessLog, error)
	Count(ctx context.Context, filter AccessLogFilter) (int64, error)
	DeleteByRetentionDays(ctx context.Context, days int) (int64, error)
	GetStats(ctx context.Context, filter AccessLogFilter) (*AccessLogStats, error)
}
