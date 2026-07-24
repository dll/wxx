import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../../providers/education_provider.dart';
import '../../../widgets/error_view.dart';

class ScaleDetailPage extends StatefulWidget {
  final String scaleId;
  const ScaleDetailPage({super.key, required this.scaleId});

  @override
  State<ScaleDetailPage> createState() => _ScaleDetailPageState();
}

class _ScaleDetailPageState extends State<ScaleDetailPage> {
  int _currentQuestion = 0;
  final Map<int, int> _answers = {};
  bool _showIntro = true;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<MentalProvider>().fetchScaleDetail(widget.scaleId);
    });
  }

  @override
  void dispose() {
    context.read<MentalProvider>().clearDetail();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<MentalProvider>();
    final scale = provider.scaleDetail;

    return Scaffold(
      appBar: AppBar(title: const Text('心理测评')),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty && scale == null
              ? ErrorView.error(
                  message: provider.error,
                  onRetry: () => provider.fetchScaleDetail(widget.scaleId),
                )
              : scale == null
                  ? ErrorView.empty(
                      message: '测评不存在',
                      icon: Icons.psychology_outlined,
                    )
                  : _showIntro
                      ? _buildIntro(theme, scale)
                      : _buildQuestions(theme, scale),
    );
  }

  Widget _buildIntro(ThemeData theme, Map<String, dynamic> scale) {
    final title = scale['title'] as String? ?? scale['name'] as String? ?? '';
    final description = scale['description'] as String? ?? scale['intro'] as String? ?? '';
    final questionCount = scale['question_count'] as int? ?? ((scale['questions'] as List?)?.length ?? 0);
    final duration = scale['duration'] as String? ?? '';

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    title,
                    style: theme.textTheme.titleLarge?.copyWith(
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      Icon(Icons.help_outline, size: 16, color: theme.colorScheme.onSurfaceVariant),
                      const SizedBox(width: 4),
                      Text('$questionCount题', style: theme.textTheme.bodySmall),
                      if (duration.isNotEmpty) ...[
                        const SizedBox(width: 16),
                        Icon(Icons.timer_outlined, size: 16, color: theme.colorScheme.onSurfaceVariant),
                        const SizedBox(width: 4),
                        Text(duration, style: theme.textTheme.bodySmall),
                      ],
                    ],
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          if (description.isNotEmpty) ...[
            Text('测评说明', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
            const SizedBox(height: 12),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Text(description, style: theme.textTheme.bodyMedium),
              ),
            ),
            const SizedBox(height: 24),
          ],
          SizedBox(
            width: double.infinity,
            child: FilledButton.icon(
              onPressed: () {
                setState(() {
                  _showIntro = false;
                });
              },
              icon: const Icon(Icons.play_arrow, size: 18),
              label: const Text('开始测评'),
              style: FilledButton.styleFrom(
                padding: const EdgeInsets.symmetric(vertical: 14),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildQuestions(ThemeData theme, Map<String, dynamic> scale) {
    final questions = (scale['questions'] as List?) ?? [];
    if (questions.isEmpty) {
      return ErrorView.empty(
        message: '暂无题目',
        icon: Icons.help_outline,
      );
    }

    final currentQ = questions[_currentQuestion] as Map<String, dynamic>;
    final options = (currentQ['options'] as List?) ?? [];

    return Column(
      children: [
        LinearProgressIndicator(
          value: (_currentQuestion + 1) / questions.length,
          minHeight: 4,
        ),
        Expanded(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  '第 ${_currentQuestion + 1} / ${questions.length} 题',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 12),
                Text(
                  currentQ['title'] as String? ?? currentQ['question'] as String? ?? '',
                  style: theme.textTheme.titleLarge?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 24),
                ...options.asMap().entries.map((entry) {
                  final idx = entry.key;
                  final opt = entry.value as Map<String, dynamic>;
                  final isSelected = _answers[_currentQuestion] == idx;
                  return Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: Material(
                      color: isSelected
                          ? theme.colorScheme.primaryContainer
                          : theme.colorScheme.surfaceContainerHighest.withOpacity(0.5),
                      borderRadius: BorderRadius.circular(12),
                      child: InkWell(
                        onTap: () {
                          setState(() {
                            _answers[_currentQuestion] = idx;
                          });
                        },
                        borderRadius: BorderRadius.circular(12),
                        child: Container(
                          width: double.infinity,
                          padding: const EdgeInsets.all(16),
                          child: Row(
                            children: [
                              Container(
                                width: 24,
                                height: 24,
                                decoration: BoxDecoration(
                                  shape: BoxShape.circle,
                                  color: isSelected
                                      ? theme.colorScheme.primary
                                      : Colors.transparent,
                                  border: Border.all(
                                    color: isSelected
                                        ? theme.colorScheme.primary
                                        : theme.colorScheme.outline,
                                    width: 2,
                                  ),
                                ),
                                child: isSelected
                                    ? const Icon(Icons.check, size: 16, color: Colors.white)
                                    : null,
                              ),
                              const SizedBox(width: 12),
                              Expanded(
                                child: Text(
                                  opt['text'] as String? ?? opt['content'] as String? ?? '',
                                  style: theme.textTheme.bodyLarge,
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ),
                  );
                }),
              ],
            ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              if (_currentQuestion > 0)
                Expanded(
                  child: OutlinedButton(
                    onPressed: () {
                      setState(() {
                        _currentQuestion--;
                      });
                    },
                    child: const Text('上一题'),
                  ),
                )
              else
                const Spacer(),
              const SizedBox(width: 12),
              Expanded(
                child: FilledButton(
                  onPressed: _answers.containsKey(_currentQuestion)
                      ? () {
                          if (_currentQuestion < questions.length - 1) {
                            setState(() {
                              _currentQuestion++;
                            });
                          } else {
                            _submitAssessment();
                          }
                        }
                      : null,
                  child: Text(_currentQuestion < questions.length - 1 ? '下一题' : '提交测评'),
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Future<void> _submitAssessment() async {
    final provider = context.read<MentalProvider>();
    final success = await provider.submitAssessment({
      'scale_id': widget.scaleId,
      'answers': _answers,
    });
    if (mounted) {
      if (success) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('测评提交成功')),
        );
        Navigator.of(context).pop();
      } else {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('提交失败：${provider.error}')),
        );
      }
    }
  }
}
