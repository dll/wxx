-- 084: 活动复盘指标 —— 报名签到字段 + 到场统计
-- 学生会「活动复盘」：需要"到场"数据才能算 到场率/成功率。

-- 报名表增加到场标记（0=未到，1=已签到）
ALTER TABLE health_activity_signups ADD COLUMN attended INTEGER NOT NULL DEFAULT 0;
