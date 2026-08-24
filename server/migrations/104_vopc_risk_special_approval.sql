-- vOPC R3 专项审批实体（PRD §13.1：R3 禁止或专项审批 → 默认禁止，按学校制度专项审批）。
-- R3 不走平台内置双人 approve 伪机制；仅在存在学校制度批准的外部专项审批记录时，
-- 项目里程碑/发布/文件外发方可解除 R3 阻断。
-- 该记录由持有 vopc.risk.manage 的授权管理员在平台上登记学校制度批准结果
-- （reason=批准理由，approver=审批主体/机构，ref=学校制度批文或批准依据编号）。

CREATE TABLE IF NOT EXISTS vopc_risk_special_approvals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    reason TEXT NOT NULL,
    approver TEXT NOT NULL,
    ref TEXT NOT NULL DEFAULT '',
    created_by INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(project_id) REFERENCES vopc_projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_vopc_risk_special_approvals_project ON vopc_risk_special_approvals(project_id, created_at);
