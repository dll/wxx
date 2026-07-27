// deploy-pages.mjs - 通过 Cloudflare API 直接部署到 Pages
import { readdir, readFile, stat } from 'fs/promises';
import { join, relative } from 'path';
import { createHash } from 'crypto';

const CF_API_TOKEN = process.env.CLOUDFLARE_API_TOKEN;
const CF_ACCOUNT_ID = process.env.CLOUDFLARE_ACCOUNT_ID;
const PROJECT_NAME = 'wxx-agent';
const BUILD_DIR = './frontend/build/web';

if (!CF_API_TOKEN || !CF_ACCOUNT_ID) {
  console.error('缺少 CLOUDFLARE_API_TOKEN 或 CLOUDFLARE_ACCOUNT_ID 环境变量');
  process.exit(1);
}

async function listFiles(dir, base = dir) {
  const entries = await readdir(dir, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const fullPath = join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...await listFiles(fullPath, base));
    } else {
      const relPath = relative(base, fullPath).replace(/\\/g, '/');
      const content = await readFile(fullPath);
      const hash = createHash('sha256').update(content).digest('hex');
      files.push({
        path: relPath,
        content,
        hash,
        size: content.length,
      });
    }
  }
  return files;
}

async function deploy() {
  console.log('正在收集部署文件...');
  const files = await listFiles(BUILD_DIR);
  console.log(`共 ${files.length} 个文件`);

  // 创建部署
  console.log('创建部署...');
  const createRes = await fetch(
    `https://api.cloudflare.com/client/v4/accounts/${CF_ACCOUNT_ID}/pages/projects/${PROJECT_NAME}/deployments`,
    {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${CF_API_TOKEN}`,
        'Content-Type': 'application/json',
      },
    }
  );
  const createData = await createRes.json();
  if (!createData.success) {
    console.error('创建部署失败:', JSON.stringify(createData.errors, null, 2));
    process.exit(1);
  }
  const deploymentId = createData.result.id;
  console.log('部署ID:', deploymentId);

  // 上传文件清单
  const manifest = files.map(f => ({
    path: f.path,
    sha256: f.hash,
    size: f.size,
  }));

  console.log('上传文件清单...');
  const uploadRes = await fetch(
    `https://api.cloudflare.com/client/v4/accounts/${CF_ACCOUNT_ID}/pages/projects/${PROJECT_NAME}/deployments/${deploymentId}/files/upload`,
    {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${CF_API_TOKEN}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(manifest),
    }
  );
  const uploadData = await uploadRes.json();
  if (!uploadData.success) {
    console.error('上传清单失败:', JSON.stringify(uploadData.errors, null, 2));
    process.exit(1);
  }

  // 上传每个文件
  for (const file of files) {
    if (file.size === 0) continue;
    const url = `https://api.cloudflare.com/client/v4/accounts/${CF_ACCOUNT_ID}/pages/projects/${PROJECT_NAME}/deployments/${deploymentId}/files${file.path}`;
    const res = await fetch(url, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${CF_API_TOKEN}`,
        'Content-Type': 'application/octet-stream',
      },
      body: file.content,
    });
    if (!res.ok) {
      console.error(`上传失败: ${file.path}`);
    }
  }

  // 完成部署
  console.log('完成部署...');
  const finalRes = await fetch(
    `https://api.cloudflare.com/client/v4/accounts/${CF_ACCOUNT_ID}/pages/projects/${PROJECT_NAME}/deployments/${deploymentId}`,
    {
      method: 'PATCH',
      headers: {
        'Authorization': `Bearer ${CF_API_TOKEN}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ deployment_trigger: { type: 'api' } }),
    }
  );
  const finalData = await finalRes.json();
  console.log('部署结果:', JSON.stringify(finalData, null, 2));
}

deploy().catch(err => {
  console.error('部署出错:', err);
  process.exit(1);
});
