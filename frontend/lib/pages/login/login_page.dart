import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import 'package:dio/dio.dart';
import '../../config/api_config.dart';
import '../../config/release_config.dart';
import '../../providers/auth_provider.dart';
import '../../services/api_service.dart';
import '../../utils/storage.dart';

/// 登录页面 — 居中布局，Tab 切换登录/注册
class LoginPage extends StatefulWidget {
  const LoginPage({super.key});

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
  }

  @override
  void dispose() {
    _tabCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 32),
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
                        Icon(Icons.school,
                            size: 72, color: theme.colorScheme.primary),
                        const SizedBox(height: 12),
                        Text(
                          '蔚小芯',
                          style: theme.textTheme.headlineMedium?.copyWith(
                            fontWeight: FontWeight.bold,
                            color: theme.colorScheme.primary,
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
                  const SizedBox(height: 36),

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
                      unselectedLabelColor:
                          theme.colorScheme.onSurfaceVariant,
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
                    height: 420,
                    child: TabBarView(
                      controller: _tabCtrl,
                      children: [
                        _LoginTabContent(
                          onGuestTap: () => _tabCtrl.animateTo(1),
                        ),
                        _RegisterTabContent(
                          onLoginTap: () => _tabCtrl.animateTo(0),
                        ),
                      ],
                    ),
                  ),

                  const SizedBox(height: 32),

                  // 底部：版本号 + 下载
                  Center(
                    child: OutlinedButton.icon(
                      onPressed: () => _showDownloadSheet(context),
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
                        side: const BorderSide(color: Color(0xFF3DDC84), width: 2),
                        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
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
                        color: theme.colorScheme.onSurfaceVariant
                            .withOpacity(0.5),
                      ),
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

  /// 显示下载方式选择
  void _showDownloadSheet(BuildContext context) {
    final theme = Theme.of(context);
    showModalBottomSheet(
      context: context,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (ctx) => Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Center(
              child: Container(
                width: 40,
                height: 4,
                decoration: BoxDecoration(
                  color: theme.colorScheme.outlineVariant,
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
            ),
            const SizedBox(height: 20),
            Text(
              '选择版本',
              style: theme.textTheme.titleLarge?.copyWith(
                fontWeight: FontWeight.bold,
              ),
            ),
            const SizedBox(height: 20),
            ListTile(
              leading: Container(
                width: 48,
                height: 48,
                decoration: BoxDecoration(
                  color: Colors.blue.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: const Icon(Icons.language, color: Colors.blue),
              ),
              title: const Text('Web 版',
                  style: TextStyle(fontWeight: FontWeight.w600)),
              subtitle: const Text('浏览器直接使用，无需下载'),
              trailing: const Icon(Icons.open_in_new),
              onTap: () => Navigator.pop(ctx),
            ),
            const SizedBox(height: 8),
            ListTile(
              leading: Container(
                width: 48,
                height: 48,
                decoration: BoxDecoration(
                  color: Colors.green.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: const Icon(Icons.android, color: Colors.green),
              ),
              title: const Text('Android 版',
                  style: TextStyle(fontWeight: FontWeight.w600)),
              subtitle: const Text('扫码下载 APK 安装包'),
              trailing: const Icon(Icons.qr_code),
              onTap: () {
                Navigator.pop(ctx);
                _showApkQrDialog(context);
              },
            ),
            const SizedBox(height: 16),
          ],
        ),
      ),
    );
  }

  /// 显示 APK 下载二维码
  void _showApkQrDialog(BuildContext context) {
    final qrUrl =
        'https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=${Uri.encodeComponent(ReleaseConfig.apkDownloadUrl)}&margin=10';
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
            Text(
              'v${ReleaseConfig.version}+${ReleaseConfig.buildNumber}',
              style: const TextStyle(fontWeight: FontWeight.w600),
            ),
            const SizedBox(height: 4),
            const Text(
              '手机扫码下载安装',
              style: TextStyle(fontSize: 13, color: Colors.grey),
            ),
          ],
        ),
        actions: [
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
  final VoidCallback? onGuestTap;
  const _LoginTabContent({this.onGuestTap});

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
                child: _PasswordLoginForm(onGuestTap: widget.onGuestTap),
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
  final VoidCallback? onGuestTap;
  const _PasswordLoginForm({this.onGuestTap});

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
      context.go('/home');
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(auth.error ?? '登录失败')),
      );
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
            fillColor: theme.colorScheme.surfaceContainerHighest
                .withOpacity(0.4),
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
            fillColor: theme.colorScheme.surfaceContainerHighest
                .withOpacity(0.4),
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
      final msg = e is DioException
          ? (e.response?.data?['message'] ?? '发送失败')
          : '发送失败';
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
      final msg = e is DioException
          ? (e.response?.data?['message'] ?? '网络错误')
          : '网络错误';
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
              border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(12)),
              filled: true,
              fillColor: theme.colorScheme.surfaceContainerHighest
                  .withOpacity(0.4),
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
              border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(12)),
              filled: true,
              fillColor: theme.colorScheme.surfaceContainerHighest
                  .withOpacity(0.4),
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
        final encodedUrl = Uri.encodeComponent(
            'https://wxx-agent.pages.dev/#/login?qr=$sessionId');
        setState(() {
          _qrImageUrl =
              'https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=$encodedUrl&margin=10';
          _qrStatus = 'active';
          _message = '请使用手机浏览器扫描二维码';
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
    final encodedUrl =
        Uri.encodeComponent('https://wxx-agent.pages.dev/#/login');
    setState(() {
      _qrImageUrl =
          'https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=$encodedUrl&margin=10';
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
