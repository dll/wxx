import 'package:flutter/foundation.dart';
import '../config/api_config.dart';
import '../services/api_service.dart';

/// 身体基本信息模型
class HealthBasicInfo {
  final int id;
  final double heightCm;
  final double weightKg;
  final String bloodType;
  final String visionLeft;
  final String visionRight;
  final String allergies;
  final String pastIllness;
  final String familyHistory;
  final String emergencyContact;
  final String emergencyPhone;
  final String updatedAt;

  const HealthBasicInfo({
    this.id = 0,
    this.heightCm = 0,
    this.weightKg = 0,
    this.bloodType = '',
    this.visionLeft = '',
    this.visionRight = '',
    this.allergies = '',
    this.pastIllness = '',
    this.familyHistory = '',
    this.emergencyContact = '',
    this.emergencyPhone = '',
    this.updatedAt = '',
  });

  factory HealthBasicInfo.fromJson(Map<String, dynamic> json) {
    return HealthBasicInfo(
      id: json['id'] ?? 0,
      heightCm: (json['height_cm'] as num?)?.toDouble() ?? 0,
      weightKg: (json['weight_kg'] as num?)?.toDouble() ?? 0,
      bloodType: json['blood_type'] ?? '',
      visionLeft: json['vision_left'] ?? '',
      visionRight: json['vision_right'] ?? '',
      allergies: json['allergies'] ?? '',
      pastIllness: json['past_illness'] ?? '',
      familyHistory: json['family_history'] ?? '',
      emergencyContact: json['emergency_contact'] ?? '',
      emergencyPhone: json['emergency_phone'] ?? '',
      updatedAt: json['updated_at'] ?? '',
    );
  }
}

/// 体检记录模型
class HealthCheckup {
  final int id;
  final String checkupDate;
  final String hospital;
  final String conclusion;
  final String details;
  final List<String> attachments;
  final String createdAt;

  const HealthCheckup({
    this.id = 0,
    this.checkupDate = '',
    this.hospital = '',
    this.conclusion = '',
    this.details = '',
    this.attachments = const [],
    this.createdAt = '',
  });

  factory HealthCheckup.fromJson(Map<String, dynamic> json) {
    final att = (json['attachments'] as List?)?.cast<String>() ?? const <String>[];
    return HealthCheckup(
      id: json['id'] ?? 0,
      checkupDate: json['checkup_date'] ?? '',
      hospital: json['hospital'] ?? '',
      conclusion: json['conclusion'] ?? '',
      details: json['details'] ?? '',
      attachments: att,
      createdAt: json['created_at'] ?? '',
    );
  }
}

/// 病历记录模型
class HealthRecordItem {
  final int id;
  final String recordDate;
  final String hospital;
  final String department;
  final String diagnosis;
  final String treatment;
  final List<String> attachments;
  final String createdAt;

  const HealthRecordItem({
    this.id = 0,
    this.recordDate = '',
    this.hospital = '',
    this.department = '',
    this.diagnosis = '',
    this.treatment = '',
    this.attachments = const [],
    this.createdAt = '',
  });

  factory HealthRecordItem.fromJson(Map<String, dynamic> json) {
    final att = (json['attachments'] as List?)?.cast<String>() ?? const <String>[];
    return HealthRecordItem(
      id: json['id'] ?? 0,
      recordDate: json['record_date'] ?? '',
      hospital: json['hospital'] ?? '',
      department: json['department'] ?? '',
      diagnosis: json['diagnosis'] ?? '',
      treatment: json['treatment'] ?? '',
      attachments: att,
      createdAt: json['created_at'] ?? '',
    );
  }
}

/// 日常健康记录模型（身高/体重/血压/心率）
class HealthDailyRecord {
  final int id;
  final String recordDate;
  final double heightCm;
  final double weightKg;
  final int systolic;
  final int diastolic;
  final int heartRate;
  final String note;
  final String createdAt;

  const HealthDailyRecord({
    this.id = 0,
    this.recordDate = '',
    this.heightCm = 0,
    this.weightKg = 0,
    this.systolic = 0,
    this.diastolic = 0,
    this.heartRate = 0,
    this.note = '',
    this.createdAt = '',
  });

  factory HealthDailyRecord.fromJson(Map<String, dynamic> json) {
    return HealthDailyRecord(
      id: json['id'] ?? 0,
      recordDate: json['record_date'] ?? '',
      heightCm: (json['height_cm'] as num?)?.toDouble() ?? 0,
      weightKg: (json['weight_kg'] as num?)?.toDouble() ?? 0,
      systolic: json['systolic'] ?? 0,
      diastolic: json['diastolic'] ?? 0,
      heartRate: json['heart_rate'] ?? 0,
      note: json['note'] ?? '',
      createdAt: json['created_at'] ?? '',
    );
  }
}

