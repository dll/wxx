import 'package:flutter/material.dart';

import '../about/about_page.dart';
import '../../utils/storage.dart';

/// 帮助中心 —— 系统简介 + 各角色操作指南
///
/// 通过左侧主菜单「帮助」进入。以 Tab 方式组织：
/// 1. 系统入门（简介、架构、特色、亮点）
/// 2. 按角色（学生 / 辅导员 / 教师 / 教辅 / 学生会 / 管理员）的日常操作流程
/// 3. 关于
///
/// 首次进入自动定位到「当前角色」标签，便于用户快速找到自己的操作指南。
class HelpPage extends StatefulWidget {
  const HelpPage({super.key});

  @override
  State<HelpPage> createState() => _HelpPageState();
}

class _HelpPageState extends State<HelpPage> {
  late int _initialTab;

  /// 角色 → 页签顺序定位
  static int _tabForRole(String? role) {
    switch (role) {
      case 'student':
      case 'student_union':
        return 1; // 学生
      case 'counselor':
        return 2; // 辅导员
      case 'teacher':
        return 3; // 教师
      case 'assistant':
        return 4; // 教辅
      case 'college_admin':
      case 'school_admin':
      case 'sys_admin':
        return 6; // 管理员
      default:
        return 0; // 系统概览
    }
  }

  @override
  void initState() {
    super.initState();
    _initialTab = _tabForRole(Storage.role);
  }

  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: 8,
      initialIndex: _initialTab,
      child: Scaffold(
        appBar: AppBar(
          title: const Text('帮助'),
          bottom: const TabBar(
            isScrollable: true,
            tabAlignment: TabAlignment.start,
            tabs: [
              Tab(text: '系统概览'),
              Tab(text: '学生'),
              Tab(text: '辅导员'),
              Tab(text: '教师'),
              Tab(text: '教辅'),
              Tab(text: '学生会'),
              Tab(text: '管理员'),
              Tab(text: '关于'),
            ],
          ),
        ),
        body: const TabBarView(
          children: [
            _SystemOverviewTab(),
            _RoleGuideTab(role: 'student'),
            _RoleGuideTab(role: 'counselor'),
            _RoleGuideTab(role: 'teacher'),
            _RoleGuideTab(role: 'assistant'),
            _RoleGuideTab(role: 'student_union'),
            _RoleGuideTab(role: 'admin'),
            _AboutTab(),
          ],
        ),
      ),
    );
  }
}

// ─────────────────────────────────────────────
// 一、系统概览
// ─────────────────────────────────────────────
class _SystemOverviewTab extends StatelessWidget {
  const _SystemOverviewTab();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _sectionTitle(theme, '一、蔚小芯是什么'),
        _card(theme, [
          _row(theme, Icons.school, '定位',
              '蔚小芯是滁州学院计算机科学与工程学院（网络空间安全学院）学生工作数字化智能体，'
                  '面向「入学—在校—离校—就业」全生命周期，融合政治、思想、心理、专业四位一体育人。'),
          _row(theme, Icons.auto_awesome, 'AI 原生',
              '不是简单的信息查询器，而是用大模型生成日报、学习日记、数字孪生画像、心理陪伴与个性化建议的 AI 育人伙伴。'),
        ]),

        _sectionTitle(theme, '二、技术架构（前后端与大模型）'),
        _card(theme, [
          _row(theme, Icons.storage, '前端',
              'Flutter 跨端应用：Web 浏览器可直接访问 + Android APK，一套代码两端一致。'),
          _row(theme, Icons.dns, '后端',
              'Go / Gin 服务，部署于腾讯云 Lighthouse 常驻服务器，本地 SQLite 数据库（含全文检索 FTS5）。'),
          _row(theme, Icons.public, '入口',
              '正式入口 https://wxx-agent.online，备用 https://wxx-agent.pages.dev，APK 与首页二维码均可打开。'),
          _row(theme, Icons.smart_toy, '大模型',
              '接入智谱、DeepSeek、讯飞等第三方大模型：语义问答、日报、学习日记、性格洞察、出题评分等均由模型完成；语音识别 / 合成由讯飞提供。'),
        ]),

