import 'package:flutter/material.dart';

/// 骨架屏加载占位组件 — 用于列表/卡片/消息气泡的加载状态
class SkeletonBox extends StatefulWidget {
  final double width;
  final double height;
  final double borderRadius;

  const SkeletonBox({
    super.key,
    this.width = double.infinity,
    this.height = 16,
    this.borderRadius = 8,
  });

  @override
  State<SkeletonBox> createState() => _SkeletonBoxState();
}

class _SkeletonBoxState extends State<SkeletonBox>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;
  late final Animation<double> _anim;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1200),
    );
    _anim = Tween<double>(begin: 0.3, end: 0.7).animate(
      CurvedAnimation(parent: _ctrl, curve: Curves.easeInOut),
    );
    _ctrl.repeat(reverse: true);
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return FadeTransition(
      opacity: _anim,
      child: Container(
        width: widget.width,
        height: widget.height,
        decoration: BoxDecoration(
          color: theme.colorScheme.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(widget.borderRadius),
        ),
      ),
    );
  }
}

/// 对话页消息骨架屏 — 模拟 3 条消息气泡
class ChatSkeleton extends StatelessWidget {
  const ChatSkeleton({super.key});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        children: [
          // 用户消息
          _buildBubble(context, isUser: true, lines: 1),
          // AI 回复 + AnswerCard
          _buildBubble(context, isUser: false, lines: 3),
          const SizedBox(height: 12),
          // 用户追问
          _buildBubble(context, isUser: true, lines: 2),
          // AI 回复
          _buildBubble(context, isUser: false, lines: 4),
        ],
      ),
    );
  }

  Widget _buildBubble(BuildContext context, {required bool isUser, required int lines}) {
    final screenWidth = MediaQuery.of(context).size.width;
    final maxWidth = isUser ? screenWidth * 0.6 : screenWidth * 0.8;
    final alignment = isUser ? MainAxisAlignment.end : MainAxisAlignment.start;

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        mainAxisAlignment: alignment,
        children: [
          Container(
            width: maxWidth,
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.surfaceContainerHighest,
              borderRadius: isUser
                  ? const BorderRadius.only(
                      topLeft: Radius.circular(16),
                      topRight: Radius.circular(16),
                      bottomLeft: Radius.circular(16),
                      bottomRight: Radius.circular(4),
                    )
                  : const BorderRadius.only(
                      topLeft: Radius.circular(16),
                      topRight: Radius.circular(16),
                      bottomLeft: Radius.circular(4),
                      bottomRight: Radius.circular(16),
                    ),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: List.generate(
                lines,
                (_) => Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: SkeletonBox(
                    height: 14,
                    borderRadius: 4,
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

/// 会话列表骨架屏
class SessionsSkeleton extends StatelessWidget {
  const SessionsSkeleton({super.key});

  @override
  Widget build(BuildContext context) {
    return ListView.separated(
      padding: const EdgeInsets.all(12),
      itemCount: 6,
      separatorBuilder: (_, __) => const SizedBox(height: 8),
      itemBuilder: (context, index) => const _SessionItemSkeleton(),
    );
  }
}

class _SessionItemSkeleton extends StatelessWidget {
  const _SessionItemSkeleton();

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            const SkeletonBox(width: 40, height: 40, borderRadius: 20),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  SkeletonBox(
                    width: MediaQuery.of(context).size.width * 0.5,
                    height: 16,
                    borderRadius: 4,
                  ),
                  const SizedBox(height: 6),
                  SkeletonBox(
                    width: MediaQuery.of(context).size.width * 0.3,
                    height: 12,
                    borderRadius: 4,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
