import 'dart:html' as html;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// 校园服务入口 — 高德地图 / VR全景 / 学校首页 / 招生抖音
class CampusMapPage extends StatefulWidget {
  const CampusMapPage({super.key});

  @override
  State<CampusMapPage> createState() => _CampusMapPageState();
}

enum _CampusTab { map, vr, home, douyin }

class _CampusTabInfo {
  final _CampusTab tab;
  final String label;
  final IconData icon;
  final Color color;
  final String url;
  final String subtitle;
  const _CampusTabInfo(this.tab, this.label, this.icon, this.color, this.url, this.subtitle);
}

const _tabs = [
  _CampusTabInfo(_CampusTab.map, '地图', Icons.map_outlined, Color(0xFF1677FF), '', '导航到校'),
  _CampusTabInfo(_CampusTab.vr, 'VR全景', Icons.view_in_ar, Color(0xFF7B1FA2), 'https://www.chzu.edu.cn/vr/index.html', '足不出户漫游校园'),
  _CampusTabInfo(_CampusTab.home, '官网', Icons.school, Color(0xFF1565C0), 'https://www.chzu.edu.cn', '滁州学院官方网站'),
  _CampusTabInfo(_CampusTab.douyin, '抖音', Icons.music_note, Color(0xFFC62828), 'https://www.douyin.com/search/%E6%BB%81%E5%B7%9E%E5%AD%A6%E9%99%A2', '搜索滁州学院官方抖音'),
];

class _CampusMapPageState extends State<CampusMapPage> {
  _CampusTab _currentTab = _CampusTab.map;
  String _copiedText = '';
  double _imageScale = 1.0;

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
                          Icon(t.icon, size: 16, color: selected ? t.color : theme.colorScheme.onSurfaceVariant),
                          const SizedBox(width: 4),
                          Text(
                            t.label,
                            style: TextStyle(
                              fontSize: 13,
                              fontWeight: selected ? FontWeight.w600 : FontWeight.normal,
                              color: selected ? t.color : theme.colorScheme.onSurfaceVariant,
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
      body: _currentTab == _CampusTab.map ? _buildMapTab(theme) : _buildServiceTab(theme),
    );
  }

