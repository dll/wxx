import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../providers/twin_portrait_provider.dart';
import '../../providers/personal_detail_provider.dart';
import '../../utils/storage.dart';
import '../../utils/portrait_photo_picker.dart';
import '../../widgets/avatar_card.dart';
import '../../widgets/error_view.dart';
import '../../widgets/md_text.dart';

class DigitalTwinPage extends StatefulWidget {
  const DigitalTwinPage({super.key});
  @override
  State<DigitalTwinPage> createState() => _DigitalTwinPageState();
}

class _DigitalTwinPageState extends State<DigitalTwinPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final provider = context.read<StudentFeatureProvider>();
      // fire-and-forget：必须兜底捕获，避免未 await 的 Future 把异常抛进
      // Flutter zone（DioException 原始堆栈直接冒到界面/控制台）。
      provider.fetchDigitalTwin().catchError((Object e) {
        debugPrint('[digital-twin] 加载数字孪生失败: $e');
      });
      provider.fetchAvatar(
        displayName: Storage.displayName ?? '同学',
      ).catchError((Object e) {
        debugPrint('[digital-twin] 加载数字人形象失败: $e');
      });
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    return Scaffold(
      appBar: AppBar(title: const Text('数字孪生画像')),
      body: RefreshIndicator(
        onRefresh: () async {
          final p = context.read<StudentFeatureProvider>();
          await p.fetchDigitalTwin();
          await p.fetchAvatar(displayName: Storage.displayName ?? '同学', role: Storage.role ?? 'student');
        },
        child: provider.loading && provider.twin == null
            ? const Center(child: CircularProgressIndicator())
            : provider.error.isNotEmpty && provider.twin == null
                ? ErrorView.error(
                    message: provider.error,
                    onRetry: () => provider.fetchDigitalTwin())
                : _buildContent(theme, provider),
      ),
    );
  }

  Widget _buildContent(ThemeData theme, StudentFeatureProvider provider) {
    final t = provider.twin;
    if (t == null) return const Center(child: Text('暂无数据'));
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        // 数字人形象卡片（可由系统设置显示/隐藏）
        if (Storage.showAvatar) ...[
          if (provider.avatar != null)
            AvatarCard(
              config: provider.avatar!,
              height: 320,
            )
          else
            _buildAvatarLoading(theme),
          const SizedBox(height: 16),
        ],

        // 数字孪生画像（AI 文生图/图生图，蔚小芯风格）
        _buildPortraitSection(theme),

        const SizedBox(height: 16),

        // 信息概览：综合分 + 姓名 + 专业
        _buildOverviewCard(theme, t, provider),

        const SizedBox(height: 16),

        // 能力维度 Tab 区
        if (t.dimensions.isNotEmpty)
          _buildTabsSection(theme, provider),
      ],
    );
  }

  /// 数字人加载占位
  Widget _buildAvatarLoading(ThemeData theme) {
    return Container(
      height: 320,
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.5),
        borderRadius: BorderRadius.circular(24),
      ),
      child: const Center(child: CircularProgressIndicator()),
    );
  }

  // ── 数字孪生画像（AI 生成）──

  Widget _buildPortraitSection(ThemeData theme) {
    final p = context.watch<TwinPortraitProvider>();
    // 首次进入拉取已生成画像 + 个人信息（含头像，供图生图）
    if (!p.loading && p.current == null && p.error.isEmpty) {
      Future.microtask(() {
        if (mounted) {
          context.read<TwinPortraitProvider>().fetchPortraits().catchError(
              (Object e) {
            debugPrint('[digital-twin] 加载孪生画像失败: $e');
          });
        }
      });
    }
    final detail = context.watch<PersonalDetailProvider>();
    if (!detail.loading && detail.detail == null) {
      Future.microtask(() {
        if (mounted) {
          context.read<PersonalDetailProvider>().fetchAll().catchError(
              (Object e) {
            debugPrint('[digital-twin] 加载个人信息失败: $e');
          });
        }
      });
    }
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.auto_awesome, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text('AI 数字孪生画像',
                    style: theme.textTheme.titleMedium
                        ?.copyWith(fontWeight: FontWeight.w700)),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              '以照片或校园原型生成卡通 AI 数字孪生画像：柔和卡通 + 科技光效，保留真实五官辨识度。',
              style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant),
            ),
            const SizedBox(height: 12),
            if (p.generating)
              const Padding(
                padding: EdgeInsets.symmetric(vertical: 24),
                child: Center(child: CircularProgressIndicator()),
              )
            else if (p.current != null)
              _buildPortraitImage(theme, p)
            else
              _buildPortraitEmpty(theme, p),
            if (p.error.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(p.error,
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: theme.colorScheme.error)),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildPortraitImage(ThemeData theme, TwinPortraitProvider p) {
    final portrait = p.current!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        ClipRRect(
          borderRadius: BorderRadius.circular(12),
          child: Image.memory(
            base64Decode(portrait.imageBase64),
            fit: BoxFit.cover,
            width: double.infinity,
            // 与生成尺寸（256x256）一致，避免放大模糊
            height: 256,
            errorBuilder: (_, __, ___) => const SizedBox(
              height: 120,
              child: Center(child: Text('画像数据异常')),
            ),
          ),
        ),
        const SizedBox(height: 8),
        Row(
          children: [
            Chip(
              label: Text(portrait.prototypeType == 'photo' ? '照片版' : '校园原型版'),
              visualDensity: VisualDensity.compact,
            ),
            const Spacer(),
            TextButton.icon(
              onPressed: () => _showGenerateDialog(p),
              icon: const Icon(Icons.refresh, size: 16),
              label: const Text('重新生成'),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildPortraitEmpty(ThemeData theme, TwinPortraitProvider p) {
    // 若个人中心已上传头像，可直接用它生成（图生图）
    final detail = context.read<PersonalDetailProvider>().detail;
    final hasAvatar = (detail?.avatarBase64 ?? '').isNotEmpty;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (hasAvatar)
          OutlinedButton.icon(
            onPressed: () => _generateFromAvatar(p, detail!.avatarBase64),
            icon: const Icon(Icons.face_retouching_natural),
            label: const Text('用我的头像生成'),
          )
        else
          OutlinedButton.icon(
            onPressed: () => _showGenerateDialog(p),
            icon: const Icon(Icons.add_a_photo_outlined),
            label: const Text('上传照片生成'),
          ),
        const SizedBox(height: 8),
        TextButton.icon(
          onPressed: () => _generateChaoXing(p),
          icon: const Icon(Icons.person_outline),
          label: const Text('以校园原型生成（无需照片）'),
        ),
      ],
    );
  }

  /// 用个人中心头像（图生图）生成画像
  Future<void> _generateFromAvatar(TwinPortraitProvider p, String avatarB64) async {
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('正在用你的头像生成画像…')));
    }
    final ok = await p.generate(
      prototypeType: 'photo',
      photoBase64: avatarB64,
      photoMime: 'image/png',
    );
    if (!ok && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(p.error.isNotEmpty ? p.error : '生成失败')));
    }
  }

  /// 生成弹窗：选择照片模式（可拍照/相册/粘贴）或直接生成
  void _showGenerateDialog(TwinPortraitProvider p) {
    final controller = TextEditingController();
    var highlights = '';
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('生成 AI 数字孪生画像'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('照片模式：以上传照片为原型（图生图）'),
              const SizedBox(height: 8),
              _buildPhotoUploadRow(p, controller),
              const SizedBox(height: 12),
              const Text('校园原型模式：以标准校园学生形象生成'),
              const SizedBox(height: 8),
              TextField(
                controller: controller,
                decoration: const InputDecoration(
                  labelText: '个性化亮点（可选，如：学业优秀、运动健将）',
                  isDense: true,
                  border: OutlineInputBorder(),
                ),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(
            onPressed: () async {
              highlights = controller.text.trim();
              Navigator.pop(ctx);
              final ok = await p.generate(
                prototypeType: 'chao_xing',
                highlights: highlights,
              );
              if (!ok && mounted) {
                ScaffoldMessenger.of(context).showSnackBar(SnackBar(
                    content: Text(p.error.isNotEmpty ? p.error : '生成失败')));
              }
            },
            child: const Text('以校园原型生成'),
          ),
        ],
      ),
    );
  }

  Widget _buildPhotoUploadRow(TwinPortraitProvider p, TextEditingController highlightsCtrl) {
    return FutureBuilder<void>(
      future: null,
      builder: (context, _) {
        // 平台上传由各端实现；Web 用 file picker，移动端用 image_picker。
        // 此处提供 Web 文件选择 + 移动端引导。
        return _buildPhotoPickerButton(p);
      },
    );
  }

  Widget _buildPhotoPickerButton(TwinPortraitProvider p) {
    return OutlinedButton.icon(
      onPressed: () => _pickAndGenerate(p),
      icon: const Icon(Icons.upload),
      label: const Text('选择照片'),
    );
  }

  Future<void> _pickAndGenerate(TwinPortraitProvider p) async {
    // 平台条件编译：Web 用 FilePicker 转 base64，移动端用 image_picker。
    // 当前统一走 Web 文件选择；移动端可通过 image_picker 扩展。
    try {
      final result = await _pickImageBytes();
      if (result == null) return;
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('照片已选择，正在生成画像…')));
      }
      final ok = await p.generate(
        prototypeType: 'photo',
        photoBase64: base64Encode(result.bytes),
        photoMime: result.mime,
      );
      if (!ok && mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
            content: Text(p.error.isNotEmpty ? p.error : '生成失败')));
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('选择照片失败：$e')));
      }
    }
  }

  Future<void> _generateChaoXing(TwinPortraitProvider p) async {
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('正在以校园原型生成画像…')));
    }
    final ok = await p.generate(prototypeType: 'chao_xing');
    if (!ok && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(p.error.isNotEmpty ? p.error : '生成失败')));
    }
  }

  Future<({Uint8List bytes, String mime})?> _pickImageBytes() async {
    return pickPortraitPhoto();
  }

  /// 综合概览卡片
  Widget _buildOverviewCard(
      ThemeData theme, dynamic t, StudentFeatureProvider provider) {
    // 综合分：五维加权（后端顺序：学业/能力/思想/情感/社交，权重 0.30/0.25/0.15/0.15/0.15）
    // 仅对 data_available 的维度加权，并对缺失维度权重重新归一化，避免无数据维度拉低总分
    const weights = [0.30, 0.25, 0.15, 0.15, 0.15];
    final dims = (t.dimensions as List);
    double overall = 0;
    double weightSum = 0;
    for (int i = 0; i < dims.length && i < weights.length; i++) {
      final d = dims[i];
      final available = d.dataAvailable ?? true;
      if (!available) continue;
      final s = (d.score as num).toDouble();
      overall += (s > 1 ? s / 100.0 : s) * weights[i];
      weightSum += weights[i];
    }
    if (weightSum > 0) overall = overall / weightSum * 100;
    final label = overall >= 80
        ? '优秀'
        : overall >= 60
            ? '良好'
            : '待提升';

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            Container(
              width: 64,
              height: 64,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                gradient: LinearGradient(
                  colors: [
                    theme.colorScheme.primary,
                    theme.colorScheme.tertiary,
                  ],
                ),
              ),
              alignment: Alignment.center,
              child: Text(
                overall.toStringAsFixed(0),
                style: const TextStyle(
                  color: Colors.white,
                  fontSize: 24,
                  fontWeight: FontWeight.bold,
                ),
              ),
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    '综合画像 · $label',
                    style: theme.textTheme.titleMedium
                        ?.copyWith(fontWeight: FontWeight.bold),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '${Storage.displayName ?? '同学'} · ${Storage.role == 'student_union' ? '学生会' : '学生'}',
                    style: TextStyle(
                      color: theme.colorScheme.onSurfaceVariant,
                      fontSize: 13,
                    ),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    '点击下方雷达图查看各维度详情，数字人形象随数据自动变化',
                    style: TextStyle(
                      color: theme.colorScheme.outline,
                      fontSize: 12,
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// Tab 区：雷达图 / AI 分析 / 成长建议
  Widget _buildTabsSection(ThemeData theme, StudentFeatureProvider provider) {
    final t = provider.twin!;
    return DefaultTabController(
      length: 3,
      child: Card(
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
          side: BorderSide(color: theme.colorScheme.outlineVariant),
        ),
        child: Column(
          children: [
            const TabBar(
              tabs: [
                Tab(icon: Icon(Icons.radar, size: 20), text: '能力雷达'),
                Tab(icon: Icon(Icons.psychology, size: 20), text: 'AI 分析'),
                Tab(icon: Icon(Icons.lightbulb_outline, size: 20), text: '成长建议'),
              ],
            ),
            SizedBox(
              height: 380,
              child: TabBarView(
                children: [
                  // 雷达图
                  SingleChildScrollView(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      children: [
                        SizedBox(
                          height: 240,
                          child: _RadarChart(
                            dimensions: t.dimensions,
                            idealDimensions: t.idealDimensions,
                            color: theme.colorScheme.primary,
                            secondaryColor: theme.colorScheme.tertiary,
                          ),
                        ),
                        const SizedBox(height: 16),
                        // 各维度详情
                        ...t.dimensions.map((d) {
                          if (!d.dataAvailable) {
                            return Padding(
                              padding: const EdgeInsets.only(bottom: 12),
                              child: Row(
                                mainAxisAlignment:
                                    MainAxisAlignment.spaceBetween,
                                children: [
                                  Text(d.name),
                                  Text(
                                    '数据积累中',
                                    style: theme.textTheme.bodySmall?.copyWith(
                                        color:
                                            theme.colorScheme.onSurfaceVariant),
                                  ),
                                ],
                              ),
                            );
                          }
                          final normalized =
                              d.score > 1 ? d.score / 100.0 : d.score;
                          return Padding(
                            padding: const EdgeInsets.only(bottom: 12),
                            child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Row(
                                    mainAxisAlignment:
                                        MainAxisAlignment.spaceBetween,
                                    children: [
                                      Text(d.name),
                                      Text(
                                        d.label.isNotEmpty
                                            ? d.label
                                            : '${(normalized * 100).toInt()}%',
                                        style: theme.textTheme.bodySmall,
                                      ),
                                    ],
                                  ),
                                  const SizedBox(height: 4),
                                  LinearProgressIndicator(
                                    value: normalized.clamp(0.0, 1.0),
                                    backgroundColor: theme
                                        .colorScheme.surfaceContainerHighest,
                                    color: normalized >= 0.8
                                        ? Colors.green
                                        : normalized >= 0.5
                                            ? Colors.orange
                                            : Colors.red,
                                  ),
                                ]),
                          );
                        }),
                      ],
                    ),
                  ),
                  // AI 分析
                  SingleChildScrollView(
                    padding: const EdgeInsets.all(16),
                    child: t.aiSummary.isNotEmpty
                        ? Card(
                            color: theme.colorScheme.primaryContainer,
                            child: Padding(
                              padding: const EdgeInsets.all(16),
                              child: Column(
                                  crossAxisAlignment:
                                      CrossAxisAlignment.start,
                                  children: [
                                    Row(children: [
                                      Icon(Icons.psychology,
                                          color: theme
                                              .colorScheme.onPrimaryContainer),
                                      const SizedBox(width: 8),
                                      Text('AI 分析',
                                          style: theme.textTheme.titleSmall),
                                    ]),
                                    const SizedBox(height: 8),
                                    MdText(t.aiSummary),
                                  ]),
                            ),
                          )
                        : const Center(child: Text('暂无 AI 分析')),
                  ),
                  // 成长建议
                  SingleChildScrollView(
                    padding: const EdgeInsets.all(16),
                    child: t.suggestions.isNotEmpty
                        ? Column(
                            children: t.suggestions.asMap().entries.map((e) {
                              return Card(
                                child: ListTile(
                                  leading: CircleAvatar(
                                    backgroundColor: theme
                                        .colorScheme.secondaryContainer,
                                    child: Text('${e.key + 1}'),
                                  ),
                                  title: Text(e.value),
                                ),
                              );
                            }).toList(),
                          )
                        : const Center(child: Text('暂无成长建议')),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// 五维雷达图绘制组件
class _RadarChart extends StatelessWidget {
  final List<dynamic> dimensions; // TwinDimension 列表
  final List<dynamic> idealDimensions; // 理想值（可选）
  final Color color;
  final Color secondaryColor;

  const _RadarChart({
    required this.dimensions,
    required this.idealDimensions,
    required this.color,
    required this.secondaryColor,
  });

  @override
  Widget build(BuildContext context) {
    return CustomPaint(
      size: const Size(double.infinity, 240),
      painter: _RadarChartPainter(
        labels: dimensions.map((d) => d.name as String).toList(),
        values: dimensions.map((d) {
          final s = (d.score as num).toDouble();
          return s > 1 ? s / 100.0 : s;
        }).toList(),
        idealValues: idealDimensions.map((d) {
          final s = (d.score as num).toDouble();
          return s > 1 ? s / 100.0 : s;
        }).toList(),
        color: color,
        secondaryColor: secondaryColor,
      ),
    );
  }
}

class _RadarChartPainter extends CustomPainter {
  final List<String> labels;
  final List<double> values;
  final List<double> idealValues;
  final Color color;
  final Color secondaryColor;

  _RadarChartPainter({
    required this.labels,
    required this.values,
    required this.idealValues,
    required this.color,
    required this.secondaryColor,
  });

  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height / 2);
    final radius = min(size.width, size.height) / 2 - 30;
    final n = labels.length;
    if (n == 0) return;

    final angleStep = 2 * pi / n;
    // 从顶部开始 (-pi/2)
    final startAngle = -pi / 2;

    // 绘制网格（5 层同心多边形）
    for (int level = 1; level <= 5; level++) {
      final levelRadius = radius * level / 5.0;
      final path = Path();
      for (int i = 0; i < n; i++) {
        final angle = startAngle + i * angleStep;
        final x = center.dx + levelRadius * cos(angle);
        final y = center.dy + levelRadius * sin(angle);
        if (i == 0) {
          path.moveTo(x, y);
        } else {
          path.lineTo(x, y);
        }
      }
      path.close();
      canvas.drawPath(
        path,
        Paint()
          ..color = Colors.grey.withOpacity(0.15)
          ..style = PaintingStyle.stroke
          ..strokeWidth = 1,
      );
    }

    // 绘制轴线
    for (int i = 0; i < n; i++) {
      final angle = startAngle + i * angleStep;
      final x = center.dx + radius * cos(angle);
      final y = center.dy + radius * sin(angle);
      canvas.drawLine(
        center,
        Offset(x, y),
        Paint()
          ..color = Colors.grey.withOpacity(0.3)
          ..strokeWidth = 1,
      );
    }

    // 绘制理想值（淡色虚线多边形）
    if (idealValues.length == n) {
      final idealPath = Path();
      for (int i = 0; i < n; i++) {
        final angle = startAngle + i * angleStep;
        final value = idealValues[i].clamp(0.0, 1.0);
        final vRadius = radius * value;
        final x = center.dx + vRadius * cos(angle);
        final y = center.dy + vRadius * sin(angle);
        if (i == 0) {
          idealPath.moveTo(x, y);
        } else {
          idealPath.lineTo(x, y);
        }
      }
      idealPath.close();
      canvas.drawPath(
        idealPath,
        Paint()
          ..color = secondaryColor.withOpacity(0.4)
          ..style = PaintingStyle.stroke
          ..strokeWidth = 2,
      );
      // 理想值数据点
      for (int i = 0; i < n; i++) {
        final angle = startAngle + i * angleStep;
        final value = idealValues[i].clamp(0.0, 1.0);
        final vRadius = radius * value;
        canvas.drawCircle(
          Offset(center.dx + vRadius * cos(angle),
              center.dy + vRadius * sin(angle)),
          3,
          Paint()..color = secondaryColor.withOpacity(0.6),
        );
      }
    }

    // 绘制实际值（填充 + 描边）
    final dataPath = Path();
    for (int i = 0; i < n; i++) {
      final angle = startAngle + i * angleStep;
      final value = values[i].clamp(0.0, 1.0);
      final vRadius = radius * value;
      final x = center.dx + vRadius * cos(angle);
      final y = center.dy + vRadius * sin(angle);
      if (i == 0) {
        dataPath.moveTo(x, y);
      } else {
        dataPath.lineTo(x, y);
      }
    }
    dataPath.close();

    // 填充
    canvas.drawPath(
      dataPath,
      Paint()
        ..color = color.withOpacity(0.2)
        ..style = PaintingStyle.fill,
    );

    // 描边
    canvas.drawPath(
      dataPath,
      Paint()
        ..color = color
        ..style = PaintingStyle.stroke
        ..strokeWidth = 2.5,
    );

    // 数据点
    for (int i = 0; i < n; i++) {
      final angle = startAngle + i * angleStep;
      final value = values[i].clamp(0.0, 1.0);
      final vRadius = radius * value;
      final point = Offset(
        center.dx + vRadius * cos(angle),
        center.dy + vRadius * sin(angle),
      );
      canvas.drawCircle(
        point,
        5,
        Paint()..color = color,
      );
      canvas.drawCircle(
        point,
        2.5,
        Paint()..color = Colors.white,
      );
    }

    // 标签
    for (int i = 0; i < n; i++) {
      final angle = startAngle + i * angleStep;
      // 标签放在多边形外侧
      final labelRadius = radius + 22;
      final x = center.dx + labelRadius * cos(angle);
      final y = center.dy + labelRadius * sin(angle);

      final textPainter = TextPainter(
        text: TextSpan(
          text: labels[i],
          style: TextStyle(
            color: Colors.grey.shade700,
            fontSize: 12,
            fontWeight: FontWeight.w500,
          ),
        ),
        textDirection: TextDirection.ltr,
        textAlign: TextAlign.center,
      );
      textPainter.layout(maxWidth: 80);
      textPainter.paint(
        canvas,
        Offset(x - textPainter.width / 2, y - textPainter.height / 2),
      );
    }
  }

  @override
  bool shouldRepaint(covariant _RadarChartPainter oldDelegate) {
    return values != oldDelegate.values ||
        idealValues != oldDelegate.idealValues;
  }
}
