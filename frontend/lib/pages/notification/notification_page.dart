import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/notification_provider.dart';
import '../../utils/capability_utils.dart';
import '../../widgets/error_view.dart';

/// 通知页面
class NotificationPage extends StatefulWidget {
  const NotificationPage({super.key});

  @override
  State<NotificationPage> createState() => _NotificationPageState();
}

class _NotificationPageState extends State<NotificationPage>
    with SingleTickerProviderStateMixin {
  late TabController _tabController;

  final List<_TabItem> _tabs = const [
    _TabItem('全部', ''),
    _TabItem('系统', 'system'),
    _TabItem('反馈', 'feedback'),
    _TabItem('知识', 'knowledge'),
    _TabItem('活动', 'activity'),
  ];

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: _tabs.length, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<NotificationProvider>().fetchNotifications();
    });
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  /// 获取通知类型对应的图标和颜色
  _TypeInfo _getTypeInfo(String type) {
    switch (type) {
      case 'system':
        return const _TypeInfo(Icons.notifications, Colors.blue);
      case 'feedback':
        return const _TypeInfo(Icons.feedback, Colors.orange);
      case 'knowledge':
        return const _TypeInfo(Icons.menu_book, Colors.green);
      case 'activity':
        return const _TypeInfo(Icons.event, Colors.purple);
      case 'career':
        return const _TypeInfo(Icons.work, Colors.brown);
      default:
        return const _TypeInfo(Icons.notifications_outlined, Colors.grey);
    }
  }

  /// 格式化时间
  String _formatTime(String createdAt) {
    try {
      final dt = DateTime.parse(createdAt);
      final now = DateTime.now();
      final diff = now.difference(dt);
      if (diff.inMinutes < 1) {
        return '刚刚';
      } else if (diff.inHours < 1) {
        return '${diff.inMinutes}分钟前';
      } else if (diff.inDays < 1) {
        return '${diff.inHours}小时前';
      } else if (diff.inDays < 7) {
        return '${diff.inDays}天前';
      } else {
        return '${dt.month}月${dt.day}日';
      }
    } catch (e) {
      return createdAt;
    }
  }

  /// 处理通知点击
  void _onNotificationTap(NotificationItem item) async {
    // 标记已读
    if (item.isRead == 0) {
      await context.read<NotificationProvider>().markAsRead(item.id);
    }
    // 跳转到关联页面
    if (item.relatedType.isNotEmpty && item.relatedId > 0) {
      // TODO: 根据 related_type 跳转到不同页面
      // 例如：feedback -> 反馈详情，resource -> 知识详情等
    }
  }

  /// 全部已读
  Future<void> _markAllAsRead() async {
    final success = await context.read<NotificationProvider>().markAllAsRead();
    if (success && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('已全部标记为已读')),
      );
    }
  }

  /// 管理员发送系统通知
  Future<void> _showSendDialog(BuildContext context) async {
    final titleCtrl = TextEditingController();
    final contentCtrl = TextEditingController();
    await showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('发送系统通知'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: titleCtrl,
              decoration: const InputDecoration(
                  labelText: '通知标题', hintText: '如：新生报到须知'),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: contentCtrl,
              maxLines: 4,
              decoration: const InputDecoration(
                  labelText: '通知内容', hintText: '通知详情...'),
            ),
          ],
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(
            onPressed: () async {
              final title = titleCtrl.text.trim();
              final content = contentCtrl.text.trim();
              if (title.isEmpty || content.isEmpty) return;
              Navigator.pop(ctx);
              final ok = await context
                  .read<NotificationProvider>()
                  .sendSystemNotification(title, content);
              if (context.mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text(ok ? '通知已发送' : '发送失败')),
                );
              }
            },
            child: const Text('发送'),
          ),
        ],
      ),
    );
  }

  /// 管理员清空全部通知
  Future<void> _confirmClear(BuildContext context) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('清空通知'),
        content: const Text('确定清空全部通知吗？此操作不可恢复。'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('取消')),
          TextButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('清空')),
        ],
      ),
    );
    if (ok != true) return;
    final success =
        await context.read<NotificationProvider>().clearAllNotifications();
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(success ? '通知已清空' : '清空失败')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<NotificationProvider>();

    return Scaffold(
      appBar: AppBar(
        title: const Text('通知中心'),
        actions: [
          if (CapabilityUtils.has(Capability.systemSettingsWrite)) ...[
            IconButton(
              tooltip: '发送通知',
              icon: const Icon(Icons.send_outlined),
              onPressed: () => _showSendDialog(context),
            ),
            IconButton(
              tooltip: '清空通知',
              icon: const Icon(Icons.delete_sweep_outlined),
              onPressed: () => _confirmClear(context),
            ),
          ],
          if (provider.unreadCount > 0)
            TextButton.icon(
              onPressed: _markAllAsRead,
              icon: const Icon(Icons.done_all, size: 20),
              label: const Text('全部已读'),
            ),
        ],
        bottom: TabBar(
          controller: _tabController,
          isScrollable: true,
          tabs: _tabs.map((tab) => Tab(text: tab.label)).toList(),
          onTap: (index) {
            provider.fetchNotifications(type: _tabs[index].type);
          },
        ),
      ),
      body: TabBarView(
        controller: _tabController,
        children: _tabs.map((tab) {
          return _buildNotificationList(theme, provider);
        }).toList(),
      ),
    );
  }

  Widget _buildNotificationList(ThemeData theme, NotificationProvider provider) {
    if (provider.loading && provider.items.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    if (provider.error.isNotEmpty && provider.items.isEmpty) {
      return ErrorView.error(
        message: provider.error,
        onRetry: () => provider.refresh(),
      );
    }
    if (provider.items.isEmpty) {
      return ErrorView.empty(
        message: '暂无通知',
        subtitle: '有新通知时会在这里显示',
        icon: Icons.notifications_none,
      );
    }

    return RefreshIndicator(
      onRefresh: () => provider.refresh(),
      child: ListView.builder(
        padding: const EdgeInsets.symmetric(vertical: 8),
        itemCount: provider.items.length + 1,
        itemBuilder: (context, index) {
          if (index == provider.items.length) {
            // 加载更多
            if (provider.items.length < provider.total && !provider.loading) {
              return Padding(
                padding: const EdgeInsets.all(16),
                child: Center(
                  child: TextButton(
                    onPressed: () => provider.loadMore(),
                    child: const Text('加载更多'),
                  ),
                ),
              );
            } else if (provider.loading) {
              return const Padding(
                padding: EdgeInsets.all(16),
                child: Center(child: CircularProgressIndicator()),
              );
            } else {
              return const SizedBox.shrink();
            }
          }

          final item = provider.items[index];
          return _buildNotificationItem(theme, item);
        },
      ),
    );
  }

  Widget _buildNotificationItem(ThemeData theme, NotificationItem item) {
    final typeInfo = _getTypeInfo(item.type);
    final isUnread = item.isRead == 0;

    return InkWell(
      onTap: () => _onNotificationTap(item),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          border: Border(
            bottom: BorderSide(
              color: theme.colorScheme.outlineVariant.withOpacity(0.3),
            ),
          ),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // 未读小圆点
            Container(
              width: 8,
              height: 8,
              margin: const EdgeInsets.only(top: 8, right: 8),
              decoration: BoxDecoration(
                color: isUnread ? Colors.blue : Colors.transparent,
                shape: BoxShape.circle,
              ),
            ),
            // 类型图标
            Container(
              width: 40,
              height: 40,
              margin: const EdgeInsets.only(right: 12),
              decoration: BoxDecoration(
                color: typeInfo.color.withOpacity(0.1),
                borderRadius: BorderRadius.circular(10),
              ),
              child: Icon(typeInfo.icon, color: typeInfo.color, size: 22),
            ),
            // 内容
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          item.title,
                          style: theme.textTheme.titleMedium?.copyWith(
                            fontWeight: isUnread ? FontWeight.bold : FontWeight.normal,
                            color: isUnread
                                ? theme.colorScheme.onSurface
                                : theme.colorScheme.onSurface.withOpacity(0.6),
                          ),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      const SizedBox(width: 8),
                      Text(
                        _formatTime(item.createdAt),
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurface.withOpacity(0.5),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 4),
                  Text(
                    item.content,
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: theme.colorScheme.onSurface.withOpacity(0.7),
                    ),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _TabItem {
  final String label;
  final String type;
  const _TabItem(this.label, this.type);
}

class _TypeInfo {
  final IconData icon;
  final Color color;
  const _TypeInfo(this.icon, this.color);
}
