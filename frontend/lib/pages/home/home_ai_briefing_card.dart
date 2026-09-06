import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../providers/ai_briefing_provider.dart';

class HomeAIBriefingCard extends StatelessWidget {
  const HomeAIBriefingCard({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Consumer<AIBriefingProvider>(
      builder: (context, provider, _) {
        if (!provider.userLoaded && !provider.userLoading) {
          Future.microtask(() {
            if (context.mounted) provider.fetchUserBriefings();
          });
        }
        final latest = provider.userBriefings.take(3).toList();
        return Material(
          color: theme.colorScheme.primaryContainer.withOpacity(0.5),
          borderRadius: BorderRadius.circular(16),
          child: InkWell(
            borderRadius: BorderRadius.circular(16),
            onTap: () => context.go('/ai-briefings'),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Container(
                        padding: const EdgeInsets.all(8),
                        decoration: BoxDecoration(
                          color: theme.colorScheme.primary,
                          borderRadius: BorderRadius.circular(10),
                        ),
                        child: const Icon(Icons.newspaper,
                            color: Colors.white, size: 20),
                      ),
                      const SizedBox(width: 10),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text('AI 简讯',
                                style: theme.textTheme.titleMedium
                                    ?.copyWith(fontWeight: FontWeight.w700)),
                            Text('AI 教学 / 工具 / 版本 / 行业热点',
                                style: theme.textTheme.bodySmall),
                          ],
                        ),
                      ),
                      const Icon(Icons.chevron_right),
                    ],
                  ),
                  if (latest.isNotEmpty) ...[
                    const SizedBox(height: 10),
                    ...latest.map((b) => Padding(
                          padding: const EdgeInsets.only(bottom: 4),
                          child: Row(
                            children: [
                              const Icon(Icons.circle, size: 6),
                              const SizedBox(width: 6),
                              Expanded(
                                child: Text(b.topic,
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                    style: theme.textTheme.bodyMedium),
                              ),
                            ],
                          ),
                        )),
                  ],
                ],
              ),
            ),
          ),
        );
      },
    );
  }
}
