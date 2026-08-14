import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../models/models.dart';
import '../../providers/counselor_feature_provider.dart';

/// 辅导员 - 谈心谈话记录（粘性增强版）
///
/// 在原有"新增/列表"基础上补充：
/// 1. 顶部统计面板：总谈话次数、跟进中、已解决、近30天交流数——直观反映"与学生的交流情况/效果"
/// 2. 按状态筛选（全部 / 跟进中 / 已解决）
/// 3. 记录卡片按 detail 展示学生、话题、情绪、摘要、跟进项、状态
/// 4. 新增记录补全 话题/情绪/状态 字段
class TalkRecordPage extends StatefulWidget {
  const TalkRecordPage({super.key});
  @override
  State<TalkRecordPage> createState() => _TalkRecordPageState();
}

class _TalkRecordPageState extends State<TalkRecordPage> {
  String _filter = 'all'; // all / following / resolved

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<CounselorFeatureProvider>().fetchTalkRecords();
    });
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<CounselorFeatureProvider>();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('谈心谈话记录')),
      floatingActionButton: FloatingActionButton(
        onPressed: () => _showAddDialog(context, provider),
        child: const Icon(Icons.add),
      ),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty
              ? Center(child: Text(provider.error, style: TextStyle(color: theme.colorScheme.error)))
              : RefreshIndicator(
                  onRefresh: () => provider.fetchTalkRecords(),
                  child: ListView(
                    padding: const EdgeInsets.all(16),
                    children: [
                      _buildStats(provider, theme),
                      const SizedBox(height: 16),
                      _buildFilterTabs(theme),
                      const SizedBox(height: 12),
                      ..._buildRecords(provider, theme),
                    ],
                  ),
                ),
    );
  }

  /// 统计面板：总次数 / 跟进中 / 已解决 / 近30天
  Widget _buildStats(CounselorFeatureProvider provider, ThemeData theme) {
    final records = provider.talkRecords;
    final following = records.where((r) => r.status == 'following').length;
    final resolved = records.where((r) => r.status == 'resolved').length;
    final now = DateTime.now();
    final recent = records.where((r) {
      final d = DateTime.tryParse(r.date);
      if (d == null) return false;
      return now.difference(d).inDays <= 30;
    }).length;

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.primaryContainer.withOpacity(0.25),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: theme.colorScheme.primary.withOpacity(0.2)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('近期跟进概览', style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w700)),
          const SizedBox(height: 12),
          Row(
            children: [
              _statItem(theme, '${records.length}', '谈话总次数', Icons.forum_outlined),
              _statItem(theme, '$following', '跟进中', Icons.pending_outlined),
              _statItem(theme, '$resolved', '已解决', Icons.task_alt),
              _statItem(theme, '$recent', '近30天', Icons.date_range_outlined),
            ],
          ),
        ],
      ),
    );
  }

  Widget _statItem(ThemeData theme, String value, String label, IconData icon) {
    return Expanded(
      child: Column(
        children: [
          Icon(icon, size: 20, color: theme.colorScheme.primary),
          const SizedBox(height: 6),
          Text(value, style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w800)),
          Text(label, style: theme.textTheme.labelSmall?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
        ],
      ),
    );
  }

  Widget _buildFilterTabs(ThemeData theme) {
    final tabs = [
      ('all', '全部'),
      ('following', '跟进中'),
      ('resolved', '已解决'),
    ];
    return Row(
      children: tabs.map((t) {
        final selected = _filter == t.$1;
        return Expanded(
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 3),
            child: ChoiceChip(
              label: Text(t.$2),
              selected: selected,
              onSelected: (_) => setState(() => _filter = t.$1),
            ),
          ),
        );
      }).toList(),
    );
  }

  List<Widget> _buildRecords(CounselorFeatureProvider provider, ThemeData theme) {
    final records = provider.talkRecords
        .where((r) => _filter == 'all' || r.status == _filter)
        .toList();
    if (records.isEmpty) {
      return [
        Padding(
          padding: const EdgeInsets.only(top: 40),
          child: Center(
            child: Text('暂无谈话记录',
                style: TextStyle(color: theme.colorScheme.onSurfaceVariant)),
          ),
        ),
      ];
    }
    return records.map((r) => _buildRecordCard(theme, r)).toList();
  }

  Widget _buildRecordCard(ThemeData theme, TalkRecord r) {
    final isResolved = r.status == 'resolved';
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                CircleAvatar(child: Text(r.studentName.isNotEmpty ? r.studentName[0] : '?')),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(r.studentName,
                          style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w700)),
                      Text(r.date, style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
                    ],
                  ),
                ),
                _StatusBadge(resolved: isResolved),
              ],
            ),
            if (r.topic.isNotEmpty) ...[
              const SizedBox(height: 10),
              Row(children: [
                Icon(Icons.tag, size: 16, color: theme.colorScheme.primary),
                const SizedBox(width: 6),
                Expanded(child: Text('话题：${r.topic}', style: theme.textTheme.bodySmall)),
              ]),
            ],
            if (r.emotion.isNotEmpty) ...[
              const SizedBox(height: 4),
              Row(children: [
                Icon(Icons.mood, size: 16, color: theme.colorScheme.secondary),
                const SizedBox(width: 6),
                Expanded(child: Text('情绪：${r.emotion}', style: theme.textTheme.bodySmall)),
              ]),
            ],
            if (r.summary.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(r.summary,
                  maxLines: 3,
                  overflow: TextOverflow.ellipsis,
                  style: theme.textTheme.bodyMedium),
            ],
            if (r.followUps.isNotEmpty) ...[
              const SizedBox(height: 8),
              ...r.followUps.map((f) => Padding(
                    padding: const EdgeInsets.only(top: 2),
                    child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                      Icon(Icons.assignment_turned_in_outlined, size: 15, color: theme.colorScheme.tertiary),
                      const SizedBox(width: 6),
                      Expanded(child: Text('跟进：$f', style: theme.textTheme.bodySmall)),
                    ]),
                  )),
            ],
          ],
        ),
      ),
    );
  }

  void _showAddDialog(BuildContext context, CounselorFeatureProvider provider) {
    final nameCtrl = TextEditingController();
    final topicCtrl = TextEditingController();
    final emotionCtrl = TextEditingController();
    final contentCtrl = TextEditingController();
    String status = 'following';
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('新增谈话记录'),
        content: SingleChildScrollView(
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            TextField(controller: nameCtrl, decoration: const InputDecoration(labelText: '学生姓名 *')),
            const SizedBox(height: 12),
            TextField(controller: topicCtrl, decoration: const InputDecoration(labelText: '话题（如：学业压力/宿舍矛盾/就业规划）')),
            const SizedBox(height: 12),
            TextField(controller: emotionCtrl, decoration: const InputDecoration(labelText: '情绪状态（如：平稳/焦虑/低落）')),
            const SizedBox(height: 12),
            TextField(controller: contentCtrl, decoration: const InputDecoration(labelText: '谈话内容'), maxLines: 3),
            const SizedBox(height: 12),
            DropdownButtonFormField<String>(
              value: status,
              decoration: const InputDecoration(labelText: '状态'),
              items: const [
                DropdownMenuItem(value: 'following', child: Text('跟进中')),
                DropdownMenuItem(value: 'resolved', child: Text('已解决')),
              ],
              onChanged: (v) => status = v ?? 'following',
            ),
          ]),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(
            onPressed: () {
              if (nameCtrl.text.trim().isEmpty) {
                ScaffoldMessenger.of(ctx).showSnackBar(const SnackBar(content: Text('请填写学生姓名')));
                return;
              }
              provider.saveTalkRecord({
                'student_name': nameCtrl.text,
                'topic': topicCtrl.text,
                'emotion': emotionCtrl.text,
                'content': contentCtrl.text,
                'status': status,
              });
              Navigator.pop(ctx);
            },
            child: const Text('保存'),
          ),
        ],
      ),
    );
  }
}

class _StatusBadge extends StatelessWidget {
  final bool resolved;
  const _StatusBadge({required this.resolved});
  @override
  Widget build(BuildContext context) {
    final color = resolved ? Colors.green : Colors.orange;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(color: color.withOpacity(0.12), borderRadius: BorderRadius.circular(12)),
      child: Text(resolved ? '已解决' : '跟进中',
          style: TextStyle(fontSize: 12, color: color, fontWeight: FontWeight.w600)),
    );
  }
}
