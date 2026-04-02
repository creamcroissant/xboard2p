#!/bin/sh
set -e

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

DEFAULT_INSTALL_DIR="/opt/xboard/panel"
INSTALL_DIR="${INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"
FRONTEND_RELEASE_ASSET="${XBOARD_FRONTEND_RELEASE_ASSET:-frontend-dist.tar.gz}"
INSTALL_UI_RELEASE_ASSET="${XBOARD_INSTALL_UI_RELEASE_ASSET:-install-ui.tar.gz}"
SKIP_SYSTEMD="${XBOARD_INSTALL_SKIP_SYSTEMD:-0}"

XBOARD_RELEASE_REPO="${XBOARD_RELEASE_REPO:-creamcroissant/xboard2p}"
XBOARD_RELEASE_TAG="${XBOARD_RELEASE_TAG:-latest}"
XBOARD_RELEASE_BASE_URL="${XBOARD_RELEASE_BASE_URL:-https://github.com}"
# Deprecated compatibility flag. Release download is strict-only now.
: "${XBOARD_RELEASE_DOWNLOAD_STRICT:=1}"

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

DISTRO_ID=""
DISTRO_ID_LIKE=""
PKG_MANAGER=""
PKG_CACHE_UPDATED=0
OPENRC_SERVICE_CMD=""
OPENRC_UPDATE_CMD=""

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
        curl)
            command -v curl >/dev/null 2>&1
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

