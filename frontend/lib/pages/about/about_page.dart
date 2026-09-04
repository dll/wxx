import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import '../../config/release_config.dart';
import '../../providers/update_provider.dart';

class AboutPage extends StatefulWidget {
  const AboutPage({super.key});

  @override
  State<AboutPage> createState() => _AboutPageState();
}

class _AboutPageState extends State<AboutPage> {
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final updateProvider = context.watch<UpdateProvider>();

    return Scaffold(
      appBar: AppBar(
        title: const Text('关于蔚小芯'),
      ),
      body: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          // Logo 和应用名称
          Center(
            child: Column(
              children: [
                Container(
                  width: 80,
                  height: 80,
                  decoration: BoxDecoration(
                    color: theme.colorScheme.primaryContainer,
                    borderRadius: BorderRadius.circular(20),
                  ),
                  child: Icon(
                    Icons.school,
                    size: 48,
                    color: theme.colorScheme.primary,
                  ),
                ),
                const SizedBox(height: 16),
                Text(
                  '蔚小芯',
                  style: theme.textTheme.headlineSmall?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  'v${ReleaseConfig.version} (${ReleaseConfig.buildNumber})',
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  '滁州学院计算机学院',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 32),

          // 版本更新
          Card(
            elevation: 0,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(12),
              side: BorderSide(color: theme.colorScheme.outlineVariant),
            ),
            child: Column(
              children: [
                ListTile(
                  leading: const Icon(Icons.system_update),
                  title: const Text('检查更新'),
                  subtitle: Text(
                    updateProvider.hasUpdate
                        ? '发现新版本 ${updateProvider.latestVersion?['version_name'] ?? ''}'
                        : '当前已是最新版本',
                  ),
                  trailing: updateProvider.checking
                      ? const SizedBox(
                          width: 20,
                          height: 20,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : Icon(
                          updateProvider.hasUpdate
                              ? Icons.new_releases
                              : Icons.check_circle,
                          color: updateProvider.hasUpdate
                              ? Colors.orange
                              : Colors.green,
                        ),
                  onTap: updateProvider.checking
                      ? null
                      : () async {
                          final hasUpdate = await updateProvider.checkUpdate();
                          if (hasUpdate && mounted) {
                            updateProvider.showUpdateDialog(context);
                          } else if (mounted) {
                            ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(
                                content: Text('当前已是最新版本'),
                                duration: Duration(seconds: 2),
                              ),
                            );
                          }
                        },
                ),
                if (updateProvider.hasUpdate &&
                    updateProvider.latestVersion != null) ...[
                  const Divider(height: 1),
                  Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          '新版本：v${updateProvider.latestVersion!['version_name'] ?? ''}',
                          style: const TextStyle(
                            fontWeight: FontWeight.w600,
                            color: Colors.blue,
                          ),
                        ),
                        const SizedBox(height: 8),
                        if ((updateProvider.latestVersion!['title'] ?? '')
                            .isNotEmpty)
                          Text(
                            updateProvider.latestVersion!['title'],
                            style: const TextStyle(fontWeight: FontWeight.w500),
                          ),
                        const SizedBox(height: 8),
                        if ((updateProvider.latestVersion!['changelog'] ?? '')
                            .isNotEmpty)
                          Text(
                            updateProvider.latestVersion!['changelog'],
                            style: TextStyle(
                              color: theme.colorScheme.onSurfaceVariant,
                              fontSize: 13,
                            ),
                          ),
                      ],
                    ),
                  ),
                ],
              ],
            ),
          ),
          const SizedBox(height: 16),

          // 功能介绍
          Card(
            elevation: 0,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(12),
              side: BorderSide(color: theme.colorScheme.outlineVariant),
            ),
            child: Column(
              children: [
                const ListTile(
                  leading: Icon(Icons.auto_awesome),
                  title: Text('AI 智能问答'),
                  subtitle: Text('基于知识库的精准问答服务'),
                ),
                const Divider(height: 1),
                const ListTile(
                  leading: Icon(Icons.menu_book),
                  title: Text('知识治理'),
                  subtitle: Text('完善的知识库管理与审核机制'),
                ),
                const Divider(height: 1),
                ListTile(
                  leading: const Icon(Icons.school),
                  title: const Text('学业辅导'),
                  subtitle: const Text('学习计划、课程表、成绩分析'),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => context.go('/student/study-plan'),
                ),
                const Divider(height: 1),
                ListTile(
                  leading: const Icon(Icons.favorite),
                  title: const Text('心理陪伴'),
                  subtitle: const Text('心理健康测评与咨询预约'),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => context.go('/student/mental'),
                ),
                const Divider(height: 1),
                ListTile(
                  leading: const Icon(Icons.work),
                  title: const Text('就业指导'),
                  subtitle: const Text('就业政策、岗位推荐、面试技巧'),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => context.go('/student/career'),
                ),
              ],
            ),
          ),
          const SizedBox(height: 24),

          // 版权信息
          Center(
            child: Column(
              children: [
                Text(
                  '© ${DateTime.now().year} 滁州学院计算机学院',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  'All rights reserved.',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
