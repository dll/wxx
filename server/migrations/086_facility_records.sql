-- 086: 后勤服务台 —— 后勤服务记录表
-- 教辅角色后勤保障类工作（实验室开门/关门、教室保洁卫生、热水供应、
-- 宿舍晚归查岗、校园环卫学习环境、图书馆借阅管理）的真实服务记录。
-- 全部为操作人手动登记（data_source=real，非参考/编造），并接入绩效画像。

CREATE TABLE IF NOT EXISTS facility_records (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  role          TEXT NOT NULL,                -- 岗位类型: lab/clean/hotwater/dorm/envir/library
  title         TEXT NOT NULL,                -- 事项简述，如「实验楼A 301 开门」
  location      TEXT NOT NULL DEFAULT '',     -- 地点: 实验楼A/3号宿舍楼/图书馆2楼...
  detail        TEXT NOT NULL DEFAULT '',     -- 详情/数量/备注
  operator_id   INTEGER NOT NULL,             -- 登记人(教辅用户)
  operator_name TEXT NOT NULL DEFAULT '',
  student_id    INTEGER NOT NULL DEFAULT 0,   -- 关联学生(0=无)。如借阅人/查岗对象
  student_name  TEXT NOT NULL DEFAULT '',
  occurred_at   TEXT NOT NULL,                -- 服务发生时间(ISO)
  created_at    TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  data_source   TEXT NOT NULL DEFAULT 'real'  -- 真实登记，非参考/编造
);

CREATE INDEX IF NOT EXISTS idx_facility_operator ON facility_records(operator_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_facility_role ON facility_records(role, occurred_at);
