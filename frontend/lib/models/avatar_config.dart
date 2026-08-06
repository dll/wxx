import 'package:flutter/material.dart';

/// 数字人形象配置 — 由五维孪生分数 + 性格洞察推导的可视化参数
class AvatarConfig {
  // 五维分数（0-100）
  final double academic;
  final double ability;
  final double ideological;
  final double emotional;
  final double social;

  // 大五人格（0-100，可缺省）
  final double openness;
  final double extraversion;

  // 性格类型（MBTI），如 INTJ/ENFP
  final String personalityType;

  // 基础信息
  final String displayName;
  final String major;

  /// 用户角色（student/counselor/teacher/college_admin 等），驱动帽子/服装样式
  final String role;

  AvatarConfig({
    required this.academic,
    required this.ability,
    required this.ideological,
    required this.emotional,
    required this.social,
    this.openness = 50,
    this.extraversion = 50,
    this.personalityType = '',
    this.displayName = '同学',
    this.major = '',
    this.role = 'student',
  });

  /// 综合分（五维加权，与后端一致：学业0.3/能力0.25/思想0.15/情感0.15/社交0.15）
  double get overall =>
      academic * 0.30 + ability * 0.25 + ideological * 0.15 +
      emotional * 0.15 + social * 0.15;

  /// 是否外向（E 型人格）
  bool get isExtrovert =>
      personalityType.contains('E') || extraversion >= 60;

  /// 是否学业突出 → 戴眼镜 + 拿书本
  bool get hasGlasses => academic >= 75;
  bool get hasBook => academic >= 85;

  /// 是否能力突出 → 胸前奖牌
  bool get hasMedal => ability >= 70;

  /// 是否思想突出 → 红色徽章
  bool get hasRedBadge => ideological >= 80;

  /// 是否社交活跃 → 灿烂笑容
  bool get isSmiling => social >= 70;

  /// 情感分 → 眼睛明亮度（0.4~1.0）
  double get eyeBrightness =>
      0.4 + (emotional / 100.0) * 0.6;

  /// 开放分 → 发型风格：>60 蓬松创意，<40 利落短发，中间标准
  String get hairStyle {
    if (openness >= 60) return 'fluffy';
    if (openness <= 40) return 'short';
    return 'standard';
  }

  /// 服装主色：外向偏暖、内向偏冷，叠加综合分亮度
  Color get outfitColor {
    final warm = isExtrovert;
    final base = warm ? const Color(0xFFFF8A65) : const Color(0xFF64B5F6);
    // 综合分高 → 更鲜艳
    final t = (overall / 100.0).clamp(0.3, 1.0);
    return Color.lerp(base, warm ? const Color(0xFFFF7043) : const Color(0xFF42A5F5), t)!;
  }

  /// 滁州学院校徽蓝
  static const Color schoolBlue = Color(0xFF1565C0);

  /// 帽子样式：teacher 用中式学位帽，其余用学士帽
  String get hatStyle => role == 'teacher' ? 'chinese' : 'graduate';

  /// 学位服主色（角色区分）：教师红/管理员紫/学生蓝/其他灰
  Color get gownColor {
    switch (role) {
      case 'teacher':
        return const Color(0xFFB71C1C);
      case 'college_admin':
      case 'school_admin':
      case 'sys_admin':
        return const Color(0xFF6A1B9A);
      case 'counselor':
        return const Color(0xFF1A237E);
      default:
        return const Color(0xFF1565C0);
    }
  }

  /// 从五维孪生数据 + 性格数据构建形象配置
  factory AvatarConfig.fromData({
    Map<String, dynamic>? twinJson,
    Map<String, dynamic>? personalityJson,
    String displayName = '同学',
    String major = '',
    String role = 'student',
  }) {
    // 解析五维分数（后端 TwinResult.dimensions[]，key=academic 等）
    double dim(String key) {
      if (twinJson == null) return 0;
      final dims = twinJson['dimensions'] as List? ?? [];
      for (final d in dims) {
        if (d is Map && d['key'] == key) {
          final s = d['score'];
          return (s is num) ? s.toDouble() : 0;
        }
      }
      return 0;
    }

    // 解析大五人格（后端 PersonalityResult.big_five）
    double bigFive(String key) {
      final bf = personalityJson?['big_five'];
      if (bf is! Map) return 50;
      final v = bf[key];
      return (v is num) ? v.toDouble() : 50;
    }

    return AvatarConfig(
      academic: dim('academic'),
      ability: dim('ability'),
      ideological: dim('ideological'),
      emotional: dim('emotional'),
      social: dim('social'),
      openness: bigFive('openness'),
      extraversion: bigFive('extraversion'),
      personalityType: (personalityJson?['type'] as String?) ?? '',
      displayName: displayName,
      major: major,
      role: role,
    );
  }
}
