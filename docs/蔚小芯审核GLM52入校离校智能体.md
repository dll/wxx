# 蔚小芯 — 入校 / 离校智能体审核报告

> **审核范围**：新生入学（enrollment）/ 毕业生离校（graduation）两个智能体的设计与实现
> **审核基准**：`docs/蔚小芯智能体.md` v1.5、`docs/蔚小芯角色功能.md` v5.2、`docs/context-engine.md`、`specs/`
> **审核方式**：源码逐文件通读 + 契约对照（设计 ↔ 实现 ↔ 种子数据）
> **审核人**：GLM-5.2
> **审核日期**：2026-06-17
> **结论**：**基本可用、可演示**（P0 达标），但与「AI 办事流程增强」等规划契约存在 **3 处高优先级缺口**，需在试点前修复。

---

## 0. 一句话结论

入校 / 离校**不是两个独立的智能体进程**，而是由「**流程指引子智能体（process-guide）+ 结构化流程端点（/student/process-enhanced）+ 办事进度持久化（process_records）**」三件套共同实现的同一套能力，前端通过 `enrollment_page` 切换 `enrollment`/`graduation` 复用。实现质量**结构正确、数据真实（滁州学院）**，但**前端契约与后端实体字段脱节**，导致规划要求的「联系人/电话/办公时间/FAQ」四类详细信息恒为空。

---

## 1. 审核对象定位

| 维度 | 入校（enrollment） | 离校（graduation） |
|------|--------------------|--------------------|
| 后端资源 ID | `process-registration-2026` | `process-graduation-2026` |
| 知识库标题 | 新生入学报到流程 | 滁州学院毕业生离校流程 |
| 步骤数 | 7 步（003_seed_knowledge.sql） | 8 步（020_seed_graduation_process.sql） |
| 数据来源 | 滁州学院真实域名（cw.chzu.edu.cn） | 滁州学院真实域名（ybt.chzu.edu.cn） |
| 前端入口 | `enrollment_page.dart`「入学流程」chip | `enrollment_page.dart`「离校流程」chip |
| 后端端点 | `GET /student/process-enhanced?type=...` | 同左 |
| 进度持久化 | `process_records`（flow_type=enrollment） | `process_records`（flow_type=graduation） |
| 自然语言问答路径 | `Router → process-guide Agent` | 同左 |

**关键澄清**：仓库里存在一个 `frontend/lib/pages/student/process_enhanced_page.dart`，但它是一个**写死的「缓考申请」mock 页**（标题、步骤、截止日期全部硬编码），**与入学/离校无关**，属于误导性命名，详见 §5.4。

---

## 2. 架构与数据流（实测）

```
┌───────────────────────────── 前端 ─────────────────────────────┐
│ enrollment_page.dart                                          │
│   └─ [入学流程] / [离校流程] chip → EnrollmentProvider        │
│        ├─ loadFlow()                                          │
│        │   GET /student/process-enhanced?type=enrollment|graduation
│        │   → { processes:[{steps:[…]}], answer_card:{…} }     │
│        ├─ _restoreFromBackend()                               │
│        │   POST /process/records/:flow/start  (StartOrResume) │
│        ├─ toggleStep / completeAll / resetProgress            │
│        │   POST /process/records/:flow/progress               │
│        └─ （降级）POST /chat/ask  通用对话接口                 │
└────────────────────────────────────────────────────────────────┘
                              │
┌──────────────────────────── 后端 ──────────────────────────────┐
│ student_handler.ProcessEnhanced                               │
│   ├─ mapFlowToResource(type) → resource_id                    │
│   ├─ kbRepo.GetByResourceID()   → AnswerCard（结论+来源）     │
│   └─ kbRepo.GetProcessSteps()   → steps[]                     │
│                                                               │
│ process_record_handler (StartOrResume / UpdateProgress)       │
│   └─ process_record_service → process_record_repo             │
│                                                               │
│ 自然语言问「入学/离校怎么办理」时另走：                         │
│   Router(intent=Process) → Orchestrator → ProcessAgent         │
│        → kbRepo.Search(BM25) → 拼装 → LLM → AnswerCard        │
└────────────────────────────────────────────────────────────────┘
```

