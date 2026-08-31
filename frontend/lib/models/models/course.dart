// 由 models.dart 拆分（P4-c）；part 文件不可被直接 import，请使用 models.dart barrel
part of wxx_models;

class CourseNode {
  final String id;
  final String name;
  final int credits;
  final int semester;
  final String status;
  final List<String> prerequisites;
  final String category;
  final double mastery;

  CourseNode(
      {this.id = '',
      this.name = '',
      this.credits = 0,
      this.semester = 1,
      this.status = 'pending',
      this.prerequisites = const [],
      this.category = '',
      this.mastery = 0});

  factory CourseNode.fromJson(Map<String, dynamic> json) {
    return CourseNode(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      credits: (json['credits'] ?? 0).toDouble().toInt(),
      semester: json['semester'] ?? 1,
      status: json['status'] ?? 'pending',
      prerequisites: List<String>.from(json['prerequisites'] ?? []),
      category: json['category'] ?? '',
      mastery: (json['mastery'] ?? 0).toDouble(),
    );
  }

  String get statusLabel =>
      {
        'completed': '已修',
        'current': '在修',
        'pending': '待修',
        'elective': '可选'
      }[status] ??
      '待修';
}

/// 课程学情看板

class CourseAnalyticsData {
  final String courseName;
  final double score;
  final double gpa;
  final String gradeLevel;
  final bool passed;
  final double credits;
  final String semester;
  final double progress;
  final int rankPercentile;
  final List<KnowledgePoint> knowledgePoints;
  final List<String> weakPoints;

  CourseAnalyticsData(
      {this.courseName = '',
      this.score = 0,
      this.gpa = 0,
      this.gradeLevel = '',
      this.passed = false,
      this.credits = 0,
      this.semester = '',
      this.progress = 0,
      this.rankPercentile = 50,
      this.knowledgePoints = const [],
      this.weakPoints = const []});

  factory CourseAnalyticsData.fromJson(Map<String, dynamic> json) {
    final score = (json['score'] ?? 0).toDouble();
    return CourseAnalyticsData(
      courseName: json['course_name'] ?? '',
      score: score,
      gpa: (json['gpa'] ?? 0).toDouble(),
      gradeLevel: json['grade_level'] ?? '',
      passed: json['passed'] ?? false,
      credits: (json['credits'] ?? 0).toDouble(),
      semester: (json['semester'] ?? '').toString(),
      progress:
          (json['progress'] ?? (score > 0 ? score / 100.0 : 0)).toDouble(),
      rankPercentile: json['rank_percentile'] ?? 50,
      knowledgePoints: (json['knowledge_points'] as List?)
              ?.map((e) => KnowledgePoint.fromJson(e))
              .toList() ??
          [],
      weakPoints: List<String>.from(json['weak_points'] ?? []),
    );
  }
}

class KnowledgePoint {
  final String name;
  final double mastery;

  KnowledgePoint({this.name = '', this.mastery = 0});

  factory KnowledgePoint.fromJson(Map<String, dynamic> json) {
    return KnowledgePoint(
        name: json['name'] ?? '', mastery: (json['mastery'] ?? 0).toDouble());
  }

  String get level => mastery >= 0.8
      ? 'good'
      : mastery >= 0.5
          ? 'medium'
          : 'weak';
}

// ═══════════════════════════════════════════════════════════════
// 辅导员 AI 功能模型
// ═══════════════════════════════════════════════════════════════

/// AI 今日关注

class DailyFocusData {
  final String date;
  final double classHealthScore;
  final List<FocusStudent> topStudents;
  final Map<String, int> overview;
  final String aiNarrative;
  final String dataSource;

  DailyFocusData({
    this.date = '',
    this.classHealthScore = 0,
    this.topStudents = const [],
    this.overview = const {},
    this.aiNarrative = '',
    this.dataSource = '',
  });

  factory DailyFocusData.fromJson(Map<String, dynamic> json) {
    return DailyFocusData(
      date: json['date'] ?? '',
      classHealthScore: (json['class_health_score'] ?? 0).toDouble(),
      topStudents: (json['top_students'] as List?)
              ?.map((e) => FocusStudent.fromJson(e))
              .toList() ??
          [],
      overview: Map<String, int>.from(json['overview'] ?? {}),
      aiNarrative: json['ai_narrative'] ?? '',
      dataSource: json['data_source'] ?? '',
    );
  }
}

class FocusStudent {
  final String name;
  final String reason;
  final String riskLevel;
  final String suggestion;

  FocusStudent(
      {this.name = '',
      this.reason = '',
      this.riskLevel = 'low',
      this.suggestion = ''});

