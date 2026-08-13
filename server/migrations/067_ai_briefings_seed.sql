-- 067_ai_briefings_seed.sql — AI 简讯内置默认数据
-- 开源 / 闭源大模型各 Top10（2026-08 更新至最新版本，首次初始化时插入，幂等）
INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top1：DeepSeek-V4（Flash / Pro）', 'DeepSeek V4 Flash 于 2026-07-31 上线，V4 Pro Preview 于 2026-04-24 上线。V4 系列在推理、编程与 Agent 能力上大幅领先前代，API 定价显著低于闭源旗舰（输入 $0.14/百万 token），是当前开源模型中性价比与能力均衡的代表。', '', 'https://www.deepseek.com/', 'DeepSeek,开源,MoE,推理', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%DeepSeek-V4%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top2：Qwen3.8 Max（通义千问）', '阿里通义千问 Qwen3.8 Max 于 2026-07-19 上线，综合评分位居开源前列；Qwen3.7/3.5 系列覆盖多尺寸，中文能力强，配套百炼平台提供企业级 API，是国产开源生态的中坚力量。', '', 'https://github.com/QwenLM', '通义千问,Qwen,开源,中文', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Qwen3.8%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top3：Muse 系列（Meta）', 'Meta 发布开源 Muse Glimmer（30B 智能体模型，Apache 2.0）与 Muse Spark 1.2（2026-08-05 上线）。Muse Glimmer 可在 24GB 显存运行，不损失智能体可靠性，将开源智能体部署门槛从企业服务器降到个人开发者。', '', 'https://github.com/meta-llama', 'Meta,开源,智能体,Muse', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Muse%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top4：Kimi K3（月之暗面）', 'Kimi K3 于 2026-07-16 上线，综合评分 82.0 位列全球前四；Kimi 以超长上下文与文档问答著称，K2 推理模型开放部分权重，Agent 化能力强，是国产开源与商用双赛道标杆。', '', 'https://www.moonshot.cn/', 'Kimi,月之暗面,长上下文,Agent', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%开源大模型 Top4：Kimi%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top5：InternLM 3（上海 AI Lab）', '上海 AI Lab 开源书生 InternLM 系列，多尺寸覆盖，擅长中文长文本与工具调用，配套全链路开源工具链，是高校科研与国产开源生态的重要力量。', '', 'https://github.com/InternLM', 'InternLM,书生,开源,中文', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%InternLM 3%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top6：GLM-5.2（智谱）', '智谱 GLM-5.2 于 2026-06-16 上线，评测完整度 94.4% 位居开源前列；GLM 系列中文能力对标同量级开源顶尖，配套 CogView 文生图、GLM-4V 视觉等全模态能力，是国产大模型代表之一。', '', 'https://github.com/THUDM', 'GLM,智谱,开源,中文', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%GLM-5.2%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top7：MiniMax-01 系列', 'MiniMax 开源 456B 超大规模 MoE 模型，长上下文能力突出，适配长文档与复杂推理场景，是开源大参数模型的代表。', '', 'https://github.com/MiniMax-AI/MiniMax-01', 'MiniMax,开源,MoE,长上下文', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%开源大模型 Top7：MiniMax%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top8：Llama 4 / Gemma 3（海外开源）', 'Meta Llama 4 与 Google Gemma 3 继续迭代开源生态，多尺寸覆盖移动端到数据中心，配合社区微调与部署工具，仍是全球开源大模型的重要基线。', '', 'https://github.com/meta-llama', 'Llama,Gemma,开源,基线', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Llama 4 / Gemma 3%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top9：Qwen3.7 / 3.5（阿里通义）', '阿里通义 Qwen3.7 Max 于 2026-05-19 上线，Qwen3.5 系列覆盖 0.5B-72B 全尺寸；Qwen3.5-72B 多语能力突出，是国内开源生态部署最广的系列之一。', '', 'https://github.com/QwenLM', 'Qwen,通义千问,开源,多尺寸', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Qwen3.7 / 3.5%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '开源大模型 Top10：Mistral / 其他开源新星', 'Mistral 持续迭代开源 7B-123B 系列，以高效推理与 MoE 架构见长；另有 R1 推理模型、Ling 等国产新星持续丰富开源生态，本地部署方案日益成熟。', '', 'https://mistral.ai/', 'Mistral,开源,MoE,推理', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Mistral / 其他开源新星%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top1：GPT-5.6（OpenAI）', 'OpenAI GPT-5.6 系列（Sol / Terra / Luna）于 2026-07-09 上线，综合评分 84.4 位列前三；新增网络安全专用 GPT-5.6-Cyber，多模态与推理能力持续领先，是闭源旗舰代表。', '', 'https://openai.com/', 'GPT,OpenAI,闭源,旗舰', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%GPT-5.6%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top2：Claude Opus 5 / Fable 5（Anthropic）', 'Claude Opus 5 于 2026-07-24 上线，综合评分 86.2 登顶全球第一；Claude Fable 5 于 2026-06-09 上线（评分 85.3）。Claude 系列以长上下文、编程与 Agent 能力见长，是开发者社区最受信赖的模型之一。', '', 'https://claude.ai/', 'Claude,Anthropic,编程,Agent', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Claude Opus 5%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top3：Gemini 3.6（Google）', 'Google Gemini 3.6 Flash 于 2026-07-21 上线，原生多模态、超长上下文（百万 token），深度集成 Google 生态；Gemini 3.1 Pro 等系列覆盖通用与推理场景。', '', 'https://gemini.google.com/', 'Gemini,Google,多模态,长上下文', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Gemini 3.6%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top4：Kimi K3（月之暗面）', 'Kimi K3 于 2026-07-16 上线，综合评分 82.0 位列全球第四；以超长上下文与文档问答著称，Agent 化能力强，是国产闭源模型的头部选手。', '', 'https://www.moonshot.cn/', 'Kimi,月之暗面,长上下文,闭源', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%闭源大模型 Top4：Kimi%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top5：豆包（字节跳动）', '字节豆包大模型家族覆盖通用/角色扮演/语音等多场景，通过豆包 App 与火山引擎开放 API，是国内 C 端用户量最大的大模型应用之一。', '', 'https://www.doubao.com/', '豆包,字节,多场景,C端', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%闭源大模型 Top5：豆包%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top6：文心一言 5（百度）', '百度文心大模型迭代至 5.0 时代，中文理解与生成能力强，深度绑定百度搜索与百度智能云产业生态，覆盖办公与产业场景。', '', 'https://yiyan.baidu.com/', '文心一言,百度,中文,产业', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%文心一言 5%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top7：通义千问 Max（阿里云）', '阿里云通义千问 Qwen3.8 Max 商业版，依托百炼平台提供企业级 API 与行业方案，中文与多模态能力均衡，是国内云上大模型主力之一。', '', 'https://www.alibabacloud.com/', '通义千问,阿里云,企业级,百炼', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%闭源大模型 Top7：通义千问 Max%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top8：讯飞星火 5', '科大讯飞星火大模型迭代至 5.0 时代，中文语音交互优势突出，深耕教育场景，提供教学、办公、医疗等垂直解决方案。', '', 'https://xinghuo.xfyun.cn/', '讯飞星火,语音,教育,中文', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%讯飞星火 5%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top9：腾讯混元 Hy 3', '腾讯混元 Hy 3 于 2026-07-06 上线，接入微信/QQ 等国民级入口，Hy 3 主打低延迟高性价比，覆盖对话、内容与产业场景。', '', 'https://hunyuan.tencent.com/', '腾讯混元,微信,产业,Hy3', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%腾讯混元 Hy 3%');

INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, status)
SELECT 'AI 内参', 'ai_version', '闭源大模型 Top10：Grok 4.5（xAI）', 'xAI 推出 Grok 4.5（2026-07-08 上线），实时接入 X（推特）平台数据，风格自由，推理与多模态能力进入第一梯队。', '', 'https://x.ai/', 'Grok,xAI,实时,X平台', datetime('now'), 1
WHERE NOT EXISTS (SELECT 1 FROM ai_briefings WHERE topic LIKE '%Grok 4.5%');
