import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../config/api_config.dart';
import '../../providers/personal_detail_provider.dart';
import '../../utils/portal_html_view.dart';
import '../../utils/storage.dart';

/// 学校门户 — 填写账号密码后即可访问校内网站
class PortalBrowsePage extends StatefulWidget {
  const PortalBrowsePage({super.key});

  @override
  State<PortalBrowsePage> createState() => _PortalBrowsePageState();
}

class _PortalBrowsePageState extends State<PortalBrowsePage> {
  final TextEditingController _pathCtrl = TextEditingController(text: '/');
  final TextEditingController _accountCtrl = TextEditingController();
  final TextEditingController _pwdCtrl = TextEditingController();
  bool _pwdVisible = false;
  bool _saving = false;
  String _path = '/';
  bool _loading = false;
  String _error = '';
  String _body = '';
  String _contentType = 'text/html';

  @override
  void initState() {
    super.initState();
    // 账号默认当前登录账号（学号）
    if (Storage.username != null && Storage.username!.isNotEmpty) {
      _accountCtrl.text = Storage.username!;
    }
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _load('/');
    });
  }

  @override
  void dispose() {
    _pathCtrl.dispose();
    _accountCtrl.dispose();
    _pwdCtrl.dispose();
    super.dispose();
  }

  Future<void> _load(String path) async {
    final p = context.read<PersonalDetailProvider>();
    // 确保个人资料（含门户绑定状态）已加载
    if (p.detail == null) await p.fetchAll();
    final portal = p.portal;
    if (portal == null || !portal.bound) {
      setState(() => _error = '');
      return; // 未绑定 → 显示填写表单
    }
    setState(() {
      _loading = true;
      _error = '';
    });
    final result = await p.proxyPortal(path);
    if (!mounted) return;
    setState(() {
      _loading = false;
      if (result == null) {
        _error = p.error.isNotEmpty ? p.error : '门户访问失败';
      } else {
        _body = result.body;
        _contentType = result.contentType;
      }
    });
  }

  /// 重新填写账号密码（清除绑定回到表单）
  Future<void> _rebind() async {
    final p = context.read<PersonalDetailProvider>();
    await p.clearPortalCredential();
    if (mounted) {
      setState(() {
        _error = '';
        _body = '';
      });
    }
  }

  Future<void> _saveAndEnter() async {
    final messenger = ScaffoldMessenger.of(context);
    final account = _accountCtrl.text.trim();
    final password = _pwdCtrl.text.trim();
    if (account.isEmpty || password.isEmpty) {
      messenger.showSnackBar(const SnackBar(content: Text('请填写门户账号和密码')));
      return;
    }
    setState(() => _saving = true);
    final p = context.read<PersonalDetailProvider>();
    final ok = await p.savePortalCredential(
      portalUrl: ApiConfig.schoolPortalUrl,
      account: account,
      password: password,
    );
    if (!mounted) return;
    setState(() => _saving = false);
    if (ok) {
      messenger.showSnackBar(
          const SnackBar(content: Text('门户绑定成功，正在进入…')));
      _pwdCtrl.clear();
      _load('/');
    } else {
      messenger.showSnackBar(SnackBar(
          content: Text(p.error.isNotEmpty ? p.error : '保存失败')));
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final p = context.watch<PersonalDetailProvider>();
    final bound = p.portal?.bound ?? false;

    return Scaffold(
      appBar: AppBar(title: const Text('学校门户')),
      body: bound ? _buildPortalBody(theme) : _buildBindForm(theme),
    );
  }

  // ── 已绑定：门户浏览 ──
  Widget _buildPortalBody(ThemeData theme) {
    return Column(
      children: [
        // 地址栏
        Padding(
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 8),
          child: Row(
            children: [
              Expanded(
                child: TextField(
                  controller: _pathCtrl,
                  decoration: const InputDecoration(
                    hintText: '路径：/home  /edu/course  /library',
                    isDense: true,
                    prefixIcon: Icon(Icons.link, size: 18),
                    border: OutlineInputBorder(),
                  ),
                  onSubmitted: (v) {
                    final q = v.trim().startsWith('/') ? v.trim() : '/${v.trim()}';
                    _pathCtrl.text = q;
                    setState(() => _path = q);
                    _load(q);
                  },
                ),
              ),
              const SizedBox(width: 8),
              IconButton(
                tooltip: '刷新',
                onPressed: () => _load(_path),
                icon: const Icon(Icons.refresh),
              ),
            ],
          ),
        ),
        const Divider(height: 1),
        Expanded(
          child: _loading
              ? const Center(child: CircularProgressIndicator())
              : _error.isNotEmpty
                  ? Center(
                      child: Padding(
                        padding: const EdgeInsets.all(24),
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Icon(Icons.cloud_off_outlined,
                                size: 48, color: theme.colorScheme.outline),
                            const SizedBox(height: 12),
                            Text(_error, textAlign: TextAlign.center),
                            const SizedBox(height: 12),
                            if (_error.contains('登录') ||
                                _error.contains('密码'))
                              OutlinedButton.icon(
                                onPressed: () => _rebind(),
                                icon: const Icon(Icons.edit, size: 18),
                                label: const Text('重新填写账号密码'),
                              )
                            else
                              FilledButton.tonal(
                                onPressed: () => _load(_path),
                                child: const Text('重试'),
                              ),
                          ],
                        ),
                      ),
                    )
                  : _body.isEmpty
                      ? const Center(child: Text('（空白页面）'))
                      : _contentType.contains('text/html')
                          ? portalHtmlView(_body)
                          : SingleChildScrollView(
                              padding: const EdgeInsets.all(16),
                              child:
                                  Text(_body, style: theme.textTheme.bodySmall),
                            ),
        ),
      ],
    );
  }

  // ── 未绑定：填写账号密码 ──
  Widget _buildBindForm(ThemeData theme) {
    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 420),
          child: Card(
            elevation: 0,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(16),
              side: BorderSide(color: theme.colorScheme.outlineVariant),
            ),
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Icon(Icons.school, color: theme.colorScheme.primary),
                      const SizedBox(width: 8),
                      Text('绑定学校门户',
                          style: theme.textTheme.titleMedium
                              ?.copyWith(fontWeight: FontWeight.w700)),
                    ],
                  ),
                  const SizedBox(height: 6),
                  Text(
                    '填写学校门户（my0.chzu.edu.cn）的账号和密码，即可访问校内各级网站（学工、一表通等）。账号即学号，密码加密存储，仅你本人可见。',
                    style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: _accountCtrl,
                    decoration: const InputDecoration(
                      labelText: '门户账号（学号）',
                      isDense: true,
                      border: OutlineInputBorder(),
                    ),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _pwdCtrl,
                    obscureText: !_pwdVisible,
                    decoration: InputDecoration(
                      labelText: '门户密码',
                      isDense: true,
                      border: const OutlineInputBorder(),
                      suffixIcon: IconButton(
                        icon: Icon(_pwdVisible
                            ? Icons.visibility_off
                            : Icons.visibility,
                            size: 20),
                        onPressed: () =>
                            setState(() => _pwdVisible = !_pwdVisible),
                      ),
                    ),
                  ),
                  const SizedBox(height: 16),
                  SizedBox(
                    width: double.infinity,
                    child: FilledButton.icon(
                      onPressed: _saving ? null : _saveAndEnter,
                      icon: _saving
                          ? const SizedBox(
                              width: 16,
                              height: 16,
                              child:
                                  CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Icon(Icons.login, size: 18),
                      label: Text(_saving ? '绑定中…' : '绑定并进入门户'),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
