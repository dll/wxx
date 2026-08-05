import 'dart:convert' show HtmlEscape;
import '../../utils/web_export.dart';
import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
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
import '../../utils/page_reload.dart';
import '../../utils/screenshot_capture.dart';
import '../../widgets/answer_card.dart';
import '../../widgets/export_dialog.dart';

const _htmlEscaper = HtmlEscape();

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
        title: const Text('蔚小芯'),
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

  /// 智能体选择器 — 水平滚动的 chip 列表
  Widget _buildAgentSelector(ChatProvider chat, ThemeData theme) {
    if (chat.agents.isEmpty) return const SizedBox.shrink();

    final agents = chat.agents;
    return Container(
      height: 44,
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border(
          bottom: BorderSide(
            color: theme.colorScheme.outlineVariant.withOpacity(0.3),
          ),
        ),
      ),
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        itemCount: agents.length + 1, // +1 for "默认"
        separatorBuilder: (_, __) => const SizedBox(width: 6),
        itemBuilder: (context, index) {
          if (index == 0) {
            // 默认智能体选项
            final selected = chat.selectedAgentId == null;
            return FilterChip(
              label: const Text('默认', style: TextStyle(fontSize: 12)),
              selected: selected,
              onSelected: (_) => chat.selectAgent(null),
              materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
              visualDensity: VisualDensity.compact,
              selectedColor: theme.colorScheme.primaryContainer,
              showCheckmark: false,
            );
          }
          final agent = agents[index - 1];
          final selected = chat.selectedAgentId == agent.agentId;
          return FilterChip(
            label: Text(agent.name, style: const TextStyle(fontSize: 12)),
            selected: selected,
            onSelected: (_) => chat.selectAgent(
              selected ? null : agent.agentId,
            ),
            materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
            visualDensity: VisualDensity.compact,
            selectedColor: theme.colorScheme.primaryContainer,
            avatar: _agentTypeIcon(agent.agentType),
            showCheckmark: false,
          );
        },
      ),
    );
  }

  /// 智能体类型小图标
  Widget _agentTypeIcon(String type) {
    switch (type) {
      case 'qa':
        return const Icon(Icons.chat, size: 14, color: Color(0xFF1565C0));
      case 'policy':
        return const Icon(Icons.gavel, size: 14, color: Color(0xFFE65100));
      case 'emotion':
        return const Icon(Icons.favorite_border,
            size: 14, color: Color(0xFFC62828));
      default:
        return const Icon(Icons.smart_toy, size: 14, color: Color(0xFF7B1FA2));
    }
  }

  Widget _buildEmptyState(ThemeData theme) {
    return Center(
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
            ]
                .map((q) => ActionChip(
                      label: Text(q, style: const TextStyle(fontSize: 13)),
                      onPressed: () {
                        _inputCtrl.text = q;
                        _send();
                      },
                    ))
                .toList(),
          ),
        ],
      ),
    );
  }

  Widget _buildMessage(Message msg, int index) {
    if (msg.isUser) {
      return _buildUserBubble(msg);
    }
    return _buildAssistantMessage(msg, index);
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

  Widget _buildAssistantMessage(Message msg, int msgIndex) {
    final chat = context.read<ChatProvider>();
    final isPlayingThis = chat.isPlaying && chat.playingIndex == msgIndex;
    final theme = Theme.of(context);

    // 找到此回答对应的用户提问
    final question = _findQuestionFor(msgIndex);

    // 操作栏：朗读 + 复制 + PDF + 收藏 + 保存到知识库
    Widget actionBar = Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Consumer<BookmarkProvider>(
        builder: (_, bm, __) {
          final isMarked = bm.isBookmarked(msg.content);
          return Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              _ActionChip(
                icon: isPlayingThis ? Icons.stop_circle_outlined : Icons.volume_up,
                label: isPlayingThis ? '停止' : '朗读',
                onTap: () => chat.playTTS(msgIndex),
              ),
              const SizedBox(width: 4),
              _ActionChip(
                icon: Icons.content_copy,
                label: '复制',
                onTap: () => _copyAnswer(msg),
              ),
              const SizedBox(width: 4),
              _ActionChip(
                icon: Icons.download,
                label: '导出',
                onTap: () => _showExportDialog(question, msg),
              ),
              const SizedBox(width: 4),
              _ActionChip(
                icon: Icons.feedback_outlined,
                label: '纠错',
                onTap: () => _showFeedbackDialog(msg),
              ),
              const SizedBox(width: 4),
              _ActionChip(
                icon: isMarked ? Icons.star : Icons.star_outline,
                label: isMarked ? '已收藏' : '收藏',
                onTap: () => bm.toggle(
                  question: question,
                  conclusion: msg.content,
                  sources: msg.answerCard?.sources.map((s) => s.title).toList() ?? [],
                  followUps: msg.answerCard?.followUps ?? [],
                ),
              ),
              const SizedBox(width: 4),
              _ActionChip(
                icon: Icons.save_outlined,
                label: '保存',
                onTap: () => _saveToKnowledgeBase(question, msg),
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
            child: actionBar,
          ),
          // 无知识库引用时提示用户前往知识大厅
          if (msg.answerCard == null || msg.answerCard!.sources.isEmpty)
            Padding(
              padding: const EdgeInsets.only(left: 20, top: 2),
              child: GestureDetector(
                onTap: () => context.go('/browse'),
                child: Text(
                  '💡 该回答未引用知识库，仅供参考。前往知识大厅浏览已收录内容 →',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.primary,
                    fontSize: 11,
                  ),
                ),
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
  /// Web 端：删除成功后自动整页刷新（reloadPage），从根上规避
  /// CanvasKit 渲染冻结导致的空白页（旧方案仅应用内重建无法保证恢复）。
  /// 移动端：乐观调 newChat 清空当前对话 + 重建页面（_rebuildKey++），
  /// 彻底清除 ListView/动画等子组件残留状态。
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

    // 移动端先乐观清空界面，避免等待期间仍显示旧会话；
    // Web 端不做本地清理（防止重建触发渲染冻结），依赖整页刷新兜底。
    if (!kIsWeb) {
      chat.newChat();
    }

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

    if (kIsWeb) {
      // 删除成功 → 自动整页刷新，回到全新对话（用户无需手动刷新）
      if (ok) {
        reloadPage();
        return;
      }
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('删除失败，请稍后重试')),
      );
      return;
    }

    // 移动端：重建页面清除 ListView/动画等子组件残留状态
    setState(() => _rebuildKey++);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(ok ? '对话已删除' : '删除失败，请稍后重试')),
    );
  }

  /// 保存问答对到知识库
  Future<void> _saveToKnowledgeBase(String question, Message msg) async {
    if (!mounted) return;
    final kb = context.read<KnowledgeProvider>();
    final title = question.length > 50
        ? '${question.substring(0, 50)}...'
        : question;
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
    Uint8List? screenshotBytes = shot.bytes;

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
                  // 截屏预览
                  if (screenshotBytes != null)
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
                              Image.memory(screenshotBytes, fit: BoxFit.cover),
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
                          Text(shot.error ?? '截图不可用',
                              style: Theme.of(context)
                                  .textTheme
                                  .labelSmall
                                  ?.copyWith(
                                      color:
                                          Theme.of(context).colorScheme.error)),
                        ],
                      ),
                    ),
                  if (screenshotBytes != null) const SizedBox(height: 16),

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
                  TextField(
                    controller: contentCtrl,
                    maxLines: 4,
                    decoration: const InputDecoration(
                      hintText: '请描述问题或建议...',
                      border: OutlineInputBorder(),
                    ),
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

                // 上传截图
                String screenshotUrl = '';
                bool uploadFailed = false;
                if (screenshotBytes != null && screenshotBytes.isNotEmpty) {
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

/// 消息操作芯片 — 用于朗读、复制、PDF、收藏等操作
class _ActionChip extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  const _ActionChip(
      {required this.icon, required this.label, required this.onTap});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Material(
      color: Colors.transparent,
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 4),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 14, color: theme.colorScheme.outline),
              const SizedBox(width: 2),
              Text(
                label,
                style:
                    TextStyle(fontSize: 11, color: theme.colorScheme.outline),
              ),
            ],
          ),
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
