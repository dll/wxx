-- 042_add_step_geo_media_fields.sql — 打通办事流程步骤六类信息剩余字段（RF-01）
--
-- 021 已补 contact/phone/office_hours/faq；本迁移补齐 RF-01 要求的其余三类：
--   contact_wechat — 联系人微信（部分部门以企业微信/个人微信办理咨询）
--   geo_lat/geo_lng — 办理地点经纬度（供前端地图导航到办理窗口）
--   media_urls — 办理指引媒体资源 JSON 数组（图片/视频 URL，如窗口位置示意图）
--
-- 幂等：execSQL 已处理 ADD COLUMN 重复列错误（跳过），冷启动可重复执行。
-- 说明：仅打通字段承载链路，不注入虚构坐标/图片；待运营录入真实素材后即生效。

ALTER TABLE process_steps ADD COLUMN contact_wechat TEXT NOT NULL DEFAULT '';
ALTER TABLE process_steps ADD COLUMN geo_lat        REAL NOT NULL DEFAULT 0;
ALTER TABLE process_steps ADD COLUMN geo_lng        REAL NOT NULL DEFAULT 0;
ALTER TABLE process_steps ADD COLUMN media_urls     TEXT NOT NULL DEFAULT '[]';  -- JSON 数组：["https://.../step1.jpg"]
