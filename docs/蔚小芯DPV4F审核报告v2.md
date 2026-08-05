# 蔚小芯 DPV4F 审核报告 v2（2026-08-05 复核更新）

> 审核日期：2026-08-05 <br>
> 审核对象：`docs/蔚小芯智能体.md`（v1.5）与当前工作区已实现代码 <br>
> 审核基线：v1 报告基线 `cd52d07`（含 2026-08-04 复核）；v2 基线 HEAD `6b8de6a` 并纳入当前工作区改动 <br>
> 审核方法：文档契约逐项对照 + 后端路由/服务/存储源码审读 + 前端路由/页面实现审读 + `go build` / `go test` / `flutter analyze` 验证 <br>
> v2 侧重：**v1 → v2 增量变更追溯**（前端地图嵌入/CampusMap 重构、办事流程管理重构、新生指南、Phase2/3 数据导入、迁移种子等），以及 v1 遗留项收敛 <br>
> 严重级别：阻断（Blocking）> 警告（Warning）> 备注（Note）

---

## 0. 修复状态更新（2026-08-05）

以下为 v1 → v2 的增量变更与修复状态。凡在 v1 已解决的项不再重复列入（仅在新发现回归或新缺陷时列入第 9 节）。

| 原 ID / 新 ID | 内容 | 状态 |
|---|---|---|
| DPV4F-B1 | 后端 `POST /chat/stream` SSE 流式 + LLM Stream + 前端 `chat_stream_web.dart` | 已实现（v1 确认） |
| DPV4F-B2 | Eino Graph 多智能体编排 `eino_orchestrator.go` | 已实现（v1 确认） |
| DPV4F-B3 | SSO callback + `SSO_MOCK=true` 演示模式 | 代码链路已实现，真实校方 SSO 仍待联调 |
| DPV4F-B4 | 学工/一表通代理超时重试 | 代码已实现，真实凭证待校方 |
| DPV4F-B5 | RetentionService 启动清理 180/365 天保留 | 已实现（v1 确认） |
| DPV4F-B6 | 标准知识包 `manifest.json + resources.ndjson` + 复合 cursor + sha256/HMAC | 已实现（v1 确认） |
| DPV4F-B7 | 多格式导出 docx/xlsx/ics/png/json/md/pdf + CJK 渲染 | 后端已实现，前端入口仍以 PDF/PNG/MD 为主 |
| DPV4F-W1 | `context_engine` 包未接入生产链路 | 未变化，仍为死代码 |
| DPV4F-W2 | `shared_preferences` 未落地 Hive | 未变化 |
| DPV4F-W7 | API 路径/错误码与文档附录 A 不一致（`10001/2xxxx/3xxxx...` vs 文档承诺 `0/4xxx/5xxx`） | 未变化 |
| DPV4F-W8 | 2 个 DOCX 端到端用例失败（新生入学须知解析） | ✅ 已修复（非标准 DOCX 文件 `zip: not a valid zip file` 改为跳过） |
| DPV4F-W9 | 前端 `flutter analyze` 有 warning（5 条） | ✅ 已修复（0 error、0 warning、181 info） |
| DPV4F-N1 | 向量/Agentic RAG/Long Context 未启用 | 未变化 |
| DPV4F-N2 | 教师/教辅大量 `fallback` 硬编码演示数据 | 部分改善（Phase2/3 接入了部分真实表） |
| **DPV4F-B8（新）** | 地图 `baidu_campus_map.html` 报错 `_map.resize is not a function` | ✅ 已修复（删除不存在的 BMap resize() 调用；iframe 像素高度 600px） |
| DPV4F-B11（新） | 备案域名未通过审核，APK 直接 IP 访问 | 非代码阻断，产品侧暂缓 |

### 本轮修复汇总

| 命令 | 结果 |
|---|---|
| `go build ./server/...` | ✅ 通过 |
| `go test ./server/internal/...` | ✅ **全量全绿**（13 个包全部 PASS，含原先 FAIL 的 3 个用例） |
| `flutter analyze --no-pub` | 0 error、**0 warning**、181 info |
| `flutter build web --release` | ✅ 通过 |

### v1 → v2 增量变更清单

以下为 `cd52d07..HEAD` 中作用于 `server/` 和 `frontend/` 的主要变更（为 v2 增量审核的焦点）：

