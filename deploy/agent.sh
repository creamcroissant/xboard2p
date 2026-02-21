#!/bin/sh
set -e

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

INSTALL_DIR="${INSTALL_DIR:-/opt/xboard}"
SKIP_SYSTEMD="${XBOARD_INSTALL_SKIP_SYSTEMD:-0}"

USER_XBOARD_RELEASE_REPO="${XBOARD_RELEASE_REPO:-}"
USER_XBOARD_RELEASE_TAG="${XBOARD_RELEASE_TAG:-}"
USER_XBOARD_RELEASE_BASE_URL="${XBOARD_RELEASE_BASE_URL:-}"

DEFAULT_XBOARD_RELEASE_REPO="${USER_XBOARD_RELEASE_REPO:-creamcroissant/xboard2p}"
DEFAULT_XBOARD_RELEASE_TAG="${USER_XBOARD_RELEASE_TAG:-latest}"
DEFAULT_XBOARD_RELEASE_BASE_URL="${USER_XBOARD_RELEASE_BASE_URL:-https://github.com}"
XBOARD_RELEASE_REPO="$DEFAULT_XBOARD_RELEASE_REPO"
XBOARD_RELEASE_TAG="$DEFAULT_XBOARD_RELEASE_TAG"
XBOARD_RELEASE_BASE_URL="$DEFAULT_XBOARD_RELEASE_BASE_URL"
# Deprecated compatibility flag. Release download is strict-only now.
: "${XBOARD_RELEASE_DOWNLOAD_STRICT:=1}"

XBOARD_BOOTSTRAP_REPO="${XBOARD_BOOTSTRAP_REPO:-creamcroissant/xboard2p}"
XBOARD_BOOTSTRAP_REF="${XBOARD_BOOTSTRAP_REF:-latest}"
XBOARD_BOOTSTRAP_RAW_BASE_URL="${XBOARD_BOOTSTRAP_RAW_BASE_URL:-https://raw.githubusercontent.com}"
XBOARD_BOOTSTRAP_RELEASE_BASE_URL="${XBOARD_BOOTSTRAP_RELEASE_BASE_URL:-https://github.com}"
XBOARD_BOOTSTRAP_API_BASE_URL="${XBOARD_BOOTSTRAP_API_BASE_URL:-https://api.github.com}"
XBOARD_AGENT_SCRIPT_URL="${XBOARD_AGENT_SCRIPT_URL:-}"
XBOARD_AGENT_SERVICE_URL="${XBOARD_AGENT_SERVICE_URL:-}"
XBOARD_BOOTSTRAP_KEEP_TEMP="${XBOARD_BOOTSTRAP_KEEP_TEMP:-0}"
# Deprecated compatibility flag. Bootstrap is strict-only now.
: "${XBOARD_BOOTSTRAP_DOWNLOAD_STRICT:=1}"

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

