# mirasim2api

以 [sub2api](https://github.com/Wei-Shaw/sub2api) 的产品架构为蓝本（多账号池、API Key 分发、
精确计费、智能调度、并发控制、管理后台），上游适配 **Mirasim** 订阅额度的模型网关。

单一 Go 二进制交付，管理后台内嵌，SQLite 持久化，无外部中间件依赖。

> ⚠️ **免责声明**：本项目仅供技术学习与研究。使用可能违反 Mirasim 及上游（Anthropic/OpenAI 等）
> 的服务条款，风险由使用者自负；不得用于商业运营。

---

## 功能特性

- **多账号池**：GitHub/Google OAuth（loopback 自动回调或粘贴回调链接）+ 邮箱验证码 + 直接粘贴 refresh token 三种加号方式；refresh token AES-256-GCM 加密落盘，轮换自动回写
- **API Key 分发**：`sk-` 密钥（只存 sha256），每 Key 可配并发上限 / 每分钟请求数 / 模型白名单 / 过期时间
- **精确计费**：从响应（含 SSE 流末帧）解析 token 用量，模型价格表 × 倍率计费，写用量日志并累加到 Key
- **智能调度**：额度加权随机选号 + 会话粘性（prompt cache 友好）+ 出错指数退避熔断
- **并发与限流**：每账号并发信号量 + 每 Key 并发槽 + 每 Key 滑动窗口限流
- **协议兼容**：实现 Mirasim 的 mrs-sig-v2 签名与 mrs-seal-v1 密封，见 `docs/PROTOCOL.md`
- **管理后台**：内嵌单页 WebUI，仪表盘 / 账号池 / API Keys / 用量日志 / 设置，无需前端工具链

## 支持的端点

| 端点 | 模型家族 |
|---|---|
| `POST /v1/messages` | `claude-*`、`kimi-k3`、`deepseek-*`、`glm-*` |
| `POST /v1/responses` | `gpt-*`（仅流式）、第三方模型 |
| `POST /v1/chat/completions` | `kimi-k3`、`deepseek-*`、`glm-*` |
| `GET /v1/models` | 模型清单 |
| `GET /health` | 健康检查（账号池摘要） |

> 模型家族路由遵循 Mirasim relay 实测约束：`claude-*` 仅 `/v1/messages`，`gpt-*` 仅 `/v1/responses`，
> 第三方模型三个端点皆可。详见 `docs/PROTOCOL.md`。

同时注册裸路径（`POST /messages` 等，无 `/v1` 前缀），方便不改 base_url 的客户端。

---

## 快速开始

### 方式一：本地运行

需要 Go 1.27+。

```bash
go build -o mirasim2api ./cmd/server
ADMIN_PASSWORD=your-strong-password ./mirasim2api
```

打开 `http://127.0.0.1:8787/admin/`，用上面设置的密码登录。

### 方式二：Docker Compose

```bash
cd deploy
cp .env.example .env
# 编辑 .env：至少设置 ADMIN_PASSWORD 与 MASTER_KEY
docker compose up -d --build
```

数据持久化在名为 `mirasim-data` 的 volume（`/app/data`）。

### 方式三：macOS 菜单栏 App（轻量后台模式）

常驻菜单栏运行网关：无 Dock 图标、无终端窗口，菜单一键打开管理后台、复制接口地址、
开关「开机自动启动」（macOS LaunchAgent，登录即自动拉起）。

```bash
./deploy/macos/build_app.sh        # 产出 build/macos/Mirasim2API.app
open build/macos/Mirasim2API.app   # 立即运行；或拖入 /Applications 长期使用
```

- 菜单项：运行状态（启动中 / 运行中·地址 / 启动失败原因）、打开管理后台、复制接口地址、
  打开数据目录、**开机自动启动**（勾选后下次登录生效，再勾取消）、退出
- 桌面默认与终端不同：`HOST=127.0.0.1`、`DATA_DIR=~/Library/Application Support/mirasim2api`
  （终端模式仍是 `./data`）；环境变量优先，如 `PORT=9999` 可换端口
- 日志在 `DATA_DIR/logs/app.log`；启动失败（如端口被占）不会弹窗，状态栏显示原因
- 技术栈：`getlantern/systray`（CGO），`LSUIElement` 菜单栏 agent，图标由
  `deploy/macos/genicon` 纯 Go 生成

### 方式四：Windows 托盘 App（x64）

与 macOS 版同一套代码：右下角系统托盘常驻、无控制台窗，菜单功能完全一致
（状态 / 打开管理后台 / 复制接口地址 / 打开数据目录 / **开机自动启动** / 退出）。

```bash
./deploy/windows/build_exe.sh      # 任意平台上交叉编译，产出 build/windows/Mirasim2API.exe（+zip）
```

- 适用机型：**x86-64（amd64）CPU 即可**（如 Intel / AMD 锐龙全系），纯静态单 exe，无依赖
- 「开机自动启动」写 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`，登录即起，免管理员权限
- 数据目录默认 `%LOCALAPPDATA%\mirasim2api`；exe 已嵌入图标与 PerMonitorV2 DPI 清单，高分辨率下不糊
- 未签名，首次运行 SmartScreen 可能提示「未知的发布者」，点「仍要运行」即可

### 添加账号

管理后台 →「账号池」→ 右上角三种方式任选：

- **OAuth 登录**：选 GitHub/Google，自动在本机 127.0.0.1 起临时端口接收回调（推荐，全自动）；
  或选手动模式，授权后把浏览器地址栏的完整回调链接粘贴回来
- **邮箱验证码**：输入邮箱收码，填码即入池
- **粘贴 Token**：直接粘贴一个有效的 refresh token（三段 JWT）

### 创建 API Key

管理后台 →「API Keys」→「新建 Key」，可设并发 / RPM / 模型白名单 / 过期时间。
**明文密钥只在创建时显示一次**，请立即保存。

### 调用模型

```bash
curl http://127.0.0.1:8787/v1/messages \
  -H "Authorization: Bearer sk-你的key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-5",
    "max_tokens": 256,
    "messages": [{"role":"user","content":"你好"}]
  }'
