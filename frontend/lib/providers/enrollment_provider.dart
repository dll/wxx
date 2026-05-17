import 'package:flutter/foundation.dart';
import '../config/api_config.dart';
import '../models/models.dart';
import '../services/api_service.dart';

/// 办事流程状态管理（入学/离校引导）
class EnrollmentProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  String _flowType = 'enrollment'; // enrollment | graduation
  bool _loading = false;
  String? _error;
  AnswerCard? _answerCard;
  List<ProcessStepDetail> _stepDetails = [];

  // 已完成步骤索引集合（本地追踪）
  final Set<int> _completedSteps = {};

  String get flowType => _flowType;
  bool get loading => _loading;
  String? get error => _error;
  AnswerCard? get answerCard => _answerCard;
  List<String> get steps => _answerCard?.steps ?? [];
  List<ProcessStepDetail> get stepDetails => _stepDetails;
  Set<int> get completedSteps => Set.unmodifiable(_completedSteps);

  int get totalSteps {
    if (_stepDetails.isNotEmpty) return _stepDetails.length;
    return steps.length;
  }

  int get completedCount => _completedSteps.length;
  double get progress => totalSteps > 0 ? completedCount / totalSteps : 0;

  /// 切换流程类型
  void setFlowType(String type) {
    if (type == _flowType) return;
    _flowType = type;
    _answerCard = null;
    _stepDetails = [];
    _completedSteps.clear();
    _error = null;
    notifyListeners();
  }

  /// 加载流程指引（优先使用流程增强端点获取富文本步骤）
  Future<void> loadFlow() async {
    _loading = true;
    _error = null;
    notifyListeners();

    try {
      // 先尝试流程增强端点（返回富文本步骤详情）
      final processType = _flowType == 'enrollment' ? 'enrollment' : 'graduation';
      final resp = await _api.get(
        ApiConfig.processEnhanced,
        params: {'type': processType},
      );
      final data = resp.data;

      if (data['code'] == 0 && data['data'] != null) {
        final respData = data['data'] as Map<String, dynamic>;

        // 解析 AnswerCard
        if (respData['answer_card'] != null) {
          _answerCard = AnswerCard.fromJson(respData['answer_card']);
        }

        // 解析富文本步骤列表
        if (respData['processes'] is List) {
          final processes = respData['processes'] as List;
          if (processes.isNotEmpty) {
            final firstProcess = processes[0] as Map<String, dynamic>;
            if (firstProcess['steps'] is List) {
              _stepDetails = (firstProcess['steps'] as List)
                  .map((s) => ProcessStepDetail.fromJson(s))
                  .toList();
            }
          }
        }

        _completedSteps.clear();
        _loading = false;
        notifyListeners();
        return;
      }

      // 降级：使用通用对话接口
      final question = _flowType == 'enrollment'
          ? '新生入学流程及所需材料'
          : '毕业生离校手续办理流程及步骤';

      final req = ChatRequest(question: question);
      final chatResp = await _api.post(ApiConfig.chat, data: req.toJson());
      final chatData = ChatResponse.fromJson(chatResp.data);

      if (chatData.code != 0) {
        _error = chatData.message;
        _loading = false;
        notifyListeners();
        return;
      }

      _answerCard = chatData.data;
      _stepDetails = chatData.data?.stepDetails ?? [];
      _completedSteps.clear();
      _loading = false;
      notifyListeners();
    } catch (e) {
      _error = '加载流程失败，请稍后重试';
      _loading = false;
      notifyListeners();
    }
  }

  /// 切换步骤完成状态
  void toggleStep(int index) {
    if (_completedSteps.contains(index)) {
      _completedSteps.remove(index);
    } else {
      _completedSteps.add(index);
    }
    notifyListeners();
  }

  /// 全部标记为完成
  void completeAll() {
    for (var i = 0; i < totalSteps; i++) {
      _completedSteps.add(i);
    }
    notifyListeners();
  }

  /// 重置进度
  void resetProgress() {
    _completedSteps.clear();
    notifyListeners();
  }
}