download_release_binary() {
    bin_name=$1
    target_path=$2

    ext=""
    if [ "$OS" = "windows" ]; then
        ext=".exe"
    fi

    asset="${bin_name}-${OS}-${ARCH}${ext}"
    base="${XBOARD_RELEASE_BASE_URL%/}"

    if [ "$XBOARD_RELEASE_TAG" = "latest" ]; then
        url="${base}/${XBOARD_RELEASE_REPO}/releases/latest/download/${asset}"
        checksum_url="${base}/${XBOARD_RELEASE_REPO}/releases/latest/download/SHA256SUMS.txt"
    else
        url="${base}/${XBOARD_RELEASE_REPO}/releases/download/${XBOARD_RELEASE_TAG}/${asset}"
        checksum_url="${base}/${XBOARD_RELEASE_REPO}/releases/download/${XBOARD_RELEASE_TAG}/SHA256SUMS.txt"
    fi

    if ! command -v curl >/dev/null 2>&1; then
        echo "Error: curl not found for release download of ${bin_name}."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} os=${OS} arch=${ARCH} url=${url}"
        return 1
    fi

    if ! has_ca_certificates; then
        echo "Error: CA certificates not found for release download of ${bin_name}."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} os=${OS} arch=${ARCH} url=${url}"
        return 1
    fi

    tmp_bin=$(mktemp)
    if [ -z "$tmp_bin" ]; then
        echo "Error: failed to create temporary file for ${asset}."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} os=${OS} arch=${ARCH} url=${url}"
        return 1
    fi

    tmp_checksums=$(mktemp)
    if [ -z "$tmp_checksums" ]; then
        echo "Error: failed to create temporary file for checksum manifest."
        rm -f "$tmp_bin"
        return 1
    fi

    if ! curl --fail --silent --show-error --location --retry 3 --retry-delay 1 --output "$tmp_bin" "$url"; then
        echo "Error: failed to download release asset ${asset}."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} os=${OS} arch=${ARCH} url=${url}"
        rm -f "$tmp_bin" "$tmp_checksums"
        return 1
    fi

    if [ ! -s "$tmp_bin" ]; then
        echo "Error: downloaded ${asset} is empty."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} os=${OS} arch=${ARCH} url=${url}"
        rm -f "$tmp_bin" "$tmp_checksums"
        return 1
    fi

    if ! curl --fail --silent --show-error --location --retry 3 --retry-delay 1 --output "$tmp_checksums" "$checksum_url"; then
        echo "Error: failed to download checksum manifest SHA256SUMS.txt."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} os=${OS} arch=${ARCH} checksum_url=${checksum_url}"
        rm -f "$tmp_bin" "$tmp_checksums"
        return 1
    fi

    if [ ! -s "$tmp_checksums" ]; then
        echo "Error: downloaded checksum manifest is empty."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} checksum_url=${checksum_url}"
        rm -f "$tmp_bin" "$tmp_checksums"
        return 1
    fi

    if ! verify_checksum "$asset" "$tmp_bin" "$tmp_checksums"; then
        echo "Error: checksum verification failed for release asset ${asset}."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} checksum_url=${checksum_url}"
        rm -f "$tmp_bin" "$tmp_checksums"
        return 1
    fi

    if ! install_executable_file "$tmp_bin" "$target_path"; then
        echo "Error: failed to install ${asset} into ${target_path}."
        echo "repo=${XBOARD_RELEASE_REPO} tag=${XBOARD_RELEASE_TAG} os=${OS} arch=${ARCH} url=${url}"
        rm -f "$tmp_bin" "$tmp_checksums"
        return 1
    fi

    rm -f "$tmp_bin" "$tmp_checksums"
    echo "Installed ${bin_name} from release asset: ${url}"
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

    if ! download_release_binary "$bin_name" "$INSTALL_DIR/$target_bin"; then
        echo "Error: failed to install ${bin_name} from GitHub release asset."
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

require_command() {
    cmd_name=$1
    if ! command -v "$cmd_name" >/dev/null 2>&1; then
        echo "Error: required command '${cmd_name}' not found."
        return 1
    fi
    return 0
}

download_file() {
    source_url=$1
    target_path=$2
    label=$3

    if ! curl --fail --silent --show-error --location --retry 3 --retry-delay 1 --output "$target_path" "$source_url"; then
        echo "Error: failed to download ${label} from ${source_url}."
        return 1
    fi

    if [ ! -s "$target_path" ]; then
        echo "Error: downloaded ${label} is empty."
        return 1
    fi

    return 0
}

is_commit_ref() {
    ref_value=$1
    if printf '%s' "$ref_value" | grep -Eq '^[0-9a-fA-F]{7,40}$'; then
        return 0
    fi
    return 1
}

resolve_latest_tag() {
    latest_meta_file=$1
    api_url="${XBOARD_BOOTSTRAP_API_BASE_URL%/}/repos/${XBOARD_BOOTSTRAP_REPO}/releases/latest"

    if ! download_file "$api_url" "$latest_meta_file" "latest release metadata"; then
        return 1
    fi

    latest_tag=$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$latest_meta_file" | head -n 1)
    if [ -z "$latest_tag" ]; then
        echo "Error: failed to parse latest release tag from ${api_url}."
        return 1
    fi

    printf '%s' "$latest_tag"
    return 0
}

