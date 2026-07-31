import 'dart:typed_data';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../models/models.dart';
import '../../providers/admin_provider.dart';
import '../../utils/capability_utils.dart';

/// 学生 Excel 导入对话框。
class ImportStudentDialog extends StatefulWidget {
  const ImportStudentDialog({super.key});

  @override
  State<ImportStudentDialog> createState() => _ImportStudentDialogState();
}

class _ImportStudentDialogState extends State<ImportStudentDialog> {
  static const _maxFileSize = 100 * 1024 * 1024;

  final _passwordController = TextEditingController();
  Uint8List? _fileBytes;
  String? _filename;
  bool _obscurePassword = true;
  ImportResultData? _result;

  @override
  void dispose() {
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _pickFile() async {
    // 选择器初始化失败时（如 Web 端插件未注册）会同步抛错，
    // 必须捕获后给出提示，否则用户只会看到「点击无反应」。
    FilePickerResult? picked;
    try {
      picked = await FilePicker.platform.pickFiles(
        type: FileType.custom,
        allowedExtensions: const ['xlsx'],
        allowMultiple: false,
        withData: true,
      );
    } catch (e) {
      if (mounted) _showMessage('无法打开文件选择器：$e');
      return;
    }
    if (!mounted || picked == null || picked.files.isEmpty) return;

    final file = picked.files.single;
    if (file.size <= 0 || file.size > _maxFileSize) {
      _showMessage('文件大小必须在 100MB 以内');
      return;
    }
    if (file.bytes == null) {
      _showMessage('无法读取文件，请重新选择');
      return;
    }

    setState(() {
      _filename = file.name;
      _fileBytes = file.bytes;
      _result = null;
    });
  }

  Future<void> _import() async {
    final bytes = _fileBytes;
    final filename = _filename;
    if (bytes == null || filename == null) {
      _showMessage('请先选择 .xlsx 文件');
      return;
    }

    final password = _passwordController.text.trim();
    if (password.isNotEmpty && password.characters.length < 6) {
      _showMessage('统一初始密码不能少于 6 位');
      return;
    }

    final provider = context.read<AdminProvider>();
    final ok = await provider.importStudents(
      bytes,
      filename,
      defaultPassword: password.isEmpty ? null : password,
    );
    if (!mounted) return;

    if (!ok || provider.importResult == null) {
      _showMessage(provider.error.isEmpty ? '导入失败' : provider.error);
      return;
    }

    if (CapabilityUtils.has(Capability.collegeUserRead)) {
      await provider.fetchUsers(refresh: true);
      if (!mounted) return;
    }
    setState(() => _result = provider.importResult);
  }

  void _reset() {
    context.read<AdminProvider>().clearImportResult();
    setState(() {
      _fileBytes = null;
      _filename = null;
      _result = null;
      _passwordController.clear();
    });
  }

  void _showMessage(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(message)),
    );
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<AdminProvider>();
    return AlertDialog(
      title: const Text('导入学生用户'),
      content: SizedBox(
        width: 560,
        child: _result == null
            ? _buildForm(context, provider.importing)
            : _buildResult(context, _result!),
      ),
      actions: [
        if (_result != null)
          TextButton(onPressed: _reset, child: const Text('继续导入')),
        TextButton(
          onPressed: provider.importing ? null : () => Navigator.pop(context),
          child: const Text('关闭'),
        ),
      ],
    );
  }

  Widget _buildForm(BuildContext context, bool importing) {
    final theme = Theme.of(context);
    return SingleChildScrollView(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text('Excel 格式', style: theme.textTheme.titleSmall),
          const SizedBox(height: 8),
          Text(
            '支持 .xlsx，必填列为“学号、姓名”；可选列为“院系、专业、班级、入学时间、入学年份、角色”。角色为空或“学生”。',
            style: theme.textTheme.bodyMedium,
          ),
          const SizedBox(height: 16),
          OutlinedButton.icon(
            onPressed: importing ? null : _pickFile,
            icon: const Icon(Icons.upload_file),
            label: Text(_filename ?? '选择 Excel 文件'),
          ),
          if (_filename != null) ...[
            const SizedBox(height: 8),
            Text(
              '已选择：$_filename',
              overflow: TextOverflow.ellipsis,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.primary,
              ),
            ),
          ],
          const SizedBox(height: 16),
          TextField(
            controller: _passwordController,
            obscureText: _obscurePassword,
            enabled: !importing,
            decoration: InputDecoration(
              labelText: '统一初始密码（可选）',
              helperText: '留空时，每名学生的初始密码为其学号',
              border: const OutlineInputBorder(),
              suffixIcon: IconButton(
                onPressed: () => setState(
                  () => _obscurePassword = !_obscurePassword,
                ),
                icon: Icon(
                  _obscurePassword ? Icons.visibility_off : Icons.visibility,
                ),
              ),
            ),
            textInputAction: TextInputAction.done,
            onSubmitted: importing ? null : (_) => _import(),
          ),
          const SizedBox(height: 16),
          SizedBox(
            height: 48,
            child: FilledButton.icon(
              onPressed: importing ? null : _import,
              icon: importing
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.cloud_upload_outlined),
              label: Text(importing ? '正在导入...' : '开始导入'),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildResult(BuildContext context, ImportResultData result) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            Icon(
              result.failed == 0 ? Icons.check_circle : Icons.warning_amber,
              color:
                  result.failed == 0 ? colorScheme.primary : colorScheme.error,
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                '共 ${result.total} 条：成功 ${result.success} 条，失败 ${result.failed} 条',
                style: theme.textTheme.titleMedium,
              ),
            ),
          ],
        ),
        if (result.details.isNotEmpty) ...[
          const SizedBox(height: 16),
          ConstrainedBox(
            constraints: const BoxConstraints(maxHeight: 260),
            child: ListView.separated(
              shrinkWrap: true,
              itemCount: result.details.length,
              separatorBuilder: (_, __) => const Divider(height: 1),
              itemBuilder: (_, index) {
                final detail = result.details[index];
                return ListTile(
                  dense: true,
                  leading: Icon(
                    detail.success ? Icons.check : Icons.close,
                    color: detail.success
                        ? colorScheme.primary
                        : colorScheme.error,
                  ),
                  title: Text('${detail.username} ${detail.displayName}'),
                  subtitle: detail.success ? null : Text(detail.error),
                );
              },
            ),
          ),
        ],
      ],
    );
  }
}
