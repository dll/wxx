import 'package:flutter/foundation.dart';
import '../config/api_config.dart';
import '../models/models.dart';
import '../services/api_service.dart';

/// 办事流程状态管理（入学/离校引导）
class EnrollmentProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  // 当前选中的流程类型
  String _flowType = 'enrollment'; // enrollment | graduation
  bool _loading = false;
  String? _error;
  AnswerCard? _answerCard;

  // 已完成步骤索引集合（本地追踪）
  final Set<int> _completedSteps = {};

  String get flowType => _flowType;
  bool get loading => _loading;
  String? get error => _error;
  AnswerCard? get answerCard => _answerCard;
  List<String> get steps => _answerCard?.steps ?? [];
  Set<int> get completedSteps => Set.unmodifiable(_completedSteps);

  int get totalSteps => steps.length;
  int get completedCount => _completedSteps.length;
  double get progress => totalSteps > 0 ? completedCount / totalSteps : 0;

  /// 切换流程类型
  void setFlowType(String type) {
    if (type == _flowType) return;
    _flowType = type;
    _answerCard = null;
    _completedSteps.clear();
    _error = null;
    notifyListeners();
  }

  /// 加载流程指引
  Future<void> loadFlow() async {
    _loading = true;
    _error = null;
    notifyListeners();

    try {
      final question = _flowType == 'enrollment'
          ? '新生入学流程及所需材料'
          : '毕业生离校手续办理流程及步骤';

      final req = ChatRequest(question: question);
      final resp = await _api.post(ApiConfig.chat, data: req.toJson());
      final chatResp = ChatResponse.fromJson(resp.data);

      if (chatResp.code != 0) {
        _error = chatResp.message;
        _loading = false;
        notifyListeners();
        return;
      }

      _answerCard = chatResp.data;
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
