import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import '../../../providers/education_provider.dart';
import '../../../widgets/error_view.dart';

class CareerPage extends StatefulWidget {
  const CareerPage({super.key});

  @override
  State<CareerPage> createState() => _CareerPageState();
}

class _CareerPageState extends State<CareerPage> with SingleTickerProviderStateMixin {
  late TabController _tabController;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 4, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<CareerProvider>().fetchJobs();
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
    final provider = context.watch<CareerProvider>();

    return Scaffold(
      appBar: AppBar(
        title: const Text('就业服务'),
        bottom: TabBar(
          controller: _tabController,
          tabs: const [
            Tab(text: '推荐'),
            Tab(text: '招聘'),
            Tab(text: '宣讲'),
            Tab(text: '政策'),
          ],
          onTap: (index) {
            switch (index) {
              case 0:
              case 1:
                provider.fetchJobs();
                break;
              case 2:
                provider.fetchSessions();
                break;
              case 3:
                provider.fetchPolicies();
                break;
            }
          },
        ),
      ),
      body: TabBarView(
        controller: _tabController,
        children: [
          _buildRecommendTab(theme, provider),
          _buildJobsTab(theme, provider),
          _buildSessionsTab(theme, provider),
          _buildPoliciesTab(theme, provider),
        ],
      ),
    );
  }

  Widget _buildRecommendTab(ThemeData theme, CareerProvider provider) {
    if (provider.loading && provider.jobs.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    if (provider.error.isNotEmpty && provider.jobs.isEmpty) {
      return ErrorView.error(
        message: provider.error,
        onRetry: () => provider.fetchJobs(),
      );
    }
    if (provider.jobs.isEmpty) {
      return ErrorView.empty(
        message: '暂无推荐职位',
        subtitle: '稍后再来看看吧',
        icon: Icons.work_outline,
      );
    }
    return RefreshIndicator(
      onRefresh: () => provider.fetchJobs(),
      child: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: provider.jobs.length,
        itemBuilder: (_, i) {
          final job = provider.jobs[i] as Map<String, dynamic>;
          return _buildJobCard(theme, job);
        },
      ),
    );
  }

  Widget _buildJobsTab(ThemeData theme, CareerProvider provider) {
    if (provider.loading && provider.jobs.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    if (provider.error.isNotEmpty && provider.jobs.isEmpty) {
      return ErrorView.error(
        message: provider.error,
        onRetry: () => provider.fetchJobs(),
      );
    }
    if (provider.jobs.isEmpty) {
      return ErrorView.empty(
        message: '暂无招聘信息',
        subtitle: '稍后再来看看吧',
        icon: Icons.work_outline,
      );
    }
    return RefreshIndicator(
      onRefresh: () => provider.fetchJobs(),
      child: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: provider.jobs.length,
        itemBuilder: (_, i) {
          final job = provider.jobs[i] as Map<String, dynamic>;
          return _buildJobCard(theme, job);
        },
      ),
    );
  }

  Widget _buildSessionsTab(ThemeData theme, CareerProvider provider) {
    if (provider.loading && provider.sessions.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    if (provider.error.isNotEmpty && provider.sessions.isEmpty) {
      return ErrorView.error(
        message: provider.error,
        onRetry: () => provider.fetchSessions(),
      );
    }
    if (provider.sessions.isEmpty) {
      return ErrorView.empty(
        message: '暂无宣讲会',
        subtitle: '稍后再来看看吧',
        icon: Icons.event_outlined,
      );
    }
    return RefreshIndicator(
      onRefresh: () => provider.fetchSessions(),
      child: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: provider.sessions.length,
        itemBuilder: (_, i) {
          final session = provider.sessions[i] as Map<String, dynamic>;
          return _buildSessionCard(theme, session);
        },
      ),
    );
  }

  Widget _buildPoliciesTab(ThemeData theme, CareerProvider provider) {
    if (provider.loading && provider.policies.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    if (provider.error.isNotEmpty && provider.policies.isEmpty) {
      return ErrorView.error(
        message: provider.error,
        onRetry: () => provider.fetchPolicies(),
      );
    }
    if (provider.policies.isEmpty) {
      return ErrorView.empty(
        message: '暂无政策文件',
        subtitle: '稍后再来看看吧',
        icon: Icons.description_outlined,
      );
    }
    return RefreshIndicator(
      onRefresh: () => provider.fetchPolicies(),
      child: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: provider.policies.length,
        itemBuilder: (_, i) {
          final policy = provider.policies[i] as Map<String, dynamic>;
          return _buildPolicyCard(theme, policy);
        },
      ),
    );
  }

  Widget _buildJobCard(ThemeData theme, Map<String, dynamic> job) {
    final id = job['id']?.toString() ?? '';
    final title = job['title'] as String? ?? job['position'] as String? ?? '职位';
    final company = job['company'] as String? ?? '';
    final salary = job['salary'] as String? ?? '';
    final location = job['location'] as String? ?? '';
    final tags = (job['tags'] as List?) ?? [];

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: InkWell(
        onTap: () {
          context.go('/student/career/job/$id');
        },
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Text(
                      title,
                      style: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                  if (salary.isNotEmpty)
                    Text(
                      salary,
                      style: TextStyle(
                        color: theme.colorScheme.error,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                ],
              ),
              const SizedBox(height: 8),
              Row(
                children: [
                  Icon(Icons.business_outlined, size: 16, color: theme.colorScheme.onSurfaceVariant),
                  const SizedBox(width: 4),
                  Text(company, style: theme.textTheme.bodySmall),
                  const SizedBox(width: 16),
                  Icon(Icons.location_on_outlined, size: 16, color: theme.colorScheme.onSurfaceVariant),
                  const SizedBox(width: 4),
                  Text(location, style: theme.textTheme.bodySmall),
                ],
              ),
              if (tags.isNotEmpty) ...[
                const SizedBox(height: 10),
                Wrap(
                  spacing: 6,
                  runSpacing: 4,
                  children: [
                    for (final tag in tags)
                      Chip(
                        label: Text(tag.toString(), style: const TextStyle(fontSize: 11)),
                        visualDensity: VisualDensity.compact,
                        padding: EdgeInsets.zero,
                      ),
                  ],
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildSessionCard(ThemeData theme, Map<String, dynamic> session) {
    final title = session['title'] as String? ?? session['company'] as String? ?? '宣讲会';
    final company = session['company'] as String? ?? '';
    final time = session['time'] as String? ?? session['date'] as String? ?? '';
    final location = session['location'] as String? ?? session['venue'] as String? ?? '';

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
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
            const SizedBox(height: 8),
            if (company.isNotEmpty)
              Row(
                children: [
                  Icon(Icons.business_outlined, size: 16, color: theme.colorScheme.onSurfaceVariant),
                  const SizedBox(width: 4),
                  Text(company, style: theme.textTheme.bodySmall),
                ],
              ),
            const SizedBox(height: 4),
            Row(
              children: [
                Icon(Icons.access_time, size: 16, color: theme.colorScheme.onSurfaceVariant),
                const SizedBox(width: 4),
                Text(time, style: theme.textTheme.bodySmall),
              ],
            ),
            const SizedBox(height: 4),
            Row(
              children: [
                Icon(Icons.location_on_outlined, size: 16, color: theme.colorScheme.onSurfaceVariant),
                const SizedBox(width: 4),
                Text(location, style: theme.textTheme.bodySmall),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildPolicyCard(ThemeData theme, Map<String, dynamic> policy) {
    final id = policy['id']?.toString() ?? '';
    final title = policy['title'] as String? ?? '政策文件';
    final category = policy['category'] as String? ?? '';
    final date = policy['date'] as String? ?? policy['publish_date'] as String? ?? '';

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: InkWell(
        onTap: () {
          context.go('/student/career/policy/$id');
        },
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                    decoration: BoxDecoration(
                      color: theme.colorScheme.primary.withOpacity(0.12),
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: Text(
                      category,
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: theme.colorScheme.primary,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              Text(
                title,
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w600,
                ),
              ),
              if (date.isNotEmpty) ...[
                const SizedBox(height: 8),
                Text(
                  date,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
