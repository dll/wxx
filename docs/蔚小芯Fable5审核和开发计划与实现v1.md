# 蔚小芯 Fable5 审核报告、开发计划与实现方案 v1

> 编制日期：2026-07-25 ｜ 编制：Fable5 综合审核
> 输入：`蔚小芯审核GLM52入校离校智能体.md`、`蔚小芯DPV4P审核报告v1.md`、`蔚小芯TRAEAuto审核报告v1.md`、`蔚小芯GPT56SOL审核报告v1.md`、`docs/role-features.md`、`docs/蔚小芯角色功能.md`、`docs/蔚小芯待完成.md`、`specs/rbac-matrix.md`，并结合对 `server/`、`frontend/` 源码的实际核查。
> 目标：指出问题与改进方案，给出**满足全部角色、以学生角色为核心**的完善系统功能蓝图与落地开发计划。

---

## 0. 一页纸结论（TL;DR）

蔚小芯已具备扎实的工程骨架：分层清晰的 Go/Gin 后端（约 200 个源文件）、真实接入三家国产大模型、自研多智能体编排已接入问答主链路、结构化优先 + FTS/BM25 + 兜底的检索链路真实可用、六级 + teacher/assistant 的 RBAC 能力继承体系（约 90 个 capability）已落地、入校/离校真实可追溯数据已就位。功能广度可观，八角色路由页面大量存在。

但四份审核报告形成强共识：**当前不具备正式上线条件**（GPT56 判定 No-Go，DPV4P 5.4/10，TRAE 6.7/10）。核心风险集中在四条底线：

1. **身份与权限未以数据库为权威** —— 旧 JWT 可复活已停用/降权账号，短信验证码可任意绕过并回显，二维码登录会话可被窃取。
2. **敏感数据服务端闭环缺失** —— 隐私同意无服务端效力、PII 脱敏可被历史上下文绕过、模型 API Key 明文存储回显、`.env` 含活密钥、RBAC 数据范围过滤不级联导致知识/心理/审计越权。
3. **检索与契约一致性不足** —— 零命中仍强行输出来源（违反兜底硬约束）、FAQ 自动入库绕过审核形成错误自我强化、办事流程步骤六类详细信息字段链路未打通、同步包完整性/增量语义未实现。
4. **制品与质量门禁不可信** —— Android release 使用 debug 签名、v0.0.5 下载链路必然断链、5 个后端测试失败、前端测试近乎为零、`gofmt`/`analyze` 未清零、文档与实现漂移。

改进路线分三阶段：**S0 安全底线（上线阻断项，约 2 周）→ S1 检索与角色核心能力（以学生为核心，约 4 周）→ S2 全角色 AI 原生特色 + 运维体系（约 6 周）**。学生角色功能是产品差异化核心，本报告在第 5 章给出以「个人数字孪生」为数据底座的完整学生功能蓝图，并在第 6 章覆盖其余七角色。

---

## 1. 审核方法与范围

本次审核采用「多报告交叉 + 源码实证」双轨：先归并四份既有审核报告的发现并按主题去重，再对 `server/internal`、`frontend/lib`、`server/migrations` 做实际核查，判定每条问题在当前代码中**是否仍然存在**，避免仅凭文档高估或低估完成度。严重度沿用上线阻断（P0）/ 高优先级（P1）/ 工程治理（P2）三档。

一个关键前提：文档存在版本口径漂移。`role-features.md`（v1.0）将学生 25 项功能登记为「已有路由页面」，而更晚的 `蔚小芯角色功能.md`（v5.2）与待完成清单将其中大部分标为「未实现的 AI 特色功能」。**页面壳存在但 AI 原生能力尚未落地**，本报告一律以更晚的 v5.2 完成状态为准。

---

## 2. 现状核查：已具备的能力（肯定项）

在提出问题前，先明确已扎实落地的部分，避免返工。

- **分层架构落地**：handler / service / repository / agent / llm / middleware / model / auth / temporal 分层完整，主流程遵守「handler 不直接碰 repository/llm」的护栏（存在若干例外，见 3.4）。
- **多模型真实接入**：`llm/` 真实实现智谱、DeepSeek、讯飞星火三家客户端 + mock，符合「第三方 API、不本地部署」约束。
- **多智能体自研编排已接入主链路**：`agent/` 下 orchestrator/router/merger + qa/policy/process/emotion 子 Agent，`app.go` 中 `chatSvc.SetOrchestrator(...)` 已真实接入问答链路（goroutine 并行 + 结果汇聚）。「多智能体管理中心自研」约束已满足。
- **检索主链路真实可用**：结构化优先 → FTS5（BM25）→ 相关性预检 → 上下文拼装 → LLM → AnswerCard + sources + 兜底，功能存在（散落在 `service/chat_service.go`，非 `context_engine` 包内）。
- **RBAC 能力继承体系**：`auth/capabilities.go` 约 90 个 capability，六级 + teacher/assistant/guest 全部落地，DFS 递归授权 + 多父继承（college_admin 继承辅导员/教师/教辅三线），非占位。
- **入校/离校数据真实可追溯**：种子数据含滁州学院真实域名、具体到楼的地点、真实电话，`role_perspective.go` 为八角色注入回答视角。
- **PII 与兜底基础件已具雏形**：`SanitizeForLLM` / `CheckLLMOutput`、`pii.go` 检测中间件、`SelfProcessRead` 等能力门控到位。
- **Temporal 可选启用**：真实接入 SDK，`hostPort` 为空优雅降级，非占位。

这意味着改进以「加固与补全」为主，而非推倒重来。

---

## 3. 问题清单与改进方案（按主题）

严重度：🔴 上线阻断（P0）｜🟠 高优先级（P1）｜🟡 工程治理（P2）。「来源」列标注提出报告，多报告共识者加粗。

