# 蔚小芯 · 高校学生教育教学功能模块建设方案

> 版本：v1.1  
> 状态：已实现（三大核心模块）  
> 负责人：蔚小芯团队  
> 目标：全面覆盖高校学生教育教学场景，以「知识可追溯 + 子智能体协同」为核心，服务所有学生

---

## 一、现状回顾

### 1.1 已有模块（完整度 70%+）

| 模块 | 说明 | 完整度 |
|------|------|--------|
| 基础架构 | 认证/JWT、RBAC+Capability、知识库（FTS/BM25）、多智能体协同 | ✅ 85% |
| 办事流程 | 入学、毕业、请假、转专业、助学贷款、奖学金申请，6种流程 | ✅ 80% |
| 毕设选题 | 导师库、选题库、学生选题、里程碑、进度追踪 | ✅ 85% |
| 学科竞赛 | 竞赛列表/详情/报名/作品提交/我的报名/统计 | ✅ 80% |
| 入党教育 | 5个阶段（申请人→正式党员）、进度追踪、学习记录 | ✅ 80% |
| 大学规划 | 规划模板、我的规划、审核、进度记录 | ✅ 75% |
| 社团生活 | 社团列表/详情/加入/活动/报名 | ✅ 75% |
| 情感预警 | 情感分析+风险评估+通知 | ✅ 70% |

### 1.2 已有子智能体（4个）

| 智能体 | 类型 | 职责 |
|--------|------|------|
| qa-default | 通用问答 | 基于FTS/BM25检索+LLM生成，覆盖面最广 |
| policy-expert | 政策解读 | 专注政策类检索，强调原文引用与来源追溯 |
| process-guide | 流程指引 | 专注流程类检索，输出步骤清单式回答 |
| emotion-counselor | 心理疏导 | 处理学生情感/心理类问题，含危机关键词检测 |

### 1.3 骨架/Mock 状态（30%-50%）

- **学生画像/数字孪生**：3个层级页面都有UI，核心数据全是硬编码Mock
- **校园文化**：校歌/广播/讲座/活动/志愿 5个页面，数据待填充
- **问题预案**：数据模型+部分接口有了，UI待完善
- **问答广场**：3个页面（广场/热点/排行榜）+ 站内私聊，全是占位

### 1.4 缺失方向

- ❌ **就业指导/求职**：完全缺失
- ❌ **绩点/学业成绩查询**：课程地图/学情分析有但都是Mock，无真实成绩数据
- ❌ **心理健康专业服务**：只有情感预警和AI对话，无心理咨询预约/心理测评
- ❌ **垂直子智能体**：缺少学科专业类、就业类、心理类等垂直子智能体

---

## 二、建设目标与原则

### 2.1 目标

1. **补全教育教学核心模块**：就业指导、学业学习、心理健康三大刚需
2. **完善子智能体体系**：从4个通用Agent扩展到10+垂直领域Agent
3. **完善现有功能**：数字孪生从Mock变真实、问答广场落地等
4. **知识可追溯**：所有回答必须引用来源，与知识库体系打通

### 2.2 原则

- **知识库+路由式**：每个子Agent专注特定知识库领域，通过意图路由自动选择
- **渐进式交付**：分两期建设，P0 优先交付核心功能
- **复用现有架构**：不重复造轮子，在现有多智能体编排框架上扩展
- **学生端优先**：聚焦服务学生，辅导员/教师端后续扩展

---

## 三、P0 核心模块（第一期）

### 3.1 就业指导模块

#### 3.1.1 功能清单

| 功能 | 说明 | 优先级 |
|------|------|--------|
| 就业政策库 | 国家/省/校三级就业政策、求职补贴、档案户口等 | P0 |
| 招聘信息 | 校园招聘、宣讲会、实习信息、企业库 | P0 |
| 简历助手 | AI简历优化、简历模板、自荐信生成 | P0 |
| 面试指导 | 常见面试题、模拟面试、面试技巧 | P0 |
| 职业规划 | 职业测评、职业探索、行业介绍 | P1 |
| 求职日历 | 宣讲会提醒、截止日期提醒、面试日程 | P1 |
| 签约管理 | 三方协议、签约流程、违约办理 | P1 |

#### 3.1.2 数据模型

