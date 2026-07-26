// Vercel API 直接部署脚本 - 绕过 git 关联
// 用法: node scripts/deploy-vercel.js

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const https = require('https');

const TOKEN = process.env.VERCEL_TOKEN;
const TEAM_ID = 'team_gea581HPgTeBL8GKn46YwmO3';
const PROJECT_NAME = 'wxx-server';
const ROOT = path.resolve(__dirname, '..');

if (!TOKEN) {
  console.error('请设置 VERCEL_TOKEN 环境变量');
  process.exit(1);
}

// .vercelignore 规则
const IGNORE_DIRS = new Set([
  'frontend', 'bin', 'build', 'server/bin', 'server/build',
  'data', 'docs', 'specs', 'knowledge', 'scripts',
  '.idea', '.vscode', '.git', '.github', '.vercel',
  'node_modules', 'worker', 'miniprogram',
  // 临时和缓存目录
  'tmp', 'temp', '.tmp', '.cache', '.playwright-mcp',
  '.claude', '.trae', 'coverage', 'dist',
]);

// 仅上传后端必需的目录（白名单）
const ALLOWED_TOP_DIRS = new Set([
  'api', 'server', 'internal',
]);
const ALLOWED_ROOT_FILES = new Set([
  'go.mod', 'go.sum', 'main.go', 'Makefile',
  'vercel.json', '.vercelignore', '.env', '.gitignore',
  'server/embed.go',
]);

const IGNORE_PATTERNS = [
  /\.exe$/i, /\.apk$/i, /\.dll$/i, /\.so$/i, /\.dylib$/i,
  /\.a$/i, /\.o$/i, /\.out$/i, /\.test$/i,
  /\.db$/i, /\.sqlite$/i, /\.sqlite3$/i,
  /\.md$/i, /\.pdf$/i, /\.docx$/i, /\.xlsx$/i,
  /\.log$/i, /\.tmp$/i, /\.cache$/i,
  /^device_code\.json$/i, /^job_log.*\.txt$/i, /^response\.txt$/i,
];

const INCLUDE_EXTS = new Set([
  '.go', '.mod', '.sum', '.json', '.yaml', '.yml', '.toml',
  '.env', '.gitignore', '.vercelignore',
]);

function shouldIgnore(filePath, relativePath) {
  // 检查目录
  const parts = relativePath.split(/[\\/]/);
  for (let i = 0; i < parts.length - 1; i++) {
    if (IGNORE_DIRS.has(parts.slice(0, i + 1).join('/'))) return true;
    if (IGNORE_DIRS.has(parts[i])) return true;
  }

  // 检查文件名模式
  for (const pattern of IGNORE_PATTERNS) {
    if (pattern.test(filePath)) return true;
  }

  // 检查扩展名
  const ext = path.extname(filePath).toLowerCase();
  if (ext && !INCLUDE_EXTS.has(ext)) {
    // 允许无扩展名文件（如 .gitignore, .vercelignore）
    return true;
  }

  return false;
}

function collectFiles(dir, base = dir) {
  const files = [];
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    const relativePath = path.relative(base, fullPath).replace(/\\/g, '/');

    if (entry.isDirectory()) {
      // 白名单：仅允许指定顶级目录
      const topDir = relativePath.split('/')[0];
      if (!ALLOWED_TOP_DIRS.has(topDir)) continue;
      if (IGNORE_DIRS.has(relativePath)) continue;
      files.push(...collectFiles(fullPath, base));
    } else if (entry.isFile()) {
      // 根目录文件仅允许白名单
      if (!relativePath.includes('/')) {
        if (!ALLOWED_ROOT_FILES.has(entry.name)) continue;
      }
      if (shouldIgnore(entry.name, relativePath)) continue;
      files.push({ path: relativePath, fullPath });
    }
  }
  return files;
}

function computeSha(buffer) {
  return crypto.createHash('sha1').update(buffer).digest('hex');
}

function httpsRequest(options, body) {
  return new Promise((resolve, reject) => {
    const req = https.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => data += chunk);
      res.on('end', () => {
        try {
          const json = data ? JSON.parse(data) : {};
          resolve({ status: res.statusCode, headers: res.headers, body: json, raw: data });
        } catch (e) {
          resolve({ status: res.statusCode, headers: res.headers, body: data, raw: data });
        }
      });
    });
    req.on('error', reject);
    if (body) {
      if (Buffer.isBuffer(body)) {
        req.write(body);
      } else {
        req.write(JSON.stringify(body));
      }
    }
    req.end();
  });
}

