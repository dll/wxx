-- 062_health_activities.sql — 健身活动 / 竞技比赛（与校园文化活动数据链通）
-- 学生会等组织可发起活动（如新生杯足球赛），学生可关注（喜欢）与报名参加。
CREATE TABLE IF NOT EXISTS health_activities (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    activity_id  TEXT    NOT NULL UNIQUE,   -- 业务 ID，如 act-freshman-cup-2026
    title        TEXT    NOT NULL,          -- 活动名称
    category     TEXT    NOT NULL DEFAULT 'sports', -- sports健身/竞技/culture文体/other
    description  TEXT    DEFAULT '',        -- 活动介绍
    start_at     TEXT    DEFAULT '',        -- 开始时间
    end_at       TEXT    DEFAULT '',        -- 结束时间
    venue        TEXT    DEFAULT '',        -- 地点
    organizer    TEXT    DEFAULT '',        -- 组织方（如 学生会体育部）
    capacity     INTEGER DEFAULT 0,         -- 名额上限，0=不限
    signup_deadline TEXT DEFAULT '',        -- 报名截止
    status       TEXT    NOT NULL DEFAULT 'active', -- active/closed/ended
    creator_id   INTEGER DEFAULT 0,         -- 发起人 user_id（学生会）
    creator_role TEXT    DEFAULT '',        -- 发起人角色（student_union 等）
    created_at   TEXT    NOT NULL DEFAULT (datetime('now', 'localtime'))
);

-- 学生关注（喜欢）
CREATE TABLE IF NOT EXISTS health_activity_favorites (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL,
    activity_id TEXT    NOT NULL,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now', 'localtime')),
    UNIQUE(user_id, activity_id)
);

-- 学生报名参加
CREATE TABLE IF NOT EXISTS health_activity_signups (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL,
    activity_id TEXT    NOT NULL,
    status      TEXT    NOT NULL DEFAULT 'registered', -- registered/cancelled
    created_at  TEXT    NOT NULL DEFAULT (datetime('now', 'localtime')),
    UNIQUE(user_id, activity_id)
);

CREATE INDEX IF NOT EXISTS idx_ha_category ON health_activities(category);
CREATE INDEX IF NOT EXISTS idx_ha_status ON health_activities(status);
CREATE INDEX IF NOT EXISTS idx_haf_user ON health_activity_favorites(user_id);
CREATE INDEX IF NOT EXISTS idx_has_user ON health_activity_signups(user_id);

-- 种子活动（健身 / 竞技 / 文体，与校园文化 Events 数据链通）
INSERT OR IGNORE INTO health_activities
  (activity_id, title, category, description, start_at, end_at, venue, organizer, capacity, signup_deadline, status, creator_role)
VALUES
  ('act-xinshengbei-football-2026', '新生杯足球赛', 'sports',
   '2026级新生足球友谊赛，学生会体育部主办，按学院组队参赛。', '2026-09-15 14:00', '2026-10-10 17:00',
   '会峰校区足球场', '学生会体育部', 120, '2026-09-12', 'active', 'student_union'),
  ('act-xinshengbei-basketball-2026', '新生杯篮球赛', 'sports',
   '新生篮球交流赛，以班级/学院为单位报名，优胜队伍获荣誉证书。', '2026-09-20 16:00', '2026-10-20 18:00',
   '琅琊校区篮球场', '学生会体育部', 150, '2026-09-18', 'active', 'student_union'),
  ('act-morning-run-2026', '晨跑打卡计划', 'fitness',
   '健康晨跑打卡，连续打卡30天可获体育健康之星称号。', '2026-09-01 06:30', '2026-11-30 07:30',
   '田径场', '学生会·体育健康部', 0, '', 'active', 'student_union'),
  ('act-yoga-2026', '周末瑜伽体验课', 'fitness',
   '减压瑜伽入门课，由校瑜伽社协办，限报30人。', '2026-09-14 09:00', '2026-09-14 10:30',
   '大学生活动中心102', '校瑜伽社', 30, '2026-09-13', 'active', 'student_union'),
  ('act-tabletennis-2026', '乒乓球个人赛', 'sports',
   '校园乒乓球单打锦标赛，分男单/女单，面向全体在校生。', '2026-10-08 15:00', '2026-10-22 18:00',
   '体育馆乒乓球室', '学生会体育部', 80, '2026-10-06', 'active', 'student_union'),
  ('act-marathon-2026', '秋季校园迷你马拉松', 'fitness',
   '5公里校园迷你马拉松，完赛即获纪念奖牌与体育积分。', '2026-11-08 08:00', '2026-11-08 11:00',
   '校园环线', '校团委·体育部', 300, '2026-11-05', 'active', 'student_union');

