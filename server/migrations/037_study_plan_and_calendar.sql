-- 037_study_plan_and_calendar.sql — 学习计划与校历模块
-- 包含：校历/学期、校历事件、学生课表、学习计划、学习计划任务
-- 配合 study_plan_handler.go 提供 /api/v1/study/calendar|timetable|plans 系列接口

-- ============================================
-- 一、校历/学期表
-- ============================================

CREATE TABLE IF NOT EXISTS academic_calendars (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    academic_year INTEGER NOT NULL,          -- 学年如2026
    semester INTEGER NOT NULL,              -- 1第一学期/2第二学期
    semester_code TEXT NOT NULL UNIQUE,     -- "2025-2026-2"
    semester_name TEXT NOT NULL,            -- "2025-2026学年第二学期"
    start_date TEXT NOT NULL,               -- 开学日期
    end_date TEXT NOT NULL,                 -- 放假日期
    register_date TEXT,                     -- 报到注册日
    total_weeks INTEGER NOT NULL DEFAULT 20,
    week_start_day TEXT DEFAULT 'monday',
    status TEXT DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(academic_year, semester)
);

-- ============================================
-- 二、校历事件表
-- ============================================

CREATE TABLE IF NOT EXISTS academic_calendar_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    semester_code TEXT NOT NULL,
    event_name TEXT NOT NULL,
    event_type TEXT NOT NULL,    -- holiday/exam/activity/deadline/register/break
    start_date TEXT NOT NULL,
    end_date TEXT,
    week_no INTEGER,
    affects_classes INTEGER DEFAULT 0,
    description TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_calendar_events_semester ON academic_calendar_events(semester_code);
CREATE INDEX IF NOT EXISTS idx_calendar_events_date ON academic_calendar_events(start_date, end_date);

-- ============================================
-- 三、学生课表表
-- ============================================

CREATE TABLE IF NOT EXISTS course_schedules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    course_id TEXT NOT NULL,
    course_name TEXT NOT NULL,
    semester_code TEXT NOT NULL,
    weekday INTEGER NOT NULL,        -- 1-7
    start_period INTEGER NOT NULL,   -- 第几节课
    end_period INTEGER NOT NULL,
    weeks_pattern TEXT NOT NULL DEFAULT '1-20',  -- 上课周次
    location TEXT,
    teacher TEXT,
    color TEXT DEFAULT '#1565C0',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, course_id, weekday, start_period, semester_code)
);

CREATE INDEX IF NOT EXISTS idx_course_schedules_user_semester ON course_schedules(user_id, semester_code);
CREATE INDEX IF NOT EXISTS idx_course_schedules_weekday ON course_schedules(weekday);

-- ============================================
-- 四、学习计划表
-- ============================================

CREATE TABLE IF NOT EXISTS study_plans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    plan_type TEXT NOT NULL DEFAULT 'weekly',  -- weekly/monthly/quarterly/semester/yearly/four_year
    semester_code TEXT,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    goals_json TEXT NOT NULL DEFAULT '[]',     -- JSON数组
    progress REAL DEFAULT 0,
    ai_generated INTEGER DEFAULT 0,
    status TEXT DEFAULT 'active',              -- active/completed/archived
    linked_plan_id INTEGER,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_study_plans_user ON study_plans(user_id);
CREATE INDEX IF NOT EXISTS idx_study_plans_type ON study_plans(plan_type);
CREATE INDEX IF NOT EXISTS idx_study_plans_status ON study_plans(status);

-- ============================================
-- 五、学习计划任务表
-- ============================================

