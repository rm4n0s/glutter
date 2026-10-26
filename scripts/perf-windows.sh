#!/usr/bin/env bash
# Measures the Windows numbers of SPEC §15 (the "Windows" paragraph, step 28.15) with
# `glutter-windows perf` (cmd/glutter-windows/perf.go): for each app, the size of the .exe per
# architecture, the cold start to the first frame (the engine's own "first frame presented
# since_run", with an empty and with a warm shader cache), the idle CPU (utime+stime of the Wine
# processes of the prefix, from /proc, over a window with no input), the memory (VmRSS, VmHWM and
# Pss of those processes) and the frame times (GLUTTER_FRAME_STATS=1: FPS, build and raster times,
# frames over 16 ms and the Go heap, per second). The benchmarks of the framework alone are
# `go test -bench` (docs/PERFORMANCE.md lists them).
#
# THESE ARE WINE'S NUMBERS: the app runs under Wine on a headless Xvfb, its Direct3D 12 is vkd3d on
# the host's Vulkan (Mesa's lavapipe, a CPU rasterizer), and its CPU and memory are those of Wine's
# processes. They are for noticing regressions between steps. They say nothing about a GPU on
# Windows: the targets of SPEC §15 are confirmed on a PC (docs/MANUAL-CHECKS.md, 28.15).
#
# Usage: scripts/perf-windows.sh [-o report.md] [-m] [-t tests] [-n starts] [-s seconds] [-S settle]
#                                [-g seconds] [-r renderer] [-a archs]
#
#   -o file      write the report to this file (default: standard output)
#   -m           a Markdown report, for docs/PERFORMANCE.md (default: plain text)
#   -t tests     which measurements to run, comma separated (default: all five):
#                  counter     examples/counter, release: size, cold start, idle CPU, memory
#                  gallery     examples/gallery on its home page, release: the same
#                  animation   examples/gallery, release, page "data" with its progress indicators
#                              animating (GLUTTER_GALLERY_PAGE=data, GLUTTER_GALLERY_ANIMATE=1): the
#                              frame times of an animation with no input
#                  rebuild     examples/gallery, release, page "buttons" built again in every frame
#                              (GLUTTER_GALLERY_REBUILD=1): the build time of a full rebuild
#                  navigation  examples/gallery, a DEBUG build, driven by a dev-events script
#                              (below): the frame times of scrolling, of changing pages and of the
#                              theme switch
#   -n starts    cold starts per state of the shader cache (default 5)
#   -s seconds   the idle window (default 10)
#   -S settle    the time the app runs before the idle window (default 8s)
#   -g seconds   the frame window of "animation" and "rebuild" (default 10) and of "navigation"
#                (this plus 8)
#   -r renderer  auto, d3d12 or soft (GLUTTER_RENDERER; default d3d12)
#   -a archs     the architectures to build and size, comma separated (default amd64,arm64); amd64
#                is the one that runs under Wine
#
# The scenario of "navigation": GLUTTER_DEV_EVENTS is read only by debug builds (foundation.DebugMode),
# so this one measures a debug build, whose frame times are slower than a release build's (the checks
# of DebugMode run). Its script is made for the gallery's window of 1100x760 logical pixels: wheel
# scrolls of the home page down and up, a tap on the entries "Lists & cards", "Data & progress" and
# "Buttons" of the list at the left (44 px apart, the first, "Home", at y = 86) with a scroll of each
# page, two taps on the theme switch at the top right, and back to "Home". Set GALLERY_EVENTS to a
# script of your own (the language is in internal/platform/windows/devevents.go: wait, move, tap,
# wheel, key, text, ...). Look at one frame of it with
#   go run ./cmd/glutter-windows shot -release=false -env GLUTTER_DEV_EVENTS='wait:1s;tap:100,482;wait:1s' -o /tmp/s.png ./examples/gallery
# before trusting a number.
#
# Environment:
#   GALLERY_EVENTS   replaces the script of the "navigation" measurement
#
# How it works: `glutter-windows perf` builds each architecture, starts the amd64 .exe under Wine
# (scripts/wine-exec.sh's environment, through internal/winetest: a headless X server, a private prefix
# in .wine-test/prefix, `timeout -k`, `wineserver -k` afterwards) over and over, and reads /proc and the
# app's log. It stops what it starts. This script runs it under `timeout -k` and, when interrupted,
# stops what it started (`pgrep -a "wine|Xvfb"` shows nothing afterwards). Run one measurement at a
# time: it shares the prefix with every other Wine run, and the machine needs about 1 GB free for each
# app (lavapipe's JIT and the Wine processes: SPEC §15, docs/PERFORMANCE.md).
set -euo pipefail