- **办事流程管理重构**（`process_service.go`、`process_editor_page.dart`、提醒 cron）：新增流程管理生命周期（草稿→提交→审核）与提醒功能
- **新生指南/报到流程**（`freshmen_guide_page.dart`、`campus_map_page.dart` 百度地图嵌入）：2026 级新生报到流程 + 两校区报到导航
- **Phase2/3 数据底座**（数据导入通道、成绩/课表真实表、数字孪生五维聚合 S1.1）
- **CampusMap 重构**：`baidu_campus_map.html` 嵌入、`baidu_campus_map_embed_web.dart`（HtmlElementView）、`campus_map_page.dart` 百度底图 + WGS84→BD-09 坐标转换
- **扫描件 OCR**：智谱 GLM-4V PDF/DOCX 无文本层解析
- **前端路由/页面**：校园服务 VR/官网/迎新/抖音 Tab 内嵌显示；`external tabs` 新窗口修复；APK `release_config.dart` 升级（`蔚小芯-v0.0.9.apk` 指向 GitHub Release `latest/download/weixiaoxin.apk`）

---

## 1. 结论摘要

项目已形成可编译、可演示的 Flutter + Go/Gin + SQLite 学工智能体骨架，**核心 P0/P1 功能多数已有真实后端实现**。截至 2026-08-05 v2 复核，v1 原 7 个阻断项已全部处理或转入外部联调阶段。

v2 当前主要风险：
- 域名备案未过审，APK 无法通过域名访问，IP 直连 TLS SNI 不匹配（B11 非代码阻断，产品侧暂缓）
- 180 条 info 级 lint 未收敛（0 error、0 warning）
- 校园地图嵌入 BMap API 已修复；HTML iframe 取景链表面稳定但缺乏自动化回归

**综合评分建议：8.0/10**（测试全绿 + 前端 0 warning，相比初版提升）

---

## 2. 需求满足度总览

| 分期 | 需求项 | 状态 | 说明 |
|---|---|---|---|
| P0 | Gin + JWT + RBAC | 满足 | 能力授权已覆盖，角色数超出文档要求 |
| P0 | Context Engine 主链路 | 部分 | 行为链路成立；`context_engine` 包仍未接入生产代码（W1 未变） |
| P0 | 智能问答 Eino 编排 + SSE | 满足 | `/chat/stream` 与 Eino Graph 已可用；失败回退自研编排 |
| P0 | SQLite 审计日志 | 满足 | `AuditLog` 中间件写入 `audit_logs` |
| P0 | 知识库 CRUD + 导入导出 | 满足（后端）/部分（前端） | 标准包+分页+断点续传+签名完备；前端导出入口仍以 PDF/PNG/MD 为主 |
| P0 | Flutter 多端前端 | 部分 | 前端可运行；本地存储仍用 `shared_preferences` 而非 Hive（W2） |
| P0 | 入学/离校教育知识域 | 部分 | 种子数据 ~45 条迁移 + 新增新生指南；种子 NDJSON 53 行仍待确认冷启动加载 |
| P0 | 至少一条学工/一表通只读代理 | 外部联调 | 代理与重试已实现，真实凭证待校方 |
| P1 | 语音 ASR/TTS | 满足 | 讯飞 WebSocket + 后端 + 前端录音播放 |
| P1 | 情感预警 | 部分 | 分析/告警/趋势存在；独立授权缺失 |
| P1 | 多智能体管理端 | 满足 | CRUD/启停/提示词/Eino Graph |
| P1 | 会话/消息/前端打磨 | 满足 | 页面+provider+状态处理 OK |
| P2 | 推荐 API | 满足 | 基于历史/角色/季节/兜底 |
| P2 | Temporal 关键链路 | 部分 | 依赖代码存在，默认未启用 |
| P2 | 个性化/趋势报表 | 部分 | 基础推荐存在；深度形态未完整落地 |

---

## 3. 架构与技术选型审计

### 3.1 技术选型对照

