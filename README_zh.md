# XBoard (Go 版本)

<div align="center">

![Go](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)
![SQLite](https://img.shields.io/badge/SQLite-Embedded-003B57.svg)
![License](https://img.shields.io/badge/License-MIT-yellow.svg)

</div>

XBoard 已完全由 Go 重写：单一可执行文件即可提供 API、节点通信、后台任务与通知流水线，默认依赖 SQLite 与内存缓存，适合个人自托管或轻量服务器。仓库中已不再包含 Laravel/PHP 代码。

## ✨ 亮点

- **Go + Chi**：无需 PHP 运行时，接口保持与旧版兼容。
- **内置 SQLite + 迁移**：启动即自动执行 Goose 风格迁移，无需手动脚本。
- **后台作业内建**：订单处理、流量统计、节点遥测、通知队列全部内置。
- **真实数据策略**：所有接口访问真实仓储；未实现部分明确返回 501。
- **非商业定位**：聚焦 Config / Plan / User / Server / Stat 等“生存级”功能，订单/优惠券/支付等商业模块已移除。

## 📁 目录概览

```
cmd/              # xboard (统一 CLI 入口)
internal/         # API、Service、Repository、Job、Async、Bootstrap 等核心模块
pkg/, test/       # 预留扩展库与契约/集成测试
Dockerfile        # Go 多阶段构建
.env.example      # 环境变量示例
config.example.yml # YAML 配置示例
coding.md         # 官方架构文档
README.md         # 英文概览
README_zh.md      # 中文概览
todo.list         # 开发任务板
```

详细架构、约束与规划请参阅 `coding.md`。

## 🚀 快速开始

### 本地运行

```bash
# 1. 启用 Go 工具链（示例使用 gvm）
source ~/.gvm/scripts/gvm && gvm use go1.25.1

# 2. 准备配置
mkdir -p data
cp config.example.yml config.yml # 使用 YAML 配置（推荐）
# 或
cp .env.example .env   # 使用 .env（向后兼容）

# 3. 启动服务
go run ./cmd/xboard serve
```

服务默认监听 `0.0.0.0:8080`，首次启动会在 `data/xboard.db` 自动执行 SQLite 迁移。

### CLI 命令

`xboard` 二进制文件提供以下子命令：

- `xboard serve`: 启动 HTTP 服务（默认）。
- `xboard user`: 用户管理（创建、列表、重置密码等）。
- `xboard config`: 查看或更新系统配置。
- `xboard migrate`: 数据库迁移管理。
- `xboard backup`: 备份数据库。
- `xboard restore`: 从备份恢复数据库。
- `xboard job`: 管理后台任务。
- `xboard version`: 查看版本信息。

### 初始化向导

- 当数据库中尚未存在管理员账号时，服务会自动跳转到 `/install`，展示与面板同风格的安装引导。
- 引导界面允许填写“用户名（可选）/ 邮箱（可选）+ 密码”，至少提供其一即可完成初始化。
- 也可使用 CLI (`go run ./cmd/xboard user create --email admin@example.com --password secret --admin`) 手动创建。

### 管理前端

- Admin 前端已迁移至 Vite/React，构建产物已嵌入二进制文件中。
- 浏览器访问 `/{secure_path}`（默认 `/admin`）即可进入登录页，支持"邮箱 / 用户名"登录。
- 可通过 `config.yml` 中的 `ui.admin.enabled: false` 关闭内置前端。

### 用户前端

- 用户前端使用 Vite/React + HeroUI 组件库，支持亮色/暗色主题和中英双语。
- 浏览器访问 `/` 进入用户面板（需登录）。
- 功能：仪表盘、节点列表、套餐详情、流量统计、知识库、个人设置。
- 可通过 `config.yml` 中的 `ui.user.enabled: false` 关闭内置前端。

### Docker

```bash
docker build -t xboard .
docker run --rm -it \
  -p 8080:8080 \
  -v $(pwd)/data:/data \
  --name xboard \
  xboard serve
```

镜像中只包含编译后的二进制；`/data` 用于持久化 SQLite 文件。

### Systemd (Linux)

使用提供的脚本安装为 systemd 服务：

```bash
# 安装 panel + agent（需要 root）
sudo ./deploy/install.sh --full

# 仅安装 panel
sudo ./deploy/panel.sh

# 仅安装 agent
sudo ./deploy/agent.sh

# 通过 CloudPaste 短链安装 agent（stable）
sudo CLOUDPASTE_API_ENDPOINT="https://cloudpaste.example.com" \
  CLOUDPASTE_SLUG_PREFIX="xboard" \
  CLOUDPASTE_CHANNEL="stable" \
  ./deploy/agent.sh

# 通过显式直链安装 agent
sudo XBOARD_AGENT_DOWNLOAD_URL="https://cloudpaste.example.com/api/s/xboard-stable-agent-linux-amd64" \
  ./deploy/agent.sh

# 严格下载模式（短链下载失败即退出）
sudo CLOUDPASTE_API_ENDPOINT="https://cloudpaste.example.com" \
  XBOARD_AGENT_DOWNLOAD_STRICT=1 \
  ./deploy/agent.sh

# 启动服务
sudo systemctl start xboard

# 查看状态
sudo systemctl status xboard

# 卸载
sudo ./deploy/uninstall.sh
```

默认安装目录为 `/opt/xboard`。

agent 短链下载相关环境变量：
- `CLOUDPASTE_API_ENDPOINT`：CloudPaste 基址（支持带或不带 `/api`）。
- `CLOUDPASTE_SLUG_PREFIX`：slug 前缀（默认 `xboard`）。
- `CLOUDPASTE_CHANNEL`：`stable` 或 `pre`（默认 `stable`）。
- `CLOUDPASTE_ALLOW_CHANNEL_DRIFT`：允许回退到对侧通道 slug（默认 `true`）。
- `XBOARD_AGENT_DOWNLOAD_URL`：显式直链覆盖（优先于 endpoint+slug 方式）。
- `XBOARD_AGENT_DOWNLOAD_STRICT=1`：失败即退出，不回退本地二进制/源码构建。

## ⚙️ 配置参数

配置优先读取 `config.yml`，同时支持环境变量覆盖（适合容器化部署）。

详见 `config.example.yml` 及 `coding.md`。

## 🧪 开发流程

| 动作 | 命令 |
| --- | --- |
| 安装依赖 | `go mod tidy` |
| 代码格式化 | `gofmt -w ./cmd ./internal ./pkg ./test` |
| 单元测试 | `go test ./...` |
| 启动服务 | `go run ./cmd/xboard serve` |
| 完整构建 | `make build` |
| 仅构建前端 | `make build-frontend` |
| 仅构建后端 | `make build-backend` |
| 冒烟测试 | `make smoke` |

## 📊 功能状态（2025-12）

- ✅ Admin：Config / Plan / User / Server / Stat / Notice / Knowledge。
- ✅ Admin 前端：Vite/React，已嵌入二进制。
- ✅ User：订阅、流量日志、节点列表、公告、知识库（订单入口已移除）。
- ✅ 用户前端：仪表盘、节点、套餐、流量、知识库、设置（Vite/React/HeroUI）。
- ✅ Server：心跳、遥测、流量上报。
- ✅ Background Jobs：流量汇总、节点采样、通知队列。
- 🚫 Deferred：支付、礼品卡、插件、主题、Ticket 等商业模块（默认返回 501）。

## ⚠️ 免责声明

本项目仅供个人研究与自托管使用，严禁用于任何商业化或违法行为；所有风险由使用者自行承担。

## 📄 许可证

[MIT](LICENSE)