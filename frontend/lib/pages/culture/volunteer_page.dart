import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/culture_provider.dart';

/// 志愿服务 — 个人时长 + 项目推荐
class VolunteerPage extends StatefulWidget {
  const VolunteerPage({super.key});

  @override
  State<VolunteerPage> createState() => _VolunteerPageState();
}

class _VolunteerPageState extends State<VolunteerPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<CultureProvider>().fetchVolunteer();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final p = context.watch<CultureProvider>();
    final data = p.volunteer;
    final summary = (data?['my_summary'] as Map<String, dynamic>?) ?? const {};
    final projects = ((data?['projects'] as List?) ?? const []).cast<Map<String, dynamic>>();
    return Scaffold(
      appBar: AppBar(title: const Text('志愿服务')),
      body: p.loading && data == null
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                _SummaryCard(summary: summary, theme: theme),
                const SizedBox(height: 20),
                Text('推荐项目', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
                const SizedBox(height: 8),
                ...projects.map((it) => _ProjectCard(data: it, theme: theme)),
              ],
            ),
    );
  }
}

class _SummaryCard extends StatelessWidget {
  final Map<String, dynamic> summary;
  final ThemeData theme;
  const _SummaryCard({required this.summary, required this.theme});

  @override
  Widget build(BuildContext context) {
    final badges = (summary['badges'] as List?) ?? const [];
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [Color(0xFF388E3C), Color(0xFF1B5E20)],
          begin: Alignment.topLeft, end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('我的志愿时长',
              style: TextStyle(color: Colors.white70, fontSize: 13, letterSpacing: 1.2)),
          const SizedBox(height: 8),
          RichText(
            text: TextSpan(children: [
              TextSpan(
                  text: '${summary['total_hours'] ?? 0}',
                  style: const TextStyle(color: Colors.white, fontSize: 38, fontWeight: FontWeight.bold)),
              const TextSpan(text: ' 小时', style: TextStyle(color: Colors.white70, fontSize: 16)),
            ]),
          ),
          const SizedBox(height: 8),
          Text(
            '已认证 ${summary['verified_hours']}h · 待审核 ${summary['pending_hours']}h · 累计 ${summary['projects_joined']} 个项目',
            style: const TextStyle(color: Colors.white70, fontSize: 12),
          ),
          if (badges.isNotEmpty) ...[
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              children: [
                for (final b in badges)
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                    decoration: BoxDecoration(
                      color: Colors.white.withValues(alpha: 0.18),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Text(b.toString(), style: const TextStyle(color: Colors.white, fontSize: 12)),
                  ),
              ],
            ),
          ],
        ],
      ),
    );
  }
}

class _ProjectCard extends StatelessWidget {
  final Map<String, dynamic> data;
  final ThemeData theme;
  const _ProjectCard({required this.data, required this.theme});

  @override
  Widget build(BuildContext context) {
    final cap = data['capacity'] as int? ?? 1;
    final reg = data['participants'] as int? ?? 0;
    final progress = cap > 0 ? (reg / cap).clamp(0.0, 1.0) : 0.0;
    final tags = (data['tags'] as List?) ?? const [];
    return Card(
      margin: const EdgeInsets.only(bottom: 10),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              const Icon(Icons.volunteer_activism, color: Colors.green),
              const SizedBox(width: 8),
              Expanded(
                child: Text(data['title'] as String? ?? '',
                    style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w600)),
              ),
              Text('${data['hours']}h/次',
                  style: theme.textTheme.labelSmall?.copyWith(color: theme.colorScheme.primary)),
            ]),
            const SizedBox(height: 6),
            Text('${data['organizer']} · ${data['location']} · ${data['frequency']}',
                style: theme.textTheme.bodySmall),
            const SizedBox(height: 8),
            ClipRRect(
              borderRadius: BorderRadius.circular(6),
              child: LinearProgressIndicator(value: progress, minHeight: 4),
            ),
            const SizedBox(height: 6),
            Row(children: [
              Text('$reg / $cap 人', style: theme.textTheme.labelSmall),
              const Spacer(),
              ...tags.take(3).map((t) => Padding(
                    padding: const EdgeInsets.only(left: 4),
                    child: Chip(
                      label: Text(t.toString(), style: const TextStyle(fontSize: 10)),
                      visualDensity: VisualDensity.compact,
                      materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                    ),
                  )),
            ]),
            const SizedBox(height: 4),
            Align(
              alignment: Alignment.centerRight,
              child: FilledButton.tonalIcon(
                icon: const Icon(Icons.handshake_outlined, size: 16),
                label: const Text('加入项目'),
                onPressed: () {
                  ScaffoldMessenger.of(context).showSnackBar(
                    SnackBar(content: Text('申请加入：${data['title']}（提交流程下一阶段对接）')),
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }
}