**两条链路并存**：结构化端点（确定性最高）+ 自然语言 RAG（覆盖「口语化提问」）。这与 `蔚小芯智能体.md` §3.3「结构化优先 + 全文检索为主」的主路线**一致**。

---

## 3. 与设计契约的逐项对照

### 3.1 Context Engine 触发策略表（§3.3.1）

| 契约要求 | 实现状态 | 证据 |
|----------|----------|------|
| 流程类 → 结构化库（SQLite），输出「步骤清单+入口+材料+截止时间+sources」 | ✅ 达标 | `process-enhanced` 端点直接读 `process_steps` 表，返回 step/materials/entry_url/deadline/location/notes + answer_card.sources |
| 政策条款 → FTS/BM25 + 引用卡 | ✅ 达标 | `kb_repo.Search` 用 `kb_fts MATCH` + BM25 `rank`，AnswerCard 携带 sources |
| FAQ 长尾 → 向量检索（可选） | 🔶 未做向量，用 BM25+Jaccard 兜底 | `chat_service.faqLookup`（阈值 score≤-8.0 且 Jaccard>0.6）。P0 可接受，向量按规划本就是「可插拔」 |
| 未命中/低置信 → 兜底，不编造 | ✅ 达标 | `ProcessEnhanced` 资源未命中时返回空 steps；`ProcessAgent` 无结果时 Confidence=0.1、Content="" |
| 来源可追溯（硬约束：必须带 sources） | ⚠️ **部分达标，见 §4.1** | 结构化端点带 sources；但多智能体/LLM 兜底链路 sources 可能为空 |

### 3.2 引用与合规输出（§3.3.3）

| 契约要求 | 实现状态 |
|----------|----------|
| 每条回答必须返回 `sources[]`，≥1 条 | ⚠️ 结构化端点达标；RAG 链路在「检索 0 命中」时 `buildAnswerCard` 仍返回空 sources（仅降 Confidence=0.3、Fallback=true） |
| sources 含：标题/链接/版本/生效时间/段落摘要 | ❌ **缺字段**。`Source` 结构体只有 ResourceID/Title/Version/SourceLink/RelevanceScore，**无 effectiveAt、无 snippet/段落摘要**（详见 §4.2） |
| 不同角色不同粒度 | 🔶 部分。`role_perspective.go` 给 LLM 注入角色视角提示词（学生「我该怎么做」/ 辅导员「如何指导」/ 管理员「政策依据」），但**仅作用于自然语言链路**，结构化端点对所有角色返回相同内容 |

### 3.3 权限与治理（§3.3.4、§9.2）

| 契约要求 | 实现状态 | 证据 |
|----------|----------|------|
| 检索前按 ownerScope/roleScope/status=published 过滤 | ✅ 达标 | `kb_repo.Search` SQL 含 `status='published' AND owner_scope AND role_scope LIKE '%role%'` |
| 个人敏感信息不得入可检索正文 | ✅ 达标 | 种子 steps 仅有「身份证/录取通知书」等材料名，无真实 PII |
| 办事进度记录按用户隔离 | ✅ 达标 | `process_record_repo` 全部按 `user_id` 过滤；handler 取 `middleware.GetUserContext().UserID` |
| 路由鉴权 + 能力门控 | ✅ 达标 | `app.go:612-614,645` 三端点均挂 `auth.RequireCapability(auth.SelfProcessRead)` |

---

## 4. 高优先级问题（P0，试点前必修）

### 4.1 【契约缺口】步骤级「联系人/电话/办公时间/FAQ」四类信息恒为空

**这是本次审核发现的最严重问题。**

