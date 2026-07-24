import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../../providers/education_provider.dart';
import '../../../widgets/error_view.dart';

class PolicyDetailPage extends StatefulWidget {
  final String policyId;
  const PolicyDetailPage({super.key, required this.policyId});

  @override
  State<PolicyDetailPage> createState() => _PolicyDetailPageState();
}

class _PolicyDetailPageState extends State<PolicyDetailPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<CareerProvider>().fetchPolicyDetail(widget.policyId);
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
    final policy = provider.policyDetail;

    return Scaffold(
      appBar: AppBar(title: const Text('政策详情')),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty && policy == null
              ? ErrorView.error(
                  message: provider.error,
                  onRetry: () => provider.fetchPolicyDetail(widget.policyId),
                )
              : policy == null
                  ? ErrorView.empty(
                      message: '政策不存在',
                      icon: Icons.description_outlined,
                    )
                  : _buildDetail(theme, policy),
    );
  }

  Widget _buildDetail(ThemeData theme, Map<String, dynamic> policy) {
    final title = policy['title'] as String? ?? '';
    final category = policy['category'] as String? ?? '';
    final source = policy['source'] as String? ?? '';
    final date = policy['date'] as String? ?? policy['publish_date'] as String? ?? '';
    final content = policy['content'] as String? ?? policy['body'] as String? ?? '';

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
                  if (category.isNotEmpty)
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                      decoration: BoxDecoration(
                        color: theme.colorScheme.primary.withOpacity(0.12),
                        borderRadius: BorderRadius.circular(6),
                      ),
                      child: Text(
                        category,
                        style: theme.textTheme.labelMedium?.copyWith(
                          color: theme.colorScheme.primary,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                  if (category.isNotEmpty) const SizedBox(height: 12),
                  Text(
                    title,
                    style: theme.textTheme.titleLarge?.copyWith(
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      if (source.isNotEmpty) ...[
                        Icon(Icons.source_outlined, size: 16, color: theme.colorScheme.onSurfaceVariant),
                        const SizedBox(width: 4),
                        Text(source, style: theme.textTheme.bodySmall),
                        const SizedBox(width: 16),
                      ],
                      if (date.isNotEmpty) ...[
                        Icon(Icons.event_outlined, size: 16, color: theme.colorScheme.onSurfaceVariant),
                        const SizedBox(width: 4),
                        Text(date, style: theme.textTheme.bodySmall),
                      ],
                    ],
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          if (content.isNotEmpty)
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: SelectableText(
                  content,
                  style: theme.textTheme.bodyMedium?.copyWith(height: 1.8),
                ),
              ),
            ),
        ],
      ),
    );
  }
}
