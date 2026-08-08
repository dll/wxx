-- 060_health_records.sql — 学生「身体健康」模块
-- 三张表：身体基本信息（每用户一行）、体检记录、病历记录。
-- 均为学生本人数据，通过 user_id 归属，仅本人可读写（RBAC 由 handler 层控制）。

-- 身体基本信息（用户一行）
CREATE TABLE IF NOT EXISTS health_basic_info (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id           INTEGER NOT NULL UNIQUE REFERENCES users(id),
    height_cm         REAL    DEFAULT 0,
    weight_kg         REAL    DEFAULT 0,
    blood_type        TEXT    DEFAULT '',
    vision_left       TEXT    DEFAULT '',
    vision_right      TEXT    DEFAULT '',
    allergies         TEXT    DEFAULT '',   -- 过敏史
    past_illness      TEXT    DEFAULT '',   -- 既往病史
    family_history    TEXT    DEFAULT '',   -- 家族病史
    emergency_contact TEXT    DEFAULT '',   -- 紧急联系人
    emergency_phone   TEXT    DEFAULT '',   -- 紧急联系电话
    created_at        TEXT    NOT NULL DEFAULT (datetime('now', 'localtime')),
    updated_at        TEXT    NOT NULL DEFAULT (datetime('now', 'localtime'))
);

-- 体检记录
CREATE TABLE IF NOT EXISTS health_checkups (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    checkup_date TEXT    DEFAULT '',   -- 体检日期 yyyy-MM-dd
    hospital    TEXT    DEFAULT '',   -- 体检机构
    conclusion  TEXT    DEFAULT '',   -- 体检结论（正常/异常等）
    details     TEXT    DEFAULT '',   -- 体检详情/指标说明
    attachments TEXT    DEFAULT '[]', -- 附件（图片 URL 列表，JSON）
    created_at  TEXT    NOT NULL DEFAULT (datetime('now', 'localtime'))
);

-- 病历记录
CREATE TABLE IF NOT EXISTS health_records (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER NOT NULL REFERENCES users(id),
    record_date  TEXT    DEFAULT '',   -- 就诊日期 yyyy-MM-dd
    hospital     TEXT    DEFAULT '',   -- 就诊医院
    department   TEXT    DEFAULT '',   -- 科室
    diagnosis    TEXT    DEFAULT '',   -- 诊断
    treatment    TEXT    DEFAULT '',   -- 治疗方案/用药
    attachments  TEXT    DEFAULT '[]', -- 附件（图片 URL 列表，JSON）
    created_at   TEXT    NOT NULL DEFAULT (datetime('now', 'localtime'))
);

CREATE INDEX IF NOT EXISTS idx_health_checkups_user ON health_checkups(user_id);
CREATE INDEX IF NOT EXISTS idx_health_records_user ON health_records(user_id);
