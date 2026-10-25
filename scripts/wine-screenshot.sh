#!/usr/bin/env bash
# Saves a screenshot of the root window of a headless X server while a command runs (docs/SPEC.md
# §25.13, proof method F). It is for what lies outside an app's own window: the tray, a dialog,
# several windows. For one window, the app's own self-capture (GLUTTER_SCREENSHOT) is better.
#
# Usage: scripts/wine-screenshot.sh <file.png> -- <command...>
#        scripts/wine-screenshot.sh docs/implemented-proof-images/28.11-tray.png -- scripts/wine-exec.sh ./tray.exe
#
# The command runs inside the headless X server (scripts/headless-x11.sh) with GLUTTER_WINE_INSIDE=1,
# so scripts/wine-exec.sh uses that server instead of starting its own. After GLUTTER_SHOT_DELAY
# seconds (3 by default) the root window is saved with ImageMagick's `import -window root`, and the
# command is stopped (TERM, then `wineserver -k`). If the command ends before the delay, nothing is
# saved and the status is 1.
#
# Needs Wine, Xvfb and ImageMagick: sudo dnf install ImageMagick
#
# Exit status: 0 when the file was saved; 1 when the command ended before the capture or the capture
# failed; 69 when a tool is missing; 64 for a usage mistake.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  echo "usage: $0 <file.png> -- <command...>" >&2
  exit 64
}

[[ $# -ge 3 && "$2" == "--" ]] || usage
OUT="$1"
shift 2

if ! command -v import >/dev/null 2>&1; then
  echo "error: ImageMagick (import) is not installed." >&2
  echo "       install it with: sudo dnf install ImageMagick" >&2
  exit 69
fi
if ! command -v Xvfb >/dev/null 2>&1; then
  echo "error: Xvfb is not installed." >&2
  echo "       install it with: sudo dnf install xorg-x11-server-Xvfb" >&2
  exit 69
fi

# The prefix is the one scripts/wine-exec.sh uses, so that the wineserver -k below stops the right
# server and never touches a prefix of the user's.
export WINEPREFIX="${GLUTTER_WINEPREFIX:-$ROOT/.wine-test/prefix}"

mkdir -p "$(dirname "$OUT")"
OUT="$(cd "$(dirname "$OUT")" && pwd)/$(basename "$OUT")"
export GLUTTER_WINE_INSIDE=1
export GLUTTER_SHOT_OUT="$OUT"
export GLUTTER_SHOT_DELAY="${GLUTTER_SHOT_DELAY:-3}"

exec "$ROOT/scripts/headless-x11.sh" -- bash -c '
  ulimit -c 1
  "$@" &
  cmd=$!
  status=1
  for ((i = 0; i < GLUTTER_SHOT_DELAY * 10; i++)); do
    if ! kill -0 "$cmd" 2>/dev/null; then
      echo "wine-screenshot: the command ended before the capture" >&2
      exit 1
    fi
    sleep 0.1
  done
  if import -window root "$GLUTTER_SHOT_OUT"; then
    status=0
  else
    echo "wine-screenshot: the capture failed" >&2
  fi
  kill -TERM "$cmd" 2>/dev/null
  for ((i = 0; i < 100; i++)); do
    kill -0 "$cmd" 2>/dev/null || break
    sleep 0.1
  done
  kill -KILL "$cmd" 2>/dev/null
  wait "$cmd" 2>/dev/null
  wineserver -k >/dev/null 2>&1
  exit "$status"
' bash "$@"
