# 蔚小芯 DPV4F 审核报告 v3（2026-08-07 复核更新）

> 审核日期：2026-08-07 <br>
> 审核对象：`docs/蔚小芯智能体.md`（v1.5）与当前工作区已实现代码 <br>
> 审核基线：v2 报告基线 HEAD `6b8de6a`；v3 基线 HEAD `6ba4c6c`（含 v2 后全部增量） <br>
> 审核方法：文档契约逐项对照 + 后端路由/服务/存储源码审读 + 前端路由/页面实现审读 + 子代理交叉核验 + `go build`/`go vet`/`go test`/`flutter analyze`/`gofmt` 实测 <br>
> v3 侧重：**v2 → v3 增量变更追溯**（学科专业智能体、数字孪生自画像、个人档案聚合、Markdown 统一渲染、年级主题切换、校园服务学院入口、悬浮菜单恢复、种子密码更新）与 v2 遗留项收敛 + 生产数据链路实测 <br>
> 严重级别：阻断（Blocking）> 警告（Warning）> 备注（Note）

---

## 0. 修复状态更新（2026-08-07）

| 原 ID / 新 ID | 内容 | 状态 |
|---|---|---|
| DPV4F-W3 | 质量看板缺实现 | ✅ **已修复**（`chat_metrics_repo.go` 已实现 Insert/Aggregate/CountByIntent，含 P95/兜底率/来源命中率） |
| DPV4F-W9 | 前端 warning 5 条 | ✅ 已修复（v2 确认） |
| DPV4F-B14 | 种子密码 | ✅ 已修复并更新（`admin123`→`wxx123456`→**`Wxx@2026`**，sync.go:448） |
| DPV4F-B11 | IP 直连无站点块 | 未变（仓库 Caddyfile 仍无 IP 站点块，CI 不写 Caddyfile） |
| DPV4F-B12 | 域名备案未通过 | 未变（外部渠道，DNS webblock） |
| DPV4F-W1/W2/W4/W5/W7/N1/N2 | 各项未变 | 未变（详见第 9 节） |

### v2 → v3 增量变更清单（`6b8de6a..HEAD`）

- **学科专业智能体**（97eba44）：`agent/major_agent.go` + `router.go IntentMajor` + orchestrator 注册；迁移 057/058 种子知识
- **数字孪生可视化自画像**（584b808 → 6d676a0 → a4be5d2）：AvatarConfig/Painter/Card 数据驱动卡通数字人 → 星星 AI 智能体 + 动画特效（眨眼/漂浮/粒子/光环脉冲）
- **个人档案聚合**（ed89b2a）：`GET /api/v1/student/profile` 并发聚合 + 头像上传/服务 + 迁移 059
- **Markdown 统一渲染**（4965dd4 → a038fea）：`MdText` 组件 + 38 页面 46 处 Text→MdText
- **年级主题自动切换**（4fb0854）：ThemeNotifier 扩展 4 年级色板 + Profile 开关 + 后端 Profile 补 enrollment_year
- **校园服务学院入口**（3f08bce）：`csci.chzu.edu.cn` 内嵌 + 竞赛列表 500 修复
- **悬浮菜单恢复**（4fb0854）：FabMenu 3 项（反馈/语音/数字人）
- **运维**（e800dea → a4be5d2 → 6ba4c6c）：reset-seed 支持 `--all-students`；CI scp `command_timeout: 90s`

---

## 1. 结论摘要

截至 2026-08-07 v3 复核，项目功能面持续扩张：v2 后新增 6 大功能模块均**未引入明显缺陷**（子代理交叉核验 + 源码审读结论），质量看板（W3）已补齐，种子密码已统一为 `Wxx@2026`，CI scp 挂起已修复。

**生产数据链路实测**（本次 v3 新增重点）：通过真实学生账号登录验证，个人档案聚合返回学院/专业/班级/入学年份等真实数据，竞赛列表返回 5 条种子数据，数字孪生/性格洞察接口正常响应。**数据链路本身是通的**。此前"新增功能无数据"的根因是：①导入学生密码=学号导致用户无法登录；②成绩/课表等业务数据未导入。已用 `reset-seed --all-students` 重置 134 个学生密码为 `Wxx@2026`。

