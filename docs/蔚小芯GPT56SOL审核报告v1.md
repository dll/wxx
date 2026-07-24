# 蔚小芯 GPT-5.6 SOL 审核报告 v1

> 审核日期：2026-07-25  
> 审核对象：`main` 分支，基线提交 `1d8ef579a39a4f9120952f0ab55f2ca21e2f51df` 及审核时当前工作树  
> 审核范围：Flutter/Web/Android、小程序、Go/Gin 后端、Context Engine、知识同步契约、数据库、权限与隐私、测试、CI/CD、部署文档  
> 总体结论：**No-Go。在 P0 问题关闭并完成复测前，不建议正式上线，也不建议扩大真实学生数据试点。**

---

## 1. 执行摘要

蔚小芯已经具备较完整的产品骨架：Flutter 多端客户端、Go/Gin 服务端、JWT/RBAC、SQLite FTS、AnswerCard、来源字段、流程步骤、知识导入导出、模型适配器和较丰富的业务页面均已落地。项目文档也清楚表达了“结构化优先、FTS/BM25 为主、sources 可追溯、最小权限、隐私授权、同步包可验签”等正确方向。

但当前实现与这些核心承诺之间仍有明显断层，且断层集中在认证、数据范围、隐私、同步完整性和发布链路等上线底线：

- 旧 JWT 可把被停用、降权或删除的账号重新写回数据库，并恢复旧角色。
- 短信验证码可被任意六位数字绕过，接口还直接返回验证码；待审核游客立即获得可聊天的 JWT。
- 二维码登录把会话标识发送给第三方二维码服务，而公开状态接口会向持有该标识者返回 JWT。
- 隐私同意没有服务端持久化效力；聊天历史和情感分析仍可能把原始个人信息发送给第三方模型。
- 普通学生可调用知识导出接口下载全部已发布知识正文；知识 CRUD、审核、导入也没有统一的数据范围授权。
- 心理数据、审计数据和反馈截图存在跨用户、跨学院或匿名读取风险。
- 同步包的 ZIP、SHA-256、HMAC、幂等和可靠 cursor 契约尚未实现，下架资源不会进入增量同步。
- 外部系统集成接口接近“携带服务 Token 的任意 GET 代理”，存在 SSRF 与凭据外送风险。
- Flutter 切换账号后会保留上一用户聊天和收藏；Android release 使用 debug 签名；v0.0.5 下载链路在干净部署中会断裂。
- 当前后端核心测试仍有 5 个失败用例，Flutter 分析存在 397 项诊断，前端自动部署不依赖前端质量门禁。

因此，本报告不建议采用“边上线边修”的方式。应先冻结正式发布入口，按第 9 节路线图关闭 P0，再通过第 10 节的验收矩阵恢复试点。

### 1.1 分领域结论

| 领域 | 结论 | 主要原因 |
|---|---|---|
| 认证与会话 | 红色 / 阻断 | 旧 Token 可恢复账号；验证码和二维码登录可被绕过或窃取 |
| 权限与数据范围 | 红色 / 阻断 | 知识、心理、审计等资源缺少 Repository 层强制范围过滤 |
| 隐私与模型安全 | 红色 / 阻断 | consent 无服务端效力；原始 PII/心理文本可能外发 |
| Context Engine | 橙色 / 高风险 | 未形成统一 Query；零相关结果仍可能成为来源；整篇索引缺少真实 snippet |
| 同步与契约 | 红色 / 阻断 | 无 HMAC/哈希/ZIP；cursor 可能漏数；retired 不传播 |
| Flutter 与发布 | 红色 / 阻断 | 跨账号状态泄露、debug 签名、下载制品断链 |
| 测试与 CI | 红色 / 阻断 | Go 回归失败；Flutter CI 缺失；文档“零警告”与实测不符 |
| 工程结构 | 橙色 / 高风险 | 多个千行级文件、Handler 越层、迁移和审计写入不可靠 |

### 1.2 严重级别定义

- **P0 / 上线阻断**：可导致认证绕过、权限恢复、敏感数据泄露、供应链/发布失真、同步篡改或正式环境不可用。修复前不得生产上线。
- **P1 / 高优先级**：会显著影响正确性、可靠性、合规性或扩大故障半径。应在一周级迭代中关闭。
- **P2 / 工程治理**：短期不一定直接造成事故，但会持续增加回归、维护和交付成本。应纳入一月级治理。

---

## 2. 审核基线、方法与限制

### 2.1 已对照的项目基线

按项目协作索引要求，本次以以下文档为主要判定依据：

1. `docs/蔚小芯开发规范.md`
2. `docs/蔚小芯智能体.md`
3. `docs/context-engine.md`
4. `specs/export-package.md`

同时检查了 `specs/resource-schema.md`、`specs/rbac-matrix.md`、部署与用户导入文档，以及代码、迁移和 CI 配置。

### 2.2 审核方法

- 静态检查路由、中间件、Handler、Service、Repository、SQL 和 Flutter 状态生命周期。
- 逐项对照 RBAC、ownerScope/ownerID/roleScope、隐私授权、AnswerCard、sources 和同步包契约。
- 检查 Android/Web/小程序构建、版本、制品下载、Token 存储和账号切换。
- 执行 Go 单元测试、`go vet`、`gofmt` 检查，以及 Flutter 分析、最小测试和小程序 Node 测试。
- 对高风险链路从入口追到持久化或外部模型调用，避免仅凭文件名或注释下结论。

### 2.3 审核限制

- 审核基于当前本地工作树；工作树本身存在未提交业务改动，因此本报告反映的是“基线提交 + 当前修改”的实际状态，而不是纯净 HEAD。
- 未对生产域名执行侵入式渗透、真实账号越权操作或第三方密钥有效性验证。
- 未调用真实短信、模型、学工、一表通或 SSO 服务，外部系统结论来自代码路径与契约检查。
- 本报告未记录任何本地密钥值。

