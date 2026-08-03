-- 052: 阶段三 数据底座 — 清理知识库测试残留（乱码/测试导入项）
DELETE FROM kb_resources WHERE resource_id IN ('ndjson-001', 'test-import-001');
