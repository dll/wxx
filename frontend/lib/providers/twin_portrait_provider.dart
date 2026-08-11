import 'dart:convert';

import 'package:flutter/foundation.dart';

import '../config/api_config.dart';
import '../models/models.dart';
import '../services/api_service.dart';

/// 数字孪生画像 Provider — 拉取/生成蔚小芯风格画像
class TwinPortraitProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  bool _loading = false;
  bool _generating = false;
  String _error = '';
  TwinPortrait? _photoPortrait; // 基于用户照片生成
  TwinPortrait? _chaoXingPortrait; // 基于超星原型生成

  bool get loading => _loading;
  bool get generating => _generating;
  String get error => _error;
  TwinPortrait? get photoPortrait => _photoPortrait;
  TwinPortrait? get chaoXingPortrait => _chaoXingPortrait;

  /// 当前应展示的画像（优先照片版）
  TwinPortrait? get current =>
      _photoPortrait ?? _chaoXingPortrait;

  Future<void> fetchPortraits() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.twinPortraits);
      if (res.statusCode == 200 && res.data != null) {
        final data = res.data is Map ? (res.data as Map)['data'] : res.data;
        if (data is List) {
          _photoPortrait = null;
          _chaoXingPortrait = null;
          for (final e in data.whereType<Map>()) {
            final p = TwinPortrait.fromJson(Map<String, dynamic>.from(e));
            if (p.prototypeType == 'photo') {
              _photoPortrait = p;
            } else if (p.prototypeType == 'chao_xing') {
              _chaoXingPortrait = p;
            }
          }
        }
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  /// 生成画像。
  /// photoBase64: 用户照片 base64（photo 模式必填）
  /// prototypeType: photo | chao_xing
  Future<bool> generate({
    required String prototypeType,
    String? photoBase64,
    String? photoMime,
    String? highlights,
  }) async {
    _generating = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.post(ApiConfig.twinPortraitGenerate, data: {
        'type': prototypeType,
        if (photoBase64 != null) 'photo_base64': photoBase64,
        if (photoMime != null) 'photo_mime': photoMime,
        if (highlights != null && highlights.isNotEmpty) 'highlights': highlights,
      });
      if (res.statusCode == 200 && res.data != null) {
        final d = (res.data as Map)['data'];
        if (d is Map) {
          final p = TwinPortrait.fromJson(Map<String, dynamic>.from(d));
          if (p.prototypeType == 'photo') {
            _photoPortrait = p;
          } else if (p.prototypeType == 'chao_xing') {
            _chaoXingPortrait = p;
          }
          notifyListeners();
          return true;
        }
      }
      // 后端可能返回 400 错误消息
      final msg = (res.data as Map?)?['message'];
      if (msg != null) _error = msg.toString();
    } catch (e) {
      _error = e.toString();
    } finally {
      _generating = false;
      notifyListeners();
    }
    return false;
  }
}

/// 工具：图片文件 → base64
Future<String> fileToBase64(Uint8List bytes) {
  return Future.value(base64Encode(bytes));
}