import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/secretary_provider.dart';
import '../../widgets/error_view.dart';
import '../../widgets/secretary_dashboard_sections.dart';

/// 书记党建育人专项可视化页（D1-1 功能补齐，2026-08-16）
///
/// 承载「党建育人（思想政治）」专项可视化，数据取自
/// `SecretaryProvider.partyDashboard`（复用既有 party-dashboard 端点聚合，
/// 不新造数据、不改后端聚合逻辑）。
///
/// 仅做该区域数据的独立容器与导航入口，渲染复用
/// [PartyDashboardSection]，沿用 data_source 三态诚实边界（DataSrcBadge），
/// 绝不伪造数值。为「轻量专项页」，不加载教育成果大屏其余区块。
class PartyDashboardPage extends StatefulWidget {
  const PartyDashboardPage({super.key});

  @override
  State<PartyDashboardPage> createState() => _PartyDashboardPageState();
}

class _PartyDashboardPageState extends State<PartyDashboardPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<SecretaryProvider>().fetchPartyDashboard();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('党建育人专项')),
      body: Consumer<SecretaryProvider>(
        builder: (_, provider, __) {
          if (provider.partyDashboardLoading && provider.partyDashboard == null) {
            return const Center(child: CircularProgressIndicator());
          }
          if (provider.error.isNotEmpty && provider.partyDashboard == null) {
            return ErrorView.error(
              message: provider.error,
              onRetry: () => provider.fetchPartyDashboard(),
            );
          }
          final party = provider.partyDashboard;
          if (party == null) {
            return ErrorView.empty(message: '暂无党建育人数据（数据待充实）');
          }
          return RefreshIndicator(
            onRefresh: () => provider.fetchPartyDashboard(),
            child: ListView(
              padding: const EdgeInsets.all(16),
              children: [
                const _PartyHeader(),
                const SizedBox(height: 12),
                PartyDashboardSection(party: party),
              ],
            ),
          );
        },
      ),
    );
  }
}

/// 党建育人专项页头：说明专项可视化定位。
class _PartyHeader extends StatelessWidget {
  const _PartyHeader();

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          children: [
            Icon(Icons.flag, color: Colors.indigo.shade700),
            const SizedBox(width: 8),
            const Expanded(
              child: Text('党建育人（思想政治）专项 · 书记视角',
                  style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
            ),
            const Icon(Icons.auto_graph, color: Colors.indigo),
          ],
        ),
      ),
    );
  }
}
