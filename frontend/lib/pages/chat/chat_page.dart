import 'dart:convert' show HtmlEscape;
import '../../utils/web_export.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import '../../models/models.dart';
import '../../providers/chat_provider.dart';
import '../../providers/bookmark_provider.dart';
import '../../providers/feedback_provider.dart';
import '../../services/voice/voice_navigator.dart';
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
        curve: Curves.easeOutCubic,
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
                    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                    child: Text(
                      chat.error!,
                      style: TextStyle(color: theme.colorScheme.onErrorContainer, fontSize: 13),
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
            color: theme.colorScheme.outlineVariant.withValues(alpha: 0.3),
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
        return const Icon(Icons.favorite_border, size: 14, color: Color(0xFFC62828));
      default:
        return const Icon(Icons.smart_toy, size: 14, color: Color(0xFF7B1FA2));
    }
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

  Widget _buildAssistantMessage(Message msg, int msgIndex) {
    final chat = context.read<ChatProvider>();
    final bookmarkProv = context.read<BookmarkProvider>();
    final isPlayingThis = chat.isPlaying && chat.playingIndex == msgIndex;
    final theme = Theme.of(context);

    // 找到此回答对应的用户提问
    final question = _findQuestionFor(msgIndex);
    final isMarked = bookmarkProv.isBookmarked(msg.content);

    // 操作栏：朗读 + 复制 + PDF + 收藏
    Widget actionBar = Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          // 朗读按钮
          _ActionChip(
            icon: isPlayingThis ? Icons.stop_circle_outlined : Icons.volume_up,
            label: isPlayingThis ? '停止' : '朗读',
            onTap: () => chat.playTTS(msgIndex),
          ),
          const SizedBox(width: 4),
          // 复制按钮
          _ActionChip(
            icon: Icons.content_copy,
            label: '复制',
            onTap: () => _copyAnswer(msg),
          ),
          const SizedBox(width: 4),
          // 多格式导出按钮
          _ActionChip(
            icon: Icons.download,
            label: '导出',
            onTap: () => _showExportDialog(question, msg),
          ),
          const SizedBox(width: 4),
          // 纠错反馈按钮
          _ActionChip(
            icon: Icons.feedback_outlined,
            label: '纠错',
            onTap: () => _showFeedbackDialog(msg),
          ),
          const SizedBox(width: 4),
          // 收藏按钮
          _ActionChip(
            icon: isMarked ? Icons.star : Icons.star_outline,
            label: isMarked ? '已收藏' : '收藏',
            onTap: () => bookmarkProv.toggle(
              question: question,
              conclusion: msg.content,
              sources: msg.answerCard?.sources.map((s) => s.title).toList() ?? [],
              followUps: msg.answerCard?.followUps ?? [],
            ),
          ),
        ],
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

  String _buildExportHtml(String question, Message msg, {bool forPrint = false, bool forPng = false}) {
    final esc = _htmlEscaper.convert;
    final sources = msg.answerCard?.sources ?? [];
    final pngExtra = forPng ? ' width: 600px; margin: 0 auto;' : '';
    final answerStyle = forPng ? ' white-space: pre-wrap;' : '';

    String sourceHtml = '';
    if (sources.isNotEmpty) {
      if (forPng) {
        sourceHtml = '<div class="source"><strong>来源：</strong>${esc(sources.map((s) => s.title).join(', '))}</div>';
      } else {
        sourceHtml = '<div class="source"><strong>信息来源：</strong><ul>${sources.map((s) => '<li>${esc(s.title)}</li>').join()}</ul></div>';
      }
    }

    final printScript = forPrint ? '<script>window.onload = () => window.print();</script>' : '';
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
    final format = await ExportDialog.show(context, contentId: msg.id ?? '');
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
    final sources = msg.answerCard?.sources.map((s) => '- ${s.title}').join('\n') ?? '';
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
    md.writeln('> 导出时间：${DateTime.now().toString().substring(0, 19)} · 蔚小芯 AI 学工助手');

    Clipboard.setData(ClipboardData(text: md.toString()));
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Markdown 已复制到剪贴板'), duration: Duration(seconds: 2)),
      );
    }
  }

  /// 纠错反馈对话框
  void _showFeedbackDialog(Message msg) {
    final contentCtrl = TextEditingController();
    String category = 'answer_error';

    showDialog(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setState) => AlertDialog(
          title: const Text('反馈纠错'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('反馈类型', style: TextStyle(fontWeight: FontWeight.w500)),
              const SizedBox(height: 8),
              SegmentedButton<String>(
                segments: const [
                  ButtonSegment(value: 'answer_error', label: Text('回答有误')),
                  ButtonSegment(value: 'suggestion', label: Text('建议改进')),
                  ButtonSegment(value: 'other', label: Text('其他')),
                ],
                selected: {category},
                onSelectionChanged: (v) => setState(() => category = v.first),
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
                final ok = await context.read<FeedbackProvider>().submitFeedback(
                  category: category,
                  content: text,
                  messageId: msg.id ?? '',
                );
                if (mounted) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    SnackBar(content: Text(ok ? '反馈已提交，感谢！' : '提交失败，请重试')),
                  );
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

  const _ActionChip({required this.icon, required this.label, required this.onTap});

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
                style: TextStyle(fontSize: 11, color: theme.colorScheme.outline),
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
