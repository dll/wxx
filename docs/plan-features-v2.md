# 任务方案 — 第二批功能（游客/入学优化/地图/抖音/首页）

## 背景与目标

5 个新需求：游客账号体系、入学功能完善、校园地图导航、招生抖音接入、学校首页链接。

## 范围

### 做

#### 1. 游客账号（guest）
- 新增 `guest` 角色 + 能力（仅浏览公开知识）
- 用户表新增 `status` 字段（pending/active/rejected/disabled）
- 游客注册：`POST /auth/guest-register`（仅需姓名+手机号，无学号）
- 游客→学生转换流程：管理员/书记审核，`PUT /admin/guests/:id/approve`
- 游客中间件：写操作返回 403

#### 2. 首页 + 入学功能增强
- 办事页（/enrollment）增加：转专业、助学贷款两个流程入口
- 补充这两个流程的 contact/FAQ 种子数据
- 入学流程增加关键日期提示、材料清单分类

#### 3. 校园地图导航 + 全景 + 学校首页
- 新增 `url_launcher` 依赖
- 新增 `/campus/map` 路线：高德地图导航 + 腾讯地图导航
- 新增 VR 全景入口（链接到 https://www.chzu.edu.cn/vr/index.html）
- 学校首页入口（chzu1950）

#### 4. 招生抖音
- 新增招生专区页面
- 抖音链接：`https://www.douyin.com/user/54452972915`

### 不做
- 不改现有登录流程（游客和现有用户并行）
- 不加 SDK（高德/腾讯 SDK），只用 URL Scheme 跳转
- 不做实时地图渲染

## 技术要点

- 前端新增 `url_launcher` 包
- 后端新增 `guest` role + `status` 字段
- 迁移文件：`025_add_user_status.sql`（ALTER users ADD status）
- 迁移文件：`026_seed_additional_processes.sql`（补充 contact 数据）
- `capabilities.go` 新增 `guest` 角色 + `SelfGuestRead` 能力
- 影响 7 个文件：后端 4 + 前端 3

## 步骤拆分

1. 后端：迁移 025（status 字段）+ 迁移 026（流程补充数据）
2. 后端：capabilities.go 新增 guest 角色 + 用户注册/审核 handler
3. 后端：app.go 注册新路由
4. 前端：url_launcher 依赖 + 地图/抖音/全景入口
5. 前端：办事页流程类型扩展
6. 部署验证

## 验收标准

- 游客可注册登录，只能浏览知识库
- 管理员/书记可在后台审核游客升级
- 办事页可查看转专业/助学贷款流程
- 校园导航可跳转高德/腾讯地图 App
- VR 全景链接可打开
- 学校首页入口跳转正确
- 招生抖音跳转正确

## 回滚与检查点

- 每个步骤单独 Git 提交
- 迁移文件可回滚（以新 migration 修正）
