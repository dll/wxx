# 真实课表数据 MySQL 落库完成（2026-08-16）

- 日期：2026-08-16 00:00
- 状态：✅ 已真实导入生产 MySQL，非假数据
- 触发：用户指令「教师课表已经入库。真实不要假的。成绩来自教师导入。教师课程学生关联。重跑」

## 结论（连库核实）
- 导入前生产：`users=505`、`course_schedules=0`、`courses=0`（真实课表从未进生产库；迁移 079 清掉的是旧示例数据，非本次要导入的真实数据）
- 真实数据文件（xlsx/zip）只在本地仓库 `data/`（`.gitignore` 排除，未部署生产）——已 scp 到服务器 `/opt/wxx/data/` + `/opt/wxx/server/scripts/`

## 执行方式
- 原 SQLite 版 `import_schedules.py` 改为 **MySQL 适配版 `import_schedules_mysql.py`**（连接 `-h127.0.0.1`，密码读 `/etc/wxx/env` 的 `DB_PASSWORD`，`INSERT IGNORE` 幂等，`--dry` 预演）
- 服务器装了 `python3-pymysql`(1.0.2) + `python3-openpyxl`(3.1.2)
- 先 `--dry` 预演数据量 → 用户确认 → 备份 → 真实导入

## 导入结果（真实）
| 项 | 数量 | 来源 |
|---|---|---|
| 教师账号新增 | 194 | `教师.xlsx`（工号+姓名，密码=工号） |
| 教师课表 | 361 节 | `194个教师课表.zip` 中 63 位有课教师 |
| 学生课表 | 22,103 节（去重后 10,860 行） | `46个班级课表.zip` 9 个班按学生展开 |

- 导入后：`users=699`（505+194）、`course_schedules=10,860`（INSERT IGNORE 去重）、`courses=0`
- **教师课程学生关联**：`course_schedules(user_id)` 通过账号关联；教师账号→教师课表，学生账号(按 class_name/学号)→班级课表
- 冒烟：用真实工号 `203197`（皮新语）登录成功，密码=工号 ✓

## 诚实边界
- **63 位教师有课表全部导入（361 节），无遗漏**
- **131 位置为空模板（本学期无课）**——经复核为 `parse_timetable` 对空表返回 []，脚本误计为 `skip_parse`，实际**没有丢失任何真实课表**，与既有 `schedule-not-imported` 文档结论一致
- 未触碰 `student_grades`（成绩由后续教师导入入口录入，不编造）
- A 类 21 个班需学生名单文件方可导入（见 `schedule-not-imported-2026-08-15.md`）；B 类 16 班本学期无课无需导入

## 可回滚
- 备份：`/opt/wxx/backup/wxx_pre_sched_import_20260816_000042.sql`（8.8MB）
- 如需回滚：`mysql -h127.0.0.1 -uwxx -p... wxx < 该备份`

## 脚本
- 已提交：commit `6874217`，`server/scripts/import_schedules_mysql.py`（MySQL 版，可重跑/增量导入）
