---
name: wxx-code-review
description: Code review tailored to WeiXiaoXin (蔚小芯) project constraints. Triggers when the user asks to review code, check a PR, audit changes, or validate implementation quality. Also triggers on phrases like "审查", "review", "检查代码", "看看这段代码", "code review", or after completing a significant implementation task. Use proactively after multi-file changes to catch issues before commit.
---

# 蔚小芯 Code Review

This skill performs code review against the specific constraints and patterns of the 蔚小芯 project. Generic code review catches syntax issues; this catches architectural violations, missing security controls, and knowledge pipeline errors that are unique to this codebase.

## Review Checklist

### Architecture (Blocking)

- [ ] **Layer violations**: handler must not import repository/llm/context_engine/agent packages
- [ ] **Dependency direction**: calls flow handler -> service -> repository, never backwards
- [ ] **New dependencies**: any `go get` of a package not in approved list needs justification
- [ ] **Forbidden patterns**: no local LLM, no Coze SDK, no Docker requirements, no container orchestration

### Security (Blocking)

- [ ] **SQL parameterization**: all queries use `?` placeholders, especially FTS MATCH queries
- [ ] **Secrets in code**: no hardcoded API keys, JWT secrets, or tokens
- [ ] **RBAC declared**: every new endpoint has role requirement in middleware chain
- [ ] **Sensitive data masked**: student ID, phone, national ID never in plaintext responses
- [ ] **Audit logging**: sensitive operations write to `audit_logs`
- [ ] **JWT validation**: tokens checked for expiration and valid signature

### Knowledge Pipeline (Blocking for Q&A paths)

- [ ] **Sources attached**: policy/process answers include `sources[]` from matched resources
- [ ] **No fabrication**: LLM system prompt instructs "answer only from context"
- [ ] **Scope filtering**: queries filter by `owner_scope`, `role_scope`, `status=published`
- [ ] **Fallback handling**: insufficient context returns fallback response, not hallucinated answer
- [ ] **Pipeline order**: structured query -> FTS -> assembly -> LLM (not skipping stages)

### Code Quality (Non-blocking, recommend fix)

- [ ] **Error wrapping**: errors use `fmt.Errorf("context: %w", err)` for stack tracing
- [ ] **TraceID propagation**: request-scoped trace_id flows from middleware through service to audit
- [ ] **Context usage**: `context.Context` passed through all layers for cancellation
- [ ] **Resource cleanup**: database rows, HTTP responses properly closed/deferred
- [ ] **Timeout on external calls**: LLM and external API clients have explicit timeouts

### Documentation (Non-blocking, recommend fix)

- [ ] **API contract logged**: new/changed endpoints reflected in `specs/api-contracts-index.md`
- [ ] **RBAC matrix updated**: new role permissions documented in `specs/rbac-matrix.md`
- [ ] **Migration exists**: schema changes have corresponding file in `server/migrations/`

## How to Review

1. **Read the diff** — understand what changed and why
2. **Check architecture** — verify layer boundaries are respected
3. **Check security** — run through the security checklist items
4. **Check knowledge pipeline** — if Q&A path is affected, verify sources and fallback
5. **Check quality** — error handling, context propagation, resource cleanup
6. **Check docs** — are affected docs updated?

## Severity Levels

| Level | Meaning | Action |
|-------|---------|--------|
| **BLOCK** | Architecture violation, security hole, missing sources on policy answer | Must fix before commit |
| **WARN** | Missing docs update, suboptimal error handling, missing timeout | Should fix, can defer with justification |
| **NOTE** | Style suggestion, potential optimization | Nice to have |

## Output Format

Present review findings as:

```
## Review: [file or feature name]

**BLOCK** [file:line] — [issue description]
  Fix: [concrete suggestion]

**WARN** [file:line] — [issue description]
  Fix: [concrete suggestion]

**NOTE** [file:line] — [issue description]

### Summary
- Blocking issues: N
- Warnings: N
- Notes: N
- Verdict: PASS / NEEDS FIX
```
