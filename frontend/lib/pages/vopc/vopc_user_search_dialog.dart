import 'package:flutter/material.dart';

class VopcUserSearchDialog extends StatefulWidget {
  final TextEditingController controller;
  final Future<List<Map<String, dynamic>>> Function(String keyword) findUsers;
  const VopcUserSearchDialog(
      {super.key, required this.controller, required this.findUsers});
  @override
  State<VopcUserSearchDialog> createState() => _VopcUserSearchDialogState();
}

class _VopcUserSearchDialogState extends State<VopcUserSearchDialog> {
  List<Map<String, dynamic>> _users = const [];
  bool _loading = false;
  String _role = 'member';
  String? _selectedUserId;
  bool _searchDone = false;

  Future<void> _search(String query) async {
    if (query.trim().isEmpty) {
      setState(() {
        _users = const [];
        _searchDone = false;
      });
      return;
    }
    setState(() {
      _loading = true;
      _searchDone = true;
    });
    final result = await widget.findUsers(query.trim());
    if (!mounted) return;
    setState(() {
      _users = result;
      _loading = false;
    });
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('邀请成员'),
      content: SizedBox(
        width: 480,
        child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              TextField(
                  controller: widget.controller,
                  decoration: const InputDecoration(
                      labelText: '搜索用户（按姓名/账号）',
                      prefixIcon: Icon(Icons.search)),
                  onSubmitted: _search,
                  onChanged: _search),
              const SizedBox(height: 8),
              DropdownButton<String>(
                  value: _role,
                  items: const [
                    DropdownMenuItem(value: 'member', child: Text('成员')),
                    DropdownMenuItem(value: 'co_owner', child: Text('联合主理人')),
                    DropdownMenuItem(value: 'mentor', child: Text('导师')),
                    DropdownMenuItem(value: 'reviewer', child: Text('评审')),
                  ],
                  onChanged: (value) =>
                      setState(() => _role = value ?? 'member')),
              const SizedBox(height: 8),
              if (_loading)
                const LinearProgressIndicator()
              else if (!_searchDone)
                const Text('输入关键字后回车或继续输入以搜索',
                    style: TextStyle(color: Colors.grey))
              else if (_users.isEmpty)
                const Text('未找到可邀请用户', style: TextStyle(color: Colors.grey))
              else
                Flexible(
                    child: ListView(shrinkWrap: true, children: [
                  ..._users.map((user) => ListTile(
                        dense: true,
                        leading: Icon(_selectedUserId == user['id'].toString()
                            ? Icons.radio_button_checked
                            : Icons.person_outline),
                        title: Text(user['display_name']?.toString() ??
                            '用户 #${user['id']}'),
                        subtitle:
                            Text('@${user['username']} · ${user['role']}'),
                        onTap: () => setState(
                            () => _selectedUserId = user['id'].toString()),
                      )),
                ])),
            ]),
      ),
      actions: [
        TextButton(
            onPressed: () => Navigator.pop(context), child: const Text('取消')),
        FilledButton(
            onPressed: _selectedUserId == null
                ? null
                : () => Navigator.pop(context,
                    {'user_id': _selectedUserId!, 'role': _role, 'note': ''}),
            child: const Text('邀请')),
      ],
    );
  }
}
