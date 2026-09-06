import 'package:flutter/material.dart';

class VopcQuizCard extends StatefulWidget {
  final String question;
  final List<String> options;
  final int? answer;
  const VopcQuizCard(
      {super.key, required this.question, required this.options, this.answer});
  @override
  State<VopcQuizCard> createState() => _VopcQuizCardState();
}

class _VopcQuizCardState extends State<VopcQuizCard> {
  int? _selected;
  bool _checked = false;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final correct = _checked && _selected == widget.answer;
    final wrong = _checked && _selected != widget.answer;
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(widget.question,
              style: theme.textTheme.titleSmall
                  ?.copyWith(fontWeight: FontWeight.w700)),
          const SizedBox(height: 8),
          for (var i = 0; i < widget.options.length; i++)
            RadioListTile<int>(
              dense: true,
              value: i,
              groupValue: _selected,
              onChanged: _checked
                  ? null
                  : (value) => setState(() => _selected = value),
              title: Text(widget.options[i]),
              secondary: _checked
                  ? Icon(
                      i == widget.answer
                          ? Icons.check_circle
                          : (_selected == i ? Icons.cancel : null),
                      color: i == widget.answer ? Colors.green : Colors.red)
                  : null,
            ),
          const SizedBox(height: 6),
          Row(children: [
            if (!_checked)
              FilledButton.tonal(
                  onPressed: _selected == null
                      ? null
                      : () => setState(() => _checked = true),
                  child: const Text('确认答案')),
            if (_checked) ...[
              if (correct)
                const Row(children: [
                  Icon(Icons.check_circle, color: Colors.green, size: 18),
                  SizedBox(width: 4),
                  Text('回答正确', style: TextStyle(color: Colors.green))
                ])
              else
                const Row(children: [
                  Icon(Icons.cancel, color: Colors.red, size: 18),
                  SizedBox(width: 4),
                  Text('再想想正确答案', style: TextStyle(color: Colors.red))
                ]),
              if (wrong) ...[
                const SizedBox(width: 8),
                TextButton(
                    onPressed: () => setState(() {
                          _checked = false;
                          _selected = null;
                        }),
                    child: const Text('重试')),
              ],
            ],
          ]),
        ]),
      ),
    );
  }
}
