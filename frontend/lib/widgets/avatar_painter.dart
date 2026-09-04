import 'dart:math';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';

import '../models/avatar_config.dart';

/// 星星造型 AI 智能体数字人 — 数据驱动生成 + 动画特效。
///
/// 设计：
///   - 头部：五角星形状（代表 AI 智能），戴学士帽/中式学位帽（角色区分）
///   - 眼睛：两个大圆点，情感分越高越亮，随动画眨眼
///   - 嘴：弧形微笑，社交分越高越开
///   - 身体：学位服（V领 + 彩色镶边），角色/外向度驱动颜色
///   - 光环：头顶漂浮光环，随动画脉冲发光
///   - 星星：随动画轻微上下浮动（呼吸感）
///   - 配饰：眼镜（学业高）/ 奖牌（能力高）/ 徽章（思想高）
///   - 背景：滁州学院建筑剪影 + 校徽水印 + 漂移粒子
///   - 标签：右上角「蔚小芯·AI助手」
class AvatarPainter extends CustomPainter {
  final AvatarConfig config;
  final Color primary;
  final Color secondary;

  /// 动画相位 0~1（由外层 AnimationController 驱动，-1 表示无动画）
  final double t;

  AvatarPainter({
    required this.config,
    required this.primary,
    required this.secondary,
    this.t = -1,
  });

  // 确定性伪随机（固定 seed，避免每次重绘位置跳动）
  final Random _rand = Random(42);

  @override
  void paint(Canvas canvas, Size size) {
    _drawBackground(canvas, size);
    _drawPerson(canvas, size);
    _drawParticles(canvas, size);
    _drawSchoolBadge(canvas, size);
    _drawBrandLabel(canvas, size);
  }

