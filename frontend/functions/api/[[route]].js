// 文档解析服务 - 简化版，运行在 Cloudflare Pages Functions 中
// 支持 TXT、MD、CSV 文件解析
// PDF、DOCX、XLSX 暂返回模拟数据

const SUPPORTED_FORMATS = ['txt', 'md', 'pdf', 'docx', 'csv', 'xlsx'];
const MAX_SIZE_MB = 10;

export async function onRequest(context) {
  const { request } = context;
  const url = new URL(request.url);

  if (request.method === 'OPTIONS') {
    return new Response(null, {
      status: 204,
      headers: corsHeaders(),
    });
  }

  // 路由处理
  const path = url.pathname;

  if (path === '/api/v1/documents/formats' && request.method === 'GET') {
    return handleDocumentFormats();
  }

  if (path === '/api/v1/documents/parse' && request.method === 'POST') {
    return handleDocumentParse(request);
  }

  if (path === '/api/v1/kb/formats' && request.method === 'GET') {
    return handleKBFormats();
  }

  if (path === '/api/v1/kb/upload' && request.method === 'POST') {
    return handleKBUpload(request);
  }

  // 其他请求继续代理到后端
  return proxyToBackend(request, url, context);
}

function handleDocumentFormats() {
  return jsonResponse({
    code: 0,
    message: 'success',
    data: {
      formats: SUPPORTED_FORMATS,
      max_size_mb: MAX_SIZE_MB,
      note: '支持的文档格式，解析后返回标题、摘要、关键词、字数、段落数等信息',
    },
  });
}

function handleKBFormats() {
  return jsonResponse({
    formats: [
      'txt', 'csv', 'pdf', 'docx', 'xlsx',
      'png', 'jpg', 'jpeg', 'gif', 'bmp', 'webp',
      'mp4', 'avi', 'mov', 'mkv',
    ],
    max_size_mb: 50,
    note: '上传后自动提取文本并入库至知识库',
  });
}

async function handleDocumentParse(request) {
  try {
    const formData = await request.formData();
    const file = formData.get('file');

    if (!file) {
      return jsonResponse({ code: 400, message: '请选择要上传的文件' }, 400);
    }

    const filename = file.name || 'document';
    const ext = getExtension(filename).toLowerCase();

    if (!SUPPORTED_FORMATS.includes(ext)) {
      return jsonResponse({
        code: 400,
        message: `不支持的文件类型: .${ext}。支持的格式：TXT, MD, PDF, DOCX, CSV, XLSX`,
      }, 400);
    }

    const fileSize = file.size;
    if (fileSize > MAX_SIZE_MB * 1024 * 1024) {
      return jsonResponse({
        code: 400,
        message: `文件大小超过限制：最大 ${MAX_SIZE_MB}MB`,
      }, 400);
    }

    // 解析文档
    const result = await parseDocument(file, filename, ext);

    return jsonResponse({
      code: 0,
      message: '解析成功',
      data: result,
    });
  } catch (error) {
    console.error('文档解析失败:', error);
    return jsonResponse({
      code: 500,
      message: '文档解析失败: ' + error.message,
    }, 500);
  }
}

