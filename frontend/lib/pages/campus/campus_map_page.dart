import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:url_launcher/url_launcher.dart';
import '../../config/api_config.dart';
import '../../services/api_service.dart';
import '../../utils/storage.dart';
import '../../widgets/baidu_campus_map_embed.dart';
import 'campus_step_admin_panel.dart';

class CampusMapPage extends StatefulWidget {
  const CampusMapPage({super.key});

  @override
  State<CampusMapPage> createState() => _CampusMapPageState();
}

enum _CampusTab { map, vr, home, yxwz, douyin }

/// 底图图层：标准矢量图 / 卫星影像图。
enum _MapLayer { standard, satellite }

enum _MapMode { twoD, threeD }

class _CampusTabInfo {
  final _CampusTab tab;
  final String label;
  final IconData icon;
  final Color color;
  final String url;
  final String subtitle;

  const _CampusTabInfo(
      this.tab, this.label, this.icon, this.color, this.url, this.subtitle);
}

class _CheckinStep {
  final String title;
  final String location;
  final double lat;
  final double lng;
  final String duration;
  final String task;
  final String materials;
  final String contact;
  final String note;
  final IconData icon;

  const _CheckinStep({
    required this.title,
    required this.location,
    required this.lat,
    required this.lng,
    required this.duration,
    required this.task,
    required this.materials,
    required this.contact,
    required this.note,
    required this.icon,
  });
}

class _CampusPlan {
  final String id;
  final String name;
  final String address;
  final double lat;
  final double lng;
  final String entrance;
  final List<_CheckinStep> steps;

  const _CampusPlan({
    required this.id,
    required this.name,
    required this.address,
    required this.lat,
    required this.lng,
    required this.entrance,
    required this.steps,
  });
}

const _tabs = [
  _CampusTabInfo(_CampusTab.map, '报到导航', Icons.route_outlined,
      Color(0xFF1677FF), '', '从当前位置进入校园，按报到顺序逐步办理'),
  _CampusTabInfo(_CampusTab.vr, 'VR全景', Icons.view_in_ar, Color(0xFF7B1FA2),
      'https://www.chzu.edu.cn/vr/index.html', '足不出户漫游校园'),
  _CampusTabInfo(_CampusTab.home, '官网', Icons.school, Color(0xFF1565C0),
      'https://www.chzu.edu.cn', '滁州学院官方网站'),
  _CampusTabInfo(_CampusTab.yxwz, '迎新网站', Icons.how_to_reg_outlined,
      Color(0xFF00897B), 'https://xgpt.chzu.edu.cn/yxwz', '滁州学院迎新服务入口'),
  _CampusTabInfo(
      _CampusTab.douyin,
      '抖音',
      Icons.music_note,
      Color(0xFFC62828),
      'https://www.douyin.com/search/%E6%BB%81%E5%B7%9E%E5%AD%A6%E9%99%A2',
      '搜索滁州学院官方抖音'),
];

/// 会峰校区报到步骤（WGS-84，来源：OpenStreetMap 会峰校区 way 734826227）
/// 校区四至 N32.2803/S32.2688/E118.3120/W118.2986，中心 32.2743/118.3055
/// 此前坐标错误地指向琅琊校区位置，已按 OSM 权威数据纠正。
/// final（非 const）：后端加载成功后会用服务端数据覆盖此本地默认值。
final _huifengSteps = [
  _CheckinStep(
    title: '校门入校核验',
    location: '会峰校区南门',
    lat: 32.2705,
    lng: 118.3055,
    duration: '约 5 分钟',
    task: '核验录取通知书、身份证，按学院引导进入校园。',
    materials: '录取通知书、身份证',
    contact: '迎新志愿者 / 保卫处 0550-3510110',
    note: '建议提前准备证件，车辆服从现场引导。',
    icon: Icons.login,
  ),
  _CheckinStep(
    title: '学院报到',
    location: '计算机学院报到点',
    lat: 32.2745,
    lng: 118.3070,
    duration: '约 15 分钟',
    task: '领取班级信息、辅导员联系方式、报到流程单。',
    materials: '录取通知书、身份证、档案袋',
    contact: '学院辅导员，见班级群通知',
    note: '这是后续宿舍、体检等流程的起点。',
    icon: Icons.account_balance,
  ),
  _CheckinStep(
    title: '缴费与绿色通道',
    location: '财务缴费点 / 绿色通道',
    lat: 32.2735,
    lng: 118.3060,
    duration: '约 10-20 分钟',
    task: '完成学杂费确认，助学贷款或缓缴学生办理绿色通道。',
    materials: '缴费凭证、贷款受理证明（如有）',
    contact: '财务处 0550-3510033',
    note: '已线上缴费的学生现场只需核验状态。',
    icon: Icons.payments_outlined,
  ),
  _CheckinStep(
    title: '宿舍入住',
    location: '学生公寓楼值班室',
    lat: 32.2770,
    lng: 118.3040,
    duration: '约 15 分钟',
    task: '确认宿舍信息，领取钥匙，办理入住。',
    materials: '校园卡或身份证',
    contact: '公寓值班室 0550-3510088',
    note: '入住后请检查床位、门锁、水电设施。',
    icon: Icons.bed_outlined,
  ),
  _CheckinStep(
    title: '校园卡与网络',
    location: '一卡通/信息服务点',
    lat: 32.2740,
    lng: 118.3090,
    duration: '约 10 分钟',
    task: '领取或激活校园卡，开通校园网账号。',
    materials: '身份证、学号信息',
    contact: '信息中心 0550-3510999',
    note: '校园卡用于门禁、食堂、图书馆等场景。',
    icon: Icons.credit_card,
  ),
  _CheckinStep(
    title: '入学体检与学籍核验',
    location: '校医院 / 教务处学籍点',
    lat: 32.2720,
    lng: 118.3030,
    duration: '约 30-45 分钟',
    task: '按学院批次完成体检、照片采集和学籍信息核验。',
    materials: '身份证、体检表、录取通知书',
    contact: '校医院 0550-3510120 / 教务处 0550-3510015',
    note: '抽血项目一般需空腹，请按学院通知批次办理。',
    icon: Icons.health_and_safety_outlined,
  ),
];

