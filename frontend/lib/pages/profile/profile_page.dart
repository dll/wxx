import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import '../../providers/auth_provider.dart';
import '../../utils/storage.dart';

/// 个人中心页
class ProfilePage extends StatefulWidget {
  const ProfilePage({super.key});

  @override
  State<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends State<ProfilePage> {
  @override
  void initState() {
    super.initState();
    // 加载用户资料
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) context.read<AuthProvider>().fetchProfile();
    });
  }

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthProvider>();
    final theme = Theme.of(context);
    final profile = auth.profile;

    return Scaffold(
      appBar: AppBar(title: const Text('个人中心')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // 用户信息卡片
          Card(
            elevation: 0,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(16),
              side: BorderSide(color: theme.colorScheme.outlineVariant),
            ),
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                children: [
                  CircleAvatar(
                    radius: 36,
                    backgroundColor: theme.colorScheme.primaryContainer,
                    child: Text(
                      (profile?.displayName ?? Storage.displayName ?? '?').characters.first,
                      style: TextStyle(
                        fontSize: 28,
                        fontWeight: FontWeight.bold,
                        color: theme.colorScheme.onPrimaryContainer,
                      ),
                    ),
                  ),
                  const SizedBox(height: 12),
                  Text(
                    profile?.displayName ?? Storage.displayName ?? '未登录',
                    style: theme.textTheme.titleLarge?.copyWith(fontWeight: FontWeight.bold),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    profile?.roleLabel ?? '',
                    style: TextStyle(color: theme.colorScheme.primary),
                  ),
                ],
              ),
            ),
          ),

          const SizedBox(height: 16),

          // 信息列表
          if (profile != null) ...[
            _buildInfoTile(context, Icons.badge_outlined, '学号/工号', profile.username),
            if (profile.college.isNotEmpty)
              _buildInfoTile(context, Icons.account_balance_outlined, '学院', profile.college),
            if (profile.major.isNotEmpty)
              _buildInfoTile(context, Icons.book_outlined, '专业', profile.major),
          ],

          const SizedBox(height: 24),

          // 智能体管理入口（管理员可访问）
          if (_canAccessAgents(profile?.role))
            Card(
              elevation: 0,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(12),
                side: BorderSide(color: theme.colorScheme.outlineVariant),
              ),
              child: ListTile(
                leading: Icon(Icons.smart_toy_outlined,
                    color: theme.colorScheme.primary),
                title: const Text('智能体管理'),
                subtitle: const Text('管理 AI 智能体的注册、配置和状态'),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => context.go('/agents'),
              ),
            ),

          // 情感预警入口（辅导员及以上角色可访问）
          if (_canAccessEmotion(profile?.role))
            Card(
              elevation: 0,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(12),
                side: BorderSide(color: theme.colorScheme.outlineVariant),
              ),
              child: ListTile(
                leading: Icon(Icons.warning_amber_rounded,
                    color: theme.colorScheme.error),
                title: const Text('情感预警'),
                subtitle: const Text('查看和管理学生情感告警'),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => context.go('/emotion'),
              ),
            ),

          const SizedBox(height: 16),

          // 关于
          Card(
            elevation: 0,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(12),
              side: BorderSide(color: theme.colorScheme.outlineVariant),
            ),
            child: const ListTile(
              leading: Icon(Icons.info_outline),
              title: Text('关于蔚小芯'),
              subtitle: Text('v0.1.0 · 滁州学院信息学院'),
            ),
          ),

          const SizedBox(height: 24),

          // 退出登录
          SizedBox(
            width: double.infinity,
            height: 48,
            child: OutlinedButton.icon(
              onPressed: () async {
                await auth.logout();
                if (context.mounted) {
                  context.go('/login');
                }
              },
              icon: const Icon(Icons.logout),
              label: const Text('退出登录'),
              style: OutlinedButton.styleFrom(
                foregroundColor: theme.colorScheme.error,
                side: BorderSide(color: theme.colorScheme.error),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildInfoTile(BuildContext context, IconData icon, String label, String value) {
    final theme = Theme.of(context);
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: ListTile(
        leading: Icon(icon, color: theme.colorScheme.primary),
        title: Text(label),
        trailing: Text(
          value,
          style: TextStyle(color: theme.colorScheme.onSurfaceVariant),
        ),
      ),
    );
  }

  /// 判断角色是否可访问情感预警
  bool _canAccessEmotion(String? role) {
    const allowedRoles = {
      'sys_admin',
      'school_admin',
      'college_admin',
      'counselor',
    };
    return role != null && allowedRoles.contains(role);
  }

  /// 判断角色是否可访问智能体管理
  bool _canAccessAgents(String? role) {
    const allowedRoles = {'sys_admin', 'school_admin'};
    return role != null && allowedRoles.contains(role);
  }
}
