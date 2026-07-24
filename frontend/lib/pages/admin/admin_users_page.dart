import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../models/models.dart';
import '../../providers/admin_provider.dart';
import '../../utils/capability_utils.dart';
import '../../widgets/error_view.dart';
import '_import_dialog.dart';

/// 用户管理页面。非学生组织角色可导入；管理操作按能力分别显示。
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
      if (CapabilityUtils.has(Capability.collegeUserRead)) {
        context.read<AdminProvider>().fetchUsers();
      }
    });
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_scrollController.position.pixels >=
        _scrollController.position.maxScrollExtent - 200) {
      final provider = context.read<AdminProvider>();
      if (!provider.usersLoading &&
          provider.users.length < provider.userTotal) {
        provider.fetchUsers();
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final canImport = CapabilityUtils.has(Capability.counselorImportStudent);
    final canRead = CapabilityUtils.has(Capability.collegeUserRead);
    return Scaffold(
      appBar: AppBar(title: const Text('用户管理')),
      floatingActionButton: canImport
          ? FloatingActionButton.extended(
              onPressed: () => showDialog(
                context: context,
                builder: (_) => const ImportStudentDialog(),
              ),
              icon: const Icon(Icons.upload_file),
              label: const Text('导入学生'),
            )
          : null,
      body: canRead
          ? Column(
              children: [
                _buildFilterBar(),
                Expanded(child: _buildUserList()),
              ],
            )
          : _buildImportOnly(context, canImport),
    );
  }

  Widget _buildImportOnly(BuildContext context, bool canImport) {
    final theme = Theme.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 520),
          child: Card(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(
                    Icons.group_add_outlined,
                    size: 56,
                    color: theme.colorScheme.primary,
                  ),
                  const SizedBox(height: 16),
                  Text('导入学生用户', style: theme.textTheme.titleLarge),
                  const SizedBox(height: 8),
                  const Text(
                    '当前角色可按 Excel 模板批量创建学生账号。学生角色没有此权限。',
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: 20),
                  FilledButton.icon(
                    onPressed: canImport
                        ? () => showDialog(
                              context: context,
                              builder: (_) => const ImportStudentDialog(),
                            )
                        : null,
                    icon: const Icon(Icons.upload_file),
                    label: const Text('选择 Excel 文件'),
                  ),
                ],
              ),
            ),
          ),
        ),
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
                  value: provider.userRoleFilter.isEmpty
                      ? null
                      : provider.userRoleFilter,
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
                    DropdownMenuItem(
                        value: 'student_union', child: Text('学生会')),
                    DropdownMenuItem(value: 'counselor', child: Text('辅导员')),
                    DropdownMenuItem(value: 'teacher', child: Text('教师')),
                    DropdownMenuItem(value: 'assistant', child: Text('教辅')),
                    DropdownMenuItem(
                        value: 'college_admin', child: Text('学院管理员')),
                    DropdownMenuItem(
                        value: 'school_admin', child: Text('学校管理员')),
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
                  value: provider.userScopeFilter.isEmpty
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
            itemCount: provider.users.length +
                (provider.users.length < provider.userTotal ? 1 : 0),
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
    final canManage = CapabilityUtils.has(Capability.schoolUserUpdate) ||
        CapabilityUtils.has(Capability.systemPasswordReset);
    final details = [
      '@${user.username}',
      if (user.college.isNotEmpty) user.college,
      if (user.major.isNotEmpty) user.major,
      if (user.className.isNotEmpty) user.className,
    ].join(' · ');
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
        title: Text(user.displayName, style: theme.textTheme.titleSmall),
        subtitle: Text(details),
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
                  style: theme.textTheme.labelSmall?.copyWith(
                      color: theme.colorScheme.onSecondaryContainer)),
            ),
            if (user.status != 'active') ...[
              const SizedBox(width: 6),
              Icon(Icons.block, size: 18, color: theme.colorScheme.error),
            ],
            if (canManage) ...[
              const SizedBox(width: 6),
              const Icon(Icons.chevron_right),
            ],
          ],
        ),
        onTap: canManage ? () => _showEditDialog(context) : null,
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
  late final TextEditingController _displayNameController;
  late final TextEditingController _ownerIdController;
  final _passwordController = TextEditingController();
  late String _role;
  late String _scope;
  late String _status;
  bool _saving = false;
  bool _resetting = false;
  bool _deleting = false;

  @override
  void initState() {
    super.initState();
    _displayNameController =
        TextEditingController(text: widget.user.displayName);
    _ownerIdController = TextEditingController(text: widget.user.ownerId);
    _role = widget.user.role;
    _scope =
        widget.user.ownerScope.isEmpty ? 'college' : widget.user.ownerScope;
    _status = widget.user.status;
  }

  @override
  void dispose() {
    _displayNameController.dispose();
    _ownerIdController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final canUpdate = CapabilityUtils.has(Capability.schoolUserUpdate);
    final canReset = CapabilityUtils.has(Capability.systemPasswordReset);
    return AlertDialog(
      title: Text('编辑用户: ${widget.user.displayName}'),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: _displayNameController,
              enabled: canUpdate,
              decoration: const InputDecoration(
                labelText: '姓名',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            DropdownButtonFormField<String>(
              value: _role,
              decoration: const InputDecoration(
                labelText: '角色',
                border: OutlineInputBorder(),
                isDense: true,
              ),
              items: const [
                DropdownMenuItem(value: 'student', child: Text('学生')),
                DropdownMenuItem(value: 'student_union', child: Text('学生会')),
                DropdownMenuItem(value: 'counselor', child: Text('辅导员')),
                DropdownMenuItem(value: 'teacher', child: Text('教师')),
                DropdownMenuItem(value: 'assistant', child: Text('教辅')),
                DropdownMenuItem(value: 'college_admin', child: Text('学院管理员')),
                DropdownMenuItem(value: 'school_admin', child: Text('学校管理员')),
                DropdownMenuItem(value: 'sys_admin', child: Text('系统管理员')),
                DropdownMenuItem(value: 'guest', child: Text('游客')),
              ],
              onChanged: canUpdate
                  ? (v) {
                      if (v != null) setState(() => _role = v);
                    }
                  : null,
            ),
            const SizedBox(height: 12),
            DropdownButtonFormField<String>(
              value: _scope,
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
              onChanged: canUpdate
                  ? (v) {
                      if (v != null) setState(() => _scope = v);
                    }
                  : null,
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _ownerIdController,
              enabled: canUpdate,
              decoration: const InputDecoration(
                labelText: '归属 ID',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            DropdownButtonFormField<String>(
              value: _status,
              decoration: const InputDecoration(
                labelText: '账号状态',
                border: OutlineInputBorder(),
              ),
              items: const [
                DropdownMenuItem(value: 'active', child: Text('正常')),
                DropdownMenuItem(value: 'disabled', child: Text('停用')),
                DropdownMenuItem(value: 'pending', child: Text('待审核')),
                DropdownMenuItem(value: 'rejected', child: Text('已拒绝')),
              ],
              onChanged: canUpdate
                  ? (value) {
                      if (value != null) setState(() => _status = value);
                    }
                  : null,
            ),
            if (canReset) ...[
              const Divider(height: 24),
              TextField(
                controller: _passwordController,
                obscureText: true,
                decoration: const InputDecoration(
                  labelText: '新密码（重置用）',
                  hintText: '输入新密码后点击下方按钮',
                  border: OutlineInputBorder(),
                  isDense: true,
                ),
              ),
              const SizedBox(height: 8),
              SizedBox(
                width: double.infinity,
                child: OutlinedButton.icon(
                  onPressed: _resetting ? null : _handleResetPassword,
                  icon: _resetting
                      ? const SizedBox(
                          width: 14,
                          height: 14,
                          child: CircularProgressIndicator(strokeWidth: 2))
                      : const Icon(Icons.lock_reset, size: 18),
                  label: Text(_resetting ? '重置中...' : '重置密码'),
                ),
              ),
            ],
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: const Text('取消'),
        ),
        if (canUpdate) ...[
          TextButton.icon(
            onPressed: _deleting ? null : _handleDelete,
            icon: _deleting
                ? const SizedBox(
                    width: 14,
                    height: 14,
                    child: CircularProgressIndicator(strokeWidth: 2))
                : const Icon(Icons.delete_outline, size: 18),
            label: Text(_deleting ? '删除中...' : '删除用户'),
            style: TextButton.styleFrom(
              foregroundColor: Theme.of(context).colorScheme.error,
            ),
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
      ],
    );
  }

  Future<void> _handleSave() async {
    if (_displayNameController.text.trim().isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('姓名不能为空')),
      );
      return;
    }
    setState(() => _saving = true);
    final ok = await context.read<AdminProvider>().updateUser(
      widget.user.id,
      {
        'display_name': _displayNameController.text.trim(),
        'role': _role,
        'owner_scope': _scope,
        'owner_id': _ownerIdController.text.trim(),
        'status': _status,
      },
    );
    if (mounted) {
      if (ok) {
        Navigator.pop(context);
        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('更新成功')));
      } else {
        setState(() => _saving = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(context.read<AdminProvider>().error)),
        );
      }
    }
  }

  Future<void> _handleResetPassword() async {
    final pwd = _passwordController.text;
    if (pwd.length < 6) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('新密码至少 6 位')),
      );
      return;
    }
    setState(() => _resetting = true);
    final ok = await context.read<AdminProvider>().resetUserPassword(
          widget.user.id,
          pwd,
        );
    if (mounted) {
      setState(() => _resetting = false);
      if (ok) {
        _passwordController.clear();
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('密码已重置')),
        );
      } else {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(context.read<AdminProvider>().error),
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
        );
      }
    }
  }

  Future<void> _handleDelete() async {
    // 二次确认
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('确认删除'),
        content: Text(
            '确定要删除用户「${widget.user.displayName}」（@${widget.user.username}）吗？\n\n该操作会同时删除其所有关联数据（会话、聊天记录、办事记录等），不可恢复。'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('取消'),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error,
            ),
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('确认删除'),
          ),
        ],
      ),
    );
    if (confirmed != true) return;

    setState(() => _deleting = true);
    final ok = await context.read<AdminProvider>().deleteUser(widget.user.id);
    if (mounted) {
      if (ok) {
        Navigator.pop(context);
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('用户已删除')),
        );
      } else {
        setState(() => _deleting = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(context.read<AdminProvider>().error),
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
        );
      }
    }
  }
}
