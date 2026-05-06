// Package temporal Temporal 工作流引擎集成层。
// 提供 Temporal Client 单例、工作流定义、活动包装。
// 所有工作流通过 activity 调用 service 层，复用现有业务逻辑。
// 当 TEMPORAL_HOST_PORT 为空时，本包所有功能优雅降级（不启用）。
package temporal
