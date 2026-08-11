-- 069_user_contact_fields.sql — 用户联系方式字段
-- 幂等：execSQL 已处理 ADD COLUMN 重复列错误（跳过而不报错）
ALTER TABLE users ADD COLUMN phone TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN wechat TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN qq TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN email TEXT DEFAULT '';
