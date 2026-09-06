import 'package:flutter/material.dart';

import 'vopc_meta_chip.dart';

class VopcInvitationCard extends StatelessWidget {
  final Map<String, dynamic> invitation;
  final VoidCallback onDecline;
  final VoidCallback onAccept;

  const VopcInvitationCard({
    super.key,
    required this.invitation,
    required this.onDecline,
    required this.onAccept,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
        margin: const EdgeInsets.only(bottom: 10),
        child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 14, 10, 10),
            child:
                Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(Icons.mail_outline_rounded,
                    color: theme.colorScheme.primary),
                const SizedBox(width: 10),
                Expanded(
                    child: Text(
                        invitation['project_name']?.toString() ?? '项目邀请',
                        style: theme.textTheme.titleMedium
                            ?.copyWith(fontWeight: FontWeight.w700))),
                VopcMetaChip(invitation['project_role']?.toString() ?? '成员')
              ]),
              if ((invitation['message']?.toString() ?? '').isNotEmpty) ...[
                const SizedBox(height: 8),
                Text(invitation['message'].toString(),
                    style: theme.textTheme.bodySmall)
              ],
              Align(
                  alignment: Alignment.centerRight,
                  child: Wrap(spacing: 4, children: [
                    TextButton(onPressed: onDecline, child: const Text('拒绝')),
                    FilledButton.tonal(
                        onPressed: onAccept, child: const Text('接受邀请'))
                  ])),
            ])));
  }
}
