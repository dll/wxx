-- 063_feedback_module.sql
-- 反馈新增「所属模块」字段，用于管理员在线修复时快速定位代码。
ALTER TABLE feedback ADD COLUMN module TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_feedback_module ON feedback(module);