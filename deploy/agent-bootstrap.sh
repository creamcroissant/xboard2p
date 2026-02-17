#!/bin/sh
set -e

XBOARD_BOOTSTRAP_REPO="${XBOARD_BOOTSTRAP_REPO:-creamcroissant/xboard2p}"
XBOARD_BOOTSTRAP_REF="${XBOARD_BOOTSTRAP_REF:-latest}"
XBOARD_BOOTSTRAP_RAW_BASE_URL="${XBOARD_BOOTSTRAP_RAW_BASE_URL:-https://raw.githubusercontent.com}"
XBOARD_BOOTSTRAP_RELEASE_BASE_URL="${XBOARD_BOOTSTRAP_RELEASE_BASE_URL:-https://github.com}"
XBOARD_BOOTSTRAP_API_BASE_URL="${XBOARD_BOOTSTRAP_API_BASE_URL:-https://api.github.com}"
XBOARD_BOOTSTRAP_CHECKSUM_URL="${XBOARD_BOOTSTRAP_CHECKSUM_URL:-}"
XBOARD_AGENT_SCRIPT_URL="${XBOARD_AGENT_SCRIPT_URL:-}"
XBOARD_COMMON_SCRIPT_URL="${XBOARD_COMMON_SCRIPT_URL:-}"
XBOARD_AGENT_SERVICE_URL="${XBOARD_AGENT_SERVICE_URL:-}"
XBOARD_BOOTSTRAP_KEEP_TEMP="${XBOARD_BOOTSTRAP_KEEP_TEMP:-0}"
XBOARD_BOOTSTRAP_DOWNLOAD_STRICT="${XBOARD_BOOTSTRAP_DOWNLOAD_STRICT:-0}"

show_usage() {
    cat <<'EOF'
Usage: sh agent-bootstrap.sh [options] [-- <agent.sh args>]

Options:
  --ref <latest|tag|commit>   Bootstrap source ref (default: latest)
  --repo <owner/repo>         GitHub repo (default: creamcroissant/xboard2p)
  --checksum-url <url>        Override checksum manifest URL
  --service-url <url>         Override agent.service URL
  -h, --help                  Show this help message

Environment overrides:
  XBOARD_BOOTSTRAP_REF, XBOARD_BOOTSTRAP_REPO
  XBOARD_BOOTSTRAP_RAW_BASE_URL, XBOARD_BOOTSTRAP_RELEASE_BASE_URL
  XBOARD_BOOTSTRAP_API_BASE_URL, XBOARD_BOOTSTRAP_CHECKSUM_URL
  XBOARD_BOOTSTRAP_DOWNLOAD_STRICT
  XBOARD_AGENT_SCRIPT_URL, XBOARD_COMMON_SCRIPT_URL
  XBOARD_AGENT_SERVICE_URL, XBOARD_AGENT_SERVICE_FILE
  XBOARD_RELEASE_TAG, XBOARD_RELEASE_REPO, XBOARD_RELEASE_BASE_URL
EOF
}

require_command() {
    cmd_name=$1
    if ! command -v "$cmd_name" >/dev/null 2>&1; then
        echo "Error: required command '$cmd_name' not found."
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
            "${wanted_name}"|"deploy/${wanted_name}")
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

is_bootstrap_strict_mode() {
    [ "$XBOARD_BOOTSTRAP_DOWNLOAD_STRICT" = "1" ]
}

resolve_local_agent_service_fallback() {
    if [ -n "${XBOARD_AGENT_SERVICE_FILE:-}" ] && [ -f "${XBOARD_AGENT_SERVICE_FILE}" ]; then
        printf '%s' "${XBOARD_AGENT_SERVICE_FILE}"
        return 0
    fi

    if [ -f "${CALLER_DIR}/deploy/agent.service" ]; then
        printf '%s' "${CALLER_DIR}/deploy/agent.service"
        return 0
    fi

    if [ -f "${CALLER_DIR}/agent.service" ]; then
        printf '%s' "${CALLER_DIR}/agent.service"
        return 0
    fi

    return 1
}

fallback_agent_service() {
    failure_reason=$1

    if is_bootstrap_strict_mode; then
        echo "Error: ${failure_reason}"
        echo "Error: strict mode enabled (XBOARD_BOOTSTRAP_DOWNLOAD_STRICT=1); refusing local fallback."
        return 1
    fi

    fallback_source=$(resolve_local_agent_service_fallback || true)
    if [ -z "$fallback_source" ]; then
        echo "Error: ${failure_reason}"
        echo "Error: local fallback not found (checked XBOARD_AGENT_SERVICE_FILE, ${CALLER_DIR}/deploy/agent.service, ${CALLER_DIR}/agent.service)."
        return 1
    fi

    if ! cp "$fallback_source" "$WORKDIR/agent.service"; then
        echo "Error: ${failure_reason}"
        echo "Error: failed to copy local fallback agent.service from ${fallback_source}."
        return 1
    fi

    echo "Warning: ${failure_reason}"
    echo "Warning: using local fallback agent.service from ${fallback_source}."
    return 0
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --ref)
            if [ "$#" -lt 2 ]; then
                echo "Error: --ref requires a value."
                exit 1
            fi
            XBOARD_BOOTSTRAP_REF=$2
            shift 2
            ;;
        --repo)
            if [ "$#" -lt 2 ]; then
                echo "Error: --repo requires a value."
                exit 1
            fi
            XBOARD_BOOTSTRAP_REPO=$2
            shift 2
            ;;
        --checksum-url)
            if [ "$#" -lt 2 ]; then
                echo "Error: --checksum-url requires a value."
                exit 1
            fi
            XBOARD_BOOTSTRAP_CHECKSUM_URL=$2
            shift 2
            ;;
        --service-url)
            if [ "$#" -lt 2 ]; then
                echo "Error: --service-url requires a value."
                exit 1
            fi
            XBOARD_AGENT_SERVICE_URL=$2
            shift 2
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        --)
            shift
            break
            ;;
        *)
            echo "Error: unknown argument: $1"
            show_usage
            exit 1
            ;;
    esac
