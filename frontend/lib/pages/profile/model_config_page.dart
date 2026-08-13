import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../models/models.dart';
import '../../providers/model_config_provider.dart';
import '../../providers/token_stats_provider.dart';

/// AI 模型配置页面 — 配置 DeepSeek / 智谱清言 / 讯飞星火 三大模型参数
/// 兼作 AI 使用入口：顶部展示本月 token 用量，绑定个人 Key 后对话不消耗校内额度
class ModelConfigPage extends StatefulWidget {
  const ModelConfigPage({super.key});

  @override
  State<ModelConfigPage> createState() => _ModelConfigPageState();
}

class _ModelConfigPageState extends State<ModelConfigPage> {
  final _pageController = PageController();
  int _currentTab = 0;

  // 各模型参数控制器
  final _dsKeyCtrl = TextEditingController();
  final _dsModelCtrl = TextEditingController();
  double _dsTemp = 0.7;
  int _dsMaxTokens = 2048;

  final _zpKeyCtrl = TextEditingController();
  final _zpModelCtrl = TextEditingController();
  double _zpTemp = 0.7;
  int _zpMaxTokens = 2048;

  final _xfAppIdCtrl = TextEditingController();
  final _xfKeyCtrl = TextEditingController();
  final _xfSecretCtrl = TextEditingController();
  final _xfModelCtrl = TextEditingController();
  double _xfTemp = 0.7;
  int _xfMaxTokens = 2048;

  String _defaultProvider = 'deepseek';
  bool _saving = false;
  bool _initialised = false;

