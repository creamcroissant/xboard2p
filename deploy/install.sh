#!/bin/sh
set -e

MODE="full"
ACTION="install"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PANEL_SCRIPT="${SCRIPT_DIR}/panel.sh"
AGENT_SCRIPT="${SCRIPT_DIR}/agent.sh"

run_installer() {
    script_path=$1
    shift

    XBOARD_RELEASE_BASE_URL="${XBOARD_RELEASE_BASE_URL:-}" \
    XBOARD_RELEASE_REPO="${XBOARD_RELEASE_REPO:-}" \
    XBOARD_RELEASE_TAG="${XBOARD_RELEASE_TAG:-}" \
    sh "$script_path" "$@"
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --panel-only)
            MODE="panel"
            shift
            ;;
        --agent-only)
            MODE="agent"
            shift
            ;;
        --full)
            MODE="full"
            shift
            ;;
        --uninstall)
            ACTION="uninstall"
            shift
            ;;
        *)
            echo "Unknown parameter passed: $1"
            exit 1
            ;;
    esac
done

if [ "$ACTION" = "uninstall" ]; then
    echo "Uninstalling XBoard managed artifacts (Mode: $MODE) from ${INSTALL_DIR:-/opt/xboard}..."
else
    echo "Installing XBoard (Mode: $MODE) to ${INSTALL_DIR:-/opt/xboard}..."
fi

case "$MODE" in
    panel)
        if [ ! -f "$PANEL_SCRIPT" ]; then
            echo "Error: panel installer not found at $PANEL_SCRIPT"
            exit 1
        fi
        if [ "$ACTION" = "uninstall" ]; then
            run_installer "$PANEL_SCRIPT" --uninstall
        else
            run_installer "$PANEL_SCRIPT"
        fi
        ;;
    agent)
        if [ ! -f "$AGENT_SCRIPT" ]; then
            echo "Error: agent installer not found at $AGENT_SCRIPT"
            exit 1
        fi
        if [ "$ACTION" = "uninstall" ]; then
            run_installer "$AGENT_SCRIPT" --uninstall
        else
            run_installer "$AGENT_SCRIPT"
        fi
        ;;
    full)
        if [ ! -f "$PANEL_SCRIPT" ]; then
            echo "Error: panel installer not found at $PANEL_SCRIPT"
            exit 1
        fi
        if [ ! -f "$AGENT_SCRIPT" ]; then
            echo "Error: agent installer not found at $AGENT_SCRIPT"
            exit 1
        fi
        if [ "$ACTION" = "uninstall" ]; then
            run_installer "$AGENT_SCRIPT" --uninstall
            run_installer "$PANEL_SCRIPT" --uninstall
        else
            run_installer "$PANEL_SCRIPT"
            run_installer "$AGENT_SCRIPT"
        fi
        ;;
esac

if [ "$ACTION" = "uninstall" ]; then
    echo "=== Uninstall Complete ==="
else
    echo "=== Installation Complete ==="
fi
if [ "$ACTION" != "uninstall" ] && { [ "$MODE" = "panel" ] || [ "$MODE" = "full" ]; }; then
    echo "- Panel: systemctl start xboard"
fi
if [ "$ACTION" != "uninstall" ] && { [ "$MODE" = "agent" ] || [ "$MODE" = "full" ]; }; then
    echo "- Agent: systemctl start xboard-agent"
fi
