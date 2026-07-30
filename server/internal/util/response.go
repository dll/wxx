package util

import (
	"errors"
	"net/http"

	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// ApiResponse 统一 API 响应结构体
type ApiResponse struct {
	Code    int         `json:"code"`               // 状态码：0=成功，其他=失败
	Message string      `json:"message"`            // 状态描述
	Data    interface{} `json:"data,omitempty"`     // 响应数据（成功时返回）
	TraceID string      `json:"trace_id,omitempty"` // 追踪 ID（用于链路追踪）
}

// getTraceID 从 gin.Context 中获取 TraceID
func getTraceID(c *gin.Context) string {
	if v, exists := c.Get("trace_id"); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Success 成功响应（code=0, message="success"）
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, ApiResponse{
		Code:    0,
		Message: "success",
		Data:    data,
		TraceID: getTraceID(c),
	})
}

// SuccessWithMessage 成功响应（自定义 message）
func SuccessWithMessage(c *gin.Context, data interface{}, msg string) {
	c.JSON(http.StatusOK, ApiResponse{
		Code:    0,
		Message: msg,
		Data:    data,
		TraceID: getTraceID(c),
	})
}

// Fail 失败响应（自定义 code 和 message）
func Fail(c *gin.Context, code int, message string) {
	statusCode := code
	if statusCode < 100 || statusCode > 599 {
		statusCode = http.StatusInternalServerError
	}
	c.JSON(statusCode, ApiResponse{
		Code:    code,
		Message: message,
		TraceID: getTraceID(c),
	})
}

// FailBadRequest 400 错误响应（请求参数错误）
func FailBadRequest(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, message)
}

// FailUnauthorized 401 错误响应（未认证）
func FailUnauthorized(c *gin.Context, message string) {
	Fail(c, http.StatusUnauthorized, message)
}

// FailForbidden 403 错误响应（无权限）
func FailForbidden(c *gin.Context, message string) {
	Fail(c, http.StatusForbidden, message)
}

// FailNotFound 404 错误响应（资源不存在）
func FailNotFound(c *gin.Context, message string) {
	Fail(c, http.StatusNotFound, message)
}

// FailInternalError 500 错误响应（服务器内部错误）
func FailInternalError(c *gin.Context, message string) {
	Fail(c, http.StatusInternalServerError, message)
}

// FailBizError 业务错误响应（A-01 错误码注册表适配）
// 从 BizError 中提取业务码和 HTTP 状态码，返回统一格式
func FailBizError(c *gin.Context, bizErr *model.BizError) {
	c.JSON(bizErr.HTTPStatus, ApiResponse{
		Code:    bizErr.Code,
		Message: bizErr.Message,
		TraceID: getTraceID(c),
	})
}

// FailFromError 从 error 自动识别 BizError 或回退为 500
func FailFromError(c *gin.Context, err error) {
	var bizErr *model.BizError
	if errors.As(err, &bizErr) {
		FailBizError(c, bizErr)
		return
	}
	FailInternalError(c, "服务器内部错误")
}
