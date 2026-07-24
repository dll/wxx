# 蔚小芯 Code Wiki

> 计算机科学与工程学院（网络空间安全学院）**蔚小芯**：Flutter 客户端 + Go/Gin 后端 + Context Engine（结构化 + FTS/BM25 为主） + Eino 编排 + 第三方大模型 API；sources 可追溯，向量与 Agentic RAG 可插拔。

---

## 目录

- [1. 项目概述](#1-项目概述)
- [2. 整体架构](#2-整体架构)
- [3. 后端架构详解](#3-后端架构详解)
- [4. 前端架构详解](#4-前端架构详解)
- [5. Context Engine 上下文引擎](#5-context-engine-上下文引擎)
- [6. 多智能体编排系统](#6-多智能体编排系统)
- [7. 数据库设计](#7-数据库设计)
- [8. 权限模型（RBAC + Capability）](#8-权限模型rbac--capability)
- [9. LLM 集成与客户端抽象](#9-llm-集成与客户端抽象)
- [10. 关键流程时序](#10-关键流程时序)
- [11. 依赖关系](#11-依赖关系)
- [12. 配置说明](#12-配置说明)
- [13. 构建与运行](#13-构建与运行)
- [14. 关键文件索引](#14-关键文件索引)
- [15. 设计原则](#15-设计原则)

---

## 1. 项目概述

### 1.1 项目定位

蔚小芯是滁州学院计算机学院的智慧学工 AI 助手，面向学生、辅导员、教师、教辅、学生会、学院管理员等多角色，提供 AI 问答、情感预警、知识管理、办事流程指引、学业分析等一体化智能服务。

### 1.2 技术栈速查

| 层级 | 技术选型 |
|------|---------|
| 前端 | Flutter 3.x / Dart / go_router / Provider / Dio / Hive |
| 后端 | Go 1.25 / Gin / JWT / SQLite (FTS5) |
| 编排 | Eino 风格多 Agent 编排 / Temporal（可选） |
| 模型 | 智谱清言 GLM-4 / DeepSeek / 讯飞星火（语音） |
| 数据库 | SQLite + FTS5（BM25 全文检索） |
| 部署 | Vercel（后端 Serverless）/ Cloudflare Pages（前端） |
| 小程序 | 微信小程序（WebView 壳，AppID: wx811d1225e67b8f38） |

### 1.3 角色体系

| 角色 | 层级 | 主要职责 |
|------|------|---------|
| `sys_admin` | 系统管理员 | 全局配置、全量审计 |
| `school_admin` | 学校管理员 | 智能体管理、校级用户管理 |
| `college_admin` | 学院管理员 | 学院数据看板、问题预案、毕设管理 |
| `counselor` | 辅导员 | 情感预警、班级管理、知识管理 |
| `teacher` | 教师 | 备课助手、考试出题、学情分析 |
| `assistant` | 教辅 | 排课检查、毕业审核、考试安排 |
| `student_union` | 学生会 | 活动策划、海报生成、知识提交 |
| `student` | 学生 | 个人 AI 助手、学业规划、社区问答 |
| `guest` | 游客 | 公开知识浏览、基础对话 |

---

## 2. 整体架构

### 2.1 架构总览

```
┌─────────────────────────────────────────────────────────────────────┐
│                         客户端层（Flutter）                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │
│  │  学生端  │  │ 辅导员端 │  │  教师端  │  │  管理/教辅/学生会 │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────────┘   │
│                    go_router + Provider + Dio                        │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ HTTPS + JWT
┌──────────────────────────────▼──────────────────────────────────────┐
│                      后端 API 层（Go/Gin）                           │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                        Middleware                             │   │
│  │   CORS / JWT / RBAC / Audit / PII Mask / TraceID            │   │
│  └──────────────────────────────┬───────────────────────────────┘   │
│                                 │                                   │
│  ┌──────────────────────────────▼───────────────────────────────┐  │
│  │                        Handler 层                             │  │
│  │   auth / chat / kb / session / emotion / agent / admin ...   │  │
│  └──────────────────────────────┬───────────────────────────────┘  │
│                                 │                                   │
│  ┌──────────────────────────────▼───────────────────────────────┐  │
│  │                        Service 层                             │  │
│  │   ChatService / KbService / EmotionService / AgentService... │  │
│  └──────────────────────────────┬───────────────────────────────┘  │
│                                 │                                   │
│  ┌──────────────────────────────▼───────────────────────────────┐  │
│  │                      Repository 层                            │  │
│  │   UserRepo / KBRepo / SessionRepo / MessageRepo ...          │  │
│  └──────────────────────────────┬───────────────────────────────┘  │
└─────────────────────────────────┼───────────────────────────────────┘
                                  │
┌─────────────────────────────────▼───────────────────────────────────┐
│                        数据与外部服务层                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                │
│  │  SQLite DB  │  │  LLM APIs   │  │  校外系统    │                │
│  │  (FTS5)     │  │  (智谱/DS)  │  │  (学工/一表通)│                │
│  └─────────────┘  └─────────────┘  └─────────────┘                │
│  ┌─────────────┐  ┌─────────────┐                                   │
│  │ Temporal(可)│  │ 讯飞语音ASR │                                   │
│  └─────────────┘  └─────────────┘                                   │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 分层架构原则

项目遵循 **Clean Architecture**（整洁架构）原则，依赖方向由外向内：

1. **Handler 层（接口适配层）**：HTTP 请求处理、参数校验、响应封装
2. **Service 层（用例层）**：业务逻辑编排、事务边界
3. **Repository 层（数据访问层）**：数据库操作，通过 Port 接口解耦
4. **Model 层（实体层）**：核心业务实体与 DTO

---

## 3. 后端架构详解

### 3.1 目录结构

```
server/
├── cmd/
│   ├── server/          # 主服务入口
│   ├── migrate/         # 数据库迁移工具
│   ├── eval/            # 评估工具
│   └── stress/          # 压测工具
├── internal/
│   ├── agent/           # 多智能体编排
│   ├── auth/            # 认证与权限（Capability 模型）
│   ├── config/          # 配置加载
│   ├── context_engine/  # Context Engine 文档
│   ├── handler/         # HTTP Handler 层
│   ├── llm/             # LLM 客户端抽象与实现
│   ├── middleware/      # Gin 中间件
│   ├── model/           # 数据模型（Entity + DTO）
│   ├── ports/           # 端口（接口定义，依赖倒置）
│   ├── repository/      # Repository 实现
│   ├── service/         # 业务服务层
│   ├── temporal/        # Temporal 工作流（可选）
│   ├── testutil/        # 测试工具
│   └── util/            # 工具函数
├── migrations/          # SQL 迁移文件
├── pkg/
│   └── app/             # 应用组装（依赖注入）
├── data/
│   └── seed/            # 种子数据
└── testdata/            # 测试数据
```

### 3.2 应用启动与依赖注入

后端采用 **手动依赖注入** 模式，所有组件在 `pkg/app/app.go` 中组装：

**启动流程：**

1. `config.Load()` 加载环境变量配置
2. `initDB()` 初始化 SQLite 连接（WAL 模式 + 外键约束）
3. `runMigrations()` 执行数据库迁移（版本化管理）
4. 初始化各层依赖：Repository → Service → Handler
5. `setupRouter()` 构建 Gin 路由树
6. 启动 HTTP 服务，支持优雅关闭

**关键代码位置：** [app.go](server/pkg/app/app.go)

### 3.3 Handler 层

Handler 层负责 HTTP 请求的接收、参数解析、调用 Service、返回响应。每个业务域对应一个 Handler 文件。

**主要 Handler：**

| Handler | 职责 | 核心方法 |
|---------|------|---------|
| `AuthHandler` | 认证与用户信息 | Login / Profile / ChangePassword / GetCapabilities |
| `ChatHandler` | AI 对话 | Ask |
| `SessionHandler` | 会话管理 | List / GetMessages / Delete / Rename |
| `KBHandler` | 知识库管理 | List / Create / Update / Import / Approve / Reject |
| `EmotionHandler` | 情感预警 | Analyze / ListAlerts / Trends |
| `AgentHandler` | 智能体管理 | List / Create / Update / Delete |
| `AdminHandler` | 管理后台 | Metrics / ListUsers / Audit / ImportStudents |
| `FeedbackHandler` | 反馈管理 | Submit / List / Resolve |
| `ExportHandler` | 知识导出 | Export / ExportAnswer |
| `VoiceHandler` | 语音服务 | ASR / TTS |

### 3.4 Service 层

Service 层封装核心业务逻辑，通过 Port 接口依赖 Repository，不直接依赖具体数据库实现。

**核心 Service：**

#### ChatService — 问答主链路

[chat_service.go](server/internal/service/chat_service.go)

问答服务是 Context Engine 的核心入口，主链路包含 6 个阶段：

1. **缓存检查** — 固定流程问题命中内存缓存直接返回（24h TTL）
2. **FAQ 持久化缓存** — 在 FAQ 资源中按 BM25 + Jaccard 相似度检索
3. **会话管理** — 创建/获取会话，保存用户消息
4. **多智能体协同** — agentID 为空时启用编排器
5. **FTS5/BM25 知识检索** — 从知识库检索相关资料
6. **LLM 调用 + AnswerCard 构造** — 拼装上下文、调用 LLM、构造带引用的回答

**缓存策略：**
- 内存缓存：24h TTL，后台每 30 分钟清理过期条目
- FAQ 持久化缓存：写入 kb_resources（type=FAQ），用户反馈"回答有误"时立即失效
- 缓存键：问题去空格小写后 FNV-1a 64-bit 哈希

#### KbService — 知识管理服务

负责知识资源的 CRUD、导入、审核等全生命周期管理。

#### EmotionService — 情感预警服务

基于 LLM 对对话内容进行情感分析，识别风险等级（low/medium/high），支持告警通知。

### 3.5 Repository 层

Repository 层封装数据访问，实现 `internal/ports/` 中定义的接口。

**主要 Repository：**

| Repository | 对应表 | 核心能力 |
|-----------|--------|---------|
| `UserRepo` | users | 用户 CRUD、角色管理 |
| `SessionRepo` | sessions | 会话管理、Touch 更新 |
| `MessageRepo` | messages | 消息存储、上下文获取 |
| `KBRepo` | kb_resources + kb_fts | FTS5/BM25 检索、资源 CRUD |
| `EmotionRepo` | emotion_logs | 情感记录、告警查询 |
| `AuditRepo` | audit_logs | 审计日志 |
| `AgentRepo` | agents | 智能体配置 |

**KBRepo 全文检索：**
- 使用 SQLite FTS5 虚拟表 + BM25 算法
- 权重分配：title > summary > content
- 支持 owner_scope / role_scope 权限过滤
- 触发器自动同步 kb_resources → kb_fts

### 3.6 Middleware 中间件

| 中间件 | 职责 |
|--------|------|
| `CORS` | 跨域资源共享 |
| `JWTAuth` | JWT 令牌验证，注入 UserContext |
| `TraceID` | 请求链路追踪 ID |
| `PIIMask` | 个人敏感信息检测与脱敏 |
| `AuditLog` | 操作审计日志 |
| `Consent` | 隐私同意校验 |
| `RBAC` | 基于角色的访问控制（兼容旧接口） |

---

## 4. 前端架构详解

### 4.1 目录结构

```
frontend/lib/
├── config/              # 配置与路由
│   ├── api_config.dart  # API 地址配置
│   ├── release_config.dart
│   └── router.dart      # go_router 路由定义
├── models/              # 数据模型
│   └── models.dart
├── pages/               # 页面（按角色/功能分组）
│   ├── home/
│   ├── chat/
│   ├── student/         # 学生功能页
│   ├── counselor/       # 辅导员功能页
│   ├── teacher/         # 教师功能页
│   ├── admin/           # 管理后台页
│   ├── profile/
│   └── ...
├── providers/           # 状态管理（Provider）
│   ├── auth_provider.dart
│   ├── chat_provider.dart
│   ├── session_provider.dart
│   └── ...（共 22 个 Provider）
├── services/            # API 服务
│   ├── api_service.dart
│   └── voice/           # 语音服务
├── utils/               # 工具函数
├── widgets/             # 通用组件
│   ├── answer_card.dart # 回答卡片（含来源引用）
│   ├── fab_menu.dart    # 浮动操作按钮菜单
│   └── ...
└── main.dart            # 应用入口
```

### 4.2 状态管理

使用 **Provider** 进行状态管理，共注册 22 个 ChangeNotifier：

| Provider | 职责 |
|----------|------|
| `AuthProvider` | 登录状态、用户信息 |
| `ChatProvider` | 对话状态、消息列表 |
| `SessionProvider` | 会话列表管理 |
| `EnrollmentProvider` | 入学/办事流程 |
| `KnowledgeProvider` | 知识库浏览 |
| `EmotionProvider` | 情感数据 |
| `AgentProvider` | 智能体管理 |
| `AdminProvider` | 管理后台数据 |
| `FeedbackProvider` | 反馈功能 |
| `TokenStatsProvider` | 词元统计 |
| `StudentFeatureProvider` | 学生 AI 功能 |
| `CounselorFeatureProvider` | 辅导员 AI 功能 |
| `TeacherFeatureProvider` | 教师 AI 功能 |
| `ThemeNotifier` | 主题切换（亮/暗/跟随系统） |

### 4.3 路由系统

使用 **go_router** 实现声明式路由，支持：

- **路由守卫**：首次启动强制隐私同意、未登录重定向
- **响应式布局**：桌面端 NavigationRail + 移动端底部导航
- **ShellRoute**：主框架外壳（磨砂玻璃导航效果）
- **深链接支持**：`/chat?ask=xxx` 直接发起提问

**主导航项（5 个）：** 首页 / 对话 / 知识 / 办事 / 我的

**路由定义位置：** [router.dart](frontend/lib/config/router.dart)

### 4.4 核心组件

#### AnswerCard — 回答卡片

[answer_card.dart](frontend/lib/widgets/answer_card.dart)

蔚小芯标志性组件，展示 AI 回答的同时附带：
- 来源引用列表（可追溯）
- 置信度指示
- 追问建议
- 导出/反馈功能

#### FabMenu — 浮动操作按钮菜单

快捷入口集合，根据角色动态显示可用功能。

### 4.5 平台适配

- **Web**：Cloudflare Pages 部署，支持下载重定向
- **Android**：APK 打包，支持原生语音录制
- **iOS**：支持原生功能
- **Windows / macOS / Linux**：桌面端支持
- **微信小程序**：WebView 壳（`frontend/miniprogram/`）

---

## 5. Context Engine 上下文引擎

### 5.1 设计理念

Context Engine 是蔚小芯的核心竞争力，遵循 **"结构化优先、FTS/BM25 为主、向量可插拔"** 的设计原则：

1. **结构化优先** — 流程、政策等结构化数据优先精确匹配
2. **FTS/BM25 为主** — 全文检索作为主力检索方式，性能稳定、可解释
3. **向量可插拔** — 向量检索作为可选增强，不强制依赖
4. **Sources 可追溯** — 所有回答必须附带来源引用，支持审计

### 5.2 检索策略

```
用户问题
   │
   ├─► 结构化查询（Process / Policy）
   │     精确匹配、步骤化输出
   │
   ├─► FTS5/BM25 全文检索
   │     多字段加权：title > summary > content
   │     权限过滤：owner_scope + role_scope
   │
   ├─► FAQ 持久化缓存
   │     BM25 得分阈值 + Jaccard 相似度校验
   │
   └─► 多智能体协同结果补充
         并行执行 → 结果汇聚 → 去重合并
```

### 5.3 AnswerCard 结构

```go
type AnswerCard struct {
    Conclusion  string   // 回答结论
    TraceID     string   // 链路追踪 ID
    Confidence  float64  // 置信度（0-1）
    Fallback    bool     // 是否兜底回答
    Sources     []Source // 来源引用列表
    FollowUps   []string // 追问建议
}

type Source struct {
    ResourceID      string  // 资源 ID
    Title           string  // 标题
    Version         string  // 版本
    SourceLink      string  // 原文链接
    RelevanceScore  float64 // 相关性分数
    EffectiveAt     string  // 生效时间
    Snippet         string  // 摘要片段
}
```

---

## 6. 多智能体编排系统

### 6.1 架构概览

[orchestrator.go](server/internal/agent/orchestrator.go)

多智能体编排器采用 **意图路由 → 并行执行 → 结果汇聚** 三段式架构：

```
用户问题
   │
   ▼
┌─────────┐   意图分类    ┌──────────────┐
│ Router  │ ──────────►  │ 子 Agent 池  │
└─────────┘              └──────┬───────┘
                                │ 并行执行
                                ▼
                         ┌──────────────┐
                         │   Merger     │  结果汇聚
                         └──────┬───────┘
                                │
                                ▼
                         统一 AnswerCard
```

### 6.2 意图路由器

[router.go](server/internal/agent/router.go)

基于关键词规则的意图分类器，支持多意图同时命中：

| 意图 | 触发关键词示例 | 对应 Agent |
|------|-------------|-----------|
| `IntentPolicy` | 政策、规定、办法、奖学金、处分 | policy-expert |
| `IntentProcess` | 流程、步骤、申请、入学、毕业 | process-guide |
| `IntentActivity` | 活动、比赛、讲座、社团 | qa-default |
| `IntentEmotion` | 焦虑、抑郁、心理、压力 | emotion-counselor |
| `IntentFAQ` | 是什么、在哪里、时间、电话 | qa-default |

### 6.3 子 Agent 列表

| Agent 名称 | 职责 |
|-----------|------|
| `qa-default` (QAAgent) | 默认问答 Agent，通用知识检索 |
| `policy-expert` (PolicyAgent) | 政策专家，聚焦政策条款解读 |
| `process-guide` (ProcessAgent) | 流程向导，聚焦办事步骤指引 |
| `emotion-counselor` (EmotionAgent) | 情感顾问，心理支持与风险识别 |

### 6.4 结果汇聚

[merger.go](server/internal/agent/merger.go)

ResultMerger 负责将多个 Agent 的执行结果合并：
- 去重合并来源引用
- 综合置信度评分
- 拼接内容摘要
- 按相关性排序

---

## 7. 数据库设计

### 7.1 核心表结构

[001_init.sql](server/migrations/001_init.sql)

#### users — 用户表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER | 主键 |
| username | TEXT | 用户名（唯一） |
| display_name | TEXT | 显示名称 |
| role | TEXT | 角色（8 种） |
| owner_scope | TEXT | 归属范围（school/college/class） |
| owner_id | TEXT | 归属 ID |
| password_hash | TEXT | 密码哈希 |
| status | TEXT | 用户状态 |

#### sessions — 会话表

| 字段 | 类型 | 说明 |
|------|------|------|
| session_id | TEXT | 会话 UUID |
| user_id | INTEGER | 用户 ID |
| title | TEXT | 会话标题 |

#### messages — 消息表

| 字段 | 类型 | 说明 |
|------|------|------|
| session_id | TEXT | 会话 ID |
| role | TEXT | 角色（user/assistant/system） |
| content | TEXT | 消息内容 |
| trace_id | TEXT | 链路追踪 ID |

#### kb_resources — 知识资源表

| 字段 | 类型 | 说明 |
|------|------|------|
| resource_id | TEXT | 资源唯一 ID |
| resource_type | TEXT | 类型（Policy/Process/FAQ/Activity） |
| owner_scope | TEXT | 归属范围 |
| owner_id | TEXT | 归属 ID |
| role_scope | TEXT | 角色范围（JSON 数组） |
| version | TEXT | 版本号 |
| status | TEXT | 状态（draft/pending/published/retired） |
| title | TEXT | 标题 |
| summary | TEXT | 摘要 |
| content | TEXT | 内容 |
| source_link | TEXT | 原文链接 |
| effective_at | TEXT | 生效时间 |
| expired_at | TEXT | 失效时间 |
| tags | TEXT | 标签（JSON 数组） |

#### kb_fts — FTS5 全文索引

虚拟表，与 kb_resources 通过触发器自动同步。
- tokenize: unicode61（支持中文分词基础）
- 排序: BM25 算法

#### process_steps — 流程节点表

结构化流程数据，支持精确步骤查询：
- step_order: 步骤序号
- materials: 所需材料（JSON 数组）
- entry_url: 办理入口
- deadline: 截止时间
- location: 办理地点

#### audit_logs — 审计日志表

记录所有操作的审计轨迹，包含 trace_id、IP、耗时、结果码。

#### emotion_logs — 情感评估表

记录用户情感分析结果，含风险等级（low/medium/high）。

### 7.2 迁移系统

- 迁移文件位于 `server/migrations/`，按 `NNN_*.sql` 编号
- 使用 `_migrations` 表记录已执行迁移
- 支持幂等执行（重复列 ALTER 自动跳过）
- 触发器等复合语句正确分割

---

## 8. 权限模型（RBAC + Capability）

### 8.1 设计思路

[capabilities.go](server/internal/auth/capabilities.go)

采用 **"角色继承 + 能力授权"** 的混合模型：

- **Capability（能力）** 是最小授权单元，命名格式：`domain.action`
- **角色** 绑定一组 capability，并通过父角色实现继承
- **DFS 继承算法**：递归向上查询角色继承链
- 与 `owner_scope/owner_id` 数据范围正交

### 8.2 角色继承图

```
sys_admin → school_admin → college_admin
                                      │
              ┌───────────────────┬───┴───┬───────────────────┐
              ▼                   ▼       ▼                   ▼
          counselor           teacher  assistant         student_union
              │                   │       │                   │
              └───────────────────┴───┬───┴───────────────────┘
                                      ▼
                                student_union
                                      │
                                      ▼
                                  student
```

**关键点：**
- `college_admin` 同时继承 `counselor` + `teacher` + `assistant`（多父继承）
- `counselor` / `teacher` / `assistant` 三者平级，互不继承
- `student` 是个人能力基线，所有登录角色均继承学生能力

### 8.3 Capability 分类

| 能力域 | 示例 | 说明 |
|--------|------|------|
| `self.*` | self.chat, self.knowledge.read | 个人能力（所有登录用户） |
| `counselor.*` | counselor.alert.read | 辅导员专属能力 |
| `teacher.*` | teacher.lesson.prep | 教师专属能力 |
| `college.*` | college.metrics.read | 学院管理员能力 |
| `school.*` | school.agent.write | 学校管理员能力 |
| `system.*` | system.settings.write | 系统管理员能力 |
| `union.*` | union.event.plan | 学生会能力 |

### 8.4 使用方式

```go
// 单能力检查
secured.POST("/chat", auth.RequireCapability(auth.SelfChat), chatH.Ask)

// 多能力（满足任一即可）
secured.POST("/kb/upload", 
    auth.RequireAnyCapability(auth.UnionKBSubmit, auth.CounselorKBWrite), 
    uploadH.Upload)
```

---

## 9. LLM 集成与客户端抽象

### 9.1 统一接口

[client.go](server/internal/llm/client.go)

所有 LLM 提供商实现统一的 `ChatClient` 接口：

```go
type ChatClient interface {
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    Name() string
}
```

### 9.2 支持的模型

| 模型提供商 | 实现文件 | 默认模型 | 用途 |
|-----------|---------|---------|------|
| 智谱清言 | [zhipu.go](server/internal/llm/zhipu.go) | glm-4 | 主力对话模型 |
| DeepSeek | [deepseek.go](server/internal/llm/deepseek.go) | deepseek-chat | 主力对话模型 |
| 智谱 GLM-4V | zhipu_4v | glm-4v | 多模态（预留） |
| 讯飞星火 | [xfyun.go](server/internal/llm/xfyun.go) | — | 语音 ASR/TTS |

### 9.3 故障转移机制

[fallback.go](server/internal/llm/fallback.go)

使用 `ChainClient` 实现链式故障转移：
- 配置多个 LLM 客户端
- 主客户端失败时自动切换到下一个
- 保证服务可用性

### 9.4 Mock 客户端

[mock_client.go](server/internal/llm/mock_client.go)

支持测试的 Mock 实现，可预设响应、模拟延迟、模拟错误。

---

## 10. 关键流程时序

### 10.1 问答主链路

```
用户 → 前端 → ChatHandler → ChatService.Ask()
                                 │
                                 ├─► 缓存检查（内存/FAQ）─命中─► 返回
                                 │
                                 ├─► 会话管理（创建/获取）
                                 │
                                 ├─► 多智能体编排（可选）
                                 │     ├─► Router 意图分类
                                 │     ├─► 并行执行子 Agent
                                 │     └─► Merger 结果汇聚
                                 │
                                 ├─► KBRepo.Search() — FTS5/BM25 检索
                                 │
                                 ├─► buildMessages() — 拼装上下文
                                 │     ├─► 系统提示词
                                 │     ├─► 多智能体结果
                                 │     ├─► 知识库引用
                                 │     ├─► 历史对话（最近 6 条）
                                 │     └─► 当前问题
                                 │
                                 ├─► LLM 调用
                                 │     └─► 失败 → 兜底回答（保留 sources）
                                 │
                                 ├─► PII 脱敏 + 内容安全检查
                                 │
                                 └─► 构造 AnswerCard
                                       ├─► 结论
                                       ├─► 来源引用（去重）
                                       ├─► 置信度
                                       └─► 追问建议
```

### 10.2 认证流程

```
用户登录 → AuthHandler.Login()
              │
              ├─► 参数校验
              ├─► AuthService.Login()
              │     ├─► 用户查询
              │     ├─► 密码验证（bcrypt）
              │     └─► 生成 JWT Token
              └─► 返回 Token + 用户信息 + Capability 列表
```

### 10.3 知识审核流程

```
学生会提交 → kb/resources/:id/submit → 状态 pending
辅导员审核 → kb/resources/:id/approve → 状态 published
                 reject  → 状态 retired（附原因）
```

---

## 11. 依赖关系

### 11.1 后端 Go 依赖

[go.mod](go.mod)

| 依赖 | 版本 | 用途 |
|------|------|------|
| gin-gonic/gin | v1.10.0 | Web 框架 |
| golang-jwt/jwt/v5 | v5.3.1 | JWT 认证 |
| google/uuid | v1.6.0 | UUID 生成 |
| joho/godotenv | v1.5.1 | .env 配置加载 |
| modernc.org/sqlite | v1.29.9 | 纯 Go SQLite 驱动（含 FTS5） |
| go.temporal.io/sdk | v1.43.0 | Temporal 工作流（可选） |
| stretchr/testify | v1.11.1 | 测试框架 |
| xuri/excelize/v2 | v2.11.0 | Excel 导入导出 |
| gorilla/websocket | v1.5.3 | WebSocket |
| golang.org/x/crypto | v0.53.0 | 密码哈希 |

### 11.2 前端 Flutter 依赖

[pubspec.yaml](frontend/pubspec.yaml)

| 依赖 | 版本 | 用途 |
|------|------|------|
| provider | ^6.1.0 | 状态管理 |
| go_router | ^14.1.0 | 声明式路由 |
| dio | ^5.7.0 | HTTP 请求 |
| shared_preferences | ^2.3.0 | 本地存储 |
| flutter_markdown_plus | ^1.0.0 | Markdown 渲染 |
| fl_chart | ^0.69.0 | 图表可视化 |
| audioplayers | ^6.1.0 | 音频播放 |
| record | ^5.2.0 | 语音录制 |
| url_launcher | ^6.3.0 | 外部链接 |
| file_picker | ^8.0.7 | 文件选择 |
| path_provider | ^2.1.0 | 文件路径 |

---

## 12. 配置说明

### 12.1 环境变量

[config.go](server/internal/config/config.go)

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `APP_PORT` | 8080 | 服务监听端口 |
| `APP_MODE` | debug | 运行模式（debug/release） |
| `JWT_SECRET` | — | JWT 签名密钥（必填） |
| `JWT_EXPIRE_HOURS` | 2 | Token 过期时间（小时） |
| `SQLITE_PATH` | ./data/wxx.db | SQLite 数据库路径 |
| `ZHIPU_API_KEY` | — | 智谱清言 API Key |
| `DEEPSEEK_API_KEY` | — | DeepSeek API Key |
| `XFYUN_APP_ID` | — | 讯飞应用 ID |
| `XFYUN_API_KEY` | — | 讯飞 API Key |
| `XFYUN_API_SECRET` | — | 讯飞 API Secret |
| `XUEGONG_BASE_URL` | — | 学工系统地址 |
| `YBT_BASE_URL` | — | 一表通地址 |
| `QQ_WEBHOOK_URL` | — | QQ 群机器人 Webhook |
| `WECHAT_WEBHOOK_URL` | — | 企业微信机器人 Webhook |
| `TEMPORAL_HOST_PORT` | — | Temporal 服务地址（空=禁用） |

### 12.2 配置加载顺序

1. 尝试当前目录 `.env`
2. 尝试父目录 `.env`
3. 系统环境变量覆盖

---

## 13. 构建与运行

### 13.1 后端

**开发运行：**
```bash
cd server
go run cmd/server/main.go
```

**数据库迁移：**
```bash
go run cmd/migrate/main.go
```

**生产构建：**
```bash
cd server
go build -o wxx-server cmd/server/main.go
```

**Vercel 部署：**
- 入口文件: `api/index.go`
- 数据库: `/tmp/wxx.db`（临时文件系统）
- Temporal 自动禁用

### 13.2 前端

**开发运行：**
```bash
cd frontend
flutter run -d chrome    # Web
flutter run -d windows   # Windows 桌面
```

**Web 构建：**
```bash
flutter build web --release
```

**APK 构建：**
```bash
flutter build apk --release
```

**全量构建脚本：**
```bash
pwsh scripts/build-all.ps1
# 或
make all-frontend
```

### 13.3 测试

**后端测试：**
```bash
cd server
go test ./... -v
```

**前端测试：**
```bash
cd frontend
flutter test
```

---

## 14. 关键文件索引

### 14.1 后端核心文件

| 文件 | 说明 |
|------|------|
| [pkg/app/app.go](server/pkg/app/app.go) | 应用组装与依赖注入 |
| [cmd/server/main.go](server/cmd/server/main.go) | 主服务入口 |
| [internal/service/chat_service.go](server/internal/service/chat_service.go) | 问答主链路（Context Engine 核心） |
| [internal/agent/orchestrator.go](server/internal/agent/orchestrator.go) | 多智能体编排器 |
| [internal/agent/router.go](server/internal/agent/router.go) | 意图路由器 |
| [internal/auth/capabilities.go](server/internal/auth/capabilities.go) | 权限能力模型 |
| [internal/llm/client.go](server/internal/llm/client.go) | LLM 客户端接口 |
| [internal/repository/kb_repo.go](server/internal/repository/kb_repo.go) | 知识库 FTS5 检索 |
| [internal/middleware/rbac.go](server/internal/middleware/rbac.go) | RBAC 中间件 |
| [internal/config/config.go](server/internal/config/config.go) | 配置加载 |
| [migrations/001_init.sql](server/migrations/001_init.sql) | 数据库初始 Schema |

### 14.2 前端核心文件

| 文件 | 说明 |
|------|------|
| [lib/main.dart](frontend/lib/main.dart) | 应用入口与 Provider 注册 |
| [lib/config/router.dart](frontend/lib/config/router.dart) | 路由配置 |
| [lib/services/api_service.dart](frontend/lib/services/api_service.dart) | API 服务封装 |
| [lib/widgets/answer_card.dart](frontend/lib/widgets/answer_card.dart) | 回答卡片组件 |
| [lib/providers/chat_provider.dart](frontend/lib/providers/chat_provider.dart) | 对话状态管理 |
| [lib/pages/chat/chat_page.dart](frontend/lib/pages/chat/chat_page.dart) | 对话页面 |

### 14.3 项目文档

| 文档 | 说明 |
|------|------|
| [AGENTS.md](AGENTS.md) | 协作索引（必读） |
| [docs/蔚小芯开发规范.md](docs/蔚小芯开发规范.md) | 主开发规范 |
| [docs/蔚小芯智能体.md](docs/蔚小芯智能体.md) | 产品与技术总纲 |
| [docs/context-engine.md](docs/context-engine.md) | Context Engine 详解 |
| [docs/deployment.md](docs/deployment.md) | 部署指南 |
| [specs/export-package.md](specs/export-package.md) | 导出契约 |

---

## 15. 设计原则

### 15.1 架构原则

1. **依赖倒置** — Service 依赖 Port 接口，不依赖具体 Repository 实现
2. **可观测性** — 每个请求带 TraceID，全链路可追踪
3. **优雅降级** — LLM 失败返回兜底回答，保留检索到的 sources
4. **安全优先** — PII 脱敏、内容安全过滤、审计日志
5. **幂等设计** — 数据库迁移、FAQ 缓存键均为幂等

### 15.2 开发原则

1. **Plan → 人审 → 编码** — 每增量文档 + Git 提交
2. **测试驱动** — 核心逻辑必须有单元测试
3. **小步提交** — 原子化变更，便于 review 和回滚
4. **文档同步** — 代码变更同步更新相关文档

### 15.3 产品原则

1. **Sources 可追溯** — 所有 AI 回答必须附带来源引用
2. **结构化优先** — 能结构化解决的不依赖 LLM 生成
3. **角色驱动** — 功能按角色分层，高角色自动继承低角色能力
4. **隐私保护** — 用户数据最小化使用，PII 全程脱敏

---

*文档版本：v1.0 | 最后更新：2026-07-25*
