#!/usr/bin/env bash
# Measures the Android numbers of SPEC §15 (the "Android" paragraph, step 26.5) with
# `glutter-android perf` (cmd/glutter-android/perf.go): for each app, the size of the APK per ABI
# and of its libapp.so, the cold start (am start -W, and the engine's own "first frame presented
# since_run"), the idle CPU (utime+stime from /proc/<pid>/stat over a window with no input), the
# memory (dumpsys meminfo, VmRSS, VmHWM) and the frame times (GLUTTER_FRAME_STATS=1: FPS, build and
# raster times, frames over 16 ms and the Go heap, per second). The benchmarks of the framework alone
# are `go test -bench` (docs/PERFORMANCE.md lists them).
#
# Usage: scripts/perf-android.sh [-o report.md] [-m] [-d serial] [-t tests] [-n starts] [-s seconds]
#                                [-S settle] [-g seconds] [-r renderer] [-a abis]
#
#   -o file      write the report to this file (default: standard output)
#   -m           a Markdown report, for docs/PERFORMANCE.md (default: plain text)
#   -d serial    measure on this device (a phone with USB debugging, or a running emulator).
#                Without it every measurement starts the emulator glutter-37 of the test harness
#                (internal/androidtest, a private adb server, headless) and stops it afterwards, so
#                each test boots it again: give -d for a faster run. The emulator is not a phone:
#                its CPU is the host's, its GPU is SwiftShader, so the numbers are not a device's.
#   -t tests     which measurements to run, comma separated (default: all four):
#                  counter     examples/counter, release: size, start, idle CPU, memory
#                  gallery     examples/gallery on its home page, release: the same
#                  animation   examples/gallery, release, page "data" with its progress indicators
#                              animating (GLUTTER_GALLERY_PAGE=data, GLUTTER_GALLERY_ANIMATE=1): the
#                              frame times of an animation with no input
#                  navigation  examples/gallery, a DEBUG build, driven by a dev-events script
#                              (below): the frame times of scrolling and of changing pages
#   -n starts    cold starts per app (default 5; the first is reported separately)
#   -s seconds   the idle window (default 10)
#   -S settle    the time the app runs before the idle window (default 8s)
#   -g seconds   the frame window of "animation" (default 10) and of "navigation" (this plus 14)
#   -r renderer  auto, vulkan, gles or soft (GLUTTER_RENDERER; default: the app's choice)
#   -a abis      the ABIs to build and size, comma separated (default arm64,amd64)
#
# The scenario of "navigation": GLUTTER_DEV_EVENTS is read only by debug builds (foundation.DebugMode),
# so this one measures a debug build, whose frame times are slower than a release build's (the
# checks of DebugMode run). Its script, on the 1080x2400 screen of the emulator at a density of 2.625
# (it is made of fractions of that screen, so a phone of another size needs other numbers: set
# GALLERY_EVENTS to a script of your own): scroll the home page down and up twice, open the drawer
# with its button and go to "Lists & cards", scroll it, open the drawer for "Data & progress", scroll
# it, then "Buttons", scroll it, and back to "Home". The taps on the drawer's entries assume a status
# bar of 63 to 118 px (an entry is 147 px tall): look at one frame of it with
#   go run ./cmd/glutter-android shot -screen -o /tmp/s.png -release=false \
#       -env GLUTTER_DEV_EVENTS='wait:1s;tap:74,164;wait:1s;tap:300,1487' ./examples/gallery
# before trusting a number.
#
# Environment:
#   GALLERY_EVENTS   replaces the script of the "navigation" measurement (the language is in
#                    internal/platform/android/devevents.go: wait, tap, swipe, key, text, ...)
#   ANDROID_HOME     the Android SDK, with the NDK and the build tools (SPEC §24.1)
#
# How it works: `glutter-android perf` builds each ABI alone and both together, installs the one the
# device runs, stops and starts the app over and over, and reads adb: am, /proc, dumpsys and logcat.
# Every emulator and adb server it starts is stopped when it ends. This script runs it under
# `timeout -k` and, if it is interrupted or ends, stops what it started and waits for the emulator
# and its adb server to go (`pgrep -a emulator` and `adb` show nothing afterwards).
set -euo pipefail

