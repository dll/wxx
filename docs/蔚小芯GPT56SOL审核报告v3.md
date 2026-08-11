# 蔚小芯 GPT-5.6 SOL 审核报告 v3

> 审核日期：2026-08-11
> 审核基线：工作树 HEAD `3dbbc89`（`main`；远端 `origin/main` 为 `ab6bc93`，本地领先 1 个提交）
> 工作树状态：7 个文件已修改未提交，3 个未跟踪目录（`docs/蔚小芯智能体UI/`、`frontend/lib/pages/services/`、`frontend/lib/theme/`）
> 对照文档：`docs/蔚小芯GPT56SOL审核报告v2.md`、`docs/蔚小芯DPV4F审核报告v3.md`、`docs/蔚小芯开发规范.md`、`specs/export-package.md`、`specs/rbac-matrix.md`、`docs/蔚小芯待完成.md`
> 审核方法：只读静态审读（`rg`/`Read`）+ 本地实测（`go vet`、`gofmt`、分包 `go test -tags fts5`、`flutter analyze --no-pub`、`flutter test --no-pub`、`go mod verify`）

## 1. 最终结论

**结论：维持 No-Go（正式生产与扩大真实学生数据试点）；受控内部演示可继续。**

v3 相对 v2 的核心判断变化有三点，均需如实记录：

1. **v2 的"全量测试超时"结论需要修正**。分包实测证明 `repository`（57s）、`handler`（84s）、`service`（109s）全部**通过**，此前的 FAIL 是超时阈值不足所致，而非断言失败。真实问题是**测试整体过慢**（三包合计约 250s），在 CI 常用超时下必然报红。
2. **发现 v2 未记录的两个新问题**：`gofmt -l server/` 命中 **17 个文件**（当前 CI 的 gofmt 检查会直接失败）；`scripts/build-all.ps1:96` 硬编码第三方 Flutter 资源镜像（供应链风险）。
3. **v2 列出的 P0-01 ~ P0-07 与 P1-08 ~ P1-10 经逐项核验，除 P0-04 的密钥回显部分已修复外，其余全部仍然存在**。因此安全结论不能上调。

### 1.1 评分

| 维度 | v2 | v3 | 变化说明 |
|---|---:|---:|---|
| PRD/需求覆盖 | 6.8 | 6.8 | `待完成.md` 未见收敛 |
| 架构与分层 | 6.5 | 6.3 | 超大文件无一拆分，合计 14260 行 |
| 后端正确性 | 6.8 | 7.0 | 分包测试全绿，上调；但 gofmt 17 处失格 |
| 前端质量 | 6.2 | 6.2 | 216 条 info（v2 为 211），0 error/0 warning |
| 安全与合规 | 5.0 | 5.2 | 仅密钥回显掩码化，其余 P0 未动 |
| Context Engine/RAG | 5.8 | 5.8 | 仍零 import，未接线 |
| 依赖与可复现构建 | 6.0 | 5.6 | 新增第三方镜像硬编码；无 SBOM/漏洞扫描 |
| CI/CD 与发布 | 6.0 | 5.5 | 实测 `ci.yml` 中 flutter 计数为 0；deploy 三 job 均无 `needs` |
| 运维与可观测性 | 6.3 | 6.3 | 无实质变化 |
| **综合** | **6.1** | **6.0** | 后端正确性上调被工程门禁与供应链下调抵消 |

## 2. 本次实测结果（可复现证据）

| 命令 | 结果 |
|---|---|
| `go vet -tags fts5 ./server/...` | **通过**（exit 0） |
| `gofmt -l server/` | **17 个文件未格式化** ⚠️ 新问题 |
| `go test -tags fts5 ./internal/auth` | ok 3.9s |
| `go test -tags fts5 ./internal/config` | ok 2.2s |
| `go test -tags fts5 ./internal/context_engine` | ok 4.3s |
| `go test -tags fts5 ./internal/llm` | ok 6.1s |
| `go test -tags fts5 ./internal/middleware` | ok 4.0s |
| `go test -tags fts5 ./internal/agent` | ok 8.9s |
| `go test -tags fts5 ./internal/repository` | **ok 57.4s** |
| `go test -tags fts5 ./internal/handler` | **ok 83.8s** |
| `go test -tags fts5 ./internal/service` | **ok 109.4s**（90s 超时下 FAIL） |
| `go test -tags fts5 ./internal/temporal/...` | ok（3 包，19s/9s/18s） |
| `./internal/jwtutil` | `[no test files]` |
| `flutter analyze --no-pub` | **216 issues，全部 info**，0 error / 0 warning（253.7s） |
| `flutter test --no-pub` | **All tests passed**（2 条用例） |
| `go mod verify` | exit 0 |

