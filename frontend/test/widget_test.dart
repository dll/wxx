import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wxx_app/theme/app_theme.dart';

void main() {
  test('年级主题只改变局部强调色', () {
    const gradeAccent = Color(0xFFEF6C00);
    final theme = AppTheme.light(gradeAccent: gradeAccent);
    final appColors = theme.extension<AppColors>();

    expect(theme.colorScheme.primary, AppColors.brandPrimary);
    expect(appColors?.gradeAccent, gradeAccent);
    expect(appColors?.ai, AppColors.aiAccent);
  });

  test('亮暗主题均提供统一设计 Token', () {
    const gradeAccent = Color(0xFF00897B);
    final light = AppTheme.light(gradeAccent: gradeAccent);
    final dark = AppTheme.dark(gradeAccent: gradeAccent);

    expect(light.brightness, Brightness.light);
    expect(dark.brightness, Brightness.dark);
    expect(light.extension<AppColors>(), isNotNull);
    expect(dark.extension<AppColors>(), isNotNull);
  });
}
