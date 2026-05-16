import 'package:flutter/material.dart';
import '../config/api_config.dart';
import '../services/api_service.dart';

/// 校园文化智能体 Provider — 5 个能力共用一个 provider，按需取数
class CultureProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  Map<String, dynamic>? _anthems;
  Map<String, dynamic>? _radio;
  Map<String, dynamic>? _lectures;
  Map<String, dynamic>? _events;
  Map<String, dynamic>? _volunteer;

  String? _error;
  bool _loading = false;

  Map<String, dynamic>? get anthems => _anthems;
  Map<String, dynamic>? get radio => _radio;
  Map<String, dynamic>? get lectures => _lectures;
  Map<String, dynamic>? get events => _events;
  Map<String, dynamic>? get volunteer => _volunteer;
  String? get error => _error;
  bool get loading => _loading;

  Future<void> fetchAnthems() => _fetch(ApiConfig.cultureAnthems, (d) => _anthems = d);
  Future<void> fetchRadio() => _fetch(ApiConfig.cultureRadio, (d) => _radio = d);
  Future<void> fetchLectures() => _fetch(ApiConfig.cultureLectures, (d) => _lectures = d);
  Future<void> fetchEvents() => _fetch(ApiConfig.cultureEvents, (d) => _events = d);
  Future<void> fetchVolunteer() => _fetch(ApiConfig.cultureVolunteer, (d) => _volunteer = d);

  Future<void> _fetch(String url, void Function(Map<String, dynamic>) setter) async {
    _loading = true;
    _error = null;
    notifyListeners();
    try {
      final resp = await _api.get(url);
      if (resp.data is Map<String, dynamic>) {
        setter(resp.data as Map<String, dynamic>);
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }
}
