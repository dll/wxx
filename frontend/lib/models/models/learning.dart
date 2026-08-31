// 由 models.dart 拆分（P4-c）；part 文件不可被直接 import，请使用 models.dart barrel
part of wxx_models;

class DailyBriefing {
  final String date;
  final String greeting;
  final List<BriefingItem> courses;
  final List<BriefingItem> deadlines;
  final List<BriefingItem> activities;
  final String weather;
  final String motto;

  DailyBriefing({
    required this.date,
    this.greeting = '',
    this.courses = const [],
    this.deadlines = const [],
    this.activities = const [],
    this.weather = '',
    this.motto = '',
  });

  factory DailyBriefing.fromJson(Map<String, dynamic> json) {
    return DailyBriefing(
      date: json['date'] ?? '',
      greeting: json['greeting'] ?? '',
      courses: (json['courses'] as List?)
              ?.map((e) => BriefingItem.fromJson(e))
              .toList() ??
          [],
      deadlines: (json['deadlines'] as List?)
              ?.map((e) => BriefingItem.fromJson(e))
              .toList() ??
          [],
      activities: (json['activities'] as List?)
              ?.map((e) => BriefingItem.fromJson(e))
              .toList() ??
          [],
      weather: json['weather'] ?? '',
      motto: json['motto'] ?? '',
    );
  }
}

class BriefingItem {
  final String title;
  final String subtitle;
  final String time;
  final String icon;

  BriefingItem(
      {this.title = '', this.subtitle = '', this.time = '', this.icon = ''});

  factory BriefingItem.fromJson(Map<String, dynamic> json) {
    return BriefingItem(
      title: json['title'] ?? '',
      subtitle: json['subtitle'] ?? '',
      time: json['time'] ?? '',
      icon: json['icon'] ?? '',
    );
  }
}

/// AI 学习日记

class LearningDiary {
  final String date;
  final List<String> coursesStudied;
  final List<String> keyPoints;
  final int studyMinutes;
  final List<QuizItem> quiz;
  final String tomorrowPlan;
  final String encouragement;

  LearningDiary({
    required this.date,
    this.coursesStudied = const [],
    this.keyPoints = const [],
    this.studyMinutes = 0,
    this.quiz = const [],
    this.tomorrowPlan = '',
    this.encouragement = '',
  });

  factory LearningDiary.fromJson(Map<String, dynamic> json) {
    return LearningDiary(
      date: json['date'] ?? '',
      coursesStudied: List<String>.from(json['courses_studied'] ?? []),
      keyPoints: List<String>.from(json['key_points'] ?? []),
      studyMinutes: json['study_minutes'] ?? 0,
      quiz:
          (json['quiz'] as List?)?.map((e) => QuizItem.fromJson(e)).toList() ??
              [],
      tomorrowPlan: json['tomorrow_plan'] ?? '',
      encouragement: json['encouragement'] ?? '',
    );
  }
}

class QuizItem {
  final String question;
  final List<String> options;
  final int correctIndex;
  final String explanation;

  QuizItem(
      {this.question = '',
      this.options = const [],
      this.correctIndex = 0,
      this.explanation = ''});

  factory QuizItem.fromJson(Map<String, dynamic> json) {
    return QuizItem(
      question: json['question'] ?? '',
      options: List<String>.from(json['options'] ?? []),
      correctIndex: json['correct_index'] ?? 0,
      explanation: json['explanation'] ?? '',
    );
  }
}

/// 个人数字孪生

class CheckinRecord {
  final String date;
  final int streak;
  final int totalDays;
  final int longestStreak;
  final bool todayChecked;
  final int makeupLeft;
  final List<String> recentDates;

  CheckinRecord(
      {this.date = '',
      this.streak = 0,
      this.totalDays = 0,
      this.longestStreak = 0,
      this.todayChecked = false,
      this.makeupLeft = 2,
      this.recentDates = const []});

  factory CheckinRecord.fromJson(Map<String, dynamic> json) {
    return CheckinRecord(
      date: json['date'] ?? '',
      streak: json['streak'] ?? 0,
      totalDays: json['total_days'] ?? 0,
      longestStreak: json['longest_streak'] ?? 0,
      todayChecked: json['today_checked'] ?? false,
      makeupLeft: json['makeup_left'] ?? 2,
      recentDates: List<String>.from(json['recent_dates'] ?? []),
    );
  }
}

/// 学习积分与成就

class AchievementData {
  final int totalPoints;
  final int level;
  final String levelName;
  final int nextLevelPoints;
  final List<Achievement> badges;
  final int weeklyRank;

  AchievementData(
      {this.totalPoints = 0,
      this.level = 1,
      this.levelName = '青铜',
      this.nextLevelPoints = 100,
      this.badges = const [],
      this.weeklyRank = 0});

  factory AchievementData.fromJson(Map<String, dynamic> json) {
    return AchievementData(
      totalPoints: json['total_points'] ?? 0,
      level: json['level'] ?? 1,
      levelName: json['level_name'] ?? '青铜',
      nextLevelPoints: json['next_level_points'] ?? 100,
      badges: (json['badges'] as List?)
              ?.map((e) => Achievement.fromJson(e))
              .toList() ??
          [],
      weeklyRank: json['weekly_rank'] ?? 0,
    );
  }
}

class Achievement {
  final String id;
  final String name;
  final String icon;
  final String description;
  final bool unlocked;
  final String unlockedAt;

  Achievement(
      {this.id = '',
      this.name = '',
      this.icon = '',
      this.description = '',
      this.unlocked = false,
      this.unlockedAt = ''});

  factory Achievement.fromJson(Map<String, dynamic> json) {
    return Achievement(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      icon: json['icon'] ?? '',
      description: json['description'] ?? '',
      unlocked: json['unlocked'] ?? false,
      unlockedAt: json['unlocked_at'] ?? '',
    );
  }
}

/// 课程地图节点