### 3.1 身份与鉴权（🔴 最高危，多为 P0）

| # | 问题 | 严重度 | 来源 | 改进方案 |
|---|------|--------|------|----------|
| S-01 | 旧 JWT 可复活已停用/降权/删除账号：每请求 JIT upsert 把 JWT 旧角色/scope 写回用户，用户不存在时以 active 重建 | 🔴 | GPT56 | JWT 只承载最小会话信息（userID + tokenVersion）；每请求以数据库状态为权威；停用/改密/降权递增 `token_version`，校验不匹配即拒绝；移除 `user_upsert` 的角色回写逻辑 |
| S-02 | 短信验证码任意六位数字均通过，且接口原样返回验证码；pending 游客立即签发可聊天 JWT | 🔴 | GPT56 | 接入真实短信通道；验证码服务端哈希存储 + 5 分钟有效 + 单次消费；响应绝不回显验证码；游客态最小权限，聊天需完成登录 |
| S-03 | 二维码登录会话可被第三方窃取（QR session ID 经第三方 `api.qrserver.com` 生成，公开状态接口向任意持有者返回 JWT） | 🔴 | GPT56 | 客户端本地生成二维码；引入 verifier / 一次性 code 换 token；状态接口校验轮询方身份，不直接吐 JWT |
| S-03b | **JWT 密钥弱/默认值 + 声明校验不足**（`change-me`/`dev-secret` 占位符仍可启动；未校验 nbf/iss/aud/jti） | 🔴 | **DPV4P/TRAE/GPT56** | release 模式空/默认密钥直接 panic（fail-closed）；强制 issuer/audience/jti/tokenVersion 校验；实现 refresh token（access 15min + refresh 7d）+ 黑名单/版本号吊销；密钥从平台 Secret 注入 |
| S-03c | 初始密码默认为学号（撞库风险）；Token 存 SharedPreferences（Web 可被同源 JS 读取） | 🟠 | GPT56 | 首登强制改密；Web 端改用 HttpOnly Cookie 或内存态 + 短时效 |

改进落点：`server/internal/middleware/{jwt.go,user_upsert.go}`、`server/internal/auth/`、`server/internal/service/auth_service.go`、`server/internal/handler/{auth_handler.go,qr_handler.go}`、`frontend/lib/pages/login_page.dart`。

### 3.2 敏感数据与合规（🔴/🟠，P0 为主）

| # | 问题 | 严重度 | 来源 | 改进方案 |
|---|------|--------|------|----------|
| SEC-01 | `.env` 含活 API 密钥（智谱/DeepSeek/讯飞） | 🔴 | DPV4P | 立即轮换三家密钥；`git log --all -- .env` 排查 + 必要时 `git filter-repo` 清历史；密钥仅从部署平台 Secret 注入 |
| SEC-02 | **用户模型 API Key 明文存储并回显** | 🔴 | **DPV4P/GPT56** | AES-256-GCM 信封加密（或 KMS）+ 轮换；写入后永不回显，仅返回掩码 + 末四位；密钥只在服务端调用时解密，禁入日志/错误响应 |
| SEC-03 | 隐私 consent 无服务端效力：登录即置 `Consented=true`，同意接口只记日志不持久化 | 🔴 | GPT56 | 建 `user_consent` 表（版本 + 时间戳 + IP）；敏感路由强制校验 consent；未同意拒绝进入问答/情感分析 |
| SEC-04 | PII 脱敏被历史上下文绕过：先存原始问题，历史原文再入模型上下文；情感分析传原始问题 | 🔴 | GPT56 | 统一 DLP 脱敏管线，入库/入上下文/入模型三处均走脱敏副本；情感分析同样脱敏；`pii.go` 补 GET 请求检查 |
| SEC-05 | 外部集成 SSRF + 统一集成 Token 外送（调用者可影响代理路径/目标，可探测内网/云元数据） | 🔴 | GPT56 | 取消通用 URL 代理；每集成固定 base URL；DNS 解析后拒绝私网/回环/链路本地地址；Token 按集成隔离 |
| SEC-06 | PII 明文入日志（`jwt.go`/`user_upsert.go` 学号明文） | 🟠 | DPV4P | 引入 `MaskStudentID()`；结构化日志脱敏字段 |
| SEC-07 | Prompt 注入防护不足 + 内容过滤仅关键词（谐音/拆字/生僻字可绕过） | 🟠 | TRAE | 输入侧注入检测 + 系统提示隔离；内容过滤引入分类模型或多策略（规则 + 语义），返回后二次过滤 + 兜底 |

改进落点：`server/internal/middleware/{consent.go,pii.go}`、`server/internal/repository/model_config_repo.go`、`server/internal/handler/{model_config_handler.go,integration_handler.go}`、`server/internal/service/{chat_service.go,integration_service.go}`。

### 3.3 RBAC 数据范围（🔴/🟠）

| # | 问题 | 严重度 | 来源 | 改进方案 |
|---|------|--------|------|----------|
| RB-01 | 普通学生可调 `/kb/export` 与 `/export` 下载全部 published 知识正文（导出路径不校验 scope） | 🔴 | GPT56 | 拆分 `self.answer.export`（导出本人回答）与 `school/college.kb.sync.export`（知识同步导出）两种能力；导出前强制 scope 校验 |
| RB-02 | 知识 CRUD/审核/导入接受调用者提交的 scope，辅导员可跨学院操作 | 🔴 | GPT56 | 请求范围必须是调用者范围的子集；服务端以 token scope 覆写，忽略客户端提交 |
| RB-03 | 心理/情感数据跨用户跨学院越权（学生统计未按本人过滤） | 🔴 | GPT56 | `emotion_repo` 学生查询强制 `user_id = self`；辅导员限本院；建立 Repository 层默认拒绝谓词 |
| RB-04 | **scope 不级联**（school 与精确匹配，缺 college→class；`cs2101` 看不到 `college=cs` 资源；FAQ 检索无 scope 过滤） | 🟠 | **DPV4P/GPT56** | Repository 层统一 `CanReadScope/CanWriteScope/CanReviewScope`，三层级联查询，默认拒绝 |
| RB-05 | 学院管理员可读全校审计日志 | 🟠 | GPT56 | 审计查询按 owner_scope 收窄，college_admin 限本院 |
| RB-06 | `LIKE '%role%'` 角色名匹配过宽（"student" 命中 "student_union"） | 🟡 | DPV4P | 改 JSON 精确匹配或 role 集合包含判断 |

