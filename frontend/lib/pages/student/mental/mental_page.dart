import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import '../../../providers/education_provider.dart';
import '../../../widgets/error_view.dart';

class MentalPage extends StatefulWidget {
  const MentalPage({super.key});

  @override
  State<MentalPage> createState() => _MentalPageState();
}

class _MentalPageState extends State<MentalPage> with SingleTickerProviderStateMixin {
  late TabController _tabController;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 4, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<MentalProvider>().fetchScales();
    });
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<MentalProvider>();

    return Scaffold(
      appBar: AppBar(
        title: const Text('心理健康'),
        bottom: TabBar(
          controller: _tabController,
          tabs: const [
            Tab(text: '测评'),
            Tab(text: '咨询'),
            Tab(text: '科普'),
            Tab(text: '情绪'),
          ],
          onTap: (index) {
            switch (index) {
              case 0:
                provider.fetchScales();
                break;
              case 1:
                provider.fetchCounselors();
                break;
              case 2:
                provider.fetchArticles();
                break;
              case 3:
                provider.fetchMoodRecords();
                break;
            }
          },
        ),
      ),
      body: TabBarView(
        controller: _tabController,
        children: [
          _buildScalesTab(theme, provider),
          _buildCounselingTab(theme, provider),
          _buildArticlesTab(theme, provider),
          _buildMoodTab(theme, provider),
        ],
      ),
    );
  }

  Widget _buildScalesTab(ThemeData theme, MentalProvider provider) {
    if (provider.loading && provider.scales.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    if (provider.error.isNotEmpty && provider.scales.isEmpty) {
      return ErrorView.error(
        message: provider.error,
        onRetry: () => provider.fetchScales(),
      );
    }
    if (provider.scales.isEmpty) {
      return ErrorView.empty(
        message: '暂无心理测评',
        subtitle: '稍后再来看看吧',
        icon: Icons.psychology_outlined,
      );
    }
    return RefreshIndicator(
      onRefresh: () => provider.fetchScales(),
      child: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: provider.scales.length,
        itemBuilder: (_, i) {
          final scale = provider.scales[i] as Map<String, dynamic>;
          return _buildScaleCard(theme, scale);
        },
      ),
    );
  }

  Widget _buildCounselingTab(ThemeData theme, MentalProvider provider) {
    return RefreshIndicator(
      onRefresh: () async {
        await Future.wait([
          provider.fetchCounselors(),
          provider.fetchHotlines(),
        ]);
      },
      child: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildQuickActions(theme),
          const SizedBox(height: 20),
          Text('心理咨询师', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
          const SizedBox(height: 12),
          if (provider.loading && provider.counselors.isEmpty)
            const Center(child: Padding(padding: EdgeInsets.all(32), child: CircularProgressIndicator()))
          else if (provider.counselors.isEmpty)
            ErrorView.empty(
              message: '暂无咨询师',
              icon: Icons.person_outline,
            )
          else
            ...provider.counselors.map((c) {
              final counselor = c as Map<String, dynamic>;
              return _buildCounselorCard(theme, counselor);
            }),
          const SizedBox(height: 20),
          Text('心理援助热线', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
          const SizedBox(height: 12),
          if (provider.hotlines.isEmpty)
            ErrorView.empty(
              message: '暂无热线信息',
              icon: Icons.phone_outlined,
            )
          else
            ...provider.hotlines.map((h) {
              final hotline = h as Map<String, dynamic>;
              return _buildHotlineCard(theme, hotline);
            }),
        ],
      ),
    );
  }

  Widget _buildArticlesTab(ThemeData theme, MentalProvider provider) {
    if (provider.loading && provider.articles.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    if (provider.error.isNotEmpty && provider.articles.isEmpty) {
      return ErrorView.error(
        message: provider.error,
        onRetry: () => provider.fetchArticles(),
      );
    }
    if (provider.articles.isEmpty) {
      return ErrorView.empty(
        message: '暂无科普文章',
        subtitle: '稍后再来看看吧',
        icon: Icons.article_outlined,
      );
    }
    return RefreshIndicator(
      onRefresh: () => provider.fetchArticles(),
      child: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: provider.articles.length,
        itemBuilder: (_, i) {
          final article = provider.articles[i] as Map<String, dynamic>;
          return _buildArticleCard(theme, article);
        },
      ),
    );
  }

  Widget _buildMoodTab(ThemeData theme, MentalProvider provider) {
    return RefreshIndicator(
      onRefresh: () => provider.fetchMoodRecords(),
      child: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildMoodEntryCard(theme),
          const SizedBox(height: 20),
          Text('情绪日记', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
          const SizedBox(height: 12),
          if (provider.loading && provider.moodRecords.isEmpty)
            const Center(child: Padding(padding: EdgeInsets.all(32), child: CircularProgressIndicator()))
          else if (provider.moodRecords.isEmpty)
            ErrorView.empty(
              message: '暂无情绪记录',
              subtitle: '点击上方卡片记录今天的心情',
              icon: Icons.mood_outlined,
            )
          else
            ...provider.moodRecords.map((m) {
              final mood = m as Map<String, dynamic>;
              return _buildMoodRecordCard(theme, mood);
            }),
        ],
      ),
    );
  }

  Widget _buildScaleCard(ThemeData theme, Map<String, dynamic> scale) {
    final id = scale['id']?.toString() ?? '';
    final title = scale['title'] as String? ?? scale['name'] as String? ?? '测评量表';
    final description = scale['description'] as String? ?? scale['intro'] as String? ?? '';
    final questionCount = scale['question_count'] as int? ?? scale['questions'] as int? ?? 0;
    final duration = scale['duration'] as String? ?? '';

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: InkWell(
        onTap: () {
          context.go('/student/mental/scale/$id');
        },
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                title,
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w600,
                ),
              ),
              if (description.isNotEmpty) ...[
                const SizedBox(height: 8),
                Text(
                  description,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
              ],
              const SizedBox(height: 12),
              Row(
                children: [
                  Icon(Icons.help_outline, size: 16, color: theme.colorScheme.onSurfaceVariant),
                  const SizedBox(width: 4),
                  Text('$questionCount题', style: theme.textTheme.bodySmall),
                  const SizedBox(width: 16),
                  if (duration.isNotEmpty) ...[
                    Icon(Icons.timer_outlined, size: 16, color: theme.colorScheme.onSurfaceVariant),
                    const SizedBox(width: 4),
                    Text(duration, style: theme.textTheme.bodySmall),
                  ],
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildQuickActions(ThemeData theme) {
    return Row(
      children: [
        Expanded(
          child: _buildQuickActionCard(
            theme,
            icon: Icons.calendar_month_outlined,
            label: '预约咨询',
            color: theme.colorScheme.primary,
            onTap: () {
              context.go('/student/mental/counseling');
            },
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: _buildQuickActionCard(
            theme,
            icon: Icons.history,
            label: '我的预约',
            color: theme.colorScheme.secondary,
            onTap: () {
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(content: Text('功能开发中')),
              );
            },
          ),
        ),
      ],
    );
  }

  Widget _buildQuickActionCard(
    ThemeData theme, {
    required IconData icon,
    required String label,
    required Color color,
    required VoidCallback onTap,
  }) {
    return Material(
      color: color.withOpacity(0.08),
      borderRadius: BorderRadius.circular(12),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 20),
          child: Column(
            children: [
              Icon(icon, color: color, size: 28),
              const SizedBox(height: 8),
              Text(label, style: TextStyle(color: color, fontWeight: FontWeight.w600)),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildCounselorCard(ThemeData theme, Map<String, dynamic> counselor) {
    final name = counselor['name'] as String? ?? '';
    final title = counselor['title'] as String? ?? counselor['position'] as String? ?? '';
    final specialty = counselor['specialty'] as String? ?? '';
    final avatar = counselor['avatar'] as String? ?? '';

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            CircleAvatar(
              radius: 28,
              backgroundImage: avatar.isNotEmpty ? NetworkImage(avatar) : null,
              child: avatar.isEmpty ? Text(name.isNotEmpty ? name[0] : '?', style: const TextStyle(fontSize: 20)) : null,
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    name,
                    style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600),
                  ),
                  if (title.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Text(title, style: theme.textTheme.bodySmall),
                  ],
                  if (specialty.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Text(
                      '擅长：$specialty',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildHotlineCard(ThemeData theme, Map<String, dynamic> hotline) {
    final name = hotline['name'] as String? ?? '';
    final phone = hotline['phone'] as String? ?? hotline['number'] as String? ?? '';
    final serviceTime = hotline['service_time'] as String? ?? '';

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            Container(
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: theme.colorScheme.error.withOpacity(0.1),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Icon(Icons.phone_in_talk_outlined, color: theme.colorScheme.error),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    name,
                    style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w600),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    phone,
                    style: theme.textTheme.bodyLarge?.copyWith(
                      color: theme.colorScheme.error,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  if (serviceTime.isNotEmpty) ...[
                    const SizedBox(height: 2),
                    Text(serviceTime, style: theme.textTheme.bodySmall),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildArticleCard(ThemeData theme, Map<String, dynamic> article) {
    final id = article['id']?.toString() ?? '';
    final title = article['title'] as String? ?? '科普文章';
    final summary = article['summary'] as String? ?? article['description'] as String? ?? '';
    final category = article['category'] as String? ?? '';
    final date = article['date'] as String? ?? article['publish_date'] as String? ?? '';
    final views = article['views'] as int? ?? article['read_count'] as int? ?? 0;

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: InkWell(
        onTap: () {
          context.go('/student/mental/article/$id');
        },
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (category.isNotEmpty) ...[
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.tertiary.withOpacity(0.12),
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: Text(
                    category,
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: theme.colorScheme.tertiary,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                const SizedBox(height: 8),
              ],
              Text(
                title,
                style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600),
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
              if (summary.isNotEmpty) ...[
                const SizedBox(height: 8),
                Text(
                  summary,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
              ],
              const SizedBox(height: 12),
              Row(
                children: [
                  if (date.isNotEmpty) ...[
                    Icon(Icons.event_outlined, size: 14, color: theme.colorScheme.onSurfaceVariant),
                    const SizedBox(width: 4),
                    Text(date, style: theme.textTheme.bodySmall),
                    const SizedBox(width: 16),
                  ],
                  Icon(Icons.visibility_outlined, size: 14, color: theme.colorScheme.onSurfaceVariant),
                  const SizedBox(width: 4),
                  Text('$views', style: theme.textTheme.bodySmall),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildMoodEntryCard(ThemeData theme) {
    return Material(
      color: theme.colorScheme.primaryContainer.withOpacity(0.5),
      borderRadius: BorderRadius.circular(16),
      child: InkWell(
        onTap: () {
          context.go('/student/mental/mood');
        },
        borderRadius: BorderRadius.circular(16),
        child: Padding(
          padding: const EdgeInsets.all(20),
          child: Row(
            children: [
              Container(
                padding: const EdgeInsets.all(14),
                decoration: BoxDecoration(
                  color: theme.colorScheme.primary,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Icon(Icons.add_reaction_outlined, color: theme.colorScheme.onPrimary, size: 28),
              ),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('记录今日心情', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
                    const SizedBox(height: 4),
                    Text(
                      '用心情日记记录生活，关注自己的情绪变化',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              Icon(Icons.chevron_right, color: theme.colorScheme.onSurfaceVariant),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildMoodRecordCard(ThemeData theme, Map<String, dynamic> mood) {
    final date = mood['date'] as String? ?? mood['created_at'] as String? ?? '';
    final moodType = mood['mood'] as String? ?? mood['type'] as String? ?? '平静';
    final note = mood['note'] as String? ?? mood['content'] as String? ?? '';
    final score = mood['score'] as int? ?? 0;

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text(_getMoodEmoji(moodType), style: const TextStyle(fontSize: 24)),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        moodType,
                        style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w600),
                      ),
                      if (date.isNotEmpty)
                        Text(date, style: theme.textTheme.bodySmall),
                    ],
                  ),
                ),
                if (score > 0)
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                    decoration: BoxDecoration(
                      color: _getMoodColor(moodType).withOpacity(0.15),
                      borderRadius: BorderRadius.circular(20),
                    ),
                    child: Text(
                      '$score分',
                      style: TextStyle(
                        color: _getMoodColor(moodType),
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  ),
              ],
            ),
            if (note.isNotEmpty) ...[
              const SizedBox(height: 10),
              Text(
                note,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  String _getMoodEmoji(String mood) {
    switch (mood) {
      case '开心':
      case 'happy':
        return '😊';
      case '平静':
      case 'calm':
        return '😌';
      case '疲惫':
      case 'tired':
        return '😫';
      case '焦虑':
      case 'anxious':
        return '😰';
      case '难过':
      case 'sad':
        return '😢';
      case '愤怒':
      case 'angry':
        return '😤';
      default:
        return '🙂';
    }
  }

  Color _getMoodColor(String mood) {
    switch (mood) {
      case '开心':
      case 'happy':
        return Colors.green;
      case '平静':
      case 'calm':
        return Colors.blue;
      case '疲惫':
      case 'tired':
        return Colors.grey;
      case '焦虑':
      case 'anxious':
        return Colors.orange;
      case '难过':
      case 'sad':
        return Colors.blueGrey;
      case '愤怒':
      case 'angry':
        return Colors.red;
      default:
        return Colors.grey;
    }
  }
}
