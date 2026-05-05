package llm

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// ── buildAuthURL 测试 ──

func TestBuildAuthURL_Valid(t *testing.T) {
	url, err := buildAuthURL("iat-api.xfyun.cn", "/v2/iat", "test-key", "test-secret")
	if err != nil {
		t.Fatalf("buildAuthURL 失败: %v", err)
	}

	// 验证必要的组成部分
	if !strings.HasPrefix(url, "wss://iat-api.xfyun.cn/v2/iat?") {
		t.Errorf("URL 应以 wss:// 开头，得到: %s", url)
	}
	if !strings.Contains(url, "authorization=") {
		t.Error("URL 应包含 authorization 参数")
	}
	if !strings.Contains(url, "date=") {
		t.Error("URL 应包含 date 参数")
	}
	if !strings.Contains(url, "host=") {
		t.Error("URL 应包含 host 参数")
	}
}

func TestBuildAuthURL_DifferentHosts(t *testing.T) {
	url, err := buildAuthURL("tts-api.xfyun.cn", "/v2/tts", "key", "secret")
	if err != nil {
		t.Fatalf("buildAuthURL 失败: %v", err)
	}
	if !strings.HasPrefix(url, "wss://tts-api.xfyun.cn/v2/tts?") {
		t.Errorf("URL 应包含 TTS host，得到: %s", url)
	}
}

// ── buildASRFrame 测试 ──

