# 蔚小芯重构基线与批次计划

> 建立日期：2026-09-04。本文记录重构前可复现基线，避免结构调整改变既有业务契约。

## 一、当前基线

| 检查项 | 命令 | 结果 |
|---|---|---|
| Go 单元测试 | `go test ./server/...` | 通过 |
| Flutter 单元/Widget 测试 | `cd frontend && flutter test` | 通过（33 项） |
| Flutter 静态分析 | `flutter analyze --no-pub --no-fatal-infos --no-fatal-warnings` | 通过，50 条 info（已自动修复 260 条确定性 lint） |
| Go 文件规模 | `server/` | 453 个 Go 文件，144 个测试文件 |
| Flutter 文件规模 | `frontend/lib/` | 269 个 Dart 文件 |
| 数据库迁移 | `server/migrations/` | 114 个迁移 |

Go 测试包含 agent、auth、context_engine、handler、repository、service、Temporal 和 app 等关键包。Flutter 分析结果目前以 info 为主，不阻断构建，但应按风险分批收敛。

## 二、已知重构热点

### Flutter

- `pages/vopc/vopc_page.dart`：约 126 KB
- `pages/home/home_page.dart`：约 95 KB
- `pages/admin/my_submissions_page.dart`：约 76 KB
- `pages/chat/chat_page.dart`：约 71 KB
- `pages/profile/profile_page.dart`：约 66 KB
- `pages/campus/campus_map_page.dart`：约 66 KB

### Go

- `internal/service/student_service.go`：约 78 KB
- `pkg/app/routes.go`：约 74 KB
- `internal/service/document_service.go`：约 52 KB
- `internal/model/entity.go`：约 49 KB
- `internal/repository/kb_repo.go`：约 48 KB
- `internal/handler/vopc_handler.go`：约 40 KB

这些文件优先按职责拆分，保持公开 API、路由、数据库字段和 AnswerCard 契约不变。

## 三、重构批次

### 批次一：治理与基线固化

- README、AGENTS 与真实目录结构对齐。
- CI 同时覆盖 Go 和 Flutter 基础检查。
- 保留未跟踪运行状态文件，未经确认不删除；临时文件后续单独归档。

### 批次二：后端路由与依赖组装（阶段完成）

- 已完成：将根路由、健康检查、Flutter 静态资源和 SPA 回退提取至 `server/pkg/app/routes_static.go`。
- 已完成：将公开 API（认证、版本、校园报到、公开知识）提取至 `server/pkg/app/routes_public.go`。
- 已完成：将 vOPC 项目协作与风险治理路由提取至 `server/pkg/app/routes_vopc.go`。
- 已完成：将问答、会话、知识浏览和情感路由提取至 `server/pkg/app/routes_self.go`。
- 已完成：将问题预案和毕设选题路由提取至 `routes_forecast.go`、`routes_graduation.go`。
- 已完成：将竞赛、大学规划、入党教育和社团生活路由提取至 `routes_student_features.go`。
- 已完成：将知识库、知识包同步、智能体、语音、集成和词元统计路由提取至 `routes_platform.go`。
- 已完成：将管理端统计、用户、审计、设置和版本路由提取至 `routes_admin_core.go`。
- 已完成：将当前用户资料、凭证、偏好和安全设置提取至 `routes_user.go`，并增加拆分注册函数接入断言。
- 回归测试已改为读取拆分后的路由文件集合，路由路径与中间件行为保持不变。
- 保持中间件顺序、鉴权和路由路径不变。

### 批次三：后端领域服务（准备中）

- 优先拆分学生、文档、辅导员和知识库服务/仓储。
- repository 只负责数据访问，service 负责业务规则，handler 只负责协议适配。

当前已完成路由层领域边界，尚未大规模迁移 service/repository 业务实现。已为 `student_service.go` 建立兜底、LLM 解析与质量门槛接口测试；`kb_repo.go` 已有搜索、权限过滤、CRUD 与分页相关测试；问题预案服务已补齐对话热点关键词聚合并加入排序测试。下一步在保持这些契约的前提下，再进行方法迁移。

### 批次四：Flutter 页面组件化（待开始）

- 优先处理 VOPC、首页、问芯、校园地图。
- 将页面容器、状态、请求、区块 Widget、表单和对话框分离。

### 批次五：静态质量收敛（阶段完成）

已完成：使用 `dart fix` 修复 244 条确定性问题（const、花括号、废弃 API、无效转换等）。
剩余 50 条主要是异步 `BuildContext`、Web 平台专用库提示和少量未使用私有方法，需逐项人工确认后处理；模型 `part-of` 提示已统一为文件 URI。

## 四、每批验收

- Go：`gofmt -l .` 为空，`go vet -tags fts5 ./...` 通过，`go test -race -tags fts5 ./...` 通过。
- Flutter：`flutter test` 通过，`flutter analyze` 不新增 error/warning。
- API 路由、RBAC、sources、fallback、数据库迁移行为保持兼容。
- 每个可演示增量同步更新文档并单独 Git 提交。