- **规划要求**：`角色功能.md` §3.1.8「AI 办事流程增强」明确要求每个步骤补充 **6 类**详细信息：①联系人+联系方式 ②办理地点+办公时间 ③多媒体辅助 ④步骤级 FAQ ⑤办理状态追踪 ⑥智能提醒。`待完成.md` P0 第 1 项也标注「⚠️ 部分完成」。
- **前端**：`models.dart` 的 `ProcessStepDetail` 定义了 `contact/phone/officeHours/faq` 字段，`enrollment_page.dart` 第 438-503 行**渲染了**这四个字段（联系人+电话、地点+办公时间、FAQ 折叠面板）。
- **后端实体**：`entity.go` 的 `ProcessStep` 结构体**只有** `StepOrder/Title/Materials/EntryURL/Deadline/Location/Notes`，**没有** `contact/phone/office_hours/faq` 字段。
- **后端 handler**：`student_handler.ProcessEnhanced`（第 366-379 行）把这四个字段**硬编码为空**：

```go
steps = append(steps, gin.H{
    ...
    "contact":      "",
    "phone":        "",
    "office_hours": "",
    "faq":          []gin.H{},
})
```

- **结论**：前端 UI 预留了展示位、后端表结构没字段、handler 永远填空字符串 → **这四类信息在界面上永远不显示**。学生点开任意步骤，只能看到「材料/地点/入口/截止/备注」，看不到「找谁、电话多少、几点上班、常见问题」。

**修复建议**（最小改动）：
1. `migrations/021_*.sql`：给 `process_steps` 加列 `contact TEXT DEFAULT ''`、`phone TEXT DEFAULT ''`、`office_hours TEXT DEFAULT ''`、`faq TEXT DEFAULT '[]'`（faq 存 JSON：`[{"q":"…","a":"…"}]`）。
2. `entity.go` `ProcessStep` 补对应字段。
3. `kb_repo.GetProcessSteps` 的 `Scan` 与 SELECT 列补齐。
4. `student_handler.ProcessEnhanced` 改为读实体字段而非硬编码空串。
5. 种子 SQL 补真实联系人/电话/FAQ（滁州学院各办理点公开信息）。

### 4.2 【契约缺口】Source 结构体缺 effectiveAt / snippet

`蔚小芯智能体.md` §3.3.3 与附录 A.4 的 AnswerCard schema 都要求每条 source 含 `effectiveAt`（生效时间）与 `snippet`（段落摘要）。当前 `model.Source` 只有 `ResourceID/Title/Version/SourceLink/RelevanceScore`，**缺这两个字段**。`AnswerCard` 的 `sources` 因此无法满足「引用卡必须包含生效时间/段落」的硬约束。

修复：`model.Source` 增字段，`buildAnswerCard`/`ProcessEnhanced` 从 KBResource 映射时填充（KB 资源已有 `effective_at` 与 `summary`）。

### 4.3 【健壮性】降级 fallback 会用空问题命中通用对话，且无来源时仍可「无依据回答」

- `enrollment_provider.loadFlow()` 的降级路径用 `'新生入学流程及所需材料'` / `'毕业生离校手续办理流程及步骤'` 调 `/chat/ask`。该链路在 LLM 可用但 KB 检索 0 命中时，仍会生成一段**无 sources 的自然语言回答**（`buildAnswerCard` 仅置 `Fallback=true / Confidence=0.3`，但 `Conclusion` 已是 LLM 输出）。
- 这与 §3.3.3「sources 为空或置信不足：**禁止**给出确定条款与金额/时间等关键数字；必须输出兜底提示」**冲突**。

修复：`buildAnswerCard` 在 `len(results)==0` 时，应把 `Conclusion` 替换为兜底文案（引导联系辅导员/学工办），而非保留 LLM 自由生成的文本；或在 LLM prompt 中强约束「无参考资料时只允许输出兜底语」。

---

