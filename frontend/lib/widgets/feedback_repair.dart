// 管理员「在线修复」代码定位助手
// 反馈详情内置：基于反馈内容与分类，自动定位项目中可能相关的代码文件/模块，
// 并给出修复建议，帮助管理员在本机（具备开发环境与项目代码）快速定位并修复问题。
import 'package:flutter/material.dart';
import '../../models/models.dart';

/// 功能模块 → 项目代码文件映射（用于反馈问题定位）。
/// key 为模块名，files 为对应代码文件路径（相对仓库根），keywords 用于关键词匹配。
class _ModuleFileMap {
  final String module;
  final List<String> files;
  final List<String> keywords;
  const _ModuleFileMap(this.module, this.files, this.keywords);
}

const _moduleMap = <_ModuleFileMap>[
  _ModuleFileMap('登录 / 认证', [
    'frontend/lib/pages/login/login_page.dart',
    'frontend/lib/providers/auth_provider.dart',
    'server/internal/handler/auth_handler.go',
  ], ['登录', '认证', '账号', '密码', '扫码', 'token', '验证码', 'login', 'auth']),
  _ModuleFileMap('对话 / 问答', [
    'frontend/lib/pages/chat/chat_page.dart',
    'frontend/lib/providers/chat_provider.dart',
    'server/internal/service/chat_service.go',
    'server/internal/context_engine/engine.go',
  ], ['回答', '问答', '对话', '聊天', '回复', '答案', 'chat', 'AI', '智能']),
  _ModuleFileMap('知识库 / 检索', [
    'server/internal/context_engine/',
    'server/internal/repository/kb_repo.go',
    'frontend/lib/pages/knowledge/',
  ], ['知识', '检索', '搜索', '词条', '知识库', 'FTS', '搜索结果', '查不到']),
  _ModuleFileMap('办事流程', [
    'frontend/lib/pages/process/',
    'server/internal/handler/process_handler.go',
    'frontend/lib/providers/process_provider.dart',
  ], ['办事', '流程', '手续', '申请', '审批', 'process']),
  _ModuleFileMap('报到 / 校园导航', [
    'frontend/lib/pages/campus/campus_map_page.dart',
    'frontend/lib/widgets/baidu_campus_map_embed_web.dart',
    'frontend/lib/widgets/baidu_campus_map_embed_android.dart',
    'server/internal/handler/campus_handler.go',
  ], ['报到', '地图', '导航', '校园', '节点', '校区', 'campus', 'map']),
  _ModuleFileMap('语音', [
    'frontend/lib/services/voice/voice_service_web.dart',
    'frontend/lib/services/voice/voice_service_mobile.dart',
    'frontend/lib/pages/chat/chat_page.dart',
    'server/internal/handler/voice_handler.go',
  ], ['语音', '说话', '录音', '麦克风', 'TTS', 'ASR', 'voice']),
  _ModuleFileMap('我的 / 个人中心', [
    'frontend/lib/pages/profile/profile_page.dart',
    'frontend/lib/providers/auth_provider.dart',
  ], ['我的', '个人', '资料', '头像', '设置', 'profile', '个人信息']),
  _ModuleFileMap('反馈系统', [
    'frontend/lib/pages/admin/feedback_page.dart',
    'frontend/lib/providers/feedback_provider.dart',
    'server/internal/handler/feedback_handler.go',
  ], ['反馈', '意见', '投诉', 'feedback']),
  _ModuleFileMap('消息 / 通知', [
    'frontend/lib/pages/notifications/',
    'server/internal/handler/notification_handler.go',
  ], ['通知', '消息', '提醒', '公告', 'notification']),
  _ModuleFileMap('学生服务', [
    'frontend/lib/pages/student/',
    'server/internal/service/student_service.go',
  ], ['学生', '学情', '打卡', '日记', '日报', '晨报', 'student']),
  _ModuleFileMap('教务 / 课表', [
    'frontend/lib/pages/student/',
    'server/internal/service/study_service.go',
  ], ['课表', '课程', '成绩', '选课', '考试', '排课', 'study', 'schedule']),
  _ModuleFileMap('心理 / 情感', [
    'frontend/lib/pages/student/mental/',
    'server/internal/service/emotion_service.go',
  ], ['心理', '情感', '心情', '咨询', '焦虑', 'emotion', 'mental']),
  _ModuleFileMap('管理端 / 数据', [
    'frontend/lib/pages/admin/',
    'server/internal/handler/admin_handler.go',
  ], ['管理', '统计', '看板', '用户管理', '导入', 'admin', '仪表']),
];

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

