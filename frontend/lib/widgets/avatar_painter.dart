import 'dart:math';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';

import '../models/avatar_config.dart';

/// 卡通数字人绘制 — 根据 AvatarConfig 数据驱动生成个性化卡通形象。
///
/// 布局：背景(滁州学院建筑剪影 + 校徽水印 + 浮动粒子) → 人物(头部+发型+表情+身体+服装+配饰)
/// 元素全部由五维孪生分数 + 性格洞察驱动，同一用户数据不同 → 数字人不同。
class AvatarPainter extends CustomPainter {
  final AvatarConfig config;
  final Color primary;
  final Color secondary;

  AvatarPainter({required this.config, required this.primary, required this.secondary});

  // 确定性伪随机（固定 seed，避免每次重绘位置跳动）
  final Random _rand = Random(42);

  @override
  void paint(Canvas canvas, Size size) {
    _drawBackground(canvas, size);
    _drawPerson(canvas, size);
    _drawParticles(canvas, size);
    _drawSchoolBadge(canvas, size);
  }

  // ─────────────────────────────────────────────────────────
  // 背景：滁州学院元素
  // ─────────────────────────────────────────────────────────
  void _drawBackground(Canvas canvas, Size size) {
    final w = size.width;
    final h = size.height;

    // 渐变天空（校徽蓝 → 淡紫）
    final bg = Paint()
      ..shader = ui.Gradient.linear(
        Offset(0, 0),
        Offset(0, h),
        [primary.withOpacity(0.20), primary.withOpacity(0.05), secondary.withOpacity(0.10)],
      );
    canvas.drawRect(Offset.zero & size, bg);

    // 地平线附近：建筑剪影（教学楼轮廓）
    final building = Paint()
      ..color = primary.withOpacity(0.12)
      ..style = PaintingStyle.fill;
    final baseY = h * 0.88;
    final roof = Paint()
      ..color = primary.withOpacity(0.08)
      ..style = PaintingStyle.fill;

    // 主楼
    final bw = w * 0.34;
    final bh = h * 0.16;
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        Rect.fromLTWH(w * 0.12, baseY - bh, bw, bh),
        const Radius.circular(4),
      ),
      building,
    );
    // 主楼顶（三角屋顶，学院风）
    final roofPath = Path()
      ..moveTo(w * 0.12 - bw * 0.10, baseY - bh)
      ..lineTo(w * 0.12 + bw / 2, baseY - bh - h * 0.07)
      ..lineTo(w * 0.12 + bw + bw * 0.10, baseY - bh)
      ..close();
    canvas.drawPath(roofPath, roof);

    // 左侧副楼
    final bw2 = w * 0.20;
    final bh2 = h * 0.10;
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        Rect.fromLTWH(w * 0.55, baseY - bh2, bw2, bh2),
        const Radius.circular(4),
      ),
      building,
    );
    // 副楼屋顶
    final roof2 = Path()
      ..moveTo(w * 0.55 - bw2 * 0.10, baseY - bh2)
      ..lineTo(w * 0.55 + bw2 / 2, baseY - bh2 - h * 0.05)
      ..lineTo(w * 0.55 + bw2 + bw2 * 0.10, baseY - bh2)
      ..close();
    canvas.drawPath(roof2, roof);

    // 窗户（主楼，网格排列，营造教学楼感）
    final win = Paint()..color = Colors.white.withOpacity(0.35);
    final wx = w * 0.15;
    final wy = baseY - bh + h * 0.025;
    for (int r = 0; r < 3; r++) {
      for (int c = 0; c < 5; c++) {
        canvas.drawRRect(
          RRect.fromRectAndRadius(
            Rect.fromLTWH(wx + c * bw * 0.15, wy + r * h * 0.035, bw * 0.08, h * 0.022),
            const Radius.circular(1.5),
          ),
          win,
        );
      }
    }

    // 右侧绿色树（校园绿化）
    final tree = Paint()..color = Colors.green.withOpacity(0.20);
    canvas.drawRect(Rect.fromLTWH(w * 0.82, baseY - h * 0.05, w * 0.03, h * 0.05), tree);
    canvas.drawCircle(Offset(w * 0.835, baseY - h * 0.07), w * 0.035, tree);
  }

  // ─────────────────────────────────────────────────────────
  // 人物
  // ─────────────────────────────────────────────────────────
  void _drawPerson(Canvas canvas, Size size) {
    final w = size.width;
    final h = size.height;
    final cx = w / 2;
    final bodyTop = h * 0.42;

    // 阴影（脚下椭圆，营造漂浮/3D感）
    canvas.drawOval(
      Rect.fromCenter(center: Offset(cx, h * 0.80), width: w * 0.36, height: h * 0.035),
      Paint()..color = Colors.black.withOpacity(0.10),
    );

    // 身体 → 头部（从下往上画，头在上）
    _drawBody(canvas, cx, bodyTop, w, h);
    _drawHead(canvas, cx, bodyTop, w, h);
  }

  // ── 身体 + 服装 ──
  void _drawBody(Canvas canvas, double cx, double bodyTop, double w, double h) {
    final outfit = config.outfitColor;
    final neckW = w * 0.09;
    final neckH = h * 0.035;

    // 脖子
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        Rect.fromCenter(center: Offset(cx, bodyTop - neckH / 2), width: neckW, height: neckH),
        Radius.circular(neckH / 2),
      ),
      Paint()..color = const Color(0xFFE8B98A),
    );

    // 躯干（圆角梯形 → 用 RRect 近似）
    final torsoPath = Path()
      ..moveTo(cx - w * 0.16, bodyTop)
      ..quadraticBezierTo(cx, bodyTop - h * 0.03, cx + w * 0.16, bodyTop)
      ..lineTo(cx + w * 0.20, bodyTop + h * 0.30)
      ..quadraticBezierTo(cx, bodyTop + h * 0.36, cx - w * 0.20, bodyTop + h * 0.30)
      ..close();
    canvas.drawPath(torsoPath, Paint()..color = outfit);

    // 上衣高光（左侧光）
    final glow = Paint()
      ..color = Colors.white.withOpacity(0.18)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 3
      ..strokeCap = StrokeCap.round;
    canvas.drawLine(
      Offset(cx - w * 0.13, bodyTop + h * 0.03),
      Offset(cx - w * 0.16, bodyTop + h * 0.26),
      glow,
    );

    // 衣领（滁院蓝点缀）
    canvas.drawPath(
      Path()
        ..moveTo(cx - w * 0.05, bodyTop)
        ..lineTo(cx, bodyTop + h * 0.05)
        ..lineTo(cx + w * 0.05, bodyTop)
        ..close(),
      Paint()..color = AvatarConfig.schoolBlue,
    );

    // 手臂（身体两侧）
    final arm = Paint()..color = const Color(0xFFE8B98A);
    final sleeve = Paint()..color = outfit;
    // 左臂
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        Rect.fromCenter(center: Offset(cx - w * 0.21, bodyTop + h * 0.13), width: w * 0.075, height: h * 0.17),
        Radius.circular(w * 0.04),
      ),
      arm,
    );
    // 左袖
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        Rect.fromCenter(center: Offset(cx - w * 0.205, bodyTop + h * 0.055), width: w * 0.09, height: h * 0.055),
        Radius.circular(w * 0.04),
      ),
      sleeve,
    );
    // 右臂
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        Rect.fromCenter(center: Offset(cx + w * 0.21, bodyTop + h * 0.13), width: w * 0.075, height: h * 0.17),
        Radius.circular(w * 0.04),
      ),
      arm,
    );
    // 右袖
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        Rect.fromCenter(center: Offset(cx + w * 0.205, bodyTop + h * 0.055), width: w * 0.09, height: h * 0.055),
        Radius.circular(w * 0.04),
      ),
      sleeve,
    );

    // 书本（学业突出 → 捧书）
    if (config.hasBook) {
      final book = Paint()..color = AvatarConfig.schoolBlue;
      final page = Paint()..color = Colors.white;
      final bx = cx - w * 0.02;
      final by = bodyTop + h * 0.24;
      // 书脊 + 封面
      canvas.drawRRect(
        RRect.fromRectAndRadius(Rect.fromCenter(center: Offset(bx, by), width: w * 0.20, height: h * 0.11), const Radius.circular(3)),
        book,
      );
      // 书页（白边）
      canvas.drawRRect(
        RRect.fromRectAndRadius(Rect.fromCenter(center: Offset(bx, by), width: w * 0.16, height: h * 0.095), const Radius.circular(2)),
        page,
      );
      // 书页中线
      canvas.drawLine(
        Offset(bx, by - h * 0.045),
        Offset(bx, by + h * 0.045),
        Paint()..color = Colors.grey.shade300..strokeWidth = 1.5,
      );
    }

    // 奖牌（能力突出）
    if (config.hasMedal) {
      final medal = Paint()..color = const Color(0xFFFFC107);
      final ribbon = Paint()..color = const Color(0xFFE53935)..strokeWidth = 2.5;
      final mx = cx + w * 0.15;
      final my = bodyTop + h * 0.045;
      canvas.drawLine(Offset(cx + w * 0.07, bodyTop), Offset(mx, my - h * 0.02), ribbon);
      canvas.drawCircle(Offset(mx, my), w * 0.035, medal);
      canvas.drawCircle(Offset(mx, my), w * 0.035, Paint()..color = medal.color..style = PaintingStyle.stroke..strokeWidth = 1.5);
      // 星形
      final star = _starPoints(Offset(mx, my), w * 0.016, w * 0.007);
      canvas.drawPath(star, Paint()..color = Colors.white);
    }

    // 红色徽章（思想突出）
    if (config.hasRedBadge) {
      final badge = Paint()..color = const Color(0xFFE53935);
      final bx = cx - w * 0.14;
      final by = bodyTop + h * 0.06;
      canvas.drawCircle(Offset(bx, by), w * 0.028, badge);
      // 五角星
      final star = _starPoints(Offset(bx, by), w * 0.016, w * 0.007);
      canvas.drawPath(star, Paint()..color = Colors.white);
    }
  }

  // ── 头部 + 表情 ──
  void _drawHead(Canvas canvas, double cx, double bodyTop, double w, double h) {
    final skin = const Color(0xFFF5CBA7);
    final headW = w * 0.26;
    final headH = h * 0.22;
    final headCenter = Offset(cx, bodyTop - headH / 2 - h * 0.035);

    // 耳朵
    canvas.drawCircle(Offset(cx - headW / 2, headCenter.dy + headH * 0.05), w * 0.032, Paint()..color = skin);
    canvas.drawCircle(Offset(cx + headW / 2, headCenter.dy + headH * 0.05), w * 0.032, Paint()..color = skin);

    // 头部圆角矩形（卡通脸）
    final facePath = Path()
      ..addRRect(RRect.fromRectAndRadius(
        Rect.fromCenter(center: headCenter, width: headW, height: headH),
        Radius.circular(headH * 0.42),
      ));
    canvas.drawPath(facePath, Paint()..color = skin);

    // 腮红（社交分越高越明显）
    final blush = Paint()..color = const Color(0xFFFF8A80).withOpacity(0.3 + config.social / 100 * 0.3);
    canvas.drawOval(Rect.fromCenter(center: Offset(cx - headW * 0.30, headCenter.dy + headH * 0.10), width: w * 0.06, height: h * 0.018), blush);
    canvas.drawOval(Rect.fromCenter(center: Offset(cx + headW * 0.30, headCenter.dy + headH * 0.10), width: w * 0.06, height: h * 0.018), blush);

    // 眼睛（情感分 → 明亮度）
    final eyeY = headCenter.dy - headH * 0.05;
    final eyeBright = config.eyeBrightness;
    final eyeWhite = Paint()..color = Colors.white;
    final pupil = Paint()..color = const Color(0xFF37474F);
    final iris = Paint()..color = Color.lerp(const Color(0xFF1565C0), const Color(0xFF00ACC1), eyeBright)!;
    for (final side in [-1, 1]) {
      final ex = cx + side * headW * 0.22;
      // 眼白
      canvas.drawOval(Rect.fromCenter(center: Offset(ex, eyeY), width: w * 0.075, height: h * 0.042), eyeWhite);
      // 虹膜
      canvas.drawCircle(Offset(ex, eyeY), w * 0.024, iris);
      // 瞳孔
      canvas.drawCircle(Offset(ex, eyeY), w * 0.012, pupil);
      // 高光（情感越高光越亮）
      canvas.drawCircle(Offset(ex - w * 0.008, eyeY - h * 0.008), w * 0.006, Paint()..color = Colors.white.withOpacity(0.5 + eyeBright * 0.5));
    }

    // 嘴型（社交分决定笑容弧度）
    final mouthPaint = Paint()
      ..color = const Color(0xFFE57373)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2.5
      ..strokeCap = StrokeCap.round;
    final mouthY = headCenter.dy + headH * 0.16;
    if (config.isSmiling) {
      // 灿烂笑容（开口笑）
      canvas.drawArc(
        Rect.fromCenter(center: Offset(cx, mouthY + h * 0.004), width: w * 0.10, height: h * 0.045),
        pi * 0.15, pi * 0.7, false, mouthPaint,
      );
      // 露齿
      canvas.drawArc(
        Rect.fromCenter(center: Offset(cx, mouthY + h * 0.004), width: w * 0.075, height: h * 0.02),
        pi * 0.15, pi * 0.7, false, Paint()..color = Colors.white..style = PaintingStyle.fill,
      );
    } else {
      // 含蓄微笑
      canvas.drawArc(
        Rect.fromCenter(center: Offset(cx, mouthY), width: w * 0.075, height: h * 0.025),
        pi * 0.2, pi * 0.6, false, mouthPaint,
      );
    }

    // 鼻子（小圆点）
    canvas.drawCircle(Offset(cx, headCenter.dy + headH * 0.055), w * 0.009, Paint()..color = const Color(0xFFE8B98A));

    // 发型（开放分驱动）
    _drawHair(canvas, headCenter, headW, headH, w, h);
  }

  // ── 发型 ──
  void _drawHair(Canvas canvas, Offset headCenter, double headW, double headH, double w, double h) {
    final hairColor = const Color(0xFF5D4037);
    final hair = Paint()..color = hairColor;
    final top = headCenter.dy - headH / 2;

    switch (config.hairStyle) {
      case 'fluffy':
        // 蓬松创意发型（多个半圆鼓包）
        for (int i = 0; i < 6; i++) {
          final px = headCenter.dx - headW * 0.32 + i * headW * 0.13;
          final py = top + (i.isEven ? -h * 0.02 : 0);
          canvas.drawCircle(Offset(px, py + headH * 0.03), headW * 0.105, hair);
        }
        canvas.drawRRect(
          RRect.fromRectAndRadius(
            Rect.fromLTWH(headCenter.dx - headW / 2, top - h * 0.015, headW, headH * 0.16),
            Radius.circular(headH * 0.10),
          ),
          hair,
        );
        break;
      case 'short':
        // 利落短发（贴头平刘海）
        canvas.drawRRect(
          RRect.fromRectAndRadius(
            Rect.fromLTWH(headCenter.dx - headW / 2, top - h * 0.01, headW, headH * 0.14),
            Radius.circular(headH * 0.09),
          ),
          hair,
        );
        // 两侧鬓角
        canvas.drawRRect(
          RRect.fromRectAndRadius(
            Rect.fromCenter(center: Offset(headCenter.dx - headW * 0.40, headCenter.dy - headH * 0.05), width: headW * 0.09, height: headH * 0.22),
            Radius.circular(headW * 0.04),
          ),
          hair,
        );
        canvas.drawRRect(
          RRect.fromRectAndRadius(
            Rect.fromCenter(center: Offset(headCenter.dx + headW * 0.40, headCenter.dy - headH * 0.05), width: headW * 0.09, height: headH * 0.22),
            Radius.circular(headW * 0.04),
          ),
          hair,
        );
        break;
      default:
        // 标准短发（半圆刘海 + 顶部）
        canvas.drawCircle(Offset(headCenter.dx, top + headH * 0.04), headW * 0.55, hair);
        canvas.drawRRect(
          RRect.fromRectAndRadius(
            Rect.fromLTWH(headCenter.dx - headW / 2, top - h * 0.005, headW, headH * 0.12),
            Radius.circular(headH * 0.08),
          ),
          hair,
        );
    }

    // 眼镜（学业突出）
    if (config.hasGlasses) {
      final frame = Paint()
        ..color = const Color(0xFF37474F)
        ..style = PaintingStyle.stroke
        ..strokeWidth = 2;
      final lensY = headCenter.dy - headH * 0.05;
      canvas.drawOval(Rect.fromCenter(center: Offset(headCenter.dx - headW * 0.22, lensY), width: headW * 0.20, height: headH * 0.16), frame);
      canvas.drawOval(Rect.fromCenter(center: Offset(headCenter.dx + headW * 0.22, lensY), width: headW * 0.20, height: headH * 0.16), frame);
      // 鼻梁
      canvas.drawLine(
        Offset(headCenter.dx - headW * 0.02, lensY),
        Offset(headCenter.dx + headW * 0.02, lensY),
        frame,
      );
    }
  }

  // ── 浮动粒子（营造 3D 纵深与科技感） ──
  void _drawParticles(Canvas canvas, Size size) {
    final w = size.width;
    final h = size.height;
    final particle = Paint()..color = Colors.white.withOpacity(0.35);
    for (int i = 0; i < 14; i++) {
      final x = _rand.nextDouble() * w;
      final y = _rand.nextDouble() * h;
      final r = _rand.nextDouble() * 2.5 + 1.0;
      final alpha = 0.15 + _rand.nextDouble() * 0.3;
      particle.color = Colors.white.withOpacity(alpha);
      canvas.drawCircle(Offset(x, y), r, particle);
    }
  }

  // ── 校徽水印（右下角，滁州学院元素） ──
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
    // 校徽星形
    final star = _starPoints(Offset(cx, cy), r * 0.55, r * 0.24);
    canvas.drawPath(star, Paint()..color = primary.withOpacity(0.4));

    // 文字（"滁"）
    final tp = TextPainter(
      text: TextSpan(text: '滁', style: TextStyle(color: primary.withOpacity(0.6), fontSize: r * 1.4, fontWeight: FontWeight.bold)),
      textDirection: TextDirection.ltr,
    )..layout();
    tp.paint(canvas, Offset(cx - tp.width / 2, cy - tp.height / 2));
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
        oldDelegate.secondary != secondary;
  }
}
