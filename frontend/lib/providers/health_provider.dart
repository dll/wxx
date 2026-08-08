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

  bool get loading => _loading;
  String? get error => _error;
  HealthBasicInfo? get basicInfo => _basicInfo;
  List<HealthCheckup> get checkups => _checkups;
  List<HealthRecordItem> get records => _records;

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
}
