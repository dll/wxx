-- ============================================================
-- 024_student_features.sql
-- 学科竞赛 + 大学规划 + 入党教育 + 社团生活
-- ============================================================

-- ══════════════════════════════════════════════════════════════
-- 一、学科竞赛
-- ══════════════════════════════════════════════════════════════

-- 1. 竞赛信息表
CREATE TABLE IF NOT EXISTS competitions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,                          -- 竞赛名称
    level TEXT NOT NULL,                         -- 级别：national(国家级)/provincial(省级)/municipal(市级)/school(校级)
    category TEXT NOT NULL,                      -- 类别：programming(程序设计)/math(数学)/electronics(电子)/innovation(创新创业)/english(英语)/other
    organizer TEXT,                              -- 主办方
    description TEXT,                            -- 竞赛简介
    requirements TEXT,                           -- 参赛条件
    features TEXT,                               -- 竞赛特点（JSON数组）
    registration_start TEXT,                     -- 报名开始时间
    registration_end TEXT,                       -- 报名截止时间
    competition_date TEXT,                       -- 比赛时间
    result_date TEXT,                            -- 公布结果时间
    website TEXT,                                -- 官网链接
    resource_links TEXT,                         -- 备赛资源链接（JSON数组）
    max_team_size INTEGER DEFAULT 1,             -- 最大团队人数
    is_team_competition INTEGER DEFAULT 0,       -- 是否团队赛
    status TEXT DEFAULT 'upcoming',              -- 状态：upcoming/registration/open/finished
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 2. 竞赛报名表
CREATE TABLE IF NOT EXISTS competition_registrations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    competition_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    student_id TEXT,
    student_name TEXT,
    college TEXT,
    major TEXT,
    class_name TEXT,
    team_name TEXT,                              -- 团队名称
    team_members TEXT,                           -- 团队成员（JSON数组）
    advisor_name TEXT,                           -- 指导教师
    status TEXT DEFAULT 'registered',            -- 状态：registered/submitted/awarded/not_awarded
    work_title TEXT,                             -- 作品标题
    work_description TEXT,                       -- 作品描述
    work_file_url TEXT,                          -- 作品文件URL
    award_level TEXT,                            -- 获奖等级：first/second/third/honorable
    award_date TEXT,                             -- 获奖日期
    certificate_url TEXT,                        -- 证书图片URL
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (competition_id) REFERENCES competitions(id)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_competitions_level ON competitions(level);
CREATE INDEX IF NOT EXISTS idx_competitions_category ON competitions(category);
CREATE INDEX IF NOT EXISTS idx_competitions_status ON competitions(status);
CREATE INDEX IF NOT EXISTS idx_registrations_competition_id ON competition_registrations(competition_id);
CREATE INDEX IF NOT EXISTS idx_registrations_user_id ON competition_registrations(user_id);

-- ══════════════════════════════════════════════════════════════
-- 二、大学规划
-- ══════════════════════════════════════════════════════════════

-- 3. 规划模板表
CREATE TABLE IF NOT EXISTS plan_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,                          -- 模板名称
    category TEXT NOT NULL,                      -- 类别：academic(学业)/career(就业)/research(科研)/comprehensive(综合)
    description TEXT,                            -- 模板描述
    target_audience TEXT,                        -- 适用对象
    duration TEXT,                               -- 周期（如：4年/1学期）
    goals TEXT NOT NULL,                         -- 目标内容（JSON数组，每个目标含title/description/semester）
    milestones TEXT,                             -- 里程碑（JSON数组）
    success_cases TEXT,                          -- 成功案例（JSON数组）
    ai_prompt TEXT,                              -- AI辅助提示词
    is_active INTEGER DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 4. 学生规划表
CREATE TABLE IF NOT EXISTS student_plans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    template_id INTEGER,                         -- 基于哪个模板
    title TEXT NOT NULL,                         -- 规划标题
    category TEXT NOT NULL,                      -- 类别
    academic_year INTEGER,                       -- 学年（如2026）
    semester INTEGER,                            -- 学期（1或2）
    goals TEXT DEFAULT '[]',                      -- 目标列表（JSON数组）
    progress REAL DEFAULT 0,                     -- 进度百分比（0-100）
    status TEXT DEFAULT 'draft',                 -- 状态：draft/submitted/approved/in_progress/completed
    reviewer_id INTEGER,                         -- 审核人ID
    reviewer_comment TEXT,                       -- 审核意见
    reviewed_at TEXT,                            -- 审核时间
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (template_id) REFERENCES plan_templates(id)
);