---

## 3. P0：上线阻断问题

### P0-01 旧 JWT 可恢复已停用、降权或删除账号

**证据**

- `server/internal/middleware/jwt.go:83-133` 直接信任 Token 中的角色、范围和 consent Claims。
- `server/internal/middleware/user_upsert.go:27-45` 每次请求按 JWT 上下文执行 JIT upsert。
- `server/internal/repository/user_repo.go:457-495` 会把 JWT 中的旧角色、ownerScope、ownerID 写回现有用户；用户不存在时直接以 `active` 状态重建。

**影响**

管理员即使已经停用、删除或降权账号，只要旧 Token 尚未过期，持有者仍可能恢复旧权限，甚至重新创建数据库用户。这会反向破坏后台管理动作，是典型的授权源倒置。

**改进方案**

- JWT 只承载 `sub/session_id/token_version` 等最小会话信息。
- 每次请求以数据库中的状态、角色、范围和 token version 为准。
- 停用、改密、降权、删除时递增 token version 或吊销会话。
- 删除请求链路中的“按 JWT 回写数据库角色/状态”行为；JIT 创建只能发生在受信 SSO 回调中。

**验收标准**

- 账号停用、删除或降权后，所有旧 Token 立即返回 401/403。
- 旧 Token 不得修改或重建用户记录。
- 增加停用、删除、降权、改密四类会话吊销回归测试。

### P0-02 短信验证码与游客审核可被绕过

**证据**

- `server/internal/service/auth_service.go:61-89` 中任意六位验证码均直接通过。
- `server/internal/handler/auth_handler.go:240-267` 把验证码原样返回客户端。
- `server/internal/service/auth_service.go:92-131` 为 `pending` 游客立即签发 JWT。
- `server/internal/auth/capabilities.go:173-181` 赋予游客聊天、知识和流程读取能力。

当前虽已加入 IP 令牌桶，但它不能修复“验证码本身无效”和“审核前已授权”的逻辑缺陷。

**影响**

攻击者可冒用任意手机号批量注册，并在未审核状态下调用聊天和知识接口，造成账号滥用、模型费用消耗和数据访问风险。

**改进方案**

- 接入真实短信通道；验证码仅保存服务端哈希，设置 5 分钟有效期、单次消费、失败次数和手机号/设备/IP 联合限流。
- 响应永不包含验证码，日志也不记录完整验证码。
- `pending` 账号只获得“查询审核状态”能力，不签发业务 Token或由网关强制拒绝业务路由。

**验收标准**

- 错误验证码和已消费验证码始终失败。
- API 响应、日志、trace 中均不存在完整验证码。
- pending 用户调用聊天、知识、流程、导出接口均返回 403。

### P0-03 二维码登录会话可被第三方窃取

**证据**

- `frontend/lib/pages/login/login_page.dart:679-683` 将包含 QR session ID 的登录 URL 发送到 `api.qrserver.com` 生成二维码。
- `server/internal/handler/qr_handler.go:74-102` 的公开状态接口向任何持有 session ID 的调用者返回状态和 JWT。
- `frontend/lib/pages/login/login_page.dart:723-745` 桌面端轮询该接口并直接保存返回 Token。

**影响**

第三方二维码服务能看到 session ID。若其或中间链路轮询状态接口，用户确认后即可先于桌面端取得 JWT。该问题不需要破解密码，只需要掌握二维码内容。

**改进方案**

- 在客户端本地生成二维码，不把登录秘密发送给第三方。
- 桌面端创建会话时同时生成仅保存在本机的 verifier；状态接口必须校验 verifier。
- Token 不应通过可匿名轮询接口直接返回；使用一次性 code，由创建会话的客户端交换 Token。
- 会话确认绑定创建端、扫描端、用户和短有效期，并保证一次消费。

**验收标准**

- 抓取二维码内容但没有创建端 verifier 时无法获得 Token。
- 第三方二维码服务不再接触 session ID。
- 同一确认结果只能交换一次，重放返回失败。

### P0-04 隐私同意无效，原始 PII 和心理文本仍可能外发

**证据**

- `server/internal/middleware/jwt.go:45` 登录即把 `Consented` 设置为 true。
- `server/internal/service/auth_service.go:198-211` 的同意接口只记录日志，不持久化协议版本、时间或撤回状态。
- `server/internal/middleware/consent.go:9-39` 已有中间件但未覆盖敏感业务路由。
- `server/internal/service/chat_service.go:159-165` 先保存原始问题；`293-299` 又把历史原文加入模型上下文，绕过当前问题的脱敏副本。
- `server/internal/handler/chat_handler.go:55-68` 对每次聊天异步执行情感分析，并传入原始问题。

**影响**

用户未真正授权也能进入敏感功能；姓名、电话、学号、身份证、学业或心理描述可能未经有效同意发送给第三方模型并持久化。

**改进方案**

- 建立 consent 表，保存协议版本、同意时间、用途、供应商范围和撤回状态。
- 聊天、情感、语音、导出等敏感路由强制执行 consent。
- 在进入历史存储和模型请求之前执行统一 DLP/脱敏，历史消息也必须使用脱敏副本。
- 高敏心理文本默认不外发；如确需外发，应单独授权并明确供应商、地域和保留策略。

**验收标准**

- 未同意、协议版本过期或已撤回用户无法调用敏感能力。
- 测试手机号、身份证号、学号等不出现在模型请求、日志、trace 和非必要数据库字段中。
- 脱敏失败时请求被阻断并产生安全审计。

### P0-05 心理数据存在跨用户、跨学院越权

**证据**

- `server/internal/repository/emotion_repo.go:111-143` 的学生统计未强制按本人过滤。
- `server/internal/repository/emotion_repo.go:53-60,215-228` 与 `server/internal/handler/emotion_handler.go:162-205` 未统一实施学院范围。
- `server/pkg/app/app.go:518-530` 将统计与告警能力暴露给多个角色。
- 告警更新按 alert ID 执行，缺少调用者范围复核。

