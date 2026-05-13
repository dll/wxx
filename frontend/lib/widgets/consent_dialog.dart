import 'package:flutter/material.dart';

/// 隐私政策与用户协议授权弹窗
/// 首次启动时弹出，需用户勾选同意后方可继续使用
class ConsentDialog extends StatefulWidget {
  final VoidCallback onConsent;

  const ConsentDialog({super.key, required this.onConsent});

  /// 显示授权弹窗
  static Future<bool?> show(BuildContext context) {
    return showDialog<bool>(
      context: context,
      barrierDismissible: false,
      builder: (_) => ConsentDialog(
        onConsent: () => Navigator.of(context).pop(true),
      ),
    );
  }

  @override
  State<ConsentDialog> createState() => _ConsentDialogState();
}

class _ConsentDialogState extends State<ConsentDialog> {
  bool _privacyAccepted = false;
  bool _termsAccepted = false;

  bool get _allAccepted => _privacyAccepted && _termsAccepted;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return AlertDialog(
      title: const Text('欢迎使用蔚小芯'),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              '使用前请阅读并同意以下协议：',
              style: theme.textTheme.bodyMedium,
            ),
            const SizedBox(height: 16),

            // 隐私政策
            CheckboxListTile(
              value: _privacyAccepted,
              onChanged: (v) => setState(() => _privacyAccepted = v ?? false),
              title: const Text('《蔚小芯隐私政策》'),
              subtitle: const Text('了解我们如何收集、使用和保护你的信息'),
              controlAffinity: ListTileControlAffinity.leading,
              dense: true,
            ),

            // 用户协议
            CheckboxListTile(
              value: _termsAccepted,
              onChanged: (v) => setState(() => _termsAccepted = v ?? false),
              title: const Text('《蔚小芯用户协议》'),
              subtitle: const Text('了解服务规则与免责声明'),
              controlAffinity: ListTileControlAffinity.leading,
              dense: true,
            ),

            const SizedBox(height: 8),
            Text(
              '同意后即可使用全部功能。你可以随时在"个人中心"查看协议内容。',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: _allAccepted ? widget.onConsent : null,
          child: const Text('同意并继续'),
        ),
      ],
    );
  }
}
