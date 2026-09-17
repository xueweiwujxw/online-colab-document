#!/usr/bin/env bash
# Run against a disposable RustFS, never an existing application bucket.
set -euo pipefail
cd "$(dirname "$0")/.."
: "${RUSTFS_BINARY:?set RUSTFS_BINARY to the RustFS 1.0.0 executable}"
GO=${GO:-go}
work_dir=$(mktemp -d)
mkdir -p "$work_dir/data"
rustfs_pid=
cleanup() {
  if [ -n "$rustfs_pid" ]; then kill "$rustfs_pid" 2>/dev/null || true; wait "$rustfs_pid" 2>/dev/null || true; fi
  rm -rf "$work_dir"
}
trap cleanup EXIT
export RUSTFS_TEST_ENDPOINT="http://127.0.0.1:${RUSTFS_TEST_PORT:-19000}"
export RUSTFS_TEST_ACCESS_KEY="test-$(openssl rand -hex 8)"
export RUSTFS_TEST_SECRET_KEY="$(openssl rand -hex 24)"
RUSTFS_ACCESS_KEY="$RUSTFS_TEST_ACCESS_KEY" RUSTFS_SECRET_KEY="$RUSTFS_TEST_SECRET_KEY" \
 RUSTFS_ADDRESS="127.0.0.1:${RUSTFS_TEST_PORT:-19000}" RUSTFS_CONSOLE_ENABLE=false \
 "$RUSTFS_BINARY" "$work_dir/data" > "$work_dir/rustfs.log" 2>&1 &
rustfs_pid=$!
for attempt in $(seq 1 60); do
  kill -0 "$rustfs_pid" 2>/dev/null || { echo 'RustFS exited before readiness' >&2; exit 1; }
  if curl -fsS "$RUSTFS_TEST_ENDPOINT/health/ready" >/dev/null 2>&1; then break; fi
  sleep 1
done
curl -fsS "$RUSTFS_TEST_ENDPOINT/health/ready" >/dev/null
cd backend
"$GO" test -race -count=1 -v ./...