run_bootstrap_mode() {
    if ! require_command curl; then
        return 1
    fi

    WORKDIR=$(mktemp -d 2>/dev/null || mktemp -d -t xboard-agent-bootstrap)
    if [ -z "$WORKDIR" ]; then
        echo "Error: failed to create temporary working directory."
        return 1
    fi

    cleanup_bootstrap() {
        if [ "$XBOARD_BOOTSTRAP_KEEP_TEMP" = "1" ]; then
            echo "Bootstrap temp directory retained: ${WORKDIR}"
            return
        fi

        rm -rf "$WORKDIR"
    }
    trap cleanup_bootstrap EXIT INT TERM

    LATEST_META_FILE="$WORKDIR/latest-release.json"
    if [ "$XBOARD_BOOTSTRAP_REF" = "latest" ]; then
        RESOLVED_REF=$(resolve_latest_tag "$LATEST_META_FILE")
        DEFAULT_RELEASE_TAG="$RESOLVED_REF"
    elif is_commit_ref "$XBOARD_BOOTSTRAP_REF"; then
        RESOLVED_REF="$XBOARD_BOOTSTRAP_REF"
        DEFAULT_RELEASE_TAG=""
    else
        RESOLVED_REF="$XBOARD_BOOTSTRAP_REF"
        DEFAULT_RELEASE_TAG="$RESOLVED_REF"
    fi

    if [ -n "$USER_XBOARD_RELEASE_TAG" ]; then
        RELEASE_TAG_TO_USE="$USER_XBOARD_RELEASE_TAG"
    else
        RELEASE_TAG_TO_USE="$DEFAULT_RELEASE_TAG"
    fi

    if [ -z "$RELEASE_TAG_TO_USE" ]; then
        echo "Error: bootstrap ref '${RESOLVED_REF}' looks like a commit hash."
        echo "Please set XBOARD_RELEASE_TAG to a release tag to keep script and binary versions consistent."
        return 1
    fi

    RAW_BASE="${XBOARD_BOOTSTRAP_RAW_BASE_URL%/}"
    RELEASE_BASE="${XBOARD_BOOTSTRAP_RELEASE_BASE_URL%/}"

    if [ -n "$XBOARD_AGENT_SCRIPT_URL" ]; then
        AGENT_URL="$XBOARD_AGENT_SCRIPT_URL"
    else
        AGENT_URL="${RAW_BASE}/${XBOARD_BOOTSTRAP_REPO}/${RESOLVED_REF}/deploy/agent.sh"
    fi
    if [ -n "$XBOARD_AGENT_SERVICE_URL" ]; then
        SERVICE_URL="$XBOARD_AGENT_SERVICE_URL"
    else
        SERVICE_URL="${RAW_BASE}/${XBOARD_BOOTSTRAP_REPO}/${RESOLVED_REF}/deploy/agent.service"
    fi

    CHECKSUM_URL="${RELEASE_BASE}/${XBOARD_BOOTSTRAP_REPO}/releases/download/${RELEASE_TAG_TO_USE}/SHA256SUMS.txt"

    if ! download_file "$AGENT_URL" "$WORKDIR/agent.sh" "agent.sh"; then
        return 1
    fi

    if ! download_file "$SERVICE_URL" "$WORKDIR/agent.service" "agent.service"; then
        echo "Error: failed to download agent.service from ${SERVICE_URL}."
        return 1
    fi

    if ! download_file "$CHECKSUM_URL" "$WORKDIR/checksums.txt" "checksum manifest"; then
        echo "Error: checksum manifest download failed: ${CHECKSUM_URL}"
        return 1
    fi

    if ! verify_checksum "agent.sh" "$WORKDIR/agent.sh" "$WORKDIR/checksums.txt"; then
        return 1
    fi
    if ! verify_checksum "agent.service" "$WORKDIR/agent.service" "$WORKDIR/checksums.txt"; then
        echo "Error: checksum verification failed for agent.service."
        return 1
    fi

    if [ -n "$USER_XBOARD_RELEASE_REPO" ]; then
        XBOARD_RELEASE_REPO="$USER_XBOARD_RELEASE_REPO"
    else
        XBOARD_RELEASE_REPO="$XBOARD_BOOTSTRAP_REPO"
    fi

    if [ -n "$USER_XBOARD_RELEASE_BASE_URL" ]; then
        XBOARD_RELEASE_BASE_URL="$USER_XBOARD_RELEASE_BASE_URL"
    else
        XBOARD_RELEASE_BASE_URL="$XBOARD_BOOTSTRAP_RELEASE_BASE_URL"
    fi

    XBOARD_RELEASE_TAG="$RELEASE_TAG_TO_USE"
    export XBOARD_RELEASE_REPO XBOARD_RELEASE_BASE_URL XBOARD_RELEASE_TAG

    XBOARD_AGENT_SERVICE_FILE="$WORKDIR/agent.service"
    export XBOARD_AGENT_SERVICE_FILE

    chmod +x "$WORKDIR/agent.sh"

    echo "Running agent installer (ref=${RESOLVED_REF}, release_tag=${XBOARD_RELEASE_TAG})..."
    (
        cd "$WORKDIR"
        sh ./agent.sh "$@"
    )
}

