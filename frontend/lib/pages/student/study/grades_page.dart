import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../../providers/education_provider.dart';
import '../../../widgets/error_view.dart';

class GradesPage extends StatefulWidget {
  const GradesPage({super.key});

  @override
  State<GradesPage> createState() => _GradesPageState();
}

class _GradesPageState extends State<GradesPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudyProvider>().fetchGradesSummary();
      context.read<StudyProvider>().fetchGrades();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudyProvider>();

    return Scaffold(
      appBar: AppBar(title: const Text('成绩详情')),
      body: RefreshIndicator(
        onRefresh: () async {
          await Future.wait([
            provider.fetchGradesSummary(),
            provider.fetchGrades(),
          ]);
        },
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            _buildGPAWidget(theme, provider),
            const SizedBox(height: 20),
            Text('成绩单', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
            const SizedBox(height: 12),
            if (provider.loading && provider.grades.isEmpty)
              const Center(child: Padding(padding: EdgeInsets.all(32), child: CircularProgressIndicator()))
            else if (provider.error.isNotEmpty && provider.grades.isEmpty)
              ErrorView.error(
                message: provider.error,
                onRetry: () {
                  provider.fetchGradesSummary();
                  provider.fetchGrades();
                },
              )
            else if (provider.grades.isEmpty)
              ErrorView.empty(
                message: '暂无成绩',
                icon: Icons.grade_outlined,
              )
            else
              ..._groupGradesBySemester(theme, provider.grades),
          ],
        ),
      ),
    );
  }

  Widget _buildGPAWidget(ThemeData theme, StudyProvider provider) {
    final summary = provider.gradesSummary;
    final gpa = summary?['gpa'] as num? ?? 0.0;
    final rank = summary?['rank'] as int? ?? 0;
    final total = summary?['total'] as int? ?? 0;
    final credits = summary?['credits'] as num? ?? 0;

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: [theme.colorScheme.primary, theme.colorScheme.primary.withOpacity(0.7)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceAround,
            children: [
              Column(
                children: [
                  Text(
                    gpa.toStringAsFixed(2),
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 36,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text('GPA', style: TextStyle(color: Colors.white.withOpacity(0.85), fontSize: 13)),
                ],
              ),
              Column(
                children: [
                  Text(
                    rank > 0 && total > 0 ? '$rank / $total' : '-',
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 28,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text('专业排名', style: TextStyle(color: Colors.white.withOpacity(0.85), fontSize: 13)),
                ],
              ),
              Column(
                children: [
                  Text(
                    '$credits',
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 28,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text('已修学分', style: TextStyle(color: Colors.white.withOpacity(0.85), fontSize: 13)),
                ],
              ),
            ],
          ),
        ],
      ),
    );
  }

  List<Widget> _groupGradesBySemester(ThemeData theme, List<dynamic> grades) {
    final Map<String, List<dynamic>> grouped = {};
    for (final g in grades) {
      if (g is! Map) continue; // 跳过非法元素，避免单条异常拖垮整页
      final grade = Map<String, dynamic>.from(g);
      final semester = grade['semester'] as String? ?? '其他';
      if (!grouped.containsKey(semester)) {
        grouped[semester] = [];
      }
      grouped[semester]!.add(grade);
    }

    final List<Widget> widgets = [];
    grouped.forEach((semester, semesterGrades) {
      widgets.addAll([
        Padding(
          padding: const EdgeInsets.only(bottom: 8),
          child: Text(
            semester,
            style: theme.textTheme.titleSmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
        Card(
          margin: const EdgeInsets.only(bottom: 16),
          child: Column(
            children: semesterGrades.map((g) {
              if (g is! Map) return const SizedBox.shrink();
              final grade = Map<String, dynamic>.from(g);
              final courseName = grade['course_name'] as String? ?? grade['name'] as String? ?? '';
              final score = grade['score'] as num? ?? grade['grade'] as num? ?? 0;
              final credit = grade['credit'] as num? ?? 0;
              return Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        courseName,
                        style: theme.textTheme.bodyMedium,
                      ),
                    ),
                    Text(
                      '$credit学分',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                    const SizedBox(width: 16),
                    Text(
                      score.toString(),
                      style: TextStyle(
                        fontSize: 18,
                        fontWeight: FontWeight.bold,
                        color: score >= 90
                            ? theme.colorScheme.primary
                            : score >= 60
                                ? theme.colorScheme.secondary
                                : theme.colorScheme.error,
                      ),
                    ),
                  ],
                ),
              );
            }).toList(),
          ),
        ),
      ]);
    });

    return widgets;
  }
}