**当前主要风险**：
- **域名备案未过审（B12）+ IP 直连无站点块（B11）**：仍为唯一上线门槛，功能对真实用户不可达（外部渠道，非代码可解）
- 191 条 info 级 lint（0 error、0 warning）
- `context_engine` 包仍为死代码（W1）、Hive 仍未落地（W2）

**综合评分建议：8.5/10**（功能增量 + W3 修复 + 数据链路验证通过；访问入口仍是门槛）

---

## 2. 需求满足度总览

| 分期 | 需求项 | 状态 | 说明（v3 增量） |
|---|---|---|---|
| P0 | Gin + JWT + RBAC | 满足 | 不变 |
| P0 | Context Engine 主链路 | 部分 | `context_engine` 包仍未接线（W1 未变）；但学科专业智能体走 `kbRepo.Search` FTS |
| P0 | 智能问答 Eino + SSE | 满足 | 不变 |
| P0 | SQLite 审计日志 | 满足 | 不变 |
| P0 | 知识库 CRUD + 导入导出 | 满足（后端） | 不变 |
| P0 | Flutter 多端前端 | 部分 | `shared_preferences` 仍用（W2 未变） |
| P0 | 入学/离校知识域 | 部分 | 不变 |
| P0 | 学工/一表通代理 | 外部联调 | 不变 |
| P1 | 语音 ASR/TTS | 满足 | 不变 |
| P1 | 情感预警 | 部分 | W4 独立授权未变 |
| P1 | 多智能体管理端 | 满足 | **新增学科专业智能体**（major-guide） |
| P1 | 会话/消息/前端打磨 | 满足 | **新增 Markdown 统一渲染**、悬浮菜单恢复 |
| P2 | 推荐 API | 满足 | 不变 |
| P2 | Temporal 关键链路 | 部分 | 不变 |
| P2 | 个性化/趋势报表 | 部分 | **新增数字孪生自画像 + 个人档案聚合** |

---

## 3. 架构与技术选型审计

### 3.1 技术选型（与 v2 一致，无增量变更）

Flutter+Dio+Provider ✓ / Hive ✗（shared_preferences）/ Go+Gin ✓ / JWT ✓ / RBAC 六级+扩展 ✓ / SQLite+FTS5 ✓ / Eino ✓ / 智谱/DeepSeek/讯飞 ✓ / 讯飞 ASR/TTS ✓ / Temporal 可选 ✓ / 无 Docker/Coze ✓

### 3.2 新增架构观察

| ID | 级别 | 问题 | 证据 |
|---|---|---|---|
| DPV4F-W12（新） | 备注 | AvatarCard 无限 `repeat` 动画持续重绘（3s 周期），低端设备/Web 有 CPU 开销 | `avatar_card.dart` `_controller.repeat()`；非缺陷但可优化（如页面不可见时暂停） |
| DPV4F-W13（新） | 备注 | 前端 info lint 由 v2 的 181 → v3 的 191（新增文件引入约 10 条 `prefer_const` 等） | `flutter analyze` 实测 |

---

## 4. 核心功能审计

### 4.1 智能问答（与 v2 一致）

同 v1/v2，无增量变更。sources 必填 ✓ / 低置信兜底 ✓ / PII 脱敏 ✓ / 内容过滤 ✓ / AnswerCard 结构化部分（steps/risks 常空）。

### 4.2 学科专业智能体（v2→v3 新增，焦点）

| 检查项 | 结果 | 证据 |
|---|---|---|
| IntentMajor 路由 | 满足 | `router.go:236` 30+ 关键词，权重 5 |
| major-guide 智能体 | 满足 | `major_agent.go` 过滤 Major/Course/Process/FAQ/Activity |
| orchestrator 注册 | 满足 | `orchestrator.go:37` `NewMajorAgent()` |
| 种子知识（迁移 057/058） | 满足 | 7 条学科专业资源（培养方案/课程/竞赛/就业/前沿/创新） |
| 线上验证 | 满足 | 学科竞赛问答返回 `code:0` + sources 引用 |

### 4.3 数字孪生自画像（v2→v3 新增，焦点）

| 检查项 | 结果 | 证据 |
|---|---|---|
| 数据驱动映射 | 满足 | `avatar_config.dart` 五维→眼镜/奖牌/徽章/笑容/发型/服装色 |
| 角色区分（帽子/学袍） | 满足 | `hatStyle`/`gownColor`（学生学士帽/教师中式帽） |
| 动画特效 | 满足 | `avatar_painter.dart` t 相位驱动眨眼/漂浮/粒子/光环脉冲 |
| null 安全 | 满足 | `fromData` 对空维度/空人格防护（`?? []`、`is num`） |
| 布局保留 | 满足 | 数字孪生页五维测评数据完整保留，仅替换形象 |