**影响**

学生可能看到全局心理统计，学院管理者可能读取或更新其他学院记录。心理数据属于高敏信息，任何范围错误都不应容忍。

**改进方案**

- 在 Repository 层建立默认拒绝的统一授权谓词。
- 学生查询强制 `user_id = caller`；学院角色强制 `owner_id = caller.owner_id`。
- 查询、统计、确认、关闭告警必须复用同一范围条件，直接构造 ID 也不能绕过。

**验收标准**

- 学生只能看到本人数据。
- 学院 A 任何角色都无法查询或更新学院 B 的记录。
- 跨范围资源统一返回 404/403，不暴露其是否存在。

### P0-06 普通学生可导出全部已发布知识正文

**证据**

- `server/internal/auth/capabilities.go:29,183-194` 给学生授予“导出自己的回答”能力。
- `server/pkg/app/app.go:631-632,650-652` 把 `/kb/export` 和 `/export` 也绑定到同一能力。
- `server/internal/handler/export_handler.go:83-123` 未把用户范围传入查询。
- `server/internal/repository/kb_repo.go:553-595` 返回全部 `published` 资源正文，不检查 ownerScope、ownerID 或 roleScope。

**影响**

普通学生可下载其他学院、其他角色甚至本应受限的完整知识正文，属于直接数据越权。

**改进方案**

- 拆分 `self.answer.export` 与 `school/college.kb.sync.export`。
- 知识同步导出仅允许专用服务账号或校级管理员。
- 所有导出 SQL 强制应用 owner/role scope，不能依赖 Handler 过滤。

**验收标准**

- 学生访问两个知识导出入口均返回 403。
- 学院账号不能导出其他学院或超出自身角色范围的数据。
- 导出行为写入可靠审计，并记录范围、数量、cursor 和包哈希。

### P0-07 知识 CRUD、审核和导入缺少数据范围授权

**证据**

- `server/pkg/app/app.go:601-629` 主要只做粗粒度 capability 检查。
- `server/internal/handler/kb_handler.go:60-265` 接受调用者提交的 scope。
- `server/internal/service/kb_service.go:86-180,206-328` 未在所有操作中接收和验证完整 `UserContext`。
- 外部导入可携带状态，存在绕过正常审核流的空间。

**影响**

拥有知识写权限的辅导员可能读取、修改、审核或发布其他学院乃至 school 范围资源。

**改进方案**

- 所有 KB Service 方法接收 `UserContext`。
- 实现统一 `CanReadScope/CanWriteScope/CanReviewScope`，请求范围必须是调用者范围的子集。
- 先读取资源再复核范围；导入默认进入 quarantine/pending，禁止直接 published。

**验收标准**

- 学院 A 辅导员对学院 B、school 资源的读写、审核、批量和导入操作均失败。
- 越权请求不产生数据库修改，也不能从错误信息推断资源内容。

### P0-08 同步包完整性和增量语义未实现

**证据**

- 文档要求 ZIP、NDJSON、SHA-256、HMAC、`packageId` 幂等和 4006～4008 错误码，但 `server/internal/model/dto.go:69-84` 没有对应字段。
- `WeiyuanSyncSecret` 仅存在于配置，未进入同步校验链路。
- `server/internal/handler/export_handler.go:109-123` 只返回普通 JSON。
- `server/internal/repository/kb_repo.go:553-573` 固定只导出 `published`，按 `updated_at DESC LIMIT 5000`。
- 响应 cursor 使用当前 RFC3339 时间，而数据库时间格式和排序语义不同。

**影响**

同步方身份和数据完整性无法验证；篡改包可进入导入链路。retired 永远不传播，超过 5000 条或同时间窗更新可能永久漏数。

**改进方案**

- 实现固定 ZIP 结构、规范化哈希输入、HMAC-SHA256 常量时间验签和 package 幂等表。
- 先验签、验哈希，再解析和落库；失败必须零落库。
- 建立单调递增 change sequence，所有创建、更新、审核、下架都写变更事件。
- 按 `sequence > cursor` 升序分页，cursor 取本页最后一条序号。

**验收标准**

- 缺文件、正文篡改、签名篡改分别返回 4006、4007、4008。
- retired 在下一次 delta 中出现。
- 同日超过 5000 条变更可分页完整同步，无重无漏。

### P0-09 外部系统集成接口存在 SSRF 与服务凭据外送风险

**证据**

- `server/internal/handler/integration_handler.go:23-97` 和 `server/internal/service/integration_service.go:59-97` 允许调用者影响代理路径/目标。
- 服务端请求附带统一集成 Token，缺少严格主机、解析后 IP、路径和重定向约束。

**影响**

攻击者可能探测内网、回环、云元数据地址，或诱导服务把集成 Token 发送到攻击者控制的主机。

**改进方案**

- 取消通用 URL 代理，为每个集成定义固定 base URL、固定方法和允许路径。
- DNS 解析后拒绝私网、回环、链路本地和云元数据地址。
- 禁止跨主机重定向；凭据只向备案目标发送。
- 把用户 ownerScope 映射为上游字段级范围，而不是复用高权限服务 Token。

**验收标准**

- 私网 IP、DNS rebinding、userinfo、跨域重定向和非白名单主机全部被拒绝。
- 抓包确认 Token 只发送给备案域名。

### P0-10 用户模型 API Key 明文存储并完整回显

**证据**

- `server/migrations/013_user_model_config.sql:4-27` 为密钥建立普通文本字段。
- `server/internal/repository/model_config_repo.go:20-74` 明文读写。
- `server/internal/handler/model_config_handler.go:22-82` 读取接口返回完整配置。

**影响**

数据库、备份、管理接口、浏览器状态或日志任一泄露，都可能直接导致第三方模型与语音账号被盗用。

