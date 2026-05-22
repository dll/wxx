import 'package:flutter/foundation.dart';
import '../models/models.dart';
import '../services/api_service.dart';
import '../config/api_config.dart';
import '../utils/capability_utils.dart';

/// 智能体管理状态
class AgentProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  List<Agent> _agents = [];
  bool _loading = false;
  String _error = '';

  List<Agent> get agents => _agents;
  bool get loading => _loading;
  String get error => _error;

  /// 加载所有智能体
  Future<void> loadAgents() async {
    if (!CapabilityUtils.has(Capability.schoolAgentWrite)) {
      _error = '当前角色无权访问智能体管理';
      notifyListeners();
      return;
    }
    _loading = true;
    _error = '';
    notifyListeners();

    try {
      final response = await _api.get(ApiConfig.agents);
      if (response.data['code'] == 0) {
        final List raw = response.data['data'] ?? [];
        _agents = raw
            .map((e) => Agent.fromJson(e as Map<String, dynamic>))
            .toList();
      } else {
        _error = response.data['message'] ?? '加载失败';
      }
    } catch (e) {
      _error = '网络错误: $e';
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  /// 创建智能体
  Future<bool> create(AgentSaveRequest req) async {
    try {
      final response = await _api.post(
        ApiConfig.agents,
        data: req.toJson(),
      );
      if (response.data['code'] == 0) {
        await loadAgents(); // 刷新列表
        return true;
      }
      _error = response.data['message'] ?? '创建失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  /// 更新智能体
  Future<bool> update(String agentId, Map<String, dynamic> updates) async {
    try {
      final response = await _api.put(
        ApiConfig.agentDetail(agentId),
        data: updates,
      );
      if (response.data['code'] == 0) {
        await loadAgents();
        return true;
      }
      _error = response.data['message'] ?? '更新失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  /// 切换启用/停用状态
  Future<bool> toggleStatus(Agent agent) async {
    return update(agent.agentId, {
      'status': agent.isActive ? 'inactive' : 'active',
    });
  }

  /// 删除智能体
  Future<bool> delete(String agentId) async {
    try {
      final response = await _api.delete(ApiConfig.agentDetail(agentId));
      if (response.data['code'] == 0) {
        await loadAgents();
        return true;
      }
      _error = response.data['message'] ?? '删除失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }
}
