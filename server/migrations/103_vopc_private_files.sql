-- vOPC 私有文件受控上传与鉴权下载（P0）。
-- 存储层为本地受控目录（.uploads/vopc 下按项目隔离），数据库只存不可猜 object_key 与元数据；
-- 不引入云对象存储 SDK，不存储文件原始磁盘路径，不暴露可猜 URL。
-- 本迁移仅新增一张表，不改历史迁移、不改既有 artifact/version 门禁列定义。

CREATE TABLE IF NOT EXISTS vopc_files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    uploader_user_id INTEGER NOT NULL,
    object_key TEXT NOT NULL,
    file_name TEXT NOT NULL DEFAULT '',
    mime_type TEXT NOT NULL DEFAULT '',
    size_bytes INTEGER NOT NULL DEFAULT 0,
    checksum TEXT NOT NULL DEFAULT '',
    storage_status TEXT NOT NULL DEFAULT 'pending',
    artifact_version_id INTEGER,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, object_key),
    FOREIGN KEY(project_id) REFERENCES vopc_projects(id) ON DELETE CASCADE,
    FOREIGN KEY(uploader_user_id) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_vopc_files_project ON vopc_files(project_id, created_at);
CREATE INDEX IF NOT EXISTS idx_vopc_files_key ON vopc_files(object_key);
CREATE INDEX IF NOT EXISTS idx_vopc_files_version ON vopc_files(artifact_version_id);
