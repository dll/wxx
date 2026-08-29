# 蔚小芯 GPT-5.6 SOL 审核报告 v4

> 审核日期：2026-08-29
> 审核基线：`8b86aee42e5e827a643bc907f4704da25a4d621c`（`main`，与 `origin/main` 一致）
> 对照基线：`docs/蔚小芯GPT56SOL审核报告v3.md`，v3 基线为 `3dbbc89`
> 审核范围：v3 基线至当前 HEAD 的提交历史、后端/前端质量门禁、认证授权、知识包协议、Context Engine、隐私存储、CI/CD、迁移与部署文档
> 审核性质：只读审核；未修改业务源码、迁移文件或 CI 配置

---

## 1. 最终结论

### 1.1 结论：条件 No-Go

**不建议直接用于正式生产或扩大真实学生数据试点；可继续受控内部演示和小范围开发验证。**

本次相较 v3 有明显进展，不能再沿用 v3“P0-01～P0-06 全部仍存在”的旧结论：

- **P0-01 pending 游客业务访问**：已有 `b643bd7` 等提交，将非 `active` 账号拒绝，并收窄游客能力；当前代码证据支持已修复。
- **P0-02 JIT 用户默认 active/信任 JWT 角色**：当前 JIT 强制 `pending + guest + consented=0`，数据库权威覆盖 JWT 角色；当前代码证据支持已修复。
- **P0-03 外部系统代理 SSRF**：当前已有主机白名单、路径归一化、DialContext 私网地址拦截和跨主机重定向阻断；当前代码证据支持已修复。
- **P0-04 加密明文降级**：当前缺少 `WXX_ENCRYPTION_KEY` 时返回错误，且启动入口执行校验；当前代码证据支持已修复。
- **P0-05 情感告警越权写入**：更新和读回均传入 `UserContext` 并执行范围谓词校验；当前代码证据支持已修复。
- **P0-06 知识库客户端 scope/status 信任**：创建范围由服务端决定，更新忽略客户端状态，审核/下架复核资源范围；当前代码证据支持已修复。
- **N3-01 gofmt**：当前 `gofmt -l server/` 为空，已修复。
- **N3-03 部署无质量门禁**：当前三个发布 job 均有 `needs: [quality-gate]`，已显著整改。

但以下阻断项仍未收口：

1. **P0-07 知识包协议安全契约仍未完成**：生产环境 HMAC 双向可选；删除 `signature` 即可绕过验签；没有 package receipt/幂等账本；`retired` 仍未进入增量导出。
2. **P1-08 前端 JWT 等敏感状态仍使用 SharedPreferences 明文存储**，没有 `flutter_secure_storage` 或 Web HttpOnly Cookie/BFF 方案。
3. **P1-09 Context Engine 仍是未接线的平行实现**，生产问答链路继续在 `chat_service`/agent 中自行检索，存在双实现漂移。
4. **P1-10 consent ledger 仍未实现**：只有单一 `users.consented` 布尔字段，没有版本、用途、供应商、同意/撤回时间和历史记录。
5. **构建供应链仍使用硬编码第三方 Flutter 资源镜像**，且 CI workflow 仍保留地图 AK fallback 明文。
6. **当前工作树不干净，文档和部署口径存在冲突**，不满足可追溯发布基线要求。

因此建议结论为：**代码整改大幅推进，后端核心 P0-01～06 已有较强关闭证据；但协议安全、隐私治理、检索架构统一、供应链与发布治理仍未达生产验收门槛。**

### 1.2 评分

