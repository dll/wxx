# 校园导航「管理端拖动节点坐标无法持久化、刷新回退」重构核对清单

> 核对专员：pm-wxx（只读，未修改任何代码）
> 核对日期：2026-08-27
> 结论：leader 定位的三层根因全部核实属实，修复方案可行。以下为核对确认与验收标准。

---

## 一、根因确认（全部核实属实）

### 根因 1：数据层 —— 迁移 079 删空种子数据，且不回填

- 文件：`server/migrations/079_clear_preloaded_data.sql`
- 第 9 节明确执行 `DELETE FROM campus_checkin_steps;`，注释标注「9. 报到打卡点 campus_checkin_steps（12 条）」。
- 该迁移已执行（迁移 runner 幂等，`_migrations` 表已记录 `079_clear_preloaded_data.sql`），且 DELETE 不会自动回填。
- **后果：生产 MySQL 表 `campus_checkin_steps` 为空。** ✅ 属实

> 附带影响：079 同时删除了 kb_resources、process_steps、ai_briefings、competitions、毕设、career_policies、校历、course_schedules、health_activities 等大量演示数据，属**上线前统一清空的既定行为**，非本次 bug 独有。但 campus 模块因前端存在「假节点降级」而放大为拖拽不可用的用户可见故障。

### 根因 2：前端静默降级陷阱

- 文件：`frontend/lib/pages/campus/campus_map_page.dart`
- `_loadStepsFromServer()` 逻辑核实：
  1. 进入时先 `_remoteIds.clear()`；
  2. 管理员走 admin 接口 `ApiConfig.adminCampusSteps?campus=...`，普通用户走公开接口；
  3. `if (resp.data['code'] == 0)` 后取 `list`，**仅当 `list.isNotEmpty()` 时**才填充 `_remoteIds` 并覆盖 `_campusStepsMap`；
  4. 否则（空列表 / code != 0 / 异常 catch）**静默回退到本地硬编码 `_campus.steps`（前端 `_huifengSteps` 6 个 + `_langyaSteps` 6 个假节点），`_remoteIds` 保持为空**。
- **确认：`_remoteIds` 只在 admin 接口成功且 data 非空时填充。** ✅ 属实
- 关键点：本地假节点坐标与 050 纠正后的权威值**一致**（前端常量已按 OSM 纠正过），所以管理员看到的节点位置是「正确的假点」，更具迷惑性——看似可拖动，实则永远写不进库。

### 根因 3：拖拽保存取 `_remoteIds[index]` 为 null

- `_saveCoordinate()` 首行：`final stepId = _remoteIds[index]; if (stepId == null) return false;`
- 因 `_remoteIds` 为空，`stepId` 恒为 null，`_saveCoordinate` 直接返回 false。
- `_savePendingCoordinates()` 中所有请求「失败」，`_pendingCoordinates` 无法清空。
- `_toggleNodeEditing()` 退出编辑时报错：「保存失败：未加载到后端节点，请检查登录状态和网络后重试」。
- **坐标从未写入数据库，刷新即回退。** ✅ 属实

---

## 二、辅助机制核对（均正常）

### 1. 后端接口 `campus_handler.go`

- `ListAdminSteps`（GET /admin/campus/steps）：调用 `h.repo.ListAll(campus)`，空结果时 `steps == nil → steps = []model.CampusStep{}`，返回 `code:0, data:[]` ✅ 正常（不报错，返回空列表）。
- `UpdateStepCoords`（PATCH /admin/campus/steps/:id/coords）：参数绑定 lat/lng（required），含中国范围校验（lat 3~54、lng 73~136），调用 `h.repo.UpdateCoords`。**接口本身正常**，问题仅在于前端传不到有效 id。

### 2. 迁移机制 `server/cmd/migrate/main.go`

- `_migrations` 表记录已执行文件名，`SELECT COUNT(*) ... WHERE filename=?` 判断，已执行则 skip。
- 按 `sort.Strings(files)` 文件名排序执行；执行成功后 `INSERT INTO _migrations (filename)` 记录。
- **幂等确认 ✅**：同一文件名不重复执行。
- **迁移编号确认 ✅**：目录当前最大为 `109_feedback_repair_tasks.sql`，新增应为 **`110_restore_campus_steps.sql`**。

