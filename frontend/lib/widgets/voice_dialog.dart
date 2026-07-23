// ignore_for_file: avoid_web_libraries_in_flutter

import 'dart:async';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import '../providers/chat_provider.dart';
import '../services/voice/web_speech_recognizer.dart';

/// 语音助手对话框 — 双模式
/// 1. 导航模式：识别"去首页"等命令，自动跳转
/// 2. AI 对话模式：与 AI 助手实时语音对话（朗读 AI 回复）
Future<void> showVoiceDialog(BuildContext context) {
  return showDialog(
    context: context,
    barrierDismissible: false,
    builder: (ctx) => const _VoiceDialog(),
  );
}

class _VoiceDialog extends StatefulWidget {
  const _VoiceDialog();

  @override
  State<_VoiceDialog> createState() => _VoiceDialogState();
}

class _VoiceDialogState extends State<_VoiceDialog> with SingleTickerProviderStateMixin {
  // 模式：navigation（导航）/ assistant（AI助手）
  String _mode = 'navigation';
  // 状态：waking | listening | processing | thinking | speaking | done
  String _status = 'waking';

  String _interim = '';
  String _final = '';
  String _hint = '';
  final List<_VoiceTurn> _turns = [];

  late final WebSpeechRecognizer _recognizer;
  late final AnimationController _pulseCtrl;
  bool _disposed = false;
  Timer? _restartTimer;

  static const _commands = <String, String>{
    '首页': '/home',
    '对话': '/chat',
    '聊天': '/chat',
    '知识': '/browse',
    '办事': '/enrollment',
    '入学': '/enrollment',
    '我的': '/profile',
    '个人中心': '/profile',
    '设置': '/profile',
    '登录': '/login',
    '选课': '/enrollment',
    '情感': '/emotion',
    '智能体': '/agents',
    '问题反馈': '/feedback',
    '反馈': '/feedback',
    '收藏': '/bookmarks',
    '模型配置': '/profile/model-config',
    '数字孪生': '/student/digital-twin',
    '数字镜像': '/student/digital-twin',
  };

