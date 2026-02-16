#!/bin/bash

INSTALL_DIR="${INSTALL_DIR:-/opt/xboard}"
USER_VITE_DIST="${USER_VITE_DIST:-web/user-vite/dist}"
INSTALL_UI_DIR="${INSTALL_UI_DIR:-web/install}"
SKIP_SYSTEMD="${XBOARD_INSTALL_SKIP_SYSTEMD:-0}"

XBOARD_RELEASE_REPO="${XBOARD_RELEASE_REPO:-creamcroissant/xboard2p}"
XBOARD_RELEASE_TAG="${XBOARD_RELEASE_TAG:-latest}"
XBOARD_RELEASE_BASE_URL="${XBOARD_RELEASE_BASE_URL:-https://github.com}"
XBOARD_RELEASE_DOWNLOAD_STRICT="${XBOARD_RELEASE_DOWNLOAD_STRICT:-0}"

OS_RAW=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS_RAW" in
    linux*) OS="linux" ;;
    darwin*) OS="darwin" ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
    *) OS="$OS_RAW" ;;
esac

ARCH_RAW=$(uname -m)
case "$ARCH_RAW" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) ARCH="$ARCH_RAW" ;;
esac

download_release_binary() {
    local bin_name=$1
    local target_path=$2

    if ! command -v curl >/dev/null 2>&1; then
        echo "Warning: curl not found, skipping release binary download for ${bin_name}."
        return 1
    fi

    local ext=""
    if [ "$OS" = "windows" ]; then
        ext=".exe"
    fi

    local asset="${bin_name}-${OS}-${ARCH}${ext}"
    local base="${XBOARD_RELEASE_BASE_URL%/}"
    local url

    if [ "$XBOARD_RELEASE_TAG" = "latest" ]; then
        url="${base}/${XBOARD_RELEASE_REPO}/releases/latest/download/${asset}"
    else
        url="${base}/${XBOARD_RELEASE_REPO}/releases/download/${XBOARD_RELEASE_TAG}/${asset}"
    fi

    local tmp_bin
    tmp_bin="$(mktemp)"

    if ! curl --fail --silent --show-error --location --output "$tmp_bin" "$url"; then
        rm -f "$tmp_bin"
        return 1
    fi

    if [ ! -s "$tmp_bin" ]; then
        echo "Warning: downloaded ${asset} is empty from ${url}."
        rm -f "$tmp_bin"
        return 1
    fi

    mv "$tmp_bin" "$target_path"
    chmod +x "$target_path"
    echo "Installed ${bin_name} from release asset: ${url}"
    return 0
}

install_binary() {
    local bin_name=$1
    local cmd_path=$2

    local ext=""
    local target_bin="$bin_name"
    if [ "$OS" = "windows" ]; then
        ext=".exe"
        target_bin="${bin_name}.exe"
    fi

    local arch_bin="${bin_name}-${OS}-${ARCH}${ext}"

    mkdir -p "$INSTALL_DIR"

    if download_release_binary "$bin_name" "$INSTALL_DIR/$target_bin"; then
        return 0
    fi

    if [ "$XBOARD_RELEASE_DOWNLOAD_STRICT" = "1" ]; then
        echo "Error: failed to download ${bin_name} from GitHub release and strict mode is enabled."
        exit 1
    fi

    if [ -f "$target_bin" ]; then
        echo "Installing $target_bin from current directory..."
        cp "$target_bin" "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/$target_bin"
    elif [ "$target_bin" != "$bin_name" ] && [ -f "$bin_name" ]; then
        echo "Installing $bin_name from current directory..."
        cp "$bin_name" "$INSTALL_DIR/$target_bin"
        chmod +x "$INSTALL_DIR/$target_bin"
    elif [ -f "$arch_bin" ]; then
        echo "Installing $arch_bin from current directory..."
        cp "$arch_bin" "$INSTALL_DIR/$target_bin"
        chmod +x "$INSTALL_DIR/$target_bin"
    elif [ -f "$cmd_path" ]; then
        echo "Building $bin_name..."
        if ! command -v go &> /dev/null; then
            echo "Error: 'go' command not found and binary not provided. Please install Go or provide the compiled binary."
            exit 1
        fi
        go build -o "$INSTALL_DIR/$target_bin" "$cmd_path"
    else
        echo "Error: ${bin_name} binary or source not found."
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
