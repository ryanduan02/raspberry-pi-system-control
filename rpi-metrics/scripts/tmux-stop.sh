#!/usr/bin/env bash
set -euo pipefail

SESSION_NAME=${RPI_METRICS_TMUX_SESSION:-rpi-metrics}

usage() {
  cat <<'EOF'
Usage:
  ./scripts/tmux-stop.sh [--session NAME]
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    --session)
      SESSION_NAME=${2:-}
      if [[ -z "$SESSION_NAME" ]]; then
        echo "--session requires a value" >&2
        exit 2
      fi
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if ! command -v tmux >/dev/null 2>&1; then
  echo "tmux not found. Install on the Pi: sudo apt-get install -y tmux" >&2
  exit 1
fi

if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
  tmux kill-session -t "$SESSION_NAME"
  echo "Stopped tmux session: $SESSION_NAME" >&2
else
  echo "No such tmux session: $SESSION_NAME" >&2
fi
