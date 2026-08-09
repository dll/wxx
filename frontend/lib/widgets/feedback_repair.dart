// 管理员「在线修复」代码定位助手（AI 增强版）
// 反馈详情内置：
//  - 调用后端 AI 修复接口（GLM-4V 解析截图 + 文本模型定位代码文件与修复建议）
//  - LLM/视觉不可用时自动降级为本地关键词模块匹配
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/models.dart';
import '../providers/feedback_provider.dart';

/// 功能模块 → 项目代码文件映射（AI 不可用时本地兜底）
const _moduleFileMap = <String, List<String>>{
  '登录 / 认证': [
    'frontend/lib/pages/login/login_page.dart',
    'frontend/lib/providers/auth_provider.dart',
    'server/internal/handler/auth_handler.go',
  ],
  '对话 / 问答': [
    'frontend/lib/pages/chat/chat_page.dart',
    'frontend/lib/providers/chat_provider.dart',
    'server/internal/service/chat_service.go',
    'server/internal/context_engine/engine.go',
  ],
  '知识库 / 检索': [
    'server/internal/context_engine/',
    'server/internal/repository/kb_repo.go',
    'frontend/lib/pages/knowledge/',
  ],
  '办事流程': [
    'frontend/lib/pages/process/',
    'server/internal/handler/process_handler.go',
    'frontend/lib/providers/process_provider.dart',
  ],
  '报到 / 校园导航': [
    'frontend/lib/pages/campus/campus_map_page.dart',
    'frontend/lib/widgets/baidu_campus_map_embed_web.dart',
    'server/internal/handler/campus_handler.go',
  ],
  '语音': [
    'frontend/lib/services/voice/',
    'server/internal/handler/voice_handler.go',
  ],
  '我的 / 个人中心': [
    'frontend/lib/pages/profile/profile_page.dart',
    'frontend/lib/providers/auth_provider.dart',
  ],
  '反馈系统': [
    'frontend/lib/pages/admin/feedback_page.dart',
    'frontend/lib/providers/feedback_provider.dart',
    'server/internal/handler/feedback_handler.go',
  ],
  '消息 / 通知': [
    'frontend/lib/pages/notifications/',
    'server/internal/handler/notification_handler.go',
  ],
  '学生服务': [
    'frontend/lib/pages/student/',
    'server/internal/service/student_service.go',
  ],
  '教务 / 课表': [
    'frontend/lib/pages/student/',
    'server/internal/service/study_service.go',
  ],
  '心理 / 情感': [
    'frontend/lib/pages/student/mental/',
    'server/internal/service/emotion_service.go',
  ],
  '管理端 / 数据': [
    'frontend/lib/pages/admin/',
    'server/internal/handler/admin_handler.go',
  ],
};

/// 打开「在线修复」面板
void showOnlineRepair(BuildContext context, FeedbackEntry fb) {
  showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    showDragHandle: true,
    backgroundColor: Theme.of(context).colorScheme.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (_) => _OnlineRepairSheet(fb: fb),
  );
}

/// 根据反馈内容匹配最相关的代码模块（AI 不可用时兜底）
List<String> _matchModuleNames(FeedbackEntry fb) {
  const map = <String, List<String>>{
    '登录 / 认证': ['登录', '认证', '账号', '密码', '扫码', 'token', '验证码', 'login', 'auth'],
    '对话 / 问答': ['回答', '问答', '对话', '聊天', '回复', '答案', 'chat', 'AI', '智能'],
    '知识库 / 检索': ['知识', '检索', '搜索', '词条', '知识库', 'FTS', '搜索结果', '查不到'],
    '办事流程': ['办事', '流程', '手续', '申请', '审批', 'process'],
    '报到 / 校园导航': ['报到', '地图', '导航', '校园', '节点', '校区', 'campus', 'map'],
    '语音': ['语音', '说话', '录音', '麦克风', 'TTS', 'ASR', 'voice'],
    '我的 / 个人中心': ['我的', '个人', '资料', '头像', '设置', 'profile', '个人信息'],
    '反馈系统': ['反馈', '意见', '投诉', 'feedback'],
    '消息 / 通知': ['通知', '消息', '提醒', '公告', 'notification'],
    '学生服务': ['学生', '学情', '打卡', '日记', '日报', '晨报', 'student'],
    '教务 / 课表': ['课表', '课程', '成绩', '选课', '考试', '排课', 'study', 'schedule'],
    '心理 / 情感': ['心理', '情感', '心情', '咨询', '焦虑', 'emotion', 'mental'],
    '管理端 / 数据': ['管理', '统计', '看板', '用户管理', '导入', 'admin', '仪表'],
  };
  final text = '${fb.content} ${fb.category} ${fb.resourceId}'
      .toLowerCase();
  final scored = <String, int>{};
  for (final entry in map.entries) {
    var score = 0;
    for (final kw in entry.value) {
      if (text.contains(kw.toLowerCase())) score++;
    }
    if (score > 0) scored[entry.key] = score;
  }
  final list = scored.entries.toList()
    ..sort((a, b) => b.value.compareTo(a.value));
  return list.take(4).map((e) => e.key).toList();
}

