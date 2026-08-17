import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';
import '../../utils/api_error.dart';
import '../../widgets/data_src_badge.dart';

/// 学院管理员 - 数字孪生大屏
///
/// 概览(4 卡) + 五维全院聚合（雷达不强制，用横向条更强可读）+ AI 解读。
/// 五维聚合字段来自后端 `five_dim`（每维含 score/sample_count/data_source），
/// 诚实渲染：0 样本维显示「数据积累中」，绝不显示编造均值。
class TwinScreenPage extends StatefulWidget {
  const TwinScreenPage({super.key});
  @override
  State<TwinScreenPage> createState() => _TwinScreenPageState();
}

class _TwinScreenPageState extends State<TwinScreenPage> {
  final ApiService _api = ApiService();
  bool _loading = false;
  Map<String, dynamic>? _result;
  String _error = '';

  /// 可选下钻过滤（P1 留位，不带则统计全院）。
  Future<void> _fetch({String? major, String? className}) async {
    setState(() { _loading = true; _error = ''; });
    try {
      final query = <String, String>{
        if (major != null && major.isNotEmpty) 'major': major,
        if (className != null && className.isNotEmpty) 'class': className,
      };
      final res = query.isEmpty
          ? await _api.get(ApiConfig.collegeTwinScreen)
          : await _api.get(ApiConfig.collegeTwinScreen, params: query);
      if (res.statusCode == 200 && res.data != null) {
        setState(() => _result = res.data is Map<String, dynamic> ? res.data : {});
      }
    } catch (e) {
      setState(() => _error = friendlyApiError(e));
    } finally {
      setState(() => _loading = false);
    }
  }

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _fetch().catchError((Object e) {
        if (mounted) setState(() => _error = friendlyApiError(e));
      });
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('数字孪生大屏')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error.isNotEmpty
              ? Center(child: Text(_error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(theme),
    );
  }

