#!/bin/sh
set -e

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "${SCRIPT_DIR}/common.sh"

echo "=== Installing Panel ==="

if ! ensure_install_dir; then
    exit 1
fi

install_binary "xboard" "./cmd/xboard/main.go"

if [ -d "$USER_VITE_DIST" ]; then
    echo "Copying frontend assets..."
    if ! ensure_dir "$INSTALL_DIR/web/user-vite"; then
        exit 1
    fi
    if ! copy_recursive "$USER_VITE_DIST" "$INSTALL_DIR/web/user-vite/"; then
        echo "Error: failed to copy frontend assets."
        exit 1
    fi
else
    echo "Warning: Frontend assets not found at $USER_VITE_DIST. Skipping."
fi

if [ -d "$INSTALL_UI_DIR" ]; then
    echo "Copying install UI assets..."
    if ! ensure_dir "$INSTALL_DIR/web"; then
        exit 1
    fi
    if ! copy_recursive "$INSTALL_UI_DIR" "$INSTALL_DIR/web/"; then
        echo "Error: failed to copy install UI assets."
        exit 1
    fi
else
    echo "Warning: Install UI assets not found at $INSTALL_UI_DIR. Skipping."
fi

if [ ! -f "$INSTALL_DIR/config.yml" ] && [ ! -f "$INSTALL_DIR/.env" ]; then
    if [ -f "config.example.yml" ]; then
        if ! install_file "config.example.yml" "$INSTALL_DIR/config.yml"; then
            echo "Error: failed to create config.yml."
            exit 1
        fi
        echo "Created config.yml."
    elif [ -f ".env.example" ]; then
        if ! install_file ".env.example" "$INSTALL_DIR/.env"; then
            echo "Error: failed to create .env."
            exit 1
        fi
        echo "Created .env."
    fi
fi

if [ "$SKIP_SYSTEMD" = "1" ]; then
    echo "Skipping xboard.service installation (XBOARD_INSTALL_SKIP_SYSTEMD=1)."
elif ! is_systemd_available; then
    echo "Systemd is not available on this host. Skipping xboard.service installation."
else
    SERVICE_FILE=$(resolve_service_file "xboard.service")
    if [ -n "$SERVICE_FILE" ]; then
        if ! run_privileged cp "$SERVICE_FILE" /etc/systemd/system/xboard.service; then
            echo "Error: failed to install xboard.service."
            exit 1
        fi
        if ! run_privileged systemctl daemon-reload; then
            echo "Error: failed to run systemctl daemon-reload."
            exit 1
        fi
        if ! run_privileged systemctl enable xboard; then
            echo "Error: failed to enable xboard service."
            exit 1
        fi
        echo "xboard.service installed."
    else
        echo "Warning: deploy/xboard.service not found."
    fi
fi
