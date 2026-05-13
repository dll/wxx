package model

// AnswerCard 统一回答结构，所有问答接口的标准响应格式
type AnswerCard struct {
	Conclusion string       `json:"conclusion"`  // 简要结论
	Steps      []string     `json:"steps"`       // 步骤列表
	Sources    []Source     `json:"sources"`      // 来源引用
	Risks      []string     `json:"risks"`       // 注意事项/风险提示
	FollowUps  []string     `json:"follow_ups"`  // 追问建议
	Actions    []Action     `json:"actions"`      // 可执行动作
	TraceID    string       `json:"trace_id"`     // 追踪 ID
	Confidence float64      `json:"confidence"`   // 置信度 0-1
	Fallback   bool         `json:"fallback"`     // 是否为兜底回答
}

// Source 来源引用
type Source struct {
	ResourceID     string  `json:"resource_id"`
	Title          string  `json:"title"`
	Version        string  `json:"version"`
	SourceLink     string  `json:"source_link"`
	RelevanceScore float64 `json:"relevance_score"` // BM25 相关性分数
}

// Action 可执行动作（如"在线申请"按钮）
type Action struct {
	Label string `json:"label"` // 按钮文字
	URL   string `json:"url"`   // 跳转地址
}

// ChatRequest 对话请求
type ChatRequest struct {
	Question  string `json:"question" binding:"required"` // 用户提问
	SessionID string `json:"session_id"`                  // 会话 ID（可选，空则新建）
	AgentID   string `json:"agent_id"`                    // 智能体 ID（可选，空则用默认）
}

// ChatResponse 对话响应（包裹 AnswerCard）
type ChatResponse struct {
	Code      int        `json:"code"`       // 状态码：0=成功
	Message   string     `json:"message"`    // 状态描述
	Data      *AnswerCard `json:"data"`      // 回答内容
	SessionID string     `json:"session_id"` // 会话 ID
}

// ErrorResponse 统一错误响应
type ErrorResponse struct {
	Code    int    `json:"code"`     // 错误码
	Message string `json:"message"`  // 错误描述
	TraceID string `json:"trace_id"` // 追踪 ID
}

// UserContext 从 JWT 中提取的用户上下文，由中间件注入
type UserContext struct {
	UserID      int64  // 用户 ID
	Username    string // 用户名
	Role        string // 角色
	OwnerScope  string // 归属范围
	OwnerID     string // 归属 ID
	DisplayName string // 显示名
	Consented   bool   // 是否已同意隐私政策与用户协议
}

// ── 知识导出 DTO ──

// ExportManifest 导出清单
type ExportManifest struct {
	ExportedAt string `json:"exported_at"` // 导出时间 RFC3339
	Format     string `json:"format"`      // 格式（ndjson）
	Count      int    `json:"count"`       // 资源数量
	Cursor     string `json:"cursor"`      // 游标（供下次增量使用）
	Version    string `json:"version"`     // 包版本
}

// ExportResponse 导出响应
type ExportResponse struct {
	Code     int            `json:"code"`
	Message  string         `json:"message"`
	Manifest ExportManifest `json:"manifest"`
	Data     []*KBResource  `json:"data"`
}

// ── 知识库管理 DTO ──

// KBCreateRequest 创建知识资源请求
type KBCreateRequest struct {
	ResourceType  string  `json:"resource_type" binding:"required,oneof=Policy Process FAQ Activity"` // 资源类型
	OwnerScope    string  `json:"owner_scope" binding:"required,oneof=school college class"`          // 归属范围
	OwnerID       string  `json:"owner_id"`                                                           // 归属 ID
	RoleScope     string  `json:"role_scope" binding:"required"`                                      // 可见角色（JSON 数组）
	Title         string  `json:"title" binding:"required"`                                           // 标题
	Summary       string  `json:"summary"`                                                            // 摘要
	Content       string  `json:"content" binding:"required"`                                         // 正文
	SourceLink    string  `json:"source_link"`                                                        // 原文链接
	SourceVersion string  `json:"source_version"`                                                     // 原文版本
	EffectiveAt   *string `json:"effective_at"`                                                       // 生效时间
	ExpiredAt     *string `json:"expired_at"`                                                         // 失效时间
	Tags          string  `json:"tags"`                                                               // 标签（JSON 数组）
}

