# 本会话变更记录 — 学生分年级/关注定制 + 角色差异化补强 + 知识治理/管理决策智能体

> **会话时段**: 2026-08-18（约 05:49 ~ 14:20）
> **调度方**: `leader-wxx`（OpenClaw 多智能体协作，含 `reviewer-audit-wxx` 审计子代理）
> **分支**: `main`
> **提交链**: `4a943ab` → `ce22049`（8 个提交）
> **规模**: 25 个文件变更，+1352 / -179 行
> **配套文档**: 本文档 + `docs/蔚小芯角色功能.md`（升版 v5.4，附录 B.7）

---

## 一、会话目标

依用户三轮需求推进：
1. **学生角色按「不同年级 + 不同关注内容」的针对性设计补齐**（此前仅视觉层有年级区分，缺失内容/功能层差异化）。
2. **各角色差异化去硬编码**：无真实数据源时**诚实空**，不再返回编造的 reference/mock 样例。
3. **问芯（AI 对话）三问落地**：①对话页布局重新设计；②各角色差异化的智能体（结论：不按角色复制整套，给管理类新增独占决策智能体）；③知识库导入导出的智能体处理（结论：确定性校验保规范格式 + LLM 治理做准确性增强，只出报告不改写）。

---

## 二、提交清单与说明

| # | 提交 | 主题 | 主要内容 |
|---|------|------|---------|
| 1 | `4a943ab` | 学生年级+关注内容定制 & 新生引导年级硬门控 | 后端 `UserContext/JWT/中间件` 新增 `Grade`（入学年份推导 1~4）；AI `rolePerspective` 增加 `gradePerspective`；前端学生专区按「年级主键+关注次键」重排、副标题年级阶段、首次采集关注内容（共享组件）、「我的」可改关注；新生引导**仅大一自动弹窗**，老生不弹不写数据仍可主动进入 |
| 2 | `9335938` | 各角色差异化功能去硬编码（诚实空） | 教辅（排课冲突/毕业审核/考试安排）、教师（每日概览/学情热力图）、学院管理员（教师效能/课程质量）——无真实数据源时返回 `data_source=real` 空态+说明，不再编造「张教授/示例学生」假数据 |
| 3 | `f52d17d` | 问芯对话页智能体选择器紧凑化 | 由「两行大卡片」改为 **34px 紧凑单行 chips**，去冗余标签行、消息区更大 |
| 4 | `675358a` | 知识治理智能体 | 新增 `KnowledgeGovernanceService`：确定性检查（缺失字段/正文过短/无标签/重复标题/已失效）+ 可选 LLM 准确性审计（≥中/高风险才入报告）；路由 `GET /kb/governance`（`counselor.kb.review`）；3 条服务测试；前端知识治理页「智能体审计」弹窗。**只出报告、永不自动改写** |
| 5 | `185ab5f` | 管理决策智能体 | 迁移 `096_admin_agent.sql` 种子「管理决策师」（custom 类型）；chat_provider 硬门控：仅 `counselor/college_admin/school_admin/sys_admin` 可见 |
| 6 | `ccf961c` | 角色功能文档 v5.4 | 新增附录 B.7（B.7.1 知识治理 / B.7.2 管理决策+角色视角全貌 / B.7.3 对话页优化），更新版本头与历史变更 |
| 7 | `85db33f` | 深化 B/C | B：`GenerateDecisionAdvice` 接真实聚合指标（学生数/风险数/健康均分），`data_source` 区分 `ai+real/ai`，handler 传学院 ownerID；C：问芯空态新增**角色专属推荐提问**（`_buildRoleSuggestions` 按角色展示可点问题 chips） |
| 8 | `ce22049` | 文档修正 | 修正 v5.4 更新说明附录引用 `B.6`→`B.7` |

---

## 三、变更文件明细

### 后端（13 个文件）
- **学生年级上下文**：`jwtutil/token.go`（`CustomClaims.Grade` + `ResolveGrade`）、`middleware/jwt.go`、`model/dto.go`（`UserContext.Grade`）、`agent/role_perspective.go`（拆分 `roleBasePerspective` + `gradePerspective`）
- **角色去硬编码**：`service/assistant_service.go`、`service/teacher_service.go`、`service/college_service.go`
- **知识治理智能体（新增）**：`service/knowledge_governance_service.go`、`handler/knowledge_governance_handler.go`、`service/knowledge_governance_service_test.go`（3 用例）
- **管理决策数据落地**：`service/college_service.go`（`GenerateDecisionAdvice` 接真实指标）、`handler/college_handler.go`（传 `collegeOwnerID`）
- **装配与路由**：`pkg/app/app.go`、`pkg/app/deps.go`、`pkg/app/routes.go`（+`/kb/governance`）、`pkg/app/regression_test.go`（路由计数 478→479）
- **迁移**：`server/migrations/096_admin_agent.sql`（管理决策师种子）

### 前端（11 个文件）
- `utils/storage.dart`：关注内容持久化 + `grade` 推导
- `widgets/student_interest_pick_dialog.dart`（新增）：关注多选对话框（首页采集/「我的」修改复用）
- `pages/home/home_page.dart`：学生专区排序、年级阶段副标题、关注采集、新生引导年级硬门控
- `pages/profile/profile_page.dart`：「我的关注」入口卡片
- `pages/chat/chat_page.dart`：智能体选择器紧凑化 + 空态角色专属推荐提问
- `providers/chat_provider.dart`：`admin-decision` 管理角色硬门控
- `pages/admin/my_submissions_page.dart`：知识治理「智能体审计」弹窗
- `config/api_config.dart`：`kbGovernance` 端点

### 文档（2 个文件）
- `docs/蔚小芯角色功能.md`：升版 v5.4，附录 B.7，历史变更行
- 本文档（本变更记录）

---

## 四、设计决策要点（供后续参考）

1. **角色差异化智能体架构**：不按角色复制整套智能体（职能重叠爆炸、维护差），采用**领域智能体 + 角色感知视角**（`rolePerspective`/`gradePerspective` 注入 4 个领域 agent）；仅对差异最大的**管理类角色**新增独占「管理决策师」（硬门控可见）。
2. **知识库智能体分层**：**规则层保规范/格式**（NDJSON/hash/数量门槛/批量精修，确定性、不得交给 LLM） + **治理层做准确/质量**（LLM 审计只读报告，不自动改写，防 AI 改错）。
3. **诚实原则**：凡无真实数据源的差异化功能，一律返回 `data_source=real` 的空态/说明，而非编造 `reference` 样例（对齐 sys_admin/college twin-screen 既有健康做法）。

---

## 五、验证结果

- **后端**：`go build ./...`、`go vet ./...` 全绿；`go test ./internal/service/`（含治理 3 用例、college 决策）、`./internal/handler/`、`./pkg/app/`（含迁移 096 + 路由 479）全部通过。
- **前端**：`flutter analyze` 0 error；`flutter test` 6/6。
- 本会话历史任务 8 项**全部落地**，工作区干净，main 最新 `ce22049`。

---

## 六、遗留/后续可选（非本轮范围）

- 管理决策师的**深层数据接入**（真实学情/工单聚合的端到端决策）、以及角色专属推荐问题的前端深度定制等，可作为二期（见 `docs/蔚小芯角色功能.md` 附录 B 与 `role-diff-gap-20260818.md` 三、优先级排序）。

*本记录由 leader-wxx 依据 git 提交链与代码核验生成。*