核心改进：**在 Repository 层建立统一、默认拒绝的授权谓词**，杜绝在 handler 层零散判断。落点：`server/internal/repository/{kb_repo.go,emotion_repo.go,audit_repo.go,user_repo.go}`、`server/internal/auth/capabilities.go`（拆分导出能力）。

### 3.4 架构与代码质量（🟠/🟡）

| # | 问题 | 严重度 | 来源 | 改进方案 |
|---|------|--------|------|----------|
| A-01 | **Handler 越层直接调用 Repository/LLM**（`student_handler` 持 kbRepo、`feedback_handler` 持 screenshotRepo、`voice_handler` 直调讯飞、`education_handler`/`study_plan_handler` 持 `*sql.DB` 手写 SQL） | 🟠 | **DPV4P/TRAE/GPT56** | 抽取 `VoiceService`/`EducationService`/`StudyPlanService` 中介层 + 对应 Repo；`KBQuery`/`UserQuery` 迁至 `model` 包；加 CI 依赖方向检查防回归 |
| A-02 | **TraceID 未传播到 `context.Context`**，下游 LLM 读到空值，端到端追踪断裂 | 🟠 | **DPV4P/TRAE/GPT56** | `WithTraceID` 更新 Request Context；API/DB/模型/审计核验同一 TraceID |
| A-03 | **日志非结构化**（26 文件用 `log.Printf`，`LogLevel` 未使用） | 🟠 | **DPV4P/TRAE/GPT56** | 引入 `slog`/`zap`，含 trace_id/user_id/path/level |
| A-04 | **审计日志异步 goroutine 可能丢失/数据竞争**（请求结束后仍读 `*gin.Context`，对象池复用） | 🟠 | **DPV4P/TRAE/GPT56** | 请求内复制不可变 DTO 投递有界队列（channel + worker pool）+ shutdown 排空；`go test -race` 通过 |
| A-05 | **API 响应格式不统一 + `err.Error()` 直接暴露**（可能泄露 SQL/路径/供应商错误） | 🟠 | **DPV4P/TRAE/GPT56** | 统一 `ApiResponse{code,message,data,trace_id}` + `Success()/Fail()`；错误码 + 用户消息 + 服务端结构化日志 |
| A-06 | 动态 SQL 拼接列名/表名白名单不统一 | 🟠 | DPV4P/TRAE | 封装 `safeOrderBy(col, allowed)`；`BatchDelete` 表名常量化 |
| A-07 | `KBService` 20 个方法缺 `context.Context`；`student_handler` 静默吞错误后用 mock | 🟠 | DPV4P | 全链路补 `ctx`；错误必须记录结构化日志，不静默降级 |
| A-08 | 千行级文件（`education_handler` 1600 行 / `study_plan_handler` 1400 行 / `models.dart` 1500 行 / `app.go` setupRouter 约 500 行） | 🟡 | TRAE | 按业务域拆分文件与路由注册函数 |
| A-09 | 迁移逐句执行无事务，中途失败留半迁移；无回滚脚本 | 🟡 | GPT56/TRAE | 迁移包事务；提供 down 脚本；保持幂等 |

### 3.5 Context Engine / 检索质量（🔴/🟠）

| # | 问题 | 严重度 | 来源 | 改进方案 |
|---|------|--------|------|----------|
| CE-01 | **无关问题/零命中仍强行输出来源**（违反兜底硬约束，"地球上有什么" 命中奖学金办法得分 0 仍保留） | 🔴 | **DPV4P/GPT56/GLM52** | 空上下文预检查直接返回 `fallback=true, sources=[]`；`buildAnswerCard` 0 命中时 Conclusion 替换为兜底文案，禁止包装成"根据知识库整理" |
| CE-02 | **FAQ 自动入库跳过人工审核**，LLM 错误 FAQ 直接 published，错误自我强化；流程类问题被缓存绕过最新 process_steps | 🔴 | **TRAE/GLM52** | FAQ 自动入库设为 `pending` 需人工审核；入库前自动质检（敏感词 + 重复检测）；命中 `IntentProcess` 禁用 FAQ 缓存 |
| CE-03 | `context_engine/` 目录仅 `doc.go`，主链路实际散落在 `chat_service.go`，文档与结构不一致 | 🟠 | 本次核查 | 将检索主链路重构进 `context_engine` 包（或更新文档明确其为逻辑层），消除文档漂移 |
| CE-04 | FTS NEAR 查询语法兼容问题（`NEAR(奖* 学* 金*, 3)` 前缀通配符 FTS5 不支持，可能退化） | 🟠 | TRAE | 改 `NEAR("奖" "学" "金", 3)` 或去 NEAR 用 BM25 + 应用层字符距离重排 |
| CE-05 | FTS unicode61 单字分词中文语义粒度不足（无法理解"国家奖学金"⊃"奖学金"） | 🟠 | TRAE | 短期引入 `gojieba` 应用层分词；中期 FTS5 中文分词插件；长期向量混合检索 |
| CE-06 | 意图路由仅关键词 `strings.Contains`，覆盖不足且 map 遍历顺序不定导致同问题路由不稳定 | 🟠 | TRAE | 固定遍历顺序 + 路由置信度；中期引入 LLM few-shot 意图分类，关键词兜底 |
| CE-07 | Source 缺 `effectiveAt`/`snippet`，snippet 取 summary 非命中片段；前端来源 Chip 点击无反应 | 🟠 | GLM52/TRAE/GPT56 | `model.Source` 增 `effectiveAt`/`snippet`；引入 `kb_chunks` + chunk 级 FTS 保留条款号/页码/锚点；前端 Chip 展开详情 + 引用标记跳转 |
| CE-08 | 多智能体结果简单拼接无融合/冲突检测；置信度算术平均；子 Agent 职责边界模糊 | 🟠 | TRAE | 真正融合 + 冲突检测 + 加权平均；子 Agent 差异化能力 + 按类型调 topK；空 Agent 不参与均值 |
| CE-09 | 来源可信度不分层（Policy/FAQ 一视同仁）；`expired_at` 存在但检索未用 | 🟡 | TRAE/DPV4P | 按 resource_type 加权（Policy>Process>FAQ>Activity）+ 过滤过期 |
| CE-10 | 上下文固定 6 条历史，不考虑相关性，长对话关键信息被挤出 | 🟡 | TRAE | 对话摘要 + 相关性检索选取历史 |

