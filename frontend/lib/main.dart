import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'config/router.dart';
import 'services/api_service.dart';
import 'providers/auth_provider.dart';
import 'providers/chat_provider.dart';
import 'providers/session_provider.dart';
import 'providers/enrollment_provider.dart';
import 'providers/process_record_provider.dart';
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
import 'providers/forecast_provider.dart';
import 'providers/student_new_features_provider.dart';
import 'providers/guest_provider.dart';
import 'providers/education_provider.dart';
import 'providers/study_plan_provider.dart';
import 'providers/notification_provider.dart';
import 'providers/update_provider.dart';
import 'utils/download_redirect.dart';
import 'utils/storage.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  redirectDownloadFallbackIfNeeded();

  FlutterError.onError = (details) {
    FlutterError.presentError(details);
  };

  // release 模式下 build 异常默认显示空白 ErrorWidget，改为友好的错误占位
  ErrorWidget.builder = (FlutterErrorDetails details) {
    return Material(
      color: const Color(0xFFF5F5F5),
      child: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.error_outline, size: 48, color: Colors.grey.shade400),
            const SizedBox(height: 12),
            Text('页面渲染异常',
                style: TextStyle(fontSize: 16, color: Colors.grey.shade600)),
            const SizedBox(height: 8),
            Text('请刷新页面重试',
                style: TextStyle(fontSize: 13, color: Colors.grey.shade500)),
          ],
        ),
      ),
    );
  };

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
        ChangeNotifierProvider(create: (_) {
          final p = ChatProvider();
          sessionResetCallbacks.add(p.reset);
          return p;
        }),
        ChangeNotifierProvider(create: (_) => SessionProvider()),
        ChangeNotifierProvider(create: (_) => EnrollmentProvider()),
        ChangeNotifierProvider(create: (_) => ProcessRecordProvider()),
        ChangeNotifierProvider(create: (_) => KnowledgeProvider()),
        ChangeNotifierProvider(create: (_) {
          final p = EmotionProvider();
          sessionResetCallbacks.add(p.reset);
          return p;
        }),
        ChangeNotifierProvider(create: (_) => AgentProvider()),
        ChangeNotifierProvider(create: (_) => HomeProvider()),
        ChangeNotifierProvider(create: (_) {
          final p = AdminProvider();
          sessionResetCallbacks.add(p.reset);
          return p;
        }),
        ChangeNotifierProvider(create: (_) => FeedbackProvider()),
        ChangeNotifierProvider(create: (_) {
          final p = BookmarkProvider();
          sessionResetCallbacks.add(p.reset);
          return p;
        }),
        ChangeNotifierProvider(create: (_) => StudentFeatureProvider()),
        ChangeNotifierProvider(create: (_) => CounselorFeatureProvider()),
        ChangeNotifierProvider(create: (_) => TeacherFeatureProvider()),
        ChangeNotifierProvider(create: (_) => ModelConfigProvider()),
        ChangeNotifierProvider(create: (_) => CultureProvider()),
        ChangeNotifierProvider(create: (_) => TokenStatsProvider()),
        ChangeNotifierProvider(create: (_) => ForecastProvider()),
        ChangeNotifierProvider(create: (_) => StudentNewFeaturesProvider()),
        ChangeNotifierProvider(create: (_) => GuestProvider()),
        ChangeNotifierProvider(create: (_) => CareerProvider()),
        ChangeNotifierProvider(create: (_) => StudyProvider()),
        ChangeNotifierProvider(create: (_) => MentalProvider()),
        ChangeNotifierProvider(create: (_) => StudyPlanProvider()),
        ChangeNotifierProvider(create: (_) => NotificationProvider()),
        ChangeNotifierProvider(create: (_) => UpdateProvider()),
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
              // 使用本地打包的 Roboto（见 pubspec fonts 段），避免 Web 引擎
              // 运行时从 fonts.gstatic.com 拉取字体导致空白/延迟
              fontFamily: 'Roboto',
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
              fontFamily: 'Roboto',
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
