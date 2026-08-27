# 代码评审报告（audit-report.md）— 复审（第二轮）

- 评审对象：校园导航坐标持久化 bug 修复（迁移 110 + 前端 fallback + 测试）
- 评审人：reviewer-audit-wxx（只读，未修改任何源码/迁移文件）
- 评审日期：2026-08-27（复审）
- 复审背景：上一轮结论“有条件通过”，提出 H1/H2 两个高危问题（迁移 110 用
  `CREATE UNIQUE INDEX (campus_id, step_order)` + `INSERT OR IGNORE` 会在①建索引
  阶段因重复 step_order 触发 duplicate entry 中断迁移、②唯一索引不含 status 破坏
  draft/pending_review/published 多版本并存）。dev 已据此回修，本轮复审最新代码。
- 评审范围（复审）：
  1. `server/migrations/110_restore_campus_steps.sql`（重写为 `WHERE NOT EXISTS` 守卫）
  2. `server/internal/db/migration110_campus_test.go`（断言反转）
  3. `server/internal/db/migration110_functional_test.go`（新增功能回归）
  4. `server/internal/db/dialect.go`（`ToMySQL` 方言转换，验证新写法兼容性）
  5. `frontend/lib/pages/campus/campus_map_page.dart`（M2 是否残留）

## 一、评审结论

**通过（Pass）**

上一轮的 H1、H2 两个高危问题已被有效解决，方案方向正确、实现一致、测试覆盖到位。
剩余问题均降级为 Medium / Low（M2 前端体验、M3 测试断言强度、L1/L2 低风险），不构成合并阻断。

## 二、H1/H2 回修核实结果

### ✅ H1（建唯一索引导致 duplicate entry 中断迁移）— 已解决

- **核实**：`110_restore_campus_steps.sql` 中已【完全移除】`CREATE UNIQUE INDEX`，
  全文件不含任何 `CREATE [UNIQUE] INDEX` 语句。幂等去重改为 12 条独立的
  `INSERT INTO ... SELECT ... WHERE NOT EXISTS (...)` 守卫。
- **结论**：不再建索引 → 不存在“建索引阶段因重复 step_order 触发 Duplicate entry”
  的可能，根因消除。该方案天然容忍表内已存在同 (campus_id, step_order) 的多 status 行，
  也不会因重复数据报错。**H1 已解决。**

### ✅ H2（唯一索引 (campus_id, step_order) 不含 status 破坏审核流）— 已解决

- **核实**：不再建任何唯一/普通索引，因此不再对表施加 (campus_id, step_order)
  业务唯一约束。`campus_checkin_steps` 表维持 048 至今的“仅 id 主键”结构，
  draft → pending_review → published 多状态行可在同 step_order 下共存。
- **结论**：审核流多版本并存不再受限。**H2 已解决。**

### 额外确认：`WHERE NOT EXISTS` 判定语义正确

- 守卫条件为 `campus_id=? AND step_order=? AND status='published'`，即**只判断
  published 种子是否已存在**：
  - 已存在 published → 不重复插入（幂等）。
  - 仅存在 draft / pending_review → 判定不命中 → **仍会插入独立的 published 种子**，
    与业务“多状态共存”一致，且不删除、不覆盖既有 draft/pending_review。
  - 已存在 published（管理员重建后）→ 不覆盖管理员修正值，只补齐缺失的其余种子。
- **语义正确**，同时满足“只补齐缺失 published、不动 draft/pending_review、不覆盖已有 published”。

## 三、方言转换（ToMySQL）逐项核对

重点核实 `INSERT INTO ... SELECT 'huifeng',1,... WHERE NOT EXISTS(...)` 在 SQLite 与
MySQL（经 `ToMySQL`）下的合法性：

1. **`SELECT 常量列表`（无 FROM）合法性**：MySQL 8 支持无 `FROM` 的常量 `SELECT`
   （`SELECT 1, 'a'` 等价于 `SELECT 1, 'a' FROM DUAL`），等价于 DUAL；SQLite 亦支持。
   **合法。** ✅
2. **方言转换是否误伤该语句**：逐条审阅 `dialect.go` 的 `ToMySQL`：
   - `integerRe`（`\bINTEGER\b`→BIGINT）只作用于列定义关键字，110 迁移文件为纯
     `INSERT...SELECT` 数据语句，不含 INTEGER/INTEGER 列定义，不误伤。
   - `textDefaultRe`/`plainTextRe` 同理作用于列定义，不影响 SELECT 常量。
   - `insertOrIgnoreRe`（`INSERT OR IGNORE`）在本文件已无匹配（已移除 IGNORE）。
   - `insertColsRe` 只处理 INSERT 列清单中的 key/value/rank 保留字，本文件的列清单
     （campus_id, step_order, title, ... status）不含这三者，不误改。
   - 无 `datetime('now')`、无 `ALTER`、无 `||`、无 PRAGMA。**转换不误伤。** ✅
3. **测试覆盖**：`migration110_campus_test.go` 新增 `TestToMySQL_Migration110_RestoreCampusSteps`
   明确断言“不得含 CREATE INDEX / CREATE UNIQUE INDEX / INSERT OR IGNORE”，
   并断言 `WHERE NOT EXISTS` 出现 12 次、`INSERT INTO campus_checkin_steps` 12 次。
   覆盖到位。✅

## 四、测试充分性

- **`migration110_campus_test.go`**：断言反转为“不得含索引”“必须含 12 条 WHERE NOT EXISTS
  守卫”“不得含 DELETE”“不得含 INSERT OR IGNORE”，新增 `stripSQLComments` 辅助剥离注释，
  避免注释中引用的 “CREATE UNIQUE INDEX” 字样被误判。设计合理。✅
