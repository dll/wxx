---
name: wxx-context-engine
description: Guides Context Engine implementation for WeiXiaoXin (蔚小芯). Triggers when working on knowledge retrieval, FTS/BM25 search, structured queries, context assembly, AnswerCard generation, knowledge resource CRUD, or the query pipeline. Also triggers on phrases like "知识库", "检索", "Context Engine", "FTS", "BM25", "sources", "召回", "上下文", "AnswerCard", or when implementing any Q&A feature. Use this skill whenever touch internal/context_engine/ or internal/service/ code that assembles LLM prompts.
---

# 蔚小芯 Context Engine

This skill guides implementation of the Context Engine — the core knowledge retrieval pipeline that makes 蔚小芯's answers accurate and traceable. Without it, the LLM would hallucinate policies and fabricate procedure steps.

## The Pipeline

Every user question follows this path:

```
User Query
    |
    v
[1. Intent Classification]  --> determines resource_type + trigger strategy
    |
    v
[2. Structured Query]       --> SQLite direct lookup (exact match on resource_id, type, scope)
    |
    v
[3. FTS/BM25 Search]        --> kb_fts MATCH query, ranked by BM25 relevance
    |
    v
[4. Context Assembly]       --> merge structured + FTS results, apply role filter, truncate
    |
    v
[5. LLM Generation]         --> assembled context + query -> model API -> AnswerCard
    |
    v
[6. Source Attachment]       --> attach sources[] from matched kb_resources
```

**This is the main path. Vector search and Agentic RAG are optional add-ons, not defaults.**

## Stage Details

### Stage 1: Intent Classification

Classify user query into one of four resource types:

| Type | Trigger Keywords | Example |
|------|---------|---------|
| `Policy` | 政策, 规定, 条件, 标准, 要求 | "转专业的条件是什么？" |
| `Process` | 流程, 步骤, 怎么办, 如何申请 | "奖学金申请流程？" |
| `FAQ` | 常见问题, 一般来说, 是不是 | "宿舍可以养宠物吗？" |
| `Activity` | 活动, 比赛, 报名, 讲座 | "最近有什么志愿者活动？" |

If classification confidence is low, query across all types with reduced weight.

### Stage 2: Structured Query

For `Process` type, `process_steps` table gives step-by-step answers directly:

```sql
SELECT ps.step_order, ps.title, ps.materials, ps.entry_url, ps.deadline, ps.location
FROM process_steps ps
JOIN kb_resources kr ON ps.resource_id = kr.resource_id
WHERE kr.resource_type = 'Process'
  AND kr.status = 'published'
  AND kr.owner_scope = ?    -- RBAC scope filter
ORDER BY ps.step_order;
```

If structured query returns a complete answer, skip FTS — structured data is authoritative.

### Stage 3: FTS/BM25 Search

```sql
SELECT kr.resource_id, kr.title, kr.summary, kr.content,
       bm25(kb_fts) as score
FROM kb_fts
JOIN kb_resources kr ON kb_fts.rowid = kr.id
WHERE kb_fts MATCH ?
  AND kr.status = 'published'
  AND kr.owner_scope IN (?)   -- RBAC scope filter
ORDER BY score
LIMIT ?;                       -- Top-K, default 5
```

Parameters (from `docs/context-engine.md`):
- **Top-K**: 5 (default), adjustable per resource type
- **Tokenizer**: `unicode61` (CJK-aware)
- **Minimum score threshold**: configurable, drop low-relevance noise

### Stage 4: Context Assembly

Merge results from Stages 2 and 3:

1. Deduplicate by `resource_id`
2. Structured results rank higher than FTS results
3. Apply `role_scope` filter — only include resources the user's role can see
4. Check `effective_at` / `expired_at` — exclude expired resources
5. Truncate total context to fit model's context window (leave room for system prompt + user query)
6. Format as structured context block for LLM prompt

### Stage 5: LLM Generation

Send assembled context + user query to model API. The system prompt must instruct the model:
- Answer ONLY based on provided context
- If context is insufficient, say so (fallback response)
- Never fabricate policy numbers, dates, or conditions
- Structure response as AnswerCard format

### Stage 6: Source Attachment

Every answer must include `sources[]` with:

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

Rules:
- **Policy/Process answers**: `sources[]` is MANDATORY (100% coverage required)
- **FAQ answers**: `sources[]` required if referencing specific policies (95%+ coverage)
- **Activity answers**: `sources[]` recommended but not mandatory
- **Fallback answers**: empty `sources[]`, clearly state "未找到相关信息"

## AnswerCard Structure

All Q&A responses use this unified structure (see `docs/ui-answer-card.md`):

```json
{
  "conclusion": "简要结论",
  "steps": [...],
  "sources": [...],
  "risks": ["注意事项..."],
  "follow_ups": ["你可能还想了解..."],
  "actions": [{"label": "在线申请", "url": "..."}],
  "trace_id": "uuid",
  "confidence": 0.85,
  "fallback": false
}
```

## Fallback Strategy

When Context Engine returns insufficient results (no match or low confidence):
1. Set `fallback = true` in AnswerCard
2. Provide a polite "未找到确切信息" message
3. Suggest related topics from FTS near-matches
4. NEVER fabricate an answer — this is a compliance requirement
5. Log the miss for knowledge gap analysis

Target: fallback rate <= 10% (quality KPI)

## When to Add Vector Search

Vector search is NOT part of the default pipeline. Consider adding it when:
- FTS/BM25 hit rate drops below 85% on synonym-heavy queries
- Users frequently rephrase questions that should match the same resource
- Long-tail queries consistently miss relevant content

Even then, vector search supplements FTS — it does not replace structured + FTS as the primary path.

## Code Location

All Context Engine logic lives in `server/internal/context_engine/`. It should expose a clean interface:

```go
type ContextEngine interface {
    Query(ctx context.Context, query string, userRole string, scope string) (*ContextResult, error)
}
```

Service layer calls `ContextEngine.Query()`, receives assembled context, then passes it to LLM client. The handler never touches Context Engine directly.
