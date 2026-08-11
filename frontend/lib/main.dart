import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'config/router.dart';
import 'services/api_service.dart';
import 'providers/auth_provider.dart';
import 'providers/chat_provider.dart';
import 'providers/session_provider.dart';
import 'providers/enrollment_provider.dart';
import 'providers/process_record_provider.dart';
import 'providers/process_provider.dart';
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
import 'providers/health_provider.dart';
import 'providers/app_center_provider.dart';
import 'providers/ai_briefing_provider.dart';
import 'providers/twin_portrait_provider.dart';
import 'providers/personal_detail_provider.dart';
import 'utils/download_redirect.dart';
import 'utils/storage.dart';
import 'services/voice/deep_link.dart';

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

/// 主题模式通知器 — 支持亮色/暗色/跟随系统 + 年级主题自动切换
class ThemeNotifier extends ChangeNotifier {
  ThemeMode _mode;

  /// 入学年份（如 2025），用于推导年级主题；null = 未登录/未知
  int? _enrollmentYear;

  /// 年级主题自动切换开关（默认开启）
  bool _gradeThemeEnabled;

  ThemeNotifier()
      : _mode = _fromString(Storage.themeMode),
        _gradeThemeEnabled = Storage.gradeThemeEnabled;

  ThemeMode get mode => _mode;

  int? get enrollmentYear => _enrollmentYear;

  bool get gradeThemeEnabled => _gradeThemeEnabled;

  void setMode(ThemeMode mode) {
    if (_mode == mode) return;
    _mode = mode;
    Storage.setThemeMode(mode.name);
    notifyListeners();
  }

  /// 设置入学年份（登录/刷新资料时调用），触发年级主题重算
  void setEnrollmentYear(int? year) {
    if (_enrollmentYear == year) return;
    _enrollmentYear = year;
    if (year != null) Storage.setEnrollmentYear(year);
    notifyListeners();
  }

  /// 关闭/开启年级主题自动切换
  void setGradeThemeEnabled(bool enabled) {
    if (_gradeThemeEnabled == enabled) return;
    _gradeThemeEnabled = enabled;
    Storage.setGradeThemeEnabled(enabled);
    notifyListeners();
  }

  /// 当前入学年份对应的年级（1~4），超出范围按 4 处理；未知返回 0
  int get grade {
    final y = _enrollmentYear;
    if (y == null || y <= 0) return 0;
    final currentYear = DateTime.now().year;
    final g = currentYear - y + 1;
    return g.clamp(1, 4);
  }

  /// 当前生效的年级主题 seed 色（开关关闭或年级未知时用统一滁院蓝）
  Color get seedColor {
    if (!_gradeThemeEnabled) return _GradeThemes.schoolBlue;
    final g = grade;
    if (g == 0) return _GradeThemes.schoolBlue;
    return _GradeThemes.all[g - 1].seed;
  }

  /// 当前年级主题名（如「迎新」「追梦」）
  String get gradeThemeName {
    if (!_gradeThemeEnabled) return '滁院蓝';
    final g = grade;
    if (g == 0) return '滁院蓝';
    return _GradeThemes.all[g - 1].name;
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

/// 四个年级主题定义（按入学年份推导年级：1 大一迎新 → 4 大四创业）
class _GradeThemes {
  final String name;
  final Color seed;
  const _GradeThemes(this.name, this.seed);

  /// 滁州学院统一蓝（默认/关闭开关时）
  static const schoolBlue = Color(0xFF1565C0);

  static const List<_GradeThemes> all = [
    _GradeThemes('迎新', Color(0xFF00897B)), // 大一：温暖青绿
    _GradeThemes('追梦', Color(0xFF1565C0)), // 大二：滁院蓝
    _GradeThemes('奋斗', Color(0xFFEF6C00)), // 大三：奋斗橙
    _GradeThemes('创业', Color(0xFF6A1B9A)), // 大四：创业紫
  ];
}

/// 蔚小芯应用入口
class WxxApp extends StatefulWidget {
  const WxxApp({super.key});

  @override
  State<WxxApp> createState() => _WxxAppState();
}

class _WxxAppState extends State<WxxApp> {
  // 全局共享 ThemeNotifier：登录/刷新资料时由 AuthProvider 回写入学年份
  final ThemeNotifier _themeNotifier = ThemeNotifier();

  @override
  void initState() {
    super.initState();
    // 扫码登录 deep link：APK 被 App Links 唤起（qr-login?qr=xxx）时跳登录页确认
    initQrDeepLink((sessionId) {
      rootNavigatorKey.currentState?.context.go('/login?qr=$sessionId');
    });
  }

  @override
  void dispose() {
    _themeNotifier.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider(create: (_) => AuthProvider(_themeNotifier)),
        ChangeNotifierProvider(create: (_) {
          final p = ChatProvider();
          sessionResetCallbacks.add(p.reset);
          return p;
        }),
        ChangeNotifierProvider(create: (_) => SessionProvider()),
        ChangeNotifierProvider(create: (_) => EnrollmentProvider()),
        ChangeNotifierProvider(create: (_) => ProcessRecordProvider()),
        ChangeNotifierProvider(create: (_) => ProcessProvider()),
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
        ChangeNotifierProvider(create: (_) => HealthProvider()),
        ChangeNotifierProvider(create: (_) => AppCenterProvider()),
        ChangeNotifierProvider(create: (_) => AIBriefingProvider()),
        ChangeNotifierProvider(create: (_) => TwinPortraitProvider()),
        ChangeNotifierProvider(create: (_) => PersonalDetailProvider()),
        ChangeNotifierProvider(create: (_) => _themeNotifier),
      ],
      child: Consumer<ThemeNotifier>(
        builder: (_, themeNotifier, __) {
          final seed = themeNotifier.seedColor;
          return MaterialApp.router(
            title: '蔚小芯',
            debugShowCheckedModeBanner: false,
            themeMode: themeNotifier.mode,
            theme: ThemeData(
              colorSchemeSeed: seed, // 年级主题 seed（滁院蓝/迎新青绿/奋斗橙/创业紫）
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
              colorSchemeSeed: seed,
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