async function checkFilesExist(files) {
  // 批量检查文件是否已上传
  const payload = {
    sha: files.map(f => f.sha),
    projectId: 'prj_jnYd0sSw7GKji8pbWgxT6OPR55HD',
    teamId: TEAM_ID,
  };

  const result = await httpsRequest({
    hostname: 'api.vercel.com',
    path: '/v2/files/ls',
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${TOKEN}`,
      'Content-Type': 'application/json',
    },
  }, payload);

  return result;
}

async function uploadFile(file) {
  const buffer = fs.readFileSync(file.fullPath);

  const result = await httpsRequest({
    hostname: 'api.vercel.com',
    path: '/v2/files?teamId=' + TEAM_ID,
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${TOKEN}`,
      'Content-Type': 'application/octet-stream',
      'x-vercel-digest': file.sha,
      'x-vercel-size': String(file.size),
      'Content-Length': String(buffer.length),
    },
  }, buffer);

  return result;
}

// 带重试的上传（429 速率限制时重试）
async function uploadFileWithRetry(file, maxRetries = 3) {
  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      const result = await uploadFile(file);
      if (result.status >= 200 && result.status < 300) {
        return result;
      }
      if (result.status === 429 && attempt < maxRetries) {
        // 速率限制，等待更长时间
        const waitMs = 3000 * (attempt + 1);
        await new Promise(r => setTimeout(r, waitMs));
        continue;
      }
      return result;
    } catch (e) {
      if (attempt === maxRetries) throw e;
      await new Promise(r => setTimeout(r, 2000));
    }
  }
}

async function createDeployment(files) {
  const payload = {
    name: PROJECT_NAME,
    files: files.map(f => ({
      sha: f.sha,
      size: f.size,
      file: f.path,
    })),
    projectSettings: {
      framework: 'go',
      buildCommand: null,
      outputDirectory: null,
      installCommand: null,
    },
    rewrites: [
      { source: '/(.*)', destination: '/api' },
    ],
    target: 'production',
  };

  const result = await httpsRequest({
    hostname: 'api.vercel.com',
    path: '/v13/deployments?teamId=' + TEAM_ID,
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${TOKEN}`,
      'Content-Type': 'application/json',
    },
  }, payload);

  return result;
}

async function main() {
  console.log('=== 收集文件 ===');
  const files = collectFiles(ROOT);
  console.log(`共 ${files.length} 个文件`);

  // 计算每个文件的 SHA 和 size
  console.log('=== 计算 SHA ===');
  for (const file of files) {
    const buffer = fs.readFileSync(file.fullPath);
    file.sha = computeSha(buffer);
    file.size = buffer.length;
  }

  // 检查哪些文件已存在
  console.log('=== 检查已上传文件 ===');
  const existResult = await checkFilesExist(files);
  const existingShas = new Set();
  if (existResult.status === 200 && Array.isArray(existResult.body)) {
    existResult.body.forEach(item => existingShas.add(item.sha || item));
  } else if (existResult.body && existResult.body.files) {
    existResult.body.files.forEach(item => existingShas.add(item.sha || item));
  }
  console.log(`已上传: ${existingShas.size}/${files.length}`);

  // 上传缺失的文件
  const toUpload = files.filter(f => !existingShas.has(f.sha));
  console.log(`=== 上传 ${toUpload.length} 个文件 ===`);
  let failedCount = 0;
  for (let i = 0; i < toUpload.length; i++) {
    const file = toUpload[i];
    process.stdout.write(`[${i + 1}/${toUpload.length}] ${file.path} ... `);
    try {
      const result = await uploadFileWithRetry(file);
      if (result.status >= 200 && result.status < 300) {
        console.log('OK');
      } else {
        console.log(`FAIL (${result.status}): ${JSON.stringify(result.body).substring(0, 100)}`);
        failedCount++;
      }
    } catch (e) {
      console.log(`ERROR: ${e.message}`);
      failedCount++;
    }
    // 每个文件之间间隔 200ms，避免触发速率限制
    await new Promise(r => setTimeout(r, 200));
  }
  console.log(`上传完成: 失败 ${failedCount} 个`);

  // 创建部署
  console.log('=== 创建部署 ===');
  const deployResult = await createDeployment(files);
  if (deployResult.status >= 200 && deployResult.status < 300) {
    console.log('部署创建成功:');
    console.log(`  ID: ${deployResult.body.id}`);
    console.log(`  URL: https://${deployResult.body.url}`);
    console.log(`  Inspect: https://vercel.com/czldl/wxx-server/${deployResult.body.id}`);
    console.log(`  State: ${deployResult.body.readyState || deployResult.body.status}`);
  } else {
    console.log(`部署创建失败 (${deployResult.status}):`);
    console.log(JSON.stringify(deployResult.body, null, 2));
  }
}

main().catch(e => {
  console.error('脚本失败:', e);
  process.exit(1);
});
