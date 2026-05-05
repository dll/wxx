package handler

import (
	"net/http"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// agentService 智能体管理服务接口（用于测试 mock）
type agentService interface {
	List() ([]*model.Agent, error)
	Create(req *model.AgentCreateRequest) (*model.Agent, error)
	Get(agentID string) (*model.Agent, error)
	Update(agentID string, req *model.AgentUpdateRequest) (*model.Agent, error)
	Delete(agentID string) error
}

// AgentHandler 智能体管理 HTTP handler
type AgentHandler struct {
	agentSvc agentService
}

// NewAgentHandler 创建智能体管理 handler
func NewAgentHandler(agentSvc *service.AgentService) *AgentHandler {
	return &AgentHandler{agentSvc: agentSvc}
}

// List 列出所有智能体
// GET /api/v1/agents
func (h *AgentHandler) List(c *gin.Context) {
	agents, err := h.agentSvc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询智能体列表失败，请稍后重试",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}
	if agents == nil {
		agents = []*model.Agent{}
	}

	c.JSON(http.StatusOK, model.AgentListResponse{
		Code:    0,
		Message: "success",
		Data:    agents,
		Total:   len(agents),
	})
}

// Create 创建智能体
// POST /api/v1/agents
func (h *AgentHandler) Create(c *gin.Context) {
	var req model.AgentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	agent, err := h.agentSvc.Create(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "创建智能体失败，请稍后重试",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, model.AgentDetailResponse{
		Code:    0,
		Message: "创建成功",
		Data:    agent,
	})
}

// Get 获取单个智能体
// GET /api/v1/agents/:id
func (h *AgentHandler) Get(c *gin.Context) {
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "智能体 ID 不能为空",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	agent, err := h.agentSvc.Get(agentID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Code:    404,
			Message: "智能体不存在",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, model.AgentDetailResponse{
		Code:    0,
		Message: "success",
		Data:    agent,
	})
}

// Update 更新智能体
// PUT /api/v1/agents/:id
func (h *AgentHandler) Update(c *gin.Context) {
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "智能体 ID 不能为空",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	var req model.AgentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	agent, err := h.agentSvc.Update(agentID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "更新智能体失败，请稍后重试",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, model.AgentDetailResponse{
		Code:    0,
		Message: "更新成功",
		Data:    agent,
	})
}

// Delete 删除智能体
// DELETE /api/v1/agents/:id
func (h *AgentHandler) Delete(c *gin.Context) {
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "智能体 ID 不能为空",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	if err := h.agentSvc.Delete(agentID); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "删除智能体失败，请稍后重试",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}
