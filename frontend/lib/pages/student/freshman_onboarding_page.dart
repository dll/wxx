import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../../utils/storage.dart';

/// 新生应用内引导（首次使用）
///
/// 以大一新生视角设计的三步极简引导：走进来先看懂"问芯 / 办事 / 报到"。
/// 展示后可点"开学待办"进入新生入学待办清单（本地持久化），
/// 也可直接进入首页。
class FreshmanOnboardingPage extends StatefulWidget {
  const FreshmanOnboardingPage({super.key});

  @override
  State<FreshmanOnboardingPage> createState() => _FreshmanOnboardingPageState();
}

class _FreshmanOnboardingPageState extends State<FreshmanOnboardingPage> {
  int _step = 0;
  final PageController _controller = PageController();

  static const _steps = [
    _OnboardItem(
      icon: Icons.auto_awesome,
      color: Color(0xFF1565C0),
      title: '问芯 —— 不懂就问',
      desc: '政策、流程、学习、校园生活，直接问蔚小芯。\n答案带来源、可追溯，不怕它瞎编。',
    ),
    _OnboardItem(
      icon: Icons.account_tree_outlined,
      color: Color(0xFF2E7D32),
      title: '办事 —— 办手续就找它',
      desc: '请假、入党、报到、离校……\n分步指引 + 材料清单 + 办理地点，一步不缺。',
    ),
    _OnboardItem(
      icon: Icons.checklist_rtl,
      color: Color(0xFFE65100),
      title: '报到 —— 从入学清单开始',
      desc: '报到→体检→军训→选课→领教材，\n开学待办一步步打勾，心里有数不慌张。',
    ),
  ];

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _finish() async {
    await Storage.setFreshmanGuideSeen();
    if (mounted) context.go('/home');
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      backgroundColor: theme.colorScheme.surface,
      body: SafeArea(
        child: Column(
          children: [
            // 顶部进度点
            Padding(
              padding: const EdgeInsets.fromLTRB(24, 20, 24, 0),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  for (int i = 0; i < _steps.length; i++)
                    AnimatedContainer(
                      duration: const Duration(milliseconds: 250),
                      margin: const EdgeInsets.symmetric(horizontal: 4),
                      width: _step == i ? 24 : 8,
                      height: 8,
                      decoration: BoxDecoration(
                        color: _step == i
                            ? theme.colorScheme.primary
                            : theme.colorScheme.outlineVariant,
                        borderRadius: BorderRadius.circular(4),
                      ),
                    ),
                ],
              ),
            ),
            Expanded(
              child: PageView(
                controller: _controller,
                onPageChanged: (i) => setState(() => _step = i),
                children: [
                  for (final s in _steps) _buildStep(theme, s),
                ],
              ),
            ),
            // 底部操作
            Padding(
              padding: const EdgeInsets.fromLTRB(24, 8, 24, 24),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  if (_step == _steps.length - 1)
                    SizedBox(
                      width: double.infinity,
                      child: FilledButton.icon(
                        onPressed: () => context.push('/student/freshman-agenda'),
                        icon: const Icon(Icons.checklist, size: 18),
                        label: const Text('打开我的开学待办'),
                        style: FilledButton.styleFrom(
                          padding: const EdgeInsets.symmetric(vertical: 14),
                        ),
                      ),
                    ),
                  const SizedBox(height: 10),
                  Row(
                    children: [
                      if (_step > 0)
                        TextButton(
                          onPressed: () {
                            _controller.previousPage(
                              duration: const Duration(milliseconds: 250),
                              curve: Curves.easeOut,
                            );
                          },
                          child: const Text('上一步'),
                        )
                      else
                        const SizedBox.shrink(),
                      const Spacer(),
                      if (_step < _steps.length - 1)
                        FilledButton(
                          onPressed: () {
                            _controller.nextPage(
                              duration: const Duration(milliseconds: 250),
                              curve: Curves.easeOut,
                            );
                          },
                          child: const Text('下一步'),
                        )
                      else
                        TextButton(
                          onPressed: _finish,
                          child: const Text('跳过，去首页'),
                        ),
                    ],
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildStep(ThemeData theme, _OnboardItem item) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 40),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Container(
            width: 120,
            height: 120,
            decoration: BoxDecoration(
              color: item.color.withOpacity(0.12),
              shape: BoxShape.circle,
            ),
            child: Icon(item.icon, size: 60, color: item.color),
          ),
          const SizedBox(height: 32),
          Text(
            item.title,
            textAlign: TextAlign.center,
            style: theme.textTheme.headlineSmall
                ?.copyWith(fontWeight: FontWeight.w800),
          ),
          const SizedBox(height: 14),
          Text(
            item.desc,
            textAlign: TextAlign.center,
            style: theme.textTheme.bodyLarge?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
              height: 1.6,
            ),
          ),
        ],
      ),
    );
  }
}

class _OnboardItem {
  final IconData icon;
  final Color color;
  final String title;
  final String desc;
  const _OnboardItem({
    required this.icon,
    required this.color,
    required this.title,
    required this.desc,
  });
}
