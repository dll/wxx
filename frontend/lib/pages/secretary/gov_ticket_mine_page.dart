import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/secretary_provider.dart';
import '../../widgets/error_view.dart';

/// 我的督办工单（辅导员/教辅/党群责任人视角，D5-3「洞察→工单」治理回环）
///
/// 展示分派给本人的治理督办件（含补料督办），可推进状态：待办->处理中->已完成。
class GovTicketMinePage extends StatefulWidget {
  const GovTicketMinePage({super.key});

  @override
  State<GovTicketMinePage> createState() => _GovTicketMinePageState();
}

class _GovTicketMinePageState extends State<GovTicketMinePage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<SecretaryProvider>().fetchMyTickets();
    });
  }

  Future<void> _advance(SecretaryProvider p, GovTicket t) async {
    final target = switch (t.status) {
      'pending' => 'processing',
      'processing' => 'completed',
      _ => null,
    };
    if (target == null) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('该工单已完结')));
      return;
    }
    final result = await context
        .read<SecretaryProvider>()
        .updateTicketStatus(id: t.id, status: target, asManager: false);
    if (!mounted) return;
    ScaffoldMessenger.of(context)
        .showSnackBar(SnackBar(content: Text(result.ok ? '已推进' : result.msg)));
    if (result.ok) {
      await context.read<SecretaryProvider>().fetchMyTickets();
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('我的督办工单'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: () => context.read<SecretaryProvider>().fetchMyTickets(),
          ),
        ],
      ),
      body: Consumer<SecretaryProvider>(
        builder: (_, p, __) {
          if (p.ticketsLoading && p.myTickets.isEmpty) {
            return const Center(child: CircularProgressIndicator());
          }
          final list = p.myTickets.isEmpty ? p.tickets : p.myTickets;
          if (list.isEmpty) {
            return ErrorView.empty(message: '暂无分派给您的督办工单');
          }
          return RefreshIndicator(
            onRefresh: () => p.fetchMyTickets(),
            child: ListView(
              padding: const EdgeInsets.all(12),
              children: [
                for (final t in list)
                  Card(
                    margin: const EdgeInsets.symmetric(vertical: 4),
                    child: ListTile(
                      leading: _statusIcon(t.status),
                      title: Text(t.title,
                          style:
                              const TextStyle(fontWeight: FontWeight.bold)),
                      subtitle: Text(
                          '${t.categoryLabel} · ${t.statusLabel}\n${t.remark.isNotEmpty ? t.remark : t.sourceDesc}',
                          maxLines: 3,
                          overflow: TextOverflow.ellipsis),
                      trailing: (t.status == 'pending' ||
                              t.status == 'processing')
                          ? OutlinedButton(
                              onPressed: () => _advance(p, t),
                              style: OutlinedButton.styleFrom(
                                  minimumSize: const Size(0, 32)),
                              child: Text(
                                  t.status == 'pending' ? '开始处理' : '完成',
                                  style: const TextStyle(fontSize: 12)),
                            )
                          : null,
                    ),
                  ),
              ],
            ),
          );
        },
      ),
    );
  }

  Widget _statusIcon(String status) {
    return switch (status) {
      'pending' => const Icon(Icons.radio_button_unchecked,
          color: Colors.orange),
      'processing' => const Icon(Icons.hourglass_top, color: Colors.blue),
      'completed' => const Icon(Icons.check_circle, color: Colors.green),
      'closed' => const Icon(Icons.cancel, color: Colors.grey),
      _ => const Icon(Icons.circle_outlined, color: Colors.grey),
    };
  }
}
