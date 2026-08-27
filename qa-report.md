# QA 回归测试报告 — 110 迁移 + campus_map_page 管理员加载逻辑

- **测试专员**：qa-regression-wxx
- **日期**：2026-08-27
- **被测改动**：
  1. 新增迁移 `server/migrations/110_restore_campus_steps.sql`
  2. 修改 `frontend/lib/pages/campus/campus_map_page.dart` 的 `_loadStepsFromServer()`

---

## 一、测试用例与执行结果

### 后端（Go test）

| 用例编号 | 描述 | 结果 |
|---------|------|------|
| B1 | `110_restore_campus_steps.sql` 经 `ToMySQL` 转换后，`INSERT OR IGNORE` → `INSERT IGNORE`，无 SQLite 专有语法残留（AUTOINCREMENT / datetime('now' / ON CONFLICT） | ✅ PASS |
| B2 | 110 迁移转换后恰好输出 1 条 INSERT IGNORE，含 `'huifeng'` + `'langya'` 两类 campus_id | ✅ PASS |
| B3 | 110 迁移坐标采用 050 权威纠正值（会峰 step1=32.2705/118.3055、琅琊 step1=32.2921/118.2988） | ✅ PASS |
| B4 | 原始 SQL 保留 `INSERT OR IGNORE`（SQLite 方言合法，SQLite 下可直接执行） | ✅ PASS |
| B5 | `internal/db`、`internal/repository`、`internal/handler` 三包全量单元测试 | ✅ PASS（0 失败） |

### 前端（Dart analyze 静态检查）

| 用例编号 | 描述 | 结果 |
|---------|------|------|
| F1 | `dart analyze lib/pages/campus/campus_map_page.dart` | ✅ 0 error / 0 warning，23 条 info 级 lint（均为既有的 `prefer_const` / `unused_element` / `use_build_context_synchronously` 风格提示，非本次改动引入） |

### 逻辑走查（代码审阅验证，非运行时）

| 用例编号 | 描述 | 结果 |
|---------|------|------|
| R1 | 普通用户（非管理员）行为不变：`_canEditNodes=false` 时走公开接口 `campusSteps`，空列表/异常静默回退硬编码常量 | ✅ PASS（`_loadStepsFromServer` 中仅当 `_canEditNodes` 才弹提示；`_campusStepsMap` 初值即硬编码常量） |
| R2 | 管理员空列表/失败分支：弹提示「后端无报到节点数据…」且 `_remoteIds` 保持为空 | ✅ PASS（`_showNoRemoteStepsHint` 三处调用点均在 `_canEditNodes` 为真分支；方法开头 `_remoteIds.clear()`） |
| R3 | 拖拽保存在 `_remoteIds` 为空时正确报错而非假成功 | ✅ PASS（`_saveCoordinate` 中 `stepId==null → return false`；`_toggleNodeEditing` 中 `_remoteIds.isEmpty` 时提示「保存失败：未加载到后端节点…」） |
| R4 | 管理员加载真实节点后拖拽保存完整链路：`_remoteIds` 填充 → `_api.patch(adminCampusStepCoords)` → `UpdateStepCoords` handler → `UpdateCoords` UPDATE SQL | ✅ PASS（代码链路完整，坐标范围校验 `lat∈[3,54] lng∈[73,136]` 通过） |
| R5 | 公开接口 `ListPublicSteps` → `ListPublished`（status='published'）不受影响 | ✅ PASS（迁移未改动 repository/handler 查询逻辑） |
| R6 | 迁移 048/050/079 既有行为不受 110 影响 | ✅ PASS（110 为纯 INSERT，不改表结构、不改既有数据；且按文件名字典序 110 在 079 之后执行，符合「先清空后恢复」的时序） |

---

## 二、缺陷分级

### P0（阻断上线，必须修复）

无。

### P1（必须修复的功能/数据正确性问题）

- **P1-1：`INSERT OR IGNORE` 在无唯一约束下无法幂等去重**
  - 位置：`110_restore_campus_steps.sql` + `048_campus_map_steps.sql` 建表定义
  - 现象：`campus_checkin_steps` 表仅定义 `id INTEGER PRIMARY KEY AUTOINCREMENT`（id 自增），**未在 `(campus_id, step_order)` 或任何业务键上定义 UNIQUE 约束**（B 系列测试已程序化核实：建表语句含 PRIMARY KEY、不含 UNIQUE）。
  - 后果：SQLite 的 `INSERT OR IGNORE`、MySQL 的 `INSERT IGNORE` 只会忽略「唯一键冲突」错误。**没有唯一约束就没有冲突可忽略**，因此：
    1. 若迁移 110 被重复执行（例如：先手动补数据、再跑迁移，或迁移记录 `_migrations` 表被重置后重跑），会**重复插入 12 条数据**，导致 admin 接口返回 24 条，前端地图出现重复标注、`_remoteIds` 错位。
    2. 迁移内部的 `INSERT OR IGNORE` 语义名不副实——它并不能像设计意图那样「幂等恢复」。
  - 建议修复（二选一）：
    - 在 `048` 或 `110` 中为 `campus_checkin_steps` 增加唯一索引：`CREATE UNIQUE INDEX idx_campus_step ON campus_checkin_steps(campus_id, step_order)`，使 `INSERT OR IGNORE` 真正生效；
    - 或改用「先 `DELETE` 该校区再 `INSERT`」或 `ON DUPLICATE KEY UPDATE`（MySQL）/ `ON CONFLICT ... DO UPDATE`（SQLite）的确定性幂等写法。
  - 风险缓解：当前生产环境 079 已清空该表、110 仅首次执行一次，故**当前一次性执行不会产生重复**；但这是「依赖外部时序而非迁移自身幂等」的脆弱状态，一旦重跑即出问题。

### P2（建议改进 / 低风险）

- **P2-1**：前端 `campus_map_page.dart` 存在 23 条 info 级 lint（`prefer_const_constructors`、`unused_element`、`use_build_context_synchronously` 等），均为既有代码风格问题，非本次改动引入，不影响功能。
- **P2-2**：`_loadStepsFromServer` 中复用 `_campus.id` 构造接口 URL 时，若管理员快速切换校区，旧的异步响应可能晚于新请求返回并覆盖 `_campusStepsMap`（竞态）。本次改动未引入，但管理员加载逻辑的路由切换场景下仍存在潜在串扰（`_remoteIds` 已在方法开头 clear，风险有限）。建议后续加请求序号/token 防竞态。

---

## 三、结论

1. **方言转换正确**：`dialect.go` 的 `insertOrIgnoreRe` 正则能正确将 `INSERT OR IGNORE` 转为 `INSERT IGNORE`（MySQL），且转换输出不含 SQLite 专有残留；B 系列测试全绿。
2. **前端改动逻辑正确**：普通用户静默回退、管理员空/失败分支提示且 `_remoteIds` 置空、拖拽保存在空 `_remoteIds` 下正确失败——三条核心需求（改动2 的 ①②③）均通过代码走查验证。
3. **回归不退化**：公开接口 `ListPublicSteps`、管理员 `_remoteIds 填充→patch→UpdateCoords→UPDATE SQL` 完整链路、048/050/079 既有行为均未受 110 影响。
4. **存在 1 个 P1 幂等性缺陷**：`campus_checkin_steps` 表无唯一约束，`INSERT OR IGNORE` 无法真正幂等，重复执行会重复插入。当前一次性部署不受影响，但强烈建议为 `(campus_id, step_order)` 增加唯一索引以消除隐患。

**总体结论**：改动功能正确、无回归，**可上线**；但需跟进 P1-1 的幂等性加固（否则未来任何一次迁移重跑都会污染数据）。

---

## 四、未能覆盖的项（如实标注）

1. **无 MySQL 真库**：本环境仅有 SQLite 驱动，未连接真实 MySQL 实例，`INSERT IGNORE` 在 MySQL 下的实际执行结果未能运行时验证——仅通过 `ToMySQL` 单元测试验证了「转换后的 SQL 文本正确」，MySQL 端执行语义（尤其无唯一约束时 INSERT IGNORE 的去重行为）是基于 MySQL 文档语义的静态判断，未做实库验证。
2. **前端运行时测试**：未启动 Flutter Web/App 实际点击验证「管理员弹提示」「拖拽保存」的 UI 行为，仅做静态分析 + 代码走查（无浏览器/设备运行环境与后端联调）。
3. **083 及之前已部署生产库的实际 110 执行**：未在包含真实生产数据（含 079 清空后的状态）的库上实跑 `cmd/migrate` 工具。