class _OnlineRepairSheet extends StatefulWidget {
  final FeedbackEntry fb;
  const _OnlineRepairSheet({required this.fb});

  @override
  State<_OnlineRepairSheet> createState() => _OnlineRepairSheetState();
}

class _OnlineRepairSheetState extends State<_OnlineRepairSheet> {
  AIRepairResult? _ai;
  bool _loading = false;
  bool _failed = false;
  String _localModule = '';
  List<String> _localFiles = const [];

  @override
  void initState() {
    super.initState();
    final local = _matchLocalCode(widget.fb);
    _localModule = local.isEmpty ? '' : local.first;
    _runAIRepair();
  }

  /// 本地关键词匹配：返回匹配的模块（前4个），并据此收集对应代码文件
  List<String> _matchLocalCode(FeedbackEntry fb) {
    final modules = _matchModuleNames(fb);
    final files = <String>[];
    for (final m in modules) {
      final f = _moduleFileMap[m];
      if (f != null) {
        for (final path in f) {
          if (!files.contains(path)) files.add(path);
        }
      }
    }
    _localFiles = files;
    return modules;
  }

  Future<void> _runAIRepair() async {
    setState(() => _loading = true);
    final result = await context
        .read<FeedbackProvider>()
        .aiRepair(widget.fb.feedbackId);
    if (!mounted) return;
    setState(() {
      _ai = result;
      _loading = false;
      _failed = result == null;
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final fb = widget.fb;
    return DraggableScrollableSheet(
      expand: false,
      initialChildSize: 0.72,
      maxChildSize: 0.94,
      builder: (context, scrollCtrl) => ListView(
        controller: scrollCtrl,
        padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
        children: [
          Row(
            children: [
              Icon(Icons.build_circle_outlined,
                  color: theme.colorScheme.primary, size: 22),
              const SizedBox(width: 8),
              Expanded(
                child: Text('在线修复助手',
                    style: theme.textTheme.titleLarge
                        ?.copyWith(fontWeight: FontWeight.bold)),
              ),
            ],
          ),
          const SizedBox(height: 4),
          Text('AI 解析截图 + 智能定位代码 · 截图经由智谱 GLM-4.6V',
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          const SizedBox(height: 16),

          // 问题摘要
          _sectionTitle(theme, '问题摘要'),
          Card(
            elevation: 0,
            color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.4),
            shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(12)),
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Wrap(
                    spacing: 6,
                    crossAxisAlignment: WrapCrossAlignment.center,
                    children: [
                      Text('ID: ${fb.feedbackId}  ·  ${fb.categoryLabel}',
                          style: theme.textTheme.labelMedium),
                      if (fb.module.isNotEmpty)
                        _chip(theme, fb.module, theme.colorScheme.primary),
                      if (_localModule.isNotEmpty &&
                          _localModule != fb.module)
                        _chip(theme, 'AI匹配:$_localModule',
                            theme.colorScheme.tertiary),
                    ],
                  ),
                  const SizedBox(height: 6),
                  Text(
                    _ai?.summary.isNotEmpty == true ? _ai!.summary : fb.content,
                    style: theme.textTheme.bodyMedium,
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),

          // 加载状态
          if (_loading) ...[
            Center(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  children: [
                    const CircularProgressIndicator(),
                    const SizedBox(height: 8),
                    Text('AI 正在解析截图与定位代码...',
                        style: theme.textTheme.bodySmall),
                  ],
                ),
              ),
            ),
          ] else ...[
            // AI 摘要 / OCR
            if (_ai != null && _ai!.ocrText.trim().isNotEmpty) ...[
              _sectionTitle(theme, '截图解析（OCR）'),
              Card(
                elevation: 0,
                color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.4),
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(12)),
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: SelectableText(_ai!.ocrText,
                      maxLines: 6,
                      style: theme.textTheme.bodySmall),
                ),
              ),
              const SizedBox(height: 16),
            ],
            if (_ai != null && _ai!.rootCause.isNotEmpty) ...[
              _sectionTitle(theme, '根因分析'),
              Card(
                elevation: 0,
                color: theme.colorScheme.tertiaryContainer.withOpacity(0.3),
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(12)),
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Text(_ai!.rootCause, style: theme.textTheme.bodyMedium),
                ),
              ),
              const SizedBox(height: 16),
            ],