| 维度 | v3 | v4 | 评价 |
|---|---:|---:|---|
| 需求与功能覆盖 | 6.8 | 8.0 | 多角色教育、VOPC、反馈闭环、移动端能力显著增加 |
| 架构与分层 | 6.3 | 6.5 | 模块增多，但重复检索与超大文件仍存在 |
| 后端正确性 | 7.0 | 7.8 | 全量后端测试通过，关键安全修复有代码证据；但耗时过长 |
| 前端质量 | 6.2 | 6.4 | 测试通过，但 279 条 info、分析耗时过长 |
| 安全与合规 | 5.2 | 6.6 | P0-01～06 有实质整改；P0-07、P1-08、P1-10 仍阻断生产 |
| Context Engine/RAG | 5.8 | 5.9 | 参考实现完整，但仍未接入生产装配层 |
| 依赖与可复现构建 | 5.6 | 5.8 | go mod 可验证；Flutter 第三方镜像和明文 fallback 未收口 |
| CI/CD 与发布 | 5.5 | 6.6 | 已有质量门禁和迁移/健康检查，但门禁不等于完整发布验收 |
| 运维与可观测性 | 6.3 | 6.5 | MySQL/Redis、备份和部署流程增多，但正式备份状态仍需实机确认 |
| **综合** | **6.0** | **6.8** | 能力显著增强，生产安全边界仍未闭环 |

---

## 2. 当前版本实测证据

审核命令均在工作区本地执行；不把子任务超时或历史文档中的“全绿”当作当前实测。

| 命令 | 结果 | 备注 |
|---|---|---|
| `go test -tags fts5 ./server/...` | **通过** | 所有包通过；约 830 秒（约 13 分 50 秒） |
| `go vet -tags fts5 ./server/...` | **通过** | exit 0 |
| `gofmt -l server/` | **通过** | 输出为空，v3 的 17 文件问题已消失 |
| `go mod verify` | **通过** | `all modules verified` |
| `flutter analyze --no-pub` | **完成，279 issues** | 均为 info；无 error/warning；约 783 秒 |
| `flutter test --no-pub` | **15/15 通过** | exit 0；覆盖 VOPC、年级主题、党建协同和空数据态等 |
| `git diff --check` | **未通过** | 当前未提交 `audit-report.md` 尾部存在空白/多余 EOF 空行 |
| 当前工作树 | **不干净** | 4 个已修改文件、11 个未跟踪项（详见 §8） |

### 2.1 质量门禁评价

后端的“全量测试通过”是本版本的重要正面证据，但耗时约 14 分钟，主要集中在 `service`、`handler`、`repository`、`pkg/app` 等包。当前 `.github/workflows/ci.yml` 的 Go 测试使用 `go test -v -race -cover ./...`，而发布 workflow 的 `quality-gate` 只运行：

```text
go test -tags fts5 -count=1 -timeout=300s
  ./internal/auth ./internal/config ./internal/middleware
```

也就是说，发布前门禁没有执行当前本地耗时最长的 service、handler、repository、pkg/app 全量测试；它只能证明“快包”通过，不能替代全量回归。建议将全量后端测试拆为可并行 job，或建立经过缓存/模板库优化的完整门禁。

Flutter 方面，15 条测试全部通过，但 `flutter analyze` 产生 **279 条 info**，包含大量 `prefer_const_constructors`、`curly_braces_in_flow_control_structures`、`use_build_context_synchronously` 和弃用 API 提示。info 不阻断当前 CI，但已明显超过“可接受噪声”范围；其中异步 BuildContext 使用和 API 弃用不宜长期忽略。

---

## 3. v3 遗留问题逐项复核

### 3.1 P0-01 pending 游客获得业务 JWT —— **已修复，需保留回归测试**

当前 `AuthService.GuestRegister` 仍会签发 token，但用户状态为 `pending`、角色为 `guest`。关键安全边界已下沉到 `UserRepo.UpsertFromContext`：

- 数据库已有用户时，`existing.Status != "active"` 一律拒绝；pending/disabled/rejected 均不能通过 JWT 中间件进入业务链路。
- 游客能力已收窄，不应再拥有业务聊天、知识和流程能力。

因此“签发 token”本身不再等于“放行”。仍建议保留“pending 游客访问 chat/kb/process 返回 403”的 handler 回归测试，防止未来新增路由只挂 JWT、不挂状态校验。

### 3.2 P0-02 JIT 新用户 active/consented/角色自述 —— **已修复**

`UpsertFromContext` 当前对未知用户插入：

- `status = pending`
- `role = guest`
- `consented = 0`

对于已存在用户，数据库角色、归属、状态、同意状态覆盖 JWT 内容，不再把旧 JWT 的角色和 owner scope 写回数据库。该修复明显降低了伪造/过期令牌复活权限的风险。

