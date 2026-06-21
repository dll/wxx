import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/guest_provider.dart';
import '../../widgets/error_view.dart';

/// 游客审核页面（college_admin 及以上可访问）
class AdminGuestReviewPage extends StatefulWidget {
  const AdminGuestReviewPage({super.key});

  @override
  State<AdminGuestReviewPage> createState() => _AdminGuestReviewPageState();
}

class _AdminGuestReviewPageState extends State<AdminGuestReviewPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<GuestProvider>().fetchPendingGuests();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('游客审核')),
      body: Consumer<GuestProvider>(
        builder: (_, provider, __) {
          if (provider.loading && provider.pendingGuests.isEmpty) {
            return const Center(child: CircularProgressIndicator());
          }
          if (provider.error.isNotEmpty && provider.pendingGuests.isEmpty) {
            return ErrorView.error(
              message: provider.error,
              onRetry: () => provider.fetchPendingGuests(),
            );
          }
          if (provider.pendingGuests.isEmpty) {
            return ErrorView.empty(
              message: '暂无待审核游客',
              icon: Icons.person_outline,
            );
          }
          return RefreshIndicator(
            onRefresh: () => provider.fetchPendingGuests(),
            child: ListView.builder(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              itemCount: provider.pendingGuests.length,
              itemBuilder: (context, index) {
                final guest = provider.pendingGuests[index];
                return _GuestTile(guest: guest);
              },
            ),
          );
        },
      ),
    );
  }
}

class _GuestTile extends StatelessWidget {
  final dynamic guest;
  const _GuestTile({required this.guest});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            CircleAvatar(
              backgroundColor: theme.colorScheme.tertiaryContainer,
              child: Text(
                guest.displayName.isNotEmpty
                    ? guest.displayName[0].toUpperCase()
                    : '?',
                style: TextStyle(color: theme.colorScheme.onTertiaryContainer),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(guest.displayName,
                      style: theme.textTheme.titleSmall),
                  const SizedBox(height: 4),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                    decoration: BoxDecoration(
                      color: Colors.orange.withValues(alpha: 0.1),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: const Text('待审核',
                        style: TextStyle(fontSize: 11, color: Colors.orange)),
                  ),
                ],
              ),
            ),
            FilledButton.tonalIcon(
              onPressed: () => _handleApprove(context),
              icon: const Icon(Icons.check, size: 16),
              label: const Text('通过'),
              style: FilledButton.styleFrom(
                foregroundColor: Colors.green,
              ),
            ),
            const SizedBox(width: 8),
            OutlinedButton.icon(
              onPressed: () => _handleReject(context),
              icon: const Icon(Icons.close, size: 16),
              label: const Text('拒绝'),
              style: OutlinedButton.styleFrom(
                foregroundColor: Colors.red,
                side: const BorderSide(color: Colors.red),
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _handleApprove(BuildContext context) {
    final studentIdCtrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('确认通过'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('确认通过「${guest.displayName}」的游客审核？'),
            const SizedBox(height: 16),
            TextField(
              controller: studentIdCtrl,
              decoration: const InputDecoration(
                labelText: '分配学号',
                hintText: '输入学号后该游客升级为学生',
                border: OutlineInputBorder(),
                isDense: true,
              ),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(
            onPressed: () async {
              final ok = await context.read<GuestProvider>().approveGuest(
                guest.id.toString(),
                studentId: studentIdCtrl.text.trim(),
              );
              if (ctx.mounted) {
                Navigator.pop(ctx);
                if (ok) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('审核通过')),
                  );
                }
              }
            },
            child: const Text('确认通过'),
          ),
        ],
      ),
    );
  }

  void _handleReject(BuildContext context) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('确认拒绝'),
        content: Text('确认拒绝「${guest.displayName}」的游客申请？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(
            onPressed: () async {
              final ok = await context.read<GuestProvider>().rejectGuest(guest.id.toString());
              if (ctx.mounted) {
                Navigator.pop(ctx);
                if (ok) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('已拒绝')),
                  );
                }
              }
            },
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            child: const Text('确认拒绝'),
          ),
        ],
      ),
    );
  }
}
