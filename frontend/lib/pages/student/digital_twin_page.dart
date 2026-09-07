import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../providers/twin_portrait_provider.dart';
import '../../providers/personal_detail_provider.dart';
import '../../models/avatar_config.dart';
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
  /// 当前登录者是否为教辅/教师（counselor/teacher/assistant）
  bool get _isStaff {
    final r = Storage.role ?? '';
    return r == 'counselor' || r == 'teacher' || r == 'assistant';
  }

  /// 教辅/教师中文角色名（学生/学生会回退历史逻辑）
  String get _roleLabel {
    switch (Storage.role) {
      case 'counselor':
        return '辅导员';
      case 'teacher':
        return '教师';
      case 'assistant':
        return '教辅';
      case 'student_union':
        return '学生会';
      default:
        return '学生';
    }
  }

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
      provider
          .fetchAvatar(
        displayName: Storage.displayName ?? '同学',
      )
          .catchError((Object e) {
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
          await p.fetchAvatar(
              displayName: Storage.displayName ?? '同学',
              role: Storage.role ?? 'student');
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
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 32),
      children: [
        _buildPageIntro(theme, t),
        const SizedBox(height: 12),
        // 核心状态首屏可见：综合分、成长阶段、数据覆盖。
        _buildOverviewCard(theme, t, provider),
        const SizedBox(height: 12),
        _buildGrowthIdentityCard(theme, t, provider),
        const SizedBox(height: 12),
        if (t.dimensions.isNotEmpty) _buildTabsSection(theme, provider),
        const SizedBox(height: 12),
        // 数字人是表达层，不抢占真实成长数据的首屏位置。
        if (Storage.showAvatar && provider.avatar != null)
          _buildAvatarPreview(theme, provider.avatar!),
        const SizedBox(height: 12),
        _buildPortraitSection(theme),
      ],
    );
  }

  Widget _buildPageIntro(ThemeData theme, dynamic t) {
    final cs = theme.colorScheme;
    return Container(
      padding: const EdgeInsets.fromLTRB(18, 18, 18, 16),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(24),
        gradient: LinearGradient(
          colors: [cs.primaryContainer, cs.surface],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        border: Border.all(color: cs.outlineVariant),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 46,
            height: 46,
            decoration: BoxDecoration(
              color: cs.primary,
              borderRadius: BorderRadius.circular(16),
            ),
            child: Icon(Icons.insights_rounded, color: cs.onPrimary, size: 25),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  _isStaff ? '我的绩效画像' : '我的成长画像',
                  style: theme.textTheme.titleLarge?.copyWith(
                    fontWeight: FontWeight.w800,
                    letterSpacing: -0.3,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  t.profileTag.isNotEmpty ? t.profileTag : '用真实记录，看见正在发生的成长',
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: cs.onSurfaceVariant,
                    height: 1.35,
                  ),
                ),
              ],
            ),
          ),
          IconButton(
            tooltip: '刷新画像',
            onPressed: () =>
                context.read<StudentFeatureProvider>().fetchDigitalTwin(),
            icon: const Icon(Icons.refresh_rounded),
          ),
        ],
      ),
    );
  }

  Widget _buildAvatarPreview(ThemeData theme, AvatarConfig config) {
    return Card(
      elevation: 0,
      margin: EdgeInsets.zero,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(20),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 10, 12, 12),
        child: Row(
          children: [
            Icon(Icons.face_retouching_natural,
                color: theme.colorScheme.primary),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                '数字人表达',
                style: theme.textTheme.titleSmall
                    ?.copyWith(fontWeight: FontWeight.w700),
              ),
            ),
            SizedBox(
              width: 132,
              height: 86,
              child: ClipRRect(
                borderRadius: BorderRadius.circular(14),
                child: AvatarCard(config: config, height: 86),
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// 数字人加载占位
  // ── 数字孪生画像（AI 生成）──

  Widget _buildPortraitSection(ThemeData theme) {
    final p = context.watch<TwinPortraitProvider>();
    // 首次进入拉取已生成画像 + 个人信息（含头像，供图生图）
    if (!p.loading && p.current == null && p.error.isEmpty) {
      Future.microtask(() {
        if (mounted) {
          context
              .read<TwinPortraitProvider>()
              .fetchPortraits()
              .catchError((Object e) {
            debugPrint('[digital-twin] 加载孪生画像失败: $e');
          });
        }
      });
    }
    final detail = context.watch<PersonalDetailProvider>();
    if (!detail.loading && detail.detail == null) {
      Future.microtask(() {
        if (mounted) {
          context
              .read<PersonalDetailProvider>()
              .fetchAll()
              .catchError((Object e) {
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
              'Q 版可爱精灵数字人画像，大头小身萌态十足，超星风格 3D 卡通渲染。',
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
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
    final cs = theme.colorScheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Center(
          child: Container(
            width: 120,
            height: 120,
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(24),
              boxShadow: [
                BoxShadow(
                  color: cs.primary.withOpacity(0.15),
                  blurRadius: 16,
                  offset: const Offset(0, 4),
                ),
              ],
            ),
            child: ClipRRect(
              borderRadius: BorderRadius.circular(24),
              child: Image.memory(
                base64Decode(portrait.imageBase64),
                fit: BoxFit.cover,
                width: 120,
                height: 120,
                errorBuilder: (_, __, ___) => Container(
                  width: 120,
                  height: 120,
                  color: cs.surfaceContainerHighest,
                  child:
                      Icon(Icons.person, size: 48, color: cs.onSurfaceVariant),
                ),
              ),
            ),
          ),
        ),
        const SizedBox(height: 10),
        Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.smart_toy_outlined, size: 14, color: cs.primary),
            const SizedBox(width: 4),
            Text(
              portrait.prototypeType == 'photo' ? '照片版' : '校园精灵',
              style: theme.textTheme.labelMedium?.copyWith(color: cs.primary),
            ),
            const SizedBox(width: 8),
            TextButton.icon(
              onPressed: () => _showGenerateDialog(p),
              icon: const Icon(Icons.refresh, size: 14),
              label: const Text('重新生成', style: TextStyle(fontSize: 12)),
              style: TextButton.styleFrom(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 0),
                minimumSize: Size.zero,
                tapTargetSize: MaterialTapTargetSize.shrinkWrap,
              ),
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
  Future<void> _generateFromAvatar(
      TwinPortraitProvider p, String avatarB64) async {
    if (mounted) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('正在用你的头像生成画像…')));
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

  Widget _buildPhotoUploadRow(
      TwinPortraitProvider p, TextEditingController highlightsCtrl) {
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
        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('照片已选择，正在生成画像…')));
      }
      final ok = await p.generate(
        prototypeType: 'photo',
        photoBase64: base64Encode(result.bytes),
        photoMime: result.mime,
      );
      if (!ok && mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(p.error.isNotEmpty ? p.error : '生成失败')));
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text('选择照片失败：$e')));
      }
    }
  }

  Future<void> _generateChaoXing(TwinPortraitProvider p) async {
    if (mounted) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('正在以校园原型生成画像…')));
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

  /// 蔚小芯成长身份卡：画像不是给学生贴标签，而是说明当前阶段与下一步行动。
  Widget _buildGrowthIdentityCard(
      ThemeData theme, dynamic t, StudentFeatureProvider provider) {
    final coverage = (t.dataCoverage as num).toDouble();
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
          side: BorderSide(color: theme.colorScheme.outlineVariant)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Icon(Icons.verified_outlined, color: theme.colorScheme.primary),
            const SizedBox(width: 8),
            Expanded(
                child: Text('画像可信度',
                    style: theme.textTheme.titleMedium
                        ?.copyWith(fontWeight: FontWeight.bold))),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
              decoration: BoxDecoration(
                  color: theme.colorScheme.secondaryContainer,
                  borderRadius: BorderRadius.circular(999)),
              child: Text(t.growthStage.isNotEmpty ? t.growthStage : '在校成长',
                  style: theme.textTheme.labelMedium),
            ),
          ]),
          const SizedBox(height: 8),
          Text('画像基于蔚小芯中的真实记录，不替代心理测评或人工判断。',
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          const SizedBox(height: 12),
          Row(children: [
            Expanded(
                child: LinearProgressIndicator(
                    value: (coverage / 100).clamp(0, 1),
                    minHeight: 8,
                    borderRadius: BorderRadius.circular(8))),
            const SizedBox(width: 10),
            Text('真实数据覆盖 ${coverage.toStringAsFixed(0)}%',
                style: theme.textTheme.labelMedium),
          ]),
          if (t.fallback || t.computedAt.isNotEmpty) ...[
            const SizedBox(height: 8),
            Text(
              t.fallback
                  ? '当前为规则分析结果，数据会随你的记录持续更新。'
                  : '数据已更新 · ${_formatComputedAt(t.computedAt)}',
              style: theme.textTheme.labelSmall
                  ?.copyWith(color: theme.colorScheme.outline),
            ),
          ],
        ]),
      ),
    );
  }

  String _formatComputedAt(String raw) {
    if (raw.isEmpty) return '刚刚';
    return raw.length >= 16 ? raw.substring(0, 16).replaceFirst('T', ' ') : raw;
  }

  /// 综合概览卡片
  Widget _buildOverviewCard(
      ThemeData theme, dynamic t, StudentFeatureProvider provider) {
    final overall = (t.overallScore as num).toDouble();
    final availableCount = t.dimensions.where((d) => d.dataAvailable).length;
    // 无任何真实维度数据时不做伪判断：综合分 0 只是"未计算"，不是"待提升"
    final hasAnyRealData = availableCount > 0;
    final label = !hasAnyRealData
        ? '数据积累中'
        : overall >= 80
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
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            _ScoreRing(score: overall, color: theme.colorScheme.primary),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    _isStaff ? '绩效状态 · $label' : '当前成长状态 · $label',
                    key: const ValueKey('twin-status-headline'),
                    style: theme.textTheme.titleMedium
                        ?.copyWith(fontWeight: FontWeight.bold),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '${Storage.displayName ?? '同学'} · $_roleLabel · $availableCount/${t.dimensions.length} 个维度有记录',
                    style: TextStyle(
                      color: theme.colorScheme.onSurfaceVariant,
                      fontSize: 13,
                    ),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    _isStaff ? '来自真实工作记录与服务学生数据' : '这是成长参考，不是给你贴上的标签',
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
            TabBar(
              tabs: [
                Tab(icon: Icon(Icons.radar, size: 20), text: '能力雷达'),
                Tab(icon: Icon(Icons.psychology, size: 20), text: 'AI 分析'),
                Tab(
                    icon: Icon(Icons.lightbulb_outline, size: 20),
                    text: '成长建议'),
              ],
            ),
            SizedBox(
              height: 440,
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
                                  Expanded(child: Text(d.name)),
                                  _DataStatusChip(label: '数据积累中', muted: true),
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
                                  if (d.evidence.isNotEmpty) ...[
                                    const SizedBox(height: 4),
                                    Text(d.evidence.join(' · '),
                                        style: theme.textTheme.labelSmall
                                            ?.copyWith(
                                                color:
                                                    theme.colorScheme.outline)),
                                  ],
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
                                  crossAxisAlignment: CrossAxisAlignment.start,
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
                                    backgroundColor:
                                        theme.colorScheme.secondaryContainer,
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
        // 无数据维度不画成 0 分拉向圆心（伪短板），改在网格外圈画空心圆提示
        availability: dimensions.map((d) => d.dataAvailable == true).toList(),
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
  /// 与 labels 对齐的可用性标记；false 的维度不参与数据多边形（避免伪 0 分）
  final List<bool> availability;
  final Color color;
  final Color secondaryColor;

  _RadarChartPainter({
    required this.labels,
    required this.values,
    required this.idealValues,
    this.availability = const [],
    required this.color,
    required this.secondaryColor,
  });

  bool _available(int i) =>
      i < availability.length ? availability[i] : true;

  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height / 2);
    final radius = min(size.width, size.height) / 2 - 30;
    final n = labels.length;
    if (n == 0) return;

    final angleStep = 2 * pi / n;
    // 从顶部开始 (-pi/2)
    const startAngle = -pi / 2;

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

    // 绘制实际值（填充 + 描边）：仅包含有真实数据的维度，
    // 无数据维度跳过并在外圈画空心圆提示「数据积累中」，不生成伪 0 分。
    final dataPath = Path();
    var hasDataPoint = false;
    for (int i = 0; i < n; i++) {
      if (!_available(i)) continue;
      final angle = startAngle + i * angleStep;
      final value = values[i].clamp(0.0, 1.0);
      final vRadius = radius * value;
      final x = center.dx + vRadius * cos(angle);
      final y = center.dy + vRadius * sin(angle);
      if (!hasDataPoint) {
        dataPath.moveTo(x, y);
        hasDataPoint = true;
      } else {
        dataPath.lineTo(x, y);
      }
    }

    if (hasDataPoint && availability.where((a) => a).length >= 3) {
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
    } else if (hasDataPoint) {
      // 可用维度不足 3 个时不画多边形（视觉误导），只画数据点
      canvas.drawPath(
        dataPath,
        Paint()
          ..color = color
          ..style = PaintingStyle.stroke
          ..strokeWidth = 2.5,
      );
    }

    // 数据点 + 无数据维度的「数据积累中」提示
    for (int i = 0; i < n; i++) {
      final angle = startAngle + i * angleStep;
      if (!_available(i)) {
        // 空心圆画在最大半径处，表示该维度尚无数据，而非 0 分
        canvas.drawCircle(
          Offset(center.dx + radius * cos(angle),
              center.dy + radius * sin(angle)),
          4,
          Paint()
            ..color = Colors.grey.withOpacity(0.5)
            ..style = PaintingStyle.stroke
            ..strokeWidth = 1.5,
        );
        continue;
      }
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

    // 标签（维度多时自适应：更多维度 → 更小字号 + 稍外移半径，避免重叠）
    final labelFontSize = n >= 8 ? 9.0 : (n >= 6 ? 10.5 : 12.0);
    final labelRadius = radius + (n >= 8 ? 30 : 22);
    for (int i = 0; i < n; i++) {
      final angle = startAngle + i * angleStep;
      final x = center.dx + labelRadius * cos(angle);
      final y = center.dy + labelRadius * sin(angle);

      final textPainter = TextPainter(
        text: TextSpan(
          text: labels[i],
          style: TextStyle(
            color: Colors.grey.shade700,
            fontSize: labelFontSize,
            fontWeight: FontWeight.w500,
          ),
        ),
        textDirection: TextDirection.ltr,
        textAlign: TextAlign.center,
      );
      textPainter.layout(maxWidth: 72);
      textPainter.paint(
        canvas,
        Offset(x - textPainter.width / 2, y - textPainter.height / 2),
      );
    }
  }

  @override
  bool shouldRepaint(covariant _RadarChartPainter oldDelegate) {
    return values != oldDelegate.values ||
        idealValues != oldDelegate.idealValues ||
        availability != oldDelegate.availability;
  }
}

