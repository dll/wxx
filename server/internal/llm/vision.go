package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
)

// maxOCRImageDim 单张图片 OCR 前的最大边长（像素）。扫描件多为 300dpi A4（约 2480×3508），
// 直接送视觉模型易超图片/上下文限制，先等比缩小。
const maxOCRImageDim = 1280

// OCRImage 待识别的图片（用于扫描件 OCR）
type OCRImage struct {
	Data []byte // 图片二进制
	MIME string // "image/jpeg" / "image/png" 等
}

// VisionClient 视觉（多模态）模型客户端，用于扫描件 OCR。
// 与 ChatClient 相互独立：DeepSeek/Mock 不实现视觉，仅智谱 GLM-4V 实现。
type VisionClient interface {
	// OCR 识别一组图片中的文字，按传入顺序拼接返回
	OCR(ctx context.Context, images []OCRImage) (string, error)
	Name() string
}

// Zhipu4VClient 智谱 GLM-4V 视觉客户端（OpenAI 兼容 /chat/completions）。
// 用于扫描件 OCR：把图片以 data URL 形式放入消息 content 数组。
type Zhipu4VClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewZhipu4VClient 创建智谱 GLM-4V 客户端。
// API Key 优先取 ZHIPU_4V_API_KEY，未配置则复用 ZHIPU_API_KEY。
func NewZhipu4VClient(cfg *config.Config) *Zhipu4VClient {
	key := cfg.Zhipu4VAPIKEY
	if key == "" {
		key = cfg.ZhipuAPIKey
	}
	return &Zhipu4VClient{
		apiKey:  key,
		baseURL: cfg.ZhipuBaseURL,
		model:   cfg.Zhipu4VModel,
		client: &http.Client{
			Timeout: 120 * time.Second, // 多图识别耗时较长
		},
	}
}

// Name 返回客户端名称
func (c *Zhipu4VClient) Name() string {
	return "zhipu-glm4v"
}

// ocrContentPart 消息 content 数组元素（text / image_url）
type ocrContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL *struct {
		URL string `json:"url"`
	} `json:"image_url,omitempty"`
}

type ocrVisionMessage struct {
	Role    string           `json:"role"`
	Content []ocrContentPart `json:"content"`
}

type ocrVisionRequest struct {
	Model    string             `json:"model"`
	Messages []ocrVisionMessage `json:"messages"`
}

// OCR 识别一组图片中的文字，按传入顺序拼接。单请求最多携带 4 张图，超出分批。
// 失败批次自动降级为逐张识别，最大限度保留可识别内容。
func (c *Zhipu4VClient) OCR(ctx context.Context, images []OCRImage) (string, error) {
	if len(images) == 0 {
		return "", fmt.Errorf("没有可识别的图片")
	}
	const maxPerBatch = 4
	var out bytes.Buffer
	for start := 0; start < len(images); start += maxPerBatch {
		end := start + maxPerBatch
		if end > len(images) {
			end = len(images)
		}
		text, err := c.ocrBatch(ctx, images[start:end])
		if err == nil {
			if text != "" {
				out.WriteString(text)
				out.WriteString("\n")
			}
			continue
		}
		// 分批失败（某张图超限/解码异常）→ 逐张识别，保留成功项
		for _, img := range images[start:end] {
			if t, e2 := c.ocrBatch(ctx, []OCRImage{img}); e2 == nil && t != "" {
				out.WriteString(t)
				out.WriteString("\n")
			}
		}
	}
	if out.Len() == 0 {
		return "", fmt.Errorf("OCR 未识别出有效文本")
	}
	return out.String(), nil
}

// ocrBatch 单次请求识别一批图片
func (c *Zhipu4VClient) ocrBatch(ctx context.Context, images []OCRImage) (string, error) {
	content := []ocrContentPart{
		{
			Type: "text",
			Text: "请识别以下图片中的所有文字，并原样按阅读顺序输出。若图片是扫描的文档/表格，请完整转录正文内容，不要总结、不要省略。多张图片按给出的顺序依次输出，图片之间用空行分隔。",
		},
	}
	for _, img := range images {
		data, mime, err := prepareImage(img)
		if err != nil {
			continue
		}
		content = append(content, ocrContentPart{
			Type: "image_url",
			ImageURL: &struct {
				URL string `json:"url"`
			}{
				URL: "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data),
			},
		})
	}
	// 全部图片解码失败 → 报错由上层逐张降级
	if len(content) == 1 {
		return "", fmt.Errorf("所有图片解码失败")
	}

	body, err := json.Marshal(ocrVisionRequest{
		Model: c.model,
		Messages: []ocrVisionMessage{
			{Role: "user", Content: content},
		},
	})
	if err != nil {
		return "", fmt.Errorf("序列化 OCR 请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建 OCR 请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	if tid := middleware.GetTraceIDFromContext(ctx); tid != "" {
		httpReq.Header.Set("X-Trace-ID", tid)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("GLM-4V OCR 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 OCR 响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GLM-4V API 错误 (HTTP %d): %s", resp.StatusCode, truncate(string(respBody), 300))
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return "", fmt.Errorf("解析 OCR 响应失败: %w", err)
	}
	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("GLM-4V 返回空结果")
	}
	return openAIResp.Choices[0].Message.Content, nil
}

// prepareImage 对 OCR 图片做预处理：解码 → 超长边自动均值缩小 → 重新编码 JPEG。
// 扫描件多为 300dpi A4（约 2480×3508），直接发送易超视觉模型图片限制。
func prepareImage(img OCRImage) ([]byte, string, error) {
	src, _, err := image.Decode(bytes.NewReader(img.Data))
	if err != nil {
		return nil, "", err
	}
	scaled := downscaleByBox(src, maxOCRImageDim)
	if scaled == src {
		mime := img.MIME
		if mime == "" {
			mime = "image/jpeg"
		}
		return img.Data, mime, nil
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, scaled, &jpeg.Options{Quality: 90}); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "image/jpeg", nil
}

// downscaleByBox 按整数倍均值缩小，保持宽高比；不超过 maxDim 时原样返回。
func downscaleByBox(src image.Image, maxDim int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxDim && h <= maxDim {
		return src
	}
	m := w
	if h > m {
		m = h
	}
	factor := (m + maxDim - 1) / maxDim
	if factor < 2 {
		factor = 2
	}
	nw, nh := w/factor, h/factor
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		for x := 0; x < nw; x++ {
			var r, g, bb, a, count uint64
			for dy := 0; dy < factor; dy++ {
				for dx := 0; dx < factor; dx++ {
					sx, sy := x*factor+dx, y*factor+dy
					if sx >= w || sy >= h {
						continue
					}
					c := color.NRGBAModel.Convert(src.At(b.Min.X+sx, b.Min.Y+sy)).(color.NRGBA)
					r += uint64(c.R)
					g += uint64(c.G)
					bb += uint64(c.B)
					a += uint64(c.A)
					count++
				}
			}
			if count == 0 {
				continue
			}
			dst.Set(x, y, color.NRGBA{
				R: uint8(r / count), G: uint8(g / count),
				B: uint8(bb / count), A: uint8(a / count),
			})
		}
	}
	return dst
}
