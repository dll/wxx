# wxx 问题反馈→管理员反馈管理→反馈详情→在线修复/复制提交 全链路只读审计

> 审计日期：2026-08-26 · 审计方式：**纯只读**（未修改任何源码、未执行任何写操作、未提交 git）
> 说明：本文件覆盖了 2026-08-21 的 vOPC PRD 核对清单；旧版内容可在 git 历史找回（commit `2aae19d`）。
> 目标流程假设：**管理员审核后单个/批量创建修复任务 → 本机受控修复 → 自动验证 → 管理员验收 → 人工确认部署**。

---

## 一、链路总览（现状）

```
用户端                                    管理端
──────                                   ──────
feedback_dialog.dart 提交反馈             pages/admin/feedback_page.dart 列表
(自动截屏+分类+模块+草稿)        ──►       ├─ 状态筛选 / 仪表盘统计
POST /api/v1/feedback                     ├─ 行内：驳回/处理中/解决
                                          └─ 详情页 FeedbackDetailPage
my_feedbacks_page.dart「我的反馈」             ├─ 基本信息/内容/截图/回复/评分/时间线
├─ 查看状态与回复                             ├─ 复制完整报告(Markdown)
├─ 复制 JSON                                  ├─ 复制结构化(JSON)
└─ 满意度评价                                  └─ 「在线修复」按钮 → feedback_repair.dart 面板
                                                  ├─ 自动调用 POST /feedback/:id/ai-repair
                                                  ├─ 展示：摘要/OCR/根因/代码定位/修复建议
                                                  ├─ 复制完整报告(Markdown) / 复制JSON   ←两种格式
                                                  ├─ 「本机复现指引」静态文本
                                                  └─ 「验证通过并标记已解决」→ PUT /feedback/:id
```

**核心结论：当前「在线修复」是"AI 诊断 + 复制到剪贴板 + 人工去本机修"的辅助面板，不是自动修复系统。**
后端没有任何执行代码修改、构建或部署的代码路径；`feedback_repair_jobs` 工单表虽有 build/deploy/healthcheck 阶段常量，但实际只会写入 `diagnose→done` 两段。

---

## 二、前端页面与交互