async function handleKBUpload(request) {
  try {
    const formData = await request.formData();
    const file = formData.get('file');
    const resourceType = formData.get('resource_type') || 'FAQ';

    if (!file) {
      return jsonResponse({ error: '请选择要上传的文件' }, 400);
    }

    const filename = file.name || 'document';
    const ext = getExtension(filename).toLowerCase();

    const allowedExts = {
      txt: true, md: true, csv: true, pdf: true, docx: true,
      xlsx: true, png: true, jpg: true, jpeg: true,
      gif: true, bmp: true, webp: true,
      mp4: true, avi: true, mov: true, mkv: true,
    };

    if (!allowedExts[ext]) {
      return jsonResponse({
        error: `不支持的文件类型: .${ext}。支持的格式：TXT, MD, CSV, PDF, DOCX, XLSX, PNG, JPG, GIF, BMP, WEBP, MP4, AVI, MOV, MKV`,
      }, 400);
    }

    const fileSize = file.size;
    const isTextDoc = { txt: true, md: true, csv: true, pdf: true, docx: true, xlsx: true };

    let title = '';
    let summary = '';
    let content = '';
    let keywords = [];
    let wordCount = 0;
    let paragraphs = 0;
    let pages = 0;

    if (isTextDoc[ext]) {
      const parsed = await parseDocument(file, filename, ext);
      title = parsed.title;
      summary = parsed.summary;
      content = parsed.content;
      keywords = parsed.keywords;
      wordCount = parsed.word_count;
      paragraphs = parsed.paragraphs;
      pages = parsed.pages;
    } else {
      title = filename.replace(/\.[^/.]+$/, '');
      summary = `上传文档：${filename}（${ext.toUpperCase()}, ${bytesToSize(fileSize)}）`;
      content = '';
      pages = 0;
    }

    // 注意：Cloudflare Functions 环境中没有数据库，暂时不入库
    // 返回解析结果，in_knowledge_base 设为 false

    return jsonResponse({
      code: 0,
      message: '上传成功',
      file: filename,
      file_type: ext,
      file_size: bytesToSize(fileSize),
      title: title,
      summary: summary,
      content: content,
      keywords: keywords,
      word_count: wordCount,
      paragraphs: paragraphs,
      pages: pages,
      in_knowledge_base: false,
      resource_id: '',
      note: '当前为预览模式，文档暂未入库至知识库',
    });
  } catch (error) {
    console.error('知识上传失败:', error);
    return jsonResponse({ error: '上传失败: ' + error.message }, 500);
  }
}

async function parseDocument(file, filename, ext) {
  let content = '';
  let pages = 1;

  if (ext === 'txt' || ext === 'md' || ext === 'csv') {
    content = await file.text();
  } else if (ext === 'pdf') {
    // PDF 解析在 Cloudflare Workers 中较复杂，暂时返回模拟数据
    content = `[PDF 文档 ${filename}]\n\nPDF 文档解析功能正在优化中，暂显示文件名和基本信息。\n\n文件大小: ${bytesToSize(file.size)}\n文件类型: ${ext.toUpperCase()}`;
    pages = Math.max(1, Math.floor(file.size / 5000));
  } else if (ext === 'docx') {
    // DOCX 解析在 Cloudflare Workers 中较复杂，暂时返回模拟数据
    content = `[Word 文档 ${filename}]\n\nWord 文档解析功能正在优化中，暂显示文件名和基本信息。\n\n文件大小: ${bytesToSize(file.size)}\n文件类型: ${ext.toUpperCase()}`;
    pages = Math.max(1, Math.floor(file.size / 10000));
  } else if (ext === 'xlsx') {
    content = `[Excel 文档 ${filename}]\n\nExcel 文档解析功能正在优化中，暂显示文件名和基本信息。\n\n文件大小: ${bytesToSize(file.size)}\n文件类型: ${ext.toUpperCase()}`;
    pages = 1;
  }

  // 提取标题
  let title = extractTitle(content, filename);

  // 生成摘要
  let summary = generateSummary(content);

  // 提取关键词
  let keywords = extractKeywords(content);

  // 统计字数和段落数
  let wordCount = countWords(content);
  let paragraphCount = countParagraphs(content);

  return {
    title: title,
    summary: summary,
    content: content,
    keywords: keywords,
    word_count: wordCount,
    paragraphs: paragraphCount,
    pages: pages,
    file_name: filename,
    file_type: ext,
    file_size: file.size,
  };
}