残留注意：JIT 仍使用 JWT 中的 username/displayName/owner 字段作为新记录的部分输入，虽然权限角色已固定为 guest，仍应在生产关闭不必要的 JIT fallback，或增加来源可信度和用户名格式校验。

### 3.3 P0-03 外部系统代理 SSRF —— **已修复，仍需真实 DNS/HTTP 回归**

当前 `IntegrationService` 已包含：

- baseURL 主机白名单；
- `url.JoinPath` 和 scheme/`//` 路径走私拦截；
- `DialContext` 对私网、环回、链路本地、组播和未指定地址拦截；
- 跨主机重定向拒绝；
- 10 秒 HTTP 超时、连接 5 秒超时和响应体 1 MiB 限制。

这是对 v3 的实质性修复。风险仍在于：

- 代码允许“显式配置在白名单内的内网地址”作为例外，生产配置必须经过人工审核；
- 当前仅能从本地静态代码确认，尚未执行真实 DNS rebinding、IPv6、代理环境变量和重定向集成测试。

### 3.4 P0-04 密钥回显/明文降级 —— **已修复，需确认既有数据迁移**

`repository/crypto.go` 当前在缺少 `WXX_ENCRYPTION_KEY` 时返回错误，不再把敏感字段原样写入数据库；`ValidateEncryptionKey` 供生产启动入口 fail-fast。配置返回层已有掩码视图。

必须补充生产运维动作：扫描并迁移历史上可能由旧明文 fallback 写入的密钥，轮换既有密钥，并验证启动失败不会导致数据库部分迁移或错误日志泄密。

### 3.5 P0-05 情感告警越权更新/读取 —— **已修复**

`EmotionService.UpdateAlertStatus` 当前接收 `*model.UserContext`，调用 repository 更新时传入 `OwnerScope/OwnerID/Role`，更新后同样按范围读回。v3 中“读路径有 scope、写路径无 scope”的断裂已补齐。

仍需进行完整矩阵测试：学院 A 用户更新学院 B 告警、普通学生读取辅导员视角字段、资源不存在与无权限场景的错误是否泄露存在性。

### 3.6 P0-06 知识库客户端 scope/status 信任 —— **已修复，仍有接口审计要求**

当前 `KBService` 已体现以下边界：

- `Create` 用 `resolveCreateScope(userCtx)`，忽略客户端 owner scope/owner ID；新资源固定为 `draft`。
- `Update` 使用 `requireWritable` 和 `canWriteScope`；忽略请求体 status，禁止 PUT 直接发布。
- `ApproveResource`、`RejectResource`、`RetireResource` 均先做可写范围复核。
- 内容在写库前清洗，降低 FTS/来源/导出污染风险。

当前证据支持 P0-06 已关闭。下一步应继续审计 `ImportResources`、批量导入、分片导入和所有新 VOPC 写接口，确保它们同样接收 UserContext，而不是仅依赖路由 capability。

### 3.7 P0-07 知识包协议安全契约 —— **仍为 P0，生产阻断**

当前 `KnowledgePackageService` 仍存在三处与 `specs/export-package.md` 不一致的关键问题：

#### (a) HMAC 双向可选，可删字段绕过

导出只有在 `s.secret != ""` 时设置 `SignAlg` 和 `Signature`；导入只有在 `s.secret != "" && manifest.Signature != ""` 时验签。当前配置默认允许空 HMAC secret，因此：

- 生产环境配置遗漏时，导出包没有签名；
- 攻击者删除 `manifest.signature` 后，导入端不会拒绝；
- `resourcesSha256` 只是包内自带 hash，不能在无认证密钥时证明来源。

**修复门槛**：生产模式缺 HMAC secret 必须启动失败；导入必须要求 `SignAlg=hmac-sha256`、签名非空且验证通过；签名缺失、算法不匹配、篡改和重放均须自动化测试。

#### (b) packageId 没有 receipt/幂等账本

