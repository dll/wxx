// Package server 提供迁移文件嵌入
package server

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS
