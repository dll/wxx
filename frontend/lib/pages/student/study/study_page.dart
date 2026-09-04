import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import '../../../providers/education_provider.dart';
import '../../../widgets/error_view.dart';

class StudyPage extends StatefulWidget {
  const StudyPage({super.key});

  @override
  State<StudyPage> createState() => _StudyPageState();
}

class _StudyPageState extends State<StudyPage> with SingleTickerProviderStateMixin {
  late TabController _tabController;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 4, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudyProvider>().fetchCourses();
    });
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudyProvider>();

    return Scaffold(
      appBar: AppBar(
        title: const Text('学业服务'),
        bottom: TabBar(
          controller: _tabController,
          tabs: const [
            Tab(text: '课程'),
            Tab(text: '成绩'),
            Tab(text: '资源'),
            Tab(text: '考试'),
          ],
          onTap: (index) {
            switch (index) {
              case 0:
                provider.fetchCourses();
                break;
              case 1:
                provider.fetchGradesSummary();
                provider.fetchGrades();
                break;
              case 2:
                provider.fetchResources();
                break;
              case 3:
                provider.fetchExams();
                break;
            }
          },
        ),
      ),
      body: TabBarView(
        controller: _tabController,
        children: [
          _buildCoursesTab(theme, provider),
          _buildGradesTab(theme, provider),
          _buildResourcesTab(theme, provider),
          _buildExamsTab(theme, provider),
        ],
      ),
    );
  }

  Widget _buildCoursesTab(ThemeData theme, StudyProvider provider) {
    if (provider.loading && provider.courses.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    if (provider.error.isNotEmpty && provider.courses.isEmpty) {
      return ErrorView.error(
        message: provider.error,
        onRetry: () => provider.fetchCourses(),
      );
    }
    if (provider.courses.isEmpty) {
      return ErrorView.empty(
        message: '暂无课程',
        subtitle: '稍后再来看看吧',
        icon: Icons.menu_book_outlined,
      );
    }
    return RefreshIndicator(
      onRefresh: () => provider.fetchCourses(),
      child: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: provider.courses.length,
        itemBuilder: (_, i) {
          final course = provider.courses[i] as Map<String, dynamic>;
          return _buildCourseCard(theme, course);
        },
      ),
    );
  }

  Widget _buildGradesTab(ThemeData theme, StudyProvider provider) {
    return RefreshIndicator(
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
          const SizedBox(height: 16),
          Text('成绩列表', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
          const SizedBox(height: 12),
          if (provider.loading && provider.grades.isEmpty)
            const Center(child: Padding(padding: EdgeInsets.all(32), child: CircularProgressIndicator()))
          else if (provider.grades.isEmpty)
            ErrorView.empty(
              message: '暂无成绩',
              icon: Icons.grade_outlined,
            )
          else
            ...provider.grades.map((g) {
              final grade = g as Map<String, dynamic>;
              return _buildGradeCard(theme, grade);
            }),
        ],
      ),
    );
  }

  Widget _buildGPAWidget(ThemeData theme, StudyProvider provider) {
    final summary = provider.gradesSummary;
    final gpa = summary?['gpa'] as num? ?? 0.0;
    final rank = summary?['rank'] as int? ?? 0;
    final total = summary?['total'] as int? ?? 0;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          children: [
            Row(
              children: [
                Expanded(
                  child: Column(
                    children: [
                      Text(
                        gpa.toStringAsFixed(2),
                        style: TextStyle(
                          fontSize: 36,
                          fontWeight: FontWeight.bold,
                          color: theme.colorScheme.primary,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text('GPA', style: theme.textTheme.bodySmall),
                    ],
                  ),
                ),
                Container(
                  width: 1,
                  height: 50,
                  color: theme.colorScheme.outlineVariant,
                ),
                Expanded(
                  child: Column(
                    children: [
                      Text(
                        rank > 0 && total > 0 ? '$rank / $total' : '-',
                        style: TextStyle(
                          fontSize: 28,
                          fontWeight: FontWeight.bold,
                          color: theme.colorScheme.secondary,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text('专业排名', style: theme.textTheme.bodySmall),
                    ],
                  ),
                ),
                Container(
                  width: 1,
                  height: 50,
                  color: theme.colorScheme.outlineVariant,
                ),
                Expanded(
                  child: Column(
                    children: [
                      Text(
                        provider.grades.length.toString(),
                        style: TextStyle(
                          fontSize: 28,
                          fontWeight: FontWeight.bold,
                          color: theme.colorScheme.tertiary,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text('课程门数', style: theme.textTheme.bodySmall),
                    ],
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildResourcesTab(ThemeData theme, StudyProvider provider) {
    if (provider.loading && provider.resources.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    if (provider.error.isNotEmpty && provider.resources.isEmpty) {
      return ErrorView.error(
        message: provider.error,
        onRetry: () => provider.fetchResources(),
      );
    }
    if (provider.resources.isEmpty) {
      return ErrorView.empty(
        message: '暂无学习资源',
        subtitle: '稍后再来看看吧',
        icon: Icons.folder_outlined,
      );
    }
    return RefreshIndicator(
      onRefresh: () => provider.fetchResources(),
      child: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: provider.resources.length,
        itemBuilder: (_, i) {
          final resource = provider.resources[i] as Map<String, dynamic>;
          return _buildResourceCard(theme, resource);
        },
      ),
    );
  }

  Widget _buildExamsTab(ThemeData theme, StudyProvider provider) {
    if (provider.loading && provider.exams.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    if (provider.error.isNotEmpty && provider.exams.isEmpty) {
      return ErrorView.error(
        message: provider.error,
        onRetry: () => provider.fetchExams(),
      );
    }
    if (provider.exams.isEmpty) {
      return ErrorView.empty(
        message: '暂无考试安排',
        subtitle: '稍后再来看看吧',
        icon: Icons.event_note_outlined,
      );
    }
    return RefreshIndicator(
      onRefresh: () => provider.fetchExams(),
      child: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: provider.exams.length,
        itemBuilder: (_, i) {
          final exam = provider.exams[i] as Map<String, dynamic>;
          return _buildExamCard(theme, exam);
        },
      ),
    );
  }

  Widget _buildCourseCard(ThemeData theme, Map<String, dynamic> course) {
    final id = course['id']?.toString() ?? '';
    final name = course['name'] as String? ?? course['title'] as String? ?? '课程';
    final teacher = course['teacher'] as String? ?? course['instructor'] as String? ?? '';
    final credit = course['credit'] as num? ?? 0;
    final schedule = course['schedule'] as String? ?? '';

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: InkWell(
        onTap: () {
          context.go('/student/study/course/$id');
        },
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Text(
                      name,
                      style: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                    decoration: BoxDecoration(
                      color: theme.colorScheme.primaryContainer,
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: Text(
                      '$credit学分',
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: theme.colorScheme.onPrimaryContainer,
                      ),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              Row(
                children: [
                  Icon(Icons.person_outline, size: 16, color: theme.colorScheme.onSurfaceVariant),
                  const SizedBox(width: 4),
                  Text(teacher, style: theme.textTheme.bodySmall),
                ],
              ),
              if (schedule.isNotEmpty) ...[
                const SizedBox(height: 4),
                Row(
                  children: [
                    Icon(Icons.access_time, size: 16, color: theme.colorScheme.onSurfaceVariant),
                    const SizedBox(width: 4),
                    Text(schedule, style: theme.textTheme.bodySmall),
                  ],
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildGradeCard(ThemeData theme, Map<String, dynamic> grade) {
    final courseName = grade['course_name'] as String? ?? grade['name'] as String? ?? '课程';
    final score = grade['score'] as num? ?? grade['grade'] as num? ?? 0;
    final credit = grade['credit'] as num? ?? 0;
    final semester = grade['semester'] as String? ?? '';

    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    courseName,
                    style: theme.textTheme.titleSmall?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 4),
                  if (semester.isNotEmpty)
                    Text(semester, style: theme.textTheme.bodySmall),
                ],
              ),
            ),
            Column(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Text(
                  score.toString(),
                  style: TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.bold,
                    color: score >= 90
                        ? theme.colorScheme.primary
                        : score >= 60
                            ? theme.colorScheme.secondary
                            : theme.colorScheme.error,
                  ),
                ),
                Text('$credit学分', style: theme.textTheme.bodySmall),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildResourceCard(ThemeData theme, Map<String, dynamic> resource) {
    final id = resource['id']?.toString() ?? '';
    final title = resource['title'] as String? ?? resource['name'] as String? ?? '资源';
    final type = resource['type'] as String? ?? '';
    final size = resource['size'] as String? ?? '';
    final downloads = resource['downloads'] as int? ?? 0;

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: InkWell(
        onTap: () {
          context.go('/student/study/resource/$id');
        },
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: theme.colorScheme.primary.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Icon(
                  _getResourceIcon(type),
                  color: theme.colorScheme.primary,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: theme.textTheme.titleSmall?.copyWith(
                        fontWeight: FontWeight.w600,
                      ),
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 4),
                    Row(
                      children: [
                        if (type.isNotEmpty) ...[
                          Text(type, style: theme.textTheme.bodySmall),
                          const SizedBox(width: 12),
                        ],
                        if (size.isNotEmpty) ...[
                          Text(size, style: theme.textTheme.bodySmall),
                          const SizedBox(width: 12),
                        ],
                        Icon(Icons.download_outlined, size: 14, color: theme.colorScheme.onSurfaceVariant),
                        const SizedBox(width: 2),
                        Text('$downloads', style: theme.textTheme.bodySmall),
                      ],
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  IconData _getResourceIcon(String type) {
    switch (type.toLowerCase()) {
      case 'pdf':
        return Icons.picture_as_pdf;
      case 'doc':
      case 'docx':
      case '文档':
        return Icons.description;
      case 'ppt':
      case 'pptx':
      case '课件':
        return Icons.slideshow;
      case '视频':
      case 'video':
        return Icons.play_circle_outline;
      default:
        return Icons.folder_outlined;
    }
  }

  Widget _buildExamCard(ThemeData theme, Map<String, dynamic> exam) {
    final courseName = exam['course_name'] as String? ?? exam['name'] as String? ?? '考试';
    final time = exam['time'] as String? ?? exam['date'] as String? ?? '';
    final location = exam['location'] as String? ?? exam['venue'] as String? ?? '';
    final type = exam['type'] as String? ?? '';

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    courseName,
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                if (type.isNotEmpty)
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                    decoration: BoxDecoration(
                      color: theme.colorScheme.error.withOpacity(0.1),
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: Text(
                      type,
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: theme.colorScheme.error,
                      ),
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 10),
            Row(
              children: [
                Icon(Icons.access_time, size: 16, color: theme.colorScheme.onSurfaceVariant),
                const SizedBox(width: 6),
                Expanded(child: Text(time, style: theme.textTheme.bodySmall)),
              ],
            ),
            const SizedBox(height: 6),
            Row(
              children: [
                Icon(Icons.location_on_outlined, size: 16, color: theme.colorScheme.onSurfaceVariant),
                const SizedBox(width: 6),
                Expanded(child: Text(location, style: theme.textTheme.bodySmall)),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
