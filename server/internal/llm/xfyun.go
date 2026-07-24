package llm

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/gorilla/websocket"
)

// ── 讯飞语音客户端 ──
// 封装讯飞星火 ASR（语音识别）和 TTS（语音合成）WebSocket API。
// 所有语音操作通过后端代理，API Secret 不暴露到前端。

// iFlytek ASR / TTS 服务地址
const (
	iFlytekASRHost = "iat-api.xfyun.cn"
	iFlytekASRPath = "/v2/iat"
	iFlytekTTSHost = "tts-api.xfyun.cn"
	iFlytekTTSPath = "/v2/tts"
)

// XfyunClient 讯飞语音客户端
type XfyunClient struct {
	appID     string
	apiKey    string
	apiSecret string
}

// NewXfyunClient 创建讯飞客户端实例
func NewXfyunClient(cfg *config.Config) *XfyunClient {
	return &XfyunClient{
		appID:     cfg.XfyunAppID,
		apiKey:    cfg.XfyunAPIKey,
		apiSecret: cfg.XfyunAPISecret,
	}
}

// Name 返回客户端名称（实现 ChatClient 接口，用于标识）
func (c *XfyunClient) Name() string {
	return "xfyun"
}

// ── ASR 语音识别 ──

// ASR 将 PCM 16kHz 16-bit 单声道音频识别为文本。
// audioBytes 为原始 PCM 音频数据（不是 base64）。
func (c *XfyunClient) ASR(ctx context.Context, audioBytes []byte) (string, error) {
	wsURL, err := buildAuthURL(iFlytekASRHost, iFlytekASRPath, c.apiKey, c.apiSecret)
	if err != nil {
		return "", fmt.Errorf("构造 ASR 鉴权 URL 失败: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return "", fmt.Errorf("连接讯飞 ASR WebSocket 失败: %w", err)
	}
	defer conn.Close()
	done := make(chan struct{})
	defer close(done)
	go closeWebSocketOnCancel(ctx, conn, done)

	// 分片参数：1280 字节 = 80ms @ 16kHz 16-bit
	const chunkSize = 1280
	totalLen := len(audioBytes)

	// 用于收集识别结果的文本
	var fullText strings.Builder

	// 发送音频分片
	for offset := 0; offset < totalLen; offset += chunkSize {
		end := offset + chunkSize
		if end > totalLen {
			end = totalLen
		}
		chunk := audioBytes[offset:end]
		status := 1 // 中间帧
		if offset == 0 {
			status = 0 // 首帧
		}
		if end >= totalLen {
			status = 2 // 末帧
		}

		frame, err := c.buildASRFrame(chunk, status)
		if err != nil {
			return "", fmt.Errorf("构造 ASR 帧失败: %w", err)
		}

		if err := conn.WriteMessage(websocket.TextMessage, frame); err != nil {
			return "", fmt.Errorf("发送 ASR 音频帧失败: %w", err)
		}

		// 每发送一帧，读取可能的响应（非阻塞式，仅检查是否有结果返回）
		// 讯飞会在最后一帧发送完后返回完整结果
	}

	// 读取识别结果
	var resultText string
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			// WebSocket 关闭，正常结束
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				break
			}
			return fullText.String(), fmt.Errorf("读取 ASR 结果失败: %w", err)
		}

		text, done, err := parseASRResult(msg)
		if err != nil {
			return fullText.String(), fmt.Errorf("解析 ASR 结果失败: %w", err)
		}
		if text != "" {
			resultText = text
		}
		if done {
			break
		}
	}

	result := resultText
	if result == "" {
		return "", fmt.Errorf("未识别到语音内容")
	}
	return result, nil
}

// buildASRFrame 构造 ASR 请求帧（JSON）
func (c *XfyunClient) buildASRFrame(chunk []byte, status int) ([]byte, error) {
	audioB64 := base64.StdEncoding.EncodeToString(chunk)

	type asrCommon struct {
		AppID string `json:"app_id"`
	}
	type asrBusiness struct {
		Language string `json:"language"`
		Domain   string `json:"domain"`
		Accent   string `json:"accent"`
		VadEOS   int    `json:"vad_eos"`
	}
	type asrData struct {
		Status   int    `json:"status"`
		Format   string `json:"format"`
		Encoding string `json:"encoding"`
		Audio    string `json:"audio"`
	}

	frame := struct {
		Common   asrCommon   `json:"common"`
		Business asrBusiness `json:"business"`
		Data     asrData     `json:"data"`
	}{
		Common: asrCommon{AppID: c.appID},
		Business: asrBusiness{
			Language: "zh_cn",
			Domain:   "iat",
			Accent:   "mandarin",
			VadEOS:   10000,
		},
		Data: asrData{
			Status:   status,
			Format:   "audio/L16;rate=16000",
			Encoding: "raw",
			Audio:    audioB64,
		},
	}

	return json.Marshal(frame)
}

