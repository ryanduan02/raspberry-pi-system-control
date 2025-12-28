#!/usr/bin/env bash
set -euo pipefail

SESSION_NAME=${RPI_METRICS_TMUX_SESSION:-rpi-metrics}
DETACH=0

usage() {
  cat <<'EOF'
Usage:
  ./scripts/tmux-start.sh [--session NAME] [--detached] -- [rpi-metrics args...]

Examples:
  ./scripts/tmux-start.sh -- -interval=5s
  ./scripts/tmux-start.sh -- -interval=5s -discord-webhook="https://discord.com/api/webhooks/REPLACE_ME" -discord-every=1m
  ./scripts/tmux-start.sh --session rpi --detached -- -interval=5s

Notes:
  - This keeps running after SSH disconnect.
  - Reattach with: ./scripts/tmux-attach.sh [--session NAME]
  - Stop with:     ./scripts/tmux-stop.sh [--session NAME]
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
    --detached)
      DETACH=1
      shift
      ;;
    --)
      shift
      break
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

REPO_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
BIN="$REPO_DIR/bin/rpi-metrics"

if [[ ! -x "$BIN" ]]; then
  echo "Binary not found at $BIN" >&2
  echo "Build it from $REPO_DIR:" >&2
  echo "  go build -o ./bin/rpi-metrics ./cmd/rpi-metrics" >&2
  exit 1
fi

# If session already exists, just attach (or no-op if detached).
if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
  echo "tmux session already running: $SESSION_NAME" >&2
  if [[ "$DETACH" -eq 0 ]]; then
    exec tmux attach -t "$SESSION_NAME"
  fi
  exit 0
fi

ARGS=("$@")
CMD=("$BIN" "${ARGS[@]}")

# Create session detached, set working dir, run command.
tmux new-session -d -s "$SESSION_NAME" -c "$REPO_DIR"
# shellcheck disable=SC2145
printf -v CMD_STR '%q ' "${CMD[@]}"
tmux send-keys -t "$SESSION_NAME" "$CMD_STR" C-m

echo "Started rpi-metrics in tmux session: $SESSION_NAME" >&2

if [[ "$DETACH" -eq 0 ]]; then
  exec tmux attach -t "$SESSION_NAME"
fi