### 3. 方言转换 `server/internal/db/dialect.go`

- `ToMySQL()` 中 `insertOrIgnoreRe` 把 `INSERT OR IGNORE` → `INSERT IGNORE`（第 7 步）✅。
- `pkAutoRe` 把 `INTEGER PRIMARY KEY AUTOINCREMENT` → `BIGINT PRIMARY KEY AUTO_INCREMENT`（第 2 步）✅。
- 但注意：本次恢复数据的 `INSERT OR IGNORE INTO campus_checkin_steps (...)` 是 **DML 而非 DDL**，转换仅依赖 `insertOrIgnoreRe`；若采用 `INSERT OR IGNORE` 需指定唯一键才能真正「幂等去重」（见下方风险点 R2）。
- `INSERT` 列清单中的保留字列名会被 `insertColsRe` 加反引号处理（status/step_order 均非保留字，不受影响）。

---

## 三、风险点（需 leader/dev 关注）

### R1：恢复数据与 079 清空的语义冲突（高）
079 的意图是「上线前清空演示数据、改由管理员上传真实资源」。110 恢复 campus 种子数据在语义上是**逆向操作**。需确认产品决策：campus 报到节点应视为「系统必需默认流程」（随 080 之后恢复），还是应继续由管理员通过「流程管理」自行创建？
- 若管理员已通过流程管理创建了新的 campus_checkin_steps，110 恢复需避免冲突（用 INSERT IGNORE + 唯一键，或用 WHERE NOT EXISTS 判断「该 campus 无任何节点」才插入）。

### R2：INSERT OR IGNORE 的去重依据（高）
`INSERT OR IGNORE`（MySQL `INSERT IGNORE`）只有在**存在唯一约束命中冲突**时才忽略。当前 048 建表未对 `(campus_id, step_order)` 建唯一索引，若生产表为空则无所谓；但若需真正幂等（防止重复插入 12 条），应：
- 优先方案 A：显式 `WHERE NOT EXISTS` 判断该 campus 无节点才插入；
- 或方案 B：先给 `INSERT IGNORE` 增加唯一约束依赖（迁移补 `UNIQUE(campus_id, step_order)` 或显式指定 id 保证主键冲突）。

### R3：id 策略（中）
048 依赖自增 id（未显式指定）。110 若显式指定 id=1..12，可能与管理员后续追加的节点 id 冲突、或与 AUTO_INCREMENT 计数不一致。建议：**不显式指定 id，依赖 AUTO_INCREMENT 自增**，避免主键冲突；URL 中 `:id` 由 `_remoteIds` 从接口返回动态读取，不依赖固定 id。

### R4：坐标权威取值（中）
050 纠正后的权威坐标 = 前端 `_huifengSteps` / `_langyaSteps` 常量（已核对，二者一致）：
- 会峰：step1~6 = (32.2705,118.3055)(32.2745,118.3070)(32.2735,118.3060)(32.2770,118.3040)(32.2740,118.3090)(32.2720,118.3030)
- 琅琊：step1~6 = (32.2921,118.2988)(32.2932,118.3002)(32.2928,118.2995)(32.2940,118.2976)(32.2926,118.3000)(32.2917,118.2992)
- **注意：048 原始种子是错误坐标（会峰/琅琊写反 + 琅琊偏移 2km），绝不能照抄 048 的 lat/lng，须用 050 纠正后的值。** ✅ 方案中已明确，予以确认。

### R5：前端降级逻辑改动范围（中）
当前 `_loadStepsFromServer` 的静默降级对**普通用户**是合理兜底（离线可用），不能一刀切取消。需区分：
- **管理员（`_canEditNodes`）**：`_remoteIds` 为空时不再允许进入编辑/拖动，明确提示「后端无节点数据，请先在流程管理创建节点」。
- **普通用户**：保持现有静默回退到本地常量（只读展示，不涉及写库，无危害）。
- 改动点聚焦于 `_toggleNodeEditing()` 进入编辑前的守卫，以及（建议）admin 加载结果为空时的提示。

