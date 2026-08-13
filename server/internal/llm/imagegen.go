package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
)

// ImageGenClient 文生图/图生图客户端（智谱 CogView-3，OpenAI images 兼容）。
// 用于数字孪生画像生成：以用户照片为原型（图生图）或内置原型（文生图）。
type ImageGenClient interface {
	// Generate 生成一张图片。
	// prompt: 文生图提示词；refImageData: 参考图 base64（图生图时非空），mime 为参考图类型
	Generate(ctx context.Context, prompt string, refImageData []byte, mime string) ([]byte, error)
	Name() string
}

// ZhipuCogViewClient 智谱 CogView 文生图客户端
type ZhipuCogViewClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewZhipuCogViewClient 创建 CogView 客户端。
// API Key 优先取 ZHIPU_COGVIEW_API_KEY，未配置则复用 ZHIPU_API_KEY。
func NewZhipuCogViewClient(cfg *config.Config) *ZhipuCogViewClient {
	key := cfg.ZhipuCogViewAPIKey
	if key == "" {
		key = cfg.ZhipuAPIKey
	}
	return &ZhipuCogViewClient{
		apiKey:  key,
		baseURL: cfg.ZhipuCogViewBaseURL,
		model:   cfg.ZhipuCogViewModel,
		client: &http.Client{
			Timeout: 180 * time.Second, // 图片生成较慢
		},
	}
}

// Name 返回客户端名称
func (c *ZhipuCogViewClient) Name() string { return "zhipu-cogview" }

// cogViewRequest CogView 请求体（OpenAI images 兼容）
type cogViewRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Size   string `json:"size,omitempty"`
	// 图生图：image 为 base64（data URL 或纯 base64）
	Image string `json:"image,omitempty"`
	// 可选：参考图强度
	ReferenceImageStrength float64 `json:"reference_image_strength,omitempty"`
	ResponseFormat         string  `json:"response_format,omitempty"`
}

// cogViewResponse CogView 响应体
type cogViewResponse struct {
	Data []struct {
		URL       string `json:"url"`
		B64JSON   string `json:"b64_json"`
		RevPrompt string `json:"revised_prompt"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Generate 生成图片。refImageData 为空时走文生图；非空时走图生图。
func (c *ZhipuCogViewClient) Generate(ctx context.Context, prompt string, refImageData []byte, mime string) ([]byte, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("CogView API Key 未配置")
	}
	req := cogViewRequest{
		Model:  c.model,
		Prompt: prompt,
		// 256x256：头像/画像展示场景足够清晰，显著降低响应体积/成本/时延
		Size: "256x256",
	}
	if len(refImageData) > 0 {
		// 图生图：转为 data URL base64
		req.Image = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(refImageData)
		req.ReferenceImageStrength = 0.7
	}
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	if tid := middleware.GetTraceIDFromContext(ctx); tid != "" {
		httpReq.Header.Set("X-Trace-ID", tid)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("CogView 请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var parsed cogViewResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("解析响应失败(HTTP %d): %s", resp.StatusCode, truncate(string(respBody), 300))
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("CogView 错误: %s (%s)", parsed.Error.Message, parsed.Error.Code)
	}
	if resp.StatusCode != http.StatusOK || len(parsed.Data) == 0 {
		return nil, fmt.Errorf("CogView 错误 (HTTP %d): %s", resp.StatusCode, truncate(string(respBody), 300))
	}

	item := parsed.Data[0]
	if item.B64JSON != "" {
		data, err := base64.StdEncoding.DecodeString(item.B64JSON)
		if err != nil {
			return nil, fmt.Errorf("解码 b64 图片失败: %w", err)
		}
		return data, nil
	}
	if item.URL != "" {
		imgResp, err := c.client.Get(item.URL)
		if err != nil {
			return nil, fmt.Errorf("下载生成图片失败: %w", err)
		}
		defer imgResp.Body.Close()
		return io.ReadAll(io.LimitReader(imgResp.Body, 10<<20))
	}
	return nil, fmt.Errorf("CogView 响应无图片数据")
}
