// 文件路径: internal/repository/filters.go
// 模块说明: 这是 internal 模块里的 filters 逻辑，下面的注释会用非常通俗的中文帮你理解每一步。
package repository

// UserSearchFilter constrains admin user listings.
type UserSearchFilter struct {
	Keyword string // Changed from Query to Keyword to match usage
	Status  *int
	PlanID  *int64
	Limit   int
	Offset  int
}

// StatUserSumFilter constrains traffic summations.
type StatUserSumFilter struct {
	UserID      *int64 // nil = all users
	AgentHostID *int64 // nil = all hosts
	RecordType  int
	StartAt     int64
	EndAt       int64
}

// StatUserTopFilter selects the top-N traffic users.
type StatUserTopFilter struct {
	AgentHostID *int64 // nil = all hosts
	RecordType  int
	StartAt     int64
	EndAt       int64
	Limit       int
}

// InboundSpecFilter constrains inbound spec listing queries.
type InboundSpecFilter struct {
	AgentHostID *int64
	CoreType    *string
	Tag         *string
	Enabled     *bool
	Limit       int
	Offset      int
}

// DesiredArtifactFilter constrains artifact listing queries.
type DesiredArtifactFilter struct {
	AgentHostID     int64
	CoreType        *string
	DesiredRevision *int64
	SourceTag       *string
	Filename        *string
	ExcludeContent  bool
	Limit           int
	Offset          int
}

// ApplyRunFilter constrains apply run listing queries.
type ApplyRunFilter struct {
	AgentHostID *int64
	CoreType    *string
	Status      *string
	Limit       int
	Offset      int
}

// AgentConfigInventoryFilter constrains inventory listing queries.
type AgentConfigInventoryFilter struct {
	AgentHostID *int64
	CoreType    *string
	Source      *string
	Filename    *string
	ParseStatus *string
	Limit       int
	Offset      int
}

// InboundIndexFilter constrains inbound index listing queries.
type InboundIndexFilter struct {
	AgentHostID *int64
	CoreType    *string
	Source      *string
	Tag         *string
	Protocol    *string
	Filename    *string
	Limit       int
	Offset      int
}

// DriftStateFilter constrains drift listing queries.
type DriftStateFilter struct {
	AgentHostID *int64
	CoreType    *string
	Status      *string
	DriftType   *string
	Tag         *string
	Filename    *string
	Limit       int
	Offset      int
}
