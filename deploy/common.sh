#!/bin/bash

INSTALL_DIR="${INSTALL_DIR:-/opt/xboard}"
USER_VITE_DIST="${USER_VITE_DIST:-web/user-vite/dist}"
INSTALL_UI_DIR="${INSTALL_UI_DIR:-web/install}"
SKIP_SYSTEMD="${XBOARD_INSTALL_SKIP_SYSTEMD:-0}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH_RAW=$(uname -m)
case "$ARCH_RAW" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) ARCH="$ARCH_RAW" ;;
esac

install_binary() {
    local bin_name=$1
    local cmd_path=$2

    local arch_bin="${bin_name}-${OS}-${ARCH}"

    if [ -f "$bin_name" ]; then
        echo "Installing $bin_name from current directory..."
        cp "$bin_name" "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/$bin_name"
    elif [ -f "$arch_bin" ]; then
        echo "Installing $arch_bin from current directory..."
        cp "$arch_bin" "$INSTALL_DIR/$bin_name"
        chmod +x "$INSTALL_DIR/$bin_name"
    elif [ -f "$cmd_path" ]; then
        echo "Building $bin_name..."
        if ! command -v go &> /dev/null; then
            echo "Error: 'go' command not found and binary not provided. Please install Go or provide the compiled binary."
            exit 1
        fi
        go build -o "$INSTALL_DIR/$bin_name" "$cmd_path"
    else
        echo "Error: $bin_name binary or source not found."
        echo "Please run this script from the project root or provide the compiled binary."
        exit 1
    fi
}

resolve_service_file() {
    local service_name=$1

    if [ -f "${SCRIPT_DIR}/${service_name}" ]; then
        echo "${SCRIPT_DIR}/${service_name}"
    elif [ -f "deploy/${service_name}" ]; then
        echo "deploy/${service_name}"
    elif [ -f "${service_name}" ]; then
        echo "${service_name}"
    fi

    return 0
}
