import 'dart:convert';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../models/models.dart';
import '../../providers/knowledge_provider.dart';
import '../../utils/web_export.dart';
import '../../widgets/error_view.dart';

/// 我的提交页面（student_union 及以上可访问）
class MySubmissionsPage extends StatefulWidget {
  const MySubmissionsPage({super.key});

  @override
  State<MySubmissionsPage> createState() => _MySubmissionsPageState();
}

class _MySubmissionsPageState extends State<MySubmissionsPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<KnowledgeProvider>().listResources();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('我的提交'),
        actions: [
          IconButton(
            icon: const Icon(Icons.add),
            tooltip: '创建知识资源',
            onPressed: () => _showCreateDialog(),
          ),
        ],
      ),
      body: Consumer<KnowledgeProvider>(
        builder: (_, provider, __) {
          if (provider.resourcesLoading && provider.resources.isEmpty) {
            return const Center(child: CircularProgressIndicator());
          }
          if (provider.resourceError.isNotEmpty && provider.resources.isEmpty) {
            return ErrorView.error(
              message: provider.resourceError,
              onRetry: () => provider.listResources(),
            );
          }
          if (provider.resources.isEmpty) {
            return ErrorView.empty(
              message: '暂无提交记录',
              subtitle: '点击右上角 + 创建知识资源，可上传 PDF/DOCX/XLSX 等材料解析回填',
              icon: Icons.note_add_outlined,
            );
          }
          return RefreshIndicator(
            onRefresh: () => provider.listResources(refresh: true),
            child: ListView.builder(
              padding: const EdgeInsets.all(12),
              itemCount: provider.resources.length + 1,
              itemBuilder: (context, index) {
                if (index == provider.resources.length) {
                  if (provider.resources.length < provider.resourceTotal) {
                    provider.listResources();
                    return const Padding(
                      padding: EdgeInsets.all(16),
                      child: Center(child: CircularProgressIndicator()),
                    );
                  }
                  return const SizedBox.shrink();
                }
                return _ResourceCard(
                  resource: provider.resources[index],
                  onPreview: () =>
                      _previewResource(provider, provider.resources[index]),
                  onEdit: () =>
                      _editResource(provider, provider.resources[index]),
                  onPrint: () =>
                      _printResource(provider, provider.resources[index]),
                  onSubmit: () => _handleSubmit(
                      provider, provider.resources[index].resourceId),
                );
              },
            ),
          );
        },
      ),
    );
  }

  Future<KnowledgeCard> _fullResource(
      KnowledgeProvider provider, KnowledgeCard r) async {
    if (r.content.isNotEmpty) return r;
    return await provider.getResource(r.resourceId) ?? r;
  }

  Future<void> _previewResource(
      KnowledgeProvider provider, KnowledgeCard r) async {
    final full = await _fullResource(provider, r);
    if (!mounted) return;
    showDialog(
      context: context,
      builder: (_) => AlertDialog(
        title: Text(full.title),
        content: SizedBox(
          width: 720,
          child: SingleChildScrollView(
            child: SelectableText(
                full.content.isEmpty ? full.summary : full.content),
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context), child: const Text('关闭'))
        ],
      ),
    );
  }

  Future<void> _editResource(
      KnowledgeProvider provider, KnowledgeCard r) async {
    final full = await _fullResource(provider, r);
    if (!mounted) return;
    showDialog(
        context: context,
        builder: (_) => _CreateResourceDialog(resource: full)).then((_) {
      if (mounted) provider.listResources(refresh: true);
    });
  }

  Future<void> _printResource(
      KnowledgeProvider provider, KnowledgeCard r) async {
    final full = await _fullResource(provider, r);
    openHtmlInNewTab(
        '''<!doctype html><html><head><meta charset="utf-8"><title>${_escapeHtml(full.title)}</title><style>body{font-family:"Microsoft YaHei",sans-serif;line-height:1.8;padding:32px;max-width:860px;margin:auto}pre{white-space:pre-wrap}</style></head><body><h1>${_escapeHtml(full.title)}</h1><p>${_escapeHtml(full.summary)}</p><pre>${_escapeHtml(full.content)}</pre><script>window.print()</script></body></html>''');
  }

  Future<void> _handleSubmit(
      KnowledgeProvider provider, String resourceId) async {
    final ok = await provider.submitForReview(resourceId);
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(ok ? '已提交审核，可在本页继续预览、编辑、打印' : '提交失败')),
      );
      if (ok) provider.listResources(refresh: true);
    }
  }

  void _showCreateDialog() {
    showDialog(context: context, builder: (_) => const _CreateResourceDialog())
        .then((_) {
      if (mounted)
        context.read<KnowledgeProvider>().listResources(refresh: true);
    });
  }

  static String _escapeHtml(String input) => input
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;');
}

