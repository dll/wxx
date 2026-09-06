import 'package:flutter/material.dart';

import 'vopc_core_idea_card.dart';
import 'vopc_flow_strip.dart';
import 'vopc_quiz_card.dart';

class VopcLearningSheet extends StatelessWidget {
  final Map<String, dynamic> learning;
  final List<Map<String, dynamic>> defaultCards;
  final List<Map<String, String>> defaultSteps;
  const VopcLearningSheet(
      {super.key,
      required this.learning,
      required this.defaultCards,
      required this.defaultSteps});

  List<Map<String, dynamic>> _list(String key) {
    final raw = learning[key];
    if (raw is List)
      return raw
          .whereType<Map>()
          .map((e) => Map<String, dynamic>.from(e))
          .toList();
    return const [];
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cards = _list('knowledge_cards');
    final steps = _list('flow_steps');
    final quizzes = _list('quizzes');
    final allCards = cards.isEmpty ? defaultCards : cards;
    final allSteps = steps.isEmpty ? defaultSteps : steps;
    return DraggableScrollableSheet(
      expand: false,
      initialChildSize: .82,
      maxChildSize: .95,
      minChildSize: .5,
      builder: (context, controller) => ListView(
        controller: controller,
        padding: const EdgeInsets.fromLTRB(20, 16, 20, 32),
        children: [
          Center(
              child: Container(
                  width: 40,
                  height: 4,
                  decoration: BoxDecoration(
                      color: theme.colorScheme.outlineVariant,
                      borderRadius: BorderRadius.circular(2)))),
          const SizedBox(height: 12),
          Text('OPC 核心知识卡',
              style: theme.textTheme.titleLarge
                  ?.copyWith(fontWeight: FontWeight.w800)),
          const SizedBox(height: 4),
          Text('一人公司最小闭环：需求方 + 产品/服务 + 交付 + 反馈',
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: theme.colorScheme.outline)),
          const SizedBox(height: 14),
          for (final card in allCards) ...[
            VopcCoreIdeaCard(
                title: card['title']?.toString() ?? '',
                body: card['body']?.toString() ?? ''),
            const SizedBox(height: 10),
          ],
          const SizedBox(height: 8),
          Text('OPC 核心流程图',
              style: theme.textTheme.titleMedium
                  ?.copyWith(fontWeight: FontWeight.w700)),
          const SizedBox(height: 10),
          VopcFlowStrip(steps: allSteps),
          const SizedBox(height: 20),
          if (quizzes.isNotEmpty) ...[
            Text('自测小问卷',
                style: theme.textTheme.titleMedium
                    ?.copyWith(fontWeight: FontWeight.w700)),
            const SizedBox(height: 10),
            ...quizzes.map((quiz) => VopcQuizCard(
                question: quiz['q']?.toString() ?? '',
                options: (quiz['options'] as List?)
                        ?.map((e) => e.toString())
                        .toList() ??
                    const [],
                answer: (quiz['answer'] as num?)?.toInt())),
          ],
        ],
      ),
    );
  }
}
