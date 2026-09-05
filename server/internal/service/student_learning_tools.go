package service

import (
	"context"
	"fmt"

	"github.com/dll/wxx/server/internal/llm"
)

type KnowledgeGraph struct {
	CourseName string                   `json:"course_name"`
	Nodes      []map[string]interface{} `json:"nodes"`
	Edges      []map[string]interface{} `json:"edges"`
	DataSource string                   `json:"data_source"`
}

func (s *StudentService) GenerateKnowledgeGraph(ctx context.Context, courseName string) *KnowledgeGraph {
	_ = ctx
	if courseName == "" {
		courseName = "数据结构"
	}
	return &KnowledgeGraph{CourseName: courseName, Nodes: []map[string]interface{}{{"id": "ds", "name": "数据结构", "category": "root", "mastery": 0.75}, {"id": "linear", "name": "线性结构", "category": "branch", "mastery": 0.88}, {"id": "tree", "name": "树形结构", "category": "branch", "mastery": 0.65}, {"id": "graph", "name": "图结构", "category": "branch", "mastery": 0.45}, {"id": "sort", "name": "排序算法", "category": "branch", "mastery": 0.72}, {"id": "search", "name": "查找算法", "category": "branch", "mastery": 0.58}}, Edges: []map[string]interface{}{{"from": "ds", "to": "linear", "relation": "包含"}, {"from": "ds", "to": "tree", "relation": "包含"}, {"from": "ds", "to": "graph", "relation": "包含"}, {"from": "ds", "to": "sort", "relation": "包含"}, {"from": "ds", "to": "search", "relation": "包含"}, {"from": "tree", "to": "graph", "relation": "关联(遍历算法通用)"}}, DataSource: "reference"}
}

type NoteAssistant struct {
	Title         string                   `json:"title"`
	KeyPoints     []string                 `json:"key_points"`
	MindMap       string                   `json:"mind_map"`
	KeyConcepts   []string                 `json:"key_concepts"`
	QuizQuestions []map[string]interface{} `json:"quiz_questions"`
	DataSource    string                   `json:"data_source"`
}

func (s *StudentService) GenerateNoteAssistant(ctx context.Context, content string) *NoteAssistant {
	result := &NoteAssistant{Title: "学习笔记", KeyPoints: []string{"核心概念理解", "算法步骤梳理", "典型应用场景"}, MindMap: "中心主题 → 子概念1 → 子概念2", KeyConcepts: []string{"定义", "性质", "算法", "应用"}, QuizQuestions: []map[string]interface{}{{"q": "请简述核心概念的定义", "a": "核心概念是..."}}, DataSource: "fallback"}
	if s.llmClient != nil && content != "" {
		text := content
		if len([]rune(text)) > 500 {
			text = string([]rune(text)[:500])
		}
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "user", Content: fmt.Sprintf("你是学习笔记助手。请从以下内容提取知识点、生成思维导图和自测题。80字：\n%s", text)}}, Temperature: 0.4, MaxTokens: 400})
		if err == nil && resp != nil && resp.Content != "" {
			result.KeyPoints = append(result.KeyPoints, "AI提取："+resp.Content)
			result.DataSource = "ai"
		}
	}
	return result
}