### 3.6 角色功能缺失（🔴/🟠）

| # | 问题 | 严重度 | 来源 | 改进方案 |
|---|------|--------|------|----------|
| RF-01 | **办事流程步骤六类详细信息恒为空**：前端已定义并渲染 `ProcessStepDetail`（联系人/电话/办公时间/FAQ 等），但后端 `ProcessStep` 无对应字段，`ProcessEnhanced` 硬编码为空 | 🔴 | GLM52 | 迁移给 `process_steps` 加 8 字段（contact_person/contact_phone/contact_wechat/geo_lat/geo_lng/office_hours/media_urls/faq_items）+ 实体 + repo Scan + handler 读实体 + 种子真实数据；`KBRepo.GetProcessSteps()` 已实现但从未被调用，需接入 |
| RF-02 | `process_enhanced_page.dart` 是误导性死代码（硬编码"缓考申请"mock，却与 enrollment 共用端点名） | 🟠 | GLM52 | 删除或重命名，接真实端点 |
| RF-03 | 进度"全部完成"判定脆弱（信任前端传的 total_steps，降级时 stepDetails 为空导致永不 completed） | 🟠 | GLM52 | `StartOrResume` 以后端 `process_steps` 实际行数校准 |
| RF-04 | Router 把 Activity/FAQ 意图退化到 qa-default；Emotion Agent 子智能体未实现 | 🟠 | GLM52/本次核查 | 补全 Activity/FAQ Agent；实现 Emotion Agent 接入心理陪伴/情感预警协同 |
| RF-05 | 学生 25 项 AI 特色功能多为页面壳（v5.2 标未实现）；teacher/assistant 文档滞后于代码 | 🟠 | 本次核查/DPV4P | 见第 5、6 章功能蓝图逐项落地；同步更新文档完成度 |

### 3.7 测试、部署与运维（🔴/🟠/🟡）

| # | 问题 | 严重度 | 来源 | 改进方案 |
|---|------|--------|------|----------|
| Q-01 | **Android release 使用 debug 签名** | 🔴 | GPT56 | 独立生产 keystore 从 CI Secret 注入；缺签名参数直接失败 |
| Q-02 | **v0.0.5 下载制品在干净部署必然断链**（声明 v0.0.5，下载函数硬编码只允许 v0.0.4） | 🔴 | GPT56 | APK 存 GitHub Release/R2 不可变仓库；原子发布任务；版本从机器可读配置生成 |
| Q-03 | 5 个后端测试失败 + `gofmt` 9 文件未格式化 + Flutter analyze 397 项诊断 + 前端测试近乎为零 | 🔴 | GPT56/TRAE | 修复失败用例；`gofmt`/`analyze` 清零；补前端 widget/单元测试；CI 门禁：test + lint + analyze + race |
| Q-04 | **缺 API 速率限制的完善实现 + LLM 调用配额**（登录暴力破解、聊天刷爆费用；限流内存 map 无清理、可伪造 ClientIP） | 🔴 | **DPV4P/TRAE/GPT56** | IP 全局限流 + 登录单独限流 + 聊天按用户限流；用户级日/月 LLM 配额 + token 统计 + 费用告警；可信代理头 + TTL + 多实例共享存储 |
| Q-05 | 同步包完整性/增量语义未实现（无 ZIP/NDJSON/SHA-256/HMAC/packageId 幂等；固定 published LIMIT 5000，retired 永不传播，可能永久漏数） | 🔴 | GPT56 | 实现 HMAC-SHA256 常量时间验签 + package 幂等表 + 单调递增 change sequence；参见 `specs/export-package.md` |
| Q-06 | CORS 白名单硬编码过宽，`evil.com` 被回显，默认 `*` | 🟠 | TRAE/GPT56 | 域名移环境变量；release 严格白名单；禁 `*` 与 credentials 组合 |
| Q-07 | 缺 APM 监控告警 + 性能压测（P95 无验证）；备份自动化不明确 | 🟠 | TRAE | 接入 APM/健康探针；50/100/200 并发压测验证 P95≤300ms（网关）/≤2500ms（含模型）；备份定时任务 |
| Q-08 | 前端跨账号泄露（Provider 放根、退出不 reset、bookmark 共用键） | 🔴 | GPT56 | Provider 树以 `userId/sessionEpoch` 为 key + 统一 `reset()`；本地键 `bookmarks:<userId>` |
| Q-09 | HTTP Server 缺超时（slowloris）；Android release 允许明文 HTTP；`pubspec.lock` 未纳管；CI 与本地版本不一致；两小程序工程冲突；活动报名无唯一约束可超卖；中文 PDF 导出乱码 | 🟠 | GPT56 | 逐项修复：设 Server 超时 / 禁明文 / 纳管 lock / 统一版本 / 合并小程序工程 / 报名加唯一约束 + 事务 / PDF 用中文字体 |
| Q-10 | 文档与实现漂移（版本/域名/完成度：过期域名、teacher/assistant 标占位、Context Engine 标已完成实际未实现、三处版本号不一致） | 🟡 | DPV4P/GPT56/TRAE | "已完成"状态绑定自动化验收证据（SHA/命令/日期/结果）；版本/域名/制品名从机器可读配置生成 |
| Q-11 | 前端缺无障碍 + i18n + 列表懒加载 + 图片缓存；错误处理不统一 | 🟡 | TRAE | 补 Semantics 无障碍标注 / intl 国际化 / 分页懒加载 / cached_network_image |