```sql
-- 就业政策表
CREATE TABLE career_policies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    policy_id TEXT NOT NULL UNIQUE,        -- 政策ID
    title TEXT NOT NULL,                    -- 标题
    category TEXT NOT NULL,                 -- 分类：就业政策/创业扶持/落户政策/档案管理
    level TEXT NOT NULL,                    -- 级别：国家/省/校
    source TEXT NOT NULL DEFAULT '',        -- 来源链接
    content TEXT NOT NULL,                  -- 内容（Markdown）
    publish_date TEXT NOT NULL,             -- 发布日期
    effective_date TEXT,                    -- 生效日期
    expiry_date TEXT,                       -- 失效日期
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 招聘信息表
CREATE TABLE job_postings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id TEXT NOT NULL UNIQUE,
    company_name TEXT NOT NULL,             -- 公司名称
    company_logo TEXT DEFAULT '',           -- 公司Logo
    position_name TEXT NOT NULL,            -- 职位名称
    position_type TEXT NOT NULL,            -- 类型：校招/实习/社招
    industry TEXT NOT NULL,                 -- 行业
    salary_min INTEGER,                     -- 最低薪资
    salary_max INTEGER,                     -- 最高薪资
    salary_unit TEXT DEFAULT 'K/月',        -- 薪资单位
    location TEXT NOT NULL,                 -- 工作地点
    education TEXT NOT NULL,                -- 学历要求
    major_requirement TEXT DEFAULT '',      -- 专业要求
    description TEXT NOT NULL,              -- 职位描述
    requirement TEXT NOT NULL,              -- 任职要求
    application_url TEXT DEFAULT '',        -- 投递链接
    deadline TEXT,                          -- 截止日期
    source TEXT DEFAULT '',                 -- 来源
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 宣讲会表
CREATE TABLE info_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL UNIQUE,
    company_name TEXT NOT NULL,
    company_logo TEXT DEFAULT '',
    title TEXT NOT NULL,                    -- 宣讲会标题
    date TEXT NOT NULL,                     -- 日期
    time_start TEXT NOT NULL,               -- 开始时间
    time_end TEXT NOT NULL,                 -- 结束时间
    location TEXT NOT NULL,                 -- 地点
    campus TEXT DEFAULT '',                 -- 校区
    description TEXT DEFAULT '',            -- 简介
    registration_url TEXT DEFAULT '',       -- 报名链接
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 简历模板表
CREATE TABLE resume_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,                     -- 模板名称
    style TEXT NOT NULL,                    -- 风格：简约/创意/专业/学术
    target_position TEXT DEFAULT '',        -- 适合岗位
    preview_image TEXT DEFAULT '',          -- 预览图
    content_json TEXT NOT NULL DEFAULT '{}',-- 模板内容结构
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 用户简历表
CREATE TABLE user_resumes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    resume_id TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,                    -- 简历标题
    template_id TEXT DEFAULT '',            -- 使用的模板
    content_json TEXT NOT NULL DEFAULT '{}',-- 简历内容
    is_default INTEGER NOT NULL DEFAULT 0,  -- 是否默认
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, resume_id)
);

-- 面试题库表
CREATE TABLE interview_questions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    question_id TEXT NOT NULL UNIQUE,
    category TEXT NOT NULL,                 -- 分类：HR面/技术面/行为面/专业面
    industry TEXT DEFAULT '',               -- 行业
    position TEXT DEFAULT '',               -- 岗位
    question TEXT NOT NULL,                 -- 问题
    answer_hint TEXT DEFAULT '',            -- 回答提示
    keywords TEXT DEFAULT '',               -- 关键词
    difficulty INTEGER NOT NULL DEFAULT 2,  -- 难度 1-5
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

#### 3.1.3 前端页面

- `/career` → 就业主页（Tab：推荐/招聘/宣讲/政策）
- `/career/job/:id` → 职位详情
- `/career/resume` → 我的简历
- `/career/resume/edit` → 简历编辑
- `/career/interview` → 面试题库/模拟面试

#### 3.1.4 子智能体

| 智能体Key | 名称 | 职责 |
|----------|------|------|
| career-advisor | 就业指导专家 | 就业政策、求职技巧、行业分析、职业规划问答 |
| resume-expert | 简历优化助手 | 简历撰写指导、优化建议、求职信生成 |
| interview-coach | 面试教练 | 模拟面试、面试题解析、应答技巧 |

---

### 3.2 学业学习模块

#### 3.2.1 功能清单

| 功能 | 说明 | 优先级 |
|------|------|--------|
| 课程信息 | 课程列表、课程详情、教学大纲 | P0 |
| 绩点查询 | 成绩查询、GPA计算、学分统计 | P0 |
| 学业分析 | 成绩趋势、薄弱科目、学习建议 | P0 |
| 学习资源 | 课件、习题、参考资料、学习笔记 | P0 |
| 考试助手 | 考试安排、复习计划、错题本 | P1 |
| 课程地图 | 先修关系、知识图谱、学习路径 | P1 |
| 学习伙伴 | 学习小组、组队学习、互助答疑 | P2 |

#### 3.2.2 数据模型

```sql
-- 课程表
CREATE TABLE courses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    course_id TEXT NOT NULL UNIQUE,
    course_code TEXT NOT NULL,              -- 课程代码
    course_name TEXT NOT NULL,              -- 课程名称
    credit REAL NOT NULL,                   -- 学分
    hours INTEGER NOT NULL,                 -- 学时
    category TEXT NOT NULL,                 -- 分类：必修/选修/通识/专业
    department TEXT NOT NULL,               -- 开课学院
    description TEXT DEFAULT '',            -- 课程简介
    syllabus TEXT DEFAULT '',               -- 教学大纲（Markdown）
    prerequisites TEXT DEFAULT '',          -- 先修课程（JSON数组）
    textbook TEXT DEFAULT '',               -- 教材
    references TEXT DEFAULT '',             -- 参考资料（JSON数组）
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 学生成绩表
CREATE TABLE student_grades (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    course_id TEXT NOT NULL,
    semester TEXT NOT NULL,                 -- 学期：2025-2026-1
    grade_type TEXT NOT NULL DEFAULT 'final',-- 成绩类型：平时/期中/期末/总评
    score REAL,                             -- 分数
    gpa REAL,                               -- 绩点
    rank INTEGER,                           -- 排名
    grade_level TEXT DEFAULT '',            -- 等级：优/良/中/及格/不及格
    passed INTEGER NOT NULL DEFAULT 0,      -- 是否通过
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, course_id, semester, grade_type)
);

