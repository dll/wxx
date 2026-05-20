import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'config/router.dart';
import 'providers/auth_provider.dart';
import 'providers/chat_provider.dart';
import 'providers/session_provider.dart';
import 'providers/enrollment_provider.dart';
import 'providers/knowledge_provider.dart';
import 'providers/emotion_provider.dart';
import 'providers/agent_provider.dart';
import 'providers/home_provider.dart';
import 'providers/admin_provider.dart';
import 'providers/feedback_provider.dart';
import 'providers/bookmark_provider.dart';
import 'providers/student_feature_provider.dart';
import 'providers/counselor_feature_provider.dart';
import 'providers/teacher_feature_provider.dart';
import 'providers/model_config_provider.dart';
import 'providers/culture_provider.dart';
import 'providers/token_stats_provider.dart';
import 'utils/storage.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // 初始化本地存储
  await Storage.init();

  runApp(const WxxApp());
}

/// 主题模式通知器 — 供全局切换亮色/暗色/跟随系统
class ThemeNotifier extends ChangeNotifier {
  ThemeMode _mode;

  ThemeNotifier() : _mode = _fromString(Storage.themeMode);

  ThemeMode get mode => _mode;

  void setMode(ThemeMode mode) {
    if (_mode == mode) return;
    _mode = mode;
    Storage.setThemeMode(mode.name);
    notifyListeners();
  }

  static ThemeMode _fromString(String s) {
    switch (s) {
      case 'light':
        return ThemeMode.light;
      case 'dark':
        return ThemeMode.dark;
      default:
        return ThemeMode.system;
    }
  }
}

/// 蔚小芯应用入口
class WxxApp extends StatelessWidget {
  const WxxApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider(create: (_) => AuthProvider()),
        ChangeNotifierProvider(create: (_) => ChatProvider()),
        ChangeNotifierProvider(create: (_) => SessionProvider()),
        ChangeNotifierProvider(create: (_) => EnrollmentProvider()),
        ChangeNotifierProvider(create: (_) => KnowledgeProvider()),
        ChangeNotifierProvider(create: (_) => EmotionProvider()),
        ChangeNotifierProvider(create: (_) => AgentProvider()),
        ChangeNotifierProvider(create: (_) => HomeProvider()),
        ChangeNotifierProvider(create: (_) => AdminProvider()),
        ChangeNotifierProvider(create: (_) => FeedbackProvider()),
        ChangeNotifierProvider(create: (_) => BookmarkProvider()),
        ChangeNotifierProvider(create: (_) => StudentFeatureProvider()),
        ChangeNotifierProvider(create: (_) => CounselorFeatureProvider()),
        ChangeNotifierProvider(create: (_) => TeacherFeatureProvider()),
        ChangeNotifierProvider(create: (_) => ModelConfigProvider()),
        ChangeNotifierProvider(create: (_) => CultureProvider()),
        ChangeNotifierProvider(create: (_) => TokenStatsProvider()),
        ChangeNotifierProvider(create: (_) => ThemeNotifier()),
      ],
      child: Consumer<ThemeNotifier>(
        builder: (_, themeNotifier, __) {
          return MaterialApp.router(
            title: '蔚小芯',
            debugShowCheckedModeBanner: false,
            themeMode: themeNotifier.mode,
            theme: ThemeData(
              colorSchemeSeed: const Color(0xFF1565C0), // 滁州学院蓝
              useMaterial3: true,
              brightness: Brightness.light,
              appBarTheme: const AppBarTheme(
                centerTitle: true,
                elevation: 0,
              ),
              pageTransitionsTheme: const PageTransitionsTheme(
                builders: {
                  TargetPlatform.android: FadeUpwardsPageTransitionsBuilder(),
                  TargetPlatform.iOS: CupertinoPageTransitionsBuilder(),
                  TargetPlatform.windows: FadeUpwardsPageTransitionsBuilder(),
                  TargetPlatform.macOS: CupertinoPageTransitionsBuilder(),
                  TargetPlatform.linux: FadeUpwardsPageTransitionsBuilder(),
                },
              ),
            ),
            darkTheme: ThemeData(
              colorSchemeSeed: const Color(0xFF1565C0),
              useMaterial3: true,
              brightness: Brightness.dark,
              appBarTheme: const AppBarTheme(
                centerTitle: true,
                elevation: 0,
              ),
              pageTransitionsTheme: const PageTransitionsTheme(
                builders: {
                  TargetPlatform.android: FadeUpwardsPageTransitionsBuilder(),
                  TargetPlatform.iOS: CupertinoPageTransitionsBuilder(),
                  TargetPlatform.windows: FadeUpwardsPageTransitionsBuilder(),
                  TargetPlatform.macOS: CupertinoPageTransitionsBuilder(),
                  TargetPlatform.linux: FadeUpwardsPageTransitionsBuilder(),
                },
              ),
            ),
            routerConfig: appRouter,
          );
        },
      ),
    );
  }
}
