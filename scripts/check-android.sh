#!/usr/bin/env bash
# Builds and vets the whole module for Android (docs/SPEC.md §24.13), with the NDK's clang
# and cgo, for android/arm64 and android/amd64. It needs the NDK and no device.
#
# Usage: scripts/check-android.sh
#
# The NDK is $ANDROID_NDK_HOME, else the newest $ANDROID_HOME/ndk/<version>. The API level
# of the clang is the minimum API level (SPEC §24.1, D24).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIN_SDK=26

log() { printf '%s\n' "$*" >&2; }

# ndk_dir prints the NDK to use, or nothing.
ndk_dir() {
  if [[ -n "${ANDROID_NDK_HOME:-}" && -d "$ANDROID_NDK_HOME" ]]; then
    printf '%s' "$ANDROID_NDK_HOME"
    return
  fi
  local home="${ANDROID_HOME:-${ANDROID_SDK_ROOT:-}}"
  if [[ -n "$home" && -d "$home/ndk" ]]; then
    find "$home/ndk" -mindepth 1 -maxdepth 1 -type d | sort -V | tail -n 1
  fi
}

NDK="$(ndk_dir)"
if [[ -z "$NDK" ]]; then
  log "error: the Android NDK was not found (set ANDROID_NDK_HOME, or ANDROID_HOME to the SDK)"
  log "install it with: sdkmanager \"ndk;30.0.16248370\""
  exit 1
fi
BIN="$NDK/toolchains/llvm/prebuilt/linux-x86_64/bin"

cd "$ROOT"
# check ARCH TRIPLE runs go build and go vet for one ABI.
check() {
  local arch="$1" triple="$2"
  local cc="$BIN/${triple}${MIN_SDK}-clang"
  if [[ ! -x "$cc" ]]; then
    log "error: $cc is missing; the NDK at $NDK is not complete"
    exit 1
  fi
  log "android/$arch: go build ./..."
  GOOS=android GOARCH="$arch" CGO_ENABLED=1 CC="$cc" go build ./...
  log "android/$arch: go vet ./..."
  GOOS=android GOARCH="$arch" CGO_ENABLED=1 CC="$cc" go vet ./...
}

check arm64 aarch64-linux-android
check amd64 x86_64-linux-android
log "android: ok"
