-- 021_add_step_contact_fields.sql — 给 process_steps 表补充步骤级详细信息字段
-- 修复审核问题 §4.1：联系人/电话/办公时间/FAQ 四类信息恒为空
-- 前端 ProcessStepDetail 已定义并渲染这些字段，后端补齐表结构、实体与种子数据

-- 1. 新增列（SQLite 不支持 ADD COLUMN IF NOT EXISTS，用 ALTER TABLE ADD COLUMN）
ALTER TABLE process_steps ADD COLUMN contact      TEXT NOT NULL DEFAULT '';
ALTER TABLE process_steps ADD COLUMN phone        TEXT NOT NULL DEFAULT '';
ALTER TABLE process_steps ADD COLUMN office_hours TEXT NOT NULL DEFAULT '';
ALTER TABLE process_steps ADD COLUMN faq          TEXT NOT NULL DEFAULT '[]';  -- JSON 数组：[{"q":"…","a":"…"}]