cd "$(dirname "$0")/.."
out=""
format=text
tests=counter,gallery,animation,rebuild,navigation
starts=5
seconds=10
settle=8s
gseconds=10
renderer=""
archs=amd64,arm64
while getopts "o:mt:n:s:S:g:r:a:" opt; do
  case "$opt" in
    o) out="$OPTARG" ;;
    m) format=markdown ;;
    t) tests="$OPTARG" ;;
    n) starts="$OPTARG" ;;
    s) seconds="$OPTARG" ;;
    S) settle="$OPTARG" ;;
    g) gseconds="$OPTARG" ;;
    r) renderer="$OPTARG" ;;
    a) archs="$OPTARG" ;;
    *) sed -n '/^# Usage:/,/^#   -a archs/p' "$0" | sed 's/^# \{0,1\}//' >&2; exit 64 ;;
  esac
done

# The dev events of the "navigation" measurement (see the header). The gallery's window is 1100x760;
# its list of pages is at the left, an entry 44 px tall, "Home" at y = 86 (a tap at x = 100).
wheel() { echo "wheel:650,400,0,$1"; }
scroll="$(wheel 150);wait:40ms;$(wheel 150);wait:40ms;$(wheel 150);wait:40ms;$(wheel 150);wait:300ms;$(wheel -300);wait:40ms;$(wheel -300);wait:300ms"
page() { echo "tap:100,$((86 + 44 * $1));wait:1200ms"; }
theme="tap:1072,28;wait:1200ms;tap:1072,28;wait:1200ms"
events="${GALLERY_EVENTS:-wait:1s;$scroll;$(page 9);$scroll;$(page 10);$scroll;$(page 3);$scroll;$theme;$(page 0);wait:1s}"
# That script takes about 16 seconds after the first frame; the frame window of "navigation" is -g plus 8.
nseconds=$((gseconds + 8))

tmp="$(mktemp -d)"
cleaned=0
cleanup() {
  [[ $cleaned == 1 ]] && return
  cleaned=1
  trap '' INT TERM
  # Stop what this script started: timeout (and through it glutter-windows, which stops Wine and
  # Xvfb). Give it a few seconds to do so before killing.
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

echo "building glutter-windows..." >&2
go build -o "$tmp/glutter-windows" ./cmd/glutter-windows
gw="$tmp/glutter-windows"

rend=()
[[ -n $renderer ]] && rend=(-renderer "$renderer")

# perf TITLE TIMEOUT ARGS... runs one measurement into $tmp/report.N and prints a heading.
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
  timeout -k 30 "$limit" "$gw" perf -format "$format" -arch "$archs" -starts "$starts" -settle "$settle" -seconds "$seconds" \
    ${rend[@]+"${rend[@]}"} "$@" >>"$tmp/report.$n" 2>"$tmp/err.$n" &
  if ! wait $!; then
    cat "$tmp/err.$n" >&2
    echo "perf-windows.sh: the measurement \"$title\" failed" >&2
    exit 1
  fi
  echo >>"$tmp/report.$n"
}

for t in ${tests//,/ }; do
  case "$t" in
    counter) perf "counter (release)" 1800 -release=true -frames=false ./examples/counter ;;
    gallery) perf "gallery, home page (release)" 1800 -release=true -frames=false ./examples/gallery ;;
    animation)
      perf "gallery, an animation with no input (release)" 1800 -release=true -seconds "$gseconds" -starts 1 \
        -env GLUTTER_GALLERY_PAGE=data -env GLUTTER_GALLERY_ANIMATE=1 ./examples/gallery ;;
    rebuild)
      perf "gallery, a full rebuild in every frame (release)" 1800 -release=true -seconds "$gseconds" -starts 1 \
        -env GLUTTER_GALLERY_PAGE=buttons -env GLUTTER_GALLERY_REBUILD=1 ./examples/gallery ;;
    navigation)
      perf "gallery, scrolling, changing pages and the theme (debug build: dev events)" 2400 -release=false -seconds "$nseconds" -starts 1 \
        -env "GLUTTER_DEV_EVENTS=$events" ./examples/gallery ;;
    *) echo "perf-windows.sh: unknown test $t (counter, gallery, animation, rebuild, navigation)" >&2; exit 64 ;;
  esac
done

if [[ -n $out ]]; then
  mkdir -p "$(dirname "$out")"
  cat "$tmp"/report.* >"$out"
  echo "wrote $out" >&2
else
  cat "$tmp"/report.*
fi