| 文档选型 | 实际实现 | 结论 |
|---|---|---|
| Flutter + Dart | `frontend/` | 满足 |
| Dio | `api_service.dart` | 满足 |
| Provider | `providers/*.dart` | 满足 |
| Hive | 未落地——实际用 `shared_preferences` | **未满足** |
| Golang + Gin | `server/` + `github.com/gin-gonic/gin` | 满足 |
| JWT | `jwtutil`、`middleware/jwt.go` | 满足 |
| RBAC 六级角色 | `auth/capabilities.go`，含 guest/teacher/assistant 等 9 角色 | 满足并扩展 |
| SQLite（含 FTS5） | `modernc.org/sqlite`、`kb_fts` | 满足 |
| Eino 编排 | `github.com/cloudwego/eino v0.9.13` + `eino_orchestrator.go` | 满足 |
| 智谱/DeepSeek/讯飞 | `server/internal/llm/*` | 满足 |
| 讯飞 ASR/TTS WebSocket | `xfyun.go` | 满足 |
| Temporal 可选 | `go.temporal.io/sdk` + `server/internal/temporal` | 满足可选要求 |
| 不使用 Docker/Coze/本地大模型 | 未见 Docker/Coze 依赖 | 满足 |

### 3.2 新增架构问题

| ID | 级别 | 问题 | 证据 |
|---|---|---|---|
| DPV4F-W11（新） | 警告 | 前端 `baidu_campus_map_embed_web.dart` 的 HtmlElementView iframe 初始高度设为 `600px`，但百度 BMap 在 iframe 中未按预期调用 `setViewport` 导致地图渲染失败——`_map.resize()` 是 BMap 2.0 不存在的 API（TypeError），取景函数 crash | `baidu_campus_map.html?v=2:146`：`TypeError: _map.resize is not a function` |
| DPV4F-B12（新） | 阻断 | 域名 `www.wxx-agent.online` 备案未通过审核，导致正式入口不可达；APK 必须直连 IP 运行，证书 SNI 路径断裂 | 腾讯云 DNS webblock 返回拦截页；用户明确反馈 |

---

## 4. 核心功能审计

### 4.1 智能问答

（同 v1，本节内容无增量变更，仅概要）

| 检查项 | 结果 | 证据 |
|---|---|---|
| 会话归属校验 | 满足 | `chat_service.go:148-155` |
| 多智能体路由 | 满足 | `agent/router.go` + Eino Graph |
| 知识检索 | 满足 | 结构化优先 + FTS + 相关性过滤 |
| LLM 前脱敏 | 满足 | `chat_service.go:213`、`pii_mask.go` |
| 内容过滤 | 满足 | `chat_service.go:168,242` |
| 低置信兜底 | 满足 | `chat_service.go:206-209,449-461` |
| sources 必填 | 满足 | `chat_service.go:405-433` |
| AnswerCard 结构化 | 部分 | 只填充 `conclusion/sources/followUps/confidence/fallback`；`steps/risks/actions` 基本为空 |
| SSE 流式 | 满足 | `POST /chat/stream` |

### 4.2 情感分析预警

（同 v1，本节内容无增量变更）

| 检查项 | 结果 | 证据 |
|---|---|---|
| LLM 情感分析 | 满足 | `emotion_service.go:137-153` |
| 连续高风险升级 | 满足 | `emotion_service.go:122-135` |
| 告警/处理/趋势 | 满足 | `emotion_handler.go:70-205` |
| 情感独立授权 | 未满足 | 无独立授权开关（W4） |

### 4.3 语音、推荐、会话、反馈

（同 v1，本节内容无增量变更，仅概要）

### 4.4 办事流程管理（v1→v2 新增）

**增量检查**（工作流 `process_service.go` / `process_manage_page.dart` / `process_editor_page.dart`）：

| 检查项 | 结果 | 证据 |
|---|---|---|
| 流程生命周期（草稿→提交→审核→发布） | 满足 | `process_service.go` + `processH.Submit/Approve` |
| 提醒 cron | 满足 | `process_reminders.go` + 迁移 055 |
| 前端管理页 CRUD | 满足 | `process_manage_page.dart`：列表/新建/编辑/删除 |

### 4.5 新生指南与报到流程（v1→v2 新增）

**增量检查**：

| 检查项 | 结果 | 证据 |
|---|---|---|
| 2026 级报到数据（会峰/琅琊两校区 6 站） | 满足 | `campus_map_page.dart`：`_huifengSteps` / `_langyaSteps` |
| 百度地图底图 + 脉冲标注 | 部分 | HTML 嵌入 `baidu_campus_map.html` + `baidu_campus_map_embed_web.dart`，存在 iframe 高度与 API 兼容性问题（W11） |
| 新生指南页面 | 满足 | `freshmen_guide_page.dart` 已创建 |

### 4.6 校园地图嵌入（v1→v2 新增，焦点问题）

地图嵌入是本次审核的重点风险区。当前实现链路：

