import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../providers/vopc_provider.dart';
import 'vopc_core_idea_card.dart';
import 'vopc_flow_strip.dart';
import 'vopc_learning_sheet.dart';

class VopcIntroSection extends StatelessWidget {
  final String defaultCoreIdea;
  final List<Map<String, String>> defaultFlowSteps;
  final List<Map<String, String>> defaultCards;

  const VopcIntroSection({
    super.key,
    required this.defaultCoreIdea,
    required this.defaultFlowSteps,
    required this.defaultCards,
  });

  List<Map<String, dynamic>> _learningList(
      Map<String, dynamic> learning, String key) {
    final raw = learning[key];
    if (raw is List) {
      return raw
          .whereType<Map>()
          .map((e) => Map<String, dynamic>.from(e))
          .toList();
    }
    return const [];
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final learning = context.watch<VopcProvider>().learning;
    final cards = _learningList(learning, 'knowledge_cards');
    final steps = _learningList(learning, 'flow_steps');
    return Card(
      margin: EdgeInsets.zero,
      elevation: 0,
      shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
          side: BorderSide(
              color: theme.colorScheme.outlineVariant.withOpacity(.55))),
      child: Padding(
        padding: const EdgeInsets.all(18),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Container(
              width: 34,
              height: 34,
              decoration: BoxDecoration(
                  color: theme.colorScheme.primary.withOpacity(.1),
                  borderRadius: BorderRadius.circular(10)),
              child: Icon(Icons.school_outlined,
                  size: 19, color: theme.colorScheme.primary),
            ),
            const SizedBox(width: 10),
            Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text('OPC 入门 · L1 概念层',
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontWeight: FontWeight.w700)),
              Text('先理解核心思想，再进入项目流程',
                  style: theme.textTheme.labelSmall
                      ?.copyWith(color: theme.colorScheme.outline)),
            ]),
          ]),
          const SizedBox(height: 14),
          if (cards.isNotEmpty)
            VopcCoreIdeaCard(
                title: cards.first['title'] ?? 'OPC 核心思想',
                body: cards.first['body'] ?? defaultCoreIdea),
          const SizedBox(height: 14),
          if (steps.isNotEmpty)
            VopcFlowStrip(steps: steps)
          else
            VopcFlowStrip(steps: defaultFlowSteps),
          const SizedBox(height: 14),
          FilledButton.tonalIcon(
              onPressed: () => _openLearning(context),
              icon: const Icon(Icons.menu_book_outlined),
              label: const Text('进入 OPC 学习')),
        ]),
      ),
    );
  }

  void _openLearning(BuildContext context) {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (sheetContext) => VopcLearningSheet(
        learning: sheetContext.read<VopcProvider>().learning,
        defaultCards: defaultCards,
        defaultSteps: defaultFlowSteps,
      ),
    );
  }
}