-- 5. 规划进度记录表
CREATE TABLE IF NOT EXISTS plan_progress (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_id INTEGER NOT NULL,
    goal_index INTEGER NOT NULL,                 -- 目标索引
    goal_title TEXT,                             -- 目标标题
    status TEXT DEFAULT 'pending',               -- 状态：pending/in_progress/completed
    evidence TEXT,                               -- 完成证据（描述或链接）
    score INTEGER,                               -- 评分
    feedback TEXT,                               -- 反馈
    recorded_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (plan_id) REFERENCES student_plans(id)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_plan_templates_category ON plan_templates(category);
CREATE INDEX IF NOT EXISTS idx_student_plans_user_id ON student_plans(user_id);
CREATE INDEX IF NOT EXISTS idx_student_plans_status ON student_plans(status);
CREATE INDEX IF NOT EXISTS idx_plan_progress_plan_id ON plan_progress(plan_id);

-- ══════════════════════════════════════════════════════════════
-- 三、入党教育
-- ══════════════════════════════════════════════════════════════

-- 6. 入党流程阶段表
CREATE TABLE IF NOT EXISTS party_stages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL UNIQUE,                   -- 阶段代码
    name TEXT NOT NULL,                          -- 阶段名称
    description TEXT,                            -- 阶段说明
    required_docs TEXT,                          -- 所需材料（JSON数组）
    sort_order INTEGER DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 7. 学生入党进度表
CREATE TABLE IF NOT EXISTS party_progress (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    student_id TEXT,
    student_name TEXT,
    college TEXT,
    current_stage TEXT NOT NULL,                 -- 当前阶段代码
    apply_date TEXT,                             -- 申请入党日期
    activator_date TEXT,                         -- 确定为入党积极分子日期
    development_date TEXT,                       -- 确定为发展对象日期
    probation_start TEXT,                        -- 预备期开始日期
    conversion_date TEXT,                        -- 转正日期
    status TEXT DEFAULT 'applicant',             -- 状态：applicant/activist/development/probation/member
    party_member_id INTEGER,                     -- 入党介绍人
    branch_secretary TEXT,                        -- 支部书记
    notes TEXT,                                  -- 备注
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 8. 入党学习记录表
CREATE TABLE IF NOT EXISTS party_study_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    study_type TEXT NOT NULL,                    -- 学习类型：theory(理论学习)/practice(实践活动)/meeting(组织生活)/volunteer(志愿服务)
    title TEXT NOT NULL,                         -- 学习主题
    content TEXT,                                -- 学习内容
    duration INTEGER,                            -- 学习时长（分钟）
    study_date TEXT NOT NULL,                    -- 学习日期
    certificate TEXT,                            -- 证明材料URL
    status TEXT DEFAULT 'completed',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_party_progress_user_id ON party_progress(user_id);
CREATE INDEX IF NOT EXISTS idx_party_progress_status ON party_progress(status);
CREATE INDEX IF NOT EXISTS idx_party_study_user_id ON party_study_records(user_id);

-- ══════════════════════════════════════════════════════════════
-- 四、社团生活
-- ══════════════════════════════════════════════════════════════

-- 9. 社团表
CREATE TABLE IF NOT EXISTS clubs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,                          -- 社团名称
    category TEXT NOT NULL,                      -- 类别：technology(科技)/art(文艺)/sports(体育)/public welfare(公益)/academic(学术)
    description TEXT,                            -- 社团简介
    founder TEXT,                                -- 创始人
    president TEXT,                              -- 现任社长
    contact_info TEXT,                           -- 联系方式
    member_count INTEGER DEFAULT 0,              -- 成员数
    max_members INTEGER DEFAULT 50,              -- 最大人数
    status TEXT DEFAULT 'active',                -- 状态：active/inactive/dissolved
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 10. 社团成员表
CREATE TABLE IF NOT EXISTS club_members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    club_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    student_id TEXT,
    student_name TEXT,
    role TEXT DEFAULT 'member',                  -- 角色：member(成员)/officer(干部)/president(社长)/advisor(指导老师)
    join_date TEXT NOT NULL,                     -- 加入日期
    leave_date TEXT,                             -- 离开日期
    contribution TEXT,                           -- 贡献描述
    status TEXT DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (club_id) REFERENCES clubs(id)
);

-- 11. 社团活动表
CREATE TABLE IF NOT EXISTS club_activities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    club_id INTEGER NOT NULL,
    title TEXT NOT NULL,                         -- 活动名称
    description TEXT,                            -- 活动描述
    activity_type TEXT NOT NULL,                 -- 活动类型：regular(常规)/competition(比赛)/exhibition(展示)/social(社交)/training(培训)
    start_time TEXT NOT NULL,                    -- 开始时间
    end_time TEXT,                               -- 结束时间
    location TEXT,                               -- 活动地点
    max_participants INTEGER,                    -- 最大参与人数
    current_participants INTEGER DEFAULT 0,      -- 当前参与人数
    status TEXT DEFAULT 'upcoming',              -- 状态：upcoming/ongoing/finished/cancelled
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (club_id) REFERENCES clubs(id)
);

