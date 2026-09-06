import 'package:flutter/material.dart';

class ResourceStatChip extends StatelessWidget {
  const ResourceStatChip(
      {super.key,
      required this.label,
      required this.count,
      required this.color});
  final String label;
  final dynamic count;
  final Color color;
  @override
  Widget build(BuildContext context) {
    final value = count is int ? count : 0;
    return Padding(
        padding: const EdgeInsets.only(right: 8),
        child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
            decoration: BoxDecoration(
                color: color.withOpacity(0.1),
                borderRadius: BorderRadius.circular(10),
                border: Border.all(color: color.withOpacity(0.3))),
            child: Row(mainAxisSize: MainAxisSize.min, children: [
              Text('$value',
                  style: TextStyle(
                      color: color, fontWeight: FontWeight.w600, fontSize: 12)),
              const SizedBox(width: 4),
              Text(label,
                  style: TextStyle(color: color.withOpacity(0.8), fontSize: 11))
            ])));
  }
}
