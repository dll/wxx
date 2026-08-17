-- 090 督办工单（D5-3「洞察→工单」治理回环，2026-08-16）
-- 用于学院/学校治理看板产出洞察（如育人 KPI 补料、效能风险）后一键转为可排办的督办动作，
-- 分派给辅导员/教辅/党群责任人并跟踪状态流转。全部字段为新增表，不改动既有表，纯增量。
--
-- 状态流转（复用 feedback 流转心智，单表+操作记录，不做复杂状态机）：
--   pending(待办) -> processing(处理中) -> completed(完成) / closed(关闭)
-- source_type/source_key/source_desc 记录工单来源（洞察/KPI 语义，沿用到工单来源，诚实边界）
-- data_source 语义沿用洞察端：real=来自真实指标，not_available=补料类（缺数据需催办补料）
CREATE TABLE gov_tickets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_no TEXT NOT NULL UNIQUE,               -- 工单号（如 GT-20260816-XXXX）
    title TEXT NOT NULL,                          -- 督办标题
    category TEXT NOT NULL DEFAULT 'insight',     -- insight=治理洞察 / supplement=补料督办（D5-1 联动）
    source_type TEXT NOT NULL DEFAULT 'insight',  -- kpi / insight（来源端类型）
    source_key TEXT NOT NULL DEFAULT '',          -- 来源主键（如 KPI key nurture.employment_rate）
    source_desc TEXT NOT NULL DEFAULT '',         -- 来源描述（洞察说明，取自 source_desc）
    data_source TEXT NOT NULL DEFAULT 'not_available', -- 沿用洞察端三态：real / not_available / synthetic
    priority TEXT NOT NULL DEFAULT 'normal',      -- low / normal / high
    status TEXT NOT NULL DEFAULT 'pending',       -- pending/processing/completed/closed
    college TEXT NOT NULL DEFAULT '',             -- 学院归属（空=全校，school_admin；非空=本院）
    assignee_role TEXT NOT NULL DEFAULT '',       -- 责任人角色：counselor / teacher / assistant / party
    assignee_id BIGINT NOT NULL DEFAULT 0,        -- 责任人用户 id（0=未分派）
    assignee_name TEXT NOT NULL DEFAULT '',       -- 责任人姓名（分派时记录）
    deadline TEXT NOT NULL DEFAULT '',            -- 督办截止日期（YYYY-MM-DD，可为空）
    remark TEXT NOT NULL DEFAULT '',              -- 督办要求/备注
    created_by BIGINT NOT NULL DEFAULT 0,         -- 创建人（书记/学院管理员）
    created_by_role TEXT NOT NULL DEFAULT '',     -- 创建人角色
    closed_by BIGINT NOT NULL DEFAULT 0,          -- 关闭/完成操作人
    created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    closed_at TEXT
);

CREATE INDEX idx_gov_tickets_status ON gov_tickets(status);
CREATE INDEX idx_gov_tickets_assignee ON gov_tickets(assignee_id);
CREATE INDEX idx_gov_tickets_college ON gov_tickets(college);
CREATE INDEX idx_gov_tickets_source ON gov_tickets(source_type, source_key);

-- 工单操作记录（分派/状态流转背书的轻量审计，复用 feedback_logs 心智）
CREATE TABLE gov_ticket_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket_id BIGINT NOT NULL,
    action TEXT NOT NULL,           -- created / assigned / processing / completed / closed
    operator_id BIGINT NOT NULL DEFAULT 0,
    operator_name TEXT NOT NULL DEFAULT '',
    detail TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);

CREATE INDEX idx_gov_ticket_logs_ticket ON gov_ticket_logs(ticket_id);
