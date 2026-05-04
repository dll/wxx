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
}