---

## 4. 三阶段开发计划总览

修复与建设并行，按依赖关系分三阶段。每阶段设明确验收门禁，不达标不进入下一阶段。

### S0 — 安全底线（上线阻断项）｜约 2 周

目标：关闭全部 P0，使 GPT56 判定的四条底线达标（身份以 DB 为权威、敏感数据服务端强制闭环、检索契约一致、制品测试门禁可信）。

内容：3.1 全部（S-01~S-03c）、3.2 SEC-01~SEC-05、3.3 RB-01~RB-03、3.5 CE-01~CE-02、3.6 RF-01、3.7 Q-01~Q-05、Q-08。

**S0 验收门禁**：`go test ./...` 全绿 + `go test -race` 无竞争；`gofmt -l` / `flutter analyze` 清零；JWT 弱密钥启动即 panic；越权用例（学生导出全量知识、跨用户读心理数据、旧 token 复活）全部拒绝；零命中问答返回 `fallback=true, sources=[]`；Android release 用生产签名且下载链路连通。

### S1 — 检索质量 + 学生核心能力｜约 4 周

目标：检索链路专业化 + 以「个人数字孪生」为数据底座，落地学生角色 P1 核心功能闭环。

内容：3.4 A-01~A-05、3.5 CE-03~CE-10、3.6 RF-02~RF-04、3.7 Q-06~Q-07；第 5 章学生 P1 功能（数字孪生、今日速览、学习日记、性格洞察、课程学情看板、成长路径、思政学习、心理陪伴、办事流程增强、问答广场）。

**S1 验收门禁**：检索抽检命中率 ≥85%，政策类引用覆盖率 100%、流程类 ≥95%，兜底率 ≤10%；学生核心 10 功能端到端可用且数据来自真实底座；P95（网关）≤300ms。

### S2 — 全角色 AI 原生特色 + 运维体系｜约 6 周

目标：补齐辅导员/教师/教辅/学生会/管理员特色功能，形成跨角色数据闭环；建立监控、评测、A/B 基线。

内容：第 6 章全角色 P1/P2 功能；3.4 A-06~A-09、3.7 Q-09~Q-11；评测基线 200 条（8 域×25 条）+ LLM-as-judge（准确率/召回率/幻觉率）+ MRR/Recall@K/NDCG；APM + 备份自动化 + CI/CD 完善。

**S2 验收门禁**：八角色功能矩阵全项可用；评测基线自动跑通并出报告；多实例部署下限流/审计/配额正确;文档完成度与代码绑定验收证据。

### 里程碑甘特（相对周）

| 阶段 | W1 | W2 | W3 | W4 | W5 | W6 | W7 | W8 | W9 | W10 | W11 | W12 |
|------|----|----|----|----|----|----|----|----|----|-----|-----|-----|
| S0 安全底线 | ■ | ■ |  |  |  |  |  |  |  |  |  |  |
| S1 检索+学生核心 |  |  | ■ | ■ | ■ | ■ |  |  |  |  |  |  |
| S2 全角色+运维 |  |  |  |  |  |  | ■ | ■ | ■ | ■ | ■ | ■ |

---

## 5. 学生角色功能蓝图（核心）

学生是产品的核心角色，功能按「入学—在校—离校—就业」全生命周期与「政治·思想·心理·专业」四位一体育人体系组织，采用**七层粘性架构**：每日陪伴 → 学情洞察 → 成长规划 → 能力发展 → 校园生活 → 社区互动 → 成长激励。

### 5.1 数据底座：个人数字孪生（最高优先，先行落地）

个人数字孪生是所有学生个性化功能的数据底座，同时向上支撑辅导员看板、教师授课视图、学院/学校大屏。它以五维模型刻画学生：**学业、能力、思想、情感、社交**。

- **数据来源**：课程学情（成绩/进度）、学习日记与打卡、性格洞察（VARK + 大五）、情感打卡与预警、社区互动与竞赛记录。
- **能力输出**：五维雷达图 + AI 状态解读 + 差距分析 + 阶段建议。
- **接口**：`GET /api/v1/student/digital-twin`（能力 `self.twin.read`）；辅导员视图 `counselor.twin.board`、学院大屏 `college.twin.screen` 复用同一底座按 scope 收窄。
- **依赖链**：性格洞察 → 所有个性化功能（推荐/匹配/规划/学伴）；数字孪生 → 新生规划/成长路径/学院大屏；学习日记 → 班级学情日报（辅导员）。

> 建议先建统一的 `student_profile_snapshot`（快照）+ 各维度明细表，避免每个功能各自拉数据造成口径不一。

### 5.2 学生功能全景表