- **`migration110_functional_test.go`（新增）**：在内存 SQLite 上重建 048 schema →
  模拟 079 清空 → 播种“同 step_order 的 published + draft、以及 pending_review” →
  执行 110 → 断言 published 恢复 12 条、既有 draft/pending_review 不删、重复执行幂等。
  **直接覆盖 H1 + H2 的核心场景**，功能回归测试强度充分。✅

## 五、发现的问题清单（复审后，无 High 级）

### 🟠 中（Medium）

#### M2. 前端进入编辑模式仍【未提前拦截】`_remoteIds.isEmpty`（未回修）

- **位置**：`frontend/lib/pages/campus/campus_map_page.dart` `_toggleNodeEditing` 的
  `!_editMode` 分支。
- **现状（核实）**：进入编辑时仍是
  `setState(() => _editMode = true)` + 提示“拖动标注校正位置”，**未在进入前检查
  `_remoteIds.isEmpty`**。`_remoteIds.isEmpty` 的拦截只发生在**退出编辑**时
  （`_savePendingCoordinates` / 退出 SnackBar）才提示“未加载到后端节点”。
- **影响**：中等（体验/误导，非数据损坏）。管理员在后端无节点数据时仍能进入编辑模式、
  拖动假节点，退出时才被告知白忙——与上一轮 M2 完全一致，**该点未见回修**。
  （注：dev 本轮回修范围是迁移 110，未承诺改前端，故标注为残留 Medium，不阻断。）
- **建议**：进入编辑模式前增加 `if (_remoteIds.isEmpty) { 提示并 return; }`。

#### M3. 测试断言强度（部分相关性已改善，但仍有 1 处结构性脆弱）

- **改善**：上一轮 M3 指出的“唯一索引断言只验文本存在、不验列集合”问题，已随
  “不建索引”方案的采用而自然消除（现在断言的是“不得含索引”）。
- **残留**：`TestToMySQL_Migration110_RestoreCampusSteps` 用
  `strings.Count(upper, "WHERE NOT EXISTS") == 12`、`strings.Count(..., "INSERT INTO ...") == 12`
  等文本计数断言。当前正确，但属“计数锁死”式断言：未来若在 110 文件追加其它
  INSERT 或 WHERE NOT EXISTS（如回填别的表），计数断言需同步维护，否则误报。
  属维护性脆弱点，非当前缺陷。
- **影响**：中等（测试维护性）。

### 🟡 低（Low）

#### L1. 方言转换脆弱点（残留）

- 上一轮 L1 关于 `createIndexRe` 多行锚点的脆弱性，随“不建索引”方案已不再相关
  （110 无任何 CREATE INDEX）。但 `ToMySQL` 的总体转换仍是正则驱动，`integerRe` 全词
  替换 `INTEGER` 等存在字符串内容误伤的理论可能。当前 110 无此风险。保留低风险提示，
  建议后续对含 `INSERT...SELECT...WHERE NOT EXISTS` 的样例做一次端到端 MySQL 实测
  （当前仅在 SQLite + 字符串断言层面验证，未在真实 MySQL 引擎上跑过）。
- **影响**：低。

#### L2. SnackBar 文案无 campus 上下文（残留，未回修）

- 与上一轮 L2 一致，未改动。影响极小。

## 六、结论与建议汇总

| 编号 | 原级别 | 复审后状态 | 说明 |
| --- | --- | --- | --- |
| H1 | 高 | ✅ 已解决 | 移除唯一索引，改为 12×WHERE NOT EXISTS，消除建索引中断风险 |
| H2 | 高 | ✅ 已解决 | 不再施加 (campus_id, step_order) 唯一约束，审核流多版本共存不受限 |
| M1 | 中 | ✅ 已消除 | 该问题（INSERT OR IGNORE 静默吞已存在节点）随 IGNORE 方案废弃而消失 |
| M2 | 中 | ⚠️ 残留 | 前端进入编辑模式仍未提前拦截 `_remoteIds.isEmpty` |
| M3 | 中 | ⚠️ 部分 | 唯一索引断言已随方案消除；但 12 条计数断言偏脆弱 |
| L1 | 低 | ⚠️ 残留 | 方言转换整体正则驱动，未在真实 MySQL 实测 |
| L2 | 低 | ⚠️ 残留 | SnackBar 文案无 campus 上下文 |

**最终结论：通过（Pass）。**

H1/H2 两个高危问题均已彻底解决，方案（不建索引 + WHERE NOT EXISTS 守卫）在语义、幂等性、
审核流兼容性、双方言合法性上均正确，且有功能回归测试直接覆盖核心场景。剩余 M2/M3/L1/L2
均为中低优先级，建议但不阻断合并；其中 M2（前端编辑模式提前拦截）建议在后续迭代中一并处理。

## 七、复审新增风险点检查

1. **无新的 High 级风险**：`WHERE NOT EXISTS` 方案不引入 DELETE、不建索引、不施加约束，
   不会在 MySQL/SQLite 产生新的数据丢失或锁/约束风险。
2. **MySQL 并行执行的竞态（理论）**：`WHERE NOT EXISTS` 非原子防重（非唯一约束），
   理论上并发同时执行 110 可能各插一份，但迁移在启动初始化阶段单连接顺序执行，
   无并发，风险可忽略。
3. **`SELECT` 常量列数与列清单一致**：逐条核对 12 条 INSERT 的列清单（13 列）与
   SELECT 常量（13 个值）一一对应，无列数不匹配。✅
4. **执行计划/性能**：`WHERE NOT EXISTS` 子查询在无索引的 `campus_checkin_steps` 上
   为全表扫描判定，但该表仅十余行，性能无影响。✅