`ImportPackage` 校验后直接调用 `KBService.ImportResources`，只在响应中回显 `manifest.PackageID`，没有持久化“已接收 packageId、签名、游标、接收时间、结果”的唯一记录。资源级 upsert 可以缓解简单重复行，但不能提供协议要求的包级重放审计与一致性语义。

**修复门槛**：新增 package receipt 表，`package_id` 唯一；记录 producer、signature digest、cursor、resource count、状态、traceId、操作者和失败原因；同 packageId 重放必须返回既有结果或明确拒绝，不能重新执行写入。

#### (c) retired 没有进入增量导出

当前 repository 的增量导出仍以 `status='published'` 为条件，而协议明确要求 `retired` 必须传播。结果是源端下架后，接收端可能长期保留旧 published 资源并继续检索。

**修复门槛**：建立变更序列/事件表，增量导出至少覆盖 published→retired 等状态变更；消费端接收 retired 后从检索集合移除但保留审计历史。

### 3.8 P1-08 前端敏感状态明文存储 —— **仍存在，生产前必须修复**

`frontend/lib/utils/storage.dart` 仍使用 `shared_preferences` 保存：

- JWT token；
- 角色、owner scope/ID；
- capability 清单；
- consent 状态。

`pubspec.yaml` 没有 `flutter_secure_storage`。移动端 SharedPreferences 不适合作为长期 bearer token 安全存储；Web 端也没有 HttpOnly Cookie/BFF 会话方案。

**修复门槛**：移动端使用系统安全存储；Web 端改为同源 BFF + HttpOnly/Secure/SameSite Cookie，或至少完成 session epoch、XSS 风险控制和 token 轮换；能力与角色只作为缓存展示，服务端每次请求仍必须权威校验。

### 3.9 P1-09 Context Engine 生产链路分叉 —— **仍存在，架构 P1**

`server/internal/context_engine/engine.go` 的包注释仍明确写“实验性/未接线”。当前生产问答仍由 `chat_service`、agent 和 repository 直接组合结构化/FTS 检索；Context Engine 有独立的 `Query`、意图分类、结果加权、历史选择和上下文拼装实现，但没有接入 `pkg/app` 装配层。

这不是“没有检索能力”：当前生产链路确实能检索并返回 sources；问题是项目规范要求 Context Engine 作为唯一主路线，而现在存在两套实现，导致：

- 触发策略和 Top-K 可能漂移；
- 过期过滤、来源加权和历史选择可能不一致；
- 修复一处逻辑时另一处继续产生不同答案；
- 测试无法证明线上真实链路覆盖 Context Engine。

**修复门槛**：通过 `KBSearcher/HistoryProvider` 适配器把 Context Engine 接入生产链路，删除 chat_service 内联检索副本；增加 sources 非空、低相关兜底、过期过滤、跨范围隔离和线上装配回归测试。

### 3.10 P1-10 consent ledger —— **仍存在，生产前必须修复**

当前数据库仍主要依赖 `users.consented` 单布尔字段，`RequireConsent` 只检查布尔值。缺失：

- 协议/隐私政策版本；
- 处理用途；
- 第三方供应商；
- 同意时间、撤回时间、来源和操作审计；
- 版本升级后的重新同意；
- 针对情感、健康、画像等敏感用途的独立授权。

此外，当前 `RequireConsent` 只保护少数聊天路由，不能证明所有健康、情感、画像、导入和其他敏感写入均在同一授权策略下执行。

**修复门槛**：新增 consent ledger 迁移与模型，记录 subject、policy version、purpose、vendor、granted/revokedAt、source、traceId；敏感写入统一挂载用途级授权；撤回后立即阻断新处理并提供数据处置策略。

---

## 4. 新增与升级问题

### N4-01 前端分析噪声和反馈周期过长 —— P1

- `flutter analyze --no-pub` 实测 279 条 info，耗时约 783 秒。
- 主要包括异步间隙使用 BuildContext、弃用 API、未使用成员、缺少 const 和控制流花括号问题。
- 后端全量测试约 830 秒；前后端连续全量检查反馈周期超过 25 分钟。

建议按 error-risk 分层：先清理 `use_build_context_synchronously` 和弃用 API，再设置 info budget；后端按包并行、迁移模板复用，目标是完整门禁控制在 5 分钟级别。

