import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/feedback_provider.dart';

/// 反馈提交对话框 — 包含分类选择、内容输入、截图上传、自动提交
Future<void> showFeedbackDialog(BuildContext context) {
  return showDialog(
    context: context,
    builder: (ctx) => const _FeedbackDialog(),
  );
}

class _FeedbackDialog extends StatefulWidget {
  const _FeedbackDialog();

  @override
  State<_FeedbackDialog> createState() => _FeedbackDialogState();
}

class _FeedbackDialogState extends State<_FeedbackDialog> {
  String _category = 'answer_error';
  final _contentCtrl = TextEditingController();
  bool _submitting = false;

  @override
  void dispose() {
    _contentCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return AlertDialog(
      title: Row(
        children: [
          Icon(Icons.feedback_outlined, color: theme.colorScheme.primary),
          const SizedBox(width: 8),
          const Text('提交反馈'),
        ],
      ),
      content: SizedBox(
        width: 400,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            // 分类选择
            Text('反馈类型', style: theme.textTheme.labelLarge),
            const SizedBox(height: 8),
            SegmentedButton<String>(
              segments: const [
                ButtonSegment(value: 'answer_error', label: Text('回答有误')),
                ButtonSegment(value: 'suggestion', label: Text('功能建议')),
                ButtonSegment(value: 'other', label: Text('其他')),
              ],
              selected: {_category},
              onSelectionChanged: (v) => setState(() => _category = v.first),
              style: ButtonStyle(
                visualDensity: VisualDensity.compact,
                textStyle: WidgetStateProperty.all(const TextStyle(fontSize: 12)),
                tapTargetSize: MaterialTapTargetSize.shrinkWrap,
              ),
            ),
            const SizedBox(height: 16),
            // 内容输入
            TextField(
              controller: _contentCtrl,
              maxLines: 4,
              decoration: InputDecoration(
                hintText: '请描述您遇到的问题或建议...',
                border: const OutlineInputBorder(),
                isDense: true,
                filled: true,
                fillColor: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.3),
              ),
              textInputAction: TextInputAction.newline,
            ),
            const SizedBox(height: 12),
            // 提示信息
            Text(
              '提交后将自动截取当前页面截图作为参考',
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: _submitting ? null : () => Navigator.pop(context),
          child: const Text('取消'),
        ),
        FilledButton.icon(
          onPressed: _submitting ? null : _submit,
          icon: _submitting
              ? const SizedBox(
                  width: 16, height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Icon(Icons.send, size: 18),
          label: Text(_submitting ? '提交中...' : '提交反馈'),
        ),
      ],
    );
  }

  Future<void> _submit() async {
    setState(() => _submitting = true);
    try {
      final provider = context.read<FeedbackProvider>();
      final ok = await provider.submitFeedback(
        category: _category,
        content: _contentCtrl.text,
      );
      if (context.mounted) {
        if (ok) {
          Navigator.pop(context);
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('反馈已提交，感谢您的反馈！')),
          );
        } else {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text(provider.error.isNotEmpty ? provider.error : '提交失败，请稍后重试'),
              backgroundColor: Theme.of(context).colorScheme.error,
            ),
          );
        }
      }
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }
}
