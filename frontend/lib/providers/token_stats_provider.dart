import 'package:flutter/foundation.dart';
import '../models/models.dart';
import '../services/api_service.dart';
import '../config/api_config.dart';
import '../utils/capability_utils.dart';

class TokenStatsProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  TokenStatsData? _myStats;
  List<SubordinateTokenStats> _subordinateStats = [];
  bool _loading = false;
  int _days = 30;
  String _error = '';

  TokenStatsData? get myStats => _myStats;
  List<SubordinateTokenStats> get subordinateStats => _subordinateStats;
  bool get loading => _loading;
  int get days => _days;
  String get error => _error;

  void setDays(int d) {
    _days = d;
    fetchAll();
  }

  Future<void> fetchAll() async {
    if (_loading) return;
    _loading = true;
    _error = '';
    notifyListeners();

    try {
      // 个人统计：所有人都能看；下级统计仅 counselor 及以上可见
      final tasks = <Future<void>>[fetchMyStats()];
      if (CapabilityUtils.has(Capability.counselorTokenSubordinates)) {
        tasks.add(fetchSubordinateStats());
      }
      await Future.wait(tasks);
    } catch (e) {
      _error = '获取词元统计失败: $e';
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> fetchMyStats() async {
    if (!CapabilityUtils.has(Capability.selfTokenStats)) return;
    try {
      final response = await _api.get(ApiConfig.tokenStatsMy, params: {'days': _days});
      if (response.data['code'] == 0 && response.data['data'] != null) {
        _myStats = TokenStatsData.fromJson(response.data['data'] as Map<String, dynamic>);
      }
    } catch (e) {
      _error = '获取个人词元统计失败: $e';
    }
    notifyListeners();
  }

  Future<void> fetchSubordinateStats() async {
    if (!CapabilityUtils.has(Capability.counselorTokenSubordinates)) return;
    try {
      final response = await _api.get(ApiConfig.tokenStatsSubordinates, params: {'days': _days});
      if (response.data['code'] == 0 && response.data['data'] != null) {
        _subordinateStats = (response.data['data'] as List)
            .map((e) => SubordinateTokenStats.fromJson(e as Map<String, dynamic>))
            .toList();
      }
    } catch (e) {
      // 下级统计获取失败不影响个人统计
    }
    notifyListeners();
  }
}
