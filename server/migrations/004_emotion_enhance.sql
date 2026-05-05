-- 004_emotion_enhance.sql — 情感预警表增强
-- 在已有 emotion_logs 基础上扩展字段，支持告警管理

-- 新增消息文本字段
ALTER TABLE emotion_logs ADD COLUMN message_text TEXT NOT NULL DEFAULT '';

-- 新增 LLM 分析原始结果
ALTER TABLE emotion_logs ADD COLUMN analysis_json TEXT NOT NULL DEFAULT '{}';

-- 新增告警处理状态
ALTER TABLE emotion_logs ADD COLUMN status TEXT NOT NULL DEFAULT 'pending'
    CHECK(status IN ('pending','acknowledged','resolved'));

-- 新增处理人信息
ALTER TABLE emotion_logs ADD COLUMN acknowledged_by TEXT NOT NULL DEFAULT '';
ALTER TABLE emotion_logs ADD COLUMN acknowledged_at TEXT DEFAULT NULL;

-- 新增唯一告警 ID（用于 API 操作）
ALTER TABLE emotion_logs ADD COLUMN alert_id TEXT NOT NULL DEFAULT '';

-- 新增用户名（冗余，方便查询）
ALTER TABLE emotion_logs ADD COLUMN username TEXT NOT NULL DEFAULT '';

-- 为已有行生成 alert_id（如果有历史数据）
UPDATE emotion_logs SET alert_id = 'alert-' || id WHERE alert_id = '';
