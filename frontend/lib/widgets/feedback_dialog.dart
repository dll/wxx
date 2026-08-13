import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import '../providers/feedback_provider.dart';
import '../models/models.dart';
import '../utils/feedback_report.dart';
import '../utils/screenshot_capture.dart';
import '../utils/storage.dart';
import 'feedback_screenshot.dart';

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
  String _module = '';
  final _contentCtrl = TextEditingController();
  Uint8List? _screenshotBytes;
  bool _submitting = false;
  String? _screenshotError;
  bool _imageDecoded = false;
  bool _retryingCapture = false;

  @override
  void initState() {
    super.initState();
    _screenshotBytes = widget.initialBytes;
    _screenshotError = widget.initialError;
    if (_screenshotBytes != null && _screenshotBytes!.isNotEmpty) {
      _validateScreenshot();
    }
    // 恢复本地草稿（上次提交失败或未完成的输入）
    final draft = Storage.feedbackDraft;
    if (draft.isNotEmpty) {
      _contentCtrl.text = draft;
      final cat = Storage.feedbackDraftCategory;
      final mod = Storage.feedbackDraftModule;
      if (cat.isNotEmpty) _category = cat;
      if (mod.isNotEmpty) _module = mod;
    }
  }

  /// 校验截图像素是否真实可解码，避免"已截屏"但实际无图
  Future<void> _validateScreenshot() async {
    final bytes = _screenshotBytes;
    if (bytes == null || bytes.isEmpty) {
      setState(() => _imageDecoded = false);
      return;
    }
    final ok = await isDecodableImage(bytes);
    if (!mounted) return;
    setState(() {
      _imageDecoded = ok;
      if (!ok) _screenshotError = '截图无效，已忽略（截图可选）';
    });
  }

  /// 重新截屏（用户可手动补充佐证）
  Future<void> _retryCapture() async {
    if (_retryingCapture) return;
    setState(() => _retryingCapture = true);
    final shot = await captureScreenshot();
    if (!mounted) return;
    setState(() {
      _retryingCapture = false;
      _screenshotBytes = shot.bytes;
      _screenshotError = shot.success ? null : (shot.error ?? '截图不可用');
      _imageDecoded = false;
    });
    if (shot.success) _validateScreenshot();
  }

  @override
  void dispose() {
    // 关闭对话框时保存草稿（未提交的输入不丢失）
    _saveDraft();
    _contentCtrl.dispose();
    super.dispose();
  }

  void _saveDraft() {
    Storage.saveFeedbackDraft(
      content: _contentCtrl.text,
      category: _category,
      module: _module,
    );
  }

  /// 有效截图转 data URL（供复制报告内嵌，确保在线修复有佐证）
  String get _screenshotDataUrl {
    final bytes = _screenshotBytes;
    if (bytes == null || bytes.isEmpty || !_imageDecoded) return '';
    return 'data:image/png;base64,${base64Encode(bytes)}';
  }

  /// 复制全部反馈数据（JSON）到剪贴板，供手工提交 AI 工具修复
  Future<void> _copyJson() async {
    final text = FeedbackReport.buildDraftJson(
      category: _category,
      module: _module,
      content: _contentCtrl.text,
      screenshotDataUrl: _screenshotDataUrl,
    );
    await Clipboard.setData(ClipboardData(text: text));
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('已复制 JSON 数据，可粘贴到 AI 工具修复')),
      );
    }
  }

  /// 复制完整 Markdown 报告（反馈信息 + 内容 + 截图 base64）
  Future<void> _copyMarkdown() async {
    final text = FeedbackReport.buildDraftMarkdown(
      category: _category,
      module: _module,
      content: _contentCtrl.text,
      screenshotDataUrl: _screenshotDataUrl,
    );
    await Clipboard.setData(ClipboardData(text: text));
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('已复制报告，可粘贴到 AI 工具修复')),
      );
    }
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

      // 上传截图（仅真实有效的截图才上传）
      String screenshotUrl = '';
      if (_screenshotBytes != null &&
          _screenshotBytes!.isNotEmpty &&
          _imageDecoded) {
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
        module: _module,
        screenshotUrl: screenshotUrl,
      );

      if (!mounted) return;

      if (ok) {
        // 提交成功，清除草稿
        Storage.clearFeedbackDraft();
        Navigator.pop(context);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: const Text('反馈已提交，感谢您的反馈！'),
            backgroundColor: Theme.of(context).colorScheme.primary,
            behavior: SnackBarBehavior.floating,
          ),
        );
      } else {
        // 提交失败：保留输入（草稿已在 dispose 保存），提示重试
        _saveDraft();
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
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: const Color(0xFF6750A4).withOpacity(0.12),
              shape: BoxShape.circle,
            ),
            child:
                const Icon(Icons.feedback_outlined, color: Color(0xFF6750A4)),
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
              Text('反馈类型',
                  style: theme.textTheme.labelLarge
                      ?.copyWith(fontWeight: FontWeight.w600)),
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

              // ── 所属模块 ──
              Text('所属模块（便于快速修复）',
                  style: theme.textTheme.labelLarge
                      ?.copyWith(fontWeight: FontWeight.w600)),
              const SizedBox(height: 8),
              DropdownButtonFormField<String>(
                value: _module.isEmpty ? null : _module,
                isExpanded: true,
                decoration: InputDecoration(
                  hintText: '选择问题所属模块（可选）',
                  border: const OutlineInputBorder(),
                  isDense: true,
                  filled: true,
                  fillColor: theme.colorScheme.surfaceContainerHighest
                      .withOpacity(0.4),
                ),
                items: [
                  const DropdownMenuItem<String>(
                    value: '',
                    child: Text('暂不选择'),
                  ),
                  for (final m in feedbackModules)
                    DropdownMenuItem<String>(value: m, child: Text(m)),
                ],
                onChanged: (v) => setState(() => _module = v ?? ''),
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
                  fillColor: theme.colorScheme.surfaceContainerHighest
                      .withOpacity(0.4),
                ),
                textInputAction: TextInputAction.newline,
              ),
              const SizedBox(height: 12),
              // ── 复制反馈数据（手工提交 AI 工具修复）──
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton.icon(
                      onPressed: _copyJson,
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
                      onPressed: _copyMarkdown,
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
                style: theme.textTheme.labelSmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
                textAlign: TextAlign.center,
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
                  width: 18,
                  height: 18,
                  child: CircularProgressIndicator(
                      strokeWidth: 2, color: Colors.white),
                )
              : const Icon(Icons.send, size: 18),
          label: Text(_submitting ? '提交中...' : '提交反馈'),
        ),
      ],
    );
  }

  Widget _buildScreenshotPreview(ThemeData theme) {
    final showImage = _screenshotBytes != null &&
        _screenshotBytes!.isNotEmpty &&
        _imageDecoded;
    return Container(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
            color: theme.colorScheme.outlineVariant.withOpacity(0.4)),
        color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.3),
      ),
      child: AspectRatio(
        aspectRatio: 16 / 9,
        child: ClipRRect(
          borderRadius: BorderRadius.circular(11),
          child: showImage
              ? Stack(
                  fit: StackFit.expand,
                  children: [
                    Image.memory(
                      _screenshotBytes!,
                      fit: BoxFit.cover,
                      errorBuilder: (_, __, ___) => _buildNoScreenshot(theme),
                    ),
                    // 截屏成功标记（仅当真实可解码时才显示）
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
                            Icon(Icons.check, color: Colors.white, size: 14),
                            SizedBox(width: 4),
                            Text('已截屏',
                                style: TextStyle(
                                    color: Colors.white, fontSize: 11)),
                          ],
                        ),
                      ),
                    ),
                    Positioned(
                      bottom: 8,
                      right: 8,
                      child: _retryButton(theme),
                    ),
                  ],
                )
              : Stack(
                  fit: StackFit.expand,
                  children: [
                    _buildNoScreenshot(theme),
                    Positioned(bottom: 8, right: 8, child: _retryButton(theme)),
                  ],
                ),
        ),
      ),
    );
  }

  /// 重新截屏按钮（截图可选佐证，允许用户重试）
  Widget _retryButton(ThemeData theme) {
    return Material(
      color: Colors.black.withOpacity(0.55),
      borderRadius: BorderRadius.circular(16),
      child: InkWell(
        borderRadius: BorderRadius.circular(16),
        onTap: _retryingCapture ? null : _retryCapture,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              _retryingCapture
                  ? const SizedBox(
                      width: 12,
                      height: 12,
                      child: CircularProgressIndicator(
                          strokeWidth: 2, color: Colors.white),
                    )
                  : const Icon(Icons.refresh, color: Colors.white, size: 14),
              const SizedBox(width: 4),
              Text(_retryingCapture ? '截屏中...' : '重新截屏',
                  style: const TextStyle(color: Colors.white, fontSize: 11)),
            ],
          ),
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
            Icon(Icons.screenshot_outlined,
                size: 28, color: theme.colorScheme.onSurfaceVariant),
            const SizedBox(height: 4),
            Text(
              _screenshotError ?? '未截屏（截图可选）',
              style: theme.textTheme.labelSmall?.copyWith(
                color:
                    _screenshotError != null ? theme.colorScheme.error : null,
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
