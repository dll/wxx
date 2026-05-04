---
name: wxx-harness
description: Enforces the Harness Engineering workflow for the WeiXiaoXin (蔚小芯) project. Triggers whenever the user asks to implement a feature, fix a bug, add an endpoint, modify business logic, or make any non-trivial code change in the WXX project. Also triggers on phrases like "Plan", "方案", "开发", "实现", "添加功能", "修复", or when someone is about to write code without a plan. Use this skill proactively - if you see code being written without following Plan-Review-Code, intervene.
---

# 蔚小芯 Harness Workflow

This skill enforces **Harness Engineering** discipline for the 蔚小芯 project: every non-trivial change follows **Plan -> Human Review -> Code -> Verify -> Commit** with documentation and architectural guardrails baked in.

The goal is preventing two common failure modes: (1) jumping straight into code on a complex task and realizing mid-way the approach is wrong, and (2) finishing code that works but breaks layering rules, skips docs, or introduces forbidden dependencies.

## When to Activate

Activate this skill for ANY code change that:
- Touches more than one file
- Adds or modifies a handler, service, repository, or agent
- Changes database schema or migrations
- Modifies middleware (auth, RBAC, audit)
- Integrates with external APIs (智谱, DeepSeek, 讯飞, 学工系统, 一表通)
- Affects the Context Engine pipeline

For single-line fixes (typos, log messages, obvious bugs), skip directly to the Commit step.

## Workflow

### Step 1: Plan

Before writing any code, create a plan using the template at `templates/plan.template.md`:

```markdown
# 任务方案 — 【title】
## 背景与目标
## 范围（做 / 不做）
## 技术要点（栈、接口、风险）
## 步骤拆分
## 验收标准
## 回滚与检查点（Git / 数据）
```

The plan should address:
- Which layers are affected (handler / service / repository / agent / context_engine / llm)
- Which docs need updating (`docs/`, `specs/`, CLAUDE.md)
- Whether new database migrations are required
- RBAC implications (which of the 6+2 roles are affected)
- Whether `sources[]` tracing is required (yes for any policy/process answer path)

Present the plan to the user and wait for approval before proceeding.

### Step 2: Architectural Guard Check

Before implementation, verify these constraints. If any would be violated, stop and discuss with the user:

**Layering rules** (see `server/README.md`):
- handler -> service -> repository (one direction only)
- handler NEVER calls repository or llm directly
- repository NEVER depends on HTTP or model APIs

**Forbidden dependencies** - these must NEVER be introduced without written project change approval:
- Local LLM deployment (all models are API-only: 智谱/DeepSeek/讯飞)
- Coze or any third-party agent SaaS
- Docker/container/cluster requirements (lightweight single-binary deployment)
- Any dependency that forces containerization

**Knowledge pipeline** - the main path is always:
1. Structured query (SQLite direct) -> 2. FTS/BM25 search -> 3. Context assembly -> 4. Model generation
- Vector search and Agentic RAG are opt-in, never default
- Policy/process answers MUST include `sources[]` — fabricating citations or key numbers is prohibited

**RBAC** - every new endpoint must declare which roles can access it. The six baseline roles:
`sys_admin > school_admin > college_admin > counselor > student_union > student`
Plus two extension roles: `teacher`, `assistant`

### Step 3: Implement

Write code following the plan. For each sub-task:

1. Implement the change
2. Run lint: `make lint` (go vet)
3. Run tests: `make test`
4. If tests fail, fix before moving on — do not accumulate broken tests

Key implementation conventions:
- Go backend follows `server/internal/` package structure
- All HTTP responses use the unified `AnswerCard` structure for Q&A endpoints (see `docs/ui-answer-card.md`)
- All knowledge queries go through Context Engine (`internal/context_engine/`), not direct DB calls from handlers
- Audit logging for sensitive operations (`audit_logs` table)
- Error responses include `trace_id` for debugging

### Step 4: Verify

Before committing, run the full check:

```bash
make lint          # go vet
make test          # unit tests
```

Also verify manually:
- New/modified endpoints are listed in `specs/api-contracts-index.md`
- RBAC permissions for new endpoints are documented in `specs/rbac-matrix.md`
- If schema changed, a new migration file exists in `server/migrations/`
- If knowledge types changed, `specs/resource-schema.md` is updated

### Step 5: Commit & Document

Every commit that completes a meaningful increment must:

1. Update relevant docs in `docs/` or `specs/` if behavior changed
2. Write a clear commit message following conventional commits:
   - `feat:` new feature
   - `fix:` bug fix
   - `docs:` documentation only
   - `refactor:` no behavior change
   - `test:` test additions
3. Commit atomically — one logical change per commit

### Step 6: Session Hygiene

If a conversation is getting long (context pollution), the correct action is:
- Commit current progress
- Update documentation to capture decisions made
- Start a new session with CLAUDE.md as the entry point

CLAUDE.md and AGENTS.md are designed to restore context quickly. Trust them.

## Quick Reference: File Responsibilities

| Need to understand... | Read this |
|---|---|
| Full architecture & API contracts | `docs/蔚小芯智能体.md` |
| Development constraints & rules | `docs/蔚小芯开发规范.md` |
| Context Engine trigger strategies | `docs/context-engine.md` |
| Knowledge sync with 蔚园智答 | `specs/export-package.md` |
| RBAC role permissions | `specs/rbac-matrix.md` |
| AnswerCard response structure | `docs/ui-answer-card.md` |
| External system integration | `docs/integrations.md` |

## Anti-Patterns to Block

If you detect any of these, pause and flag to the user:

1. **Handler calling repository directly** — must go through service layer
2. **Hardcoded API keys** — must come from `.env` via config package
3. **Missing `sources[]` on policy answers** — compliance requirement
4. **New dependency not in approved stack** — needs explicit approval
5. **Schema change without migration file** — will break deployments
6. **Code without plan on multi-file changes** — Harness violation
7. **Skipping tests to move faster** — technical debt that compounds
