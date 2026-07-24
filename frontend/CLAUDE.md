# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 构建与开发命令

```bash
# 安装依赖
flutter pub get

# 静态分析（零 error 为通过标准，info 级别可忽略）
flutter analyze --no-pub

# 开发运行（Web）
flutter run -d chrome

# ── 一键构建 Web + APK（推荐）────────────────────────────
# 同时构建 Web 和 APK 两个版本，产物：
#   Web:  build/web/index.html
#   APK:  build/app/outputs/flutter-apk/weixiaoxin-release.apk
# 注意：确保项目路径不含中文，否则 Web 构建会因 impellerc bug 失败
make all-frontend

# ── 单独构建 ──────────────────────────────────────────────

# 构建 Web（中文路径需用临时目录绕过 impellerc bug）
flutter build web --release

# 构建 APK（推荐直接使用 gradlew 以避免 flutter build 管道丢失 Gradle 输出）
cd android && ./gradlew assembleRelease

# 构建 APK（通过 Flutter CLI，产物自动重命名为 蔚小芯.apk，详见 docs/deployment.md）
make flutter-build-apk        # ASCII 路径
make flutter-build-apk-safe   # 路径含中文时使用

# 部署 Web 到 Cloudflare Pages（项目 wxx-agent，域名 wxx-agent.pages.dev）
# 详见 docs/蔚小芯前端重新部署.md

# 测试
flutter test

# 单个测试文件
flutter test test/path_to_test.dart
```

**已知问题**：Flutter `impellerc` 对中文路径有 bug，Web 构建时需使用不含中文的输出路径（junction/symlink 绕过）。

## 应用命名规范（强制）

- 用户可见名称统一为「蔚小芯」：`AndroidManifest.xml` 的 `android:label`、`web/index.html` 的 `<title>` 和 `apple-mobile-web-app-title`、`web/manifest.json` 的 `name` / `short_name`
- 技术 ID 仍用 `wxx_app` / `com.wxx.wxx_app`
- APK 分发文件名固定为 `蔚小芯.apk`（Makefile 自动从 `apk/release/蔚小芯-release.apk` 复制到 `flutter-apk/蔚小芯.apk`）
- 前端正式入口是 Cloudflare Pages `https://wxx-agent.pages.dev`，不要再使用已停用的 Vercel 前端旧域名


## 架构概览

Provider + GoRouter 单向数据流架构，Material Design 3 主题。

```
用户操作 → Page 调用 Provider 方法 → Provider 通过 ApiService 请求后端
         ← Provider.notifyListeners() ← Consumer/watch 重建 UI
```

### 关键分层

| 层 | 职责 | 约束 |
|---|---|---|
| `pages/` | UI 页面，通过 `context.read/watch<Provider>()` 获取状态 | 不直接调用 ApiService |
| `providers/` | 业务状态 + 数据获取，继承 `ChangeNotifier` | 通过 ApiService 单例发请求 |
| `services/api_service.dart` | Dio 单例，JWT 拦截器，401 自动跳转 | 全局唯一实例 |
| `config/api_config.dart` | 所有 API 路径常量 | 后端地址 `https://api.pydaydayup.xyz` |
| `config/router.dart` | GoRouter 路由表 + ShellRoute 底部导航 + 鉴权 redirect | 5 个主 tab：首页/对话/知识/办事/我的 |
| `models/models.dart` | 所有数据模型集中定义，`fromJson` 工厂构造 | 单文件，按功能分区 |
| `utils/` | 工具类（Storage、RoleUtils） | 无状态，纯函数/静态方法 |
| `widgets/` | 可复用组件（AnswerCard、ErrorView、ConsentDialog） | 无业务逻辑 |

### 导航结构

`ShellRoute` 包裹所有认证后页面，提供响应式布局：
- 移动端：磨砂玻璃底部 NavigationBar
- 桌面端（>900px）：左侧 NavigationRail

路由守卫在 `appRouter.redirect` 中统一处理：未登录 → `/login`，已登录访问 `/login` → `/home`。

### 语音服务（平台条件编译）

```
voice_service.dart       → 导出入口（条件 import）
voice_service_web.dart   → Web：MediaRecorder 录音 + SpeechSynthesis 朗读
voice_service_mobile.dart → Android/iOS：record 包录音 + audioplayers 播放
voice_service_stub.dart  → 兜底空实现
```

### 角色体系

六级 RBAC，权限判断统一使用 `utils/role_utils.dart`：
- `sys_admin` > `school_admin` > `college_admin` > `counselor` > `student_union` > `student`

## 编码约定

- **全中文注释**，变量/函数名用英文
- Provider 命名：`XxxProvider`，对应文件 `xxx_provider.dart`
- 页面结构：`pages/<feature>/<feature>_page.dart`
- 管理端页面统一放 `pages/admin/`
- 菜单卡片使用 `_buildMenuCard` 辅助方法（profile_page 模式）
- `dart:html` 仅在 Web 专用功能中使用（TTS、PDF 导出），会产生 info 级别 lint 警告，可忽略
- 主题切换通过 `ThemeNotifier`（定义在 main.dart），支持亮色/暗色/跟随系统

## 状态管理模式

所有 Provider 在 `main.dart` 的 `MultiProvider` 中注册。典型 Provider 结构：

```dart
class XxxProvider extends ChangeNotifier {
  final ApiService _api = ApiService();
  bool _loading = false;
  String? _error;
  
  Future<void> fetchData() async {
    _loading = true; notifyListeners();
    try { /* ... */ }
    catch (e) { _error = e.toString(); }
    finally { _loading = false; notifyListeners(); }
  }
}
```

## 部署

Flutter Web 构建产物部署到 Cloudflare Pages **`wxx-agent`** 项目（域名 `https://wxx-agent.pages.dev`）。

```bash
make deploy-web   # 自动构建 + 同步 Pages Functions + wrangler pages deploy
```

**不要再使用 Vercel 前端旧域名**：原 `wxx.pydaydayup.xyz` 已停用。前端发布和验收均以 `https://wxx-agent.pages.dev` 为准。