/// 琅琊校区报到步骤（WGS-84，来源：OpenStreetMap 琅琊校区 way 734826234）
/// 校区四至 N32.2962/S32.2913/E118.3005/W118.2931，中心 32.2943/118.2978
/// 此前坐标偏移到校区以北 2km 处（32.314），已按 OSM 权威数据纠正。
const _langyaSteps = [
  _CheckinStep(
    title: '校门入校核验',
    location: '琅琊校区主入口',
    lat: 32.2921,
    lng: 118.2988,
    duration: '约 5 分钟',
    task: '核验录取通知书、身份证，确认学院迎新引导点。',
    materials: '录取通知书、身份证',
    contact: '迎新志愿者 / 保卫处 0550-3510110',
    note: '老校区道路较集中，请按现场志愿者指引步行前往报到点。',
    icon: Icons.login,
  ),
  _CheckinStep(
    title: '学院报到',
    location: '琅琊校区学院集中报到点',
    lat: 32.2932,
    lng: 118.3002,
    duration: '约 15 分钟',
    task: '领取班级信息、辅导员联系方式、报到流程单。',
    materials: '录取通知书、身份证、档案袋',
    contact: '学院辅导员，见班级群通知',
    note: '如专业报到点调整，以现场公告和蔚小芯通知为准。',
    icon: Icons.account_balance,
  ),
  _CheckinStep(
    title: '缴费与绿色通道',
    location: '琅琊校区综合服务点',
    lat: 32.2928,
    lng: 118.2995,
    duration: '约 10-20 分钟',
    task: '完成学杂费确认，助学贷款或缓缴学生办理绿色通道。',
    materials: '缴费凭证、贷款受理证明（如有）',
    contact: '财务处 0550-3510033',
    note: '线上已缴费学生可快速核验，未缴费学生按现场窗口办理。',
    icon: Icons.payments_outlined,
  ),
  _CheckinStep(
    title: '宿舍入住',
    location: '琅琊校区学生公寓值班室',
    lat: 32.2940,
    lng: 118.2976,
    duration: '约 15 分钟',
    task: '确认宿舍信息，领取钥匙，办理入住。',
    materials: '校园卡或身份证',
    contact: '公寓值班室 0550-3510088',
    note: '入住后请检查床位、门锁、水电设施，问题现场登记。',
    icon: Icons.bed_outlined,
  ),
  _CheckinStep(
    title: '校园卡与网络',
    location: '琅琊校区信息服务点',
    lat: 32.2926,
    lng: 118.3000,
    duration: '约 10 分钟',
    task: '领取或激活校园卡，开通校园网账号。',
    materials: '身份证、学号信息',
    contact: '信息中心 0550-3510999',
    note: '校园卡用于门禁、食堂、图书馆等场景。',
    icon: Icons.credit_card,
  ),
  _CheckinStep(
    title: '入学体检与学籍核验',
    location: '琅琊校区医务/学籍核验点',
    lat: 32.2917,
    lng: 118.2992,
    duration: '约 30-45 分钟',
    task: '按学院批次完成体检、照片采集和学籍信息核验。',
    materials: '身份证、体检表、录取通知书',
    contact: '校医院 0550-3510120 / 教务处 0550-3510015',
    note: '抽血项目一般需空腹，请按学院通知批次办理。',
    icon: Icons.health_and_safety_outlined,
  ),
];

final _campuses = [
  _CampusPlan(
    id: 'huifeng',
    name: '会峰校区',
    address: '安徽省滁州市丰乐大道1528号 滁州学院会峰校区',
    lat: 32.2743,
    lng: 118.3055,
    entrance: '会峰校区南门',
    steps: _huifengSteps,
  ),
  _CampusPlan(
    id: 'langya',
    name: '琅琊校区',
    address: '安徽省滁州市琅琊西路2号 滁州学院琅琊校区',
    lat: 32.2943,
    lng: 118.2978,
    entrance: '琅琊校区主入口',
    steps: _langyaSteps,
  ),
];

class _CampusMapPageState extends State<CampusMapPage> {
  _CampusTab _currentTab = _CampusTab.map;
  _MapLayer _layer = _MapLayer.standard;
  _MapMode _mode = _MapMode.twoD;
  int _campusIndex = 0;
  int _currentStep = 0;
  final Set<int> _completed = {};
  String _copiedText = '';
  /// 地图控制器，用于从外部（如管理员编辑后）同步刷新标注。
  final _mapController = BaiduCampusMapController();

  // ── 后端步骤加载 + 管理员编辑模式 ──
  final ApiService _api = ApiService();
  /// 各校区步骤的可变副本（后端加载成功后覆盖本地默认值）。
  final Map<String, List<_CheckinStep>> _campusStepsMap = {
    'huifeng': _huifengSteps,
    'langya': _langyaSteps,
  };
  /// 步骤后端 ID 映射（index → remote id），拖拽保存时用。
  /// 未从后端加载时为空，_remoteStepId 返回 null 走纯本地模式。
  final Map<int, int> _remoteIds = {};
  bool _loadingSteps = false;
  /// 管理员编辑模式：开启后地图标注可拖拽，松手自动保存坐标。
  bool _editMode = false;

  _CampusPlan get _campus => _campuses[_campusIndex];
  List<_CheckinStep> get _steps =>
      _campusStepsMap[_campus.id] ?? _campus.steps;

  /// 判断当前用户是否有权编辑节点坐标（sys/school/college_admin）。
  bool get _canEditNodes {
    final role = Storage.role ?? 'student';
    return role == 'sys_admin' ||
        role == 'school_admin' ||
        role == 'college_admin';
  }

  @override
  void initState() {
    super.initState();
    // 异步加载后端步骤，不阻塞首帧渲染（失败静默回退本地常量）
    WidgetsBinding.instance.addPostFrameCallback((_) => _loadStepsFromServer());
  }

