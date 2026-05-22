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

  // ── 我的反馈（学生端只读视图，独立分页与状态）──
  final List<FeedbackEntry> _myFeedbacks = [];
  bool _myLoading = false;
  int _myPage = 1;
  int _myTotal = 0;
  String _myStatusFilter = '';

  List<FeedbackEntry> get feedbacks => _feedbacks;
  bool get loading => _loading;
  int get total => _total;
  String get statusFilter => _statusFilter;
  String get error => _error;

  List<FeedbackEntry> get myFeedbacks => _myFeedbacks;
  bool get myLoading => _myLoading;
  int get myTotal => _myTotal;
  String get myStatusFilter => _myStatusFilter;

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
    if (_statusFilter == status) return;
    _statusFilter = status;
    fetchFeedbacks(refresh: true);
  }

  /// 我的反馈：分页拉取当前用户提交过的反馈（含 admin 回复与状态）
  Future<void> fetchMyFeedbacks({bool refresh = false}) async {
    if (_myLoading) return;
    if (refresh) {
      _myPage = 1;
      _myFeedbacks.clear();
    }
    _myLoading = true;
    notifyListeners();

    try {
      final params = <String, dynamic>{
        'page': _myPage,
        'page_size': 20,
      };
      if (_myStatusFilter.isNotEmpty) params['status'] = _myStatusFilter;

      final response = await _api.get(ApiConfig.feedbackMine, params: params);
      if (response.data['code'] == 0) {
        final list = (response.data['data'] as List?)
                ?.map((e) =>
                    FeedbackEntry.fromJson(e as Map<String, dynamic>))
                .toList() ??
            [];
        _myFeedbacks.addAll(list);
        _myTotal = response.data['total'] ?? 0;
        _myPage++;
      }
    } catch (e) {
      _error = '获取我的反馈失败: $e';
    } finally {
      _myLoading = false;
      notifyListeners();
    }
  }

  void setMyStatusFilter(String status) {
    if (_myStatusFilter == status) return;
    _myStatusFilter = status;
    fetchMyFeedbacks(refresh: true);
  }

  /// 上传截图，返回文件 URL（上传成功后前端用于回填）
  Future<String?> uploadScreenshot(String filePath) async {
    try {
      final response = await _api.upload(
        ApiConfig.feedbackScreenshot,
        filePath: filePath,
        fieldName: 'file',
      );
      if (response.data['code'] == 0) {
        return response.data['data']?['url'] as String?;
      }
      _error = response.data['message'] ?? '上传截图失败';
      notifyListeners();
      return null;
    } catch (e) {
      _error = '截图上传网络错误: $e';
      notifyListeners();
      return null;
    }
  }

  /// 通过 bytes 上传截图（Web 端）
  Future<String?> uploadScreenshotBytes(List<int> bytes, String filename) async {
    try {
      final response = await _api.uploadBytes(
        ApiConfig.feedbackScreenshot,
        bytes: bytes,
        filename: filename,
        fieldName: 'file',
      );
      if (response.data['code'] == 0) {
        return response.data['data']?['url'] as String?;
      }
      _error = response.data['message'] ?? '上传截图失败';
      notifyListeners();
      return null;
    } catch (e) {
      _error = '截图上传网络错误: $e';
      notifyListeners();
      return null;
    }
  }

  Future<bool> submitFeedback({
    required String category,
    required String content,
    String messageId = '',
    String resourceId = '',
    String screenshotUrl = '',
  }) async {
    try {
      final response = await _api.post(ApiConfig.feedback, data: {
        'category': category,
        'content': content,
        'message_id': messageId,
        'resource_id': resourceId,
        'screenshot_url': screenshotUrl,
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

  Future<bool> resolveFeedback(String feedbackId, String status, {String reply = ''}) async {
    try {
      final response = await _api.put(
        ApiConfig.feedbackResolve(feedbackId),
        data: {
          'status': status,
          'reply': reply,
        },
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
