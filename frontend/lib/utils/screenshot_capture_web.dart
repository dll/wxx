import 'dart:async';
import 'dart:html' as html;
import 'dart:typed_data';

import 'package:flutter/foundation.dart';

import 'screenshot_capture.dart';

/// Web 端截图：直接从 Flutter 渲染的 <flutter-view> / <canvas> 元素抓取像素
///
/// CanvasKit 渲染会把整个 Flutter 视图渲染到 <flt-glass-pane> 的 ShadowRoot 内的 <canvas>。
/// 必须穿透 ShadowRoot 才能拿到 canvas。
/// 这条路径绕开了 RenderRepaintBoundary.toImage 的兼容问题。
Future<ScreenshotResult> captureWebScreenshot() async {
  try {
    // 让 Flutter 完成当前帧渲染（确保 canvas 内容是最新的）
    await Future<void>.delayed(const Duration(milliseconds: 30));

    // 1. 收集所有 canvas（含 ShadowRoot 内）
    final canvases = _collectAllCanvases();
    if (canvases.isEmpty) {
      return const ScreenshotResult(error: '未找到画布元素');
    }
    return await _captureFromCanvases(canvases);
  } catch (e) {
    if (kDebugMode) debugPrint('captureWebScreenshot 失败: $e');
    return ScreenshotResult(error: '截图失败：$e');
  }
}

/// 递归遍历 DOM（包括 ShadowRoot），收集所有 canvas 元素
List<html.CanvasElement> _collectAllCanvases() {
  final result = <html.CanvasElement>[];
  void walk(html.Node node) {
    if (node is html.CanvasElement) {
      result.add(node);
    }
    // 检查元素的 shadowRoot
    if (node is html.Element) {
      final shadow = node.shadowRoot;
      if (shadow != null) {
        for (final child in shadow.childNodes) {
          walk(child);
        }
      }
    }
    for (final child in node.childNodes) {
      walk(child);
    }
  }
  final body = html.document.body;
  if (body != null) walk(body);
  return result;
}

/// 从找到的 canvas 集合中选择最大的可见 canvas，并把它转为 PNG bytes
Future<ScreenshotResult> _captureFromCanvases(List<html.CanvasElement> canvases) async {
  html.CanvasElement? target;
  int maxArea = 0;

  for (final node in canvases) {
    final rect = node.getBoundingClientRect();
    final area = (rect.width * rect.height).toInt();
    if (area > maxArea) {
      maxArea = area;
      target = node;
    }
  }

  if (target == null || maxArea == 0) {
    return const ScreenshotResult(error: '画布尺寸为 0');
  }

  // 优先 toDataUrl（同步，最稳定），失败再走 toBlob
  try {
    final dataUrl = target.toDataUrl('image/png');
    final commaIdx = dataUrl.indexOf(',');
    if (commaIdx <= 0) {
      return const ScreenshotResult(error: '画布导出格式异常');
    }
    final base64 = dataUrl.substring(commaIdx + 1);
    final bytes = _base64ToBytes(base64);
    if (bytes.isEmpty) {
      return const ScreenshotResult(error: '画布导出为空');
    }
    return ScreenshotResult(bytes: bytes);
  } catch (e1) {
    if (kDebugMode) debugPrint('toDataUrl 失败，回退 toBlob: $e1');
    return _captureViaToBlob(target);
  }
}

/// toBlob + FileReader 兜底路径
Future<ScreenshotResult> _captureViaToBlob(html.CanvasElement canvas) async {
  final completer = Completer<ScreenshotResult>();
  try {
    canvas.toBlob().then((blob) {
      final reader = html.FileReader();
      reader.onLoadEnd.listen((_) {
        try {
          final result = reader.result;
          if (result is Uint8List) {
            completer.complete(ScreenshotResult(bytes: result));
          } else if (result is ByteBuffer) {
            completer.complete(ScreenshotResult(bytes: result.asUint8List()));
          } else {
            completer.complete(const ScreenshotResult(error: '截图编码异常'));
          }
        } catch (e) {
          completer.complete(ScreenshotResult(error: '读取截图失败: $e'));
        }
      });
      reader.onError.listen((_) {
        completer.complete(const ScreenshotResult(error: '读取截图失败'));
      });
      reader.readAsArrayBuffer(blob);
    }).catchError((e) {
      completer.complete(ScreenshotResult(error: '画布导出失败：$e'));
    });
  } catch (e) {
    completer.complete(ScreenshotResult(error: '画布导出异常：$e'));
  }
  return completer.future.timeout(
    const Duration(seconds: 5),
    onTimeout: () => const ScreenshotResult(error: '截图超时'),
  );
}

/// base64 → Uint8List
Uint8List _base64ToBytes(String base64) {
  final binary = html.window.atob(base64);
  final bytes = Uint8List(binary.length);
  for (var i = 0; i < binary.length; i++) {
    bytes[i] = binary.codeUnitAt(i);
  }
  return bytes;
}
