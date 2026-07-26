// 快速重新部署 - 仅上传变更文件（vercel.json）
// 用法: node scripts/redeploy-vercel.js

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

async function uploadFile(file) {
  const result = await httpsRequest({
    hostname: 'api.vercel.com',
    path: '/v2/files?teamId=' + TEAM_ID,
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${TOKEN}`,
      'Content-Type': 'application/octet-stream',
      'x-vercel-digest': file.sha,
      'x-vercel-size': String(file.size),
      'Content-Length': String(file.buffer.length),
    },
  }, file.buffer);
  return result;
}

// 列出项目最近的部署，获取上一个部署的文件列表
async function getLatestDeployment() {
  const result = await httpsRequest({
    hostname: 'api.vercel.com',
    path: `/v6/deployments?projectId=prj_jnYd0sSw7GKji8pbWgxT6OPR55HD&teamId=${TEAM_ID}&limit=5`,
    method: 'GET',
    headers: {
      'Authorization': `Bearer ${TOKEN}`,
    },
  });
  return result;
}

// 获取单个部署的文件清单
async function getDeploymentFiles(deploymentId) {
  const result = await httpsRequest({
    hostname: 'api.vercel.com',
    path: `/v13/deployments/${deploymentId}/files?teamId=${TEAM_ID}`,
    method: 'GET',
    headers: {
      'Authorization': `Bearer ${TOKEN}`,
    },
  });
  return result;
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
  // 1. 上传变更文件（vercel.json, go.mod, go.sum）
  const changedFiles = ['vercel.json', 'go.mod', 'go.sum'];
  const uploadedFiles = {};

  for (const fileName of changedFiles) {
    console.log(`=== 上传 ${fileName} ===`);
    const filePath = path.join(ROOT, fileName);
    if (!fs.existsSync(filePath)) {
      console.log(`  文件不存在，跳过: ${fileName}`);
      continue;
    }
    const buffer = fs.readFileSync(filePath);
    const file = {
      path: fileName,
      buffer,
      sha: computeSha(buffer),
      size: buffer.length,
    };
    const uploadResult = await uploadFile(file);
    console.log(`  上传 ${fileName}: ${uploadResult.status}`);
    if (uploadResult.status >= 200 && uploadResult.status < 300) {
      uploadedFiles[fileName] = file;
    }
    // 文件之间间隔 500ms
    await new Promise(r => setTimeout(r, 500));
  }

  // 1.5 上传 SQL 迁移文件
  console.log('\n=== 上传 SQL 迁移文件 ===');
  const migrationsDir = path.join(ROOT, 'server', 'migrations');
  if (fs.existsSync(migrationsDir)) {
    const sqlFiles = fs.readdirSync(migrationsDir).filter(f => f.endsWith('.sql'));
    console.log(`找到 ${sqlFiles.length} 个 SQL 文件`);
    for (const sqlFile of sqlFiles) {
      const fullPath = path.join(migrationsDir, sqlFile);
      const buffer = fs.readFileSync(fullPath);
      const relPath = `server/migrations/${sqlFile}`;
      const file = {
        path: relPath,
        buffer,
        sha: computeSha(buffer),
        size: buffer.length,
      };
      const uploadResult = await uploadFile(file);
      if (uploadResult.status >= 200 && uploadResult.status < 300) {
        uploadedFiles[relPath] = file;
        console.log(`  ${relPath} OK`);
      } else {
        console.log(`  ${relPath} FAIL (${uploadResult.status})`);
      }
      // 避免速率限制
      await new Promise(r => setTimeout(r, 300));
    }
  }

  // 2. 获取上一个部署的文件清单
  console.log('\n=== 获取上一个部署的文件清单 ===');
  const latestResult = await getLatestDeployment();
  if (latestResult.status !== 200) {
    console.log(`获取部署列表失败: ${latestResult.status}`);
    console.log(JSON.stringify(latestResult.body, null, 2));
    return;
  }

  const deployments = latestResult.body.deployments || [];
  console.log(`找到 ${deployments.length} 个最近的部署`);
  if (deployments.length === 0) {
    console.log('没有可用的部署');
    return;
  }

  // 递归遍历文件树，提取所有文件
  function walkTree(node, parentPath = '') {
    const files = [];
    if (!node) return files;

    if (node.type === 'file') {
      files.push({
        sha: node.uid,
        size: node.size || 0,
        file: parentPath ? `${parentPath}/${node.name}` : node.name,
      });
    } else if (node.type === 'directory' && Array.isArray(node.children)) {
      const currentPath = parentPath ? `${parentPath}/${node.name}` : node.name;
      // 跳过根目录 "src"
      const basePath = node.name === 'src' ? '' : currentPath;
      for (const child of node.children) {
        files.push(...walkTree(child, basePath));
      }
    }
    return files;
  }

  // 找一个有文件的部署
  let fileSet = null;
  for (const d of deployments) {
    console.log(`检查部署 ${d.uid} (${d.state || d.readyState})`);
    const filesResult = await getDeploymentFiles(d.uid);
    if (filesResult.status === 200 && Array.isArray(filesResult.body) && filesResult.body.length > 0) {
      // 递归遍历文件树
      fileSet = [];
      for (const node of filesResult.body) {
        fileSet.push(...walkTree(node));
      }
      console.log(`  共找到 ${fileSet.length} 个文件`);
      if (fileSet.length > 0) {
        console.log(`  示例: ${JSON.stringify(fileSet[0])}`);
      }
      if (fileSet.length > 1) break;
    } else {
      console.log(`  文件数: ${Array.isArray(filesResult.body) ? filesResult.body.length : 0}`);
    }
  }

  if (!fileSet || fileSet.length === 0) {
    console.log('未找到任何部署的文件清单');
    return;
  }

  // 3. 替换变更文件
  console.log('\n=== 替换变更文件 ===');
  for (const [fileName, file] of Object.entries(uploadedFiles)) {
    const index = fileSet.findIndex(f => f.file === fileName);
    if (index >= 0) {
      fileSet[index] = {
        sha: file.sha,
        size: file.size,
        file: fileName,
      };
      console.log(`已替换 ${fileName}`);
    } else {
      fileSet.push({
        sha: file.sha,
        size: file.size,
        file: fileName,
      });
      console.log(`已添加 ${fileName}`);
    }
  }

  // 4. 创建新部署
  console.log('\n=== 创建新部署 ===');
  console.log(`使用 ${fileSet.length} 个文件`);

  const payload = {
    name: PROJECT_NAME,
    files: fileSet,
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

  const deployResult = await httpsRequest({
    hostname: 'api.vercel.com',
    path: '/v13/deployments?teamId=' + TEAM_ID,
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${TOKEN}`,
      'Content-Type': 'application/json',
    },
  }, payload);

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
