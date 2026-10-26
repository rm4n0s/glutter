#!/usr/bin/env bash
# Runs the unit tests of every package that builds for the web, under Node (SPEC §17,
# "Web unit tests"): GOOS=js GOARCH=wasm go test, through Go's go_js_wasm_exec.
#
# Usage: scripts/test-wasm.sh [go test flags and packages]   (default: every package
#        that builds for js, except the tools of cmd/ and docs/ and the host-side web
#        tools internal/webtool, internal/webtest and internal/wintool, whose tests run
#        the go command, browsers and an HTTP server and cannot run under Node)
#
# A package whose files all carry an OS tag has nothing for js, so `go test` skips it;
# the tests of a package that need Linux are in files with the linux tag.
#
# Go's wasm loader fails when the environment is larger than its limit, so the tests
# run with a minimal one (env -i with HOME, PATH, GOCACHE, GOMODCACHE and GOFLAGS).
# Needs Node.js (`node --version`; sudo dnf install nodejs).
set -euo pipefail

cd "$(dirname "$0")/.."
if ! command -v node >/dev/null 2>&1; then
  echo "test-wasm.sh: Node.js is needed (sudo dnf install nodejs)" >&2
  exit 69
fi

goroot="$(go env GOROOT)"
gocache="$(go env GOCACHE)"
gomodcache="$(go env GOMODCACHE)"
goflags="$(go env GOFLAGS)"
gobin="$(command -v go)"

args=("$@")
if [[ ${#args[@]} -eq 0 ]]; then
  mapfile -t args < <(GOOS=js GOARCH=wasm go list ./... | grep -v -e '/cmd/' -e '/docs$' -e '/internal/webtool$' -e '/internal/webtest$' -e '/internal/wintool$')
fi

exec env -i \
  HOME="${HOME:-/tmp}" \
  PATH="$(dirname "$gobin"):$goroot/lib/wasm:$(dirname "$(command -v node)"):/usr/bin:/bin" \
  GOCACHE="$gocache" \
  GOMODCACHE="$gomodcache" \
  GOFLAGS="$goflags" \
  GOOS=js GOARCH=wasm \
  go test "${args[@]}"