        _sectionTitle(theme, '三、核心特色与亮点'),
        _card(theme, [
          _row(theme, Icons.radar, '数字孪生画像',
              '五维画像（学业 / 能力 / 思想 / 情感 / 社交）可视化，让成长可量化、可追踪、可对比。'),
          _row(theme, Icons.calendar_view_day, '每日陪伴',
              'AI 今日速览、学习日记、每日打卡，让系统每天都有「值得打开」的理由。'),
          _row(theme, Icons.forum, '社区共创',
              '问答广场 + 热点追踪 + 排行榜，学生提问、AI 先答、师生共创知识库。'),
          _row(theme, Icons.psychology, '心理关怀',
              '心情打卡 + 正念引导 + 危机转介，心理健康可量化、可疏导。'),
          _row(theme, Icons.book, '档案治理',
              '思想政治成长档案 + 入党入团进度 + 学习周报，成长全程留痕。'),
          _row(theme, Icons.map, '报到与办事',
              '报到节点坐标地图 + 办事流程手把手引导，地点、联系人、材料一目了然。'),
          _row(theme, Icons.dashboard, '分级管理',
              '学生 / 辅导员 / 教师 / 教辅 / 各级管理员，各有专属 AI 工作台与数据看板。'),
        ]),
        const SizedBox(height: 4),
      ],
    );
  }

  Widget _sectionTitle(ThemeData theme, String text) {
    return Padding(
      padding: const EdgeInsets.only(top: 12, bottom: 8),
      child: Text(text,
          style:
              theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
    );
  }

  Widget _card(ThemeData theme, List<Widget> children) {
    return Card(
      elevation: 0,
      margin: const EdgeInsets.only(bottom: 8),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: children,
        ),
      ),
    );
  }

  Widget _row(ThemeData theme, IconData icon, String title, String text) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 20, color: theme.colorScheme.primary),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title,
                    style: theme.textTheme.titleSmall
                        ?.copyWith(fontWeight: FontWeight.w600)),
                const SizedBox(height: 2),
                Text(text, style: theme.textTheme.bodyMedium),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

