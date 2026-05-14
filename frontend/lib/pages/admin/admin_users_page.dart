import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/admin_provider.dart';
import '../../models/models.dart';
import '../../widgets/error_view.dart';

/// 用户管理页面（college_admin 及以上可访问）
class AdminUsersPage extends StatefulWidget {
  const AdminUsersPage({super.key});

  @override
  State<AdminUsersPage> createState() => _AdminUsersPageState();
}

class _AdminUsersPageState extends State<AdminUsersPage> {
  final _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AdminProvider>().fetchUsers();
    });
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_scrollController.position.pixels >= _scrollController.position.maxScrollExtent - 200) {
      final provider = context.read<AdminProvider>();
      if (!provider.usersLoading && provider.users.length < provider.userTotal) {
        provider.fetchUsers();
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('用户管理')),
      body: Column(
        children: [
          _buildFilterBar(),
          Expanded(child: _buildUserList()),
        ],
      ),
    );
  }

  Widget _buildFilterBar() {
    return Consumer<AdminProvider>(
      builder: (_, provider, __) {
        return Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          child: Row(
            children: [
              Expanded(
                child: DropdownButtonFormField<String>(
                  initialValue:
                      provider.userRoleFilter.isEmpty ? null : provider.userRoleFilter,
                  decoration: const InputDecoration(
                    labelText: '角色',
                    border: OutlineInputBorder(),
                    isDense: true,
                    contentPadding:
                        EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                  ),
                  items: const [
                    DropdownMenuItem(value: '', child: Text('全部角色')),
                    DropdownMenuItem(value: 'student', child: Text('学生')),
                    DropdownMenuItem(value: 'student_union', child: Text('学生会')),
                    DropdownMenuItem(value: 'counselor', child: Text('辅导员')),
                    DropdownMenuItem(value: 'college_admin', child: Text('学院管理员')),
                    DropdownMenuItem(value: 'school_admin', child: Text('学校管理员')),
                    DropdownMenuItem(value: 'sys_admin', child: Text('系统管理员')),
                  ],
                  onChanged: (v) {
                    provider.setUserFilter(role: v);
                  },
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: DropdownButtonFormField<String>(
                  initialValue: provider.userScopeFilter.isEmpty
                      ? null
                      : provider.userScopeFilter,
                  decoration: const InputDecoration(
                    labelText: '范围',
                    border: OutlineInputBorder(),
                    isDense: true,
                    contentPadding:
                        EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                  ),
                  items: const [
                    DropdownMenuItem(value: '', child: Text('全部范围')),
                    DropdownMenuItem(value: 'school', child: Text('学校')),
                    DropdownMenuItem(value: 'college', child: Text('学院')),
                    DropdownMenuItem(value: 'class', child: Text('班级')),
                  ],
                  onChanged: (v) {
                    provider.setUserFilter(scope: v);
                  },
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildUserList() {
    return Consumer<AdminProvider>(
      builder: (_, provider, __) {
        if (provider.usersLoading && provider.users.isEmpty) {
          return const Center(child: CircularProgressIndicator());
        }
        if (provider.error.isNotEmpty && provider.users.isEmpty) {
          return ErrorView.error(
            message: provider.error,
            onRetry: () => provider.fetchUsers(),
          );
        }
        if (provider.users.isEmpty) {
          return ErrorView.empty(message: '暂无用户', icon: Icons.people_outline);
        }
        return RefreshIndicator(
          onRefresh: () => provider.fetchUsers(refresh: true),
          child: ListView.builder(
            controller: _scrollController,
            padding: const EdgeInsets.symmetric(horizontal: 12),
            itemCount: provider.users.length + (provider.users.length < provider.userTotal ? 1 : 0),
            itemBuilder: (context, index) {
              if (index == provider.users.length) {
                return const Padding(
                  padding: EdgeInsets.all(16),
                  child: Center(child: CircularProgressIndicator()),
                );
              }
              return _UserTile(user: provider.users[index]);
            },
          ),
        );
      },
    );
  }
}

class _UserTile extends StatelessWidget {
  final UserProfile user;
  const _UserTile({required this.user});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        leading: CircleAvatar(
          backgroundColor: theme.colorScheme.primaryContainer,
          child: Text(
            user.displayName.isNotEmpty
                ? user.displayName[0].toUpperCase()
                : '?',
            style: TextStyle(color: theme.colorScheme.onPrimaryContainer),
          ),
        ),
        title: Text(user.displayName,
            style: theme.textTheme.titleSmall),
        subtitle: Text('@${user.username} · ${user.college}'),
        trailing: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
              decoration: BoxDecoration(
                color: theme.colorScheme.secondaryContainer,
                borderRadius: BorderRadius.circular(12),
              ),
              child: Text(user.roleLabel,
                  style: theme.textTheme.labelSmall
                      ?.copyWith(color: theme.colorScheme.onSecondaryContainer)),
            ),
            const SizedBox(width: 8),
            const Icon(Icons.chevron_right),
          ],
        ),
        onTap: () => _showEditDialog(context),
      ),
    );
  }

  void _showEditDialog(BuildContext context) {
    showDialog(
      context: context,
      builder: (ctx) => _UserEditDialog(user: user),
    );
  }
}

class _UserEditDialog extends StatefulWidget {
  final UserProfile user;
  const _UserEditDialog({required this.user});

  @override
  State<_UserEditDialog> createState() => _UserEditDialogState();
}

class _UserEditDialogState extends State<_UserEditDialog> {
  late String _role;
  late String _scope;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _role = widget.user.role;
    _scope = widget.user.college; // owner_scope 复用 college 字段
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text('编辑用户: ${widget.user.displayName}'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          DropdownButtonFormField<String>(
            initialValue: _role,
            decoration: const InputDecoration(
              labelText: '角色',
              border: OutlineInputBorder(),
              isDense: true,
            ),
            items: const [
              DropdownMenuItem(value: 'student', child: Text('学生')),
              DropdownMenuItem(value: 'student_union', child: Text('学生会')),
              DropdownMenuItem(value: 'counselor', child: Text('辅导员')),
              DropdownMenuItem(value: 'college_admin', child: Text('学院管理员')),
              DropdownMenuItem(value: 'school_admin', child: Text('学校管理员')),
              DropdownMenuItem(value: 'sys_admin', child: Text('系统管理员')),
            ],
            onChanged: (v) {
              if (v != null) setState(() => _role = v);
            },
          ),
          const SizedBox(height: 12),
          DropdownButtonFormField<String>(
            initialValue: _scope,
            decoration: const InputDecoration(
              labelText: '归属范围',
              border: OutlineInputBorder(),
              isDense: true,
            ),
            items: const [
              DropdownMenuItem(value: 'school', child: Text('学校')),
              DropdownMenuItem(value: 'college', child: Text('学院')),
              DropdownMenuItem(value: 'class', child: Text('班级')),
            ],
            onChanged: (v) {
              if (v != null) setState(() => _scope = v);
            },
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: const Text('取消'),
        ),
        FilledButton(
          onPressed: _saving ? null : _handleSave,
          child: _saving
              ? const SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2))
              : const Text('保存'),
        ),
      ],
    );
  }

  Future<void> _handleSave() async {
    setState(() => _saving = true);
    final ok = await context.read<AdminProvider>().updateUser(
          widget.user.id,
          {'role': _role, 'owner_scope': _scope},
        );
    if (mounted) {
      if (ok) {
        Navigator.pop(context);
        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('更新成功')));
      } else {
        setState(() => _saving = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content:
                  Text(context.read<AdminProvider>().error)),
        );
      }
    }
  }
}