ensure_download_dependencies() {
    if ! ensure_dependency "curl"; then
        return 1
    fi

    if ! ensure_dependency "ca-certificates"; then
        return 1
    fi

    if ! ensure_dependency "tar"; then
        return 1
    fi

    return 0
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

resolve_openrc_commands() {
    if [ -n "$OPENRC_SERVICE_CMD" ] && [ -n "$OPENRC_UPDATE_CMD" ]; then
        return 0
    fi

    OPENRC_SERVICE_CMD=""
    OPENRC_UPDATE_CMD=""

    for candidate in rc-service /sbin/rc-service /usr/sbin/rc-service; do
        if [ "$candidate" = "rc-service" ]; then
            if command -v rc-service >/dev/null 2>&1; then
                OPENRC_SERVICE_CMD=$(command -v rc-service)
                break
            fi
        elif [ -x "$candidate" ]; then
            OPENRC_SERVICE_CMD="$candidate"
            break
        fi
    done

    for candidate in rc-update /sbin/rc-update /usr/sbin/rc-update; do
        if [ "$candidate" = "rc-update" ]; then
            if command -v rc-update >/dev/null 2>&1; then
                OPENRC_UPDATE_CMD=$(command -v rc-update)
                break
            fi
        elif [ -x "$candidate" ]; then
            OPENRC_UPDATE_CMD="$candidate"
            break
        fi
    done

    if [ -n "$OPENRC_SERVICE_CMD" ] && [ -n "$OPENRC_UPDATE_CMD" ]; then
        return 0
    fi

    return 1
}

is_openrc_available() {
    if [ "$SKIP_SYSTEMD" = "1" ]; then
        return 1
    fi

    if ! resolve_openrc_commands; then
        return 1
    fi
    return 0
}

render_install_service_file() {
    source_path=$1
    target_path=$2

    temp_service=$(mktemp)
    if [ -z "$temp_service" ]; then
        echo "Error: failed to create temporary service file."
        return 1
    fi

    escaped_install_dir=$(printf '%s' "$INSTALL_DIR" | sed 's/[&#\\]/\\&/g')
    if ! sed "s#/opt/xboard#${escaped_install_dir}#g" "$source_path" > "$temp_service"; then
        echo "Error: failed to render service file ${source_path}."
        rm -f "$temp_service"
        return 1
    fi

    if ! run_privileged cp "$temp_service" "$target_path"; then
        echo "Error: failed to install rendered service file ${target_path}."
        rm -f "$temp_service"
        return 1
    fi

    rm -f "$temp_service"
    return 0
}

install_openrc_service() {
    service_name=$1
    binary_path=$2
    command_args=$3

    init_script_path="/etc/init.d/${service_name}"
    temp_script=$(mktemp)
    if [ -z "$temp_script" ]; then
        echo "Error: failed to create temporary OpenRC service file."
        return 1
    fi

    cat > "$temp_script" <<EOF
#!/sbin/openrc-run
name="${service_name}"
description="xboard ${service_name} service"
directory="${INSTALL_DIR}"
command="${binary_path}"
command_args="${command_args}"
pidfile="/run/${service_name}.pid"
command_background=true

depend() {
    need net
}
EOF

    if ! run_privileged cp "$temp_script" "$init_script_path"; then
        echo "Error: failed to install OpenRC service script ${init_script_path}."
        rm -f "$temp_script"
        return 1
    fi

    if ! run_privileged chmod +x "$init_script_path"; then
        echo "Error: failed to set executable permission on ${init_script_path}."
        rm -f "$temp_script"
        return 1
    fi

    rm -f "$temp_script"

    if ! run_privileged "$OPENRC_UPDATE_CMD" add "$service_name" default; then
        echo "Error: failed to register OpenRC service ${service_name}."
        return 1
    fi

    echo "${service_name} OpenRC service installed."
    return 0
}

install_release_asset() {
    asset_name=$1
    target_path=$2

    base="${XBOARD_RELEASE_BASE_URL%/}"

    if [ "$XBOARD_RELEASE_TAG" = "latest" ]; then
        url="${base}/${XBOARD_RELEASE_REPO}/releases/latest/download/${asset_name}"
        checksum_url="${base}/${XBOARD_RELEASE_REPO}/releases/latest/download/SHA256SUMS.txt"
    else
        url="${base}/${XBOARD_RELEASE_REPO}/releases/download/${XBOARD_RELEASE_TAG}/${asset_name}"
        checksum_url="${base}/${XBOARD_RELEASE_REPO}/releases/download/${XBOARD_RELEASE_TAG}/SHA256SUMS.txt"
    fi

    if ! command -v curl >/dev/null 2>&1; then
        echo "Error: curl not found for release download of ${asset_name}."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} os=${OS} arch=${ARCH} url=${url}"
        return 1
    fi

    if ! has_ca_certificates; then
        echo "Error: CA certificates not found for release download of ${asset_name}."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} os=${OS} arch=${ARCH} url=${url}"
        return 1
    fi

    download_tmp_asset=$(mktemp)
    if [ -z "$download_tmp_asset" ]; then
        echo "Error: failed to create temporary file for ${asset_name}."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} os=${OS} arch=${ARCH} url=${url}"
        return 1
    fi

    download_tmp_checksums=$(mktemp)
    if [ -z "$download_tmp_checksums" ]; then
        echo "Error: failed to create temporary file for checksum manifest."
        rm -f "$download_tmp_asset"
        return 1
    fi

    if ! curl --fail --silent --show-error --location --retry 3 --retry-delay 1 --output "$download_tmp_asset" "$url"; then
        echo "Error: failed to download release asset ${asset_name}."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} os=${OS} arch=${ARCH} url=${url}"
        rm -f "$download_tmp_asset" "$download_tmp_checksums"
        return 1
    fi

    if [ ! -s "$download_tmp_asset" ]; then
        echo "Error: downloaded ${asset_name} is empty."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} os=${OS} arch=${ARCH} url=${url}"
        rm -f "$download_tmp_asset" "$download_tmp_checksums"
        return 1
    fi

    if ! curl --fail --silent --show-error --location --retry 3 --retry-delay 1 --output "$download_tmp_checksums" "$checksum_url"; then
        echo "Error: failed to download checksum manifest SHA256SUMS.txt."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} os=${OS} arch=${ARCH} checksum_url=${checksum_url}"
        rm -f "$download_tmp_asset" "$download_tmp_checksums"
        return 1
    fi

    if [ ! -s "$download_tmp_checksums" ]; then
        echo "Error: downloaded checksum manifest is empty."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} checksum_url=${checksum_url}"
        rm -f "$download_tmp_asset" "$download_tmp_checksums"
        return 1
    fi

    if ! verify_checksum "$asset_name" "$download_tmp_asset" "$download_tmp_checksums"; then
        echo "Error: checksum verification failed for release asset ${asset_name}."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} checksum_url=${checksum_url}"
        rm -f "$download_tmp_asset" "$download_tmp_checksums"
        return 1
    fi

    if ! install_file "$download_tmp_asset" "$target_path"; then
        echo "Error: failed to install ${asset_name} into ${target_path}."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} os=${OS} arch=${ARCH} url=${url}"
        rm -f "$download_tmp_asset" "$download_tmp_checksums"
        return 1
    fi

    rm -f "$download_tmp_asset" "$download_tmp_checksums"
    echo "Installed release asset: ${url}"
    return 0
}

