---
name: wxx-harness
description: 蔚小芯项目的 Harness 工程纪律。当用户要实现功能、修复 bug、添加接口、修改业务逻辑或进行任何非简单代码变更时触发。也在出现"Plan"、"方案"、"开发"、"实现"、"添加功能"、"修复"等短语时触发，或当发现有人未经方案直接写代码时主动介入。
---

# 蔚小芯 Harness 工作流

本技能在蔚小芯项目中强制执行 **Harness 工程纪律**：每一次非简单改动都必须遵循 **方案 → 人工审核 → 编码 → 验证 → 提交**，并内置文档和架构护栏。

目标是防止两种常见失败模式：（1）复杂任务直接上手写代码，写到一半发现思路错误；（2）代码能跑但违反分层规则、漏更新文档、或引入了被禁止的依赖。

## 何时激活

以下场景必须激活本技能：
- 改动涉及多个文件
- 新增或修改 handler、service、repository、agent
- 修改数据库 schema 或迁移脚本
- 修改中间件（鉴权、RBAC、审计）
- 对接外部 API（智谱、DeepSeek、讯飞、学工系统、一表通）
- 影响 Context Engine 管道

对于单行修复（拼写错误、日志调整、显而易见的 bug），可跳过直接进入提交步骤。

## 工作流程

### 第一步：编写方案

在写任何代码之前，使用 `templates/plan.template.md` 模板编写方案：

```markdown
# 任务方案 — 【标题】
## 背景与目标
## 范围（做 / 不做）
## 技术要点（栈、接口、风险）
## 步骤拆分
## 验收标准
## 回滚与检查点（Git / 数据）
```

方案应当说明：
- 影响哪些层（handler / service / repository / agent / context_engine / llm）
- 哪些文档需要更新（`docs/`、`specs/`、CLAUDE.md）
- 是否需要新的数据库迁移脚本
- RBAC 影响（6+2 种角色中哪些受影响）
- 是否需要 `sources[]` 追溯（策略/流程类回答路径必须）

方案提交给用户审核，**等待确认后**再开始编码。

### 第二步：架构护栏检查

实现前验证以下约束。如有违反，立即停下与用户讨论：

**分层规则**（参见 `server/README.md`）：
- handler → service → repository（单向调用）
- handler 绝不能直接调用 repository 或 llm
- repository 绝不能依赖 HTTP 或模型 API

**禁止引入的依赖** — 以下内容未经书面变更审批绝不能引入：
- 本地部署大模型（所有模型均为 API 调用：智谱/DeepSeek/讯飞）
- Coze 或任何第三方智能体 SaaS
- Docker/容器/集群要求（轻量单机部署）
- 任何强制容器化的依赖

**知识管道** — 主路径始终是：
1. 结构化查询（SQLite 直查）→ 2. FTS/BM25 检索 → 3. 上下文拼装 → 4. 模型生成
- 向量检索和 Agentic RAG 是可选项，绝不是默认项
- 政策/流程类回答必须附带 `sources[]` — 禁止编造引用和关键数字

**RBAC** — 每个新接口必须声明允许访问的角色。六级基线：
`sys_admin > school_admin > college_admin > counselor > student_union > student`
扩展角色：`teacher`、`assistant`

### 第三步：实现编码

按方案逐步实现。每完成一个子任务：

1. 实现代码改动
2. 运行静态检查：`make lint`（go vet）
3. 运行测试：`make test`
4. 测试失败必须修复后再继续 — 不允许累积失败测试

关键实现约定：
- Go 后端遵循 `server/internal/` 包结构
- 问答接口使用统一的 `AnswerCard` 结构（参见 `docs/ui-answer-card.md`）
- 所有知识查询经由 Context Engine（`internal/context_engine/`），handler 禁止直接查库
- 敏感操作记录审计日志（`audit_logs` 表）
- 错误响应包含 `trace_id` 用于调试定位

### 第四步：验证

提交前运行完整检查：

```bash
make lint          # go vet 静态检查
make test          # 单元测试
```

同时人工确认：
- 新增/变更接口已登记在 `specs/api-contracts-index.md`
- 新接口的 RBAC 权限已记录在 `specs/rbac-matrix.md`
- schema 变更有对应迁移文件（`server/migrations/`）
- 知识资源类型变更已更新 `specs/resource-schema.md`

### 第五步：提交与文档

每次完成一个有意义的增量都必须：

1. 如果行为发生变化，更新 `docs/` 或 `specs/` 中的相关文档
2. 编写清晰的提交信息，遵循约定式提交：
   - `feat:` 新功能
   - `fix:` bug 修复
   - `docs:` 仅文档变更
   - `refactor:` 无行为变化的重构
   - `test:` 测试新增
3. 原子化提交 — 每次提交只包含一个逻辑变更

### 第六步：会话卫生

如果对话过长（上下文污染），正确做法是：
- 提交当前进度
- 更新文档以记录已作出的决策
- 开启新会话，以 CLAUDE.md 为入口恢复上下文

CLAUDE.md 和 AGENTS.md 的设计目标就是快速恢复上下文，放心使用。

## 快速参考：文件职责表

| 需要了解… | 阅读 |
|---|---|
| 完整架构与 API 契约 | `docs/蔚小芯智能体.md` |
| 开发约束与规则 | `docs/蔚小芯开发规范.md` |
| Context Engine 触发策略 | `docs/context-engine.md` |
| 与蔚园智答的知识同步 | `specs/export-package.md` |
| RBAC 角色权限 | `specs/rbac-matrix.md` |
| AnswerCard 回答结构 | `docs/ui-answer-card.md` |
| 外部系统对接 | `docs/integrations.md` |

## 需要阻止的反模式

发现以下情况时，立即暂停并提醒用户：

1. **Handler 直接调用 repository** — 必须经过 service 层
2. **硬编码 API 密钥** — 必须通过 `.env` 经由 config 包加载
3. **政策类回答缺少 `sources[]`** — 合规性要求
4. **引入审批清单外的新依赖** — 需要明确授权
5. **schema 变更无迁移脚本** — 会导致部署失败
6. **多文件改动未编写方案** — 违反 Harness 纪律
7. **跳过测试赶进度** — 技术债会加速膨胀
