import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../models/models.dart';
import '../../providers/admin_provider.dart';
import '../../providers/auth_provider.dart';
import '../../utils/capability_utils.dart';
import '../../utils/storage.dart';
import '../../widgets/error_view.dart';
import '_import_dialog.dart';

/// 用户管理页面（增强版）
/// 功能：搜索、多条件筛选、批量选择、批量操作、分页
class AdminUsersPage extends StatefulWidget {
  const AdminUsersPage({super.key});

  @override
  State<AdminUsersPage> createState() => _AdminUsersPageState();
}

class _AdminUsersPageState extends State<AdminUsersPage> {
  final _scrollController = ScrollController();
  final _searchController = TextEditingController();
  bool _showAdvancedFilter = false;
  bool _dictLoaded = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (CapabilityUtils.has(Capability.collegeUserRead)) {
        context.read<AdminProvider>().searchUsers(refresh: true);
        // 学生活动统计面板（任务6：注册/登录/打卡）
        context.read<AdminProvider>().fetchUserActivityStats();
      }
    });
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _searchController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_scrollController.position.pixels >=
        _scrollController.position.maxScrollExtent - 200) {
      final provider = context.read<AdminProvider>();
      if (!provider.usersLoading &&
          provider.users.length < provider.userTotal) {
        provider.searchUsers();
      }
    }
  }

  Future<void> _loadDicts() async {
    if (_dictLoaded) return;
    _dictLoaded = true;
    final provider = context.read<AdminProvider>();
    await provider.fetchDictValues('college');
    await provider.fetchDictValues('major');
    await provider.fetchDictValues('class_name');
    await provider.fetchDictValues('enrollment_year');
  }

  @override
  Widget build(BuildContext context) {
    final canImport = CapabilityUtils.has(Capability.counselorImportStudent);
    final canRead = CapabilityUtils.has(Capability.collegeUserRead);
    final canManage = CapabilityUtils.has(Capability.schoolUserUpdate);
    final canResetPwd = CapabilityUtils.has(Capability.systemPasswordReset);

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
                _buildRoleTabs(),
                _buildSearchBar(),
                if (_showAdvancedFilter) _buildAdvancedFilter(),
                _buildStatsBar(),
                _buildUserActivityPanel(canManage || canResetPwd),
                if (canManage || canResetPwd) _buildBatchActionBar(),
                Expanded(child: _buildUserList()),
              ],
            )
          : _buildImportOnly(context, canImport),
    );
  }

  /// 学生活动统计面板（任务6）：注册/登录/打卡概览 + 推送给 120001（任务7）
  Widget _buildUserActivityPanel(bool canNotify) {
    return Consumer<AdminProvider>(
      builder: (_, provider, __) {
        final s = provider.userActivityStats;
        if (s == null) {
          return const SizedBox.shrink();
        }
        int numOf(String key) => (s[key] as num?)?.toInt() ?? 0;
        final items = [
          ('累计注册', numOf('registered_total')),
          ('待审核', numOf('pending_approval')),
          ('今日登录', numOf('login_today_users')),
          ('今日打卡', numOf('checkin_today')),
        ];
        return Container(
          margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.surfaceContainerLowest,
            borderRadius: BorderRadius.circular(10),
            border: Border.all(
              color: Theme.of(context).colorScheme.outlineVariant.withOpacity(0.3),
            ),
          ),
          child: Row(
            children: [
              for (final item in items) ...[
                Expanded(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text('${item.$2}',
                          style: Theme.of(context).textTheme.titleMedium?.copyWith(
                                fontWeight: FontWeight.w700,
                              )),
                      Text(item.$1,
                          style: Theme.of(context).textTheme.bodySmall?.copyWith(
                                color: Theme.of(context).colorScheme.onSurfaceVariant,
                              )),
                    ],
                  ),
                ),
              ],
              if (canNotify)
                IconButton(
                  tooltip: '推送统计摘要通知给胡老师（120001）',
                  icon: const Icon(Icons.notifications_active_outlined),
                  onPressed: () async {
                    final messenger = ScaffoldMessenger.of(context);
                    final n = await provider.notifyUserActivityStats();
                    messenger.showSnackBar(
                      SnackBar(
                        content: Text(n > 0
                            ? '统计摘要已推送给胡老师（120001）'
                            : '推送失败，请稍后重试'),
                      ),
                    );
                  },
                ),
            ],
          ),
        );
      },
    );
  }

  /// 角色分类 Tab：全部/学生/辅导员/教师/管理员，点击快速过滤
  Widget _buildRoleTabs() {
    return Consumer<AdminProvider>(
      builder: (_, provider, __) {
        final current = provider.userRoleFilter;
        Widget tab(String label, String value) {
          final selected = current == value;
          final theme = Theme.of(context);
          return Padding(
            padding: const EdgeInsets.symmetric(horizontal: 2),
            child: ChoiceChip(
              label: Text(label, style: const TextStyle(fontSize: 12)),
              selected: selected,
              visualDensity: VisualDensity.compact,
              onSelected: (_) =>
                  provider.setUserFilter(role: selected ? '' : value),
              selectedColor: theme.colorScheme.primaryContainer,
              labelStyle: TextStyle(
                color: selected
                    ? theme.colorScheme.onPrimaryContainer
                    : theme.colorScheme.onSurfaceVariant,
                fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
              ),
            ),
          );
        }

        return Container(
          width: double.infinity,
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 0),
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: Row(
              children: [
                tab('全部', ''),
                tab('学生', 'student'),
                tab('辅导员', 'counselor'),
                tab('教师', 'teacher'),
                tab('学生会', 'student_union'),
                tab('管理员', 'college_admin'),
              ],
            ),
          ),
        );
      },
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

  // ── 搜索栏 ──

  Widget _buildSearchBar() {
    return Container(
      padding: const EdgeInsets.fromLTRB(12, 10, 12, 6),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surface,
        border: Border(
          bottom: BorderSide(
            color:
                Theme.of(context).colorScheme.outlineVariant.withOpacity(0.3),
          ),
        ),
      ),
      child: Row(
        children: [
          Expanded(
            child: TextField(
              controller: _searchController,
              decoration: InputDecoration(
                hintText: '搜索姓名、学号、学院、专业、班级...',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: _searchController.text.isNotEmpty
                    ? IconButton(
                        icon: const Icon(Icons.clear, size: 20),
                        onPressed: () {
                          _searchController.clear();
                          context.read<AdminProvider>().setKeyword('');
                        },
                      )
                    : null,
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(10),
                ),
                isDense: true,
                contentPadding: const EdgeInsets.symmetric(vertical: 10),
              ),
              onSubmitted: (value) {
                context.read<AdminProvider>().setKeyword(value);
              },
              onChanged: (value) {
                if (value.isEmpty) {
                  context.read<AdminProvider>().setKeyword('');
                }
              },
            ),
          ),
          const SizedBox(width: 8),
          IconButton.filledTonal(
            onPressed: () {
              setState(() {
                _showAdvancedFilter = !_showAdvancedFilter;
                if (_showAdvancedFilter) _loadDicts();
              });
            },
            icon: Icon(
              _showAdvancedFilter
                  ? Icons.filter_alt
                  : Icons.filter_alt_outlined,
            ),
            tooltip: '高级筛选',
          ),
        ],
      ),
    );
  }

  // ── 高级筛选面板 ──

  Widget _buildAdvancedFilter() {
    return Consumer<AdminProvider>(
      builder: (_, provider, __) {
        return Container(
          padding: const EdgeInsets.fromLTRB(12, 4, 12, 12),
          decoration: BoxDecoration(
            color: Theme.of(context)
                .colorScheme
                .surfaceContainerHighest
                .withOpacity(0.3),
            border: Border(
              bottom: BorderSide(
                color: Theme.of(context)
                    .colorScheme
                    .outlineVariant
                    .withOpacity(0.3),
              ),
            ),
          ),
          child: Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              _buildFilterDropdown(
                label: '角色',
                value: provider.userRoleFilter.isEmpty
                    ? null
                    : provider.userRoleFilter,
                items: const [
                  DropdownMenuItem(value: '', child: Text('全部角色')),
                  DropdownMenuItem(value: 'student', child: Text('学生')),
                  DropdownMenuItem(value: 'student_union', child: Text('学生会')),
                  DropdownMenuItem(value: 'counselor', child: Text('辅导员')),
                  DropdownMenuItem(value: 'teacher', child: Text('教师')),
                  DropdownMenuItem(value: 'assistant', child: Text('教辅')),
                  DropdownMenuItem(
                      value: 'college_admin', child: Text('学院管理员')),
                  DropdownMenuItem(value: 'school_admin', child: Text('学校管理员')),
                  DropdownMenuItem(value: 'sys_admin', child: Text('系统管理员')),
                  DropdownMenuItem(value: 'guest', child: Text('游客')),
                ],
                onChanged: (v) => provider.setUserFilter(role: v ?? ''),
              ),
              _buildFilterDropdown(
                label: '状态',
                value: provider.statusFilter.isEmpty
                    ? null
                    : provider.statusFilter,
                items: const [
                  DropdownMenuItem(value: '', child: Text('全部状态')),
                  DropdownMenuItem(value: 'active', child: Text('正常')),
                  DropdownMenuItem(value: 'disabled', child: Text('已禁用')),
                  DropdownMenuItem(value: 'pending', child: Text('待审核')),
                ],
                onChanged: (v) => provider.setUserFilter(status: v ?? ''),
              ),
              _buildFilterDropdown(
                label: '学院',
                value: provider.collegeFilter.isEmpty
                    ? null
                    : provider.collegeFilter,
                items: [
                  const DropdownMenuItem(value: '', child: Text('全部学院')),
                  ...provider.collegeList
                      .map((e) => DropdownMenuItem(value: e, child: Text(e))),
                ],
                onChanged: (v) {
                  provider.setUserFilter(
                      college: v ?? '', major: '', className: '');
                },
              ),
              _buildFilterDropdown(
                label: '专业',
                value:
                    provider.majorFilter.isEmpty ? null : provider.majorFilter,
                items: [
                  const DropdownMenuItem(value: '', child: Text('全部专业')),
                  ...provider.majorList
                      .map((e) => DropdownMenuItem(value: e, child: Text(e))),
                ],
                onChanged: (v) => provider.setUserFilter(major: v ?? ''),
              ),
              _buildFilterDropdown(
                label: '班级',
                value:
                    provider.classFilter.isEmpty ? null : provider.classFilter,
                items: [
                  const DropdownMenuItem(value: '', child: Text('全部班级')),
                  ...provider.classList
                      .map((e) => DropdownMenuItem(value: e, child: Text(e))),
                ],
                onChanged: (v) => provider.setUserFilter(className: v ?? ''),
              ),
              _buildFilterDropdown(
                label: '入学年份',
                value: provider.enrollmentYearFilter.isEmpty
                    ? null
                    : provider.enrollmentYearFilter,
                items: [
                  const DropdownMenuItem(value: '', child: Text('全部年份')),
                  ...provider.enrollmentYearList
                      .map((e) => DropdownMenuItem(value: e, child: Text(e))),
                ],
                onChanged: (v) =>
                    provider.setUserFilter(enrollmentYear: v ?? ''),
              ),
              TextButton.icon(
                onPressed: () {
                  _searchController.clear();
                  provider.resetFilters();
                },
                icon: const Icon(Icons.refresh, size: 18),
                label: const Text('重置'),
              ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildFilterDropdown({
    required String label,
    required String? value,
    required List<DropdownMenuItem<String>> items,
    required ValueChanged<String?> onChanged,
  }) {
    return SizedBox(
      width: 140,
      child: DropdownButtonFormField<String>(
        value: value,
        decoration: InputDecoration(
          labelText: label,
          border: const OutlineInputBorder(),
          isDense: true,
          contentPadding:
              const EdgeInsets.symmetric(horizontal: 10, vertical: 10),
        ),
        items: items,
        onChanged: onChanged,
      ),
    );
  }

  // ── 统计栏 ──

  Widget _buildStatsBar() {
    return Consumer<AdminProvider>(
      builder: (_, provider, __) {
        return Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.surfaceContainerLowest,
            border: Border(
              bottom: BorderSide(
                color: Theme.of(context)
                    .colorScheme
                    .outlineVariant
                    .withOpacity(0.3),
              ),
            ),
          ),
          child: Row(
            children: [
              Text(
                '共 ${provider.userTotal} 人',
                style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
              ),
              const SizedBox(width: 16),
              Text(
                '已显示 ${provider.users.length} 人',
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: Theme.of(context).colorScheme.onSurfaceVariant,
                    ),
              ),
              const Spacer(),
              if (provider.selectedCount > 0) ...[
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                  decoration: BoxDecoration(
                    color: Theme.of(context).colorScheme.primaryContainer,
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Text(
                    '已选 ${provider.selectedCount} 人',
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.onPrimaryContainer,
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                TextButton(
                  onPressed: provider.deselectAll,
                  child: const Text('取消选择', style: TextStyle(fontSize: 12)),
                ),
              ],
            ],
          ),
        );
      },
    );
  }

  // ── 批量操作栏 ──

  Widget _buildBatchActionBar() {
    return Consumer<AdminProvider>(
      builder: (_, provider, __) {
        if (provider.selectedCount == 0) return const SizedBox.shrink();
        final canManage = CapabilityUtils.has(Capability.schoolUserUpdate);
        final canResetPwd = CapabilityUtils.has(Capability.systemPasswordReset);

        return Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color:
                Theme.of(context).colorScheme.primaryContainer.withOpacity(0.2),
            border: Border(
              bottom: BorderSide(
                color: Theme.of(context)
                    .colorScheme
                    .outlineVariant
                    .withOpacity(0.3),
              ),
            ),
          ),
          child: Row(
            children: [
              Icon(
                Icons.check_circle,
                size: 20,
                color: Theme.of(context).colorScheme.primary,
              ),
              const SizedBox(width: 6),
              Text(
                '已选择 ${provider.selectedCount} 位用户',
                style: TextStyle(
                  fontWeight: FontWeight.w600,
                  color: Theme.of(context).colorScheme.primary,
                ),
              ),
              const Spacer(),
              if (canManage) ...[
                _batchActionButton(
                  icon: Icons.play_arrow,
                  label: '启用',
                  color: Colors.green,
                  onPressed: () => _confirmBatchAction(
                    title: '批量启用',
                    content: '确定启用选中的 ${provider.selectedCount} 位用户吗？',
                    onConfirm: () => provider.batchUpdateStatus('active'),
                    successMsg: '批量启用成功',
                  ),
                ),
                const SizedBox(width: 6),
                _batchActionButton(
                  icon: Icons.block,
                  label: '禁用',
                  color: Colors.orange,
                  onPressed: () => _confirmBatchAction(
                    title: '批量禁用',
                    content:
                        '确定禁用选中的 ${provider.selectedCount} 位用户吗？\n禁用后用户将无法登录。',
                    onConfirm: () => provider.batchUpdateStatus('disabled'),
                    successMsg: '批量禁用成功',
                  ),
                ),
              ],
              if (canResetPwd) ...[
                const SizedBox(width: 6),
                _batchActionButton(
                  icon: Icons.vpn_key_outlined,
                  label: '重置密码',
                  color: Colors.blue,
                  onPressed: () => _showBatchResetPasswordDialog(),
                ),
              ],
              if (canManage) ...[
                const SizedBox(width: 6),
                _batchActionButton(
                  icon: Icons.delete_outline,
                  label: '删除',
                  color: Theme.of(context).colorScheme.error,
                  onPressed: () => _confirmBatchAction(
                    title: '批量删除',
                    content:
                        '确定删除选中的 ${provider.selectedCount} 位用户吗？\n该操作将同时删除其所有关联数据（会话、聊天记录、办事记录等），不可恢复！',
                    onConfirm: provider.batchDelete,
                    successMsg: '批量删除成功',
                    isDestructive: true,
                  ),
                ),
              ],
            ],
          ),
        );
      },
    );
  }

  Widget _batchActionButton({
    required IconData icon,
    required String label,
    required Color color,
    required VoidCallback onPressed,
  }) {
    return TextButton.icon(
      onPressed: onPressed,
      icon: Icon(icon, size: 18, color: color),
      label: Text(label, style: TextStyle(color: color, fontSize: 13)),
      style: TextButton.styleFrom(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
        minimumSize: Size.zero,
        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
      ),
    );
  }

  Future<void> _confirmBatchAction({
    required String title,
    required String content,
    required Future<bool> Function() onConfirm,
    required String successMsg,
    bool isDestructive = false,
  }) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(title),
        content: Text(content),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('取消'),
          ),
          FilledButton(
            style: isDestructive
                ? FilledButton.styleFrom(
                    backgroundColor: Theme.of(ctx).colorScheme.error,
                  )
                : null,
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('确认'),
          ),
        ],
      ),
    );
    if (confirmed != true) return;

    final ok = await onConfirm();
    if (!mounted) return;
    if (ok) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(successMsg)),
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

  Future<void> _showBatchResetPasswordDialog() async {
    final controller = TextEditingController();
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('批量重置密码'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
                '将为选中的 ${context.read<AdminProvider>().selectedCount} 位用户重置密码，请输入新密码：'),
            const SizedBox(height: 12),
            TextField(
              controller: controller,
              decoration: const InputDecoration(
                labelText: '新密码（至少6位）',
                border: OutlineInputBorder(),
              ),
              obscureText: true,
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () {
              if (controller.text.length >= 6) {
                Navigator.pop(ctx, true);
              }
            },
            child: const Text('确认重置'),
          ),
        ],
      ),
    );
    if (confirmed != true) return;

    final ok =
        await context.read<AdminProvider>().batchResetPassword(controller.text);
    if (!mounted) return;
    if (ok) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('批量重置密码成功')),
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

  // ── 用户列表 ──

  Widget _buildUserList() {
    return Consumer<AdminProvider>(
      builder: (_, provider, __) {
        if (provider.usersLoading && provider.users.isEmpty) {
          return const Center(child: CircularProgressIndicator());
        }
        if (provider.error.isNotEmpty && provider.users.isEmpty) {
          return ErrorView.error(
            message: provider.error,
            onRetry: () => provider.searchUsers(refresh: true),
          );
        }
        if (provider.users.isEmpty) {
          return ErrorView.empty(message: '暂无用户', icon: Icons.people_outline);
        }

        final canManage = CapabilityUtils.has(Capability.schoolUserUpdate) ||
            CapabilityUtils.has(Capability.systemPasswordReset);
        final showCheckbox = CapabilityUtils.has(Capability.schoolUserUpdate) ||
            CapabilityUtils.has(Capability.systemPasswordReset);

        return RefreshIndicator(
          onRefresh: () => provider.searchUsers(refresh: true),
          child: ListView.builder(
            controller: _scrollController,
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 24),
            itemCount: provider.users.length +
                (provider.users.length < provider.userTotal ? 1 : 0),
            itemBuilder: (context, index) {
              if (index == provider.users.length) {
                return const Padding(
                  padding: EdgeInsets.all(16),
                  child: Center(child: CircularProgressIndicator()),
                );
              }
              final user = provider.users[index];
              final selected = provider.selectedUserIds.contains(user.id);
              return _UserTile(
                user: user,
                selected: selected,
                showCheckbox: showCheckbox,
                onSelectChanged:
                    canManage ? (v) => provider.toggleSelect(user.id) : null,
                onTap: canManage ? () => _showEditDialog(user) : null,
                onResetPassword:
                    CapabilityUtils.has(Capability.systemPasswordReset)
                        ? () => _showResetPasswordDialog(user)
                        : null,
              );
            },
          ),
        );
      },
    );
  }

  void _showEditDialog(UserProfile user) {
    showDialog(
      context: context,
      builder: (ctx) => _UserEditDialog(user: user),
    );
  }

  /// 系统管理员重置单个用户密码（快捷入口）
  Future<void> _showResetPasswordDialog(UserProfile user) async {
    final controller = TextEditingController();
    var resetting = false;
    await showDialog<void>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setDialogState) => AlertDialog(
          title: const Text('重置密码'),
          content: SizedBox(
            width: 400,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  '为「${user.displayName}」（${user.username}）重置密码。'
                  '重置后该用户需用新密码重新登录，旧登录态将失效。',
                  style: Theme.of(ctx).textTheme.bodySmall,
                ),
                const SizedBox(height: 16),
                TextField(
                  controller: controller,
                  obscureText: true,
                  autofocus: true,
                  decoration: const InputDecoration(
                    labelText: '新密码（至少 6 位）',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                ),
                const SizedBox(height: 8),
                const Text(
                  '提示：若设为与学号相同，该用户下次登录会再次被要求修改密码。',
                  style: TextStyle(fontSize: 11, color: Colors.grey),
                ),
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: resetting ? null : () => Navigator.pop(ctx),
              child: const Text('取消'),
            ),
            FilledButton(
              onPressed: resetting
                  ? null
                  : () async {
                      final pwd = controller.text.trim();
                      if (pwd.length < 6) {
                        ScaffoldMessenger.of(ctx).showSnackBar(
                            const SnackBar(content: Text('新密码长度不能少于 6 位')));
                        return;
                      }
                      setDialogState(() => resetting = true);
                      final ok = await context
                          .read<AdminProvider>()
                          .resetUserPassword(user.id, pwd);
                      if (!ctx.mounted) return;
                      Navigator.pop(ctx);
                      ScaffoldMessenger.of(context).showSnackBar(
                          SnackBar(content: Text(ok ? '密码重置成功' : '重置失败')));
                    },
              child: resetting
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('确认重置'),
            ),
          ],
        ),
      ),
    );
  }
}