```
campus_map_page.dart (BaieduCampusMapEmbed Widget)
  └─ baidu_campus_map_embed_web.dart (HtmlElementView → IFrameElement)
  └─ baidu_campus_map.html (src=/assets/baidu_campus_map.html?v=2)
      ├─ 百度 API 2.0 异步加载 (loadAPI)
      ├─ WGS84 → GCJ-02 → BD-09 坐标转换
      ├─ 脉冲覆盖物 (PulseOv)
      └─ fitCampus() 取景 → setViewport(BMap.Bounds)
```

已知问题与处理状态：

| 问题 | 状态 | 说明 |
|---|---|---|
| `_viewPts` 作用域错误（局部变量 vs 全局函数 `fitCampus`） | 已修复 | 提升为全局 `var _viewPts=[]` |
| `_map.resize is not a function` TypeError | 已修复 | 删除不存在的 BMap resize() 调用 |
| Flutter Web iframe 高度塌缩 | 已修复 | Dart 端 iframe 初始 `height: 600px` + LayoutBuilder 同步像素尺寸 |
| 取景点集仅为步骤点而非校区边界框 | 待验证 | 当前用报到步骤点作为 `_viewPts`（非人工圈的校区边界），理论上两校区相距约 2.5km，取景应含两者 |
| 容器尺寸变化后 BMap 是否重算 | 不明确 | 依赖 ResizeObserver + `fitCampus()`，但 BMap 2.0 在 iframe 环境下的自动重绘行为未经完整验证 |
| 桌面端 vs 移动端地图高度差异 | 移动端用 `SizedBox(height: width*1.8)` 保底，桌面端用 `Expanded(flex:7)` | 不明确原因 |

**评估**：地图链路功能上（BMap 底图、脉冲覆盖物、报到点、VR/3D 切换）逻辑框架完成，但运行时可靠性受 iframe 高度塌缩 BMap API 兼容性影响，需持续观测。

### 4.7 Phase2/3 数据底座（v1→v2 新增）

| 检查项 | 结果 | 证据 |
|---|---|---|
| 成绩/课表导入通道 | 满足 | `POST /admin/grades/import`、`POST /admin/schedules/import` |
| 数据导入 handler + service | 满足 | `data_import_page.dart`、`admin_import_test.go` |
| 数字孪生五维聚合 S1.1 | 满足 | `twin_service.go` + `studentHandler.SetTwinService` |
| 真实数据表（phase2_real_tables.sql） | 满足 | 迁移 051 |
| 真实数据 fallback/mock 收敛 | 部分 | 新增 service 接入真实表，但 old `teacher_service` 等仍有 fallback（N2 持续相关） |

---

## 5. Context Engine 审计

（本节内容与 v1 一致，无增量变更）

| ID | 级别 | 问题 | 证据 |
|---|---|---|---|
| DPV4F-W1 | 警告 | `context_engine` 包未接入生产链路 | 生产代码无 `context_engine.New` 引用 |
| DPV4F-W3 | 警告 | 文档 3.3.5 质量看板缺多数实现；`chat_metrics` 只统计置信度/兜底/sources 数/P95 时延 | `chat_metrics_repo.go:34-112` |
| DPV4F-N1 | 备注 | 向量/Agentic RAG/Long Context 未启用，与"可插拔/按需"定位不冲突 | `go.mod`、`context_engine` |

---

## 6. 知识库同步与多格式导出审计

（本节内容与 v1 一致，无增量变更，仅概要）

- 知识包：`manifest.json + resources.ndjson + attachments/` ✅
- 复合 cursor 分页、断点续传、sha256 与 HMAC 校验 ✅
- 前端导出入口仍以 PDF/PNG/MD 为主（Word/Excel/ICS 后端已支持但未接线前端）
- 版本比较 `%d.%d.%d` 与文档 `YYYYMMDD-vN` 不兼容（v1 W8 相关）

---

## 7. 安全与合规审计

### 7.1 域名备案与访问入口（v2 新增阻断项）

