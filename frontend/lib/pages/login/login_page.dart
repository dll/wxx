import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import 'package:dio/dio.dart';
import '../../config/api_config.dart';
import '../../config/release_config.dart';
import '../../providers/auth_provider.dart';
import '../../main.dart';
import '../../services/api_service.dart';
import '../../utils/storage.dart';

/// 游客手机注册是否开放（预研期关闭：无短信通道，账号由管理员导入分配）。
/// 需开放时改为 true，并同步后端 ENABLE_GUEST_REGISTER=true。
const bool _registerOpen = false;

/// 登录页面 — 居中布局，Tab 切换登录/注册
class LoginPage extends StatefulWidget {
  /// 扫码登录会话 ID：手机端通过二维码打开登录页时携带（#/login?qr=xxx），
  /// 登录成功后自动确认该 QR 会话，使 PC 端 Web 自动登录。
  final String? qrSessionId;

  const LoginPage({super.key, this.qrSessionId});

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage>
    with SingleTickerProviderStateMixin {
  late final TabController _tabCtrl;

  @override
  void initState() {
    super.initState();
    _tabCtrl = TabController(length: 2, vsync: this);
    // 扫码登录：手机端已登录时，扫码打开本页自动确认 QR 会话并返回
    final qrId = widget.qrSessionId;
    if (qrId != null && qrId.isNotEmpty && Storage.isLoggedIn) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _autoConfirmAndEnter(qrId);
      });
    }
  }

  /// 已登录用户扫码后自动确认 QR 会话并跳转首页（无需重复登录）
  Future<void> _autoConfirmAndEnter(String sessionId) async {
    try {
      await ApiService().post('/api/v1/auth/qr-confirm', data: {
        'session_id': sessionId,
      });
    } catch (_) {
      // 忽略确认错误，继续进入应用
    }
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('已确认登录，PC 端即将自动登录'),
        duration: Duration(seconds: 2),
      ),
    );
    context.go('/home');
  }

  @override
  void dispose() {
    _tabCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    // 年级主题色：登录页背景、Logo 与品牌区跟随全站主题
    final themeNotifier = context.watch<ThemeNotifier>();
    final accent = themeNotifier.gradeAccent;
    return Scaffold(
      body: Container(
        decoration: BoxDecoration(
          gradient: LinearGradient(
            colors: [
              Color.alphaBlend(
                  accent.withOpacity(0.10), theme.colorScheme.surface),
              theme.colorScheme.surface,
              theme.colorScheme.surface,
            ],
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
          ),
        ),
        child: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  // Logo
                  Center(
                    child: Column(
                      children: [
                        Container(
                          width: 88,
                          height: 88,
                          decoration: BoxDecoration(
                            color: accent.withOpacity(0.12),
                            borderRadius: BorderRadius.circular(26),
                            boxShadow: [
                              BoxShadow(
                                color: accent.withOpacity(0.18),
                                blurRadius: 24,
                                offset: const Offset(0, 8),
                              ),
                            ],
                          ),
                          child: Icon(Icons.school,
                              size: 46, color: accent),
                        ),
                        const SizedBox(height: 14),
                        Text(
                          '蔚小芯',
                          style: theme.textTheme.headlineMedium?.copyWith(
                            fontWeight: FontWeight.bold,
                            color: accent,
                          ),
                        ),
                        const SizedBox(height: 4),
                        Text(
                          '计算机学院智慧学工智能体',
                          style: theme.textTheme.bodyMedium?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 28),

                  // 登录/注册 Tab
                  Container(
                    decoration: BoxDecoration(
                      color: theme.colorScheme.surfaceContainerHighest
                          .withOpacity(0.5),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: TabBar(
                      controller: _tabCtrl,
                      indicator: BoxDecoration(
                        color: theme.colorScheme.primary,
                        borderRadius: BorderRadius.circular(10),
                      ),
                      indicatorSize: TabBarIndicatorSize.tab,
                      labelColor: theme.colorScheme.onPrimary,
                      unselectedLabelColor: theme.colorScheme.onSurfaceVariant,
                      labelStyle: const TextStyle(
                          fontWeight: FontWeight.w600, fontSize: 15),
                      dividerColor: Colors.transparent,
                      tabs: const [
                        Tab(text: '登 录', height: 46),
                        Tab(text: '注 册', height: 46),
                      ],
                    ),
                  ),
                  const SizedBox(height: 24),

                  // Tab 内容
                  SizedBox(
                    height: 320,
                    child: TabBarView(
                      controller: _tabCtrl,
                      children: [
                        _LoginTabContent(
                          qrSessionId: widget.qrSessionId,
                          onGuestTap: () => _tabCtrl.animateTo(1),
                        ),
                        _RegisterTabContent(
                          onLoginTap: () => _tabCtrl.animateTo(0),
                        ),
                      ],
                    ),
                  ),

                  const SizedBox(height: 6),

                  // 底部：下载按钮（用 app_icon 图标）+ 版本号
                  Center(
                    child: OutlinedButton.icon(
                      onPressed: () => _showApkQrDialog(context),
                      icon: Image.asset(
                        'assets/images/app_icon.png',
                        width: 24,
                        height: 24,
                        fit: BoxFit.contain,
                      ),
                      label: const Text(
                        '下载手机安卓版本',
                        style: TextStyle(
                          fontWeight: FontWeight.bold,
                          color: Color(0xFF3DDC84),
                        ),
                      ),
                      style: OutlinedButton.styleFrom(
                        side: const BorderSide(
                            color: Color(0xFF3DDC84), width: 2),
                        padding: const EdgeInsets.symmetric(
                            horizontal: 24, vertical: 12),
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(12),
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(height: 8),
                  Center(
                    child: Text(
                      'v${ReleaseConfig.version}',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color:
                            theme.colorScheme.onSurfaceVariant.withOpacity(0.5),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
        ),
      ),
    );
  }

  /// 显示 APK 下载二维码
  void _showApkQrDialog(BuildContext context) {
    final qrUrl = ReleaseConfig.qrCodeUrl(ReleaseConfig.apkDownloadUrl);
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('下载 Android 版'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 220,
              height: 220,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: Colors.grey.shade200),
              ),
              child: Image.network(
                qrUrl,
                fit: BoxFit.contain,
                errorBuilder: (_, __, ___) =>
                    const Center(child: Icon(Icons.qr_code, size: 80)),
              ),
            ),
            const SizedBox(height: 16),
            const Text(
              'v${ReleaseConfig.version}+${ReleaseConfig.buildNumber}',
              style: TextStyle(fontWeight: FontWeight.w600),
            ),
            const SizedBox(height: 4),
            const Text(
              '手机扫码下载安装',
              style: TextStyle(fontSize: 13, color: Colors.grey),
            ),
            const SizedBox(height: 12),
            // 下载链接：可长按/选中复制
            const SelectableText(
              ReleaseConfig.apkDownloadUrl,
              textAlign: TextAlign.center,
              style: TextStyle(fontSize: 11, color: Colors.blueGrey),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () {
              Clipboard.setData(
                  const ClipboardData(text: ReleaseConfig.apkDownloadUrl));
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(
                    content: Text('下载链接已复制'), duration: Duration(seconds: 2)),
              );
            },
            child: const Text('复制链接'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('关闭'),
          ),
        ],
      ),
    );
  }
}

/// 校验手机号格式（中国大陆 11 位）
bool _isValidPhone(String phone) {
  return RegExp(r'^1[3-9]\d{9}$').hasMatch(phone);
}

// ── 登录 Tab 内容（密码登录 / 扫码登录 子 Tab） ──

class _LoginTabContent extends StatefulWidget {
  final String? qrSessionId;
  final VoidCallback? onGuestTap;
  const _LoginTabContent({this.qrSessionId, this.onGuestTap});

  @override
  State<_LoginTabContent> createState() => _LoginTabContentState();
}

class _LoginTabContentState extends State<_LoginTabContent>
    with SingleTickerProviderStateMixin {
  late final TabController _subTabCtrl;

  @override
  void initState() {
    super.initState();
    _subTabCtrl = TabController(length: 2, vsync: this);
  }

  @override
  void dispose() {
    _subTabCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      children: [
        /// 子 Tab：密码登录 / 扫码登录
        Container(
          decoration: BoxDecoration(
            color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.3),
            borderRadius: BorderRadius.circular(10),
          ),
          child: TabBar(
            controller: _subTabCtrl,
            indicator: BoxDecoration(
              color: theme.colorScheme.primaryContainer,
              borderRadius: BorderRadius.circular(8),
            ),
            indicatorSize: TabBarIndicatorSize.tab,
            labelColor: theme.colorScheme.onPrimaryContainer,
            unselectedLabelColor: theme.colorScheme.onSurfaceVariant,
            labelStyle:
                const TextStyle(fontWeight: FontWeight.w600, fontSize: 14),
            dividerColor: Colors.transparent,
            tabs: const [
              Tab(text: '密码登录', height: 40),
              Tab(text: '扫码登录', height: 40),
            ],
          ),
        ),
        const SizedBox(height: 20),

        /// 子 Tab 内容
        Expanded(
          child: TabBarView(
            controller: _subTabCtrl,
            children: [
              SingleChildScrollView(
                child: _PasswordLoginForm(
                  qrSessionId: widget.qrSessionId,
                  onGuestTap: widget.onGuestTap,
                ),
              ),
              const SingleChildScrollView(
                child: _QRCodeLoginPanel(),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

// ── 密码登录表单 ──

class _PasswordLoginForm extends StatefulWidget {
  /// 扫码登录会话 ID：非空时登录成功后自动确认 QR 会话（PC 端自动登录）
  final String? qrSessionId;
  final VoidCallback? onGuestTap;
  const _PasswordLoginForm({this.qrSessionId, this.onGuestTap});

  @override
  State<_PasswordLoginForm> createState() => _PasswordLoginFormState();
}

class _PasswordLoginFormState extends State<_PasswordLoginForm> {
  final _usernameCtrl = TextEditingController();
  final _passwordCtrl = TextEditingController();
  bool _obscure = true;

  @override
  void dispose() {
    _usernameCtrl.dispose();
    _passwordCtrl.dispose();
    super.dispose();
  }

  /// 执行密码登录
  Future<void> _doLogin() async {
    final username = _usernameCtrl.text.trim();
    if (username.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请输入学号/工号')),
      );
      return;
    }
    final password = _passwordCtrl.text;
    if (password.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请输入密码')),
      );
      return;
    }
    final auth = context.read<AuthProvider>();
    final ok = await auth.login(username, password);
    if (!mounted) return;
    if (ok) {
      // 扫码登录：登录成功后确认 QR 会话，使 PC 端 Web 自动登录
      final qrId = widget.qrSessionId;
      if (qrId != null && qrId.isNotEmpty) {
        await _confirmQrSession(qrId);
        if (!mounted) return;
      }
      // 首次登录需强制修改密码（初始密码为学号，安全要求）
      if (auth.mustChangePassword) {
        final changed =
            await _forceChangePasswordDialog(context, auth, username);
        if (!mounted) return;
        if (changed != true) {
          // 未完成改密则退出登录，避免以初始密码进入
          await auth.logout();
          if (!mounted) return;
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('为保障账号安全，请先修改初始密码')),
          );
          return;
        }
      }
      context.go('/home');
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(auth.error ?? '登录失败')),
      );
    }
  }

  /// 强制修改初始密码对话框（不可关闭，改密成功或取消登录后才能退出）
  Future<bool?> _forceChangePasswordDialog(
      BuildContext context, AuthProvider auth, String username) {
    final oldCtrl = TextEditingController();
    final newCtrl = TextEditingController();
    final confirmCtrl = TextEditingController();
    final submitting = ValueNotifier<bool>(false);

    Future<void> submitChange(BuildContext dialogCtx) async {
      final oldPwd = oldCtrl.text.trim();
      final newPwd = newCtrl.text.trim();
      final confirm = confirmCtrl.text.trim();
      if (newPwd.length < 6) {
        ScaffoldMessenger.of(dialogCtx)
            .showSnackBar(const SnackBar(content: Text('新密码长度不能少于 6 位')));
        return;
      }
      if (newPwd != confirm) {
        ScaffoldMessenger.of(dialogCtx)
            .showSnackBar(const SnackBar(content: Text('两次输入的新密码不一致')));
        return;
      }
      submitting.value = true;
      final okChange = await auth.changePassword(oldPwd, newPwd);
      submitting.value = false;
      if (!dialogCtx.mounted) return;
      if (okChange) {
        ScaffoldMessenger.of(dialogCtx)
            .showSnackBar(const SnackBar(content: Text('密码修改成功')));
        Navigator.pop(dialogCtx, true);
      } else {
        ScaffoldMessenger.of(dialogCtx)
            .showSnackBar(SnackBar(content: Text(auth.error ?? '修改密码失败，请重试')));
      }
    }

    return showDialog<bool>(
      context: context,
      barrierDismissible: false,
      builder: (ctx) => ValueListenableBuilder<bool>(
        valueListenable: submitting,
        builder: (ctx, busy, _) {
          return AlertDialog(
            shape:
                RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
            title: const Text('首次登录 · 请修改初始密码'),
            content: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    '你的初始密码为学号。为保障账号安全，请先设置一个新密码（至少 6 位）。',
                    style: Theme.of(ctx).textTheme.bodySmall,
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: oldCtrl,
                    obscureText: true,
                    decoration: const InputDecoration(
                      labelText: '当前密码（学号）',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: newCtrl,
                    obscureText: true,
                    decoration: const InputDecoration(
                      labelText: '新密码（至少 6 位）',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: confirmCtrl,
                    obscureText: true,
                    decoration: const InputDecoration(
                      labelText: '确认新密码',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                    onSubmitted: (_) {
                      if (!busy) submitChange(ctx);
                    },
                  ),
                ],
              ),
            ),
            actions: [
              TextButton(
                onPressed: busy
                    ? null
                    : () async {
                        // 拒绝改密 → 退出登录
                        Navigator.pop(ctx, false);
                      },
                child: const Text('退出登录'),
              ),
              FilledButton(
                onPressed: busy ? null : () => submitChange(ctx),
                child: Text(busy ? '提交中...' : '确认修改'),
              ),
            ],
          );
        },
      ),
    );
  }

  /// 确认扫码登录会话（手机端登录成功后调用）
  /// 后端会从 Authorization 头读取当前已登录 token 写入 QR 会话，
  /// PC 端轮询到 confirmed 后自动登录。
  Future<void> _confirmQrSession(String sessionId) async {
    try {
      await ApiService().post('/api/v1/auth/qr-confirm', data: {
        'session_id': sessionId,
      });
    } catch (_) {
      // 确认失败不影响本地登录，PC 端可重新扫码
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final auth = context.watch<AuthProvider>();
    return Column(
      children: [
        TextField(
          controller: _usernameCtrl,
          decoration: InputDecoration(
            labelText: '学号 / 工号',
            prefixIcon: const Icon(Icons.person_outline),
            border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
            filled: true,
            fillColor:
                theme.colorScheme.surfaceContainerHighest.withOpacity(0.4),
          ),
          textInputAction: TextInputAction.next,
          keyboardType: TextInputType.text,
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _passwordCtrl,
          obscureText: _obscure,
          decoration: InputDecoration(
            labelText: '密码',
            prefixIcon: const Icon(Icons.lock_outline),
            border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
            filled: true,
            fillColor:
                theme.colorScheme.surfaceContainerHighest.withOpacity(0.4),
            suffixIcon: IconButton(
              icon: Icon(_obscure ? Icons.visibility_off : Icons.visibility),
              onPressed: () => setState(() => _obscure = !_obscure),
            ),
          ),
          onSubmitted: (_) => _doLogin(),
        ),
        const SizedBox(height: 20),
        SizedBox(
          width: double.infinity,
          height: 48,
          child: FilledButton(
            onPressed: auth.loading ? null : _doLogin,
            style: FilledButton.styleFrom(
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12)),
            ),
            child: auth.loading
                ? const SizedBox(
                    width: 24,
                    height: 24,
                    child: CircularProgressIndicator(
                        strokeWidth: 2, color: Colors.white))
                : const Text('登 录', style: TextStyle(fontSize: 16)),
          ),
        ),
        const SizedBox(height: 8),
        TextButton.icon(
          onPressed: widget.onGuestTap,
          icon: const Icon(Icons.app_registration, size: 18),
          label: const Text('没有账号？游客注册'),
        ),
      ],
    );
  }
}

