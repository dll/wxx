---
name: wxx-context-engine
description: 蔚小芯 Context Engine 实现指南。当涉及知识检索、FTS/BM25 搜索、结构化查询、上下文拼装、AnswerCard 生成、知识资源增删改查或查询管道时触发。也在出现"知识库"、"检索"、"Context Engine"、"FTS"、"BM25"、"sources"、"召回"、"上下文"、"AnswerCard"等短语时触发，或在实现任何问答功能时触发。当修改 internal/context_engine/ 或 internal/service/ 中拼装 LLM 提示词的代码时使用本技能。
---

# 蔚小芯 Context Engine

本技能指导 Context Engine 的实现 — 这是让蔚小芯回答准确且可追溯的核心知识检索管道。没有它，大模型会凭空编造政策内容和流程步骤。

## 管道流程

每个用户提问都经过以下路径：

```
用户提问
    |
    v
[1. 意图分类]      --> 确定 resource_type + 触发策略
    |
    v
[2. 结构化查询]    --> SQLite 直查（按 resource_id、类型、范围精确匹配）
    |
    v
[3. FTS/BM25 检索] --> kb_fts MATCH 查询，按 BM25 相关性排序
    |
    v
[4. 上下文拼装]    --> 合并结构化 + FTS 结果，应用角色过滤，截断
    |
    v
[5. 模型生成]      --> 拼装后的上下文 + 提问 → 模型 API → AnswerCard
    |
    v
[6. 来源附加]      --> 从匹配的 kb_resources 附加 sources[]
```

**这是主路径。向量检索和 Agentic RAG 是可选附加项，不是默认项。**

## 各阶段详解

### 阶段 1：意图分类

将用户提问分类为四种资源类型之一：

| 类型 | 触发关键词 | 示例 |
|------|-----------|------|
| `Policy` | 政策、规定、条件、标准、要求 | "转专业的条件是什么？" |
| `Process` | 流程、步骤、怎么办、如何申请 | "奖学金申请流程？" |
| `FAQ` | 常见问题、一般来说、是不是 | "宿舍可以养宠物吗？" |
| `Activity` | 活动、比赛、报名、讲座 | "最近有什么志愿者活动？" |

如果分类置信度低，则跨所有类型查询并降低权重。

### 阶段 2：结构化查询

对于 `Process` 类型，`process_steps` 表可直接给出分步回答：

```sql
-- 查询流程步骤：按资源类型和发布状态筛选，按步骤序号排列
SELECT ps.step_order, ps.title, ps.materials, ps.entry_url, ps.deadline, ps.location
FROM process_steps ps
JOIN kb_resources kr ON ps.resource_id = kr.resource_id
WHERE kr.resource_type = 'Process'
  AND kr.status = 'published'
  AND kr.owner_scope = ?    -- RBAC 范围过滤
ORDER BY ps.step_order;
```

如果结构化查询已返回完整答案，跳过 FTS — 结构化数据具有权威性。

### 阶段 3：FTS/BM25 检索

```sql
-- 全文检索：按 BM25 相关性排序，筛选已发布且用户有权限的资源
SELECT kr.resource_id, kr.title, kr.summary, kr.content,
       bm25(kb_fts) as score
FROM kb_fts
JOIN kb_resources kr ON kb_fts.rowid = kr.id
WHERE kb_fts MATCH ?
  AND kr.status = 'published'
  AND kr.owner_scope IN (?)   -- RBAC 范围过滤
ORDER BY score
LIMIT ?;                       -- Top-K，默认 5
```

参数（来自 `docs/context-engine.md`）：
- **Top-K**：默认 5，可按资源类型调整
- **分词器**：`unicode61`（支持中日韩文字）
- **最低分数阈值**：可配置，过滤低相关性噪音

### 阶段 4：上下文拼装

合并阶段 2 和阶段 3 的结果：

1. 按 `resource_id` 去重
2. 结构化结果排名高于 FTS 结果
3. 应用 `role_scope` 过滤 — 只包含用户角色可见的资源
4. 检查 `effective_at` / `expired_at` — 排除过期资源
5. 截断总上下文以适应模型上下文窗口（为系统提示词 + 用户提问留出空间）
6. 格式化为结构化上下文块供 LLM 提示词使用

### 阶段 5：模型生成

将拼装好的上下文和用户提问发送到模型 API。系统提示词必须指示模型：
- 仅基于提供的上下文回答
- 如果上下文不足，如实告知（兜底回复）
- 绝不编造政策编号、日期或条件
- 按 AnswerCard 格式组织回答

### 阶段 6：来源附加

每个回答必须包含 `sources[]`，包含匹配资源的引用信息：

```json
{
  "sources": [
    {
      "resource_id": "POL-2026-001",
      "title": "转专业管理办法",
      "version": "2026.1",
      "source_link": "https://...",
      "relevance_score": 0.92
    }
  ]
}
```

规则：
- **政策/流程类回答**：`sources[]` 是强制的（100% 覆盖率要求）
- **常见问题类回答**：引用具体政策时需附带 `sources[]`（95%+ 覆盖率）
- **活动类回答**：建议附带 `sources[]` 但不强制
- **兜底回答**：空 `sources[]`，明确说明"未找到相关信息"

## AnswerCard 结构

所有问答响应使用统一结构（参见 `docs/ui-answer-card.md`）：

```json
{
  "conclusion": "简要结论",
  "steps": ["步骤列表..."],
  "sources": ["来源引用..."],
  "risks": ["注意事项..."],
  "follow_ups": ["你可能还想了解..."],
  "actions": [{"label": "在线申请", "url": "..."}],
  "trace_id": "uuid",
  "confidence": 0.85,
  "fallback": false
}
```

## 兜底策略

当 Context Engine 返回结果不足（无匹配或低置信度）时：
1. 在 AnswerCard 中设置 `fallback = true`
2. 提供礼貌的"未找到确切信息"回复
3. 通过 FTS 近似匹配推荐相关主题
4. 绝不编造回答 — 这是合规性要求
5. 记录未命中信息用于知识空白分析

目标：兜底率 ≤ 10%（质量指标）

## 何时添加向量检索

向量检索不是默认管道的一部分。以下情况可考虑添加：
- 同义词密集查询的 FTS/BM25 命中率持续低于 85%
- 用户频繁换不同措辞提问但其实指向同一资源
- 长尾查询持续错过相关内容

即便添加，向量检索也只是 FTS 的补充 — 不能取代结构化 + FTS 作为主路径。

## 代码位置

所有 Context Engine 逻辑位于 `server/internal/context_engine/`。应暴露简洁的接口：

```go
// ContextEngine 知识检索管道接口
type ContextEngine interface {
    // Query 执行完整的检索管道：意图分类 → 结构化查询 → FTS → 上下文拼装
    Query(ctx context.Context, query string, userRole string, scope string) (*ContextResult, error)
}
```

service 层调用 `ContextEngine.Query()` 获取拼装好的上下文，再传给 LLM 客户端。handler 绝不直接接触 Context Engine。
