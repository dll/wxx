-- 036_student_education_modules.sql — 学生教育教学核心模块
-- 就业指导 + 学业学习 + 心理健康 三大模块 P0 核心表

-- ============================================
-- 一、就业指导模块
-- ============================================

-- 就业政策表
CREATE TABLE IF NOT EXISTS career_policies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    policy_id TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT 'employment_policy',
    level TEXT NOT NULL DEFAULT 'school',
    source TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    summary TEXT DEFAULT '',
    publish_date TEXT,
    effective_date TEXT,
    expiry_date TEXT,
    tags TEXT DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'active',
    view_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 招聘信息表
CREATE TABLE IF NOT EXISTS job_postings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id TEXT NOT NULL UNIQUE,
    company_name TEXT NOT NULL,
    company_logo TEXT DEFAULT '',
    company_intro TEXT DEFAULT '',
    position_name TEXT NOT NULL,
    position_type TEXT NOT NULL DEFAULT 'campus',
    industry TEXT NOT NULL DEFAULT '',
    salary_min INTEGER DEFAULT 0,
    salary_max INTEGER DEFAULT 0,
    salary_unit TEXT DEFAULT 'K/月',
    location TEXT NOT NULL DEFAULT '',
    education TEXT NOT NULL DEFAULT '',
    major_requirement TEXT DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    requirement TEXT NOT NULL DEFAULT '',
    benefits TEXT DEFAULT '',
    application_url TEXT DEFAULT '',
    deadline TEXT,
    source TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    view_count INTEGER NOT NULL DEFAULT 0,
    apply_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 宣讲会表
CREATE TABLE IF NOT EXISTS info_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL UNIQUE,
    company_name TEXT NOT NULL,
    company_logo TEXT DEFAULT '',
    title TEXT NOT NULL,
    date TEXT NOT NULL,
    time_start TEXT NOT NULL,
    time_end TEXT NOT NULL,
    location TEXT NOT NULL,
    campus TEXT DEFAULT '',
    description TEXT DEFAULT '',
    registration_url TEXT DEFAULT '',
    capacity INTEGER DEFAULT 0,
    registered_count INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 简历模板表
CREATE TABLE IF NOT EXISTS resume_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    style TEXT NOT NULL DEFAULT 'professional',
    target_position TEXT DEFAULT '',
    preview_image TEXT DEFAULT '',
    content_json TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 用户简历表
CREATE TABLE IF NOT EXISTS user_resumes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    resume_id TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    template_id TEXT DEFAULT '',
    content_json TEXT NOT NULL DEFAULT '{}',
    is_default INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 面试题库表
CREATE TABLE IF NOT EXISTS interview_questions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    question_id TEXT NOT NULL UNIQUE,
    category TEXT NOT NULL DEFAULT 'hr',
    industry TEXT DEFAULT '',
    position TEXT DEFAULT '',
    question TEXT NOT NULL,
    answer_hint TEXT DEFAULT '',
    keywords TEXT DEFAULT '',
    difficulty INTEGER NOT NULL DEFAULT 2,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- ============================================
-- 二、学业学习模块
-- ============================================

-- 课程表
CREATE TABLE IF NOT EXISTS courses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    course_id TEXT NOT NULL UNIQUE,
    course_code TEXT NOT NULL DEFAULT '',
    course_name TEXT NOT NULL,
    credit REAL NOT NULL DEFAULT 0,
    hours INTEGER NOT NULL DEFAULT 0,
    category TEXT NOT NULL DEFAULT 'required',
    department TEXT NOT NULL DEFAULT '',
    teacher TEXT DEFAULT '',
    description TEXT DEFAULT '',
    syllabus TEXT DEFAULT '',
    prerequisites TEXT DEFAULT '[]',
    textbook TEXT DEFAULT '',
    references TEXT DEFAULT '[]',
    semester TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 学生成绩表
CREATE TABLE IF NOT EXISTS student_grades (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    course_id TEXT NOT NULL,
    course_name TEXT NOT NULL DEFAULT '',
    semester TEXT NOT NULL,
    grade_type TEXT NOT NULL DEFAULT 'final',
    score REAL DEFAULT 0,
    gpa REAL DEFAULT 0,
    rank INTEGER DEFAULT 0,
    grade_level TEXT DEFAULT '',
    passed INTEGER NOT NULL DEFAULT 0,
    credits_earned REAL NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, course_id, semester, grade_type)
);

