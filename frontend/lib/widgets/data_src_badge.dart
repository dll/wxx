import 'package:flutter/material.dart';

/// 数据来源徽章（真实数据 / 参考AI）
///
/// 用于各类业务页面标注数据来源，贯彻「不瞎编」原则：
/// - `src == 'real'`      → 「真实数据」绿色徽章，点击弹窗说明来自系统内真实记录。
/// - 其他（reference/ai/fallback/空）→ 「参考/AI」琥珀色徽章，点击弹窗提示为模板或 AI 生成，仅作参考。
class DataSrcBadge extends StatelessWidget {
  final String? src;
  final String? note;
  const DataSrcBadge({super.key, this.src, this.note});

  bool get _real => src == 'real';

  @override
  Widget build(BuildContext context) {
    if (src == null || src!.isEmpty) return const SizedBox.shrink();
    final color = _real ? Colors.green : Colors.amber;
    return Align(
      alignment: Alignment.centerRight,
      child: InkWell(
        onTap: () => showDialog<void>(
          context: context,
          builder: (ctx) => AlertDialog(
            title: const Text('数据来源'),
            content: Text(_real
                ? (note ?? '✅ 真实数据：来自系统内记录实时统计，非模拟。')
                : '⚠️ 参考/AI 生成：当前无对应真实数据源，内容为模板或 AI 生成，仅作参考，请以线下核实为准。'),
            actions: [
              TextButton(
                  onPressed: () => Navigator.pop(ctx), child: const Text('知道了')),
            ],
          ),
        ),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
          decoration: BoxDecoration(
            color: color.withOpacity(0.12),
            borderRadius: BorderRadius.circular(10),
          ),
          child: Text(_real ? '真实数据' : '参考/AI',
              style: TextStyle(
                  fontSize: 11,
                  color: _real ? Colors.green.shade700 : Colors.orange.shade800)),
        ),
      ),
    );
  }
}