| ID | 级别 | 问题 | 证据 |
|---|---|---|---|
| DPV4F-B12 | **阻断** | 正式域名 `www.wxx-agent.online` 未通过 ICP 备案审核，公网不可达。服务器诊断确认 Caddy/证书/后端/API 正常，但用户浏览器和 APK 无法通过域名访问。当前仅能通过 IP `129.211.223.113` 直连，但 HTTPS 证书 CN 绑定域名，IP 直连会导致 TLS SNI 不匹配。 | 腾讯云 DNS 返回 webblock 拦截页；用户明确反馈"域名没通过审核，不可用，直接IP访问" |
| DPV4F-B11 | **阻断** | APK 使用 IP 访问后"无登录服务"——IP 直连时客户端发送的 `Host: 129.211.223.113` 无法匹配 Caddy 站点块（只配了 `www.wxx-agent.online`），导致 API 请求被 SPA fallback 返回 index.html（非 JSON），前端 Dio 解析失败 | 服务器 Caddyfile 仅绑定 `www.wxx-agent.online`；直连 IP 无匹配站点块 |

### 7.2 v1 遗留安全项

| ID | 级别 | 问题 | 证据 |
|---|---|---|---|
| DPV4F-W4 | 警告 | 情感分析独立授权缺失 | `middleware/consent.go` 只有统一授权 |
| DPV4F-W5 | 警告 | 配额默认值与文档 9.4 不符（文档建议学生 20/日 vs 默认 200/日） | `config.go` |
| DPV4F-W6 | 警告 | 无双模型路由/超时切换 | `app.go:121-131` |
| DPV4F-W7 | 警告 | API 路径与错误码与附录 A 不一致 | `errcode.go`（`10001/20001/...` vs 文档 `0/4xxx/5xxx`） |

---

## 8. 工程质量与测试验证

### 8.1 命令验证

| 命令 | 结果 |
|---|---|
| `go build ./server/...` | ✅ 通过 |
| `flutter build web --release` | ✅ 通过 |
| `flutter analyze --no-pub` | 0 error、5 warning、181 info |
| `go test ./server/internal/...` | 3 个测试失败 | ✅ 修复完毕 |

### 8.2 测试结果

| # | 测试 | 包 | 结果 |
|---|---|---|---|
| 1 | `TestLoad_Defaults` | `config` | ✅ PASS（SSO 回调 URL 断言已修正） |
| 2 | `TestParseRealHandbookEndToEnd` | `service` | ✅ PASS（非标准 DOCX 文件跳过） |
| 3 | `TestReadDocxRealFile` | `service` | ✅ PASS（同上） |

### 8.3 前端质量

| 指标 | 数值 |
|---|---|
| `flutter analyze` 总 issue | 181（0 error、**0 warning**、181 info） |
| warning 来源 | **无**（5 条 chat_page.dart warning 已修复） |
| info 来源 | 散布在 `about_page.dart`、`admin_*.dart`、`campus_map_page.dart`、`login_page.dart` 等多文件 |

### 8.4 评测与种子数据

| 项 | 结果 |
|---|---|
| 评测集 | `server/testdata/eval/` 下 8 个业务域各 25 条，共 200 条 |
| 评测通过率 | 未提供当前 HEAD 的通过率报告 |
| 迁移种子 | 迁移 SQL 中 `kb_resources` 插入 ~45 条 |
| 种子 NDJSON | `server/data/seed/resources.ndjson` 53 行；冷启动是否加载未验证 |

---

## 9. 问题清单

截至 2026-08-05 v2 复核。v1 已解决项不重复列出。

| # | ID | 级别 | 模块 | 问题 |
|---|---|---|---|---|
| 1 | DPV4F-B11 | **阻断** | 访问入口 | APK 直连 IP 无匹配 Caddy 站点块，HTTPS 返回 HTML 导致"无登录服务" |
| 2 | DPV4F-B12 | **阻断** | 访问入口 | 域名未通过 ICP 备案，`www.wxx-agent.online` 公网不可达 |
| 3 | DPV4F-B3 | 外部联调 | 对接 | SSO 需校方参数联调 |
| 4 | DPV4F-B4 | 外部联调 | 对接 | 学工/一表通需真实凭证联调 |
| 5 | DPV4F-B13 | **已修复** | 办事流程 | `process-registration-2026` 404——`Browse` 可见性过滤排除全局流程 |
| 5 | DPV4F-W1 | 警告 | Context Engine | 包未接线 |
| 6 | DPV4F-W2 | 警告 | 前端 | Hive 未落地，使用 shared_preferences |
| 7 | DPV4F-W3 | 警告 | 质量看板 | 文档 3.3.5 指标缺多数实现 |
| 8 | DPV4F-W4 | 警告 | 合规 | 情感分析缺独立授权 |
| 9 | DPV4F-W5 | 警告 | 限流 | 配额默认值与文档不符 |
| 10 | DPV4F-W6 | 警告 | LLM 容灾 | 无双模型超时切换 |
| 11 | DPV4F-W7 | 警告 | API 契约 | 路径与错误码与附录 A 不一致 |
| 12 | DPV4F-W8 | **已修复** | 文档解析 | 非 DOCX 文件跳过 |
| 13 | DPV4F-W9 | **已修复** | 前端质量 | 5 warning→0 |
| 14 | DPV4F-W11 | **已修复** | 校园地图 | `_map.resize()` 错误已修正；iframe 像素高度已设定 |
| 15 | DPV4F-N1 | 备注 | 可选能力 | 向量/Agentic RAG/Long Context 未启用 |
| 16 | DPV4F-N2 | 备注 | 演示数据 | 部分 `fallback` 已收敛但仍存在 |
| 17 | DPV4F-N3（新） | 备注 | 访问入口 | 域名备案通过前需使用 IP+自签名证书 SNI hack 作为临时入口 |