### 2.1 gofmt 未格式化清单（17 个）⚠️ 新问题

```
handler/education_health_handler.go      handler/external_app_handler.go
handler/feedback_handler.go              handler/portal_credential_handler.go
handler/student_briefing_test.go         model/dto.go
model/entity.go                          model/external_app.go
repository/external_app_repo.go          repository/feedback_repair_repo.go
service/ai_briefing_service.go           service/chat_service.go
service/external_app_service.go          service/feedback_air_repair_test.go
service/feedback_service.go              service/portal_proxy_service.go
service/portal_proxy_service_test.go
```

`ci.yml` 的 `go-backend` job 含 `gofmt -l` 检查，因此当前 HEAD 推送即会导致 CI 红灯。**修复成本极低（`gofmt -w server/`），应立即执行。**

### 2.2 测试耗时问题（修正 v2 结论）

v2 记录"handler/repository 卡在迁移 SQL、service 卡在 DOCX 解析"。实测证明这三包能够通过，问题是耗时：

- 单包最慢 `service` 109s，`handler` 84s，`repository` 57s
- 根因是**每个测试用例都重跑全套迁移**（当前 72 个迁移文件）
- 影响：CI 需 ≥180s/包超时；本地开发反馈循环过慢

**修复**：迁移结果缓存为模板库文件后按用例复制，或用 `testing.M` 级别的一次性 setup 共享只读库；DOCX 解析用例增加大小与元素数上限。

## 3. v2 问题项逐项核验

### 3.1 认证与密钥类

#### P0-01 待审核游客仍获得业务 JWT —— 【仍存在】

`auth_service.go:242` 创建 `Status: "pending"` 后，`:250` 无条件 `jwtutil.GenerateToken`，`:255-260` 响应体直接带 `Token`。`capabilities.go:184-193` 中 `guest` 角色持有 `SelfGuestRead`、`SelfKnowledgeRead`、`SelfChat`、`SelfProcessRead` 四项（比 v2 描述多一项）。放行链路已验证：`user_repo.go:527-530` 只拦 `disabled`/`rejected`，注释明确"pending 游客不拦截"。

缓解仅来自配置：`config.go:98,132` 的 `ENABLE_GUEST_REGISTER` 默认 `false`。**代码路径完全未修复**，开关一开即暴露。

#### P0-02 JIT 新用户默认 active/consented —— 【仍存在】

`user_repo.go:555-571`：`status` 硬编码 `"active"`（:556）写入 INSERT（:562），`userCtx.Consented = true`（:570）。角色取自 JWT 的 `userCtx.Role`（:561），**无白名单校验**——凭签名有效的 JWT 即可 JIT 创建 active 且已同意的账号，且角色由令牌自述决定。实际影响面比 v2 描述更大。

#### P0-04 密钥回显 + 明文降级 —— 【部分修复】✅ 唯一有实质进展项

已修复（回显）：`model_config_service.go:39,105` 返回 `cfg.ToMaskedView()`；掩码实现 `entity.go:332-341` 保留末 4 位；`:343-369` 对 `DeepseekKey`/`ZhipuKey`/`XunfeiKey`/`XunfeiSecret` 全部脱敏并附 `*KeySet` 布尔位；`model_config_service.go:43-74` 的 `isMaskedOrEmpty` 正确防止掩码覆盖真实密钥。

仍存在（明文降级）：`repository/crypto.go:20-28` 在 `WXX_ENCRYPTION_KEY` 缺失时仅 `log.Printf("[WARN] ...")` 并留 `masterKey` 为 nil；`:34-36` `encrypt` 与 `:61-63` `decrypt` 在 nil 时**原样返回明文**。缺失环境变量即静默退化为明文入库，无启动阻断。

#### P1-08 前端敏感状态明文存储 —— 【仍存在】

`storage.dart:1` 导入 `shared_preferences`，`:21` `static late SharedPreferences _prefs`。JWT（`:6,29-32`）、角色（`:8,37,46`）、能力清单（`:13,62-66`）、同意状态（`:57-58`）全部明文。`pubspec.yaml:42` 仅有 `shared_preferences: ^2.3.0`，**无 `flutter_secure_storage`**。

