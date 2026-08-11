import 'dart:typed_data';

import 'package:file_picker/file_picker.dart';

/// 选择一张图片，返回字节与 MIME。
/// 跨平台（Web / Android / iOS），通过 file_picker 实现。
Future<({Uint8List bytes, String mime})?> pickPortraitPhoto() async {
  final result = await FilePicker.platform.pickFiles(
    type: FileType.image,
    withData: true,
  );
  if (result == null || result.files.isEmpty) return null;
  final file = result.files.first;
  if (file.bytes == null) return null;
  final mime = _mimeFor(file.extension ?? '');
  return (bytes: file.bytes!, mime: mime);
}

String _mimeFor(String ext) {
  switch (ext.toLowerCase()) {
    case 'png':
      return 'image/png';
    case 'gif':
      return 'image/gif';
    case 'webp':
      return 'image/webp';
    case 'bmp':
      return 'image/bmp';
    default:
      return 'image/jpeg';
  }
}