// parseASRResult 解析讯飞 ASR 返回的 JSON 结果
// 返回：识别文本、是否结束、错误
func parseASRResult(msg []byte) (text string, done bool, err error) {
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Status int `json:"status"`
			Result struct {
				Ws []struct {
					Cw []struct {
						W  string `json:"w"`
						Wp string `json:"wp"`
					} `json:"cw"`
				} `json:"ws"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.Unmarshal(msg, &resp); err != nil {
		return "", false, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	if resp.Code != 0 {
		return "", true, fmt.Errorf("讯飞 ASR 错误 (code=%d): %s", resp.Code, resp.Message)
	}

	// 拼接识别结果
	var sb strings.Builder
	for _, ws := range resp.Data.Result.Ws {
		for _, cw := range ws.Cw {
			sb.WriteString(cw.W)
		}
	}

	// status 2 表示识别结束
	done = resp.Data.Status == 2

	return sb.String(), done, nil
}

// ── TTS 语音合成 ──

// TTS 将文本合成为 MP3 音频字节。
// voiceName 可选发音人，默认 "x_xiaoyan"（讯飞小燕）。
func (c *XfyunClient) TTS(ctx context.Context, text string, voiceName string) ([]byte, error) {
	if voiceName == "" {
		voiceName = "x_xiaoyan"
	}

	wsURL, err := buildAuthURL(iFlytekTTSHost, iFlytekTTSPath, c.apiKey, c.apiSecret)
	if err != nil {
		return nil, fmt.Errorf("构造 TTS 鉴权 URL 失败: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("连接讯飞 TTS WebSocket 失败: %w", err)
	}
	defer conn.Close()
	done := make(chan struct{})
	defer close(done)
	go closeWebSocketOnCancel(ctx, conn, done)

	// 构造并发送 TTS 请求帧
	frame, err := c.buildTTSFrame(text, voiceName)
	if err != nil {
		return nil, fmt.Errorf("构造 TTS 帧失败: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, frame); err != nil {
		return nil, fmt.Errorf("发送 TTS 请求失败: %w", err)
	}

	// 收集音频数据
	var audioBuf []byte

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				break
			}
			return audioBuf, fmt.Errorf("读取 TTS 音频数据失败: %w", err)
		}

		audioData, done, err := parseTTSResult(msg)
		if err != nil {
			return audioBuf, fmt.Errorf("解析 TTS 结果失败: %w", err)
		}
		audioBuf = append(audioBuf, audioData...)

		if done {
			break
		}
	}

	if len(audioBuf) == 0 {
		return nil, fmt.Errorf("TTS 未返回音频数据")
	}

	return audioBuf, nil
}

func closeWebSocketOnCancel(ctx context.Context, conn *websocket.Conn, done <-chan struct{}) {
	select {
	case <-ctx.Done():
		_ = conn.Close()
	case <-done:
	}
}

// buildTTSFrame 构造 TTS 请求帧（JSON）
func (c *XfyunClient) buildTTSFrame(text string, voiceName string) ([]byte, error) {
	textB64 := base64.StdEncoding.EncodeToString([]byte(text))

	type ttsCommon struct {
		AppID string `json:"app_id"`
	}
	type ttsBusiness struct {
		Aue    string `json:"aue"`
		Vcn    string `json:"vcn"`
		Pitch  int    `json:"pitch"`
		Speed  int    `json:"speed"`
		Volume int    `json:"volume"`
		Tte    string `json:"tte"`
	}
	type ttsData struct {
		Status int    `json:"status"`
		Text   string `json:"text"`
	}

	frame := struct {
		Common   ttsCommon   `json:"common"`
		Business ttsBusiness `json:"business"`
		Data     ttsData     `json:"data"`
	}{
		Common: ttsCommon{AppID: c.appID},
		Business: ttsBusiness{
			Aue:    "lame", // MP3 格式
			Vcn:    voiceName,
			Pitch:  50,
			Speed:  50,
			Volume: 50,
			Tte:    "utf8",
		},
		Data: ttsData{
			Status: 2, // 一次性发送全部文本
			Text:   textB64,
		},
	}

	return json.Marshal(frame)
}

// parseTTSResult 解析讯飞 TTS 返回的 JSON 结果
// 返回：音频字节（base64 解码后）、是否结束、错误
func parseTTSResult(msg []byte) ([]byte, bool, error) {
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Audio  string `json:"audio"`
			Status int    `json:"status"`
			Ced    string `json:"ced"`
		} `json:"data"`
	}

	if err := json.Unmarshal(msg, &resp); err != nil {
		return nil, false, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	if resp.Code != 0 {
		return nil, true, fmt.Errorf("讯飞 TTS 错误 (code=%d): %s", resp.Code, resp.Message)
	}

	audio, err := base64.StdEncoding.DecodeString(resp.Data.Audio)
	if err != nil {
		return nil, false, fmt.Errorf("解码 TTS 音频数据失败: %w", err)
	}

	// status 2 表示传输结束
	done := resp.Data.Status == 2

	return audio, done, nil
}

// ── HMAC-SHA256 鉴权 ──

// buildAuthURL 构造带鉴权参数的 WebSocket URL
// 讯飞 WebSocket API 使用 URL 参数传递鉴权信息（authorization + date + host）
func buildAuthURL(host, path, apiKey, apiSecret string) (string, error) {
	now := time.Now().UTC()
	dateStr := now.Format(http.TimeFormat) // RFC 1123

	// 签名原文
	signatureOrigin := fmt.Sprintf("host: %s\ndate: %s\nGET %s HTTP/1.1", host, dateStr, path)

	// HMAC-SHA256 签名
	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(signatureOrigin))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// authorization = base64(api_key="...", algorithm="hmac-sha256", headers="host date request-line", signature="...")
	authOrigin := fmt.Sprintf(
		`api_key="%s", algorithm="hmac-sha256", headers="host date request-line", signature="%s"`,
		apiKey, signature,
	)
	auth := base64.StdEncoding.EncodeToString([]byte(authOrigin))

	// 构造 URL（手动拼接 query string，用 url.QueryEscape 确保空格编码为 %20）
	// 注意：url.Values.Encode() 将空格编码为 '+'，讯飞 API 要求 '%20'
	queryString := fmt.Sprintf(
		"authorization=%s&date=%s&host=%s",
		url.QueryEscape(auth),
		url.QueryEscape(dateStr),
		url.QueryEscape(host),
	)

	return fmt.Sprintf("wss://%s%s?%s", host, path, queryString), nil
}