done

if ! require_command curl; then
    exit 1
fi

CALLER_DIR=$(pwd)
WORKDIR=$(mktemp -d 2>/dev/null || mktemp -d -t xboard-agent-bootstrap)
if [ -z "$WORKDIR" ]; then
    echo "Error: failed to create temporary working directory."
    exit 1
fi

cleanup() {
    if [ "$XBOARD_BOOTSTRAP_KEEP_TEMP" = "1" ]; then
        echo "Bootstrap temp directory retained: ${WORKDIR}"
        return
    fi

    rm -rf "$WORKDIR"
}
trap cleanup EXIT INT TERM

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

if [ -n "${XBOARD_RELEASE_TAG:-}" ]; then
    RELEASE_TAG_TO_USE="$XBOARD_RELEASE_TAG"
else
    RELEASE_TAG_TO_USE="$DEFAULT_RELEASE_TAG"
fi

if [ -z "$RELEASE_TAG_TO_USE" ]; then
    echo "Error: bootstrap ref '${RESOLVED_REF}' looks like a commit hash."
    echo "Please set XBOARD_RELEASE_TAG to a release tag to keep script and binary versions consistent."
    exit 1
fi

RAW_BASE="${XBOARD_BOOTSTRAP_RAW_BASE_URL%/}"
RELEASE_BASE="${XBOARD_BOOTSTRAP_RELEASE_BASE_URL%/}"

if [ -n "$XBOARD_AGENT_SCRIPT_URL" ]; then
    AGENT_URL="$XBOARD_AGENT_SCRIPT_URL"
else
    AGENT_URL="${RAW_BASE}/${XBOARD_BOOTSTRAP_REPO}/${RESOLVED_REF}/deploy/agent.sh"
fi
if [ -n "$XBOARD_COMMON_SCRIPT_URL" ]; then
    COMMON_URL="$XBOARD_COMMON_SCRIPT_URL"
else
    COMMON_URL="${RAW_BASE}/${XBOARD_BOOTSTRAP_REPO}/${RESOLVED_REF}/deploy/common.sh"
fi
if [ -n "$XBOARD_AGENT_SERVICE_URL" ]; then
    SERVICE_URL="$XBOARD_AGENT_SERVICE_URL"
else
    SERVICE_URL="${RAW_BASE}/${XBOARD_BOOTSTRAP_REPO}/${RESOLVED_REF}/deploy/agent.service"
fi

if [ -n "$XBOARD_BOOTSTRAP_CHECKSUM_URL" ]; then
    CHECKSUM_URL="$XBOARD_BOOTSTRAP_CHECKSUM_URL"
elif [ "$XBOARD_BOOTSTRAP_REF" = "latest" ] || ! is_commit_ref "$XBOARD_BOOTSTRAP_REF"; then
    CHECKSUM_URL="${RELEASE_BASE}/${XBOARD_BOOTSTRAP_REPO}/releases/download/${RELEASE_TAG_TO_USE}/SHA256SUMS.txt"
else
    CHECKSUM_URL="${RAW_BASE}/${XBOARD_BOOTSTRAP_REPO}/${RESOLVED_REF}/deploy/agent-bootstrap.sha256"
fi

if ! download_file "$AGENT_URL" "$WORKDIR/agent.sh" "agent.sh"; then
    exit 1
fi
if ! download_file "$COMMON_URL" "$WORKDIR/common.sh" "common.sh"; then
    exit 1
fi

if ! download_file "$SERVICE_URL" "$WORKDIR/agent.service" "agent.service"; then
    if ! fallback_agent_service "failed to download agent.service from ${SERVICE_URL}."; then
        exit 1
    fi
fi

if ! download_file "$CHECKSUM_URL" "$WORKDIR/checksums.txt" "checksum manifest"; then
    echo "Error: checksum manifest download failed: ${CHECKSUM_URL}"
    exit 1
fi

if ! verify_checksum "agent.sh" "$WORKDIR/agent.sh" "$WORKDIR/checksums.txt"; then
    exit 1
fi
if ! verify_checksum "common.sh" "$WORKDIR/common.sh" "$WORKDIR/checksums.txt"; then
    exit 1
fi
if ! verify_checksum "agent.service" "$WORKDIR/agent.service" "$WORKDIR/checksums.txt"; then
    if ! fallback_agent_service "checksum verification failed for agent.service."; then
        exit 1
    fi
fi

if [ -z "${XBOARD_RELEASE_REPO:-}" ]; then
    XBOARD_RELEASE_REPO="$XBOARD_BOOTSTRAP_REPO"
fi
if [ -z "${XBOARD_RELEASE_BASE_URL:-}" ]; then
    XBOARD_RELEASE_BASE_URL="$XBOARD_BOOTSTRAP_RELEASE_BASE_URL"
fi
XBOARD_RELEASE_TAG="$RELEASE_TAG_TO_USE"
export XBOARD_RELEASE_REPO XBOARD_RELEASE_BASE_URL XBOARD_RELEASE_TAG

XBOARD_AGENT_SERVICE_FILE="$WORKDIR/agent.service"
export XBOARD_AGENT_SERVICE_FILE

chmod +x "$WORKDIR/agent.sh" "$WORKDIR/common.sh"

echo "Running agent installer (ref=${RESOLVED_REF}, release_tag=${XBOARD_RELEASE_TAG})..."
(
    cd "$WORKDIR"
    sh ./agent.sh "$@"
)
