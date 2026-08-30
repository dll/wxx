# Context Engine — 实施摘录

> 完整定义见同目录 **`蔚小芯智能体.md`** §3.3、§3.3.1～§3.3.5。本文仅保留 **执行时常查表**。

## 主链路（不可偏离）

1. **结构化优先**（SQLite 中的流程节点、材料、入口、时限等）  
2. **FTS/BM25 全文检索** 召回 Top-K 片段  
3. **上下文拼装**（结构化结果 + 片段 + 版本/适用范围）  
4. **模型生成**，且 **`sources[]` 必填**（政策/条件类）

## 触发策略（速查）

| 典型问题 | 首选通道 | 备注 |
|----------|----------|------|
| 办事流程 / 材料 / 入口 / 时间地点 | 结构化库 | 最低延迟、强确定性 |
| 政策条款 / 适用对象 / 条件判断 | FTS/BM25 | 引用卡必填标题、段落、版本、生效时间 |
| FAQ 长尾同义 | 向量（可选） | 仅补召回，不单独作为唯一依据 |
| 跨文档对比 / 归纳 / 多跳 | Agentic RAG（按需） | 限制轮次与超时 |
| 高价值长文手册 | Long Context（按需） | 控制文档数量与成本 |
| 未命中 / 低置信 / 权限不足 | 兜底 | 禁止编造条款与关键数字 |

## 参数默认值（可与总纲不一致时以总纲为准）

- FTS Top-K、片段字数、向量 Top-K、缓存策略：见智能体 **§3.3.2**。  
- 切分与元数据必填字段：见 **§3.3.4** 与 **§6.8**。

## 质量指标（验收）

引用覆盖率、命中率、兜底率、过期引用率、权限拦截率、P95 时延等：见智能体 **§3.3.5**。

## 实现口径（2026-08-03 对齐）

> 本文档描述的是**检索策略**（主链路不可偏离）。代码层面存在两处实现，口径如下：

- **生产主路径**：`server/internal/service/chat_service.go` 的 `Ask()`（结构化优先 → FTS5/BM25 三阶段 → 相关性过滤 → `buildMessages` 拼装 → LLM → `sources[]` 附加）。这是线上问答实际走的实现，含 FAQ 缓存与降级链。
- **参考实现**：`server/internal/context_engine/` 包（意图分类 CE-06、命中片段 CE-07、来源加权 CE-09、相关历史选取 CE-10 等）。功能完整且有单测，但当前**未接入装配层**（`pkg/app/app.go` 无注册），与生产路径为平行双实现。
- **编排**：`server/internal/agent/` 为自研关键词加权意图路由 + goroutine 并行编排，**非 Eino**（go.mod 无 `cloudwego/eino` 依赖）。`agent/doc.go` 已同步说明。
- **推进建议**：后续可将 `context_engine` 作为生产路径的检索内核（通过 `KBSearcher`/`HistoryProvider` 适配器接入），并删除 chat_service 内联副本，消除双实现。

> 2026-08-03 全面核查见 `docs/蔚小芯学生教育工作需要全面分析.md`。

## CE 2.0 增量（2026-08-30 发布冲刺，A2）

`context_engine` 已于 commit ced3f61 接入生产问答链路（`ChatService.retrieveWithContextEngine`，
含 Temporal 降级路径）。本次增量（全部规则式，零 LLM 成本）：

| 能力 | 实现 | 说明 |
|------|------|------|
| 查询改写（CE-A2） | `rewrite.go` `RewriteQuery` | 剥口语装饰词 + 指代消解（结合最近历史话题）+ 空白归一；仅用于 FTS 召回，原始问题保留给意图/结构化检索 |
| 意图加权 | `scoring.go` `applyIntentBoost` | `IntentToResourceTypes` 偏好类型 TrustScore ×1.15，分类结果首次反馈进排序 |
| 可插拔重排 | `scoring.go` `Reranker` 接口 + `Engine.SetReranker` | 默认仅初排；可接 LLM listwise / 交叉编码器 |
| 命中片段贯通 | `SearchResult.Snippet`（瞬态）→ `AnswerCard.Sources[].Snippet` | 来源卡展示"命中段落"，优先于摘要（此前引擎算了片段但 service 层丢弃） |
| 历史一次取用 | `Engine.Query` 预取历史 | 改写与 CE-10 拼装共享，省一次查询 |
