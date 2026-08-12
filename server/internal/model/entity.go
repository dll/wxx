package model

// User 用户，对应 users 表
type User struct {
	ID             int64  `json:"id" db:"id"`
	Username       string `json:"username" db:"username"`               // 用户名（唯一）
	DisplayName    string `json:"display_name" db:"display_name"`       // 显示名
	Role           string `json:"role" db:"role"`                       // 角色枚举
	OwnerScope     string `json:"owner_scope" db:"owner_scope"`         // 归属范围：school/college/class
	OwnerID        string `json:"owner_id" db:"owner_id"`               // 归属 ID
	College        string `json:"college" db:"college"`                 // 学院
	Major          string `json:"major" db:"major"`                     // 专业
	ClassName      string `json:"class_name" db:"class_name"`           // 班级
	EnrollmentDate string `json:"enrollment_date" db:"enrollment_date"` // 入学日期
	EnrollmentYear string `json:"enrollment_year" db:"enrollment_year"` // 入学年份
	Gender         string `json:"gender" db:"gender"`                   // 性别：男/女
	Campus         string `json:"campus" db:"campus"`                   // 校区（默认会峰校区）
	EducationLevel string `json:"education_level" db:"education_level"` // 学历层次：本科等
	StudyDuration  string `json:"study_duration" db:"study_duration"`   // 学制：4 等
	ExpectedGrad   string `json:"expected_graduation_date" db:"expected_graduation_date"` // 预期毕业时间
	StudyMode      string `json:"study_mode" db:"study_mode"`           // 学习形式：普通全日制等
	Ethnicity      string `json:"ethnicity" db:"ethnicity"`             // 民族：汉族等
	PoliticalStatus string `json:"political_status" db:"political_status"` // 政治面貌
	BirthDate      string `json:"birth_date" db:"birth_date"`           // 出生年月（隐私字段）
	Phone          string `json:"phone" db:"phone"`                     // 手机号
	Wechat         string `json:"wechat" db:"wechat"`                   // 微信号
	QQ             string `json:"qq" db:"qq"`                           // QQ
	Email          string `json:"email" db:"email"`                     // 邮箱
	PasswordHash   string `json:"-" db:"password_hash"`                 // bcrypt 密码哈希；可登录账号不得为空
	VoiceEnabled   int    `json:"voice_enabled" db:"voice_enabled"`     // 语音开关：0=关闭 1=开启
	Status         string `json:"status" db:"status"`                   // active/pending/rejected/disabled
	TokenVersion   int    `json:"-" db:"token_version"`                 // JWT 令牌版本，+1 即吊销该用户所有旧令牌
	Consented      int    `json:"consented" db:"consented"`             // 是否已同意隐私政策与用户协议：0=否 1=是
	CreatedAt      string `json:"created_at" db:"created_at"`
	UpdatedAt      string `json:"updated_at" db:"updated_at"`
}

// CanViewPrivate 判断角色是否能查看其他用户的私密信息。
// 私密信息（联系方式/出生年月等）仅本人、辅导员与管理员及以上可见。
func CanViewPrivate(role string) bool {
	switch role {
	case "counselor", "teacher", "assistant", "college_admin", "school_admin", "sys_admin":
		return true
	default:
		return false
	}
}

// SanitizePrivate 按调用者角色脱敏私密字段（非授权角色清空联系方式与出生年月）。
// 返回是否发生了脱敏；copy 语义安全，不影响底层数据。
func (u *User) SanitizePrivate(role string) bool {
	if u == nil {
		return false
	}
	if CanViewPrivate(role) {
		return false
	}
	u.BirthDate = ""
	u.Phone = ""
	u.Wechat = ""
	u.QQ = ""
	u.Email = ""
	return true
}

