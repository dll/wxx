// Package agent 多智能体管理中心。
// 编排采用自研的关键词加权意图路由（router.go）+ goroutine 并行编排（orchestrator.go），
// 并非 Eino 框架集成（go.mod 无 cloudwego/eino 依赖）。
// 关键链路可选 Temporal 工作流引擎。
package agent
