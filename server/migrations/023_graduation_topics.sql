-- ============================================================
-- 023_graduation_topics.sql
-- 毕设选题智能体：选题库、导师库、学生选题、进度里程碑
-- ============================================================

-- 1. 导师表
CREATE TABLE IF NOT EXISTS advisors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    advisor_id TEXT NOT NULL UNIQUE,           -- 工号
    title TEXT,                                 -- 职称（教授/副教授/讲师）
    college TEXT,                               -- 所在学院
    department TEXT,                            -- 所在教研室/系
    research_areas TEXT,                        -- 研究方向（JSON数组）
    max_students INTEGER DEFAULT 5,            -- 最大指导学生数
    is_active INTEGER DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 2. 选题表
CREATE TABLE IF NOT EXISTS thesis_topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,                        -- 题目名称
    advisor_id INTEGER NOT NULL,                -- 指导教师
    college TEXT NOT NULL,                      -- 所属学院
    major TEXT,                                 -- 所属专业
    topic_type TEXT DEFAULT 'design',           -- 题目类型：design(设计)/research(论文)/development(开发)
    nature TEXT DEFAULT 'engineering',          -- 课题性质：engineering(工程)/theory(理论)/experiment(实验)
    result_form TEXT DEFAULT 'system',          -- 成果形式：system(系统)/paper(论文)/prototype(原型)
    difficulty TEXT DEFAULT 'medium',           -- 难度：easy/medium/hard
    description TEXT,                           -- 题目简介
    requirements TEXT,                          -- 选题要求
    keywords TEXT,                              -- 关键词（逗号分隔）
    max_students INTEGER DEFAULT 1,            -- 最大选题人数
    selected_count INTEGER DEFAULT 0,          -- 已选人数
    batch INTEGER NOT NULL,                     -- 届别（如2026）
    status TEXT DEFAULT 'active',               -- 状态：active/closed/full
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (advisor_id) REFERENCES advisors(id)
);

-- 3. 学生选题记录表
CREATE TABLE IF NOT EXISTS student_topic_selections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,                   -- 关联蔚小芯用户表
    student_id TEXT NOT NULL,                   -- 学号
    student_name TEXT NOT NULL,                 -- 姓名
    college TEXT,                               -- 学院
    major TEXT,                                 -- 专业
    class_name TEXT,                            -- 班级
    batch INTEGER NOT NULL,                     -- 届别
    topic_id INTEGER,                           -- 选择的题目ID
    advisor_id INTEGER,                         -- 选择的导师ID
    status TEXT DEFAULT 'pending',              -- 状态：pending(待确认)/confirmed(已确认)/changed(已改题)
    preference_order INTEGER DEFAULT 1,         -- 志愿顺序（1=第一志愿）
    reason TEXT,                                -- 选题理由
    confirmed_at TEXT,                          -- 确认时间
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (topic_id) REFERENCES thesis_topics(id),
    FOREIGN KEY (advisor_id) REFERENCES advisors(id)
);

-- 4. 里程碑表
CREATE TABLE IF NOT EXISTS graduation_milestones (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    batch INTEGER NOT NULL,                     -- 届别
    code TEXT NOT NULL,                         -- 阶段代码
    name TEXT NOT NULL,                         -- 阶段名称
    deadline TEXT NOT NULL,                     -- 截止日期
    weight INTEGER DEFAULT 10,                  -- 权重
    description TEXT,                           -- 说明
    sort_order INTEGER DEFAULT 0,               -- 排序
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(batch, code)
);

-- 5. 毕设进度记录表
CREATE TABLE IF NOT EXISTS graduation_progress (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,                   -- 学生用户ID
    topic_id INTEGER NOT NULL,                  -- 题目ID
    milestone_code TEXT NOT NULL,               -- 里程碑阶段代码
    status TEXT DEFAULT 'pending',              -- 状态：pending/in_progress/submitted/completed
    submitted_at TEXT,                          -- 提交时间
    completed_at TEXT,                          -- 完成时间
    feedback TEXT,                              -- 导师反馈
    score INTEGER,                              -- 评分
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (topic_id) REFERENCES thesis_topics(id)
);

-- 6. 智能体配置表（毕设选题智能体）
CREATE TABLE IF NOT EXISTS graduation_agent_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_type TEXT NOT NULL UNIQUE,            -- 智能体类型
    name TEXT NOT NULL,                         -- 名称
    description TEXT,                           -- 描述
    system_prompt TEXT,                         -- 系统提示词
    is_active INTEGER DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_advisors_college ON advisors(college);
