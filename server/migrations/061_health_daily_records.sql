-- 061_health_daily_records.sql — 学生「身体健康」日常记录
-- 按日记录身高/体重/血压/心率，前端折线图可视化趋势。
CREATE TABLE IF NOT EXISTS health_daily_records (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER NOT NULL REFERENCES users(id),
    record_date  TEXT    NOT NULL,   -- 记录日期 yyyy-MM-dd（同用户同日唯一）
    height_cm    REAL    DEFAULT 0,  -- 身高(cm)
    weight_kg    REAL    DEFAULT 0,  -- 体重(kg)
    systolic     INTEGER DEFAULT 0,  -- 收缩压(mmHg)
    diastolic    INTEGER DEFAULT 0,  -- 舒张压(mmHg)
    heart_rate   INTEGER DEFAULT 0,  -- 心率(bpm)
    note         TEXT    DEFAULT '', -- 备注
    created_at   TEXT    NOT NULL DEFAULT (datetime('now', 'localtime')),
    updated_at   TEXT    NOT NULL DEFAULT (datetime('now', 'localtime')),
    UNIQUE(user_id, record_date)
);

CREATE INDEX IF NOT EXISTS idx_health_daily_user_date ON health_daily_records(user_id, record_date);