// KBUpdateRequest 更新知识资源请求
type KBUpdateRequest struct {
	ResourceType  string  `json:"resource_type" binding:"omitempty,oneof=Policy Process FAQ Activity"` // 资源类型
	OwnerScope    string  `json:"owner_scope" binding:"omitempty,oneof=school college class"`          // 归属范围
	OwnerID       string  `json:"owner_id"`
	RoleScope     string  `json:"role_scope"`
	Status        string  `json:"status" binding:"omitempty,oneof=draft pending published retired"` // 状态
	Title         string  `json:"title"`
	Summary       string  `json:"summary"`
	Content       string  `json:"content"`
	SourceLink    string  `json:"source_link"`
	SourceVersion string  `json:"source_version"`
	EffectiveAt   *string `json:"effective_at"`
	ExpiredAt     *string `json:"expired_at"`
	Tags          string  `json:"tags"`
}

// KBListResponse 知识列表响应
type KBListResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    []*KBResource `json:"data"`
	Total   int           `json:"total"`    // 总数
	Page    int           `json:"page"`     // 当前页
	PageSize int          `json:"page_size"` // 每页数
}

// KBDetailResponse 知识详情响应
type KBDetailResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    *KBResource `json:"data"`
}

// ── 知识大厅浏览 DTO ──

// KnowledgeCard 知识大厅卡片数据（轻量，不含正文）
type KnowledgeCard struct {
	ResourceID   string `json:"resource_id"`    // 资源业务 ID
	ResourceType string `json:"resource_type"`  // 资源类型：Policy/Process/FAQ/Activity
	Title        string `json:"title"`          // 标题
	Summary      string `json:"summary"`        // 摘要（卡片展示用）
	Tags         string `json:"tags"`           // 标签（JSON 数组字符串）
	SourceLink   string `json:"source_link"`    // 原文链接
}

// KnowledgeBrowseResponse 知识大厅浏览响应（按类型分组，带分页）
type KnowledgeBrowseResponse struct {
	Code     int                         `json:"code"`
	Message  string                      `json:"message"`
	Data     map[string][]*KnowledgeCard `json:"data"`     // key = resource_type
	Total    int                         `json:"total"`    // 全部已发布资源数
	Page     int                         `json:"page"`     // 当前页码
	PageSize int                         `json:"page_size"` // 每页数量
}

// ── 情感预警 DTO ──

// EmotionAnalyzeRequest 情感分析请求
type EmotionAnalyzeRequest struct {
	MessageText string `json:"message_text" binding:"required"` // 待分析文本
	SessionID   string `json:"session_id" binding:"required"`   // 会话 ID
}

// EmotionAnalyzeResponse 情感分析响应
type EmotionAnalyzeResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    *EmotionLog `json:"data"` // 分析结果
}

// EmotionListResponse 告警列表响应
type EmotionListResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    []*EmotionLog `json:"data"`
	Total   int           `json:"total"`
}

// EmotionStatsResponse 告警统计响应
type EmotionStatsResponse struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    *EmotionStats  `json:"data"`
}

// EmotionUpdateRequest 更新告警状态请求
type EmotionUpdateRequest struct {
	Status string `json:"status" binding:"required,oneof=acknowledged resolved"` // 目标状态
}

// ── 智能体管理 DTO ──

// AgentCreateRequest 创建智能体请求
type AgentCreateRequest struct {
	AgentID       string  `json:"agent_id" binding:"required"`                    // 唯一标识
	Name          string  `json:"name" binding:"required"`                        // 显示名
	Description   string  `json:"description"`                                    // 描述
	AgentType     string  `json:"agent_type" binding:"required,oneof=qa policy emotion custom"` // 类型
	SystemPrompt  string  `json:"system_prompt"`                                  // 自定义系统提示词
	ModelProvider string  `json:"model_provider"`                                 // deepseek / zhipu
	ModelName     string  `json:"model_name"`                                     // 具体模型
	Temperature   float64 `json:"temperature"`                                    // 0.0-2.0
	MaxTokens     int     `json:"max_tokens"`                                     // 最大 token 数
}

