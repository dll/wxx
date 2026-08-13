import 'package:flutter/material.dart';

/// 蔚小芯统一视觉 Token。
class AppColors extends ThemeExtension<AppColors> {
  static const brandPrimary = Color(0xFF1769AA);
  static const aiAccent = Color(0xFF008F83);
  static const attention = Color(0xFFB77900);
  static const danger = Color(0xFFC43D4B);
  static const success = Color(0xFF287A52);

  final Color ai;
  final Color warning;
  final Color positive;
  final Color gradeAccent;

  const AppColors({
    required this.ai,
    required this.warning,
    required this.positive,
    required this.gradeAccent,
  });

  @override
  AppColors copyWith({
    Color? ai,
    Color? warning,
    Color? positive,
    Color? gradeAccent,
  }) {
    return AppColors(
      ai: ai ?? this.ai,
      warning: warning ?? this.warning,
      positive: positive ?? this.positive,
      gradeAccent: gradeAccent ?? this.gradeAccent,
    );
  }

  @override
  AppColors lerp(covariant AppColors? other, double t) {
    if (other == null) return this;
    return AppColors(
      ai: Color.lerp(ai, other.ai, t)!,
      warning: Color.lerp(warning, other.warning, t)!,
      positive: Color.lerp(positive, other.positive, t)!,
      gradeAccent: Color.lerp(gradeAccent, other.gradeAccent, t)!,
    );
  }
}

class AppTheme {
  AppTheme._();

  static ThemeData light({required Color gradeAccent}) =>
      _build(Brightness.light, gradeAccent);

  static ThemeData dark({required Color gradeAccent}) =>
      _build(Brightness.dark, gradeAccent);

  static ThemeData _build(Brightness brightness, Color gradeAccent) {
    final isDark = brightness == Brightness.dark;
    final base = ColorScheme.fromSeed(
      seedColor: AppColors.brandPrimary,
      brightness: brightness,
    );
    final colors = base.copyWith(
      primary: isDark ? const Color(0xFF9CCBFF) : AppColors.brandPrimary,
      secondary: isDark ? const Color(0xFF70DBCD) : AppColors.aiAccent,
      tertiary: gradeAccent, // 年级主题色：驱动 Chip/标签/强调元素可见变化
      // 年级主题色同时渲染到导航选中态与输入框焦点，让「迎新」「追梦」等
      // 主题在全站可见（不改变操作色语义，仅增强视觉表达）。
      secondaryContainer: Color.alphaBlend(
        gradeAccent.withOpacity(isDark ? 0.28 : 0.16),
        base.secondaryContainer,
      ),
      onSecondaryContainer: base.onSecondaryContainer,
      error: isDark ? const Color(0xFFFFB3B8) : AppColors.danger,
      surface: isDark ? const Color(0xFF111418) : const Color(0xFFFAFBFC),
    );
    final textTheme = ThemeData(
      brightness: brightness,
      fontFamily: 'Roboto',
    ).textTheme.copyWith(
          headlineSmall: const TextStyle(
            fontSize: 24,
            height: 1.3,
            fontWeight: FontWeight.w700,
          ),
          titleLarge: const TextStyle(
            fontSize: 20,
            height: 1.35,
            fontWeight: FontWeight.w700,
          ),
          titleMedium: const TextStyle(
            fontSize: 16,
            height: 1.4,
            fontWeight: FontWeight.w600,
          ),
          bodyLarge: const TextStyle(fontSize: 16, height: 1.6),
          bodyMedium: const TextStyle(fontSize: 14, height: 1.55),
          bodySmall: const TextStyle(fontSize: 12, height: 1.5),
          labelLarge: const TextStyle(
            fontSize: 14,
            height: 1.3,
            fontWeight: FontWeight.w600,
          ),
        );

    return ThemeData(
      colorScheme: colors,
      useMaterial3: true,
      brightness: brightness,
      fontFamily: 'Roboto',
      textTheme: textTheme,
      scaffoldBackgroundColor: colors.surface,
      extensions: [
        AppColors(
          ai: isDark ? const Color(0xFF70DBCD) : AppColors.aiAccent,
          warning: isDark ? const Color(0xFFFFCA7A) : AppColors.attention,
          positive: isDark ? const Color(0xFF7CD7A8) : AppColors.success,
          gradeAccent: gradeAccent,
        ),
      ],
      appBarTheme: AppBarTheme(
        centerTitle: false,
        elevation: 0,
        scrolledUnderElevation: 0,
        backgroundColor: colors.surface,
        foregroundColor: colors.onSurface,
        titleTextStyle: textTheme.titleLarge?.copyWith(
          color: colors.onSurface,
        ),
      ),
      cardTheme: CardTheme(
        elevation: 0,
        margin: EdgeInsets.zero,
        color: colors.surfaceContainerLow,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
          side: BorderSide(color: colors.outlineVariant.withOpacity(0.7)),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: colors.surfaceContainerHighest.withOpacity(0.55),
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: BorderSide(color: colors.outlineVariant),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: BorderSide(color: colors.outlineVariant),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: BorderSide(color: colors.primary, width: 2),
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          minimumSize: const Size(48, 48),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(10),
          ),
          textStyle: textTheme.labelLarge,
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          minimumSize: const Size(48, 48),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(10),
          ),
          textStyle: textTheme.labelLarge,
        ),
      ),
      navigationBarTheme: NavigationBarThemeData(
        height: 68,
        elevation: 0,
        backgroundColor: colors.surface.withOpacity(0.94),
        indicatorColor: colors.secondaryContainer,
        labelTextStyle: WidgetStateProperty.resolveWith((states) {
          return textTheme.labelSmall?.copyWith(
            fontWeight: states.contains(WidgetState.selected)
                ? FontWeight.w700
                : FontWeight.w500,
          );
        }),
      ),
      navigationRailTheme: NavigationRailThemeData(
        elevation: 0,
        backgroundColor: colors.surface,
        indicatorColor: colors.secondaryContainer,
        selectedIconTheme: IconThemeData(color: colors.onSecondaryContainer),
        selectedLabelTextStyle: textTheme.labelLarge,
      ),
      dividerTheme: DividerThemeData(
        color: colors.outlineVariant.withOpacity(0.7),
        thickness: 1,
        space: 1,
      ),
      pageTransitionsTheme: const PageTransitionsTheme(
        builders: {
          TargetPlatform.android: FadeUpwardsPageTransitionsBuilder(),
          TargetPlatform.iOS: CupertinoPageTransitionsBuilder(),
          TargetPlatform.windows: FadeUpwardsPageTransitionsBuilder(),
          TargetPlatform.macOS: CupertinoPageTransitionsBuilder(),
          TargetPlatform.linux: FadeUpwardsPageTransitionsBuilder(),
        },
      ),
    );
  }
}
