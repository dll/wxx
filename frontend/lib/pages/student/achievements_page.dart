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
                ? ErrorView.error(
                    message: provider.error,
                    onRetry: () => provider.fetchAchievements())
                : _buildContent(theme, provider),
      ),
    );
  }

  Widget _buildContent(ThemeData theme, StudentFeatureProvider provider) {
    final a = provider.achievements;
    if (a == null) return const Center(child: Text('暂无数据'));
    final progress =
        a.nextLevelPoints > 0 ? a.totalPoints / a.nextLevelPoints : 1.0;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        // ── 等级 / 积分进度卡 ──
        Card(
          color: theme.colorScheme.primaryContainer,
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(children: [
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Lv.${a.level} ${a.levelName}',
                        style: theme.textTheme.headlineSmall
                            ?.copyWith(fontWeight: FontWeight.bold),
                      ),
                      Text('总积分: ${a.totalPoints}',
                          style: theme.textTheme.bodyMedium),
                    ],
                  ),
                  Column(
                    children: [
                      Text('周榜', style: theme.textTheme.bodySmall),
                      Text(
                        a.weeklyRank > 0 ? '#${a.weeklyRank}' : '--',
                        style: theme.textTheme.headlineSmall?.copyWith(
                            color: Colors.amber, fontWeight: FontWeight.bold),
                      ),
                    ],
                  ),
                ],
              ),
              const SizedBox(height: 12),
              LinearProgressIndicator(
                value: progress.clamp(0.0, 1.0),
                minHeight: 8,
                borderRadius: BorderRadius.circular(4),
              ),
              const SizedBox(height: 4),
              Text(
                '距离下一级还需 ${(a.nextLevelPoints - a.totalPoints).clamp(0, 1 << 31)} 积分',
                style: theme.textTheme.bodySmall,
              ),
            ]),
          ),
        ),
        const SizedBox(height: 12),
        // ── 排行榜入口（学习活跃周榜）──
        OutlinedButton.icon(
          onPressed: () =>
              Navigator.of(context).pushNamed('/student/qa-leaderboard'),
          icon: const Icon(Icons.leaderboard_outlined, size: 18),
          label: const Text('查看学习活跃周榜'),
        ),
        if (a.badges.isNotEmpty) ...[
          const SizedBox(height: 20),
          Text('我的徽章', style: theme.textTheme.titleMedium),
          const SizedBox(height: 12),
          GridView.builder(
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                crossAxisCount: 3,
                childAspectRatio: 0.78,
                crossAxisSpacing: 8,
                mainAxisSpacing: 8),
            itemCount: a.badges.length,
            itemBuilder: (_, i) {
              final badge = a.badges[i];
              return Card(
                color: badge.unlocked
                    ? null
                    : theme.colorScheme.surfaceContainerHighest,
                child: Padding(
                  padding: const EdgeInsets.all(8),
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Icon(
                        badge.unlocked
                            ? Icons.emoji_events
                            : Icons.lock_outline,
                        size: 32,
                        color: badge.unlocked ? Colors.amber : Colors.grey,
                      ),
                      const SizedBox(height: 4),
                      Text(
                        badge.name,
                        style: theme.textTheme.bodySmall,
                        textAlign: TextAlign.center,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                      ),
                      const SizedBox(height: 2),
                      Text(
                        badge.unlocked
                            ? '已解锁'
                            : (badge.description.isNotEmpty
                                ? badge.description
                                : '未解锁'),
                        style: theme.textTheme.labelSmall?.copyWith(
                          color: badge.unlocked
                              ? Colors.green
                              : theme.colorScheme.outline,
                        ),
                        textAlign: TextAlign.center,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ],
                  ),
                ),
              );
            },
          ),
        ],
        // ── 新手：怎么赚积分 ──
        const SizedBox(height: 20),
        Text('怎么赚积分？', style: theme.textTheme.titleMedium),
        const SizedBox(height: 8),
        _buildEarnGuide(theme, '每日学习打卡', '+5 积分'),
        _buildEarnGuide(theme, '在问答广场发帖提问', '+10 积分'),
        _buildEarnGuide(theme, '回答别人的问题', '+15 积分'),
      ],
    );
  }

  Widget _buildEarnGuide(ThemeData theme, String action, String reward) {
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(12),
        border:
            Border.all(color: theme.colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      child: Row(children: [
        Icon(Icons.add_circle_outline,
            color: theme.colorScheme.primary, size: 20),
        const SizedBox(width: 10),
        Expanded(child: Text(action, style: theme.textTheme.bodyMedium)),
        Text(
          reward,
          style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.primary, fontWeight: FontWeight.w700),
        ),
      ]),
    );
  }
}