### N4-02 Flutter 资源源仍硬编码第三方镜像 —— P1（供应链）

`scripts/build-all.ps1` 仍设置：

```powershell
$env:FLUTTER_STORAGE_BASE_URL = "https://flutter-ohos.obs.cn-south-1.myhuaweicloud.com"
```

本地 Flutter 命令输出也明确提示资源将从该地址下载。该来源不是 Flutter 官方源；当前未见 artifact SHA256 固定校验、来源审计记录或 CI 签名验证。

**修复门槛**：使用官方源或经审计的企业镜像；如必须使用该镜像，固定 Flutter SDK/engine artifact digest，并在构建日志中校验来源和哈希。

### N4-03 CI 仍把地图 AK 作为明文 fallback —— P1/P2

`deploy-frontend.yml` 的 Web 和 APK 构建仍在 secrets 缺失时使用 workflow 中的默认地图 AK。地图 AK 通常不是高机密，但这会：

- 掩盖 secrets 未配置这一发布错误；
- 让正式构建不可证明使用了受控凭据；
- 增加配额盗用和来源滥用风险。

建议生产 workflow 对 secrets 缺失直接失败；公开前端 key 应限制域名、包名、IP、配额，并把“公开配置”与“secret”命名分开。

### N4-04 CI 质量门禁仍不完整 —— P1

当前发布质量门禁已存在且部署 job 已 `needs` 它，这是对 v3 的修复。但仍有缺口：

- quality-gate 没有执行 Go 全量测试，只跑 auth/config/middleware 快包；
- 没有 `go test -race` 全量结果作为部署前证据；
- 没有 Flutter Web release build 作为质量门禁；
- 没有 govulncheck、SBOM、OSV/dependency review、容器/制品签名等供应链门禁；
- 多 workflow 并存（主部署、重新部署、移动构建），常规路径和应急路径需要明确权限与适用场景。

### N4-05 大文件和架构约束再次失控 —— P1/P2

当前最大 Go 文件已达到：

| 文件 | 行数 |
|---|---:|
| `student_service.go` | 1887 |
| `document_service.go` | 1591 |
| `kb_repo.go` | 1445 |
| `secretary_outcome_repo.go` | 1146 |
| `counselor_service.go` | 1129 |
| `routes.go` | 982 |
| `user_repo.go` | 897 |
| `vopc_delivery.go` | 886 |
| `vopc_handler.go` | 868 |
| `kb_handler.go` | 827 |

项目规范的单文件约 400 行软上限没有形成有效门禁。功能增长没有同步拆分，后续安全审计和测试定位成本会持续上升。

---

## 5. 近期功能与修复评价

### 5.1 正面进展

- **VOPC**：从项目、协作、里程碑、制品版本到关闭/例外状态机，配套迁移、权限和测试较完整，是当前新增规模最大的业务模块。
- **反馈修复闭环**：服务端 token 隔离、任务状态机、worktree 隔离、反馈原文回传和 auto 模式已形成可执行闭环；`.gitignore` 和 worktree exclude 已处理 prompt 原文误入库风险。
- **多角色与真实数据原则**：学生分年级/关注内容、知识治理、管理决策和空数据态设计，减少了硬编码示例数据冒充真实数据的问题。
- **校园地图迁移 110**：已从错误的唯一索引方案改为 `WHERE NOT EXISTS`，并增加功能回归，降低了迁移中断和审核多状态冲突风险。
- **删除会话空白页**：采用先本地移除、后请求、失败刷新恢复的方案，方向正确；历史评审识别的 401 响应误判仍应排入统一写操作处理。
- **数据库与部署**：MySQL/Redis 常驻部署、runner 编译、迁移显式验证、健康检查和回滚二进制使发布流程比 v3 更接近可运维形态。

### 5.2 不能由功能提交推导出的结论

- 功能提交和单元测试通过，不等于真实学工/SSO/模型供应商联调通过。
- 迁移文件存在，不等于生产数据库已经成功执行并完成备份。
- 前端页面存在，不等于 Web、Android、鸿蒙/iOS 各端均完成正式签名、安装和回归。
- “空数据诚实返回”不等于数据源已经接通；教师、教辅、管理决策等模块仍需绑定真实数据源验收。