## 5. 中低优先级问题

### 5.1 Router 把 Activity / FAQ 意图都退化到 qa-default

`router.go` `intentToAgent` 中 `IntentActivity` 与 `IntentFAQ` 都映射到 `qa-default`，注释写「活动类暂由 QA 处理」。后果：用户问「离校典礼/迎新活动怎么报名」时不会激活 process-guide，走通用 QA。对入校/离校主流程影响小（流程词命中 Process 意图即可），但属于**规划中「活动通知」Agent 的未实现占位**，`待完成.md` 已记录「Eino 框架实际集成 60%」。

### 5.2 FAQ 持久化缓存把「流程问题」也缓存进 KB，可能与结构化流程数据混淆

`chat_service.faqStore` 在「单 LLM 调用 + 有 sources」时把整个 AnswerCard JSON 写入 `kb_resources`（resource_type=FAQ）。当用户用自然语言问「新生入学流程」时，生成的 AnswerCard 会被缓存为 FAQ，下次同类问题直接命中缓存——**绕过了结构化流程端点与最新 `process_steps` 数据**。若运营更新了流程步骤，缓存的 FAQ 仍返回旧答案（TTL 24h + 持久化，且 retired 需用户反馈触发）。

建议：对命中 `IntentProcess` 的问题**禁止** `faqStore`，或缓存 key 排除流程类意图。

### 5.3 进度持久化「全部完成」判定脆弱

`process_record_service.UpdateProgress`（第 115 行）：`if rec.TotalSteps > 0 && len(sorted) >= rec.TotalSteps → completed`。但前端 `_persistProgress` 传的 `total_steps` 来自 `enrollment_provider`，而后者在**降级到 `/chat/ask`** 时用的是 `chatData.data.stepDetails`（可能为空）。若 totalSteps=0，则永远进不了 completed；若结构化端点返回的 steps 数与后端 process_steps 行数不一致（例如管理员临时删了某行），`completed` 状态会错乱。

建议：`StartOrResume` 时以**后端 `process_steps` 实际行数**为准校准 totalSteps，而非信任前端传入。

### 5.4 【可维护性】`process_enhanced_page.dart` 是误导性死代码

`frontend/lib/pages/student/process_enhanced_page.dart` 整页是「缓考申请」硬编码 mock（标题、3 个步骤、截止「2026-05-20」全写死），与入学/离校无关，且 `api_config.processEnhanced` 常量被它和真正的 `enrollment_provider` **共用同一端点名**。极易让后续维护者误以为这就是「办事流程」页面。建议删除或重命名。

### 5.5 离校流程内容比入学流程更扎实

入学（`003`）与离校（`020`）种子质量对比：
- 离校 KB `content` 含完整「办理时间/办理项目 8 项/咨询渠道（含电话 0550-3510022）」，且单独迁移文件、注释清晰。
- 入学 KB `content` 也有「报到时间/地点/前准备/现场 7 步/材料清单」，质量尚可，但**无咨询电话**，且与助学贷款、转专业等流程混在同一个 003 文件里，可维护性略差。
- 两者 `entry_url` 均指向真实 chzu 域名，`expired_at` 离校设了 `2026-12-31`，入学**未设过期时间**——建议补齐，否则 §3.3.5「过期引用率趋近 0」无法靠机制保障。

---

## 6. 做得好的地方（肯定项）

