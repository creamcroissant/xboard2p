#!/bin/sh

INSTALL_DIR="${INSTALL_DIR:-/opt/xboard}"
USER_VITE_DIST="${USER_VITE_DIST:-web/user-vite/dist}"
INSTALL_UI_DIR="${INSTALL_UI_DIR:-web/install}"
SKIP_SYSTEMD="${XBOARD_INSTALL_SKIP_SYSTEMD:-0}"

XBOARD_RELEASE_REPO="${XBOARD_RELEASE_REPO:-creamcroissant/xboard2p}"
XBOARD_RELEASE_TAG="${XBOARD_RELEASE_TAG:-latest}"
XBOARD_RELEASE_BASE_URL="${XBOARD_RELEASE_BASE_URL:-https://github.com}"
XBOARD_RELEASE_DOWNLOAD_STRICT="${XBOARD_RELEASE_DOWNLOAD_STRICT:-0}"

DISTRO_ID=""
DISTRO_ID_LIKE=""
PKG_MANAGER=""
PKG_CACHE_UPDATED=0

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

strip_quotes() {
    value=$1
    value=${value#\"}
    value=${value%\"}
    value=${value#\'}
    value=${value%\'}
    printf '%s' "$value"
}

load_os_release() {
    if [ -n "$DISTRO_ID" ] || [ -n "$DISTRO_ID_LIKE" ]; then
        return 0
    fi

    if [ ! -r /etc/os-release ]; then
        return 0
    fi

    while IFS='=' read -r key value; do
        case "$key" in
            ID)
                DISTRO_ID=$(strip_quotes "$value")
                ;;
            ID_LIKE)
                DISTRO_ID_LIKE=$(strip_quotes "$value")
                ;;
        esac
    done < /etc/os-release

    DISTRO_ID=$(printf '%s' "$DISTRO_ID" | tr '[:upper:]' '[:lower:]')
    DISTRO_ID_LIKE=$(printf '%s' "$DISTRO_ID_LIKE" | tr '[:upper:]' '[:lower:]')
}

has_like() {
    like_word=$1
    case " $DISTRO_ID_LIKE " in
        *" $like_word "*)
            return 0
            ;;
    esac
    return 1
}

detect_pkg_manager() {
    if [ -n "$PKG_MANAGER" ]; then
        return 0
    fi

    load_os_release

    case "$DISTRO_ID" in
        ubuntu|debian)
            if command -v apt-get >/dev/null 2>&1; then
                PKG_MANAGER="apt-get"
                return 0
            fi
            ;;
        fedora)
            if command -v dnf >/dev/null 2>&1; then
                PKG_MANAGER="dnf"
                return 0
            fi
            ;;
        rhel|rocky|almalinux|ol|amzn|centos)
            if command -v dnf >/dev/null 2>&1; then
                PKG_MANAGER="dnf"
                return 0
            fi
            if command -v yum >/dev/null 2>&1; then
                PKG_MANAGER="yum"
                return 0
            fi
            ;;
        alpine)
            if command -v apk >/dev/null 2>&1; then
                PKG_MANAGER="apk"
                return 0
            fi
            ;;
        opensuse*|sles|sled)
            if command -v zypper >/dev/null 2>&1; then
                PKG_MANAGER="zypper"
                return 0
            fi
            ;;
        arch|manjaro)
            if command -v pacman >/dev/null 2>&1; then
                PKG_MANAGER="pacman"
                return 0
            fi
            ;;
    esac

    if has_like "debian" && command -v apt-get >/dev/null 2>&1; then
        PKG_MANAGER="apt-get"
        return 0
    fi

    if has_like "rhel" || has_like "fedora"; then
        if command -v dnf >/dev/null 2>&1; then
            PKG_MANAGER="dnf"
            return 0
        fi
        if command -v yum >/dev/null 2>&1; then
            PKG_MANAGER="yum"
            return 0
        fi
    fi

    if has_like "suse" && command -v zypper >/dev/null 2>&1; then
        PKG_MANAGER="zypper"
        return 0
    fi

    if has_like "arch" && command -v pacman >/dev/null 2>&1; then
        PKG_MANAGER="pacman"
        return 0
    fi

    for manager in apt-get dnf yum apk zypper pacman; do
        if command -v "$manager" >/dev/null 2>&1; then
            PKG_MANAGER="$manager"
            return 0
        fi
    done

    PKG_MANAGER=""
    return 1
}

