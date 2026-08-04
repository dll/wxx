import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import '../../models/models.dart';
import '../../utils/web_export.dart';

String buildProcessHtml(ProcessDefinition def, {bool forPrint = false}) {
  const esc = _escapeHtml;
  final steps = def.steps.map((s) {
    final materials = s.materialsList.isEmpty ? '无' : s.materialsList.join('、');
    final faq = s.faqList
        .map((f) => '<p><strong>${esc(f.q)}</strong><br/>${esc(f.a)}</p>')
        .join();
    return '''
<div class="step">
  <h3>${s.stepOrder}. ${esc(s.title)}</h3>
  <p><strong>截止时间：</strong>${esc(s.deadline)}</p>
  <p><strong>办理地点：</strong>${esc(s.location)}</p>
  <p><strong>所需材料：</strong>${esc(materials)}</p>
  <p><strong>联系人：</strong>${esc(s.contact)}${s.phone.isNotEmpty ? ' / ${esc(s.phone)}' : ''}</p>
  ${s.notes.isNotEmpty ? '<p><strong>说明：</strong>${esc(s.notes)}</p>' : ''}
  $faq
</div>''';
  }).join();
  final reminders = def.reminders.where((r) => r.isEnabled).map((r) {
    return '<li><strong>${esc(r.remindAt)}</strong>：${esc(r.title)}${r.content.isNotEmpty ? ' — ${esc(r.content)}' : ''}</li>';
  }).join();
  final printScript =
      forPrint ? '<script>window.onload = () => window.print();</script>' : '';
  final pngExtra = forPrint ? '' : ' width: 640px; margin: 0 auto;';
  return '''<!doctype html><html><head><meta charset="utf-8"><title>${esc(def.title)}</title>
<style>
body{font-family:"Microsoft YaHei",sans-serif;line-height:1.8;padding:32px;color:#333;$pngExtra}
h1{color:#1565C0;border-bottom:2px solid #1565C0;padding-bottom:8px}
.meta{color:#888;font-size:12px;margin-bottom:16px}
.summary{background:#f5f9ff;padding:14px;border-radius:8px}
.step{border:1px solid #e3e8f0;border-radius:8px;padding:12px 16px;margin:12px 0}
.step h3{margin:0 0 6px;color:#0d47a1}
.step p{margin:4px 0}
.reminder{background:#fff8e1;padding:12px 16px;border-radius:8px}
.footer{margin-top:24px;color:#999;font-size:12px;border-top:1px solid #eee;padding-top:8px}
</style></head><body>
<h1>${esc(def.title)}</h1>
<div class="meta">资源ID：${esc(def.resourceId)} · 状态：${esc(def.statusLabel)} · 更新：${esc(def.updatedAt)}</div>
${def.summary.isNotEmpty ? '<div class="summary">${esc(def.summary)}</div>' : ''}
${def.content.isNotEmpty ? '<h2>办理说明</h2><pre>${esc(def.content)}</pre>' : ''}
<h2>办理步骤</h2>$steps
${reminders.isNotEmpty ? '<h2>提醒节点</h2><ul class="reminder">$reminders</ul>' : ''}
<div class="footer">导出时间：${DateTime.now().toString().substring(0, 19)} · 蔚小芯 AI 学工助手</div>
$printScript
</body></html>''';
}

String buildProcessMarkdown(ProcessDefinition def) {
  final buffer = StringBuffer();
  buffer.writeln('# ${def.title}\n');
  buffer.writeln('- 资源ID：${def.resourceId}');
  buffer.writeln('- 状态：${def.statusLabel}');
  buffer.writeln('- 更新时间：${def.updatedAt}\n');
  if (def.summary.isNotEmpty) buffer.writeln('${def.summary}\n');
  if (def.content.isNotEmpty) buffer.writeln('## 办理说明\n\n${def.content}\n');
  buffer.writeln('## 办理步骤\n');
  for (final s in def.steps) {
    buffer.writeln('### ${s.stepOrder}. ${s.title}');
    if (s.deadline.isNotEmpty) buffer.writeln('- 截止时间：${s.deadline}');
    if (s.location.isNotEmpty) buffer.writeln('- 办理地点：${s.location}');
    if (s.materialsList.isNotEmpty) {
      buffer.writeln('- 所需材料：${s.materialsList.join('、')}');
    }
    if (s.contact.isNotEmpty) {
      buffer.writeln(
          '- 联系人：${s.contact}${s.phone.isNotEmpty ? ' / ${s.phone}' : ''}');
    }
    if (s.notes.isNotEmpty) buffer.writeln('- 说明：${s.notes}');
    buffer.writeln();
  }
  final enabled = def.reminders.where((r) => r.isEnabled).toList();
  if (enabled.isNotEmpty) {
    buffer.writeln('## 提醒节点\n');
    for (final r in enabled) {
      buffer.writeln(
          '- ${r.remindAt}：${r.title}${r.content.isNotEmpty ? ' — ${r.content}' : ''}');
    }
  }
  buffer.writeln('\n---\n');
  buffer.writeln('> 导出时间：${DateTime.now().toString().substring(0, 19)} · 蔚小芯');
  return buffer.toString();
}

void openProcessPrint(ProcessDefinition def) {
  openHtmlInNewTab(buildProcessHtml(def, forPrint: true));
}

Future<void> showProcessExportDialog(
    BuildContext context, ProcessDefinition def) async {
  final format = await showDialog<String>(
    context: context,
    builder: (ctx) => AlertDialog(
      title: const Text('导出办事流程'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          ListTile(
            leading: const Icon(Icons.picture_as_pdf),
            title: const Text('PDF 打印'),
            subtitle: const Text('打开浏览器打印窗口'),
            onTap: () => Navigator.pop(ctx, 'pdf'),
          ),
          ListTile(
            leading: const Icon(Icons.image),
            title: const Text('PNG 长图'),
            subtitle: const Text('打开可长按保存的网页'),
            onTap: () => Navigator.pop(ctx, 'png'),
          ),
          ListTile(
            leading: const Icon(Icons.code),
            title: const Text('Markdown'),
            subtitle: const Text('复制纯文本到剪贴板'),
            onTap: () => Navigator.pop(ctx, 'md'),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(ctx),
          child: const Text('取消'),
        ),
      ],
    ),
  );
  if (format == null || !context.mounted) return;
  switch (format) {
    case 'pdf':
      openProcessPrint(def);
      break;
    case 'png':
      openHtmlInNewTab(buildProcessHtml(def));
      break;
    case 'md':
      await Clipboard.setData(ClipboardData(text: buildProcessMarkdown(def)));
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Markdown 已复制到剪贴板')),
        );
      }
      break;
  }
}

String _escapeHtml(String input) => input
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');
