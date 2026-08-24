-- vOPC v2.0 重构：层级化虚拟 OPC 教学模块（L1-L4）。
-- 本迁移为纯增量、幂等（ALTER TABLE ADD COLUMN + CREATE TABLE IF NOT EXISTS），
-- 不回退、不破坏历史迁移 097-106。
--
-- 1) vopc_projects 新增 complexity_layer(1-4) 进入层级；L4 预留扩展位 ext_real_ai_enabled / ext_storage_provider（仅字段，不实现功能）。
-- 2) vopc_ai_roles 语义改为 virtual_guides（保留表名、仅加字段）：level / guide_type / template_content。
-- 3) L1 概念层内容表：vopc_learning_cards / vopc_quizzes（可播种；接口在无种子时回退内置默认）。
-- 4) vopc_client_evidence 甲方语义 → 通用反馈/自查证据：client_rep 保留列名但语义见应用层。

ALTER TABLE vopc_projects ADD COLUMN complexity_layer INTEGER NOT NULL DEFAULT 2;
ALTER TABLE vopc_projects ADD COLUMN ext_real_ai_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE vopc_projects ADD COLUMN ext_storage_provider TEXT NOT NULL DEFAULT '';

ALTER TABLE vopc_ai_roles ADD COLUMN level INTEGER NOT NULL DEFAULT 1;
ALTER TABLE vopc_ai_roles ADD COLUMN guide_type TEXT NOT NULL DEFAULT '';
ALTER TABLE vopc_ai_roles ADD COLUMN template_content TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS vopc_learning_cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_key TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    sort INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(card_key)
);

CREATE TABLE IF NOT EXISTS vopc_quizzes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    question TEXT NOT NULL,
    options TEXT NOT NULL DEFAULT '[]',
    answer_index INTEGER NOT NULL DEFAULT 0,
    sort INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_vopc_learning_cards_sort ON vopc_learning_cards(sort);
CREATE INDEX IF NOT EXISTS idx_vopc_quizzes_sort ON vopc_quizzes(sort);
