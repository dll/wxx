# 蔚小芯 GPT-5.6 SOL 审核报告 v2

> 审核日期：2026-08-11  
> 审核基线：当前工作树 `ab6bc93`（`main`，审核开始时除本报告外无未提交改动）  
> 对照文档：`docs/蔚小芯GPT56SOL审核报告v1.md`、`docs/蔚小芯DPV4F审核报告v3.md`、`docs/蔚小芯开发规范.md`、`docs/蔚小芯智能体.md`、`docs/context-engine.md`、`specs/export-package.md`、`specs/rbac-matrix.md`、`docs/蔚小芯待完成.md`  
> 审核范围：PRD/需求覆盖、Flutter/Web/小程序、Go/Gin 后端、Context Engine/Eino/LLM、SQLite/FTS、RBAC/隐私/数据安全、依赖、性能、测试、CI/CD、部署运维和增强路线。

## 1. 最终结论

**结论：受控开发/内部演示可继续，正式生产和扩大真实学生数据试点仍为 No-Go。**

项目已经不是“只有骨架”：核心登录、JWT、RBAC、知识库、FTS/BM25、SSE 问答、AnswerCard、流程、情感分析、数字孪生、个人档案、校园服务、第三方应用中心和 AI 简讯均有代码与页面；v1/v3 中部分问题也确实已修复。但“功能数量多”不能替代上线底线。当前仍存在可造成敏感数据泄露、越权写入或供应链风险的问题，且全量质量门禁不稳定，不能声明“全部测试通过”或“高达标”。

### 1.1 评分

| 维度 | 评分 | 判断 |
|---|---:|---|
| PRD/需求覆盖 | 6.8/10 | P0 主链路较完整；`待完成.md` 仍列出大量 P1/P2/P3 未完成或骨架功能 |
| 架构与分层 | 6.5/10 | 技术选型合理，但 Context Engine 未接入，存在超大文件和 Handler/Service 越层 |
| 后端正确性 | 6.8/10 | 定向测试通过；全量测试因迁移/DOCX 用例超时，可靠性证据不足 |
| 前端质量 | 6.2/10 | 多端页面丰富，账号状态有改进；211 条 analyzer info，敏感状态仍使用 SharedPreferences |
| 安全与合规 | 5.0/10 | JWT/同意/范围有修复，但 pending 账号、SSRF、密钥回显/明文降级、情感告警范围仍是高风险 |
| Context Engine/RAG | 5.8/10 | 独立包实现较完整但未接线，生产问答仍直接调用 `kbRepo.Search` |
| 依赖与可复现构建 | 6.0/10 | Go 版本统一到 1.25；Flutter SDK/CI 与本地不一致，依赖策略仍偏宽 |
| CI/CD 与发布 | 6.0/10 | 有 CI、服务器部署和制品流程，但 CI 未包含 Flutter 质量门禁，发布任务可绕过分析/测试 |
| 运维与可观测性 | 6.3/10 | systemd+Caddy+SQLite+备份文档齐全；缺少可验证的恢复演练、SLO、告警和密钥轮换闭环 |
| **综合** | **6.1/10** | 功能面中上，生产安全和工程门禁未达标 |

### 1.2 严重级别

- **P0**：认证绕过、越权、敏感信息外泄、供应链/发布可信性或数据完整性问题；关闭并复测前禁止生产。
- **P1**：显著影响正确性、性能、合规或维护成本；应在下一迭代关闭。
- **P2**：治理和体验问题；纳入月度工程债清理。

## 2. 审核方法与限制

本次以当前代码和配置为准，回看 v1/v3 的每一项结论；使用 `rg`/PowerShell 进行只读结构与文本扫描，审读路由、中间件、Handler、Service、Repository、迁移、Flutter Provider、CI 和部署文件；执行定向 Go 测试、`go vet`、全量 Go 测试（带超时）和 `flutter analyze --no-pub`。未执行生产环境侵入式渗透、真实短信/SSO/学工系统联调，也未验证真实域名 DNS/备案状态，因此外部供应商和公网可达性只能按仓库证据判定。

## 3. PRD 与需求满足度

### 3.1 已基本满足的 P0/P1 主链路

