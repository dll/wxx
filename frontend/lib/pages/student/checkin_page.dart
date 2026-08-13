import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../widgets/error_view.dart';

class CheckinPage extends StatefulWidget {
  const CheckinPage({super.key});
  @override
  State<CheckinPage> createState() => _CheckinPageState();
}

class _CheckinPageState extends State<CheckinPage> {
  bool _checking = false;
  bool _makingUp = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentFeatureProvider>().fetchCheckin();
    });
  }

  Future<void> _doCheckin() async {
    setState(() => _checking = true);
    final ok = await context.read<StudentFeatureProvider>().doCheckin();
    setState(() => _checking = false);
    if (ok && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('打卡成功！')));
    }
  }

  /// 选择本月错过的日期并补签
  Future<void> _doMakeup() async {
    final now = DateTime.now();
    final firstOfMonth = DateTime(now.year, now.month, 1);
    final picked = await showDatePicker(
      context: context,
      initialDate: now.subtract(const Duration(days: 1)),
      firstDate: firstOfMonth,
      lastDate: now.subtract(const Duration(days: 1)),
      helpText: '选择要补签的日期',
      cancelText: '取消',
      confirmText: '确定',
    );
    if (picked == null || !mounted) return;

    final dateStr = '${picked.year.toString().padLeft(4, '0')}-'
        '${picked.month.toString().padLeft(2, '0')}-'
        '${picked.day.toString().padLeft(2, '0')}';
    setState(() => _makingUp = true);
    final msg = await context.read<StudentFeatureProvider>().doMakeupCheckin(dateStr);
    if (!mounted) return;
    setState(() => _makingUp = false);
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    return Scaffold(
      appBar: AppBar(title: const Text('每日打卡')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchCheckin(),
        child: provider.error.isNotEmpty
            ? ErrorView.error(message: provider.error, onRetry: () => provider.fetchCheckin())
            : _buildContent(theme, provider),
      ),
    );
  }

  Widget _buildContent(ThemeData theme, StudentFeatureProvider provider) {
    final c = provider.checkin;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Center(child: Column(children: [
          const SizedBox(height: 24),
          GestureDetector(
            onTap: (c?.todayChecked ?? false) || _checking ? null : _doCheckin,
            child: Container(
              width: 120, height: 120,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: (c?.todayChecked ?? false) ? Colors.green : theme.colorScheme.primary,
                boxShadow: [BoxShadow(color: theme.colorScheme.primary.withOpacity( 0.3), blurRadius: 16, spreadRadius: 2)],
              ),
              child: Center(child: _checking
                  ? const CircularProgressIndicator(color: Colors.white)
                  : Column(mainAxisSize: MainAxisSize.min, children: [
                      Icon((c?.todayChecked ?? false) ? Icons.check : Icons.touch_app, color: Colors.white, size: 36),
                      const SizedBox(height: 4),
                      Text((c?.todayChecked ?? false) ? '已打卡' : '打卡', style: const TextStyle(color: Colors.white, fontSize: 16, fontWeight: FontWeight.bold)),
                    ])),
            ),
          ),
          const SizedBox(height: 24),
        ])),
        if (c != null) ...[
          Row(mainAxisAlignment: MainAxisAlignment.spaceEvenly, children: [
            _statCard(theme, '连续打卡', '${c.streak} 天', Icons.local_fire_department, Colors.orange),
            _statCard(theme, '累计打卡', '${c.totalDays} 天', Icons.calendar_month, Colors.blue),
            _statCard(theme, '最长连续', '${c.longestStreak} 天', Icons.emoji_events, Colors.amber),
          ]),
          const SizedBox(height: 16),
          // 补签（断签保护：每月 2 次）
          OutlinedButton.icon(
            onPressed: (c.makeupLeft <= 0 || _makingUp) ? null : _doMakeup,
            icon: _makingUp
                ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2))
                : const Icon(Icons.history_edu, size: 18),
            label: Text(c.makeupLeft > 0
                ? '补签（本月剩余 ${c.makeupLeft} 次）'
                : '本月补签次数已用完'),
            style: OutlinedButton.styleFrom(
              foregroundColor: c.makeupLeft > 0 ? theme.colorScheme.primary : theme.disabledColor,
            ),
          ),
          if (c.recentDates.isNotEmpty) ...[
            const SizedBox(height: 24),
            Text('近期打卡记录', style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            Wrap(spacing: 8, runSpacing: 8, children: c.recentDates.map((d) => Chip(
              avatar: const Icon(Icons.check_circle, size: 16, color: Colors.green),
              label: Text(d),
            )).toList()),
          ],
        ],
      ],
    );
  }

  Widget _statCard(ThemeData theme, String label, String value, IconData icon, Color color) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        child: Column(children: [
          Icon(icon, color: color, size: 28),
          const SizedBox(height: 4),
          Text(value, style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
          Text(label, style: theme.textTheme.bodySmall),
        ]),
      ),
    );
  }
}
