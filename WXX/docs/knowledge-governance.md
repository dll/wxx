# 知识治理与运营规范（摘录）

> 详见同目录 **`蔚小芯智能体.md`** §3.3.3、§3.3.4、§3.4、§6.4、§6.8。

## 原则

- **学生会采集整理 → 学院审核发布**；过期内容 **降权或 retired**，保留历史版本可追溯。
- **权限先于检索**：按 `ownerScope`、`roleScope`、`status` 过滤后再排序召回。
- **敏感字段**：个人敏感信息不得进入可检索正文；展示必须脱敏。

## 资源四类

`Policy` / `Process` / `FAQ` / `Activity` — 元数据与切分策略见总纲 §3.3.4、§6.8.1。

## 同步与导出

- 与「蔚园智答」：**增量 cursor**、幂等键 `(resourceId, version, status)`、包签名与 HTTPS。  
- 包结构：`manifest.json` + `resources.ndjson` + 可选 `attachments/`（见 `specs/export-package.md`）。

## 运营复盘

每周热点问题、兜底原因、修复计划：见总纲 §3.3.5 复盘机制。
