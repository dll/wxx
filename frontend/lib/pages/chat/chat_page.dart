import 'dart:convert' show HtmlEscape, base64Encode;
import '../../utils/web_export.dart';
import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import '../../models/models.dart';
import '../../providers/chat_provider.dart';
import '../../providers/bookmark_provider.dart';
import '../../providers/feedback_provider.dart';
import '../../providers/knowledge_provider.dart';
import '../../providers/session_provider.dart';
import '../../services/voice/voice_navigator.dart';
import '../../services/voice/web_speech_recognizer.dart';
import '../../utils/feedback_report.dart';
import '../../utils/screenshot_capture.dart';
import '../../widgets/answer_card.dart';
import '../../widgets/export_dialog.dart';
import '../../widgets/feedback_screenshot.dart';

const _htmlEscaper = HtmlEscape();

/// 对话主页
class ChatPage extends StatefulWidget {
  /// 从知识大厅跳转时携带的初始问题
  final String? initialQuestion;

  /// 是否进入页面后自动开启语音输入（悬浮菜单"语音导航"入口用）
  final bool autoVoice;

  const ChatPage({super.key, this.initialQuestion, this.autoVoice = false});

  @override
  State<ChatPage> createState() => _ChatPageState();
}

class _ChatPageState extends State<ChatPage> {
  final _inputCtrl = TextEditingController();
  final _scrollCtrl = ScrollController();
  bool _initialQuestionHandled = false;

  /// 页面重建 key — 删除对话后递增以强制重建 Scaffold，
  /// 彻底清除 ListView/动画等子组件状态，避免渲染卡死。
  int _rebuildKey = 0;

  // ── 浏览器实时语音识别（替代之前的 MediaRecorder + 后端 ASR）──
  WebSpeechRecognizer? _speech;
  bool _isListening = false;
  String _interimText = '';