-- 12. 社团活动报名表
CREATE TABLE IF NOT EXISTS club_activity_registrations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    activity_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    student_name TEXT,
    status TEXT DEFAULT 'registered',            -- 状态：registered/attended/cancelled
    feedback TEXT,                               -- 活动反馈
    rating INTEGER,                              -- 评分（1-5）
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (activity_id) REFERENCES club_activities(id)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_clubs_category ON clubs(category);
CREATE INDEX IF NOT EXISTS idx_club_members_club_id ON club_members(club_id);
CREATE INDEX IF NOT EXISTS idx_club_members_user_id ON club_members(user_id);
CREATE INDEX IF NOT EXISTS idx_club_activities_club_id ON club_activities(club_id);
CREATE INDEX IF NOT EXISTS idx_club_registrations_activity_id ON club_activity_registrations(activity_id);

-- ══════════════════════════════════════════════════════════════
-- 初始化数据
-- ══════════════════════════════════════════════════════════════

-- 入党流程阶段
INSERT OR IGNORE INTO party_stages (code, name, description, required_docs, sort_order) VALUES
('applicant', '提交入党申请书', '向党组织递交入党申请书', '["入党申请书"]', 1),
('activist', '确定为入党积极分子', '经过培养考察后确定为入党积极分子', '["入党申请书","思想汇报","培养联系人意见"]', 2),
('development', '确定为发展对象', '经过进一步培养考察后确定为发展对象', '["入党志愿书","政审材料","培训证书","群众意见"]', 3),
('probation', '预备党员', '支部大会通过接收为预备党员', '["入党志愿书","预备党员审批表"]', 4),
('member', '正式党员', '预备期满转为正式党员', '["转正申请书","预备期总结"]', 5);

-- 规划模板
INSERT OR IGNORE INTO plan_templates (name, category, description, target_audience, duration, goals, milestones, success_cases, ai_prompt) VALUES
('四年学业规划', 'academic', '覆盖大一到大四的学业发展规划', '全体学生', '4年',
 '[{"title":"大一适应与基础","description":"适应大学生活，打好专业基础","semester":"1-2"},{"title":"大二专业深化","description":"深入学习专业知识，参与竞赛","semester":"3-4"},{"title":"大三方向确定","description":"确定考研/就业方向，积累经验","semester":"5-6"},{"title":"大四冲刺","description":"完成毕设，求职/考研","semester":"7-8"}]',
 '[{"name":"通过英语四级","deadline":"大二上"},{"name":"获得奖学金","deadline":"大二下"},{"name":"完成实习","deadline":"大三暑假"},{"name":"通过毕业答辩","deadline":"大四下"}]',
 '[{"name":"张同学","achievement":"保研985","key":"坚持竞赛+科研"}]',
 '请根据学生的专业和兴趣，为他制定个性化的四年学业规划。'),

