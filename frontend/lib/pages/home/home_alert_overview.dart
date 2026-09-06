import 'package:flutter/material.dart';

class HomeAlertOverview extends StatelessWidget {
  final bool loading;
  final int urgent;
  final int high;
  final int pending;
  final VoidCallback onViewAll;

  const HomeAlertOverview({
    super.key,
    required this.loading,
    required this.urgent,
    required this.high,
    required this.pending,
    required this.onViewAll,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text('预警概览', style: theme.textTheme.titleMedium),
            TextButton.icon(
              onPressed: onViewAll,
              icon: const Icon(Icons.arrow_forward, size: 16),
              label: const Text('查看全部', style: TextStyle(fontSize: 13)),
            ),
          ],
        ),
        const SizedBox(height: 12),
        if (loading)
          const Center(child: CircularProgressIndicator())
        else
          Row(
            children: [
              _StatCard(
                  label: '紧急',
                  count: urgent,
                  color: const Color(0xFFC62828),
                  icon: Icons.warning_rounded),
              const SizedBox(width: 10),
              _StatCard(
                  label: '高风险',
                  count: high,
                  color: const Color(0xFFE65100),
                  icon: Icons.error_outline),
              const SizedBox(width: 10),
              _StatCard(
                  label: '待处理',
                  count: pending,
                  color: const Color(0xFF1565C0),
                  icon: Icons.pending_actions),
            ],
          ),
      ],
    );
  }
}

class _StatCard extends StatelessWidget {
  final String label;
  final int count;
  final Color color;
  final IconData icon;

  const _StatCard(
      {required this.label,
      required this.count,
      required this.color,
      required this.icon});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Expanded(
      child: Container(
        padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 12),
        decoration: BoxDecoration(
          color: color.withOpacity(0.08),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: color.withOpacity(0.2)),
        ),
        child: Column(
          children: [
            Icon(icon, color: color, size: 24),
            const SizedBox(height: 6),
            Text('$count',
                style: TextStyle(
                    fontSize: 28, fontWeight: FontWeight.bold, color: color)),
            Text(label,
                style: TextStyle(
                    fontSize: 12, color: theme.colorScheme.onSurfaceVariant)),
          ],
        ),
      ),
    );
  }
}