**改进方案**

- 使用 KMS 或主密钥信封加密，至少采用 AES-256-GCM 并支持轮换。
- 写入后永不回显，只返回掩码和末四位。
- 密钥读取仅发生在服务端调用时，严禁进入普通日志和错误响应。

**验收标准**

- 数据库和备份中不存在明文可用密钥。
- GET 响应、日志、trace 和前端缓存均不含完整密钥。
- 完成密钥轮换、撤销和访问审计测试。

### P0-11 Flutter 切换账号会泄露上一用户聊天和收藏

**证据**

- `frontend/lib/main.dart:75-103` 将业务 Provider 放在应用根，生命周期覆盖整个进程。
- `frontend/lib/providers/auth_provider.dart:124-130` 退出只清 Token 和资料。
- `frontend/lib/providers/chat_provider.dart:12,117-123` 聊天消息只有显式新会话才清除。
- `frontend/lib/providers/bookmark_provider.dart:48,63-100` 所有账号共用 `bookmarks` 持久化键。

**影响**

用户 A 退出后，用户 B 在同一浏览器或设备登录，可看到 A 的对话和收藏，可能暴露学业、心理或个人咨询内容。

**改进方案**

- Provider 树以 `userId/sessionEpoch` 为 key，主体变化时整体销毁重建。
- 所有用户态 Provider 实现统一 `reset()`，退出、401 和账号切换时同步调用。
- 本地键使用 `bookmarks:<userId>`；敏感收藏应使用安全存储或服务端按用户保存。
- 取消旧会话仍在执行的网络请求。

**验收标准**

- A 发送敏感问答并收藏，退出后 B 不重启应用登录，消息、会话、收藏和管理缓存均为空。
- 离线模式下 B 也无法看到 A 的数据。

### P0-12 Android release 使用 debug 签名

**证据**

- `frontend/android/app/build.gradle.kts:46-50` 将 release 的 signingConfig 指向 debug。
- `scripts/build-all.ps1:101-150` 通过该 release 任务生成对外 APK。

**影响**

正式包使用开发证书，不满足可信升级和商店发布要求。debug keystore 被共享或泄露后，第三方可能构造同签名替换包。

**改进方案**

- 创建独立生产 keystore，从 CI Secret 或未入库的 `key.properties` 注入。
- release 缺少签名参数时直接失败，禁止回退 debug。
- 发布任务记录并校验生产证书指纹。

**验收标准**

执行：

```powershell
apksigner verify --verbose --print-certs frontend/build/app/outputs/flutter-apk/app-release.apk
```

证书 SHA-256 必须与备案生产证书一致，且不得为 Android Debug 证书。

### P0-13 v0.0.5 下载制品在干净部署中必然断链

**证据**

- `frontend/pubspec.yaml:19`、`frontend/lib/config/release_config.dart:6-10`、`frontend/web/downloads/release.json:2-7` 均声明 v0.0.5。
- `frontend/functions/downloads/[[file]].js:1-24` 仍硬编码并只允许 v0.0.4。
- `frontend/web/downloads/` 没有受版本控制的 v0.0.5 APK。
- `.github/workflows/deploy-frontend.yml:33-40` 在干净 checkout 中重建并部署，不会携带本机忽略的制品。
- `scripts/build-all.ps1:149-163` 在未找到 APK 时仍可能输出成功且不返回非零状态。

**影响**

首页二维码和下载按钮指向 v0.0.5，但 Pages Function 会返回 404；后续自动部署还可能覆盖本地手工补丁。

**改进方案**

- APK 存入 GitHub Release、Cloudflare R2 等不可变制品仓库。
- 版本、APK、manifest、下载入口由一个原子发布任务生成。
- 任何一步失败必须退出非零并回滚版本文件。
- 发布前校验 URL、长度、SHA-256、版本和 build number。

**验收标准**

- v0.0.5 APK 在干净 checkout 的发布流程中可复现。
- 下载返回 200，文件名、版本、build number 和 SHA-256 与 manifest 完全一致。

### P0-14 当前 release 配置无法通过 JWT 安全校验

审核环境中的本地 `.env` 未被 Git 跟踪，但其 release 模式 JWT 密钥长度低于代码要求的 32 字符。本报告不记录密钥值。

**影响**

按当前配置部署会在启动校验阶段失败；若为“先跑起来”而放宽校验，又会降低签名安全性。

**改进方案**

- 通过部署平台 Secret 注入高熵随机密钥，保留 release 的 fail-closed 校验。
- 建立密钥轮换、多版本验证窗口和会话失效方案。
- 确认本地密钥未通过聊天、日志、构建产物或历史提交泄露；如无法确认，应轮换。

**验收标准**

- release 环境可正常启动。
- 密钥满足强度要求，且不进入仓库、日志和构建产物。

---

## 4. P1：安全、隐私与后端可靠性

### 4.1 高敏数据与附件

| 问题 | 证据 | 改进与验收 |
|---|---|---|
| 学院管理员可读取全校审计日志 | `server/pkg/app/app.go:683-688`、`admin_handler.go:415-425`、`admin_service.go:216-230`、`audit_repo.go:20-97` | 区分校级/学院审计能力，SQL 强制 owner scope；学院 A 查询不得出现学院 B 事件 |
| 反馈截图匿名可读且公开缓存 | `server/pkg/app/app.go:473-474`、`feedback_handler.go:248-274` | 私有对象、所有权/处理权限检查、短期签名 URL、`private, no-store`；匿名和无关账号访问均失败 |
| 知识上传/导入无强制 DLP 闸门 | `server/internal/middleware/pii.go:13-50`、`upload_handler.go:62-86` | DLP 放到 KB Service；含测试手机号/身份证/名单的内容进入 quarantine，搜索不可见 |
| PII 中间件无界读取请求体 | `server/internal/middleware/pii.go:31-39` | 全局 body limit，上传流式处理；超大 body 返回 413，内存占用有界 |

