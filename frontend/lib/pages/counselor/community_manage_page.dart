import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../config/api_config.dart';

class CommunityManagePage extends StatefulWidget {
  const CommunityManagePage({super.key});
  @override
  State<CommunityManagePage> createState() => _CommunityManagePageState();
}

class _CommunityManagePageState extends State<CommunityManagePage> {
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    try {
      await context.read<StudentFeatureProvider>().askAI(ApiConfig.counselorCommunityManage);
    } catch (_) {}
    if (mounted) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('社区管理')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                _statRow(theme),
                const SizedBox(height: 16),
                _buildSection(theme, '待审核内容', Icons.pending_actions, [
                  _reviewTile(theme, '转专业经验分享', '张三', '2小时前'),
                  _reviewTile(theme, '期末复习资料汇总', '李四', '3小时前'),
                ]),
                const SizedBox(height: 16),
                _buildSection(theme, '举报处理', Icons.report, [
                  _reportTile(theme, '不当言论', '匿名举报', '待处理'),
                ]),
              ],
            ),
    );
  }

  Widget _statRow(ThemeData theme) {
    return Row(children: [
      _statCard(theme, '今日发帖', '23', Icons.article),
      const SizedBox(width: 8),
      _statCard(theme, '待审核', '5', Icons.pending),
      const SizedBox(width: 8),
      _statCard(theme, '举报', '1', Icons.flag),
    ]);
  }

  Widget _statCard(ThemeData theme, String label, String value, IconData icon) {
    return Expanded(
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(children: [
            Icon(icon, color: theme.colorScheme.primary),
            const SizedBox(height: 4),
            Text(value, style: theme.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold)),
            Text(label, style: theme.textTheme.bodySmall),
          ]),
        ),
      ),
    );
  }

  Widget _buildSection(ThemeData theme, String title, IconData icon, List<Widget> children) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Icon(icon, size: 20, color: theme.colorScheme.primary),
            const SizedBox(width: 8),
            Text(title, style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
          ]),
          const Divider(),
          ...children,
        ]),
      ),
    );
  }

  Widget _reviewTile(ThemeData theme, String title, String author, String time) {
    return ListTile(
      title: Text(title),
      subtitle: Text('$author · $time'),
      trailing: Row(mainAxisSize: MainAxisSize.min, children: [
        IconButton(icon: const Icon(Icons.check_circle, color: Colors.green), onPressed: () {}),
        IconButton(icon: const Icon(Icons.cancel, color: Colors.red), onPressed: () {}),
      ]),
    );
  }

  Widget _reportTile(ThemeData theme, String reason, String reporter, String status) {
    return ListTile(
      leading: const Icon(Icons.warning, color: Colors.orange),
      title: Text(reason),
      subtitle: Text(reporter),
      trailing: Chip(label: Text(status)),
    );
  }
}
