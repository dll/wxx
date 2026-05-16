import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/feedback_provider.dart';
import '../utils/screenshot_capture.dart';

/// 反馈提交对话框 — 自动截屏 + 分类选择 + 内容描述
///
/// 流程：先在不弹 dialog 时抓主页面帧 → 再弹出 dialog 把帧带进去预览
Future<void> showFeedbackDialog(BuildContext context) async {
  // 先抓当前主页面帧（dialog 还没弹，截到的是用户真实看到的画面）
  final shot = await captureScreenshot();

  if (!context.mounted) return;
  await showDialog<void>(
    context: context,
    barrierDismissible: false,
    builder: (ctx) => _FeedbackDialog(
      initialBytes: shot.bytes,
      initialError: shot.success ? null : shot.error,
    ),
  );
}

class _FeedbackDialog extends StatefulWidget {
  final Uint8List? initialBytes;
  final String? initialError;
  const _FeedbackDialog({this.initialBytes, this.initialError});

  @override
  State<_FeedbackDialog> createState() => _FeedbackDialogState();
}

class _FeedbackDialogState extends State<_FeedbackDialog> {
  String _category = 'answer_error';
  final _contentCtrl = TextEditingController();
  Uint8List? _screenshotBytes;
  bool _submitting = false;
  String? _screenshotError;

  @override
  void initState() {
    super.initState();
    _screenshotBytes = widget.initialBytes;
    _screenshotError = widget.initialError;
  }

  @override
  void dispose() {
    _contentCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_contentCtrl.text.trim().isEmpty && _category.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请输入反馈内容')),
      );
      return;
    }

    setState(() => _submitting = true);
    try {
      final provider = context.read<FeedbackProvider>();

      // 上传截图
      String screenshotUrl = '';
      if (_screenshotBytes != null && _screenshotBytes!.isNotEmpty) {
        final url = await provider.uploadScreenshotBytes(
          _screenshotBytes!,
          'feedback_${DateTime.now().millisecondsSinceEpoch}.png',
        );
        if (url != null) screenshotUrl = url;
      }

      if (!mounted) return;

      // 提交反馈
      final ok = await provider.submitFeedback(
        category: _category,
        content: _contentCtrl.text.isNotEmpty ? _contentCtrl.text : '(无文字描述)',
        screenshotUrl: screenshotUrl,
      );

      if (!mounted) return;

      if (ok) {
        Navigator.pop(context);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: const Text('反馈已提交，感谢您的反馈！'),
            backgroundColor: Theme.of(context).colorScheme.primary,
            behavior: SnackBarBehavior.floating,
          ),
        );
      } else {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(provider.error.isNotEmpty ? provider.error : '提交失败'),
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return AlertDialog(
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(24)),
      title: Row(
        children: [
          Container(
            width: 40, height: 40,
            decoration: BoxDecoration(
              color: const Color(0xFF6750A4).withValues(alpha: 0.12),
              shape: BoxShape.circle,
            ),
            child: const Icon(Icons.feedback_outlined, color: Color(0xFF6750A4)),
          ),
          const SizedBox(width: 12),
          const Text('提交反馈'),
        ],
      ),
      content: SizedBox(
        width: 420,
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // ── 截屏预览 ──
              _buildScreenshotPreview(theme),
              const SizedBox(height: 16),

              // ── 反馈类型 ──
              Text('反馈类型', style: theme.textTheme.labelLarge?.copyWith(fontWeight: FontWeight.w600)),
              const SizedBox(height: 8),
              SegmentedButton<String>(
                segments: const [
                  ButtonSegment(value: 'answer_error', label: Text('回答有误')),
                  ButtonSegment(value: 'suggestion', label: Text('功能建议')),
                  ButtonSegment(value: 'other', label: Text('其他')),
                ],
                selected: {_category},
                onSelectionChanged: (v) => setState(() => _category = v.first),
                style: _compactStyle,
              ),
              const SizedBox(height: 16),

              // ── 内容输入 ──
              TextField(
                controller: _contentCtrl,
                maxLines: 4,
                decoration: InputDecoration(
                  hintText: '请描述您遇到的问题或建议...',
                  border: const OutlineInputBorder(),
                  isDense: true,
                  filled: true,
                  fillColor: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.4),
                ),
                textInputAction: TextInputAction.newline,
              ),
            ],
          ),
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
                  width: 18, height: 18,
                  child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                )
              : const Icon(Icons.send, size: 18),
          label: Text(_submitting ? '提交中...' : '提交反馈'),
        ),
      ],
    );
  }

  Widget _buildScreenshotPreview(ThemeData theme) {
    return Container(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: theme.colorScheme.outlineVariant.withValues(alpha: 0.4)),
        color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.3),
      ),
      child: AspectRatio(
        aspectRatio: 16 / 9,
        child: ClipRRect(
          borderRadius: BorderRadius.circular(11),
          child: _screenshotBytes != null
              ? Stack(
                  fit: StackFit.expand,
                  children: [
                    Image.memory(
                      _screenshotBytes!,
                      fit: BoxFit.cover,
                      errorBuilder: (_, __, ___) => _buildNoScreenshot(theme),
                    ),
                    // 截屏成功标记
                    Positioned(
                      top: 8, right: 8,
                      child: Container(
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                        decoration: BoxDecoration(
                          color: Colors.green.withValues(alpha: 0.85),
                          borderRadius: BorderRadius.circular(12),
                        ),
                        child: const Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Icon(Icons.check, color: Colors.white, size: 14),
                            SizedBox(width: 4),
                            Text('已截屏', style: TextStyle(color: Colors.white, fontSize: 11)),
                          ],
                        ),
                      ),
                    ),
                  ],
                )
              : _buildNoScreenshot(theme),
        ),
      ),
    );
  }

  Widget _buildNoScreenshot(ThemeData theme) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.screenshot_outlined, size: 28, color: theme.colorScheme.onSurfaceVariant),
            const SizedBox(height: 4),
            Text(
              _screenshotError ?? '截图不可用',
              style: theme.textTheme.labelSmall?.copyWith(
                color: _screenshotError != null ? theme.colorScheme.error : null,
              ),
              textAlign: TextAlign.center,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
            ),
          ],
        ),
      ),
    );
  }

  static const _compactStyle = ButtonStyle(
    visualDensity: VisualDensity.compact,
    textStyle: WidgetStatePropertyAll(TextStyle(fontSize: 12)),
    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
  );
}
