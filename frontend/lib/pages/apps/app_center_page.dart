import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../models/models.dart';
import '../../providers/app_center_provider.dart';
import '../../widgets/error_view.dart';

/// 应用中心 — 展示当前用户可见的第三方应用（按 category 分组）
class AppCenterPage extends StatefulWidget {
  const AppCenterPage({super.key});

  @override
  State<AppCenterPage> createState() => _AppCenterPageState();
}

class _AppCenterPageState extends State<AppCenterPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AppCenterProvider>().fetchApps();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<AppCenterProvider>();

    return Scaffold(
      appBar: AppBar(title: const Text('应用中心')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchApps(),
        child: provider.loading
            ? const Center(child: CircularProgressIndicator())
            : provider.error.isNotEmpty
                ? ErrorView.error(
                    message: provider.error,
                    onRetry: () => provider.fetchApps())
                : _buildContent(theme, provider),
      ),
    );
  }

  Widget _buildContent(ThemeData theme, AppCenterProvider provider) {
    Widget body;
    if (provider.apps.isEmpty) {
      body = const Center(child: Text('暂无可用的第三方应用'));
    } else {
      body = ListView(
        padding: const EdgeInsets.all(16),
        children: _buildCategorySections(theme, provider),
      );
    }
    return body;
  }

  List<Widget> _buildCategorySections(
      ThemeData theme, AppCenterProvider provider) {
    final grouped = provider.grouped;
    final sections = <Widget>[];

    // 按预设分类顺序输出，未知分类排在最后
    final ordered = <String>[];
    for (final cat in ['study', 'culture', 'service', 'admin', 'external']) {
      if (grouped.containsKey(cat)) ordered.add(cat);
    }
    for (final cat in grouped.keys) {
      if (!ordered.contains(cat)) ordered.add(cat);
    }

    for (final cat in ordered) {
      final apps = grouped[cat]!;
      sections.add(Padding(
        padding: const EdgeInsets.only(top: 8, bottom: 8),
        child: Text(_categoryLabels[cat] ?? cat,
            style: theme.textTheme.titleMedium),
      ));
      for (final app in apps) {
        sections.add(_buildAppCard(theme, app));
      }
    }
    return sections;
  }

  Widget _buildAppCard(ThemeData theme, ExternalAppItem app) {
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: ListTile(
        leading: _buildIcon(theme, app),
        title: Text(app.name),
        subtitle: app.summary.isNotEmpty ? Text(app.summary) : null,
        trailing: app.version.isNotEmpty
            ? Text('v${app.version}', style: theme.textTheme.bodySmall)
            : const Icon(Icons.chevron_right),
        onTap: () => _openApp(context, app),
      ),
    );
  }

  Widget _buildIcon(ThemeData theme, ExternalAppItem app) {
    if (app.icon.startsWith('http')) {
      return ClipRRect(
        borderRadius: BorderRadius.circular(8),
        child: Image.network(
          app.icon,
          width: 40,
          height: 40,
          fit: BoxFit.cover,
          errorBuilder: (_, __, ___) => _buildPlaceholder(theme, app),
        ),
      );
    }
    return _buildPlaceholder(theme, app);
  }

  Widget _buildPlaceholder(ThemeData theme, ExternalAppItem app) {
    return Container(
      width: 40,
      height: 40,
      decoration: BoxDecoration(
        color: theme.colorScheme.primaryContainer,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Icon(Icons.apps, color: theme.colorScheme.primary),
    );
  }

  Future<void> _openApp(BuildContext context, ExternalAppItem app) async {
    if (app.url.isEmpty) return;
    final uri = Uri.parse(app.url);
    final messenger = ScaffoldMessenger.of(context);
    // _self 为内嵌打开（移动端无内嵌 WebView 内核时退回外部浏览器）；
    // 其余统一走系统浏览器 / WebView，进度与跳转由外部系统负责
    try {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    } catch (_) {
      messenger.showSnackBar(const SnackBar(content: Text('无法打开应用')));
    }
  }
}

/// 分类展示标签
const Map<String, String> _categoryLabels = {
  'study': '学习',
  'culture': '校园文化',
  'service': '生活服务',
  'admin': '管理',
  'external': '其他',
};