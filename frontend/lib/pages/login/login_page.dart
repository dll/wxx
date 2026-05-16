import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import '../../providers/auth_provider.dart';
import '../../services/api_service.dart';
import '../../utils/storage.dart';

/// 登录页面 — 密码登录 + 扫码登录双模式
class LoginPage extends StatefulWidget {
  const LoginPage({super.key});

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> with SingleTickerProviderStateMixin {
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
            padding: const EdgeInsets.symmetric(horizontal: 32),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  // Logo
                  Icon(Icons.school, size: 72, color: theme.colorScheme.primary),
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
                    '信息学院智慧学工智能体',
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                  const SizedBox(height: 36),

                  // 选项卡
                  Container(
                    decoration: BoxDecoration(
                      color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.5),
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
                      labelStyle: const TextStyle(fontWeight: FontWeight.w600, fontSize: 14),
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
                      children: const [
                        _PasswordLoginForm(),
                        _QRCodeLoginPanel(),
                      ],
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

// ── 密码登录表单 ──

class _PasswordLoginForm extends StatefulWidget {
  const _PasswordLoginForm();

  @override
  State<_PasswordLoginForm> createState() => _PasswordLoginFormState();
}

class _PasswordLoginFormState extends State<_PasswordLoginForm> {
  final _usernameCtrl = TextEditingController();
  final _passwordCtrl = TextEditingController();
  bool _obscure = true;
  String _selectedRole = 'student';

  static const _roleOptions = [
    {'value': 'sys_admin', 'label': '系统管理员'},
    {'value': 'school_admin', 'label': '学校管理员'},
    {'value': 'college_admin', 'label': '学院管理员'},
    {'value': 'counselor', 'label': '辅导员'},
    {'value': 'teacher', 'label': '教师'},
    {'value': 'assistant', 'label': '教辅'},
    {'value': 'student_union', 'label': '学生会'},
    {'value': 'student', 'label': '学生'},
  ];

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
    final auth = context.read<AuthProvider>();
    final ok = await auth.login(username, _passwordCtrl.text.trim(), _selectedRole);
    if (!mounted) return;
    if (ok) {
      context.go('/chat');
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
          decoration: const InputDecoration(
            labelText: '学号 / 工号',
            prefixIcon: Icon(Icons.person_outline),
            border: OutlineInputBorder(),
          ),
          textInputAction: TextInputAction.next,
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _passwordCtrl,
          obscureText: _obscure,
          decoration: InputDecoration(
            labelText: '密码（开发环境可留空）',
            prefixIcon: const Icon(Icons.lock_outline),
            border: const OutlineInputBorder(),
            suffixIcon: IconButton(
              icon: Icon(_obscure ? Icons.visibility_off : Icons.visibility),
              onPressed: () => setState(() => _obscure = !_obscure),
            ),
          ),
          onSubmitted: (_) => _doLogin(),
        ),
        const SizedBox(height: 12),
        DropdownButtonFormField<String>(
          initialValue: _selectedRole,
          decoration: const InputDecoration(
            labelText: '角色（开发环境可选）',
            prefixIcon: Icon(Icons.badge_outlined),
            border: OutlineInputBorder(),
          ),
          items: _roleOptions.map((r) {
            return DropdownMenuItem(
              value: r['value'] as String,
              child: Text(r['label'] as String, style: const TextStyle(fontSize: 14)),
            );
          }).toList(),
          onChanged: (v) {
            if (v != null) setState(() => _selectedRole = v);
          },
        ),
        const SizedBox(height: 20),
        SizedBox(
          width: double.infinity,
          height: 48,
          child: FilledButton(
            onPressed: auth.loading ? null : _doLogin,
            child: auth.loading
                ? const SizedBox(
                    width: 24, height: 24,
                    child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                  )
                : const Text('登 录', style: TextStyle(fontSize: 16)),
          ),
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
  String _qrStatus = 'loading'; // loading | active | scanned | confirmed | expired
  Timer? _pollTimer;
  Timer? _expireTimer;
  int _pollFailureCount = 0;
  static const _maxPollFailures = 10; // 连续失败超过则停止轮询
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

  /// 生成 QR 会话并构建二维码图片 URL
  /// 安全：sessionID 完全由服务端生成，前端不参与
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
      // 调用后端创建 QR 会话，使用服务端返回的 session_id
      final api = ApiService();
      final resp = await api.post('/api/v1/auth/qr-login', data: <String, dynamic>{});
      if (!mounted) return;

      final code = resp.data['code'];
      final data = resp.data['data'];
      if (code == 0 && data is Map && data['session_id'] is String) {
        final sessionId = data['session_id'] as String;
        _qrSessionId = sessionId;
        final encodedUrl = Uri.encodeComponent('https://wxx.pydaydayup.xyz/#/login?qr=$sessionId');
        setState(() {
          _qrImageUrl = 'https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=$encodedUrl&margin=10';
          _qrStatus = 'active';
          _message = '请使用手机浏览器扫描二维码';
        });
        _startPolling();
      } else {
        // 后端 QR 端点异常，降级仅展示落地页
        _showFallbackQR('扫描二维码在手机上打开蔚小芯');
      }
    } catch (_) {
      if (!mounted) return;
      // 降级：直接生成访问码
      _showFallbackQR('扫描二维码在手机上打开蔚小芯');
    }
  }

  void _showFallbackQR(String message) {
    final encodedUrl = Uri.encodeComponent('https://wxx.pydaydayup.xyz/#/login');
    setState(() {
      _qrImageUrl = 'https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=$encodedUrl&margin=10';
      _qrStatus = 'active';
      _message = message;
    });
  }

  /// 轮询 QR 扫描状态
  void _startPolling() {
    _pollTimer?.cancel();
    final api = ApiService();
    // 在 timer 创建前 hoist 出 AuthProvider 引用，避免跨 async gap 用 context
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
        final resp = await api.get('/api/v1/auth/qr-status', params: {'session': _qrSessionId!});
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
                if (mounted) router.go('/chat');
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

    // 5 分钟后过期（用持有的 timer 引用避免 dispose 后泄漏）
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
        // QR 码容器
        Container(
          width: 240,
          height: 240,
          decoration: BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(
              color: _qrStatus == 'expired'
                  ? theme.colorScheme.error
                  : theme.colorScheme.outlineVariant.withValues(alpha: 0.4),
              width: _qrStatus == 'scanned' || _qrStatus == 'confirmed' ? 3 : 1,
            ),
            boxShadow: _qrStatus == 'scanned' || _qrStatus == 'confirmed'
                ? [
                    BoxShadow(
                      color: Colors.green.withValues(alpha: 0.4),
                      blurRadius: 20,
                      spreadRadius: 2,
                    ),
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
                              child: Icon(Icons.qr_code, size: 80, color: Colors.grey),
                            ),
                          ),
                        ),
                        if (_qrStatus == 'scanned')
                          Container(
                            width: 238,
                            height: 238,
                            decoration: BoxDecoration(
                              color: Colors.green.withValues(alpha: 0.15),
                              borderRadius: BorderRadius.circular(15),
                            ),
                            child: const Center(
                              child: Column(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  Icon(Icons.check_circle_outline, size: 48, color: Colors.green),
                                  SizedBox(height: 8),
                                  Text('已扫描', style: TextStyle(color: Colors.green, fontWeight: FontWeight.w600)),
                                ],
                              ),
                            ),
                          ),
                        if (_qrStatus == 'confirmed')
                          Container(
                            width: 238,
                            height: 238,
                            decoration: BoxDecoration(
                              color: Colors.green.withValues(alpha: 0.2),
                              borderRadius: BorderRadius.circular(15),
                            ),
                            child: const Center(
                              child: Column(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  Icon(Icons.check_circle, size: 48, color: Colors.green),
                                  SizedBox(height: 8),
                                  Text('登录成功', style: TextStyle(color: Colors.green, fontWeight: FontWeight.w600)),
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
        // 刷新按钮
        if (_qrStatus == 'expired')
          OutlinedButton.icon(
            onPressed: _generateQR,
            icon: const Icon(Icons.refresh, size: 18),
            label: const Text('刷新二维码'),
          ),
        const SizedBox(height: 16),
        Text(
          '手机扫码后打开蔚小芯，登录即可开始使用',
          style: theme.textTheme.labelSmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }

  Widget _buildExpiredOverlay(ThemeData theme) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.timer_off_outlined, size: 48, color: theme.colorScheme.error),
          const SizedBox(height: 8),
          Text('已过期', style: TextStyle(color: theme.colorScheme.error, fontWeight: FontWeight.w600)),
        ],
      ),
    );
  }
}
