-- vOPC 团队组建闭环：保留旧表，增量记录组队方式、模拟标记和岗位说明。
ALTER TABLE vopc_projects ADD COLUMN team_mode TEXT NOT NULL DEFAULT 'auto';
ALTER TABLE vopc_projects ADD COLUMN is_demo INTEGER NOT NULL DEFAULT 0;
ALTER TABLE vopc_ai_roles ADD COLUMN responsible_stages TEXT NOT NULL DEFAULT '[]';
ALTER TABLE vopc_ai_roles ADD COLUMN responsibilities TEXT NOT NULL DEFAULT '';
ALTER TABLE vopc_ai_roles ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_vopc_projects_demo ON vopc_projects(is_demo, updated_at);
