// Package context_engine 知识检索管道（Context Engine）。
// 核心流程：意图分类 → 结构化查询 → FTS/BM25 检索 → 上下文拼装 → 来源附加。
// 暴露统一的 Query 接口供 service 层调用，handler 禁止直接使用。
package context_engine