/// 根据反馈内容匹配最相关的代码模块
List<_ModuleFileMap> _matchModules(FeedbackEntry fb) {
  final text = '${fb.content} ${fb.category} ${fb.resourceId}'.toLowerCase();
  final scored = <_ModuleFileMap, int>{};
  for (final m in _moduleMap) {
    var score = 0;
    for (final kw in m.keywords) {
      if (text.contains(kw)) score++;
    }
    if (score > 0) scored[m] = score;
  }
  final list = scored.entries.toList()
    ..sort((a, b) => b.value.compareTo(a.value));
  return list.take(4).map((e) => e.key).toList();
}

/// 分类修复建议
String _repairHint(FeedbackEntry fb) {
  switch (fb.category) {
    case 'answer_error':
      return '回答有误类反馈：优先检查知识库资源内容（kb_resources）与该问题对应的检索结果，'
          '确认 Context Engine 的 FTS 分词与 role_scope 权限过滤是否将正确资源排除；'
          '同时核对该资源 content 是否准确、最新。';
    case 'suggestion':
      return '功能建议类反馈：在需求看板中登记，评估是否纳入近期迭代；'
          '可先在本机以 flutter run -d chrome 验证相关页面现有交互。';
    default:
      return '通用反馈：建议结合截图定位界面，先在本地以 release 模式复现；'
          '若为接口/数据问题，用 trace_id 在服务器日志与 audit_logs 中追踪请求链路。';
  }
}

class _OnlineRepairSheet extends StatefulWidget {
  final FeedbackEntry fb;
  const _OnlineRepairSheet({required this.fb});

  @override
  State<_OnlineRepairSheet> createState() => _OnlineRepairSheetState();
}

class _OnlineRepairSheetState extends State<_OnlineRepairSheet> {
  late final List<_ModuleFileMap> _modules;

  @override
  void initState() {
    super.initState();
    _modules = _matchModules(widget.fb);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final fb = widget.fb;
    return DraggableScrollableSheet(
      expand: false,
      initialChildSize: 0.7,
      maxChildSize: 0.92,
      builder: (context, scrollCtrl) => ListView(
        controller: scrollCtrl,
        padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
        children: [
          Row(
            children: [
              Icon(Icons.build_circle_outlined,
                  color: theme.colorScheme.primary, size: 22),
              const SizedBox(width: 8),
              Text('在线修复助手',
                  style: theme.textTheme.titleLarge
                      ?.copyWith(fontWeight: FontWeight.bold)),
            ],
          ),
          const SizedBox(height: 4),
          Text('基于反馈内容定位相关代码，在本机开发环境中直接修复',
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
                  Text('ID: ${fb.feedbackId}  ·  ${fb.categoryLabel}',
                      style: theme.textTheme.labelMedium),
                  const SizedBox(height: 6),
                  Text(fb.content, style: theme.textTheme.bodyMedium),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),

          // 代码定位
          _sectionTitle(theme, '相关代码定位'),
          if (_modules.isEmpty)
            Card(
              elevation: 0,
              child: ListTile(
                leading: const Icon(Icons.search_off),
                title: const Text('未匹配到具体模块'),
                subtitle: const Text('可结合反馈截图与分类，在下方建议中人工定位'),
              ),
            )
          else
            for (final m in _modules)
              Card(
                elevation: 0,
                margin: const EdgeInsets.only(bottom: 8),
                color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.35),
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(12)),
                child: ExpansionTile(
                  leading: Icon(Icons.folder_outlined,
                      color: theme.colorScheme.primary),
                  title: Text(m.module,
                      style: const TextStyle(fontWeight: FontWeight.w600)),
                  children: [
                    Padding(
                      padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          for (final f in m.files)
                            Padding(
                              padding: const EdgeInsets.only(bottom: 4),
                              child: Row(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Icon(Icons.code,
                                      size: 13,
                                      color:
                                          theme.colorScheme.onSurfaceVariant),
                                  const SizedBox(width: 6),
                                  Expanded(
                                    child: SelectableText(
                                      f,
                                      style: TextStyle(
                                        fontSize: 12,
                                        fontFamily: 'monospace',
                                        color:
                                            theme.colorScheme.onSurfaceVariant,
                                      ),
                                    ),
                                  ),
                                ],
                              ),
                            ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
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
              child: Text(_repairHint(fb), style: theme.textTheme.bodyMedium),
            ),
          ),
          const SizedBox(height: 16),

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