func TestBuildASRFrame_FirstFrame(t *testing.T) {
	client := &XfyunClient{appID: "test-app-id"}
	chunk := make([]byte, 100)
	for i := range chunk {
		chunk[i] = byte(i % 256)
	}

	frame, err := client.buildASRFrame(chunk, 0)
	if err != nil {
		t.Fatalf("buildASRFrame 失败: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(frame, &parsed); err != nil {
		t.Fatalf("解析帧 JSON 失败: %v", err)
	}

	// 验证 common
	common := parsed["common"].(map[string]interface{})
	if common["app_id"] != "test-app-id" {
		t.Errorf("期望 app_id=test-app-id，得到 %v", common["app_id"])
	}

	// 验证 business
	business := parsed["business"].(map[string]interface{})
	if business["language"] != "zh_cn" {
		t.Errorf("期望 language=zh_cn，得到 %v", business["language"])
	}
	if business["accent"] != "mandarin" {
		t.Errorf("期望 accent=mandarin，得到 %v", business["accent"])
	}

	// 验证 data
	data := parsed["data"].(map[string]interface{})
	if data["status"].(float64) != 0 {
		t.Errorf("首帧 status 应为 0，得到 %v", data["status"])
	}
	if data["format"] != "audio/L16;rate=16000" {
		t.Errorf("期望 format 为音频格式，得到 %v", data["format"])
	}
	if data["encoding"] != "raw" {
		t.Errorf("期望 encoding=raw，得到 %v", data["encoding"])
	}

	// 验证 audio 是有效的 base64
	audioB64 := data["audio"].(string)
	if audioB64 == "" {
		t.Error("audio 字段不应为空")
	}
	decoded, err := base64.StdEncoding.DecodeString(audioB64)
	if err != nil {
		t.Fatalf("audio 不是有效 base64: %v", err)
	}
	if len(decoded) != 100 {
		t.Errorf("解码后长度应为 100，得到 %d", len(decoded))
	}
}

func TestBuildASRFrame_MiddleFrame(t *testing.T) {
	client := &XfyunClient{appID: "app"}
	frame, err := client.buildASRFrame([]byte("hello"), 1)
	if err != nil {
		t.Fatalf("buildASRFrame 失败: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(frame, &parsed)
	data := parsed["data"].(map[string]interface{})
	if data["status"].(float64) != 1 {
		t.Errorf("中间帧 status 应为 1，得到 %v", data["status"])
	}
}

func TestBuildASRFrame_LastFrame(t *testing.T) {
	client := &XfyunClient{appID: "app"}
	frame, err := client.buildASRFrame([]byte("end"), 2)
	if err != nil {
		t.Fatalf("buildASRFrame 失败: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(frame, &parsed)
	data := parsed["data"].(map[string]interface{})
	if data["status"].(float64) != 2 {
		t.Errorf("末帧 status 应为 2，得到 %v", data["status"])
	}
}

// ── parseASRResult 测试 ──

func TestParseASRResult_Success(t *testing.T) {
	msg := []byte(`{
		"code": 0,
		"message": "success",
		"data": {
			"status": 2,
			"result": {
				"ws": [
					{
						"cw": [
							{"w": "今天", "wp": "n"},
							{"w": "天气", "wp": "n"},
							{"w": "不错", "wp": "a"}
						]
					}
				]
			}
		}
	}`)

	text, done, err := parseASRResult(msg)
	if err != nil {
		t.Fatalf("parseASRResult 失败: %v", err)
	}
	if text != "今天天气不错" {
		t.Errorf("期望 今天天气不错，得到 %s", text)
	}
	if !done {
		t.Error("status=2 时应标记为完成")
	}
}

func TestParseASRResult_NotFinished(t *testing.T) {
	msg := []byte(`{
		"code": 0,
		"message": "success",
		"data": {
			"status": 1,
			"result": {
				"ws": [
					{"cw": [{"w": "今", "wp": ""}]}
				]
			}
		}
	}`)

	text, done, err := parseASRResult(msg)
	if err != nil {
		t.Fatalf("parseASRResult 失败: %v", err)
	}
	if text != "今" {
		t.Errorf("期望 今，得到 %s", text)
	}
	if done {
		t.Error("status=1 时应标记为未完成")
	}
}

func TestParseASRResult_EmptyResult(t *testing.T) {
	msg := []byte(`{
		"code": 0,
		"message": "success",
		"data": {
			"status": 1,
			"result": {
				"ws": []
			}
		}
	}`)

	text, _, err := parseASRResult(msg)
	if err != nil {
		t.Fatalf("parseASRResult 失败: %v", err)
	}
	if text != "" {
		t.Errorf("空结果应返回空文本，得到 %s", text)
	}
}

func TestParseASRResult_ErrorCode(t *testing.T) {
	msg := []byte(`{
		"code": 10100,
		"message": "invalid parameter",
		"data": {
			"status": 0,
			"result": {}
		}
	}`)

	_, done, err := parseASRResult(msg)
	if err == nil {
		t.Fatal("非零 code 应返回错误")
	}
	if !done {
		t.Error("错误码时应标记为完成")
	}
	if !strings.Contains(err.Error(), "10100") {
		t.Errorf("错误应包含 code 10100: %v", err)
	}
}

func TestParseASRResult_InvalidJSON(t *testing.T) {
	_, _, err := parseASRResult([]byte("not json"))
	if err == nil {
		t.Fatal("无效 JSON 应返回错误")
	}
}

// ── buildTTSFrame 测试 ──

func TestBuildTTSFrame_DefaultVoice(t *testing.T) {
	client := &XfyunClient{appID: "tts-app"}
	frame, err := client.buildTTSFrame("你好世界", "x_xiaoyan")
	if err != nil {
		t.Fatalf("buildTTSFrame 失败: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(frame, &parsed); err != nil {
		t.Fatalf("解析帧 JSON 失败: %v", err)
	}

	common := parsed["common"].(map[string]interface{})
	if common["app_id"] != "tts-app" {
		t.Errorf("期望 app_id=tts-app，得到 %v", common["app_id"])
	}

	business := parsed["business"].(map[string]interface{})
	if business["aue"] != "lame" {
		t.Errorf("期望 aue=lame，得到 %v", business["aue"])
	}
	if business["vcn"] != "x_xiaoyan" {
		t.Errorf("期望 vcn=x_xiaoyan，得到 %v", business["vcn"])
	}

	data := parsed["data"].(map[string]interface{})
	if data["status"].(float64) != 2 {
		t.Errorf("TTS 帧 status 应为 2，得到 %v", data["status"])
	}

	// 验证 text 是有效 base64
	textB64 := data["text"].(string)
	decoded, err := base64.StdEncoding.DecodeString(textB64)
	if err != nil {
		t.Fatalf("text 不是有效 base64: %v", err)
	}
	if string(decoded) != "你好世界" {
		t.Errorf("文本应为 你好世界，得到 %s", string(decoded))
	}
}

func TestBuildTTSFrame_CustomVoice(t *testing.T) {
	client := &XfyunClient{appID: "app"}
	frame, err := client.buildTTSFrame("hello", "x_xiaogang")
	if err != nil {
		t.Fatalf("buildTTSFrame 失败: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(frame, &parsed)
	business := parsed["business"].(map[string]interface{})
	if business["vcn"] != "x_xiaogang" {
		t.Errorf("期望 vcn=x_xiaogang，得到 %v", business["vcn"])
	}
}

func TestBuildTTSFrame_TTSParams(t *testing.T) {
	client := &XfyunClient{appID: "app"}
	frame, err := client.buildTTSFrame("测试", "x_xiaoyan")
	if err != nil {
		t.Fatalf("buildTTSFrame 失败: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(frame, &parsed)
	business := parsed["business"].(map[string]interface{})
	if business["pitch"].(float64) != 50 {
		t.Errorf("期望 pitch=50，得到 %v", business["pitch"])
	}
	if business["speed"].(float64) != 50 {
		t.Errorf("期望 speed=50，得到 %v", business["speed"])
	}
	if business["volume"].(float64) != 50 {
		t.Errorf("期望 volume=50，得到 %v", business["volume"])
	}
	if business["tte"] != "utf8" {
		t.Errorf("期望 tte=utf8，得到 %v", business["tte"])
	}
}

// ── parseTTSResult 测试 ──

func TestParseTTSResult_Success(t *testing.T) {
	audioData := []byte{0x01, 0x02, 0x03, 0x04}
	audioB64 := base64.StdEncoding.EncodeToString(audioData)

	msg := []byte(`{
		"code": 0,
		"message": "success",
		"data": {
			"audio": "` + audioB64 + `",
			"status": 2,
			"ced": ""
		}
	}`)

	audio, done, err := parseTTSResult(msg)
	if err != nil {
		t.Fatalf("parseTTSResult 失败: %v", err)
	}
	if len(audio) != 4 {
		t.Errorf("期望 4 字节音频，得到 %d", len(audio))
	}
	if audio[0] != 0x01 || audio[3] != 0x04 {
		t.Error("音频数据不匹配")
	}
	if !done {
		t.Error("status=2 时应标记为完成")
	}
}

func TestParseTTSResult_NotFinished(t *testing.T) {
	audioB64 := base64.StdEncoding.EncodeToString([]byte{0xFF})

	msg := []byte(`{
		"code": 0,
		"message": "success",
		"data": {
			"audio": "` + audioB64 + `",
			"status": 1,
			"ced": ""
		}
	}`)

	_, done, err := parseTTSResult(msg)
	if err != nil {
		t.Fatalf("parseTTSResult 失败: %v", err)
	}
	if done {
		t.Error("status=1 时应标记为未完成")
	}
}

func TestParseTTSResult_ErrorCode(t *testing.T) {
	msg := []byte(`{
		"code": 10101,
		"message": "text too long",
		"data": {
			"audio": "",
			"status": 0,
			"ced": ""
		}
	}`)

	_, done, err := parseTTSResult(msg)
	if err == nil {
		t.Fatal("非零 code 应返回错误")
	}
	if !done {
		t.Error("错误码时应标记为完成")
	}
	if !strings.Contains(err.Error(), "10101") {
		t.Errorf("错误应包含 code 10101: %v", err)
	}
}

func TestParseTTSResult_InvalidBase64(t *testing.T) {
	msg := []byte(`{
		"code": 0,
		"message": "success",
		"data": {
			"audio": "!!!not-valid-base64!!!",
			"status": 2,
			"ced": ""
		}
	}`)

	_, _, err := parseTTSResult(msg)
	if err == nil {
		t.Fatal("无效 base64 应返回错误")
	}
}

func TestParseTTSResult_InvalidJSON(t *testing.T) {
	_, _, err := parseTTSResult([]byte("not json"))
	if err == nil {
		t.Fatal("无效 JSON 应返回错误")
	}
}

// ── NewXfyunClient 测试 ──

func TestNewXfyunClient(t *testing.T) {
	// 使用模拟配置
	client := &XfyunClient{
		appID:     "test-app",
		apiKey:    "test-key",
		apiSecret: "test-secret",
	}

	if client.appID != "test-app" {
		t.Errorf("期望 appID=test-app，得到 %s", client.appID)
	}
	if client.Name() != "xfyun" {
		t.Errorf("期望 Name=xfyun，得到 %s", client.Name())
	}
}

// ── buildAuthURL 签名确定性测试 ──

func TestBuildAuthURL_SignatureIsDeterministic(t *testing.T) {
	// 用相同参数调用两次，验证 date 字段不同（时间变化）但 host 相同
	url1, _ := buildAuthURL("iat-api.xfyun.cn", "/v2/iat", "key", "secret")
	time.Sleep(10 * time.Millisecond)
	url2, _ := buildAuthURL("iat-api.xfyun.cn", "/v2/iat", "key", "secret")

	// host 参数应相同
	if !strings.Contains(url1, "iat-api.xfyun.cn") {
		t.Error("url1 应包含 host")
	}
	if !strings.Contains(url2, "iat-api.xfyun.cn") {
		t.Error("url2 应包含 host")
	}

	// 应该有 date 参数（可能不同）
	if !strings.Contains(url1, "date=") || !strings.Contains(url2, "date=") {
		t.Error("两个 URL 都应包含 date 参数")
	}
}

// ── truncate 测试 ──

func TestTruncate_ShortString(t *testing.T) {
	result := truncate("hello", 10)
	if result != "hello" {
		t.Errorf("期望 hello，得到 %s", result)
	}
}

func TestTruncate_LongString(t *testing.T) {
	result := truncate("this is a very long string that exceeds the limit", 20)
	if len(result) != 23 { // 20 + "..."
		t.Errorf("期望 23 字符，得到 %d: %s", len(result), result)
	}
	if !strings.HasSuffix(result, "...") {
		t.Error("截断应添加 ...")
	}
}

func TestTruncate_ExactLength(t *testing.T) {
	result := truncate("hello", 5)
	if result != "hello" {
		t.Errorf("期望 hello，得到 %s", result)
	}
}