function extractTitle(content, filename) {
  // 尝试从内容中提取标题
  const lines = content.split('\n').filter(l => l.trim());
  if (lines.length > 0) {
    const firstLine = lines[0].trim();
    if (firstLine.length > 0 && firstLine.length < 200) {
      // 移除 Markdown 标题标记
      return firstLine.replace(/^#+\s*/, '').trim();
    }
  }
  // 使用文件名作为标题
  return filename.replace(/\.[^/.]+$/, '');
}

function generateSummary(content) {
  const plainText = content.replace(/[#*_`>\[\]]/g, '').trim();
  if (plainText.length <= 200) {
    return plainText;
  }
  return plainText.substring(0, 200) + '...';
}

function extractKeywords(content) {
  // 简单的关键词提取：取出现频率最高的词
  const words = content.toLowerCase()
    .replace(/[^\u4e00-\u9fa5a-zA-Z0-9\s]/g, ' ')
    .split(/\s+/)
    .filter(w => w.length > 1);

  const stopWords = new Set(['的', '了', '是', '在', '我', '有', '和', '就', '不', '人', '都', '一', '一个', '上', '也', '很', '到', '说', '要', '去', '你', '会', '着', '没有', '看', '好', '自己', '这', 'the', 'a', 'an', 'is', 'are', 'was', 'were', 'be', 'been', 'being', 'have', 'has', 'had', 'do', 'does', 'did', 'will', 'would', 'could', 'should', 'may', 'might', 'must', 'shall', 'can', 'need', 'dare', 'ought', 'used', 'to', 'of', 'in', 'for', 'on', 'with', 'at', 'by', 'from', 'as', 'into', 'through', 'during', 'before', 'after', 'above', 'below', 'between', 'out', 'off', 'over', 'under', 'again', 'further', 'then', 'once', 'here', 'there', 'when', 'where', 'why', 'how', 'all', 'each', 'every', 'both', 'few', 'more', 'most', 'other', 'some', 'such', 'no', 'nor', 'not', 'only', 'own', 'same', 'so', 'than', 'too', 'very', 'just', 'because', 'but', 'and', 'or', 'if', 'while', 'although', 'though', 'that', 'this', 'these', 'those', 'i', 'me', 'my', 'myself', 'we', 'our', 'ours', 'ourselves', 'you', 'your', 'yours', 'yourself', 'yourselves', 'he', 'him', 'his', 'himself', 'she', 'her', 'hers', 'herself', 'it', 'its', 'itself', 'they', 'them', 'their', 'theirs', 'themselves', 'what', 'which', 'who', 'whom', 'whose']);

  const freq = {};
  for (const word of words) {
    if (!stopWords.has(word)) {
      freq[word] = (freq[word] || 0) + 1;
    }
  }

  const sorted = Object.entries(freq)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 10)
    .map(([word]) => word);

  return sorted;
}

function countWords(content) {
  // 中文字数 + 英文单词数
  const chineseChars = (content.match(/[\u4e00-\u9fa5]/g) || []).length;
  const englishWords = content.match(/[a-zA-Z]+/g) || [];
  return chineseChars + englishWords.length;
}

function countParagraphs(content) {
  const paragraphs = content.split(/\n\s*\n/).filter(p => p.trim().length > 0);
  return paragraphs.length || 1;
}

function getExtension(filename) {
  const parts = filename.split('.');
  return parts.length > 1 ? parts.pop() : '';
}

function bytesToSize(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

async function proxyToBackend(request, url, context) {
  const backendPath = url.pathname === '/api/health' ? '/health' : url.pathname;
  const targetUrl = 'https://wxx-server-j1us8ki1c-czldl.vercel.app' + backendPath + url.search;

  const headers = new Headers(request.headers);
  headers.set('Host', 'wxx-server-j1us8ki1c-czldl.vercel.app');

  let requestBody;
  if (request.method !== 'GET' && request.method !== 'HEAD') {
    requestBody = await request.arrayBuffer();
  }

  const isLogin = url.pathname === '/api/v1/auth/login';
  const shouldBuffer = isLogin ||
    url.pathname === '/api/health' ||
    url.pathname === '/api/v1/user/profile' ||
    url.pathname === '/api/v1/user/capabilities';

  const maxAttempts = shouldBuffer ? 2 : 1;
  let lastError;

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      const resp = await fetch(targetUrl, {
        method: request.method,
        headers,
        body: requestBody ? requestBody.slice(0) : undefined,
      });

      const responseBody = shouldBuffer ? await resp.arrayBuffer() : resp.body;
      const out = new Headers(resp.headers);
      applyCorsHeaders(out);

      return new Response(responseBody, {
        status: resp.status,
        headers: out,
      });
    } catch (error) {
      lastError = error;
      if (attempt < maxAttempts) {
        await new Promise((resolve) => setTimeout(resolve, 200));
      }
    }
  }

  console.error('代理后端请求失败', lastError);
  return jsonResponse({
    code: 502,
    message: '后端服务暂时不可用，请稍后重试',
  }, 502);
}

function jsonResponse(data, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: {
      ...corsHeaders(),
      'Content-Type': 'application/json; charset=utf-8',
    },
  });
}

function corsHeaders() {
  return {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET,POST,PUT,DELETE,OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type,Authorization',
  };
}

function applyCorsHeaders(headers) {
  for (const [key, value] of Object.entries(corsHeaders())) {
    headers.set(key, value);
  }
}
