// Package model 定义业务错误码注册表（A-01）。
// 错误码分段：
//
//	1xxxx — 认证/授权
//	2xxxx — 参数校验
//	3xxxx — 业务逻辑
//	4xxxx — 外部依赖（LLM / 对接系统）
//	5xxxx — 系统内部
//
// 每个错误码包含：业务码（稳定）、HTTP 状态码、中文消息模板。
// handler 层统一使用 BizError 返回，前端按 code 字段做 i18n 或展示。
//
// ⚠️ DPV4F-W7 已知差异：文档附录 A 承诺响应 code 为 `0/4xxx/5xxx`，
// 本实现采用 `0/1xxxx/2xxxx/...` 内部稳定错误码。两者语义一致（0=成功），
// 分段含义不同。前端依赖 `code == 0` 判断成功，不受影响。此为设计取舍，
// 保留内部稳定码以避免大规模破坏性变更；如需严格对齐文档附录 A，
// 可在 util 响应出口增加一次映射（见 util.Success/Fail 系列）。
package model

import "net/http"

// BizError 业务错误（可携带业务码 + HTTP 状态码）
type BizError struct {
	Code       int    `json:"code"`    // 业务错误码（稳定，前端可依赖）
	HTTPStatus int    `json:"-"`       // 对应 HTTP 状态码
	Message    string `json:"message"` // 中文错误描述
}

func (e *BizError) Error() string {
	return e.Message
}

// NewBizError 创建业务错误（允许覆盖 message）
func NewBizError(code int, msgs ...string) *BizError {
	base, ok := errRegistry[code]
	if !ok {
		return &BizError{Code: code, HTTPStatus: http.StatusInternalServerError, Message: "未知错误"}
	}
	msg := base.Message
	if len(msgs) > 0 && msgs[0] != "" {
		msg = msgs[0]
	}
	return &BizError{Code: code, HTTPStatus: base.HTTPStatus, Message: msg}
}

// ── 错误码常量 ──

// 1xxxx 认证/授权
const (
	ErrCodeUnauthorized     = 10001 // 未登录或令牌过期
	ErrCodeTokenRevoked     = 10002 // 令牌已吊销
	ErrCodeAccountDisabled  = 10003 // 账户已停用
	ErrCodeForbidden        = 10004 // 无权限
	ErrCodeRoleInsufficient = 10005 // 角色权限不足
	ErrCodeCapabilityDenied = 10006 // 能力门控拒绝
)

// 2xxxx 参数校验
const (
	ErrCodeBadRequest      = 20001 // 请求参数格式错误
	ErrCodeMissingParam    = 20002 // 缺少必要参数
	ErrCodeInvalidParam    = 20003 // 参数值不合法
	ErrCodePayloadTooLarge = 20004 // 请求体过大
)

// 3xxxx 业务逻辑
const (
	ErrCodeNotFound        = 30001 // 资源不存在
	ErrCodeDuplicate       = 30002 // 资源已存在（重复创建）
	ErrCodeQuotaExceeded   = 30003 // 配额已用尽
	ErrCodeProcessNotFound = 30004 // 办事流程未定义
	ErrCodeKBNoResult      = 30005 // 知识库无匹配结果
	ErrCodeFeedbackClosed  = 30006 // 反馈已关闭不可修改
	ErrCodeSessionNotFound = 30007 // 会话不存在
	ErrCodeLowConfidence   = 30008 // 置信度过低，已兜底
)

// 4xxxx 外部依赖
const (
	ErrCodeLLMUnavailable = 40001 // LLM 服务不可用
	ErrCodeLLMTimeout     = 40002 // LLM 请求超时
	ErrCodeLLMRateLimit   = 40003 // LLM 限流
	ErrCodeXuegongFailed  = 40004 // 学工系统对接失败
	ErrCodeYBTFailed      = 40005 // 一表通对接失败
	ErrCodeSSOFailed      = 40006 // SSO 认证失败
)

// 5xxxx 系统内部
const (
	ErrCodeInternal      = 50001 // 服务器内部错误
	ErrCodeDBError       = 50002 // 数据库操作失败
	ErrCodeConfigInvalid = 50003 // 配置校验失败
	ErrCodeMigrateFailed = 50004 // 数据库迁移失败
)