/// 健身活动 / 竞技比赛模型
class HealthActivity {
  final String activityId;
  final String title;
  final String category;
  final String description;
  final String startAt;
  final String endAt;
  final String venue;
  final String organizer;
  final int capacity;
  final String signupDeadline;
  final String status;
  final String creatorRole;
  final int favoriteCount;
  final int signupCount;
  final bool isFavorite;
  final bool isSignup;

  const HealthActivity({
    this.activityId = '',
    this.title = '',
    this.category = 'sports',
    this.description = '',
    this.startAt = '',
    this.endAt = '',
    this.venue = '',
    this.organizer = '',
    this.capacity = 0,
    this.signupDeadline = '',
    this.status = 'active',
    this.creatorRole = '',
    this.favoriteCount = 0,
    this.signupCount = 0,
    this.isFavorite = false,
    this.isSignup = false,
  });

  factory HealthActivity.fromJson(Map<String, dynamic> json) {
    return HealthActivity(
      activityId: json['activity_id'] ?? '',
      title: json['title'] ?? '',
      category: json['category'] ?? 'sports',
      description: json['description'] ?? '',
      startAt: json['start_at'] ?? '',
      endAt: json['end_at'] ?? '',
      venue: json['venue'] ?? '',
      organizer: json['organizer'] ?? '',
      capacity: json['capacity'] ?? 0,
      signupDeadline: json['signup_deadline'] ?? '',
      status: json['status'] ?? 'active',
      creatorRole: json['creator_role'] ?? '',
      favoriteCount: json['favorite_count'] ?? 0,
      signupCount: json['signup_count'] ?? 0,
      isFavorite: json['is_favorite'] ?? false,
      isSignup: json['is_signup'] ?? false,
    );
  }
}

