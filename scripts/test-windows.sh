#!/usr/bin/env bash
# Runs the unit tests of every package that builds for Windows, under Wine (docs/SPEC.md §25.13):
# GOOS=windows GOARCH=amd64 go test -exec "$PWD/scripts/wine-exec.sh" -p 1, like scripts/test-wasm.sh does
# for the web.
#
# Usage: scripts/test-windows.sh [go test flags and packages]
#        (default: every package that has files for Windows, that is, a file named *_windows*.go or a
#        file whose build constraint names windows, tests included; not the tools of cmd/, and not
#        the host-side harness internal/winetest, whose tests start Wine themselves)
#
# The tests run one package at a time (-p 1): they share one prefix and one Wine server. The
# prefix is .wine-test/prefix at the module root (scripts/wine-exec.sh). A package whose tests need
# a window, input or the GPU carries the glutter_wine tag and is run with
#   GOOS=windows go test -tags glutter_wine -exec "$PWD/scripts/wine-exec.sh" -p 1 <packages>
# (docs/IMPLEMENTATION.md, rule 8).
#
# Needs Wine and Xvfb (sudo dnf install wine-core xorg-x11-server-Xvfb). Exit status: go test's;
# 69 when a tool is missing.
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT="$(pwd)"

if ! command -v wine64 >/dev/null 2>&1 && ! command -v wine >/dev/null 2>&1; then
  echo "test-windows.sh: Wine is needed (sudo dnf install wine-core)" >&2
  exit 69
fi
if ! command -v Xvfb >/dev/null 2>&1; then
  echo "test-windows.sh: Xvfb is needed (sudo dnf install xorg-x11-server-Xvfb)" >&2
  exit 69
fi

args=("$@")
if [[ ${#args[@]} -eq 0 ]]; then
  # One line per Go file that builds for Windows: "<import path> <file>".
  listing="$(GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go list -f \
    '{{$p := .ImportPath}}{{$d := .Dir}}{{range .GoFiles}}{{$p}} {{$d}}/{{.}}{{"\n"}}{{end}}{{range .TestGoFiles}}{{$p}} {{$d}}/{{.}}{{"\n"}}{{end}}{{range .XTestGoFiles}}{{$p}} {{$d}}/{{.}}{{"\n"}}{{end}}' ./...)"
  mapfile -t args < <(
    while read -r pkg file; do
      if [[ "$file" == *_windows*.go ]] || grep -qE '^//go:build.*windows' "$file"; then
        echo "$pkg"
      fi
    done <<<"$listing" | sort -u | grep -v -e '/cmd/' -e '/internal/winetest$' || true
  )
fi
if [[ ${#args[@]} -eq 0 ]]; then
  echo "test-windows.sh: no package has files for Windows" >&2
  exit 1
fi

exec env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go test -exec "$ROOT/scripts/wine-exec.sh" -p 1 "${args[@]}"
