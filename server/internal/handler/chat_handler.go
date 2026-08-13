package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/util"
	"github.com/gin-gonic/gin"
)

// ChatHandler 对话 handler
type ChatHandler struct {
	chatSvc    *service.ChatService
	emotionSvc *service.EmotionService     // 可选：情感分析服务
	metricsSvc *service.ChatMetricsService // 可选：质量指标写入
}

// NewChatHandler 创建对话 handler
func NewChatHandler(chatSvc *service.ChatService) *ChatHandler {
	return &ChatHandler{chatSvc: chatSvc}
}

// SetEmotionService 设置情感分析服务（可选）
func (h *ChatHandler) SetEmotionService(emotionSvc *service.EmotionService) {
	h.emotionSvc = emotionSvc
}

// SetMetricsService 设置质量指标 service（可选）
func (h *ChatHandler) SetMetricsService(svc *service.ChatMetricsService) {
	h.metricsSvc = svc
}

// Ask 处理对话请求
// POST /api/v1/chat
func (h *ChatHandler) Ask(c *gin.Context) {
	start := time.Now()

	// 解析请求
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("chat Ask bind err: %v", err)
		util.FailBadRequest(c, "请求参数错误")
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
		util.FailInternalError(c, "问答处理失败，请稍后重试")
		return
	}

	durationMs := time.Since(start).Milliseconds()

	// 异步写入质量指标（不阻塞响应）
	if h.metricsSvc != nil && card != nil {
		// 安全：gin.Context 在请求结束后被池复用，先拷贝 traceID 再进 goroutine
		traceID := middleware.GetTraceID(c)
		go func() {
			_ = h.metricsSvc.Insert(
				sessionID,
				userCtx.UserID,
				req.Question,
				"",
				card.Confidence,
				card.Fallback,
				len(card.Sources),
				durationMs,
				traceID,
			)
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

// Stream 流式对话（SSE）
// POST /api/v1/chat/stream
// 事件流：data: {"delta":"..."} 逐块增量 → data: {"done":true,"card":{...},"session_id":"..."}
func (h *ChatHandler) Stream(c *gin.Context) {
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("chat stream bind err: %v", err)
		util.FailBadRequest(c, "请求参数错误")
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		util.FailUnauthorized(c, "未认证")
		return
	}

	// SSE 响应头
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		util.FailInternalError(c, "当前环境不支持流式输出")
		return
	}

	writeEvent := func(payload string) {
		_, _ = c.Writer.Write([]byte("data: " + payload + "\n\n"))
		flusher.Flush()
	}

	var card *model.AnswerCard
	var sessionID string

	// 首帧握手
	writeEvent(`{"type":"start","trace_id":"` + middleware.GetTraceID(c) + `"}`)

	start := time.Now()
	var err error
	card, sessionID, err = h.chatSvc.AskStream(
		c.Request.Context(),
		userCtx,
		req.SessionID,
		req.Question,
		req.AgentID,
		func(delta string) {
			// 增量事件（JSON 转义）
			b, _ := json.Marshal(map[string]string{"type": "delta", "delta": delta})
			writeEvent(string(b))
		},
	)
	if err != nil {
		log.Printf("流式问答失败 trace=%s user=%s err=%v", middleware.GetTraceID(c), userCtx.Username, err)
		b, _ := json.Marshal(map[string]interface{}{"type": "error", "message": "问答处理失败，请稍后重试"})
		writeEvent(string(b))
		return
	}

	// 结束事件：携带完整 AnswerCard
	b, _ := json.Marshal(map[string]interface{}{
		"type": "done", "card": card, "session_id": sessionID, "trace_id": middleware.GetTraceID(c),
	})
	writeEvent(string(b))

	durationMs := time.Since(start).Milliseconds()
	if h.metricsSvc != nil && card != nil {
		traceID := middleware.GetTraceID(c)
		go func() {
			_ = h.metricsSvc.Insert(sessionID, userCtx.UserID, req.Question, "",
				card.Confidence, card.Fallback, len(card.Sources), durationMs, traceID)
		}()
	}
	if h.emotionSvc != nil {
		go func() {
			if _, err := h.emotionSvc.AnalyzeAndLog(context.Background(), userCtx.UserID, userCtx.Username, sessionID, req.Question); err != nil {
				log.Printf("异步情感分析失败: %v", err)
			}
		}()
	}
}