install_release_archive_dir() {
    asset_name=$1
    extract_parent=$2
    extracted_dir_name=$3
    target_dir=$4

    archive_tmp_asset=$(mktemp)
    if [ -z "$archive_tmp_asset" ]; then
        echo "Error: failed to create temporary file for ${asset_name}."
        return 1
    fi

    archive_tmp_extract=$(mktemp -d)
    if [ -z "$archive_tmp_extract" ]; then
        echo "Error: failed to create temporary directory for ${asset_name}."
        rm -f "$archive_tmp_asset"
        return 1
    fi

    if ! install_release_asset "$asset_name" "$archive_tmp_asset"; then
        rm -f "$archive_tmp_asset"
        rm -rf "$archive_tmp_extract"
        return 1
    fi

    if ! tar -C "$archive_tmp_extract" -xzf "$archive_tmp_asset"; then
        echo "Error: failed to extract ${asset_name}."
        rm -f "$archive_tmp_asset"
        rm -rf "$archive_tmp_extract"
        return 1
    fi

    extracted_path="${archive_tmp_extract}/${extracted_dir_name}"
    if [ ! -d "$extracted_path" ]; then
        echo "Error: extracted directory ${extracted_dir_name} not found in ${asset_name}."
        rm -f "$archive_tmp_asset"
        rm -rf "$archive_tmp_extract"
        return 1
    fi

    if ! ensure_dir "$extract_parent"; then
        echo "Error: failed to create install directory ${extract_parent}."
        rm -f "$archive_tmp_asset"
        rm -rf "$archive_tmp_extract"
        return 1
    fi

    run_privileged rm -rf "$target_dir"
    if ! run_privileged mv "$extracted_path" "$target_dir"; then
        echo "Error: failed to install extracted directory ${target_dir}."
        rm -f "$archive_tmp_asset"
        rm -rf "$archive_tmp_extract"
        return 1
    fi

    rm -f "$archive_tmp_asset"
    rm -rf "$archive_tmp_extract"
    echo "Installed ${asset_name} into ${target_dir}"
    return 0
}

hash_file_sha256() {
    target_path=$1

    if command -v sha256sum >/dev/null 2>&1; then
        set -- $(sha256sum "$target_path")
        printf '%s' "$1"
        return 0
    fi

    if command -v shasum >/dev/null 2>&1; then
        set -- $(shasum -a 256 "$target_path")
        printf '%s' "$1"
        return 0
    fi

    if command -v openssl >/dev/null 2>&1; then
        if openssl_output=$(openssl dgst -sha256 "$target_path" 2>/dev/null); then
            printf '%s' "${openssl_output##*= }"
            return 0
        fi
    fi

    echo "Error: no SHA256 tool found (requires sha256sum, shasum, or openssl)."
    return 1
}

lookup_expected_checksum() {
    wanted_name=$1
    checksum_file=$2

    while IFS= read -r line || [ -n "$line" ]; do
        case "$line" in
            ''|'#'*)
                continue
                ;;
        esac

        set -- $line
        checksum=$1
        listed_name=${2#*}
        listed_name=$(printf '%s' "$listed_name" | tr -d '\r')

        case "$listed_name" in
            "${wanted_name}"|"deploy/${wanted_name}"|"dist/release/${wanted_name}"|"./${wanted_name}"|"*/${wanted_name}")
                printf '%s' "$checksum"
                return 0
                ;;
        esac
    done < "$checksum_file"

    return 1
}

verify_checksum() {
    file_name=$1
    file_path=$2
    checksum_file=$3

    expected_checksum=$(lookup_expected_checksum "$file_name" "$checksum_file" || true)
    if [ -z "$expected_checksum" ]; then
        echo "Error: checksum entry not found for ${file_name}."
        return 1
    fi

    actual_checksum=$(hash_file_sha256 "$file_path")
    if [ "$actual_checksum" != "$expected_checksum" ]; then
        echo "Error: checksum mismatch for ${file_name}."
        echo "Expected: ${expected_checksum}"
        echo "Actual:   ${actual_checksum}"
        return 1
    fi

    return 0
}

install_binary() {
    bin_name=$1
    _cmd_path=$2

    target_bin="$bin_name"
    if [ "$OS" = "windows" ]; then
        target_bin="${bin_name}.exe"
    fi

    if ! ensure_install_dir; then
        exit 1
    fi

    asset_name="${target_bin}-${OS}-${ARCH}"
    if [ "$OS" = "windows" ]; then
        asset_name="${bin_name}-${OS}-${ARCH}.exe"
    else
        asset_name="${bin_name}-${OS}-${ARCH}"
    fi

    if ! install_release_asset "$asset_name" "$INSTALL_DIR/$target_bin"; then
        echo "Error: failed to install ${bin_name} from GitHub release asset."
        exit 1
    fi

    if ! set_file_mode +x "$INSTALL_DIR/$target_bin"; then
        echo "Error: failed to set executable permission on $INSTALL_DIR/$target_bin."
        exit 1
    fi

    return 0
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

write_default_service_template() {
    service_name=$1
    temp_service=$(mktemp)
    if [ -z "$temp_service" ]; then
        echo "Error: failed to create temporary service template."
        return 1
    fi

    case "$service_name" in
        xboard.service)
            cat > "$temp_service" <<'EOF'
[Unit]
Description=XBoard Panel Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/xboard/panel
EnvironmentFile=-/etc/default/xboard
ExecStart=/opt/xboard/panel/xboard serve --config /opt/xboard/panel/config.yml
Restart=always
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF
            ;;
        *)
            rm -f "$temp_service"
            return 1
            ;;
    esac

    echo "$temp_service"
    return 0
}

