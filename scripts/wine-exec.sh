#!/usr/bin/env bash
# Runs a Windows program under Wine on a headless X server (docs/SPEC.md §25.13). It is the runner
# for `go test -exec`, and the way the proof screenshots of method F run an app.
#
# Usage: scripts/wine-exec.sh <program.exe> [args...]
#        GOOS=windows go test -exec "$PWD/scripts/wine-exec.sh" -p 1 ./internal/platform/windows/...
#
# What it does:
#   - starts a headless X server (scripts/headless-x11.sh, WAYLAND_DISPLAY unset), unless one is
#     given already (GLUTTER_WINE_INSIDE=1, which scripts/wine-screenshot.sh sets);
#   - uses a private prefix, .wine-test/prefix at the module root (git-ignored); GLUTTER_WINEPREFIX
#     moves it. Nothing of the user's ~/.wine is read or written. A new prefix is made on the first
#     run, under a lock: its first boot hangs when the wine-cms package is not installed (setupapi
#     loops on the missing mscms.dll), so it is cut after 20 seconds and the server is stopped;
#   - sets WINEDLLOVERRIDES="mscoree,mshtml=" (no Mono or Gecko prompt), WINEDEBUG=-all
#     (GLUTTER_WINEDEBUG overrides it), WINEARCH=win64, and VK_DRIVER_FILES to lavapipe's ICD (the
#     only Vulkan driver that presents on Xvfb) unless it is set;
#   - lowers the core-file limit (ulimit -c 1), so that a crash of Wine never writes a core file or
#     reaches the system's crash reporter;
#   - runs the program under `timeout -k 10` (GLUTTER_WINE_TIMEOUT seconds, 600 by default);
#   - stops Wine with `wineserver -k` at the end (never `wineserver -w`: winedevice stays alive).
#
# The tests of different packages share the prefix and the server, so run one package at a time
# (go test -p 1).
#
# Exit status: the program's own (124 when the timeout ended it); 69 when Wine or Xvfb is not
# installed; 64 for a usage mistake.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <program.exe> [args...]" >&2
  exit 64
fi

WINE="$(command -v wine64 || command -v wine || true)"
WINESERVER="$(command -v wineserver || true)"
if [[ -z "$WINE" || -z "$WINESERVER" ]]; then
  echo "error: Wine is not installed." >&2
  echo "       install it with: sudo dnf install wine-core" >&2
  exit 69
fi
if ! command -v Xvfb >/dev/null 2>&1; then
  echo "error: Xvfb is not installed." >&2
  echo "       install it with: sudo dnf install xorg-x11-server-Xvfb" >&2
  exit 69
fi

PREFIX="${GLUTTER_WINEPREFIX:-$ROOT/.wine-test/prefix}"
export WINEPREFIX="$PREFIX"
export WINEARCH=win64
export WINEDLLOVERRIDES="mscoree,mshtml="
export WINEDEBUG="${GLUTTER_WINEDEBUG:--all}"
unset WAYLAND_DISPLAY WAYLAND_SOCKET
if [[ -z "${VK_DRIVER_FILES:-}" && -f /usr/share/vulkan/icd.d/lvp_icd.x86_64.json ]]; then
  export VK_DRIVER_FILES=/usr/share/vulkan/icd.d/lvp_icd.x86_64.json
fi
ulimit -c 1

# Outside the X server: start it and run this script again inside it.
if [[ "${GLUTTER_WINE_INSIDE:-}" != 1 ]]; then
  export GLUTTER_WINE_INSIDE=1
  exec "$ROOT/scripts/headless-x11.sh" -- "$0" "$@"
fi

stop() { "$WINESERVER" -k >/dev/null 2>&1 || true; }
trap stop EXIT

if [[ ! -f "$PREFIX/system.reg" ]]; then
  mkdir -p "$(dirname "$PREFIX")"
  (
    flock 9
    if [[ ! -f "$PREFIX/system.reg" ]]; then
      timeout -k 5 20 "$WINE" wineboot --init >/dev/null 2>&1
      stop
    fi
  ) 9>"$PREFIX.lock"
fi

# The prefix of Wine has no font directory, and an app that scans the fonts of Windows (the text
# of the Material apps asks for Segoe UI) would draw no text: give it a copy of the host's
# Liberation Sans, the way a Windows machine has C:\Windows\Fonts (SPEC §25.13).
FONTS="$PREFIX/drive_c/windows/Fonts"
if ! compgen -G "$FONTS/*.ttf" >/dev/null; then
  mkdir -p "$FONTS"
  for f in /usr/share/fonts/liberation-sans-fonts/*.ttf /usr/share/fonts/liberation/LiberationSans-*.ttf; do
    [[ -f "$f" ]] && cp "$f" "$FONTS/"
  done
fi

# The program runs in the background and this script waits for it, so that a signal is handled at
# once (bash runs a trap only when its foreground child ends): TERM and INT end the program, then
# the trap of EXIT stops Wine.
child=""
on_signal() {
  [[ -n "$child" ]] && kill -TERM "$child" 2>/dev/null
  [[ -n "$child" ]] && wait "$child" 2>/dev/null
  exit "$1"
}
trap 'on_signal 143' TERM
trap 'on_signal 130' INT
timeout -k 10 "${GLUTTER_WINE_TIMEOUT:-600}" "$WINE" "$@" &
child=$!
wait "$child"
status=$?
child=""
exit "$status"