```

流式：`"stream": true`，网关逐行透传 SSE 并从流中解析用量计费。

---

## 配置（环境变量）

| 变量 | 默认 | 说明 |
|---|---|---|
| `PORT` / `HOST` | `8787` / `0.0.0.0` | 监听地址 |
| `DATA_DIR` | `./data` | SQLite 与 master.key 目录 |
| `MASTER_KEY` | 自动生成存 `DATA_DIR/master.key`（0600） | 64 位 hex，加密 refresh token 与设备私钥；**丢失则已存凭据解不开** |
| `ADMIN_PASSWORD` | 空 | 管理后台密码；空且绑定非 loopback 时**拒绝启动** |
| `GATEWAY_KEY_REQUIRED` | `true` | 模型请求是否必须带 sk- key |
| `CLAUDE_CLOAK_MODE` | `relaxed` | `relaxed` / `strict` |
| `CAPACITY_RETRIES` / `CAPACITY_BACKOFF_MS` | `2` / `1200` | 503 容量重试次数 / 退避毫秒 |
| `ACCOUNT_MAX_CONCURRENCY` | `5` | 每账号上游并发槽上限 |
| `UPSTREAM_PROXY` | 空 | 出站 HTTP CONNECT 代理 |
| `RELAY_URL` / `AUTH_URL` | 官方地址 | 上游覆盖 |
| `MIRASIM_CLIENT_VERSION` | `0.0.322` | 参与签名的客户端版本 |
| `MIRASIM_SEAL_PUBKEY` | 内置值 | relay 密封公钥 |

标记「可热更」的设置（代理 / 伪装模式 / 重试参数 / 价格表 / 倍率）也可在管理后台「设置」页在线修改，
数据库值优先于环境变量。

---

## 项目结构

```
cmd/server/          CLI 入口
cmd/desktop/         桌面托盘 app（systray：macOS 菜单栏 agent / Windows 系统托盘）
internal/server/     服务装配与运行（CLI 与桌面 app 共用）
internal/autostart/  开机自启（macOS LaunchAgent，其他平台回退不支持）
internal/config/     环境变量配置
internal/mirasim/    协议层：设备身份、mrs-sig-v2 签名、mrs-seal-v1 密封、token 刷新、ticket 申领
internal/store/      SQLite：accounts / api_keys / usage_logs / settings / device
internal/pool/       账号池：额度加权选号、会话粘性、熔断冷却、配额轮询
internal/gateway/    模型网关：请求规范化、鉴权、并发限流、转发、SSE 透传、用量解析
internal/billing/    用量记录与费用计算
internal/admin/      管理 API（会话认证）
internal/runtimecfg/ 运行时设置（数据库优先，环境变量兜底）
web/                 管理后台静态资源（embed）
deploy/              Dockerfile、docker-compose.yml、.env.example、
                     macos/（.app 构建脚本 + 图标生成器）、windows/（.exe 交叉编译脚本）
docs/                DESIGN.md（设计）、PROTOCOL.md（协议规格）
```

## 安全说明

- refresh token 与设备私钥一律 AES-256-GCM 加密落盘
- API Key 只存 sha256，展示用 `sk-...abcd` 截断形式
- 管理会话用 HttpOnly Cookie（HMAC 签名 token，8 小时有效）
- 未设 `ADMIN_PASSWORD` 且绑定非 loopback 地址时拒绝启动

## 测试

```bash
go test ./...
```

各协议、规范化、池选号、计费、存储模块均有单元测试。