| 需求 | 当前状态 | 证据/备注 |
|---|---|---|
| Go/Gin/JWT/RBAC | 基本满足 | `server/pkg/app/app.go` 路由、中间件、`internal/auth/capabilities.go`；旧 Token 权限回写已修复 |
| 问答 + SSE + 来源 | 部分满足 | `chat_service.go`、多 Agent 和 FTS 已运行；AnswerCard/来源结构仍需统一契约 |
| 知识库 CRUD、审核、导入导出 | 部分满足 | ZIP/哈希/HMAC/分片已存在；CRUD/审核范围校验仍不完整，增量不传播 retired |
| Context Engine | 部分满足 | `internal/context_engine` 有统一 Query，但文件明确标注“未接线”，生产走 `kbRepo.Search` |
| Flutter 多端 | 基本满足 | Web/Android/桌面页面丰富；小程序存在根目录与 `frontend/miniprogram` 两套工程 |
| 隐私同意、PII/内容过滤 | 部分满足 | consent 已持久化、聊天有脱敏和过滤；情感独立授权、历史消息和第三方处理协议仍需闭环 |
| 情感预警 | 部分满足 | 记录、统计、趋势、告警页面存在；告警更新未在 Repository 层复核调用者范围 |
| 语音 ASR/TTS、推荐、数字孪生、个人档案、校园服务、AI 简讯 | 功能已落地但需联调 | 依赖真实校方数据、供应商密钥和端到端回归，不能只以页面存在视为完成 |

### 3.2 明确未完成或仅骨架的需求

`docs/蔚小芯待完成.md`（2026-08-10）仍列出大量未完成项，包括流程详情/多格式导出、真实校园生活系统、学习日记、课程学情看板、课程地图、心理陪伴、问答广场、私聊、辅导员预测预警、教师教学分析、教辅自动化、校园文化真实音频/报名闭环、应用 WebView/审计联动、Temporal 与 Orchestrator 集成、Eino Graph/Workflow、Agent 优先级/熔断/重试、动态配置和上下文隔离。故不能把“98 项角色功能总表”解释为全部交付。

## 4. P0/P1 风险与证据

### P0-01 待审核游客仍获得业务 JWT

`AuthService.GuestRegister` 创建 `status=pending` 后立即调用 `jwtutil.GenerateToken`；`capabilities.go` 中 `guest` 仍包含 `SelfChat`、`SelfKnowledgeRead`、`SelfProcessRead`。虽然 `ENABLE_GUEST_REGISTER` 默认关闭，但一旦开启，待审核用户即可进入业务能力，和审核语义冲突。

**修复**：pending 只签发审核状态票据，或仅开放 `/guest/status`；业务 JWT 必须在审核通过后签发；增加 pending 对聊天、知识、流程、导出的 403 回归测试。

### P0-02 JIT 新用户默认 active/consented，仍存在信任边界过宽

`UserRepo.UpsertFromContext` 对数据库不存在的 JWT 用户插入 `status=active`，并将 consent 设为 true。该逻辑适合冷启动兼容，却把“持有可验签 JWT”当作注册授权来源，绕过 SSO/管理员导入和真实同意流程。

**修复**：JIT 仅允许带有受信 issuer、一次性 SSO subject 和服务端映射的回调；未知用户进入 pending/quarantine，consent 默认 false；删除用户后旧 Token 必须不可重建。

### P0-03 外部系统代理未做白名单/SSRF 防护

`integration_handler.go` 接受任意路径和查询参数，`integration_service.go` 直接拼接 `baseURL + path` 并携带统一 Bearer Token。未见固定主机、解析后 IP 禁止私网/回环/链路本地、禁止跨主机重定向和路径白名单。

**影响**：可探测内网、访问云元数据或把服务 Token 发送到错误目标。

**修复**：每个系统使用固定 base URL 和允许路径；自定义 `http.Client` 解析并拒绝私网 IP，禁止重定向跨主机；上游请求只携带最小字段和最小权限凭据；补 SSRF/DNS rebinding 测试。

### P0-04 用户模型配置存在完整密钥回显和明文降级

`model_config_handler.go` 将 `ModelConfigService.Get` 的完整配置直接返回；`model_config_repo.go` 读取后解密到响应对象。`repository/crypto.go` 在 `WXX_ENCRYPTION_KEY` 缺失时记录警告并原样存储/返回明文。即使配置了 AES-GCM，普通用户 GET 也不应得到可用 API Key。

**修复**：生产启动时强制要求密钥，禁止明文 fallback；GET 只返回 provider、模型、是否配置和末四位；密钥仅在服务端调用时解密，采用 KMS/信封加密、轮换和访问审计。

### P0-05 情感告警更新缺少资源范围复核

