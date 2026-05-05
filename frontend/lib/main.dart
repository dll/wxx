import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'config/router.dart';
import 'providers/auth_provider.dart';
import 'providers/chat_provider.dart';
import 'providers/session_provider.dart';
import 'providers/enrollment_provider.dart';
import 'providers/knowledge_provider.dart';
import 'providers/emotion_provider.dart';
import 'utils/storage.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // 初始化本地存储
  await Storage.init();

  runApp(const WxxApp());
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
      ],
      child: MaterialApp.router(
        title: '蔚小芯',
        debugShowCheckedModeBanner: false,
        themeMode: ThemeMode.system, // 跟随系统暗黑模式
        theme: ThemeData(
          colorSchemeSeed: const Color(0xFF1565C0), // 滁州学院蓝
          useMaterial3: true,
          brightness: Brightness.light,
          appBarTheme: const AppBarTheme(
            centerTitle: true,
            elevation: 0,
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
        ),
        routerConfig: appRouter,
      ),
    );
  }
}
