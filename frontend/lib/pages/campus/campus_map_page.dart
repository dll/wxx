import 'dart:html' as html;

import 'package:flutter/material.dart';

/// 校园服务入口 — 地图导航 / VR全景 / 学校首页 / 招生抖音
class CampusMapPage extends StatelessWidget {
  const CampusMapPage({super.key});

  static const _schoolHomeUrl = 'https://www.chzu.edu.cn';
  static const _vrUrl = 'https://www.chzu.edu.cn/vr/index.html';
  static const _douyinUrl = 'https://www.douyin.com/user/54452972915';
  static const _amapUrl = 'https://uri.amap.com/navigation?to=118.2988,32.2921,%E6%BB%81%E5%B7%9E%E5%AD%A6%E9%99%A2&mode=car&coordinate=gaode';
  static const _qqmapUrl = 'https://apis.map.qq.com/uri/v1/routeplan?type=drive&to=%E6%BB%81%E5%B7%9E%E5%AD%A6%E9%99%A2&tolat=32.2921&tolng=118.2988';

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('校园服务')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildSectionHeader(theme, '校园导航', Icons.map_outlined),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(child: _buildNavCard(
                theme,
                icon: Icons.directions_car,
                label: '高德地图',
                subtitle: '导航到校',
                color: const Color(0xFF1677FF),
                url: _amapUrl,
              )),
              const SizedBox(width: 12),
              Expanded(child: _buildNavCard(
                theme,
                icon: Icons.map,
                label: '腾讯地图',
                subtitle: '导航到校',
                color: const Color(0xFF07C160),
                url: _qqmapUrl,
              )),
            ],
          ),
          const SizedBox(height: 12),
          Text('地址：安徽省滁州市会峰西路1号 滁州学院',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              )),
          const SizedBox(height: 24),
          _buildSectionHeader(theme, '校园探索', Icons.explore_outlined),
          const SizedBox(height: 12),
          _buildActionCard(
            context, theme,
            icon: Icons.view_in_ar,
            label: 'VR 全景校园',
            subtitle: '足不出户漫游校园',
            color: const Color(0xFF7B1FA2),
            url: _vrUrl,
          ),
          const SizedBox(height: 12),
          _buildActionCard(
            context, theme,
            icon: Icons.school,
            label: '学校首页',
            subtitle: '滁州学院官方网站',
            color: const Color(0xFF1565C0),
            url: _schoolHomeUrl,
          ),
          const SizedBox(height: 24),
          _buildSectionHeader(theme, '招生就业', Icons.work_outline),
          const SizedBox(height: 12),
          _buildActionCard(
            context, theme,
            icon: Icons.music_note,
            label: '招生抖音',
            subtitle: '关注滁州学院招生办',
            color: const Color(0xFFC62828),
            url: _douyinUrl,
          ),
        ],
      ),
    );
  }

  Widget _buildSectionHeader(ThemeData theme, String title, IconData icon) {
    return Row(
      children: [
        Icon(icon, size: 20, color: theme.colorScheme.primary),
        const SizedBox(width: 8),
        Text(title, style: theme.textTheme.titleMedium?.copyWith(
          fontWeight: FontWeight.bold,
        )),
      ],
    );
  }

  Widget _buildNavCard(
    ThemeData theme, {
    required IconData icon,
    required String label,
    required String subtitle,
    required Color color,
    required String url,
  }) {
    return Material(
      color: color.withValues(alpha: 0.06),
      borderRadius: BorderRadius.circular(12),
      child: InkWell(
        onTap: () => _openUrl(url),
        borderRadius: BorderRadius.circular(12),
        child: Container(
          padding: const EdgeInsets.symmetric(vertical: 24, horizontal: 16),
          child: Column(
            children: [
              Icon(icon, size: 36, color: color),
              const SizedBox(height: 8),
              Text(label, style: TextStyle(
                fontWeight: FontWeight.w600,
                color: color,
              )),
              const SizedBox(height: 4),
              Text(subtitle, style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              )),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildActionCard(
    BuildContext context, ThemeData theme, {
    required IconData icon,
    required String label,
    required String subtitle,
    required Color color,
    required String url,
  }) {
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: ListTile(
        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        leading: Icon(icon, color: color, size: 28),
        title: Text(label, style: const TextStyle(fontWeight: FontWeight.w600)),
        subtitle: Text(subtitle, style: theme.textTheme.bodySmall),
        trailing: const Icon(Icons.open_in_new, size: 18),
        onTap: () => _openUrl(url),
      ),
    );
  }

  void _openUrl(String url) {
    html.window.open(url, '_blank');
  }
}
