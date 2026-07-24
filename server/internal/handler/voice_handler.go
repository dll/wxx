package handler

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

const (
	voiceASRTimeout = 18 * time.Second
	voiceTTSTimeout = 15 * time.Second
	maxVoiceBytes   = 8 << 20
)

// voiceClient 语音客户端接口（ASR + TTS），*llm.XfyunClient 已实现
type voiceClient interface {
	ASR(ctx context.Context, audioBytes []byte) (string, error)
	TTS(ctx context.Context, text string, voiceName string) ([]byte, error)
}

// VoiceHandler 语音处理 handler（ASR 语音识别 + TTS 语音合成）
type VoiceHandler struct {
	xfClient voiceClient
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
	if len(audioBytes) > maxVoiceBytes {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "音频文件过大，请控制在 30 秒以内",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	pcmBytes, err := normalizeASRAudio(audioBytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	// 调用讯飞 ASR
	ctx, cancel := context.WithTimeout(c.Request.Context(), voiceASRTimeout)
	defer cancel()
	text, err := h.xfClient.ASR(ctx, pcmBytes)
	if err != nil {
		status := http.StatusInternalServerError
		message := "语音识别失败，请稍后重试"
		if errorsIsTimeout(err) {
			status = http.StatusGatewayTimeout
			message = "语音识别超时，请缩短录音后重试"
		}
		c.JSON(status, model.ErrorResponse{
			Code:    status,
			Message: message,
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), voiceTTSTimeout)
	defer cancel()

	// 调用讯飞 TTS
	audioData, err := h.xfClient.TTS(ctx, req.Text, req.Voice)
	if err != nil {
		status := http.StatusInternalServerError
		message := "语音合成失败，请稍后重试"
		if errorsIsTimeout(err) {
			status = http.StatusGatewayTimeout
			message = "语音合成超时，请缩短文本后重试"
		}
		c.JSON(status, model.ErrorResponse{
			Code:    status,
			Message: message,
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	// 返回 MP3 音频流（Gin 的 c.Data 会自动设置 Content-Length）
	c.Data(http.StatusOK, "audio/mpeg", audioData)
}

func normalizeASRAudio(audioBytes []byte) ([]byte, error) {
	if len(audioBytes) < 12 {
		return audioBytes, nil
	}
	if string(audioBytes[0:4]) != "RIFF" || string(audioBytes[8:12]) != "WAVE" {
		return audioBytes, nil
	}

	for offset := 12; offset+8 <= len(audioBytes); {
		chunkID := string(audioBytes[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(audioBytes[offset+4 : offset+8]))
		dataStart := offset + 8
		dataEnd := dataStart + chunkSize
		if dataEnd > len(audioBytes) {
			return nil, errors.New("WAV 音频数据不完整，请重新录音")
		}
		if chunkID == "data" {
			if chunkSize == 0 {
				return nil, errors.New("音频文件为空")
			}
			return audioBytes[dataStart:dataEnd], nil
		}
		offset = dataEnd
		if offset%2 == 1 {
			offset++
		}
	}

	return nil, errors.New("未找到 WAV 音频数据，请重新录音")
}

func errorsIsTimeout(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), context.DeadlineExceeded.Error())
}