`EmotionRepo.UpdateStatus` 仅按 `alert_id` 更新；Handler/Service 传入用户名但没有把 `ownerScope/ownerID/role` 作为 SQL 谓词。列表和趋势有部分学院过滤，但“知道 alert ID 即可更新”的路径未证明安全。

**修复**：`GetByAlertID`/`UpdateStatus` 必须接收 `UserContext`，学生只允许本人，辅导员只允许管辖班级/学院，管理员按矩阵授权；跨范围统一 404/403，并增加 A/B 学院隔离测试。

### P0-06 知识写入、审核和导入仍信任客户端 scope

`KBHandler.CreateResource/UpdateResource/Import` 将请求中的 `OwnerScope/OwnerID/Status` 传给 `KBService`；`KBService.Create/Update/ApproveResource/ImportResources` 主要接收 username，没有统一 `UserContext` 和 `CanRead/CanWrite/CanReviewScope` 校验。粗粒度 capability 不能防止学院 A 操作学院 B 或直接提交 published。

**修复**：所有 KB Service 接收调用者上下文；Repository 层默认拒绝并强制 scope 谓词；导入进入 quarantine/pending；状态转换由服务端决定；批量、审核、下架和字典查询复用同一授权函数。

### P0-07 知识包协议有实现但不满足完整安全契约

`KnowledgePackageService` 已生成 ZIP、`manifest.json`、NDJSON、SHA-256、可选 HMAC 和复合 cursor，这是 v1 的实质进展。但仍有三项缺口：

1. HMAC 在 `secret==""` 时不签名，导入也不强制签名；生产配置缺失不会失败。
2. `ImportPackage` 未见 `packageId` 幂等表/一次性消费，重复导入可能再次执行。
3. `ListSincePage` 固定 `WHERE status='published'`，`retired` 记录不会进入 delta；因此下架无法同步到消费者。

**修复**：生产强制 HMAC，验签/哈希失败零落库；建立 package receipt 唯一约束；增加单调 change sequence 或 revision 事件表，创建/修改/审核/下架均可增量同步。

### P1-08 JWT 与前端本地存储仍不符合高敏应用基线

Token、用户名、角色、同意状态和能力清单存储于 `SharedPreferences`。Web 端相当于可被同源脚本读取的存储，移动端也不是 Keystore/Keychain-backed。账号切换虽已为收藏增加用户键、多个 Provider 增加 reset，但 `main.dart` 仍把大量业务 Provider 放在根树，不能证明所有缓存、请求和异步任务都在 logout/401 时清理。

**修复**：移动端使用 `flutter_secure_storage`；Web 使用同域 BFF + HttpOnly/Secure/SameSite Cookie；建立统一 session epoch，账号变化时重建用户态 Provider、取消请求并清理所有内存/缓存。

### P1-09 Context Engine 生产链路分叉

`internal/context_engine/engine.go` 明确标注“实验性/未接线”；`chat_service.go` 和各 Agent 直接调用 `kbRepo.Search`。这会让意图分类、结构化优先、过期过滤、snippet、trust score、历史拼装和来源契约在不同路径重复实现，容易出现“文档声称已完成、生产行为未使用”的漂移。

**修复**：把 Engine 作为唯一检索入口，注入 UserContext 和策略版本；删除重复检索路径；为零结果、低相关结果、过期资源、跨范围和 sources 生成增加契约测试。

### P1-10 PII/心理数据的第三方处理仍缺少可证明闭环

项目有 `RequireConsent`、`SanitizeForLLM` 和内容过滤，但情感分析会处理 `message_text` 并写入 `emotion_logs`；需要证明历史消息、异步情感任务、trace/log、反馈截图、模型供应商请求都使用脱敏副本。当前同意记录只有 users 的布尔列，没有协议版本、用途、供应商范围、撤回时间和保留期限。

**修复**：建立 consent ledger（版本、用途、供应商、时间、撤回）；高敏心理文本默认不外发；脱敏失败阻断；日志/trace/截图做 DLP；完成供应商数据处理协议、地域和保留策略。

## 5. 架构、代码质量与性能

### 5.1 需要优化的结构

当前存在多个超大文件：`student_service.go` 1887 行、`student_handler.go` 1672 行、`document_service.go` 1591 行、`app.go` 1486 行、`kb_repo.go` 1326 行、`chat_service.go` 1270 行；Flutter `models.dart` 2853 行、`home_page.dart` 2132 行、多个管理页超过 1200 行。项目规范要求单文件约 400 行以内，现状显著超标。

