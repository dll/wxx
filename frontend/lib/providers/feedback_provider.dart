import 'dart:convert';
import 'package:dio/dio.dart';
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

  // ── 统计数据 ──
  FeedbackStats? _stats;
  bool _statsLoading = false;

  // ── 当前选中的反馈详情 ──
  FeedbackEntry? _currentFeedback;
  bool _detailLoading = false;
  List<FeedbackLog> _logs = [];
  bool _logsLoading = false;

  List<FeedbackEntry> get feedbacks => _feedbacks;
  bool get loading => _loading;
  int get total => _total;
  String get statusFilter => _statusFilter;
  String get error => _error;

  List<FeedbackEntry> get myFeedbacks => _myFeedbacks;
  bool get myLoading => _myLoading;
  int get myTotal => _myTotal;
  String get myStatusFilter => _myStatusFilter;

  FeedbackStats? get stats => _stats;
  bool get statsLoading => _statsLoading;

  FeedbackEntry? get currentFeedback => _currentFeedback;
  bool get detailLoading => _detailLoading;
  List<FeedbackLog> get logs => _logs;
  bool get logsLoading => _logsLoading;

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
                ?.map((e) => FeedbackEntry.fromJson(e as Map<String, dynamic>))
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
                ?.map((e) => FeedbackEntry.fromJson(e as Map<String, dynamic>))
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

  /// 获取反馈详情
  Future<FeedbackEntry?> fetchFeedbackDetail(String feedbackId) async {
    _detailLoading = true;
    _currentFeedback = null;
    notifyListeners();

    try {
      final response = await _api.get(ApiConfig.feedbackDetail(feedbackId));
      if (response.data['code'] == 0) {
        final data = response.data['data'] as Map<String, dynamic>;
        _currentFeedback = FeedbackEntry.fromJson(data);
        return _currentFeedback;
      }
    } catch (e) {
      _error = '获取反馈详情失败: $e';
    } finally {
      _detailLoading = false;
      notifyListeners();
    }
    return null;
  }

  /// 获取反馈处理记录
  Future<void> fetchFeedbackLogs(String feedbackId) async {
    _logsLoading = true;
    _logs = [];
    notifyListeners();

    try {
      final response = await _api.get(ApiConfig.feedbackLogs(feedbackId));
      if (response.data['code'] == 0) {
        final list = (response.data['data'] as List?)
                ?.map((e) => FeedbackLog.fromJson(e as Map<String, dynamic>))
                .toList() ??
            [];
        _logs = list;
      }
    } catch (e) {
      _error = '获取处理记录失败: $e';
    } finally {
      _logsLoading = false;
      notifyListeners();
    }
  }

  /// 获取反馈统计
  Future<void> fetchStats() async {
    _statsLoading = true;
    notifyListeners();

    try {
      final response = await _api.get(ApiConfig.adminFeedbackStats);
      if (response.data['code'] == 0) {
        final data = response.data['data'] as Map<String, dynamic>;
        _stats = FeedbackStats.fromJson(data);
      }
    } catch (e) {
      _error = '获取统计数据失败: $e';
    } finally {
      _statsLoading = false;
      notifyListeners();
    }
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
  ///
  /// 实现说明：Vercel serverless 多实例场景下，文件系统 / SQLite 都无法跨实例可靠持久化，
  /// 因此把截图直接 base64 编码为 data: URL，与反馈记录一起入库（feedback.screenshot_url）。
  /// 这样反馈记录走任何实例的 DB 副本都能完整渲染图片，不再依赖单独的资源接口。
  Future<String?> uploadScreenshotBytes(
      List<int> bytes, String filename) async {
    try {
      if (bytes.isEmpty) return null;
      if (bytes.length > 900 * 1024) {
        _error = '截图体积过大，已改为仅提交文字反馈';
        notifyListeners();
        return null;
      }
      final base64Str = base64Encode(bytes);
      final ext = filename.split('.').last.toLowerCase();
      final mime = (ext == 'jpg' || ext == 'jpeg')
          ? 'image/jpeg'
          : (ext == 'gif')
              ? 'image/gif'
              : (ext == 'webp')
                  ? 'image/webp'
                  : 'image/png';
      return 'data:$mime;base64,$base64Str';
    } catch (e) {
      _error = '截图编码失败: $e';
      notifyListeners();
      return null;
    }
  }

  Future<bool> submitFeedback({
    required String category,
    required String content,
    String messageId = '',
    String resourceId = '',
    String module = '',
    String screenshotUrl = '',
  }) async {
    try {
      final response = await _api.post(ApiConfig.feedback, data: {
        'category': category,
        'content': content,
        'message_id': messageId,
        'resource_id': resourceId,
        'module': module,
        'screenshot_url': screenshotUrl,
      });
      if (response.data['code'] == 0) {
        return true;
      }
      _error = response.data['message'] ?? '提交失败';
      notifyListeners();
      return false;
    } on DioException catch (e) {
      // 401 由 ApiService 拦截器统一处理（退出登录并跳转登录页）
      if (e.response?.statusCode == 401) {
        _error = '登录状态已失效，请重新登录';
        notifyListeners();
        return false;
      }
      // 优先使用服务端返回的业务错误信息，便于定位 500 等后端问题
      final serverMessage = e.response?.data is Map
          ? (e.response?.data as Map)['message']?.toString()
          : null;
      _error = serverMessage?.isNotEmpty == true
          ? serverMessage!
          : _friendlyRequestError(e, '提交反馈');
      notifyListeners();
      return false;
    } catch (_) {
      _error = '提交反馈失败，请稍后重试';
      notifyListeners();
      return false;
    }
  }

  Future<bool> resolveFeedback(String feedbackId, String status,
      {String reply = ''}) async {
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

  /// 满意度评价
  Future<bool> rateFeedback(String feedbackId, int rating,
      {String comment = ''}) async {
    try {
      final response = await _api.put(
        ApiConfig.feedbackRate(feedbackId),
        data: {
          'rating': rating,
          'comment': comment,
        },
      );
      if (response.data['code'] == 0) {
        // 刷新详情和列表
        if (_currentFeedback?.feedbackId == feedbackId) {
          await fetchFeedbackDetail(feedbackId);
        }
        await fetchMyFeedbacks(refresh: true);
        return true;
      }
      _error = response.data['message'] ?? '评价失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  /// 关联知识资源
  Future<bool> linkResource(String feedbackId, String resourceId,
      {String note = ''}) async {
    try {
      final response = await _api.put(
        ApiConfig.adminFeedbackLinkResource(feedbackId),
        data: {
          'resource_id': resourceId,
          'note': note,
        },
      );
      if (response.data['code'] == 0) {
        if (_currentFeedback?.feedbackId == feedbackId) {
          await fetchFeedbackDetail(feedbackId);
        }
        await fetchFeedbacks(refresh: true);
        return true;
      }
      _error = response.data['message'] ?? '关联失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  /// AI 在线修复诊断（管理端）：解析截图 + 定位模块与代码文件。
  /// 返回 null 表示调用失败（面板保留本地关键词定位）。
  Future<AIRepairResult?> aiRepair(String feedbackId) async {
    try {
      final response = await _api
          .post(ApiConfig.feedbackAIRepair(feedbackId), data: <String, dynamic>{});
      if (response.data['code'] == 0 &&
          response.data['data'] is Map<String, dynamic>) {
        return AIRepairResult.fromJson(
            response.data['data'] as Map<String, dynamic>);
      }
      _error = response.data['message'] ?? 'AI 诊断失败';
      notifyListeners();
      return null;
    } catch (e) {
      _error = 'AI 诊断请求失败: $e';
      notifyListeners();
      return null;
    }
  }

  String _friendlyRequestError(DioException error, String action) {
    switch (error.type) {
      case DioExceptionType.connectionTimeout:
      case DioExceptionType.sendTimeout:
      case DioExceptionType.receiveTimeout:
        return '$action超时，请稍后重试';
      case DioExceptionType.connectionError:
        return '暂时无法连接服务，请检查网络后重试';
      default:
        final status = error.response?.statusCode;
        if (status == 502 || status == 503 || status == 504) {
          return '服务暂时不可用，请稍后重试';
        }
        return '$action失败，请稍后重试';
    }
  }
}
