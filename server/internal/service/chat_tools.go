// Package service LLM function calling 循环（A3 接线）。
//
// 设计：
//   - 工具清单来自 agent.ToolRegistry（校园场景确定性工具），转 OpenAI function 格式；
//   - 循环上限 maxToolRounds=2：模型发起 tool_calls → 本地执行 → 结果以 role=tool 回填 → 再次调用；
//   - 优雅降级：带 tools 请求报错时，去掉 tools 重试一次（部分 provider 不支持）；
//   - 工具执行失败以 role=tool 内容 "tool error: ..." 回填，由模型自行决定兜底话术。
package service

import (
	"context"
	"encoding/json"

	"github.com/dll/wxx/server/internal/agent"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
)

// maxToolRounds 单次问答允许的工具调用轮数上限（防循环）。
const maxToolRounds = 2

// SetToolRegistry 注入工具注册表（A3；来自编排器的 Tools()）。
func (s *ChatService) SetToolRegistry(reg *agent.ToolRegistry) {
	s.toolRegistry = reg
}

// buildToolDefinitions 注册表 → OpenAI function 清单。
// 工具入参统一约定为 {question: string}（与工具执行语义一致）。
func buildToolDefinitions(reg *agent.ToolRegistry) []llm.ToolFunction {
	if reg == nil {
		return nil
	}
	tools := reg.List()
	if len(tools) == 0 {
		return nil
	}
	defs := make([]llm.ToolFunction, 0, len(tools))
	for _, t := range tools {
		defs = append(defs, llm.ToolFunction{
			Type: "function",
			Function: llm.ToolDefinition{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"question": map[string]any{
							"type":        "string",
							"description": "需要查询的问题原文",
						},
					},
					"required": []string{"question"},
				},
			},
		})
	}
	return defs
}

// executeToolCall 执行单次工具调用，返回回填给模型的内容。
func (s *ChatService) executeToolCall(ctx context.Context, tc llm.ToolCall) string {
	if s.toolRegistry == nil {
		return "tool error: registry unavailable"
	}
	var args agent.ToolArgs
	if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
		return "tool error: invalid arguments: " + err.Error()
	}
	result, err := s.toolRegistry.Run(ctx, tc.Name, args)
	if err != nil {
		return "tool error: " + err.Error()
	}
	if result == nil {
		return "tool error: empty result"
	}
	return result.Content
}

// chatWithToolLoop 带工具循环的同步对话：模型可发起 tool_calls，本地执行后回填再生成。
// 返回最终内容与用量（工具轮次的 token 计入同一响应）。
// 未配置工具注册表、或模型不支持工具（报错降级）时，退化为普通 Chat。
func (s *ChatService) chatWithToolLoop(ctx context.Context, userCtx *model.UserContext, sessionID, traceID string, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	toolDefs := buildToolDefinitions(s.toolRegistry)

	// 无工具：直接走网关
	if len(toolDefs) == 0 {
		return s.chatViaGateway(ctx, userCtx, sessionID, traceID, req)
	}

	// 带 tools 首轮
	req.Tools = toolDefs
	resp, err := s.chatViaGateway(ctx, userCtx, sessionID, traceID, req)
	if err != nil {
		// 优雅降级：provider 不支持 tools → 去 tools 重试一次
		req.Tools = nil
		return s.chatViaGateway(ctx, userCtx, sessionID, traceID, req)
	}

	for round := 0; round < maxToolRounds && len(resp.ToolCalls) > 0; round++ {
		// 回放 assistant 发起的调用
		req.Messages = append(req.Messages, llm.ChatMessage{
			Role:      "assistant",
			ToolCalls: resp.ToolCalls,
		})
		// 本地执行并回填结果
		for _, tc := range resp.ToolCalls {
			req.Messages = append(req.Messages, llm.ChatMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    s.executeToolCall(ctx, tc),
			})
		}
		// 续轮不再携带 tools 清单（模型已知），也可以保留；此处保留以支持多轮追加调用
		resp, err = s.chatViaGateway(ctx, userCtx, sessionID, traceID, req)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}
