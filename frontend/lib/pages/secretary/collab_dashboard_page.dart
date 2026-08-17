import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/secretary_provider.dart';
import '../../widgets/error_view.dart';
import '../../widgets/secretary_dashboard_sections.dart';

/// 书记协同育人总览专项可视化页（D1-1 功能补齐，2026-08-16）
///
/// 承载「协同育人总览（教师/教辅付出）」专项可视化，数据取自
/// `SecretaryProvider.collabDashboard`（复用既有 collab-dashboard 端点聚合，
/// 不新造数据、不改后端聚合逻辑）。
///
/// 仅做该区域数据的独立容器与导航入口，渲染复用
/// [CollabDashboardSection]，沿用 data_source 三态诚实边界（DataSrcBadge），
/// 绝不伪造数值。为「轻量专项页」，不加载教育成果大屏其余区块。
class CollabDashboardPage extends StatefulWidget {
  const CollabDashboardPage({super.key});

  @override
  State<CollabDashboardPage> createState() => _CollabDashboardPageState();
}

class _CollabDashboardPageState extends State<CollabDashboardPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<SecretaryProvider>().fetchCollabDashboard();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('协同育人总览')),
      body: Consumer<SecretaryProvider>(
        builder: (_, provider, __) {
          if (provider.collabDashboardLoading && provider.collabDashboard == null) {
            return const Center(child: CircularProgressIndicator());
          }
          if (provider.error.isNotEmpty && provider.collabDashboard == null) {
            return ErrorView.error(
              message: provider.error,
              onRetry: () => provider.fetchCollabDashboard(),
            );
          }
          final collab = provider.collabDashboard;
          if (collab == null) {
            return ErrorView.empty(message: '暂无协同育人数据（数据待充实）');
          }
          return RefreshIndicator(
            onRefresh: () => provider.fetchCollabDashboard(),
            child: ListView(
              padding: const EdgeInsets.all(16),
              children: [
                const _CollabHeader(),
                const SizedBox(height: 12),
                CollabDashboardSection(collab: collab),
              ],
            ),
          );
        },
      ),
    );
  }
}

/// 协同育人总览专项页头：说明专项可视化定位。
class _CollabHeader extends StatelessWidget {
  const _CollabHeader();

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          children: [
            Icon(Icons.groups, color: Colors.indigo.shade700),
            const SizedBox(width: 8),
            const Expanded(
              child: Text('协同育人总览专项 · 书记视角',
                  style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
            ),
            const Icon(Icons.auto_graph, color: Colors.indigo),
          ],
        ),
      ),
    );
  }
}