  /// 从后端加载报到步骤（公开接口，无需登录）。
  /// 失败时回退到本地硬编码常量，保证离线/后端不可用时不影响使用。
  Future<void> _loadStepsFromServer() async {
    setState(() => _loadingSteps = true);
    try {
      final resp =
          await _api.get('${ApiConfig.campusSteps}?campus=${_campus.id}');
      if (resp.data['code'] == 0) {
        final list = resp.data['data'] as List? ?? [];
        if (list.isNotEmpty) {
          final loaded = <_CheckinStep>[];
          _remoteIds.clear();
          for (var i = 0; i < list.length; i++) {
            final e = list[i] as Map<String, dynamic>;
            loaded.add(_CheckinStep(
              title: e['title'] ?? '',
              location: e['location'] ?? '',
              lat: (e['lat'] as num?)?.toDouble() ?? 0.0,
              lng: (e['lng'] as num?)?.toDouble() ?? 0.0,
              duration: e['duration'] ?? '',
              task: e['task'] ?? '',
              materials: e['materials'] ?? '',
              contact: e['contact'] ?? '',
              note: e['note'] ?? '',
              icon: _iconFromName(e['icon_name'] ?? 'place'),
            ));
            final id = e['id'];
            if (id is num) _remoteIds[i] = id.toInt();
          }
          _campusStepsMap[_campus.id] = loaded;
          setState(() => _loadingSteps = false);
          // 通知地图刷新标注
          _mapController.refresh(_stepsForMap, _currentStep);
          return;
        }
      }
      setState(() => _loadingSteps = false);
    } catch (_) {
      // 静默回退：使用本地硬编码常量
      setState(() => _loadingSteps = false);
    }
  }

  /// icon_name 字符串 → Material IconData（与后端 icon_name 字段对应）。
  IconData _iconFromName(String name) {
    switch (name) {
      case 'login':
        return Icons.login;
      case 'account_balance':
        return Icons.account_balance;
      case 'payments':
        return Icons.payments_outlined;
      case 'bed':
        return Icons.bed_outlined;
      case 'credit_card':
        return Icons.credit_card;
      case 'health_and_safety':
        return Icons.health_and_safety_outlined;
      default:
        return Icons.place;
    }
  }

