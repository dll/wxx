package model

// User 用户，对应 users 表
type User struct {
	ID           int64  `json:"id" db:"id"`
	Username     string `json:"username" db:"username"`         // 用户名（唯一）
	DisplayName  string `json:"display_name" db:"display_name"` // 显示名
	Role         string `json:"role" db:"role"`                 // 角色枚举
	OwnerScope   string `json:"owner_scope" db:"owner_scope"`   // 归属范围：school/college/class
	OwnerID      string `json:"owner_id" db:"owner_id"`         // 归属 ID
	PasswordHash string `json:"-" db:"password_hash"`           // bcrypt 密码哈希（空=免密）
	VoiceEnabled int    `json:"voice_enabled" db:"voice_enabled"` // 语音开关：0=关闭 1=开启
	CreatedAt    string `json:"created_at" db:"created_at"`
	UpdatedAt    string `json:"updated_at" db:"updated_at"`
}

// Session 会话，对应 sessions 表
type Session struct {
	ID        int64  `json:"id" db:"id"`
	SessionID string `json:"session_id" db:"session_id"` // 会话唯一标识
	UserID    int64  `json:"user_id" db:"user_id"`
	Title     string `json:"title" db:"title"` // 会话标题（用户可重命名，空则前端 fallback）
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
	ID             int64   `json:"id" db:"id"`
	UserID         int64   `json:"user_id" db:"user_id"`
	Username       string  `json:"username" db:"username"`
	SessionID      string  `json:"session_id" db:"session_id"`
	AlertID        string  `json:"alert_id" db:"alert_id"`
	MessageText    string  `json:"message_text" db:"message_text"`
	Score          float64 `json:"score" db:"score"`                   // 情感评分 -1.0~1.0
	RiskLevel      string  `json:"risk_level" db:"risk_level"`         // low/medium/high
	AnalysisJSON   string  `json:"analysis_json" db:"analysis_json"`   // LLM 分析原始结果
	Notified       int     `json:"notified" db:"notified"`             // 是否已通知
	Status         string  `json:"status" db:"status"`                 // pending/acknowledged/resolved
	AcknowledgedBy string  `json:"acknowledged_by" db:"acknowledged_by"`
	AcknowledgedAt string  `json:"acknowledged_at" db:"acknowledged_at"`
	CreatedAt      string  `json:"created_at" db:"created_at"`
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

// Feedback 用户反馈，对应 feedback 表
type Feedback struct {
	ID            int64   `json:"id" db:"id"`
	FeedbackID    string  `json:"feedback_id" db:"feedback_id"`
	UserID        int64   `json:"user_id" db:"user_id"`
	Username      string  `json:"username" db:"username"`
	MessageID     string  `json:"message_id" db:"message_id"`
	ResourceID    string  `json:"resource_id" db:"resource_id"`
	Category      string  `json:"category" db:"category"`
	Content       string  `json:"content" db:"content"`
	ScreenshotURL string  `json:"screenshot_url" db:"screenshot_url"` // 截图存储路径
	Status        string  `json:"status" db:"status"`
	ResolvedBy    string  `json:"resolved_by" db:"resolved_by"`
	ResolvedAt    *string `json:"resolved_at" db:"resolved_at"`
	Reply         string  `json:"reply" db:"reply"` // 管理员回复
	CreatedAt     string  `json:"created_at" db:"created_at"`
	UpdatedAt     string  `json:"updated_at" db:"updated_at"`
}

// TokenUsage 词元使用记录，对应 token_usage 表
type TokenUsage struct {
	ID            int64  `json:"id" db:"id"`
	UserID        int64  `json:"user_id" db:"user_id"`
	SessionID     string `json:"session_id" db:"session_id"`
	PromptTokens  int    `json:"prompt_tokens" db:"prompt_tokens"`
	OutputTokens  int    `json:"output_tokens" db:"output_tokens"`
	ModelProvider string `json:"model_provider" db:"model_provider"`
	CreatedAt     string `json:"created_at" db:"created_at"`
}

// UserModelConfig 用户 AI 模型配置，对应 user_model_configs 表
type UserModelConfig struct {
	ID              int64   `json:"id" db:"id"`
	UserID          int64   `json:"user_id" db:"user_id"`
	DeepseekKey     string  `json:"deepseek_key" db:"deepseek_key"`
	DeepseekModel   string  `json:"deepseek_model" db:"deepseek_model"`
	DeepseekTemp    float64 `json:"deepseek_temp" db:"deepseek_temp"`
	DeepseekMaxTok  int     `json:"deepseek_max_tokens" db:"deepseek_max_tokens"`
	ZhipuKey        string  `json:"zhipu_key" db:"zhipu_key"`
	ZhipuModel      string  `json:"zhipu_model" db:"zhipu_model"`
	ZhipuTemp       float64 `json:"zhipu_temp" db:"zhipu_temp"`
	ZhipuMaxTok     int     `json:"zhipu_max_tokens" db:"zhipu_max_tokens"`
	XunfeiAppID     string  `json:"xunfei_app_id" db:"xunfei_app_id"`
	XunfeiKey       string  `json:"xunfei_key" db:"xunfei_key"`
	XunfeiSecret    string  `json:"xunfei_secret" db:"xunfei_secret"`
	XunfeiModel     string  `json:"xunfei_model" db:"xunfei_model"`
	XunfeiTemp      float64 `json:"xunfei_temp" db:"xunfei_temp"`
	XunfeiMaxTok    int     `json:"xunfei_max_tokens" db:"xunfei_max_tokens"`
	DefaultProvider string  `json:"default_provider" db:"default_provider"`
	CreatedAt       string  `json:"created_at" db:"created_at"`
	UpdatedAt       string  `json:"updated_at" db:"updated_at"`
}

// SystemSetting 系统配置项，对应 system_settings 表
type SystemSetting struct {
	ID          int64  `json:"id" db:"id"`
	Key         string `json:"key" db:"key"`
	Value       string `json:"value" db:"value"`
	Description string `json:"description" db:"description"`
	UpdatedBy   string `json:"updated_by" db:"updated_by"`
	CreatedAt   string `json:"created_at" db:"created_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
}

// EmotionStats 情感告警统计
type EmotionStats struct {
	Pending int `json:"pending"`
	Urgent  int `json:"urgent"`
	High    int `json:"high"`
	Medium  int `json:"medium"`
	Low     int `json:"low"`
}

// Agent 智能体，对应 agents 表
type Agent struct {
	ID            int64   `json:"id" db:"id"`
	AgentID       string  `json:"agent_id" db:"agent_id"`
	Name          string  `json:"name" db:"name"`
	Description   string  `json:"description" db:"description"`
	AgentType     string  `json:"agent_type" db:"agent_type"`         // qa / policy / emotion / custom
	SystemPrompt  string  `json:"system_prompt" db:"system_prompt"`
	ModelProvider string  `json:"model_provider" db:"model_provider"` // deepseek / zhipu
	ModelName     string  `json:"model_name" db:"model_name"`
	Temperature   float64 `json:"temperature" db:"temperature"`
	MaxTokens     int     `json:"max_tokens" db:"max_tokens"`
	Status        string  `json:"status" db:"status"` // active / inactive
	ConfigJSON    string  `json:"config_json" db:"config_json"`
	CreatedAt     string  `json:"created_at" db:"created_at"`
	UpdatedAt     string  `json:"updated_at" db:"updated_at"`
}


// ProcessRecord 办事流程办理记录，对应 process_records 表
type ProcessRecord struct {
	ID              int64  `json:"id" db:"id"`
	RecordID        string `json:"record_id" db:"record_id"`
	UserID          int64  `json:"user_id" db:"user_id"`
	FlowType        string `json:"flow_type" db:"flow_type"`             // enrollment / graduation / leave / ...
	FlowLabel       string `json:"flow_label" db:"flow_label"`           // 显示名称
	CurrentStep     int    `json:"current_step" db:"current_step"`
	CompletedSteps  string `json:"completed_steps" db:"completed_steps"` // JSON 数组字符串
	TotalSteps      int    `json:"total_steps" db:"total_steps"`
	Status          string `json:"status" db:"status"` // in_progress / completed / abandoned
	Notes           string `json:"notes" db:"notes"`
	CreatedAt       string `json:"created_at" db:"created_at"`
	UpdatedAt       string `json:"updated_at" db:"updated_at"`
}
