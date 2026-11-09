#!/usr/bin/env bash
# Measures the macOS numbers of SPEC §15 (the "macOS" paragraph, step 33.15) with `glutter-macos perf`
# (cmd/glutter-macos/perf.go): for each app, the size of the executable, the start to the first frame (the
# engine's own "first frame presented since_run", with an empty and with a warm shader cache), the idle CPU and
# the memory of the process (`ps`) over a window, and the frame times (GLUTTER_FRAME_STATS=1: frames a second,
# build and raster times, frames over 16 ms and the Go heap). The benchmarks of the framework alone are
# `go test -bench` (docs/PERFORMANCE.md lists them).
#
# These are the numbers of the Mac the script runs on, with Metal on its GPU: the machine is named in the report.
# They are for noticing regressions between steps and for the targets of SPEC §15; the owner confirms them on
# other Macs (docs/MANUAL-CHECKS.md).
#
# Usage: scripts/perf-macos.sh [-o report.md] [-m] [-t tests] [-n starts] [-s seconds] [-S settle] [-g seconds] [-r renderer] [-u]
#
#   -o file      write the report to this file (default: standard output)
#   -m           a Markdown report, for docs/PERFORMANCE.md (default: plain text)
#   -t tests     which measurements to run, comma separated (default: all five):
#                  counter     examples/counter, release: size, start, idle CPU, memory
#                  gallery     examples/gallery on its home page, release: the same
#                  animation   examples/gallery, release, page "data" with its progress indicators animating
#                              (GLUTTER_GALLERY_PAGE=data, GLUTTER_GALLERY_ANIMATE=1): the frame times of an animation
#                  rebuild     examples/gallery, release, page "buttons" built again in every frame
#                              (GLUTTER_GALLERY_REBUILD=1): the build time of a full rebuild
#                  navigation  examples/gallery, a DEBUG build driven by a dev-events script (GLUTTER_DEV_EVENTS:
#                              wheel scrolls, three page changes, two taps on the theme switch)
#   -n starts    starts per measurement (default 5; the first has an empty shader cache)
#   -s seconds   the length of the window (default 10)
#   -S settle    the time the app runs before the window (default 5s)
#   -g seconds   the window of "navigation" is this plus 8 (default 10)
#   -r renderer  auto, metal or soft (GLUTTER_RENDERER; default: the app's choice, which is metal)
#   -u           build universal executables and report the size of each slice
#
# The script of "navigation" is the one of perf-windows.sh (the gallery's window is 1100x760 logical pixels); set
# GALLERY_EVENTS to replace it. The apps run as accessory apps that never take the focus. Run one measurement at a
# time: the numbers are a machine's, and another run on it changes them.
set -euo pipefail

cd "$(dirname "$0")/.."
out=""
format=text
tests=counter,gallery,animation,rebuild,navigation
starts=5
seconds=10
settle=5s
gseconds=10
renderer=""
universal=()
while getopts "o:mt:n:s:S:g:r:u" opt; do
  case "$opt" in
    o) out="$OPTARG" ;;
    m) format=markdown ;;
    t) tests="$OPTARG" ;;
    n) starts="$OPTARG" ;;
    s) seconds="$OPTARG" ;;
    S) settle="$OPTARG" ;;
    g) gseconds="$OPTARG" ;;
    r) renderer="$OPTARG" ;;
    u) universal=(-universal) ;;
    *) sed -n '/^# Usage:/,/^#   -u /p' "$0" | sed 's/^# \{0,1\}//' >&2; exit 64 ;;
  esac
done

wheel() { echo "wheel:650,400,0,$1"; }
scroll="$(wheel 150);wait:40ms;$(wheel 150);wait:40ms;$(wheel 150);wait:40ms;$(wheel 150);wait:300ms;$(wheel -300);wait:40ms;$(wheel -300);wait:300ms"
page() { echo "tap:100,$((86 + 44 * $1));wait:1200ms"; }
theme="tap:1072,28;wait:1200ms;tap:1072,28;wait:1200ms"
events="${GALLERY_EVENTS:-wait:1s;$scroll;$(page 9);$scroll;$(page 10);$scroll;$(page 3);$scroll;$theme;$(page 0);wait:1s}"
nseconds=$((gseconds + 8))

tmp="$(mktemp -d)"
cleaned=0
cleanup() {
  [[ $cleaned == 1 ]] && return
  cleaned=1
  trap '' INT TERM
  pkill -TERM -P $$ 2>/dev/null || true
  for _ in $(seq 1 30); do
    pgrep -P $$ >/dev/null 2>&1 || break
    sleep 1
  done
  pkill -KILL -P $$ 2>/dev/null || true
  rm -rf "$tmp"
}
trap cleanup EXIT
trap 'exit 130' INT TERM

echo "building glutter-macos..." >&2
go build -o "$tmp/glutter-macos" ./cmd/glutter-macos
gm="$tmp/glutter-macos"

rend=()
[[ -n $renderer ]] && rend=(-renderer "$renderer")

n=0
perf() {
  local title="$1" limit="$2"
  shift 2
  n=$((n + 1))
  echo "measuring $title..." >&2
  {
    if [[ $format == markdown ]]; then echo "## $title"; else echo "=== $title"; fi
    echo
  } >"$tmp/report.$n"
  # In the background and waited for, so that the trap runs at once when the script is interrupted.
  timeout -k 30 "$limit" "$gm" perf -format "$format" -starts "$starts" -settle "$settle" -seconds "$seconds" \
    ${universal[@]+"${universal[@]}"} ${rend[@]+"${rend[@]}"} "$@" >>"$tmp/report.$n" 2>"$tmp/err.$n" &
  if ! wait $!; then
    cat "$tmp/err.$n" >&2
    echo "perf-macos.sh: the measurement \"$title\" failed" >&2
    exit 1
  fi
  echo >>"$tmp/report.$n"
}

for t in ${tests//,/ }; do
  case "$t" in
    counter) perf "counter (release)" 1800 -release=true ./examples/counter ;;
    gallery) perf "gallery, home page (release)" 1800 -release=true ./examples/gallery ;;
    animation)
      perf "gallery, an animation with no input (release)" 1800 -release=true -seconds "$gseconds" -starts 1 \
        -env GLUTTER_GALLERY_PAGE=data -env GLUTTER_GALLERY_ANIMATE=1 ./examples/gallery ;;
    rebuild)
      perf "gallery, a full rebuild in every frame (release)" 1800 -release=true -seconds "$gseconds" -starts 1 \
        -env GLUTTER_GALLERY_PAGE=buttons -env GLUTTER_GALLERY_REBUILD=1 ./examples/gallery ;;
    navigation)
      perf "gallery, scrolling, changing pages and the theme (debug build: dev events)" 2400 -release=false -seconds "$nseconds" -starts 1 \
        -env "GLUTTER_DEV_EVENTS=$events" ./examples/gallery ;;
    *) echo "perf-macos.sh: unknown test $t (counter, gallery, animation, rebuild, navigation)" >&2; exit 64 ;;
  esac
done

if [[ -n $out ]]; then
  mkdir -p "$(dirname "$out")"
  cat "$tmp"/report.* >"$out"
  echo "wrote $out" >&2
else
  cat "$tmp"/report.*
fi
