// 文件路径: internal/repository/types.go
// 模块说明: 这是 internal 模块里的 types 逻辑，下面的注释会用非常通俗的中文帮你理解每一步。
package repository

import "encoding/json"

// User represents a subset of the v2_user columns migrated to SQLite.
type User struct {
	ID                int64
	UUID              string
	Token             string
	Username          string
	Email             string
	Password          string
	PasswordAlgo      string
	PasswordSalt      string
	BalanceCents      int64
	PlanID            int64
	GroupID           int64
	ExpiredAt         int64
	U                 int64
	D                 int64
	TransferEnable    int64
	SpeedLimit        *int64
	DeviceLimit       *int64
	CommissionBalance float64
	IsAdmin           bool
	Status            int
	Banned            bool
	TrafficExceeded   bool
	TelegramID        int64
	InviteUserID      int64
	InviteLimit       int64
	LastLoginAt       int64
	Remarks           string
	Tags              []string
	CreatedAt         int64
	UpdatedAt         int64
}

// NodeUser represents the limited subset of user columns shared with nodes.
type NodeUser struct {
	ID          int64
	UUID        string
	Email       string
	SpeedLimit  *int64
	DeviceLimit *int64
}

// PlanUserCount aggregates user totals per plan for admin analytics.
type PlanUserCount struct {
	Total  int64
	Active int64
}

// AccessToken stores refresh/access session metadata.
type AccessToken struct {
	ID               int64
	UserID           int64
	Token            string
	RefreshToken     string
	ExpiresAt        int64
	RefreshExpiresAt int64
	IP               string
	UserAgent        string
	Revoked          bool
	CreatedAt        int64
	UpdatedAt        int64
}

// LoginLog captures a single login attempt for auditing purposes.
type LoginLog struct {
	ID        int64
	UserID    *int64
	Email     string
	IP        string
	UserAgent string
	Success   bool
	Reason    string
	CreatedAt int64
	UpdatedAt int64
}

// Setting mirrors the admin settings KV pairs.
type Setting struct {
	Key       string
	Value     string
	Category  string
	UpdatedAt int64
}

// InviteCode mirrors the v2_invite_code table.
type InviteCode struct {
	ID        int64
	UserID    int64
	Code      string
	Status    int
	PV        int64
	Limit     int64
	ExpireAt  int64
	CreatedAt int64
	UpdatedAt int64
}

// Plugin models enabled plugin metadata and configuration payloads.
type Plugin struct {
	ID        int64
	Code      string
	Type      string
	IsEnabled bool
	Config    string
}

// Plan models the plans table for subscription listings.
type Plan struct {
	ID                 int64
	GroupID            *int64
	Name               string
	Prices             map[string]float64
	Sell               bool
	TransferEnable     int64
	SpeedLimit         *int64
	DeviceLimit        *int64
	Show               bool
	Renew              bool
	Content            string
	Tags               []string
	ResetTrafficMethod *int64
	CapacityLimit      *int64
	InviteLimit        *int64
	Sort               int64
	CreatedAt          int64
	UpdatedAt          int64
}

// Payment mirrors enabled payment channels users can choose during checkout.
type Payment struct {
	ID                 int64
	UUID               string
	PaymentCode        string
	Name               string
	Icon               *string
	Config             string
	NotifyDomain       *string
	HandlingFeeFixed   *int64
	HandlingFeePercent *float64
	Enable             bool
	Sort               *int64
	CreatedAt          int64
	UpdatedAt          int64
}

// ServerGroup represents a logical grouping of servers.
type ServerGroup struct {
	ID        int64
	Name      string
	Type      string
	Sort      int64
	CreatedAt int64
	UpdatedAt int64
}

// ServerRoute captures custom routing rules applied to servers.
type ServerRoute struct {
	ID          int64
	Remarks     string
	Match       json.RawMessage
	Action      string
	ActionValue string
	CreatedAt   int64
	UpdatedAt   int64
}

// Server describes a single proxy/server entry exposed to clients.
type Server struct {
	ID              int64
	Code            string
	GroupID         int64
	RouteID         int64
	ParentID        int64
	AgentHostID     int64 // 关联的 Agent 主机 ID
	Tags            json.RawMessage
	Name            string
	Rate            string
	Host            string
	Port            int
	ServerPort      int
	Cipher          string
	Obfs            string
	ObfsSettings    json.RawMessage
	Show            int
	Sort            int64
	Status          int
	Type            string
	Settings        json.RawMessage
	LastHeartbeatAt int64
	CreatedAt       int64
	UpdatedAt       int64
}