### 4.2 会话、网络与配置

| 问题 | 影响 | 改进 |
|---|---|---|
| JWT 仅限制 HMAC 家族，issuer 缺失仍接受 | 算法和签发方约束不够严格 | 固定 HS256/迁移到非对称算法；强制 issuer、audience、jti、token version |
| CORS 默认值为 `*`，且测试已复现 evil.com 被回显 | 跨域边界与测试契约失真 | release 必须显式白名单；禁用 legacy `CORS()`；不允许 `*` 与 credentials 组合 |
| HTTP Server 缺少读头、读取、写入和空闲超时 | slowloris 和资源耗尽 | 使用显式 `http.Server`，设置超时、最大 Header 和优雅关闭 |
| 初始密码默认为学号 | 批量撞库风险 | 随机一次性密码、首次强制改密、登录异常检测；禁止公开可推导默认规则 |
| 本地内存限流依赖 ClientIP 且 map 无有效清理 | 可伪造代理头绕过，多实例不一致，内存持续增长 | 配置可信代理；用户/设备/接口复合限流；采用带 TTL/容量的共享存储 |
| Android release 允许明文 HTTP | 误配置时可降级为明文传输 | release 设置 `usesCleartextTraffic=false`，debug 仅对白名单开放 |

### 4.3 审计、迁移和并发一致性

- `server/internal/middleware/audit.go:37-56` 在请求结束后由无界 goroutine 继续读取 `*gin.Context`，可能发生对象池复用、数据竞争和审计主体错配。应在请求内复制不可变 DTO，投递有界队列，并在关闭时等待刷盘。验收要求 `go test -race` 无竞争、压力测试下 goroutine 有界。
- `server/pkg/app/app.go:316-341,356-374` 的迁移逐句执行后再记录版本，中途失败会留下半迁移状态。每个迁移应使用单事务、checksum 和 dirty 标志；故意让中间语句失败时，schema 与版本表必须保持迁移前状态。
- 学生活动报名、名额更新等操作存在“检查—插入—计数更新”分步执行。应增加数据库唯一约束和事务内条件更新；并发压测不得超卖或产生重复记录。
- 多个 Handler 仍直接向客户端返回 `err.Error()`，可能泄露 SQL、内部路径和供应商错误。应统一错误码、用户消息和服务端结构化日志。

### 4.4 Trace 与模型可靠性

- `server/internal/middleware/trace.go:16-31` 只把 TraceID 写入 Gin Context，没有写回 `c.Request.Context()`；下游 LLM 客户端从标准 context 读取时得到空值。应使用 `WithTraceID` 更新 Request Context，并在 API、DB、模型、审计和错误响应中核验同一 TraceID。
- 启动时只选择一个模型客户端，调用失败缺少规范所述的真实故障切换、受限重试、熔断和预算上限。应建立供应商路由器，只对安全幂等错误重试，并增加并发隔离、费用上限和去重标识。
- Web 401 处理没有 refresh token、single-flight 或会话代际判断。旧会话的迟到 401 可能在新用户登录后清掉新 Token。应比较失败请求 Token 与当前 Token，使用 session epoch 和 CancelToken。

---

## 5. P1：Context Engine、知识治理与契约

### 5.1 主问答没有形成统一 Context Engine Query

`server/internal/context_engine/doc.go` 目前主要是包说明，主问答在 `server/internal/service/chat_service.go:178-191` 直接走 FTS；流程 Agent 也没有优先查询 `process_steps`。

建议统一为：

1. 意图识别与问题规范化；
2. 结构化流程/日历/表格查询；
3. chunk 级 FTS/BM25 补充；
4. ownerScope、ownerID、roleScope、状态和时效统一过滤；
5. 证据去重、冲突检测、置信度校准；
6. AnswerCard 与 sources 拼装。

验收时，流程材料、入口、地点和时限必须来自结构化步骤；修改步骤数据后无需改 Prompt 即反映到回答。

### 5.2 零相关结果仍被强行当作来源

- `server/internal/repository/kb_repo.go:274-295` 在过滤为空后可能返回原结果。
- `server/internal/service/chat_service.go:876-926` 全部低于阈值时仍保留“最佳”一条。
- 当前测试日志已复现：问题“地球上有什么”对“国家奖学金评选办法”得分为 0，仍被保留。
- AnswerCard 只要有结果就可能给出较高默认置信度。

这违反“未命中必须明确 fallback”的硬约束。整改后，无关问题必须返回 `fallback=true`、`sources=[]`，不得包装成“根据知识库整理”。

### 5.3 整篇资源索引，snippet 不是命中片段

- `server/migrations/001_init.sql:63-72` 对整篇 content 建 FTS。
- `server/internal/service/chat_service.go:273-283` 给模型的只是正文前 1500 字。
- sources 中的 snippet 实际多来自 summary。

应增加 `kb_chunks` 和 chunk 级 FTS，保留章节、条款号、页码或锚点。长文末尾命中时，模型上下文和引用卡都必须包含真实命中段落及 chunk ID。

### 5.4 状态、时效、版本和范围规则不统一

- 搜索与知识大厅只检查 `published`，没有统一过滤 `effective_at/expired_at`，过期资源仍可能被回答、浏览或推荐。
- 流程增强接口按 ID 读取父资源和步骤，缺少发布状态、时效和用户范围校验。
- 推荐冷启动路径没有完整复用 roleScope，并可能丢弃 ownerID。
- `compareVersion` 只适合点分数字版本，不兼容文档示例 `20260430-v2`。
- 当前表以 resource_id 唯一并更新原行，历史版本被覆盖，旧引用无法稳定回溯。

应建立统一“可见资源查询”，所有搜索、推荐、流程、浏览和导出复用；版本使用不可变 revision 表和 current pointer。

### 5.5 运行 Schema 与文档契约分叉

