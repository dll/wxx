import 'package:flutter/material.dart';

/// 导出格式选择对话框
/// 支持 PDF / PNG 长图导出，后续可扩展 Word/ICS
class ExportDialog extends StatelessWidget {
  final String contentId;
  final void Function(String format) onExport;

  const ExportDialog({
    super.key,
    required this.contentId,
    required this.onExport,
  });

  static Future<String?> show(BuildContext context, {required String contentId}) {
    return showDialog<String>(
      context: context,
      builder: (ctx) => ExportDialog(
        contentId: contentId,
        onExport: (format) => Navigator.of(ctx).pop(format),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('导出格式'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          _buildOption(
            context,
            icon: Icons.picture_as_pdf,
            label: 'PDF 文档',
            description: '含标题、时间、来源脚注、水印',
            format: 'pdf',
          ),
          const SizedBox(height: 8),
          _buildOption(
            context,
            icon: Icons.image,
            label: 'PNG 长图',
            description: '适合分享到微信/朋友圈',
            format: 'png',
          ),
          const SizedBox(height: 8),
          _buildOption(
            context,
            icon: Icons.code,
            label: 'Markdown',
            description: '纯文本格式，方便编辑',
            format: 'md',
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('取消'),
        ),
      ],
    );
  }

  Widget _buildOption(BuildContext context, {
    required IconData icon,
    required String label,
    required String description,
    required String format,
  }) {
    final theme = Theme.of(context);
    return InkWell(
      borderRadius: BorderRadius.circular(8),
      onTap: () => onExport(format),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          border: Border.all(color: theme.colorScheme.outlineVariant),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          children: [
            Icon(icon, color: theme.colorScheme.primary),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(label, style: theme.textTheme.titleSmall),
                  Text(description, style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.outline,
                  )),
                ],
              ),
            ),
            Icon(Icons.chevron_right, color: theme.colorScheme.outline),
          ],
        ),
      ),
    );
  }
}
