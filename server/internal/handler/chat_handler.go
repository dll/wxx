package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/util"
	"github.com/gin-gonic/gin"
)

// ChatHandler 对话 handler
type ChatHandler struct {
	chatSvc     *service.ChatService
	emotionSvc  *service.EmotionService      // 可选：情感分析服务
	metricsRepo *repository.ChatMetricsRepo  // 可选：质量指标写入
}

// NewChatHandler 创建对话 handler
func NewChatHandler(chatSvc *service.ChatService) *ChatHandler {
	return &ChatHandler{chatSvc: chatSvc}
}

// SetEmotionService 设置情感分析服务（可选）
func (h *ChatHandler) SetEmotionService(emotionSvc *service.EmotionService) {
	h.emotionSvc = emotionSvc
}

// SetMetricsRepo 设置质量指标 repo（可选）
func (h *ChatHandler) SetMetricsRepo(repo *repository.ChatMetricsRepo) {
	h.metricsRepo = repo
}

// Ask 处理对话请求
// POST /api/v1/chat
func (h *ChatHandler) Ask(c *gin.Context) {
	start := time.Now()

	// 解析请求
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	// 获取用户上下文
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		util.FailUnauthorized(c, "未认证")
		return
	}

	// 调用 service 层
	card, sessionID, err := h.chatSvc.Ask(c.Request.Context(), userCtx, req.SessionID, req.Question, req.AgentID)
	if err != nil {
		log.Printf("问答处理失败 trace=%s user=%s session=%s err=%v", middleware.GetTraceID(c), userCtx.Username, req.SessionID, err)
		util.FailInternalError(c, "问答处理失败："+err.Error())
		return
	}

	durationMs := time.Since(start).Milliseconds()

	// 异步写入质量指标（不阻塞响应）
	if h.metricsRepo != nil && card != nil {
		go func() {
			fallback := card.Fallback
			_ = h.metricsRepo.Insert(&repository.ChatMetric{
				SessionID:    sessionID,
				UserID:       userCtx.UserID,
				Question:     req.Question,
				Intent:       "", // 意图由 context_engine 内部分类，此处暂不传递
				Confidence:   card.Confidence,
				Fallback:     fallback,
				SourcesCount: len(card.Sources),
				DurationMs:   durationMs,
				TraceID:      middleware.GetTraceID(c),
			})
		}()
	}

	// 异步情感分析（不阻塞回复，失败不影响聊天体验）
	if h.emotionSvc != nil {
		go func() {
			_, err := h.emotionSvc.AnalyzeAndLog(
				context.Background(),
				userCtx.UserID,
				userCtx.Username,
				sessionID,
				req.Question,
			)
			if err != nil {
				log.Printf("异步情感分析失败: %v", err)
			}
		}()
	}

	c.JSON(http.StatusOK, model.ChatResponse{
		Code:      0,
		Message:   "success",
		Data:      card,
		SessionID: sessionID,
		TraceID:   middleware.GetTraceID(c),
	})
}
