import 'package:flutter/material.dart';
import 'home_empty_card.dart';
import 'home_event_item.dart';

class HomeUpcomingEvents extends StatelessWidget {
  final List<Map<String, dynamic>> events;
  const HomeUpcomingEvents({super.key, required this.events});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Text('近期提醒', style: theme.textTheme.titleMedium),
      const SizedBox(height: 8),
      if (events.isEmpty)
        const HomeEmptyCard(
            message: '近期没有重要事件', icon: Icons.event_available_outlined)
      else
        Container(
          decoration: BoxDecoration(
              color: theme.colorScheme.surface,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                  color: theme.colorScheme.outlineVariant.withOpacity(.5))),
          child: Column(children: [
            for (var i = 0; i < events.length; i++) ...[
              HomeEventItem(event: events[i]),
              if (i < events.length - 1)
                Divider(
                    height: 1,
                    indent: 16,
                    endIndent: 16,
                    color: theme.colorScheme.outlineVariant.withOpacity(.3)),
            ],
          ]),
        ),
    ]);
  }
}