// ── 注册 Tab 内容（游客注册表单） ──

class _RegisterTabContent extends StatefulWidget {
  final VoidCallback? onLoginTap;
  const _RegisterTabContent({this.onLoginTap});

  @override
  State<_RegisterTabContent> createState() => _RegisterTabContentState();
}

class _RegisterTabContentState extends State<_RegisterTabContent> {
  final _nameCtrl = TextEditingController();
  final _phoneCtrl = TextEditingController();
  final _codeCtrl = TextEditingController();
  bool _loading = false;
  bool _sendingCode = false;
  int _countdown = 0;
  Timer? _countdownTimer;

  @override
  void dispose() {
    _nameCtrl.dispose();
    _phoneCtrl.dispose();
    _codeCtrl.dispose();
    _countdownTimer?.cancel();
    super.dispose();
  }

  /// 获取验证码
  Future<void> _sendCode() async {
    final phone = _phoneCtrl.text.trim();
    if (!_isValidPhone(phone)) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请输入正确的 11 位手机号')),
      );
      return;
    }
    setState(() => _sendingCode = true);
    try {
      final resp = await ApiService().post(
        ApiConfig.sendCode,
        data: {'phone': phone},
      );
      final respCode = resp.data?['data']?['code'];
      if (respCode != null) {
        _codeCtrl.text = respCode.toString();
      }
      setState(() {
        _sendingCode = false;
        _countdown = 60;
      });
      if (respCode != null && mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
          content: Text('验证码: $respCode'),
          duration: const Duration(seconds: 5),
          behavior: SnackBarBehavior.floating,
        ));
      }
      _startCountdown();
    } catch (e) {
      setState(() => _sendingCode = false);
      final msg =
          e is DioException ? (e.response?.data?['message'] ?? '发送失败') : '发送失败';
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(msg)),
        );
      }
    }
  }

  /// 启动倒计时
  void _startCountdown() {
    _countdownTimer?.cancel();
    _countdownTimer = Timer.periodic(const Duration(seconds: 1), (timer) {
      if (!mounted) {
        timer.cancel();
        return;
      }
      setState(() {
        if (_countdown > 0) {
          _countdown--;
        } else {
          timer.cancel();
        }
      });
    });
  }

  /// 执行游客注册
  Future<void> _doRegister() async {
    final name = _nameCtrl.text.trim();
    final phone = _phoneCtrl.text.trim();
    final code = _codeCtrl.text.trim();

    if (name.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请输入您的称呼')),
      );
      return;
    }
    if (!_isValidPhone(phone)) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请输入正确的 11 位手机号')),
      );
      return;
    }
    if (code.length != 6) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请输入 6 位验证码')),
      );
      return;
    }

    setState(() => _loading = true);
    try {
      final resp = await ApiService().post(
        ApiConfig.guestRegister,
        data: {
          'display_name': name,
          'phone': phone,
          'code': code,
        },
      );
      if (resp.data['code'] == 0 && resp.data['data']?['token'] != null) {
        await Storage.setToken(resp.data['data']['token'] as String);
        if (mounted) {
          await context.read<AuthProvider>().fetchProfile();
          if (mounted) {
            context.go('/home');
          }
        }
      } else {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(resp.data['message'] ?? '注册失败')),
          );
        }
      }
    } catch (e) {
      final msg =
          e is DioException ? (e.response?.data?['message'] ?? '网络错误') : '网络错误';
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(msg)),
        );
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    // 预研期游客注册关闭：仅提示，不再展示注册表单
    if (!_registerOpen) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(Icons.lock_outline,
                  size: 48, color: theme.colorScheme.outline),
              const SizedBox(height: 16),
              Text('注册暂未开放',
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontWeight: FontWeight.bold)),
              const SizedBox(height: 8),
              Text(
                '账号由学校统一导入分配，请联系辅导员或管理员获取。',
                textAlign: TextAlign.center,
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
              ),
              const SizedBox(height: 20),
              OutlinedButton(
                onPressed: widget.onLoginTap,
                child: const Text('返回登录'),
              ),
            ],
          ),
        ),
      );
    }
    return SingleChildScrollView(
      child: Column(
        children: [
          /// 欢迎语
          Container(
            padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 16),
            decoration: BoxDecoration(
              color: theme.colorScheme.primaryContainer.withOpacity(0.5),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Row(
              children: [
                Icon(Icons.info_outline,
                    size: 20, color: theme.colorScheme.onPrimaryContainer),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    '验证手机号即可浏览公开信息',
                    style: TextStyle(
                      fontSize: 13,
                      color: theme.colorScheme.onPrimaryContainer,
                    ),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 20),

          /// 姓名输入
          TextField(
            controller: _nameCtrl,
            decoration: InputDecoration(
              labelText: '您的称呼',
              hintText: '如：王同学、李老师',
              prefixIcon: const Icon(Icons.person_outline),
              border:
                  OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
              filled: true,
              fillColor:
                  theme.colorScheme.surfaceContainerHighest.withOpacity(0.4),
            ),
          ),
          const SizedBox(height: 14),

          /// 手机号输入
          TextField(
            controller: _phoneCtrl,
            decoration: InputDecoration(
              labelText: '手机号',
              hintText: '11 位手机号',
              prefixIcon: const Icon(Icons.phone_outlined),
              border:
                  OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
              filled: true,
              fillColor:
                  theme.colorScheme.surfaceContainerHighest.withOpacity(0.4),
            ),
            keyboardType: TextInputType.phone,
            maxLength: 11,
          ),
          const SizedBox(height: 6),

          /// 验证码输入
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                child: TextField(
                  controller: _codeCtrl,
                  decoration: InputDecoration(
                    labelText: '验证码',
                    hintText: '6 位数字',
                    prefixIcon: const Icon(Icons.sms_outlined),
                    border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(12)),
                    filled: true,
                    fillColor: theme.colorScheme.surfaceContainerHighest
                        .withOpacity(0.4),
                  ),
                  keyboardType: TextInputType.number,
                  maxLength: 6,
                ),
              ),
              const SizedBox(width: 8),
              SizedBox(
                height: 56,
                child: FilledButton(
                  onPressed: _sendingCode || _countdown > 0 ? null : _sendCode,
                  style: FilledButton.styleFrom(
                    shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(12)),
                    minimumSize: const Size(100, 56),
                  ),
                  child: _sendingCode
                      ? const SizedBox(
                          width: 20,
                          height: 20,
                          child: CircularProgressIndicator(
                              strokeWidth: 2, color: Colors.white))
                      : Text(
                          _countdown > 0 ? '${_countdown}s' : '获取验证码',
                          style: const TextStyle(fontSize: 14),
                        ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 20),

          /// 注册按钮
          SizedBox(
            width: double.infinity,
            height: 48,
            child: FilledButton(
              onPressed: _loading ? null : _doRegister,
              style: FilledButton.styleFrom(
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(12)),
              ),
              child: _loading
                  ? const SizedBox(
                      width: 24,
                      height: 24,
                      child: CircularProgressIndicator(
                          strokeWidth: 2, color: Colors.white))
                  : const Text('开始浏览', style: TextStyle(fontSize: 16)),
            ),
          ),
          const SizedBox(height: 8),
          TextButton.icon(
            onPressed: widget.onLoginTap,
            icon: const Icon(Icons.login, size: 18),
            label: const Text('已有账号？去登录'),
          ),
        ],
      ),
    );
  }
}

