import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import '../../config/release_config.dart';
import '../../config/api_config.dart';
import '../../main.dart';
import '../../models/models.dart';
import '../../providers/auth_provider.dart';
import '../../providers/admin_provider.dart';
import '../../services/api_service.dart';
import '../../utils/capability_utils.dart';
import '../../utils/role_utils.dart';
import '../../utils/storage.dart';
import '../../widgets/error_view.dart';
import '../../widgets/personal_detail_dialog.dart';
import '../../widgets/student_interest_pick_dialog.dart';

class _ProfileFeature {
  final String key;
  final String category;
  final IconData icon;
  final String title;
  final String subtitle;
  final String route;
  /// true 时仅在「功能开关」Tab 出现（不渲染为普通功能卡片）；默认 false
  final bool switchOnly;

  const _ProfileFeature(this.key, this.category, this.icon, this.title,
      this.subtitle, this.route,
      {this.switchOnly = false});
}

/// 个人中心页
class ProfilePage extends StatefulWidget {
  const ProfilePage({super.key});

  @override
  State<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends State<ProfilePage> {
  @override
  void initState() {
    super.initState();
    // 加载全局功能开关（管理员在后端配置，登录用户读取）
    _loadGlobalFeatureSwitches();
    // 加载用户资料
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) context.read<AuthProvider>().fetchProfile();
    });
  }

  /// 拉取管理员全局功能开关并缓存到本地
  Future<void> _loadGlobalFeatureSwitches() async {
    try {
      final res = await ApiService().get(ApiConfig.publicFeatureSwitches);
      if (res.statusCode == 200 && res.data is Map) {
        final data = (res.data as Map)['data'];
        if (data is Map) {
          final switches = <String, String>{};
          data.forEach((k, v) => switches['$k'] = '$v');
          await Storage.setGlobalFeatureSwitches(switches);
          if (mounted) setState(() {});
        }
      }
    } catch (_) {
      // 网络异常时沿用本地缓存开关
    }
  }

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthProvider>();
    final theme = Theme.of(context);
    final profile = auth.profile;

    return Scaffold(
      appBar: AppBar(title: const Text('个人中心')),
      body: _buildBody(context, auth, theme, profile),
    );
  }

  Widget _buildBody(BuildContext context, AuthProvider auth, ThemeData theme,
      UserProfile? profile) {
    // 加载中（首次，非登录流程）
    if (auth.loading && profile == null) {
      return const Center(child: CircularProgressIndicator());
    }

    // 加载失败且无缓存数据
    if (auth.error != null && profile == null) {
      return ErrorView.error(
        message: auth.error!,
        onRetry: () => auth.fetchProfile(),
      );
    }

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        // 用户信息卡片（年级主题渐变）
        _buildUserHeaderCard(context, profile),

        const SizedBox(height: 16),

        // 年级主题横幅（让年级色彩直观可见）
        _buildGradeThemeBanner(context),

        const SizedBox(height: 8),

        // 信息列表
        if (profile != null) ...[
          _buildInfoTile(
              context, Icons.badge_outlined, '学号/工号', profile.username),
          if (profile.college.isNotEmpty)
            _buildInfoTile(
                context, Icons.account_balance_outlined, '学院', profile.college),
          if (profile.major.isNotEmpty)
            _buildInfoTile(context, Icons.book_outlined, '专业', profile.major),
          if (profile.className.isNotEmpty)
            _buildInfoTile(
                context, Icons.groups_outlined, '班级', profile.className),
          if (profile.enrollmentYear.isNotEmpty)
            _buildInfoTile(context, Icons.calendar_month_outlined, '入学年份',
                profile.enrollmentYear),
          if (profile.gender.isNotEmpty)
            _buildInfoTile(context, Icons.wc_outlined, '性别', profile.gender),
          if (profile.campus.isNotEmpty)
            _buildInfoTile(
                context, Icons.location_city_outlined, '校区', profile.campus),
          if (profile.educationLevel.isNotEmpty)
            _buildInfoTile(
                context, Icons.school_outlined, '学历层次', profile.educationLevel),
          if (profile.ethnicity.isNotEmpty)
            _buildInfoTile(
                context, Icons.people_outline, '民族', profile.ethnicity),
          if (profile.politicalStatus.isNotEmpty)
            _buildInfoTile(context, Icons.how_to_reg_outlined, '政治面貌',
                profile.politicalStatus),
        ],

        const SizedBox(height: 24),

        // 我的操作日志（所有角色独立入口，非分组菜单）
        Card(
          elevation: 0,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
            side: BorderSide(color: theme.colorScheme.outlineVariant),
          ),
          child: ListTile(
            leading: Icon(Icons.history, color: theme.colorScheme.primary),
            title: const Text('我的操作日志'),
            subtitle: const Text('查看自己的操作记录'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.go('/my-logs'),
          ),
        ),

        const SizedBox(height: 8),

        // 多角色切换（2026-09-01）：用户具备多个角色时显示
        _buildRoleSwitcher(context, auth),

        const SizedBox(height: 8),

        // 学校门户（独立入口：登录校内网站）
        Card(
          elevation: 0,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
            side: BorderSide(color: theme.colorScheme.outlineVariant),
          ),
          child: ListTile(
            leading: Icon(Icons.school, color: const Color(0xFF1565C0)),
            title: const Text('学校门户'),
            subtitle: const Text('登录校内网站，访问学工/一表通等系统'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.go('/portal'),
          ),
        ),

        const SizedBox(height: 16),

        // 语音功能开关
        _buildVoiceToggle(context, auth),

        const SizedBox(height: 16),

        // 主题模式切换
        _buildThemeSection(context),

        const SizedBox(height: 16),

        // 年级主题自动切换开关
        _buildGradeThemeToggle(context),

        const SizedBox(height: 16),

        // 我的关注（学生：定制首页学生专区排序，可随时修改）
        if (profile?.role == 'student' ||
            profile?.role == 'student_union') ...[
          _studentInterestsCard(theme),
          const SizedBox(height: 16),
        ],

        // 数字人形象显示开关（可系统设置）
        _buildAvatarToggle(context),

        const SizedBox(height: 16),

        _buildFeatureTabs(context, profile?.role),

        const SizedBox(height: 16),

        // 智能体管理入口（管理员可访问）
        if (profile?.role == '__legacy_flat_menu__') ...[
          if (_canAccessAgents(profile?.role))
            _buildMenuCard(context, Icons.smart_toy_outlined, '智能体管理',
                '管理 AI 智能体的注册、配置和状态', '/agents'),

          // 质量看板（college_admin 及以上）
          if (_canAccessAdmin(profile?.role))
            _buildMenuCard(context, Icons.dashboard_outlined, '质量看板',
                '查看系统问答质量指标', '/admin/metrics'),

          // 数据底座导入：管理员 或 具备批量导入权限的学生会/教辅/辅导员
          if (_canAccessAdmin(profile?.role) ||
              CapabilityUtils.has(Capability.batchScheduleImport))
            _buildMenuCard(context, Icons.storage_outlined, '数据底座导入',
                '批量导入成绩与课表', '/admin/data-import'),

          // 用户管理：非学生组织角色可导入，管理员可继续维护账号。
          if (_canAccessUserManagement())
            _buildMenuCard(
              context,
              Icons.people_outline,
              '用户管理',
              CapabilityUtils.has(Capability.collegeUserRead)
                  ? '导入学生并管理账号、角色和状态'
                  : '通过 Excel 批量导入学生账号',
              '/admin/users',
            ),

          // 审计日志（college_admin 及以上）
          if (_canAccessAdmin(profile?.role))
            _buildMenuCard(
                context, Icons.history, '审计日志', '查看系统操作记录', '/admin/audit'),

          // 系统配置（sys_admin 独占）
          if (profile?.role == 'sys_admin')
            _buildMenuCard(context, Icons.settings_outlined, '系统配置', '管理系统运行参数',
                '/admin/settings'),

          // AI 简讯管理（sys_admin 独占）
          if (profile?.role == 'sys_admin')
            _buildMenuCard(context, Icons.newspaper, 'AI 简讯管理',
                '资讯 CRUD、来源抓取与导出', '/admin/ai-briefings'),

          // 问题预案（sys_admin、college_admin 可访问）
          if (profile?.role == 'sys_admin' || profile?.role == 'college_admin')
            _buildMenuCard(context, Icons.warning_amber_rounded, '问题预案',
                '查看和处理系统预警问题', '/forecast'),

          // 游客审核（college_admin 及以上）
          if (_canAccessAdmin(profile?.role))
            _buildMenuCard(context, Icons.person_search_outlined, '游客审核',
                '审核待处理的游客注册申请', '/admin/guests'),

          // 知识审核（counselor 及以上）
          if (_canAccessEmotion(profile?.role))
            _buildMenuCard(context, Icons.rate_review_outlined, '知识审核',
                '审核待发布的知识资源', '/review'),

          // 办事管理/审核（counselor 及以上，学校、学院管理员继承）
          if (CapabilityUtils.has(Capability.counselorKbWrite))
            _buildMenuCard(context, Icons.edit_note, '办事管理', '新增、编辑、发布和导出办事流程',
                '/process-manage'),
          if (CapabilityUtils.has(Capability.counselorKbReview))
            _buildMenuCard(context, Icons.rate_review_outlined, '办事审核',
                '审核学校、学院管理员提交的办事流程', '/process-review'),

          // 我的提交（student_union 及以上）
          if (_canSubmitKB(profile?.role))
            _buildMenuCard(context, Icons.note_add_outlined, '知识提交',
                '创建和管理知识资源', '/my-submissions'),

          // 词元统计（所有角色可访问自己的统计）
          _buildMenuCard(
              context, Icons.bar_chart, '词元统计', '查看 AI 词元消耗统计', '/token-stats'),

          // 我的收藏（所有角色可访问）
          _buildMenuCard(
              context, Icons.star_outline, '我的收藏', '查看已收藏的问答记录', '/bookmarks'),

          // 个人信息（弹窗：基本信息/联系方式/组织关系/学校门户绑定）
          Card(
            elevation: 0,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(12),
              side: BorderSide(
                  color: Theme.of(context).colorScheme.outlineVariant),
            ),
            child: ListTile(
              leading: Icon(Icons.badge_outlined,
                  color: Theme.of(context).colorScheme.primary),
              title: const Text('个人信息'),
              subtitle: const Text('基本信息 · 联系方式 · 组织关系 · 学校门户绑定'),
              trailing: const Icon(Icons.chevron_right),
              onTap: () => PersonalDetailDialog.show(context),
            ),
          ),

          // 我的反馈（所有角色可访问，查看自己提交的反馈与处理结果）
          _buildMenuCard(context, Icons.rate_review, '我的反馈', '查看自己提交的反馈与处理状态',
              '/my-feedbacks'),

          // 我的办事记录（所有角色可访问，查看自己的办事流程进度）
          _buildMenuCard(context, Icons.assignment_turned_in_outlined, '我的办事记录',
              '查看入学/离校等流程办理进度', '/my-records'),

          // AI 模型配置（所有角色可访问）
          _buildMenuCard(context, Icons.tune, 'AI 模型配置',
              '配置 DeepSeek / 智谱 / 讯飞星火模型参数', '/profile/model-config'),

          // ── 校园文化智能体（全员可见）──
          _buildMenuCard(context, Icons.music_note, '校歌曲库', '校歌、院歌与经典曲目',
              '/culture/anthems'),
          _buildMenuCard(
              context, Icons.podcasts, '校园广播', '直播节目单与往期回放', '/culture/radio'),
          _buildMenuCard(context, Icons.school_outlined, '学术讲座', '即将开始的讲座与回放',
              '/culture/lectures'),
          _buildMenuCard(context, Icons.celebration_outlined, '校园活动',
              '活动报名与个性推送', '/culture/events'),
          _buildMenuCard(context, Icons.volunteer_activism, '志愿服务', '志愿时长与项目推荐',
              '/culture/volunteer'),
          _buildMenuCard(
              context, Icons.apps_outlined, '应用中心', '第三方应用与链接', '/apps'),
          _buildMenuCard(context, Icons.newspaper, 'AI 简讯', 'AI 教学/工具/版本/行业热点',
              '/ai-briefings'),

          // 反馈管理（管理员）
          if (_canAccessAdmin(profile?.role))
            _buildMenuCard(context, Icons.feedback_outlined, '反馈管理',
                '查看和处理用户反馈', '/feedback'),

          // ── 角色 AI 功能入口 ──
          // 学生 AI 功能
          if (profile?.role == 'student' ||
              profile?.role == 'student_union') ...[
            _buildMenuCard(context, Icons.wb_sunny_outlined, '今日速览',
                'AI 每日学习概览', '/student/daily-briefing'),
            _buildMenuCard(context, Icons.auto_stories, '学习日记', 'AI 自动生成学习日记',
                '/student/learning-diary'),
            _buildMenuCard(context, Icons.check_circle_outline, '每日打卡',
                '学习打卡与连续记录', '/student/checkin'),
            _buildMenuCard(context, Icons.person_pin, '数字孪生', '我的数字画像',
                '/student/digital-twin'),
            _buildMenuCard(context, Icons.psychology_outlined, '性格洞察',
                'AI 性格分析', '/student/personality'),
            _buildMenuCard(context, Icons.emoji_events_outlined, '积分成就',
                '学习积分与成就', '/student/achievements'),
            _buildMenuCard(context, Icons.map_outlined, '课程地图', '课程学习路径',
                '/student/course-map'),
            _buildMenuCard(context, Icons.analytics_outlined, '课程学情', '课程学习分析',
                '/student/course-analytics'),
            _buildMenuCard(context, Icons.summarize_outlined, '学习周报',
                'AI 周度学习总结', '/student/weekly-report'),
            _buildMenuCard(context, Icons.forum_outlined, '问答广场', '校园问答社区',
                '/student/qa-plaza'),
            _buildMenuCard(context, Icons.local_fire_department, '热点关注',
                '校园热点话题', '/student/hot-topics'),
            _buildMenuCard(context, Icons.leaderboard_outlined, '问答排行',
                '问答贡献排行榜', '/student/qa-leaderboard'),
            _buildMenuCard(context, Icons.chat_outlined, '站内私聊', 'AI 学伴私信',
                '/student/private-chat'),
            _buildMenuCard(context, Icons.account_tree_outlined, 'AI 办事流程',
                '智能流程引导', '/enrollment'),
            _buildMenuCard(context, Icons.timeline, '成长路径', 'AI 个性化成长规划',
                '/student/growth-path'),
            _buildMenuCard(context, Icons.flag_outlined, '新生规划', '大学四年规划蓝图',
                '/student/freshman-plan'),
            _buildMenuCard(context, Icons.school_outlined, '思政学习', 'AI 思政理论学习',
                '/student/political-study'),
            _buildMenuCard(context, Icons.auto_stories_outlined, '思想档案',
                '思想成长记录', '/student/ideological-record'),
            _buildMenuCard(context, Icons.groups_2_outlined, 'AI 学伴', '学习伙伴互动',
                '/student/study-buddy'),
            _buildMenuCard(context, Icons.weekend_outlined, '校园生活', '校园服务指南',
                '/student/campus-life'),
            _buildMenuCard(context, Icons.event_available, '日程管理', '学习日程安排',
                '/student/schedule'),
            _buildMenuCard(context, Icons.upload_file, '导入我的课表', '从门户查到课表后导入本人课表',
                '/student/schedule-import'),
            _buildMenuCard(context, Icons.favorite_border, '心理陪伴', '心情打卡与关怀',
                '/student/mental-health'),
            _buildMenuCard(context, Icons.smart_toy_outlined, '数字导师',
                'AI 语音陪伴导师', '/student/digital-mentor'),
            _buildMenuCard(context, Icons.emoji_events_outlined, '竞赛匹配',
                'AI 竞赛项目匹配', '/student/competition-match'),
          ],

          // ── 学生功能模块（毕设选题/学科竞赛/大学规划/入党教育/社团生活）──
          if (profile?.role == 'student' ||
              profile?.role == 'student_union') ...[
            _buildMenuCard(context, Icons.topic_outlined, '毕设选题', '毕业设计选题与导师选择',
                '/graduation'),
            _buildMenuCard(context, Icons.emoji_events_outlined, '学科竞赛',
                '竞赛报名与作品提交', '/competition'),
            _buildMenuCard(
                context, Icons.calendar_today, '大学规划', '四年学业与职业规划', '/plan'),
            _buildMenuCard(context, Icons.flag_outlined, '入党教育', '入党流程进度与学习',
                '/party-education'),
            _buildMenuCard(
                context, Icons.groups_outlined, '社团生活', '社团加入与活动参与', '/club'),
          ],

          // 辅导员 AI 功能
          if (profile?.role == 'counselor') ...[
            _buildMenuCard(context, Icons.visibility_outlined, 'AI 今日关注',
                '重点关注学生提醒', '/counselor/daily-focus'),
            _buildMenuCard(context, Icons.assessment_outlined, '班级学情日报',
                '班级每日学情分析', '/counselor/class-report'),
            _buildMenuCard(context, Icons.dashboard_outlined, '数字孪生看板',
                '学生数字画像看板', '/counselor/twin-board'),
            _buildMenuCard(context, Icons.warning_outlined, '预测性预警', 'AI 风险预测',
                '/counselor/prediction'),
            _buildMenuCard(context, Icons.auto_fix_high, 'AI 干预方案', '智能干预方案生成',
                '/counselor/intervention'),
            _buildMenuCard(context, Icons.record_voice_over, '谈心谈话', '谈话记录管理',
                '/counselor/talk-record'),
            _buildMenuCard(context, Icons.tips_and_updates_outlined, '话术推荐',
                'AI 谈话话术', '/counselor/talk-tips'),
            _buildMenuCard(context, Icons.psychology, '思想档案', '学生思想动态',
                '/counselor/ideological'),
            _buildMenuCard(context, Icons.groups_outlined, '班级画像', '班级性格画像',
                '/counselor/class-profile'),
            _buildMenuCard(context, Icons.admin_panel_settings, '社区管理',
                '问答社区内容管理', '/counselor/community-manage'),
            _buildMenuCard(context, Icons.trending_up, '热点感知', '校园舆情热点感知',
                '/counselor/hot-topic-sense'),
            _buildMenuCard(context, Icons.edit_note, '办事管理', '办事流程编辑管理',
                '/process-manage'),
            _buildMenuCard(context, Icons.people_alt_outlined, '学生列表',
                '查看管理学生名单', '/counselor/student-list'),
          ],

          // 教师 AI 功能
          if (profile?.role == 'teacher') ...[
            _buildMenuCard(context, Icons.school_outlined, '今日授课', 'AI 授课概览',
                '/teacher/daily-overview'),
            _buildMenuCard(context, Icons.auto_awesome, 'AI 备课', '智能备课助手',
                '/teacher/lesson-prep'),
            _buildMenuCard(context, Icons.quiz_outlined, 'AI 出题', '智能考试出题',
                '/teacher/exam-gen'),
            _buildMenuCard(context, Icons.live_help_outlined, '课堂互动', 'AI 课堂互动',
                '/teacher/class-interact'),
            _buildMenuCard(
                context, Icons.grading, 'AI 批改', '智能作业批改', '/teacher/grading'),
            _buildMenuCard(
                context, Icons.grid_on, '学情热力图', '班级学情可视化', '/teacher/heatmap'),
            _buildMenuCard(context, Icons.self_improvement, '教学反思', 'AI 教学反思',
                '/teacher/reflection'),
            _buildMenuCard(context, Icons.pie_chart_outline, '学习风格', '学生学习风格分布',
                '/teacher/style-dist'),
            _buildMenuCard(context, Icons.question_answer_outlined, '社区问答',
                '教师社区答疑', '/teacher/community-qa'),
          ],

          // 教辅 AI 功能
          if (profile?.role == 'assistant') ...[
            _buildMenuCard(context, Icons.event_busy, '排课检测', '排课冲突检测',
                '/assistant/schedule-check'),
            _buildMenuCard(context, Icons.school, '毕业审核', '毕业资格审核',
                '/assistant/grad-audit'),
            _buildMenuCard(context, Icons.event_note, '考试编排', '考试安排管理',
                '/assistant/exam-arrange'),
            _buildMenuCard(context, Icons.build, '后勤服务台', '实验/保洁/热水/查岗/环卫/借阅',
                '/assistant/facility-workbench'),
          ],

          // 学生会 AI 功能
          if (profile?.role == 'student_union') ...[
            _buildMenuCard(context, Icons.event, 'AI 活动策划', '智能活动方案生成',
                '/union/event-plan'),
            _buildMenuCard(context, Icons.brush, 'AI 海报文案', '智能海报文案生成',
                '/union/poster-gen'),
            _buildMenuCard(context, Icons.groups_outlined, '活动报名管理', '查看报名热度、新建活动',
                '/union/activity-manage'),
          ],

          // 学院管理员 AI 功能
          if (_canAccessAdmin(profile?.role)) ...[
            _buildMenuCard(context, Icons.dashboard, '数字孪生大屏', '学院全景数据',
                '/college/twin-screen'),
            _buildMenuCard(context, Icons.analytics, '数据分析', '学院数据分析报告',
                '/college/data-analysis'),
          ],

          // 情感预警入口（辅导员及以上角色可访问）
          if (_canAccessEmotion(profile?.role))
            _buildMenuCard(context, Icons.warning_amber_rounded, '情感预警',
                '查看和管理学生情感告警', '/emotion',
                iconColor: theme.colorScheme.error),
        ],

        const SizedBox(height: 16),

        // 修改密码
        Card(
          elevation: 0,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
            side: BorderSide(color: theme.colorScheme.outlineVariant),
          ),
          child: ListTile(
            leading: Icon(Icons.lock_outline, color: theme.colorScheme.primary),
            title: const Text('修改密码'),
            subtitle: const Text('修改登录密码'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => _showChangePasswordDialog(context),
          ),
        ),

        const SizedBox(height: 16),

        // 关于
        Card(
          elevation: 0,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
            side: BorderSide(color: theme.colorScheme.outlineVariant),
          ),
          child: ListTile(
            leading: const Icon(Icons.info_outline),
            title: const Text('关于蔚小芯'),
            subtitle: Text('v${ReleaseConfig.version} · 滁州学院计算机学院'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () {
              context.push('/about');
            },
          ),
        ),

        const SizedBox(height: 24),

        // 退出登录
        SizedBox(
          width: double.infinity,
          height: 48,
          child: OutlinedButton.icon(
            onPressed: () async {
              // 退出前清空全部敏感内存态，防止下一账号在同设备看到上一账号数据（Q-08）
              // 与 401 令牌吊销路径共用同一注册表，避免遗漏新增 Provider
              triggerSessionReset();
              await auth.logout();
              if (context.mounted) {
                context.go('/login');
              }
            },
            icon: const Icon(Icons.logout),
            label: const Text('退出登录'),
            style: OutlinedButton.styleFrom(
              foregroundColor: theme.colorScheme.error,
              side: BorderSide(color: theme.colorScheme.error),
            ),
          ),
        ),
      ],
    );
  }

  List<_ProfileFeature> _featuresFor(String? role) => [
        _ProfileFeature('token', '常用', Icons.bar_chart, '词元统计', '查看 AI 词元消耗统计',
            '/token-stats'),
        _ProfileFeature('bookmarks', '常用', Icons.star_outline, '我的收藏',
            '查看已收藏的问答记录', '/bookmarks'),
        _ProfileFeature('feedback_mine', '常用', Icons.rate_review, '我的反馈',
            '查看自己提交的反馈与处理状态', '/my-feedbacks'),
        _ProfileFeature('records', '办事服务', Icons.assignment_turned_in_outlined,
            '我的办事记录', '查看入学/离校等流程办理进度', '/my-records'),
        _ProfileFeature('model', '常用', Icons.tune, 'AI 模型配置', '配置模型参数',
            '/profile/model-config'),
        _ProfileFeature('culture_anthems', '校园文化', Icons.music_note, '校歌曲库',
            '校歌、院歌与经典曲目', '/culture/anthems'),
        _ProfileFeature('culture_radio', '校园文化', Icons.podcasts, '校园广播',
            '直播节目单与往期回放', '/culture/radio'),
        _ProfileFeature('culture_lectures', '校园文化', Icons.school_outlined,
            '学术讲座', '即将开始的讲座与回放', '/culture/lectures'),
        _ProfileFeature('culture_events', '校园文化', Icons.celebration_outlined,
            '校园活动', '活动报名与个性推送', '/culture/events'),
        _ProfileFeature('culture_volunteer', '校园文化', Icons.volunteer_activism,
            '志愿服务', '志愿时长与项目推荐', '/culture/volunteer'),
        if (role == 'student' || role == 'student_union') ...[
          _ProfileFeature('daily', '学生服务', Icons.wb_sunny_outlined, '今日速览',
              'AI 每日学习概览', '/student/daily-briefing'),
          _ProfileFeature('diary', '学生服务', Icons.auto_stories, '学习日记',
              'AI 自动生成学习日记', '/student/learning-diary'),
          _ProfileFeature('checkin', '学生服务', Icons.check_circle_outline, '每日打卡',
              '学习打卡与连续记录', '/student/checkin'),
          _ProfileFeature('digital_twin', '学生服务', Icons.person_pin, '数字孪生',
              '我的数字画像', '/student/digital-twin'),
          _ProfileFeature('student_profile', '学生服务', Icons.account_box, '个人档案',
              '完整学生信息聚合', '/student/profile'),
          _ProfileFeature('personality', '学生服务', Icons.psychology_outlined,
              '性格洞察', 'AI 性格分析', '/student/personality'),
          _ProfileFeature('health', '学生服务', Icons.favorite_outline, '身体健康',
              '身体信息·体检·病历', '/student/health'),
          _ProfileFeature('achievements', '学生服务', Icons.emoji_events_outlined,
              '积分成就', '学习积分与成就', '/student/achievements'),
          _ProfileFeature('course_map', '学生服务', Icons.map_outlined, '课程地图',
              '课程学习路径', '/student/course-map'),
          _ProfileFeature('course_analytics', '学生服务', Icons.analytics_outlined,
              '课程学情', '课程学习分析', '/student/course-analytics'),
          _ProfileFeature('weekly_report', '学生服务', Icons.summarize_outlined,
              '学习周报', 'AI 周度学习总结', '/student/weekly-report'),
          _ProfileFeature('qa_plaza', '学生服务', Icons.forum_outlined, '问答广场',
              '校园问答社区', '/student/qa-plaza'),
          _ProfileFeature('hot_topics', '学生服务', Icons.local_fire_department,
              '热点关注', '校园热点话题', '/student/hot-topics'),
          _ProfileFeature('qa_leaderboard', '学生服务', Icons.leaderboard_outlined,
              '问答排行', '问答贡献排行榜', '/student/qa-leaderboard'),
          _ProfileFeature('private_chat', '学生服务', Icons.chat_outlined, '站内私聊',
              'AI 学伴私信', '/student/private-chat'),
          _ProfileFeature('process_enhanced', '办事服务',
              Icons.account_tree_outlined, 'AI 办事流程', '智能流程引导', '/enrollment'),
          _ProfileFeature('graduation', '学生服务', Icons.topic_outlined, '毕设选题',
              '毕业设计选题与导师选择', '/graduation'),
          _ProfileFeature('competition', '学生服务', Icons.emoji_events_outlined,
              '学科竞赛', '竞赛报名与作品提交', '/competition'),
          _ProfileFeature('plan', '学生服务', Icons.calendar_today, '大学规划',
              '四年学业与职业规划', '/plan'),
          _ProfileFeature('party_education', '学生服务', Icons.flag_outlined,
              '入党教育', '入党流程进度与学习', '/party-education'),
          _ProfileFeature('club', '学生服务', Icons.groups_outlined, '社团生活',
              '社团加入与活动参与', '/club'),
        ],
        if (role == 'counselor') ...[
          _ProfileFeature('c_perf_twin', '辅导员服务', Icons.person_pin, '绩效画像',
              '我的帮扶咨询绩效 · 学生绑定', '/student/digital-twin'),
          _ProfileFeature('c_daily_focus', '辅导员服务', Icons.visibility_outlined,
              'AI 今日关注', '重点关注学生提醒', '/counselor/daily-focus'),
          _ProfileFeature('c_class_report', '辅导员服务', Icons.assessment_outlined,
              '班级学情日报', '班级每日学情分析', '/counselor/class-report'),
          _ProfileFeature('c_twin_board', '辅导员服务', Icons.dashboard_outlined,
              '数字孪生看板', '学生数字画像看板', '/counselor/twin-board'),
          _ProfileFeature('c_prediction', '辅导员服务', Icons.warning_outlined,
              '预测性预警', 'AI 风险预测', '/counselor/prediction'),
          _ProfileFeature('c_intervention', '辅导员服务', Icons.auto_fix_high,
              'AI 干预方案', '智能干预方案生成', '/counselor/intervention'),
          _ProfileFeature('c_talk_record', '辅导员服务', Icons.record_voice_over,
              '谈心谈话', '谈话记录管理', '/counselor/talk-record'),
          _ProfileFeature(
              'c_talk_tips',
              '辅导员服务',
              Icons.tips_and_updates_outlined,
              '话术推荐',
              'AI 谈话话术',
              '/counselor/talk-tips'),
          _ProfileFeature('c_ideological', '辅导员服务', Icons.psychology, '思想档案',
              '学生思想动态', '/counselor/ideological'),
          _ProfileFeature('c_class_profile', '辅导员服务', Icons.groups_outlined,
              '班级画像', '班级性格画像', '/counselor/class-profile'),
          _ProfileFeature('c_community', '辅导员服务', Icons.admin_panel_settings,
              '社区管理', '问答社区内容管理', '/counselor/community-manage'),
          _ProfileFeature('c_hot_sense', '辅导员服务', Icons.trending_up, '热点感知',
              '校园舆情热点感知', '/counselor/hot-topic-sense'),
          _ProfileFeature('c_process_edit', '辅导员服务', Icons.edit_note, '办事管理',
              '办事流程编辑管理', '/process-manage'),
          _ProfileFeature('c_student_list', '辅导员服务', Icons.people_alt_outlined,
              '学生列表', '查看管理学生名单', '/counselor/student-list'),
        ],
        if (role == 'teacher') ...[
          _ProfileFeature('t_perf_twin', '教师服务', Icons.person_pin, '绩效画像',
              '我的工作绩效 · 学生绑定', '/student/digital-twin'),
          _ProfileFeature('t_daily', '教师服务', Icons.school_outlined, '今日授课',
              'AI 授课概览', '/teacher/daily-overview'),
          _ProfileFeature('t_lesson', '教师服务', Icons.auto_awesome, 'AI 备课',
              '智能备课助手', '/teacher/lesson-prep'),
          _ProfileFeature('t_exam', '教师服务', Icons.quiz_outlined, 'AI 出题',
              '智能考试出题', '/teacher/exam-gen'),
          _ProfileFeature('t_interact', '教师服务', Icons.live_help_outlined,
              '课堂互动', 'AI 课堂互动', '/teacher/class-interact'),
          _ProfileFeature('t_grading', '教师服务', Icons.grading, 'AI 批改', '智能作业批改',
              '/teacher/grading'),
          _ProfileFeature('t_heatmap', '教师服务', Icons.grid_on, '学情热力图',
              '班级学情可视化', '/teacher/heatmap'),
          _ProfileFeature('t_reflection', '教师服务', Icons.self_improvement,
              '教学反思', 'AI 教学反思', '/teacher/reflection'),
          _ProfileFeature('t_style', '教师服务', Icons.pie_chart_outline, '学习风格',
              '学生学习风格分布', '/teacher/style-dist'),
          _ProfileFeature('t_qa', '教师服务', Icons.question_answer_outlined,
              '社区问答', '教师社区答疑', '/teacher/community-qa'),
        ],
        if (role == 'assistant') ...[
          _ProfileFeature('a_perf_twin', '教辅服务', Icons.person_pin, '绩效画像',
              '我的教务绩效 · 蔚小芯绑定', '/student/digital-twin'),
          _ProfileFeature('a_schedule', '教辅服务', Icons.event_busy, '排课检测',
              '排课冲突检测', '/assistant/schedule-check'),
          _ProfileFeature('a_grad', '教辅服务', Icons.school, '毕业审核', '毕业资格审核',
              '/assistant/grad-audit'),
          _ProfileFeature('a_exam', '教辅服务', Icons.event_note, '考试编排', '考试安排管理',
              '/assistant/exam-arrange'),
          _ProfileFeature('a_calendar', '教辅服务', Icons.calendar_month,
              '教学日历', '学期关键节点', '/assistant/teaching-calendar'),
          _ProfileFeature('a_student_info', '教辅服务', Icons.person_search,
              '学生信息查询', '真实学生账号查询', '/assistant/student-info'),
          _ProfileFeature('a_notify', '教辅服务', Icons.campaign_outlined,
              '通知批量', 'AI 辅助通知草稿', '/assistant/notification-draft'),
          _ProfileFeature('a_facility', '教辅服务', Icons.build,
              '后勤服务台', '实验/保洁/热水/查岗/环卫/借阅', '/assistant/facility-workbench'),
        ],
        if (role == 'student_union') ...[
          _ProfileFeature('u_event_plan', '学生会服务', Icons.event, 'AI 活动策划',
              '智能活动方案生成', '/union/event-plan'),
          _ProfileFeature('u_poster', '学生会服务', Icons.brush, 'AI 海报文案',
              '智能海报文案生成', '/union/poster-gen'),
          _ProfileFeature('u_act_mgmt', '学生会服务', Icons.groups_outlined, '活动报名管理',
              '查看报名热度、新建活动', '/union/activity-manage'),
          _ProfileFeature('u_workbench', '学生会服务', Icons.workspaces_outlined, '学生会工作台',
              '成员活跃·活动分析·招新·问卷·热点', '/union/workbench'),
        ],
        if (_canAccessAdmin(role)) ...[
          _ProfileFeature('college_twin', '管理服务', Icons.dashboard, '数字孪生大屏',
              '学院全景数据', '/college/twin-screen'),
          _ProfileFeature('college_analysis', '管理服务', Icons.analytics, '数据分析',
              '学院数据分析报告', '/college/data-analysis'),
        ],
        if (CapabilityUtils.has(Capability.outcomeDashboard))
          _ProfileFeature('secretary_outcome', '管理服务', Icons.auto_graph,
              '教育成果大屏', '书记视角：竞赛/入党/学业/毕业去向', '/secretary/education-outcome'),
        // 书记党建育人 / 协同育人专项深链（D1-1 功能补齐，2026-08-16）
        if (CapabilityUtils.has(Capability.outcomeDashboard))
          _ProfileFeature('secretary_party', '管理服务', Icons.flag, '党建育人专项',
              '书记视角：入党/党课/学习育人可视化', '/secretary/party-dashboard'),
        if (CapabilityUtils.has(Capability.collabDashboard))
          _ProfileFeature('secretary_collab', '管理服务', Icons.groups, '协同育人专项',
              '书记视角：教师/教辅育人动作总览', '/secretary/collab-dashboard'),
        if (CapabilityUtils.hasAny([
              Capability.outcomeRecordWrite,
              Capability.outcomeReview,
            ]))
          _ProfileFeature('outcome_manage', '教辅服务', Icons.task_alt,
              '毕业去向登记', '学生自报/教辅录入+审核', '/secretary/outcome-manage'),
        if (CapabilityUtils.has(Capability.unionFeedbackList))
          _ProfileFeature('feedback_manage', '管理服务', Icons.feedback_outlined,
              '反馈管理', '查看和处理用户反馈', '/feedback'),
        if (_canSubmitKB(role))
          _ProfileFeature('kb_submit', '知识治理', Icons.note_add_outlined, '知识提交',
              '创建和管理知识资源', '/my-submissions'),
        if (_canAccessEmotion(role))
          _ProfileFeature('kb_review', '知识治理', Icons.rate_review_outlined,
              '知识审核', '审核待发布的知识资源', '/review'),
        if (CapabilityUtils.has(Capability.counselorKbWrite))
          _ProfileFeature('p_manage', '办事服务', Icons.edit_note, '办事管理',
              '新增、编辑、发布和导出办事流程', '/process-manage'),
        if (CapabilityUtils.has(Capability.counselorKbReview))
          _ProfileFeature('p_review', '办事服务', Icons.rate_review_outlined,
              '办事审核', '审核学校、学院管理员提交的办事流程', '/process-review'),
        if (_canAccessEmotion(role))
          _ProfileFeature('emotion', '管理服务', Icons.warning_amber_rounded,
              '情感预警', '查看和管理学生情感告警', '/emotion'),
        if (_canAccessUserManagement())
          _ProfileFeature('users', '管理服务', Icons.people_outline, '用户管理',
              '导入学生并管理账号、角色和状态', '/admin/users'),
        if (_canAccessAdmin(role))
          _ProfileFeature('metrics', '管理服务', Icons.dashboard_outlined, '质量看板',
              '查看系统问答质量指标', '/admin/metrics'),
        if (_canAccessAdmin(role))
          _ProfileFeature('audit', '管理服务', Icons.history, '审计日志', '查看系统操作记录',
              '/admin/audit'),
        if (_canAccessAdmin(role))
          _ProfileFeature('content_admin', '管理服务', Icons.edit_note, '内容管理',
              '毕设选题/就业指导/学科竞赛管理', '/admin/content'),
        if (role == 'sys_admin')
          _ProfileFeature('settings', '管理服务', Icons.settings_outlined, '系统配置',
              '管理系统运行参数', '/admin/settings'),
        if (role == 'sys_admin')
          _ProfileFeature('ai_briefing_admin', '管理服务', Icons.newspaper,
              'AI 简讯管理', '资讯 CRUD、来源抓取与导出', '/admin/ai-briefings'),
        // 注册开关（2026-09-01 用户反馈④）：仅出现在「功能开关」Tab，不渲染为功能卡片。
        // 持久化到后端 feature.guest_register，控制游客/学生手机注册是否开放。
        if (_canAccessAdmin(role))
          _ProfileFeature(
            'guest_register',
            '管理服务',
            Icons.app_registration,
            '注册开放',
            '是否允许新用户（游客/学生）注册账号',
            '',
            switchOnly: true,
          ),
      ];

  /// 多角色切换卡片（2026-09-01）：用户账号具备 >1 个角色时显示，
  /// 点击切换以目标角色为主角色重新签发 JWT，前端据此重绘菜单/页面。
  Widget _buildRoleSwitcher(BuildContext context, AuthProvider auth) {
    final theme = Theme.of(context);
    final roles = (auth.profile?.roles.isNotEmpty ?? false)
        ? auth.profile!.roles
        : (Storage.roles.isNotEmpty ? Storage.roles : null);
    final current = auth.profile?.role ?? Storage.role ?? '';
    if (roles == null || roles.length < 2) {
      return const SizedBox.shrink();
    }

    String label(String r) {
      const map = {
        'sys_admin': '系统管理员',
        'school_admin': '学校管理员',
        'college_admin': '学院管理员',
        'counselor': '辅导员',
        'student_union': '学生会',
        'student': '学生',
        'teacher': '教师',
        'assistant': '教辅',
      };
      return map[r] ?? r;
    }

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.swap_horiz,
                    color: theme.colorScheme.primary, size: 20),
                const SizedBox(width: 8),
                const Text('切换身份',
                    style: TextStyle(fontWeight: FontWeight.w600)),
              ],
            ),
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                for (final r in roles)
                  ChoiceChip(
                    label: Text(label(r)),
                    selected: r == current,
                    onSelected: r == current
                        ? null
                        : (_) async {
                            final (ok, msg) = await auth.switchRole(r);
                            if (!context.mounted) return;
                            if (!ok) {
                              ScaffoldMessenger.of(context).showSnackBar(
                                SnackBar(content: Text(msg)),
                              );
                            }
                          },
                  ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildFeatureTabs(BuildContext context, String? role) {
    final all = _featuresFor(role);
    // 修复"功能开关无法关闭"：_listedFeatures/_enabledFeatures 是历史遗留的
    // 本机过滤缓存，不再强制 addAll 回填（否则用户关闭的项会被立即恢复）。
    // 功能可见性统一由后端 feature.<key> 全局开关决定（见 _buildFeatureSwitches）。
    final visible = all.where((f) {
      // 管理员全局功能开关：feature.<key>=false 时对普通用户隐藏该模块
      final g = Storage.globalFeatureSwitches['feature.${f.key}'];
      if (g == null) return true; // 未配置默认开放
      return g == 'true';
    }).toList();
    final categories = [
      '常用',
      '学生服务',
      '办事服务',
      '辅导员服务',
      '教师服务',
      '教辅服务',
      '学生会服务',
      '知识治理',
      '校园文化',
      '管理服务'
    ];
    final adminCanManage = role == 'sys_admin' || role == 'college_admin';
    final tabs = [
      ...categories.where((c) => visible.any((f) => f.category == c)),
      if (adminCanManage) '功能开关'
    ];
    if (tabs.isEmpty) return const SizedBox.shrink();
    // Tab 分类图标映射
    const categoryIcons = <String, IconData>{
      '常用': Icons.star_outline,
      '学生服务': Icons.school_outlined,
      '办事服务': Icons.assignment_outlined,
      '辅导员服务': Icons.people_outline,
      '教师服务': Icons.cast_for_education_outlined,
      '教辅服务': Icons.support_agent_outlined,
      '学生会服务': Icons.groups_outlined,
      '知识治理': Icons.library_books_outlined,
      '校园文化': Icons.palette_outlined,
      '管理服务': Icons.admin_panel_settings_outlined,
      '功能开关': Icons.toggle_on_outlined,
    };
    return DefaultTabController(
      length: tabs.length,
      child: Card(
        elevation: 0,
        shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(16),
            side: BorderSide(
                color: Theme.of(context).colorScheme.outlineVariant)),
        child: Column(
          children: [
            TabBar(isScrollable: true, tabs: [
              for (final t in tabs)
                Tab(
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(categoryIcons[t] ?? Icons.widgets_outlined,
                          size: 20),
                      const SizedBox(width: 6),
                      Text(t),
                    ],
                  ),
                )
            ]),
            SizedBox(
              height: 430,
              child: TabBarView(
                children: [
                  for (final t in tabs)
                    t == '功能开关'
                        ? _buildFeatureSwitches(all, role)
                        : _buildFeatureGrid(
                            visible.where((f) => f.category == t).toList()),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildFeatureGrid(List<_ProfileFeature> features) {
    return ListView(
      padding: const EdgeInsets.all(12),
      children: [
        for (final f in features)
          if (!f.switchOnly) _buildMenuCard(context, f.icon, f.title, f.subtitle, f.route)
      ],
    );
  }

  Widget _buildFeatureSwitches(List<_ProfileFeature> features, String? role) {
    final sys = role == 'sys_admin';
    // 管理员开关直接读写后端 feature.<key> 全局配置（持久化，影响所有用户可见性），
    // 不再操作仅本机的 _listedFeatures/_enabledFeatures（旧逻辑导致"无法关闭"）。
    final adminProvider = context.watch<AdminProvider>();
    if (!adminProvider.settingsLoading && adminProvider.settings.isEmpty) {
      // 懒加载系统配置（含 feature.* 键）
      WidgetsBinding.instance.addPostFrameCallback((_) {
        context.read<AdminProvider>().fetchSettings();
      });
    }
    final settings = adminProvider.settings;
    bool switchValue(String key) {
      // 查找 feature.<key> 配置，未配置（null）默认开启
      for (final s in settings) {
        if (s.key == 'feature.$key') {
          return s.value == 'true' || s.value == '1';
        }
      }
      return true;
    }

    return ListView(
      padding: const EdgeInsets.all(12),
      children: [
        Text(sys ? '系统管理员：控制应用模块是否上架' : '学院管理员：控制本学院是否启用',
            style: Theme.of(context).textTheme.titleSmall),
        const SizedBox(height: 8),
        for (final f in features)
          SwitchListTile(
            secondary: Icon(f.icon),
            title: Text(f.title),
            subtitle: Text('${f.category} · ${f.subtitle}'),
            value: switchValue(f.key),
            onChanged: (v) async {
              // 持久化到后端（feature.<key>=true/false），对普通用户即时生效
              await context.read<AdminProvider>().updateSettings({
                'feature.${f.key}': v ? 'true' : 'false',
              });
              if (!mounted) return;
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(
                  content: Text('${f.title}已${v ? '开启' : '关闭'}，对普通用户立即生效'),
                  duration: const Duration(seconds: 2),
                ),
              );
            },
          ),
      ],
    );
  }

  /// 语音功能开关卡片
  Widget _buildVoiceToggle(BuildContext context, AuthProvider auth) {
    final theme = Theme.of(context);
    // 首次加载时触发获取语音配置
    if (auth.voiceEnabled == null) {
      auth.getVoiceConfig().then((_) {
        if (mounted) setState(() {});
      });
    }
    final enabled = (auth.voiceEnabled ?? 0) == 1;
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: SwitchListTile(
        secondary: Icon(
          Icons.mic,
          color: enabled
              ? theme.colorScheme.primary
              : theme.colorScheme.onSurfaceVariant,
        ),
        title: const Text('语音功能'),
        subtitle: Text(enabled ? '语音唤醒与语音输入已开启' : '开启后可唤醒语音助手'),
        value: enabled,
        onChanged: (v) async {
          final ok = await auth.updateVoiceConfig(v ? 1 : 0);
          if (context.mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text(ok ? '语音设置已更新' : '设置失败')),
            );
          }
        },
      ),
    );
  }

  /// 主题模式切换卡片
  Widget _buildThemeSection(BuildContext context) {
    final theme = Theme.of(context);
    final themeNotifier = context.watch<ThemeNotifier>();
    final currentMode = themeNotifier.mode;

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        child: Row(
          children: [
            Icon(Icons.brightness_6, color: theme.colorScheme.primary),
            const SizedBox(width: 16),
            Expanded(
              child: Text('主题模式', style: theme.textTheme.bodyLarge),
            ),
            SegmentedButton<ThemeMode>(
              segments: const [
                ButtonSegment(
                  value: ThemeMode.light,
                  icon: Icon(Icons.light_mode, size: 18),
                ),
                ButtonSegment(
                  value: ThemeMode.system,
                  icon: Icon(Icons.brightness_auto, size: 18),
                ),
                ButtonSegment(
                  value: ThemeMode.dark,
                  icon: Icon(Icons.dark_mode, size: 18),
                ),
              ],
              selected: {currentMode},
              onSelectionChanged: (modes) {
                themeNotifier.setMode(modes.first);
              },
              style: ButtonStyle(
                visualDensity: VisualDensity.compact,
                textStyle:
                    WidgetStateProperty.all(const TextStyle(fontSize: 12)),
                tapTargetSize: MaterialTapTargetSize.shrinkWrap,
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// 年级主题横幅：展示当前年级主题色与名称（登录后随入学年份切换）
  /// 用户信息头卡：年级主题渐变 + 头像 + 身份信息
  Widget _buildUserHeaderCard(BuildContext context, UserProfile? profile) {
    final theme = Theme.of(context);
    final themeNotifier = context.watch<ThemeNotifier>();
    final accent = themeNotifier.gradeAccent;
    final name = profile?.displayName ?? Storage.displayName ?? '未登录';
    final initial = name.characters.first;

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: [
            Color.alphaBlend(accent.withOpacity(0.16), theme.colorScheme.surface),
            theme.colorScheme.surfaceContainerLow,
          ],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: accent.withOpacity(0.2)),
      ),
      child: Column(
        children: [
          Container(
            width: 76,
            height: 76,
            decoration: BoxDecoration(
              gradient: LinearGradient(
                colors: [accent, Color.alphaBlend(accent, Colors.white.withOpacity(0.3))],
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
              ),
              shape: BoxShape.circle,
              boxShadow: [
                BoxShadow(
                  color: accent.withOpacity(0.28),
                  blurRadius: 18,
                  offset: const Offset(0, 6),
                ),
              ],
            ),
            child: Center(
              child: Text(
                initial,
                style: const TextStyle(
                  fontSize: 32,
                  fontWeight: FontWeight.bold,
                  color: Colors.white,
                ),
              ),
            ),
          ),
          const SizedBox(height: 14),
          Text(name,
              style: theme.textTheme.titleLarge
                  ?.copyWith(fontWeight: FontWeight.bold)),
          const SizedBox(height: 4),
          Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(Icons.workspace_premium_outlined, size: 15, color: accent),
              const SizedBox(width: 5),
              Text(
                profile?.roleLabel ?? '',
                style: TextStyle(
                    color: accent, fontWeight: FontWeight.w600, fontSize: 13),
              ),
            ],
          ),
          if (profile?.college.isNotEmpty ?? false) ...[
            const SizedBox(height: 2),
            Text(
              profile!.college,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildGradeThemeBanner(BuildContext context) {
    final theme = Theme.of(context);
    final themeNotifier = context.watch<ThemeNotifier>();
    final enabled = themeNotifier.gradeThemeEnabled;
    final accent = themeNotifier.gradeAccent;
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(14),
        side: BorderSide(color: accent.withOpacity(0.4)),
      ),
      color: accent.withOpacity(0.10),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        child: Row(
          children: [
            Container(
              width: 12,
              height: 40,
              decoration: BoxDecoration(
                color: accent,
                borderRadius: BorderRadius.circular(6),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    enabled ? '${themeNotifier.gradeThemeName}主题' : '滁院蓝主题',
                    style: theme.textTheme.titleSmall?.copyWith(
                      fontWeight: FontWeight.w700,
                      color: accent,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    enabled ? '系统按你的入学年份自动匹配年级专属配色' : '年级主题已关闭，使用统一配色',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// 年级主题自动切换开关
  Widget _buildGradeThemeToggle(BuildContext context) {
    final theme = Theme.of(context);
    final themeNotifier = context.watch<ThemeNotifier>();
    final enabled = themeNotifier.gradeThemeEnabled;
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: SwitchListTile(
        secondary: Icon(
          Icons.palette_outlined,
          color: enabled
              ? theme.colorScheme.primary
              : theme.colorScheme.onSurfaceVariant,
        ),
        title: const Text('年级主题自动切换'),
        subtitle: Text(
          enabled
              ? '当前主题：${themeNotifier.gradeThemeName}（按入学年份自动切换）'
              : '已关闭，使用统一滁院蓝主题',
        ),
        value: enabled,
        onChanged: (v) async {
          themeNotifier.setGradeThemeEnabled(v);
        },
      ),
    );
  }

  /// 我的关注入口卡片
  Widget _studentInterestsCard(ThemeData theme) {
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: ListTile(
        leading: const Icon(Icons.interests_outlined, color: Color(0xFF1565C0)),
        title: const Text('我的关注'),
        subtitle: Text(
          Storage.studentInterests.isEmpty
              ? '设置你关心的内容，首页按关注优先展示'
              : '已选择：${Storage.studentInterests.join('、')}',
        ),
        trailing: const Icon(Icons.chevron_right),
        onTap: () async {
          final result = await pickupStudentInterests(context);
          if (result != null) {
            await Storage.setStudentInterests(result);
            if (context.mounted) setState(() {});
          }
        },
      ),
    );
  }

  /// 数字人形象显示开关
  Widget _buildAvatarToggle(BuildContext context) {
    final theme = Theme.of(context);
    final show = Storage.showAvatar;
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: SwitchListTile(
        secondary: Icon(
          Icons.person_pin_circle_outlined,
          color: show
              ? theme.colorScheme.primary
              : theme.colorScheme.onSurfaceVariant,
        ),
        title: const Text('数字人形象'),
        subtitle: Text(show ? '首页与数字孪生展示个性化卡通形象' : '已隐藏数字人形象'),
        value: show,
        onChanged: (v) async {
          await Storage.setShowAvatar(v);
          if (context.mounted) {
            setState(() {});
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text(v ? '数字人形象已开启' : '数字人形象已隐藏')),
            );
          }
        },
      ),
    );
  }

  /// 显示修改密码对话框
  void _showChangePasswordDialog(BuildContext context) {
    final oldPwdCtrl = TextEditingController();
    final newPwdCtrl = TextEditingController();
    final confirmCtrl = TextEditingController();
    final formKey = GlobalKey<FormState>();

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('修改密码'),
        content: Form(
          key: formKey,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextFormField(
                controller: oldPwdCtrl,
                obscureText: true,
                decoration: const InputDecoration(
                  labelText: '旧密码',
                  hintText: '请输入当前登录密码',
                  border: OutlineInputBorder(),
                  isDense: true,
                ),
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: newPwdCtrl,
                obscureText: true,
                decoration: const InputDecoration(
                  labelText: '新密码',
                  border: OutlineInputBorder(),
                  isDense: true,
                ),
                validator: (v) {
                  if (v == null || v.length < 6) return '新密码至少 6 位';
                  return null;
                },
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: confirmCtrl,
                obscureText: true,
                decoration: const InputDecoration(
                  labelText: '确认新密码',
                  border: OutlineInputBorder(),
                  isDense: true,
                ),
                validator: (v) {
                  if (v != newPwdCtrl.text) return '两次密码不一致';
                  return null;
                },
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () async {
              if (!formKey.currentState!.validate()) return;
              final auth = context.read<AuthProvider>();
              final ok = await auth.changePassword(
                oldPwdCtrl.text,
                newPwdCtrl.text,
              );
              if (ctx.mounted) {
                Navigator.pop(ctx);
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(
                    content: Text(ok ? '密码修改成功' : (auth.error ?? '修改失败')),
                    backgroundColor:
                        ok ? null : Theme.of(context).colorScheme.error,
                  ),
                );
              }
            },
            child: const Text('确认修改'),
          ),
        ],
      ),
    );
  }

  Widget _buildInfoTile(
      BuildContext context, IconData icon, String label, String value) {
    final theme = Theme.of(context);
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: ListTile(
        leading: Icon(icon, color: theme.colorScheme.primary),
        title: Text(label),
        trailing: Text(
          value,
          style: TextStyle(color: theme.colorScheme.onSurfaceVariant),
        ),
      ),
    );
  }

  Widget _buildMenuCard(BuildContext context, IconData icon, String title,
      String subtitle, String route,
      {Color? iconColor}) {
    final theme = Theme.of(context);
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: ListTile(
        leading: Icon(icon, color: iconColor ?? theme.colorScheme.primary),
        title: Text(title),
        subtitle: Text(subtitle),
        trailing: const Icon(Icons.chevron_right),
        onTap: () => context.go(route),
      ),
    );
  }

  /// 判断角色是否可访问情感预警
  bool _canAccessEmotion(String? role) => RoleUtils.canAccessEmotion(role);

  /// 判断角色是否可访问智能体管理
  bool _canAccessAgents(String? role) => RoleUtils.canAccessAgents(role);

  /// 判断角色是否可访问管理端（college_admin 及以上）
  bool _canAccessAdmin(String? role) => RoleUtils.canAccessAdmin(role);

  /// 判断角色是否可提交知识（student_union 及以上）
  bool _canSubmitKB(String? role) => RoleUtils.canSubmitKB(role);

  bool _canAccessUserManagement() =>
      CapabilityUtils.has(Capability.counselorImportStudent) ||
      CapabilityUtils.has(Capability.collegeUserRead);
}