CREATE TABLE IF NOT EXISTS study_plan_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_id INTEGER NOT NULL,
    course_id TEXT,
    course_name TEXT,
    title TEXT NOT NULL,
    description TEXT,
    scheduled_date TEXT,
    scheduled_duration INTEGER DEFAULT 0,   -- 计划时长(分钟)
    actual_duration INTEGER DEFAULT 0,
    status TEXT DEFAULT 'pending',          -- pending/in_progress/done/skipped
    evidence TEXT,
    reflection TEXT,
    sort_order INTEGER DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (plan_id) REFERENCES study_plans(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_study_plan_tasks_plan ON study_plan_tasks(plan_id);
CREATE INDEX IF NOT EXISTS idx_study_plan_tasks_status ON study_plan_tasks(status);
CREATE INDEX IF NOT EXISTS idx_study_plan_tasks_date ON study_plan_tasks(scheduled_date);

-- ============================================
-- 六、种子数据：校历（基于真实中国高校校历）
-- ============================================

-- 2025-2026第二学期（春季，2月-7月）
INSERT OR IGNORE INTO academic_calendars (academic_year, semester, semester_code, semester_name, start_date, end_date, register_date, total_weeks, status)
VALUES (2026, 2, '2025-2026-2', '2025-2026学年第二学期', '2026-02-23', '2026-07-05', '2026-02-22', 20, 'completed');

-- 2026-2027第一学期（秋季，9月-次年1月）
INSERT OR IGNORE INTO academic_calendars (academic_year, semester, semester_code, semester_name, start_date, end_date, register_date, total_weeks, status)
VALUES (2026, 1, '2026-2027-1', '2026-2027学年第一学期', '2026-09-01', '2027-01-16', '2026-08-31', 20, 'upcoming');

-- ============================================
-- 七、种子数据：校历事件
-- ============================================

INSERT OR IGNORE INTO academic_calendar_events (semester_code, event_name, event_type, start_date, end_date, week_no, affects_classes, description) VALUES
-- 2025-2026-2学期事件
('2025-2026-2', '学生报到注册', 'register', '2026-02-22', '2026-02-22', 0, 0, '老生报到注册'),
('2025-2026-2', '正式上课', 'activity', '2026-02-23', '2026-02-23', 1, 0, '第二学期正式上课'),
('2025-2026-2', '清明节放假', 'holiday', '2026-04-04', '2026-04-06', 7, 1, '清明假期'),
('2025-2026-2', '劳动节放假', 'holiday', '2026-05-01', '2026-05-05', 10, 1, '五一假期'),
('2025-2026-2', '期中考试周', 'exam', '2026-04-13', '2026-04-19', 8, 1, '期中考试'),
('2025-2026-2', '端午节放假', 'holiday', '2026-06-19', '2026-06-21', 17, 1, '端午假期'),
('2025-2026-2', '期末考试周', 'exam', '2026-06-29', '2026-07-05', 19, 1, '期末考试'),
('2025-2026-2', '暑假开始', 'break', '2026-07-06', '2026-08-31', 20, 0, '暑假'),
-- 2026-2027-1学期事件
('2026-2027-1', '新生报到', 'register', '2026-08-30', '2026-08-30', 0, 0, '2026级新生报到'),
('2026-2027-1', '老生报到注册', 'register', '2026-08-31', '2026-08-31', 0, 0, '老生报到注册'),
('2026-2027-1', '正式上课', 'activity', '2026-09-01', '2026-09-01', 1, 0, '第一学期正式上课'),
('2026-2027-1', '中秋节放假', 'holiday', '2026-09-25', '2026-09-27', 4, 1, '中秋假期'),
('2026-2027-1', '国庆节放假', 'holiday', '2026-10-01', '2026-10-07', 5, 1, '国庆假期'),
('2026-2027-1', '期中考试周', 'exam', '2026-11-02', '2026-11-08', 9, 1, '期中考试'),
('2026-2027-1', '元旦放假', 'holiday', '2027-01-01', '2027-01-03', 18, 1, '元旦假期'),
('2026-2027-1', '期末考试周', 'exam', '2027-01-11', '2027-01-16', 20, 1, '期末考试'),
('2026-2027-1', '寒假开始', 'break', '2027-01-17', '2027-02-28', 20, 0, '寒假');

-- ============================================
-- 八、种子数据：user_id=1 的示例课表
-- ============================================

INSERT OR IGNORE INTO course_schedules (user_id, course_id, course_name, semester_code, weekday, start_period, end_period, weeks_pattern, location, teacher, color) VALUES
(1, 'CS101', '高等数学', '2025-2026-2', 1, 1, 2, '1-18', '理工楼A201', '张教授', '#1565C0'),
(1, 'CS101', '高等数学', '2025-2026-2', 3, 1, 2, '1-18', '理工楼A201', '张教授', '#1565C0'),
(1, 'CS102', '大学英语', '2025-2026-2', 2, 3, 4, '1-18', '文科楼B105', '李老师', '#2E7D32'),
(1, 'CS102', '大学英语', '2025-2026-2', 4, 3, 4, '1-18', '文科楼B105', '李老师', '#2E7D32'),
(1, 'CS103', '数据结构', '2025-2026-2', 1, 3, 4, '1-18', '信息楼C301', '王教授', '#E65100'),
(1, 'CS103', '数据结构', '2025-2026-2', 5, 1, 2, '1-18', '信息楼C301', '王教授', '#E65100'),
(1, 'CS104', '线性代数', '2025-2026-2', 2, 1, 2, '1-18', '理工楼A203', '赵教授', '#7B1FA2'),
(1, 'CS104', '线性代数', '2025-2026-2', 4, 1, 2, '1-18', '理工楼A203', '赵教授', '#7B1FA2'),
(1, 'CS105', '计算机组成原理', '2025-2026-2', 3, 5, 6, '1-16', '信息楼C201', '陈教授', '#00695C'),
(1, 'CS105', '计算机组成原理', '2025-2026-2', 5, 3, 4, '1-16', '信息楼C201', '陈教授', '#00695C');