run_privileged() {
    if [ "$(id -u)" -eq 0 ]; then
        "$@"
        return $?
    fi

    if command -v sudo >/dev/null 2>&1; then
        sudo "$@"
        return $?
    fi

    echo "Error: root privileges are required to run: $*"
    echo "Please run as root or install sudo."
    return 1
}

ensure_dir() {
    dir_path=$1

    if mkdir -p "$dir_path" >/dev/null 2>&1; then
        return 0
    fi

    parent_dir=$(dirname "$dir_path")
    if [ -n "$parent_dir" ] && [ -w "$parent_dir" ]; then
        mkdir -p "$dir_path"
        return $?
    fi

    run_privileged mkdir -p "$dir_path"
}

install_file() {
    src_path=$1
    dst_path=$2

    if cp "$src_path" "$dst_path" >/dev/null 2>&1; then
        return 0
    fi

    run_privileged cp "$src_path" "$dst_path"
}

set_file_mode() {
    mode_value=$1
    target_path=$2

    if chmod "$mode_value" "$target_path" >/dev/null 2>&1; then
        return 0
    fi

    run_privileged chmod "$mode_value" "$target_path"
}

install_executable_file() {
    src_path=$1
    dst_path=$2

    if ! install_file "$src_path" "$dst_path"; then
        echo "Error: failed to copy file to ${dst_path}."
        return 1
    fi

    if ! set_file_mode +x "$dst_path"; then
        echo "Error: failed to set executable permission on ${dst_path}."
        return 1
    fi

    return 0
}

copy_recursive() {
    src_path=$1
    dst_path=$2

    if cp -r "$src_path" "$dst_path" >/dev/null 2>&1; then
        return 0
    fi

    run_privileged cp -r "$src_path" "$dst_path"
}

has_ca_certificates() {
    if [ -f /etc/ssl/certs/ca-certificates.crt ]; then
        return 0
    fi

    if [ -f /etc/ssl/cert.pem ]; then
        return 0
    fi

    if [ -f /etc/pki/tls/certs/ca-bundle.crt ]; then
        return 0
    fi

    if [ -f /etc/ssl/ca-bundle.pem ]; then
        return 0
    fi

    return 1
}

pkg_manager_env_key() {
    printf '%s' "$1" | tr '[:lower:]-' '[:upper:]_'
}

dependency_package_name() {
    dep_name=$1
    manager=$2

    dep_key=$(printf '%s' "$dep_name" | tr '[:lower:]-' '[:upper:]_')
    manager_key=$(pkg_manager_env_key "$manager")

    eval "override_pkg=\${XBOARD_PKG_${dep_key}_${manager_key}:-}"
    if [ -z "$override_pkg" ]; then
        eval "override_pkg=\${XBOARD_PKG_${dep_key}:-}"
    fi

    if [ -n "$override_pkg" ]; then
        printf '%s' "$override_pkg"
        return 0
    fi

    case "$dep_name" in
        curl)
            printf '%s' "curl"
            ;;
        ca-certificates)
            printf '%s' "ca-certificates"
            ;;
        go)
            case "$manager" in
                apt-get)
                    printf '%s' "golang-go"
                    ;;
                dnf|yum)
                    printf '%s' "golang"
                    ;;
                apk|zypper|pacman)
                    printf '%s' "go"
                    ;;
                *)
                    printf '%s' "go"
                    ;;
            esac
            ;;
        *)
            printf '%s' "$dep_name"
            ;;
    esac
}