---

## 6. 文档与配置一致性

当前文档治理仍是发布风险，不只是编辑问题。

1. `docs/database-migration-mysql.md` 仍出现“73 个迁移文件/成功执行 70 个”等旧数字；当前目录实测 SQL 文件数为 **107**，最大编号为 **110**，需统一统计口径并说明非 SQL/修改旧迁移情况。
2. `docs/database-migration-mysql.md` 与 `docs/deployment.md` 的正式数据库账号、Redis DB 编号存在冲突（前者曾记录 `eqs/DB 0`，后者记录 `wxx/DB 1`）。必须以生产 `/etc/wxx/env` 和实机健康检查为准，文档绑定验证日期。
3. `docs/deployment.md` 同时出现主域名、www 域名、Cloudflare 备用入口和旧方案内容；Caddy 示例、防火墙 8080 暴露、备份 cron 状态与正文架构存在不一致。
4. `AGENTS.md` 技术栈速查仍强调 SQLite，而部署文档称正式架构为 MySQL + Redis；Context Engine 文档也明确生产主路径与参考实现分叉，应在总纲中标明当前事实，而非只写目标架构。
5. `audit-report.md`、PM、QA、refactor、agent-review 多份报告采用追加式历史记录，“通过/已上线/已验证”容易脱离提交 SHA 和 workflow run。应建立唯一审核索引，标明每轮基线、状态、superseded 关系和证据链接。
6. 移动发布方案存在计划版、实测版、签名版多个近似文档，应指定一个 current 文档，其余明确归档。

---

## 7. 工作树与发布可追溯性

当前 HEAD 与 `origin/main` 一致，但工作树不干净：

### 已修改未提交（4 项）

```text
audit-report.md
frontend/lib/pages/campus/campus_map_page.dart
pm-checklist.md
refactor-notes.md
```

### 未跟踪（11 项）

包括：

```text
HEARTBEAT.md
IDENTITY.md
SOUL.md
TOOLS.md
USER.md
5 份飞书/OpenClaw/CLI OPC 方案文档
openclaw-workspace-state.json
```

其中未提交的 `frontend/lib/pages/campus/campus_map_page.dart` 对应旧报告 M2 的管理员空节点拦截建议，因此不能把它算作当前 HEAD 的正式修复。当前新报告也尚未成为提交的一部分。

另外，`git diff --check` 已发现未提交 `audit-report.md` 尾部空白。发布审核应先完成以下动作：

1. 清理或分类 agent 工作区文件；
2. 将确认的业务修复和配套文档绑定提交 SHA；
3. 清理 diff 空白；
4. 重新从干净 HEAD 执行最小质量门禁；
5. 以该 SHA 作为发布与审核唯一基线。

---

## 8. 修复路线图

### 0～4 小时：发布止血

1. 清理工作树，区分项目文件、个人 OpenClaw 文件和临时状态文件。
2. 提交本轮已确认的地图修复/文档，或明确移出仓库；修复 `git diff --check`。
3. CI workflow 删除地图 AK fallback，改为 secrets 缺失即失败；公开 key 另行限制域名和配额。
4. 在 workflow 增加 `flutter build web --release` 编译门禁，至少对发布路径进行一次完整构建。
5. 将 CI 常规门禁与应急重部署 workflow 的用途、权限和审批人写清楚。

### 0～48 小时：关闭生产阻断

1. **P0-07**：HMAC secret fail-fast；导入强制签名；建立 package receipt 唯一账本；修复 retired 增量传播；增加删除 signature、篡改、重复包、重放和分页边界测试。
2. **P1-08**：移动端 secure storage；Web 会话改 HttpOnly Cookie/BFF；建立 token/session epoch 和撤销回归。
3. **P1-10**：建立 consent ledger；将 chat、健康、情感、画像、导入等敏感写入统一接入用途级授权，并补撤回测试。
4. 建立 P0-01～06 的跨角色、跨学院越权矩阵自动化测试，防止后续模块回归。
5. 对旧版本可能落入数据库的明文密钥做扫描、轮换和迁移确认。