            // 相关代码定位
            _sectionTitle(theme, '相关代码定位'),
            if (_ai == null && _localModule.isEmpty)
              const Card(
                elevation: 0,
                child: ListTile(
                  leading: Icon(Icons.search_off),
                  title: Text('未匹配到具体模块'),
                  subtitle: Text('可结合截图与描述人工定位'),
                ),
              )
            else
              for (final f in _codeFiles()) _fileCard(theme, f),
            const SizedBox(height: 16),

            // 修复建议
            _sectionTitle(theme, '修复建议'),
            Card(
              elevation: 0,
              color: theme.colorScheme.primaryContainer.withOpacity(0.25),
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12)),
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Text(_repairHint(), style: theme.textTheme.bodyMedium),
              ),
            ),
            const SizedBox(height: 16),

            // 失败提示
            if (_failed) ...[
              Row(
                children: [
                  Icon(Icons.info_outline,
                      size: 16, color: theme.colorScheme.error),
                  const SizedBox(width: 6),
                  Expanded(
                    child: Text('AI 诊断暂不可用，已展示本地关键词定位结果',
                        style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.error)),
                  ),
                ],
              ),
              const SizedBox(height: 12),
            ],

            // 本地复现指引
            _sectionTitle(theme, '本机复现与提交'),
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.4),
                borderRadius: BorderRadius.circular(12),
              ),
              child: const SelectableText(
                '1. 前端: cd frontend && flutter run -d chrome\n'
                '2. 后端: cd server && go run .   (本地 SQLite)\n'
                '3. 修复后: flutter analyze 零错误 → make deploy-release 发版\n'
                '4. 发版后回到本页将该反馈标记为「已解决」并回复用户',
                style: TextStyle(fontFamily: 'monospace', fontSize: 12),
              ),
            ),
            const SizedBox(height: 20),
          ],
        ],
      ),
    );
  }

  /// 待展示的代码文件（AI 判定优先，其次本地兜底）
  List<String> _codeFiles() {
    if (_ai != null && _ai!.codeFiles.isNotEmpty) return _ai!.codeFiles;
    if (_ai != null && _ai!.matchedFiles.isNotEmpty) return _ai!.matchedFiles;
    return _localFiles;
  }

  String _repairHint() {
    if (_ai != null && _ai!.repairHint.isNotEmpty) return _ai!.repairHint;
    switch (widget.fb.category) {
      case 'answer_error':
        return '回答有误类反馈：优先检查知识库资源内容（kb_resources）与检索结果，'
            '确认 Context Engine 的 FTS 分词与 role_scope 权限过滤是否排除正确内容；'
            '同时核对资源 content 是否准确、最新。';
      case 'suggestion':
        return '功能建议类反馈：在需求看板中登记，评估是否纳入近期迭代；'
            '可先以 flutter run -d web 验证相关页面现有交互。';
      default:
        return '通用反馈：建议结合截图定位界面，先在本地以 release 模式复现；'
            '若为接口/数据问题，用 trace_id 在服务器日志与 audit_logs 中追踪请求链路。';
    }
  }

  Widget _chip(ThemeData theme, String text, Color color) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: color.withOpacity(0.1),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(text,
          style: theme.textTheme.labelSmall?.copyWith(
              color: color, fontWeight: FontWeight.w600)),
    );
  }

  Widget _fileCard(ThemeData theme, String file) {
    return Card(
      elevation: 0,
      margin: const EdgeInsets.only(bottom: 8),
      color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.35),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: ListTile(
        dense: true,
        leading: const Icon(Icons.code, color: Colors.green),
        title: SelectableText(
          file,
          style: TextStyle(
            fontSize: 12,
            fontFamily: 'monospace',
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ),
    );
  }

  Widget _sectionTitle(ThemeData theme, String title) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Text(title,
          style: theme.textTheme.titleMedium
              ?.copyWith(fontWeight: FontWeight.bold)),
    );
  }
}