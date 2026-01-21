#!/usr/bin/env bash
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${DIR}/data_sync"
LOG_DIR="${DIR}/logs"
RUN_DIR="${DIR}/run"

CONFIG="${CONFIG:-${DIR}/config.yaml}"
DATA_SYNC_ACTION="${DATA_SYNC_ACTION:-start}"     # 操作类型: start, test-sync, sync-once, status
DATA_SYNC_CONFIG_FILE="${DATA_SYNC_CONFIG_FILE:-}"  # 同步服务配置文件
DATA_SYNC_SYNCER="${DATA_SYNC_SYNCER:-}"          # 指定的同步器（可选）

mkdir -p "$LOG_DIR" "$RUN_DIR"
ulimit -n 65535 || true

PIDFILE="${RUN_DIR}/data_sync.pid"

if [[ ! -x "$BIN" ]]; then
  echo "[data_sync] binary not found or not executable: $BIN"
  exit 1
fi

if [[ -f "$PIDFILE" ]] && ps -p "$(cat "$PIDFILE")" -o comm= >/dev/null 2>&1; then
  echo "[data_sync] already running with pid $(cat "$PIDFILE")"
  exit 0
fi

echo "[data_sync] starting action=${DATA_SYNC_ACTION}..."
CMD_ARGS="-config \"$CONFIG\" -action \"$DATA_SYNC_ACTION\""

# 添加可选的config-file参数
if [[ -n "$DATA_SYNC_CONFIG_FILE" ]]; then
  CMD_ARGS="$CMD_ARGS -config-file \"$DATA_SYNC_CONFIG_FILE\""
fi

# 添加可选的syncer参数
if [[ -n "$DATA_SYNC_SYNCER" ]]; then
  CMD_ARGS="$CMD_ARGS -syncer \"$DATA_SYNC_SYNCER\""
fi

echo "[data_sync] command: $BIN $CMD_ARGS"
eval nohup "$BIN" $CMD_ARGS \
  >> "${LOG_DIR}/data_sync.out" 2>> "${LOG_DIR}/data_sync.err" &

echo $! > "$PIDFILE"
disown || true
echo "[data_sync] started pid $(cat "$PIDFILE"), logs: ${LOG_DIR}/data_sync.out"