图例：✅ 已实现 ｜ ⚠️ 部分 ｜ 🔶 骨架/页面壳 ｜ ❌ 未实现。优先级 P0/P1/P2/P3。

**通用基线（全员，学生直接持有）**

| 功能 | Capability | 接口 | 状态 |
|------|-----------|------|------|
| AI 对话（SSE 流式） | `self.chat` | `POST /api/v1/chat` | ✅ |
| 知识大厅浏览 | `self.knowledge.read` | `GET /api/v1/knowledge` | ✅ |
| 个性推荐 | `self.recommend.read` | `GET /api/v1/recommendations` | 🔶 |
| 办事流程引导 | `self.process.read` | `GET /api/v1/student/process-enhanced` | ⚠️ 缺六类详细信息 |
| 语音 ASR/TTS | `self.voice` | `POST /api/v1/voice/{asr,tts}` | ✅ |
| 会话管理 | `self.session.*` | `GET/DELETE /api/v1/sessions` | ✅ |
| 导出本人回答 | `self.answer.export`（拟拆分） | `POST /api/v1/export` | ⚠️ 需拆能力 + 补格式 |
| 个人资料/模型配置 | `self.profile.write` | `GET/PUT /api/v1/user/profile` | ✅（Key 需加密） |
| 提交反馈 + 自动截图 | `self.feedback.submit` | `POST /api/v1/feedback` | ✅ |
| 校园文化 5 项（校歌/广播/讲座/活动/志愿） | `self.culture.*` | `GET /api/v1/culture/*` | 🔶 骨架，待接真实数据源 |

**P1 学生特色功能（角色内优先级，★=学生独占差异化）**

| # | 功能 | 说明 | Capability | 状态 |
|---|------|------|-----------|------|
| 1 | ★个人数字孪生 | 五维雷达 + AI 解读 + 差距分析（数据底座） | `self.twin.read` | 🔶→落地 |
| 2 | AI 今日速览 | 个性化日报（课程+截止+活动+天气+鼓励），AI 排序 | `self.briefing.read` | 🔶 |
| 3 | AI 学习日记 | 总结+知识回顾+自测题+明日计划，学习闭环 | `self.diary.write` | 🔶 |
| 4 | AI 校园生活助手 | 食堂/图书馆/报修/校车自然语言查询 | `self.campuslife.read` | 🔶 |
| 5 | AI 课程学情看板 | 知识点热力图+进度+班级匿名百分位+薄弱点推荐 | `self.course.analytics` | 🔶 |
| 6 | ★新生大学规划 | 四阶段时间轴（入学→在校→离校→就业）+ 里程碑 + AI 路线图 | `self.freshman.plan` | 🔶 |
| 7 | AI 思政理论学习 | 学习卡片推送 + AI 解读 + 自测题 | `self.political.study` | 🔶 |
| 8 | 思想成长档案 | 理论记录 + 入党进度 + 年度报告 + 一键导出 | `self.ideological.record` | 🔶 |
| 9 | AI 性格洞察 | VARK + 大五人格 + 学习策略（个性化底层模型） | `self.personality.read` | 🔶 |
| 10 | AI 课程地图 | 课程节点连线 + 已修/在修/待修 + AI 选课建议 | `self.coursemap.read` | 🔶 |
| 11 | 学习积分与成就 | 积分 + 徽章 + 等级 + 排行榜 | `self.achievement.read` | 🔶 |
| 12 | ★数字人导师（2D） | 2D 形象 + 语音对话 + 风格切换（温和/严格/幽默/思政） | `self.mentor.chat` | 🔶 |
| 13 | AI 学伴 | 出题互测 + 概念讨论 + 协作解题 | `self.buddy.interact` | 🔶 |
| 14 | ★AI 成长路径规划 | 学期里程碑 + 能力建议 + 路径模拟 + 动态调整 | `self.growth.plan` | 🔶 |
| 15 | AI 心理陪伴 | 心情打卡 + 关怀语 + 正念引导 + 危机识别 | `self.mental.read` | 🔶（依赖 Emotion Agent） |
| 16 | 每日打卡激励 | 连续天数 + 里程碑 + 断签保护 | `self.checkin.write` | 🔶 |
| 17 | AI 学习周报 | 各课时长分布 + 掌握趋势 + 归因分析 | `self.weekly.report` | 🔶 |
| 18 | AI 日程管家 | 课表导入 + 冲突检测 + 自然语言创建提醒 | `self.schedule.write` | 🔶 |
| 19 | 入党入团进度追踪 | 申请→积极分子→发展对象→预备党员→转正时间轴 | `self.party.progress` | 🔶 |
| 20 | AI 竞赛/项目匹配 | 画像匹配 + 匹配度 + 队友推荐 + 获奖概率 | `self.competition.match` | 🔶 |
| 21 | 问答广场 | 学生提问 → AI 先答 → 同学跟帖 → 辅导员/教师认证 | `self.qa.plaza` | 🔶 |
| 22 | 热点关注 | AI 生成"校园热议 TOP5" | `self.hot.topics` | 🔶 |
| 23 | 问答排行榜 | 热门问题/活跃答主/知识贡献三榜 | `self.qa.leaderboard` | 🔶 |
| 24 | 站内私聊 | 学友/学长/辅导员一对一 | `self.chat.private` | 🔶 |
| 25 | ★AI 办事流程增强 | 六类详细信息（联系人/地点/多媒体/FAQ/状态追踪/智能提醒） | `self.process.read` | ⚠️→P0 打通 |

**P2 学生深度功能**：AI 价值观引导、AI 心理健康评估、AI 专业知识图谱、AI 笔记助手、AI 课堂延伸、AI 学业预警、AI 模拟面试、智能简历生成、职业模拟器、AI 学友匹配、AI 前辈连线、AI 话题摘要。

