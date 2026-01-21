#!/usr/bin/env bash
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUN_DIR="${DIR}/run"
PIDFILE="${RUN_DIR}/data_sync.pid"

if [[ ! -f "$PIDFILE" ]]; then
  echo "[data_sync] no pidfile, nothing to stop"
  exit 0
fi

PID="$(cat "$PIDFILE" 2>/dev/null || true)"
if [[ -z "$PID" ]] || ! ps -p "$PID" >/dev/null 2>&1; then
  echo "[data_sync] stale pidfile, removing"
  rm -f "$PIDFILE"
  exit 0
fi

echo "[data_sync] stopping pid $PID ..."
kill -TERM "$PID" || true

for i in {1..20}; do
  if ! ps -p "$PID" >/dev/null 2>&1; then
    rm -f "$PIDFILE"
    echo "[data_sync] stopped"
    exit 0
  fi
  sleep 1
done

echo "[data_sync] force kill..."
kill -KILL "$PID" || true
rm -f "$PIDFILE"