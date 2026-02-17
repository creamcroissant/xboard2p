#!/bin/sh
set -e

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "${SCRIPT_DIR}/common.sh"

echo "=== Installing Agent ==="

if ! ensure_install_dir; then
    exit 1
fi

install_binary "agent" "./cmd/agent/main.go"

if [ ! -f "$INSTALL_DIR/agent_config.yml" ]; then
    if [ -f "agent_config.example.yml" ]; then
        if ! install_file "agent_config.example.yml" "$INSTALL_DIR/agent_config.yml"; then
            echo "Error: failed to create agent_config.yml from template."
            exit 1
        fi
    else
        cat > "$INSTALL_DIR/agent_config.yml" <<EOF
panel:
  # REQUIRED: replace with token created from Panel agent-host API
  host_token: ""

grpc:
  enabled: true
  # REQUIRED: replace with real Panel gRPC address, e.g. 10.0.0.2:9090
  address: ""
  tls:
    enabled: false

traffic:
  type: "netio"
EOF
    fi
    echo "Created agent_config.yml."
fi

if [ "$SKIP_SYSTEMD" = "1" ]; then
    echo "Skipping xboard-agent.service installation (XBOARD_INSTALL_SKIP_SYSTEMD=1)."
elif ! is_systemd_available; then
    echo "Systemd is not available on this host. Skipping xboard-agent.service installation."
else
    SERVICE_FILE=$(resolve_service_file "agent.service")
    if [ -n "$SERVICE_FILE" ]; then
        if ! run_privileged cp "$SERVICE_FILE" /etc/systemd/system/xboard-agent.service; then
            echo "Error: failed to install xboard-agent.service."
            exit 1
        fi
        if ! run_privileged systemctl daemon-reload; then
            echo "Error: failed to run systemctl daemon-reload."
            exit 1
        fi
        if ! run_privileged systemctl enable xboard-agent; then
            echo "Error: failed to enable xboard-agent service."
            exit 1
        fi
        echo "xboard-agent.service installed."
    else
        echo "Warning: agent.service not found (checked override/env/local paths)."
    fi
fi
