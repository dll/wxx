// Package agent 多智能体管理中心。
// 编排采用 Eino Graph（eino_orchestrator.go）作为图运行时，
// 图节点内部复用关键词加权意图路由（router.go）与业务 Agent 并行执行（orchestrator.go）。
// 关键链路可选 Temporal 工作流引擎。
package agent
