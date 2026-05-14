import 'package:flutter/foundation.dart';
import '../models/models.dart';
import '../services/api_service.dart';
import '../config/api_config.dart';

/// 用户反馈状态管理
class FeedbackProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  final List<FeedbackEntry> _feedbacks = [];
  bool _loading = false;
  int _page = 1;
  int _total = 0;
  String _statusFilter = '';
  String _error = '';

  List<FeedbackEntry> get feedbacks => _feedbacks;
  bool get loading => _loading;
  int get total => _total;
  String get statusFilter => _statusFilter;
  String get error => _error;

  Future<void> fetchFeedbacks({bool refresh = false}) async {
    if (_loading) return;
    if (refresh) {
      _page = 1;
      _feedbacks.clear();
    }
    _loading = true;
    notifyListeners();

    try {
      final params = <String, dynamic>{
        'page': _page,
        'page_size': 20,
      };
      if (_statusFilter.isNotEmpty) params['status'] = _statusFilter;

      final response = await _api.get(ApiConfig.feedback, params: params);
      if (response.data['code'] == 0) {
        final list = (response.data['data'] as List?)
                ?.map((e) =>
                    FeedbackEntry.fromJson(e as Map<String, dynamic>))
                .toList() ??
            [];
        _feedbacks.addAll(list);
        _total = response.data['total'] ?? 0;
        _page++;
      }
    } catch (e) {
      _error = '获取反馈列表失败: $e';
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  void setStatusFilter(String status) {
    _statusFilter = status;
    fetchFeedbacks(refresh: true);
  }

  Future<bool> submitFeedback({
    required String category,
    required String content,
    String messageId = '',
    String resourceId = '',
  }) async {
    try {
      final response = await _api.post(ApiConfig.feedback, data: {
        'category': category,
        'content': content,
        'message_id': messageId,
        'resource_id': resourceId,
      });
      if (response.data['code'] == 0) {
        return true;
      }
      _error = response.data['message'] ?? '提交失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  Future<bool> resolveFeedback(String feedbackId, String status) async {
    try {
      final response = await _api.put(
        ApiConfig.feedbackResolve(feedbackId),
        data: {'status': status},
      );
      if (response.data['code'] == 0) {
        await fetchFeedbacks(refresh: true);
        return true;
      }
      _error = response.data['message'] ?? '处理失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }
}
