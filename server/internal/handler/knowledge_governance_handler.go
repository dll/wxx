package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// KnowledgeGovernanceHandler 知识治理智能体接口
type KnowledgeGovernanceHandler struct {
	svc *service.KnowledgeGovernanceService
}

func NewKnowledgeGovernanceHandler(svc *service.KnowledgeGovernanceService) *KnowledgeGovernanceHandler {
	return &KnowledgeGovernanceHandler{svc: svc}
}

// GovernanceRun 执行一次知识治理审计。
// GET /api/v1/kb/governance?scope=&owner_id=&with_llm=1&limit=200
// scope: 可选 school/college/class（空=全量）；owner_id 可选范围限定。
// with_llm: 1 时尝试 LLM 准确性增强审计（未注入 LLM 则自动回落为确定性检查）。
// 权限：管理员及以上（与知识治理同类管理能力）。
func (h *KnowledgeGovernanceHandler) GovernanceRun(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	scope := c.Query("scope")
	ownerID := c.Query("owner_id")
	withLLM := c.Query("with_llm") == "1"
	limit := 200
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	result := h.svc.GovernanceAudit(c.Request.Context(), scope, ownerID, withLLM, limit)
	c.JSON(http.StatusOK, model.KnowledgeGovernanceResponse{
		Code:    0,
		Message: "success",
		Data:    result,
	})
}
