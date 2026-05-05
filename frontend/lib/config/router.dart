import 'dart:ui';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../utils/storage.dart';
import '../pages/login/login_page.dart';
import '../pages/home/home_page.dart';
import '../pages/chat/chat_page.dart';
import '../pages/sessions/sessions_page.dart';
import '../pages/profile/profile_page.dart';
import '../pages/enrollment/enrollment_page.dart';
import '../pages/browse/browse_page.dart';
import '../pages/emotion/emotion_dashboard_page.dart';
import '../pages/agent/agent_management_page.dart';

/// 应用路由配置
final GoRouter appRouter = GoRouter(
  initialLocation: '/home',
  redirect: (context, state) {
    final loggedIn = Storage.isLoggedIn;
    final isLoginPage = state.matchedLocation == '/login';

    if (!loggedIn && !isLoginPage) return '/login';
    if (loggedIn && isLoginPage) return '/home';

    return null;
  },
  routes: [
    GoRoute(
      path: '/login',
      builder: (context, state) => const LoginPage(),
    ),
    ShellRoute(
      builder: (context, state, child) => MainShell(child: child),
      routes: [
        GoRoute(
          path: '/home',
          builder: (context, state) => const HomePage(),
        ),
        GoRoute(
          path: '/chat',
          builder: (context, state) {
            final askQuery = state.uri.queryParameters['ask'];
            return ChatPage(initialQuestion: askQuery);
          },
        ),
        GoRoute(
          path: '/browse',
          builder: (context, state) => const BrowsePage(),
        ),
        GoRoute(
          path: '/enrollment',
          builder: (context, state) => const EnrollmentPage(),
        ),
        GoRoute(
          path: '/sessions',
          builder: (context, state) => const SessionsPage(),
        ),
        GoRoute(
          path: '/emotion',
          builder: (context, state) => const EmotionDashboardPage(),
        ),
        GoRoute(
          path: '/agents',
          builder: (context, state) => const AgentManagementPage(),
        ),
        GoRoute(
          path: '/profile',
          builder: (context, state) => const ProfilePage(),
        ),
      ],
    ),
  ],
);

/// 导航项定义
class _NavItem {
  final String label;
  final IconData icon;
  final IconData selectedIcon;
  final String route;
  const _NavItem(this.label, this.icon, this.selectedIcon, this.route);
}

const _navItems = [
  _NavItem('首页', Icons.home_outlined, Icons.home, '/home'),
  _NavItem('对话', Icons.chat_bubble_outline, Icons.chat_bubble, '/chat'),
  _NavItem('知识', Icons.menu_book_outlined, Icons.menu_book, '/browse'),
  _NavItem('办事', Icons.assignment_outlined, Icons.assignment, '/enrollment'),
  _NavItem('我的', Icons.person_outline, Icons.person, '/profile'),
];

/// 主页面外壳 — 响应式布局 + 磨砂玻璃导航
class MainShell extends StatelessWidget {
  final Widget child;
  const MainShell({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    final width = MediaQuery.of(context).size.width;
    final isDesktop = width > 900;

    if (isDesktop) {
      return _buildDesktopLayout(context);
    }
    return _buildMobileLayout(context);
  }

  /// 桌面布局：左侧 NavigationRail + 右侧内容区（限制最大宽度）
  Widget _buildDesktopLayout(BuildContext context) {
    final index = _currentIndex(context);
    return Scaffold(
      body: Row(
        children: [
          NavigationRail(
            selectedIndex: index,
            onDestinationSelected: (i) => _onTap(context, i),
            labelType: NavigationRailLabelType.all,
            leading: Padding(
              padding: const EdgeInsets.symmetric(vertical: 12),
              child: Icon(Icons.school,
                  size: 32, color: Theme.of(context).colorScheme.primary),
            ),
            destinations: _navItems
                .map((item) => NavigationRailDestination(
                      icon: Icon(item.icon),
                      selectedIcon: Icon(item.selectedIcon),
                      label: Text(item.label),
                    ))
                .toList(),
          ),
          const VerticalDivider(width: 1),
          Expanded(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 900),
              child: child,
            ),
          ),
        ],
      ),
    );
  }

  /// 移动端布局：磨砂玻璃底部导航
  Widget _buildMobileLayout(BuildContext context) {
    final theme = Theme.of(context);
    final index = _currentIndex(context);
    return Scaffold(
      body: child,
      bottomNavigationBar: ClipRRect(
        child: BackdropFilter(
          filter: ImageFilter.blur(sigmaX: 12, sigmaY: 12),
          child: Container(
            decoration: BoxDecoration(
              color: theme.colorScheme.surface.withValues(alpha: 0.7),
              border: Border(
                top: BorderSide(
                  color: theme.colorScheme.outlineVariant.withValues(alpha: 0.3),
                ),
              ),
            ),
            child: NavigationBar(
              selectedIndex: index,
              onDestinationSelected: (i) => _onTap(context, i),
              destinations: _navItems
                  .map((item) => NavigationDestination(
                        icon: Icon(item.icon),
                        selectedIcon: Icon(item.selectedIcon),
                        label: item.label,
                      ))
                  .toList(),
            ),
          ),
        ),
      ),
    );
  }

  int _currentIndex(BuildContext context) {
    final location = GoRouterState.of(context).matchedLocation;
    for (int i = 0; i < _navItems.length; i++) {
      if (location.startsWith(_navItems[i].route)) return i;
    }
    return 0;
  }

  void _onTap(BuildContext context, int index) {
    context.go(_navItems[index].route);
  }
}