// ── 用户列表项 ──

class _UserTile extends StatelessWidget {
  final UserProfile user;
  final bool selected;
  final bool showCheckbox;
  final ValueChanged<bool?>? onSelectChanged;
  final VoidCallback? onTap;
  final VoidCallback? onResetPassword;

  const _UserTile({
    required this.user,
    required this.selected,
    required this.showCheckbox,
    this.onSelectChanged,
    this.onTap,
    this.onResetPassword,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final subtitleParts = [
      '@${user.username}',
      if (user.college.isNotEmpty) user.college,
      if (user.major.isNotEmpty) user.major,
      if (user.className.isNotEmpty) user.className,
    ];

    return Card(
      margin: const EdgeInsets.only(bottom: 6),
      elevation: selected ? 2 : 0,
      color: selected
          ? theme.colorScheme.primaryContainer.withOpacity(0.15)
          : null,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(10),
        side: BorderSide(
          color: selected
              ? theme.colorScheme.primary.withOpacity(0.5)
              : theme.colorScheme.outlineVariant.withOpacity(0.3),
          width: selected ? 1.5 : 1,
        ),
      ),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(10),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          child: Row(
            children: [
              if (showCheckbox)
                Checkbox(
                  value: selected,
                  onChanged: onSelectChanged,
                  visualDensity: VisualDensity.compact,
                ),
              CircleAvatar(
                radius: 20,
                backgroundColor: theme.colorScheme.primaryContainer,
                child: Text(
                  user.displayName.isNotEmpty
                      ? user.displayName[0].toUpperCase()
                      : '?',
                  style: TextStyle(color: theme.colorScheme.onPrimaryContainer),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Flexible(
                          child: Text(
                            user.displayName,
                            style: theme.textTheme.titleSmall
                                ?.copyWith(fontWeight: FontWeight.w600),
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                        const SizedBox(width: 8),
                        Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 8, vertical: 2),
                          decoration: BoxDecoration(
                            color: _getRoleColor(user.role, theme),
                            borderRadius: BorderRadius.circular(10),
                          ),
                          child: Text(
                            user.roleLabel,
                            style: TextStyle(
                              fontSize: 11,
                              color: _getRoleOnColor(user.role, theme),
                              fontWeight: FontWeight.w500,
                            ),
                          ),
                        ),
                        if (user.position.isNotEmpty) ...[
                          const SizedBox(width: 6),
                          Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 6, vertical: 2),
                            decoration: BoxDecoration(
                              color: theme.colorScheme.secondaryContainer,
                              borderRadius: BorderRadius.circular(8),
                            ),
                            child: Text(
                              user.position,
                              style: TextStyle(
                                fontSize: 10,
                                color: theme.colorScheme.onSecondaryContainer,
                                fontWeight: FontWeight.w500,
                              ),
                            ),
                          ),
                        ],
                        if (user.status != 'active') ...[
                          const SizedBox(width: 6),
                          Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 6, vertical: 2),
                            decoration: BoxDecoration(
                              color: theme.colorScheme.errorContainer,
                              borderRadius: BorderRadius.circular(8),
                            ),
                            child: Text(
                              _getStatusLabel(user.status),
                              style: TextStyle(
                                fontSize: 10,
                                color: theme.colorScheme.onErrorContainer,
                              ),
                            ),
                          ),
                        ],
                      ],
                    ),
                    const SizedBox(height: 4),
                    Text(
                      subtitleParts.join(' · '),
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ],
                ),
              ),
              if (onResetPassword != null)
                IconButton(
                  tooltip: '重置密码',
                  visualDensity: VisualDensity.compact,
                  iconSize: 20,
                  onPressed: onResetPassword,
                  icon: const Icon(Icons.password, color: Colors.blueGrey),
                ),
              if (onTap != null)
                Icon(
                  Icons.chevron_right,
                  size: 20,
                  color: theme.colorScheme.onSurfaceVariant,
                ),
            ],
          ),
        ),
      ),
    );
  }

  Color _getRoleColor(String role, ThemeData theme) {
    switch (role) {
      case 'student':
        return theme.colorScheme.secondaryContainer;
      case 'counselor':
        return theme.colorScheme.tertiaryContainer;
      case 'teacher':
        return theme.colorScheme.primaryContainer;
      case 'sys_admin':
        return theme.colorScheme.errorContainer;
      case 'school_admin':
      case 'college_admin':
        return theme.colorScheme.surfaceContainerHighest;
      default:
        return theme.colorScheme.surfaceContainerHighest;
    }
  }

  Color _getRoleOnColor(String role, ThemeData theme) {
    switch (role) {
      case 'student':
        return theme.colorScheme.onSecondaryContainer;
      case 'counselor':
        return theme.colorScheme.onTertiaryContainer;
      case 'teacher':
        return theme.colorScheme.onPrimaryContainer;
      case 'sys_admin':
        return theme.colorScheme.onErrorContainer;
      default:
        return theme.colorScheme.onSurfaceVariant;
    }
  }

  String _getStatusLabel(String status) {
    switch (status) {
      case 'active':
        return '正常';
      case 'disabled':
        return '已禁用';
      case 'pending':
        return '待审核';
      case 'rejected':
        return '已拒绝';
      default:
        return status;
    }
  }
}

