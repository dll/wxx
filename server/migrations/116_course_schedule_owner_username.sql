-- 116_course_schedule_owner_username.sql
-- 课表归属追溯（2026-09-01 彻底修复「显示的课程不对」）：
-- 历史课表按 user_id 导入，填错即挂错账号。本迁移新增 owner_username 列，
-- 记录课表本应归属的学号/工号（导入时由 ScheduleRow.Username 写入），
-- 供按工号归位接口（POST /admin/schedules/reassign-by-username）修正历史错挂数据。
ALTER TABLE course_schedules ADD COLUMN owner_username TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_course_schedules_owner_username ON course_schedules(owner_username);
