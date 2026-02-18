# XBoard (Go Edition)

<div align="center">

![Go](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)
![SQLite](https://img.shields.io/badge/SQLite-Embedded-003B57.svg)
![License](https://img.shields.io/badge/License-MIT-yellow.svg)

</div>

XBoard is now fully migrated to Go: a single binary provides API, node communication, background tasks, and notification pipeline. It defaults to SQLite and in-memory caching, making it easy for local or lightweight node deployment. This repository no longer contains Laravel/PHP code.

## ✨ Highlights

- **Go + Chi**: No external PHP runtime required, HTTP layer and routing are compatible with the original version.
- **SQLite + Embedded Migration**: Out-of-the-box embedded database, automatically executes Goose-style migrations on startup.
- **Built-in Scheduler**: Order processing, traffic aggregation, node telemetry, and notifications are all handled by Go jobs.
- **Real Data Strategy**: All interfaces access the repository/service; unimplemented parts return 501.
- **Non-Commercial Positioning**: Focused on core panel capabilities like Config / Plan / User / Server / Stat; commercial modules like Orders/Coupons/Payments have been removed.

## 📁 Directory

```
cmd/
├── xboard/           # Panel main program (serve, tui, user, config, etc.)
└── agent/            # Agent program
internal/             # API, Service, Repository, Jobs, Async, Bootstrap...
pkg/, test/           # Shared libraries and contract/integration tests
web/user-vite/        # Unified frontend (Vite + React + shadcn/ui)
scripts/              # Build and test scripts
Dockerfile            # Go multi-stage build
.env.example          # Environment variable example
config.example.yml    # YAML configuration example
```

For more details, stage goals, and architectural constraints, see `coding.md`.

## 🚀 Quick Start

### Local Run

```bash
# 1. Prepare Go toolchain
source ~/.gvm/scripts/gvm && gvm use go1.25.1   # Or any Go 1.25+

# 2. Initialize configuration
mkdir -p data
cp config.example.yml config.yml # Use YAML config (recommended)
# OR
cp .env.example .env    # Use .env (backward compatible)

# 3. Start service
go run ./cmd/xboard serve
```

Default listens on `0.0.0.0:8080`. First start will automatically execute SQLite migrations in `data/xboard.db`.

### CLI Commands

The `xboard` binary provides several subcommands:

- `xboard serve`: Start the HTTP server (default).
- `xboard user`: User management (create, list, reset-password, etc.).
- `xboard config`: View or update system configuration.
- `xboard migrate`: Manage database migrations.
- `xboard backup`: Backup database.
- `xboard restore`: Restore database from backup.
- `xboard job`: Manage background jobs.
- `xboard version`: Show version information.

### Initialization Wizard

- If no admin account exists in the database, the HTTP service automatically redirects to `/install` to show the initialization interface.
- The wizard allows creating the first admin account with "Username (optional) / Email (optional) + Password".
- Alternatively, use CLI: `go run ./cmd/xboard user create --email admin@example.com --password secret --admin`.

### Admin Frontend

- Admin Frontend uses Vite/React, built assets are embedded in the binary.
- Access `/{secure_path}` (default `/admin`) in browser to open login page.
- Can be disabled via config `ui.admin.enabled: false` for custom CDN deployment.

### User Frontend

- User Frontend uses Vite/React with shadcn/ui components, supporting light/dark themes and Chinese/English localization.
- Access `/` in browser to open user dashboard (requires login).
- Features: Dashboard, Server list, Plan details, Traffic statistics, Knowledge base, Settings.
- Can be disabled via config `ui.user.enabled: false`.

### Docker

```bash
docker build -t xboard .
docker run --rm -it \
  -p 8080:8080 \
  -v $(pwd)/data:/data \
  --name xboard \
  xboard serve
```

### Systemd (Linux)

Use the provided scripts to install as a systemd service:

```bash
# Install panel + agent (requires root)
sudo ./deploy/install.sh --full

# Install panel only
sudo ./deploy/panel.sh

# Install agent only
sudo ./deploy/agent.sh

# One-liner bootstrap entry (downloads agent.sh/common.sh/agent.service + verifies SHA256)
curl -fsSL https://raw.githubusercontent.com/creamcroissant/xboard/master/deploy/agent-bootstrap.sh -o /tmp/agent-bootstrap.sh && \
  sudo INSTALL_DIR=/opt/xboard sh /tmp/agent-bootstrap.sh --ref latest

# Bootstrap with explicit tag (script/service/binary version bound to same tag)
curl -fsSL https://raw.githubusercontent.com/creamcroissant/xboard/master/deploy/agent-bootstrap.sh -o /tmp/agent-bootstrap.sh && \
  sudo INSTALL_DIR=/opt/xboard sh /tmp/agent-bootstrap.sh --ref v1.2.3

# Start service
sudo systemctl start xboard

# Check status
sudo systemctl status xboard

# Uninstall
sudo ./deploy/uninstall.sh
```

Default installation directory is `/opt/xboard`.

Agent install environment variables:
- `XBOARD_BOOTSTRAP_REF`: bootstrap target ref (`latest`, release tag, or commit hash; commit hash requires `XBOARD_RELEASE_TAG` to be set explicitly for version consistency).
- `XBOARD_BOOTSTRAP_REPO`: bootstrap source repository (default `creamcroissant/xboard`).
- `XBOARD_AGENT_SCRIPT_URL` / `XBOARD_COMMON_SCRIPT_URL` / `XBOARD_AGENT_SERVICE_URL`: optional override URLs for private mirror or emergency fallback.
- `XBOARD_BOOTSTRAP_CHECKSUM_URL`: optional checksum manifest URL override.
- `XBOARD_BOOTSTRAP_DOWNLOAD_STRICT=1`: fail-closed mode for bootstrap service file handling. If `agent.service` download/checksum fails, bootstrap exits immediately (no local fallback).

Bootstrap `agent.service` local fallback priority (used when `XBOARD_BOOTSTRAP_DOWNLOAD_STRICT=0`, default):
1. `XBOARD_AGENT_SERVICE_FILE`
2. `${CALLER_DIR}/deploy/agent.service`
3. `${CALLER_DIR}/agent.service`

Fallback is triggered on:
- remote `agent.service` download failure
- `agent.service` checksum verification failure

Strict bootstrap example (production fail-closed):
```bash
curl -fsSL https://raw.githubusercontent.com/creamcroissant/xboard/master/deploy/agent-bootstrap.sh -o /tmp/agent-bootstrap.sh && \
  sudo INSTALL_DIR=/opt/xboard XBOARD_BOOTSTRAP_DOWNLOAD_STRICT=1 sh /tmp/agent-bootstrap.sh --ref latest
```

## ⚙️ Configuration

Configuration is loaded from `config.yml` (preferred) or Environment Variables (for containerization).

See `config.example.yml` for structure and `coding.md` for details.

## 🧪 Development Workflow

| Action | Command |
| --- | --- |
| Install Deps | `go mod tidy` |
| Format Code | `gofmt -w ./cmd ./internal ./pkg ./test` |
| Unit Test | `go test ./...` |
| Run Service | `go run ./cmd/xboard serve` |
| Build All | `make build` |
| Build Frontend Only | `make build-frontend` |
| Build Backend Only | `make build-backend` |
| Smoke Test | `make smoke` |

## 📊 Feature Status (2026-01)

- ✅ Admin: Config / Plan / User / Server / Stat / Notice / Knowledge / Forwarding / System Settings.
- ✅ Admin Frontend: Vite/React (shadcn/ui), embedded in binary.
- ✅ User: Subscription, Traffic Log, Node List, Announcement, Knowledge Base, Profile Settings.
- ✅ User Frontend: Dashboard, Servers, Plans, Traffic, Knowledge, Settings (Vite/React/shadcn/ui).
- ✅ Server: Heartbeat, telemetry, traffic reporting, core switching (Sing-box/Xray).
- ✅ Background Jobs: Traffic Aggregation, Node Sampling, Notification Queue, Traffic Reset.
- ✅ Security: Rate Limiting, Captcha, IP-based Restrictions, Input Validation.
- 🚫 Deferred: Payment, Gift Card, Plugin, Theme, Ticket (Handlers return 501).

## ⚠️ Disclaimer

This project is for personal research and self-hosting only. Use for commercial or illegal purposes is strictly prohibited. Users assume all risks.

## 📄 License

[MIT](LICENSE)