### 1 周：恢复可信试点

1. 将 Context Engine 接入生产装配层，删除检索副本；补来源、过期、权限、低相关和兜底契约测试。
2. 优化测试：迁移模板库/一次性 setup、包级并行、缓存依赖，把后端全量反馈周期从约 14 分钟降至 5 分钟级别。
3. 接入 `govulncheck`、SBOM、依赖漏洞扫描、Action SHA 固定/审计和制品哈希。
4. 确认 MySQL 备份 cron 已实机切换，执行一次恢复演练并记录时间、库版本和校验结果。
5. 统一数据库、Redis、域名、迁移数量、备份和发布入口文档，所有事实绑定实机验证日期。

### 1 个月：工程收敛

1. 按领域拆分 400 行以上超大文件；建立文件规模和分层依赖门禁。
2. Temporal 二选一：完成正式装配并做可靠性验收，或删除死路径和依赖；Eino/自研 agent 口径也要与总纲一致。
3. 清理 Flutter 279 条 info，至少先修 BuildContext 异步风险、弃用 API 和无效成员。
4. 建立 Web/Android/HarmonyOS/iOS 的版本、签名、哈希、安装、启动、登录和核心流程发布验收表。

---

## 9. 正式上线验收门槛

以下项目未全部通过前，维持 No-Go：

### 安全与授权

- pending/disabled/rejected 账号不得访问业务能力；旧 token 在状态/版本变更后失效。
- 学院 A/B、校级管理员、辅导员、学生的知识、情感、反馈、导出、VOPC 和代理越权矩阵全部通过。
- 缺失 HMAC secret、缺少包签名、签名删除、包重放、hash 篡改均拒绝且不落库。
- 情感、健康、画像和聊天用途均有可验证同意记录；撤回和协议升级重新同意生效。
- JWT 不在 SharedPreferences 明文存储；Web 不依赖 localStorage bearer token。

### 知识与回答

- Context Engine 进入生产装配层并成为唯一检索入口。
- 政策/条件类回答 `sources[]` 覆盖率达到 100%；低置信、无结果、权限过滤和过期内容均按规范兜底。
- retired 资源能通过增量同步传播并从接收端检索集合移除。

### 工程与发布

- `go vet`、`gofmt`、全量 `go test -race`、Flutter analyze/test/build 全部通过。
- CI 全量测试与部署 job 有明确依赖，部署不绕过质量门禁。
- Flutter 资源来源可验证，依赖漏洞扫描和 SBOM 通过，制品有 SHA256。
- MySQL/Redis 备份恢复演练通过，健康检查、迁移验证和回滚路径有记录。
- 工作树干净，发布绑定唯一 commit SHA、workflow run 和制品哈希。

---

## 10. 总结

当前版本已经不是 v3 时的“安全修复几乎未推进”的状态。214 个提交、527 个变更文件带来了显著的功能扩展和多项关键安全整改：游客/JIT、SSRF、加密、情感范围、知识库 scope/status、gofmt 和部署前置门禁均已有较强的提交与代码证据。

但项目也进入了更需要治理的阶段：功能规模增长、迁移达到 110 号、VOPC 和在线修复引入新的数据与执行边界，导致协议、隐私、供应链和文档一致性的重要性上升。当前最危险的不是普通 UI 小问题，而是**知识包可伪造/可绕过验签、敏感 token 明文存储、同意授权不可追溯、生产检索双实现以及 CI 未覆盖完整回归**。

**最终建议：条件 No-Go。**

先关闭 P0-07、P1-08、P1-10，并完成工作树/CI/构建供应链收口；再以干净提交重新执行全量质量门禁和真实部署验证。完成这些项目后，才适合把结论从“受控内部演示”上调为“有限真实试点”。

---

*本报告基于 2026-08-29 本地工作树与可复现命令输出编写。未执行生产服务器登录、真实 SSO/学工系统联调、真实 MySQL 恢复演练、外部 DNS rebinding 渗透、正式 APK/iOS/HarmonyOS 安装验收；这些项目不得从本报告中的静态检查结果推定为已通过。*
