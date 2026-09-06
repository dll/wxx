import 'package:flutter/material.dart';

class HomeCampusService extends StatelessWidget {
  final VoidCallback onMap;
  final VoidCallback onVr;
  final VoidCallback onCollege;
  final VoidCallback onSchool;

  const HomeCampusService({
    super.key,
    required this.onMap,
    required this.onVr,
    required this.onCollege,
    required this.onSchool,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(children: [
          Icon(Icons.location_city, color: theme.colorScheme.primary),
          const SizedBox(width: 8),
          Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text('校园服务',
                style: theme.textTheme.titleMedium
                    ?.copyWith(fontWeight: FontWeight.w700)),
            Text('导航 · 全景 · 学院入口', style: theme.textTheme.bodySmall),
          ]),
        ]),
        const SizedBox(height: 14),
        Row(children: [
          _Entry(
              icon: Icons.map_outlined,
              label: '校园导航',
              color: const Color(0xFF1677FF),
              onTap: onMap),
          const SizedBox(width: 10),
          _Entry(
              icon: Icons.view_in_ar,
              label: 'VR全景',
              color: const Color(0xFF7B1FA2),
              onTap: onVr),
          const SizedBox(width: 10),
          _Entry(
              icon: Icons.computer,
              label: '计算机学院',
              color: const Color(0xFF2E7D32),
              onTap: onCollege),
          const SizedBox(width: 10),
          _Entry(
              icon: Icons.school,
              label: '学校首页',
              color: const Color(0xFF1565C0),
              onTap: onSchool),
        ]),
      ],
    );
  }
}

class _Entry extends StatelessWidget {
  final IconData icon;
  final String label;
  final Color color;
  final VoidCallback onTap;

  const _Entry(
      {required this.icon,
      required this.label,
      required this.color,
      required this.onTap});

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: Material(
        color: color.withOpacity(.07),
        borderRadius: BorderRadius.circular(14),
        child: InkWell(
          onTap: onTap,
          borderRadius: BorderRadius.circular(14),
          child: Container(
            padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 6),
            decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(14),
                border: Border.all(color: color.withOpacity(.12))),
            child: Column(children: [
              Icon(icon, color: color, size: 24),
              const SizedBox(height: 8),
              Text(label,
                  textAlign: TextAlign.center,
                  style: TextStyle(
                      fontSize: 13,
                      fontWeight: FontWeight.w600,
                      color: Theme.of(context).colorScheme.onSurface)),
            ]),
          ),
        ),
      ),
    );
  }
}
