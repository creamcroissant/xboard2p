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
