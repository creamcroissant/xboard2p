#!/bin/sh
set -e

MODE="full"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PANEL_SCRIPT="${SCRIPT_DIR}/panel.sh"
AGENT_SCRIPT="${SCRIPT_DIR}/agent.sh"

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
        *)
            echo "Unknown parameter passed: $1"
            exit 1
            ;;
    esac
done

echo "Installing XBoard (Mode: $MODE) to ${INSTALL_DIR:-/opt/xboard}..."

case "$MODE" in
    panel)
        if [ ! -f "$PANEL_SCRIPT" ]; then
            echo "Error: panel installer not found at $PANEL_SCRIPT"
            exit 1
        fi
        sh "$PANEL_SCRIPT"
        ;;
    agent)
        if [ ! -f "$AGENT_SCRIPT" ]; then
            echo "Error: agent installer not found at $AGENT_SCRIPT"
            exit 1
        fi
        sh "$AGENT_SCRIPT"
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
        sh "$PANEL_SCRIPT"
        sh "$AGENT_SCRIPT"
        ;;
esac

echo "=== Installation Complete ==="
if [ "$MODE" = "panel" ] || [ "$MODE" = "full" ]; then
    echo "- Panel: systemctl start xboard"
fi
if [ "$MODE" = "agent" ] || [ "$MODE" = "full" ]; then
    echo "- Agent: systemctl start xboard-agent"
fi