  factory FocusStudent.fromJson(Map<String, dynamic> json) {
    return FocusStudent(
      name: json['name'] ?? '',
      reason: json['reason'] ?? '',
      riskLevel: json['risk_level'] ?? 'low',
      suggestion: json['suggestion'] ?? '',
    );
  }
}

/// 班级学情日报

class ClassReportData {
  final String date;
  final String className;
  final double activeRate;
  final int absentCount;
  final double homeworkRate;
  final int emotionAlertCount;
  final double checkinRate;
  final List<String> anomalies;
  final String aiNarrative;

  ClassReportData({
    this.date = '',
    this.className = '',
    this.activeRate = 0,
    this.absentCount = 0,
    this.homeworkRate = 0,
    this.emotionAlertCount = 0,
    this.checkinRate = 0,
    this.anomalies = const [],
    this.aiNarrative = '',
  });

  factory ClassReportData.fromJson(Map<String, dynamic> json) {
    return ClassReportData(
      date: json['date'] ?? '',
      className: json['class_name'] ?? '',
      activeRate: (json['active_rate'] ?? 0).toDouble(),
      absentCount: json['absent_count'] ?? 0,
      homeworkRate: (json['homework_rate'] ?? 0).toDouble(),
      emotionAlertCount: json['emotion_alert_count'] ?? 0,
      checkinRate: (json['checkin_rate'] ?? 0).toDouble(),
      anomalies: List<String>.from(json['anomalies'] ?? []),
      aiNarrative: json['ai_narrative'] ?? '',
    );
  }
}

/// 谈心谈话记录

class TalkRecord {
  final String id;
  final String studentName;
  final String date;
  final String topic;
  final String emotion;
  final String summary;
  final List<String> followUps;
  final String status;

  TalkRecord(
      {this.id = '',
      this.studentName = '',
      this.date = '',
      this.topic = '',
      this.emotion = '',
      this.summary = '',
      this.followUps = const [],
      this.status = 'pending'});

  factory TalkRecord.fromJson(Map<String, dynamic> json) {
    return TalkRecord(
      id: json['id'] ?? '',
      studentName: json['student_name'] ?? '',
      date: json['created_at'] ?? json['date'] ?? '',
      topic: json['topic'] ?? '',
      emotion: json['emotion'] ?? '',
      summary: json['summary'] ?? '',
      followUps: List<String>.from(json['follow_ups'] ?? []),
      status: json['status'] ?? 'pending',
    );
  }
}

// ═══════════════════════════════════════════════════════════════
// 教师 AI 功能模型
// ═══════════════════════════════════════════════════════════════

/// AI 备课助手输出

class LessonPlan {
  final String topic;
  final String outline;
  final List<String> keyPoints;
  final List<String> difficulties;
  final List<String> strategies;
  final List<String> interactions;
  final List<String> homework;

  LessonPlan(
      {this.topic = '',
      this.outline = '',
      this.keyPoints = const [],
      this.difficulties = const [],
      this.strategies = const [],
      this.interactions = const [],
      this.homework = const []});

  factory LessonPlan.fromJson(Map<String, dynamic> json) {
    return LessonPlan(
      topic: json['topic'] ?? '',
      outline: json['outline'] ?? '',
      keyPoints: List<String>.from(json['key_points'] ?? []),
      difficulties: List<String>.from(json['difficulties'] ?? []),
      strategies: List<String>.from(json['strategies'] ?? []),
      interactions: List<String>.from(json['interactions'] ?? []),
      homework: List<String>.from(json['homework'] ?? []),
    );
  }
}

/// 班级学情热力图数据

class ClassHeatmapData {
  final String courseName;
  final List<KnowledgePoint> points;
  final List<String> weakTopFive;
  final int totalStudents;
  final int anomalyCount;

  ClassHeatmapData(
      {this.courseName = '',
      this.points = const [],
      this.weakTopFive = const [],
      this.totalStudents = 0,
      this.anomalyCount = 0});

  factory ClassHeatmapData.fromJson(Map<String, dynamic> json) {
    return ClassHeatmapData(
      courseName: json['course_name'] ?? '',
      points: (json['points'] as List?)
              ?.map((e) => KnowledgePoint.fromJson(e))
              .toList() ??
          [],
      weakTopFive: List<String>.from(json['weak_top_five'] ?? []),
      totalStudents: json['total_students'] ?? 0,
      anomalyCount: json['anomaly_count'] ?? 0,
    );
  }
}

// ═══════════════════════════════════════════════════════════════
// AI 模型配置模型
// ═══════════════════════════════════════════════════════════════

/// 用户 AI 模型配置（对齐后端 model.UserModelConfig）

