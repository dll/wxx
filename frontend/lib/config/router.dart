import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../utils/storage.dart';
import '../pages/login/login_page.dart';
import '../pages/chat/chat_page.dart';
import '../pages/sessions/sessions_page.dart';
import '../pages/profile/profile_page.dart';

/// 应用路由配置
final GoRouter appRouter = GoRouter(
  initialLocation: '/chat',
  redirect: (context, state) {
    final loggedIn = Storage.isLoggedIn;
    final isLoginPage = state.matchedLocation == '/login';

    // 未登录且不在登录页 → 跳转登录
    if (!loggedIn && !isLoginPage) return '/login';
    // 已登录且在登录页 → 跳转对话
    if (loggedIn && isLoginPage) return '/chat';

    return null; // 不重定向
  },
  routes: [
    GoRoute(
      path: '/login',
      builder: (context, state) => const LoginPage(),
    ),
    // 带底部导航栏的主页面
    ShellRoute(
      builder: (context, state, child) => MainShell(child: child),
      routes: [
        GoRoute(
          path: '/chat',
          builder: (context, state) => const ChatPage(),
        ),
        GoRoute(
          path: '/sessions',
          builder: (context, state) => const SessionsPage(),
        ),
        GoRoute(
          path: '/profile',
          builder: (context, state) => const ProfilePage(),
        ),
      ],
    ),
  ],
);

/// 主页面外壳（底部导航栏）
class MainShell extends StatelessWidget {
  final Widget child;
  const MainShell({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: child,
      bottomNavigationBar: NavigationBar(
        selectedIndex: _currentIndex(context),
        onDestinationSelected: (index) => _onTap(context, index),
        destinations: const [
          NavigationDestination(icon: Icon(Icons.chat_bubble_outline), selectedIcon: Icon(Icons.chat_bubble), label: '对话'),
          NavigationDestination(icon: Icon(Icons.history), selectedIcon: Icon(Icons.history), label: '历史'),
          NavigationDestination(icon: Icon(Icons.person_outline), selectedIcon: Icon(Icons.person), label: '我的'),
        ],
      ),
    );
  }

  int _currentIndex(BuildContext context) {
    final location = GoRouterState.of(context).matchedLocation;
    if (location.startsWith('/sessions')) return 1;
    if (location.startsWith('/profile')) return 2;
    return 0;
  }

  void _onTap(BuildContext context, int index) {
    switch (index) {
      case 0:
        context.go('/chat');
      case 1:
        context.go('/sessions');
      case 2:
        context.go('/profile');
    }
  }
}
