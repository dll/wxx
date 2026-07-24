import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../../providers/education_provider.dart';
import '../../../widgets/error_view.dart';

class MoodDiaryPage extends StatefulWidget {
  const MoodDiaryPage({super.key});

  @override
  State<MoodDiaryPage> createState() => _MoodDiaryPageState();
}

class _MoodDiaryPageState extends State<MoodDiaryPage> {
  final _formKey = GlobalKey<FormState>();
  int _selectedMood = 2;
  final TextEditingController _noteController = TextEditingController();

  final List<Map<String, dynamic>> _moodOptions = [
    {'mood': '开心', 'emoji': '😊', 'color': Colors.green, 'score': 5},
    {'mood': '平静', 'emoji': '😌', 'color': Colors.blue, 'score': 4},
    {'mood': '疲惫', 'emoji': '😫', 'color': Colors.grey, 'score': 3},
    {'mood': '焦虑', 'emoji': '😰', 'color': Colors.orange, 'score': 2},
    {'mood': '难过', 'emoji': '😢', 'color': Colors.blueGrey, 'score': 1},
    {'mood': '愤怒', 'emoji': '😤', 'color': Colors.red, 'score': 1},
  ];

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<MentalProvider>().fetchMoodRecords();
    });
  }

  @override
  void dispose() {
    _noteController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<MentalProvider>();

    return Scaffold(
      appBar: AppBar(title: const Text('情绪日记')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Card(
              child: Padding(
                padding: const EdgeInsets.all(20),
                child: Column(
                  children: [
                    Text('今天的心情怎么样？', style: theme.textTheme.titleLarge?.copyWith(fontWeight: FontWeight.bold)),
                    const SizedBox(height: 20),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceAround,
                      children: _moodOptions.asMap().entries.map((entry) {
                        final idx = entry.key;
                        final mood = entry.value;
                        final isSelected = _selectedMood == idx;
                        return GestureDetector(
                          onTap: () {
                            setState(() {
                              _selectedMood = idx;
                            });
                          },
                          child: AnimatedContainer(
                            duration: const Duration(milliseconds: 200),
                            padding: const EdgeInsets.all(8),
                            decoration: BoxDecoration(
                              color: isSelected ? mood['color']?.withOpacity(0.2) : Colors.transparent,
                              shape: BoxShape.circle,
                              border: Border.all(
                                color: isSelected ? mood['color'] : Colors.transparent,
                                width: 2,
                              ),
                            ),
                            child: Text(mood['emoji'], style: const TextStyle(fontSize: 36)),
                          ),
                        );
                      }).toList(),
                    ),
                    const SizedBox(height: 12),
                    Text(
                      _moodOptions[_selectedMood]['mood'],
                      style: theme.textTheme.titleMedium?.copyWith(
                        color: _moodOptions[_selectedMood]['color'],
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 20),
            Text('心情记录', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
            const SizedBox(height: 12),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Form(
                  key: _formKey,
                  child: Column(
                    children: [
                      TextFormField(
                        controller: _noteController,
                        maxLines: 4,
                        decoration: const InputDecoration(
                          labelText: '记录一下今天的心情吧...',
                          hintText: '今天发生了什么？有什么感受？',
                          border: OutlineInputBorder(),
                          alignLabelWithHint: true,
                        ),
                        maxLength: 500,
                      ),
                      const SizedBox(height: 16),
                      SizedBox(
                        width: double.infinity,
                        child: FilledButton.icon(
                          onPressed: provider.loading ? null : _submitMood,
                          icon: const Icon(Icons.save_outlined, size: 18),
                          label: Text(provider.loading ? '保存中...' : '保存心情'),
                          style: FilledButton.styleFrom(
                            padding: const EdgeInsets.symmetric(vertical: 14),
                            backgroundColor: _moodOptions[_selectedMood]['color'],
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
            const SizedBox(height: 24),
            Text('历史记录', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
            const SizedBox(height: 12),
            if (provider.loading && provider.moodRecords.isEmpty)
              const Center(child: Padding(padding: EdgeInsets.all(32), child: CircularProgressIndicator()))
            else if (provider.moodRecords.isEmpty)
              ErrorView.empty(
                message: '暂无情绪记录',
                subtitle: '记录今天的心情，开始你的情绪日记吧',
                icon: Icons.mood_outlined,
              )
            else
              ...provider.moodRecords.map((m) {
                final mood = m as Map<String, dynamic>;
                return _buildMoodRecordCard(theme, mood);
              }),
          ],
        ),
      ),
    );
  }

  Widget _buildMoodRecordCard(ThemeData theme, Map<String, dynamic> mood) {
    final date = mood['date'] as String? ?? mood['created_at'] as String? ?? '';
    final moodType = mood['mood'] as String? ?? mood['type'] as String? ?? '平静';
    final note = mood['note'] as String? ?? mood['content'] as String? ?? '';
    final score = mood['score'] as int? ?? 0;
    final emoji = _getMoodEmoji(moodType);
    final color = _getMoodColor(moodType);

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text(emoji, style: const TextStyle(fontSize: 28)),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        moodType,
                        style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w600),
                      ),
                      if (date.isNotEmpty)
                        Text(date, style: theme.textTheme.bodySmall),
                    ],
                  ),
                ),
                if (score > 0)
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                    decoration: BoxDecoration(
                      color: color.withOpacity(0.15),
                      borderRadius: BorderRadius.circular(20),
                    ),
                    child: Text(
                      '$score分',
                      style: TextStyle(
                        color: color,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  ),
              ],
            ),
            if (note.isNotEmpty) ...[
              const SizedBox(height: 10),
              Text(
                note,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Future<void> _submitMood() async {
    final provider = context.read<MentalProvider>();
    final moodData = _moodOptions[_selectedMood];
    final success = await provider.submitMoodRecord({
      'mood': moodData['mood'],
      'score': moodData['score'],
      'note': _noteController.text,
    });

    if (mounted) {
      if (success) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('心情记录已保存')),
        );
        _noteController.clear();
      } else {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('保存失败：${provider.error}')),
        );
      }
    }
  }

  String _getMoodEmoji(String mood) {
    for (final option in _moodOptions) {
      if (option['mood'] == mood) return option['emoji'];
    }
    return '🙂';
  }

  Color _getMoodColor(String mood) {
    for (final option in _moodOptions) {
      if (option['mood'] == mood) return option['color'];
    }
    return Colors.grey;
  }
}