// ─────────────────────────────────────────────
// 二、角色操作指南
// ─────────────────────────────────────────────
class _RoleGuideTab extends StatelessWidget {
  final String role;
  const _RoleGuideTab({required this.role});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final title = _roleTitle(role);
    final summary = _roleSummary(role);
    final sections = _roleSections(role);
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          elevation: 0,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
            side: BorderSide(
                color: theme.colorScheme.primary.withOpacity(0.3)),
          ),
          child: Container(
            decoration: BoxDecoration(
              gradient: LinearGradient(
                colors: [
                  theme.colorScheme.primaryContainer.withOpacity(0.4),
                  theme.colorScheme.surface,
                ],
              ),
              borderRadius: BorderRadius.circular(12),
            ),
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(children: [
                  Icon(Icons.school, color: theme.colorScheme.primary),
                  const SizedBox(width: 8),
                  Text('$title工作台',
                      style: theme.textTheme.titleMedium
                          ?.copyWith(fontWeight: FontWeight.bold)),
                ]),
                const SizedBox(height: 8),
                Text(summary,
                    style: theme.textTheme.bodyMedium?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant)),
              ],
            ),
          ),
        ),
        const SizedBox(height: 16),
        ...sections.map(
          (s) => Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: _roleSectionCard(theme, s),
          ),
        ),
        const SizedBox(height: 4),
        Center(
          child: Text(
            '说明：操作顺序仅供参考，实际以页面内按钮与提示为准。',
            style: theme.textTheme.bodySmall
                ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
          ),
        ),
      ],
    );
  }

  Widget _roleSectionCard(ThemeData theme, _RoleSection s) {
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Icon(s.icon, size: 20, color: theme.colorScheme.primary),
              const SizedBox(width: 8),
              Expanded(
                child: Text(s.title,
                    style: theme.textTheme.titleSmall
                        ?.copyWith(fontWeight: FontWeight.w600)),
              ),
            ]),
            const SizedBox(height: 6),
            ...s.steps.map(
              (st) => Padding(
                padding: const EdgeInsets.symmetric(vertical: 3),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('•  ',
                        style: TextStyle(
                            color: theme.colorScheme.primary,
                            fontWeight: FontWeight.bold)),
                    Expanded(child: Text(st, style: theme.textTheme.bodyMedium)),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ─────────────────────────────────────────────
// 三、关于
// ─────────────────────────────────────────────
class _AboutTab extends StatelessWidget {
  const _AboutTab();

  @override
  Widget build(BuildContext context) {
    return const AboutPage();
  }
}

// ─────────────────────────────────────────────
// 角色指南数据
// ─────────────────────────────────────────────
class _RoleSection {
  final String title;
  final IconData icon;
  final List<String> steps;
  const _RoleSection(this.title, this.icon, this.steps);
}

String _roleTitle(String role) {
  switch (role) {
    case 'student':
      return '学生';
    case 'counselor':
      return '辅导员';
    case 'teacher':
      return '教师';
    case 'assistant':
      return '教辅';
    case 'student_union':
      return '学生会';
    case 'admin':
      return '管理员';
    default:
      return '用户';
  }
}

String _roleSummary(String role) {
  switch (role) {
    case 'student':
      return '作为一名学生，蔚小芯是你的 AI 成长伙伴：每天速览、学习日记、打卡、数字孪生画像、'
          '心理陪伴、课程学情、社区问答与职业探索一应俱全。';
    case 'counselor':
      return '身为辅导员/班主任，蔚小芯是你的 AI 增效助手：每日关注、风险预警、谈心谈话记录、'
          '学生数字孪生看板与班级画像，让育人决策有数据支撑。';
    case 'teacher':
      return '作为授课教师，蔚小芯是你的教学协同伙伴：备课、出题、批改、课堂互动、'
          '班级学情热力图与教学反思，构建完整教学闭环。';
    case 'assistant':
      return '作为教务教辅，蔚小芯聚焦「排课、考试、毕业」三大刚需场景的 AI 流程自动化。';
    case 'student_union':
      return '作为学生会/班团委，蔚小芯是你的内容运营引擎：AI 策划、海报生成、'
          '活动数据分析与知识提交管理。';
    case 'admin':
      return '作为学院/学校/系统管理员，蔚小芯是你的数据决策中枢：数字孪生大屏、全量数据分析、'
          '用户与知识治理、审计日志与全局配置。';
    default:
      return '登录后按角色加载对应功能。';
  }
}

List<_RoleSection> _roleSections(String role) {
  switch (role) {
    case 'student':
      return const [
        _RoleSection('日常使用流程', Icons.calendar_view_day, [
          '打开正式入口 wxx-agent.online 或安装蔚小芯.apk，用学号 + 初始密码登录。',
          '首次登录建议先到「我的」→ 修改密码、完善个人资料与头像。',
          '每天首次打开「首页」查看 AI 今日速览；在「对话」中随时向 AI 提问。',
          '在「我的 → AI 学习工具」使用学习日记、每日打卡，形成每日学习闭环。',
          '定期查看「数字孪生」画像，了解五维状态与成长差距。',
        ]),
        _RoleSection('学业与成长', Icons.school, [
          '学业：课程地图、课程学情、周报、学习计划与课表。',
          '能力：竞赛匹配、AI 学伴、模拟面试、简历、职业探索。',
          '思想：思政学习、思想成长档案、入党入党进度追踪。',
          '心理：心情打卡、心理测评、咨询预约、危机转介。',
          ]),
        _RoleSection('社区与办事', Icons.forum, [
          '在「问答广场」提问、回答、点赞；在「热点关注」了解全校动态。',
          '在「办事」页按引导流程办理入校/离校/日常事项，实时查看办理进度。',
          '校园报到：在「报到地图」查看各节点地点、联系人，扫码或现场办理。',
          ]),
        _RoleSection('离校与就业', Icons.work, [
          '大四阶段使用「毕业设计选题」「就业指导」择岗与政策查询。',
          '用数字孪生数据生成简历，进行模拟面试与职业探索。',
          ]),
      ];
    case 'counselor':
      return const [
        _RoleSection('每日关注', Icons.wb_sunny, [
          '登录后查看「今日关注」：AI 推送今日需关注学生 TOP5 及建议动作。',
          '查看「班级学情日报」，掌握全院/年级整体学习活跃度与异常。',
          ]),
        _RoleSection('重点学生辅导', Icons.psychology, [
          '进入「数字孪生看板」聚合查看学生多维状态，异常自动高亮。',
          '对预警/异常学生使用「预测预警」与「干预方案」，AI 生成谈话要点与干预建议。',
          '开展谈心谈话时用「谈话记录」实时转写、结构化摘要并自动归档。',
          ]),
        _RoleSection('班级与学生档案', Icons.groups, [
          '查看「学生思想档案」「班级性格画像」「班级打卡统计」。',
          '在「学生列表」查看与筛选所管学生，跟进重点学生。',
          ]),
        _RoleSection('管理与协同', Icons.manage_accounts, [
          '协作「知识库」创建/审核，参与学生会提交的知识审批。',
          '管理「社区问答」，标记官方回应，感知热点话题；维护流程步骤。',
          ]),
      ];
    case 'teacher':
      return const [
        _RoleSection('课前', Icons.event_repeat, [
          '查看「每日授课概览」了解今天授课信息与上次反思摘要。',
          '使用「备课助手」生成教学大纲、重难点与课堂互动设计。',
          ]),
        _RoleSection('带中', Icons.record_voice_over, [
          '课堂互动：随机点名、即时答题、结果可视化并汇入学生学情。',
          ]),
        _RoleSection('课后', Icons.assignment, [
          '批改作业：客观题自动批改、主观题 AI 初建，你审核后发布。',
          '查看「班级学情热力图」，定位薄弱知识点与需关注学生。',
          ]),
        _RoleSection('改进与协同', Icons.trending_up, [
          '结合学情「教学反思」，生成下节课改进建议。',
          '在「风格分布」了解班级学习风格分布，适配教学策略，并在「社区专业答疑」提供答案。',
          ]),
      ];
    case 'assistant':
      return const [
        _RoleSection('排课', Icons.schedule, [
          '使用「排课冲突检测」检查教师/教室/教室/班级/时间/容量冲突并优化。',
          ]),
        _RoleSection('考试', Icons.assignment_turned_in, [
          '「考试安排优化」按考场、人数、时间、监考教师自动排考。',
          ]),
        _RoleSection('毕业', Icons.school, [
          '「毕业资格审核」自动比对培养方案与成绩，标记达标/不达标与待修补清单。',
          ]),
      ];
    case 'student_union':
      return const [
        _RoleSection('内容创作', Icons.brush, [
          '「活动策划助手」输入主题生成策划案，一键生成活动通知。',
          '「海报/通知一键生成」生成多风格海报与多渠道文案。',
          ]),
        _RoleSection('知识运营', Icons.menu_book, [
          '「知识提交」提交知识资源（草稿→待审），追踪提交状态与查看错题反馈。',
          ]),
        _RoleSection('活动数据', Icons.analytics, [
          '「活动数据分析」自动分析报名率、到场率、反馈并生成复盘报告。',
          ]),
      ];
    case 'admin':
      return const [
        _RoleSection('数据总览', Icons.insights, [
          '查看「数字孪生大屏」，一屏掌握全院/全校学生学业、情感、活动、风险分布。',
          '在「数据分析」自然语言查询数据，生成交互式可视化报告与决策建议。',
          ]),
        _RoleSection('用户与权限', Icons.admin_panel_settings, [
          '「用户管理」创建/查看/导入用户、分配角色、重置密码。',
          '「权限与能力」按角色自动继承授权，管理员可管理用户角色。',
          ]),
        _RoleSection('知识与内容', Icons.library_books, [
          '「知识库管理」CRUD 知识资源、文档解析精修、批量审核/废弃。',
          '「内容治理」审核评价、反馈处理、问题预案（forecast）闭环。',
          ]),
        _RoleSection('系统与审计', Icons.security, [
          '「系统配置」模型参数、限流、功能开关、游客注册开关。',
          '「审计日志」查看操作记录；「质量看板」看使用指标与 Token 统计。',
          ]),
        _RoleSection('报到节点', Icons.map, [
          '在「报到地图」对节点坐标进行拖拽校正、发布、启停、功能。',
          ]),
      ];
    default:
      return const [];
  }
}

