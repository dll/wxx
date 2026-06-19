import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_new_features_provider.dart';

/// 学科竞赛页面
class CompetitionPage extends StatefulWidget {
  const CompetitionPage({super.key});
  @override
  State<CompetitionPage> createState() => _CompetitionPageState();
}

class _CompetitionPageState extends State<CompetitionPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentNewFeaturesProvider>().fetchCompetitions();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('学科竞赛')),
      body: Consumer<StudentNewFeaturesProvider>(
        builder: (_, p, __) {
          if (p.loading) return const Center(child: CircularProgressIndicator());
          if (p.competitions.isEmpty) return const Center(child: Text('暂无竞赛'));
          return RefreshIndicator(
            onRefresh: () => p.fetchCompetitions(),
            child: ListView.builder(
              padding: const EdgeInsets.all(12),
              itemCount: p.competitions.length,
              itemBuilder: (_, i) => _buildCompetitionCard(context, p.competitions[i], theme),
            ),
          );
        },
      ),
    );
  }

  Widget _buildCompetitionCard(BuildContext context, CompetitionItem c, ThemeData theme) {
    final statusColor = c.status == 'ongoing' ? Colors.green : (c.status == 'ended' ? Colors.grey : theme.colorScheme.primary);
    return Card(
      margin: const EdgeInsets.only(bottom: 10),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                decoration: BoxDecoration(
                  color: _levelColor(c.level).withOpacity(0.1),
                  borderRadius: BorderRadius.circular(4),
                ),
                child: Text(c.levelLabel, style: TextStyle(color: _levelColor(c.level), fontSize: 12, fontWeight: FontWeight.bold)),
              ),
              const SizedBox(width: 8),
              Expanded(child: Text(c.name, style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.bold))),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                decoration: BoxDecoration(color: statusColor.withOpacity(0.1), borderRadius: BorderRadius.circular(4)),
                child: Text(c.status == 'ongoing' ? '进行中' : (c.status == 'ended' ? '已结束' : '即将开始'),
                    style: TextStyle(color: statusColor, fontSize: 11)),
              ),
            ]),
            if (c.description.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(c.description, maxLines: 2, overflow: TextOverflow.ellipsis, style: theme.textTheme.bodySmall),
            ],
            const SizedBox(height: 8),
            Row(children: [
              Icon(Icons.people_outline, size: 14, color: theme.colorScheme.outline),
              const SizedBox(width: 4),
              Text('${c.registrationCount} 人已报名', style: theme.textTheme.bodySmall),
              const Spacer(),
              if (c.status != 'ended')
                ElevatedButton(
                  onPressed: () async {
                    final ok = await context.read<StudentNewFeaturesProvider>().registerCompetition(c.id);
                    if (context.mounted) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        SnackBar(content: Text(ok ? '报名成功' : '报名失败')),
                      );
                    }
                  },
                  style: ElevatedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                    minimumSize: Size.zero,
                    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  ),
                  child: const Text('报名', style: TextStyle(fontSize: 12)),
                ),
            ]),
          ],
        ),
      ),
    );
  }

  Color _levelColor(String level) {
    switch (level) {
      case 'international': return Colors.purple;
      case 'national': return Colors.red;
      case 'provincial': return Colors.orange;
      case 'school': return Colors.blue;
      default: return Colors.grey;
    }
  }
}
