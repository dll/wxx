import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../providers/chat_provider.dart';

class ChatAgentExampleGroup {
  final IconData icon;
  final Color color;
  final String name;
  final String agentType;
  final List<String> questions;
  const ChatAgentExampleGroup(
      {required this.icon,
      required this.color,
      required this.name,
      required this.agentType,
      required this.questions});
}

class ChatAgentExamples extends StatelessWidget {
  final List<ChatAgentExampleGroup> groups;
  final TextEditingController inputController;
  final VoidCallback send;
  final String? Function(String type) agentIdForType;

  const ChatAgentExamples(
      {super.key,
      required this.groups,
      required this.inputController,
      required this.send,
      required this.agentIdForType});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final chat = context.read<ChatProvider>();
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      for (final group in groups) ...[
        Row(children: [
          Icon(group.icon, size: 16, color: group.color),
          const SizedBox(width: 6),
          Text(group.name,
              style: theme.textTheme.labelLarge
                  ?.copyWith(color: group.color, fontWeight: FontWeight.w600)),
          const SizedBox(width: 8),
          InkWell(
            borderRadius: BorderRadius.circular(8),
            onTap: () => chat.selectAgent(agentIdForType(group.agentType)),
            child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                decoration: BoxDecoration(
                    color: group.color.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(6)),
                child: Text('用「${group.name}」回答 →',
                    style: TextStyle(
                        fontSize: 11,
                        color: group.color,
                        fontWeight: FontWeight.w600))),
          ),
        ]),
        const SizedBox(height: 6),
        Wrap(
            spacing: 8,
            runSpacing: 8,
            children: group.questions
                .map((question) => ActionChip(
                    label: Text(question, style: const TextStyle(fontSize: 12)),
                    onPressed: () {
                      chat.selectAgent(agentIdForType(group.agentType));
                      inputController.text = question;
                      send();
                    }))
                .toList()),
        const SizedBox(height: 12),
      ],
    ]);
  }
}