  @override
  void initState() {
    super.initState();
    // 拉取本月 token 用量展示（用量数据来自 /token-stats/my）
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final p = context.read<TokenStatsProvider>();
      if (p.myStats == null) p.fetchMyStats();
    });
  }

  @override
  void dispose() {
    _pageController.dispose();
    _dsKeyCtrl.dispose();
    _dsModelCtrl.dispose();
    _zpKeyCtrl.dispose();
    _zpModelCtrl.dispose();
    _xfAppIdCtrl.dispose();
    _xfKeyCtrl.dispose();
    _xfSecretCtrl.dispose();
    _xfModelCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('AI 模型配置')),
      body: Consumer<ModelConfigProvider>(
        builder: (context, provider, _) {
          if (provider.loading && provider.config == null) {
            return const Center(child: CircularProgressIndicator());
          }
          if (!_initialised && provider.config != null) {
            _initialised = true;
            WidgetsBinding.instance.addPostFrameCallback((_) => _fillFromConfig(provider.config!));
          }
          return Column(
            children: [
              _buildUsageBanner(context),
              _buildTabBar(),
              Expanded(
                child: PageView(
                  controller: _pageController,
                  onPageChanged: (i) => setState(() => _currentTab = i),
                  children: [
                    _buildPanel(
                      keyCtrl: _dsKeyCtrl, keyHint: 'DeepSeek API Key',
                      modelCtrl: _dsModelCtrl, modelHint: '模型名称（如 deepseek-v4-flash）',
                      temp: _dsTemp, onTempChanged: (v) => _dsTemp = v,
                      maxTokens: _dsMaxTokens, onTokensChanged: (v) => _dsMaxTokens = v,
                    ),
                    _buildPanel(
                      keyCtrl: _zpKeyCtrl, keyHint: '智谱 API Key',
                      modelCtrl: _zpModelCtrl, modelHint: '模型名称（如 glm-4）',
                      temp: _zpTemp, onTempChanged: (v) => _zpTemp = v,
                      maxTokens: _zpMaxTokens, onTokensChanged: (v) => _zpMaxTokens = v,
                    ),
                    _buildPanel(
                      keyCtrl: _xfKeyCtrl, keyHint: '讯飞 API Key',
                      modelCtrl: _xfModelCtrl, modelHint: '模型名称（如 spark-v4.0）',
                      temp: _xfTemp, onTempChanged: (v) => _xfTemp = v,
                      maxTokens: _xfMaxTokens, onTokensChanged: (v) => _xfMaxTokens = v,
                      extraFields: [
                        _ExtraField(ctrl: _xfAppIdCtrl, hint: '讯飞 App ID'),
                        _ExtraField(ctrl: _xfSecretCtrl, hint: '讯飞 API Secret', obscure: true),
                      ],
                    ),
                  ],
                ),
              ),
            ],
          );
        },
      ),
      bottomNavigationBar: _buildBottomBar(context),
    );
  }

  void _fillFromConfig(ModelConfig config) {
    _dsKeyCtrl.text = config.deepseekKey;
    _dsModelCtrl.text = config.deepseekModel;
    _dsTemp = config.deepseekTemp;
    _dsMaxTokens = config.deepseekMaxTokens;

    _zpKeyCtrl.text = config.zhipuKey;
    _zpModelCtrl.text = config.zhipuModel;
    _zpTemp = config.zhipuTemp;
    _zpMaxTokens = config.zhipuMaxTokens;

    _xfAppIdCtrl.text = config.xunfeiAppId;
    _xfKeyCtrl.text = config.xunfeiKey;
    _xfSecretCtrl.text = config.xunfeiSecret;
    _xfModelCtrl.text = config.xunfeiModel;
    _xfTemp = config.xunfeiTemp;
    _xfMaxTokens = config.xunfeiMaxTokens;

    _defaultProvider = config.defaultProvider;
  }

  /// 本月 AI token 用量横幅（额度由管理员配置）
  Widget _buildUsageBanner(BuildContext context) {
    final theme = Theme.of(context);
    return Consumer<TokenStatsProvider>(
      builder: (context, provider, _) {
        final summary = provider.myStats?.summary;
        if (summary == null) {
          return const SizedBox.shrink();
        }
        final used = summary.totalTokens;
        final progress = (used / 100000).clamp(0.0, 1.0);
        return Container(
          margin: const EdgeInsets.fromLTRB(12, 12, 12, 4),
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: theme.colorScheme.primaryContainer.withOpacity(0.25),
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
                color: theme.colorScheme.primary.withOpacity(0.2)),
          ),
          child: Row(
            children: [
              Icon(Icons.speed_outlined,
                  size: 20, color: theme.colorScheme.primary),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('本月 AI 对话用量：${_fmt(used)} tokens',
                        style: theme.textTheme.bodySmall?.copyWith(
                          fontWeight: FontWeight.w700,
                        )),
                    const SizedBox(height: 6),
                    ClipRRect(
                      borderRadius: BorderRadius.circular(4),
                      child: LinearProgressIndicator(
                        value: progress,
                        minHeight: 5,
                        backgroundColor:
                            theme.colorScheme.surfaceContainerHighest,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      '额度由系统管理员统一设置；绑定下方自己的 API Key 后对话不再消耗校内额度',
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  String _fmt(int n) {
    if (n >= 1000000) return '${(n / 1000000).toStringAsFixed(1)}M';
    if (n >= 1000) return '${(n / 1000).toStringAsFixed(1)}K';
    return '$n';
  }

  Widget _buildTabBar() {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      child: SegmentedButton<int>(
        segments: const [
          ButtonSegment(value: 0, label: Text('DeepSeek')),
          ButtonSegment(value: 1, label: Text('智谱清言')),
          ButtonSegment(value: 2, label: Text('讯飞星火')),
        ],
        selected: {_currentTab},
        onSelectionChanged: (v) {
          setState(() => _currentTab = v.first);
          _pageController.animateToPage(
            v.first,
            duration: const Duration(milliseconds: 300),
            curve: Curves.easeInOut,
          );
        },
        style: _compactSegmentedStyle,
      ),
    );
  }

  static const _compactSegmentedStyle = ButtonStyle(
    visualDensity: VisualDensity.compact,
    textStyle: WidgetStatePropertyAll(TextStyle(fontSize: 13)),
    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
  );

  Widget _buildPanel({
    required TextEditingController keyCtrl,
    required String keyHint,
    required TextEditingController modelCtrl,
    required String modelHint,
    required double temp,
    required ValueChanged<double> onTempChanged,
    required int maxTokens,
    required ValueChanged<int> onTokensChanged,
    List<_ExtraField> extraFields = const [],
  }) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _buildSectionTitle(context, 'API 密钥'),
        ...extraFields.map((f) => Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: _buildTextField(f.ctrl, f.hint, obscure: f.obscure),
            )),
        _buildTextField(keyCtrl, keyHint, obscure: true),
        const SizedBox(height: 20),
        _buildSectionTitle(context, '模型参数'),
        _buildTextField(modelCtrl, modelHint),
        const SizedBox(height: 12),
        _TempSlider(value: temp, onChanged: (v) => setState(() => onTempChanged(v))),
        const SizedBox(height: 12),
        _MaxTokensSlider(value: maxTokens, onChanged: (v) => setState(() => onTokensChanged(v))),
      ],
    );
  }

  Widget _buildBottomBar(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border(
          top: BorderSide(color: theme.colorScheme.outlineVariant.withOpacity( 0.3)),
        ),
      ),
      child: SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
              children: [
                Icon(Icons.star_outline, size: 18, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text('默认模型', style: theme.textTheme.bodyMedium),
                const Spacer(),
                DropdownButton<String>(
                  value: _defaultProvider,
                  underline: const SizedBox.shrink(),
                  isDense: true,
                  items: const [
                    DropdownMenuItem(value: 'deepseek', child: Text('DeepSeek', style: TextStyle(fontSize: 13))),
                    DropdownMenuItem(value: 'zhipu', child: Text('智谱清言', style: TextStyle(fontSize: 13))),
                    DropdownMenuItem(value: 'xunfei', child: Text('讯飞星火', style: TextStyle(fontSize: 13))),
                  ],
                  onChanged: (v) {
                    if (v != null) setState(() => _defaultProvider = v);
                  },
                ),
              ],
            ),
            const SizedBox(height: 12),
            SizedBox(
              width: double.infinity,
              height: 44,
              child: FilledButton.icon(
                onPressed: _saving ? null : _saveConfig,
                icon: _saving
                    ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
                    : const Icon(Icons.save, size: 18),
                label: Text(_saving ? '保存中...' : '保存配置'),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildSectionTitle(BuildContext context, String title) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Text(title, style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.bold)),
    );
  }

  Widget _buildTextField(TextEditingController ctrl, String hint, {bool obscure = false}) {
    return TextField(
      controller: ctrl,
      obscureText: obscure,
      decoration: InputDecoration(
        hintText: hint,
        border: const OutlineInputBorder(),
        isDense: true,
      ),
      style: const TextStyle(fontSize: 14),
    );
  }

  Future<void> _saveConfig() async {
    setState(() => _saving = true);
    try {
      final config = ModelConfig(
        deepseekKey: _dsKeyCtrl.text,
        deepseekModel: _dsModelCtrl.text,
        deepseekTemp: _dsTemp,
        deepseekMaxTokens: _dsMaxTokens,
        zhipuKey: _zpKeyCtrl.text,
        zhipuModel: _zpModelCtrl.text,
        zhipuTemp: _zpTemp,
        zhipuMaxTokens: _zpMaxTokens,
        xunfeiAppId: _xfAppIdCtrl.text,
        xunfeiKey: _xfKeyCtrl.text,
        xunfeiSecret: _xfSecretCtrl.text,
        xunfeiModel: _xfModelCtrl.text,
        xunfeiTemp: _xfTemp,
        xunfeiMaxTokens: _xfMaxTokens,
        defaultProvider: _defaultProvider,
      );
      final provider = context.read<ModelConfigProvider>();
      final ok = await provider.saveConfig(config);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(ok ? '配置已保存' : (provider.error ?? '保存失败')),
            backgroundColor: ok ? null : Theme.of(context).colorScheme.error,
          ),
        );
        if (ok) Navigator.pop(context);
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }
}

