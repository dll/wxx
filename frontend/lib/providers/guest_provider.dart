import 'package:flutter/foundation.dart';
import '../models/models.dart';
import '../services/api_service.dart';
import '../config/api_config.dart';

/// 游客审核状态管理
class GuestProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  final List<UserProfile> _pendingGuests = [];
  bool _loading = false;
  String _error = '';

  List<UserProfile> get pendingGuests => _pendingGuests;
  bool get loading => _loading;
  String get error => _error;

  Future<void> fetchPendingGuests() async {
    if (_loading) return;
    _loading = true;
    notifyListeners();

    try {
      final response = await _api.get(ApiConfig.adminGuestsPending);
      if (response.data['code'] == 0) {
        final list = (response.data['data'] as List?)
                ?.whereType<Map>()
                .map((e) => UserProfile.fromJson({'data': Map<String, dynamic>.from(e)}))
                .toList() ??
            [];
        _pendingGuests
          ..clear()
          ..addAll(list);
      }
    } catch (e) {
      _error = '获取待审核游客失败: $e';
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<bool> approveGuest(String id, {String studentId = ''}) async {
    try {
      final data = <String, dynamic>{};
      if (studentId.isNotEmpty) data['student_id'] = studentId;
      final response = await _api.put(ApiConfig.adminGuestApprove(id), data: data);
      if (response.data['code'] == 0) {
        _pendingGuests.removeWhere((g) => g.id.toString() == id);
        notifyListeners();
        return true;
      }
      _error = response.data['message'] ?? '审核通过失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  Future<bool> rejectGuest(String id) async {
    try {
      final response = await _api.put(ApiConfig.adminGuestReject(id));
      if (response.data['code'] == 0) {
        _pendingGuests.removeWhere((g) => g.id.toString() == id);
        notifyListeners();
        return true;
      }
      _error = response.data['message'] ?? '拒绝失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }
}