HOST_TOKEN="${XBOARD_AGENT_HOST_TOKEN:-}"
GRPC_ADDRESS="${XBOARD_AGENT_GRPC_ADDRESS:-}"
GRPC_TLS_ENABLED="${XBOARD_AGENT_GRPC_TLS_ENABLED:-false}"
TRAFFIC_TYPE="${XBOARD_AGENT_TRAFFIC_TYPE:-netio}"
FORCE_CONFIG_OVERWRITE="${XBOARD_AGENT_CONFIG_OVERWRITE:-0}"
BOOTSTRAP_MODE=0
UNINSTALL_MODE=0
INSTALL_SEMANTIC_ARGS_USED=0

print_usage() {
    cat <<'EOF'
Usage: sh agent.sh [options] [-- <agent install args>]

Options:
  --host-token <token>          Agent host token
  --grpc-address <address>      Panel gRPC address, e.g. 10.0.0.2:9090
  --grpc-tls-enabled <bool>     true/false, default false
  --traffic-type <type>         traffic.type, default netio
  --force-config-overwrite      overwrite existing agent_config.yml
  --bootstrap                   run bootstrap mode (download+verify+install)
  --uninstall                   remove agent artifacts managed by this script
  --ref <latest|tag|commit>     bootstrap source ref (default: latest)
  --repo <owner/repo>           bootstrap source repository
  --service-url <url>           bootstrap service URL override
  -h, --help                    show this help message

Environment:
  XBOARD_AGENT_HOST_TOKEN
  XBOARD_AGENT_GRPC_ADDRESS
  XBOARD_AGENT_GRPC_TLS_ENABLED
  XBOARD_AGENT_TRAFFIC_TYPE
  XBOARD_AGENT_CONFIG_OVERWRITE=1

  Bootstrap-related:
  XBOARD_BOOTSTRAP_REF, XBOARD_BOOTSTRAP_REPO
  XBOARD_BOOTSTRAP_RAW_BASE_URL, XBOARD_BOOTSTRAP_RELEASE_BASE_URL
  XBOARD_BOOTSTRAP_API_BASE_URL
  XBOARD_BOOTSTRAP_DOWNLOAD_STRICT (deprecated, bootstrap is strict-only)
  XBOARD_AGENT_SCRIPT_URL, XBOARD_AGENT_SERVICE_URL
EOF
}

normalize_bool() {
    case "$1" in
        1|true|TRUE|yes|YES)
            printf '%s' "true"
            ;;
        0|false|FALSE|no|NO)
            printf '%s' "false"
            ;;
        *)
            return 1
            ;;
    esac
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --host-token)
            if [ "$#" -lt 2 ]; then
                echo "Error: --host-token requires a value."
                exit 1
            fi
            HOST_TOKEN=$2
            INSTALL_SEMANTIC_ARGS_USED=1
            shift 2
            ;;
        --grpc-address)
            if [ "$#" -lt 2 ]; then
                echo "Error: --grpc-address requires a value."
                exit 1
            fi
            GRPC_ADDRESS=$2
            INSTALL_SEMANTIC_ARGS_USED=1
            shift 2
            ;;
        --grpc-tls-enabled)
            if [ "$#" -lt 2 ]; then
                echo "Error: --grpc-tls-enabled requires a value."
                exit 1
            fi
            GRPC_TLS_ENABLED=$2
            INSTALL_SEMANTIC_ARGS_USED=1
            shift 2
            ;;
        --traffic-type)
            if [ "$#" -lt 2 ]; then
                echo "Error: --traffic-type requires a value."
                exit 1
            fi
            TRAFFIC_TYPE=$2
            INSTALL_SEMANTIC_ARGS_USED=1
            shift 2
            ;;
        --force-config-overwrite)
            FORCE_CONFIG_OVERWRITE=1
            INSTALL_SEMANTIC_ARGS_USED=1
            shift
            ;;
        --bootstrap)
            BOOTSTRAP_MODE=1
            shift
            ;;
        --uninstall)
            UNINSTALL_MODE=1
            shift
            ;;
        --ref)
            if [ "$#" -lt 2 ]; then
                echo "Error: --ref requires a value."
                exit 1
            fi
            XBOARD_BOOTSTRAP_REF=$2
            INSTALL_SEMANTIC_ARGS_USED=1
            shift 2
            ;;
        --repo)
            if [ "$#" -lt 2 ]; then
                echo "Error: --repo requires a value."
                exit 1
            fi
            XBOARD_BOOTSTRAP_REPO=$2
            INSTALL_SEMANTIC_ARGS_USED=1
            shift 2
            ;;
        --service-url)
            if [ "$#" -lt 2 ]; then
                echo "Error: --service-url requires a value."
                exit 1
            fi
            XBOARD_AGENT_SERVICE_URL=$2
            INSTALL_SEMANTIC_ARGS_USED=1
            shift 2
            ;;
        -h|--help)
            print_usage
            exit 0
            ;;
        --)
            shift
            break
            ;;
        *)
            echo "Error: unknown argument: $1"
            print_usage
            exit 1
            ;;
    esac
