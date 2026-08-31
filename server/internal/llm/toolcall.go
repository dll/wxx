package llm

import (
	"encoding/json"
	"fmt"
)

// ToolCall 的线上格式与 OpenAI 兼容：{"id","type":"function","function":{"name","arguments"}}。
// 消费端使用扁平字段（ID/Name/Arguments），通过自定义序列化双向映射。

type toolCallWire struct {
	ID       string `json:"id"`
	Type     string `json:"type,omitempty"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// MarshalJSON 序列化为 OpenAI 兼容线上格式。
func (t ToolCall) MarshalJSON() ([]byte, error) {
	w := toolCallWire{ID: t.ID, Type: "function"}
	w.Function.Name = t.Name
	w.Function.Arguments = t.Arguments
	return json.Marshal(w)
}

// UnmarshalJSON 解析 OpenAI 兼容线上格式（宽容缺 type）。
func (t *ToolCall) UnmarshalJSON(data []byte) error {
	var w toolCallWire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	if w.Function.Name == "" && w.Function.Arguments == "" {
		return fmt.Errorf("tool_call 缺少 function 字段")
	}
	t.ID = w.ID
	t.Name = w.Function.Name
	t.Arguments = w.Function.Arguments
	return nil
}
