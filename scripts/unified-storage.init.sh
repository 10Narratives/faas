#!/bin/sh
set -eu

NATS_URL="${NATS_URL:-nats://127.0.0.1:4222}"

nats --server "$NATS_URL" str add TASKS --config /etc/nats/streams/tasks.json
nats --server "$NATS_URL" kv add functions
nats --server "$NATS_URL" kv add tasks
nats --server "$NATS_URL" obj add functions