- 文档资源 Schema 使用 camelCase 和数组，运行实体使用 snake_case，`role_scope/tags` 为字符串。
- AnswerCard 文档与运行字段在 title/answer/conclusion、steps 结构、action link/type/payload 等方面不一致。
- 文档路由仍出现 `/chat/ask`，实际主要为 `/chat`。
- 后端发送 `effective_at`，前端部分代码只解析 `effective_date/date`。

应确定唯一 v1 Wire DTO，生成 OpenAPI，并由契约生成 Go/Dart 类型。使用规范原样示例执行导出→导入往返，字段、数组、版本和状态必须完全一致。

### 5.6 引用与导出尚未形成审计闭环

- 聊天页没有完整传入来源详情动作，部分页面和导出只保留标题。
- 前端可本地生成 PDF/PNG/Markdown；后端导出接受客户端提交的完整 AnswerCard，而不是按 answerId 读取可信记录。
- `server/internal/service/export_service.go:213-227` 会把 PDF 中非 ASCII 字符替换为问号。
- 无可信知识时，部分学生流程仍输出硬编码日期、材料和 URL，且没有来源。

正式导出应由后端按 answerId 读取可信快照，记录操作者、角色、traceId、水印和 sources；中文 PDF 必须可复制、可搜索。无可信来源时不得输出确定性日期、材料或办理入口。

---

## 6. P1/P2：Flutter、Web、小程序与交付链路

### 6.1 Token 存储和 Web 安全边界

- `frontend/lib/utils/storage.dart:24-29` 使用 SharedPreferences 保存 JWT。Web 端等价于可被同源 JavaScript 读取的存储，Android 端也不是 Keystore-backed 安全存储。
- `frontend/web/_headers` 没有严格 CSP。
- `frontend/web/index.html:10-20` 全局吞掉未处理 Promise rejection 和部分 Error，使真实故障从控制台和监控消失。

建议 Android/iOS 使用 `flutter_secure_storage`；Web 优先采用同域 BFF 与 `HttpOnly; Secure; SameSite` Cookie。异常只按已知指纹过滤，其余上报脱敏堆栈、traceId、版本和路由。

### 6.2 依赖和构建不可复现

- `.gitignore:3` 全局忽略 `*.lock`，导致 Flutter 应用的 `pubspec.lock` 未纳入版本控制。
- `pubspec.yaml` 大量使用 caret 范围；本次最小测试解析依赖时提示 16 个依赖发生变化。
- `.github/workflows/deploy-frontend.yml` 使用 Flutter 3.38.5，而本地审核运行时使用 3.35.1；没有统一锁定工具链和依赖树。

应提交 Flutter 应用 lockfile，CI 使用 `--enforce-lockfile`，Web/APK 使用同一 Flutter SDK 和依赖树。依赖升级必须单独提交并附回归结果。

### 6.3 CI 未保护前端自动部署

- `.github/workflows/ci.yml` 只有 Go 和文档任务；Go 版本配置为 1.24，而 `go.mod` 要求 1.25.0。
- `.github/workflows/deploy-frontend.yml:30-47` 直接执行 pub get、Web build 和部署，不依赖 Flutter analyze/test 成功。
- 当前唯一 Flutter 测试 `frontend/test/widget_test.dart` 只验证 `1 + 1 == 2`。

应增加固定 SDK 的 `flutter pub get --enforce-lockfile`、`flutter analyze`、`flutter test --coverage`、Web release build、Android 签名校验和下载 manifest 一致性测试；部署任务必须依赖这些门禁。

### 6.4 两个小程序工程互相冲突

- 协作索引指定正式工程为 `frontend/miniprogram/`。
- 根目录 `miniprogram/` 仍使用占位 AppID、localhost API、旧登录契约，并引用不存在的 tab 图标。
- 现有 48 个 Node 断言只测工具函数，没有验证正式 WebView 壳、AppID、资源或生产登录。

应只保留一个正式入口；另一工程删除或明确移入 archive。CI 至少校验 AppID、生产域名、资源完整性和微信开发者工具构建。

### 6.5 体验与可访问性治理

- `frontend/lib/config/router.dart:459-463` 的 900dp 最大宽度约束位于 tight constraints 下，宽屏可能仍被拉满。
- `frontend/lib/utils/storage.dart:84-85` 首次启动默认 light，不符合“默认跟随系统”。
- 自定义 `GestureDetector + Container` 大量使用，但显式 Semantics 极少，键盘、焦点、Enter/Space 激活和读屏角色不完整。
- 存在大量硬编码颜色、字体尺寸和千行级页面。

建议建立语义化 theme token，补 light/dark、高对比度、字体缩放、键盘和 semantics 测试；主内容宽度在 1440/1920/2560 截图中均不超过 900dp。

---

## 7. 架构与代码质量问题

### 7.1 分层和文件粒度

项目规范要求单文件不超过 400 行，但当前存在：

- `server/internal/handler/education_handler.go`：约 1600 行以上；
- `server/internal/handler/study_plan_handler.go`：约 1400 行；
- `frontend/lib/models/models.dart`：约 1500 行；
- `frontend/lib/pages/admin/my_submissions_page.dart`、`chat_page.dart`、`admin_users_page.dart`：均超过 1000 行。

`study_plan_handler.go` 还直接持有数据库和 LLM，并使用 `context.Background()` 调模型；部分 Handler 直接依赖 Repository。建议按业务域拆分 DTO、Controller/Service、Repository、页面 section、dialog 和 form，并建立文件长度和依赖方向 lint。

### 7.2 学习计划和数据库事务

学习计划生成存在多次独立插入，任务写入错误可能被忽略；应在单事务内创建计划及任务，任一步失败全部回滚。模型原始响应不应直接写普通日志，调用必须继承请求 context。

### 7.3 回答缓存设计不稳定

