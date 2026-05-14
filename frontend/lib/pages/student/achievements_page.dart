import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../widgets/error_view.dart';

class AchievementsPage extends StatefulWidget {
  const AchievementsPage({super.key});
  @override
  State<AchievementsPage> createState() => _AchievementsPageState();
}

class _AchievementsPageState extends State<AchievementsPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentFeatureProvider>().fetchAchievements();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    return Scaffold(
      appBar: AppBar(title: const Text('积分与成就')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchAchievements(),
        child: provider.loading
            ? const Center(child: CircularProgressIndicator())
            : provider.error.isNotEmpty
                ? ErrorView.error(message: provider.error, onRetry: () => provider.fetchAchievements())
                : _buildContent(theme, provider),
      ),
    );
  }

  Widget _buildContent(ThemeData theme, StudentFeatureProvider provider) {
    final a = provider.achievements;
    if (a == null) return const Center(child: Text('暂无数据'));
    final progress = a.nextLevelPoints > 0 ? a.totalPoints / a.nextLevelPoints : 1.0;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          color: theme.colorScheme.primaryContainer,
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(children: [
              Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
                Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Text('Lv.${a.level} ${a.levelName}', style: theme.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold)),
                  Text('总积分: ${a.totalPoints}', style: theme.textTheme.bodyMedium),
                ]),
                if (a.weeklyRank > 0) Column(children: [
                  Text('周榜', style: theme.textTheme.bodySmall),
                  Text('#${a.weeklyRank}', style: theme.textTheme.headlineSmall?.copyWith(color: Colors.amber, fontWeight: FontWeight.bold)),
                ]),
              ]),
              const SizedBox(height: 12),
              LinearProgressIndicator(value: progress.clamp(0.0, 1.0), minHeight: 8, borderRadius: BorderRadius.circular(4)),
              const SizedBox(height: 4),
              Text('距离下一级还需 ${a.nextLevelPoints - a.totalPoints} 积分', style: theme.textTheme.bodySmall),
            ]),
          ),
        ),
        if (a.badges.isNotEmpty) ...[
          const SizedBox(height: 20),
          Text('我的徽章', style: theme.textTheme.titleMedium),
          const SizedBox(height: 12),
          GridView.builder(
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(crossAxisCount: 3, childAspectRatio: 0.85, crossAxisSpacing: 8, mainAxisSpacing: 8),
            itemCount: a.badges.length,
            itemBuilder: (_, i) {
              final badge = a.badges[i];
              return Card(
                color: badge.unlocked ? null : theme.colorScheme.surfaceContainerHighest,
                child: Padding(
                  padding: const EdgeInsets.all(8),
                  child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
                    Icon(badge.unlocked ? Icons.emoji_events : Icons.lock_outline, size: 32, color: badge.unlocked ? Colors.amber : Colors.grey),
                    const SizedBox(height: 4),
                    Text(badge.name, style: theme.textTheme.bodySmall, textAlign: TextAlign.center, maxLines: 2, overflow: TextOverflow.ellipsis),
                  ]),
                ),
              );
            },
          ),
        ],
      ],
    );
  }
}
