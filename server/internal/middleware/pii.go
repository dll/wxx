package middleware

import (
	"bytes"
	"io"
	"strings"

	"github.com/dll/wxx/server/internal/util"

	"github.com/gin-gonic/gin"
)

// PIIMask 中间件：检测请求体中的 PII 并记录审计日志
// 注意：本中间件不修改请求体，仅检测和记录 PII 存在情况
// 实际脱敏在 service 层调用 SanitizeForLLM 时完成
func PIIMask() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 仅检测有请求体的 POST/PUT/PATCH 请求
		if c.Request.Method != "POST" && c.Request.Method != "PUT" && c.Request.Method != "PATCH" {
			c.Next()
			return
		}

		// 跳过文件上传
		contentType := c.Request.Header.Get("Content-Type")
		if strings.Contains(contentType, "multipart/form-data") {
			c.Next()
			return
		}

		// 读取请求体
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}

		// 恢复请求体供后续 handler 使用
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// 检测 PII
		bodyStr := string(bodyBytes)
		if len(bodyStr) > 0 && util.DetectPII(bodyStr) {
			// 记录 PII 检测到的事实（用于审计，不做拦截）
			result := util.MaskPIIWithDetail(bodyStr)
			c.Set("pii_detected", true)
			c.Set("pii_types", result.PIITypesFound)
		}

		c.Next()
	}
}