// AgentHost represents a physical server where Agents are deployed.
type AgentHost struct {
	ID              int64
	Name            string   // 服务器名称
	Host            string   // 服务器 IP 或域名
	Token           string   // Agent 认证令牌
	Status          int      // 0: 离线, 1: 在线, 2: 警告
	TemplateID      int64    // Config Template ID
	CoreVersion     string   // 核心版本 (如 "1.10.0")
	Capabilities    []string // 支持的能力 (如 ["reality", "multiplex"])
	BuildTags       []string // 构建标签 (如 ["with_v2ray_api"])
	CPUTotal        float64  // CPU 核心数
	CPUUsed         float64  // CPU 使用率 (%)
	MemTotal        int64    // 内存总量 (bytes)
	MemUsed         int64    // 内存使用量 (bytes)
	DiskTotal       int64    // 磁盘总量 (bytes)
	DiskUsed        int64    // 磁盘使用量 (bytes)
	UploadTotal     int64    // 累计上传流量 (bytes)
	DownloadTotal   int64    // 累计下载流量 (bytes)
	LastHeartbeatAt int64    // 最后心跳时间
	CreatedAt       int64
	UpdatedAt       int64
}

// ConfigTemplate defines a configuration template for agents.
type ConfigTemplate struct {
	ID              int64
	Name            string
	Type            string   // sing-box, xray, etc.
	Content         string   // Template content (Go text/template format)
	Description     string   // Human-readable description
	MinVersion      string   // Minimum core version required (e.g., "1.8.0")
	Capabilities    []string // Required capabilities (e.g., ["reality", "multiplex"])
	SchemaVersion   int      // Template format version
	IsValid         bool     // Cached validation status
	ValidationError string   // Last validation error message
	CreatedAt       int64
	UpdatedAt       int64
}

// Notice mirrors announcements shown to users/admins.
type Notice struct {
	ID        int64
	Sort      int64
	Title     string
	Content   string
	ImgURL    string
	Tags      []string
	Show      bool
	Popup     bool
	CreatedAt int64
	UpdatedAt int64
}

// Knowledge mirrors v2_knowledge articles exposed to users/admins.
type Knowledge struct {
	ID        int64
	Language  string
	Category  string
	Title     string
	Body      string
	Sort      int64
	Show      bool
	CreatedAt int64
	UpdatedAt int64
}

// KnowledgeVisibleFilter narrows which knowledge entries are exposed to users.
type KnowledgeVisibleFilter struct {
	Language string
	Keyword  string
}

// StatUserRecord captures aggregated traffic usage per user per interval.
type StatUserRecord struct {
	UserID      int64
	AgentHostID int64   // Source agent host ID for multi-node aggregation
	ServerRate  float64
	RecordAt    int64
	RecordType  int // 0: hourly, 1: daily, 2: monthly
	Upload      int64
	Download    int64
	CreatedAt   int64
	UpdatedAt   int64
}

// StatUserSumResult sums upload/download amounts.
type StatUserSumResult struct {
	Upload   int64
	Download int64
}

// StatUserAggregate stores ranked traffic totals per user.
type StatUserAggregate struct {
	UserID   int64
	Upload   int64
	Download int64
}

// SubscriptionLog represents an access log for subscription endpoints.
type SubscriptionLog struct {
	ID        int64
	UserID    int64
	IP        string
	UserAgent string
	Type      string
	URL       string
	CreatedAt int64
}

// StatServerRecord captures aggregated node-level statistics per interval.
type StatServerRecord struct {
	ID          int64
	ServerID    int64
	RecordAt    int64
	RecordType  int // 0: hourly, 1: daily
	Upload      int64
	Download    int64
	CPUAvg      float64
	MemUsed     int64
	MemTotal    int64
	DiskUsed    int64
	DiskTotal   int64
	OnlineUsers int64
	CreatedAt   int64
	UpdatedAt   int64
}

// StatServerSumResult sums upload/download amounts for servers.
type StatServerSumResult struct {
	Upload   int64
	Download int64
}

// StatServerAggregate stores ranked traffic totals per server.
type StatServerAggregate struct {
	ServerID int64
	Upload   int64
	Download int64
}

// ServerClientConfig stores client-side configuration for a server/protocol.
type ServerClientConfig struct {
	ID          int64
	ServerID    int64  // FK to servers table
	Format      string // v2rayn, clash, singbox-pc, singbox-phone, etc.
	Content     string // Full config content for this format
	ContentHash string // MD5 hash for change detection
	CreatedAt   int64
	UpdatedAt   int64
}

// UserServerSelection represents a user's selection of a specific server node.
type UserServerSelection struct {
	ID        int64
	UserID    int64
	ServerID  int64
	CreatedAt int64
}

// UserTrafficPeriod tracks user traffic usage within a billing period.
type UserTrafficPeriod struct {
	ID            int64
	UserID        int64
	PeriodStart   int64 // Unix timestamp of period start (1st of month)
	PeriodEnd     int64 // Unix timestamp of period end (1st of next month)
	UploadBytes   int64
	DownloadBytes int64
	QuotaBytes    int64 // Traffic quota for this period
	Exceeded      bool  // True if user exceeded quota
	CreatedAt     int64
	UpdatedAt     int64
}

