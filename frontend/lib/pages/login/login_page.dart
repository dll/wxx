import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import 'package:dio/dio.dart';
import '../../config/api_config.dart';
import '../../providers/auth_provider.dart';
import '../../services/api_service.dart';
import '../../utils/storage.dart';

/// 登录页面 — 注册/登录分离，右上角按钮
class LoginPage extends StatefulWidget {
  const LoginPage({super.key});

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage>
    with SingleTickerProviderStateMixin {
  late final TabController _tabCtrl;
  bool _showLogin = false;

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
      appBar: AppBar(
        toolbarHeight: 48,
        automaticallyImplyLeading: false,
        backgroundColor: Colors.transparent,
        elevation: 0,
        actions: [
          TextButton(
            onPressed: () => _showGuestDialog(context),
            child:
                const Text('注册', style: TextStyle(fontWeight: FontWeight.w600)),
          ),
          TextButton(
            onPressed: () => setState(() => _showLogin = !_showLogin),
            child: Text('登录',
                style: TextStyle(
                  fontWeight: FontWeight.w600,
                  color: _showLogin ? theme.colorScheme.primary : null,
                )),
          ),
          const SizedBox(width: 8),
        ],
      ),
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 32),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: Column(
                mainAxisAlignment: _showLogin
                    ? MainAxisAlignment.start
                    : MainAxisAlignment.center,
                children: [
                  if (!_showLogin) const SizedBox(height: 60),
                  // Logo
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
                  const SizedBox(height: 36),

                  // 欢迎卡片（始终显示）
                  Container(
                    width: double.infinity,
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      gradient: LinearGradient(
                        colors: [
                          theme.colorScheme.primaryContainer,
                          theme.colorScheme.tertiaryContainer,
                        ],
                        begin: Alignment.topLeft,
                        end: Alignment.bottomRight,
                      ),
                      borderRadius: BorderRadius.circular(16),
                    ),
                    child: Column(
                      children: [
                        Icon(Icons.school,
                            size: 28, color: theme.colorScheme.primary),
                        const SizedBox(height: 8),
                        Text(
                          '欢迎来到滁州学院 👋',
                          style: theme.textTheme.titleMedium?.copyWith(
                            fontWeight: FontWeight.bold,
                            color: theme.colorScheme.onSurface,
                          ),
                        ),
                        const SizedBox(height: 4),
                        Text(
                          '26级新生 · 学生家长 · 中学教师 · 访客',
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                      ],
                    ),
                  ),

                  // 登录表单（点击「登录」后显示）
                  if (_showLogin) ...[
                    const SizedBox(height: 24),
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
                            fontWeight: FontWeight.w600, fontSize: 14),
                        dividerColor: Colors.transparent,
                        tabs: const [
                          Tab(text: '密码登录', height: 44),
                          Tab(text: '扫码登录', height: 44),
                        ],
                      ),
                    ),
                    const SizedBox(height: 24),
                    SizedBox(
                      height: 360,
                      child: TabBarView(
                        controller: _tabCtrl,
                        children: [
                          _PasswordLoginForm(
                              onGuestTap: () => _showGuestDialog(context)),
                          const _QRCodeLoginPanel(),
                        ],
                      ),
                    ),
                  ],
                  if (!_showLogin) const SizedBox(height: 60),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// 校验手机号格式（中国大陆 11 位）
bool _isValidPhone(String phone) {
  return RegExp(r'^1[3-9]\d{9}$').hasMatch(phone);
}