CREATE INDEX IF NOT EXISTS idx_advisors_advisor_id ON advisors(advisor_id);
CREATE INDEX IF NOT EXISTS idx_topics_advisor_id ON thesis_topics(advisor_id);
CREATE INDEX IF NOT EXISTS idx_topics_college ON thesis_topics(college);
CREATE INDEX IF NOT EXISTS idx_topics_batch ON thesis_topics(batch);
CREATE INDEX IF NOT EXISTS idx_topics_status ON thesis_topics(status);
CREATE INDEX IF NOT EXISTS idx_selections_user_id ON student_topic_selections(user_id);
CREATE INDEX IF NOT EXISTS idx_selections_topic_id ON student_topic_selections(topic_id);
CREATE INDEX IF NOT EXISTS idx_selections_batch ON student_topic_selections(batch);
CREATE INDEX IF NOT EXISTS idx_milestones_batch ON graduation_milestones(batch);
CREATE INDEX IF NOT EXISTS idx_progress_user_id ON graduation_progress(user_id);
CREATE INDEX IF NOT EXISTS idx_progress_topic_id ON graduation_progress(topic_id);

-- 插入默认智能体配置
INSERT OR IGNORE INTO graduation_agent_config (agent_type, name, description, system_prompt) VALUES
('topic_advisor', '毕设选题顾问', '根据学生专业方向、兴趣爱好和能力水平，推荐合适的毕设题目',
 '你是一个毕业设计选题顾问。根据学生的专业、兴趣和能力，从选题库中推荐合适的题目，并说明推荐理由。考虑因素：1.学生专业匹配度 2.题目难度适中 3.导师研究方向 4.就业发展方向。');

-- 插入默认里程碑（2026届）
INSERT OR IGNORE INTO graduation_milestones (batch, code, name, deadline, weight, description, sort_order) VALUES
(2026, 'topic', '选题确认', '2025-12-31', 5, '完成选题并确认指导教师', 1),
(2026, 'proposal', '开题报告', '2026-01-31', 10, '完成开题报告撰写与答辩', 2),
(2026, 'midterm', '中期检查', '2026-04-15', 20, '完成中期检查报告', 3),
(2026, 'thesis', '论文撰写', '2026-05-15', 30, '完成毕业论文初稿', 4),
(2026, 'plagiarism', '论文查重', '2026-05-25', 10, '通过论文查重检测', 5),
(2026, 'defense', '毕业答辩', '2026-06-10', 20, '完成毕业答辩', 6),
(2026, 'archive', '资料归档', '2026-06-30', 5, '完成所有资料归档', 7);

-- 插入示例导师数据
INSERT OR IGNORE INTO advisors (name, advisor_id, title, college, department, research_areas, max_students) VALUES
('张教授', 'T001', '教授', '信息学院', '计算机科学系', '["人工智能","机器学习","数据挖掘"]', 6),
('李副教授', 'T002', '副教授', '信息学院', '计算机科学系', '["软件工程","Web开发","数据库"]', 5),
('王讲师', 'T003', '讲师', '信息学院', '计算机科学系', '["网络安全","信息安全","密码学"]', 4),
('赵教授', 'T004', '教授', '信息学院', '电子信息系', '["嵌入式系统","物联网","单片机"]', 5),
('陈副教授', 'T005', '副教授', '信息学院', '电子信息系', '["通信工程","信号处理","5G技术"]', 4);

-- 插入示例选题数据
INSERT OR IGNORE INTO thesis_topics (title, advisor_id, college, major, topic_type, nature, result_form, difficulty, description, keywords, max_students, batch) VALUES
(1, '基于深度学习的图像识别系统设计与实现', 1, '信息学院', '软件工程', 'design', 'engineering', 'system', 'hard', '利用深度学习技术实现图像分类和目标检测系统', '深度学习,图像识别,CNN,PyTorch', 2, 2026),
(2, '智能家居控制系统设计', 2, '信息学院', '软件工程', 'design', 'engineering', 'system', 'medium', '设计并实现基于物联网的智能家居控制系统', '物联网,嵌入式,传感器,Android', 2, 2026),
(3, '校园二手交易平台开发', 3, '信息学院', '软件工程', 'design', 'engineering', 'system', 'medium', '开发校园二手物品交易平台，支持在线交易', 'Web开发,React,Node.js,MySQL', 3, 2026),
(4, '学生行为分析预警系统', 4, '信息学院', '软件工程', 'design', 'engineering', 'system', 'hard', '基于学生行为数据进行分析和预警', '数据分析,机器学习,Python,可视化', 2, 2026),
(5, '企业级OA系统设计与实现', 5, '信息学院', '软件工程', 'design', 'engineering', 'system', 'medium', '设计并实现企业级办公自动化系统', 'Java,Spring Boot,Vue.js,微服务', 3, 2026);