### 3.2 越权与数据完整性类

#### P0-03 外部系统代理 SSRF —— 【仍存在】

`integration_handler.go:36,74` 用 `strings.TrimPrefix` 取通配路径原样下传（路由 `/xuegong/*path`，`app.go:931-932`），`:42-47,80-85` 原样收集全部 query。`integration_service.go:63` `url := baseURL + path` 直接拼接，无 `url.Parse` 归一化、无路径白名单。`:27-29` 的 `http.Client` 只设 `Timeout`，**无 `Transport.DialContext`、无 `CheckRedirect`**——302 可跨主机跟随，且 `Authorization: Bearer` 头（`:68`）会随重定向发往新主机，构成凭证外泄。

五项防护全缺：主机白名单 ✗、路径白名单 ✗、私网 IP 拒绝 ✗、跨主机重定向禁止 ✗、DialContext 校验 ✗。

**可直接复用的既有正确实现**：`portal_proxy_service.go:52` 有 `allowedHosts`、`:86` 有 `CheckRedirect`、`:251` 有 `isAllowedHost`。该模式未回移到 integration。

#### P0-05 情感告警更新缺资源范围复核 —— 【仍存在】

`emotion_repo.go:216-224` `UpdateStatus` 的 SQL 谓词只有 `WHERE alert_id = ?`；`:147-155` `GetByAlertID` 同样只有 `WHERE e.alert_id = ?` 且返回含 `message_text`（原始敏感文本）。`emotion_service.go:208-213` 签名 `UpdateAlertStatus(alertID, status, acknowledgedBy string)` 无 UserContext。`emotion_handler.go:195` 只传 `userCtx.Username`。

对比同文件读路径 `:87`（ListAlerts）、`:118`（GetStats）、`:148`（Trends）**均已传 `userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role`**——读路径已按范围过滤，**写路径未跟上**，构成越权写 + 越权读原始文本。

#### P0-06 知识写入/审核/导入信任客户端 scope —— 【仍存在】

`kb_handler.go:178` Create、`:300` Update 均把 `req` 原样下传；`dto.go:160-161` 的 `owner_scope` 仅校验 `oneof=school college class`，不校验是否等于调用者范围。`kb_service.go:172-183` 把 `req.OwnerScope/OwnerID/Status` 直接赋给 `existing` 后写库——**普通 KBWrite 权限可通过 PUT 把 `status` 直接置为 `published`，绕过 `KBReview` 审核门**。

服务层签名全部只收 username：`:106` Create、`:158` Update、`:317` ApproveResource、`:394` ImportResources。`:317-327` ApproveResource 只校验 `existing.Status != "pending"`，无 scope 比对；`:344` Reject、`:371` Retire 同样。导入路径 `kb_handler.go:253` → `kb_repo.go:520` `UpsertDetailed` 原样落库。

**范围校验函数不存在**：全仓搜 `func CanRead|CanWrite|CanReviewScope|checkScope|assertScope` 零命中。唯一防线是路由层 capability（`app.go:873-889`），那是角色门禁而非资源范围门禁。

#### P0-07 知识包协议安全契约 —— 【三点全部仍存在】

位置 `service/knowledge_package_service.go`。

**(a) HMAC 可跳过**：`:100-103` 仅 `if s.secret != ""` 才签名；`:169-174` 仅 `if s.secret != "" && manifest.Signature != ""` 才验签——**双重可选，攻击者删掉 manifest 的 `signature` 字段即整段跳过验签**。唯一必过的 `:165-168` sha256 自校验，hash 由包内自带，可同时篡改内容与 hash，无认证价值。`config.go:190` `HMACSecret` 默认空，`app.go:279` 无条件注入——默认部署签名与验签双向关闭。

**(b) packageId 幂等**：`:129-204` ImportPackage 全程未持久化 `manifest.PackageID`，仅 `:196` 回显。72 个迁移中无幂等表。**缓解**：`kb_repo.go:531-541` 有行级幂等，简单重放不产生重复行；但 `:533` 对 `retired` 无条件覆盖、`:538` 对 `draft` 放行，重放包可反复改写本地记录。分片路径 `knowledge_import_resume.go:180-191` 同样无 ledger。