| 文件 | 角色 | 关键交互 |
|---|---|---|
| `frontend/lib/widgets/feedback_dialog.dart` (465 行) | 用户提交 | 打开前自动抓当前页面帧；分类 SegmentedButton（answer_error/suggestion/other）；模块下拉（`models.dart:1362 feedbackModules` 13 个枚举）；关闭/失败时草稿存 SharedPreferences 并可恢复；Web 截图 ≤900KB 转 base64 data URL 内联入库；「复制 JSON」「复制 Markdown」（`FeedbackReport.buildDraftJson/buildDraftMarkdown`）供手工粘贴给外部 AI 工具 |
| `frontend/lib/pages/profile/my_feedbacks_page.dart` | 用户查看 | 我的反馈列表/详情、状态与回复展示、复制 JSON（`buildJson`）、满意度评价（1-5 星，仅 resolved 且未评过） |
| `frontend/lib/pages/admin/feedback_page.dart` (1280 行) | 管理员 | 入口在「我的→管理服务」（能力 union.feedback.list 门控）；仪表盘（总数/待处理/处理中/已解决/近7天趋势/热门分类/平均时长）；列表状态筛选 pending/processing/resolved/dismissed；行内驳回/处理中/解决；详情页含截图放大、关联知识资源、处理回复、满意度、时间线；右上角「复制完整反馈」（Markdown 含日志）；底部「在线修复」按钮（任意状态可开） |
| `frontend/lib/widgets/feedback_repair.dart` (544 行) | 管理员修复助手 | `showOnlineRepair` 底部抽屉；initState **自动**触发 `_runAIRepair()`（每次打开都调一次 LLM 接口并落一条工单）；本地关键词兜底匹配模块→文件映射；展示问题摘要/OCR/根因分析/代码文件卡片/修复建议；「复制完整报告（提交 AI 修复）」「复制结构化 JSON」两按钮；静态「本机复现指引」（flutter run / go run / make deploy-release）；「验证通过并标记已解决」→ 弹窗填回复 → `resolveFeedback(id,'resolved')` |
| `frontend/lib/utils/feedback_report.dart` | 报告生成器 | 统一 JSON + Markdown 双格式生成；字段全量（含 ai_* 诊断附加段、logs、截图 base64）；三处复用保证不丢字段 |
| `frontend/lib/providers/feedback_provider.dart` (385 行) | 状态层 | fetchFeedbacks/fetchMyFeedbacks/fetchFeedbackDetail/fetchFeedbackLogs/fetchStats/uploadScreenshot(Bytes)/submitFeedback/resolveFeedback/rateFeedback/linkResource/**aiRepair**。**没有**对 `GET /feedback/:id/ai-repair/job` 的封装与轮询（全前端 grep 无引用） |

交互缺口：
- 管理端列表**无多选/批量操作**（批量能力目前只有 admin 用户管理与 KB 审核，见 api_config.dart:94-118，均不涉反馈）。
- 修复面板无工单进度展示（job 轮询接口存在但前端未接）。
- 已 resolved 的反馈打开修复面板仍显示「验证通过并标记已解决」，点击会被后端状态机拒绝（resolved 为终态），仅提示"操作失败"。
- 反馈详情页 `GET /feedback/:id` 无 owner 过滤（见下文隐私节），普通学生直接构造 ID 可看他人反馈。

---

## 三、API 路由与权限能力

路由注册：`server/pkg/app/routes.go`（secured 组 = JWTAuth + EnsureUserExists，前缀 `/api/v1`）。

| 方法与路径 | 能力门控 | Handler | 备注 |
|---|---|---|---|
| POST /api/v1/feedback | `SelfFeedbackSubmit` | Submit | 学生及以上均可提交；校验 token 用户仍存在（防 Vercel 冷启动脏 token） |
| POST /api/v1/feedback/screenshot | `SelfFeedbackSubmit` | UploadScreenshot | ≤5MB；存 `feedback_screenshots` blob 表；返回 `/uploads/feedback/{filename}` |
| GET /api/v1/feedback/mine | `SelfFeedbackSubmit` | Mine | 按 user_id 过滤，安全 |
| **GET /api/v1/feedback/:id** | **无中间件、handler 内也无 user_id 校验** | Get | 注释声称"handler 内 user_id 校验"，**实际代码没有** → 任意登录用户可读任意反馈详情（P1 隐私缺口） |
| PUT /api/v1/feedback/:id/rate | `SelfFeedbackSubmit` | Rate | Service 内校验 fb.UserID==userID、status==resolved、未评过 |
| **GET /api/v1/feedback/:id/logs** | **无能力门控、无归属校验** | GetLogs | 同上，任意登录用户可读任意反馈处理记录 |
| GET /api/v1/feedback | `UnionFeedbackList` | List | 管理员列表（student_union 及以上继承获得） |
| PUT /api/v1/feedback/:id | `UnionFeedbackList` | Resolve | 状态机校验在 Service |
| POST /api/v1/feedback/:id/ai-repair | `UnionFeedbackWrite` | AIRepair | 触发诊断；operator 取当前用户名 |
| GET /api/v1/feedback/:id/ai-repair/job | `UnionFeedbackWrite` | LatestRepairJob | 返回最新工单（前端未使用） |
| GET /api/v1/admin/feedback/stats | `UnionFeedbackRead`（常量名复用 admin 前缀） | Stats | 统计 |
| PUT /api/v1/admin/feedback/:id/link-resource | `UnionFeedbackWrite` | LinkResource | 关联知识资源 |
| GET /api/v1/uploads/feedback/:filename | 仅 JWT，**无归属/能力校验** | ServeScreenshot | 任意登录用户按文件名可取任意截图；文件名为 uuid 前 8 位，可枚举面有限但非零 |

能力定义：`server/internal/auth/capabilities.go`
- `SelfFeedbackSubmit="self.feedback.submit"`：student 基线（:247）
- `UnionFeedbackList="union.feedback.list"`、`UnionFeedbackRead="admin.feedback.read"`、`UnionFeedbackWrite="admin.feedback.write"`：student_union 获得（:280），counselor/admin 等经角色树继承
- guest 仅 SelfGuestRead，无任何反馈能力 ✅

---

## 四、Handler / Service / Repository 分层

```
handler/feedback_handler.go      参数绑定+HTTP 语义（无业务逻辑，Get/GetLogs 权限缺失在此层可见）
service/feedback_service.go      全部业务规则：
                                 - Submit：fb-xxxxxxxx ID、pending 默认态、AddLog(submit)、answer_error 异步钩子 RetireFAQ(app.go:255 注入)
                                 - Resolve：状态机 validTransitions + AddLog(status_change) + resolved 时异步站内通知(user_notifications 直插 SQL)
                                 - AIRepair：①建工单(rp-xxxx run_id) ②截图 blob→Zhipu4V OCR ③本地关键词兜底 matched_files
                                            ④文本模型 JSON 诊断(module/summary/code_files/root_cause/repair_hint，温度0.2/800tok，
                                              prompt 内嵌 moduleFilesMap 目录约束候选文件) ⑤finish() 落库 status=succeeded/stage=done
                                 - Rate/LinkResource/ListMine/GetStats/SaveScreenshot
repository/feedback_repo.go      双方言(MySQL/SQLite via dbutil.IsMySQL)；listFeedbackCols 统一列；List/Count(ListByUser/CountByUser/
                                 GetByFeedbackID/Update/UpdateRating/LinkResource/CountByStatus/CountByCategory/WeekTrend/
                                 TopIssues/AvgResolveHours/AddLog/ListLogs)
repository/feedback_repair_repo.go  Create(run_id UNIQUE)/AppendLog(log_text || 追加)/UpdateStage/Finalize/SetEditedFiles/
                                 GetByRunID/LatestByFeedback —— 注意 AppendLog 用 SQLite 语法经 AdaptForDriver 适配
repository/feedback_screenshot_repo.go  blob 存取(Save/GetByFilename)
装配：pkg/app/app.go:214-247    SetDB / SetRepairRepo(恒注入) / SetAnswerErrorHook / SetAIRepairClients(Zhipu4V 有 key 时注入)
```

测试现状：
- `service/feedback_air_repair_test.go`：5 个用例全部通过逻辑完备（LocalFallback / WithLLM(mock) / parseJSON / PersistJob / NoRepoNoRunID）。**注意：LLM 失败降级路径最终也把工单标为 succeeded**（"成功"语义=完成诊断而非完成修复）。
- `handler/feedback_handler_test.go`：Submit 成功路径等基础用例；**无 Get/GetLogs 越权负向用例**。

---

## 五、数据库表与迁移

| 迁移 | 内容 |
|---|---|
| `009_feedback_and_settings.sql` | 建 `feedback` 表：id/feedback_id(UNIQUE)/user_id/username/message_id/resource_id/category/content/screenshot_url/status/resolved_by/resolved_at/reply/created_at/updated_at + system_settings |
| `011_feedback_enhance.sql` | 补 screenshot_url/reply 列（幂等） |
| `019_feedback_screenshot_blob.sql` | 建 `feedback_screenshots`（filename UNIQUE/mime/size/data_base64/uploaded_by），解决 Vercel /tmp 易失 |
| `028_fix_feedback_fk.sql` | 重建 feedback 表去掉 users FK（应用层校验） |
| `039_feedback_closed_loop.sql` | +rating/rating_comment/rated_at/linked_resource_note/linked_at/linked_by；建 `feedback_logs`(action/operator/detail/created_at) 时间线表 |
| `063_feedback_module.sql` | +module TEXT（所属模块，在线修复定位用）+索引 |
| `064_feedback_repair_jobs.sql` | 建 `feedback_repair_jobs`：run_id UNIQUE/feedback_id/operator/status(running\|succeeded\|failed\|rolled_back)/stage(init/diagnose/gen_patch/apply/build/deploy/healthcheck/done/failed)/log_text/edited_files(JSON数组)/summary/detail |

模型常量（entity.go:266-299）已预留 gen_patch/apply/build/verify/deploy/healthcheck 阶段与 rolled_back 状态，**但 Service 从未使用**——这是现成的扩展锚点。

生产库：已从 SQLite 迁移到 **MySQL + Redis**（commit 7b92bd7），repo 层双方言兼容；服务器部署于 `/opt/wxx`（systemd 单元 `wxx` + Caddy 正式入口 wxx-agent.online），Cloudflare Pages 为备用 Web 入口。新迁移需保持 MySQL 兼容（参考 AvgResolveHours 与 AdaptForDriver 的写法）。

---

## 六、反馈状态字段与流转

`feedback.status`：`pending → processing | resolved | dismissed`；`processing → resolved | dismissed`；resolved/dismissed 为终态不可再变（Service validTransitions 硬编码）。

每次变更/提交/评价/关联/AI 诊断均写 `feedback_logs`（action: submit/status_change/link_resource/rate/ai_repair），详情页时间线可视化。

resolved 时异步插入 `user_notifications`（type='feedback'）通知提交者；answer_error 类反馈异步触发 FAQ 缓存 retired（app.go:255）。

满意度：rating 1-5，仅 resolved 后本人可评一次。

---

## 七、附件与隐私

- **双存储模式并存**：①原生上传接口→`feedback_screenshots` blob 表，URL 形如 `/uploads/feedback/xxx.png`；②Web 端 `uploadScreenshotBytes` 直接把 ≤900KB 图片编码为 data: URL 存进 `feedback.screenshot_url` 文本列（跨实例可靠但撑大主表行）。
- 隐私问题（按严重度）：
  1. `GET /feedback/:id` 与 `/feedback/:id/logs` 无归属校验 → 水平越权读任意反馈全文与截图链接（注释与实现不符，疑似回归）。
  2. `GET /uploads/feedback/:filename` 仅要求登录，无归属校验。
  3. 截图可能包含其他学生的个人信息（成绩单、聊天页等）；一旦接入自动修复任务下发链路，**不得默认把截图/全文推给执行端**。
  4. 复制到剪贴板的报告含用户名/截图 base64，属敏感外泄面（设计如此，但需在制度上限制粘贴去向）。

---

## 八、「在线修复」按钮的真实行为（关键结论）

点击后**唯一的服务端动作是诊断**：

1. 前端自动 `POST /feedback/:id/ai-repair`（每次打开面板必调一次，产生一次 GLM-4.6V OCR + 一次文本模型调用 + 一条 repair_job 记录）；
2. 返回 module/summary/code_files/root_cause/repair_hint/ocr_text/matched_files/run_id；
3. 前端渲染诊断卡片 + 两枚剪贴板按钮（Markdown/JSON 双格式，`FeedbackReport` 统一生成，字段含反馈全量+日志+AI 诊断）；
4. 管理员把剪贴板内容粘贴给 Claude/GLM 等**外部 AI 编码工具**，在本机仓库手工完成修改；
5. 回面板点「验证通过并标记已解决」→ `PUT /feedback/:id` 置 resolved（此步即"验收"，但与真实验证无强关联，纯自觉）。

**不存在**的环节：服务端改码、补丁生成/应用、构建、健康检查、热部署、回滚。`RepairStage*` 中 apply/build/deploy 等常量为死代码；`rolled_back` 状态从未出现。工单查询接口虽实现且带审计字段（edited_files/log_text），前端完全未消费。

---

## 九、本机自动修复可复用设施盘点

| 设施 | 位置 | 可复用点 |
|---|---|---|
| 构建命令 | `Makefile` | `make test`（go test ./...）、`make lint`（go vet）、`flutter-test`、`flutter-build-web(-safe/-output)`、`flutter-build-apk(-safe)`、`deploy-release`（版本自增+Web/APK 发布）、`eval/eval-gate` 质量门禁 |
| 全量构建脚本 | `scripts/build-all.ps1` | 版本号 patch 自增 + Web/APK 顺序构建 |
| CI/CD 部署流水线 | `.github/workflows/deploy-backend.yml` | SSH 私钥 secrets、mysqldump 部署前备份（/opt/wxx/backup/wxx-pre-*.sql）、二进制回滚副本（wxx-server.rollback.*）、systemctl stop/start wxx、健康确认 `systemctl is-active`；`redeploy-server-tar.yml` 支持源码 tar 重部署；`deploy-frontend.yml` rsync 到 /opt/wxx/frontend/web |
| 健康检查端点 | `routes.go:44 router.GET("/health", healthHandler)` | 公开无鉴权，可直接做部署后探活 |
| LLM 客户端 | `internal/llm/`（Zhipu4VClient OCR + ChatClient failover + MockClient） | 诊断/补丁说明生成；Mock 支持单测 |
| 工作流引擎 | `internal/temporal/`（workflows+activities，chat/kb/emotion/integration 已用） | 若修复任务需要长时编排/重试/心跳，可复用；最小方案也可不用 |
| 工单持久化骨架 | `feedback_repair_jobs` + FeedbackRepairRepo | 表结构与阶段常量可直接扩展审批/验证/验收/部署确认字段 |
| 测试基线 | go handler/service 测试 + `flutter test` + vopc_provider_test | 自动验证套件的现成组成 |
| 数据库迁移机制 | 启动时 app.NewWithConfig 自动执行；cmd/migrate | 新增迁移只需追加编号 SQL |

**注意**：仓库内**没有**任何"服务端自我改码/热更新"代码，也没有 OpenClaw/agent 类守护进程集成；"本机修复"目前只能由人在开发机执行（或人驱动 AI CLI）。

---

## 十、缺口清单（按优先级）

| # | 缺口 | 级别 |
|---|---|---|
| G1 | 反馈详情/日志接口缺归属与能力校验（水平越权读） | P1 安全 |
| G2 | 无"修复任务"实体：审核、认领、验证结果、验收、部署确认均无落库载体 | P0（对目标流程） |
| G3 | 无批量选择与批量创建任务的前后端能力 | P0（对目标流程） |
| G4 | 本机执行端协议缺失（认领/心跳/上报验证结果/上报 diff） | P0（对目标流程） |
| G5 | 自动验证无统一出口（go test/vet、flutter analyze/test 结果未结构化回传） | P0 |
| G6 | 验收人与执行人不分离、部署无人工确认闸门 | P1 流程 |
| G7 | 工单 job 接口前端未接，diagnose 即"succeeded"语义误导 | P2 |
| G8 | 修复面板对 resolved 反馈仍显示可解决按钮 | P3 UX |
| G9 | 截图双存储模式并存，data URL 撑大主表 | P3 |
| G10 | category 校验(oneof=answer_error,suggestion,other)与历史值 bug/feature_request（stats 映射仍在）不一致 | P3 |

---

## 十一、最小可用改造清单（MVP）

原则：**服务端只做状态机与审计，绝不在服务器上执行代码修改/构建**；一切改动发生在受控本机（开发机），服务器仅接收结果上报。尽量扩展现有表而非新建平行体系。

### M1 数据库（1 个迁移，编号顺延 109）

`server/migrations/109_feedback_repair_tasks.sql`（MySQL 兼容写法）：

```sql
CREATE TABLE IF NOT EXISTS feedback_repair_tasks (
    id            INTEGER/INT AUTO_INCREMENT PRIMARY KEY,
    task_no       TEXT NOT NULL UNIQUE,            -- rt-xxxxxxxx
    creator       TEXT NOT NULL,                   -- 创建(审核)管理员
    feedback_ids  TEXT NOT NULL,                   -- JSON 数组，支持单条/批量
    title         TEXT NOT NULL DEFAULT '',
    diagnosis     TEXT NOT NULL DEFAULT '',        -- 合并后的 AI 诊断(JSON: modules/code_files/root_cause/hints)
    status        TEXT NOT NULL DEFAULT 'approved',-- approved/running/verifying/verify_failed/awaiting_acceptance/
                                                   -- rejected_accepted/deploy_pending/deploying/deployed/closed/cancelled
    worker_host   TEXT NOT NULL DEFAULT '',
    base_commit   TEXT NOT NULL DEFAULT '',
    branch        TEXT NOT NULL DEFAULT '',
    verify_result TEXT NOT NULL DEFAULT '',        -- JSON: go_test/go_vet/flutter_analyze/flutter_test passed+摘要
    diff_stat     TEXT NOT NULL DEFAULT '',
    log_text      TEXT DEFAULT '',
    accept_note   TEXT NOT NULL DEFAULT '',
    accepted_by   TEXT NOT NULL DEFAULT '',
    deploy_confirmed_by TEXT NOT NULL DEFAULT '',
    deploy_ref    TEXT NOT NULL DEFAULT '',        -- 部署方式记录(GH run id / 手工命令)
    created_at/updated_at ...
);
CREATE INDEX idx_frt_status ON feedback_repair_tasks(status);
-- 同时给 feedback_logs 增加 action 取值：repair_task_created/repair_verified/repair_accepted/deploy_confirmed
```

（备选：直接 ALTER `feedback_repair_jobs` 加列；新建 tasks 表更清晰，jobs 保留为"诊断记录"。）

### M2 后端（Handler/Service/Repo 各一文件增量 + 路由 ~10 条）

新增能力常量：`SystemRepairExecute = "system.repair.execute"`（仅授予一个本机服务账号角色，如 sysadmin 树下），管理侧沿用 `UnionFeedbackWrite/List`。

| 路由 | 能力 | 行为 |
|---|---|---|
| POST /api/v1/admin/feedback/repair-tasks | UnionFeedbackWrite | body:{feedback_ids:[..], title?}；逐条跑现有 AIRepair 诊断合并 code_files；创建任务 status=approved（创建即审核，记录 creator）；每条反馈 AddLog(repair_task_created) |
| GET /api/v1/admin/feedback/repair-tasks?status=&page= | UnionFeedbackList | 任务分页列表 |
| GET /api/v1/admin/feedback/repair-tasks/:no | UnionFeedbackList | 详情（含 verify_result/diff_stat/log） |
| POST /api/v1/admin/feedback/repair-tasks/:no/cancel | UnionFeedbackWrite | 仅 approved/verify_failed 可取消 |
| POST /api/v1/admin/feedback/repair-tasks/:no/accept | UnionFeedbackWrite | awaiting_acceptance→accepted(deploy_pending)；建议校验 accepted_by != worker 上报人（单人场景可放宽为提示） |
| POST /api/v1/admin/feedback/repair-tasks/:no/reject | UnionFeedbackWrite | 驳回整改，回 verify_failed 或 cancelled |
| POST /api/v1/admin/feedback/repair-tasks/:no/deploy-confirm | UnionFeedbackWrite | deploy_pending→deploying；记录 deploy_confirmed_by；**只做标记，不触发服务器动作** |
| POST /api/v1/admin/feedback/repair-tasks/:no/deploy-done | UnionFeedbackWrite | deploying→deployed/closed；可选联动：把 feedback_ids 批量 Resolve(resolved, reply="已于 vX.X.X 修复") 复用现有通知链路 |
| GET /api/v1/internal/repair-tasks/next | SystemRepairExecute | 执行端认领：原子地把最老 approved 任务置 running（全局同时仅 1 个 running，避免并发改码冲突）；返回脱荷载荷（不含截图 base64，仅文本诊断+code_files） |
| POST .../:no/claim · /heartbeat(可选) · /verify · /abandon | SystemRepairExecute | verify 上报 {passed, go_test, go_vet, flutter_analyze, flutter_test, diff_stat, log}；passed→awaiting_acceptance，failed→verify_failed |

Service 层复用：诊断合并直接调 `s.AIRepair`（已有）；批量 resolve 复用 `Resolve()`（自带状态机/通知/日志）；日志统一走 `feedbackRepo.AddLog` + 任务自身 log_text。

### M3 本机执行端（新增脚本，不改服务端行为）

`scripts/repair-agent.ps1`（或小 Go 程序放 cmd/repair-agent）：
1. 轮询 `GET /api/v1/internal/repair-tasks/next`（环境变量 WXX_REPAIR_TOKEN，专用账号）；
2. 认领后在本机仓库 `git worktree add ../wxx-repair-{task_no} -b repair/{task_no}`（隔离工作区）；
3. 打印诊断报告（复用 Markdown 格式），提示操作者在工作区内人工/AI CLI 完成修改；
4. 自动验证：`go vet ./... && go test ./...` + `cd frontend && flutter analyze && flutter test`（全部复用 Makefile 同款命令），收集通过与否；
5. `git diff --stat` 收集 diff_stat；POST /verify 上报；
6. 结束语：**不在本机做任何部署**；部署由管理员走既有通道（GitHub Actions deploy-backend 手动触发 / `make deploy-release`），完成后回管理台点 deploy-done。

### M4 前端（最小增量）

- `api_config.dart`：+9 个任务端点常量。
- `feedback_provider.dart`：createRepairTask(List ids)/fetchRepairTasks/pollRepairTask/accept/reject/deployConfirm/deployDone。
- `pages/admin/feedback_page.dart`：列表加 Checkbox 多选 + 底部「创建修复任务(N)」按钮；新增简单任务列表/详情视图（可先做成同文件内 Widget 或独立 repair_tasks_page.dart），生命周期徽章 + 验证结果展开 + 验收/驳回/部署确认按钮（按 capability 显示，后端二次校验）。
- 顺手修 G8：resolved 状态隐藏「验证通过并标记已解决」。

### M5 顺序验收口径

1. 管理员在列表勾选 1~N 条反馈 → 创建任务（status=approved，feedback_logs 留痕）；
2. 本机执行端认领 → running（worker_host/base_commit 落库）；
3. 本地改码 + 四件套验证自动执行 → passed 上报 → awaiting_acceptance（或 failed→verify_failed→重新认领）；
4. 另一管理员（或同人在明确知悉下）点验收 → deploy_pending；
5. 人工执行既有部署（GH Actions 或 make deploy-release）→ 点部署完成 → closed，关联反馈批量 resolved + 站内信通知提交者；
6. 全程 feedback_logs + 任务 log_text 可追溯。

预估工作量：迁移+后端约 600-800 行（含测试），脚本 ~150 行，前端 ~500 行；不动现有反馈主流程，风险可控。

---

## 十二、风险清单

| # | 风险 | 缓解 |
|---|---|---|
| R1 | 执行端 token 泄漏 → 可拉取反馈内容/代码结构信息 | 专用最小能力 system.repair.execute；payload 不含截图；token 可轮换；HTTPS-only |
| R2 | 验收人=创建人=执行人同一人，闸门形同虚设 | MVP 至少强制验收时二次弹窗+记录；后续可加"验收人≠执行人"硬校验 |
| R3 | 自动验证四件套无法覆盖运行时回归（FTS、地图、语音等集成问题） | 验收步骤保留人工页面复现；部署后用公开 /health + eval-gate 抽查 |
| R4 | 并行任务互踩（两人同时改同一文件） | MVP 全局仅允许 1 个 running 任务（认领接口原子化）；worktree+分支隔离兜底 |
| R5 | 生产库已是 MySQL：迁移若沿用 SQLite 方言会炸 | 参考 028/AvgResolveHours 的双方言写法；启动自动迁移前先在本地 MySQL 演练 |
| R6 | 部署本身仍全人工，"deployed"标记可能与真实发布不同步 | deploy-done 表单要求填 deploy_ref（GH run ID/命令摘要）；Caddy 侧版本号比对可后续加 |
| R7 | 回滚路径依赖既有运维习惯（wxx-server.rollback.* / mysqldump 备份） | 把"部署前备份+回滚副本"写入 deploy-confirm 的提示文案；分支保留至稳定后再删 |
| R8 | 每次打开修复面板就打一次 LLM + 落一条 job，成本/噪声随任务化被放大 | 任务创建时才做诊断；面板浏览改为读取已存 diagnosis（G7 一并修） |
| R9 | 反馈越权读（G1）在任务链路上被间接放大（任务详情含反馈聚合文本） | 先修 G1（Get/GetLogs 加 owner-or-capability 校验）再上线任务功能 |
| R10 | Windows 开发机中文路径导致 Flutter 构建失败（Makefile 已注明 impellerc 限制） | 执行端脚本直接复用 `-safe` 构建目标，不自造构建命令 |

---

## 十三、只读声明

本轮审计仅执行了文件读取、目录列举与 git 只读查询；未修改任何源码、配置或数据；除覆盖本文件 `pm-checklist.md` 外未触碰仓库（该文件本身已被 git 跟踪，旧版可随时恢复）。