// ── 用户编辑弹窗 ──

class _UserEditDialog extends StatefulWidget {
  final UserProfile user;
  const _UserEditDialog({required this.user});

  @override
  State<_UserEditDialog> createState() => _UserEditDialogState();
}

class _UserEditDialogState extends State<_UserEditDialog> {
  late final TextEditingController _displayNameController;
  late final TextEditingController _ownerIdController;
  late final TextEditingController _positionController;
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
    _positionController = TextEditingController(text: widget.user.position);
    _role = widget.user.role;
    _scope =
        widget.user.ownerScope.isEmpty ? 'college' : widget.user.ownerScope;
    _status = widget.user.status;
  }

  @override
  void dispose() {
    _displayNameController.dispose();
    _ownerIdController.dispose();
    _positionController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  /// 当前操作者角色（用于角色指派范围限制）
  String _operatorRole(BuildContext context) {
    final profile = context.read<AuthProvider>().profile;
    if (profile != null && profile.role.isNotEmpty) return profile.role;
    return Storage.role ?? '';
  }

  /// 可按角色分组的职务快捷选项
  List<String> _positionPresets(String role) {
    switch (role) {
      case 'student_union':
        return const ['主席', '副主席', '部长', '副部长', '干事'];
      case 'counselor':
        return const ['年级辅导员', '专职辅导员', '兼职辅导员'];
      case 'teacher':
        return const ['专任教师', '讲师', '副教授', '教授'];
      case 'assistant':
        return const ['教学秘书', '教务员', '辅导员助理'];
      case 'college_admin':
        return const ['院长', '副院长', '系主任'];
      case 'school_admin':
      case 'sys_admin':
        return const ['信息中心', '学工处', '教务处'];
      case 'student':
        return const ['班长', '团支书', '学习委员', '生活委员', '无'];
      default:
        return const [];
    }
  }

  /// 当前操作者可指派的角色列表
  List<Map<String, String>> _assignableRoles(String operatorRole) {
    final base = [
      {'v': 'student', 'label': '学生'},
      {'v': 'student_union', 'label': '学生会'},
      {'v': 'counselor', 'label': '辅导员'},
      {'v': 'teacher', 'label': '教师'},
      {'v': 'assistant', 'label': '教辅'},
      {'v': 'college_admin', 'label': '学院管理员'},
      {'v': 'school_admin', 'label': '学校管理员'},
    ];
    if (operatorRole == 'college_admin') {
      // 学院管理员不能授予校级/系统管理员，也不能给其他学院管理员指派
      return base
          .where((r) => r['v'] != 'school_admin' && r['v'] != 'college_admin')
          .toList();
    }
    return base;
  }

  @override
  Widget build(BuildContext context) {
    final canUpdate = CapabilityUtils.has(Capability.schoolUserUpdate);
    final canResetPwd = CapabilityUtils.has(Capability.systemPasswordReset);
    final theme = Theme.of(context);
    final operatorRole = _operatorRole(context);
    final assignableRoles = _assignableRoles(operatorRole);
    // 当前角色若不在可指派列表（如学校管理员被学院管理员查看），仍保留显示
    if (!assignableRoles.any((r) => r['v'] == _role)) {
      assignableRoles.insert(0, {'v': _role, 'label': _role});
    }
    final positionPresets = _positionPresets(_role);

    return AlertDialog(
      backgroundColor: theme.colorScheme.surface,
      surfaceTintColor: Colors.transparent,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(
            color: theme.colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      title: Row(
        children: [
          CircleAvatar(
            child: Text(widget.user.displayName.isNotEmpty
                ? widget.user.displayName[0]
                : '?'),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(widget.user.displayName),
                Text(
                  '@${widget.user.username}',
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
                ),
              ],
            ),
          ),
        ],
      ),
      content: SingleChildScrollView(
        child: SizedBox(
          width: 400,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              if (widget.user.college.isNotEmpty ||
                  widget.user.major.isNotEmpty ||
                  widget.user.className.isNotEmpty) ...[
                _infoRow('学院', widget.user.college),
                _infoRow('专业', widget.user.major),
                _infoRow('班级', widget.user.className),
                if (widget.user.enrollmentYear.isNotEmpty)
                  _infoRow('入学年份', widget.user.enrollmentYear),
                const Divider(height: 24),
              ],
              TextField(
                controller: _displayNameController,
                enabled: canUpdate,
                decoration: const InputDecoration(
                  labelText: '显示名称',
                  border: OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                value: _role,
                decoration: const InputDecoration(
                  labelText: '角色',
                  border: OutlineInputBorder(),
                ),
                items: assignableRoles
                    .map((r) => DropdownMenuItem(
                        value: r['v'], child: Text(r['label'] ?? r['v']!)))
                    .toList(),
                onChanged: canUpdate
                    ? (v) => setState(() => _role = v ?? 'student')
                    : null,
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _positionController,
                enabled: canUpdate,
                decoration: InputDecoration(
                  labelText: '职务',
                  hintText: '如 主席 / 部长 / 干事，或职称职务',
                  helperText: '职务与角色分开，描述岗位/职责',
                  border: const OutlineInputBorder(),
                ),
              ),
              if (positionPresets.isNotEmpty) ...[
                const SizedBox(height: 8),
                Wrap(
                  spacing: 6,
                  runSpacing: 6,
                  children: positionPresets
                      .map((p) => ActionChip(
                            label: Text(p),
                            visualDensity: VisualDensity.compact,
                            labelStyle: const TextStyle(fontSize: 12),
                            onPressed: canUpdate
                                ? () => setState(
                                    () => _positionController.text = p)
                                : null,
                          ))
                      .toList(),
                ),
              ],
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                value: _scope,
                decoration: const InputDecoration(
                  labelText: '归属范围',
                  border: OutlineInputBorder(),
                ),
                items: const [
                  DropdownMenuItem(value: 'school', child: Text('学校')),
                  DropdownMenuItem(value: 'college', child: Text('学院')),
                  DropdownMenuItem(value: 'class', child: Text('班级')),
                ],
                onChanged: canUpdate
                    ? (v) => setState(() => _scope = v ?? 'college')
                    : null,
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _ownerIdController,
                enabled: canUpdate,
                decoration: const InputDecoration(
                  labelText: '归属 ID',
                  helperText: '学院/班级等具体编号',
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
                  DropdownMenuItem(value: 'disabled', child: Text('已禁用')),
                ],
                onChanged: canUpdate
                    ? (v) => setState(() => _status = v ?? 'active')
                    : null,
              ),
              if (canResetPwd) ...[
                const SizedBox(height: 16),
                const Divider(height: 1),
                const SizedBox(height: 12),
                Text('重置密码', style: theme.textTheme.titleSmall),
                const SizedBox(height: 8),
                Row(
                  children: [
                    Expanded(
                      child: TextField(
                        controller: _passwordController,
                        decoration: const InputDecoration(
                          labelText: '新密码',
                          border: OutlineInputBorder(),
                          isDense: true,
                        ),
                        obscureText: true,
                      ),
                    ),
                    const SizedBox(width: 8),
                    FilledButton.tonal(
                      onPressed: _resetting || _passwordController.text.isEmpty
                          ? null
                          : _handleResetPassword,
                      child: _resetting
                          ? const SizedBox(
                              width: 16,
                              height: 16,
                              child: CircularProgressIndicator(strokeWidth: 2))
                          : const Text('重置'),
                    ),
                  ],
                ),
              ],
            ],
          ),
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

  Widget _infoRow(String label, String value) {
    if (value.isEmpty) return const SizedBox.shrink();
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 80,
            child: Text(
              label,
              style: TextStyle(
                color: Theme.of(context).colorScheme.onSurfaceVariant,
                fontSize: 13,
              ),
            ),
          ),
          Expanded(
            child: Text(
              value,
              style: const TextStyle(fontSize: 13),
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _handleSave() async {
    setState(() => _saving = true);
    final ok = await context.read<AdminProvider>().updateUser(
      widget.user.id,
      {
        'display_name': _displayNameController.text.trim(),
        'role': _role,
        'position': _positionController.text.trim(),
        'owner_scope': _scope,
        'owner_id': _ownerIdController.text.trim(),
        'status': _status,
      },
    );
    if (!mounted) return;
    setState(() => _saving = false);
    if (ok) {
      Navigator.pop(context);
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('保存成功')),
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

  Future<void> _handleResetPassword() async {
    setState(() => _resetting = true);
    final ok = await context
        .read<AdminProvider>()
        .resetUserPassword(widget.user.id, _passwordController.text);
    if (!mounted) return;
    setState(() => _resetting = false);
    if (ok) {
      _passwordController.clear();
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('密码重置成功')),
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

  Future<void> _handleDelete() async {
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
    if (!mounted) return;
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
