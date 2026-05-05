package handler

import (
	"io"
	"net/http"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// VoiceHandler 语音处理 handler（ASR 语音识别 + TTS 语音合成）
type VoiceHandler struct {
	xfClient *llm.XfyunClient
}

// NewVoiceHandler 创建语音 handler
func NewVoiceHandler(xfClient *llm.XfyunClient) *VoiceHandler {
	return &VoiceHandler{xfClient: xfClient}
}

// ASR 语音识别接口
// POST /api/v1/voice/asr
// 接收 multipart/form-data，字段名 "audio"，上传 PCM 16kHz 16-bit 单声道音频文件
func (h *VoiceHandler) ASR(c *gin.Context) {
	// 从 multipart form 读取音频文件
	file, _, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "请上传音频文件（字段名 audio）",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}
	defer file.Close()

	audioBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "读取音频文件失败",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	if len(audioBytes) == 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "音频文件为空",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	// 调用讯飞 ASR
	text, err := h.xfClient.ASR(c.Request.Context(), audioBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "语音识别失败：" + err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "识别成功",
		"data": gin.H{
			"text":    text,
			"success": true,
		},
	})
}

// ttsRequest TTS 请求参数
type ttsRequest struct {
	Text  string `json:"text" binding:"required"`
	Voice string `json:"voice"`
}

// TTS 语音合成接口
// POST /api/v1/voice/tts
// 接收 JSON：{"text": "...", "voice": "x_xiaoyan"}，返回 audio/mpeg 二进制流
func (h *VoiceHandler) TTS(c *gin.Context) {
	var req ttsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "请求参数错误：" + err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	// 调用讯飞 TTS
	audioData, err := h.xfClient.TTS(c.Request.Context(), req.Text, req.Voice)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "语音合成失败：" + err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	// 返回 MP3 音频流（Gin 的 c.Data 会自动设置 Content-Length）
	c.Data(http.StatusOK, "audio/mpeg", audioData)
}
