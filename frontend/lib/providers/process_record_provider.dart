import 'package:flutter/foundation.dart';
import '../config/api_config.dart';
import '../models/models.dart';
import '../services/api_service.dart';

/// 办事流程办理记录管理（持久化版）
///
/// 与 EnrollmentProvider 配合使用：EnrollmentProvider 仍负责加载流程指引（步骤/详情），
/// 本 Provider 负责把用户进度持久化到后端，重启/换设备后能恢复。
class ProcessRecordProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  List<ProcessRecord> _records = [];
  bool _loading = false;
  String? _error;

  List<ProcessRecord> get records => List.unmodifiable(_records);
  bool get loading => _loading;
  String? get error => _error;

  /// 找到指定流程类型的当前记录（in_progress 优先，否则最新一条）
  ProcessRecord? recordOf(String flowType) {
    final inProgress = _records
        .where((r) => r.flowType == flowType && r.status == 'in_progress')
        .toList();
    if (inProgress.isNotEmpty) return inProgress.first;
    final any = _records.where((r) => r.flowType == flowType).toList();
    return any.isNotEmpty ? any.first : null;
  }

  /// 拉取我的全部办事记录
  Future<void> fetchAll() async {
    _loading = true;
    _error = null;
    notifyListeners();
    try {
      final resp = await _api.get(ApiConfig.processRecords);
      final list = resp.data['data'] as List? ?? [];
      _records = list.map((e) => ProcessRecord.fromJson(e as Map<String, dynamic>)).toList();
    } catch (e) {
      _error = '加载办事记录失败';
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  /// 开始/恢复某流程，返回当前记录（已存在则复用）
  Future<ProcessRecord?> startOrResume({
    required String flowType,
    String flowLabel = '',
    int totalSteps = 0,
  }) async {
    try {
      final resp = await _api.post(
        ApiConfig.processRecordStart(flowType),
        data: {
          'flow_label': flowLabel,
          'total_steps': totalSteps,
        },
      );
      if (resp.data['code'] == 0 && resp.data['data'] != null) {
        final rec = ProcessRecord.fromJson(resp.data['data']);
        _upsertLocal(rec);
        notifyListeners();
        return rec;
      }
    } catch (e) {
      _error = '启动办事流程失败：$e';
      notifyListeners();
    }
    return null;
  }

  /// 上报进度（已完成步骤集合 + 当前步骤）
  Future<bool> updateProgress({
    required String flowType,
    required int currentStep,
    required List<int> completedSteps,
    String notes = '',
  }) async {
    try {
      final resp = await _api.post(
        ApiConfig.processRecordProgress(flowType),
        data: {
          'current_step': currentStep,
          'completed_steps': completedSteps,
          'notes': notes,
        },
      );
      if (resp.data['code'] == 0 && resp.data['data'] != null) {
        _upsertLocal(ProcessRecord.fromJson(resp.data['data']));
        notifyListeners();
        return true;
      }
    } catch (e) {
      _error = '保存进度失败：$e';
      notifyListeners();
    }
    return false;
  }

  void _upsertLocal(ProcessRecord rec) {
    final idx = _records.indexWhere((r) => r.recordId == rec.recordId);
    if (idx >= 0) {
      _records[idx] = rec;
    } else {
      _records = [rec, ..._records];
    }
  }
}
