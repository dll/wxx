import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../config/api_config.dart';

class PrivateChatPage extends StatefulWidget {
  const PrivateChatPage({super.key});
  @override
  State<PrivateChatPage> createState() => _PrivateChatPageState();
}

class _PrivateChatPageState extends State<PrivateChatPage> {
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    try {
      final provider = context.read<StudentFeatureProvider>();
      await provider.askAI(ApiConfig.privateChat);
      if (mounted) setState(() { _loading = false; });
    } catch (e) {
      if (mounted) setState(() { _loading = false; });
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('站内私聊')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                _buildSection(theme, '最近对话', Icons.chat_bubble_outline, [
                  _chatTile(theme, '李辅导员', '明天下午来办公室聊聊', '10:30', 1, Colors.orange),
                  _chatTile(theme, '张学长', 'ACM训练资料已发你邮箱', '昨天', 0, Colors.blue),
                  _chatTile(theme, 'AI学友-王同学', '明天一起去图书馆复习吧', '昨天', 0, Colors.green),
                ]),
                const SizedBox(height: 16),
                _buildSection(theme, '推荐联系人', Icons.person_add, [
                  _recommendTile(theme, '赵学姐', '同专业大三，擅长算法', 88),
                  _recommendTile(theme, '刘同学', '学习风格互补，可组队复习', 82),
                ]),
              ],
            ),
      floatingActionButton: FloatingActionButton(
        onPressed: () {},
        child: const Icon(Icons.edit),
      ),
    );
  }

  Widget _buildSection(ThemeData theme, String title, IconData icon, List<Widget> children) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Icon(icon, size: 20, color: theme.colorScheme.primary),
              const SizedBox(width: 8),
              Text(title, style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
            ]),
            const SizedBox(height: 12),
            ...children,
          ],
        ),
      ),
    );
  }

  Widget _chatTile(ThemeData theme, String name, String msg, String time, int unread, Color avatar) {
    return ListTile(
      leading: CircleAvatar(backgroundColor: avatar.withValues(alpha: 0.2), child: Text(name[0], style: TextStyle(color: avatar))),
      title: Text(name),
      subtitle: Text(msg, maxLines: 1, overflow: TextOverflow.ellipsis),
      trailing: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
        Text(time, style: theme.textTheme.bodySmall),
        if (unread > 0) ...[
          const SizedBox(height: 4),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
            decoration: BoxDecoration(color: Colors.red, borderRadius: BorderRadius.circular(10)),
            child: Text('$unread', style: const TextStyle(color: Colors.white, fontSize: 11)),
          ),
        ],
      ]),
      onTap: () {},
    );
  }

  Widget _recommendTile(ThemeData theme, String name, String reason, int score) {
    return ListTile(
      leading: CircleAvatar(backgroundColor: theme.colorScheme.primaryContainer, child: Text(name[0])),
      title: Text(name),
      subtitle: Text(reason),
      trailing: Chip(label: Text('匹配 $score%'), backgroundColor: theme.colorScheme.primaryContainer),
      onTap: () {},
    );
  }
}
