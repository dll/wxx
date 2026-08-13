import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../../providers/education_provider.dart';
import '../../../widgets/error_view.dart';

class CounselingPage extends StatefulWidget {
  const CounselingPage({super.key});

  @override
  State<CounselingPage> createState() => _CounselingPageState();
}

class _CounselingPageState extends State<CounselingPage> {
  final _formKey = GlobalKey<FormState>();
  String? _selectedCounselor;
  String? _selectedDate;
  String? _selectedTime;
  String? _reason;
  String? _description;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<MentalProvider>().fetchCounselors();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<MentalProvider>();

    return Scaffold(
      appBar: AppBar(title: const Text('预约心理咨询')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Form(
          key: _formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _buildCounselorSelection(theme, provider),
              const SizedBox(height: 20),
              Text('预约时间', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
              const SizedBox(height: 12),
              _buildDateTimeSelection(theme),
              const SizedBox(height: 20),
              Text('咨询信息', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
              const SizedBox(height: 12),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    children: [
                      DropdownButtonFormField<String>(
                        value: _reason,
                        decoration: const InputDecoration(
                          labelText: '咨询类型',
                          border: OutlineInputBorder(),
                        ),
                        items: const [
                          DropdownMenuItem(value: '学业压力', child: Text('学业压力')),
                          DropdownMenuItem(value: '人际关系', child: Text('人际关系')),
                          DropdownMenuItem(value: '情感问题', child: Text('情感问题')),
                          DropdownMenuItem(value: '就业焦虑', child: Text('就业焦虑')),
                          DropdownMenuItem(value: '家庭问题', child: Text('家庭问题')),
                          DropdownMenuItem(value: '自我认知', child: Text('自我认知')),
                          DropdownMenuItem(value: '其他', child: Text('其他')),
                        ],
                        onChanged: (v) => setState(() => _reason = v),
                        validator: (v) => v == null ? '请选择咨询类型' : null,
                      ),
                      const SizedBox(height: 16),
                      TextFormField(
                        maxLines: 4,
                        decoration: const InputDecoration(
                          labelText: '问题描述',
                          hintText: '请简要描述您想咨询的问题...',
                          border: OutlineInputBorder(),
                        ),
                        onChanged: (v) => _description = v,
                      ),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 24),
              SizedBox(
                width: double.infinity,
                child: FilledButton.icon(
                  onPressed: provider.loading ? null : _submitAppointment,
                  icon: const Icon(Icons.check, size: 18),
                  label: Text(provider.loading ? '提交中...' : '提交预约'),
                  style: FilledButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 14),
                  ),
                ),
              ),
              const SizedBox(height: 16),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: theme.colorScheme.primaryContainer.withOpacity(0.3),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Icon(Icons.info_outline, size: 20, color: theme.colorScheme.primary),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        '您的咨询内容将严格保密，请放心倾诉。如遇紧急情况，请拨打心理援助热线。',
                        style: theme.textTheme.bodySmall,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildCounselorSelection(ThemeData theme, MentalProvider provider) {
    if (provider.loading && provider.counselors.isEmpty) {
      return const Center(child: Padding(padding: EdgeInsets.all(32), child: CircularProgressIndicator()));
    }
    if (provider.error.isNotEmpty && provider.counselors.isEmpty) {
      return ErrorView.error(
        message: provider.error,
        onRetry: () => provider.fetchCounselors(),
      );
    }
    if (provider.counselors.isEmpty) {
      return ErrorView.empty(
        message: '暂无可用咨询师',
        icon: Icons.person_outline,
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('选择咨询师', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
        const SizedBox(height: 12),
        ...provider.counselors.map((c) {
          if (c is! Map) return const SizedBox.shrink();
          final counselor = Map<String, dynamic>.from(c);
          final id = counselor['id']?.toString() ?? '';
          final name = counselor['name'] as String? ?? '';
          final title = counselor['title'] as String? ?? '';
          final specialty = counselor['specialty'] as String? ?? '';
          final avatar = counselor['avatar'] as String? ?? '';
          final isSelected = _selectedCounselor == id;

          return Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: Material(
              color: isSelected
                  ? theme.colorScheme.primaryContainer
                  : theme.colorScheme.surface,
              borderRadius: BorderRadius.circular(12),
              child: InkWell(
                onTap: () {
                  setState(() {
                    _selectedCounselor = id;
                  });
                },
                borderRadius: BorderRadius.circular(12),
                child: Container(
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(
                      color: isSelected ? theme.colorScheme.primary : theme.colorScheme.outlineVariant,
                      width: isSelected ? 2 : 1,
                    ),
                  ),
                  padding: const EdgeInsets.all(16),
                  child: Row(
                    children: [
                      CircleAvatar(
                        radius: 24,
                        backgroundImage: avatar.isNotEmpty ? NetworkImage(avatar) : null,
                        child: avatar.isEmpty
                            ? Text(name.isNotEmpty ? name[0] : '?', style: const TextStyle(fontSize: 18))
                            : null,
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              name,
                              style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w600),
                            ),
                            if (title.isNotEmpty)
                              Text(title, style: theme.textTheme.bodySmall),
                            if (specialty.isNotEmpty)
                              Text(
                                '擅长：$specialty',
                                style: theme.textTheme.bodySmall?.copyWith(
                                  color: theme.colorScheme.onSurfaceVariant,
                                ),
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                              ),
                          ],
                        ),
                      ),
                      if (isSelected)
                        Icon(Icons.check_circle, color: theme.colorScheme.primary),
                    ],
                  ),
                ),
              ),
            ),
          );
        }),
      ],
    );
  }

  Widget _buildDateTimeSelection(ThemeData theme) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          children: [
            InkWell(
              onTap: () async {
                final date = await showDatePicker(
                  context: context,
                  initialDate: DateTime.now().add(const Duration(days: 1)),
                  firstDate: DateTime.now().add(const Duration(days: 1)),
                  lastDate: DateTime.now().add(const Duration(days: 30)),
                );
                if (date != null) {
                  setState(() {
                    _selectedDate = '${date.year}-${date.month.toString().padLeft(2, '0')}-${date.day.toString().padLeft(2, '0')}';
                  });
                }
              },
              child: Container(
                padding: const EdgeInsets.symmetric(vertical: 12),
                child: Row(
                  children: [
                    Icon(Icons.calendar_today_outlined, color: theme.colorScheme.primary),
                    const SizedBox(width: 12),
                    Text(
                      _selectedDate ?? '选择日期',
                      style: TextStyle(
                        color: _selectedDate == null ? theme.colorScheme.onSurfaceVariant : theme.colorScheme.onSurface,
                      ),
                    ),
                    const Spacer(),
                    Icon(Icons.chevron_right, color: theme.colorScheme.onSurfaceVariant),
                  ],
                ),
              ),
            ),
            const Divider(height: 1),
            InkWell(
              onTap: () async {
                final time = await showTimePicker(
                  context: context,
                  initialTime: const TimeOfDay(hour: 9, minute: 0),
                );
                if (time != null) {
                  setState(() {
                    _selectedTime = '${time.hour.toString().padLeft(2, '0')}:${time.minute.toString().padLeft(2, '0')}';
                  });
                }
              },
              child: Container(
                padding: const EdgeInsets.symmetric(vertical: 12),
                child: Row(
                  children: [
                    Icon(Icons.access_time, color: theme.colorScheme.primary),
                    const SizedBox(width: 12),
                    Text(
                      _selectedTime ?? '选择时间',
                      style: TextStyle(
                        color: _selectedTime == null ? theme.colorScheme.onSurfaceVariant : theme.colorScheme.onSurface,
                      ),
                    ),
                    const Spacer(),
                    Icon(Icons.chevron_right, color: theme.colorScheme.onSurfaceVariant),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _submitAppointment() async {
    if (_selectedCounselor == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请选择咨询师')),
      );
      return;
    }
    if (_selectedDate == null || _selectedTime == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请选择预约时间')),
      );
      return;
    }
    if (!_formKey.currentState!.validate()) return;

    final provider = context.read<MentalProvider>();
    final success = await provider.submitAppointment({
      'counselor_id': _selectedCounselor,
      'date': _selectedDate,
      'time': _selectedTime,
      'reason': _reason,
      'description': _description,
    });

    if (mounted) {
      if (success) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('预约提交成功，请等待确认')),
        );
        Navigator.of(context).pop();
      } else {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('提交失败：${provider.error}')),
        );
      }
    }
  }
}