// ContactPerson 组织关系联系人（辅导员/领导等）
type ContactPerson struct {
	ID       int64  `json:"id" db:"id"`
	Name     string `json:"name" db:"display_name"`
	Role     string `json:"role" db:"role"`
	RoleName string `json:"role_name"`
	Phone    string `json:"phone" db:"phone"`
	Wechat   string `json:"wechat" db:"wechat"`
	Email    string `json:"email" db:"email"`
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
	ResourceID    string  `json:"resource_id" db:"resource_id"`     // 资源唯一标识
	ResourceType  string  `json:"resource_type" db:"resource_type"` // Policy/Process/FAQ/Activity
	OwnerScope    string  `json:"owner_scope" db:"owner_scope"`     // school/college/class
	OwnerID       string  `json:"owner_id" db:"owner_id"`
	RoleScope     string  `json:"role_scope" db:"role_scope"` // JSON 数组：可见角色列表
	Version       string  `json:"version" db:"version"`
	Status        string  `json:"status" db:"status"` // draft/pending/published/retired
	Title         string  `json:"title" db:"title"`
	Summary       string  `json:"summary" db:"summary"`
	Content       string  `json:"content" db:"content"`
	SourceLink    string  `json:"source_link" db:"source_link"`       // 原文链接
	SourceVersion string  `json:"source_version" db:"source_version"` // 原文版本
	EffectiveAt   *string `json:"effective_at" db:"effective_at"`     // 生效时间（可空）
	ExpiredAt     *string `json:"expired_at" db:"expired_at"`         // 失效时间（可空）
	Tags          string  `json:"tags" db:"tags"`                     // JSON 数组
	Remark        string  `json:"remark" db:"remark"`                 // 上传者备注
	UpdatedBy     string  `json:"updated_by" db:"updated_by"`
	CreatedAt     string  `json:"created_at" db:"created_at"`
	UpdatedAt     string  `json:"updated_at" db:"updated_at"`
}

// ProcessStep 流程步骤，对应 process_steps 表
type ProcessStep struct {
	ID            int64   `json:"id" db:"id"`
	ResourceID    string  `json:"resource_id" db:"resource_id"`
	StepOrder     int     `json:"step_order" db:"step_order"` // 步骤序号
	Title         string  `json:"title" db:"title"`
	Materials     string  `json:"materials" db:"materials"` // JSON 数组：所需材料
	EntryURL      string  `json:"entry_url" db:"entry_url"` // 办理入口
	Deadline      string  `json:"deadline" db:"deadline"`
	Location      string  `json:"location" db:"location"` // 办理地点
	Notes         string  `json:"notes" db:"notes"`
	Contact       string  `json:"contact" db:"contact"`               // 联系人
	Phone         string  `json:"phone" db:"phone"`                   // 联系电话
	ContactWechat string  `json:"contact_wechat" db:"contact_wechat"` // 联系人微信/企业微信
	OfficeHours   string  `json:"office_hours" db:"office_hours"`     // 办公时间
	GeoLat        float64 `json:"geo_lat" db:"geo_lat"`               // 办理地点纬度（0 表示未录入）
	GeoLng        float64 `json:"geo_lng" db:"geo_lng"`               // 办理地点经度（0 表示未录入）
	MediaURLs     string  `json:"media_urls" db:"media_urls"`         // JSON 数组：办理指引图片/视频 URL
	FAQ           string  `json:"faq" db:"faq"`                       // JSON 数组：[{"q":"…","a":"…"}]
}