// errRegistry 错误码注册表（内部映射）
var errRegistry = map[int]*BizError{
	// 认证/授权
	ErrCodeUnauthorized:     {Code: ErrCodeUnauthorized, HTTPStatus: http.StatusUnauthorized, Message: "未登录或令牌已过期，请重新登录"},
	ErrCodeTokenRevoked:     {Code: ErrCodeTokenRevoked, HTTPStatus: http.StatusUnauthorized, Message: "令牌已被吊销，请重新登录"},
	ErrCodeAccountDisabled:  {Code: ErrCodeAccountDisabled, HTTPStatus: http.StatusForbidden, Message: "账户已被停用"},
	ErrCodeForbidden:        {Code: ErrCodeForbidden, HTTPStatus: http.StatusForbidden, Message: "无权限执行此操作"},
	ErrCodeRoleInsufficient: {Code: ErrCodeRoleInsufficient, HTTPStatus: http.StatusForbidden, Message: "当前角色权限不足"},
	ErrCodeCapabilityDenied: {Code: ErrCodeCapabilityDenied, HTTPStatus: http.StatusForbidden, Message: "该功能未对当前角色开放"},

	// 参数校验
	ErrCodeBadRequest:      {Code: ErrCodeBadRequest, HTTPStatus: http.StatusBadRequest, Message: "请求参数格式错误"},
	ErrCodeMissingParam:    {Code: ErrCodeMissingParam, HTTPStatus: http.StatusBadRequest, Message: "缺少必要参数"},
	ErrCodeInvalidParam:    {Code: ErrCodeInvalidParam, HTTPStatus: http.StatusBadRequest, Message: "参数值不合法"},
	ErrCodePayloadTooLarge: {Code: ErrCodePayloadTooLarge, HTTPStatus: http.StatusRequestEntityTooLarge, Message: "请求体过大"},

	// 业务逻辑
	ErrCodeNotFound:        {Code: ErrCodeNotFound, HTTPStatus: http.StatusNotFound, Message: "资源不存在"},
	ErrCodeDuplicate:       {Code: ErrCodeDuplicate, HTTPStatus: http.StatusConflict, Message: "资源已存在"},
	ErrCodeQuotaExceeded:   {Code: ErrCodeQuotaExceeded, HTTPStatus: http.StatusTooManyRequests, Message: "今日配额已用尽，请明天再试"},
	ErrCodeProcessNotFound: {Code: ErrCodeProcessNotFound, HTTPStatus: http.StatusNotFound, Message: "该办事流程暂未定义"},
	ErrCodeKBNoResult:      {Code: ErrCodeKBNoResult, HTTPStatus: http.StatusOK, Message: "未找到相关知识，已提供通用回答"},
	ErrCodeFeedbackClosed:  {Code: ErrCodeFeedbackClosed, HTTPStatus: http.StatusConflict, Message: "反馈已关闭，不可修改"},
	ErrCodeSessionNotFound: {Code: ErrCodeSessionNotFound, HTTPStatus: http.StatusNotFound, Message: "会话不存在或已过期"},
	ErrCodeLowConfidence:   {Code: ErrCodeLowConfidence, HTTPStatus: http.StatusOK, Message: "匹配置信度较低，建议咨询辅导员确认"},

	// 外部依赖
	ErrCodeLLMUnavailable: {Code: ErrCodeLLMUnavailable, HTTPStatus: http.StatusServiceUnavailable, Message: "AI 服务暂时不可用，请稍后重试"},
	ErrCodeLLMTimeout:     {Code: ErrCodeLLMTimeout, HTTPStatus: http.StatusGatewayTimeout, Message: "AI 响应超时，请稍后重试"},
	ErrCodeLLMRateLimit:   {Code: ErrCodeLLMRateLimit, HTTPStatus: http.StatusTooManyRequests, Message: "AI 服务繁忙，请稍后重试"},
	ErrCodeXuegongFailed:  {Code: ErrCodeXuegongFailed, HTTPStatus: http.StatusBadGateway, Message: "学工系统对接失败"},
	ErrCodeYBTFailed:      {Code: ErrCodeYBTFailed, HTTPStatus: http.StatusBadGateway, Message: "一表通系统对接失败"},
	ErrCodeSSOFailed:      {Code: ErrCodeSSOFailed, HTTPStatus: http.StatusBadGateway, Message: "统一认证服务异常"},

	// 系统内部
	ErrCodeInternal:      {Code: ErrCodeInternal, HTTPStatus: http.StatusInternalServerError, Message: "服务器内部错误"},
	ErrCodeDBError:       {Code: ErrCodeDBError, HTTPStatus: http.StatusInternalServerError, Message: "数据库操作异常"},
	ErrCodeConfigInvalid: {Code: ErrCodeConfigInvalid, HTTPStatus: http.StatusInternalServerError, Message: "系统配置异常"},
	ErrCodeMigrateFailed: {Code: ErrCodeMigrateFailed, HTTPStatus: http.StatusInternalServerError, Message: "数据库迁移失败"},
}

// LookupError 按错误码查询注册表（供中间件 / 工具函数使用）
func LookupError(code int) (*BizError, bool) {
	e, ok := errRegistry[code]
	return e, ok
}
