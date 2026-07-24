import 'package:flutter/foundation.dart';
import '../config/api_config.dart';
import '../services/api_service.dart';

/// 通知类型
class NotificationType {
  static const String system = 'system';
  static const String feedback = 'feedback';
  static const String knowledge = 'knowledge';
  static const String activity = 'activity';
  static const String career = 'career';
}

/// 通知数据模型
class NotificationItem {
  final int id;
  final int userId;
  final String title;
  final String content;
  final String type;
  final String relatedType;
  final int relatedId;
  final int isRead;
  final String createdAt;

  NotificationItem({
    required this.id,
    required this.userId,
    required this.title,
    required this.content,
    required this.type,
    required this.relatedType,
    required this.relatedId,
    required this.isRead,
    required this.createdAt,
  });

  factory NotificationItem.fromJson(Map<String, dynamic> json) {
    return NotificationItem(
      id: json['id'] ?? 0,
      userId: json['user_id'] ?? 0,
      title: json['title'] ?? '',
      content: json['content'] ?? '',
      type: json['type'] ?? NotificationType.system,
      relatedType: json['related_type'] ?? '',
      relatedId: json['related_id'] ?? 0,
      isRead: json['is_read'] ?? 0,
      createdAt: json['created_at'] ?? '',
    );
  }
}

/// 通知状态管理
class NotificationProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  bool _loading = false;
  String _error = '';
  bool get loading => _loading;
  String get error => _error;

  List<NotificationItem> _items = [];
  int _total = 0;
  int _unreadCount = 0;
  int _page = 1;
  final int _pageSize = 20;
  String _currentType = '';

  List<NotificationItem> get items => _items;
  int get total => _total;
  int get unreadCount => _unreadCount;
  int get page => _page;
  int get pageSize => _pageSize;
  String get currentType => _currentType;

  /// 获取通知列表
  Future<void> fetchNotifications({String type = '', int page = 1}) async {
    _loading = true;
    _error = '';
    _currentType = type;
    _page = page;
    notifyListeners();

    try {
      final params = <String, dynamic>{
        'page': page,
        'page_size': _pageSize,
      };
      if (type.isNotEmpty) {
        params['type'] = type;
      }

      final res = await _api.get(ApiConfig.notifications, params: params);
      if (res.statusCode == 200 && res.data != null) {
        final data = res.data;
        if (data is Map<String, dynamic>) {
          _unreadCount = data['unread_count'] ?? 0;
          _total = data['total'] ?? 0;
          final itemsList = data['items'] as List? ?? [];
          if (page == 1) {
            _items = itemsList.map((e) => NotificationItem.fromJson(e)).toList();
          } else {
            _items.addAll(itemsList.map((e) => NotificationItem.fromJson(e)).toList());
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

  /// 获取未读数量
  Future<void> fetchUnreadCount() async {
    try {
      final res = await _api.get(ApiConfig.notificationsUnread);
      if (res.statusCode == 200 && res.data != null) {
        final data = res.data;
        if (data is Map<String, dynamic>) {
          _unreadCount = data['unread_count'] ?? 0;
          notifyListeners();
        }
      }
    } catch (e) {
      // 静默失败，不影响主流程
    }
  }

  /// 标记单条已读
  Future<bool> markAsRead(int id) async {
    try {
      final res = await _api.put(ApiConfig.notificationRead(id.toString()));
      if (res.statusCode == 200) {
        // 更新本地状态
        final index = _items.indexWhere((item) => item.id == id);
        if (index != -1 && _items[index].isRead == 0) {
          _items[index] = NotificationItem(
            id: _items[index].id,
            userId: _items[index].userId,
            title: _items[index].title,
            content: _items[index].content,
            type: _items[index].type,
            relatedType: _items[index].relatedType,
            relatedId: _items[index].relatedId,
            isRead: 1,
            createdAt: _items[index].createdAt,
          );
          if (_unreadCount > 0) {
            _unreadCount--;
          }
          notifyListeners();
        }
        return true;
      }
    } catch (e) {
      _error = e.toString();
    }
    return false;
  }

  /// 全部标记已读
  Future<bool> markAllAsRead() async {
    try {
      final res = await _api.put(ApiConfig.notificationsReadAll);
      if (res.statusCode == 200) {
        // 更新本地状态
        _items = _items.map((item) {
          return NotificationItem(
            id: item.id,
            userId: item.userId,
            title: item.title,
            content: item.content,
            type: item.type,
            relatedType: item.relatedType,
            relatedId: item.relatedId,
            isRead: 1,
            createdAt: item.createdAt,
          );
        }).toList();
        _unreadCount = 0;
        notifyListeners();
        return true;
      }
    } catch (e) {
      _error = e.toString();
    }
    return false;
  }

  /// 加载更多
  Future<void> loadMore() async {
    if (_loading) return;
    if (_items.length >= _total) return;
    await fetchNotifications(type: _currentType, page: _page + 1);
  }

  /// 刷新
  Future<void> refresh() async {
    await fetchNotifications(type: _currentType, page: 1);
  }
}