/// 弹出来宾注册对话框
void _showGuestDialog(BuildContext context) {
  final nameCtrl = TextEditingController();
  final phoneCtrl = TextEditingController();
  final codeCtrl = TextEditingController();
  bool loading = false;
  bool sendingCode = false;
  int countdown = 0;

  showDialog(
    context: context,
    barrierDismissible: false,
    builder: (ctx) {
      final theme = Theme.of(ctx);
      return StatefulBuilder(
        builder: (ctx, setDlgState) => Dialog(
          insetPadding:
              const EdgeInsets.symmetric(horizontal: 24, vertical: 40),
          shape:
              RoundedRectangleBorder(borderRadius: BorderRadius.circular(20)),
          child: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.fromLTRB(24, 28, 24, 24),
                  decoration: BoxDecoration(
                    gradient: LinearGradient(
                      colors: [
                        const Color(0xFF1565C0),
                        const Color(0xFF1976D2).withOpacity(0.8)
                      ],
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                    ),
                    borderRadius:
                        const BorderRadius.vertical(top: Radius.circular(20)),
                  ),
                  child: Column(
                    children: [
                      Container(
                        width: 56,
                        height: 56,
                        decoration: BoxDecoration(
                          color: Colors.white.withOpacity(0.2),
                          borderRadius: BorderRadius.circular(28),
                        ),
                        child: const Icon(Icons.school,
                            size: 32, color: Colors.white),
                      ),
                      const SizedBox(height: 12),
                      const Text('欢迎来到滁州学院',
                          style: TextStyle(
                              fontSize: 20,
                              fontWeight: FontWeight.bold,
                              color: Colors.white)),
                      const SizedBox(height: 4),
                      Text('验证手机号即可浏览公开信息',
                          style: TextStyle(
                              fontSize: 13,
                              color: Colors.white.withOpacity(0.85))),
                    ],
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.fromLTRB(24, 24, 24, 8),
                  child: Column(
                    children: [
                      TextField(
                        controller: nameCtrl,
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
                      TextField(
                        controller: phoneCtrl,
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
                      Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Expanded(
                            child: TextField(
                              controller: codeCtrl,
                              decoration: InputDecoration(
                                labelText: '验证码',
                                hintText: '6 位数字',
                                prefixIcon: const Icon(Icons.sms_outlined),
                                border: OutlineInputBorder(
                                    borderRadius: BorderRadius.circular(12)),
                                filled: true,
                                fillColor: theme
                                    .colorScheme.surfaceContainerHighest
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
                              onPressed: sendingCode || countdown > 0
                                  ? null
                                  : () async {
                                      final phone = phoneCtrl.text.trim();
                                      if (!_isValidPhone(phone)) {
                                        ScaffoldMessenger.of(context)
                                            .showSnackBar(
                                          const SnackBar(
                                              content: Text('请输入正确的 11 位手机号')),
                                        );
                                        return;
                                      }
                                      setDlgState(() => sendingCode = true);
                                      try {
                                        // 开发环境：后端返回随机 6 位数字，不实际发送短信
                                        // 生产环境：由第三方短信 SDK 发送，前端不返回 code
                                        final resp = await ApiService().post(
                                            ApiConfig.sendCode,
                                            data: {'phone': phone});
                                        final respCode =
                                            resp.data?['data']?['code'];
                                        if (respCode != null)
                                          codeCtrl.text = respCode.toString();
                                        setDlgState(() {
                                          sendingCode = false;
                                          countdown = 60;
                                        });
                                        if (respCode != null &&
                                            context.mounted) {
                                          ScaffoldMessenger.of(context)
                                              .showSnackBar(SnackBar(
                                            content: Text('验证码: $respCode'),
                                            duration:
                                                const Duration(seconds: 5),
                                            behavior: SnackBarBehavior.floating,
                                          ));
                                        }
                                        Future.doWhile(() async {
                                          await Future.delayed(
                                              const Duration(seconds: 1));
                                          if (ctx.mounted)
                                            setDlgState(() {
                                              if (countdown > 0) countdown--;
                                            });
                                          return countdown > 0;
                                        });
                                      } catch (e) {
                                        setDlgState(() => sendingCode = false);
                                        final msg = e is DioException
                                            ? (e.response?.data?['message'] ??
                                                '发送失败')
                                            : '发送失败';
                                        if (context.mounted) {
                                          ScaffoldMessenger.of(context)
                                              .showSnackBar(
                                                  SnackBar(content: Text(msg)));
                                        }
                                      }
                                    },
                              style: FilledButton.styleFrom(
                                shape: RoundedRectangleBorder(
                                    borderRadius: BorderRadius.circular(12)),
                                minimumSize: const Size(100, 56),
                              ),
                              child: sendingCode
                                  ? const SizedBox(
                                      width: 20,
                                      height: 20,
                                      child: CircularProgressIndicator(
                                          strokeWidth: 2, color: Colors.white))
                                  : Text(
                                      countdown > 0 ? '${countdown}s' : '获取验证码',
                                      style: const TextStyle(fontSize: 14)),
                            ),
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
                  child: Row(
                    children: [
                      Expanded(
                        child: OutlinedButton(
                          onPressed:
                              loading ? null : () => Navigator.of(ctx).pop(),
                          style: OutlinedButton.styleFrom(
                            shape: RoundedRectangleBorder(
                                borderRadius: BorderRadius.circular(12)),
                            minimumSize: const Size(0, 48),
                          ),
                          child: const Text('取消'),
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        flex: 2,
                        child: FilledButton(
                          onPressed: loading
                              ? null
                              : () async {
                                  final name = nameCtrl.text.trim();
                                  final phone = phoneCtrl.text.trim();
                                  final code = codeCtrl.text.trim();
                                  if (name.isEmpty) {
                                    ScaffoldMessenger.of(context).showSnackBar(
                                      const SnackBar(content: Text('请输入您的称呼')),
                                    );
                                    return;
                                  }
                                  if (!_isValidPhone(phone)) {
                                    ScaffoldMessenger.of(context).showSnackBar(
                                      const SnackBar(
                                          content: Text('请输入正确的 11 位手机号')),
                                    );
                                    return;
                                  }
                                  if (code.length != 6) {
                                    ScaffoldMessenger.of(context).showSnackBar(
                                      const SnackBar(
                                          content: Text('请输入 6 位验证码')),
                                    );
                                    return;
                                  }
                                  setDlgState(() => loading = true);
                                  try {
                                    final resp = await ApiService()
                                        .post(ApiConfig.guestRegister, data: {
                                      'display_name': name,
                                      'phone': phone,
                                      'code': code,
                                    });
                                    if (resp.data['code'] == 0 &&
                                        resp.data['data']?['token'] != null) {
                                      await Storage.setToken(
                                          resp.data['data']['token'] as String);
                                      await context
                                          .read<AuthProvider>()
                                          .fetchProfile();
                                      if (context.mounted) {
                                        Navigator.of(ctx).pop();
                                        context.go('/home');
                                      }
                                    } else {
                                      if (context.mounted) {
                                        ScaffoldMessenger.of(context)
                                            .showSnackBar(
                                          SnackBar(
                                              content: Text(
                                                  resp.data['message'] ??
                                                      '注册失败')),
                                        );
                                      }
                                    }
                                  } catch (e) {
                                    final msg = e is DioException
                                        ? (e.response?.data?['message'] ??
                                            '网络错误')
                                        : '网络错误';
                                    if (context.mounted) {
                                      ScaffoldMessenger.of(context)
                                          .showSnackBar(
                                              SnackBar(content: Text(msg)));
                                    }
                                  } finally {
                                    if (ctx.mounted)
                                      setDlgState(() => loading = false);
                                  }
                                },
                          style: FilledButton.styleFrom(
                            backgroundColor: const Color(0xFF1565C0),
                            shape: RoundedRectangleBorder(
                                borderRadius: BorderRadius.circular(12)),
                            minimumSize: const Size(0, 48),
                          ),
                          child: loading
                              ? const SizedBox(
                                  width: 20,
                                  height: 20,
                                  child: CircularProgressIndicator(
                                      strokeWidth: 2, color: Colors.white))
                              : const Text('开始浏览',
                                  style: TextStyle(
                                      fontSize: 15,
                                      fontWeight: FontWeight.w600)),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    },
  );
}

// ── 密码登录表单（含游客角色）──

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
    final auth = context.watch<AuthProvider>();
    return Column(
      children: [
        TextField(
          controller: _usernameCtrl,
          decoration: InputDecoration(
            labelText: '学号 / 工号',
            prefixIcon: const Icon(Icons.person_outline),
            border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
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
          icon: const Icon(Icons.app_registration),
          label: const Text('游客手机注册'),
        ),
      ],
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
        final encodedUrl = Uri.encodeComponent(
            'https://wxx.pydaydayup.xyz/#/login?qr=$sessionId');
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

  void _showFallbackQR(String message) {
    final encodedUrl =
        Uri.encodeComponent('https://wxx.pydaydayup.xyz/#/login');
    setState(() {
      _qrImageUrl =
          'https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=$encodedUrl&margin=10';
      _qrStatus = 'active';
      _message = message;
    });
  }

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
        final resp = await api
            .get('/api/v1/auth/qr-status', params: {'session': _qrSessionId!});
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