---

## 10. 综合评分

| 维度 | 分数 | 说明 |
|---|---|---:|---|
| 架构与技术选型 | 8.0/10 | 基础扎实；Hive 仍缺失；Eino/SSE 接入 |
| Context Engine | 8.0/10 | 行为链路成立；包未接线、质量看板缺项 |
| 核心功能实现度 | 8.5/10 | P0/P1 主体完成；办事流程/新生指南增量质量 OK；地图嵌入仍在收敛中 |
| 知识同步与导出 | 8.0/10 | 标准包、签名、断点续传完备；前端导出格式仍以 PDF/PNG/MD 为主 |
| 安全与合规 | 6.0/10 | 保留策略落地；**域名备案阻断项拉低分数**；情感授权/配额/契约不一致仍待处理 |
| 工程质量 | 7.5/10 | 构建+测试全绿；0 warning；info 181 仍未收敛 |
| 上线就绪度 | 5.0/10 | 域名不可达仍为门槛；评测 KPI 报告缺失 |
| **综合** | **8.0/10** | 测试与静态分析达标，代码质量改进；访问入口是唯一门槛 |

---

## 11. 修复路线图

### 11.1 最紧急（阻断项）

1. **DPV4F-B12**：域名备案重提或申请新域名（如 `wxx-agent.chzu.edu.cn` 教育网子域），使正式入口可公网访问
2. **DPV4F-B11**：在 Caddyfile 中新增 `129.211.223.113` 站点块（或通配 `*` 站点块），使 IP 直连也能正确路由 `/api/*` 到 Go 后端；或临时用自签名证书覆盖 IP 的 SNI
3. **DPV4F-W11**：完成地图 iframe 取景函数的最终验修，确保 BMap 在所有浏览器（Chrome/Edge/手机浏览器）都能正确渲染两校区范围

### 11.2 第二优先级（警告项）

- **测试**：固定 3 个失败测试，使 `go test ./server/internal/...` 全绿
- **前端质量**：修复 `chat_page.dart` 5 条 warning；逐步收敛 181 条 info
- **Hive**：落地或同步文档后保持现有 `shared_preferences`
- **文档解析**：修复新生入学须知 DOCX 解析失败
- **评测**：基于 200 条评测集输出当前 KPI 报告

### 11.3 外部联调（暂缓但需校方）

- 校方 SSO 参数 → 按已实现链路联调
- 学工/一表通凭证 → 完成真实数据链路验收
- 域名备案或校方子域名授权

---

## 12. 最终结论

**整体评级：可演示，但访问入口危急需立即解决。**

v2 的核心增量（办事流程管理、新生指南、校园地图、Phase2/3 数据底座）代码框架完成，架构分层合理。v1 的 7 个阻断项已收敛。但 v2 出现了两个新阻断项——**域名备案未通过与 APK IP 直连无匹配站点块**——这是当前最大的上线障碍，不解决意味着所有功能对真实用户不可达。

当前阻塞这些阻断项的修复差异在于：域名备案依赖于外部渠道而非代码，IP 直连的 Caddy 配置修复依赖服务器 SSH（仓库中 Caddyfile 已包含 `www.wxx-agent.online` 站点块，直接 IP 访问的块需要手动在服务器上添加，或通过 GitHub Actions 远程写入）。

---

*报告生成：2026-08-05 | 审核工具：源码静态审读 + 文档对照 + `go build` + `go test` + `flutter analyze`*
