import 'package:flutter/material.dart';
import '../config/api_config.dart';
import '../models/models.dart';
import '../services/api_service.dart';

/// 知识大厅状态管理
class KnowledgeProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  Map<String, List<KnowledgeCard>> _categories = {};
  bool _loading = false;
  String? _error;
  String _selectedType = ''; // 空=全部

  Map<String, List<KnowledgeCard>> get categories => _categories;
  bool get loading => _loading;
  String? get error => _error;
  String get selectedType => _selectedType;

  /// 是否有数据
  bool get isEmpty => _categories.isEmpty && !_loading;

  /// 总卡片数
  int get totalCount {
    int count = 0;
    for (final list in _categories.values) {
      count += list.length;
    }
    return count;
  }

  /// 加载知识大厅数据
  Future<void> load({String? type}) async {
    _loading = true;
    _error = null;
    _selectedType = type ?? '';
    notifyListeners();

    try {
      final queryParams = <String, dynamic>{};
      if (_selectedType.isNotEmpty) {
        queryParams['type'] = _selectedType;
      }
      final response = await _api.get(ApiConfig.knowledge, params: queryParams);

      final data = response.data['data'] as Map<String, dynamic>?;
      if (data == null) {
        _categories = {};
      } else {
        _categories = {};
        data.forEach((type, cards) {
          final list = (cards as List)
              .map((c) => KnowledgeCard.fromJson(c as Map<String, dynamic>))
              .toList();
          _categories[type] = list;
        });
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  /// 切换类型过滤
  void filterByType(String? type) {
    load(type: type);
  }

  /// 类型顺序（Policy → Process → FAQ → Activity）
  static const List<String> typeOrder = ['Policy', 'Process', 'FAQ', 'Activity'];

  /// 按固定顺序返回分类列表
  List<MapEntry<String, List<KnowledgeCard>>> get orderedCategories {
    final result = <MapEntry<String, List<KnowledgeCard>>>[];
    for (final type in typeOrder) {
      if (_categories.containsKey(type) && _categories[type]!.isNotEmpty) {
        result.add(MapEntry(type, _categories[type]!));
      }
    }
    return result;
  }
}
