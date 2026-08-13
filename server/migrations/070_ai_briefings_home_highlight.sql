-- 070_ai_briefings_home_highlight.sql — 首页 AI 简讯突出「国产开源大语言模型」
-- 将国产开源代表条目置顶（更新 published_at 为较晚时间，列表按时间倒序展示）
UPDATE ai_briefings
SET published_at = datetime('now', '+1 hour')
WHERE topic LIKE '%DeepSeek-V4%';

UPDATE ai_briefings
SET published_at = datetime('now', '+59 minutes')
WHERE topic LIKE '%Qwen3.8%';

UPDATE ai_briefings
SET published_at = datetime('now', '+58 minutes')
WHERE topic LIKE '%GLM-5.2%';

UPDATE ai_briefings
SET published_at = datetime('now', '+57 minutes')
WHERE topic LIKE '%InternLM 3%';

UPDATE ai_briefings
SET published_at = datetime('now', '+56 minutes')
WHERE topic LIKE '%MiniMax-01%';
