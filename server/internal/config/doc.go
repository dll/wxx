// Package config 配置加载包。
// 负责从环境变量（.env）读取应用配置并映射到结构体。
// 开发环境使用 godotenv 加载 .env 文件，生产环境直接读取系统环境变量。
package config
