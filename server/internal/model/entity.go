package model

// User 用户，对应 users 表
type User struct {
	ID          int64  `json:"id" db:"id"`
	Username    string `json:"username" db:"username"`         // 用户名（唯一）
	DisplayName string `json:"display_name" db:"display_name"` // 显示名
	Role        string `json:"role" db:"role"`                 // 角色枚举
	OwnerScope  string `json:"owner_scope" db:"owner_scope"`   // 归属范围：school/college/class
	OwnerID     string `json:"owner_id" db:"owner_id"`         // 归属 ID
	CreatedAt   string `json:"created_at" db:"created_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
}

// Session 会话，对应 sessions 表
type Session struct {
	ID        int64  `json:"id" db:"id"`
	SessionID string `json:"session_id" db:"session_id"` // 会话唯一标识
	UserID    int64  `json:"user_id" db:"user_id"`
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}

// Message 对话消息，对应 messages 表
type Message struct {
	ID        int64  `json:"id" db:"id"`
	SessionID string `json:"session_id" db:"session_id"`
	Role      string `json:"role" db:"role"` // user/assistant/system
	Content   string `json:"content" db:"content"`
	TraceID   string `json:"trace_id" db:"trace_id"`
	CreatedAt string `json:"created_at" db:"created_at"`
}

// KBResource 知识资源，对应 kb_resources 表
type KBResource struct {
	ID            int64   `json:"id" db:"id"`
	ResourceID    string  `json:"resource_id" db:"resource_id"`       // 资源唯一标识
	ResourceType  string  `json:"resource_type" db:"resource_type"`   // Policy/Process/FAQ/Activity
	OwnerScope    string  `json:"owner_scope" db:"owner_scope"`       // school/college/class
	OwnerID       string  `json:"owner_id" db:"owner_id"`
	RoleScope     string  `json:"role_scope" db:"role_scope"`         // JSON 数组：可见角色列表
	Version       string  `json:"version" db:"version"`
	Status        string  `json:"status" db:"status"`                 // draft/pending/published/retired
	Title         string  `json:"title" db:"title"`
	Summary       string  `json:"summary" db:"summary"`
	Content       string  `json:"content" db:"content"`
	SourceLink    string  `json:"source_link" db:"source_link"`       // 原文链接
	SourceVersion string  `json:"source_version" db:"source_version"` // 原文版本
	EffectiveAt   *string `json:"effective_at" db:"effective_at"`     // 生效时间（可空）
	ExpiredAt     *string `json:"expired_at" db:"expired_at"`         // 失效时间（可空）
	Tags          string `json:"tags" db:"tags"`                     // JSON 数组
	UpdatedBy     string `json:"updated_by" db:"updated_by"`
	CreatedAt     string `json:"created_at" db:"created_at"`
	UpdatedAt     string `json:"updated_at" db:"updated_at"`
}

// ProcessStep 流程步骤，对应 process_steps 表
type ProcessStep struct {
	ID         int64  `json:"id" db:"id"`
	ResourceID string `json:"resource_id" db:"resource_id"`
	StepOrder  int    `json:"step_order" db:"step_order"` // 步骤序号
	Title      string `json:"title" db:"title"`
	Materials  string `json:"materials" db:"materials"` // JSON 数组：所需材料
	EntryURL   string `json:"entry_url" db:"entry_url"` // 办理入口
	Deadline   string `json:"deadline" db:"deadline"`
	Location   string `json:"location" db:"location"` // 办理地点
	Notes      string `json:"notes" db:"notes"`
}

// AuditLog 审计日志，对应 audit_logs 表
type AuditLog struct {
	ID         int64  `json:"id" db:"id"`
	UserID     *int64 `json:"user_id" db:"user_id"` // 可为空（未登录操作）
	Username   string `json:"username" db:"username"`
	Role       string `json:"role" db:"role"`
	Action     string `json:"action" db:"action"`
	Resource   string `json:"resource" db:"resource"`
	Detail     string `json:"detail" db:"detail"`
	TraceID    string `json:"trace_id" db:"trace_id"`
	IP         string `json:"ip" db:"ip"`
	DurationMs int    `json:"duration_ms" db:"duration_ms"`
	ResultCode int    `json:"result_code" db:"result_code"`
	CreatedAt  string `json:"created_at" db:"created_at"`
}

// EmotionLog 情感评估记录，对应 emotion_logs 表
type EmotionLog struct {
	ID        int64   `json:"id" db:"id"`
	UserID    int64   `json:"user_id" db:"user_id"`
	SessionID string  `json:"session_id" db:"session_id"`
	Score     float64 `json:"score" db:"score"`         // 情感评分
	RiskLevel string  `json:"risk_level" db:"risk_level"` // low/medium/high
	Notified  int     `json:"notified" db:"notified"`     // 是否已通知
	CreatedAt string  `json:"created_at" db:"created_at"`
}

// ExportLog 导出记录，对应 export_logs 表
type ExportLog struct {
	ID           int64  `json:"id" db:"id"`
	UserID       int64  `json:"user_id" db:"user_id"`
	Role         string `json:"role" db:"role"`
	Format       string `json:"format" db:"format"` // pdf/word/markdown
	AnswerID     string `json:"answer_id" db:"answer_id"`
	HasSensitive int    `json:"has_sensitive" db:"has_sensitive"` // 是否含敏感数据
	TraceID      string `json:"trace_id" db:"trace_id"`
	CreatedAt    string `json:"created_at" db:"created_at"`
}

// SyncCursor 同步游标，对应 sync_cursors 表
type SyncCursor struct {
	ID        int64  `json:"id" db:"id"`
	Target    string `json:"target" db:"target"` // 同步目标标识
	CursorVal string `json:"cursor_val" db:"cursor_val"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}