-- 学习资源表
CREATE TABLE learning_resources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    resource_id TEXT NOT NULL UNIQUE,
    course_id TEXT NOT NULL,
    title TEXT NOT NULL,                    -- 标题
    resource_type TEXT NOT NULL,            -- 类型：课件/习题/试卷/笔记/视频/参考资料
    chapter TEXT DEFAULT '',                -- 章节
    file_url TEXT DEFAULT '',               -- 文件地址
    content TEXT DEFAULT '',                -- 内容（Markdown，可直接展示）
    author TEXT DEFAULT '',                 -- 作者/上传者
    download_count INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 考试安排表
CREATE TABLE exam_schedules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exam_id TEXT NOT NULL UNIQUE,
    course_id TEXT NOT NULL,
    course_name TEXT NOT NULL,
    exam_type TEXT NOT NULL,                -- 类型：期中/期末/补考/重修
    date TEXT NOT NULL,                     -- 考试日期
    time_start TEXT NOT NULL,               -- 开始时间
    time_end TEXT NOT NULL,                 -- 结束时间
    location TEXT NOT NULL,                 -- 考试地点
    seat TEXT DEFAULT '',                   -- 座位号
    semester TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(course_id, exam_type, semester)
);

-- 错题本表
CREATE TABLE mistake_notebook (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    mistake_id TEXT NOT NULL UNIQUE,
    course_id TEXT DEFAULT '',
    subject TEXT DEFAULT '',                -- 主题/知识点
    question TEXT NOT NULL,                 -- 题目
    user_answer TEXT DEFAULT '',            -- 用户答案
    correct_answer TEXT DEFAULT '',         -- 正确答案
    analysis TEXT DEFAULT '',               -- 解析
    difficulty INTEGER DEFAULT 2,           -- 难度 1-5
    master_level INTEGER DEFAULT 0,         -- 掌握程度 0-100
    review_count INTEGER DEFAULT 0,         -- 复习次数
    last_review_date TEXT,                  -- 上次复习日期
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

#### 3.2.3 前端页面

- `/study` → 学业主页（Tab：课程/成绩/资源/考试）
- `/study/course/:id` → 课程详情
- `/study/grades` → 成绩查询/学业分析
- `/study/exams` → 考试安排
- `/study/mistakes` → 错题本

#### 3.2.4 子智能体

| 智能体Key | 名称 | 职责 |
|----------|------|------|
| study-advisor | 学业辅导助手 | 学习方法、时间管理、考试复习建议 |
| course-tutor | 课程助教 | 具体课程知识点讲解、习题解答、概念解释 |
| grade-analyst | 成绩分析师 | 成绩分析、GPA计算、学业预警、提升建议 |

---

### 3.3 心理健康模块

#### 3.3.1 功能清单

| 功能 | 说明 | 优先级 |
|------|------|--------|
| 心理测评 | 常见心理量表（SDS/SAS/SCL-90等） | P0 |
| 心理咨询预约 | 预约咨询师、咨询记录 | P0 |
| 心理知识科普 | 心理健康知识、自我调节方法 | P0 |
| 情绪日记 | 情绪记录、情绪趋势分析 | P0 |
| 危机干预 | 危机识别、紧急联系、求助热线 | P0 |
| 冥想放松 | 正念冥想、呼吸练习、放松训练 | P1 |
| 成长小组 | 心理成长小组、团体辅导 | P2 |

#### 3.3.2 数据模型

```sql
-- 心理测评量表表
CREATE TABLE psych_scales (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scale_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,                     -- 量表名称
    abbreviation TEXT DEFAULT '',           -- 缩写
    category TEXT NOT NULL,                 -- 分类：情绪/人格/压力/人际/睡眠
    description TEXT DEFAULT '',            -- 量表介绍
    question_count INTEGER NOT NULL,        -- 题目数量
    estimated_minutes INTEGER NOT NULL,     -- 预计用时（分钟）
    scoring_method TEXT NOT NULL,           -- 评分方法说明
    interpretation TEXT NOT NULL,           -- 结果解释标准（JSON）
    questions_json TEXT NOT NULL,           -- 题目JSON数组
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 测评记录表
CREATE TABLE psych_assessment_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    record_id TEXT NOT NULL UNIQUE,
    scale_id TEXT NOT NULL,
    answers_json TEXT NOT NULL,             -- 答案JSON
    scores_json TEXT NOT NULL,              -- 得分明细JSON
    total_score REAL,                       -- 总分
    level TEXT NOT NULL,                    -- 结果等级：正常/轻度/中度/重度
    result_summary TEXT DEFAULT '',         -- 结果摘要
    suggestion TEXT DEFAULT '',             -- 建议
    completed_at TEXT NOT NULL DEFAULT (datetime('now')),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 心理咨询师表
CREATE TABLE counselors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    counselor_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,                     -- 姓名
    title TEXT NOT NULL,                    -- 职称
    avatar TEXT DEFAULT '',
    department TEXT DEFAULT '',             -- 所属部门
    specialties TEXT DEFAULT '',            -- 擅长领域（JSON数组）
    bio TEXT DEFAULT '',                    -- 简介
    working_days TEXT DEFAULT '',           -- 工作日（JSON数组）
    available INTEGER NOT NULL DEFAULT 1,   -- 是否可预约
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 咨询预约表
CREATE TABLE counseling_appointments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    appointment_id TEXT NOT NULL UNIQUE,
    counselor_id TEXT NOT NULL,
    appointment_date TEXT NOT NULL,         -- 预约日期
    time_slot TEXT NOT NULL,                -- 时间段
    appointment_type TEXT NOT NULL,         -- 类型：面对面/线上/电话
    reason TEXT DEFAULT '',                 -- 咨询原因
    status TEXT NOT NULL DEFAULT 'pending', -- pending/confirmed/completed/cancelled
    notes TEXT DEFAULT '',                  -- 咨询师备注
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 咨询记录表
CREATE TABLE counseling_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    record_id TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL,
    counselor_id TEXT NOT NULL,
    appointment_id TEXT DEFAULT '',
    session_date TEXT NOT NULL,             -- 咨询日期
    duration_minutes INTEGER,               -- 时长（分钟）
    content_summary TEXT DEFAULT '',        -- 内容摘要（脱敏后可看）
    counselor_assessment TEXT DEFAULT '',   -- 咨询师评估（仅咨询师可见）
    follow_up_plan TEXT DEFAULT '',         -- 后续计划
    mood_before INTEGER,                    -- 咨询前心情 1-10
    mood_after INTEGER,                     -- 咨询后心情 1-10
    is_urgent INTEGER NOT NULL DEFAULT 0,   -- 是否危机个案
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 情绪日记表
CREATE TABLE mood_diary (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    diary_id TEXT NOT NULL UNIQUE,
    date TEXT NOT NULL,                     -- 日期
    mood_score INTEGER NOT NULL,            -- 心情评分 1-10
    mood_tags TEXT DEFAULT '',              -- 情绪标签（JSON数组）
    events TEXT DEFAULT '',                 -- 影响事件
    diary_content TEXT DEFAULT '',          -- 日记内容
    sleep_hours REAL,                       -- 睡眠时长
    exercise_minutes INTEGER,               -- 运动时长
    social_level INTEGER,                   -- 社交程度 1-5
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, date)
);

-- 心理知识库表
CREATE TABLE psych_articles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    article_id TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,                    -- 标题
    category TEXT NOT NULL,                 -- 分类：情绪管理/人际交往/学业压力/自我成长/恋爱婚姻
    summary TEXT DEFAULT '',                -- 摘要
    content TEXT NOT NULL,                  -- 内容（Markdown）
    cover_image TEXT DEFAULT '',            -- 封面图
    author TEXT DEFAULT '',                 -- 作者
    read_count INTEGER NOT NULL DEFAULT 0,
    is_crisis INTEGER NOT NULL DEFAULT 0,   -- 是否危机干预内容
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 危机热线表
CREATE TABLE crisis_hotlines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    hotline_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,                     -- 名称
    phone TEXT NOT NULL,                    -- 电话
    service_time TEXT DEFAULT '',           -- 服务时间
    description TEXT DEFAULT '',            -- 说明
    level INTEGER NOT NULL DEFAULT 1,       -- 优先级
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

#### 3.3.3 前端页面

- `/mental` → 心理健康主页（Tab：测评/咨询/科普/情绪）
- `/mental/assessment` → 心理测评
- `/mental/assessment/:scaleId` → 测评详情/答题
- `/mental/counseling` → 心理咨询预约
- `/mental/mood` → 情绪日记
- `/mental/crisis` → 危机干预（紧急入口）

#### 3.3.4 子智能体

| 智能体Key | 名称 | 职责 |
|----------|------|------|
| mental-helper | 心理陪伴助手 | 情绪疏导、心理支持、自我调节指导（非诊断） |
| crisis-responder | 危机响应助手 | 危机识别、即时干预、求助指引 |
| self-growth-guide | 成长向导 | 自我成长、人际交往、压力管理建议 |

> ⚠️ 注意：AI心理助手**不做诊断**，仅提供支持和建议，严重情况必须引导专业求助。

---

### 3.4 思政教育模块

#### 3.4.1 功能清单

| 功能 | 说明 | 优先级 |
|------|------|--------|
| 思政课程 | 思政课信息、课程大纲、学习资料 | P0 |
| 理论学习 | 党的创新理论、时政热点、学习材料 | P0 |
| 思政实践 | 社会实践、志愿服务、红色教育基地 | P0 |
| 思想汇报 | 思想汇报撰写指导、模板、提交记录 | P0 |
| 思政测评 | 思政知识答题、学习效果评估 | P1 |
| 红色文化 | 红色经典、革命故事、英雄人物 | P1 |
| 课程思政 | 专业课中的思政元素挖掘、案例库 | P1 |
| 思政大数据 | 学习行为分析、思政画像 | P2 |

#### 3.4.2 数据模型

```sql
-- 思政课程表
CREATE TABLE ideological_courses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    course_id TEXT NOT NULL UNIQUE,
    course_name TEXT NOT NULL,              -- 课程名称
    course_code TEXT DEFAULT '',             -- 课程代码
    credit REAL NOT NULL,                    -- 学分
    category TEXT NOT NULL,                  -- 分类：马克思主义理论/思政课/形势与政策/课程思政
    department TEXT NOT NULL,                -- 开课学院
    teacher TEXT DEFAULT '',                 -- 授课教师
    description TEXT DEFAULT '',             -- 课程简介
    syllabus TEXT DEFAULT '',                -- 教学大纲（Markdown）
    semester TEXT DEFAULT '',                -- 开课学期
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 理论学习材料表
CREATE TABLE theory_materials (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    material_id TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,                     -- 标题
    category TEXT NOT NULL,                  -- 分类：党的理论/时政热点/法律法规/党史学习
    source TEXT DEFAULT '',                  -- 来源
    content TEXT NOT NULL,                   -- 内容（Markdown）
    summary TEXT DEFAULT '',                 -- 摘要
    author TEXT DEFAULT '',                  -- 作者
    publish_date TEXT,                       -- 发布日期
    tags TEXT DEFAULT '',                    -- 标签（JSON数组）
    read_count INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 思政实践表
CREATE TABLE ideological_practice (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    practice_id TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,                     -- 实践活动名称
    category TEXT NOT NULL,                  -- 分类：社会实践/志愿服务/红色教育/调研实践
    description TEXT NOT NULL,               -- 活动描述
    location TEXT DEFAULT '',                -- 地点
    start_date TEXT,                         -- 开始日期
    end_date TEXT,                           -- 结束日期
    hours INTEGER DEFAULT 0,                 -- 学时/时长
    capacity INTEGER DEFAULT 0,              -- 容量
    registered_count INTEGER DEFAULT 0,      -- 已报名人数
    organizer TEXT DEFAULT '',               -- 组织单位
    contact TEXT DEFAULT '',                 -- 联系人
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 思想汇报表
CREATE TABLE ideological_reports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    report_id TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,                     -- 汇报标题
    period TEXT NOT NULL,                    -- 汇报时段：月度/季度/学期
    period_value TEXT NOT NULL,              -- 时段值：2026-03
    content TEXT NOT NULL,                   -- 汇报内容
    reflections TEXT DEFAULT '',             -- 个人反思
    study_summary TEXT DEFAULT '',           -- 学习总结
    status TEXT NOT NULL DEFAULT 'draft',    -- draft/submitted/reviewed
    review_comment TEXT DEFAULT '',          -- 审阅评语
    reviewer TEXT DEFAULT '',                -- 审阅人
    reviewed_at TEXT,                        -- 审阅时间
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 思政知识题库表
CREATE TABLE ideological_questions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    question_id TEXT NOT NULL UNIQUE,
    category TEXT NOT NULL,                  -- 分类：党史/理论/时政/法律/道德
    question_type TEXT NOT NULL,             -- 题型：单选/多选/判断/简答
    question TEXT NOT NULL,                  -- 题干
    options_json TEXT DEFAULT '',            -- 选项JSON
    answer TEXT NOT NULL,                    -- 正确答案
    analysis TEXT DEFAULT '',                -- 解析
    difficulty INTEGER NOT NULL DEFAULT 2,   -- 难度 1-5
    tags TEXT DEFAULT '',                    -- 标签
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 思政答题记录表
CREATE TABLE ideological_quiz_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    record_id TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL,
    quiz_type TEXT NOT NULL,                 -- 类型：daily/exam/practice
    total_questions INTEGER NOT NULL,
    correct_count INTEGER NOT NULL,
    score REAL NOT NULL,                     -- 得分
    duration_seconds INTEGER,                -- 用时
    answers_json TEXT NOT NULL,              -- 答题详情JSON
    completed_at TEXT NOT NULL DEFAULT (datetime('now')),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 红色文化资源表
CREATE TABLE red_culture_resources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    resource_id TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,                     -- 标题
    category TEXT NOT NULL,                  -- 分类：红色经典/革命故事/英雄人物/教育基地/红色影视
    content TEXT NOT NULL,                   -- 内容（Markdown）
    summary TEXT DEFAULT '',                 -- 摘要
    cover_image TEXT DEFAULT '',             -- 封面图
    media_url TEXT DEFAULT '',               -- 媒体地址（视频/音频）
    location TEXT DEFAULT '',                -- 相关地点
    period TEXT DEFAULT '',                  -- 历史时期
    tags TEXT DEFAULT '',                    -- 标签
    read_count INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

#### 3.4.3 前端页面

- `/ideological` → 思政教育主页（Tab：理论学习/思政课/实践/答题）
- `/ideological/theory` → 理论学习材料列表/详情
- `/ideological/courses` → 思政课程列表
- `/ideological/practice` → 思政实践活动
- `/ideological/quiz` → 思政知识答题
- `/ideological/report` → 思想汇报撰写/记录
- `/ideological/red-culture` → 红色文化

#### 3.4.4 子智能体

| 智能体Key | 名称 | 职责 |
|----------|------|------|
| ideological-tutor | 思政学习导师 | 理论解读、时政分析、学习指导 |
| theory-analyst | 理论分析助手 | 党的创新理论解读、政策分析、文章辅助 |

> **说明**：思政教育模块与已有入党教育（Party）互补——入党教育侧重流程追踪，思政教育侧重学习内容和理论提升。

---

## 四、P1 完善模块（第二期）

### 4.1 完善现有功能

| 模块 | 改进内容 |
|------|----------|
| 数字孪生 | 从Mock数据改为真实计算（学业/实践/身心/社交/创新五维） |
| 问答广场 | 实现完整的问答社区：提问、回答、点赞、关注、排行 |
| 校园文化 | 填充真实数据：校歌、讲座、活动、志愿、广播 |
| 学习伙伴 | 学习小组、组队学习、互助答疑 |

### 4.2 子智能体扩展

| 智能体Key | 名称 | 职责 |
|----------|------|------|
| party-guide | 入党启蒙导师 | 入党知识、政策解答、材料撰写指导 |
| competition-coach | 竞赛教练 | 竞赛信息、备赛指导、团队建议 |
| thesis-advisor | 毕设顾问 | 选题建议、论文写作指导、进度规划 |
| life-assistant | 生活助手 | 校园生活、衣食住行、办事指南 |
| freshman-guide | 新生向导 | 入学适应、大学生活规划、环境熟悉 |

### 4.3 新功能

| 功能 | 说明 |
|------|------|
| 生涯发展 | 考研、考公、留学、职业测评 |
| 学习路径规划 | 基于目标的个性化学习路径 |
| 奖学金助手 | 奖学金信息、申请指导、材料准备 |

---

## 五、子智能体体系总览

### 5.1 智能体架构（共 16 个）

```
┌─────────────────────────────────────────────────────┐
│                   Orchestrator                       │
│              （意图路由 + 并行 + 汇聚）              │
└─────────────────────────────────────────────────────┘
          │              │              │              │
    ┌─────┴─────┐  ┌─────┴─────┐  ┌─────┴─────┐  ┌─────┴─────┐
    │ 基础通用  │  │ 教育教学  │  │ 成长发展  │  │ 思政教育  │
    └───────────┘  └───────────┘  └───────────┘  └───────────┘
          │              │              │              │
  ┌───┬───┴───┐    ┌────┴────┐    ┌────┴────┐    ┌────┴────┐
  │   │       │    │         │    │         │    │         │
  QA 政策   心理   就业     学业    入党    竞赛   思政    理论
  通用 解读  疏导  指导     辅导   启蒙    教练   导师    分析
  (3)        (2)   (3)       (3)    (1)    (1)    (1)     (1)
```

### 5.2 智能体清单

| 编号 | 智能体Key | 名称 | 类别 | 知识库领域 |
|------|----------|------|------|-----------|
| 1 | qa-default | 通用问答助手 | 基础 | 全库 |
| 2 | policy-expert | 政策解读专家 | 基础 | 政策/FAQ |
| 3 | process-guide | 流程指引助手 | 基础 | 流程/FAQ |
| 4 | mental-helper | 心理陪伴助手 | 心理 | 心理科普 |
| 5 | crisis-responder | 危机响应助手 | 心理 | 危机干预 |
| 6 | career-advisor | 就业指导专家 | 就业 | 就业政策/招聘信息 |
| 7 | resume-expert | 简历优化助手 | 就业 | 简历模板/面试 |
| 8 | interview-coach | 面试教练 | 就业 | 面试题库 |
| 9 | study-advisor | 学业辅导助手 | 学业 | 学习方法/考试 |
| 10 | course-tutor | 课程助教 | 学业 | 课程资源/知识点 |
| 11 | grade-analyst | 成绩分析师 | 学业 | 成绩/学业分析 |
| 12 | party-guide | 入党启蒙导师 | 党建 | 入党知识/材料 |
| 13 | competition-coach | 竞赛教练 | 竞赛 | 竞赛信息/备赛 |
| 14 | freshman-guide | 新生向导 | 生活 | 入学/生活指南 |
| 15 | ideological-tutor | 思政学习导师 | 思政 | 理论材料/时政 |
| 16 | theory-analyst | 理论分析助手 | 思政 | 党的理论/政策分析 |

---

## 六、实施计划

### 6.1 第一阶段：P0 核心（约 2-3 周）

1. **数据库迁移**：创建就业、学业、心理健康相关表
2. **后端API**：三大模块的基础 CRUD + 检索
3. **子智能体**：新增 8 个子智能体（心理2 + 就业3 + 学业3）
4. **前端页面**：三大模块的核心页面
5. **知识库集成**：新知识库资源类型注册到检索引擎
6. **测试**：功能测试 + 知识库导入测试

### 6.2 第二阶段：P1 完善（约 2-3 周）

1. **数字孪生真实化**：五维数据计算逻辑
2. **问答广场落地**：完整社区功能
3. **更多子智能体**：入党、竞赛、毕设、生活、新生
4. **校园文化数据填充**
5. **辅导员/教师端扩展**

### 6.3 第三阶段：优化迭代（持续）

1. 个性化推荐
2. 多模态交互（语音/图片）
3. 学习分析深度优化
4. 性能优化

---

## 七、技术要点

### 7.1 知识库扩展

- 新增资源类型：`CareerPolicy`、`JobPosting`、`PsychArticle`、`LearningResource`
- 每种资源类型接入 FTS/BM25 全文检索
- 子智能体按资源类型过滤检索结果

### 7.2 意图路由扩展

在 `router.go` 中新增意图类别：

| 意图 | 触发关键词 | 对应智能体 |
|------|-----------|-----------|
| IntentCareer | 就业、求职、简历、面试、招聘、职业 | career-advisor |
| IntentStudy | 学习、考试、成绩、课程、知识点、复习 | study-advisor |
| IntentMental | 心理、情绪、压力、焦虑、抑郁、失眠 | mental-helper |
| IntentParty | 入党、党员、党课、思想汇报 | party-guide |
| IntentCompetition | 竞赛、比赛、参赛、获奖 | competition-coach |
| IntentIdeological | 思政、理论、时政、党史、红色、马克思主义 | ideological-tutor |

### 7.3 安全与隐私

- **心理健康数据**：高敏感，严格权限控制，咨询记录仅本人和咨询师可见
- **成绩数据**：仅本人和辅导员可见
- **AI心理助手免责声明**：明确说明不替代专业诊断
- **数据脱敏**：导出/统计时自动脱敏

---

## 八、验收标准

### 8.1 功能验收

- [ ] 就业指导：政策查询、招聘浏览、简历生成、面试模拟
- [ ] 学业学习：课程查询、成绩统计、资源检索、考试安排
- [ ] 心理健康：心理测评、咨询预约、情绪日记、危机热线
- [ ] 子智能体：14个子Agent全部注册，路由正确，回答可追溯
- [ ] 知识库：新增资源类型支持全文检索，来源可追溯

### 8.2 质量验收

- [ ] 单元测试覆盖率 > 70%
- [ ] API 响应时间 < 500ms（检索类）
- [ ] AI 回答引用率 > 80%（知识库类问题）
- [ ] 前端页面 0 控制台报错

---

## 九、风险与应对

| 风险 | 影响 | 应对措施 |
|------|------|----------|
| 心理健康数据敏感 | 高 | 严格权限控制 + 数据加密 + 免责声明 |
| 知识库数据量大 | 中 | 分批导入 + 索引优化 + 增量更新 |
| 子智能体过多路由不准 | 中 | 关键词权重调优 + LLM辅助路由 + 用户反馈修正 |
| 前端页面多工作量大 | 中 | 复用现有组件 + 通用列表/详情模板 |

---

## 附录：与现有模块的集成点

1. **知识库系统**：新增资源类型，复用 KBRepo 的 CRUD 和检索
2. **多智能体编排**：新增子 Agent，复用 Orchestrator 框架
3. **用户系统**：复用现有用户模型，新增角色（咨询师等）
4. **RBAC权限**：新增能力点（career:read, psych:read 等）
5. **通知系统**：复用通知框架（宣讲会提醒、咨询预约提醒等）
6. **数字孪生**：成绩、就业、心理健康数据输入到五维模型