  @override
  void initState() {
    super.initState();
    // 延迟处理初始问题（等 Widget 构建完成后发送）
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _handleInitialQuestion();
      // 加载可用的智能体列表
      context.read<ChatProvider>().loadAgents();
      // 悬浮菜单"语音导航"入口：进入页面后自动开始语音识别/录音
      if (widget.autoVoice) {
        Future.delayed(const Duration(milliseconds: 400), () {
          if (mounted) _toggleVoiceInput();
        });
      }
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
    _speech?.dispose();
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

  /// 切换语音录入：开始/停止
  /// 使用浏览器 SpeechRecognition API 实时识别，文字直接写入输入框
  Future<void> _toggleVoiceInput() async {
    final chat = context.read<ChatProvider>();

    if (_isListening) {
      // 移动端：停止后通过后端 ASR 识别
      if (!kIsWeb && chat.isRecording) {
        await _handleVoiceResult();
        return;
      }
      _stopVoiceInput(autoSend: false);
      return;
    }

    if (!kIsWeb) {
      // 移动端走旧的 MediaRecorder + 后端 ASR 流程
      _startMobileRecording();
      return;
    }

    _speech ??= WebSpeechRecognizer()
      ..onTranscript = _onSpeechTranscript
      ..onError = _onSpeechError
      ..onEnd = _onSpeechEnd;

    setState(() {
      _isListening = true;
      _interimText = '';
    });
    _speech!.start(continuous: true);
  }

  void _onSpeechTranscript(String interim, String finalText, bool isFinal) {
    if (!mounted) return;
    setState(() {
      if (finalText.isNotEmpty) {
        // 累加最终文本到输入框：每次直接读当前输入框内容，避免 stale state
        final current = _inputCtrl.text;
        final separator = current.isEmpty || current.endsWith(' ') ? '' : ' ';
        _inputCtrl.text = '$current$separator$finalText'.trim();
        _inputCtrl.selection =
            TextSelection.collapsed(offset: _inputCtrl.text.length);
        _interimText = '';
      } else {
        _interimText = interim;
      }
    });
  }

  void _onSpeechError(String error) {
    if (!mounted) return;
    setState(() => _isListening = false);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
          content: Text('语音识别错误：$error'), duration: const Duration(seconds: 2)),
    );
  }

  void _onSpeechEnd() {
    if (!mounted) return;
    if (_isListening) {
      // 浏览器自动结束，但用户还在录音状态 → 重启
      _speech?.start(continuous: true);
    }
  }

  /// 停止语音输入
  /// [autoSend] 是否自动发送当前内容
  void _stopVoiceInput({bool autoSend = false}) {
    _speech?.stop();
    if (!mounted) return;
    setState(() {
      _isListening = false;
      _interimText = '';
    });
    if (autoSend && _inputCtrl.text.trim().isNotEmpty) {
      _send();
    }
  }

  /// 移动端录音：使用旧的 MediaRecorder + 后端 ASR
  Future<void> _startMobileRecording() async {
    final chat = context.read<ChatProvider>();
    await chat.startRecording();
    if (!mounted) return;
    setState(() => _isListening = chat.isRecording);
    if (!chat.isRecording) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('无法开始录音，请检查麦克风权限')),
      );
    }
  }

  /// 移动端语音录制结果：先检查导航指令，否则作为聊天消息发送
  Future<void> _handleVoiceResult() async {
    final chat = context.read<ChatProvider>();
    if (!chat.isRecording) return;

    chat.stopRecording();
    setState(() => _isListening = false);

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
        curve: Curves.easeOutCubic,
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final chat = context.watch<ChatProvider>();
    final theme = Theme.of(context);

    return Scaffold(
      key: ValueKey('chat-scaffold-$_rebuildKey'),
      appBar: AppBar(
        title: _buildAppBarTitle(chat, theme),
        actions: [
          if (chat.sessionId != null)
            IconButton(
              icon: const Icon(Icons.delete_outline),
              tooltip: '删除对话',
              onPressed: () => _confirmDeleteSession(chat),
            ),
          IconButton(
            icon: const Icon(Icons.add_comment_outlined),
            tooltip: '新对话',
            onPressed: () => chat.newChat(),
          ),
        ],
      ),
      body: Column(
        children: [
          // 智能体选择器
          _buildAgentSelector(chat, theme),

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
                      return _SlideInItem(
                        key: ValueKey(index),
                        child: _buildMessage(chat.messages[index], index),
                      );
                    },
                  ),
          ),

          // 错误提示（带动画过渡）
          AnimatedSwitcher(
            duration: const Duration(milliseconds: 250),
            child: chat.error != null
                ? Container(
                    key: const ValueKey('error-banner'),
                    width: double.infinity,
                    color: theme.colorScheme.errorContainer,
                    padding:
                        const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                    child: Text(
                      chat.error!,
                      style: TextStyle(
                          color: theme.colorScheme.onErrorContainer,
                          fontSize: 13),
                    ),
                  )
                : const SizedBox.shrink(key: ValueKey('no-error')),
          ),

          // 输入区域
          _buildInputBar(theme, chat.sending),
        ],
      ),
    );
  }

  /// 顶部标题：蔚小芯 + 当前智能体徽标
  Widget _buildAppBarTitle(ChatProvider chat, ThemeData theme) {
    final agent = chat.selectedAgent;
    if (agent == null) {
      return Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.auto_awesome, size: 20, color: theme.colorScheme.primary),
          const SizedBox(width: 8),
          const Text('蔚小芯'),
        ],
      );
    }
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(Icons.auto_awesome, size: 20, color: theme.colorScheme.primary),
        const SizedBox(width: 8),
        const Text('蔚小芯 · '),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
          decoration: BoxDecoration(
            color: _agentColor(agent.agentType).withOpacity(0.14),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Text(
            agent.name,
            style: theme.textTheme.titleMedium?.copyWith(
              fontSize: 14,
              fontWeight: FontWeight.w700,
              color: _agentColor(agent.agentType),
            ),
          ),
        ),
      ],
    );
  }

  /// 智能体选择器 — 带标题的卡片式横向选择条，显著可见
  Widget _buildAgentSelector(ChatProvider chat, ThemeData theme) {
    if (chat.agents.isEmpty) return const SizedBox.shrink();

    final agents = chat.agents;
    return Container(
      color: theme.colorScheme.surface,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 10, 16, 4),
            child: Row(
              children: [
                Icon(Icons.smart_toy_outlined,
                    size: 16, color: theme.colorScheme.primary),
                const SizedBox(width: 6),
                Text(
                  '智能体',
                  style: theme.textTheme.labelMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    chat.selectedAgent == null
                        ? '点击选择，或直接提问自动匹配'
                        : '当前：${chat.selectedAgent!.name}',
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: chat.selectedAgent == null
                          ? theme.colorScheme.outline
                          : _agentColor(chat.selectedAgent!.agentType),
                      fontWeight: FontWeight.w600,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
            ),
          ),
          SizedBox(
            height: 56,
            child: ListView.separated(
              scrollDirection: Axis.horizontal,
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 10),
              itemCount: agents.length + 1,
              separatorBuilder: (_, __) => const SizedBox(width: 8),
              itemBuilder: (context, index) {
                if (index == 0) {
                  final selected = chat.selectedAgentId == null;
                  return _buildAgentOption(
                    theme,
                    icon: Icons.auto_awesome,
                    label: '默认',
                    color: theme.colorScheme.primary,
                    selected: selected,
                    onTap: () => chat.selectAgent(null),
                  );
                }
                final agent = agents[index - 1];
                final selected = chat.selectedAgentId == agent.agentId;
                return _buildAgentOption(
                  theme,
                  icon: _agentIcon(agent.agentType),
                  label: agent.name,
                  color: _agentColor(agent.agentType),
                  selected: selected,
                  onTap: () =>
                      chat.selectAgent(selected ? null : agent.agentId),
                );
              },
            ),
          ),
        ],
      ),
    );
  }

  /// 单个智能体选项（图标 + 名称 + 选中态描边）
  Widget _buildAgentOption(
    ThemeData theme, {
    required IconData icon,
    required String label,
    required Color color,
    required bool selected,
    required VoidCallback onTap,
  }) {
    return Material(
      color: selected ? color.withOpacity(0.14) : theme.colorScheme.surface,
      borderRadius: BorderRadius.circular(12),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 180),
          padding: const EdgeInsets.symmetric(horizontal: 12),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
              color: selected ? color : theme.colorScheme.outlineVariant,
              width: selected ? 1.6 : 1,
            ),
          ),
          child: Row(
            children: [
              Icon(icon, size: 18, color: selected ? color : theme.colorScheme.onSurfaceVariant),
              const SizedBox(width: 6),
              Text(
                label,
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
                  color: selected ? color : theme.colorScheme.onSurface,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  /// 智能体类型图标（对齐 5 个专用智能体）
  IconData _agentIcon(String type) {
    switch (type) {
      case 'policy':
        return Icons.gavel;
      case 'process':
        return Icons.assignment_outlined;
      case 'major':
        return Icons.menu_book_outlined;
      case 'emotion':
        return Icons.favorite_border;
      case 'qa':
        return Icons.chat;
      default:
        return Icons.smart_toy;
    }
  }

  /// 智能体类型主题色（与各智能体定位一致）
  Color _agentColor(String type) {
    switch (type) {
      case 'policy':
        return const Color(0xFFE65100);
      case 'process':
        return const Color(0xFF6A1B9A);
      case 'major':
        return const Color(0xFF00838F);
      case 'emotion':
        return const Color(0xFFC62828);
      case 'qa':
        return const Color(0xFF1565C0);
      default:
        return const Color(0xFF7B1FA2);
    }
  }

  Widget _buildEmptyState(ThemeData theme) {
    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(vertical: 24, horizontal: 16),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 560),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(Icons.school,
                  size: 64, color: theme.colorScheme.primary.withOpacity(0.3)),
              const SizedBox(height: 16),
              Text(
                '你好！我是蔚小芯',
                style: theme.textTheme.titleMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                '内置 5 个专用智能体，有任何学工问题都可以问我',
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.outline,
                ),
              ),
              const SizedBox(height: 24),
              // 5 个智能体：各配典型提问示例
              for (final group in _agentExampleGroups) ...[
                _buildAgentExampleGroup(theme, group),
                const SizedBox(height: 12),
              ],
            ],
          ),
        ),
      ),
    );
  }

  /// 典型提问示例分组（对齐 5 个专用智能体）
  static const _agentExampleGroups = <_AgentExampleGroup>[
    _AgentExampleGroup(
      icon: Icons.gavel,
      color: Color(0xFFE65100),
      name: '政策解读',
      agentType: 'policy',
      questions: [
        '国家奖学金的申请条件是什么？',
        '转专业需要满足哪些要求？',
        '助学贷款额度多少？',
      ],
    ),
    _AgentExampleGroup(
      icon: Icons.assignment_outlined,
      color: Color(0xFF6A1B9A),
      name: '流程指引',
      agentType: 'process',
      questions: [
        '请假怎么办理？需要什么材料？',
        '入党流程是什么？',
        '毕业离校要办哪些手续？',
      ],
    ),
    _AgentExampleGroup(
      icon: Icons.favorite_border,
      color: Color(0xFFC62828),
      name: '心理疏导',
      agentType: 'emotion',
      questions: [
        '最近压力大睡不着怎么办？',
        '考试焦虑怎么缓解？',
        '和室友关系不好，很郁闷',
      ],
    ),
    _AgentExampleGroup(
      icon: Icons.menu_book_outlined,
      color: Color(0xFF6A1B9A),
      name: '学科专业',
      agentType: 'major',
      questions: [
        '网络空间安全专业学什么？',
        '人工智能方向有哪些课程？',
        '计算机专业就业前景如何？',
      ],
    ),
    _AgentExampleGroup(
      icon: Icons.chat,
      color: Color(0xFF1565C0),
      name: '通用问答',
      agentType: 'qa',
      questions: [
        '图书馆几点开门？',
        '校医院电话是多少？',
        '怎么加入社团？',
      ],
    ),
  ];

  Widget _buildAgentExampleGroup(ThemeData theme, _AgentExampleGroup group) {
    final chat = context.read<ChatProvider>();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Icon(group.icon, size: 16, color: group.color),
            const SizedBox(width: 6),
            Text(
              group.name,
              style: theme.textTheme.labelLarge?.copyWith(
                color: group.color,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(width: 8),
            // 点击即切换到对应智能体
            InkWell(
              borderRadius: BorderRadius.circular(8),
              onTap: () => chat.selectAgent(_agentIdForType(group.agentType)),
              child: Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                decoration: BoxDecoration(
                  color: group.color.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(6),
                ),
                child: Text(
                  '用「${group.name}」回答 →',
                  style: TextStyle(
                    fontSize: 11,
                    color: group.color,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 6),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: group.questions
              .map((q) => ActionChip(
                    label: Text(q, style: const TextStyle(fontSize: 12)),
                    onPressed: () {
                      // 提问同时选中对应智能体
                      chat.selectAgent(_agentIdForType(group.agentType));
                      _inputCtrl.text = q;
                      _send();
                    },
                  ))
              .toList(),
        ),
      ],
    );
  }

  /// 智能体类型 → agent_id 映射（与后端 agents 表一致）
  String? _agentIdForType(String type) {
    switch (type) {
      case 'policy':
        return 'policy-expert';
      case 'process':
        return 'process-guide';
      case 'major':
        return 'major-guide';
      case 'emotion':
        return 'emotion-counselor';
      case 'qa':
        return 'qa-default';
      default:
        return null;
    }
  }

  Widget _buildMessage(Message msg, int index) {
    if (msg.isUser) {
      return _buildUserBubble(msg);
    }
    return _buildAssistantMessage(msg, index);
  }

  /// 助手消息头部：圆形头像 + 当前智能体名称
  Widget _buildAssistantHeader(ThemeData theme, ChatProvider chat) {
    final agent = chat.selectedAgent;
    final color = agent != null
        ? _agentColor(agent.agentType)
        : theme.colorScheme.primary;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 28,
            height: 28,
            decoration: BoxDecoration(
              color: color.withOpacity(0.14),
              shape: BoxShape.circle,
            ),
            child: Icon(Icons.auto_awesome, size: 15, color: color),
          ),
          const SizedBox(width: 8),
          Text(
            agent?.name ?? '蔚小芯',
            style: theme.textTheme.labelMedium?.copyWith(
              color: color,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
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
                  final index =
                      context.read<ChatProvider>().messages.indexOf(msg);
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
          // 消息气泡（主题色渐变，避免单一实色块）
          Flexible(
            child: Container(
              margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
              constraints: BoxConstraints(
                maxWidth: MediaQuery.of(context).size.width * 0.75,
              ),
              decoration: BoxDecoration(
                gradient: isFailed
                    ? null
                    : LinearGradient(
                        colors: [
                          theme.colorScheme.primary,
                          Color.alphaBlend(
                              theme.colorScheme.primary,
                              theme.colorScheme.tertiary.withOpacity(0.25)),
                        ],
                        begin: Alignment.topLeft,
                        end: Alignment.bottomRight,
                      ),
                color: isFailed
                    ? theme.colorScheme.errorContainer
                    : theme.colorScheme.primary,
                borderRadius: const BorderRadius.only(
                  topLeft: Radius.circular(16),
                  topRight: Radius.circular(16),
                  bottomLeft: Radius.circular(16),
                  bottomRight: Radius.circular(4),
                ),
                boxShadow: isFailed
                    ? null
                    : [
                        BoxShadow(
                          color: theme.colorScheme.primary.withOpacity(0.15),
                          blurRadius: 8,
                          offset: const Offset(0, 2),
                        ),
                      ],
              ),
              child: Text(
                msg.content,
                style: TextStyle(
                  color: isFailed
                      ? theme.colorScheme.onErrorContainer
                      : theme.colorScheme.onPrimary,
                  fontSize: 15,
                  height: 1.45,
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildAssistantMessage(Message msg, int msgIndex) {
    final chat = context.read<ChatProvider>();
    final isPlayingThis = chat.isPlaying && chat.playingIndex == msgIndex;
    final theme = Theme.of(context);

    // 找到此回答对应的用户提问
    final question = _findQuestionFor(msgIndex);

    // 高频操作直接展示，低频操作收纳到更多菜单，避免移动端拥挤。
    Widget actionBar = Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Consumer<BookmarkProvider>(
        builder: (_, bm, __) {
          final isMarked = bm.isBookmarked(msg.content);
          return Wrap(
            spacing: 2,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              IconButton(
                onPressed: () => _copyAnswer(msg),
                icon: const Icon(Icons.content_copy_outlined),
                tooltip: '复制回答',
                visualDensity: VisualDensity.compact,
              ),
              IconButton(
                onPressed: () => bm.toggle(
                  question: question,
                  conclusion: msg.content,
                  sources:
                      msg.answerCard?.sources.map((s) => s.title).toList() ??
                          [],
                  followUps: msg.answerCard?.followUps ?? [],
                ),
                icon: Icon(isMarked ? Icons.star : Icons.star_outline),
                tooltip: isMarked ? '取消收藏' : '收藏回答',
                visualDensity: VisualDensity.compact,
              ),
              IconButton(
                onPressed: () => _showFeedbackDialog(msg),
                icon: const Icon(Icons.feedback_outlined),
                tooltip: '反馈纠错',
                visualDensity: VisualDensity.compact,
              ),
              PopupMenuButton<String>(
                tooltip: '更多操作',
                icon: const Icon(Icons.more_horiz),
                onSelected: (value) {
                  switch (value) {
                    case 'speak':
                      chat.playTTS(msgIndex);
                      break;
                    case 'export':
                      _showExportDialog(question, msg);
                      break;
                    case 'save':
                      _saveToKnowledgeBase(question, msg);
                      break;
                  }
                },
                itemBuilder: (_) => [
                  PopupMenuItem(
                    value: 'speak',
                    child: ListTile(
                      contentPadding: EdgeInsets.zero,
                      leading: Icon(isPlayingThis
                          ? Icons.stop_circle_outlined
                          : Icons.volume_up_outlined),
                      title: Text(isPlayingThis ? '停止朗读' : '朗读回答'),
                    ),
                  ),
                  const PopupMenuItem(
                    value: 'export',
                    child: ListTile(
                      contentPadding: EdgeInsets.zero,
                      leading: Icon(Icons.download_outlined),
                      title: Text('导出回答'),
                    ),
                  ),
                  const PopupMenuItem(
                    value: 'save',
                    child: ListTile(
                      contentPadding: EdgeInsets.zero,
                      leading: Icon(Icons.save_outlined),
                      title: Text('保存到知识库'),
                    ),
                  ),
                ],
              ),
            ],
          );
        },
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
              _buildAssistantHeader(theme, chat),
              const SizedBox(height: 6),
              AnswerCardWidget(
                card: msg.answerCard!,
                onFollowUp: (q) => chat.askFollowUp(q),
              ),
              Padding(
                padding: const EdgeInsets.only(left: 12),
                child: actionBar,
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
          _buildAssistantHeader(theme, chat),
          const SizedBox(height: 6),
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
            child: MarkdownBody(
              data: msg.content,
              selectable: true,
              styleSheet: MarkdownStyleSheet(
                p: const TextStyle(fontSize: 15, height: 1.6),
                strong: const TextStyle(
                    fontSize: 15, fontWeight: FontWeight.bold, height: 1.6),
                h1: theme.textTheme.titleLarge,
                h2: theme.textTheme.titleMedium,
                h3: theme.textTheme.titleSmall,
                listBullet: const TextStyle(fontSize: 15, height: 1.6),
                code: TextStyle(
                  fontFamily: 'monospace',
                  fontSize: 13,
                  backgroundColor: theme.colorScheme.surfaceContainerHighest,
                  color: theme.colorScheme.onSurface,
                ),
                blockquoteDecoration: BoxDecoration(
                  color: theme.colorScheme.primaryContainer.withOpacity(0.4),
                  border: Border(
                    left:
                        BorderSide(color: theme.colorScheme.primary, width: 3),
                  ),
                ),
                tableBorder: TableBorder.all(
                  color: theme.colorScheme.outlineVariant,
                ),
                a: TextStyle(
                  color: theme.colorScheme.primary,
                  decoration: TextDecoration.none,
                ),
              ),
            ),
          ),
          Padding(
            padding: const EdgeInsets.only(left: 16),
            child: actionBar,
          ),
          // 无知识库引用时使用显著状态块说明可信边界。
          if (msg.answerCard == null || msg.answerCard!.sources.isEmpty)
            Container(
              margin: const EdgeInsets.fromLTRB(16, 4, 16, 8),
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: theme.colorScheme.tertiaryContainer.withOpacity(0.45),
                borderRadius: BorderRadius.circular(8),
                border: Border.all(
                  color: theme.colorScheme.tertiary.withOpacity(0.35),
                ),
              ),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Icon(Icons.info_outline,
                      size: 20, color: theme.colorScheme.tertiary),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      '此回答未引用已审核资料，不应作为政策或流程依据。',
                      style: theme.textTheme.bodySmall,
                    ),
                  ),
                  TextButton(
                    onPressed: () => context.go('/browse'),
                    child: const Text('查资料'),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }

  /// 找到指定位置助手消息对应的用户提问
  String _findQuestionFor(int assistantIndex) {
    final messages = context.read<ChatProvider>().messages;
    for (int i = assistantIndex - 1; i >= 0; i--) {
      if (messages[i].isUser) return messages[i].content;
    }
    return '';
  }

  /// 复制回答内容到剪贴板
  void _copyAnswer(Message msg) {
    final text = msg.content;
    if (text.isEmpty) return;
    Clipboard.setData(ClipboardData(text: text));
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('已复制到剪贴板'),
        duration: Duration(seconds: 1),
      ),
    );
  }

  /// 导出为 PDF（打开浏览器打印对话框）
  void _exportPdf(String question, Message msg) {
    final body = _buildExportHtml(question, msg, forPrint: true);
    openHtmlInNewTab(body);
  }

  /// 导出为 PNG 长图
  void _exportPng(String question, Message msg) {
    final body = _buildExportHtml(question, msg, forPng: true);
    openHtmlInNewTab(body);
  }

  String _buildExportHtml(String question, Message msg,
      {bool forPrint = false, bool forPng = false}) {
    final esc = _htmlEscaper.convert;
    final sources = msg.answerCard?.sources ?? [];
    final pngExtra = forPng ? ' width: 600px; margin: 0 auto;' : '';
    final answerStyle = forPng ? ' white-space: pre-wrap;' : '';

    String sourceHtml = '';
    if (sources.isNotEmpty) {
      if (forPng) {
        sourceHtml =
            '<div class="source"><strong>来源：</strong>${esc(sources.map((s) => s.title).join(', '))}</div>';
      } else {
        sourceHtml =
            '<div class="source"><strong>信息来源：</strong><ul>${sources.map((s) => '<li>${esc(s.title)}</li>').join()}</ul></div>';
      }
    }

    final printScript = forPrint
        ? '<script>window.onload = () => window.print();</script>'
        : '';
    final tipHtml = forPng ? '<div class="tip">长按保存图片 · 蔚小芯 AI 学工助手</div>' : '';

    return '''
<!DOCTYPE html><html><head><meta charset="utf-8">
<style>
  body { font-family: "Microsoft YaHei", sans-serif; padding: 40px; color: #333;$pngExtra }
  h1 { color: #1565C0; font-size: 20px; border-bottom: 2px solid #1565C0; padding-bottom: 8px; }
  .question { background: #f5f5f5; padding: 16px; border-radius: 8px; margin: 16px 0; }
  .answer { padding: 16px; line-height: 1.8;$answerStyle }
  .source { background: #e3f2fd; padding: 12px; border-radius: 6px; margin: 8px 0; font-size: 14px; }
  .meta { color: #999; font-size: 12px; margin-top: 24px; border-top: 1px solid #eee; padding-top: 8px; }
  .tip { text-align: center; color: #999; font-size: 11px; margin-top: 16px; }
</style></head><body>
<h1>蔚小芯 · 问答记录</h1>
<div class="question"><strong>问题：</strong>${esc(question)}</div>
<div class="answer"><strong>回答：</strong>${esc(msg.content)}</div>
$sourceHtml
<div class="meta">导出时间：${DateTime.now().toString().substring(0, 19)} · 蔚小芯 AI 学工助手</div>
$tipHtml
$printScript
</body></html>''';
  }

  /// 多格式导出对话框
  void _showExportDialog(String question, Message msg) async {
    final format = await ExportDialog.show(context, contentId: msg.id);
    if (format == null || !mounted) return;

    switch (format) {
      case 'pdf':
        _exportPdf(question, msg);
        break;
      case 'png':
        _exportPng(question, msg);
        break;
      case 'md':
        _exportMarkdown(question, msg);
        break;
    }
  }

  /// 导出为 Markdown
  void _exportMarkdown(String question, Message msg) {
    final sources =
        msg.answerCard?.sources.map((s) => '- ${s.title}').join('\n') ?? '';
    final md = StringBuffer();
    md.writeln('# 蔚小芯 · 问答记录\n');
    md.writeln('## 问题\n');
    md.writeln('$question\n');
    md.writeln('## 回答\n');
    md.writeln('${msg.content}\n');
    if (sources.isNotEmpty) {
      md.writeln('## 参考来源\n');
      md.writeln('$sources\n');
    }
    md.writeln('---\n');
    md.writeln(
        '> 导出时间：${DateTime.now().toString().substring(0, 19)} · 蔚小芯 AI 学工助手');

    Clipboard.setData(ClipboardData(text: md.toString()));
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
            content: Text('Markdown 已复制到剪贴板'), duration: Duration(seconds: 2)),
      );
    }
  }

  /// 确认删除当前对话
  ///
  /// 流程：弹确认框 → 调 deleteSession 删除后端记录。
  ///
  /// 删除成功后统一乐观调 newChat 清空当前对话 + 重建页面（_rebuildKey++），
  /// 彻底清除 ListView/动画等子组件残留状态。
  /// 不再使用 Web 整页刷新（reloadPage）：Cloudflare Pages 缓存下 CanvasKit
  /// 资源曾出现 0 字节导致 Flutter 根视图未挂载，刷新后反而空白页
  /// （见 docs/蔚小芯智能体UI/UX设计与实现.md 的 P0 空白风险记录）。
  Future<void> _confirmDeleteSession(ChatProvider chat) async {
    // 在弹框前快照 sessionId，避免弹框期间 provider 状态被其他逻辑修改
    final sessionId = chat.sessionId;
    if (sessionId == null) return;

    final confirm = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('删除对话'),
        content: const Text('确定删除当前对话吗？此操作不可恢复。'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('取消')),
          TextButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('删除')),
        ],
      ),
    );
    if (confirm != true) return;
    if (!mounted) return;

    // 先乐观清空界面，避免等待期间仍显示旧会话；
    // 同时重置 _sending（若 AI 正在回复，删除后 _sending 卡 true 会导致渲染卡死）
    chat.newChat();

    final bool ok;
    try {
      ok = await context.read<SessionProvider>().deleteSession(sessionId);
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('删除失败：$e')),
      );
      return;
    }
    if (!mounted) return;

    // 重建页面清除 ListView/动画等子组件残留状态
    setState(() => _rebuildKey++);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(ok ? '对话已删除' : '删除失败，请稍后重试')),
    );
  }

  /// 保存问答对到知识库
  Future<void> _saveToKnowledgeBase(String question, Message msg) async {
    if (!mounted) return;
    final kb = context.read<KnowledgeProvider>();
    final title =
        question.length > 50 ? '${question.substring(0, 50)}...' : question;
    final content = '问：$question\n\n答：${msg.content}';

    final ok = await kb.createResource({
      'title': title,
      'summary': question,
      'content': content,
      'resource_type': 'FAQ',
      'owner_scope': 'school',
      'owner_id': '',
      'role_scope': '["student","counselor","student_union","college_admin"]',
      'tags': '["用户收藏","问答"]',
    });

    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(ok ? '已保存到知识库' : '保存失败，请重试')),
      );
    }
  }

  /// 纠错反馈对话框 — 自动截屏 + 上传 + 提交
  void _showFeedbackDialog(Message msg) async {
    // 先自动截屏（dialog 未弹出，截取用户真实看到的画面）
    final shot = await captureScreenshot();

    if (!mounted) return;

    final contentCtrl = TextEditingController();
    String category = 'answer_error';
    String module = '';
    Uint8List? screenshotBytes = shot.bytes;
    bool screenshotValid = false;
    if (screenshotBytes != null && screenshotBytes.isNotEmpty) {
      screenshotValid = await isDecodableImage(screenshotBytes);
      if (!screenshotValid) screenshotBytes = null;
    }

    if (!mounted) return;

    showDialog(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setState) => AlertDialog(
          title: const Text('反馈纠错'),
          content: SizedBox(
            width: 420,
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // 截屏预览（仅真实有效的截图才显示"已截屏"标记）
                  if (screenshotBytes != null && screenshotValid)
                    Container(
                      decoration: BoxDecoration(
                        borderRadius: BorderRadius.circular(12),
                        border: Border.all(
                            color: Theme.of(context)
                                .colorScheme
                                .outlineVariant
                                .withOpacity(0.4)),
                      ),
                      child: AspectRatio(
                        aspectRatio: 16 / 9,
                        child: ClipRRect(
                          borderRadius: BorderRadius.circular(11),
                          child: Stack(
                            fit: StackFit.expand,
                            children: [
                              Image.memory(
                                screenshotBytes,
                                fit: BoxFit.cover,
                                errorBuilder: (_, __, ___) => Container(
                                  color: Theme.of(context)
                                      .colorScheme
                                      .surfaceContainerHighest
                                      .withOpacity(0.3),
                                  alignment: Alignment.center,
                                  child: Text(
                                    '截图加载失败',
                                    style: Theme.of(context)
                                        .textTheme
                                        .labelSmall
                                        ?.copyWith(
                                            color: Theme.of(context)
                                                .colorScheme
                                                .error),
                                  ),
                                ),
                              ),
                              Positioned(
                                top: 8,
                                right: 8,
                                child: Container(
                                  padding: const EdgeInsets.symmetric(
                                      horizontal: 8, vertical: 3),
                                  decoration: BoxDecoration(
                                    color: Colors.green.withOpacity(0.85),
                                    borderRadius: BorderRadius.circular(12),
                                  ),
                                  child: const Row(
                                    mainAxisSize: MainAxisSize.min,
                                    children: [
                                      Icon(Icons.check,
                                          color: Colors.white, size: 14),
                                      SizedBox(width: 4),
                                      Text('已截屏',
                                          style: TextStyle(
                                              color: Colors.white,
                                              fontSize: 11)),
                                    ],
                                  ),
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    )
                  else
                    Container(
                      padding: const EdgeInsets.all(12),
                      decoration: BoxDecoration(
                        borderRadius: BorderRadius.circular(12),
                        color: Theme.of(context)
                            .colorScheme
                            .surfaceContainerHighest
                            .withOpacity(0.3),
                      ),
                      child: Row(
                        children: [
                          Icon(Icons.screenshot_outlined,
                              size: 20,
                              color: Theme.of(context).colorScheme.error),
                          const SizedBox(width: 8),
                          Text(shot.error ?? '未截屏（截图可选）',
                              style: Theme.of(context)
                                  .textTheme
                                  .labelSmall
                                  ?.copyWith(
                                      color: Theme.of(context)
                                          .colorScheme
                                          .error)),
                        ],
                      ),
                    ),
                  if (screenshotBytes != null && screenshotValid)
                    const SizedBox(height: 16),

                  const Text('反馈类型',
                      style: TextStyle(fontWeight: FontWeight.w500)),
                  const SizedBox(height: 8),
                  SegmentedButton<String>(
                    segments: const [
                      ButtonSegment(value: 'answer_error', label: Text('回答有误')),
                      ButtonSegment(value: 'suggestion', label: Text('建议改进')),
                      ButtonSegment(value: 'other', label: Text('其他')),
                    ],
                    selected: {category},
                    onSelectionChanged: (v) =>
                        setState(() => category = v.first),
                  ),
                  const SizedBox(height: 16),
                  const Text('所属模块（便于快速修复）',
                      style: TextStyle(fontWeight: FontWeight.w500)),
                  const SizedBox(height: 8),
                  DropdownButtonFormField<String>(
                    value: module.isEmpty ? null : module,
                    isExpanded: true,
                    decoration: const InputDecoration(
                      hintText: '选择问题所属模块（可选）',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                    items: [
                      const DropdownMenuItem<String>(
                        value: '',
                        child: Text('暂不选择'),
                      ),
                      for (final m in feedbackModules)
                        DropdownMenuItem<String>(value: m, child: Text(m)),
                    ],
                    onChanged: (v) => setState(() => module = v ?? ''),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: contentCtrl,
                    maxLines: 4,
                    decoration: const InputDecoration(
                      hintText: '请描述问题或建议...',
                      border: OutlineInputBorder(),
                    ),
                  ),
                  const SizedBox(height: 12),
                  // ── 复制反馈数据（手工提交 AI 工具修复）──
                  Row(
                    children: [
                      Expanded(
                        child: OutlinedButton.icon(
                          onPressed: () => _copyDraftJson(
                              contentCtrl, category, module, screenshotBytes,
                              screenshotValid, msg),
                          icon: const Icon(Icons.data_object, size: 16),
                          label: const Text('复制 JSON'),
                          style: OutlinedButton.styleFrom(
                            visualDensity: VisualDensity.compact,
                          ),
                        ),
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        child: OutlinedButton.icon(
                          onPressed: () => _copyDraftMarkdown(
                              contentCtrl, category, module, screenshotBytes,
                              screenshotValid, msg),
                          icon: const Icon(Icons.copy_all_outlined, size: 16),
                          label: const Text('复制报告(提交AI修复)'),
                          style: OutlinedButton.styleFrom(
                            visualDensity: VisualDensity.compact,
                          ),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '截图佐证可选；复制 JSON/报告可将本反馈全部数据手工提交给 AI 工具修复',
                    style: Theme.of(context).textTheme.labelSmall?.copyWith(
                        color:
                            Theme.of(context).colorScheme.onSurfaceVariant),
                    textAlign: TextAlign.center,
                  ),
                ],
              ),
            ),
          ),
          actions: [
            TextButton(
              onPressed: () {
                contentCtrl.dispose();
                Navigator.of(ctx).pop();
              },
              child: const Text('取消'),
            ),
            FilledButton(
              onPressed: () async {
                if (contentCtrl.text.trim().isEmpty) return;
                final text = contentCtrl.text.trim();
                contentCtrl.dispose();
                Navigator.of(ctx).pop();

                // 上传截图（仅真实有效截图）
                String screenshotUrl = '';
                bool uploadFailed = false;
                if (screenshotBytes != null &&
                    screenshotBytes.isNotEmpty &&
                    screenshotValid) {
                  final url = await context
                      .read<FeedbackProvider>()
                      .uploadScreenshotBytes(
                        screenshotBytes,
                        'feedback_${DateTime.now().millisecondsSinceEpoch}.png',
                      );
                  if (url != null) {
                    screenshotUrl = url;
                  } else {
                    uploadFailed = true;
                  }
                }

                if (!mounted) return;

                final ok =
                    await context.read<FeedbackProvider>().submitFeedback(
                          category: category,
                          content: text,
                          messageId: msg.id,
                          module: module,
                          screenshotUrl: screenshotUrl,
                        );
                if (mounted) {
                  final fbError = context.read<FeedbackProvider>().error;
                  if (ok && uploadFailed) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(
                        content: Text('反馈已提交，但截图上传失败'),
                        duration: Duration(seconds: 3),
                      ),
                    );
                  } else if (ok) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(
                        content: Text('反馈已提交，感谢！'),
                        duration: Duration(seconds: 2),
                      ),
                    );
                  } else {
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(
                        content: Text(
                            fbError.isNotEmpty ? '提交失败：$fbError' : '提交失败，请重试'),
                        duration: const Duration(seconds: 4),
                      ),
                    );
                  }
                }
              },
              child: const Text('提交'),
            ),
          ],
        ),
      ),
    );
  }

  /// 复制纠错反馈草稿（JSON）到剪贴板，供手工提交 AI 工具修复
  void _copyDraftJson(TextEditingController contentCtrl, String category,
      String module, Uint8List? screenshotBytes, bool screenshotValid,
      Message msg) {
    final dataUrl = (screenshotBytes != null && screenshotValid)
        ? 'data:image/png;base64,${base64Encode(screenshotBytes)}'
        : '';
    Clipboard.setData(ClipboardData(
      text: FeedbackReport.buildDraftJson(
        category: category,
        module: module,
        content: contentCtrl.text,
        screenshotDataUrl: dataUrl,
        messageId: msg.id,
      ),
    ));
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('已复制 JSON 数据，可粘贴到 AI 工具修复')),
      );
    }
  }

  /// 复制纠错反馈草稿（Markdown 报告）
  void _copyDraftMarkdown(TextEditingController contentCtrl, String category,
      String module, Uint8List? screenshotBytes, bool screenshotValid,
      Message msg) {
    final dataUrl = (screenshotBytes != null && screenshotValid)
        ? 'data:image/png;base64,${base64Encode(screenshotBytes)}'
        : '';
    Clipboard.setData(ClipboardData(
      text: FeedbackReport.buildDraftMarkdown(
        category: category,
        module: module,
        content: contentCtrl.text,
        screenshotDataUrl: dataUrl,
        messageId: msg.id,
      ),
    ));
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('已复制报告，可粘贴到 AI 工具修复')),
      );
    }
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
              width: 16,
              height: 16,
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
    final isListening = _isListening;

    return Container(
      padding: const EdgeInsets.fromLTRB(8, 8, 8, 12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border:
            Border(top: BorderSide(color: theme.colorScheme.outlineVariant)),
      ),
      child: SafeArea(
        top: false,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            // 录音状态条 + 实时识别文字 + 一键退出
            if (isListening)
              Container(
                width: double.infinity,
                margin: const EdgeInsets.only(bottom: 6),
                padding:
                    const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                decoration: BoxDecoration(
                  color: theme.colorScheme.errorContainer.withOpacity(0.4),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: theme.colorScheme.error.withOpacity(0.3),
                  ),
                ),
                child: Row(
                  children: [
                    _PulseIcon(
                        icon: Icons.graphic_eq, color: theme.colorScheme.error),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Text(
                            '正在聆听... 再次点击麦克风停止',
                            style: TextStyle(
                              color: theme.colorScheme.error,
                              fontSize: 12,
                              fontWeight: FontWeight.w500,
                            ),
                          ),
                          if (_interimText.isNotEmpty)
                            Padding(
                              padding: const EdgeInsets.only(top: 2),
                              child: Text(
                                _interimText,
                                style: TextStyle(
                                  color: theme.colorScheme.onErrorContainer,
                                  fontSize: 13,
                                  fontStyle: FontStyle.italic,
                                ),
                                maxLines: 2,
                                overflow: TextOverflow.ellipsis,
                              ),
                            ),
                        ],
                      ),
                    ),
                    // 一键退出按钮
                    IconButton(
                      onPressed: () => _stopVoiceInput(autoSend: false),
                      icon: Icon(Icons.close, color: theme.colorScheme.error),
                      tooltip: '退出录音',
                      visualDensity: VisualDensity.compact,
                    ),
                  ],
                ),
              ),
            // 输入栏
            Row(
              children: [
                // 麦克风按钮（点击切换）
                Material(
                  color: Colors.transparent,
                  child: InkWell(
                    customBorder: const CircleBorder(),
                    onTap: _toggleVoiceInput,
                    child: AnimatedContainer(
                      duration: const Duration(milliseconds: 200),
                      margin: const EdgeInsets.only(right: 4),
                      decoration: BoxDecoration(
                        color: isListening
                            ? theme.colorScheme.error
                            : theme.colorScheme.surfaceContainerHighest,
                        shape: BoxShape.circle,
                      ),
                      padding: const EdgeInsets.all(10),
                      child: Icon(
                        isListening ? Icons.stop : Icons.mic,
                        size: 22,
                        color: isListening
                            ? Colors.white
                            : theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ),
                ),
                // 文本输入框
                Expanded(
                  child: TextField(
                    controller: _inputCtrl,
                    maxLines: 4,
                    minLines: 1,
                    enabled: !isListening,
                    decoration: InputDecoration(
                      hintText: isListening ? '语音识别中...' : '输入你的问题...',
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(24),
                        borderSide: BorderSide.none,
                      ),
                      filled: true,
                      fillColor: theme.colorScheme.surfaceContainerHighest,
                      contentPadding: const EdgeInsets.symmetric(
                          horizontal: 16, vertical: 10),
                    ),
                    textInputAction: TextInputAction.send,
                    onSubmitted: (_) => _send(),
                  ),
                ),
                const SizedBox(width: 8),
                // 发送按钮
                IconButton.filled(
                  onPressed: sending
                      ? null
                      : (isListening
                          ? () => _stopVoiceInput(autoSend: true)
                          : _send),
                  icon: const Icon(Icons.send),
                  tooltip: isListening ? '停止并发送' : '发送',
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

/// 消息入场动画 — 新消息从下方滑入并淡入
class _SlideInItem extends StatefulWidget {
  final Widget child;
  const _SlideInItem({super.key, required this.child});

  @override
  State<_SlideInItem> createState() => _SlideInItemState();
}

class _SlideInItemState extends State<_SlideInItem>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;
  late final Animation<Offset> _slide;
  late final Animation<double> _fade;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 300),
    );
    _slide = Tween<Offset>(
      begin: const Offset(0, 0.15),
      end: Offset.zero,
    ).animate(CurvedAnimation(parent: _ctrl, curve: Curves.easeOutCubic));
    _fade = Tween<double>(begin: 0.0, end: 1.0).animate(_ctrl);
    _ctrl.forward();
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return SlideTransition(
      position: _slide,
      child: FadeTransition(opacity: _fade, child: widget.child),
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

class _PulseIconState extends State<_PulseIcon>
    with SingleTickerProviderStateMixin {
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

/// 智能体典型提问示例分组
class _AgentExampleGroup {
  final IconData icon;
  final Color color;
  final String name;

  /// 关联智能体类型（qa/policy/process/major/emotion），提问时自动选中
  final String agentType;
  final List<String> questions;

  const _AgentExampleGroup({
    required this.icon,
    required this.color,
    required this.name,
    required this.agentType,
    required this.questions,
  });
}
