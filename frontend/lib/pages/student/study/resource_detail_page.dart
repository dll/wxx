import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../../providers/education_provider.dart';
import '../../../widgets/error_view.dart';

class ResourceDetailPage extends StatefulWidget {
  final String resourceId;
  const ResourceDetailPage({super.key, required this.resourceId});

  @override
  State<ResourceDetailPage> createState() => _ResourceDetailPageState();
}

class _ResourceDetailPageState extends State<ResourceDetailPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudyProvider>().fetchResourceDetail(widget.resourceId);
    });
  }

  @override
  void dispose() {
    context.read<StudyProvider>().clearDetail();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudyProvider>();
    final resource = provider.resourceDetail;

    return Scaffold(
      appBar: AppBar(title: const Text('资源详情')),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty && resource == null
              ? ErrorView.error(
                  message: provider.error,
                  onRetry: () => provider.fetchResourceDetail(widget.resourceId),
                )
              : resource == null
                  ? ErrorView.empty(
                      message: '资源不存在',
                      icon: Icons.folder_outlined,
                    )
                  : _buildDetail(theme, resource),
    );
  }

  Widget _buildDetail(ThemeData theme, Map<String, dynamic> resource) {
    final title = resource['title'] as String? ?? resource['name'] as String? ?? '';
    final type = resource['type'] as String? ?? '';
    final size = resource['size'] as String? ?? '';
    final description = resource['description'] as String? ?? resource['intro'] as String? ?? '';
    final author = resource['author'] as String? ?? resource['uploader'] as String? ?? '';
    final uploadTime = resource['upload_time'] as String? ?? resource['date'] as String? ?? '';
    final downloads = resource['downloads'] as int? ?? 0;

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Row(
                children: [
                  Container(
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      color: theme.colorScheme.primary.withOpacity(0.1),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Icon(
                      _getResourceIcon(type),
                      size: 40,
                      color: theme.colorScheme.primary,
                    ),
                  ),
                  const SizedBox(width: 16),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          title,
                          style: theme.textTheme.titleLarge?.copyWith(
                            fontWeight: FontWeight.bold,
                          ),
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),
                        const SizedBox(height: 8),
                        Row(
                          children: [
                            if (type.isNotEmpty) ...[
                              Text(type, style: theme.textTheme.bodySmall),
                              const SizedBox(width: 12),
                            ],
                            if (size.isNotEmpty) ...[
                              Text(size, style: theme.textTheme.bodySmall),
                              const SizedBox(width: 12),
                            ],
                            Icon(Icons.download_outlined, size: 14, color: theme.colorScheme.onSurfaceVariant),
                            const SizedBox(width: 2),
                            Text('$downloads', style: theme.textTheme.bodySmall),
                          ],
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          if (author.isNotEmpty || uploadTime.isNotEmpty) ...[
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(
                  children: [
                    if (author.isNotEmpty) ...[
                      Icon(Icons.person_outline, size: 18, color: theme.colorScheme.onSurfaceVariant),
                      const SizedBox(width: 6),
                      Expanded(child: Text(author, style: theme.textTheme.bodyMedium)),
                    ],
                    if (uploadTime.isNotEmpty) ...[
                      Icon(Icons.event_outlined, size: 18, color: theme.colorScheme.onSurfaceVariant),
                      const SizedBox(width: 6),
                      Text(uploadTime, style: theme.textTheme.bodyMedium),
                    ],
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),
          ],
          if (description.isNotEmpty) ...[
            Text('资源简介', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
            const SizedBox(height: 12),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Text(description, style: theme.textTheme.bodyMedium),
              ),
            ),
            const SizedBox(height: 24),
          ],
          SizedBox(
            width: double.infinity,
            child: FilledButton.icon(
              onPressed: () {
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('下载功能即将开放')),
                );
              },
              icon: const Icon(Icons.download, size: 18),
              label: const Text('下载资源'),
              style: FilledButton.styleFrom(
                padding: const EdgeInsets.symmetric(vertical: 14),
              ),
            ),
          ),
        ],
      ),
    );
  }

  IconData _getResourceIcon(String type) {
    switch (type.toLowerCase()) {
      case 'pdf':
        return Icons.picture_as_pdf;
      case 'doc':
      case 'docx':
      case '文档':
        return Icons.description;
      case 'ppt':
      case 'pptx':
      case '课件':
        return Icons.slideshow;
      case '视频':
      case 'video':
        return Icons.play_circle_outline;
      default:
        return Icons.folder_outlined;
    }
  }
}