  /// 管理员拖拽标注后，调用后端接口保存新坐标。
  /// 先本地立即更新避免视觉回弹，再异步保存到后端；保存失败时 Snack 提示。
  Future<void> _onMarkerMoved(int index, double lat, double lng) async {
    if (!_canEditNodes) return;
    final step = _steps[index];
    final updated = _CheckinStep(
      title: step.title,
      location: step.location,
      lat: lat,
      lng: lng,
      duration: step.duration,
      task: step.task,
      materials: step.materials,
      contact: step.contact,
      note: step.note,
      icon: step.icon,
    );
    final list = List<_CheckinStep>.from(_steps);
    list[index] = updated;
    _campusStepsMap[_campus.id] = list;
    setState(() {});

    final stepId = _remoteIds[index];
    if (stepId == null) {
      // 后端无对应记录（本地回退模式），仅本地保存
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('后端未加载，坐标仅本地保存'),
            duration: Duration(seconds: 2),
          ),
        );
      }
      return;
    }
    try {
      final resp = await _api.patch(
        ApiConfig.adminCampusStepCoords(stepId.toString()),
        data: {'lat': lat, 'lng': lng},
      );
      if (resp.data['code'] == 0) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
              content: Text('节点坐标已保存'),
              duration: Duration(seconds: 2),
            ),
          );
        }
      } else {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text('保存失败：${resp.data['message'] ?? '未知错误'}'),
              duration: const Duration(seconds: 3),
            ),
          );
        }
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('保存失败：$e'),
            duration: const Duration(seconds: 3),
          ),
        );
      }
    }
  }

  /// 打开报到流程管理面板（管理员专用）。
  /// 面板关闭后重新加载后端步骤，确保地图标注与 CRUD 结果一致。
  ///
  /// 注意：本方法可能从 PopupMenuButton.onSelected 回调中调用，
  /// 此时 PopupMenu 的 overlay 仍在关闭过程中，若立即调用
  /// showModalBottomSheet，其 barrier 会被 PopupMenu 的遮盖拦截，
  /// 表现为"点击菜单项后无弹窗"。用 addPostFrameCallback 延迟一帧，
  /// 等 PopupMenu 完全关闭后再弹出 BottomSheet。
  Future<void> _openAdminPanel() async {
    await WidgetsBinding.instance.endOfFrame;
    if (!mounted) return;
    await CampusStepAdminPanel.show(
      context,
      campusId: _campus.id,
      campusName: _campus.name,
    );
    // 面板关闭后刷新地图标注（CRUD 可能改变了步骤列表/坐标）
    if (mounted) _loadStepsFromServer();
  }

  /// 将步骤列表转为地图 HTML 期望的 Map 格式（WGS-84 坐标）。
  List<Map<String, dynamic>> get _stepsForMap => _steps
      .asMap()
      .entries
      .map((e) => {
            'id': e.key,
            'title': e.value.title,
            'location': e.value.location,
            'lat': e.value.lat,
            'lng': e.value.lng,
          })
      .toList();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('校园服务'),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(48),
          child: Container(
            color: theme.colorScheme.surface,
            child: Row(
              children: _tabs.map((t) {
                final selected = t.tab == _currentTab;
                return Expanded(
                  child: GestureDetector(
                    onTap: () => setState(() => _currentTab = t.tab),
                    child: Container(
                      padding: const EdgeInsets.symmetric(vertical: 10),
                      decoration: BoxDecoration(
                        border: Border(
                          bottom: BorderSide(
                            color: selected ? t.color : Colors.transparent,
                            width: 2.5,
                          ),
                        ),
                      ),
                      child: Row(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          Icon(t.icon,
                              size: 16,
                              color: selected
                                  ? t.color
                                  : theme.colorScheme.onSurfaceVariant),
                          const SizedBox(width: 4),
                          Text(
                            t.label,
                            style: TextStyle(
                              fontSize: 13,
                              fontWeight: selected
                                  ? FontWeight.w600
                                  : FontWeight.normal,
                              color: selected
                                  ? t.color
                                  : theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                );
              }).toList(),
            ),
          ),
        ),
      ),
      body: _currentTab == _CampusTab.map
          ? _buildCheckinNavigator(theme)
          : _buildServiceTab(theme),
    );
  }

  Widget _buildCheckinNavigator(ThemeData theme) {
    return LayoutBuilder(
      builder: (context, constraints) {
        // 桌面阈值 880：MainShell 桌面布局用 ConstrainedBox(maxWidth:900) 限制
        // 内容区宽度，原阈值 980 会让 campus 在桌面端误走移动端上下布局，
        // 地图被 Column 固定项挤塌。880 < 900，保证桌面端走左右分栏、地图
        // stretch 拿全高。
        final desktop = constraints.maxWidth >= 880;
        final map = _buildMapPanel(theme, desktop);
        if (desktop) {
          // 桌面端：左侧地图区（内含顶部控件栏+地图+底部当前步骤面板），
          // 右侧独立完整步骤列表。
          return Row(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Expanded(flex: 7, child: map),
              VerticalDivider(
                  width: 1, color: theme.colorScheme.outlineVariant),
              SizedBox(width: 390, child: _buildStepsPanel(theme)),
            ],
          );
        }
        // 移动端：地图区独占全屏（内含顶部控件栏+地图+底部当前步骤面板）。
        // 不再单独保留 140px 步骤列表栏，让地图获得最大高度（视觉放大），
        // 用户通过底部当前步骤面板的「完成此步」推进、或点击地图标注切换。
        return map;
      },
    );
  }

  Widget _buildMapPanel(ThemeData theme, bool desktop) {
    // 控件与地图分层布局（非浮动）：
    // Flutter Web 下 HtmlElementView 创建的 iframe 是真实 DOM 元素，
    // z-index 高于 Flutter canvas，浮动在地图上的 Positioned 控件会被
    // iframe 遮挡无法点击。改为 Column 上下分层：顶部控件栏 → 中间地图
    // （Expanded 占满）→ 底部当前步骤面板，控件在地图之外可正常点击。
    // 同时地图获得最大高度，视觉上“放大”。
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(8, 8, 8, 4),
          child:
              desktop ? _buildDesktopTopBar(theme) : _buildMobileTopBar(theme),
        ),
        Expanded(
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: _buildCampusMapCanvas(theme, desktop: desktop),
          ),
        ),
        Padding(
          padding: const EdgeInsets.fromLTRB(8, 4, 8, 8),
          child: _buildFloatingStepPanel(theme),
        ),
      ],
    );
  }

  Widget _buildHeader(ThemeData theme) {
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(18),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: theme.colorScheme.primary.withOpacity(0.1),
                borderRadius: BorderRadius.circular(16),
              ),
              child: Icon(Icons.route, color: theme.colorScheme.primary),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('新生报到实时导航',
                      style: theme.textTheme.titleMedium
                          ?.copyWith(fontWeight: FontWeight.bold)),
                  const SizedBox(height: 4),
                  Text('当前位置 → ${_campus.name} → 按报到顺序逐站办理',
                      style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant)),
                ],
              ),
            ),
            Text('${_completed.length}/${_steps.length} 已完成',
                style: theme.textTheme.labelLarge
                    ?.copyWith(color: theme.colorScheme.primary)),
          ],
        ),
      ),
    );
  }

  Widget _buildControls(ThemeData theme) {
    return Wrap(
      spacing: 10,
      runSpacing: 10,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: [
        SegmentedButton<_MapLayer>(
          segments: const [
            ButtonSegment(value: _MapLayer.standard, label: Text('标准')),
            ButtonSegment(value: _MapLayer.satellite, label: Text('卫星')),
          ],
          selected: {_layer},
          onSelectionChanged: (v) {
            final l = v.first;
            setState(() => _layer = l);
            // 通知地图切换底图图层（标准矢量 / 卫星影像）
            _mapController.setLayer(l == _MapLayer.satellite
                ? 'satellite'
                : 'standard');
          },
        ),
        SegmentedButton<_MapMode>(
          segments: const [
            ButtonSegment(value: _MapMode.twoD, label: Text('2D')),
            ButtonSegment(value: _MapMode.threeD, label: Text('3D')),
          ],
          selected: {_mode},
          onSelectionChanged: (v) {
            final m = v.first;
            setState(() => _mode = m);
            // 通知地图引擎切换 2D/3D 视角（BMapGL 原生倾斜透视+建筑=实景导航）
            _mapController.set3D(m == _MapMode.threeD);
          },
        ),
      ],
    );
  }

  Widget _buildCampusMapCanvas(ThemeData theme, {required bool desktop}) {
    // 仅使用百度地图（BMapGL），2D/3D 视角 + 标准/卫星图层切换。
    // VR 全景已由顶部「VR全景」Tab 独立提供，不再占用地图区域。
    const baiduAk = String.fromEnvironment('BAIDU_MAP_AK', defaultValue: '');
    return ClipRRect(
      borderRadius: BorderRadius.circular(12),
      child: Stack(
        fit: StackFit.expand,
        children: [
          // ── 百度地图底图 + 脉冲标注 ──
          Positioned.fill(
            child: BaiduCampusMapEmbed(
              baiduAk: baiduAk,
              amapAk: '',
              tencentAk: '',
              provider: 'baidu',
              steps: _stepsForMap,
              currentStep: _currentStep,
              campusId: _campus.id,
              editMode: _editMode,
              controller: _mapController,
              onStepSelected: (idx) => setState(() => _currentStep = idx),
              onMarkerMoved: _canEditNodes ? _onMarkerMoved : null,
            ),
          ),
          // ── 未配置 AK 时的友好提示 ──
          if (baiduAk.isEmpty)
            Positioned.fill(
              child: Container(
                color: Colors.black54,
                child: Center(
                  child: Column(mainAxisSize: MainAxisSize.min, children: [
                    const Icon(Icons.key_off_outlined,
                        color: Colors.white, size: 40),
                    const SizedBox(height: 8),
                    const Text('地图需要百度地图 AK',
                        style: TextStyle(
                            color: Colors.white,
                            fontWeight: FontWeight.bold)),
                    const SizedBox(height: 4),
                    const Text(
                        '构建时添加 --dart-define=BAIDU_MAP_AK=你的AK',
                        style: TextStyle(color: Colors.white70, fontSize: 11)),
                  ]),
                ),
              ),
            ),
          // ── 后端步骤加载中指示器 ──
          if (_loadingSteps)
            Positioned(
              top: 10,
              left: 0,
              right: 0,
              child: Center(
                child: Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.surface.withOpacity(0.9),
                    borderRadius: BorderRadius.circular(12),
                    boxShadow: [
                      BoxShadow(
                          color: Colors.black.withOpacity(0.1), blurRadius: 8)
                    ],
                  ),
                  child: Row(mainAxisSize: MainAxisSize.min, children: [
                    const SizedBox(
                      width: 14,
                      height: 14,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ),
                    const SizedBox(width: 8),
                    Text('加载报到节点...',
                        style: theme.textTheme.labelSmall),
                  ]),
                ),
              ),
            ),
        ],
      ),
    );
  }

  // _buildMapMiniCard 和 _buildCampusGateLabel 已随 CustomPainter 底图一并移除。
  // ignore: unused_element
  Widget _buildMapMiniCard(ThemeData theme) {
    return Container(
      width: 180,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface.withOpacity(0.94),
        borderRadius: BorderRadius.circular(16),
        boxShadow: [
          BoxShadow(color: Colors.black.withOpacity(0.12), blurRadius: 16)
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            children: [
              Icon(Icons.map, size: 18, color: theme.colorScheme.primary),
              const SizedBox(width: 6),
              Text('校园范围',
                  style: theme.textTheme.labelLarge
                      ?.copyWith(fontWeight: FontWeight.bold)),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            '${_campus.name}\n默认使用百度地图导航，页面内不要求登录地图账号。',
            style: theme.textTheme.bodySmall
                ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
          ),
        ],
      ),
    );
  }

  // ignore: unused_element
  Widget _buildCampusGateLabel(ThemeData theme) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface.withOpacity(0.9),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: theme.colorScheme.primary.withOpacity(0.25)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.flag_outlined, size: 16, color: theme.colorScheme.primary),
          const SizedBox(width: 6),
          Text(_campus.entrance, style: theme.textTheme.labelMedium),
        ],
      ),
    );
  }

  Widget _buildCampusSelector(ThemeData theme) {
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.account_balance_outlined,
                    size: 18, color: theme.colorScheme.primary),
                const SizedBox(width: 6),
                Text('选择报到校区',
                    style: theme.textTheme.labelLarge
                        ?.copyWith(fontWeight: FontWeight.bold)),
              ],
            ),
            const SizedBox(height: 10),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: List.generate(_campuses.length, (index) {
                final campus = _campuses[index];
                final selected = _campusIndex == index;
                return ChoiceChip(
                  selected: selected,
                  avatar: Icon(
                    selected ? Icons.check_circle : Icons.location_city,
                    size: 18,
                    color: selected ? theme.colorScheme.primary : null,
                  ),
                  label: Text(campus.name),
                  onSelected: (_) => _switchCampus(index),
                );
              }),
            ),
            const SizedBox(height: 8),
            Text('${_campus.entrance} · ${_campus.address}',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          ],
        ),
      ),
    );
  }

  Widget _buildMapBadge(ThemeData theme) {
    // 已弃用：原浮动在地图右上角的 2D/3D 角标会被 iframe 遮挡无法显示，
    // 改为在顶部控件栏中通过 SegmentedButton 体现 provider 与 mode。
    return const SizedBox.shrink();
  }

  /// 浮动当前步骤+操作面板（桌面端浮在地图底部、移动端浮在地图底部）。
  /// 合并原 _buildCurrentStepOverlay 与 _buildNavigationButtons，半透明可读，
  /// 不再占用 Column 垂直空间，从而避免地图被挤塌。
  Widget _buildFloatingStepPanel(ThemeData theme) {
    final step = _steps[_currentStep];
    final done = _completed.contains(_currentStep);
    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surface.withOpacity(0.96),
        borderRadius: BorderRadius.circular(16),
        boxShadow: [
          BoxShadow(color: Colors.black.withOpacity(0.18), blurRadius: 20)
        ],
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(step.icon, color: theme.colorScheme.primary, size: 18),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(
                    '${step.title} · ${step.location}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: theme.textTheme.labelLarge
                        ?.copyWith(fontWeight: FontWeight.bold),
                  ),
                ),
                FilledButton.tonal(
                  onPressed: _markCurrentDone,
                  style: FilledButton.styleFrom(
                    padding: const EdgeInsets.symmetric(horizontal: 12),
                    minimumSize: const Size(0, 32),
                  ),
                  child: Text(done ? '已完成' : '完成此步',
                      style: const TextStyle(fontSize: 12)),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Row(
              children: [
                Icon(Icons.schedule, size: 12,
                    color: theme.colorScheme.onSurfaceVariant),
                const SizedBox(width: 3),
                Text(step.duration,
                    style: theme.textTheme.labelSmall),
                const Spacer(),
                _compactBtn(
                  icon: Icons.my_location,
                  label: '导航',
                  onTap: () => _openUrl(_routeUrl),
                ),
                const SizedBox(width: 4),
                _compactBtn(
                  icon: Icons.copy,
                  label: _copiedText.isEmpty ? '复制' : _copiedText,
                  onTap: () {
                    Clipboard.setData(ClipboardData(text: _campus.address));
                    setState(() => _copiedText = '已复制');
                    Future.delayed(const Duration(seconds: 2), () {
                      if (mounted) setState(() => _copiedText = '');
                    });
                  },
                ),
                const SizedBox(width: 4),
                _compactBtn(
                  icon: Icons.restart_alt,
                  label: '重置',
                  onTap: _resetProgress,
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  /// 底部面板用紧凑文字按钮（无填充，节省高度）。
  Widget _compactBtn(
      {required IconData icon,
      required String label,
      required VoidCallback onTap}) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(6),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 4),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 13),
            const SizedBox(width: 2),
            Text(label, style: const TextStyle(fontSize: 11)),
          ],
        ),
      ),
    );
  }

  /// 移动端顶部紧凑控件栏：进度+校区+服务商+2D/3D，半透明浮在地图顶部。
  Widget _buildMobileTopBar(ThemeData theme) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface.withOpacity(0.95),
        borderRadius: BorderRadius.circular(14),
        boxShadow: [
          BoxShadow(color: Colors.black.withOpacity(0.12), blurRadius: 12)
        ],
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.route, size: 16, color: theme.colorScheme.primary),
              const SizedBox(width: 6),
              Text(_campus.name,
                  style: theme.textTheme.labelLarge
                      ?.copyWith(fontWeight: FontWeight.bold)),
              const SizedBox(width: 8),
              Text('${_completed.length}/${_steps.length}',
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: theme.colorScheme.primary)),
              const Spacer(),
              Wrap(
                spacing: 6,
                children: List.generate(_campuses.length, (index) {
                  final campus = _campuses[index];
                  final selected = _campusIndex == index;
                  return ChoiceChip(
                    selected: selected,
                    label: Text(campus.name,
                        style: const TextStyle(fontSize: 11)),
                    onSelected: (_) => _switchCampus(index),
                    visualDensity: VisualDensity.compact,
                  );
                }),
              ),
            ],
          ),
          const SizedBox(height: 6),
          Wrap(
            spacing: 6,
            runSpacing: 6,
            children: [
              SegmentedButton<_MapLayer>(
                segments: const [
                  ButtonSegment(value: _MapLayer.standard, label: Text('标准')),
                  ButtonSegment(
                      value: _MapLayer.satellite, label: Text('卫星')),
                ],
                selected: {_layer},
                onSelectionChanged: (v) {
                  final l = v.first;
                  setState(() => _layer = l);
                  _mapController.setLayer(
                      l == _MapLayer.satellite ? 'satellite' : 'standard');
                },
                style: const ButtonStyle(
                  visualDensity: VisualDensity.compact,
                  textStyle: WidgetStatePropertyAll(TextStyle(fontSize: 11)),
                ),
              ),
              SegmentedButton<_MapMode>(
                segments: const [
                  ButtonSegment(value: _MapMode.twoD, label: Text('2D')),
                  ButtonSegment(value: _MapMode.threeD, label: Text('3D')),
                ],
                selected: {_mode},
                onSelectionChanged: (v) {
                  final m = v.first;
                  setState(() => _mode = m);
                  _mapController.set3D(m == _MapMode.threeD);
                },
                style: const ButtonStyle(
                  visualDensity: VisualDensity.compact,
                  textStyle: WidgetStatePropertyAll(TextStyle(fontSize: 11)),
                ),
              ),
              if (_canEditNodes)
                PopupMenuButton<String>(
                  icon: Icon(Icons.admin_panel_settings,
                      size: 18, color: _editMode ? theme.colorScheme.primary : theme.colorScheme.onSurfaceVariant),
                  tooltip: '管理',
                  padding: const EdgeInsets.symmetric(horizontal: 6),
                  onSelected: (v) {
                    if (v == 'edit') {
                      setState(() => _editMode = !_editMode);
                      if (_editMode) {
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(
                            content: Text('编辑模式：拖动标注即可校正位置，松手自动保存'),
                            duration: Duration(seconds: 3),
                          ),
                        );
                      }
                    } else if (v == 'panel') {
                      _openAdminPanel();
                    }
                  },
                  itemBuilder: (_) => [
                    PopupMenuItem<String>(
                      value: 'edit',
                      child: Row(children: [
                        Icon(_editMode ? Icons.check_box_outlined : Icons.edit_location_alt,
                            size: 18, color: theme.colorScheme.primary),
                        const SizedBox(width: 8),
                        Text(_editMode ? '退出编辑节点' : '编辑节点（拖拽校正）'),
                      ]),
                    ),
                    const PopupMenuDivider(),
                    const PopupMenuItem<String>(
                      value: 'panel',
                      child: Row(children: [
                        Icon(Icons.settings, size: 18),
                        SizedBox(width: 8),
                        Text('流程管理（CRUD）'),
                      ]),
                    ),
                  ],
                ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildStepsPanel(ThemeData theme) {
    return Container(
      color: theme.colorScheme.surface,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
            child: Text('报到流程',
                style: theme.textTheme.titleMedium
                    ?.copyWith(fontWeight: FontWeight.bold)),
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
            child: Text('${_campus.name} · ${_campus.steps.length} 个节点',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          ),
          Expanded(
            child: ListView.builder(
              padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
              itemCount: _steps.length,
              itemBuilder: (context, index) => _buildStepCard(theme, index),
            ),
          ),
        ],
      ),
    );
  }

  /// 桌面端顶部浮动控件栏：单行紧凑布局，校区+服务商+2D/3D 全部在一行。
  Widget _buildDesktopTopBar(ThemeData theme) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface.withOpacity(0.95),
        borderRadius: BorderRadius.circular(14),
        boxShadow: [
          BoxShadow(color: Colors.black.withOpacity(0.12), blurRadius: 12)
        ],
      ),
      child: Row(
        children: [
          Icon(Icons.route, size: 18, color: theme.colorScheme.primary),
          const SizedBox(width: 6),
          Text(_campus.name,
              style: theme.textTheme.titleSmall
                  ?.copyWith(fontWeight: FontWeight.bold)),
          const SizedBox(width: 8),
          Text('${_completed.length}/${_steps.length}',
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: theme.colorScheme.primary)),
          const SizedBox(width: 14),
          Wrap(
            spacing: 6,
            children: List.generate(_campuses.length, (index) {
              final campus = _campuses[index];
              final selected = _campusIndex == index;
              return ChoiceChip(
                selected: selected,
                label: Text(campus.name,
                    style: const TextStyle(fontSize: 12)),
                onSelected: (_) => _switchCampus(index),
                visualDensity: VisualDensity.compact,
              );
            }),
          ),
          const Spacer(),
          SegmentedButton<_MapLayer>(
            segments: const [
              ButtonSegment(value: _MapLayer.standard, label: Text('标准')),
              ButtonSegment(value: _MapLayer.satellite, label: Text('卫星')),
            ],
            selected: {_layer},
            onSelectionChanged: (v) {
              final l = v.first;
              setState(() => _layer = l);
              _mapController.setLayer(
                  l == _MapLayer.satellite ? 'satellite' : 'standard');
            },
            style: const ButtonStyle(
              visualDensity: VisualDensity.compact,
              textStyle: WidgetStatePropertyAll(TextStyle(fontSize: 12)),
            ),
          ),
          const SizedBox(width: 8),
          SegmentedButton<_MapMode>(
            segments: const [
              ButtonSegment(value: _MapMode.twoD, label: Text('2D')),
              ButtonSegment(value: _MapMode.threeD, label: Text('3D')),
            ],
            selected: {_mode},
            onSelectionChanged: (v) {
              final m = v.first;
              setState(() => _mode = m);
              _mapController.set3D(m == _MapMode.threeD);
            },
            style: const ButtonStyle(
              visualDensity: VisualDensity.compact,
              textStyle: WidgetStatePropertyAll(TextStyle(fontSize: 12)),
            ),
          ),
          if (_canEditNodes) ...[
            const SizedBox(width: 8),
            PopupMenuButton<String>(
              icon: Icon(Icons.admin_panel_settings,
                  size: 20, color: _editMode ? theme.colorScheme.primary : theme.colorScheme.onSurfaceVariant),
              tooltip: '管理',
              onSelected: (v) {
                if (v == 'edit') {
                  setState(() => _editMode = !_editMode);
                  if (_editMode) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(
                        content: Text('编辑模式：拖动标注即可校正位置，松手自动保存'),
                        duration: Duration(seconds: 3),
                      ),
                    );
                  }
                } else if (v == 'panel') {
                  _openAdminPanel();
                }
              },
              itemBuilder: (_) => [
                PopupMenuItem<String>(
                  value: 'edit',
                  child: Row(children: [
                    Icon(_editMode ? Icons.check_box_outlined : Icons.edit_location_alt,
                        size: 18, color: theme.colorScheme.primary),
                    const SizedBox(width: 8),
                    Text(_editMode ? '退出编辑节点' : '编辑节点（拖拽校正）'),
                  ]),
                ),
                const PopupMenuDivider(),
                const PopupMenuItem<String>(
                  value: 'panel',
                  child: Row(children: [
                    Icon(Icons.settings, size: 18),
                    SizedBox(width: 8),
                    Text('流程管理（CRUD）'),
                  ]),
                ),
              ],
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildStepCard(ThemeData theme, int index) {
    final step = _steps[index];
    final done = _completed.contains(index);
    final active = _currentStep == index;
    final color = done
        ? const Color(0xFF2E7D32)
        : active
            ? theme.colorScheme.primary
            : theme.colorScheme.outline;
    return Card(
      elevation: active ? 2 : 0,
      color:
          active ? theme.colorScheme.primaryContainer.withOpacity(0.38) : null,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(14),
        side: BorderSide(
            color: active
                ? theme.colorScheme.primary
                : theme.colorScheme.outlineVariant),
      ),
      child: InkWell(
        borderRadius: BorderRadius.circular(14),
        onTap: () => setState(() => _currentStep = index),
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Column(
                children: [
                  CircleAvatar(
                    radius: 18,
                    backgroundColor: color.withOpacity(0.12),
                    child: Icon(done ? Icons.check : step.icon,
                        size: 18, color: color),
                  ),
                  if (index < _steps.length - 1)
                    Container(
                        width: 2, height: 92, color: color.withOpacity(0.25)),
                ],
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: Text('${index + 1}. ${step.title}',
                              style: theme.textTheme.titleSmall?.copyWith(
                                  fontWeight: FontWeight.bold,
                                  color: done ? color : null)),
                        ),
                        Text(
                            done
                                ? '已完成'
                                : active
                                    ? '进行中'
                                    : '待办理',
                            style: theme.textTheme.labelSmall
                                ?.copyWith(color: color)),
                      ],
                    ),
                    const SizedBox(height: 6),
                    _buildMetaLine(Icons.place_outlined, step.location),
                    _buildMetaLine(Icons.schedule, step.duration),
                    _buildMetaLine(Icons.assignment_outlined, step.task),
                    _buildMetaLine(
                        Icons.inventory_2_outlined, '材料：${step.materials}'),
                    _buildMetaLine(Icons.phone_outlined, step.contact),
                    const SizedBox(height: 8),
                    Text(step.note,
                        style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant)),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildMetaLine(IconData icon, String text) {
    return Padding(
      padding: const EdgeInsets.only(top: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 14),
          const SizedBox(width: 5),
          Expanded(child: Text(text, style: const TextStyle(fontSize: 12.5))),
        ],
      ),
    );
  }

  void _markCurrentDone() {
    setState(() {
      _completed.add(_currentStep);
      if (_currentStep < _steps.length - 1) _currentStep++;
    });
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
          content: Text(
              '已完成：${_steps[_currentStep == 0 ? 0 : _currentStep - 1].title}')),
    );
  }

  void _resetProgress() {
    setState(() {
      _completed.clear();
      _currentStep = 0;
    });
  }

  void _switchCampus(int index) {
    if (_campusIndex == index) return;
    setState(() {
      _campusIndex = index;
      _completed.clear();
      _currentStep = 0;
      _copiedText = '';
    });
    // 校区切换：让地图重新取景到新校区完整范围。
    // widget.campusId 的变化也会通过 didUpdateWidget 触发一次，
    // 但这里显式调用一次，保证地图在任何状态下都立即响应。
    _mapController.fitCampus(_campuses[index].id);
    // 重新加载新校区的后端步骤数据
    _loadStepsFromServer();
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('已切换到${_campus.name}报到流程')),
    );
  }

  Widget _buildServiceTab(ThemeData theme) {
    final tab = _tabs.firstWhere((t) => t.tab == _currentTab);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Container(
              width: 100,
              height: 100,
              decoration: BoxDecoration(
                color: tab.color.withOpacity(0.1),
                borderRadius: BorderRadius.circular(50),
              ),
              child: Icon(tab.icon, size: 48, color: tab.color),
            ),
            const SizedBox(height: 24),
            Text(tab.label,
                style: theme.textTheme.headlineSmall?.copyWith(
                  fontWeight: FontWeight.bold,
                )),
            const SizedBox(height: 8),
            Text(tab.subtitle,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
                textAlign: TextAlign.center),
            const SizedBox(height: 8),
            Text(tab.url,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.outline,
                  fontFamily: 'monospace',
                ),
                textAlign: TextAlign.center),
            const SizedBox(height: 32),
            SizedBox(
              width: double.infinity,
              height: 52,
              child: ElevatedButton.icon(
                onPressed: () => _openUrl(tab.url),
                icon: const Icon(Icons.open_in_new, size: 20),
                label: Text('打开 ${tab.label}'),
                style: ElevatedButton.styleFrom(
                  backgroundColor: tab.color,
                  foregroundColor: Colors.white,
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(14)),
                  textStyle: const TextStyle(
                      fontSize: 16, fontWeight: FontWeight.w600),
                ),
              ),
            ),
            const SizedBox(height: 16),
            SizedBox(
              width: double.infinity,
              height: 44,
              child: OutlinedButton.icon(
                onPressed: () {
                  Clipboard.setData(ClipboardData(text: tab.url));
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(
                        content: Text('链接已复制'), duration: Duration(seconds: 1)),
                  );
                },
                icon: const Icon(Icons.copy, size: 18),
                label: const Text('复制链接'),
                style: OutlinedButton.styleFrom(
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(12)),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  String get _routeUrl {
    final step = _steps[_currentStep];
    final encodedName = Uri.encodeComponent('${_campus.name} ${step.title}');
    // 仅使用百度地图导航
    return 'https://api.map.baidu.com/direction?destination=latlng:${step.lat},${step.lng}|name:$encodedName&mode=walking&output=html&coord_type=wgs84';
  }

  Future<void> _openUrl(String url) async {
    final uri = Uri.parse(url);
    await launchUrl(uri, mode: LaunchMode.externalApplication);
  }
}