**(c) retired 不进增量**：`kb_repo.go:721` `FROM kb_resources WHERE status = 'published'` 硬编码，`:724-741` 无处放宽 status。与消费端契约直接冲突——`kb_repo.go:532-534` 注释写明"retired 状态必须传播"并为此实现覆盖分支，但生产端永不导出 retired 行，**该分支在包同步场景下不可达**。下架政策无法传播，接收方会长期保留已撤销条目并继续用于检索与 AnswerCard 生成。

### 3.3 工程质量类

#### P1-09 Context Engine 生产链路分叉 —— 【仍存在】

`context_engine/engine.go:5-7` 包注释仍自我标注"⚠️ 实验性/未接线"。`grep -rn "wxx/server/internal/context_engine" --include=*.go .` **零结果**——零 import 孤儿包，`app.go` 中 `context_engine`/`ContextEngine` 均无匹配。

生产代码直接调 `kbRepo.Search` 共 **9 处、7 个文件**：`chat_service.go`（192/342/923，3 处）、`recommendation_service.go:103`、`agent/major_agent.go:29`、`qa_agent.go:28`、`process_agent.go:28`、`policy_agent.go:29`、`emotion_agent.go:83`。另 `temporal/activities/chat_activities.go:70` 为第 8 个文件。`chat_service.go` 另有 4 处 `SearchStructured`/`SearchFAQ`（180/334/914/1031）——结构化优先逻辑在单文件内重复 3 遍。

#### P1-10 consent ledger —— 【未实现】

72 个迁移中 `consent` 仅命中 `041_add_user_consented.sql`，核心只有一行：`ALTER TABLE users ADD COLUMN consented INTEGER NOT NULL DEFAULT 1`。协议版本、用途、供应商、撤回时间四字段全缺，无历史留痕表。

两个附加风险写在迁移注释里：`DEFAULT 1`（存量用户视为已授权）；`user_repo.go:569-570` 对 JIT/SSO 新用户直接置 true（**从未真正征得同意**）。

`middleware/consent.go` 为 41 行单布尔判断，无版本比对、无用途区分、无撤回入口。且 `RequireConsent()` **仅挂载 2 个路由**（`app.go:759-760` 的 `/chat` 与 `/chat/stream`），其余写入类接口（学生画像、健康记录、情感）均未挂载。

情感独立授权仅为 RBAC capability 而非授权记录：`capabilities.go:40` 定义 `self.emotion.consent`，`app.go:774` 用 **`RequireAnyCapability`（或关系）**——持有 `SelfEmotionStats` 即可通过，`SelfEmotionConsent` 并非必需条件，**无强制力**，且无任何持久化记录或撤回时间。

#### 超大文件 —— 【无一拆分，8/8 超阈值】

| 文件 | 行数 |
|---|---:|
| `frontend/lib/models/models.dart` | 2853 |
| `frontend/lib/pages/home/home_page.dart` | 2175 |
| `server/internal/service/student_service.go` | 1887 |
| `server/internal/handler/student_handler.go` | 1672 |
| `server/internal/service/document_service.go` | 1591 |
| `server/pkg/app/app.go` | 1486 |
| `server/internal/repository/kb_repo.go` | 1326 |
| `server/internal/service/chat_service.go` | 1270 |
| **合计** | **14260** |

规范要求单文件约 400 行以内，现状显著超标。

#### Eino / Temporal —— 【Eino 已接线但为薄壳；Temporal 为死路径】

注：`go.mod` 在仓库根（单模块布局），非 `server/`。

**Eino 非死依赖**：`agent/eino_orchestrator.go:6` import `cloudwego/eino/compose`，且 `app.go:174-175` 已接入装配层。但实现是薄壳——Graph 只有单个 Lambda 节点 `route`，内部直接转调自研 `base.Execute`，拓扑 `START → route → END`。依赖真实生效，但未获得 Eino 图编排能力。

**Temporal 是死路径**：代码规模不小（`temporal/` 下 `client.go`、`worker.go`、4 个 workflow、`activities/chat_activities.go`），`chat_service.go:825` `askViaTemporal` 与 `emotion_service.go:344` `analyzeViaTemporal` 均完整实现。但 `SetTemporalClient`（`chat_service.go:93`、`emotion_service.go:37`）**全仓无任何非测试调用点**；`app.go:159-164` 的 Temporal 块只打日志，不构造 client、不注入 service。因此 `s.temporalClient` 恒为 nil，分支恒假，始终降级到 `askDirect`。`config.go:178` 默认空亦为禁用。

