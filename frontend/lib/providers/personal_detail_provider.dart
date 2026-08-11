import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

import '../config/api_config.dart';
import '../models/models.dart';
import '../services/api_service.dart';

/// 个人详细信息 Provider — 基本信息/联系方式/组织关系 + 学校门户凭证
class PersonalDetailProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  bool _loading = false;
  String _error = '';
  PersonalDetail? _detail;
  PortalCredential? _portal;

  bool get loading => _loading;
  String get error => _error;
  PersonalDetail? get detail => _detail;
  PortalCredential? get portal => _portal;

  Future<void> fetchAll() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final detailRes = await _api.get(ApiConfig.profileDetail);
      if (detailRes.statusCode == 200 && detailRes.data != null) {
        final d = (detailRes.data as Map)['data'];
        if (d is Map) {
          _detail = PersonalDetail.fromJson(Map<String, dynamic>.from(d));
        }
      }
      final portalRes = await _api.get(ApiConfig.portalCredential);
      if (portalRes.statusCode == 200 && portalRes.data != null) {
        final d = (portalRes.data as Map)['data'];
        if (d is Map) {
          _portal = PortalCredential.fromJson(Map<String, dynamic>.from(d));
        }
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  /// 保存学校门户凭证（密码仅加密传输，本地不落盘）
  Future<bool> savePortalCredential({
    required String portalUrl,
    required String account,
    required String password,
  }) async {
    try {
      final res = await _api.put(ApiConfig.portalCredential, data: {
        'portal_url': portalUrl,
        'portal_account': account,
        'portal_password': password,
      });
      if (res.statusCode == 200) {
        await fetchAll();
        return true;
      }
      _error = (res.data as Map?)?['message']?.toString() ?? '保存失败';
    } catch (e) {
      _error = e.toString();
    }
    notifyListeners();
    return false;
  }

  /// 清除学校门户凭证
  Future<bool> clearPortalCredential() async {
    try {
      final res = await _api.delete(ApiConfig.portalCredential);
      if (res.statusCode == 200) {
        _portal = null;
        notifyListeners();
        return true;
      }
    } catch (_) {}
    return false;
  }

  /// 通过门户代理访问校内页面（Dio 携带登录态）
  Future<({int status, String body, String contentType})?> proxyPortal(
      String path) async {
    try {
      final res = await _api.get('${ApiConfig.portalProxy}$path',
          options: Options(responseType: ResponseType.plain));
      if (res.statusCode == 200 || res.statusCode == 302) {
        final st = res.statusCode ?? 0;
        return (
          status: st,
          body: res.data?.toString() ?? '',
          contentType: res.headers.value('content-type') ?? 'text/html',
        );
      }
    } catch (e) {
      _error = e.toString();
    }
    notifyListeners();
    return null;
  }
}