// CustomPainter 示意图实现已由百度地图 embed 替换，保留供参考。
// ignore: unused_element
class _CampusMapPainter extends CustomPainter {
  final ThemeData theme;
  final List<_CheckinStep> steps;
  final int currentStep;
  final Set<int> completed;

  _CampusMapPainter(this.theme, this.steps, this.currentStep, this.completed);

  @override
  void paint(Canvas canvas, Size size) {
    final campusPaint = Paint()
      ..color = theme.colorScheme.surface.withOpacity(0.62)
      ..style = PaintingStyle.fill;
    final campusBorder = Paint()
      ..color = theme.colorScheme.primary.withOpacity(0.28)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2;
    final roadPaint = Paint()
      ..color = Colors.white.withOpacity(0.72)
      ..style = PaintingStyle.stroke
      ..strokeCap = StrokeCap.round
      ..strokeWidth = 18;
    final roadLinePaint = Paint()
      ..color = theme.colorScheme.primary.withOpacity(0.22)
      ..style = PaintingStyle.stroke
      ..strokeCap = StrokeCap.round
      ..strokeWidth = 2;
    final routePaint = Paint()
      ..color = theme.colorScheme.primary
      ..style = PaintingStyle.stroke
      ..strokeCap = StrokeCap.round
      ..strokeWidth = 4;

    final campus = Path()
      ..moveTo(size.width * 0.14, size.height * 0.18)
      ..quadraticBezierTo(size.width * 0.45, size.height * 0.04,
          size.width * 0.84, size.height * 0.18)
      ..quadraticBezierTo(size.width * 0.95, size.height * 0.48,
          size.width * 0.80, size.height * 0.82)
      ..quadraticBezierTo(size.width * 0.46, size.height * 0.95,
          size.width * 0.16, size.height * 0.78)
      ..quadraticBezierTo(size.width * 0.05, size.height * 0.45,
          size.width * 0.14, size.height * 0.18)
      ..close();
    canvas.drawPath(campus, campusPaint);
    canvas.drawPath(campus, campusBorder);

    final road = Path()
      ..moveTo(size.width * 0.16, size.height * 0.70)
      ..cubicTo(size.width * 0.32, size.height * 0.58, size.width * 0.40,
          size.height * 0.45, size.width * 0.52, size.height * 0.48)
      ..cubicTo(size.width * 0.68, size.height * 0.52, size.width * 0.72,
          size.height * 0.34, size.width * 0.83, size.height * 0.25);
    canvas.drawPath(road, roadPaint);
    canvas.drawPath(road, roadLinePaint);

    final route = Path();
    final points = _points(size);
    for (var i = 0; i < points.length; i++) {
      if (i == 0) {
        route.moveTo(points[i].dx, points[i].dy);
      } else {
        route.lineTo(points[i].dx, points[i].dy);
      }
    }
    canvas.drawPath(route, routePaint);

    for (var i = 0; i < points.length; i++) {
      final active = i == currentStep;
      final done = completed.contains(i);
      final color = done
          ? const Color(0xFF2E7D32)
          : active
              ? theme.colorScheme.primary
              : theme.colorScheme.outline;
      final outer = Paint()..color = color.withOpacity(active ? 0.24 : 0.14);
      final inner = Paint()..color = color;
      canvas.drawCircle(points[i], active ? 18 : 14, outer);
      canvas.drawCircle(points[i], active ? 8 : 6, inner);
      _drawText(canvas, '${i + 1}', points[i] + const Offset(13, -27), color,
          active ? 14 : 12, FontWeight.bold);
      _drawText(canvas, steps[i].location, points[i] + const Offset(13, -8),
          theme.colorScheme.onSurface, 12, FontWeight.w600);
    }
  }

  List<Offset> _points(Size size) {
    const anchors = [
      Offset(0.18, 0.70),
      Offset(0.36, 0.56),
      Offset(0.52, 0.48),
      Offset(0.70, 0.58),
      Offset(0.64, 0.34),
      Offset(0.82, 0.26),
    ];
    return List.generate(steps.length, (i) {
      final a = anchors[i.clamp(0, anchors.length - 1)];
      return Offset(size.width * a.dx, size.height * a.dy);
    });
  }

  void _drawText(Canvas canvas, String text, Offset offset, Color color,
      double fontSize, FontWeight weight) {
    final span = TextSpan(
        text: text,
        style: TextStyle(color: color, fontSize: fontSize, fontWeight: weight));
    final painter =
        TextPainter(text: span, maxLines: 1, textDirection: TextDirection.ltr)
          ..layout(maxWidth: 140);
    painter.paint(canvas, offset);
  }

  @override
  bool shouldRepaint(covariant _CampusMapPainter oldDelegate) {
    return oldDelegate.currentStep != currentStep ||
        oldDelegate.completed.length != completed.length ||
        oldDelegate.steps != steps;
  }
}