1. **数据真实可追溯**：入校/离校 KB 与 steps 全部绑定滁州学院真实系统（一表通、财务处、教务处），域名、电话、地点具体到楼，不是占位 `example.edu`。这是上线上报的最大底气。
2. **结构化优先路线落地正确**：`process_steps` 表 + `process-enhanced` 端点实现了规划要求的「最低延迟、最强确定性」主链路，不依赖 LLM 即可返回完整步骤。
3. **进度持久化闭环完整**：`StartOrResume`（每用户每流程唯一进行中记录）+ `UpdateProgress` + `ListMine`，前端可勾选/全选/重置并跨端恢复，符合「办事记录持久化」契约。
4. **RBAC 与脱敏到位**：三端点统一 `SelfProcessRead` 能力门控；`chat_service` 对用户问题与 LLM 返回均做 PII 脱敏 + 内容安全过滤（`util.SanitizeForLLM` / `CheckLLMOutput`）。
5. **角色视角分化**：`role_perspective.go` 为 8 个角色分别注入回答视角约束，体现了「同一流程不同角色不同答法」的设计意图。
6. **多智能体编排可观测**：`Orchestrator` 日志记录路由结果、参与 Agent 数、汇聚置信度与 sources 数，便于试点期复盘。

---

## 7. 修复优先级排序

| # | 问题 | 优先级 | 工作量 | 风险 |
|---|------|--------|--------|------|
| 1 | §4.1 步骤四类信息（联系人/电话/办公时间/FAQ）字段链路打通 | **P0** | 中（1 迁移+实体+repo+handler+种子） | 直接影响「能不能办成事」的核心体验 |
| 2 | §4.3 无 sources 时仍输出 LLM 自由文本 | **P0** | 小 | 合规风险（可能编造金额/时间） |
| 3 | §4.2 Source 补 effectiveAt/snippet | P1 | 小 | 引用合规未达标 |
| 4 | §5.2 流程类问题禁止 FAQ 缓存 | P1 | 小 | 旧答案残留 |
| 5 | §5.3 totalSteps 以后端为准 | P1 | 小 | 状态错乱 |
| 6 | §5.4 删除/重命名 process_enhanced_page.dart | P1 | 极小 | 可维护性 |
| 7 | §5.1 Activity/FAQ Agent 占位实现 | P2 | 中 | 影响活动类提问 |
| 8 | §5.5 入学 KB 补过期时间/咨询电话 | P2 | 极小 | 治理完善度 |

---

## 8. 验收建议（试点前 Checklist）

- [ ] 迁移 021：`process_steps` 增 `contact/phone/office_hours/faq` 列
- [ ] 入校/离校每一步至少补 1 条真实联系人+电话+办公时间（或标注「详见班级群通知」）
- [ ] 离校每步补 1-2 条高频 FAQ（如「图书馆欠费怎么查」「校园卡余额退到哪」）
- [ ] `buildAnswerCard` 在 0 命中时 Conclusion 替换为兜底文案
- [ ] `Source` 结构体补 `effectiveAt/snippet` 并在两处 builder 填充
- [ ] 入学 KB 设 `expired_at`（建议 `2026-09-30`，报到季结束）
- [ ] 删除 `process_enhanced_page.dart` 死代码
- [ ] 回归测试：学生/辅导员两角色分别打开入学、离校，确认步骤详情四类信息均渲染、进度可勾选可恢复、sources 非空

---

## 9. 评分汇总

| 维度 | 评分 | 说明 |
|------|------|------|
| 架构正确性 | ★★★★☆ | 结构化优先 + RAG 兜底的双链路设计符合规划 |
| 数据真实性 | ★★★★★ | 滁州学院真实数据，可上线 |
| 契约符合度 | ★★★☆☆ | sources 合规与步骤详情未达标（§4.1/4.2/4.3） |
| RBAC/安全 | ★★★★☆ | 能力门控 + 脱敏 + 过滤齐全 |
| 可维护性 | ★★★☆☆ | 死代码命名误导、迁移文件组织略乱 |
| **综合（试点就绪度）** | **★★★★☆** | **修复 §4.1/4.3 后即可进入小规模试点** |

---

> **审核人备注**：入校/离校作为 P0 MVP 的两端能力，**骨架与数据均已就位**，核心缺口集中在「前后端字段契约不一致」这一类**工程一致性**问题，而非设计缺陷。建议优先按 §7 的 1、2 项闭环后，再启动试点班级运行。
