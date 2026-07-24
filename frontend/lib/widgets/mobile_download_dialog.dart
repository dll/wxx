import 'package:flutter/material.dart';

import 'apk_download_card.dart';

/// 显示「移动」APK 下载弹窗
Future<void> showMobileDownloadDialog(BuildContext context) async {
  return showDialog(
    context: context,
    builder: (context) => const MobileDownloadDialog(),
  );
}

class MobileDownloadDialog extends StatelessWidget {
  const MobileDownloadDialog({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return AlertDialog(
      backgroundColor: theme.colorScheme.surface,
      surfaceTintColor: Colors.transparent,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(20),
        side: BorderSide(color: theme.colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      contentPadding: const EdgeInsets.fromLTRB(20, 20, 20, 8),
      content: SizedBox(
        width: 420,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Icon(Icons.smartphone_outlined,
                    color: theme.colorScheme.primary, size: 28),
                const SizedBox(width: 10),
                Text(
                  '移动应用下载',
                  style: theme.textTheme.titleLarge?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),
            const ApkDownloadCard(compact: true),
            const SizedBox(height: 16),
            TextButton.icon(
              onPressed: () => Navigator.of(context).pop(),
              icon: const Icon(Icons.close, size: 18),
              label: const Text('关闭'),
            ),
          ],
        ),
      ),
    );
  }
}
