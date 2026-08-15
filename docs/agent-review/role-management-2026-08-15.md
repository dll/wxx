# 角色管理增强 — 审核与实现记录

> 日期：2026-08-15
> 范围：用户管理页的角色/职务分配、角色变更授权加固
> 状态：实现完成，本机验证通过（Go build + tests、flutter analyze）

## 一、背景与缺口

用户指出角色目前由**硬编码账号**决定（如 `stunion` 账号即代表学生会）是错的，要求：

1. 六种角色应有正式的**角色分配入口**（当前无）。
2. 在**用户管理**中新增**角色管理**能力。
3. 系统管理员与**学院管理员**都应能分配用户角色（学生会学生应被分配角色，而非用 `stunion` 账号标记）。
4. 角色需支持**职务/职位**——如学生会成员应有职务（主席/部长/干事）。

### 诊断出的真实缺口
- **无 `position`（职务）字段**：`users` 表没有职务列，角色与职务无法分离表达。
- **角色变更授权缺失**：通用 `UpdateUser` 无越权防护——学院管理员可把用户改为校级管理员，或修改其他学院/管理员。
- **职务无展示**：列表与编辑弹窗均无职务入口。

> 注：`userRepo.Update` 已无条件 `token_version+1`，因此角色变更本身是**强制重新登录生效**的（安全），无需额外改动；此前担心的「改角色不失效旧 JWT」经核读为误判——`UpdateUser` 走的就是会 bump token_version 的 `Update`。

## 二、改动清单

### 后端
| 文件 | 改动 |
|---|---|
| `server/migrations/085_user_position.sql` | **新增**，`users` 表加 `position TEXT NOT NULL DEFAULT ''` |
| `server/internal/model/entity.go` | `User` 实体加 `Position string`（`json:"position" db:"position"`） |
| `server/internal/model/dto.go` | `UserUpdateRequest` 加 `Position *string` |
| `server/internal/repository/user_repo.go` | `Update()` 的 UPDATE 语句增加 `position=?` |
| `server/internal/service/admin_service.go` | 重写 `UpdateUser`（签名 `operator *model.UserContext`，支持 position）；新增 `checkRoleChangeAuth` 越权校验 |
| `server/internal/handler/admin_handler.go` | `UpdateUser` 调用由 `userCtx.Username` 改为传 `userCtx` 指针 |

### 授权规则（`checkRoleChangeAuth`）
- 任何人都**不能**修改 `sys_admin` 的**角色/状态**（防止锁死/越权）。
- `sys_admin` / `school_admin`：可管理任意角色、职务、归属。
- `college_admin`：
  - 不能授予 `sys_admin` / `school_admin` / 其他 `college_admin`；
  - 不能修改其他**管理员**账户；
  - 仅能管理**本院**用户（`college`/`owner_id` 与操作者 `OwnerID` 均非空且匹配才放行；无法判定时不误拦）。
- 其他角色（counselor/teacher/assistant/student 等）：**无权**修改角色/职务/归属。

### 前端（`frontend/lib/pages/admin/admin_users_page.dart`）
- `UserProfile`（`models.dart`）加 `position` 字段与解析。
- 列表项（`_UserTile`）：角色徽章旁新增**职务徽章**（secondaryContainer 底色）。
- 编辑弹窗（`_UserEditDialog`）：
  - 新增**职务**输入框（`position` 落库）；
  - 按当前登录者角色动态生成**可指派角色列表**：学院管理员隐藏「学校管理员」（也无法指派学院管理员）；
  - 角色切换时展示**职务快捷选项**（ActionChip，如 学生会→主席/副主席/部长/副部长/干事；辅导员→年级/专职/兼职；教师→讲师/副教授/教授；学生→班长/团支书等；可自定义兜底文本）。
- 保存时提交 `position`。

## 三、验证结果
- `go build -tags fts5 ./...`：通过（EXIT:0）。
- `gofmt -l`：无输出（干净）。
- `go test -tags fts5 ./internal/service/`：`ok ... 128.299s`（通过）。
- `flutter analyze --no-fatal-infos`：0 error / 0 warning；231 条 info（基线 + 新增 prefer_const，CI `--no-fatal-infos` 通过）。

## 四、未覆盖 / 后续
- 本增量未做「前端『角色管理』独立 Tab」的大改版——角色分配已内嵌于用户编辑弹窗（含职务），满足「正式分配入口」诉求；如需独立「角色管理」页（角色列表/人数/一键批量指派），可作后续增量。
- 学院管理员的「本院」判定依赖 `OwnerID`/`college` 命名一致性；若学校数据两种命名不一致需在数据层统一。
- `guest`、`pending/rejected` 等边缘状态未纳入编辑下拉（沿用现有「正常/已禁用」两项），可后续补全。

## 五、结论
角色分配从「硬编码账号」升级为「用户管理 → 编辑 → 角色 + 职务」的正式入口；角色变更经 `token_version` 强制重新登录生效；越权提权路径（学院管理员授校级管理员 / 跨学院操作）已在服务端封堵。本机全链路编译、测试、静态分析通过。