cd "$(dirname "$0")/.."
out=""
format=text
device=""
tests=counter,gallery,animation,navigation
starts=5
seconds=10
settle=8s
gseconds=10
renderer=""
abis=arm64,amd64
while getopts "o:md:t:n:s:S:g:r:a:" opt; do
  case "$opt" in
    o) out="$OPTARG" ;;
    m) format=markdown ;;
    d) device="$OPTARG" ;;
    t) tests="$OPTARG" ;;
    n) starts="$OPTARG" ;;
    s) seconds="$OPTARG" ;;
    S) settle="$OPTARG" ;;
    g) gseconds="$OPTARG" ;;
    r) renderer="$OPTARG" ;;
    a) abis="$OPTARG" ;;
    *) sed -n '/^# Usage:/,/^#   -a abis/p' "$0" | sed 's/^# \{0,1\}//' >&2; exit 64 ;;
  esac
done

# The dev events of the "navigation" measurement (see the header). The gallery on a phone has an
# app bar with a drawer button at the left; the drawer lists the 13 pages, 147 px apart (56 dp), the
# first at about y = 164.
BTN="tap:74,164"
down="swipe:540,1900,540,500,350"
up="swipe:540,500,540,1900,350"
scroll="$down;wait:200ms;$down;wait:200ms;$up;wait:200ms;$up;wait:400ms"
page() { echo "wait:400ms;$BTN;wait:900ms;tap:300,$((164 + 147 * $1));wait:1200ms"; }
events="${GALLERY_EVENTS:-wait:1s;$scroll;$(page 9);$scroll;$(page 10);$scroll;$(page 3);$scroll;$(page 0);wait:1s}"
# That script takes about 22 seconds; the frame window (and the idle window of that run) is -g plus 14, 24 s by default.
nseconds=$((gseconds + 14))

tmp="$(mktemp -d)"
cleaned=0
cleanup() {
  [[ $cleaned == 1 ]] && return
  cleaned=1
  trap '' INT TERM
  # Stop what this script started: timeout (and through it glutter-android, which stops its emulator
  # and adb server). The emulator needs a few seconds to go; wait for it before killing.
  pkill -TERM -P $$ 2>/dev/null || true
  for _ in $(seq 1 60); do
    pgrep -P $$ >/dev/null 2>&1 || break
    sleep 1
  done
  pkill -KILL -P $$ 2>/dev/null || true
  rm -rf "$tmp"
}
trap cleanup EXIT
trap 'exit 130' INT TERM

echo "building glutter-android..." >&2
go build -o "$tmp/glutter-android" ./cmd/glutter-android
ga="$tmp/glutter-android"

dev=()
[[ -n $device ]] && dev=(-device "$device")
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
  timeout -k 30 "$limit" "$ga" perf -format "$format" -abi "$abis" -starts "$starts" -settle "$settle" -seconds "$seconds" \
    ${dev[@]+"${dev[@]}"} ${rend[@]+"${rend[@]}"} "$@" >>"$tmp/report.$n" 2>"$tmp/err.$n" &
  if ! wait $!; then
    cat "$tmp/err.$n" >&2
    echo "perf-android.sh: the measurement \"$title\" failed" >&2
    exit 1
  fi
  echo >>"$tmp/report.$n"
}

for t in ${tests//,/ }; do
  case "$t" in
    counter) perf "counter (release)" 1800 -release=true -frames=false ./examples/counter ;;
    gallery) perf "gallery, home page (release)" 1800 -release=true -frames=false ./examples/gallery ;;
    animation)
      perf "gallery, an animation with no input (release)" 1800 -release=true -seconds "$gseconds" \
        -env GLUTTER_GALLERY_PAGE=data -env GLUTTER_GALLERY_ANIMATE=1 ./examples/gallery ;;
    navigation)
      perf "gallery, scrolling and changing pages (debug build: dev events)" 2400 -release=false -seconds "$nseconds" \
        -env "GLUTTER_DEV_EVENTS=$events" ./examples/gallery ;;
    *) echo "perf-android.sh: unknown test $t (counter, gallery, animation, navigation)" >&2; exit 64 ;;
  esac
done

if [[ -n $out ]]; then
  mkdir -p "$(dirname "$out")"
  cat "$tmp"/report.* >"$out"
  echo "wrote $out" >&2
else
  cat "$tmp"/report.*
fi