`chat_service.go` 当前缓存读取发生在创建 session 前，但写入时 sessionID 已非空，guard 会使正常路径基本无法写入；若仅删除 guard，缓存键又只有问题文本，会产生跨学院、跨角色和跨版本复用风险。

建议只缓存“经过权限过滤的检索证据”，键至少包含：

`question + ownerScope + ownerID + role + retrievalPolicyVersion + resourceWatermark`

并为权限、版本和过期时间建立失效测试。

### 7.4 文档与实现持续漂移

- 总纲仍写“9 页面 + 8 Provider”，实际规模远大于此。
- 总纲写 go_router 17.x，`pubspec.yaml` 当前约束为 14.x。
- 总纲宣称 Flutter analyze 零错误零警告，实测有 397 项诊断。
- 个人页和服务根路由仍展示 v0.0.1，发布配置已为 v0.0.5，下载函数仍为 v0.0.4。
- 前端重新部署文档中的代理域名与 Pages Function 实际上游不一致。
- 总纲将 Context Engine、NDJSON 幂等和 cursor 标为“已完成”，但关键验收能力尚未实现。

建议所有“已完成”状态绑定自动化验收证据，并记录最后验证提交 SHA、命令、日期和结果；版本、域名、制品名应从机器可读配置生成。

---

## 8. 实际验证结果

### 8.1 Go

| 命令 | 结果 |
|---|---|
| `go test ./server/internal/config` | 通过，约 4.7 秒 |
| `go test ./server/internal/repository` | 通过，约 83 秒，耗时偏高 |
| `go test ./server/internal/handler ./server/internal/service ./server/internal/middleware ./server/pkg/app` | 失败：Service 通过；Handler 4 个失败；Middleware 1 个失败；pkg/app 无测试 |
| `go test ./...` | 180 秒内未完成，仅部分包完成；全量门禁时长不可控 |
| `go vet ./...` | 通过 |
| `gofmt -l server` | 返回 9 个未格式化文件 |

当前失败用例：

1. `TestChatHandler_Ask_Success`：回答 conclusion 为空，session_id 缺失。
2. `TestChatHandler_Ask_LLMError`：LLM 错误时未返回预期兜底回答。
3. `TestKBHandler_BrowseKnowledge_WithTypeFilter`：类型过滤返回空结构。
4. `TestKBHandler_BrowseKnowledge_PaginationBoundary`：分页边界返回空结构。
5. `TestCORS_DisallowedOrigin`：不允许的 `https://evil.com` 仍获得 Allow-Origin。

`gofmt -l server` 当前返回：

- `server/internal/auth/capabilities.go`
- `server/internal/handler/education_handler.go`
- `server/internal/handler/study_plan_handler.go`
- `server/internal/model/dto.go`
- `server/internal/repository/user_repo.go`
- `server/internal/service/kb_service_extra_test.go`
- `server/internal/service/kb_service_test.go`
- `server/internal/service/token_stats_service.go`
- `server/internal/util/response.go`

### 8.2 Flutter

使用绑定 Flutter SDK 的分析入口执行后得到：

- 397 项诊断；
- 9 项 warning；
- 388 项 info；
- 其中至少 8 项 `use_build_context_synchronously`。

命令显式关闭 fatal 后可以退出 0，但这不等于“零诊断”。原生 `flutter analyze` 应作为最终门禁。

Flutter 测试目前仅 1/1 通过，唯一用例是 `1 + 1 == 2`，不能覆盖认证切换、Provider 清理、401、AnswerCard、下载、路由守卫或关键页面。

### 8.3 小程序

根目录小程序的 48/48 Node 断言通过，但只覆盖工具函数，未验证正式 WebView 工程、资源、AppID、生产域名或真实登录契约，不能作为可交付证明。

### 8.4 质量门禁结论

当前不能声明“测试全部通过”或“Flutter 零警告”。应先修复上述回归并让以下原生命令全部稳定退出 0：

```powershell
go test ./...
go test -race ./...
go vet ./...
gofmt -l server

flutter pub get --enforce-lockfile
flutter analyze
flutter test --coverage
flutter build web --release
flutter build apk --release
```

其中 `gofmt -l server` 必须零输出。

---

## 9. 分阶段修复路线图

### 9.1 0～24 小时：止血和冻结

1. 暂停正式发布和扩大试点；保留最小内部测试环境。
2. 临时关闭或收紧游客注册、二维码登录、知识全量导出、通用集成代理、模型密钥读取和匿名截图路由。
3. 禁止 JWT JIT upsert 回写角色/状态；停用账号必须由数据库实时判定。
4. 修复验证码返回与任意六位通过；pending 游客不得使用业务能力。
5. 将知识导出和 CRUD 限制到校级管理员/专用服务账号，先做显式白名单止血。
6. 修复 Android release 签名、v0.0.5 制品和下载函数一致性。
7. 注入合规 JWT Secret，并轮换可能已暴露的模型/语音/集成凭据。
8. 将反馈截图改为认证后私有读取并关闭公开缓存。
9. 修复 5 个当前失败测试，禁止带红门禁部署。

### 9.2 1 周：建立可信安全边界

1. 完成数据库权威角色、token version、会话吊销和可信 SSO JIT 流程。
2. 持久化 consent，并为聊天、情感、语音和导出加服务端强制授权。
3. 在模型调用前建立统一 DLP/脱敏管线，历史消息和情感分析不得绕过。
4. 为知识、心理、审计、附件建立 Repository 层统一数据范围策略和越权测试。
5. 删除通用外部代理，完成 SSRF 白名单、防重定向和解析后 IP 校验。
6. 模型密钥改为加密存储、只写不回显，并完成轮换。
7. 实现同步包 ZIP、哈希、HMAC、package 幂等与失败零落库。
8. Flutter 完成账号状态 reset、按用户存储、旧请求取消和安全 Token 存储。
9. 提交 `pubspec.lock`，CI 增加 Flutter analyze/test/build 和签名/制品校验。