install_packages() {
    if [ "$#" -eq 0 ]; then
        return 0
    fi

    if ! detect_pkg_manager; then
        echo "Error: no supported package manager detected."
        echo "Please manually install required dependencies: $*"
        return 1
    fi

    echo "Installing packages via ${PKG_MANAGER}: $*"

    case "$PKG_MANAGER" in
        apt-get)
            if [ "$PKG_CACHE_UPDATED" != "1" ]; then
                if ! run_privileged apt-get update; then
                    return 1
                fi
                PKG_CACHE_UPDATED=1
            fi
            run_privileged env DEBIAN_FRONTEND=noninteractive apt-get install -y "$@"
            ;;
        dnf)
            run_privileged dnf install -y "$@"
            ;;
        yum)
            run_privileged yum install -y "$@"
            ;;
        apk)
            run_privileged apk add --no-cache "$@"
            ;;
        zypper)
            run_privileged zypper --non-interactive install "$@"
            ;;
        pacman)
            run_privileged pacman -Sy --noconfirm --needed "$@"
            ;;
        *)
            echo "Error: unsupported package manager: ${PKG_MANAGER}"
            return 1
            ;;
    esac
}

dependency_available() {
    dep_name=$1

    case "$dep_name" in
        curl|go)
            command -v "$dep_name" >/dev/null 2>&1
            ;;
        ca-certificates)
            has_ca_certificates
            ;;
        *)
            command -v "$dep_name" >/dev/null 2>&1
            ;;
    esac
}

manual_dependency_hint() {
    dep_name=$1

    case "$dep_name" in
        ca-certificates)
            printf '%s' "ca-certificates"
            ;;
        go)
            printf '%s' "go compiler"
            ;;
        *)
            printf '%s' "$dep_name"
            ;;
    esac
}

ensure_dependency() {
    dep_name=$1

    if dependency_available "$dep_name"; then
        return 0
    fi

    if ! detect_pkg_manager; then
        echo "Error: dependency '${dep_name}' is missing and no supported package manager was detected."
        echo "Please manually install: $(manual_dependency_hint "$dep_name")."
        return 1
    fi

    pkg_name=$(dependency_package_name "$dep_name" "$PKG_MANAGER")
    if [ -z "$pkg_name" ]; then
        echo "Error: failed to resolve package name for dependency '${dep_name}'."
        return 1
    fi

    if ! install_packages "$pkg_name"; then
        echo "Error: failed to install dependency '${dep_name}' (package: ${pkg_name})."
        echo "Please manually install: $(manual_dependency_hint "$dep_name")."
        return 1
    fi

    if dependency_available "$dep_name"; then
        return 0
    fi

    echo "Error: dependency '${dep_name}' is still unavailable after installation."
    return 1
}

ensure_install_dir() {
    if ! ensure_dir "$INSTALL_DIR"; then
        echo "Error: cannot create install directory ${INSTALL_DIR}."
        return 1
    fi

    return 0
}

is_systemd_available() {
    if ! command -v systemctl >/dev/null 2>&1; then
        return 1
    fi

    if [ ! -d /run/systemd/system ]; then
        return 1
    fi

    if ! systemctl --version >/dev/null 2>&1; then
        return 1
    fi

    return 0
}

download_release_binary() {
    bin_name=$1
    target_path=$2

    ensure_download_deps=${XBOARD_AUTO_INSTALL_DOWNLOAD_DEPS:-1}
    if [ "$ensure_download_deps" = "1" ]; then
        if ! ensure_dependency "curl"; then
            echo "Warning: curl is unavailable, skipping release binary download for ${bin_name}."
            return 1
        fi

        if ! ensure_dependency "ca-certificates"; then
            echo "Warning: CA certificates are unavailable, skipping release binary download for ${bin_name}."
            return 1
        fi
    else
        if ! command -v curl >/dev/null 2>&1; then
            echo "Warning: curl not found, skipping release binary download for ${bin_name}."
            return 1
        fi
    fi

    ext=""
    if [ "$OS" = "windows" ]; then
        ext=".exe"
    fi

    asset="${bin_name}-${OS}-${ARCH}${ext}"
    base="${XBOARD_RELEASE_BASE_URL%/}"

    if [ "$XBOARD_RELEASE_TAG" = "latest" ]; then
        url="${base}/${XBOARD_RELEASE_REPO}/releases/latest/download/${asset}"
    else
        url="${base}/${XBOARD_RELEASE_REPO}/releases/download/${XBOARD_RELEASE_TAG}/${asset}"
    fi

    tmp_bin=$(mktemp)
    if [ -z "$tmp_bin" ]; then
        echo "Warning: failed to create temporary file for ${asset}."
        return 1
    fi

    if ! curl --fail --silent --show-error --location --output "$tmp_bin" "$url"; then
        rm -f "$tmp_bin"
        return 1
    fi

    if [ ! -s "$tmp_bin" ]; then
        echo "Warning: downloaded ${asset} is empty from ${url}."
        rm -f "$tmp_bin"
        return 1
    fi

    if ! install_executable_file "$tmp_bin" "$target_path"; then
        rm -f "$tmp_bin"
        return 1
    fi

    rm -f "$tmp_bin"
    echo "Installed ${bin_name} from release asset: ${url}"
    return 0
}

