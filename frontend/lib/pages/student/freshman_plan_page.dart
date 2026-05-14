import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../widgets/error_view.dart';

/// 入学到就业四阶段规划路线图
class FreshmanPlanPage extends StatefulWidget {
  const FreshmanPlanPage({super.key});

  @override
  State<FreshmanPlanPage> createState() => _FreshmanPlanPageState();
}

class _FreshmanPlanPageState extends State<FreshmanPlanPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentFeatureProvider>().fetchDigitalTwin();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();

    return Scaffold(
      appBar: AppBar(title: const Text('新生大学规划')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchDigitalTwin(),
        child: provider.loading
            ? const Center(child: CircularProgressIndicator())
            : ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  _buildHeader(theme),
                  const SizedBox(height: 16),
                  _buildContent(theme, provider),
                ],
              ),
      ),
    );
  }

  Widget _buildHeader(ThemeData theme) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: [theme.colorScheme.primary, theme.colorScheme.primary.withValues(alpha: 0.7)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Row(
        children: [
          Icon(Icons.school, color: theme.colorScheme.onPrimary, size: 32),
          const SizedBox(width: 16),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('新生大学规划', style: TextStyle(color: theme.colorScheme.onPrimary, fontSize: 18, fontWeight: FontWeight.bold)),
                const SizedBox(height: 4),
                Text('入学到就业四阶段规划路线图', style: TextStyle(color: theme.colorScheme.onPrimary.withValues(alpha: 0.8), fontSize: 13)),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildContent(ThemeData theme, StudentFeatureProvider provider) {
    if (provider.error.isNotEmpty) {
      return ErrorView.error(message: provider.error, onRetry: () => provider.fetchDigitalTwin());
    }
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12), side: BorderSide(color: theme.colorScheme.outlineVariant)),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          children: [
            Icon(Icons.school, size: 48, color: theme.colorScheme.primary.withValues(alpha: 0.5)),
            const SizedBox(height: 12),
            Text('功能开发中', style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            Text('入学到就业四阶段规划路线图', style: theme.textTheme.bodyMedium?.copyWith(color: theme.colorScheme.onSurfaceVariant), textAlign: TextAlign.center),
          ],
        ),
      ),
    );
  }
}
