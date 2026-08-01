import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:url_launcher/url_launcher.dart';
import '../../widgets/baidu_campus_map_embed.dart';

class CampusMapPage extends StatefulWidget {
  const CampusMapPage({super.key});

  @override
  State<CampusMapPage> createState() => _CampusMapPageState();
}

enum _CampusTab { map, vr, home, yxwz, douyin }

enum _MapProvider { amap, baidu, tencent }

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

const _huifengSteps = [
  _CheckinStep(
    title: '校门入校核验',
    location: '会峰校区南门',
    lat: 32.2921,
    lng: 118.2988,
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
    lat: 32.2932,
    lng: 118.3005,
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
    lat: 32.2928,
    lng: 118.2995,
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
    lat: 32.2940,
    lng: 118.2976,
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
    location: '校医院 / 教务处学籍点',
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

const _langyaSteps = [
  _CheckinStep(
    title: '校门入校核验',
    location: '琅琊校区主入口',
    lat: 32.3136,
    lng: 118.3098,
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
    lat: 32.3142,
    lng: 118.3107,
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
    lat: 32.3140,
    lng: 118.3094,
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
    lat: 32.3150,
    lng: 118.3089,
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
    lat: 32.3138,
    lng: 118.3101,
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
    lat: 32.3147,
    lng: 118.3102,
    duration: '约 30-45 分钟',
    task: '按学院批次完成体检、照片采集和学籍信息核验。',
    materials: '身份证、体检表、录取通知书',
    contact: '校医院 0550-3510120 / 教务处 0550-3510015',
    note: '抽血项目一般需空腹，请按学院通知批次办理。',
    icon: Icons.health_and_safety_outlined,
  ),
];

const _campuses = [
  _CampusPlan(
    id: 'huifeng',
    name: '会峰校区',
    address: '安徽省滁州市会峰西路1号 滁州学院会峰校区',
    lat: 32.2921,
    lng: 118.2988,
    entrance: '会峰校区南门',
    steps: _huifengSteps,
  ),
  _CampusPlan(
    id: 'langya',
    name: '琅琊校区',
    address: '安徽省滁州市琅琊区 滁州学院琅琊校区',
    lat: 32.3136,
    lng: 118.3098,
    entrance: '琅琊校区主入口',
    steps: _langyaSteps,
  ),
];

class _CampusMapPageState extends State<CampusMapPage> {
  _CampusTab _currentTab = _CampusTab.map;
  _MapProvider _provider = _MapProvider.baidu;
  _MapMode _mode = _MapMode.twoD;
  int _campusIndex = 0;
  int _currentStep = 0;
  final Set<int> _completed = {};
  String _copiedText = '';
  /// 地图控制器，用于从外部（如管理员编辑后）同步刷新标注。
  final _mapController = BaiduCampusMapController();

  _CampusPlan get _campus => _campuses[_campusIndex];
  List<_CheckinStep> get _steps => _campus.steps;

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
        final steps = _buildStepsPanel(theme);
        if (desktop) {
          return Row(
            // 让地图和步骤栏都占满整屏高度，避免地图被压矮
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Expanded(flex: 7, child: map),
              VerticalDivider(
                  width: 1, color: theme.colorScheme.outlineVariant),
              SizedBox(width: 390, child: steps),
            ],
          );
        }
        // 移动端：地图给更大比例(7:3)，配合地图内浮动控件避免 Column 固定项挤塌地图。
        return Column(
          children: [
            Expanded(flex: 7, child: map),
            const Divider(height: 1),
            Expanded(flex: 3, child: steps),
          ],
        );
      },
    );
  }

  Widget _buildMapPanel(ThemeData theme, bool desktop) {
    // 桌面端和移动端统一使用「全屏地图 + 浮动控件」布局：
    // 地图填满整个面板，所有 UI 元素（校区选择、服务商切换、当前步骤等）
    // 以半透明浮动层叠加在地图上方，不再用 Column 固定项挤占地图高度。
    // 这样地图始终获得面板 100% 高度，彻底解决「地图太扁」问题。
    return Padding(
      padding: const EdgeInsets.all(8),
      child: Stack(
        children: [
          // ── 地图画布填满整个面板 ──
          Positioned.fill(
            child: _buildCampusMapCanvas(theme, desktop: desktop),
          ),
          // ── 顶部浮动控件栏 ──
          Positioned(
            top: 8,
            left: 8,
            right: 8,
            child: desktop
                ? _buildDesktopTopBar(theme)
                : _buildMobileTopBar(theme),
          ),
          // ── 底部浮动当前步骤+操作面板 ──
          Positioned(
            left: 8,
            right: 8,
            bottom: 8,
            child: _buildFloatingStepPanel(theme),
          ),
        ],
      ),
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
        SegmentedButton<_MapProvider>(
          segments: const [
            ButtonSegment(value: _MapProvider.baidu, label: Text('百度')),
            ButtonSegment(value: _MapProvider.amap, label: Text('高德')),
            ButtonSegment(value: _MapProvider.tencent, label: Text('腾讯')),
          ],
          selected: {_provider},
          onSelectionChanged: (v) => setState(() => _provider = v.first),
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
            // 通知地图引擎切换 2D/3D 视角（BMapGL 原生倾斜透视+建筑）
            _mapController.set3D(m == _MapMode.threeD);
          },
        ),
      ],
    );
  }

  Widget _buildCampusMapCanvas(ThemeData theme, {required bool desktop}) {
    // 2D/3D 都在同一张地图上切换视角（3D=倾斜透视+建筑），
    // VR 全景已由顶部「VR全景」Tab 独立提供，不再占用地图区域。
    const baiduAk = String.fromEnvironment('BAIDU_MAP_AK', defaultValue: '');
    const amapAk = String.fromEnvironment('GAODE_MAP_AK', defaultValue: '');
    const tencentAk = String.fromEnvironment('TENXUN_MAP_AK', defaultValue: '');
    final provider = _provider.name; // baidu / amap / tencent
    final ak = switch (_provider) {
      _MapProvider.baidu => baiduAk,
      _MapProvider.amap => amapAk,
      _MapProvider.tencent => tencentAk,
    };
    return ClipRRect(
      borderRadius: BorderRadius.circular(18),
      child: Stack(children: [
        // ── 真实地图底图 + 脉冲标注（百度/高德/腾讯三家可切）──
        // ValueKey(provider)：切换地图服务商时强制重建 iframe，
        // 保证不同 provider 加载各自的 HTML 与 AK，不复用旧 iframe。
        Positioned.fill(
          child: BaiduCampusMapEmbed(
            key: ValueKey('campus-map-$provider'),
            baiduAk: baiduAk,
            amapAk: amapAk,
            tencentAk: tencentAk,
            provider: provider,
            steps: _stepsForMap,
            currentStep: _currentStep,
            campusId: _campus.id,
            controller: _mapController,
            onStepSelected: (idx) => setState(() => _currentStep = idx),
          ),
        ),
        // ── 当前模式角标（2D / 3D）──
        Positioned(top: 10, right: 10, child: _buildMapBadge(theme)),
        // ── 未配置 AK 时的友好提示 ──
        if (ak.isEmpty)
          Positioned.fill(
            child: Container(
              color: Colors.black54,
              child: Center(
                child: Column(mainAxisSize: MainAxisSize.min, children: [
                  const Icon(Icons.key_off_outlined,
                      color: Colors.white, size: 40),
                  const SizedBox(height: 8),
                  Text('地图需要$_providerLabel AK',
                      style: const TextStyle(
                          color: Colors.white,
                          fontWeight: FontWeight.bold)),
                  const SizedBox(height: 4),
                  Text(
                      '构建时添加 --dart-define=$_providerAkName=你的AK',
                      style: const TextStyle(color: Colors.white70, fontSize: 11)),
                ]),
              ),
            ),
          ),
      ]),
    );
  }

  String get _providerAkName {
    switch (_provider) {
      case _MapProvider.baidu:
        return 'BAIDU_MAP_AK';
      case _MapProvider.tencent:
        return 'TENXUN_MAP_AK';
      case _MapProvider.amap:
        return 'GAODE_MAP_AK';
    }
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
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface.withOpacity(0.92),
        borderRadius: BorderRadius.circular(999),
        boxShadow: [
          BoxShadow(color: Colors.black.withOpacity(0.12), blurRadius: 12)
        ],
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(_mode == _MapMode.twoD ? Icons.map_outlined : Icons.view_in_ar,
              size: 16, color: theme.colorScheme.primary),
          const SizedBox(width: 6),
          Text(
              '$_providerLabel · ${_mode == _MapMode.twoD ? '2D 地图' : '3D/全景'}'),
        ],
      ),
    );
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
        padding: const EdgeInsets.all(12),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(step.icon, color: theme.colorScheme.primary, size: 20),
                const SizedBox(width: 8),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text('当前步骤：${step.title}',
                          style: theme.textTheme.titleSmall
                              ?.copyWith(fontWeight: FontWeight.bold)),
                      const SizedBox(height: 2),
                      Text('${step.location} · ${step.duration}',
                          style: theme.textTheme.bodySmall),
                    ],
                  ),
                ),
                FilledButton.tonal(
                  onPressed: _markCurrentDone,
                  child: Text(done ? '已完成' : '完成此步'),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                FilledButton.icon(
                  onPressed: () => _openUrl(_routeUrl),
                  icon: const Icon(Icons.my_location, size: 18),
                  label: Text('导航到${step.location}'),
                ),
                OutlinedButton.icon(
                  onPressed: () {
                    Clipboard.setData(ClipboardData(text: _campus.address));
                    setState(() => _copiedText = '地址已复制');
                    Future.delayed(const Duration(seconds: 2), () {
                      if (mounted) setState(() => _copiedText = '');
                    });
                  },
                  icon: const Icon(Icons.copy, size: 18),
                  label: Text(_copiedText.isEmpty ? '复制地址' : _copiedText),
                ),
                OutlinedButton.icon(
                  onPressed: _resetProgress,
                  icon: const Icon(Icons.restart_alt, size: 18),
                  label: const Text('重置'),
                ),
              ],
            ),
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
              SegmentedButton<_MapProvider>(
                segments: const [
                  ButtonSegment(value: _MapProvider.baidu, label: Text('百度')),
                  ButtonSegment(
                      value: _MapProvider.amap, label: Text('高德')),
                  ButtonSegment(
                      value: _MapProvider.tencent, label: Text('腾讯')),
                ],
                selected: {_provider},
                onSelectionChanged: (v) => setState(() => _provider = v.first),
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
          SegmentedButton<_MapProvider>(
            segments: const [
              ButtonSegment(value: _MapProvider.baidu, label: Text('百度')),
              ButtonSegment(value: _MapProvider.amap, label: Text('高德')),
              ButtonSegment(value: _MapProvider.tencent, label: Text('腾讯')),
            ],
            selected: {_provider},
            onSelectionChanged: (v) => setState(() => _provider = v.first),
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

  String get _providerLabel {
    switch (_provider) {
      case _MapProvider.baidu:
        return '百度地图';
      case _MapProvider.tencent:
        return '腾讯地图';
      case _MapProvider.amap:
        return '高德地图';
    }
  }

  String get _routeUrl {
    final step = _steps[_currentStep];
    final encodedName = Uri.encodeComponent('${_campus.name} ${step.title}');
    switch (_provider) {
      case _MapProvider.baidu:
        return 'https://api.map.baidu.com/direction?destination=latlng:${step.lat},${step.lng}|name:$encodedName&mode=walking&output=html&coord_type=wgs84';
      case _MapProvider.tencent:
        return 'https://apis.map.qq.com/uri/v1/routeplan?type=walk&to=$encodedName&tolat=${step.lat}&tolng=${step.lng}&referer=wxx';
      case _MapProvider.amap:
        return 'https://uri.amap.com/navigation?to=${step.lng},${step.lat},$encodedName&mode=walk&coordinate=gaode';
    }
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
