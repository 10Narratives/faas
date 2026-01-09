#!/bin/sh
set -eu

nats-server "$@" &
NATS_PID="$!"

until wget -qO- "http://127.0.0.1:8222/healthz?js-enabled-only=true" | grep -q '"status":"ok"'; do
  sleep 0.2
done

if ! /usr/local/bin/unified-storage.init.sh; then
  kill "$NATS_PID" 2>/dev/null || true
  exit 1
fi

mkdir -p /data/.health
echo ok > /data/.health/init.ok

wait "$NATS_PID"
