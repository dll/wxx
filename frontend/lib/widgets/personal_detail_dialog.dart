import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../config/api_config.dart';
import '../../models/models.dart';
import '../../providers/personal_detail_provider.dart';
import '../../services/api_service.dart';
import '../../utils/portrait_photo_picker.dart';
import '../../utils/storage.dart';

/// 个人详细信息弹窗 — 基本信息 / 联系方式 / 组织关系 / 学校门户绑定
class PersonalDetailDialog extends StatefulWidget {
  const PersonalDetailDialog({super.key});

  @override
  State<PersonalDetailDialog> createState() => _PersonalDetailDialogState();

  /// 展示弹窗
  static Future<void> show(BuildContext context) {
    return showDialog(
      context: context,
      builder: (_) => const PersonalDetailDialog(),
    );
  }
}

class _PersonalDetailDialogState extends State<PersonalDetailDialog>
    with SingleTickerProviderStateMixin {
  late final TabController _tab;
  final TextEditingController _portalUrl =
      TextEditingController(text: ApiConfig.schoolPortalUrl);
  final TextEditingController _portalAccount = TextEditingController();
  final TextEditingController _portalPassword = TextEditingController();
  bool _pwdVisible = false;
  bool _uploadingAvatar = false;

  @override
  void initState() {
    super.initState();
    _tab = TabController(length: 3, vsync: this);
    // 门户账号默认取当前登录账号（学号，与本系统一致）
    if (Storage.username != null && Storage.username!.isNotEmpty) {
      _portalAccount.text = Storage.username!;
    }
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<PersonalDetailProvider>().fetchAll();
    });
  }

  @override
  void dispose() {
    _tab.dispose();
    _portalUrl.dispose();
    _portalAccount.dispose();
    _portalPassword.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final p = context.watch<PersonalDetailProvider>();
    return AlertDialog(
      title: const Text('个人信息'),
      contentPadding: const EdgeInsets.fromLTRB(0, 8, 0, 0),
      content: SizedBox(
        width: 520,
        height: 500,
        child: Column(
          children: [
            _buildAvatarHeader(theme, p.detail),
            const Divider(height: 1),
            TabBar(
              controller: _tab,
              tabs: const [
                Tab(text: '基本信息'),
                Tab(text: '联系方式'),
                Tab(text: '组织关系'),
              ],
            ),
            Expanded(
              child: p.loading && p.detail == null
                  ? const Center(child: CircularProgressIndicator())
                  : TabBarView(
                      controller: _tab,
                      children: [
                        _buildBasicTab(theme, p.detail),
                        _buildContactTab(theme, p.detail),
                        _buildOrgTab(theme, p),
                      ],
                    ),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: const Text('关闭'),
        ),
      ],
    );
  }

  // ── 头像 ──
  Widget _buildAvatarHeader(ThemeData theme, PersonalDetail? d) {
    final name = d?.displayName ?? Storage.displayName ?? '?';
    final hasPhoto = (d?.avatarBase64 ?? '').isNotEmpty;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
      child: Row(
        children: [
          Stack(
            clipBehavior: Clip.none,
            children: [
              CircleAvatar(
                radius: 32,
                backgroundColor: theme.colorScheme.primaryContainer,
                foregroundImage: hasPhoto
                    ? MemoryImage(base64Decode(d!.avatarBase64))
                    : null,
                child: hasPhoto
                    ? null
                    : Text(name.characters.first,
                        style: TextStyle(
                          fontSize: 26,
                          fontWeight: FontWeight.bold,
                          color: theme.colorScheme.onPrimaryContainer,
                        )),
              ),
              Positioned(
                right: -2,
                bottom: -2,
                child: InkWell(
                  onTap: _uploadingAvatar ? null : () => _changeAvatar(),
                  borderRadius: BorderRadius.circular(20),
                  child: Container(
                    padding: const EdgeInsets.all(6),
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      color: theme.colorScheme.primary,
                    ),
                    child: _uploadingAvatar
                        ? const SizedBox(
                            width: 14,
                            height: 14,
                            child: CircularProgressIndicator(
                                strokeWidth: 2, color: Colors.white))
                        : const Icon(Icons.camera_alt,
                            size: 14, color: Colors.white),
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(d?.displayName ?? name,
                    style: theme.textTheme.titleMedium
                        ?.copyWith(fontWeight: FontWeight.w700)),
                Text(
                  hasPhoto ? '已上传照片，可用于数字孪生画像' : '点击相机上传照片，生成更好的数字孪生画像',
                  style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _changeAvatar() async {
    final messenger = ScaffoldMessenger.of(context);
    final picked = await pickPortraitPhoto();
    if (picked == null) return;
    setState(() => _uploadingAvatar = true);
    try {
      final ext = _mimeToExt(picked.mime);
      final res = await ApiService().uploadBytes(
        ApiConfig.avatarUpload,
        bytes: picked.bytes,
        filename: 'avatar.$ext',
        fieldName: 'file',
      );
      if (res.statusCode == 200) {
        // 刷新个人信息（含头像）
        await context.read<PersonalDetailProvider>().fetchAll();
        messenger.showSnackBar(
            const SnackBar(content: Text('头像已更新，可用于生成数字孪生画像')));
      } else {
        messenger.showSnackBar(
            const SnackBar(content: Text('头像上传失败')));
      }
    } catch (e) {
      messenger.showSnackBar(
          SnackBar(content: Text('头像上传失败：$e')));
    } finally {
      if (mounted) setState(() => _uploadingAvatar = false);
    }
  }

  String _mimeToExt(String mime) {
    switch (mime) {
      case 'image/png':
        return 'png';
      case 'image/webp':
        return 'webp';
      case 'image/gif':
        return 'gif';
      default:
        return 'jpg';
    }
  }

  // ── 基本信息 ──
  Widget _buildBasicTab(ThemeData theme, PersonalDetail? d) {
    if (d == null) return const Center(child: Text('暂无数据'));
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _row(theme, '登录账号', d.username),
        _row(theme, '姓名', d.displayName),
        _row(theme, '角色', _roleLabel(d.role)),
        _row(theme, '学院', d.college),
        _row(theme, '专业', d.major),
        _row(theme, '班级', d.className),
        _row(theme, '入学时间', d.enrollmentDate),
        _row(theme, '入学年份', d.enrollmentYear),
      ],
    );
  }

  // ── 联系方式 ──
  Widget _buildContactTab(ThemeData theme, PersonalDetail? d) {
    if (d == null) return const Center(child: Text('暂无数据'));
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _row(theme, '手机号', _empty(d.phone)),
        _row(theme, '微信号', _empty(d.wechat)),
        _row(theme, 'QQ', _empty(d.qq)),
        _row(theme, '邮箱', _empty(d.email)),
        const SizedBox(height: 8),
        Text(
          '注：联系方式由学校导入或本人完善，仅你本人可见。',
          style: theme.textTheme.bodySmall
              ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
        ),
      ],
    );
  }

  // ── 组织关系 ──
  Widget _buildOrgTab(ThemeData theme, PersonalDetailProvider p) {
    final d = p.detail;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text('上级 / 辅导员 / 领导',
            style: theme.textTheme.titleSmall
                ?.copyWith(fontWeight: FontWeight.w600)),
        const SizedBox(height: 8),
        if (d == null || d.supervisors.isEmpty)
          Text('暂无相关联系人',
              style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant))
        else
          ...d.supervisors.map((c) => _contactCard(theme, c)),
        if (d != null && d.subordinates > 0) ...[
          const SizedBox(height: 16),
          Text('管辖情况',
              style: theme.textTheme.titleSmall
                  ?.copyWith(fontWeight: FontWeight.w600)),
          const SizedBox(height: 8),
          Card(
            elevation: 0,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(10),
              side: BorderSide(color: theme.colorScheme.outlineVariant),
            ),
            child: ListTile(
              leading: const Icon(Icons.groups_outlined),
              title: Text('关联人数 ${d.subordinates}'),
              subtitle: const Text('由组织架构自动统计'),
            ),
          ),
        ],
        const Divider(height: 32),
        _buildPortalSection(theme, p),
      ],
    );
  }

  Widget _contactCard(ThemeData theme, ContactPerson c) {
    return Card(
      elevation: 0,
      margin: const EdgeInsets.only(bottom: 8),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(10),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: ListTile(
        leading: CircleAvatar(
          child: Text(c.name.isNotEmpty ? c.name.characters.first : '?'),
        ),
        title: Text(c.name),
        subtitle: Text([
          if (c.roleName.isNotEmpty) _roleLabel(c.roleName),
          if (c.phone.isNotEmpty) c.phone,
          if (c.wechat.isNotEmpty) '微信：${c.wechat}',
          if (c.email.isNotEmpty) c.email,
        ].join(' · ')),
      ),
    );
  }

  // ── 学校门户绑定 ──
  Widget _buildPortalSection(ThemeData theme, PersonalDetailProvider p) {
    final portal = p.portal;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('学校门户',
            style: theme.textTheme.titleSmall
                ?.copyWith(fontWeight: FontWeight.w600)),
        const SizedBox(height: 4),
        Text(
          '保存学校门户（my0.chzu.edu.cn）登录信息后，可从蔚小芯直接访问学校各级网站。凭证加密存储、私密不泄露。',
          style: theme.textTheme.bodySmall
              ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
        ),
        const SizedBox(height: 8),
        if (portal != null && portal.bound) ...[
          Card(
            elevation: 0,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(10),
              side: BorderSide(color: theme.colorScheme.outlineVariant),
            ),
            child: ListTile(
              leading: const Icon(Icons.lock_outline, color: Colors.green),
              title: Text('已绑定：${portal.portalAccount}'),
              subtitle: Text(portal.portalUrl),
              trailing: TextButton(
                onPressed: () => _confirmClearPortal(p),
                child: const Text('解除'),
              ),
            ),
          ),
          const SizedBox(height: 4),
          Align(
            alignment: Alignment.centerRight,
            child: TextButton.icon(
              onPressed: () {
                Navigator.pop(context);
                context.push('/portal');
              },
              icon: const Icon(Icons.open_in_new, size: 16),
              label: const Text('进入学校门户'),
            ),
          ),
        ] else
          _buildPortalForm(theme, p),
      ],
    );
  }

  Widget _buildPortalForm(ThemeData theme, PersonalDetailProvider p) {
    return Column(
      children: [
        TextField(
          controller: _portalUrl,
          decoration: const InputDecoration(
            labelText: '门户地址',
            isDense: true,
            border: OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 8),
        TextField(
          controller: _portalAccount,
          decoration: const InputDecoration(
            labelText: '门户账号（学号，已自动填入）',
            isDense: true,
            border: OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 8),
        TextField(
          controller: _portalPassword,
          obscureText: !_pwdVisible,
          decoration: InputDecoration(
            labelText: '门户密码',
            isDense: true,
            border: const OutlineInputBorder(),
            suffixIcon: IconButton(
              icon: Icon(_pwdVisible ? Icons.visibility_off : Icons.visibility,
                  size: 20),
              onPressed: () => setState(() => _pwdVisible = !_pwdVisible),
            ),
          ),
        ),
        const SizedBox(height: 8),
        SizedBox(
          width: double.infinity,
          child: FilledButton.icon(
            onPressed: () => _savePortal(p),
            icon: const Icon(Icons.lock_outline, size: 18),
            label: const Text('保存学校门户登录信息'),
          ),
        ),
      ],
    );
  }

  Future<void> _savePortal(PersonalDetailProvider p) async {
    final messenger = ScaffoldMessenger.of(context);
    final account = _portalAccount.text.trim();
    final password = _portalPassword.text.trim();
    if (account.isEmpty || password.isEmpty) {
      messenger.showSnackBar(
          const SnackBar(content: Text('请填写账号与密码')));
      return;
    }
    final ok = await p.savePortalCredential(
      portalUrl: _portalUrl.text.trim(),
      account: account,
      password: password,
    );
    messenger.showSnackBar(SnackBar(
        content: Text(ok
            ? '已保存，门户凭证加密存储，仅你本人可见'
            : p.error.isNotEmpty
                ? p.error
                : '保存失败')));
    if (ok) {
      _portalPassword.clear();
      setState(() {});
    }
  }

  void _confirmClearPortal(PersonalDetailProvider p) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('解除学校门户绑定'),
        content: const Text('确定清除已保存的门户登录信息？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(
            onPressed: () async {
              Navigator.pop(ctx);
              await p.clearPortalCredential();
            },
            child: const Text('解除'),
          ),
        ],
      ),
    );
  }

  Widget _row(ThemeData theme, String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 90,
            child: Text(label,
                style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant)),
          ),
          Expanded(
            child: Text(value.isEmpty ? '—' : value,
                style: theme.textTheme.bodyMedium
                    ?.copyWith(fontWeight: FontWeight.w500)),
          ),
        ],
      ),
    );
  }

  String _empty(String v) => v.isEmpty ? '—' : v;

  String _roleLabel(String role) {
    switch (role) {
      case 'student':
        return '学生';
      case 'student_union':
        return '学生会成员';
      case 'counselor':
        return '辅导员';
      case 'teacher':
        return '教师';
      case 'assistant':
        return '教辅人员';
      case 'college_admin':
        return '学院管理员';
      case 'school_admin':
        return '学校管理员';
      case 'sys_admin':
        return '系统管理员';
      case 'guest':
        return '游客';
      default:
        return role;
    }
  }
}
