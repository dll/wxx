import 'package:flutter/foundation.dart';
import '../services/api_service.dart';
import '../config/api_config.dart';

/// 问题预案数据模型
class IssueForecast {
  final String forecastId;
  final String title;
  final String summary;
  final String description;
  final String category;
  final String riskLevel;
  final String status;
  final String sources;
  final String evidence;
  final String rootCause;
  final String recommendedAction;
  final String ownerName;
  final String ownerScope;
  final String createdAt;
  final String updatedAt;

  IssueForecast({
    required this.forecastId,
    required this.title,
    required this.summary,
    this.description = '',
    this.category = 'comprehensive',
    this.riskLevel = 'medium',
    this.status = 'pending',
    this.sources = '[]',
    this.evidence = '',
    this.rootCause = '',
    this.recommendedAction = '',
    this.ownerName = '',
    this.ownerScope = '',
    this.createdAt = '',
    this.updatedAt = '',
  });

  factory IssueForecast.fromJson(Map<String, dynamic> json) {
    return IssueForecast(
      forecastId: json['forecast_id'] ?? '',
      title: json['title'] ?? '',
      summary: json['summary'] ?? '',
      description: json['description'] ?? '',
      category: json['category'] ?? 'comprehensive',
      riskLevel: json['risk_level'] ?? 'medium',
      status: json['status'] ?? 'pending',
      sources: json['sources'] ?? '[]',
      evidence: json['evidence'] ?? '',
      rootCause: json['root_cause'] ?? '',
      recommendedAction: json['recommended_action'] ?? '',
      ownerName: json['owner_name'] ?? '',
      ownerScope: json['owner_scope'] ?? '',
      createdAt: json['created_at'] ?? '',
      updatedAt: json['updated_at'] ?? '',
    );
  }

  String get categoryLabel {
    switch (category) {
      case 'emotion':
        return '情感问题';
      case 'process':
        return '流程问题';
      case 'graduation':
        return '毕业问题';
      case 'feedback':
        return '反馈问题';
      default:
        return '综合问题';
    }
  }

  String get riskLevelLabel {
    switch (riskLevel) {
      case 'urgent':
        return '紧急';
      case 'high':
        return '高';
      case 'medium':
        return '中';
      case 'low':
        return '低';
      default:
        return '未知';
    }
  }

  String get statusLabel {
    switch (status) {
      case 'pending':
        return '待处理';
      case 'processing':
        return '处理中';
      case 'resolved':
        return '已解决';
      case 'archived':
        return '已归档';
      default:
        return '未知';
    }
  }
}

/// 问题预案统计
class ForecastStatistics {
  final Map<String, int> riskDistribution;
  final Map<String, int> categoryDistribution;
  final List<Map<String, dynamic>> dailyTrend;

  ForecastStatistics({
    required this.riskDistribution,
    required this.categoryDistribution,
    required this.dailyTrend,
  });
}

/// 问题预案状态管理
class ForecastProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  final List<IssueForecast> _forecasts = [];
  bool _loading = false;
  int _page = 1;
  int _total = 0;
  String _error = '';
  String _categoryFilter = '';
  String _riskLevelFilter = '';
  String _statusFilter = '';
  ForecastStatistics? _statistics;
  bool _analyzing = false;

  List<IssueForecast> get forecasts => _forecasts;
  bool get loading => _loading;
  int get total => _total;
  String get error => _error;
  String get categoryFilter => _categoryFilter;
  String get riskLevelFilter => _riskLevelFilter;
  String get statusFilter => _statusFilter;
  ForecastStatistics? get statistics => _statistics;
  bool get analyzing => _analyzing;

  /// 获取问题预案列表
  Future<void> fetchForecasts({bool refresh = false}) async {
    if (_loading) return;
    if (refresh) {
      _page = 1;
      _forecasts.clear();
    }
    _loading = true;
    notifyListeners();

    try {
      final params = <String, dynamic>{
        'page': _page,
        'page_size': 20,
      };
      if (_categoryFilter.isNotEmpty) params['category'] = _categoryFilter;
      if (_riskLevelFilter.isNotEmpty) params['risk_level'] = _riskLevelFilter;
      if (_statusFilter.isNotEmpty) params['status'] = _statusFilter;

      final response = await _api.get(ApiConfig.forecastIssues, params: params);
      if (response.data['code'] == 0) {
        final list = (response.data['data'] as List?)
                ?.map((e) =>
                    IssueForecast.fromJson(e as Map<String, dynamic>))
                .toList() ??
            [];
        _forecasts.addAll(list);
        _total = response.data['total'] ?? 0;
        _page++;
      }
    } catch (e) {
      _error = '获取问题预案列表失败: $e';
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  /// 执行问题分析
  Future<bool> analyzeIssues({
    String? collegeId,
    String timeRange = 'last_30_days',
    String analysisType = 'comprehensive',
  }) async {
    if (_analyzing) return false;
    _analyzing = true;
    _error = '';
    notifyListeners();

    try {
      final response = await _api.post(ApiConfig.forecastAnalysis, data: {
        'college_id': collegeId ?? '',
        'time_range': timeRange,
        'analysis_type': analysisType,
      });
      if (response.data['code'] == 0) {
        await fetchForecasts(refresh: true);
        return true;
      }
      _error = response.data['message'] ?? '分析失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    } finally {
      _analyzing = false;
      notifyListeners();
    }
  }

  /// 获取问题预案详情
  Future<IssueForecast?> getForecast(String forecastId) async {
    try {
      final response = await _api.get(ApiConfig.forecastIssueDetail(forecastId));
      if (response.data['code'] == 0) {
        return IssueForecast.fromJson(response.data['data']);
      }
      _error = response.data['message'] ?? '获取详情失败';
      notifyListeners();
      return null;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return null;
    }
  }

  /// 更新问题预案状态
  Future<bool> updateStatus(String forecastId, String status) async {
    try {
      final response = await _api.put(
        ApiConfig.forecastIssueStatus(forecastId),
        data: {'status': status},
      );
      if (response.data['code'] == 0) {
        await fetchForecasts(refresh: true);
        return true;
      }
      _error = response.data['message'] ?? '更新状态失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  /// 获取统计数据
  Future<void> fetchStatistics({int days = 30}) async {
    try {
      final response = await _api.get(
        ApiConfig.forecastStatistics,
        params: {'days': days},
      );
      if (response.data['code'] == 0) {
        final data = response.data['data'];
        _statistics = ForecastStatistics(
          riskDistribution: Map<String, int>.from(data['risk_distribution'] ?? {}),
          categoryDistribution: Map<String, int>.from(data['category_distribution'] ?? {}),
          dailyTrend: List<Map<String, dynamic>>.from(data['daily_trend'] ?? []),
        );
        notifyListeners();
      }
    } catch (e) {
      _error = '获取统计数据失败: $e';
    }
  }

  /// 设置筛选条件
  void setCategoryFilter(String category) {
    if (_categoryFilter == category) return;
    _categoryFilter = category;
    fetchForecasts(refresh: true);
  }

  void setRiskLevelFilter(String riskLevel) {
    if (_riskLevelFilter == riskLevel) return;
    _riskLevelFilter = riskLevel;
    fetchForecasts(refresh: true);
  }

  void setStatusFilter(String status) {
    if (_statusFilter == status) return;
    _statusFilter = status;
    fetchForecasts(refresh: true);
  }
}
