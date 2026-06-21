import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

/// 校园服务入口 — 地图导航 / VR全景 / 学校首页 / 招生抖音
class CampusMapPage extends StatelessWidget {
  const CampusMapPage({super.key});

  static const _schoolHomeUrl = 'https://www.chzu.edu.cn';
  static const _vrUrl = 'https://www.chzu.edu.cn/vr/index.html';
  static const _douyinUrl = 'https://www.douyin.com/user/54452972915';

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
                onTap: () => _openMap(context, 'amap'),
              )),
              const SizedBox(width: 12),
              Expanded(child: _buildNavCard(
                theme,
                icon: Icons.map,
                label: '腾讯地图',
                subtitle: '导航到校',
                color: const Color(0xFF07C160),
                onTap: () => _openMap(context, 'qqmap'),
              )),
            ],
          ),
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
    required VoidCallback onTap,
  }) {
    return Material(
      color: color.withValues(alpha: 0.06),
      borderRadius: BorderRadius.circular(12),
      child: InkWell(
        onTap: onTap,
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
        onTap: () => _launchURL(context, url),
      ),
    );
  }

  Future<void> _openMap(BuildContext context, String mapType) async {
    final lat = 32.2921; // 滁州学院经纬度
    final lng = 118.2988;
    final name = '滁州学院';

    Uri? uri;
    if (mapType == 'amap') {
      uri = Uri.parse('androidamap://route?sourceApplication=蔚小芯&slat=&slon=&sname=&dlat=$lat&dlon=$lng&dname=$name&dev=0&t=0');
    } else if (mapType == 'qqmap') {
      uri = Uri.parse('qqmap://map/routeplan?type=drive&to=$name&tolat=$lat&tolng=$lng');
    }

    if (uri != null && await canLaunchUrl(uri)) {
      await launchUrl(uri);
    } else {
      // 降级到网页版
      final webUrl = mapType == 'amap'
          ? 'https://uri.amap.com/navigation?to=$lng,$lat,$name&mode=car&coordinate=gaode'
          : 'https://apis.map.qq.com/uri/v1/routeplan?type=drive&to=$name&tolat=$lat&tolng=$lng';
      await _launchURL(context, webUrl);
    }
  }

  Future<void> _launchURL(BuildContext context, String url) async {
    final uri = Uri.parse(url);
    if (await canLaunchUrl(uri)) {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    } else {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('无法打开链接')),
        );
      }
    }
  }
}