print_usage() {
    cat <<'EOF'
Usage: sh panel.sh [options]

Options:
  --uninstall    remove panel artifacts managed by this script
  -h, --help     show this help message
EOF
}

run_uninstall_mode() {
    echo "=== Uninstalling Panel ==="

    has_service_manager=0

    if is_systemd_available; then
        has_service_manager=1
        run_privileged systemctl disable --now xboard >/dev/null 2>&1 || true
    else
        echo "Systemd is not available on this host. Skipping systemctl operations for xboard."
    fi

    if is_openrc_available; then
        has_service_manager=1
        run_privileged "$OPENRC_SERVICE_CMD" xboard stop >/dev/null 2>&1 || true
        run_privileged "$OPENRC_UPDATE_CMD" del xboard default >/dev/null 2>&1 || run_privileged "$OPENRC_UPDATE_CMD" del xboard >/dev/null 2>&1 || true
    fi

    if [ "$has_service_manager" = "0" ]; then
        echo "No supported service manager detected. Removed files only."
    fi

    run_privileged rm -f /etc/systemd/system/xboard.service || true
    run_privileged rm -f /etc/init.d/xboard || true

    if is_systemd_available; then
        if ! run_privileged systemctl daemon-reload; then
            echo "Error: failed to run systemctl daemon-reload."
            return 1
        fi
    fi

    run_privileged rm -f "$INSTALL_DIR/xboard" || true
    run_privileged rm -rf "$INSTALL_DIR/web/user-vite/dist" || true
    run_privileged rm -rf "$INSTALL_DIR/web/install" || true
    run_privileged rm -f "$INSTALL_DIR/config.yml" || true
    run_privileged rm -f "$INSTALL_DIR/.env" || true

    echo "Panel uninstall completed."
    return 0
}

UNINSTALL_MODE=0

while [ "$#" -gt 0 ]; do
    case "$1" in
        --uninstall)
            UNINSTALL_MODE=1
            shift
            ;;
        -h|--help)
            print_usage
            exit 0
            ;;
        *)
            echo "Error: unknown argument: $1"
            print_usage
            exit 1
            ;;
    esac
done

if [ "$UNINSTALL_MODE" = "1" ]; then
    if ! run_uninstall_mode; then
        exit 1
    fi
    exit 0
fi

echo "=== Installing Panel ==="

if ! ensure_install_dir; then
    exit 1
fi

if ! ensure_download_dependencies; then
    echo "Error: release download dependency check failed for xboard."
    exit 1
fi

install_binary "xboard" "./cmd/xboard/main.go"

if ! install_release_archive_dir "$FRONTEND_RELEASE_ASSET" "$INSTALL_DIR/web/user-vite" "dist" "$INSTALL_DIR/web/user-vite/dist"; then
    echo "Error: failed to install frontend assets from GitHub release asset."
    exit 1
fi

if ! install_release_archive_dir "$INSTALL_UI_RELEASE_ASSET" "$INSTALL_DIR/web" "install" "$INSTALL_DIR/web/install"; then
    echo "Error: failed to install install UI assets from GitHub release asset."
    exit 1
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
elif is_systemd_available; then
    SERVICE_FILE=$(resolve_service_file "xboard.service")
    TEMPLATE_SOURCE="$SERVICE_FILE"
    TEMP_TEMPLATE=""
    if [ -z "$TEMPLATE_SOURCE" ]; then
        if TEMP_TEMPLATE=$(write_default_service_template "xboard.service"); then
            TEMPLATE_SOURCE="$TEMP_TEMPLATE"
            echo "xboard.service template not found; using embedded default template."
        fi
    fi
    if [ -n "$TEMPLATE_SOURCE" ]; then
        if ! render_install_service_file "$TEMPLATE_SOURCE" /etc/systemd/system/xboard.service; then
            if [ -n "$TEMP_TEMPLATE" ]; then
                rm -f "$TEMP_TEMPLATE"
            fi
            echo "Error: failed to install xboard.service."
            exit 1
        fi
        if [ -n "$TEMP_TEMPLATE" ]; then
            rm -f "$TEMP_TEMPLATE"
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
        echo "Warning: deploy/xboard.service not found and embedded template generation failed."
    fi
elif is_openrc_available; then
    if ! install_openrc_service "xboard" "${INSTALL_DIR}/xboard" "serve --config ${INSTALL_DIR}/config.yml"; then
        exit 1
    fi
else
    echo "No supported service manager detected (systemd/openrc). Please manage panel process manually."
fi

