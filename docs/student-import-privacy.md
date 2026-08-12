# 任务方案 — 新生数据导入 + 个人中心分级可见 + 用户管理优化

> 版本：v1.0 · 2026-08-12

## 背景与目标

1. 将 `data/2026级新生录取数据.xlsx`（360 名新生，16 列）按格式导入系统数据库，安全保密。
2. 个人中心：公共信息全员可见，私密信息（联系方式/出生年月等）仅本人 + 辅导员 + 管理员可见，其他用户不可见。
3. 用户管理：优化分类显示与筛选。
4. 支持直接导入库，或按该格式通过现有「导入学生」界面手工导入。

## 现状

- 现有导入（`admin_service.go`）只识别 8 列：学号/姓名/院系/专业/班级/入学时间/入学年份/角色。
- `users` 表已有：college/major/class_name/enrollment_date/enrollment_year/phone/wechat/qq/email。
- 个人中心（profile_page）显示：学号/学院/专业/班级，未展示联系方式；后端 `model.User` 直接序列化全部字段（含 phone/email），无分级。
- 用户管理页已有：搜索、高级筛选（学院/专业/班级/入学年份）、统计条、批量操作、分页。

## 范围

### 做

**A. 数据导入扩展（新生 16 列）**
1. `users` 表新增列（迁移 `074_student_profile_fields.sql`）：`gender`、`campus`、`education_level`、`study_duration`、`enrollment_date`（复用现有）、`expected_graduation_date`、`study_mode`、`ethnicity`、`political_status`、`birth_date`。
2. `ImportStudentRow` 扩展字段 + `ParseStudentXLSX` 增加列名映射（性别/校区/学历层次/学制/预期毕业时间/学习形式/民族/政治面貌/出生年月；专业院系→college、入校时间→enrollment_date）。
3. `ImportStudents` 写入新字段；`BatchCreateStudents` 对应扩展。
4. 用实际 xlsx 做一次导入验证（开发库/生产库，初始密码=学号，bcrypt 哈希，不落明文）。

**B. 个人中心分级可见（隐私保护）**
1. 后端 `model.User` 拆分级视图：
   - `UserPublic`：学号、姓名、角色、学院、专业、班级、入学年份、校区、民族、政治面貌（公共）。
   - `UserPrivate`（仅本人/辅导员/管理员）：手机号、微信、QQ、邮箱、出生年月、住址等。
2. `GET /api/v1/user/profile` 返回公共 + 本人私密字段；`GET /api/v1/user/:id/profile`（或现有查看他人资料接口）按调用者角色过滤——非本人/辅导员/管理员只返回公共字段。
3. 前端个人中心：公共信息卡片全员可见；「联系方式」等私密区块仅本人（编辑）与辅导员/管理员（查看）可见。

**C. 用户管理优化**
1. 顶部分类 Tab：按角色分组（全部/学生/教师/辅导员/管理员），点击切换过滤。
2. 高级筛选补充：性别、状态（active/pending/disabled）、政治面貌、校区。
3. 列表卡片/行展示新字段（班级、院系、入学年份、状态）；统计条按角色计数。

### 不做
- 不做身份证号存储（xlsx 无此字段）。
- 不做前端「看他人资料页」新页面（复用现有用户管理/个人中心展示能力）。
- 不动密码逻辑（沿用 bcrypt + 学号为默认密码）。

## 技术要点

| 项 | 说明 |
|----|------|
| 迁移 | `server/migrations/074_student_profile_fields.sql`（10 列，幂等） |
| 隐私过滤 | 在 service/handler 层按 `userCtx.Role + userCtx.UserID` 判定；涉及 `user_repo`/`admin_service` 查询路径 |
| 导入 | 扩展 `ParseStudentXLSX` + `ImportStudentRow` + `BatchCreateStudents` |
| 前端 | `profile_page.dart` 分级展示；`admin_users_page.dart` 角色 Tab + 新筛选 |

## 接口变更

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/user/profile` | 返回公共 + 本人私密字段（本人可见） |
| GET | `/api/v1/admin/users` | 列表/详情按调用者角色过滤私密字段（辅导员/管理员可见） |
| POST | `/api/v1/admin/users/import` | 扩展解析 16 列（后端兼容旧 8 列） |

## 步骤拆分

1. 迁移 074 + model.User 扩展字段
2. 导入服务扩展（ParseStudentXLSX / ImportStudents / repo BatchCreate）
3. 隐私分级：UserPublic/UserPrivate 视图 + profile 接口过滤 + 用户列表过滤
4. 前端 profile_page 分级展示
5. 前端 admin_users_page 角色 Tab + 筛选增强
6. 实际导入 360 名新生（开发库验证 + 生产库导入，安全保密）
7. 验证：go test / flutter analyze+test / 构建
8. 文档更新（user-import.md / ui-feedback 无关）+ 提交

## 验收标准

- 360 名新生按 xlsx 导入成功，字段完整（性别/校区/学历/学制/民族/政治面貌/出生年月等），初始密码=学号可登录，密码 bcrypt 存储。
- 个人中心：公共信息（学号/姓名/学院/专业/班级/校区等）其他用户可见；联系方式/出生年月仅本人、辅导员、管理员可见。
- 用户管理：角色 Tab 分类、性别/状态/政治面貌/校区筛选可用，统计准确。
- `go test ./...` 反馈/用户相关通过；`flutter analyze` 0 error/warning；`flutter test` 通过；构建成功。
- 导入过程无明文密码落库、无敏感信息日志输出。

## 回滚与检查点

- Git 检查点：迁移+后端、前端、导入执行各一次提交。
- 数据回滚：迁移幂等；导入可对个别账号停用/删除。
- 隐私回滚：恢复 User 单视图即可。