  Widget _buildContent(ThemeData theme) {
    if (_result == null) return const Center(child: Text('暂无数据'));
    final overview = (_result!['overview'] as Map?)?.cast<String, dynamic>() ?? {};
    final aiInsight = (_result!['ai_insight'] ?? '').toString();
    final metricCards = <Map<String, String>>[
      {'value': '${overview['total_students'] ?? 0}', 'label': '学生总数'},
      {'value': '${overview['health_score'] ?? 0}', 'label': '健康度'},
      {'value': '${overview['risk_students'] ?? 0}', 'label': '风险关注'},
      {'value': '${((overview['active_rate'] ?? 0) as num? ?? 0 * 100).toStringAsFixed(0)}%', 'label': '健康率'},
    ];
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          child: Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(12),
              gradient: LinearGradient(colors: [theme.colorScheme.primary.withOpacity(0.15), theme.colorScheme.tertiary.withOpacity(0.05)]),
            ),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(Icons.dashboard, color: theme.colorScheme.primary, size: 28),
                const SizedBox(width: 8),
                Text(_result!['college'] ?? '学院', style: theme.textTheme.titleLarge?.copyWith(fontWeight: FontWeight.bold)),
              ]),
              const SizedBox(height: 4),
              Text('更新时间：${_result!['updated_at'] ?? ''}', style: theme.textTheme.bodySmall),
            ]),
          ),
        ),
        const SizedBox(height: 16),
        GridView.count(
          crossAxisCount: 2, shrinkWrap: true, physics: const NeverScrollableScrollPhysics(),
          mainAxisSpacing: 12, crossAxisSpacing: 12, childAspectRatio: 1.6,
          children: metricCards.map<Widget>((m) => Card(
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
                Text(m['value']!, style: theme.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold, color: theme.colorScheme.primary)),
                const SizedBox(height: 4),
                Text(m['label']!, style: theme.textTheme.bodySmall),
              ]),
            ),
          )).toList(),
        ),
        const SizedBox(height: 16),
        ..._buildFiveDimSection(theme),
        if (aiInsight.isNotEmpty) ...[
          const SizedBox(height: 16),
          Card(
            color: theme.colorScheme.primaryContainer,
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Row(children: [
                  Icon(Icons.psychology, color: theme.colorScheme.onPrimaryContainer, size: 18),
                  const SizedBox(width: 6),
                  Text('AI 解读', style: theme.textTheme.titleSmall),
                ]),
                const SizedBox(height: 8),
                Text(aiInsight, style: theme.textTheme.bodyMedium),
              ]),
            ),
          ),
        ],
      ],
    );
  }

  /// 五维全院聚合区块（增量追加，不破坏既有 4 卡与 AI 解读）。
  ///
  /// 每维：名称 + 分数横向条 + sample_count + data_src_badge；0 样本维显示「数据积累中」。
  /// five_dim==null 或 sample_count==0 → 显示「五维画像数据积累中」空态，不硬画 0 分。
  List<Widget> _buildFiveDimSection(ThemeData theme) {
    final f = (_result!['five_dim'] as Map?)?.cast<String, dynamic>();
    final dimensions = (f?['dimensions'] as List?) ?? [];
    final sampleCount = (f?['sample_count'] as num?)?.toInt() ?? 0;
    final trendNote = (f?['trend_note'] ?? '').toString();

    // 空态：无五维结构或整体 0 样本
    if (f == null || dimensions.isEmpty || sampleCount == 0) {
      return [
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(Icons.radar, color: theme.colorScheme.primary, size: 20),
                const SizedBox(width: 6),
                Text('五维全院画像', style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.bold)),
              ]),
              const SizedBox(height: 12),
              Text('五维画像数据积累中：完成更多学生画像计算后即可看到全院学术/能力/思想/情感/社交评分。',
                  style: theme.textTheme.bodyMedium?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
              const SizedBox(height: 4),
              Text(trendNote.isNotEmpty ? trendNote : '趋势数据积累中', style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.outline)),
            ]),
          ),
        ),
      ];
    }

    return [
      Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Row(children: [
              Icon(Icons.radar, color: theme.colorScheme.primary, size: 20),
              const SizedBox(width: 6),
              Text('五维全院画像', style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.bold)),
              const Spacer(),
              Text('全院 $sampleCount 名有快照学生', style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.outline)),
            ]),
            const SizedBox(height: 12),
            ...dimensions.map<Widget>((d) => _buildDimRow(theme, d as Map)),
            if (trendNote.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(trendNote, style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.outline)),
            ],
          ]),
        ),
      ),
    ];
  }

  Widget _buildDimRow(ThemeData theme, Map d) {
    final name = (d['name'] ?? '').toString();
    final dataSource = (d['data_source'] ?? 'not_available').toString();
    final sampleCount = (d['sample_count'] as num?)?.toInt() ?? 0;
    final scoreValue = d['score'] is num ? (d['score'] as num).toDouble() : null;
    // 诚实二态（M1 fix）：score==null 才是「无样本/数据积累中」；
    // score==0.0 是「有样本但均值恰为 0」的真实 0 分，应渲染 0.0 + real badge（后端 REAL NOT NULL DEFAULT 0）。
    final hasScore = scoreValue != null;

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          SizedBox(width: 48, child: Text(name, style: theme.textTheme.bodyMedium?.copyWith(fontWeight: FontWeight.w600))),
          Expanded(
            child: hasScore
                ? ClipRRect(
                    borderRadius: BorderRadius.circular(6),
                    child: LinearProgressIndicator(
                      value: scoreValue.clamp(0.0, 100.0).toDouble() / 100,
                      minHeight: 8,
                      backgroundColor: theme.colorScheme.surfaceContainerHighest,
                    ),
                  )
                : Container(
                    height: 8,
                    width: double.infinity,
                    decoration: BoxDecoration(
                      color: theme.colorScheme.surfaceContainerHighest,
                      borderRadius: BorderRadius.circular(6),
                    ),
                  ),
          ),
          const SizedBox(width: 8),
          SizedBox(
            width: 44,
            child: Text(
              hasScore ? scoreValue.toStringAsFixed(1) : '—',
              textAlign: TextAlign.right,
              style: theme.textTheme.bodyMedium?.copyWith(fontWeight: FontWeight.bold),
            ),
          ),
          const SizedBox(width: 4),
          DataSrcBadge(src: dataSource == 'real' ? 'real' : null),
        ]),
        Row(children: [
          const SizedBox(width: 48),
          Expanded(
            child: Text(
              hasScore
                  ? '样本 $sampleCount'
                  : '数据积累中（0 样本，不显示伪均值）',
              style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.outline),
            ),
          ),
        ]),
      ]),
    );
  }
}