  @override
  void initState() {
    super.initState();
    _pulseCtrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1200),
    );

    _recognizer = WebSpeechRecognizer()
      ..onTranscript = _onTranscript
      ..onError = _onError
      ..onEnd = _onRecognitionEnd;

    _setHint();
    Future.delayed(const Duration(milliseconds: 400), _startListening);
  }

  @override
  void dispose() {
    _disposed = true;
    _restartTimer?.cancel();
    _pulseCtrl.dispose();
    _recognizer.dispose();

    // 停止 TTS（避免对话框关闭后还在朗读）
    if (mounted) {
      try {
        context.read<ChatProvider>().voice.stopPlayback();
      } catch (_) {}
    }
    super.dispose();
  }

  void _setHint() {
    setState(() {
      _hint = _mode == 'navigation'
          ? '想去哪？试试说："去首页"、"去对话"、"去我的"'
          : '与 AI 助手对话，比如："国家奖学金怎么申请？"';
    });
  }

  void _startListening() {
    if (_disposed) return;
    setState(() {
      _status = 'listening';
      _interim = '';
      _final = '';
    });
    _pulseCtrl.repeat(reverse: true);
    _recognizer.start(continuous: false);
  }

  void _onTranscript(String interim, String finalText, bool isFinal) {
    if (_disposed) return;
    setState(() {
      _interim = interim;
      if (finalText.isNotEmpty) _final = finalText;
    });
    if (isFinal && finalText.trim().isNotEmpty) {
      _processInput(finalText.trim());
    }
  }

  void _onError(String error) {
    if (_disposed) return;
    _scheduleRestart();
  }

  void _onRecognitionEnd() {
    if (_disposed) return;
    // 自动重启聆听（仅当当前处于聆听状态）
    if (_status == 'listening') {
      _scheduleRestart();
    }
  }

  void _scheduleRestart() {
    if (_disposed) return;
    _pulseCtrl.stop();
    _restartTimer?.cancel();
    _restartTimer = Timer(const Duration(milliseconds: 600), () {
      if (!_disposed && _status != 'thinking' && _status != 'speaking' && _status != 'done') {
        _startListening();
      }
    });
  }

  Future<void> _processInput(String text) async {
    if (_disposed) return;
    _recognizer.stop();
    _pulseCtrl.stop();
    setState(() => _status = 'processing');

    if (_mode == 'navigation') {
      _handleNavigation(text);
    } else {
      await _handleAssistantQuery(text);
    }
  }

  void _handleNavigation(String text) {
    final matched = _matchCommand(text);
    if (matched != null) {
      setState(() {
        _status = 'done';
        _hint = '已识别："${matched['label']}"，正在跳转...';
      });
      Future.delayed(const Duration(milliseconds: 600), () {
        if (!_disposed && mounted) {
          Navigator.pop(context);
          context.go(matched['route']!);
        }
      });
    } else {
      setState(() {
        _hint = '未识别到命令："$text"，请重新尝试';
      });
      _scheduleRestart();
    }
  }

  Future<void> _handleAssistantQuery(String text) async {
    if (!mounted) return;
    setState(() {
      _turns.add(_VoiceTurn(role: 'user', content: text));
      _status = 'thinking';
      _interim = '';
      _final = '';
    });

    try {
      final chat = context.read<ChatProvider>();
      // 在保留页面上下文的同时让 ChatProvider 处理调用 + 历史
      await chat.ask(text);
      if (_disposed || !mounted) return;

      // 取最后一条 AI 消息
      final messages = chat.messages;
      String reply = '';
      for (int i = messages.length - 1; i >= 0; i--) {
        if (!messages[i].isUser) {
          reply = messages[i].content;
          break;
        }
      }

      if (reply.isEmpty) {
        setState(() {
          _hint = '未获取到 AI 回复，请重试';
        });
        _scheduleRestart();
        return;
      }

      setState(() {
        _turns.add(_VoiceTurn(role: 'assistant', content: reply));
        _status = 'speaking';
        _hint = 'AI 助手正在回答...';
      });

      // 朗读回复（按句号软边界截断防止过长）
      final speakText = _truncateAtSentenceBoundary(reply, 250);
      await chat.voice.textToSpeech(speakText);

      if (_disposed || !mounted) return;
      setState(() => _hint = '继续提问');
      _startListening();
    } catch (e) {
      if (_disposed || !mounted) return;
      setState(() => _hint = '请求失败，请重试');
      _scheduleRestart();
    }
  }

  /// 在句号/换行等软边界截断长文本，避免 TTS 朗读时硬切到一半
  String _truncateAtSentenceBoundary(String text, int maxLen) {
    if (text.length <= maxLen) return text;
    final boundaries = ['。', '！', '？', '；', '\n', '. ', '! ', '? '];
    int cut = -1;
    for (final b in boundaries) {
      final idx = text.lastIndexOf(b, maxLen);
      if (idx > cut) cut = idx + b.length;
    }
    if (cut <= 0) cut = maxLen;
    return '${text.substring(0, cut)}...';
  }

  Map<String, String>? _matchCommand(String text) {    for (final entry in _commands.entries) {
      if (text.contains(entry.key)) {
        return {'label': entry.key, 'route': entry.value};
      }
    }
    final goPattern = RegExp(r'(?:去|打开|导航到|前往|跳到|进入)(.+)');
    final match = goPattern.firstMatch(text);
    if (match != null) {
      final target = match.group(1) ?? '';
      for (final entry in _commands.entries) {
        if (target.contains(entry.key)) {
          return {'label': entry.key, 'route': entry.value};
        }
      }
    }
    return null;
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Dialog(
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(24)),
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 460, maxHeight: 620),
        child: Padding(
          padding: const EdgeInsets.fromLTRB(20, 20, 20, 12),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              // 标题
              Row(
                children: [
                  Container(
                    width: 40, height: 40,
                    decoration: BoxDecoration(
                      color: const Color(0xFFE65100).withOpacity( 0.12),
                      shape: BoxShape.circle,
                    ),
                    child: const Icon(Icons.mic, color: Color(0xFFE65100)),
                  ),
                  const SizedBox(width: 12),
                  Text('语音助手', style: theme.textTheme.titleLarge),
                  const Spacer(),
                  IconButton(
                    icon: const Icon(Icons.close),
                    onPressed: () {
                      _recognizer.abort();
                      Navigator.pop(context);
                    },
                  ),
                ],
              ),
              const SizedBox(height: 8),
              // 模式切换
              SegmentedButton<String>(
                segments: const [
                  ButtonSegment(value: 'navigation', label: Text('语音导航'), icon: Icon(Icons.explore_outlined, size: 16)),
                  ButtonSegment(value: 'assistant', label: Text('AI 对话'), icon: Icon(Icons.chat_outlined, size: 16)),
                ],
                selected: {_mode},
                onSelectionChanged: (v) {
                  if (_status == 'thinking' || _status == 'speaking') return;
                  // 切换模式时停止可能正在朗读的 TTS
                  try {
                    context.read<ChatProvider>().voice.stopPlayback();
                  } catch (_) {}
                  setState(() {
                    _mode = v.first;
                    _turns.clear();
                  });
                  _setHint();
                  _recognizer.stop();
                  _restartTimer?.cancel();
                  _restartTimer = Timer(const Duration(milliseconds: 200), _startListening);
                },
                style: const ButtonStyle(
                  visualDensity: VisualDensity.compact,
                ),
              ),
              const SizedBox(height: 16),

              // 麦克风脉冲 + 状态
              SizedBox(
                height: 100,
                child: Center(
                  child: AnimatedBuilder(
                    animation: _pulseCtrl,
                    builder: (_, __) {
                      final isActive = _status == 'listening';
                      final scale = isActive ? 1.0 + 0.15 * _pulseCtrl.value : 1.0;
                      final color = _modeColor();
                      return Transform.scale(
                        scale: scale,
                        child: Container(
                          width: 72, height: 72,
                          decoration: BoxDecoration(
                            shape: BoxShape.circle,
                            color: isActive ? color : color.withOpacity( 0.18),
                            border: Border.all(color: color.withOpacity( 0.5), width: 2),
                          ),
                          child: Icon(
                            _statusIcon(),
                            color: isActive ? Colors.white : color,
                            size: 36,
                          ),
                        ),
                      );
                    },
                  ),
                ),
              ),
              const SizedBox(height: 8),
              Text(_statusText(), style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
                fontWeight: FontWeight.w500,
              )),
              const SizedBox(height: 4),
              Text(_hint, textAlign: TextAlign.center, style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
              )),
              const SizedBox(height: 12),

              // 实时识别 / AI 对话区域
              Expanded(
                child: Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.surfaceContainerHighest.withOpacity( 0.4),
                    borderRadius: BorderRadius.circular(14),
                  ),
                  child: _mode == 'assistant' && _turns.isNotEmpty
                      ? _buildConversation(theme)
                      : _buildLiveTranscript(theme),
                ),
              ),
              const SizedBox(height: 12),

              // 操作按钮
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton.icon(
                      onPressed: _stopAll,
                      icon: const Icon(Icons.pause_circle_outline, size: 18),
                      label: const Text('暂停聆听'),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: FilledButton.icon(
                      onPressed: _restartListening,
                      icon: const Icon(Icons.refresh, size: 18),
                      label: const Text('重新聆听'),
                      style: FilledButton.styleFrom(backgroundColor: _modeColor()),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildLiveTranscript(ThemeData theme) {
    final hasContent = _final.isNotEmpty || _interim.isNotEmpty;
    if (!hasContent) {
      return Center(
        child: Text(
          _mode == 'navigation' ? '请说出您要前往的页面' : '请说出您想询问的问题',
          style: theme.textTheme.bodyMedium?.copyWith(
            color: theme.colorScheme.outline,
          ),
        ),
      );
    }
    return SingleChildScrollView(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (_final.isNotEmpty)
            Text(_final, style: theme.textTheme.bodyLarge?.copyWith(
              fontWeight: FontWeight.w500,
            )),
          if (_interim.isNotEmpty)
            Text(_interim, style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.outline,
              fontStyle: FontStyle.italic,
            )),
        ],
      ),
    );
  }

  Widget _buildConversation(ThemeData theme) {
    return ListView.separated(
      itemCount: _turns.length + ((_interim.isNotEmpty || _final.isNotEmpty) ? 1 : 0),
      separatorBuilder: (_, __) => const SizedBox(height: 8),
      itemBuilder: (context, index) {
        if (index == _turns.length) {
          // 当前正在识别的内容
          return _bubble(
            isUser: true,
            text: _final.isNotEmpty ? _final : _interim,
            theme: theme,
            isInterim: _interim.isNotEmpty && _final.isEmpty,
          );
        }
        final turn = _turns[index];
        return _bubble(
          isUser: turn.role == 'user',
          text: turn.content,
          theme: theme,
        );
      },
    );
  }

  Widget _bubble({
    required bool isUser,
    required String text,
    required ThemeData theme,
    bool isInterim = false,
  }) {
    return Align(
      alignment: isUser ? Alignment.centerRight : Alignment.centerLeft,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        constraints: const BoxConstraints(maxWidth: 320),
        decoration: BoxDecoration(
          color: isUser
              ? theme.colorScheme.primary.withOpacity( isInterim ? 0.4 : 1.0)
              : theme.colorScheme.surface,
          borderRadius: BorderRadius.circular(12),
          border: !isUser
              ? Border.all(color: theme.colorScheme.outlineVariant.withOpacity( 0.4))
              : null,
        ),
        child: Text(
          text,
          style: TextStyle(
            color: isUser ? theme.colorScheme.onPrimary : theme.colorScheme.onSurface,
            fontSize: 14,
            fontStyle: isInterim ? FontStyle.italic : FontStyle.normal,
          ),
        ),
      ),
    );
  }

  void _stopAll() {
    _recognizer.abort();
    _restartTimer?.cancel();
    _pulseCtrl.stop();
    setState(() {
      _status = 'paused';
      _hint = '已暂停聆听，点击"重新聆听"继续';
    });
  }

  void _restartListening() {
    _recognizer.abort();
    _restartTimer?.cancel();
    setState(() {
      _interim = '';
      _final = '';
    });
    _setHint();
    _startListening();
  }

  Color _modeColor() => _mode == 'navigation'
      ? const Color(0xFFE65100)
      : const Color(0xFF6750A4);

  IconData _statusIcon() {
    switch (_status) {
      case 'listening':
        return Icons.mic;
      case 'thinking':
      case 'processing':
        return Icons.psychology;
      case 'speaking':
        return Icons.volume_up;
      case 'done':
        return Icons.check_circle;
      case 'paused':
        return Icons.pause;
      default:
        return Icons.mic_none;
    }
  }

  String _statusText() {
    switch (_status) {
      case 'waking':
        return '正在唤醒...';
      case 'listening':
        return '正在聆听';
      case 'processing':
        return '识别中...';
      case 'thinking':
        return 'AI 思考中...';
      case 'speaking':
        return 'AI 朗读回复';
      case 'done':
        return '完成';
      case 'paused':
        return '已暂停';
      default:
        return '';
    }
  }
}

class _VoiceTurn {
  final String role;
  final String content;
  _VoiceTurn({required this.role, required this.content});
}