// ProcessReminder 办事流程提醒，对应 process_reminders 表
type ProcessReminder struct {
	ID        int64  `json:"id" db:"id"`
	ProcessID string `json:"process_id" db:"process_id"`
	StepOrder int    `json:"step_order" db:"step_order"` // 关联步骤序号，0 表示流程级提醒
	RemindAt  string `json:"remind_at" db:"remind_at"`
	Title     string `json:"title" db:"title"`
	Content   string `json:"content" db:"content"`
	IsEnabled int    `json:"is_enabled" db:"is_enabled"`
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
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

// AuditSnapshot 审计恢复快照（可恢复的写操作前后状态）
type AuditSnapshot struct {
	ID         int64  `json:"id" db:"id"`
	AuditID    int64  `json:"audit_id" db:"audit_id"`
	OpTable    string `json:"op_table" db:"op_table"`
	RecordID   string `json:"record_id" db:"record_id"`
	Operation  string `json:"operation" db:"operation"`
	BeforeJSON string `json:"before_json" db:"before_json"`
	AfterJSON  string `json:"after_json" db:"after_json"`
	Restored   int    `json:"restored" db:"restored"`
	RestoredAt string `json:"restored_at" db:"restored_at"`
	RestoredBy string `json:"restored_by" db:"restored_by"`
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
	Score          float64 `json:"score" db:"score"`                 // 情感评分 -1.0~1.0
	RiskLevel      string  `json:"risk_level" db:"risk_level"`       // low/medium/high
	AnalysisJSON   string  `json:"analysis_json" db:"analysis_json"` // LLM 分析原始结果
	Notified       int     `json:"notified" db:"notified"`           // 是否已通知
	Status         string  `json:"status" db:"status"`               // pending/acknowledged/resolved
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
	ID                 int64   `json:"id" db:"id"`
	FeedbackID         string  `json:"feedback_id" db:"feedback_id"`
	UserID             int64   `json:"user_id" db:"user_id"`
	Username           string  `json:"username" db:"username"`
	MessageID          string  `json:"message_id" db:"message_id"`
	ResourceID         string  `json:"resource_id" db:"resource_id"`
	Category           string  `json:"category" db:"category"`
	Module             string  `json:"module" db:"module"` // 所属模块（用于在线修复代码定位）
	Content            string  `json:"content" db:"content"`
	ScreenshotURL      string  `json:"screenshot_url" db:"screenshot_url"` // 截图存储路径
	Status             string  `json:"status" db:"status"`
	ResolvedBy         string  `json:"resolved_by" db:"resolved_by"`
	ResolvedAt         *string `json:"resolved_at" db:"resolved_at"`
	Reply              string  `json:"reply" db:"reply"` // 管理员回复
	Rating             int     `json:"rating" db:"rating"`
	RatingComment      string  `json:"rating_comment" db:"rating_comment"`
	RatedAt            *string `json:"rated_at" db:"rated_at"`
	LinkedResourceNote string  `json:"linked_resource_note" db:"linked_resource_note"`
	LinkedAt           *string `json:"linked_at" db:"linked_at"`
	LinkedBy           string  `json:"linked_by" db:"linked_by"`
	CreatedAt          string  `json:"created_at" db:"created_at"`
	UpdatedAt          string  `json:"updated_at" db:"updated_at"`
}

// FeedbackLog 反馈处理记录，对应 feedback_logs 表
type FeedbackLog struct {
	ID         int64  `json:"id" db:"id"`
	FeedbackID string `json:"feedback_id" db:"feedback_id"`
	Action     string `json:"action" db:"action"`
	Operator   string `json:"operator" db:"operator"`
	Detail     string `json:"detail" db:"detail"`
	CreatedAt  string `json:"created_at" db:"created_at"`
}

// FeedbackRepairJob 反馈 AI 自动修复工单，对应 feedback_repair_jobs 表。
// 每次「在线修复并部署」创建一条，记录执行阶段 / 日志 / 结果 / 被修改文件，供前端轮询与审计。
type FeedbackRepairJob struct {
	ID          int64  `json:"id" db:"id"`
	RunID       string `json:"run_id" db:"run_id"`
	FeedbackID  string `json:"feedback_id" db:"feedback_id"`
	Operator    string `json:"operator" db:"operator"`
	Status      string `json:"status" db:"status"`             // running | succeeded | failed | rolled_back
	Stage       string `json:"stage" db:"stage"`               // init/diagnose/apply/build/deploy/healthcheck/done/failed
	LogText     string `json:"log_text" db:"log_text"`         // 执行日志（多行，前端滚动展示）
	EditedFiles string `json:"edited_files" db:"edited_files"` // JSON 数组：被修改文件（相对 /opt/wxx 仓库根）
	Summary     string `json:"summary" db:"summary"`           // AI 问题摘要
	Detail      string `json:"detail" db:"detail"`             // 修复说明 / 错误描述
	CreatedAt   string `json:"created_at" db:"created_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
}

// RepairJobStage 自动修复阶段常量
const (
	RepairStageInit        = "init"
	RepairStageDiagnose    = "diagnose"
	RepairStageGenPatch    = "gen_patch"
	RepairStageApply       = "apply"
	RepairStageBuild       = "build"
	RepairStageVerify      = "verify" // 独立端口健康检查
	RepairStageDeploy      = "deploy"
	RepairStageHealthCheck = "healthcheck"
	RepairStageDone        = "done"
	RepairStageFailed      = "failed"
)

// RepairStatus 自动修复结果状态常量
const (
	RepairStatusRunning    = "running"
	RepairStatusSucceeded  = "succeeded"
	RepairStatusFailed     = "failed"
	RepairStatusRolledBack = "rolled_back"
)

// FeedbackStats 反馈统计数据
type FeedbackStats struct {
	Total           int             `json:"total"`
	ByStatus        map[string]int  `json:"by_status"`
	ByCategory      map[string]int  `json:"by_category"`
	WeekTrend       []WeekTrendItem `json:"week_trend"`
	TopIssues       []TopIssueItem  `json:"top_issues"`
	AvgResolveHours float64         `json:"avg_resolve_hours"`
}

// WeekTrendItem 周趋势数据项
type WeekTrendItem struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// TopIssueItem 热门问题项
type TopIssueItem struct {
	Keyword string `json:"keyword"`
	Count   int    `json:"count"`
}

// FeedbackLinkResourceRequest 关联知识资源请求
type FeedbackLinkResourceRequest struct {
	ResourceID string `json:"resource_id" binding:"required"`
	Note       string `json:"note"`
}

// FeedbackRateRequest 满意度评价请求
type FeedbackRateRequest struct {
	Rating  int    `json:"rating" binding:"required,min=1,max=5"`
	Comment string `json:"comment"`
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

// maskSecret 对密钥脱敏：保留末 4 位，前缀用 **** 代替；空串返回空串。
// 安全修复 SEC-05：读取模型配置时绝不回显密钥明文。
func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	if len(r) <= 4 {
		return "****"
	}
	return "****" + string(r[len(r)-4:])
}

// ToMaskedView 将模型配置转为脱敏视图，密钥仅返回掩码 + 是否已配置标志。
func (c *UserModelConfig) ToMaskedView() *ModelConfigView {
	return &ModelConfigView{
		ID:              c.ID,
		UserID:          c.UserID,
		DeepseekKey:     maskSecret(c.DeepseekKey),
		DeepseekKeySet:  c.DeepseekKey != "",
		DeepseekModel:   c.DeepseekModel,
		DeepseekTemp:    c.DeepseekTemp,
		DeepseekMaxTok:  c.DeepseekMaxTok,
		ZhipuKey:        maskSecret(c.ZhipuKey),
		ZhipuKeySet:     c.ZhipuKey != "",
		ZhipuModel:      c.ZhipuModel,
		ZhipuTemp:       c.ZhipuTemp,
		ZhipuMaxTok:     c.ZhipuMaxTok,
		XunfeiAppID:     c.XunfeiAppID,
		XunfeiKey:       maskSecret(c.XunfeiKey),
		XunfeiKeySet:    c.XunfeiKey != "",
		XunfeiSecret:    maskSecret(c.XunfeiSecret),
		XunfeiSecretSet: c.XunfeiSecret != "",
		XunfeiModel:     c.XunfeiModel,
		XunfeiTemp:      c.XunfeiTemp,
		XunfeiMaxTok:    c.XunfeiMaxTok,
		DefaultProvider: c.DefaultProvider,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
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
	AgentType     string  `json:"agent_type" db:"agent_type"` // qa / policy / emotion / custom
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
	ID             int64  `json:"id" db:"id"`
	RecordID       string `json:"record_id" db:"record_id"`
	UserID         int64  `json:"user_id" db:"user_id"`
	FlowType       string `json:"flow_type" db:"flow_type"`   // enrollment / graduation / leave / ...
	FlowLabel      string `json:"flow_label" db:"flow_label"` // 显示名称
	CurrentStep    int    `json:"current_step" db:"current_step"`
	CompletedSteps string `json:"completed_steps" db:"completed_steps"` // JSON 数组字符串
	TotalSteps     int    `json:"total_steps" db:"total_steps"`
	Status         string `json:"status" db:"status"` // in_progress / completed / abandoned
	Notes          string `json:"notes" db:"notes"`
	CreatedAt      string `json:"created_at" db:"created_at"`
	UpdatedAt      string `json:"updated_at" db:"updated_at"`
}

// IssueForecast 问题预案，对应 issue_forecasts 表
type IssueForecast struct {
	ID               int64   `json:"id" db:"id"`
	ForecastID       string  `json:"forecast_id" db:"forecast_id"`             // 预案ID（UUID）
	CollegeID        string  `json:"college_id" db:"college_id"`               // 学院ID（空=全校）
	Category         string  `json:"category" db:"category"`                   // 问题分类
	Subcategory      string  `json:"subcategory" db:"subcategory"`             // 子分类
	Title            string  `json:"title" db:"title"`                         // 问题标题
	RiskLevel        string  `json:"risk_level" db:"risk_level"`               // 风险等级
	Status           string  `json:"status" db:"status"`                       // 状态
	AffectedCount    int     `json:"affected_count" db:"affected_count"`       // 影响人数
	RootCause        string  `json:"root_cause" db:"root_cause"`               // 原因分析
	SuggestedActions string  `json:"suggested_actions" db:"suggested_actions"` // 建议措施（JSON数组）
	DataSummary      string  `json:"data_summary" db:"data_summary"`           // 数据摘要（JSON）
	Sources          string  `json:"sources" db:"sources"`                     // 数据来源（JSON数组）
	AIAnalysis       string  `json:"ai_analysis" db:"ai_analysis"`             // AI分析结果
	CreatedBy        *int64  `json:"created_by" db:"created_by"`               // 创建人ID
	CreatedAt        string  `json:"created_at" db:"created_at"`
	UpdatedAt        string  `json:"updated_at" db:"updated_at"`
	ResolvedAt       *string `json:"resolved_at" db:"resolved_at"` // 解决时间
	ResolvedBy       *int64  `json:"resolved_by" db:"resolved_by"` // 解决人ID
}

// IssueDetail 问题详情，对应 issue_details 表
type IssueDetail struct {
	ID          int64   `json:"id" db:"id"`
	ForecastID  string  `json:"forecast_id" db:"forecast_id"`
	UserID      *int64  `json:"user_id" db:"user_id"`     // 用户ID
	UserType    string  `json:"user_type" db:"user_type"` // 用户类型
	Username    string  `json:"username" db:"username"`
	DisplayName string  `json:"display_name" db:"display_name"`
	College     string  `json:"college" db:"college"`
	ClassName   string  `json:"class_name" db:"class_name"`
	DetailType  string  `json:"detail_type" db:"detail_type"` // 详情类型
	DetailData  string  `json:"detail_data" db:"detail_data"` // 详情数据（JSON）
	RiskScore   float64 `json:"risk_score" db:"risk_score"`   // 风险分数
	CreatedAt   string  `json:"created_at" db:"created_at"`
}

// IssueForecastHistory 问题预案历史，对应 issue_forecast_history 表
type IssueForecastHistory struct {
	ID           int64  `json:"id" db:"id"`
	ForecastID   string `json:"forecast_id" db:"forecast_id"`
	Action       string `json:"action" db:"action"`           // 操作类型
	OperatorID   *int64 `json:"operator_id" db:"operator_id"` // 操作人ID
	OperatorName string `json:"operator_name" db:"operator_name"`
	Detail       string `json:"detail" db:"detail"` // 操作详情（JSON）
	CreatedAt    string `json:"created_at" db:"created_at"`
}

// ForecastSummary 问题预案统计摘要
type ForecastSummary struct {
	TotalIssues       int            `json:"total_issues"`
	RiskDistribution  map[string]int `json:"risk_distribution"`
	CategoryDistribut map[string]int `json:"category_distribution"`
	Trend             string         `json:"trend"` // increasing/decreasing/stable
	KeyFindings       []string       `json:"key_findings"`
}

// ForecastAnalysisResponse 问题预案分析响应
type ForecastAnalysisResponse struct {
	Summary   ForecastSummary  `json:"summary"`
	Issues    []*IssueForecast `json:"issues"`
	ReportURL string           `json:"report_url,omitempty"`
}

// ══════════════════════════════════════════════════════════════
// 毕设选题智能体模型
// ══════════════════════════════════════════════════════════════

// Advisor 导师
type Advisor struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	AdvisorID     string `json:"advisor_id"`
	Title         string `json:"title"`
	College       string `json:"college"`
	Department    string `json:"department"`
	ResearchAreas string `json:"research_areas"`
	MaxStudents   int    `json:"max_students"`
	IsActive      int    `json:"is_active"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// ThesisTopic 毕设选题
type ThesisTopic struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	AdvisorID     int64  `json:"advisor_id"`
	AdvisorName   string `json:"advisor_name,omitempty"`
	College       string `json:"college"`
	Major         string `json:"major"`
	TopicType     string `json:"topic_type"`
	Nature        string `json:"nature"`
	ResultForm    string `json:"result_form"`
	Difficulty    string `json:"difficulty"`
	Description   string `json:"description"`
	Requirements  string `json:"requirements"`
	Keywords      string `json:"keywords"`
	MaxStudents   int    `json:"max_students"`
	SelectedCount int    `json:"selected_count"`
	Batch         int    `json:"batch"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// StudentTopicSelection 学生选题记录
type StudentTopicSelection struct {
	ID              int64  `json:"id"`
	UserID          int64  `json:"user_id"`
	StudentID       string `json:"student_id"`
	StudentName     string `json:"student_name"`
	College         string `json:"college"`
	Major           string `json:"major"`
	ClassName       string `json:"class_name"`
	Batch           int    `json:"batch"`
	TopicID         int64  `json:"topic_id"`
	TopicName       string `json:"topic_name,omitempty"`
	AdvisorID       int64  `json:"advisor_id"`
	AdvisorName     string `json:"advisor_name,omitempty"`
	Status          string `json:"status"`
	PreferenceOrder int    `json:"preference_order"`
	Reason          string `json:"reason"`
	ConfirmedAt     string `json:"confirmed_at"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// GraduationMilestone 毕设里程碑
type GraduationMilestone struct {
	ID          int64  `json:"id"`
	Batch       int    `json:"batch"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Deadline    string `json:"deadline"`
	Weight      int    `json:"weight"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
	CreatedAt   string `json:"created_at"`
}

// GraduationProgress 毕设进度
type GraduationProgress struct {
	ID            int64  `json:"id"`
	UserID        int64  `json:"user_id"`
	TopicID       int64  `json:"topic_id"`
	MilestoneCode string `json:"milestone_code"`
	Status        string `json:"status"`
	SubmittedAt   string `json:"submitted_at"`
	CompletedAt   string `json:"completed_at"`
	Feedback      string `json:"feedback"`
	Score         int    `json:"score"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// ══════════════════════════════════════════════════════════════
// 学科竞赛模型
// ══════════════════════════════════════════════════════════════

// Competition 竞赛信息
type Competition struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Level             string `json:"level"`
	Category          string `json:"category"`
	Organizer         string `json:"organizer"`
	Description       string `json:"description"`
	Requirements      string `json:"requirements"`
	Features          string `json:"features"`
	RegistrationStart string `json:"registration_start"`
	RegistrationEnd   string `json:"registration_end"`
	CompetitionDate   string `json:"competition_date"`
	ResultDate        string `json:"result_date"`
	Website           string `json:"website"`
	ResourceLinks     string `json:"resource_links"`
	MaxTeamSize       int    `json:"max_team_size"`
	IsTeamCompetition int    `json:"is_team_competition"`
	Status            string `json:"status"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// CompetitionRegistration 竞赛报名
type CompetitionRegistration struct {
	ID              int64  `json:"id"`
	CompetitionID   int64  `json:"competition_id"`
	CompetitionName string `json:"competition_name,omitempty"`
	UserID          int64  `json:"user_id"`
	StudentID       string `json:"student_id"`
	StudentName     string `json:"student_name"`
	College         string `json:"college"`
	Major           string `json:"major"`
	ClassName       string `json:"class_name"`
	TeamName        string `json:"team_name"`
	TeamMembers     string `json:"team_members"`
	AdvisorName     string `json:"advisor_name"`
	Status          string `json:"status"`
	WorkTitle       string `json:"work_title"`
	WorkDescription string `json:"work_description"`
	WorkFileURL     string `json:"work_file_url"`
	AwardLevel      string `json:"award_level"`
	AwardDate       string `json:"award_date"`
	CertificateURL  string `json:"certificate_url"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// ══════════════════════════════════════════════════════════════
// 大学规划模型
// ══════════════════════════════════════════════════════════════

// PlanTemplate 规划模板
type PlanTemplate struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	TargetAudience string `json:"target_audience"`
	Duration       string `json:"duration"`
	Goals          string `json:"goals"`
	Milestones     string `json:"milestones"`
	SuccessCases   string `json:"success_cases"`
	AIPrompt       string `json:"ai_prompt"`
	IsActive       int    `json:"is_active"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// StudentPlan 学生规划
type StudentPlan struct {
	ID              int64   `json:"id"`
	UserID          int64   `json:"user_id"`
	TemplateID      int64   `json:"template_id"`
	TemplateName    string  `json:"template_name,omitempty"`
	Title           string  `json:"title"`
	Category        string  `json:"category"`
	AcademicYear    int     `json:"academic_year"`
	Semester        int     `json:"semester"`
	Goals           string  `json:"goals"`
	Progress        float64 `json:"progress"`
	Status          string  `json:"status"`
	ReviewerID      int64   `json:"reviewer_id"`
	ReviewerComment string  `json:"reviewer_comment"`
	ReviewedAt      string  `json:"reviewed_at"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// PlanProgressRecord 规划进度记录
type PlanProgressRecord struct {
	ID         int64  `json:"id"`
	PlanID     int64  `json:"plan_id"`
	GoalIndex  int    `json:"goal_index"`
	GoalTitle  string `json:"goal_title"`
	Status     string `json:"status"`
	Evidence   string `json:"evidence"`
	Score      int    `json:"score"`
	Feedback   string `json:"feedback"`
	RecordedAt string `json:"recorded_at"`
}

// ══════════════════════════════════════════════════════════════
// 入党教育模型
// ══════════════════════════════════════════════════════════════

// PartyStage 入党阶段
type PartyStage struct {
	ID           int64  `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	RequiredDocs string `json:"required_docs"`
	SortOrder    int    `json:"sort_order"`
}

// PartyProgress 学生入党进度
type PartyProgress struct {
	ID               int64  `json:"id"`
	UserID           int64  `json:"user_id"`
	StudentID        string `json:"student_id"`
	StudentName      string `json:"student_name"`
	College          string `json:"college"`
	CurrentStage     string `json:"current_stage"`
	CurrentStageName string `json:"current_stage_name,omitempty"`
	ApplyDate        string `json:"apply_date"`
	ActivatorDate    string `json:"activator_date"`
	DevelopmentDate  string `json:"development_date"`
	ProbationStart   string `json:"probation_start"`
	ConversionDate   string `json:"conversion_date"`
	Status           string `json:"status"`
	Notes            string `json:"notes"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// PartyStudyRecord 学习记录
type PartyStudyRecord struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	StudyType   string `json:"study_type"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Duration    int    `json:"duration"`
	StudyDate   string `json:"study_date"`
	Certificate string `json:"certificate"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// ══════════════════════════════════════════════════════════════
// 社团生活模型
// ══════════════════════════════════════════════════════════════

// Club 社团
type Club struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Founder     string `json:"founder"`
	President   string `json:"president"`
	ContactInfo string `json:"contact_info"`
	MemberCount int    `json:"member_count"`
	MaxMembers  int    `json:"max_members"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ClubMember 社团成员
type ClubMember struct {
	ID          int64  `json:"id"`
	ClubID      int64  `json:"club_id"`
	ClubName    string `json:"club_name,omitempty"`
	UserID      int64  `json:"user_id"`
	StudentID   string `json:"student_id"`
	StudentName string `json:"student_name"`
	Role        string `json:"role"`
	JoinDate    string `json:"join_date"`
	LeaveDate   string `json:"leave_date"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// ClubActivity 社团活动
type ClubActivity struct {
	ID                  int64  `json:"id"`
	ClubID              int64  `json:"club_id"`
	ClubName            string `json:"club_name,omitempty"`
	Title               string `json:"title"`
	Description         string `json:"description"`
	ActivityType        string `json:"activity_type"`
	StartTime           string `json:"start_time"`
	EndTime             string `json:"end_time"`
	Location            string `json:"location"`
	MaxParticipants     int    `json:"max_participants"`
	CurrentParticipants int    `json:"current_participants"`
	Status              string `json:"status"`
	CreatedAt           string `json:"created_at"`
}

// ClubActivityRegistration 社团活动报名
type ClubActivityRegistration struct {
	ID          int64  `json:"id"`
	ActivityID  int64  `json:"activity_id"`
	UserID      int64  `json:"user_id"`
	StudentName string `json:"student_name"`
	Status      string `json:"status"`
	Feedback    string `json:"feedback"`
	Rating      int    `json:"rating"`
	CreatedAt   string `json:"created_at"`
}

// AppVersion 应用版本
type AppVersion struct {
	ID          int64  `json:"id" db:"id"`
	VersionCode int    `json:"version_code" db:"version_code"`
	VersionName string `json:"version_name" db:"version_name"`
	Platform    string `json:"platform" db:"platform"`
	Title       string `json:"title" db:"title"`
	Changelog   string `json:"changelog" db:"changelog"`
	DownloadURL string `json:"download_url" db:"download_url"`
	ForceUpdate int    `json:"force_update" db:"force_update"`
	Status      int    `json:"status" db:"status"`
	CreatedAt   string `json:"created_at" db:"created_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
}
