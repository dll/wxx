# server/ — 蔚小芯 Go 后端

## 目录结构

```
server/
├── cmd/
│   ├── server/main.go       # HTTP 服务入口
│   └── migrate/main.go      # SQLite 迁移工具
├── internal/
│   ├── config/              # 配置加载（.env → struct）
│   ├── handler/             # Gin HTTP handler（按业务域拆分）
│   ├── middleware/          # JWT 鉴权、RBAC、限流、审计、CORS
│   ├── model/               # 数据模型（struct + 表映射）
│   ├── service/             # 业务逻辑层
│   ├── repository/          # SQLite 数据访问层
│   ├── agent/               # 多智能体管理中心 + Eino 编排封装
│   ├── context_engine/      # Context Engine：结构化查询 + FTS + 拼装
│   └── llm/                 # 智谱/DeepSeek/讯飞 API 客户端
├── migrations/              # SQLite DDL 脚本（001_init.sql ...）
├── data/                    # 运行时数据目录（.gitignore）
└── go.mod
```

## 分层规则

- **handler** 只做参数校验和响应组装，不写业务逻辑
- **service** 编排 repository + llm + agent，实现业务规则
- **repository** 只做 SQL 操作，不依赖 HTTP 或模型 API
- **禁止** handler 直接调用 repository 或 llm