class _ResourceCard extends StatelessWidget {
  final KnowledgeCard resource;
  final VoidCallback onPreview;
  final VoidCallback onEdit;
  final VoidCallback onPrint;
  final VoidCallback onSubmit;

  const _ResourceCard({
    required this.resource,
    required this.onPreview,
    required this.onEdit,
    required this.onPrint,
    required this.onSubmit,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final submitted =
        resource.status == 'pending' || resource.status == 'published';
    return Card(
      margin: const EdgeInsets.only(bottom: 10),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                CircleAvatar(
                  backgroundColor: theme.colorScheme.secondaryContainer,
                  child: Icon(Icons.article_outlined,
                      color: theme.colorScheme.onSecondaryContainer),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(resource.title, style: theme.textTheme.titleSmall),
                      Text(
                          '${resource.typeLabel} · ${resource.statusLabel} · ${resource.resourceId}',
                          style: theme.textTheme.bodySmall),
                    ],
                  ),
                ),
              ],
            ),
            if (resource.summary.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(resource.summary,
                  maxLines: 2, overflow: TextOverflow.ellipsis),
            ],
            const SizedBox(height: 10),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                OutlinedButton.icon(
                    onPressed: onPreview,
                    icon: const Icon(Icons.visibility_outlined, size: 16),
                    label: const Text('预览')),
                OutlinedButton.icon(
                    onPressed: onEdit,
                    icon: const Icon(Icons.edit_outlined, size: 16),
                    label: const Text('编辑')),
                OutlinedButton.icon(
                    onPressed: onPrint,
                    icon: const Icon(Icons.print_outlined, size: 16),
                    label: const Text('打印')),
                FilledButton.tonal(
                    onPressed: submitted ? null : onSubmit,
                    child: Text(submitted ? '已提交' : '提交审核')),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _CreateResourceDialog extends StatefulWidget {
  final KnowledgeCard? resource;

  const _CreateResourceDialog({this.resource});

  @override
  State<_CreateResourceDialog> createState() => _CreateResourceDialogState();
}

class _CreateResourceDialogState extends State<_CreateResourceDialog> {
  final _formKey = GlobalKey<FormState>();
  final _titleCtrl = TextEditingController();
  final _summaryCtrl = TextEditingController();
  final _contentCtrl = TextEditingController();
  final _tagsCtrl = TextEditingController();
  String _type = 'FAQ';
  String _scope = 'school';
  final String _roleScope = 'student';
  bool _saving = false;
  bool _uploading = false;

  bool get _isEdit => widget.resource != null;

  @override
  void initState() {
    super.initState();
    final r = widget.resource;
    if (r != null) {
      _titleCtrl.text = r.title;
      _summaryCtrl.text = r.summary;
      _contentCtrl.text = r.content;
      _tagsCtrl.text = r.tags.join(',');
      if (['Policy', 'Process', 'FAQ', 'Activity'].contains(r.resourceType))
        _type = r.resourceType;
    }
  }