done

if [ "$BOOTSTRAP_MODE" = "1" ] && [ "$UNINSTALL_MODE" = "1" ]; then
    echo "Error: --bootstrap and --uninstall cannot be used together."
    exit 1
fi

if [ "$UNINSTALL_MODE" = "1" ]; then
    if [ "$INSTALL_SEMANTIC_ARGS_USED" = "1" ]; then
        echo "Error: --uninstall cannot be combined with install/bootstrap parameters."
        exit 1
    fi

    echo "=== Uninstalling Agent ==="

    if is_systemd_available; then
        run_privileged systemctl disable --now xboard-agent >/dev/null 2>&1 || true
    else
        echo "Systemd is not available on this host. Skipping systemctl operations for xboard-agent."
    fi

    run_privileged rm -f /etc/systemd/system/xboard-agent.service || true

    if is_systemd_available; then
        if ! run_privileged systemctl daemon-reload; then
            echo "Error: failed to run systemctl daemon-reload."
            exit 1
        fi
    fi

    run_privileged rm -f "$INSTALL_DIR/agent" || true
    run_privileged rm -f "$INSTALL_DIR/agent_config.yml" || true

    echo "Agent uninstall completed."
    exit 0
fi

if [ "$BOOTSTRAP_MODE" = "1" ]; then
    if ! run_bootstrap_mode "$@"; then
        exit 1
    fi
    exit 0
fi

echo "=== Installing Agent ==="

GRPC_TLS_ENABLED_NORMALIZED=$(normalize_bool "$GRPC_TLS_ENABLED" || true)
if [ -z "$GRPC_TLS_ENABLED_NORMALIZED" ]; then
    echo "Error: invalid grpc tls flag '${GRPC_TLS_ENABLED}'. Expected true/false."
    exit 1
fi
GRPC_TLS_ENABLED=$GRPC_TLS_ENABLED_NORMALIZED

if ! ensure_install_dir; then
    exit 1
fi

if ! ensure_download_dependencies; then
    echo "Error: release download dependency check failed for agent."
    exit 1
fi

install_binary "agent" "./cmd/agent/main.go"

CONFIG_PATH="$INSTALL_DIR/agent_config.yml"
if [ -f "$CONFIG_PATH" ] && [ "$FORCE_CONFIG_OVERWRITE" != "1" ]; then
    echo "agent_config.yml already exists. Keep existing file (use --force-config-overwrite to overwrite)."
else
    if [ -z "$HOST_TOKEN" ] || [ -z "$GRPC_ADDRESS" ]; then
        echo "Error: missing required config parameters."
        echo "Both host token and grpc address are required to initialize agent_config.yml."
        echo "Example:"
        echo "  sh ./deploy/agent.sh --host-token '<token>' --grpc-address '127.0.0.1:9090'"
        echo "  or set XBOARD_AGENT_HOST_TOKEN and XBOARD_AGENT_GRPC_ADDRESS"
        exit 1
    fi

    umask 077
    cat > "$CONFIG_PATH" <<EOF
panel:
  host_token: "${HOST_TOKEN}"

grpc:
  enabled: true
  address: "${GRPC_ADDRESS}"
  tls:
    enabled: ${GRPC_TLS_ENABLED}

traffic:
  type: "${TRAFFIC_TYPE}"
EOF
    echo "Initialized agent_config.yml at ${CONFIG_PATH}."
fi

if [ "$SKIP_SYSTEMD" = "1" ]; then
    echo "Skipping xboard-agent.service installation (XBOARD_INSTALL_SKIP_SYSTEMD=1)."
elif ! is_systemd_available; then
    echo "Systemd is not available on this host. Please manage agent process manually."
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