// ── 扫码登录面板 ──

class _QRCodeLoginPanel extends StatefulWidget {
  const _QRCodeLoginPanel();

  @override
  State<_QRCodeLoginPanel> createState() => _QRCodeLoginPanelState();
}

class _QRCodeLoginPanelState extends State<_QRCodeLoginPanel> {
  String? _qrSessionId;
  String? _qrPollSecret;
  String? _qrImageUrl;
  String _qrStatus = 'loading';
  Timer? _pollTimer;
  Timer? _expireTimer;
  int _pollFailureCount = 0;
  static const _maxPollFailures = 10;
  String _message = '正在生成二维码...';

  @override
  void initState() {
    super.initState();
    _generateQR();
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    _expireTimer?.cancel();
    super.dispose();
  }

  /// 生成二维码
  Future<void> _generateQR() async {
    _pollTimer?.cancel();
    _expireTimer?.cancel();
    if (!mounted) return;
    setState(() {
      _qrStatus = 'loading';
      _message = '正在生成二维码...';
      _pollFailureCount = 0;
    });

    try {
      final api = ApiService();
      final resp =
          await api.post('/api/v1/auth/qr-login', data: <String, dynamic>{});
      if (!mounted) return;

      final code = resp.data['code'];
      final data = resp.data['data'];
      if (code == 0 && data is Map && data['session_id'] is String) {
        final sessionId = data['session_id'] as String;
        _qrSessionId = sessionId;
        _qrPollSecret = data['poll_secret'] as String?;
        // 二维码内容用 App Links URL：已安装 APK 的系统自动唤起本应用；
        // 未安装则手机浏览器打开 Web（经 Caddy 重定向到 #/login?qr=xxx）。
        setState(() {
          _qrImageUrl = ReleaseConfig.qrCodeUrl(
              '${ReleaseConfig.webUrl}/qr-login?qr=$sessionId');
          _qrStatus = 'active';
          _message = '请使用手机扫描二维码，已安装APP将自动唤起，否则打开网页';
        });
        _startPolling();
      } else {
        _showFallbackQR('扫描二维码在手机上打开蔚小芯');
      }
    } catch (_) {
      if (!mounted) return;
      _showFallbackQR('扫描二维码在手机上打开蔚小芯');
    }
  }