install_binary() {
    bin_name=$1
    cmd_path=$2

    ext=""
    target_bin="$bin_name"
    if [ "$OS" = "windows" ]; then
        ext=".exe"
        target_bin="${bin_name}.exe"
    fi

    arch_bin="${bin_name}-${OS}-${ARCH}${ext}"

    if ! ensure_install_dir; then
        exit 1
    fi

    if download_release_binary "$bin_name" "$INSTALL_DIR/$target_bin"; then
        return 0
    fi

    if [ "$XBOARD_RELEASE_DOWNLOAD_STRICT" = "1" ]; then
        echo "Error: failed to download ${bin_name} from GitHub release and strict mode is enabled."
        exit 1
    fi

    if [ -f "$target_bin" ]; then
        echo "Installing $target_bin from current directory..."
        if ! install_executable_file "$target_bin" "$INSTALL_DIR/$target_bin"; then
            exit 1
        fi
    elif [ "$target_bin" != "$bin_name" ] && [ -f "$bin_name" ]; then
        echo "Installing $bin_name from current directory..."
        if ! install_executable_file "$bin_name" "$INSTALL_DIR/$target_bin"; then
            exit 1
        fi
    elif [ -f "$arch_bin" ]; then
        echo "Installing $arch_bin from current directory..."
        if ! install_executable_file "$arch_bin" "$INSTALL_DIR/$target_bin"; then
            exit 1
        fi
    elif [ -f "$cmd_path" ]; then
        echo "Building $bin_name..."
        if ! command -v go >/dev/null 2>&1; then
            echo "Go not found, attempting to install..."
            if ! ensure_dependency "go"; then
                echo "Error: 'go' command not found and automatic installation failed."
                exit 1
            fi
        fi

        build_tmp=$(mktemp)
        if [ -z "$build_tmp" ]; then
            echo "Error: failed to create temporary file for build output."
            exit 1
        fi

        if ! go build -o "$build_tmp" "$cmd_path"; then
            rm -f "$build_tmp"
            exit 1
        fi

        if ! install_executable_file "$build_tmp" "$INSTALL_DIR/$target_bin"; then
            rm -f "$build_tmp"
            exit 1
        fi

        rm -f "$build_tmp"
    else
        echo "Error: ${bin_name} binary or source not found."
        echo "Please run this script from the project root or provide the compiled binary."
        exit 1
    fi
}

resolve_service_file() {
    service_name=$1

    if [ "$service_name" = "agent.service" ] && [ -n "${XBOARD_AGENT_SERVICE_FILE:-}" ] && [ -f "${XBOARD_AGENT_SERVICE_FILE}" ]; then
        echo "${XBOARD_AGENT_SERVICE_FILE}"
        return 0
    fi

    if [ "$service_name" = "xboard.service" ] && [ -n "${XBOARD_PANEL_SERVICE_FILE:-}" ] && [ -f "${XBOARD_PANEL_SERVICE_FILE}" ]; then
        echo "${XBOARD_PANEL_SERVICE_FILE}"
        return 0
    fi

    if [ -n "${XBOARD_SERVICE_FILE:-}" ] && [ -f "${XBOARD_SERVICE_FILE}" ]; then
        echo "${XBOARD_SERVICE_FILE}"
        return 0
    fi

    if [ -f "${SCRIPT_DIR}/${service_name}" ]; then
        echo "${SCRIPT_DIR}/${service_name}"
    elif [ -f "deploy/${service_name}" ]; then
        echo "deploy/${service_name}"
    elif [ -f "${service_name}" ]; then
        echo "${service_name}"
    fi

    return 0
}