-- 学习资源表
CREATE TABLE IF NOT EXISTS learning_resources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    resource_id TEXT NOT NULL UNIQUE,
    course_id TEXT NOT NULL DEFAULT '',
    course_name TEXT DEFAULT '',
    title TEXT NOT NULL,
    resource_type TEXT NOT NULL DEFAULT 'courseware',
    chapter TEXT DEFAULT '',
    file_url TEXT DEFAULT '',
    content TEXT DEFAULT '',
    author TEXT DEFAULT '',
    download_count INTEGER NOT NULL DEFAULT 0,
    view_count INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    tags TEXT DEFAULT '[]',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 考试安排表
CREATE TABLE IF NOT EXISTS exam_schedules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exam_id TEXT NOT NULL UNIQUE,
    course_id TEXT NOT NULL DEFAULT '',
    course_name TEXT NOT NULL,
    exam_type TEXT NOT NULL DEFAULT 'final',
    date TEXT NOT NULL,
    time_start TEXT NOT NULL,
    time_end TEXT NOT NULL,
    location TEXT NOT NULL,
    seat TEXT DEFAULT '',
    semester TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(course_id, exam_type, semester)
);

-- 错题本表
CREATE TABLE IF NOT EXISTS mistake_notebook (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    mistake_id TEXT NOT NULL UNIQUE,
    course_id TEXT DEFAULT '',
    course_name TEXT DEFAULT '',
    subject TEXT DEFAULT '',
    question TEXT NOT NULL,
    user_answer TEXT DEFAULT '',
    correct_answer TEXT DEFAULT '',
    analysis TEXT DEFAULT '',
    difficulty INTEGER DEFAULT 2,
    master_level INTEGER DEFAULT 0,
    review_count INTEGER DEFAULT 0,
    last_review_date TEXT,
    source TEXT DEFAULT '',
    tags TEXT DEFAULT '[]',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- ============================================
-- 三、心理健康模块
-- ============================================

-- 心理测评量表表
CREATE TABLE IF NOT EXISTS psych_scales (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scale_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    abbreviation TEXT DEFAULT '',
    category TEXT NOT NULL DEFAULT 'emotion',
    description TEXT DEFAULT '',
    question_count INTEGER NOT NULL DEFAULT 0,
    estimated_minutes INTEGER NOT NULL DEFAULT 5,
    scoring_method TEXT DEFAULT '',
    interpretation TEXT NOT NULL DEFAULT '{}',
    questions_json TEXT NOT NULL DEFAULT '[]',
    is_crisis INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 测评记录表
CREATE TABLE IF NOT EXISTS psych_assessment_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    record_id TEXT NOT NULL UNIQUE,
    scale_id TEXT NOT NULL,
    scale_name TEXT DEFAULT '',
    answers_json TEXT NOT NULL DEFAULT '[]',
    scores_json TEXT NOT NULL DEFAULT '{}',
    total_score REAL DEFAULT 0,
    level TEXT NOT NULL DEFAULT 'normal',
    result_summary TEXT DEFAULT '',
    suggestion TEXT DEFAULT '',
    completed_at TEXT NOT NULL DEFAULT (datetime('now')),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 心理咨询师表
CREATE TABLE IF NOT EXISTS counselors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    counselor_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    avatar TEXT DEFAULT '',
    gender TEXT DEFAULT '',
    department TEXT DEFAULT '',
    specialties TEXT DEFAULT '[]',
    bio TEXT DEFAULT '',
    working_days TEXT DEFAULT '[]',
    available INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 咨询预约表
CREATE TABLE IF NOT EXISTS counseling_appointments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    appointment_id TEXT NOT NULL UNIQUE,
    counselor_id TEXT NOT NULL,
    counselor_name TEXT DEFAULT '',
    appointment_date TEXT NOT NULL,
    time_slot TEXT NOT NULL,
    appointment_type TEXT NOT NULL DEFAULT 'face_to_face',
    reason TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    cancel_reason TEXT DEFAULT '',
    notes TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 情绪日记表
CREATE TABLE IF NOT EXISTS mood_diary (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    diary_id TEXT NOT NULL UNIQUE,
    date TEXT NOT NULL,
    mood_score INTEGER NOT NULL DEFAULT 5,
    mood_tags TEXT DEFAULT '[]',
    events TEXT DEFAULT '',
    diary_content TEXT DEFAULT '',
    sleep_hours REAL DEFAULT 0,
    exercise_minutes INTEGER DEFAULT 0,
    social_level INTEGER DEFAULT 3,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, date)
);

-- 心理科普文章表
CREATE TABLE IF NOT EXISTS psych_articles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    article_id TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT 'emotion',
    summary TEXT DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    cover_image TEXT DEFAULT '',
    author TEXT DEFAULT '',
    read_count INTEGER NOT NULL DEFAULT 0,
    is_crisis INTEGER NOT NULL DEFAULT 0,
    tags TEXT DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 危机热线表
CREATE TABLE IF NOT EXISTS crisis_hotlines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    hotline_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    phone TEXT NOT NULL,
    service_time TEXT DEFAULT '',
    description TEXT DEFAULT '',
    level INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- ============================================
-- 四、索引优化
-- ============================================

CREATE INDEX IF NOT EXISTS idx_career_policies_category ON career_policies(category, status);
CREATE INDEX IF NOT EXISTS idx_job_postings_type ON job_postings(position_type, status);
CREATE INDEX IF NOT EXISTS idx_job_postings_industry ON job_postings(industry, status);
CREATE INDEX IF NOT EXISTS idx_info_sessions_date ON info_sessions(date, status);
CREATE INDEX IF NOT EXISTS idx_user_resumes_user ON user_resumes(user_id);
CREATE INDEX IF NOT EXISTS idx_courses_department ON courses(department, status);
CREATE INDEX IF NOT EXISTS idx_student_grades_user ON student_grades(user_id, semester);
CREATE INDEX IF NOT EXISTS idx_learning_resources_course ON learning_resources(course_id);
CREATE INDEX IF NOT EXISTS idx_exam_schedules_semester ON exam_schedules(semester);
CREATE INDEX IF NOT EXISTS idx_mistake_notebook_user ON mistake_notebook(user_id);
CREATE INDEX IF NOT EXISTS idx_psych_scales_category ON psych_scales(category, status);
CREATE INDEX IF NOT EXISTS idx_psych_assessment_user ON psych_assessment_records(user_id);
CREATE INDEX IF NOT EXISTS idx_counselors_status ON counselors(status);
CREATE INDEX IF NOT EXISTS idx_counseling_user ON counseling_appointments(user_id);
CREATE INDEX IF NOT EXISTS idx_mood_diary_user ON mood_diary(user_id, date);
CREATE INDEX IF NOT EXISTS idx_psych_articles_category ON psych_articles(category, status);
CREATE INDEX IF NOT EXISTS idx_crisis_hotlines_status ON crisis_hotlines(status);

-- ============================================
-- 五、种子数据 - 心理测评量表（简化版 SDS 抑郁自评量表）
-- ============================================

INSERT OR IGNORE INTO psych_scales (scale_id, name, abbreviation, category, description, question_count, estimated_minutes, scoring_method, interpretation, questions_json, is_crisis, status)
VALUES (
    'scale-sds',
    '抑郁自评量表（SDS）',
    'SDS',
    'emotion',
    '抑郁自评量表（Self-Rating Depression Scale，SDS）由Zung编制于1965年，用于衡量抑郁状态的轻重程度及其在治疗中的变化。',
    20,
    10,
    '将20个项目的各项得分相加得到粗分，粗分乘以1.25以后取整数部分，得到标准分。标准分越低越好。',
    '{"levels": [{"name":"正常","min":0,"max":52,"color":"#10b981","suggestion":"您的心理状态良好，继续保持健康的生活方式。"},{"name":"轻度抑郁","min":53,"max":62,"color":"#f59e0b","suggestion":"您可能有轻度抑郁情绪，建议关注自身情绪变化，多与亲友交流，适当运动。如有需要可以咨询心理咨询师。"},{"name":"中度抑郁","min":63,"max":72,"color":"#ef4444","suggestion":"您的抑郁症状较为明显，建议尽快寻求专业心理咨询或就医诊断。"},{"name":"重度抑郁","min":73,"max":100,"color":"#dc2626","suggestion":"您的抑郁症状严重，请立即寻求专业医疗帮助。如有自杀念头请拨打危机干预热线。"}]}',
    '[{"id":1,"text":"我感到情绪沮丧，郁闷","reverse":false},{"id":2,"text":"我感到早晨心情最好","reverse":true},{"id":3,"text":"我要哭或想哭","reverse":false},{"id":4,"text":"我夜间睡眠不好","reverse":false},{"id":5,"text":"我吃饭像平时一样多","reverse":true},{"id":6,"text":"我的性功能正常","reverse":true},{"id":7,"text":"我感到体重减轻","reverse":false},{"id":8,"text":"我为便秘烦恼","reverse":false},{"id":9,"text":"我的心跳比平时快","reverse":false},{"id":10,"text":"我无故感到疲劳","reverse":false},{"id":11,"text":"我的头脑像往常一样清楚","reverse":true},{"id":12,"text":"我做事情像平时一样不感到困难","reverse":true},{"id":13,"text":"我坐卧不安，难以保持平静","reverse":false},{"id":14,"text":"我对未来感到有希望","reverse":true},{"id":15,"text":"我比平时更容易激怒","reverse":false},{"id":16,"text":"我觉得决定什么事很容易","reverse":true},{"id":17,"text":"我感到自己是有用的和不可缺少的人","reverse":true},{"id":18,"text":"我的生活很有意义","reverse":true},{"id":19,"text":"假若我死了别人会过得更好","reverse":false},{"id":20,"text":"我仍旧喜爱自己平时喜爱的东西","reverse":true}]',
    1,
    'active'
);

INSERT OR IGNORE INTO psych_scales (scale_id, name, abbreviation, category, description, question_count, estimated_minutes, scoring_method, interpretation, questions_json, is_crisis, status)
VALUES (
    'scale-sas',
    '焦虑自评量表（SAS）',
    'SAS',
    'emotion',
    '焦虑自评量表（Self-Rating Anxiety Scale，SAS）由Zung编制于1971年，用于评估焦虑症状的轻重程度。',
    20,
    10,
    '将20个项目的各项得分相加得到粗分，粗分乘以1.25以后取整数部分，得到标准分。',
    '{"levels": [{"name":"正常","min":0,"max":49,"color":"#10b981","suggestion":"您的焦虑水平在正常范围内，继续保持良好的心态。"},{"name":"轻度焦虑","min":50,"max":59,"color":"#f59e0b","suggestion":"您可能有轻度焦虑，建议学习放松技巧，规律作息，适当运动。"},{"name":"中度焦虑","min":60,"max":69,"color":"#ef4444","suggestion":"您的焦虑症状较为明显，建议寻求专业心理咨询帮助。"},{"name":"重度焦虑","min":70,"max":100,"color":"#dc2626","suggestion":"您的焦虑症状严重，请尽快寻求专业医疗帮助。"}]}',
    '[{"id":1,"text":"我感到比往常更加神经过敏和焦虑","reverse":false},{"id":2,"text":"我无缘无故感到担心","reverse":false},{"id":3,"text":"我容易心烦意乱或感到恐慌","reverse":false},{"id":4,"text":"我感到我的身体好像被分成几块","reverse":false},{"id":5,"text":"我感到一切都很好，不会发生什么不幸","reverse":true},{"id":6,"text":"我手脚发抖打颤","reverse":false},{"id":7,"text":"我因头痛、颈痛和背痛而烦恼","reverse":false},{"id":8,"text":"我感到无力且容易疲劳","reverse":false},{"id":9,"text":"我能安静地坐下来并轻松地放松自己","reverse":true},{"id":10,"text":"我感到我的心跳很快","reverse":false},{"id":11,"text":"我因一阵阵的头晕而苦恼","reverse":false},{"id":12,"text":"我有晕倒发作或觉得要晕倒似的","reverse":false},{"id":13,"text":"我呼气吸气都感到很容易","reverse":true},{"id":14,"text":"我手脚麻木和刺痛","reverse":false},{"id":15,"text":"我因胃痛和消化不良而苦恼","reverse":false},{"id":16,"text":"我常常要小便","reverse":false},{"id":17,"text":"我的手常常是干燥温暖的","reverse":true},{"id":18,"text":"我脸红发热","reverse":false},{"id":19,"text":"我容易入睡并且一夜睡得很好","reverse":true},{"id":20,"text":"我做恶梦","reverse":false}]',
    0,
    'active'
);

-- ============================================
-- 六、种子数据 - 危机热线
-- ============================================

INSERT OR IGNORE INTO crisis_hotlines (hotline_id, name, phone, service_time, description, level, status)
VALUES
('hotline-1', '全国心理援助热线', '400-161-9995', '24小时', '全国心理援助热线，提供专业的心理危机干预服务', 1, 'active'),
('hotline-2', '北京心理危机研究与干预中心', '010-82951332', '24小时', '国内最早的心理危机干预机构之一', 1, 'active'),
('hotline-3', '希望24热线', '400-161-9995', '24小时', '生命教育与危机干预热线', 2, 'active'),
('hotline-4', '校心理咨询中心', '010-12345678', '工作日 8:00-17:00', '学校心理咨询中心预约电话', 3, 'active');

-- ============================================
-- 七、种子数据 - 心理咨询师
-- ============================================

INSERT OR IGNORE INTO counselors (counselor_id, name, title, gender, department, specialties, bio, working_days, available, status)
VALUES
('counselor-001', '张老师', '国家二级心理咨询师', '女', '学生处心理健康教育中心', '["情绪调节","人际关系","学业压力"]', '从事心理咨询工作10年，擅长认知行为疗法，温暖接纳的咨询风格。', '["周一","周三","周五"]', 1, 'active'),
('counselor-002', '李老师', '国家三级心理咨询师', '男', '学生处心理健康教育中心', '["职业发展","焦虑抑郁","家庭关系"]', '心理学硕士，专注于青少年心理发展与职业规划咨询。', '["周二","周四"]', 1, 'active'),
('counselor-003', '王老师', '副教授/心理咨询师', '女', '马克思主义学院', '["思政教育","心理成长","人生规划"]', '资深心理辅导教师，20年学生工作经验，擅长思想引导与心理疏导结合。', '["周一","周二","周四","周五"]', 1, 'active');

-- ============================================
-- 八、种子数据 - 心理科普文章
-- ============================================

INSERT OR IGNORE INTO psych_articles (article_id, title, category, summary, content, author, read_count, is_crisis, tags, status)
VALUES
('psych-article-001', '如何应对考试焦虑？', 'emotion', '考试焦虑是学生常见的心理问题，本文介绍几种有效的应对方法。', '考试焦虑是很多学生都会遇到的问题。适度的焦虑可以提高学习效率，但过度焦虑会影响发挥。

## 应对方法

### 1. 深呼吸放松法
- 找一个安静的地方坐下
- 慢慢吸气4秒，屏住呼吸4秒，慢慢呼气6秒
- 重复5-10次

### 2. 积极自我暗示
- 把"我肯定考不好"换成"我已经准备好了"
- 把"我好紧张"换成"我很兴奋，这说明我在乎"

### 3. 合理的复习计划
- 提前规划，避免临时抱佛脚
- 劳逸结合，保证充足睡眠
- 适当运动，释放压力

### 4. 接纳焦虑情绪
- 焦虑是正常的，每个人都会有
- 不要因为焦虑而更加焦虑
- 带着焦虑去做事，它会慢慢减弱

如果焦虑严重影响了你的生活和学习，请及时寻求专业心理咨询帮助。', '心理健康中心', 128, 0, '["考试焦虑","情绪调节","放松技巧"]', 'active'),
('psych-article-002', '大学生常见心理问题及调适方法', 'self_growth', '进入大学后，学生可能面临各种心理适应问题，了解这些问题有助于更好地应对。', '## 大学生常见心理问题

### 1. 环境适应问题
从高中到大学，生活环境、学习方式、人际关系都发生了很大变化。
- **调适方法**：主动熟悉校园环境，建立新的生活规律，参加社团活动。

### 2. 学业压力问题
大学学习更强调自主性，课程难度也有所增加。
- **调适方法**：制定合理的学习计划，找到适合自己的学习方法，遇到困难及时求助。

### 3. 人际关系问题
大学同学来自全国各地，生活习惯和性格差异较大。
- **调适方法**：尊重差异，真诚待人，学会沟通，有矛盾及时化解。

### 4. 情感问题
恋爱、失恋、暗恋都是大学生可能经历的情感体验。
- **调适方法**：树立正确的恋爱观，爱情不是生活的全部，失恋不代表失败。

### 5. 就业焦虑
面对未来的职业发展，很多同学会感到迷茫和焦虑。
- **调适方法**：提前规划，多尝试，在实践中找到方向。

记住：遇到心理困扰是正常的，寻求帮助是勇敢的表现。学校心理咨询中心随时为你服务。', '心理健康中心', 256, 0, '["适应","学业压力","人际关系","情绪管理"]', 'active'),
('psych-article-003', '如何提高睡眠质量', 'emotion', '良好的睡眠对身心健康至关重要，本文分享几个改善睡眠的实用技巧。', '睡眠对我们的身心健康非常重要。长期睡眠不足会影响注意力、记忆力和情绪。

## 改善睡眠的方法

### 1. 建立规律的作息
- 每天同一时间上床，同一时间起床
- 周末也不要相差超过1小时
- 建立睡前仪式（如泡脚、听轻音乐）

### 2. 优化睡眠环境
- 保持卧室安静、黑暗、凉爽
- 使用舒适的床垫和枕头
- 不要在床上玩手机、学习

### 3. 注意饮食
- 睡前2-3小时不要吃太饱
- 下午4点后避免咖啡因
- 睡前不要大量饮水

### 4. 适当运动
- 每天30分钟有氧运动有助于改善睡眠
- 但睡前3小时内不要剧烈运动

### 5. 管理压力
- 睡前不要想烦心事
- 可以尝试冥想、渐进式肌肉放松
- 如果躺在床上30分钟还睡不着，起来做些放松的事

如果长期失眠，请寻求专业帮助。', '心理健康中心', 189, 0, '["睡眠","健康","放松"]', 'active'),
('psych-article-004', '发现同学有自杀念头怎么办？', 'crisis', '了解自杀风险信号，掌握基本的干预方法，可能挽救一条生命。', '## 自杀的预警信号

### 言语信号
- 直接说"我不想活了"、"活着没意义"
- 间接表达"我走了你们就解脱了"、"下辈子再见"
- 谈论死亡和自杀方式

### 行为信号
- 突然的情绪转变（从低落变得异常平静）
- 分发自己珍贵的物品
- 与亲友告别
- 查找自杀方法
- 增加酒精或药物使用

### 情绪信号
- 极度绝望
- 强烈的内疚或羞耻感
- 感到被困住，没有出路

## 如何帮助

### 1. 认真对待
不要以为对方只是说说而已。每一次表达都是求助信号。

### 2. 直接询问
直接问"你是不是想自杀？"不会把想法植入对方大脑，反而会让对方感到被理解。

### 3. 倾听陪伴
- 不要评判
- 不要说"你怎么这么脆弱"
- 让对方知道你在乎他

### 4. 寻求专业帮助
- 不要承诺保密
- 及时联系老师、家长或专业人士
- 必要时拨打危机干预热线

## 危机热线
- 全国心理援助热线：400-161-9995（24小时）
- 北京心理危机研究与干预中心：010-82951332（24小时）

你的关心可能会挽救一条生命。', '心理健康中心', 56, 1, '["危机干预","自杀","求助"]', 'active');

-- ============================================
-- 九、种子数据 - 学习资源示例
-- ============================================

INSERT OR IGNORE INTO learning_resources (resource_id, course_id, course_name, title, resource_type, chapter, content, author, view_count, status, tags)
VALUES
('learn-res-001', 'CS101', '程序设计基础', '第一章 C语言入门', 'courseware', '第一章', '# C语言入门\n\n## C语言简介\nC语言是一门通用的计算机编程语言，广泛应用于底层开发。\n\n## 第一个程序\n```c\n#include <stdio.h>\n\nint main() {\n    printf("Hello, World!\\n");\n    return 0;\n}\n```', '张教授', 520, 'active', '["C语言","入门","程序设计"]'),
('learn-res-002', 'MATH101', '高等数学', '极限与连续知识点总结', 'notes', '第一章', '# 极限与连续\n\n## 数列极限\n设 {Xn} 为实数数列，a 为定数。若对任给的正数 ε，总存在正整数N，使得当 n>N 时，有 |Xn - a| < ε，则称数列 {Xn} 收敛于 a，定数 a 称为数列的极限。\n\n## 两个重要极限\n1. lim(x→0) sinx/x = 1\n2. lim(x→∞) (1+1/x)^x = e', '李教授', 368, 'active', '["高等数学","极限","知识点"]'),
('learn-res-003', 'ENG101', '大学英语', '四级词汇高频词表', 'courseware', '词汇', '# 四级高频词汇\n\n## 常考词汇\n- abandon v. 放弃，抛弃\n- ability n. 能力，才能\n- absolute a. 绝对的，完全的\n- absorb v. 吸收，吸引\n- abstract a. 抽象的 n. 摘要\n...', '外语学院', 420, 'active', '["英语","四级","词汇"]'),
('learn-res-004', 'CS101', '程序设计基础', '期末复习题精选', 'exercise', '期末复习', '## 一、选择题\n\n1. 以下哪个是C语言的正确注释方式？\n   A. // 单行注释\n   B. /* 多行注释 */\n   C. 以上都对\n   D. 以上都不对\n\n答案：C\n\n2. 以下数据类型中，占用内存最小的是？\n   A. int\n   B. char\n   C. float\n   D. double\n\n答案：B', '计算机学院', 680, 'active', '["C语言","习题","期末复习"]');

-- ============================================
-- 十、种子数据 - 就业政策示例
-- ============================================

INSERT OR IGNORE INTO career_policies (policy_id, title, category, level, source, content, summary, publish_date, tags, status)
VALUES
('career-pol-001', '关于做好2026届毕业生就业创业工作的通知', 'employment_policy', 'school', '学生处就业指导中心', '# 关于做好2026届毕业生就业创业工作的通知\n\n各学院：\n\n为深入贯彻落实党中央、国务院关于高校毕业生就业创业工作的决策部署，现就做好我校2026届毕业生就业创业工作通知如下：\n\n## 一、工作目标\n确保2026届毕业生就业局势总体稳定，就业率不低于去年同期水平。\n\n## 二、重点工作\n1. 拓展就业渠道，深化校企合作\n2. 强化就业指导，提升就业能力\n3. 做好精准帮扶，关注重点群体\n4. 鼓励自主创业，支持灵活就业\n5. 规范就业管理，提升服务质量\n\n## 三、时间安排\n- 9月-10月：就业指导月活动\n- 11月：秋季校园招聘会\n- 12月-次年1月：企业宣讲季\n- 3月-4月：春季校园招聘会\n- 5月-6月：就业冲刺阶段\n\n请各学院高度重视，切实做好毕业生就业创业各项工作。', '学校2026届毕业生就业创业工作部署，含工作目标、重点工作和时间安排', '2025-09-01', '["就业政策","毕业生","2026届"]', 'active'),
('career-pol-002', '国家鼓励高校毕业生基层就业政策汇总', 'employment_policy', 'national', '教育部', '# 国家鼓励高校毕业生基层就业政策汇总\n\n## 一、西部计划\n招募一定数量的普通高等学校应届毕业生或在读研究生，到西部基层开展志愿服务工作。\n- 服务期：1-3年\n- 优惠政策：考研加分、公务员定向招录、学费补偿等\n\n## 二、三支一扶\n高校毕业生到农村基层从事支教、支农、支医和扶贫工作。\n- 服务期：2年\n- 优惠政策：工作期满后考核合格可直接落编\n\n## 三、特岗教师\n农村义务教育阶段学校教师特设岗位计划。\n- 服务期：3年\n- 优惠政策：服务期满考核合格可转为正式编制教师\n\n## 四、大学生村官\n选聘高校毕业生到村任职。\n- 服务期：2-3年\n- 优惠政策：公务员定向考录、事业单位专项招聘\n\n## 五、学费补偿和助学贷款代偿\n- 到中西部地区和艰苦边远地区基层单位就业\n- 服务期在3年以上（含3年）\n- 学费补偿或国家助学贷款代偿最高每年8000元', '国家鼓励高校毕业生到基层就业的五大类政策及优惠措施', '2025-08-15', '["基层就业","政策","西部计划","三支一扶"]', 'active');
