import 'dart:convert';

import '../models/models.dart';

/// 反馈结构化报告生成（统一复用，保证不丢字段）。
///
/// 供三处使用：管理端详情「复制完整反馈」、在线修复面板「复制完整报告」、
/// 用户端「我的反馈」复制。生成两种格式：
/// - `json`：反馈全字段结构化 JSON，便于 AI 工具精确解析。
/// - `markdown`：人类可读的报告（含 AI 诊断信息，若有）。
class FeedbackReport {
  FeedbackReport._();

  /// 生成完整 JSON 文本（含日志/诊断可选注入）
  static String buildJson(FeedbackEntry fb,
      {List<FeedbackLog> logs = const [], FeedbackReportExtra? extra}) {
    final map = <String, dynamic>{
      'id': fb.feedbackId,
      'user_id': fb.userId,
      'username': fb.username,
      'category': fb.category,
      'category_label': fb.categoryLabel,
      'module': fb.module,
      'content': fb.content,
      'status': fb.status,
      'status_label': fb.statusLabel,
      'message_id': fb.messageId,
      'resource_id': fb.resourceId,
      'screenshot_url': fb.screenshotUrl,
      'created_at': fb.createdAt,
      'resolved_by': fb.resolvedBy,
      'resolved_at': fb.resolvedAt,
      'reply': fb.reply,
      'rating': fb.rating,
      'rating_comment': fb.ratingComment,
      'logs': logs
          .map((l) => {
                'action': l.actionLabel,
                'operator': l.operator,
                'detail': l.detail,
                'created_at': l.createdAt,
              })
          .toList(),
    };
    if (extra != null) {
      if (extra.summary.isNotEmpty) map['ai_summary'] = extra.summary;
      if (extra.ocrText.isNotEmpty) map['ai_ocr_text'] = extra.ocrText;
      if (extra.rootCause.isNotEmpty) map['ai_root_cause'] = extra.rootCause;
      if (extra.repairHint.isNotEmpty) map['ai_repair_hint'] = extra.repairHint;
      if (extra.module.isNotEmpty) map['ai_module'] = extra.module;
      if (extra.codeFiles.isNotEmpty) map['code_files'] = extra.codeFiles;
      if (extra.matchedFiles.isNotEmpty) {
        map['ai_matched_files'] = extra.matchedFiles;
      }
    }
    return const JsonEncoder.withIndent('  ').convert(map);
  }

  /// 生成 Markdown 报告（含日志与诊断）
  static String buildMarkdown(FeedbackEntry fb,
      {List<FeedbackLog> logs = const [], FeedbackReportExtra? extra}) {
    final sb = StringBuffer()
      ..writeln('# 反馈问题诊断报告')
      ..writeln()
      ..writeln('## 反馈信息')
      ..writeln('- ID: ${fb.feedbackId}')
      ..writeln('- 用户: ${fb.username}（${fb.createdAt}）')
      ..writeln(
          '- 类型: ${fb.categoryLabel} | 模块: ${fb.module.isNotEmpty ? fb.module : '未指定'}')
      ..writeln(
          '- 状态: ${fb.statusLabel}${fb.resolvedBy.isNotEmpty ? ' | 处理人: ${fb.resolvedBy}' : ''}')
      ..writeln('- 消息ID: ${fb.messageId}')
      ..writeln('- 关联资源: ${fb.resourceId.isEmpty ? '无' : fb.resourceId}')
      ..writeln('- 截图: ${fb.screenshotUrl.isEmpty ? '无' : fb.screenshotUrl}')
      ..writeln()
      ..writeln('## 反馈内容')
      ..writeln()
      ..writeln(fb.content);
    if (fb.reply.isNotEmpty) {
      sb
        ..writeln()
        ..writeln('## 处理回复')
        ..writeln()
        ..writeln(fb.reply);
    }
    if (fb.rating > 0) {
      sb
        ..writeln()
        ..writeln('## 满意度')
        ..writeln()
        ..writeln(
            '${fb.rating} 星${fb.ratingComment.isNotEmpty ? ' | ${fb.ratingComment}' : ''}');
    }
    if (extra != null) {
      if (extra.summary.isNotEmpty) {
        sb
          ..writeln()
          ..writeln('## 问题摘要')
          ..writeln()
          ..writeln(extra.summary);
      }
      if (extra.ocrText.trim().isNotEmpty) {
        sb
          ..writeln()
          ..writeln('## 截图解析（OCR）')
          ..writeln()
          ..writeln(extra.ocrText);
      }
      if (extra.rootCause.isNotEmpty) {
        sb
          ..writeln()
          ..writeln('## 根因分析')
          ..writeln()
          ..writeln(extra.rootCause);
      }
      if (extra.module.isNotEmpty || fb.module.isNotEmpty) {
        sb
          ..writeln()
          ..writeln('## 相关模块')
          ..writeln()
          ..writeln('- 反馈模块: ${fb.module.isNotEmpty ? fb.module : '未指定'}')
          ..writeln(
              '- AI 匹配: ${extra.module.isNotEmpty ? extra.module : '（未匹配）'}');
      }
      final files = [
        ...extra.codeFiles,
        ...extra.matchedFiles.where((f) => !extra.codeFiles.contains(f)),
      ];
      if (files.isNotEmpty) {
        sb
          ..writeln()
          ..writeln('## 相关代码文件')
          ..writeln();
        for (final f in files) {
          sb.writeln('- `$f`');
        }
      }
      if (extra.repairHint.isNotEmpty) {
        sb
          ..writeln()
          ..writeln('## 修复建议')
          ..writeln()
          ..writeln(extra.repairHint);
      }
    }
    if (logs.isNotEmpty) {
      sb
        ..writeln()
        ..writeln('## 处理记录')
        ..writeln();
      for (final log in logs) {
        sb.writeln('- [${log.createdAt}] ${log.actionLabel}'
            '${log.operator.isNotEmpty ? ' by ${log.operator}' : ''}'
            '${log.detail.isNotEmpty ? '（${log.detail}）' : ''}');
      }
    }
    return sb.toString();
  }
}

/// 反馈报告附带的 AI 诊断信息（可选）
class FeedbackReportExtra {
  final String module;
  final String summary;
  final String ocrText;
  final String rootCause;
  final String repairHint;
  final List<String> codeFiles;
  final List<String> matchedFiles;

  const FeedbackReportExtra({
    this.module = '',
    this.summary = '',
    this.ocrText = '',
    this.rootCause = '',
    this.repairHint = '',
    this.codeFiles = const [],
    this.matchedFiles = const [],
  });
}