// AgentUpdateRequest 更新智能体请求
type AgentUpdateRequest struct {
	Name          *string  `json:"name"`          // 显示名
	Description   *string  `json:"description"`   // 描述
	AgentType     *string  `json:"agent_type"`    // 类型
	SystemPrompt  *string  `json:"system_prompt"` // 自定义系统提示词
	ModelProvider *string  `json:"model_provider"`// deepseek / zhipu
	ModelName     *string  `json:"model_name"`    // 具体模型
	Temperature   *float64 `json:"temperature"`   // 0.0-2.0
	MaxTokens     *int     `json:"max_tokens"`    // 最大 token 数
	Status        *string  `json:"status"`        // active / inactive
}

// AgentListResponse 智能体列表响应
type AgentListResponse struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Data    []*Agent `json:"data"`
	Total   int      `json:"total"`
}

// AgentDetailResponse 智能体详情响应
type AgentDetailResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *Agent `json:"data"`
}

// ── 知识导入 DTO ──

// KBImportRequest 导入知识资源请求（NDJSON 格式，每行一个 KBResource JSON）
type KBImportRequest struct {
	Resources []*KBResource `json:"resources" binding:"required"` // 待导入资源列表
}

// KBImportResult 单条导入结果
type KBImportResult struct {
	ResourceID string `json:"resource_id"` // 资源 ID
	Title      string `json:"title"`       // 标题
	Action     string `json:"action"`      // created / updated / skipped
	Message    string `json:"message"`     // 说明
}

// KBImportResponse 导入响应
type KBImportResponse struct {
	Code     int               `json:"code"`
	Message  string            `json:"message"`
	Data     []*KBImportResult `json:"data"`   // 逐条结果
	Total    int               `json:"total"`  // 总条数
	Created  int               `json:"created"` // 新建数
	Updated  int               `json:"updated"` // 更新数
	Skipped  int               `json:"skipped"` // 跳过数
}

// ── 回答导出 DTO ──

// ExportAnswerRequest 回答卡片导出请求
type ExportAnswerRequest struct {
	AnswerCard *AnswerCard `json:"answer_card" binding:"required"` // 回答卡片完整数据
	Format     string      `json:"format"`                         // 导出格式：pdf / json / md（默认 pdf）
	Watermark  bool        `json:"watermark"`                      // 是否添加水印
}

// ── 情感趋势分析 DTO ──

// EmotionTrendPoint 单日情感趋势数据点
type EmotionTrendPoint struct {
	Date   string `json:"date"`   // 日期 YYYY-MM-DD
	Total  int    `json:"total"`  // 当日分析总数
	Urgent int    `json:"urgent"` // 紧急数
	High   int    `json:"high"`   // 高风险数
	Medium int    `json:"medium"` // 中风险数
	Low    int    `json:"low"`    // 低风险数
}

// EmotionTrendReport 情感趋势报告
type EmotionTrendReport struct {
	Days   int                  `json:"days"`   // 统计天数
	Points []*EmotionTrendPoint `json:"points"` // 每日数据点
	Summary struct {
		TotalAnalyses int `json:"total_analyses"` // 周期内总分析次数
		TotalUrgent   int `json:"total_urgent"`   // 周期内紧急总数
		TotalHigh     int `json:"total_high"`     // 周期内高风险总数
		AvgDaily      int `json:"avg_daily"`      // 日均分析次数
	} `json:"summary"`
}

// EmotionTrendResponse 趋势响应
type EmotionTrendResponse struct {
	Code    int                  `json:"code"`
	Message string               `json:"message"`
	Data    *EmotionTrendReport  `json:"data"`
}
