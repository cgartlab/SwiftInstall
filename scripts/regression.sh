#!/usr/bin/env bash
# End-to-end regression smoke for SwiftInstall: CLI exit codes + GUI routes.
# Usage: bash scripts/regression.sh   (exit 0 = all pass)
set -uo pipefail
cd "$(dirname "$0")/.." || exit 1
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"

BIN="$(mktemp -t sis.XXXXXX)"
trap 'rm -f "$BIN"' EXIT
if ! go build -ldflags "-X main.Version=$(cat VERSION 2>/dev/null || echo dev)" -o "$BIN" ./cmd/sis/; then
  echo "build failed"; exit 1
fi

pass=0; fail=0

# Cross-platform build check (other OSes must still compile).
XDIR="$(mktemp -d)"
for t in "windows amd64" "linux amd64" "darwin arm64"; do
  set -- $t
  if GOOS=$1 GOARCH=$2 go build -o "$XDIR/sis-$1-$2" ./cmd/sis/ 2>/dev/null; then
    pass=$((pass+1))
  else
    fail=$((fail+1)); echo "FAIL(cross-build) $1/$2"
  fi
done
rm -rf "$XDIR"

MAN="$(mktemp -t man.XXXXXX).yaml"
printf 'packages:\n  - id: git\n  - id: gh\n' > "$MAN"

chk() { local want=$1; shift; "$@" >/dev/null 2>&1; local c=$?;
  if [ "$c" = "$want" ]; then pass=$((pass+1)); else fail=$((fail+1)); echo "FAIL(cli) exit=$c want=$want :: $*"; fi; }

chk 0 "$BIN" version
chk 0 "$BIN" doctor --format json
chk 0 "$BIN" audit --toollist privacy --format json
chk 1 "$BIN" audit --toollist bogus --format json
chk 0 "$BIN" toollist list --format json
chk 0 "$BIN" guide --list --format json
chk 0 "$BIN" status -f "$MAN" --format json
chk 0 "$BIN" mirror show --format json
chk 0 "$BIN" llm show --format json
chk 1 "$BIN" list -f "$MAN" --format xml
chk 1 "$BIN" install -f "$MAN" extra-arg
chk 1 "$BIN" explain SomeApp        # no key configured

PORT=$(( (RANDOM % 2000) + 9000 ))
"$BIN" gui --no-open --port "$PORT" >/dev/null 2>&1 & GPID=$!
sleep 2
G="http://127.0.0.1:$PORT"
gchk() { local u=$1 want=$2 m=${3:-GET} code; if [ "$m" = POST ]; then code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$G$u"); else code=$(curl -s -o /dev/null -w '%{http_code}' "$G$u"); fi
  if [ "$code" = "$want" ]; then pass=$((pass+1)); else fail=$((fail+1)); echo "FAIL(web) $u=$code want=$want"; fi; }
for spec in "/:200" "/audit:200" "/toollists:200" "/health:200" "/swap?toollist=privacy:200" "/share?toollist=privacy:200" \
            "/api/health:200" "/api/audit?toollist=privacy:200" "/api/toollist?toollist=bogus:404" "/cmdk.js:200"; do
  gchk "${spec%:*}" "${spec##*:}"
done
gchk "/api/state" 405 POST
kill "$GPID" 2>/dev/null; wait "$GPID" 2>/dev/null

echo "regression: PASS=$pass FAIL=$fail"
[ "$fail" -eq 0 ]
