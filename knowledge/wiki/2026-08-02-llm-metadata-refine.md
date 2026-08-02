---
title: 文档元数据 LLM 精修方案与兜底策略
status: stable
created: 2026-08-02
last_reviewed: 2026-08-02
owner: 项目组
tags: 文档解析, llm, 元数据, 精修, 知识质量
sources:
  - docs/knowledge-refine.md
  - server/internal/service/document_service.go
  - server/internal/service/kb_service.go
---

# 文档元数据 LLM 精修方案与兜底策略

## 结论

上传文件的标题/摘要/关键词由纯规则生成，质量差会拉低检索召回。采用「一次 LLM 调用
精修元数据 + 三层兜底」方案，精修结果**只回填编辑表单，不自动写库**；管理端可批量
精修存量资源（`POST /kb/batch/refine`）。

## 背景

- 原规则：摘要 = 前 200 字（常截到文头噪声）；关键词 = n-gram 词频（无分词、无领域词表）；
  标题 = 前几行启发式。
- `filterByRelevance` 要求 title+summary 命中查询二元组，低质量元数据直接降低召回。

## 细节

- 精修参数：temperature 0.3、30s 超时、MaxTokens 500、正文截断 6000 字（头 2/3 + 尾 1/3）。
- 三层兜底：未注入 LLM / 调用失败 / 非 JSON / 校验不过 → 回退原值（`fallback=true`）。
- 写库路径：精修仅回填表单；写库仍走 `KBService`（人工确认）。批量精修逐条走
  `KBService.BatchRefine`，有效且非回退才写库，单条失败不影响其它条。
- 规则兜底也做了增强：摘要跳过文头噪声、标题识别《…》书名号、关键词领域词表加权。

## 反哺建议

该机制面向管理端，无需对学生开放。后续可在「解析质量门槛」上继续完善
（正文过短/无中文/乱码时拒绝或强制预览）。
