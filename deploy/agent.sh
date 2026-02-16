#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

echo "=== Installing Agent ==="

install_binary "agent" "./cmd/agent/main.go"


if [ ! -f "$INSTALL_DIR/agent_config.yml" ]; then
    if [ -f "agent_config.example.yml" ]; then
        cp agent_config.example.yml "$INSTALL_DIR/agent_config.yml"
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
else
    SERVICE_FILE="$(resolve_service_file "agent.service")"
    if [ -n "$SERVICE_FILE" ]; then
        cp "$SERVICE_FILE" /etc/systemd/system/xboard-agent.service
        systemctl daemon-reload
        systemctl enable xboard-agent
        echo "xboard-agent.service installed."
    else
        echo "Warning: deploy/agent.service not found."
    fi
fi