### 9.3 1 个月：收敛 Context Engine 与工程质量

1. 实现统一 Context Engine Query 和结构化流程优先。
2. 引入 `kb_chunks`、真实 snippet、时效过滤、版本 revision 和变更 sequence。
3. 统一资源 Wire DTO、AnswerCard 和 OpenAPI，生成 Go/Dart 契约类型。
4. 正式导出改为 answerId 服务端可信快照，支持中文 PDF 和完整审计。
5. 审计日志改为有界队列；迁移、学习计划和报名流程全部事务化。
6. 拆分千行级 Handler、页面和模型文件，建立依赖方向与文件长度门禁。
7. 建立错误率、延迟、模型费用、fallback 率、零来源回答率、越权拒绝和同步失败告警。
8. 文档状态由 CI 证据生成，消除版本、域名和完成度漂移。

---

## 10. 上线前统一验收矩阵

### 10.1 认证与会话

- [ ] 停用、删除、降权、改密后旧 Token 立即失效，且不能重建或修改账号。
- [ ] 错误、过期、重放验证码全部失败；响应和日志无验证码。
- [ ] pending 游客不能调用业务 API。
- [ ] QR session 泄露后，没有创建端 verifier 仍无法换取 Token。
- [ ] issuer、audience、算法、jti 和 token version 均严格校验。

### 10.2 隐私与敏感数据

- [ ] 未同意或已撤回用户不能调用聊天、情感、语音和敏感导出。
- [ ] 测试 PII 不出现在模型请求、日志、trace 和非必要存储中。
- [ ] 学生只能访问本人心理数据，学院角色只能访问本学院。
- [ ] 匿名和无关账号不能读取反馈截图。
- [ ] 数据库和 GET 响应中不存在完整模型 API Key。

### 10.3 RBAC 与知识治理

- [ ] 学生访问知识同步导出返回 403。
- [ ] 学院 A 不能读取、修改、审核、导入或导出学院 B/school 资源。
- [ ] draft、pending、retired、未生效、已过期和角色受限资源不会进入搜索、推荐、流程或 sources。
- [ ] 上传含测试 PII 的文件进入 quarantine，搜索不可见。

### 10.4 同步与契约

- [ ] 缺文件、篡改 NDJSON、篡改签名分别返回 4006、4007、4008，失败零落库。
- [ ] retired 出现在下一次 delta。
- [ ] 同日超过 5000 条变更分页无重无漏，cursor 单调。
- [ ] `20260430-v2` 正确高于 v1，历史版本可回滚且旧引用可定位。
- [ ] 规范示例导出→导入往返完全一致。

### 10.5 问答与来源

- [ ] 无关问题返回 `fallback=true`、`sources=[]`，不得引用零相关资源。
- [ ] 长文末尾条款返回真实命中段落、chunk ID、版本和生效时间。
- [ ] 流程答案来自结构化步骤，修改步骤后无需改 Prompt。
- [ ] 引用卡可打开受权限保护的原文详情。
- [ ] 正式导出按 answerId 生成，中文可复制搜索，内容不可由客户端篡改。

### 10.6 Flutter 与发布

- [ ] A 退出后 B 登录，所有聊天、会话、收藏和管理缓存为空。
- [ ] Web JavaScript 不能读取可复用 JWT；Android Token 不在普通 SharedPreferences。
- [ ] release APK 使用备案生产证书且禁止明文 HTTP。
- [ ] APK、release.json、下载函数、页面版本、SHA-256 完全一致，URL 返回 200。
- [ ] 干净 checkout 可复现 Web/APK 构建，使用固定 Flutter SDK 和 lockfile。

### 10.7 自动化门禁

- [ ] `go test ./...`、`go test -race ./...`、`go vet ./...` 全部通过。
- [ ] `gofmt -l server` 零输出。
- [ ] `flutter analyze` 零诊断或达到团队明确批准的零新增基线。
- [ ] `flutter test --coverage` 覆盖认证切换、401、Provider、AnswerCard、下载和关键路由。
- [ ] 前端部署只在全部门禁通过后执行。

---

## 11. 已有基础与建议保留的方向

本次结论虽为 No-Go，但项目并非需要推倒重来。以下基础值得保留并继续收敛：

- 文档已经明确结构化优先、FTS/BM25、sources、RBAC、知识治理和同步完整性的正确方向。
- capability 命名和角色层级为权限治理提供了可扩展骨架。
- 已有 AnswerCard、Source、process_steps、FTS、导入导出和模型适配器，可作为统一 Context Engine 的基础组件。
- 当前 `go vet ./...` 通过，配置中也已加入 release JWT Secret 的 fail-closed 校验。
- 已新增全局、登录和聊天限流雏形，虽需补可信代理、TTL 和多实例能力，但方向正确。
- Repository 和 Service 定向测试当前可以通过，说明核心层仍具备继续收敛的可测试基础。

建议后续坚持“先契约与验收、再编码”的工作流：每个 P0/P1 都先补失败测试，再实现修复；只有自动验收通过后，文档才能标记为“已完成”。

---

## 12. 最终结论

蔚小芯当前已经达到“功能面较完整的内部开发版本”，但尚未达到“可承载真实学生数据的生产系统”标准。核心阻断不在页面数量或功能丰富度，而在以下四条基础边界尚不可信：

1. 身份与权限尚未以数据库和会话状态作为最终权威；
2. 敏感数据范围和隐私授权没有在服务端形成强制闭环；
3. Context Engine、sources 和同步包契约与文档承诺不一致；
4. 正式制品、测试和部署门禁不能证明发布内容可信。

**建议发布决策：No-Go。**

只有在全部 P0 关闭、关键 P1 完成、测试矩阵通过，并由独立账号/学院进行越权复测后，才建议恢复受控试点；扩大到全校前，还应完成真实模型供应商数据处理协议、密钥轮换、备份恢复和安全演练。
