-- 005_agents.sql
-- 多智能体管理表（P1 雏形）

CREATE TABLE IF NOT EXISTS agents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_id TEXT NOT NULL UNIQUE,           -- 唯一标识，如 "qa-default"
    name TEXT NOT NULL,                      -- 显示名称，如 "通用问答助手"
    description TEXT NOT NULL DEFAULT '',    -- 描述
    agent_type TEXT NOT NULL DEFAULT 'qa',   -- qa / policy / emotion / custom
    system_prompt TEXT NOT NULL DEFAULT '',  -- 自定义系统提示词
    model_provider TEXT NOT NULL DEFAULT '', -- deepseek / zhipu（空表示用默认）
    model_name TEXT NOT NULL DEFAULT '',     -- 具体模型名
    temperature REAL NOT NULL DEFAULT 0.7,   -- 温度 0.0-2.0
    max_tokens INTEGER NOT NULL DEFAULT 2048,
    status TEXT NOT NULL DEFAULT 'active'    -- active / inactive
        CHECK(status IN ('active','inactive')),
    config_json TEXT NOT NULL DEFAULT '{}',  -- 额外配置（JSON）
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 默认智能体种子数据
INSERT OR IGNORE INTO agents (agent_id, name, description, agent_type, system_prompt, model_provider)
VALUES
    ('qa-default', '通用问答助手', '处理日常学习、生活、校园事务等通用咨询', 'qa',
     '你是滁州学院信息学院的智能学工助手"蔚小芯"。你友好、耐心、专业，擅长解答学生关于学习、生活、校园事务的各种问题。回答应简洁清晰，使用中文，对政策类问题必须引用具体条款。',
     ''),

    ('policy-expert', '政策解读专家', '专门处理校规校纪、奖助学金、学籍管理等政策类问题', 'policy',
     '你是滁州学院政策解读专家。你的职责是准确解读学校各项规章制度、奖助学金政策、学籍管理规定等。回答必须引用具体政策条款，标明出处，不得编造或曲解政策内容。对于不确定的政策，明确告知学生咨询渠道。',
     ''),

    ('emotion-counselor', '心理关怀助手', '情感预警分析专用代理，评估学生心理状态', 'emotion',
     '你是高校学生心理健康评估助手。你的任务是分析学生消息中的情感状态，识别潜在的心理风险。你必须严格按JSON格式返回分析结果。重点关注自我伤害意图、严重绝望感、极端社会孤立和剧烈情绪变化。不要过度敏感。',
     '');