  // ─────────────────────────────────────────────────────────
  // 背景：滁州学院元素
  // ─────────────────────────────────────────────────────────
  void _drawBackground(Canvas canvas, Size size) {
    final w = size.width;
    final h = size.height;

    final bg = Paint()
      ..shader = ui.Gradient.linear(
        const Offset(0, 0),
        Offset(0, h),
        [
          primary.withOpacity(0.20),
          primary.withOpacity(0.05),
          secondary.withOpacity(0.10)
        ],
      );
    canvas.drawRect(Offset.zero & size, bg);

    // 地平线建筑剪影（教学楼）
    final building = Paint()
      ..color = primary.withOpacity(0.12)
      ..style = PaintingStyle.fill;
    final roof = Paint()
      ..color = primary.withOpacity(0.08)
      ..style = PaintingStyle.fill;
    final baseY = h * 0.88;

    final bw = w * 0.34;
    final bh = h * 0.16;
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        Rect.fromLTWH(w * 0.12, baseY - bh, bw, bh),
        const Radius.circular(4),
      ),
      building,
    );
    final roofPath = Path()
      ..moveTo(w * 0.12 - bw * 0.10, baseY - bh)
      ..lineTo(w * 0.12 + bw / 2, baseY - bh - h * 0.07)
      ..lineTo(w * 0.12 + bw + bw * 0.10, baseY - bh)
      ..close();
    canvas.drawPath(roofPath, roof);

    final bw2 = w * 0.20;
    final bh2 = h * 0.10;
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        Rect.fromLTWH(w * 0.55, baseY - bh2, bw2, bh2),
        const Radius.circular(4),
      ),
      building,
    );
    final roof2 = Path()
      ..moveTo(w * 0.55 - bw2 * 0.10, baseY - bh2)
      ..lineTo(w * 0.55 + bw2 / 2, baseY - bh2 - h * 0.05)
      ..lineTo(w * 0.55 + bw2 + bw2 * 0.10, baseY - bh2)
      ..close();
    canvas.drawPath(roof2, roof);

    // 主楼窗户
    final win = Paint()..color = Colors.white.withOpacity(0.35);
    final wx = w * 0.15;
    final wy = baseY - bh + h * 0.025;
    for (int r = 0; r < 3; r++) {
      for (int c = 0; c < 5; c++) {
        canvas.drawRRect(
          RRect.fromRectAndRadius(
            Rect.fromLTWH(
                wx + c * bw * 0.15, wy + r * h * 0.035, bw * 0.08, h * 0.022),
            const Radius.circular(1.5),
          ),
          win,
        );
      }
    }

    // 右侧绿树
    final tree = Paint()..color = Colors.green.withOpacity(0.20);
    canvas.drawRect(
        Rect.fromLTWH(w * 0.82, baseY - h * 0.05, w * 0.03, h * 0.05), tree);
    canvas.drawCircle(Offset(w * 0.835, baseY - h * 0.07), w * 0.035, tree);
  }

  // ─────────────────────────────────────────────────────────
  // 人物：星星智能体
  // ─────────────────────────────────────────────────────────
  void _drawPerson(Canvas canvas, Size size) {
    final w = size.width;
    final h = size.height;
    final cx = w / 2;
    final bodyTop = h * 0.46;

    // 星星上下浮动（呼吸感）：t 相位驱动 0~6px 位移
    final floatY = t >= 0 ? sin(t * 2 * pi) * h * 0.015 : 0.0;

    // 脚下阴影（随浮动缩放）
    final shadowScale = t >= 0 ? 1.0 - sin(t * 2 * pi) * 0.12 : 1.0;
    canvas.drawOval(
      Rect.fromCenter(
          center: Offset(cx, h * 0.82 + floatY * 0.5),
          width: w * 0.32 * shadowScale,
          height: h * 0.03 * shadowScale),
      Paint()..color = Colors.black.withOpacity(0.10),
    );

    // 光环（综合分高 → 更亮，随动画脉冲）
    _drawHalo(canvas, cx, bodyTop, w, h);

    // 整体上移 = floatY，模拟漂浮
    canvas.save();
    canvas.translate(0, floatY);

    // 人体打散组装：每个动画周期前 25% 让头部、身体从两侧归位，
    // 之后进入呼吸/眨眼/粒子循环，形成“蔚小芯正在组装自己”的陪伴感。
    final assembly = t < 0 ? 1.0 : (t / 0.25).clamp(0.0, 1.0);
    final ease = Curves.easeOut.transform(assembly);
    canvas.save();
    canvas.translate((1 - ease) * -w * 0.24, (1 - ease) * h * 0.08);
    _drawBody(canvas, cx, bodyTop, w, h);
    canvas.restore();
    canvas.save();
    canvas.translate((1 - ease) * w * 0.24, (1 - ease) * -h * 0.10);
    _drawStarHead(canvas, cx, bodyTop, w, h);
    canvas.restore();

    canvas.restore();
  }

  // ── 光环（AI 感漂浮光晕，随动画脉冲） ──
  void _drawHalo(Canvas canvas, double cx, double bodyTop, double w, double h) {
    final glow = config.overall / 100.0;
    // 脉冲：t 驱动 0.9~1.1 缩放
    final pulse = t >= 0 ? 1.0 + sin(t * 2 * pi) * 0.10 : 1.0;
    final ring = Paint()
      ..color = Colors.white.withOpacity(
          (0.25 + glow * 0.30) * (t >= 0 ? 1.0 + sin(t * 2 * pi) * 0.15 : 1.0))
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2.5;
    final r = w * 0.20 * pulse;
    // 头顶后方光晕（部分遮挡）
    canvas.drawOval(
      Rect.fromCenter(
          center: Offset(cx, bodyTop - h * 0.20),
          width: r * 2.2,
          height: r * 0.9),
      ring,
    );
    // 微光晕
    canvas.drawOval(
      Rect.fromCenter(
          center: Offset(cx, bodyTop - h * 0.20),
          width: r * 1.8,
          height: r * 0.6),
      Paint()
        ..color = Colors.white.withOpacity(0.10 + glow * 0.12)
        ..style = PaintingStyle.fill,
    );
  }

  // ── 学位服身体 ──
  void _drawBody(Canvas canvas, double cx, double bodyTop, double w, double h) {
    final gown = config.gownColor;

    // 躯干（梯形学位服）
    final body = Path()
      ..moveTo(cx - w * 0.17, bodyTop)
      ..lineTo(cx - w * 0.22, bodyTop + h * 0.34)
      ..quadraticBezierTo(
          cx, bodyTop + h * 0.40, cx + w * 0.22, bodyTop + h * 0.34)
      ..lineTo(cx + w * 0.17, bodyTop)
      ..close();
    canvas.drawPath(body, Paint()..color = gown);

    // 衣摆（深色边）
    final hem = Paint()..color = gown.withOpacity(0.85);
    canvas.drawPath(
      Path()
        ..moveTo(cx - w * 0.22, bodyTop + h * 0.34)
        ..quadraticBezierTo(
            cx, bodyTop + h * 0.40, cx + w * 0.22, bodyTop + h * 0.34)
        ..lineTo(cx + w * 0.24, bodyTop + h * 0.365)
        ..quadraticBezierTo(
            cx, bodyTop + h * 0.42, cx - w * 0.24, bodyTop + h * 0.365)
        ..close(),
      hem,
    );

    // V 领（露出衬衫）
    final collar = Path()
      ..moveTo(cx - w * 0.07, bodyTop)
      ..lineTo(cx, bodyTop + h * 0.07)
      ..lineTo(cx + w * 0.07, bodyTop)
      ..close();
    canvas.drawPath(collar, Paint()..color = const Color(0xFFF5F5F5));

    // 胸前 V 领镶边（校徽蓝点缀）
    canvas.drawPath(
      Path()
        ..moveTo(cx - w * 0.07, bodyTop)
        ..lineTo(cx, bodyTop + h * 0.07)
        ..lineTo(cx + w * 0.07, bodyTop),
      Paint()
        ..color = const Color(0xFF1565C0)
        ..style = PaintingStyle.stroke
        ..strokeWidth = 3
        ..strokeCap = StrokeCap.round,
    );

    // 领带/领结（红色）
    canvas.drawCircle(Offset(cx, bodyTop + h * 0.015), w * 0.018,
        Paint()..color = const Color(0xFFE53935));

    // 手臂（学位服袖）
    final sleeve = Paint()..color = gown.withOpacity(0.85);
    // 左袖
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        Rect.fromCenter(
            center: Offset(cx - w * 0.20, bodyTop + h * 0.13),
            width: w * 0.10,
            height: h * 0.20),
        Radius.circular(w * 0.05),
      ),
      sleeve,
    );
    // 右手（捧书）
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        Rect.fromCenter(
            center: Offset(cx + w * 0.20, bodyTop + h * 0.13),
            width: w * 0.10,
            height: h * 0.20),
        Radius.circular(w * 0.05),
      ),
      sleeve,
    );

    // 书本（学业突出 → 捧书）
    if (config.hasBook) {
      final book = Paint()..color = const Color(0xFF1565C0);
      final page = Paint()..color = Colors.white;
      final bx = cx;
      final by = bodyTop + h * 0.24;
      canvas.drawRRect(
        RRect.fromRectAndRadius(
            Rect.fromCenter(
                center: Offset(bx, by), width: w * 0.20, height: h * 0.10),
            const Radius.circular(3)),
        book,
      );
      canvas.drawRRect(
        RRect.fromRectAndRadius(
            Rect.fromCenter(
                center: Offset(bx, by), width: w * 0.16, height: h * 0.085),
            const Radius.circular(2)),
        page,
      );
      canvas.drawLine(
        Offset(bx, by - h * 0.04),
        Offset(bx, by + h * 0.04),
        Paint()
          ..color = Colors.grey.shade300
          ..strokeWidth = 1.5,
      );
    }

    // 奖牌（能力突出）
    if (config.hasMedal) {
      final medal = Paint()..color = const Color(0xFFFFC107);
      final ribbon = Paint()
        ..color = const Color(0xFFE53935)
        ..strokeWidth = 2.5;
      final mx = cx + w * 0.15;
      final my = bodyTop + h * 0.035;
      canvas.drawLine(
          Offset(cx + w * 0.06, bodyTop), Offset(mx, my - h * 0.02), ribbon);
      canvas.drawCircle(Offset(mx, my), w * 0.032, medal);
      canvas.drawCircle(
          Offset(mx, my),
          w * 0.032,
          Paint()
            ..color = medal.color
            ..style = PaintingStyle.stroke
            ..strokeWidth = 1.5);
      canvas.drawPath(_starPoints(Offset(mx, my), w * 0.015, w * 0.007),
          Paint()..color = Colors.white);
    }

    // 红色徽章（思想突出）
    if (config.hasRedBadge) {
      final badge = Paint()..color = const Color(0xFFE53935);
      final bx = cx - w * 0.14;
      final by = bodyTop + h * 0.05;
      canvas.drawCircle(Offset(bx, by), w * 0.026, badge);
      canvas.drawPath(_starPoints(Offset(bx, by), w * 0.015, w * 0.006),
          Paint()..color = Colors.white);
    }
  }

  // ── 五角星头 + 帽子 + 表情 ──
  void _drawStarHead(
      Canvas canvas, double cx, double bodyTop, double w, double h) {
    final headW = w * 0.30;
    final headH = h * 0.24;
    final headCenter = Offset(cx, bodyTop - headH / 2 - h * 0.03);

    // 星星头（五角星，代表 AI 智能）
    const starColor = Color(0xFFFFD54F); // 金色星星
    final starPath = _starPoints(headCenter, headW * 0.62, headW * 0.42);
    canvas.drawPath(starPath, Paint()..color = starColor);

    // 星星描边（暖色轮廓）
    canvas.drawPath(
      starPath,
      Paint()
        ..color = const Color(0xFFFFB300)
        ..style = PaintingStyle.stroke
        ..strokeWidth = 3,
    );

    // 顶部高光
    canvas.drawPath(
      _starPoints(headCenter, headW * 0.60, headW * 0.40),
      Paint()
        ..color = Colors.white.withOpacity(0.25)
        ..style = PaintingStyle.stroke
        ..strokeWidth = 2,
    );

    // 腮红（社交分越高越明显）
    final blush = Paint()
      ..color =
          const Color(0xFFFF8A80).withOpacity(0.25 + config.social / 100 * 0.3);
    canvas.drawOval(
        Rect.fromCenter(
            center: Offset(cx - headW * 0.30, headCenter.dy + headH * 0.12),
            width: w * 0.05,
            height: h * 0.015),
        blush);
    canvas.drawOval(
        Rect.fromCenter(
            center: Offset(cx + headW * 0.30, headCenter.dy + headH * 0.12),
            width: w * 0.05,
            height: h * 0.015),
        blush);

    // 眼睛（情感分 → 明亮度，随动画眨眼）
    final eyeBright = config.eyeBrightness;
    final eyeY = headCenter.dy - headH * 0.05;
    // 眨眼：t 在 0.42~0.52 区间眼睛高度收缩
    final blinkT = t >= 0 ? (t - 0.42) / 0.10 : -1.0;
    final isBlinking = blinkT >= 0 && blinkT <= 1.0;
    final eyeHeight = isBlinking ? h * 0.012 : h * 0.045; // 闭眼变细
    final eyeWhite = Paint()..color = Colors.white;
    final iris = Paint()
      ..color = Color.lerp(
          const Color(0xFF1565C0), const Color(0xFF00ACC1), eyeBright)!;
    final pupil = Paint()..color = const Color(0xFF37474F);
    for (final side in [-1, 1]) {
      final ex = cx + side * headW * 0.23;
      canvas.drawOval(
          Rect.fromCenter(
              center: Offset(ex, eyeY), width: w * 0.07, height: eyeHeight),
          eyeWhite);
      // 非眨眼时画瞳孔
      if (!isBlinking) {
        canvas.drawCircle(Offset(ex, eyeY), w * 0.022, iris);
        canvas.drawCircle(Offset(ex, eyeY), w * 0.011, pupil);
        // 高光
        canvas.drawCircle(Offset(ex - w * 0.008, eyeY - h * 0.008), w * 0.006,
            Paint()..color = Colors.white.withOpacity(0.5 + eyeBright * 0.5));
      }
    }

    // 嘴（社交分 → 微笑弧度）
    final mouthPaint = Paint()
      ..color = const Color(0xFFE57373)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2.5
      ..strokeCap = StrokeCap.round;
    final mouthY = headCenter.dy + headH * 0.14;
    if (config.isSmiling) {
      canvas.drawArc(
        Rect.fromCenter(
            center: Offset(cx, mouthY + h * 0.004),
            width: w * 0.09,
            height: h * 0.04),
        pi * 0.15,
        pi * 0.7,
        false,
        mouthPaint,
      );
    } else {
      canvas.drawArc(
        Rect.fromCenter(
            center: Offset(cx, mouthY), width: w * 0.07, height: h * 0.022),
        pi * 0.2,
        pi * 0.6,
        false,
        mouthPaint,
      );
    }

    // 眼镜（学业突出）
    if (config.hasGlasses) {
      final frame = Paint()
        ..color = const Color(0xFF37474F)
        ..style = PaintingStyle.stroke
        ..strokeWidth = 2;
      final lensY = eyeY;
      canvas.drawOval(
          Rect.fromCenter(
              center: Offset(cx - headW * 0.23, lensY),
              width: headW * 0.20,
              height: headH * 0.16),
          frame);
      canvas.drawOval(
          Rect.fromCenter(
              center: Offset(cx + headW * 0.23, lensY),
              width: headW * 0.20,
              height: headH * 0.16),
          frame);
      canvas.drawLine(Offset(cx - headW * 0.02, lensY),
          Offset(cx + headW * 0.02, lensY), frame);
    }

    // 帽子（角色区分）
    _drawHat(canvas, headCenter, headW, headH, w, h);
  }

  // ── 帽子 ──
  void _drawHat(Canvas canvas, Offset headCenter, double headW, double headH,
      double w, double h) {
    final top = headCenter.dy - headH * 0.62;
    if (config.hatStyle == 'chinese') {
      // 中式学位帽（教师）：红色圆顶 + 金边
      final cap = Paint()..color = const Color(0xFFB71C1C);
      canvas.drawRRect(
        RRect.fromRectAndRadius(
          Rect.fromLTWH(headCenter.dx - headW * 0.38, top - h * 0.035,
              headW * 0.76, headH * 0.16),
          Radius.circular(headH * 0.10),
        ),
        cap,
      );
      // 金色镶边
      canvas.drawRRect(
        RRect.fromRectAndRadius(
          Rect.fromLTWH(headCenter.dx - headW * 0.38, top - h * 0.035,
              headW * 0.76, headH * 0.06),
          Radius.circular(headH * 0.04),
        ),
        Paint()..color = const Color(0xFFFFD54F),
      );
      // 顶珠
      canvas.drawCircle(Offset(headCenter.dx, top - h * 0.045), w * 0.018,
          Paint()..color = const Color(0xFFFFC107));
    } else {
      // 学士帽（学生）：黑色方帽 + 帽穗
      final cap = Paint()..color = const Color(0xFF37474F);
      // 帽檐（菱形）
      final brim = Path()
        ..moveTo(headCenter.dx - headW * 0.48, top - h * 0.012)
        ..lineTo(headCenter.dx, top - h * 0.06)
        ..lineTo(headCenter.dx + headW * 0.48, top - h * 0.012)
        ..lineTo(headCenter.dx, top + h * 0.02)
        ..close();
      canvas.drawPath(brim, Paint()..color = const Color(0xFF263238));

      // 帽筒
      canvas.drawRRect(
        RRect.fromRectAndRadius(
          Rect.fromLTWH(headCenter.dx - headW * 0.34, top - h * 0.02,
              headW * 0.68, headH * 0.12),
          Radius.circular(headH * 0.05),
        ),
        cap,
      );

      // 帽穗（金线垂下）
      final tassel = Paint()
        ..color = const Color(0xFFFFC107)
        ..strokeWidth = 2.5
        ..strokeCap = StrokeCap.round;
      canvas.drawLine(
        Offset(headCenter.dx + headW * 0.42, top - h * 0.01),
        Offset(headCenter.dx + headW * 0.42, top + h * 0.08),
        tassel,
      );
      canvas.drawCircle(Offset(headCenter.dx + headW * 0.42, top + h * 0.08),
          w * 0.015, Paint()..color = const Color(0xFFFFC107));
    }
  }

  // ── 浮动粒子（随动画缓慢漂移，营造 AI 科技感） ──
  void _drawParticles(Canvas canvas, Size size) {
    final w = size.width;
    final h = size.height;
    final particle = Paint();
    final density = 10 + (config.overall / 100 * 8).round(); // 综合分越高粒子越多
    for (int i = 0; i < density; i++) {
      // 基于 i 的相位偏移，避免所有粒子同步动
      final phase = i * 0.7;
      final driftX = t >= 0 ? sin(t * 2 * pi + phase) * 4 : 0.0;
      final driftY = t >= 0 ? cos(t * 2 * pi + phase) * 3 : 0.0;
      final x = _rand.nextDouble() * w + driftX;
      final y = _rand.nextDouble() * h + driftY;
      final r = _rand.nextDouble() * 2.5 + 1.0;
      final alpha = 0.15 + _rand.nextDouble() * 0.3;
      particle.color = Colors.white.withOpacity(alpha);
      canvas.drawCircle(Offset(x, y), r, particle);
    }
  }

  // ── 校徽水印 ──
  void _drawSchoolBadge(Canvas canvas, Size size) {
    final w = size.width;
    final h = size.height;
    final cx = w * 0.85;
    final cy = h * 0.92;
    final r = w * 0.055;

    final ring = Paint()
      ..color = primary.withOpacity(0.35)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2;
    canvas.drawCircle(Offset(cx, cy), r, ring);
    canvas.drawPath(_starPoints(Offset(cx, cy), r * 0.55, r * 0.24),
        Paint()..color = primary.withOpacity(0.4));

    final tp = TextPainter(
      text: TextSpan(
          text: '滁',
          style: TextStyle(
              color: primary.withOpacity(0.6),
              fontSize: r * 1.4,
              fontWeight: FontWeight.bold)),
      textDirection: TextDirection.ltr,
    )..layout();
    tp.paint(canvas, Offset(cx - tp.width / 2, cy - tp.height / 2));
  }

  // ── 品牌标签「蔚小芯·AI助手」 ──
  void _drawBrandLabel(Canvas canvas, Size size) {
    final w = size.width;
    const label = '蔚小芯·AI助手';
    final tp = TextPainter(
      text: TextSpan(
        text: label,
        style: TextStyle(
          color: Colors.white.withOpacity(0.85),
          fontSize: 12,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.5,
        ),
      ),
      textDirection: TextDirection.ltr,
    )..layout();

    // 标签背景（圆角胶囊）
    const padX = 10.0;
    const padY = 5.0;
    final rect = RRect.fromRectAndRadius(
      Rect.fromLTWH(w * 0.62, 12, tp.width + padX * 2, tp.height + padY * 2),
      const Radius.circular(999),
    );
    canvas.drawRRect(
        rect, Paint()..color = const Color(0xFF1565C0).withOpacity(0.75));
    tp.paint(canvas, Offset(w * 0.62 + padX, 12 + padY));
  }

  // ── 五角星路径工具 ──
  Path _starPoints(Offset center, double outer, double inner) {
    final path = Path();
    for (int i = 0; i < 10; i++) {
      final angle = -pi / 2 + i * pi / 5;
      final r = i.isEven ? outer : inner;
      final p = Offset(center.dx + r * cos(angle), center.dy + r * sin(angle));
      if (i == 0) {
        path.moveTo(p.dx, p.dy);
      } else {
        path.lineTo(p.dx, p.dy);
      }
    }
    path.close();
    return path;
  }

  @override
  bool shouldRepaint(covariant AvatarPainter oldDelegate) {
    return oldDelegate.config != config ||
        oldDelegate.primary != primary ||
        oldDelegate.secondary != secondary ||
        oldDelegate.t != t;
  }
}
