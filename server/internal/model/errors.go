package model

import "errors"

// 认证/授权相关哨兵错误（S-01 令牌吊销与账户状态强制）
// 由 repository 层在 UpsertFromContext 等处返回，中间件据此映射为 401/403。
var (
	// ErrAccountDisabled 账户已被停用或拒绝，禁止访问（映射 403）
	ErrAccountDisabled = errors.New("账户已被停用")

	// ErrTokenRevoked JWT 令牌版本旧于数据库权威版本，令牌已被吊销（映射 401）
	ErrTokenRevoked = errors.New("令牌已被吊销，请重新登录")

	// ErrAlertNotFound 情感告警不存在或不在当前用户范围内（映射 404/403）
	ErrAlertNotFound = errors.New("告警不存在或无权访问")
)
