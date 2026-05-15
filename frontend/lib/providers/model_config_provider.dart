import 'package:flutter/foundation.dart';
import '../models/models.dart';
import '../services/api_service.dart';
import '../config/api_config.dart';

/// 用户 AI 模型配置状态管理
class ModelConfigProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  ModelConfig? _config;
  bool _loading = false;
  String? _error;

  ModelConfig? get config => _config;
  bool get loading => _loading;
  String? get error => _error;

  /// 获取当前用户的模型配置
  Future<void> fetchConfig() async {
    _loading = true;
    _error = null;
    notifyListeners();

    try {
      final resp = await _api.get(ApiConfig.modelConfig);
      if (resp.data['code'] == 0 && resp.data['data'] != null) {
        _config = ModelConfig.fromJson(resp.data['data']);
      }
    } catch (e) {
      _error = '获取模型配置失败';
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  /// 保存模型配置
  Future<bool> saveConfig(ModelConfig config) async {
    _loading = true;
    _error = null;
    notifyListeners();

    try {
      final resp = await _api.put(ApiConfig.modelConfig, data: config.toJson());
      if (resp.data['code'] == 0) {
        _config = config;
        _loading = false;
        notifyListeners();
        return true;
      }
      _error = resp.data['message'] ?? '保存失败';
    } catch (e) {
      _error = '网络错误: $e';
    }

    _loading = false;
    notifyListeners();
    return false;
  }
}