  Widget _buildMapTab(ThemeData theme) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        // 地址卡片
        Card(
          elevation: 0,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(16),
            side: BorderSide(color: theme.colorScheme.outlineVariant),
          ),
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(
              children: [
                Container(
                  width: 80,
                  height: 80,
                  decoration: BoxDecoration(
                    color: const Color(0xFF1677FF).withValues(alpha: 0.1),
                    borderRadius: BorderRadius.circular(40),
                  ),
                  child: const Icon(Icons.map_outlined, size: 40, color: Color(0xFF1677FF)),
                ),
                const SizedBox(height: 16),
                Text('滁州学院（会峰校区）',
                    style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    const Icon(Icons.location_on, size: 16, color: Colors.red),
                    const SizedBox(width: 4),
                    Text('安徽省滁州市会峰西路1号',
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        )),
                    IconButton(
                      icon: const Icon(Icons.copy, size: 16),
                      tooltip: '复制地址',
                      onPressed: () {
                        Clipboard.setData(const ClipboardData(text: '安徽省滁州市会峰西路1号 滁州学院'));
                        setState(() => _copiedText = '地址已复制');
                        Future.delayed(const Duration(seconds: 2), () {
                          if (mounted) setState(() => _copiedText = '');
                        });
                      },
                      constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
                      padding: EdgeInsets.zero,
                    ),
                  ],
                ),
                const SizedBox(height: 4),
                Text('118.2988, 32.2921',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.outline,
                      fontFamily: 'monospace',
                    )),
                if (_copiedText.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.only(top: 4),
                    child: Text(_copiedText,
                        style: const TextStyle(fontSize: 12, color: Colors.green)),
                  ),
              ],
            ),
          ),
        ),
        const SizedBox(height: 20),
        // 导航按钮
        Row(
          children: [
            Expanded(
              child: SizedBox(
                height: 48,
                child: ElevatedButton.icon(
                  onPressed: () => html.window.open(
                    'https://uri.amap.com/navigation?to=118.2988,32.2921,%E6%BB%81%E5%B7%9E%E5%AD%A6%E9%99%A2&mode=car&coordinate=gaode',
                    '_blank',
                  ),
                  icon: const Icon(Icons.directions_car, size: 18),
                  label: const Text('高德地图导航'),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: const Color(0xFF1677FF),
                    foregroundColor: Colors.white,
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                  ),
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(
              child: SizedBox(
                height: 48,
                child: ElevatedButton.icon(
                  onPressed: () => html.window.open(
                    'https://apis.map.qq.com/uri/v1/routeplan?type=drive&to=%E6%BB%81%E5%B7%9E%E5%AD%A6%E9%99%A2&tolat=32.2921&tolng=118.2988',
                    '_blank',
                  ),
                  icon: const Icon(Icons.map, size: 18),
                  label: const Text('腾讯地图导航'),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: const Color(0xFF07C160),
                    foregroundColor: Colors.white,
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                  ),
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 12),
        _buildEnrollmentMapCard(theme),
        const SizedBox(height: 12),
        // 路线说明
        Card(
          elevation: 0,
          color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.5),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Icon(Icons.info_outline, size: 16, color: theme.colorScheme.primary),
                    const SizedBox(width: 6),
                    Text('交通指南', style: theme.textTheme.titleSmall),
                  ],
                ),
                const SizedBox(height: 8),
                _buildTransportItem(theme, Icons.directions_bus, '公交', '乘坐 4路、15路、18路、101路 到「滁州学院」站'),
                const SizedBox(height: 6),
                _buildTransportItem(theme, Icons.train, '高铁', '滁州站 → 乘 18路/101路 到滁州学院（约30分钟）'),
                const SizedBox(height: 6),
                _buildTransportItem(theme, Icons.local_taxi, '自驾', '导航至「滁州学院会峰校区」（会峰西路1号）'),
              ],
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildTransportItem(ThemeData theme, IconData icon, String label, String desc) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Icon(icon, size: 16, color: theme.colorScheme.primary),
        const SizedBox(width: 8),
        Text('$label：', style: const TextStyle(fontWeight: FontWeight.w500, fontSize: 13)),
        Expanded(child: Text(desc, style: TextStyle(fontSize: 13, color: theme.colorScheme.onSurfaceVariant))),
      ],
    );
  }

  Widget _buildEnrollmentMapCard(ThemeData theme) {
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.map_outlined, size: 20, color: const Color(0xFF1677FF)),
                const SizedBox(width: 8),
                Text('新生入学流程地图',
                    style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                const Spacer(),
                IconButton(
                  icon: const Icon(Icons.fullscreen, size: 20),
                  tooltip: '全屏查看',
                  onPressed: () => _showFullscreenMap(theme),
                ),
              ],
            ),
            const SizedBox(height: 12),
            ClipRRect(
              borderRadius: BorderRadius.circular(12),
              child: InteractiveViewer(
                minScale: 0.5,
                maxScale: 4.0,
                child: Image.asset(
                  'assets/images/会峰校区2003新生报到交通指示图01.png',
                  height: 200 * _imageScale,
                  width: double.infinity,
                  fit: BoxFit.contain,
                  errorBuilder: (context, error, stackTrace) => Container(
                    height: 150,
                    decoration: BoxDecoration(
                      color: theme.colorScheme.surfaceContainerHighest,
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Center(
                      child: Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          Icon(Icons.image_not_supported,
                              size: 40, color: theme.colorScheme.outline),
                          const SizedBox(height: 8),
                          Text('图片未加载',
                              style: TextStyle(color: theme.colorScheme.outline)),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Icon(Icons.photo_size_select_small,
                    size: 16, color: theme.colorScheme.outline),
                Expanded(
                  child: Slider(
                    value: _imageScale,
                    min: 0.5,
                    max: 2.0,
                    divisions: 15,
                    label: '${(_imageScale * 100).toInt()}%',
                    onChanged: (v) => setState(() => _imageScale = v),
                  ),
                ),
                Icon(Icons.photo_size_select_large,
                    size: 16, color: theme.colorScheme.outline),
              ],
            ),
            Center(
              child: Text('${(_imageScale * 100).toInt()}%',
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: theme.colorScheme.outline)),
            ),
            const SizedBox(height: 8),
            SizedBox(
              width: double.infinity,
              child: OutlinedButton.icon(
                onPressed: () => _showFullscreenMap(theme),
                icon: const Icon(Icons.open_in_full, size: 16),
                label: const Text('全屏查看'),
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

  void _showFullscreenMap(ThemeData theme) {
    showGeneralDialog(
      context: context,
      barrierDismissible: true,
      barrierLabel: '关闭',
      barrierColor: Colors.black87,
      transitionDuration: const Duration(milliseconds: 300),
      pageBuilder: (context, anim1, anim2) {
        return Scaffold(
          backgroundColor: Colors.black,
          appBar: AppBar(
            backgroundColor: Colors.black,
            foregroundColor: Colors.white,
            title: const Text('新生入学流程地图'),
            actions: [
              IconButton(
                icon: const Icon(Icons.close),
                onPressed: () => Navigator.of(context).pop(),
              ),
            ],
          ),
          body: InteractiveViewer(
            minScale: 0.5,
            maxScale: 5.0,
            child: Center(
              child: Image.asset(
                'assets/images/会峰校区2003新生报到交通指示图01.png',
                fit: BoxFit.contain,
                errorBuilder: (context, error, stackTrace) => const Center(
                  child: Text('图片未加载',
                      style: TextStyle(color: Colors.white)),
                ),
              ),
            ),
          ),
        );
      },
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
                color: tab.color.withValues(alpha: 0.1),
                borderRadius: BorderRadius.circular(50),
              ),
              child: Icon(tab.icon, size: 48, color: tab.color),
            ),
            const SizedBox(height: 24),
            Text(tab.label, style: theme.textTheme.headlineSmall?.copyWith(
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
                onPressed: () => html.window.open(tab.url, '_blank'),
                icon: const Icon(Icons.open_in_new, size: 20),
                label: Text('打开 ${tab.label}'),
                style: ElevatedButton.styleFrom(
                  backgroundColor: tab.color,
                  foregroundColor: Colors.white,
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                  textStyle: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
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
                    const SnackBar(content: Text('链接已复制'), duration: Duration(seconds: 1)),
                  );
                },
                icon: const Icon(Icons.copy, size: 18),
                label: const Text('复制链接'),
                style: OutlinedButton.styleFrom(
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
