#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

echo "=== Installing Agent ==="

install_binary_from_shortlink() {
    local endpoint="${CLOUDPASTE_API_ENDPOINT:-}"
    local direct_url="${XBOARD_AGENT_DOWNLOAD_URL:-}"

    if [ -z "$endpoint" ] && [ -z "$direct_url" ]; then
        return 1
    fi

    if ! command -v curl >/dev/null 2>&1; then
        echo "Warning: curl not found, skipping shortlink download."
        return 1
    fi

    local channel="${CLOUDPASTE_CHANNEL:-stable}"
    case "$channel" in
        stable|pre) ;;
        *)
            echo "Warning: invalid CLOUDPASTE_CHANNEL=$channel, fallback to stable."
            channel="stable"
            ;;
    esac

    local slug_prefix
    slug_prefix="$(printf '%s' "${CLOUDPASTE_SLUG_PREFIX:-xboard}" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//')"
    if [ -z "$slug_prefix" ]; then
        slug_prefix="xboard"
    fi

    local ext=""
    if [ "$OS" = "windows" ]; then
        ext=".exe"
    fi

    local asset="agent-${OS}-${ARCH}${ext}"
    local asset_slug_token
    asset_slug_token="$(printf '%s' "$asset" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9._-]+/-/g; s/[.]+/-/g; s/-+/-/g; s/^-+//; s/-+$//')"

    local expected_slug="${slug_prefix}-${channel}-${asset_slug_token}"
    local alt_channel="stable"
    if [ "$channel" = "stable" ]; then
        alt_channel="pre"
    fi

    local allow_drift="${CLOUDPASTE_ALLOW_CHANNEL_DRIFT:-true}"
    local slugs=("$expected_slug")
    case "$allow_drift" in
        1|true|TRUE|yes|YES)
            slugs+=("${slug_prefix}-${alt_channel}-${asset_slug_token}")
            ;;
    esac

    local urls=()
    if [ -n "$direct_url" ]; then
        urls+=("$direct_url")
    fi

    if [ -n "$endpoint" ]; then
        local base="${endpoint%/}"
        if [[ "$base" == */api ]]; then
            base="${base%/api}"
        elif [[ "$base" == */api/* ]]; then
            base="${base%%/api/*}"
        fi

        local slug
        for slug in "${slugs[@]}"; do
            urls+=("${base}/api/s/${slug}")
        done
    fi

    local tmp_bin
    tmp_bin="$(mktemp)"
    local url
    local downloaded_url=""

    for url in "${urls[@]}"; do
        if curl --fail --silent --show-error --location --output "$tmp_bin" "$url"; then
            if [ ! -s "$tmp_bin" ]; then
                echo "Warning: downloaded file is empty from $url."
                continue
            fi
            cp "$tmp_bin" "$INSTALL_DIR/agent"
            chmod +x "$INSTALL_DIR/agent"
            downloaded_url="$url"
            break
        fi
    done

    rm -f "$tmp_bin"

    if [ -n "$downloaded_url" ]; then
        echo "Installed agent from shortlink: $downloaded_url"
        return 0
    fi

    return 1
}

mkdir -p "$INSTALL_DIR"

if install_binary_from_shortlink; then
    :
else
    if [ -n "${CLOUDPASTE_API_ENDPOINT:-}" ] || [ -n "${XBOARD_AGENT_DOWNLOAD_URL:-}" ]; then
        if [ "${XBOARD_AGENT_DOWNLOAD_STRICT:-0}" = "1" ]; then
            echo "Error: failed to download agent from shortlink and strict mode is enabled."
            exit 1
        fi
        echo "Warning: shortlink download failed, fallback to local binary/source build."
    fi
    install_binary "agent" "./cmd/agent/main.go"
fi

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
