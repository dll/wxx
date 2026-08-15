# 云端部署调通验证（2026-08-15）

## 结论
云端 `https://wxx-agent.online` 已成功跑上**含全部新功能**的后端版本（外部活动导入 + 生命周期管理 + 签到 + 复盘指标），登录与新接口均验证通过。

## 根因定位
之前"云端无活动管理入口 + 登录 401"的真正根因有两个，逐一确认并解决：

### 1. 后端 CI 部署的二进制路径错误（scp 挂起之外的第二个坑）
- `docker/deploy-frontend.yml` 的 `deploy-server` job 把编译产物写到 `/opt/wxx/bin/wxx-server`
- 但 **systemd 服务 `wxx` 的 ExecStart 指向 `/opt/wxx/wxx-server`**（旧二进制 10:03，45MB）
- 结果：CI 显示 success，但服务重启后仍跑**旧二进制**，新接口（review-stats 等）404

### 2. 服务器访问 GitHub 不稳定（git fetch / codeload 均超时）
- 服务器 `git fetch origin`、`--depth 1`、codeload tarball、公共 ghproxy 镜像**全部 timeout**（github.com 网页可达，但 git 协议/大文件慢）
- 服务器 HEAD 停在 `033eeb5`（源码本身已含全部新功能，编译正常）
- 需确认服务器已有完整源码仓库 + Go 编译环境（`GOPROXY=mirrors.tencent.com`）

## 解决措施
1. **手动 SSH 部署**（`wxx_deploy.pem`，root@129.211.223.113）：
   - `systemctl stop wxx` → `cp /opt/wxx/bin/wxx-server /opt/wxx/wxx-server` → `systemctl start wxx`
   - 新二进制 64MB；健康检查 `{"status":"healthy","database":"mysql","latency":~90µs}`
2. **修正 CI**（commit `e75df78`）：二进制写入 systemd 实际路径 `/opt/wxx/wxx-server`，stop→cp→start
3. **CI git fetch 容错**（commit `a3604d0`）：fetch 超时不阻塞 job，用服务器现有源码编译（避免网络不稳定导致部署 job 失败被取消）

## 云端验证结果（实测）
| 项目 | 结果 |
|------|------|
| 登录 `stunion / Wxx@2026` | ✅ code:0，token 正常，role=student_union |
| GET /health/activities/review-stats | ✅ code:0（真实统计接口，非 404） |
| GET /health/activities | ✅ code:0（空列表，云端库暂无活动属正常） |
| /api/health | ✅ healthy，mysql 驱动 |

## 生产环境关键事实（从服务器 /etc/wxx/env 确认）
- 数据库：**MySQL**（`DB_DRIVER=mysql`，`DB_NAME=wxx`，非 SQLite！），非文档所述的 SQLite
- systemd ExecStart：`/opt/wxx/wxx-server`
- 前端静态：`FRONTEND_STATIC_DIR=/opt/wxx/frontend/web`（Caddy 服务）
- 登录正确密码：**`Wxx@2026`**（非 `wxx@2025`；文档 `内置账号与登录问题运维指引.md` 说默认 `wxx@2025` 与实测不符，以 `Wxx@2026` 为准）

## 遗留风险
- CI 自动后端部署依赖服务器的 git fetch；若 GitHub 持续不通，自动部署会 fallback 编译服务器现有源码（功能完整但可能滞后最新 commit）
- 若需每次都部署最新代码，需给服务器配 GitHub 加速（如 gitee 镜像 / ghproxy 代理）或改用 CI runner 侧打包上传的稳定通道
- 服务器 `wxx_deploy.pem` 私钥存在于本地 `~/.ssh/`，注意保管