/// Panel 额外字段描述
class _ExtraField {
  final TextEditingController ctrl;
  final String hint;
  final bool obscure;
  const _ExtraField({required this.ctrl, required this.hint, this.obscure = false});
}

/// 独立的温度滑条组件 — setState 仅影响自身
class _TempSlider extends StatefulWidget {
  final double value;
  final ValueChanged<double> onChanged;
  const _TempSlider({required this.value, required this.onChanged});

  @override
  State<_TempSlider> createState() => _TempSliderState();
}

class _TempSliderState extends State<_TempSlider> {
  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('温度 Temperature: ${widget.value.toStringAsFixed(1)}', style: const TextStyle(fontSize: 13)),
        Slider(
          value: widget.value,
          min: 0,
          max: 1.5,
          divisions: 15,
          label: widget.value.toStringAsFixed(1),
          onChanged: widget.onChanged,
        ),
      ],
    );
  }
}

/// 独立的 MaxTokens 滑条组件 — setState 仅影响自身
class _MaxTokensSlider extends StatefulWidget {
  final int value;
  final ValueChanged<int> onChanged;
  const _MaxTokensSlider({required this.value, required this.onChanged});

  @override
  State<_MaxTokensSlider> createState() => _MaxTokensSliderState();
}

class _MaxTokensSliderState extends State<_MaxTokensSlider> {
  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('最大 Token 数: ${widget.value}', style: const TextStyle(fontSize: 13)),
        Slider(
          value: widget.value.toDouble(),
          min: 256,
          max: 8192,
          divisions: 31,
          label: '${widget.value}',
          onChanged: (v) => widget.onChanged(v.toInt()),
        ),
      ],
    );
  }
}
