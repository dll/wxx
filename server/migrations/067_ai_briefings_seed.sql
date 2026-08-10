-- 067_ai_briefings_seed.sql — AI 简讯内置默认数据
-- 开源 / 闭源大模型各 Top10（首次初始化时插入，幂等）
INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top1：DeepSeek-V3 / R1', '深度求索开源 MoE 模型，V3 以 671B 总参（37B 激活）对标闭源旗舰，R1 推理模型采用强化学习蒸馏，中文场景表现突出，商用宽松。', '', 'https://www.deepseek.com/', 'DeepSeek,开源,MoE,推理', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%DeepSeek-V3 / R1%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top2：Qwen2.5（通义千问）', '阿里开源千问系列，覆盖 0.5B-72B 全尺寸，Qwen2.5-72B 在多语言与代码能力上领先开源梯队，配套 Qwen-VL 视觉与 Qwen-Audio 多模态。', '', 'https://qwenlm.github.io/', '通义千问,Qwen,开源,多模态', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Qwen2.5%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top3：Llama 3（Meta）', 'Meta 开源的 Llama 3 系列（8B/70B/405B），405B 为开源最大规模之一，生态适配最广，被大量下游微调与部署方案采用。', '', 'https://ai.meta.com/llama/', 'Llama,Meta,开源,生态', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Llama 3%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top4：Mistral 系列', '法国 Mistral AI 开源 7B/8x7B 系列，以高参效比著称，Mixtral 8x7B 采用稀疏专家架构，被广泛用于边缘与私有化部署。', '', 'https://mistral.ai/', 'Mistral,开源,MoE,参效比', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Mistral%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top5：Gemma（Google）', 'Google 开源轻量级系列，基于 Gemini 技术沉淀，2B/7B 尺寸适合端侧推理，安全与责任对齐规范完善。', '', 'https://ai.google.dev/gemma', 'Gemma,Google,开源,轻量', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Gemma%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top6：GLM-4（智谱）', '智谱开源的 GLM-4 系列（9B/32B），中文能力对标同量级开源顶尖，配套 CodeGeeX 代码助手与 Agent 工具链。', '', 'https://github.com/THUDM/GLM-4', 'GLM,智谱,开源,中文,Agent', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%GLM-4%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top7：InternLM / 书生·浦语', '上海 AI Lab 开源书生系列，多尺寸覆盖，擅长中文长文本与工具调用，配套全链路训练工具链 OpenCompass。', '', 'https://github.com/InternLM/InternLM', '书生,InternLM,开源,中文', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%InternLM%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top8：Yi-1.5 / Baichuan', '零一万物 Yi 与百川 Baichuan 系列，中文开源先行者，覆盖 6B-34B 尺寸，适合中文场景私有化落地。', '', 'https://github.com/01-ai/Yi', 'Yi,Baichuan,开源,中文', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Yi-1.5 / Baichuan%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top9：Phi-4（微软）', '微软开源小型化推理模型，以数据质量取胜，小尺寸达到中大模型水平，适合低资源环境部署。', '', 'https://azure.microsoft.com/products/phi-4', 'Phi,微软,开源,小型化', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Phi-4%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top10：MiniMax-01 / Kimi-K2（部分开源）', 'MiniMax 开源 456B 超大规模 MoE，Kimi 开放 K2 推理模型权重，国产开源在长上下文与推理能力上持续突破。', '', 'https://github.com/MiniMax-AI/MiniMax-01', 'MiniMax,Kimi,开源,MoE', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%MiniMax-01%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top1：GPT-4o / o3（OpenAI）', 'OpenAI 旗舰 GPT-4o 支持文本/视觉/语音统一原生理解，o3 推理模型在数学与代码竞赛上刷新 SOTA，是闭源生态事实标杆。', '', 'https://openai.com/', 'GPT-4o,o3,OpenAI,推理', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%GPT-4o / o3%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top2：Claude（Anthropic）', 'Claude 系列以长上下文、编程与 Agent 能力见长，Claude 3.7 具备混合推理模式，企业安全对齐评分领先。', '', 'https://claude.ai/', 'Claude,Anthropic,长上下文,Agent', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Claude%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top3：Gemini（Google）', 'Gemini 1.5/2.0 原生多模态、超长上下文（百万 token），深度集成 Google 搜索与 Workspace 生态。', '', 'https://gemini.google.com/', 'Gemini,Google,多模态,长上下文', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Gemini%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top4：文心一言（百度）', '百度文心大模型 4.0，中文理解与生成能力强，深度绑定百度搜索与百度智能云产业场景。', '', 'https://yiyan.baidu.com/', '文心一言,百度,中文', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%文心一言%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top5：豆包（字节跳动）', '字节豆包大模型家族覆盖通用/角色扮演/语音等多场景，通过豆包 App 与火山引擎大规模商用。', '', 'https://www.doubao.com/', '豆包,字节跳动,多场景', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%豆包%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top6：Kimi（月之暗面）', 'Kimi 以超长上下文与文档问答著称，K2 推理能力开放部分权重，Agent 化产品 Moonshot 生态活跃。', '', 'https://www.kimi.com/', 'Kimi,月之暗面,长上下文,Agent', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%闭源大模型 Top6：Kimi%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top7：通义千问商业版（阿里云）', '阿里云通义千问 qwen-max 商业版，依托百炼平台提供企业级 API 与行业大模型解决方案。', '', 'https://tongyi.aliyun.com/', '通义千问,阿里云,企业级', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%通义千问商业版%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top8：讯飞星火', '科大讯飞星火大模型，中文语音交互优势突出，教育场景深耕，星火 4.0 提升多模态与工具调用。', '', 'https://xinghuo.xfyun.cn/', '讯飞星火,语音,教育', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%讯飞星火%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top9：腾讯混元', '腾讯混元大模型，接入微信/QQ 等国民级入口，Hunyuan Turbo 主打低延迟高性价比商用。', '', 'https://hunyuan.tencent.com/', '混元,腾讯,商用', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%腾讯混元%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top10：Grok（xAI）', 'xAI 推出的 Grok 系列，实时接入 X（推特）平台数据，风格自由，Grok 3 提升推理与多模态能力。', '', 'https://x.ai/', 'Grok,xAI,实时数据', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Grok%');