### R6：status 字段（低）
方案要求恢复 `status='published'`，与 048 一致。前端管理员走 admin 接口（含 draft）、普通用户走公开接口（仅 published），恢复 published 可同时服务两端。确认无冲突。

---

## 四、修复范围

| 层 | 文件 | 改动类型 | 说明 |
|---|---|---|---|
| 数据层 | `server/migrations/110_restore_campus_steps.sql`（新增） | 新增迁移 | 用 `INSERT OR IGNORE`（SQLite 写法）恢复会峰6+琅琊6共12节点，坐标取 050 权威值，`status='published'`，依赖自增 id（建议加幂等守卫） |
| 前端 | `frontend/lib/pages/campus/campus_map_page.dart` | 修改逻辑 | 管理员编辑模式下，`_remoteIds` 为空时禁止进入编辑并提示，不再静默降级到可拖动假节点；普通用户保留原兜底 |

**明确不在本次范围**：
- 不改 `campus_handler.go`（接口本身正常）。
- 不改 `dialect.go`（转换逻辑已支持所需语句）。
- 不改 079（保留既有清空语义）。
- 不改 048（历史种子保持不动，仅作为字段结构参照）。

---

## 五、验收标准

### 数据层（110 迁移）
1. 迁移文件存在且命名 `110_restore_campus_steps.sql`，编号为当前最大 +1（110）。
2. MySQL 下执行成功：`INSERT OR IGNORE` 被正确转换为 `INSERT IGNORE`，12 条数据落库。
3. SQLite 下执行成功：12 条数据落库。
4. 重复执行迁移不报错、不产生重复数据（幂等：依赖文件名去重 + 插入去重守卫）。
5. 落库坐标与 050 权威值一致（会峰/琅琊分开，无写反、无偏移）。
6. 12 条 `status` 均为 `published`；id 自增不冲突。

### 后端接口
7. `GET /admin/campus/steps?campus=huifeng` 返回 `code:0, data:[6 条节点]`（含 id）。
8. `GET /admin/campus/steps?campus=langya` 返回 `code:0, data:[6 条节点]`。
9. `PATCH /admin/campus/steps/:id/coords` 传入有效 id + 合法坐标，返回 `code:0`，DB 坐标更新。

### 前端（管理员端到端）
10. 管理员进入报到导航，地图正确加载后端 12 节点（非假节点），`_remoteIds` 非空。
11. 管理员拖动节点 → 松手暂存 → 退出编辑 → 校验坐标与 DB 一致 → 提示「全部节点坐标已保存」。
12. **刷新页面后坐标保持不回退**（对照 DB 值）。
13. 后端无节点（空表）时，管理员进入编辑**被阻止**，明确提示「后端无节点数据，请先在流程管理创建节点」，而非静默显示可拖动的假节点。
14. 普通用户体验不受影响：后端不可用时仍能离线看到只读流程（静默回退本地常量）。

### 回归
15. 流程管理（CRUD）面板增删改查、审核发布流程不受 110 迁移影响。
16. 079 之外的其他模块（知识库/流程/竞赛/毕设等）数据不受本次修复影响。
17. 已完成旧数据无冲突：若管理员已在流程管理手动创建过节点，110 不产生重复（依赖幂等守卫）。

---

## 六、待 leader/dev 决策项

1. **R1 语义决策**：campus 节点是否应随 110 恢复为「系统默认流程」？（若产品要求管理员自建，则应改为前端明确引导 + 不恢复数据，仅修前端提示。）
2. **R2 幂等实现**：选「WHERE NOT EXISTS 守卫」还是「补唯一约束 + INSERT IGNORE」？
3. **R3 id 策略**：是否确认「不显式指定 id、依赖自增」？

以上三项目前方案默认：恢复数据 + INSERT IGNORE（加守卫）+ 依赖自增，建议 dev 在 refactor-notes 中明确落地方案后再动工。
