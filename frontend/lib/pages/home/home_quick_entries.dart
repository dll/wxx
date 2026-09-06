import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'home_quick_entry_card.dart';

class HomeQuickEntries extends StatelessWidget {
  final List<Map<String, dynamic>> entries;
  const HomeQuickEntries({super.key, required this.entries});

  static const iconMap = <String, IconData>{
    'chat': Icons.chat_bubble_outline,
    'study_plan': Icons.fact_check_outlined,
    'timetable': Icons.calendar_month_outlined,
    'career': Icons.work_outline,
    'study': Icons.menu_book_outlined,
    'mental': Icons.favorite_outline,
    'agenda': Icons.checklist
  };
  static const colorMap = <String, Color>{
    'chat': Color(0xFF1565C0),
    'study_plan': Color(0xFF2E7D32),
    'timetable': Color(0xFF00695C),
    'career': Color(0xFFE65100),
    'study': Color(0xFF7B1FA2),
    'mental': Color(0xFFC62828),
    'agenda': Color(0xFF00695C)
  };

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Text('功能入口', style: theme.textTheme.titleMedium),
      const SizedBox(height: 12),
      GridView.builder(
        shrinkWrap: true,
        physics: const NeverScrollableScrollPhysics(),
        gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount: 3,
            mainAxisSpacing: 10,
            crossAxisSpacing: 10,
            childAspectRatio: 1.1),
        itemCount: entries.length,
        itemBuilder: (context, index) {
          final entry = entries[index];
          final key = entry['icon'] as String? ?? 'chat';
          return HomeQuickEntryCard(
              icon: iconMap[key] ?? Icons.widgets_outlined,
              label: entry['title'] ?? '',
              color: colorMap[key] ?? theme.colorScheme.primary,
              onTap: () => context.go(entry['route'] ?? '/'));
        },
      ),
    ]);
  }
}
