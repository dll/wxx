import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../../providers/education_provider.dart';
import '../../../widgets/error_view.dart';

class ArticleDetailPage extends StatefulWidget {
  final String articleId;
  const ArticleDetailPage({super.key, required this.articleId});

  @override
  State<ArticleDetailPage> createState() => _ArticleDetailPageState();
}

class _ArticleDetailPageState extends State<ArticleDetailPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<MentalProvider>().fetchArticleDetail(widget.articleId);
    });
  }

  @override
  void dispose() {
    context.read<MentalProvider>().clearDetail();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<MentalProvider>();
    final article = provider.articleDetail;

    return Scaffold(
      appBar: AppBar(title: const Text('科普文章')),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty && article == null
              ? ErrorView.error(
                  message: provider.error,
                  onRetry: () => provider.fetchArticleDetail(widget.articleId),
                )
              : article == null
                  ? ErrorView.empty(
                      message: '文章不存在',
                      icon: Icons.article_outlined,
                    )
                  : _buildDetail(theme, article),
    );
  }

  Widget _buildDetail(ThemeData theme, Map<String, dynamic> article) {
    final title = article['title'] as String? ?? '';
    final author = article['author'] as String? ?? '';
    final date = article['date'] as String? ?? article['publish_date'] as String? ?? '';
    final content = article['content'] as String? ?? article['body'] as String? ?? '';
    final category = article['category'] as String? ?? '';
    final views = article['views'] as int? ?? article['read_count'] as int? ?? 0;

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (category.isNotEmpty) ...[
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
              decoration: BoxDecoration(
                color: theme.colorScheme.tertiary.withOpacity(0.12),
                borderRadius: BorderRadius.circular(6),
              ),
              child: Text(
                category,
                style: theme.textTheme.labelMedium?.copyWith(
                  color: theme.colorScheme.tertiary,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
            const SizedBox(height: 12),
          ],
          Text(
            title,
            style: theme.textTheme.headlineSmall?.copyWith(
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              if (author.isNotEmpty) ...[
                Icon(Icons.person_outline, size: 16, color: theme.colorScheme.onSurfaceVariant),
                const SizedBox(width: 4),
                Text(author, style: theme.textTheme.bodySmall),
                const SizedBox(width: 16),
              ],
              if (date.isNotEmpty) ...[
                Icon(Icons.event_outlined, size: 16, color: theme.colorScheme.onSurfaceVariant),
                const SizedBox(width: 4),
                Text(date, style: theme.textTheme.bodySmall),
                const SizedBox(width: 16),
              ],
              Icon(Icons.visibility_outlined, size: 16, color: theme.colorScheme.onSurfaceVariant),
              const SizedBox(width: 4),
              Text('$views', style: theme.textTheme.bodySmall),
            ],
          ),
          const SizedBox(height: 20),
          const Divider(),
          const SizedBox(height: 16),
          SelectableText(
            content,
            style: theme.textTheme.bodyLarge?.copyWith(height: 1.8),
          ),
        ],
      ),
    );
  }
}