**P3 学生生态扩展**：数字人导师动态形象升级（长期记忆 + 主动关怀）、职业模拟器数据驱动仿真。

### 5.3 学生功能落地节奏建议

- **S0 内必须打通**：办事流程增强六类信息（RF-01，直接影响入学/离校 MVP 体验）；数字孪生数据底座建表。
- **S1 优先 10 功能闭环**（形成每日使用粘性）：数字孪生、今日速览、学习日记、性格洞察、课程学情看板、成长路径、思政理论学习、心理陪伴、办事流程增强、问答广场。
- **S2 补齐其余 P1 + 启动 P2**：数字人导师、AI 学伴、竞赛匹配、日程管家、周报、积分成就、课程地图、热点/排行/私聊，以及 P2 学业预警/模拟面试/简历生成等就业向能力。

---

## 6. 其余七角色功能蓝图

角色继承：`sys_admin → school_admin → college_admin → {counselor, teacher, assistant} → student_union → student`。高阶角色自动继承低阶能力；数据范围按角色收窄（全校/本院/本人）。

### 6.1 学生会（student_union）

专属 4 项 + 全部学生功能。已实现：知识库提交 `union.kb.submit`、反馈管理 `union.feedback.list`。骨架：活动策划 `union.event.plan`、海报生成 `union.poster.gen`。

待建：P1 AI 活动策划助手、AI 海报/通知一键生成；P2 AI 招新助手、成员管理、问卷生成、热点追踪、活动数据分析。

### 6.2 辅导员/班主任（counselor）

专属约 22 项 + 学生会全部。已实现：情感预警、趋势报告、告警处理、会话查看、知识库 CRUD、知识审核、学生列表（7 项）。

P1 待建（12 项）：AI 今日关注 `counselor.daily_focus.read`、班级学情日报 `counselor.class.report`、学生数字孪生看板 `counselor.twin.board`（复用 5.1 底座）、AI 预测性预警 `counselor.prediction.read`、AI 干预方案生成 `counselor.intervention.write`、AI 谈心谈话记录 `counselor.talk.record`、谈话话术推荐 `counselor.talk.tips`、学生思想档案 `counselor.ideological`、班级性格画像 `counselor.class.profile`、社区问答管理 `counselor.community.manage`、热点话题感知 `counselor.hot_topic.sense`、流程步骤编辑 `counselor.process.edit`。

P2：谈话跟进提醒、班级打卡统计、智能群发助手、AI 月度工作简报、AI 会话洞察。

依赖：Emotion Agent 落地（RF-04）是预测性预警与心理协同的前提。

### 6.3 教师（teacher，扩展角色，骨架完成）

专属 9 项 + 学生会基线。权限边界：知识资源仅创建学业/课程类，禁止审核发布；情感预警默认关闭需单独授权；仅见本人授课摘要。

P1 待建：AI 备课助手 `teacher.lesson.prep`、AI 考试出题 `teacher.exam.gen`、AI 课堂互动 `teacher.class.interact`、AI 作业批改辅助 `teacher.grading`、班级学情热力图 `teacher.heatmap.read`、AI 教学反思 `teacher.reflection`、学习风格分布 `teacher.style.dist`、AI 今日授课概览 `teacher.daily.overview`、社区专业答疑 `teacher.community.qa`。

P2：答疑知识库管理、授课班级数字孪生视图、知识点覆盖检查、课程思政建议、个性化教学建议。

### 6.4 教辅（assistant，扩展角色，骨架完成）

专属 3 项 + 学生会基线。权限边界：知识资源仅创建事务/流程类；流程步骤协办（编辑材料）；情感预警禁止。

P1 待建：AI 排课冲突检测 `assistant.schedule.check`、AI 毕业资格审核 `assistant.grad.audit`、AI 考试安排优化 `assistant.exam.arrange`。

P2：通知批量生成、教学日历管理、材料模板库、学生信息查询、文档智能处理、流程自动化、流程步骤详情管理。

### 6.5 学院管理员（college_admin）

专属 5 项 + 继承辅导员/教师/教辅三线。已实现：本院用户管理 `college.user.read`、本院审计 `college.audit.read`（需按 RB-05 收窄至本院）、本院指标 `college.metrics.read`。

P1 待建：学院数字孪生大屏 `college.twin.screen`（复用 5.1 底座聚合）、AI 全量数据分析可视化 `college.data.analysis`。

P2：AI 决策建议、教师效能分析、课程质量评估、周报/月报。

### 6.6 学校管理员（school_admin）

专属 3 项 + 学院全部。已实现：智能体管理 `school.agent.write`（仅 sys/school）、用户管理（学校级）、修改用户。

P2 待建：全校数字孪生全景、AI 政策影响模拟、跨学院智能对比分析、AI 校级学情总览。

### 6.7 系统管理员（sys_admin）

专属 3 项 + 学校全部。已实现：全局配置 `system.settings.write`、全局审计 `system.audit.all`、重置任意用户密码 `system.password.reset`。

P2 待建：AI 运维助手、知识质量 AI 评估、AI 用户行为分析。

### 6.8 跨角色协同 TOP10（P1 优先落地顺序）

1. 个人数字孪生（数据底座）→ 学生 + 辅导员 + 教师 + 管理员四线
2. AI 今日速览 + AI 今日关注（学生 + 辅导员双钩子）
3. 问答广场 + 热点关注 + 社区问答管理（学生 + 辅导员 + 教师权威链路）
4. AI 学习日记 + 班级学情日报（学习数据闭环）
5. AI 备课助手 + 班级学情热力图（教师教学闭环）
6. 数字人导师（2D 静态版）
7. AI 谈心谈话记录 + 话术推荐（辅导员）
8. AI 课程学情看板（学生）
9. AI 排课冲突检测 + 毕业资格审核（教辅）
10. 新生大学规划 + 思政理论学习（学生）