('考研规划', 'academic', '针对计划考研学生的专项规划', '意向考研学生', '2年',
 '[{"title":"基础阶段","description":"数学、英语基础复习","semester":"1-2"},{"title":"强化阶段","description":"专业课强化，真题练习","semester":"3-4"},{"title":"冲刺阶段","description":"模拟考试，查漏补缺","semester":"5-6"}]',
 '[{"name":"完成数学一轮","deadline":"暑假前"},{"name":"英语真题做完","deadline":"10月"},{"name":"模拟考350+","deadline":"11月"}]',
 '[{"name":"李同学","achievement":"上岸211","key":"早准备、稳心态"}]',
 '请为计划考研的学生制定详细的备考规划。'),

('就业规划', 'career', '针对计划就业学生的专项规划', '意向就业学生', '2年',
 '[{"title":"技能储备","description":"学习就业所需技能","semester":"1-2"},{"title":"实习积累","description":"寻找实习机会","semester":"3-4"},{"title":"求职冲刺","description":"简历投递、面试准备","semester":"5-6"}]',
 '[{"name":"掌握一项核心技能","deadline":"大三上"},{"name":"获得实习offer","deadline":"大三暑假"},{"name":"拿到3个面试机会","deadline":"大四上"}]',
 '[{"name":"王同学","achievement":"入职大厂","key":"项目经验+刷题"}]',
 '请为计划就业的学生制定求职准备规划。');

-- 竞赛示例数据
INSERT OR IGNORE INTO competitions (name, level, category, organizer, description, requirements, features, registration_start, registration_end, competition_date, website, max_team_size, is_team_competition, status) VALUES
('ACM-ICPC国际大学生程序设计竞赛', 'national', 'programming', 'ACM', '全球最具影响力的程序设计竞赛', '在校本科生，熟悉算法与数据结构', '["团队协作","算法能力","限时编程"]', '2026-03-01', '2026-04-30', '2026-05-15', 'https://icpc.com', 3, 1, 'upcoming'),
('蓝桥杯全国软件和信息技术专业人才大赛', 'national', 'programming', '工信部', '面向高校学生的IT学科竞赛', '在校学生，热爱编程', '["个人赛","编程能力","分组竞赛"]', '2026-01-15', '2026-03-15', '2026-04-20', 'https://dasai.lanqiao.cn', 1, 0, 'registration'),
('全国大学生数学建模竞赛', 'national', 'math', '教育部', '培养学生用数学方法解决实际问题的能力', '三人一组，熟悉数学建模', '["团队协作","建模能力","论文写作"]', '2026-04-01', '2026-06-30', '2026-09-01', 'https://mcm.edu.cn', 3, 1, 'upcoming'),
('挑战杯全国大学生课外学术科技作品竞赛', 'national', 'innovation', '团中央', '大学生科技创新的奥林匹克', '有创新性项目或作品', '["创新性","实践能力","答辩展示"]', '2026-02-01', '2026-05-31', '2026-10-01', 'https://tiaozhanbei.net', 5, 1, 'upcoming'),
('全国大学生电子设计竞赛', 'national', 'electronics', '教育部', '电子信息类最重要的学科竞赛', '电子、通信、自动化等相关专业', '["硬件设计","编程能力","团队协作"]', '2026-03-01', '2026-05-15', '2026-07-20', 'https://nuedc.com.cn', 3, 1, 'upcoming');

-- 社团示例数据
INSERT OR IGNORE INTO clubs (name, category, description, president, member_count, max_members, status) VALUES
('计算机协会', 'technology', '热爱计算机技术的学生组织，定期举办技术分享和编程马拉松', '张三', 45, 60, 'active'),
('数学建模社', 'academic', '培养数学建模能力，参加各类建模竞赛', '李四', 30, 40, 'active'),
('摄影协会', 'art', '用镜头记录校园美好瞬间', '王五', 35, 50, 'active'),
('青年志愿者协会', 'public welfare', '参与志愿服务，传递爱心与温暖', '赵六', 60, 80, 'active'),
('篮球社', 'sports', '热爱篮球的学生之家', '钱七', 40, 50, 'active');
