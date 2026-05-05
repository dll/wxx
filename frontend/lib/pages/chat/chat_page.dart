import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import '../../models/models.dart';
import '../../providers/chat_provider.dart';
import '../../services/voice/voice_navigator.dart';
import '../../widgets/answer_card.dart';

/// 对话主页
class ChatPage extends StatefulWidget {
  /// 从知识大厅跳转时携带的初始问题
  final String? initialQuestion;

  const ChatPage({super.key, this.initialQuestion});

  @override
  State<ChatPage> createState() => _ChatPageState();
}

class _ChatPageState extends State<ChatPage> {
  final _inputCtrl = TextEditingController();
  final _scrollCtrl = ScrollController();
  bool _initialQuestionHandled = false;

  @override
  void initState() {
    super.initState();
    // 延迟处理初始问题（等 Widget 构建完成后发送）
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _handleInitialQuestion();
    });
  }

  /// 处理从知识大厅跳转带来的初始问题
  void _handleInitialQuestion() {
    if (_initialQuestionHandled) return;
    final question = widget.initialQuestion;
    if (question != null && question.isNotEmpty) {
      _initialQuestionHandled = true;
      _inputCtrl.text = question;
      _send();
    }
  }

  @override
  void dispose() {
    _inputCtrl.dispose();
    _scrollCtrl.dispose();
    super.dispose();
  }

  void _send() {
    final text = _inputCtrl.text.trim();
    if (text.isEmpty) return;

    _inputCtrl.clear();
    context.read<ChatProvider>().ask(text);

    // 滚动到底部
    Future.delayed(const Duration(milliseconds: 100), _scrollToBottom);
  }

  /// 处理语音录制结果：先检查导航指令，否则作为聊天消息发送
  Future<void> _handleVoiceResult() async {
    final chat = context.read<ChatProvider>();
    if (!chat.isRecording) return;

    chat.stopRecording();

    final audioData = await chat.voice.stopRecording();
    if (audioData == null) return;

    final text = await chat.voice.speechToText(audioData);
    if (text == null || text.trim().isEmpty) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('未识别到语音内容，请重试')),
        );
      }
      return;
    }

    // 检查语音导航指令
    final route = VoiceNavigator.matchCommand(text);
    if (route != null && mounted) {
      context.go(route);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('已识别指令：${text.trim()}'),
          duration: const Duration(seconds: 1),
        ),
      );
      return;
    }

    // 普通对话消息
    chat.ask(text.trim());
  }

  void _scrollToBottom() {
    if (_scrollCtrl.hasClients) {
      _scrollCtrl.animateTo(
        _scrollCtrl.position.maxScrollExtent + 100,
        duration: const Duration(milliseconds: 300),
        curve: Curves.easeOut,
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final chat = context.watch<ChatProvider>();
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('蔚小芯'),
        actions: [
          IconButton(
            icon: const Icon(Icons.add_comment_outlined),
            tooltip: '新对话',
            onPressed: () => chat.newChat(),
          ),
        ],
      ),
      body: Column(
        children: [
          // 消息列表
          Expanded(
            child: chat.messages.isEmpty
                ? _buildEmptyState(theme)
                : ListView.builder(
                    controller: _scrollCtrl,
                    padding: const EdgeInsets.symmetric(vertical: 8),
                    itemCount: chat.messages.length + (chat.sending ? 1 : 0),
                    itemBuilder: (context, index) {
                      // 加载指示器
                      if (index == chat.messages.length && chat.sending) {
                        return _buildLoadingBubble(theme);
                      }
                      return _buildMessage(chat.messages[index]);
                    },
                  ),
          ),

          // 错误提示
          if (chat.error != null)
            Container(
              width: double.infinity,
              color: theme.colorScheme.errorContainer,
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
              child: Text(
                chat.error!,
                style: TextStyle(color: theme.colorScheme.onErrorContainer, fontSize: 13),
              ),
            ),

          // 输入区域
          _buildInputBar(theme, chat.sending),
        ],
      ),
    );
  }

  Widget _buildEmptyState(ThemeData theme) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.school, size: 64, color: theme.colorScheme.primary.withValues(alpha: 0.3)),
          const SizedBox(height: 16),
          Text(
            '你好！我是蔚小芯',
            style: theme.textTheme.titleMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            '有任何学工相关问题，都可以问我',
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.outline,
            ),
          ),
          const SizedBox(height: 24),
          // 快捷问题示例
          Wrap(
            spacing: 8,
            runSpacing: 8,
            alignment: WrapAlignment.center,
            children: [
              '国家奖学金怎么申请？',
              '请假需要什么流程？',
              '助学贷款的条件是什么？',
            ].map((q) => ActionChip(
              label: Text(q, style: const TextStyle(fontSize: 13)),
              onPressed: () {
                _inputCtrl.text = q;
                _send();
              },
            )).toList(),
          ),
        ],
      ),
    );
  }

  Widget _buildMessage(Message msg) {
    if (msg.isUser) {
      return _buildUserBubble(msg);
    }
    return _buildAssistantMessage(msg);
  }

  Widget _buildUserBubble(Message msg) {
    final theme = Theme.of(context);
    final isFailed = msg.isFailed;

    return Align(
      alignment: Alignment.centerRight,
      child: Row(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          // 重试按钮（仅失败消息显示）
          if (isFailed)
            Padding(
              padding: const EdgeInsets.only(right: 4),
              child: InkWell(
                borderRadius: BorderRadius.circular(16),
                onTap: () {
                  final index = context.read<ChatProvider>().messages.indexOf(msg);
                  if (index >= 0) {
                    context.read<ChatProvider>().resendMessage(index);
                  }
                },
                child: Container(
                  padding: const EdgeInsets.all(6),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.errorContainer,
                    shape: BoxShape.circle,
                  ),
                  child: Icon(
                    Icons.refresh,
                    size: 18,
                    color: theme.colorScheme.onErrorContainer,
                  ),
                ),
              ),
            ),
          // 消息气泡
          Flexible(
            child: Container(
              margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
              constraints: BoxConstraints(
                maxWidth: MediaQuery.of(context).size.width * 0.75,
              ),
              decoration: BoxDecoration(
                color: isFailed
                    ? theme.colorScheme.errorContainer
                    : theme.colorScheme.primary,
                borderRadius: const BorderRadius.only(
                  topLeft: Radius.circular(16),
                  topRight: Radius.circular(16),
                  bottomLeft: Radius.circular(16),
                  bottomRight: Radius.circular(4),
                ),
              ),
              child: Text(
                msg.content,
                style: TextStyle(
                  color: isFailed
                      ? theme.colorScheme.onErrorContainer
                      : theme.colorScheme.onPrimary,
                  fontSize: 15,
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildAssistantMessage(Message msg) {
    final chat = context.read<ChatProvider>();
    final msgIndex = chat.messages.indexOf(msg);
    final isPlayingThis = chat.isPlaying && chat.playingIndex == msgIndex;
    final theme = Theme.of(context);

    // TTS 播放按钮
    Widget playButton = GestureDetector(
      onTap: () => chat.playTTS(msgIndex),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 2),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              isPlayingThis ? Icons.stop_circle_outlined : Icons.volume_up,
              size: 16,
              color: theme.colorScheme.outline,
            ),
            const SizedBox(width: 4),
            Text(
              isPlayingThis ? '停止' : '朗读',
              style: TextStyle(
                fontSize: 12,
                color: theme.colorScheme.outline,
              ),
            ),
          ],
        ),
      ),
    );

    // 如果有 AnswerCard，用卡片渲染
    if (msg.answerCard != null) {
      return Align(
        alignment: Alignment.centerLeft,
        child: ConstrainedBox(
          constraints: BoxConstraints(
            maxWidth: MediaQuery.of(context).size.width * 0.85,
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              AnswerCardWidget(
                card: msg.answerCard!,
                onFollowUp: (q) => chat.askFollowUp(q),
              ),
              Padding(
                padding: const EdgeInsets.only(left: 12),
                child: playButton,
              ),
            ],
          ),
        ),
      );
    }

    // 纯文本回复
    return Align(
      alignment: Alignment.centerLeft,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
            constraints: BoxConstraints(
              maxWidth: MediaQuery.of(context).size.width * 0.75,
            ),
            decoration: BoxDecoration(
              color: theme.colorScheme.surfaceContainerHighest,
              borderRadius: const BorderRadius.only(
                topLeft: Radius.circular(16),
                topRight: Radius.circular(16),
                bottomLeft: Radius.circular(4),
                bottomRight: Radius.circular(16),
              ),
            ),
            child: Text(msg.content, style: const TextStyle(fontSize: 15)),
          ),
          Padding(
            padding: const EdgeInsets.only(left: 16),
            child: playButton,
          ),
        ],
      ),
    );
  }

  Widget _buildLoadingBubble(ThemeData theme) {
    return Align(
      alignment: Alignment.centerLeft,
      child: Container(
        margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
        decoration: BoxDecoration(
          color: theme.colorScheme.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(16),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            SizedBox(
              width: 16, height: 16,
              child: CircularProgressIndicator(
                strokeWidth: 2,
                color: theme.colorScheme.primary,
              ),
            ),
            const SizedBox(width: 10),
            Text('思考中...', style: TextStyle(color: theme.colorScheme.outline)),
          ],
        ),
      ),
    );
  }

  Widget _buildInputBar(ThemeData theme, bool sending) {
    final chat = context.read<ChatProvider>();
    final isRecording = chat.isRecording;

    return Container(
      padding: const EdgeInsets.fromLTRB(8, 8, 8, 12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border(top: BorderSide(color: theme.colorScheme.outlineVariant)),
      ),
      child: SafeArea(
        top: false,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            // 录音状态提示
            if (isRecording)
              Container(
                width: double.infinity,
                padding: const EdgeInsets.symmetric(vertical: 4, horizontal: 12),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    _PulseIcon(icon: Icons.mic, color: theme.colorScheme.error),
                    const SizedBox(width: 8),
                    Text(
                      '正在聆听...',
                      style: TextStyle(
                        color: theme.colorScheme.error,
                        fontSize: 13,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  ],
                ),
              ),
            // 输入栏
            Row(
              children: [
                // 麦克风按钮（按住录音）
                GestureDetector(
                  onLongPressStart: (_) => chat.startRecording(),
                  onLongPressEnd: (_) => _handleVoiceResult(),
                  onLongPressCancel: () => _handleVoiceResult(),
                  child: AnimatedContainer(
                    duration: const Duration(milliseconds: 200),
                    margin: const EdgeInsets.only(right: 4),
                    decoration: BoxDecoration(
                      color: isRecording
                          ? theme.colorScheme.errorContainer
                          : theme.colorScheme.surfaceContainerHighest,
                      shape: BoxShape.circle,
                    ),
                    padding: const EdgeInsets.all(10),
                    child: Icon(
                      Icons.mic,
                      size: 22,
                      color: isRecording
                          ? theme.colorScheme.onErrorContainer
                          : theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ),
                // 文本输入框
                Expanded(
                  child: TextField(
                    controller: _inputCtrl,
                    maxLines: 4,
                    minLines: 1,
                    decoration: InputDecoration(
                      hintText: isRecording ? '松开发送语音...' : '输入你的问题...',
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(24),
                        borderSide: BorderSide.none,
                      ),
                      filled: true,
                      fillColor: theme.colorScheme.surfaceContainerHighest,
                      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
                    ),
                    textInputAction: TextInputAction.send,
                    onSubmitted: (_) => _send(),
                  ),
                ),
                const SizedBox(width: 8),
                // 发送按钮（录音中隐藏）
                if (!isRecording)
                  IconButton.filled(
                    onPressed: sending ? null : _send,
                    icon: const Icon(Icons.send),
                  ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

/// 脉冲动画图标 — 用于录音状态指示（红色呼吸灯效果）
class _PulseIcon extends StatefulWidget {
  final IconData icon;
  final Color color;
  const _PulseIcon({required this.icon, required this.color});

  @override
  State<_PulseIcon> createState() => _PulseIconState();
}

class _PulseIconState extends State<_PulseIcon> with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;
  late final Animation<double> _anim;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 800),
    );
    _anim = Tween<double>(begin: 0.6, end: 1.0).animate(
      CurvedAnimation(parent: _ctrl, curve: Curves.easeInOut),
    );
    _ctrl.repeat(reverse: true);
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return FadeTransition(
      opacity: _anim,
      child: Icon(widget.icon, color: widget.color, size: 20),
    );
  }
}
