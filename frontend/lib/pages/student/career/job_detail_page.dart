import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../../providers/education_provider.dart';
import '../../../widgets/error_view.dart';
import '../../../widgets/md_text.dart';

class JobDetailPage extends StatefulWidget {
  final String jobId;
  const JobDetailPage({super.key, required this.jobId});

  @override
  State<JobDetailPage> createState() => _JobDetailPageState();
}

class _JobDetailPageState extends State<JobDetailPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<CareerProvider>().fetchJobDetail(widget.jobId);
    });
  }

  @override
  void dispose() {
    context.read<CareerProvider>().clearDetail();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<CareerProvider>();
    final job = provider.jobDetail;

    return Scaffold(
      appBar: AppBar(title: const Text('职位详情')),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty && job == null
              ? ErrorView.error(
                  message: provider.error,
                  onRetry: () => provider.fetchJobDetail(widget.jobId),
                )
              : job == null
                  ? ErrorView.empty(
                      message: '职位不存在',
                      icon: Icons.work_off_outlined,
                    )
                  : _buildDetail(theme, job),
    );
  }

  Widget _buildDetail(ThemeData theme, Map<String, dynamic> job) {
    final title = job['title'] as String? ?? job['position'] as String? ?? '';
    final company = job['company'] as String? ?? '';
    final salary = job['salary'] as String? ?? '';
    final location = job['location'] as String? ?? '';
    final description = job['description'] as String? ?? job['content'] as String? ?? '';
    final requirements = job['requirements'] as String? ?? '';
    final benefits = job['benefits'] as String? ?? '';
    final tags = (job['tags'] as List?) ?? [];

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Expanded(
                        child: Text(
                          title,
                          style: theme.textTheme.titleLarge?.copyWith(
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                      ),
                      if (salary.isNotEmpty)
                        Text(
                          salary,
                          style: TextStyle(
                            color: theme.colorScheme.error,
                            fontSize: 20,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                    ],
                  ),
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      Icon(Icons.business_outlined, size: 18, color: theme.colorScheme.onSurfaceVariant),
                      const SizedBox(width: 6),
                      Text(company, style: theme.textTheme.bodyMedium),
                      const SizedBox(width: 20),
                      Icon(Icons.location_on_outlined, size: 18, color: theme.colorScheme.onSurfaceVariant),
                      const SizedBox(width: 6),
                      Text(location, style: theme.textTheme.bodyMedium),
                    ],
                  ),
                  if (tags.isNotEmpty) ...[
                    const SizedBox(height: 12),
                    Wrap(
                      spacing: 8,
                      runSpacing: 6,
                      children: [
                        for (final tag in tags)
                          Chip(
                            label: Text(tag.toString()),
                            shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(20),
                            ),
                          ),
                      ],
                    ),
                  ],
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          if (description.isNotEmpty) ...[
            Text('职位描述', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
            const SizedBox(height: 12),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: MdText(description, style: theme.textTheme.bodyMedium),
              ),
            ),
            const SizedBox(height: 16),
          ],
          if (requirements.isNotEmpty) ...[
            Text('任职要求', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
            const SizedBox(height: 12),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: MdText(requirements, style: theme.textTheme.bodyMedium),
              ),
            ),
            const SizedBox(height: 16),
          ],
          if (benefits.isNotEmpty) ...[
            Text('福利待遇', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
            const SizedBox(height: 12),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: MdText(benefits, style: theme.textTheme.bodyMedium),
              ),
            ),
            const SizedBox(height: 24),
          ],
          SizedBox(
            width: double.infinity,
            child: FilledButton.icon(
              onPressed: () {
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('简历投递功能即将开放')),
                );
              },
              icon: const Icon(Icons.send_outlined, size: 18),
              label: const Text('投递简历'),
              style: FilledButton.styleFrom(
                padding: const EdgeInsets.symmetric(vertical: 14),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