即：完整实现 + 零装配 + 拖着 `go.temporal.io/sdk v1.43.0` 依赖体积，并给 P1-09 的检索逻辑又添一份副本（`chat_activities.go:70`）。**要么接线，要么整体移除。**

## 4. 新增问题（v2 未记录）

### N3-01 gofmt 17 个文件未格式化 —— 【P1，修复成本极低】

见 §2.1。`ci.yml` 的 gofmt 检查会直接失败。**立即执行 `gofmt -w server/`。**

### N3-02 构建脚本硬编码第三方 Flutter 资源镜像 —— 【P1，供应链】

`scripts/build-all.ps1:96`：

```powershell
$env:FLUTTER_STORAGE_BASE_URL = "https://flutter-ohos.obs.cn-south-1.myhuaweicloud.com"
```

本次 `flutter test` 实测输出印证："Flutter assets will be downloaded from https://flutter-ohos.obs.cn-south-1.myhuaweicloud.com. Make sure you trust this source!"

该镜像是第三方（OpenHarmony 社区）华为云 OBS 存储桶，非 Google 官方 `storage.googleapis.com`，也非可信企业镜像。构建产物的 Dart SDK / engine artifacts 来源不可验证，**存在供应链投毒面**。

**修复**：改用官方源或经审计的企业内网镜像；若因网络必须用镜像，则固定 artifact SHA256 并在 CI 校验；在 `docs/deployment.md` 记录来源与信任依据。

### N3-03 CI 门禁实测数据（强化 v2 §8.2 结论）

| 检查项 | 实测结果 |
|---|---|
| `ci.yml` 中 `flutter` 出现次数 | **0** |
| `ci.yml` 的 job | 仅 `go-backend`、`lint-docs` |
| `deploy-frontend.yml` 的 job | `deploy-web`(19)、`deploy-server`(92)、`build-apk`(160) |
| 上述 job 的 `needs:` | **三者均无** |
| 供应链扫描关键字（govulncheck/trivy/syft/sbom/dependency-review/codeql/gosec/osv-scanner） | **零命中** |

即：前端 5000+ 行 Dart 无任何 CI 静态分析与测试；**即使 `ci.yml` 全红，`deploy-server` 仍会照常编译、scp 到腾讯云并 `systemctl restart wxx`**——生产部署无质量门禁前置。

另附带观察：`deploy-frontend.yml:46-48` 与 `:192-194` 将百度、高德、腾讯地图 AK 以**明文默认值硬编码**在 workflow 中作为 secrets 缺失时的 fallback。

## 5. 工作树卫生问题

审核基线存在 7 个已修改未提交文件与 3 个未跟踪目录（含 `frontend/lib/theme/`、`frontend/lib/pages/services/` 两个疑似新功能目录）。本地 `main` 领先 `origin/main` 一个提交。

这使"当前线上行为"与"仓库 HEAD"与"远端"三者不一致，审核结论的可追溯性受损。**要求**：审核/发布前必须干净工作树，所有"已完成"声明绑定提交 SHA。

## 6. 修复路线图

### 0～4 小时：零成本止血

1. `gofmt -w server/` 并提交（解除 CI 红灯）。
2. 提交或清理工作树中的 7 个改动与 3 个未跟踪目录，使基线可追溯。
3. `deploy-frontend.yml` 三个 job 增加 `needs: [quality-gate]`（先指向现有 `go-backend`）。

### 0～48 小时：冻结安全风险

1. **P0-01**：pending 只签发审核态票据；或从 `guest` 移除 `SelfChat`/`SelfKnowledgeRead`/`SelfProcessRead`/`SelfGuestRead`；补 pending → 403 回归测试。
2. **P0-02**：JIT 未知用户进入 pending，`consented` 默认 false，角色不得取自 JWT 自述而须走服务端映射。
3. **P0-04 残留**：`crypto.go` 生产环境缺 `WXX_ENCRYPTION_KEY` 时 **fail-fast**，删除明文 fallback；迁移既有明文数据。
4. **P0-03**：把 `portal_proxy_service.go` 的 `allowedHosts` + `CheckRedirect` + `isAllowedHost` 模式回移到 `integration_service.go`；增加 DialContext 私网 IP 拒绝；补 SSRF/DNS-rebinding 测试。
5. **P0-05**：`UpdateStatus`/`GetByAlertID` 接收 `UserContext` 并作为 SQL 谓词；对齐同文件读路径已有的 scope 过滤；补 A/B 学院隔离测试。
6. **P0-06**：KB Service 全部接收 `UserContext`；新建 `CanWriteScope`/`CanReviewScope` 并在 Repository 层默认拒绝；`status` 转换由服务端决定，禁止请求体直接置 `published`。
7. **N3-02**：移除或固定校验第三方 Flutter 镜像。