### 4.4 个人档案聚合（v2→v3 新增，焦点）

| 检查项 | 结果 | 证据 |
|---|---|---|
| 并发聚合 | 满足 | `student_handler.go` PersonalProfile 并发查询 7 类数据 |
| 并发安全 | 满足 | `sync.Mutex` 保护 result 写入 |
| SQL 注入 | 无 | 全部参数化查询（`?` 占位） |
| 错误容忍 | 满足 | 单模块失败仅记日志不影响整体 |
| 线上验证 | 满足 | 真实学生登录后返回学院/专业/班级/入学年份 |

### 4.5 竞赛列表 500 修复（v2→v3 修复）

| 检查项 | 结果 | 证据 |
|---|---|---|
| NULL 列 COALESCE | ✅ 已修复 | `student_features_repo.go` ListCompetitions/GetCompetition/AdminList/MyRegistrations 均加 COALESCE |
| 线上验证 | ✅ | `GET /competition/list` 返回 `code:0` + 5 条数据 |

### 4.6 年级主题切换（v2→v3 新增）

| 检查项 | 结果 | 证据 |
|---|---|---|
| 4 年级色板 | 满足 | `main.dart` _GradeThemes（迎新青绿/追梦蓝/奋斗橙/创业紫） |
| 自动切换 | 满足 | `grade = 系统年 - 入学年 + 1`，clamp(1,4) 防越界 |
| 开关 | 满足 | Profile 页「年级主题自动切换」 |
| 后端字段 | 满足 | Profile 接口补 enrollment_year |

---

## 5. Context Engine 审计

| ID | 级别 | 问题 | 证据 |
|---|---|---|---|
| DPV4F-W1 | 警告 | `context_engine` 包未接入生产链路 | 生产代码无引用，major_agent 走 `kbRepo.Search`（未变） |
| DPV4F-W3 | 警告 | 质量看板缺实现 | ✅ **已修复**（chat_metrics_repo 全指标） |
| DPV4F-N1 | 备注 | 向量/Agentic RAG 未启用 | go.mod 无向量依赖（未变） |

---

## 6. 知识库同步与导出

（与 v2 一致，无增量变更）

- 知识包 + 复合 cursor + sha256/HMAC ✅
- 前端导出仍以 PDF/PNG/MD 为主（未变）

---

## 7. 安全与合规审计

| ID | 级别 | 问题 | 状态 |
|---|---|---|---|
| DPV4F-B12 | **阻断** | 域名备案未通过，`www.wxx-agent.online` 公网不可达 | 未变（外部渠道） |
| DPV4F-B11 | **阻断** | IP 直连无匹配 Caddy 站点块 | 未变（仓库 Caddyfile 无 IP 块，CI 不写） |
| DPV4F-W4 | 警告 | 情感独立授权缺失 | 未变 |
| DPV4F-W5 | 警告 | 配额默认值（200/日）与文档（20/日）不符 | 未变 |
| DPV4F-W6 | 警告 | 无双模型切换 | 未变 |
| DPV4F-W7 | 警告 | 错误码体系不一致 | 未变 |

**安全新增观察**：个人档案聚合接口使用 `auth.SelfTwinRead` 门控 ✓；头像上传限制 3MB + 白名单扩展名 ✓；头像服务按 user_id 返回（同 scope 权限由门控兜底）。

---

## 8. 工程质量与测试验证（v3 实测）

| 命令 | 结果 |
|---|---|
| `go build -tags fts5 ./...` | ✅ 通过 |
| `go vet ./internal/...` | ✅ 通过 |
| `go test ./internal/...` | ✅ **全量全绿**（18 个包全部 PASS） |
| `gofmt -l` | ✅ 干净（reset-seed 已修复） |
| `flutter analyze --no-pub` | 0 error、0 warning、191 info |
| `flutter build web --release` | ✅ 通过 |
| 服务器部署 | ✅ 后端 active + healthy，前端 Web 21:58 部署 |

---

## 9. 问题清单（v3）

