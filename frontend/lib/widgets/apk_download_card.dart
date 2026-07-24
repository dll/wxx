import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import '../config/release_config.dart';

class ApkDownloadCard extends StatelessWidget {
  const ApkDownloadCard({super.key, this.compact = false});

  final bool compact;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final qrData = Uri.encodeComponent(ReleaseConfig.apkDownloadUrl);
    final qrUrl =
        'https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=$qrData&margin=10';
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(18),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: compact
            ? Column(
                children: [
                  _info(context),
                  const SizedBox(height: 14),
                  _qr(qrUrl),
                ],
              )
            : Row(
                children: [
                  Expanded(child: _info(context)),
                  const SizedBox(width: 16),
                  _qr(qrUrl),
                ],
              ),
      ),
    );
  }

  Widget _info(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            const Icon(Icons.android, color: Color(0xFF2E7D32)),
            const SizedBox(width: 8),
            Expanded(
              child: Text('手机扫码下载安装',
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontWeight: FontWeight.bold)),
            ),
          ],
        ),
        const SizedBox(height: 8),
        Text('无需登录即可下载蔚小芯 Android APK。扫码地址直接指向安装包，不会进入 Web 页面。',
            style: theme.textTheme.bodySmall
                ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
        const SizedBox(height: 10),
        Text(
            'v${ReleaseConfig.version}+${ReleaseConfig.buildNumber} · ${ReleaseConfig.apkFileName}',
            style: theme.textTheme.labelMedium),
        const SizedBox(height: 12),
        FilledButton.icon(
          onPressed: () => launchUrl(Uri.parse(ReleaseConfig.apkDownloadUrl),
              mode: LaunchMode.externalApplication),
          icon: const Icon(Icons.download_rounded, size: 18),
          label: const Text('直接下载 APK'),
        ),
      ],
    );
  }

  Widget _qr(String qrUrl) {
    return Container(
      width: 150,
      padding: const EdgeInsets.all(9),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Image.network(
            qrUrl,
            width: 132,
            height: 132,
            fit: BoxFit.cover,
            errorBuilder: (_, __, ___) => const SizedBox(
              width: 132,
              height: 132,
              child: Icon(Icons.qr_code_2, size: 96, color: Colors.black54),
            ),
          ),
          const SizedBox(height: 6),
          const Text('扫码下载',
              style: TextStyle(
                  color: Colors.black87, fontWeight: FontWeight.bold)),
        ],
      ),
    );
  }
}