### 1 周：恢复可信试点

1. **P0-07**：生产强制 HMAC（缺失即 fail-fast）；验签失败零落库；建立 package receipt 唯一约束；引入 change sequence 或 revision 事件表使 retired 可增量传播。
2. **P1-09**：Context Engine 作为唯一检索入口，消除 9 处 `kbRepo.Search` 重复；补零结果/低相关/过期/跨范围/sources 契约测试。
3. **P1-10**：建立 consent ledger（版本/用途/供应商/时间/撤回）；`RequireConsent` 扩展到所有敏感写入路由；情感授权改为 `RequireCapability`（且关系）并持久化记录。
4. **测试耗时**：迁移结果模板化复用，把三大包从 250s 压到 60s 内。
5. **CI**：新增 `quality-gate` job 串行执行 Go 全量测试（≥180s 超时）+ Flutter lockfile/analyze/test/build；部署 job 全部 `needs` 它；接入 `govulncheck` + SBOM + `go mod verify`。
6. **P1-08**：移动端引入 `flutter_secure_storage`；Web 走同域 BFF + HttpOnly Cookie；建立 session epoch。

### 1 个月：工程收敛

1. 按领域拆分 8 个超大文件（14260 行），建立 400 行软上限与依赖方向检查。
2. Temporal 二选一：完成装配或整体移除（含 4 个 workflow、activities 包、2 处 service 分支）；Eino 薄壳升级为真实图编排或在文档标注"形式化包装"。
3. 移除 workflow 中的明文地图 AK 默认值，改为 secrets 必需。
4. SQLite WAL/备份恢复/锁等待压测；建立 P95/P99、错误率、LLM 超时、检索命中率与成本预算。
5. 按 `待完成.md` 的 P1→P2→P3 推进，每项绑定验收测试与外部联调负责人。

## 7. 上线验收门槛（沿用 v2，补充实测项）

- **格式与门禁**：`gofmt -l server/` 输出为空；`ci.yml` 含 Flutter analyze/test/build；所有 deploy job 有 `needs`。
- **认证**：停用/降权/删除/改密后旧 Token 立即失效；pending 不得访问业务能力；JIT 不得创建 active 账号。
- **授权**：学生、学院 A、学院 B、校级管理员对知识、情感、反馈、导出、代理执行越权矩阵，全部拒绝且不泄露存在性。
- **隐私**：未同意/撤回/版本过期均阻断敏感能力；PII 不出现在模型请求、日志、trace。
- **同步**：缺文件、哈希篡改、签名篡改（含删除 signature 字段）、重复 package、retired 传播、分页边界，各有自动化验收。
- **供应链**：构建 artifact 来源可验证；`go mod verify` + 漏洞扫描 + SBOM 通过。
- **发布**：Go test/vet/race + Flutter analyze/test/build + APK 生产签名 + 制品哈希 + 健康检查全绿。

## 8. 总结

v3 的准确表述是：**后端正确性证据较 v2 更好（分包测试全绿，v2 的"超时即失败"结论已修正为"过慢"），但安全边界没有实质推进——P0-01/02/03/05/06/07 与 P1-08/09/10 全部仍在，仅 P0-04 的密钥回显完成掩码化。同时新暴露 gofmt 17 处失格与第三方构建镜像两项工程/供应链问题，CI 门禁缺位得到实测确证。**

优先级建议：先花 4 小时做完 §6 的零成本止血（gofmt、工作树、`needs`），再用 48 小时关闭 §3.2 的越权类 P0——这批问题有仓库内既有正确实现（`portal_proxy_service.go` 的白名单、emotion 读路径的 scope 谓词）可直接对齐，无需引入新依赖，修复性价比最高。在此之前维持 No-Go。

---

*报告生成：2026-08-11。所有行号、行数、命令结果均为本次实测值。未执行侵入式生产渗透、真实 SSO/学工系统联调，未验证公网 DNS/备案状态。*
