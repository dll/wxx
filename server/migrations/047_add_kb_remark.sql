-- 为 kb_resources 添加备注字段
ALTER TABLE kb_resources ADD COLUMN remark TEXT NOT NULL DEFAULT '';
