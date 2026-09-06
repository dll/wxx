import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../utils/vopc_access.dart';

/// B3 L4 现实延伸引流卡片。
class VopcRealityExtensionCard extends StatelessWidget {
  const VopcRealityExtensionCard({super.key});

  Future<void> _open(BuildContext context) async {
    final uri = Uri.parse(vopcSiteUrl);
    try {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    } catch (_) {
      if (context.mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('无法打开链接，请稍后再试')));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      margin: EdgeInsets.zero,
      elevation: 0,
      shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
          side: BorderSide(color: theme.colorScheme.tertiary.withOpacity(.45))),
      child: Padding(
        padding: const EdgeInsets.all(18),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Container(
                width: 34,
                height: 34,
                decoration: BoxDecoration(
                    color: theme.colorScheme.tertiary.withOpacity(.12),
                    borderRadius: BorderRadius.circular(10)),
                child: Icon(Icons.public_rounded,
                    size: 19, color: theme.colorScheme.tertiary)),
            const SizedBox(width: 10),
            Text('现实延伸 · L4',
                style: theme.textTheme.titleMedium
                    ?.copyWith(fontWeight: FontWeight.w700)),
          ]),
          const SizedBox(height: 12),
          Text(
            '更真实的 AI 执行、云对象存储、真实经营与传播，即将在「虚拟OPC」网站上线。',
            style: theme.textTheme.bodyMedium,
          ),
          const SizedBox(height: 14),
          Align(
            alignment: Alignment.centerRight,
            child: FilledButton.tonalIcon(
              onPressed: () => _open(context),
              icon: const Icon(Icons.open_in_new, size: 18),
              label: const Text('前往虚拟OPC网站'),
            ),
          ),
        ]),
      ),
    );
  }
}
