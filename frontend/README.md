# frontend/ — 蔚小芯 Flutter 客户端

## 初始化步骤

本目录为 Flutter 项目占位。初始化前请确保环境就绪：

```bash
# 1. 确认 Flutter SDK 版本（建议 3.22+）
flutter --version

# 2. 在本目录创建 Flutter 项目
flutter create --org com.wxx --project-name wxx_app .

# 3. 安装依赖
flutter pub get
```

## 技术选型

| 功能 | 库 | 说明 |
|------|-----|------|
| HTTP 请求 | `dio` | 支持拦截器、JWT 注入 |
| 状态管理 | `provider` | 轻量，团队易上手 |
| 本地存储 | `hive` | 缓存会话/偏好 |
| 路由 | `go_router` | 声明式路由 |
| Markdown 渲染 | `flutter_markdown` | AnswerCard 内容渲染 |

## 目标平台

- **P0**: Web + Android APK
- **P1**: 微信小程序（需 uni-app 或独立轻量端）

## 目录结构（初始化后建议）

```
lib/
├── main.dart
├── app.dart               # MaterialApp 配置
├── config/                # 环境配置、API 地址
├── models/                # 数据模型
├── services/              # API 调用层（Dio 封装）
├── providers/             # Provider 状态管理
├── pages/                 # 页面
│   ├── chat/              # 对话主页
│   ├── login/             # 登录
│   ├── knowledge/         # 知识浏览（管理端）
│   └── profile/           # 个人中心
├── widgets/               # 复用组件
│   ├── answer_card.dart   # AnswerCard 统一回答卡片
│   └── message_bubble.dart
└── utils/                 # 工具函数
```

## 构建命令

```bash
# 开发运行（Web）
flutter run -d chrome

# 开发运行（Android 模拟器）
flutter run

# 构建 Web
flutter build web --release

# 构建 APK
flutter build apk --release

# 测试
flutter test
```