建议按领域拆分 DTO/Handler/Service/Repository，页面拆为 section、dialog、form 和 state；禁止 Handler 直接持有数据库/LLM；统一传递 request context，禁止后台 goroutine 丢失取消信号；为长事务、导入、文档解析和异步通知增加队列或可恢复任务。

### 5.2 性能与可靠性

- 全量 Go 测试在当前 Windows 环境以 `-timeout=30s` 复现超时：`handler.TestKBHandler_ListResources_Empty`、`repository.TestKBRepo_ListSince` 卡在迁移 SQL；`service.TestExtractDocKeywordsPerformance` 卡在 DOCX/XML 解析。此前 `go test ./...` 也在 120 秒内未完成。应把迁移拆分/缓存、为测试数据库设置连接与锁策略，并为 DOCX 解析设置大小、元素数和超时上限。
- `chat_service.go` 含异步 FAQ 写入和情感任务；需确认数据库关闭、请求取消时不会后台继续写入。测试日志已出现“database is closed”异步写入失败。
- SQLite 单机方案适合小规模试点，但必须明确并发写入、WAL、磁盘耗尽、备份锁和升级回滚策略；不得把本地单实例性能数据外推为全校 SLO。
- AvatarCard 无限动画会持续重绘；页面不可见时应暂停。大列表、图表、个人档案聚合应增加分页、缓存和超时预算。

## 6. 文档规范与一致性

### 6.1 优点

文档地图、角色矩阵、部署方案、同步包、隐私政策、待完成清单较丰富；v1/v3 审核报告具备证据、严重级别、验收标准和路线图，适合作为治理基础。

### 6.2 需要立即治理的漂移

- `DPV4F审核报告v3.md` 仍把域名备案/IP 直连列为阻断，但当前 `Caddyfile` 已有 `wxx-agent.online`、www 重定向和 `129.211.223.113` IP 站点块；应改为“仓库已配置，公网 DNS/备案待外部验证”，避免继续引用过时结论。
- 部署文档同时保留当前 Lighthouse+SQLite、Cloudflare Pages、历史 Vercel/Turso、旧域名和旧命令，读者容易误部署。应标记唯一正式路径，历史内容移到 archive。
- “已完成”状态没有统一绑定提交 SHA、命令、日期和结果；`待完成.md` 与角色功能文档仍显示大量骨架功能，应建立机器可读需求矩阵。
- PRD/运行 Schema、AnswerCard、错误码、路由命名和 cursor 仍有分叉风险。建议以 OpenAPI/JSON Schema 生成 Go/Dart DTO，所有导入导出做往返测试。

## 7. 依赖与供应链

- Go 模块要求 `go 1.25.0`，CI 已使用 Go 1.25；应增加 `go mod verify`、依赖漏洞扫描和 SBOM 产物。
- Flutter `pubspec.lock` 已存在并被跟踪，这是进步；但 CI 使用 Flutter 3.22.0，本地审核环境为不同版本，且大量 caret 约束，需固定 Flutter SDK、`flutter pub get --enforce-lockfile` 和升级窗口。
- `WXX_ENCRYPTION_KEY`、JWT、HMAC、LLM、SSO 和上游 Token 依赖环境变量；当前缺失时部分配置会静默降级。生产启动应执行 fail-fast 配置校验，禁止 debug 默认密钥和明文密钥。
- 依赖中包含 Eino 与 Temporal，但核心编排仍是 Go 原生并行；应明确“已使用/可选/实验性”，删除误导性死依赖或完成接线。

## 8. 测试与 CI/CD

### 8.1 本次实测

| 命令 | 结果 |
|---|---|
| `go test ./server/internal/auth ./server/internal/config ./server/internal/context_engine ./server/internal/jwtutil` | 通过 |
| `go test ./server/internal/repository ./server/internal/service ./server/internal/handler` | 通过（缓存结果） |
| `go vet ./server/...` | 通过 |
| `go test ./... -count=1 -timeout=30s` | 失败：handler/repository/service 测试超时；见 §5.2 |
| `flutter analyze --no-pub` | 211 条 info，命令非零退出；无 error/warning，但不能称“零诊断” |
| `flutter test --no-pub` | 本次未获得稳定结果；与并行 Flutter 命令发生启动锁竞争，应在 CI 单独串行执行 |

### 8.2 CI 缺口

`.github/workflows/ci.yml` 目前覆盖 Go vet、gofmt、build、race test 和基础文档检查，但没有 Flutter analyze/test/build、Android 签名校验、制品 manifest 校验、小程序构建、依赖审计和安全扫描。`deploy-frontend.yml` 可直接 `pub get`、构建和部署，不依赖前端质量门禁；这使生产部署可以绕过失败的分析或测试。