/// 身体健康 Provider：身体基本信息 + 体检记录 + 病历记录
class HealthProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  bool _loading = false;
  String? _error;

  // 身体基本信息
  HealthBasicInfo? _basicInfo;
  // 体检记录
  List<HealthCheckup> _checkups = [];
  // 病历记录
  List<HealthRecordItem> _records = [];
  // 日常记录
  List<HealthDailyRecord> _daily = [];
  // 健身活动/竞技比赛
  List<HealthActivity> _activities = [];

  bool get loading => _loading;
  String? get error => _error;
  HealthBasicInfo? get basicInfo => _basicInfo;
  List<HealthCheckup> get checkups => _checkups;
  List<HealthRecordItem> get records => _records;
  List<HealthDailyRecord> get daily => _daily;
  List<HealthActivity> get activities => _activities;

  Future<void> _guard(Future<void> Function() fn) async {
    _loading = true;
    _error = null;
    notifyListeners();
    try {
      await fn();
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 身体基本信息 ──

  Future<void> fetchBasicInfo() {
    return _guard(() async {
      final res = await _api.get(ApiConfig.healthBasic);
      final data = res.data['data'];
      _basicInfo = data is Map
          ? HealthBasicInfo.fromJson(Map<String, dynamic>.from(data))
          : null;
    });
  }

  Future<bool> saveBasicInfo(Map<String, dynamic> body) async {
    try {
      final res = await _api.put(ApiConfig.healthBasic, data: body);
      if (res.data['code'] == 0) {
        await fetchBasicInfo();
        return true;
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  // ── 体检记录 ──

  Future<void> fetchCheckups() {
    return _guard(() async {
      final res = await _api.get(ApiConfig.healthCheckups);
      final list = (res.data['data'] as List?) ?? const [];
      _checkups = list
          .map((e) => HealthCheckup.fromJson(Map<String, dynamic>.from(e as Map)))
          .toList();
    });
  }

  Future<bool> createCheckup(Map<String, dynamic> body) async {
    try {
      final res = await _api.post(ApiConfig.healthCheckups, data: body);
      if (res.data['code'] == 0) {
        await fetchCheckups();
        return true;
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  Future<bool> updateCheckup(int id, Map<String, dynamic> body) async {
    try {
      final res = await _api.put('${ApiConfig.healthCheckups}/$id', data: body);
      if (res.data['code'] == 0) {
        await fetchCheckups();
        return true;
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  Future<bool> deleteCheckup(int id) async {
    try {
      final res = await _api.delete('${ApiConfig.healthCheckups}/$id');
      if (res.data['code'] == 0) {
        await fetchCheckups();
        return true;
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  // ── 病历记录 ──

  Future<void> fetchRecords() {
    return _guard(() async {
      final res = await _api.get(ApiConfig.healthRecords);
      final list = (res.data['data'] as List?) ?? const [];
      _records = list
          .map((e) => HealthRecordItem.fromJson(Map<String, dynamic>.from(e as Map)))
          .toList();
    });
  }

  Future<bool> createRecord(Map<String, dynamic> body) async {
    try {
      final res = await _api.post(ApiConfig.healthRecords, data: body);
      if (res.data['code'] == 0) {
        await fetchRecords();
        return true;
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  Future<bool> updateRecord(int id, Map<String, dynamic> body) async {
    try {
      final res = await _api.put('${ApiConfig.healthRecords}/$id', data: body);
      if (res.data['code'] == 0) {
        await fetchRecords();
        return true;
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  Future<bool> deleteRecord(int id) async {
    try {
      final res = await _api.delete('${ApiConfig.healthRecords}/$id');
      if (res.data['code'] == 0) {
        await fetchRecords();
        return true;
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  // ── 日常记录 ──

  Future<void> fetchDaily() {
    return _guard(() async {
      final res = await _api.get('${ApiConfig.healthDaily}?limit=180');
      final list = (res.data['data'] as List?) ?? const [];
      _daily = list
          .map((e) =>
              HealthDailyRecord.fromJson(Map<String, dynamic>.from(e as Map)))
          .toList();
    });
  }

  Future<bool> saveDaily(Map<String, dynamic> body) async {
    try {
      final res = await _api.put(ApiConfig.healthDaily, data: body);
      if (res.data['code'] == 0) {
        await fetchDaily();
        return true;
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  Future<bool> deleteDaily(String date) async {
    try {
      final res = await _api.delete('${ApiConfig.healthDaily}/$date');
      if (res.data['code'] == 0) {
        await fetchDaily();
        return true;
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  // ── 健身活动 / 竞技比赛 ──

  Future<void> fetchActivities() {
    return _guard(() async {
      final res = await _api.get('${ApiConfig.healthActivities}?category=');
      final list = (res.data['data'] as List?) ?? const [];
      _activities = list
          .map((e) =>
              HealthActivity.fromJson(Map<String, dynamic>.from(e as Map)))
          .toList();
    });
  }

  Future<bool> toggleFavorite(String activityId, bool favorite) async {
    try {
      final res = await _api.post(
          '${ApiConfig.healthActivities}/$activityId/favorite',
          data: {'favorite': favorite});
      if (res.data['code'] == 0) {
        await fetchActivities();
        return true;
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  Future<bool> toggleSignup(String activityId, bool signup) async {
    try {
      final res = await _api.post(
          '${ApiConfig.healthActivities}/$activityId/signup',
          data: {'signup': signup});
      if (res.data['code'] == 0) {
        await fetchActivities();
        return true;
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  Future<bool> createActivity(Map<String, dynamic> body) async {
    try {
      final res = await _api.post(ApiConfig.healthActivities, data: body);
      if (res.data['code'] == 0) {
        await fetchActivities();
        return true;
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  /// 更新活动状态（end/close/active），活动生命周期管理
  Future<bool> updateActivityStatus(String activityId, String status) async {
    try {
      final res = await _api.post('${ApiConfig.healthActivities}/$activityId/status',
          data: {'status': status});
      if (res.data['code'] == 0) {
        await fetchActivities();
        return true;
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  /// 活动复盘指标（真实统计）
  Future<Map<String, dynamic>?> fetchReviewStats() async {
    try {
      final res = await _api.get('${ApiConfig.healthActivities}/review-stats');
      if (res.data['code'] == 0) return Map<String, dynamic>.from(res.data);
      return null;
    } catch (_) {
      return null;
    }
  }

  /// 活动报名/到场名单
  Future<List<Map<String, dynamic>>> listActivitySignups(String activityId) async {
    try {
      final res = await _api.get('${ApiConfig.healthActivities}/$activityId/signups');
      if (res.data['code'] == 0) {
        return (res.data['items'] as List).cast<Map<String, dynamic>>();
      }
      return [];
    } catch (_) {
      return [];
    }
  }

  /// 标记报名者签到/取消
  Future<bool> attendSignup(String activityId, int userId, bool attended) async {
    try {
      final res = await _api.post('${ApiConfig.healthActivities}/$activityId/attend/$userId',
          data: {'attended': attended});
      return res.data['code'] == 0;
    } catch (_) {
      return false;
    }
  }
}
