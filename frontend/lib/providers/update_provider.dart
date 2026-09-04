import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:flutter/material.dart';
import '../config/api_config.dart';
import '../config/release_config.dart';
import '../services/api_service.dart';

/// 版本更新状态管理
class UpdateProvider extends ChangeNotifier {
  bool _checking = false;
  bool _hasUpdate = false;
  bool _isForce = false;
  Map<String, dynamic>? _latestVersion;
  String? _error;

  bool get checking => _checking;
  bool get hasUpdate => _hasUpdate;
  bool get isForce => _isForce;
  Map<String, dynamic>? get latestVersion => _latestVersion;
  String? get error => _error;

  String get currentVersion => ReleaseConfig.version;
  int get currentBuildNumber => ReleaseConfig.buildNumber;

  /// 检查更新
  Future<bool> checkUpdate({bool silent = false}) async {
    if (_checking) return false;

    _checking = true;
    _error = null;
    if (!silent) notifyListeners();

    try {
      const platform = kIsWeb ? 'web' : 'android';
      final res = await ApiService().get(
        ApiConfig.versionCheck,
        params: {
          'platform': platform,
          'version_code': currentBuildNumber,
          'version_name': currentVersion,
        },
      );

      if (res.statusCode == 200 && res.data != null) {
        final data = res.data is Map ? res.data['data'] : null;
        if (data != null) {
          _hasUpdate = data['has_update'] ?? false;
          _isForce = data['is_force'] ?? false;
          _latestVersion = data['latest'] != null
              ? Map<String, dynamic>.from(data['latest'])
              : null;
        }
      }

      return _hasUpdate;
    } on DioException catch (e) {
      if (e.response?.statusCode == 404) {
        _hasUpdate = false;
        _latestVersion = null;
        return false;
      }
      _error = e.toString();
      return false;
    } catch (e) {
      _error = e.toString();
      return false;
    } finally {
      _checking = false;
      notifyListeners();
    }
  }

  /// 显示更新对话框
  void showUpdateDialog(BuildContext context) {
    if (!_hasUpdate || _latestVersion == null) return;

    final versionName = _latestVersion!['version_name'] ?? '';
    final title = _latestVersion!['title'] ?? '发现新版本';
    final changelog = _latestVersion!['changelog'] ?? '';
    final downloadUrl = _latestVersion!['download_url'] ?? '';
    final isForce = _isForce;

    showDialog(
      context: context,
      barrierDismissible: !isForce,
      builder: (context) => AlertDialog(
        title: Row(
          children: [
            const Icon(Icons.system_update, color: Colors.blue),
            const SizedBox(width: 8),
            Expanded(child: Text(title)),
          ],
        ),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                '新版本 v$versionName',
                style: const TextStyle(
                  fontSize: 16,
                  fontWeight: FontWeight.bold,
                  color: Colors.blue,
                ),
              ),
              const SizedBox(height: 12),
              if (changelog.isNotEmpty) ...[
                const Text(
                  '更新内容：',
                  style: TextStyle(fontWeight: FontWeight.w600),
                ),
                const SizedBox(height: 8),
                Text(changelog),
              ],
              const SizedBox(height: 16),
              if (kIsWeb)
                const Text(
                  '点击"立即更新"将刷新页面加载最新版本',
                  style: TextStyle(fontSize: 12, color: Colors.grey),
                ),
            ],
          ),
        ),
        actions: [
          if (!isForce)
            TextButton(
              onPressed: () => Navigator.of(context).pop(),
              child: const Text('稍后再说'),
            ),
          ElevatedButton.icon(
            onPressed: () {
              Navigator.of(context).pop();
              _handleUpdate(downloadUrl);
            },
            icon: const Icon(Icons.download),
            label: const Text('立即更新'),
          ),
        ],
      ),
    );
  }

  /// 处理更新操作
  void _handleUpdate(String downloadUrl) {
    if (kIsWeb) {
      // Web 端：刷新页面
      // 浏览器会加载最新的静态资源
    } else {
      // 移动端：跳转到下载地址
      if (downloadUrl.isNotEmpty) {
        // 使用 url_launcher 打开下载链接
      }
    }
  }

  /// 重置状态
  void reset() {
    _checking = false;
    _hasUpdate = false;
    _isForce = false;
    _latestVersion = null;
    _error = null;
    notifyListeners();
  }
}