**要求**：拆出 `quality-gate`，顺序执行 Go 全量测试（明确超时）、Flutter lockfile/analyze/test/web build、APK release 证书指纹、Cloudflare/服务器健康检查、回滚演练；部署 job 必须 `needs: quality-gate`。

## 9. 部署与运维

当前正式方案文档为 Lighthouse systemd + Caddy + 本地 SQLite，包含健康检查、每日备份和回滚概览；Caddy 配置也已加入 IP 站点块，较 v3 的访问入口结论有更新。仍需补齐：

1. 生产域名、证书、DNS、备案和 `/api` 反代的外部合成监控；
2. SQLite 备份加密、异地复制、恢复 RTO/RPO 演练和备份完整性校验；
3. systemd/Caddy/应用/LLM/磁盘/数据库锁的指标和告警；
4. 密钥轮换、最小权限、SSH/服务器补丁、审计日志防篡改和保留策略；
5. 灰度发布、数据库迁移回滚、制品 SHA256/签名记录；
6. 明确 Cloudflare Pages 备用入口与正式入口的 CORS、API 上游和版本一致性。

## 10. 修复路线图

### 0～48 小时：冻结风险

1. 关闭游客注册或移除 pending 的聊天/知识/流程能力；禁止未知 JWT JIT 创建 active 用户。
2. 外部代理改为白名单目标并增加 SSRF 测试；生产缺少 JWT/HMAC/加密密钥时 fail-fast。
3. 模型配置 API 只返回掩码；完成现有明文数据迁移和密钥轮换。
4. 情感告警、反馈详情、知识 CRUD/审核/导入全部补齐 Repository 层范围谓词。
5. 暂停正式发布，标记全量测试为红灯；修复迁移测试超时和异步数据库关闭问题。

### 1 周：恢复可信试点

1. 接通 Context Engine 唯一生产入口，统一 Query、sources、snippet、低置信和过期策略。
2. 完成知识包强制 HMAC、package 幂等、retired/change sequence 和往返验收。
3. 建立 consent ledger、情感独立授权、第三方模型 DLP/日志审计和供应商协议清单。
4. 引入安全存储/session epoch，验证 A/B 账号切换、401、离线缓存和异步请求取消。
5. CI 增加 Flutter/小程序/Android/依赖安全门禁，部署 job 依赖质量门禁。

### 1 个月：工程收敛与扩展

1. 按领域拆分超大文件，建立 400 行软上限、依赖方向检查和 API 契约生成。
2. SQLite WAL/备份恢复/锁等待压测；建立 P95/P99、错误率、LLM 超时、检索命中率和成本预算。
3. 固定 Flutter/Go 工具链和 SBOM，清理死代码与未接线 Eino/Temporal 依赖。
4. 按 `待完成.md` 的 P1→P2→P3 顺序推进，每项绑定验收测试、数据依赖和外部联调负责人。

## 11. 上线验收门槛

- 认证：停用、降权、删除、改密后旧 Token 均立即失效；pending 不得访问业务能力。
- 授权：学生、学院 A、学院 B、校级管理员分别执行知识、情感、反馈、导出和代理越权矩阵，全部拒绝并不泄露存在性。
- 隐私：未同意/撤回/版本过期均阻断敏感能力；测试 PII 不出现在模型请求、日志、trace 和非必要库字段。
- 同步：缺文件、哈希篡改、签名篡改、重复 package、retired、同一时间窗分页分别有自动化验收。
- 发布：Flutter analyze/test/build、Go test/vet/race、APK 生产签名、制品哈希和健康检查全部通过。
- 运维：完成备份恢复、迁移回滚、密钥轮换、证书续期和故障演练，给出 RTO/RPO/SLO 记录。

## 12. 总结

v2 的准确表述应是：**项目功能面较丰富，核心技术路线基本成形，部分 v1 阻断项已经修复；但安全边界、数据范围、知识同步完整性、敏感数据治理和自动化质量门禁仍未达到生产级高标准。** 当前最优先不是继续堆叠页面，而是关闭 §4 的 P0、让全量测试稳定通过、把 Context Engine 和统一契约真正接入生产，再依据可复现实测重新评估 Go/受控试点资格。

---

*报告生成：2026-08-11。证据来自当前工作树静态审读与本地命令；未执行侵入式生产渗透和真实第三方系统联调。*
