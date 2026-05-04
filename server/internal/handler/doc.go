// Package handler HTTP 请求处理层。
// 职责：解析请求参数、校验输入、调用 service 层、组装响应。
// 禁止在 handler 中编写业务逻辑或直接调用 repository/llm。
package handler