  @override
  void dispose() {
    _titleCtrl.dispose();
    _summaryCtrl.dispose();
    _contentCtrl.dispose();
    _tagsCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(_isEdit ? '编辑知识资源' : '创建知识资源'),
      content: SizedBox(
        width: 560,
        child: Form(
          key: _formKey,
          child: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                SizedBox(
                  width: double.infinity,
                  child: OutlinedButton.icon(
                    onPressed: _uploading ? null : _handleUpload,
                    icon: _uploading
                        ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(strokeWidth: 2))
                        : const Icon(Icons.upload_file),
                    label: Text(_uploading ? '正在解析文档...' : '上传材料并解析回填'),
                  ),
                ),
                const SizedBox(height: 10),
                TextFormField(
                    controller: _titleCtrl,
                    decoration: const InputDecoration(
                        labelText: '标题 *',
                        border: OutlineInputBorder(),
                        isDense: true),
                    validator: (v) =>
                        (v == null || v.trim().isEmpty) ? '必填' : null),
                const SizedBox(height: 10),
                TextFormField(
                    controller: _summaryCtrl,
                    decoration: const InputDecoration(
                        labelText: '摘要',
                        border: OutlineInputBorder(),
                        isDense: true),
                    maxLines: 2),
                const SizedBox(height: 10),
                TextFormField(
                    controller: _contentCtrl,
                    decoration: const InputDecoration(
                        labelText: '正文 *',
                        border: OutlineInputBorder(),
                        isDense: true),
                    maxLines: 8,
                    validator: (v) =>
                        (v == null || v.trim().isEmpty) ? '必填' : null),
                const SizedBox(height: 10),
                Row(
                  children: [
                    Expanded(
                      child: DropdownButtonFormField<String>(
                        value: _type,
                        decoration: const InputDecoration(
                            labelText: '类型',
                            border: OutlineInputBorder(),
                            isDense: true),
                        items: const [
                          DropdownMenuItem(value: 'Policy', child: Text('政策')),
                          DropdownMenuItem(value: 'Process', child: Text('流程')),
                          DropdownMenuItem(value: 'FAQ', child: Text('问答')),
                          DropdownMenuItem(
                              value: 'Activity', child: Text('活动')),
                        ],
                        onChanged: (v) {
                          if (v != null) setState(() => _type = v);
                        },
                      ),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: DropdownButtonFormField<String>(
                        value: _scope,
                        decoration: const InputDecoration(
                            labelText: '范围',
                            border: OutlineInputBorder(),
                            isDense: true),
                        items: const [
                          DropdownMenuItem(value: 'school', child: Text('学校')),
                          DropdownMenuItem(value: 'college', child: Text('学院')),
                          DropdownMenuItem(value: 'class', child: Text('班级')),
                        ],
                        onChanged: (v) {
                          if (v != null) setState(() => _scope = v);
                        },
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 10),
                TextFormField(
                    controller: _tagsCtrl,
                    decoration: const InputDecoration(
                        labelText: '标签（逗号分隔）',
                        border: OutlineInputBorder(),
                        isDense: true)),
              ],
            ),
          ),
        ),
      ),
      actions: [
        TextButton(
            onPressed: () => Navigator.pop(context), child: const Text('取消')),
        FilledButton(
            onPressed: _saving ? null : _handleSave,
            child: _saving
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2))
                : Text(_isEdit ? '保存' : '创建')),
      ],
    );
  }

  Future<void> _handleSave() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _saving = true);
    final tags = _tagsCtrl.text
        .split(',')
        .map((e) => e.trim())
        .where((e) => e.isNotEmpty)
        .toList();
    final data = {
      'title': _titleCtrl.text.trim(),
      'summary': _summaryCtrl.text.trim(),
      'content': _contentCtrl.text.trim(),
      'resource_type': _type,
      'owner_scope': _scope,
      'owner_id': '',
      'role_scope': '["$_roleScope"]',
      'tags': jsonEncode(tags),
    };
    final provider = context.read<KnowledgeProvider>();
    final ok = _isEdit
        ? await provider.updateResource(widget.resource!.resourceId, data)
        : await provider.createResource(data);
    if (!mounted) return;
    if (ok) {
      Navigator.pop(context);
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(_isEdit ? '保存成功' : '创建成功')));
    } else {
      setState(() => _saving = false);
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(_isEdit ? '保存失败' : '创建失败')));
    }
  }

  Future<void> _handleUpload() async {
    final picked = await FilePicker.platform.pickFiles(
      withData: true,
      type: FileType.custom,
      allowedExtensions: const [
        'txt',
        'csv',
        'pdf',
        'docx',
        'xlsx',
        'png',
        'jpg',
        'jpeg',
        'gif',
        'bmp',
        'webp',
        'mp4',
        'avi',
        'mov',
        'mkv'
      ],
    );
    if (picked == null || picked.files.isEmpty) return;
    final file = picked.files.single;
    final bytes = file.bytes;
    if (bytes == null) {
      if (mounted)
        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('无法读取文件内容')));
      return;
    }
    setState(() => _uploading = true);
    final result = await context
        .read<KnowledgeProvider>()
        .uploadKnowledgeDocument(
            bytes: bytes, filename: file.name, resourceType: _type);
    if (!mounted) return;
    setState(() => _uploading = false);
    if (result == null) {
      final errMsg = context.read<KnowledgeProvider>().resourceError;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(errMsg.isEmpty ? '上传解析失败' : errMsg)));
      return;
    }
    _titleCtrl.text = (result['title'] ?? _titleCtrl.text).toString();
    _summaryCtrl.text =
        (result['summary'] ?? result['content_preview'] ?? _summaryCtrl.text)
            .toString();
    _contentCtrl.text =
        (result['content'] ?? result['content_preview'] ?? '').toString();
    if (_tagsCtrl.text.trim().isEmpty)
      _tagsCtrl.text = '上传文档,${file.extension ?? ''}';
    ScaffoldMessenger.of(context)
        .showSnackBar(const SnackBar(content: Text('文档已解析并回填，可继续编辑后保存')));
  }
}

extension on KnowledgeCard {
  String get statusLabel {
    const map = {
      'draft': '草稿',
      'pending': '待审核',
      'published': '已发布',
      'retired': '已下架'
    };
    return map[status] ?? (status.isEmpty ? '未知' : status);
  }
}
