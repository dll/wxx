package llm

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestDownscaleByBox 大图应被缩小，小图原样返回
func TestDownscaleByBox(t *testing.T) {
	big := image.NewRGBA(image.Rect(0, 0, 2480, 3437))
	for y := 0; y < 3437; y++ {
		for x := 0; x < 2480; x++ {
			big.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 100, 255})
		}
	}
	small := downscaleByBox(big, maxOCRImageDim)
	if small == big {
		t.Fatal("大图应被缩小")
	}
	b := small.Bounds()
	if b.Dx() > maxOCRImageDim && b.Dy() > maxOCRImageDim {
		t.Fatalf("缩小后仍超限: %dx%d", b.Dx(), b.Dy())
	}

	tiny := image.NewRGBA(image.Rect(0, 0, 200, 100))
	if downscaleByBox(tiny, maxOCRImageDim) != tiny {
		t.Fatal("小图不应被缩放")
	}
}

// TestOCRRequestShape 用 httptest 校验 OCR 请求体（data URL 图片 + 指令文本）与响应解析
func TestOCRRequestShape(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"识别出的文字"}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`))
	}))
	defer srv.Close()

	c := &Zhipu4VClient{apiKey: "test-key", baseURL: srv.URL, model: "glm-4v-flash", client: srv.Client()}

	// 构造一张 PNG 图片
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	var buf strings.Builder
	_ = png.Encode(&buf, img)
	imgBytes := []byte(buf.String())

	text, err := c.OCR(context.Background(), []OCRImage{{Data: imgBytes, MIME: "image/png"}})
	if err != nil {
		t.Fatalf("OCR 调用失败: %v", err)
	}
	if strings.TrimSpace(text) != "识别出的文字" {
		t.Fatalf("OCR 返回不符: %q", text)
	}

	var req struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content []struct {
				Type     string `json:"type"`
				Text     string `json:"text"`
				ImageURL *struct {
					URL string `json:"url"`
				} `json:"image_url"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal([]byte(gotBody), &req); err != nil {
		t.Fatalf("请求体非 JSON: %v", err)
	}
	if req.Model != "glm-4v-flash" {
		t.Fatalf("模型不符: %s", req.Model)
	}
	if len(req.Messages) == 0 || len(req.Messages[0].Content) < 2 {
		t.Fatal("消息 content 应含指令文本 + 图片")
	}
	if req.Messages[0].Content[1].ImageURL == nil || !strings.HasPrefix(req.Messages[0].Content[1].ImageURL.URL, "data:image/png;base64,") {
		t.Fatalf("图片应为 data URL: %+v", req.Messages[0].Content[1])
	}
}

// TestOCRNoImages 无图片应报错
func TestOCRNoImages(t *testing.T) {
	c := &Zhipu4VClient{apiKey: "k", baseURL: "http://x", model: "glm-4v"}
	if _, err := c.OCR(context.Background(), nil); err == nil {
		t.Fatal("空图片应报错")
	}
}
