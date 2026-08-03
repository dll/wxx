-- 051: 阶段二 去演示化 — 真实数据三表（积分 / 问答广场 / 谈心记录）
-- 目的：让积分成就、问答广场、谈心谈话从写死数据升级为真实落库

-- 积分明细
CREATE TABLE IF NOT EXISTS student_points (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL,
    points     INTEGER NOT NULL DEFAULT 0,
    reason     TEXT    NOT NULL DEFAULT '',
    source     TEXT    NOT NULL DEFAULT '',          -- checkin / qa_post / qa_answer / party / other
    created_at TEXT    NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_student_points_user ON student_points(user_id, id DESC);

-- 问答广场帖子
CREATE TABLE IF NOT EXISTS qa_posts (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL,
    title      TEXT    NOT NULL,
    content    TEXT    NOT NULL DEFAULT '',
    category   TEXT    NOT NULL DEFAULT '综合',
    answers    INTEGER NOT NULL DEFAULT 0,
    views      INTEGER NOT NULL DEFAULT 0,
    status     TEXT    NOT NULL DEFAULT 'published', -- published / hidden
    created_at TEXT    NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at TEXT    NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_qa_posts_created ON qa_posts(created_at DESC);

-- 问答广场回答
CREATE TABLE IF NOT EXISTS qa_answers (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id    INTEGER NOT NULL,
    user_id    INTEGER NOT NULL,
    content    TEXT    NOT NULL,
    adopted    INTEGER NOT NULL DEFAULT 0,
    created_at TEXT    NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_qa_answers_post ON qa_answers(post_id);

-- 谈心谈话记录
CREATE TABLE IF NOT EXISTS talk_records (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    counselor_id INTEGER NOT NULL,
    student_id   INTEGER NOT NULL DEFAULT 0,
    student_name TEXT    NOT NULL DEFAULT '',
    topic        TEXT    NOT NULL DEFAULT '',
    emotion      TEXT    NOT NULL DEFAULT '',
    content      TEXT    NOT NULL DEFAULT '',
    summary      TEXT    NOT NULL DEFAULT '',
    follow_ups   TEXT    NOT NULL DEFAULT '[]',      -- JSON 数组字符串
    status       TEXT    NOT NULL DEFAULT 'following', -- following / resolved
    created_at   TEXT    NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at   TEXT    NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_talk_records_counselor ON talk_records(counselor_id, id DESC);
