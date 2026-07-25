# 知识导出包规范（摘要）

> 权威 JSON 示例与字段见 `docs/蔚小芯智能体.md`（相对 `WXX/`）§6.8.6～§6.8.9。

## 包结构

```
manifest.json          # 包元数据、校验、游标、签名
resources.ndjson       # 每行一条资源 JSON
attachments/           # 可选：原始附件
```

## 增量同步

- **cursor** 单调递增；**幂等键** `(resourceId, version, status)`。  
- **冲突**：高版本覆盖低版本；`retired` 必须传播。

## 安全底线

- HTTPS、`Authorization: Bearer`、**HMAC 包签名**、哈希校验、导入审计。

## API 形态（建议）

- `GET /kb/export?ownerScope=&ownerId=&sinceCursor=&limit=`  
- `POST /kb/import`  

返回体字段示例见总纲 §6.8.8.1。
