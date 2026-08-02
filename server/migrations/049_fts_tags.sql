-- 049_fts_tags.sql
-- 将 tags（关键词）纳入 FTS5 索引，使精修后的关键词参与 BM25 检索。
--
-- 背景：001_init.sql 的 kb_fts 只索引 resource_id/title/summary/content，
-- 关键词（tags）此前仅用于 SearchStructured 的 LIKE 匹配，不参与全文检索。
-- 本迁移补齐该缺口：重建 kb_fts（含 tags 列）+ 改写触发器携带 tags + 重建索引。
--
-- 说明：FTS5 虚拟表不支持 ALTER TABLE ADD COLUMN，故采用 DROP + CREATE 重建；
-- kb_resources 为外部内容表不受影响，重建后通过 'rebuild' 从存量行回填。
--
-- 注意：FTS5 虚拟表与触发器在 Turso 上不可用，由 runMigrations(isTurso) 自动跳过。

DROP TRIGGER IF EXISTS kb_fts_insert;
DROP TRIGGER IF EXISTS kb_fts_update;
DROP TRIGGER IF EXISTS kb_fts_delete;
DROP TABLE IF EXISTS kb_fts;

CREATE VIRTUAL TABLE IF NOT EXISTS kb_fts USING fts5(
    resource_id,
    title,
    summary,
    content,
    tags,
    content=kb_resources,
    content_rowid=id,
    tokenize='unicode61'
);

CREATE TRIGGER IF NOT EXISTS kb_fts_insert AFTER INSERT ON kb_resources BEGIN
    INSERT INTO kb_fts(rowid, resource_id, title, summary, content, tags)
    VALUES (new.id, new.resource_id, new.title, new.summary, new.content, new.tags);
END;

CREATE TRIGGER IF NOT EXISTS kb_fts_update AFTER UPDATE ON kb_resources BEGIN
    INSERT INTO kb_fts(kb_fts, rowid, resource_id, title, summary, content, tags)
    VALUES ('delete', old.id, old.resource_id, old.title, old.summary, old.content, old.tags);
    INSERT INTO kb_fts(rowid, resource_id, title, summary, content, tags)
    VALUES (new.id, new.resource_id, new.title, new.summary, new.content, new.tags);
END;

CREATE TRIGGER IF NOT EXISTS kb_fts_delete AFTER DELETE ON kb_resources BEGIN
    INSERT INTO kb_fts(kb_fts, rowid, resource_id, title, summary, content, tags)
    VALUES ('delete', old.id, old.resource_id, old.title, old.summary, old.content, old.tags);
END;

-- 从外部内容表重建索引，为存量行回填 tags 列
INSERT INTO kb_fts(kb_fts) VALUES('rebuild');
