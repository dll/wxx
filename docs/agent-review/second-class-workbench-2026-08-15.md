# 辅导员「第二课堂」班级看板落地（2026-08-15）

> 交付：commit `c3270e9`，已通过 CI/CD 自动部署到生产并线上实测通过。
> 背景：用户「第二课堂不能没有，所有缺少都要补充」+「按你建议，不用再征求」。

## 一、实施方案（D → 已完成）

| 块 | 内容 | 状态 |
|----|------|------|
| A | 制度文档入库（《第二课堂成绩单制度实施办法》） | ✅ 早已完成（迁移 088，kb_resources id=230 Policy/school） |
| B1 | 辅导员工作台「第二课堂」入口 + 班级看板 | ✅ 本次实现 |
| B2 | 辅导员发起第二课堂活动 | 复用学生会 `POST /health/activities`（辅导员继承 student_union 已有 `union.event.plan` 能力），前端可随时接入，本次未单独做页面 |
| C | 诚实边界：不伪造积分/学分；无数据时诚实显示 | ✅ 看板 data_source=real/not_available，无记录显示 0/暂无 |

## 二、实现内容

### 后端
- **新能力** `counselor.secondclass.board`（`server/internal/auth/capabilities.go`，绑定 counselor）
- **新 repo** `second_class_repo.go`：`ClassSecondClassBoard(ownerScope, ownerID)` — 按辅导员归属范围拉名下学生 + 聚合 `health_activity_signups`（报名/到场）+ `student_points`（积分），范围锁定防越权，无学生时诚实返回 `not_available`
- **service**：`CounselorService.SecondClassBoard(ctx, scope, ownerID)` + `SetSecondClassRepo` 注入
- **handler**：`CounselorHandler.SecondClassBoard`（真实数据，失败诚实空看板不造假）
- **路由**：`GET /counselor/second-class-board`（`counselor.secondclass.board`）

### 前端
- **新页面** `second_class_board_page.dart`：顶部汇总卡（学生/活动/到场/积分）+ 搜索 + 学生参与明细列表（姓名脱敏）+ `DataSrcBadge` 动态标记真实数据
- **provider**：`CounselorFeatureProvider.fetchSecondClassBoard`
- **工作台入口**：辅导员工作台「第二课堂」卡片（`Icons.school_outlined`，能力门控）
- **路由**：`/counselor/second-class-board`

## 三、验证（全部实测）

- **本地**：`go build -tags fts5 ./...` 通过；`gofmt -l server/` 干净；冒烟测试 SMOKE_OK（范围过滤正确排除非本院学生、积分/报名/到场去重聚合正确、空范围返回 not_available）
- **前端**：`flutter analyze --no-fatal-infos` → 292 issues 全 info，0 error/warning（与 CI 门禁一致）
- **CI/CD**：Deploy Frontend 质量门禁 1m13s ✓ / 后端部署 37s ✓ / 前端 Lighthouse 部署 2m57s ✓，服务器 git 重置到 `c3270e9`
- **线上实测**：
  - `POST /api/v1/auth/login`（counselor_cs）→ 拿到 token
  - `GET /api/v1/counselor/second-class-board` → `data_source=real`，正确返回名下 2 个学生（student_cs、student1），当前活动参与 0（诚实，不造假）
  - counselor_math → 只看到自己的 1 个学生（student_math）→ **范围隔离正确**，辅导员间不越权
  - 前端 main.dart.js 已更新（23:23），公网 `wxx-agent.online` HTTP 200

## 四、数据归属断点（记录待办，本次未擅自改）

- 辅导员账号 `owner_id` 用短编码（`cs`/`math`），而**多数学生（491/494）`owner_id` 是中文全名学院**（如「计算机科学与工程学院（网络空间安全学院）」）。
- 效果：李辅导员（cs）按范围只能查到 2 个学生，其余主体学生（中文全名归属）查不到。
- **本次诚实地按实际可查范围显示**（不编造、不看全），并记录该断点。
- **待用户决定**：是否统一对齐学生 `owner_id` 与辅导员编码（影响面大，涉及 491 条账号归属），对齐后看板即可覆盖全部学生。

## 五、边界与不造假声明

- 不伪造积分/学分：`student_points` 空即显示 0/暂无，等真实活动（报名/到场/积分）产生记录后自动聚合。
- 学生姓名脱敏展示；数据来源徽章动态反映真实数据或「参考/AI」。
- 学分换算（学时→学分）：制度文档已入库（id=230），换算展示待确认口径后再做，本次不引入。