class _ScoreRing extends StatelessWidget {
  final double score;
  final Color color;

  const _ScoreRing({required this.score, required this.color});

  @override
  Widget build(BuildContext context) {
    final value = score.clamp(0.0, 100.0) / 100;
    return SizedBox(
      width: 78,
      height: 78,
      child: Stack(
        alignment: Alignment.center,
        children: [
          CircularProgressIndicator(
            value: value,
            strokeWidth: 8,
            backgroundColor: color.withOpacity(0.12),
            valueColor: AlwaysStoppedAnimation(color),
          ),
          Text(
            score > 0 ? score.toStringAsFixed(0) : '—',
            style: Theme.of(context).textTheme.titleLarge?.copyWith(
                  fontWeight: FontWeight.w800,
                  color: color,
                ),
          ),
        ],
      ),
    );
  }
}

class _DataStatusChip extends StatelessWidget {
  final String label;
  final bool muted;

  const _DataStatusChip({required this.label, this.muted = false});

  @override
  Widget build(BuildContext context) {
    final color = muted
        ? Theme.of(context).colorScheme.outline
        : Theme.of(context).colorScheme.primary;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: color.withOpacity(0.10),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(label,
          style:
              Theme.of(context).textTheme.labelSmall?.copyWith(color: color)),
    );
  }
}
