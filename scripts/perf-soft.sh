#!/usr/bin/env bash
# Measures what an idle app costs in CPU while its text caret blinks, with the software renderer
# (soft) and with OpenGL ES (gles), on this machine's software OpenGL (Mesa's llvmpipe), under a
# headless Sway and a headless X server (step 32.3, docs/PERFORMANCE.md "The software renderer on
# Linux"). The app is examples/editor, whose caret blinks twice a second and which draws nothing else.
#
# Usage: scripts/perf-soft.sh [-s seconds] [-r repeats] [-b wayland|x11]
#
#   -s seconds   how long each case is measured, after 4 seconds of settling (default 20)
#   -r repeats   how many times each case is measured (default 3); the table shows each run
#   -b backend   only that display server (default both)
#
# Environment:
#   PERF_TAGS    build tags of the editor (default glutter_release: no debug checks)
#   HEADLESS_SIZE, HEADLESS_SCALE   the output (default 1280x800 at 1; the window is 820x460)
#
# Needs sway (scripts/headless-wayland.sh), Xvfb (scripts/headless-x11.sh) and xdotool. The CPU time
# is the user and system time of the whole process (every thread, llvmpipe's included) that
# /proc/<pid>/stat reports, so it is what top shows, not a time on the wall clock.
set -euo pipefail

cd "$(dirname "$0")/.."
seconds=20
repeats=3
only=""
inner=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    -s) seconds="$2"; shift 2 ;;
    -r) repeats="$2"; shift 2 ;;
    -b) only="$2"; shift 2 ;;
    --inner) inner="$2"; shift 2 ;;
    *) echo "usage: $0 [-s seconds] [-r repeats] [-b wayland|x11]" >&2; exit 64 ;;
  esac
done
tags="${PERF_TAGS:-glutter_release}"

if [[ -z "$inner" ]]; then
  bin="$(mktemp -d)"
  trap 'rm -rf "$bin"' EXIT
  go build -tags "$tags" -o "$bin/editor" ./examples/editor
  export PERF_BIN="$bin" PERF_SECONDS="$seconds" PERF_REPEATS="$repeats"
  export HEADLESS_SIZE="${HEADLESS_SIZE:-1280x800}" HEADLESS_SCALE="${HEADLESS_SCALE:-1}" HEADLESS_FLOATING=0
  echo "| display | renderer | run | CPU % | CPU ms per second | frames per second | the renderer says |"
  echo "|---|---|---|---|---|---|---|"
  if [[ "$only" != x11 ]]; then
    scripts/headless-wayland.sh -- "$0" --inner wayland
  fi
  if [[ "$only" != wayland ]]; then
    scripts/headless-x11.sh -- "$0" --inner x11
  fi
  exit 0
fi

seconds="$PERF_SECONDS"
bin="$PERF_BIN"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
ticks_per_second="$(getconf CLK_TCK)"

# cpu_ticks is the user and system time of the process in clock ticks.
cpu_ticks() { awk '{ sub(/^.*\) /, ""); print $12 + $13 }' "/proc/$1/stat"; }

# measure runs the editor with a renderer and prints one row of the table.
measure() {
  local backend="$1" renderer="$2" run="$3" log="$tmp/$1-$2-$3.log" pid i t0 t1 w
  env GLUTTER_BACKEND="$backend" GLUTTER_RENDERER="$renderer" GLUTTER_FRAME_STATS=1 "$bin/editor" >"$log" 2>&1 &
  pid=$!
  for i in $(seq 1 300); do
    grep -q "first frame presented" "$log" 2>/dev/null && break
    kill -0 "$pid" 2>/dev/null || { cat "$log" >&2; echo "the editor stopped before its first frame" >&2; return 1; }
    sleep 0.1
  done
  if [[ "$backend" == x11 ]]; then
    # There is no window manager on the headless X server: give the window the focus, as one would.
    for w in $(xdotool search --pid "$pid" 2>/dev/null); do xdotool windowfocus "$w" 2>/dev/null || true; done
  fi
  sleep 4
  local lines0
  lines0="$(grep -c ' frames renderer=' "$log" || true)"
  t0="$(cpu_ticks "$pid")"
  sleep "$seconds"
  t1="$(cpu_ticks "$pid")"
  kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
  # The frames of the measured seconds, from the lines the engine logs every second (fps=).
  local fps
  fps="$(grep ' frames renderer=' "$log" | tail -n +"$((lines0 + 1))" | sed -n 's/.* fps=\([0-9]*\) .*/\1/p' | awk '{ s += $1; n++ } END { if (n) printf "%.1f", s / n; else print "0" }')"
  local said
  said="$(sed -n 's/.* renderer chosen renderer="\(.*\)"$/\1/p' "$log" | head -1)"
  awk -v b="$backend" -v r="$renderer" -v run="$run" -v t0="$t0" -v t1="$t1" -v hz="$ticks_per_second" -v s="$seconds" -v fps="$fps" -v said="$said" \
    'BEGIN { ms = (t1 - t0) * 1000 / hz; printf "| %s | %s | %s | %.1f | %.0f | %s | %s |\n", b, r, run, ms / (s * 10), ms / s, fps, said }'
}

for renderer in gles soft; do
  for run in $(seq 1 "$PERF_REPEATS"); do
    measure "$inner" "$renderer" "$run"
  done
done
