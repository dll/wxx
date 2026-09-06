import 'package:flutter/material.dart';

class VopcErrorCard extends StatelessWidget {
  const VopcErrorCard(
      {super.key,
      required this.message,
      required this.code,
      required this.onRetry});
  final String message;
  final int? code;
  final Future<void> Function() onRetry;
  @override
  Widget build(BuildContext context) => Card(
      color: Theme.of(context).colorScheme.errorContainer,
      child: Padding(
          padding: const EdgeInsets.all(20),
          child: Column(children: [
            Icon(code == 403 ? Icons.lock_outline : Icons.error_outline),
            const SizedBox(height: 8),
            Text(code == null ? message : 'HTTP $code · $message'),
            const SizedBox(height: 12),
            OutlinedButton(onPressed: onRetry, child: const Text('重试'))
          ])));
}