---

## 7. 关键实现方案（选摘）

### 7.1 身份以数据库为权威（S-01/S-02/S-03/S-03b）

```
登录 → 校验凭据（真实短信/密码）→ 签发 access(15min)+refresh(7d)
       JWT payload = { userID, tokenVersion, iss, aud, jti, exp }
每请求 → 校验 iss/aud/exp/nbf → 按 userID 查库取角色/scope（不信 JWT 内角色）
       → 比对 tokenVersion，不匹配即 401
停用/改密/降权 → users.token_version += 1（旧 token 立即失效）
```
移除 `user_upsert` 的角色回写；游客态最小权限，聊天前必须完成登录并持久化 consent。

### 7.2 Repository 层统一授权谓词（RB-01~RB-05）

```go
// 默认拒绝，请求 scope 必须是调用者 scope 的子集
func CanReadScope(caller Principal, target Scope) bool
func CanWriteScope(caller Principal, target Scope) bool
func CanReviewScope(caller Principal, target Scope) bool
```
所有 kb/emotion/audit 查询强制经过谓词；导出能力拆分为 `self.answer.export`（本人回答）与 `school|college.kb.sync.export`（知识同步）。学生 emotion 查询强制 `user_id = self`。

### 7.3 零命中兜底（CE-01）

```
搜索 + Agent 均空 → 直接返回 { fallback:true, sources:[], conclusion:兜底文案 }
                    不调 LLM 包装、不输出任何来源
buildAnswerCard：命中数为 0 时用兜底文案覆盖 Conclusion
命中 IntentProcess：禁用 FAQ 缓存，强制走结构化 process_steps
```

### 7.4 办事流程六类信息打通（RF-01，S0 必做）

迁移给 `process_steps` 增列：`contact_person / contact_phone / contact_wechat / geo_lat / geo_lng / office_hours / media_urls(JSON) / faq_items(JSON)`；`ProcessStep` 实体与 repo Scan 同步；`student_handler.ProcessEnhanced` 改为读实体（接入已存在但从未被调用的 `KBRepo.GetProcessSteps()`）；补真实种子数据；`StartOrResume` 以后端实际步骤行数校准进度，杜绝 totalSteps=0 永不完成。

### 7.5 数字孪生数据底座（5.1）

先建 `student_profile_snapshot`（五维分数 + 更新时间）与各维度明细表，统一供数字孪生、辅导员看板、学院大屏读取，按 scope 收窄，避免各功能各自拉数据口径不一。

### 7.6 新增能力标准步骤（沿用 role-features.md §11）

1. `server/internal/auth/capabilities.go` 增 `Capability` 常量
2. 对应 `roleNode.capabilities` 追加（高阶自动继承）
3. `server/pkg/app/app.go` 注册路由 + `auth.RequireCapability(...)`
4. 前端 `lib/config/api_config.dart` 增路径常量
5. 前端 `lib/config/router.dart` 注册路由 + `CapabilityUtils.has(...)` 门控
6. 同步登记 `role-features.md` 与 `specs/rbac-matrix.md`

---

## 8. 验收指标（对齐 CLAUDE.md 质量门禁）

| 指标 | 目标 | 校验阶段 |
|------|------|----------|
| 网关 P95（不含模型） | ≤ 300ms | S1/S2 压测 |
| 问答整体 P95（含模型） | ≤ 2500ms | S1/S2 压测 |
| 核心接口成功率 | ≥ 99.5% | S2 |
| 智能问答命中率（抽检） | ≥ 85% | S1 评测 |
| 引用覆盖率（政策类 / 流程类） | 100% / ≥95% | S1 评测 |
| 兜底率 | ≤ 10% | S1 评测 |
| 后端测试 | 全绿 + `-race` 无竞争 | S0 门禁 |
| `gofmt -l` / `flutter analyze` | 清零 | S0 门禁 |
| 越权用例（导出/心理/审计/旧token） | 全部拒绝 | S0 门禁 |
| 评测基线 | 200 条（8 域×25）自动跑通 | S2 |

---

## 9. 风险与依赖

- **数字孪生先行**：它是学生 + 三类管理角色的共同底座，若延后将阻塞大量 P1 功能，必须在 S1 首周建表。
- **Emotion Agent**：心理陪伴（学生）与预测性预警（辅导员）共同依赖，属 RF-04，需在 S1 内落地。
- **密钥轮换不可逆延后**：`.env` 活密钥（SEC-01）一经发现应立即轮换，不等排期。
- **文档口径统一**：以 `蔚小芯角色功能.md` v5.2 为完成度权威，避免 `role-features.md` v1.0 造成的高估；所有"已完成"绑定验收证据（SHA/命令/日期）。
- **契约变更登记**：新增/变更端点须同步 `specs/api-contracts-index.md`，P2 模块（forecast/competition/plan/party/club/graduation）当前未登记，需补齐。

---

## 10. 附录：问题总量与分布

- 四份报告合计去重后约 90+ 条问题，其中 P0 上线阻断 14 项（GPT56 判定），高危 17 项（TRAE）。
- 待完成功能 98 项：学生 33 / 辅导员 17 / 教师 15 / 教辅 12 / 学生会 7 / 学院管理员 7 / 学校管理员 4 / 系统管理员 3。
- P1 全角色特色功能合计约 53 项（学生 25 为核心）。
- 各报告评分：DPV4P 5.4/10（需改善）、TRAE 6.7/10（基础扎实）、GPT56 No-Go（P0 关闭前不上线）、GLM52 ★★★★☆（修复入校/离校专项后可小规模试点）。

> 本报告为 v1，建议在 S0 完成后据实回填"已关闭"状态并出具 v2。







