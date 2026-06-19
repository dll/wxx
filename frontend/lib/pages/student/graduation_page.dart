import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_new_features_provider.dart';

/// 毕设选题页面
class GraduationPage extends StatefulWidget {
  const GraduationPage({super.key});
  @override
  State<GraduationPage> createState() => _GraduationPageState();
}

class _GraduationPageState extends State<GraduationPage> with SingleTickerProviderStateMixin {
  late TabController _tabCtrl;

  @override
  void initState() {
    super.initState();
    _tabCtrl = TabController(length: 3, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final p = context.read<StudentNewFeaturesProvider>();
      p.fetchAdvisors();
      p.fetchTopics();
      p.fetchMySelection();
      p.fetchMilestones();
    });
  }

  @override
  void dispose() {
    _tabCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('毕设选题'),
        bottom: TabBar(
          controller: _tabCtrl,
          tabs: const [
            Tab(text: '选题库'),
            Tab(text: '导师'),
            Tab(text: '我的选题'),
          ],
        ),
      ),
      body: Consumer<StudentNewFeaturesProvider>(
        builder: (_, p, __) {
          if (p.loading) return const Center(child: CircularProgressIndicator());
          return TabBarView(
            controller: _tabCtrl,
            children: [
              _buildTopicsTab(context, p, theme),
              _buildAdvisorsTab(context, p, theme),
              _buildMySelectionTab(context, p, theme),
            ],
          );
        },
      ),
    );
  }

  Widget _buildTopicsTab(BuildContext context, StudentNewFeaturesProvider p, ThemeData theme) {
    if (p.topics.isEmpty) return const Center(child: Text('暂无选题'));
    return ListView.builder(
      padding: const EdgeInsets.all(12),
      itemCount: p.topics.length,
      itemBuilder: (_, i) {
        final t = p.topics[i];
        return Card(
          margin: const EdgeInsets.only(bottom: 10),
          child: Padding(
            padding: const EdgeInsets.all(14),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(children: [
                  Expanded(child: Text(t.title, style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.bold))),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                    decoration: BoxDecoration(
                      color: _difficultyColor(t.difficulty).withOpacity(0.1),
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: Text(t.difficultyLabel, style: TextStyle(color: _difficultyColor(t.difficulty), fontSize: 12)),
                  ),
                ]),
                if (t.advisorName.isNotEmpty) ...[
                  const SizedBox(height: 6),
                  Text('指导教师: ${t.advisorName}  |  ${t.major}', style: theme.textTheme.bodySmall),
                ],
                if (t.description.isNotEmpty) ...[
                  const SizedBox(height: 6),
                  Text(t.description, maxLines: 2, overflow: TextOverflow.ellipsis, style: theme.textTheme.bodySmall),
                ],
                if (t.keywords.isNotEmpty) ...[
                  const SizedBox(height: 8),
                  Wrap(
                    spacing: 6,
                    children: t.keywords.split(',').map((k) => Chip(
                      label: Text(k.trim(), style: const TextStyle(fontSize: 11)),
                      materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                      visualDensity: VisualDensity.compact,
                    )).toList(),
                  ),
                ],
                const SizedBox(height: 8),
                Row(children: [
                  Text('已选 ${t.selectedCount}/${t.maxStudents} 人', style: theme.textTheme.bodySmall),
                  const Spacer(),
                  if (t.selectedCount < t.maxStudents)
                    ElevatedButton(
                      onPressed: () async {
                        final ok = await p.selectTopic(t.id);
                        if (context.mounted) {
                          ScaffoldMessenger.of(context).showSnackBar(
                            SnackBar(content: Text(ok ? '选题成功' : p.error)),
                          );
                        }
                      },
                      style: ElevatedButton.styleFrom(
                        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                        minimumSize: Size.zero,
                        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                      ),
                      child: const Text('选择此题', style: TextStyle(fontSize: 12)),
                    )
                  else
                    Text('已满', style: TextStyle(color: theme.colorScheme.error, fontSize: 12)),
                ]),
              ],
            ),
          ),
        );
      },
    );
  }

  Widget _buildAdvisorsTab(BuildContext context, StudentNewFeaturesProvider p, ThemeData theme) {
    if (p.advisors.isEmpty) return const Center(child: Text('暂无导师'));
    return ListView.builder(
      padding: const EdgeInsets.all(12),
      itemCount: p.advisors.length,
      itemBuilder: (_, i) {
        final a = p.advisors[i];
        return Card(
          margin: const EdgeInsets.only(bottom: 10),
          child: ListTile(
            leading: CircleAvatar(child: Text(a.name.isNotEmpty ? a.name[0] : '?')),
            title: Text('${a.name} (${a.title})'),
            subtitle: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('${a.department} | ${a.researchAreas.join("、")}', style: theme.textTheme.bodySmall),
                Text('已带 ${a.currentStudents}/${a.maxStudents} 人', style: theme.textTheme.bodySmall),
              ],
            ),
          ),
        );
      },
    );
  }

  Widget _buildMySelectionTab(BuildContext context, StudentNewFeaturesProvider p, ThemeData theme) {
    if (p.mySelection == null) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.topic_outlined, size: 64, color: theme.colorScheme.outline),
            const SizedBox(height: 16),
            Text('尚未选择毕设题目', style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            Text('请在选题库中选择您感兴趣的题目', style: theme.textTheme.bodySmall),
          ],
        ),
      );
    }
    final s = p.mySelection!;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('我的选题', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                const Divider(),
                Text(s.title, style: theme.textTheme.titleSmall),
                const SizedBox(height: 8),
                _infoRow('指导教师', s.advisorName),
                _infoRow('难度', s.difficultyLabel),
                _infoRow('专业', s.major),
                if (s.description.isNotEmpty) ...[
                  const SizedBox(height: 8),
                  Text('题目说明', style: theme.textTheme.bodySmall?.copyWith(fontWeight: FontWeight.bold)),
                  Text(s.description),
                ],
              ],
            ),
          ),
        ),
        if (p.milestones.isNotEmpty) ...[
          const SizedBox(height: 16),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('毕业里程碑', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                  const Divider(),
                  ...p.milestones.map((m) => ListTile(
                    dense: true,
                    leading: Icon(Icons.flag_outlined, color: theme.colorScheme.primary),
                    title: Text(m['name']?.toString() ?? ''),
                    subtitle: Text('截止: ${m['deadline']?.toString() ?? "未设置"}'),
                    trailing: Text('${m['weight'] ?? 0}%'),
                  )),
                ],
              ),
            ),
          ),
        ],
      ],
    );
  }

  Widget _infoRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        children: [
          SizedBox(width: 80, child: Text(label, style: const TextStyle(color: Colors.grey, fontSize: 13))),
          Expanded(child: Text(value, style: const TextStyle(fontSize: 13))),
        ],
      ),
    );
  }

  Color _difficultyColor(String d) {
    switch (d) {
      case 'easy': return Colors.green;
      case 'medium': return Colors.orange;
      case 'hard': return Colors.red;
      default: return Colors.grey;
    }
  }
}
