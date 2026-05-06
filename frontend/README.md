# frontend/ — 蔚小芯 Flutter 客户端

## 技术选型

| 功能 | 库 | 说明 |
|------|-----|------|
| HTTP 请求 | `dio` | 支持拦截器、JWT 注入 |
| 状态管理 | `provider` | 轻量，团队易上手 |
| 本地存储 | `shared_preferences` | Token、偏好缓存 |
| 路由 | `go_router` | 声明式路由 + 守卫 |
| Markdown 渲染 | `flutter_markdown_plus` | AnswerCard 内容渲染 |

## 目标平台

- **P0**: Web + Android APK
- **P1**: 微信小程序（需 uni-app 或独立轻量端）

## 目录结构

```
lib/
├── main.dart                  # 应用入口
├── config/
│   ├── api_config.dart        # API 地址、超时配置
│   └── router.dart            # go_router 路由表 + 底部导航
├── models/
│   └── models.dart            # 数据模型（AnswerCard、ChatRequest/Response 等）
├── services/
│   └── api_service.dart       # Dio 封装 + JWT 拦截器
├── providers/
│   ├── auth_provider.dart     # 认证状态（登录、退出、用户信息）
│   ├── chat_provider.dart     # 对话状态（消息列表、发送）
│   └── session_provider.dart  # 会话列表状态
├── pages/
│   ├── login/login_page.dart  # 登录页
│   ├── chat/chat_page.dart    # 对话主页
│   ├── sessions/sessions_page.dart  # 会话历史
│   └── profile/profile_page.dart    # 个人中心
├── widgets/
│   └── answer_card.dart       # AnswerCard 卡片渲染组件
└── utils/
    └── storage.dart           # SharedPreferences 封装
```

## 环境变量

构建前需设置以下环境变量，否则依赖解析可能失败：

| 变量 | 说明 | 示例 |
|------|------|------|
| `PUB_HOSTED_URL` | pub 镜像源（中国用户必设） | `https://pub.flutter-io.cn` |
| `PUB_CACHE` | pub 缓存目录（可选，避免跨盘路径问题） | `D:\PUB` |

```powershell
# PowerShell
$env:PUB_HOSTED_URL = "https://pub.flutter-io.cn"
$env:PUB_CACHE = "D:\PUB"
```

```bash
# bash / Git Bash
export PUB_HOSTED_URL="https://pub.flutter-io.cn"
export PUB_CACHE="D:/PUB"
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

# 静态检查
flutter analyze

# 测试
flutter test
```

## 已知问题

- Flutter `impellerc` 对中文路径支持有 bug，Web 构建时需使用不含中文的路径（可通过 junction/symlink 绕过）