  /// 显示备用二维码
  void _showFallbackQR(String message) {
    setState(() {
      _qrImageUrl = ReleaseConfig.qrCodeUrl('${Uri.base.origin}/#/login');
      _qrStatus = 'active';
      _message = message;
    });
  }

  /// 开始轮询登录状态
  void _startPolling() {
    _pollTimer?.cancel();
    final api = ApiService();
    final authProvider = context.read<AuthProvider>();
    final router = GoRouter.of(context);

    _pollTimer = Timer.periodic(const Duration(seconds: 2), (timer) async {
      if (!mounted ||
          _qrSessionId == null ||
          _qrStatus == 'confirmed' ||
          _qrStatus == 'expired') {
        timer.cancel();
        return;
      }

      try {
        final resp = await api.get('/api/v1/auth/qr-status', params: {
          'session': _qrSessionId!,
          if (_qrPollSecret != null) 'poll_secret': _qrPollSecret!,
        });
        if (!mounted) return;
        _pollFailureCount = 0;

        if (resp.data['code'] == 0 && resp.data['data'] != null) {
          final qrData = resp.data['data'] as Map<String, dynamic>;
          if (qrData['status'] == 'scanned' && _qrStatus != 'scanned') {
            setState(() {
              _qrStatus = 'scanned';
              _message = '已扫描，请在手机上确认登录';
            });
          } else if (qrData['status'] == 'confirmed') {
            timer.cancel();
            _expireTimer?.cancel();
            setState(() {
              _qrStatus = 'confirmed';
              _message = '登录成功！正在跳转...';
            });
            if (qrData['token'] != null) {
              await Storage.setToken(qrData['token'] as String);
              await authProvider.fetchProfile();
            }
            if (mounted) {
              Future.delayed(const Duration(milliseconds: 500), () {
                if (mounted) router.go('/home');
              });
            }
          } else if (qrData['status'] == 'expired') {
            timer.cancel();
            _expireTimer?.cancel();
            if (mounted) {
              setState(() {
                _qrStatus = 'expired';
                _message = '二维码已过期，请点击刷新';
              });
            }
          }
        }
      } catch (_) {
        _pollFailureCount++;
        if (_pollFailureCount >= _maxPollFailures) {
          timer.cancel();
          if (mounted) {
            setState(() {
              _qrStatus = 'expired';
              _message = '网络异常，请点击刷新重试';
            });
          }
        }
      }
    });

    _expireTimer = Timer(const Duration(minutes: 5), () {
      if (!mounted) return;
      if (_qrStatus == 'active' || _qrStatus == 'scanned') {
        _pollTimer?.cancel();
        setState(() {
          _qrStatus = 'expired';
          _message = '二维码已过期，请点击刷新';
        });
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      children: [
        Container(
          width: 240,
          height: 240,
          decoration: BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(
              color: _qrStatus == 'expired'
                  ? theme.colorScheme.error
                  : theme.colorScheme.outlineVariant.withOpacity(0.4),
              width: _qrStatus == 'scanned' || _qrStatus == 'confirmed' ? 3 : 1,
            ),
            boxShadow: _qrStatus == 'scanned' || _qrStatus == 'confirmed'
                ? [
                    BoxShadow(
                        color: Colors.green.withOpacity(0.4),
                        blurRadius: 20,
                        spreadRadius: 2)
                  ]
                : null,
          ),
          child: _qrStatus == 'loading'
              ? const Center(child: CircularProgressIndicator(strokeWidth: 2))
              : _qrStatus == 'expired'
                  ? _buildExpiredOverlay(theme)
                  : Stack(
                      alignment: Alignment.center,
                      children: [
                        ClipRRect(
                          borderRadius: BorderRadius.circular(15),
                          child: Image.network(
                            _qrImageUrl!,
                            width: 238,
                            height: 238,
                            fit: BoxFit.contain,
                            errorBuilder: (_, __, ___) => const Center(
                                child: Icon(Icons.qr_code,
                                    size: 80, color: Colors.grey)),
                          ),
                        ),
                        if (_qrStatus == 'scanned')
                          Container(
                            width: 238,
                            height: 238,
                            decoration: BoxDecoration(
                                color: Colors.green.withOpacity(0.15),
                                borderRadius: BorderRadius.circular(15)),
                            child: const Center(
                              child: Column(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  Icon(Icons.check_circle_outline,
                                      size: 48, color: Colors.green),
                                  SizedBox(height: 8),
                                  Text('已扫描',
                                      style: TextStyle(
                                          color: Colors.green,
                                          fontWeight: FontWeight.w600)),
                                ],
                              ),
                            ),
                          ),
                        if (_qrStatus == 'confirmed')
                          Container(
                            width: 238,
                            height: 238,
                            decoration: BoxDecoration(
                                color: Colors.green.withOpacity(0.2),
                                borderRadius: BorderRadius.circular(15)),
                            child: const Center(
                              child: Column(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  Icon(Icons.check_circle,
                                      size: 48, color: Colors.green),
                                  SizedBox(height: 8),
                                  Text('登录成功',
                                      style: TextStyle(
                                          color: Colors.green,
                                          fontWeight: FontWeight.w600)),
                                ],
                              ),
                            ),
                          ),
                      ],
                    ),
        ),
        const SizedBox(height: 16),
        Text(_message, style: theme.textTheme.bodyMedium),
        const SizedBox(height: 12),
        if (_qrStatus == 'expired')
          OutlinedButton.icon(
            onPressed: _generateQR,
            icon: const Icon(Icons.refresh, size: 18),
            label: const Text('刷新二维码'),
          ),
        const SizedBox(height: 16),
        Text('手机扫码后打开蔚小芯，登录即可开始使用',
            style: theme.textTheme.labelSmall
                ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
      ],
    );
  }

  /// 构建过期覆盖层
  Widget _buildExpiredOverlay(ThemeData theme) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.timer_off_outlined,
              size: 48, color: theme.colorScheme.error),
          const SizedBox(height: 8),
          Text('已过期',
              style: TextStyle(
                  color: theme.colorScheme.error, fontWeight: FontWeight.w600)),
        ],
      ),
    );
  }
}
