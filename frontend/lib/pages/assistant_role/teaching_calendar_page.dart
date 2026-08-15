import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';
import '../../utils/api_error.dart';
import '../../widgets/data_src_badge.dart';

/// 教辅 - 教学日历（学期关键节点安排）
class TeachingCalendarPage extends StatefulWidget {
  const TeachingCalendarPage({super.key});
  @override
  State<TeachingCalendarPage> createState() => _TeachingCalendarPageState();
}

class _TeachingCalendarPageState extends State<TeachingCalendarPage> {
  final ApiService _api = ApiService();
  bool _loading = false;
  Map<String, dynamic>? _result;
  String _error = '';

  Future<void> _fetch() async {
    setState(() {
      _loading = true;
      _error = '';
    });
    try {
      final res = await _api.get(ApiConfig.assistantTeachingCalendar);
      if (res.statusCode == 200 && res.data != null) {
        setState(() => _result =
            res.data is Map<String, dynamic> ? res.data : {});
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
      appBar: AppBar(title: const Text('教学日历')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error.isNotEmpty
              ? Center(
                  child:
                      Text(_error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(theme),
    );
  }

  Widget _buildContent(ThemeData theme) {
    if (_result == null) return const Center(child: Text('暂无数据'));
    final keyDates =
        (_result!['key_dates'] as List?)?.cast<Map<String, dynamic>>() ?? [];
    final suggestions =
        (_result!['suggestions'] as List?)?.cast<String>() ?? [];
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        DataSrcBadge(src: _result?['data_source']),
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(Icons.calendar_month, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text('${_result!['semester'] ?? '本学期'} 关键节点',
                    style: theme.textTheme.titleMedium
                        ?.copyWith(fontWeight: FontWeight.bold)),
              ]),
              const SizedBox(height: 4),
              Text('教辅工作日程参考（真实教务安排接入后自动更新）',
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: theme.colorScheme.outline)),
            ]),
          ),
        ),
        const SizedBox(height: 12),
        ...keyDates.map((d) => Card(
              margin: const EdgeInsets.only(bottom: 8),
              child: ListTile(
                leading: Icon(_iconForType(d['type']?.toString() ?? ''),
                    color: _colorForType(d['type']?.toString() ?? '')),
                title: Text(d['event']?.toString() ?? ''),
                subtitle: Text('${d['date']} · ${d['type']}'),
                trailing: d['remind'] == true
                    ? const Chip(
                        label: Text('提醒', style: TextStyle(fontSize: 11)),
                        visualDensity: VisualDensity.compact,
                      )
                    : null,
              ),
            )),
        if (suggestions.isNotEmpty) ...[
          const SizedBox(height: 8),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('工作建议',
                    style: theme.textTheme.titleSmall
                        ?.copyWith(fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                ...suggestions.map((s) => Padding(
                      padding: const EdgeInsets.symmetric(vertical: 4),
                      child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        Icon(Icons.check_circle_outline,
                            size: 16, color: theme.colorScheme.primary),
                        const SizedBox(width: 8),
                        Expanded(child: Text(s)),
                      ]),
                    )),
              ]),
            ),
          ),
        ],
      ],
    );
  }

  IconData _iconForType(String type) {
    switch (type) {
      case '考试':
        return Icons.quiz_outlined;
      case 'deadline':
        return Icons.flag_outlined;
      default:
        return Icons.event_note;
    }
  }

  Color _colorForType(String type) {
    switch (type) {
      case '考试':
        return Colors.orange;
      case 'deadline':
        return Colors.redAccent;
      default:
        return Colors.blueGrey;
    }
  }
}
