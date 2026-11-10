#!/usr/bin/env bash
# Builds and vets the whole module for iOS (docs/SPEC.md §30.1), for the device (iphoneos) and for the Simulator
# (iphonesimulator), with Xcode's clang and cgo, for ios/arm64. It needs the full Xcode and no device or Simulator.
#
# Usage: scripts/check-ios.sh
#
# It skips, with the instruction, when Xcode is not installed or selected (a Linux machine, or a Mac with only the Command Line
# Tools), as the Simulator tests do.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

log() { printf '%s\n' "$*" >&2; }

if [[ "$(uname -s)" != "Darwin" ]] || ! command -v xcode-select >/dev/null 2>&1; then
  log "skipped: iOS builds need a Mac with Xcode"
  exit 0
fi
DEV="$(xcode-select -p 2>/dev/null || true)"
if [[ "$DEV" != *Xcode*.app* ]]; then
  log "skipped: iOS needs the full Xcode: run 'xcode-select -s /Applications/Xcode.app'"
  exit 0
fi

cd "$ROOT"
# check SDK TARGET runs go build and go vet for one SDK.
check() {
  local sdk="$1" target="$2"
  local sysroot clang flags
  sysroot="$(xcrun --sdk "$sdk" --show-sdk-path)"
  clang="$(xcrun --sdk "$sdk" --find clang)"
  flags="-isysroot $sysroot -target $target"
  log "ios/arm64 ($sdk): go build ./..."
  GOOS=ios GOARCH=arm64 CGO_ENABLED=1 CC="$clang $flags" CGO_CFLAGS="$flags" CGO_LDFLAGS="$flags" go build ./...
  log "ios/arm64 ($sdk): go vet ./..."
  GOOS=ios GOARCH=arm64 CGO_ENABLED=1 CC="$clang $flags" CGO_CFLAGS="$flags" CGO_LDFLAGS="$flags" go vet ./...
  log "ios/arm64 ($sdk): go vet -tags glutter_ios ./..."
  GOOS=ios GOARCH=arm64 CGO_ENABLED=1 CC="$clang $flags" CGO_CFLAGS="$flags" CGO_LDFLAGS="$flags" go vet -tags glutter_ios ./...
}

check iphoneos arm64-apple-ios15.0
check iphonesimulator arm64-apple-ios15.0-simulator
log "ok"