// UserTrafficStats provides a summary of user's traffic usage.
type UserTrafficStats struct {
	PeriodStart   int64
	PeriodEnd     int64
	UploadBytes   int64
	DownloadBytes int64
	TotalBytes    int64
	QuotaBytes    int64
	UsedPercent   float64
	Exceeded      bool
}

// ShortLink represents a short URL mapping for subscription links.
type ShortLink struct {
	ID             int64
	Code           string // Short code (e.g., "abc123")
	UserID         int64
	TargetPath     string // Target path (default: /api/v1/client/subscribe)
	CustomParams   string // Custom query parameters (JSON)
	ExpiresAt      int64  // Optional expiration timestamp
	AccessCount    int64  // Number of times accessed
	LastAccessedAt int64  // Last access timestamp
	CreatedAt      int64
	UpdatedAt      int64
}

// SubscriptionTemplate represents a customizable template for subscription output.
type SubscriptionTemplate struct {
	ID          int64
	Name        string // Template display name
	Description string // Human-readable description
	Type        string // clash, singbox, surge, general
	Content     string // Template content (Go text/template or raw config)
	IsDefault   bool   // Whether this is the default template for its type
	IsPublic    bool   // Whether this template is visible to users
	SortOrder   int    // Display order
	CreatedAt   int64
	UpdatedAt   int64
}

// ForwardingRule represents a nftables port forwarding rule.
type ForwardingRule struct {
	ID            int64
	AgentHostID   int64  // 关联的 Agent 主机 ID
	Name          string // 规则名称
	Protocol      string // tcp/udp/both
	ListenPort    int    // 本地监听端口
	TargetAddress string // 目标地址 (IP 或域名)
	TargetPort    int    // 目标端口
	Enabled       bool   // 是否启用
	Priority      int    // 优先级（越小越优先）
	Remark        string // 备注
	Version       int64  // 规则版本
	CreatedAt     int64
	UpdatedAt     int64
}

// ForwardingRuleLog records audit logs for forwarding rule changes.
type ForwardingRuleLog struct {
	ID          int64
	RuleID      *int64 // 关联的规则 ID (可为空，规则删除后保留日志)
	AgentHostID int64  // 关联的 Agent 主机 ID
	Action      string // 操作类型: create, update, delete, enable, disable, apply_success, apply_failed
	OperatorID  *int64 // 操作者 ID (可为空，系统操作时)
	Detail      string // 操作详情 (JSON 格式)
	CreatedAt   int64
}

// AgentCoreInstance represents a persisted core instance on an agent host.
type AgentCoreInstance struct {
	ID               int64  `json:"id"`
	AgentHostID      int64  `json:"agent_host_id"`
	InstanceID       string `json:"instance_id"`
	CoreType         string `json:"core_type"`
	Status           string `json:"status"`
	ListenPorts      []int  `json:"listen_ports"`
	ConfigTemplateID *int64 `json:"config_template_id"`
	ConfigHash       string `json:"config_hash"`
	StartedAt        *int64 `json:"started_at"`
	LastHeartbeatAt  *int64 `json:"last_heartbeat_at"`
	ErrorMessage     string `json:"error_message"`
	CreatedAt        int64  `json:"created_at"`
	UpdatedAt        int64  `json:"updated_at"`
}

// AgentCoreSwitchLog captures core switching audit logs.
type AgentCoreSwitchLog struct {
	ID             int64   `json:"id"`
	AgentHostID    int64   `json:"agent_host_id"`
	FromInstanceID *string `json:"from_instance_id"`
	FromCoreType   *string `json:"from_core_type"`
	ToInstanceID   string  `json:"to_instance_id"`
	ToCoreType     string  `json:"to_core_type"`
	OperatorID     *int64  `json:"operator_id"`
	Status         string  `json:"status"`
	Detail         string  `json:"detail"`
	CreatedAt      int64   `json:"created_at"`
	CompletedAt    *int64  `json:"completed_at"`
}

// AccessLog records user traffic access history.
type AccessLog struct {
	ID              int64
	UserID          *int64
	UserEmail       string
	AgentHostID     int64
	SourceIP        string
	TargetDomain    string
	TargetIP        string
	TargetPort      int
	Protocol        string
	Upload          int64
	Download        int64
	ConnectionStart *int64 // Unix timestamp
	ConnectionEnd   *int64 // Unix timestamp
	CreatedAt       int64
}

// AccessLogFilter defines filter conditions for querying access logs.
type AccessLogFilter struct {
	UserID       *int64
	AgentHostID  *int64
	TargetDomain *string // Use LIKE match
	SourceIP     *string
	Protocol     *string
	StartAt      *int64
	EndAt        *int64
	Limit        int
	Offset       int
}

// AccessLogStats provides aggregated statistics of access logs.
type AccessLogStats struct {
	TotalCount    int64
	TotalUpload   int64
	TotalDownload int64
}
