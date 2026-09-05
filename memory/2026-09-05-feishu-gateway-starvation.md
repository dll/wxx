# 2026-09-05 飞书机器人无回复事故(gateway 事件循环饿死)

## 现象
- @DeepSeek-v4-Pro(deepseek-eqs / leader-eqs)群内 @ 不回复;期间所有飞书账号出站 Update 超时。

## 根因(两层)
1. **孤儿 `find /` 进程饿死 gateway**:9-03 Claude Code(ogdp 项目,sonnet-5)在 Git Bash 执行 `find / -iname 'plantuml*.jar' | head -20`,head 退出后 find 未被杀,孤儿进程扫盘 48h、累计 50+ 小时 CPU(≈占满 1 核),gateway Node 事件循环单次阻塞达 80s,出站 HTTPS 全超时。
2. **leader-eqs 主模型不可用**:`opencode/deepseek-v4-flash-free` 上游 400(Model is unavailable),每次先失败再 fallback 到 v4-pro,拖 1-2 分钟。

## 处置
- 强杀孤儿 find(pid 40844 + 残余 40064);杀后 30s 内 6 个账号 bot 身份恢复、模型调用 200、回复送达。
- `openclaw.json` `agents.defaults.model.primary` 改为 `deepseek-openclaw/deepseek-v4-flash`,fallback `v4-pro`(备份 `.bak-eqs-model-20260905-1845`)。热加载生效,无需重启。只影响无 model 字段的 leader-eqs 及其子代理。

## 教训
- `find /` 在 Git Bash 会扫 Git 安装目录起的整个盘;Claude Code Bash 工具超时后不杀子进程。**禁止 `find /`**,搜文件用 Everything(es.exe)或限定目录。
- 事件循环阻塞 80s 时,别急着重启 gateway——先查 CPU 异常进程(Get-Process 按 CPU 排序,CPU 秒数按核龄折算)。
- gateway 18789 端口监听者是 session 0 服务会话进程时,`Get-CimInstance` 查不到命令行属正常,不是僵尸;判断谁在干活看 TCP 443 连接归属和日志时间戳。
- 日志:`$env:LOCALAPPDATA\Temp\openclaw\openclaw-YYYY-MM-DD.log`;关键事件:`model fallback decision`、`stuck session recovery`、`per-chat task exceeded 300000ms cap`、`bot open_id recovered via background retry`。
