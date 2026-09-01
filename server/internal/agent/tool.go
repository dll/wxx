package agent

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// ── A3: 校园场景工具注册表 ──
//
// 工具 = 不依赖 LLM 的确定性数据查询（流程步骤、部门联系方式、校历等）。
// 与检索型 Agent 互补：Agent 负责"找资料"，工具负责"查准确数据"。
// 后续接入 LLM function calling 时，注册表即工具清单来源。

// ToolArgs 工具入参
type ToolArgs struct {
	Question   string // 用户问题（原文）
	OwnerScope string // 权限范围（school/college/class）
	OwnerID    string
	Role       string
	Limit      int               // 返回条数上限（默认 5）
	Extra      map[string]string // 工具自定义参数
}

// ToolResult 工具出参
type ToolResult struct {
	Content string              `json:"content"`           // 面向 LLM 的结构化文本
	Sources []model.Source      `json:"sources,omitempty"` // 可引用来源（与 AnswerCard.Sources 同构）
	Data    []map[string]string `json:"data,omitempty"`    // 结构化字段（如步骤名/地点/时限）
}

// Tool 校园场景工具接口
type Tool interface {
	// Name 工具唯一名（snake_case，供 function calling 使用）
	Name() string
	// Description 功能描述（供 LLM 选择工具时参考）
	Description() string
	// Execute 执行确定性查询
	Execute(ctx context.Context, args ToolArgs) (*ToolResult, error)
}

// ToolRegistry 工具注册表（并发安全）
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewToolRegistry 创建工具注册表
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]Tool)}
}

// Register 注册工具（同名覆盖）
func (r *ToolRegistry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// Get 按名取工具
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List 按名称排序返回全部工具（输出稳定）
func (r *ToolRegistry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for n := range r.tools {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]Tool, 0, len(names))
	for _, n := range names {
		out = append(out, r.tools[n])
	}
	return out
}

// Run 执行工具并记录日志（统一入口：耗时、成败留痕）
func (r *ToolRegistry) Run(ctx context.Context, name string, args ToolArgs) (*ToolResult, error) {
	tool, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("工具不存在: %s", name)
	}
	result, err := tool.Execute(ctx, args)
	if err != nil {
		log.Printf("[WARN] 工具执行失败 [tool=%s]: %v", name, err)
	}
	return result, err
}

// ── 内置工具 ──

// ProcessNodeTool 办事流程步骤查询：结构化检索 Process 类型资源，零 LLM、强确定性。
type ProcessNodeTool struct {
	kbRepo *repository.KBRepo
	limit  int
}

// NewProcessNodeTool 创建流程查询工具
func NewProcessNodeTool(kbRepo *repository.KBRepo) *ProcessNodeTool {
	return &ProcessNodeTool{kbRepo: kbRepo, limit: 3}
}

func (t *ProcessNodeTool) Name() string { return "query_process_steps" }
func (t *ProcessNodeTool) Description() string {
	return "查询办事流程的步骤、材料、办理地点与时限（入学/请假/毕业/补考等）"
}

func (t *ProcessNodeTool) Execute(ctx context.Context, args ToolArgs) (*ToolResult, error) {
	limit := args.Limit
	if limit <= 0 {
		limit = t.limit
	}
	// 复用全局检索（含 FTS 语法防护与权限过滤），资源类型过滤在结果侧收敛，
	// 避免为单一工具复制一条带 typeFilter 的 SQL（保持 kbRepo 单一出口）。
	results, err := t.kbRepo.Search(args.Question, args.OwnerScope, args.OwnerID, args.Role, limit*3)
	if err != nil {
		return nil, err
	}

	tool := &ToolResult{}
	for _, r := range results {
		if r.Resource.ResourceType != "Process" {
			continue
		}
		tool.Data = append(tool.Data, map[string]string{
			"title":   r.Resource.Title,
			"content": r.Resource.Content,
			"link":    r.Resource.SourceLink,
			"version": r.Resource.Version,
		})
		if len(tool.Data) >= limit {
			break
		}
	}
	if len(tool.Data) == 0 {
		tool.Content = "未找到匹配的办事流程。"
		return tool, nil
	}

	for i, d := range tool.Data {
		tool.Content += fmt.Sprintf("【流程%d】%s（版本 %s）\n%s\n", i+1, d["title"], d["version"], d["content"])
	}
	for _, d := range tool.Data {
		tool.Sources = append(tool.Sources, model.Source{
			Title: d["title"], SourceLink: d["link"], Version: d["version"],
		})
	}
	return tool, nil
}
