#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

echo "=== Installing Panel ==="

mkdir -p "$INSTALL_DIR"

install_binary "xboard" "./cmd/xboard/main.go"

if [ -d "$USER_VITE_DIST" ]; then
    echo "Copying frontend assets..."
    mkdir -p "$INSTALL_DIR/web/user-vite"
    cp -r "$USER_VITE_DIST" "$INSTALL_DIR/web/user-vite/"
else
    echo "Warning: Frontend assets not found at $USER_VITE_DIST. Skipping."
fi

if [ -d "$INSTALL_UI_DIR" ]; then
    echo "Copying install UI assets..."
    mkdir -p "$INSTALL_DIR/web"
    cp -r "$INSTALL_UI_DIR" "$INSTALL_DIR/web/"
else
    echo "Warning: Install UI assets not found at $INSTALL_UI_DIR. Skipping."
fi

if [ ! -f "$INSTALL_DIR/config.yml" ] && [ ! -f "$INSTALL_DIR/.env" ]; then
    if [ -f "config.example.yml" ]; then
        cp config.example.yml "$INSTALL_DIR/config.yml"
        echo "Created config.yml."
    elif [ -f ".env.example" ]; then
        cp .env.example "$INSTALL_DIR/.env"
        echo "Created .env."
    fi
fi

if [ "$SKIP_SYSTEMD" = "1" ]; then
    echo "Skipping xboard.service installation (XBOARD_INSTALL_SKIP_SYSTEMD=1)."
else
    SERVICE_FILE="$(resolve_service_file "xboard.service")"
    if [ -n "$SERVICE_FILE" ]; then
        cp "$SERVICE_FILE" /etc/systemd/system/
        systemctl daemon-reload
        systemctl enable xboard
        echo "xboard.service installed."
    else
        echo "Warning: deploy/xboard.service not found."
    fi
fi