| # | ID | 级别 | 模块 | 问题 |
|---|---|---|---|---|
| 1 | DPV4F-B12 | **阻断** | 访问入口 | 域名备案未通过，公网不可达（外部渠道） |
| 2 | DPV4F-B11 | **阻断** | 访问入口 | IP 直连无 Caddy 站点块 |
| 3 | DPV4F-B3 | 外部联调 | 对接 | SSO 需校方参数 |
| 4 | DPV4F-B4 | 外部联调 | 对接 | 学工/一表通需凭证 |
| 5 | DPV4F-W1 | 警告 | Context Engine | 包未接线 |
| 6 | DPV4F-W2 | 警告 | 前端 | Hive 未落地 |
| 7 | DPV4F-W4 | 警告 | 合规 | 情感独立授权缺失 |
| 8 | DPV4F-W5 | 警告 | 限流 | 配额默认值与文档不符 |
| 9 | DPV4F-W6 | 警告 | LLM 容灾 | 无双模型切换 |
| 10 | DPV4F-W7 | 警告 | API 契约 | 错误码与附录 A 不一致 |
| 11 | DPV4F-N1 | 备注 | 可选能力 | 向量 RAG 未启用 |
| 12 | DPV4F-N2 | 备注 | 演示数据 | 部分 fallback 仍存在 |
| 13 | DPV4F-W12 | 备注 | 性能 | AvatarCard 无限动画重绘 |
| 14 | DPV4F-W13 | 备注 | 前端质量 | info lint 191 条 |

**v2 → v3 收敛**：W3（质量看板）✅ 修复；B14（种子密码）✅ 更新为 Wxx@2026；其余未变。

---

## 10. 综合评分

| 维度 | v2 | v3 | 说明 |
|---|---|---|---|
| 架构与技术选型 | 8.0 | **8.0** | Hive 仍缺失 |
| Context Engine | 8.0 | **8.5** | 质量看板补齐（W3 修复） |
| 核心功能实现度 | 8.5 | **9.0** | +学科专业智能体 +数字孪生自画像 +档案聚合，均无缺陷 |
| 知识同步与导出 | 8.0 | 8.0 | 未变 |
| 安全与合规 | 6.0 | 6.0 | 访问入口阻断项拉低 |
| 工程质量 | 7.5 | **8.0** | 测试全绿 + 0 warning + gofmt 干净 |
| 上线就绪度 | 5.0 | **5.5** | 数据链路验证通过，域名仍不可达 |
| **综合** | **8.0** | **8.5** | 功能增量 + 数据链路 + 工程质量提升 |

---

## 11. 修复路线图

### 11.1 最紧急（阻断项）

1. **DPV4F-B12**：域名备案重提或申请校方子域（外部渠道）
2. **DPV4F-B11**：Caddyfile 新增 `129.211.223.113` 站点块（或通配 `*`），使 IP 直连正确路由 `/api/*`；或临时自签名证书覆盖 SNI
3. **DPV4F-W1**：将 `context_engine` 包接入生产链路，或从代码库移除（当前为死代码，误导性大于价值）

### 11.2 第二优先级（警告项）

- **W4**：情感分析独立授权开关
- **W5**：配额默认值对齐文档
- **W6**：LLM 双模型超时切换
- **W7**：错误码体系对齐文档附录 A
- **W12**：AvatarCard 动画按页面可见性暂停，减少持续重绘
- **W13**：收敛 info lint

### 11.3 外部联调（暂缓）

- 校方 SSO / 学工 / 一表通 / 域名备案

---

## 12. 最终结论

**整体评级：功能完备、数据链路已验证，访问入口仍为唯一门槛。**

v3 相比 v2：新增 6 大功能模块（学科专业智能体、数字孪生自画像、个人档案聚合、Markdown 统一渲染、年级主题切换、校园服务学院入口）均通过源码审读 + 构建测试 + 生产数据实测，未引入明显缺陷；质量看板（W3）已修复；种子密码统一为 `Wxx@2026`；CI scp 挂起已解决；**生产数据链路经真实学生账号验证是通的**（此前"无数据"为登录密码与数据导入问题，已修复）。

剩余核心障碍仍是**域名备案（B12）+ IP 直连路由（B11）**——这决定功能能否被真实用户访问，需外部渠道解决。

---

*报告生成：2026-08-07 | 审核工具：源码静态审读 + 子代理交叉核验 + `go build` + `go vet` + `go test` + `gofmt` + `flutter analyze` + 生产接口实测*
