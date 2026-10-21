#!/usr/bin/env bash
# Measures the web numbers of SPEC §15 (the "Web" table) with release builds: the sizes of the
# counter's and the gallery's app.wasm, the time to the first frame, the idle CPU, the frame times of
# the gallery, a full rebuild of a gallery page, and the response of the gallery to clicks on its
# navigation rail, in Chromium (WebGPU) and in Firefox (WebGL 2). The benchmarks of the framework alone
# are `go test -bench` (docs/PERFORMANCE.md lists them).
#
# Usage: scripts/perf-web.sh [-s seconds] [-l loads] [-b chromium|firefox ...]
#
#   -s seconds   how long the steady state is measured (default 10)
#   -l loads     how many times the page is loaded to time the start, the first with an empty cache
#                (default 4)
#   -b browser   measure only this browser (default: both; may be repeated)
#
# Environment:
#   GALLERY_CLICKS   the points of the clicks on the gallery's navigation rail, "x,y;x,y", in CSS pixels
#                    of a 1536x864 view (the default visits every page; examples/gallery tests that they
#                    hit the entries of the rail)
#
# How it works: `glutter-web perf` (cmd/glutter-web, internal/webtest/perf.go) opens the build in a
# headless browser with a temporary profile, runs the app with GLUTTER_FRAME_STATS=1, and reads what the
# page posts: the engine's frame statistics, the calls of requestAnimationFrame and setTimeout, the CPU time
# of every process of the browser (from /proc), and, for the clicks, the frames that follow each. Every
# browser and X server it starts runs under `timeout -k` in a process group of its own and is killed when
# the measurement ends; this script stops them too if it is interrupted.
#
# The machine matters: Chromium here draws on the machine's GPU (Vulkan through ANGLE), but headless Firefox
# on Linux has no GPU context without a display and draws WebGL 2 with Mesa's software rasterizer behind a
# private Xvfb (internal/webtest/browser.go), so its CPU numbers include the rasterizer and its raster times
# are not a GPU's. Needs chromium, firefox and Xvfb (sudo dnf install chromium firefox xorg-x11-server-Xvfb).
set -euo pipefail

cd "$(dirname "$0")/.."
seconds=10
loads=4
browsers=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    -s) seconds="$2"; shift 2 ;;
    -l) loads="$2"; shift 2 ;;
    -b) browsers+=("$2"); shift 2 ;;
    *) echo "usage: $0 [-s seconds] [-l loads] [-b chromium|firefox]" >&2; exit 64 ;;
  esac
done
if [[ ${#browsers[@]} -eq 0 ]]; then
  browsers=(chromium firefox)
fi
# The rail's entries are 44 px apart from y = 86; every page once, then Home again.
clicks="${GALLERY_CLICKS:-100,218;100,306;100,526;100,614;100,482;100,438;100,394;100,262;100,350;100,570;100,130;100,174;100,86}"

tmp="$(mktemp -d)"
cleanup() {
  # Stop what this script started: the children of the shell (glutter-web, and through its context the
  # browsers and the X server), then the directory.
  pkill -TERM -P $$ 2>/dev/null || true
  sleep 0.5
  pkill -KILL -P $$ 2>/dev/null || true
  rm -rf "$tmp"
}
trap cleanup EXIT INT TERM

go build -o "$tmp/glutter-web" ./cmd/glutter-web
gw="$tmp/glutter-web"

# val FILE KEY prints the value of a "key value" line of the output of glutter-web perf.
val() { awk -v k="$2" '$1 == k { sub(/^[^ ]+ /, ""); print; exit }' "$1"; }
mb() { awk -v b="$1" 'BEGIN { printf "%.1f MB", b / 1048576 }'; }

# Sizes: raw (release, with symbols), stripped (-ldflags=-s -w), and gzip as glutter-web serves it.
echo "building the counter and the gallery for the web (release)..." >&2
declare -A raw stripped gz
for app in counter gallery; do
  GOOS=js GOARCH=wasm go build -tags glutter_release -o "$tmp/$app-raw.wasm" "./examples/$app"
  raw[$app]=$(stat -c %s "$tmp/$app-raw.wasm")
  "$gw" build -release -o "$tmp/$app-site" "./examples/$app" >/dev/null
  stripped[$app]=$(stat -c %s "$tmp/$app-site/app.wasm")
  gz[$app]=$(stat -c %s "$tmp/$app-site/app.wasm.gz")
  rm -f "$tmp/$app-raw.wasm"
done

perf() { # name browser renderer app extra-args...
  local name="$1" browser="$2" renderer="$3" app="$4"
  shift 4
  echo "  $browser: $name..." >&2
  timeout -k 10 400 "$gw" perf -browser "$browser" -renderer "$renderer" -release -loads "$loads" -measure "${seconds}s" "$@" "./examples/$app" >"$tmp/$browser-$name.txt" 2>"$tmp/$browser-$name.err" || {
    cat "$tmp/$browser-$name.err" >&2
    echo "perf-web.sh: the measurement $name in $browser failed" >&2
    exit 1
  }
}

for browser in "${browsers[@]}"; do
  renderer=webgpu
  [[ $browser == firefox ]] && renderer=webgl
  echo "measuring in $browser ($renderer)..." >&2
  timeout -k 10 200 "$gw" perf -browser "$browser" -blank -measure "${seconds}s" >"$tmp/$browser-blank.txt"
  perf idle "$browser" "$renderer" counter
  perf animation "$browser" "$renderer" gallery -query "GLUTTER_GALLERY_PAGE=data&GLUTTER_GALLERY_ANIMATE=1"
  perf rebuild "$browser" "$renderer" gallery -query "GLUTTER_GALLERY_PAGE=buttons&GLUTTER_GALLERY_REBUILD=1"
  # The rail is clicked twice through: the first time each page is built for the first time.
  perf clicks "$browser" "$renderer" gallery -clicks "$clicks;$clicks"
done

echo
echo "| App | app.wasm (release, with symbols) | app.wasm (-ldflags=-s -w) | app.wasm.gz (as served) |"
echo "|---|---|---|---|"
for app in counter gallery; do
  echo "| $app | $(mb "${raw[$app]}") (${raw[$app]} bytes) | $(mb "${stripped[$app]}") | $(mb "${gz[$app]}") (${gz[$app]} bytes) |"
done
echo
echo "| Metric | Target (SPEC §15, web) | $(printf '%s | ' "${browsers[@]}" | sed 's/ | $//') |"
echo "|---|---|$(printf -- '---|%.0s' "${browsers[@]}")"
row() { # label target command-printing-a-cell-per-browser
  local label="$1" target="$2" fn="$3" cells=""
  for b in "${browsers[@]}"; do cells+=" $($fn "$b") |"; done
  echo "| $label | $target |$cells"
}
cell_renderer() { echo "$(val "$tmp/$1-idle.txt" renderer)" ; }
cell_idle() {
  local b="$1"
  echo "$(val "$tmp/$b-idle.txt" cpu_percent)% of a core (a blank page: $(val "$tmp/$b-blank.txt" cpu_percent)%), $(val "$tmp/$b-idle.txt" raf_calls) requestAnimationFrame and $(val "$tmp/$b-idle.txt" timer_calls) timers in $(val "$tmp/$b-idle.txt" window_s) s"
}
cell_anim_fps() {
  local b="$1" f="$tmp/$1-animation.txt"
  echo "$(val "$f" fps) frames per second; the browser's frame interval: mean $(val "$f" raf_interval_mean_ms) ms, p99 $(val "$f" raf_interval_p99_ms) ms, max $(val "$f" raf_interval_max_ms) ms, $(val "$f" raf_intervals_over_16.7ms) of $(val "$f" raf_intervals) over 16.7 ms"
}
cell_anim_cpu() {
  local f="$tmp/$1-animation.txt"
  echo "$(val "$f" ui_avg_ms) ms (build $(val "$f" build_avg_ms), raster $(val "$f" raster_avg_ms)); worst second $(val "$f" ui_worst_second_avg_ms) ms; worst frame $(val "$f" build_max_ms) + $(val "$f" raster_max_ms) ms"
}
cell_rebuild() {
  local f="$tmp/$1-rebuild.txt"
  echo "$(val "$f" ui_avg_ms) ms (build $(val "$f" build_avg_ms), raster $(val "$f" raster_avg_ms)); worst second $(val "$f" ui_worst_second_avg_ms) ms"
}
cell_start() {
  local f="$tmp/$1-idle.txt"
  echo "$(val "$f" first_frame_warm_ms) ms (median of $(val "$f" first_frame_warm_loads) loads); empty cache: $(val "$f" first_frame_cold_ms) ms"
}
cell_gallery_start() {
  local f="$tmp/$1-animation.txt"
  echo "$(val "$f" first_frame_warm_ms) ms; empty cache: $(val "$f" first_frame_cold_ms) ms"
}
cell_clicks() {
  local f="$tmp/$1-clicks.txt"
  awk '/^click / { n++; l[n] = $5; mf[n] = $7 }
       function stats(from, to,   i, c, a, s, m, w) {
         c = 0; s = 0; m = 0; w = 0
         for (i = from; i <= to; i++) { a[++c] = l[i]; s += l[i]; if (l[i] > m) m = l[i]; if (mf[i] > w) w = mf[i] }
         asort(a)
         return sprintf("median %.0f ms, mean %.0f ms, worst %.0f ms (longest single frame %.0f ms)", a[int((c + 1) / 2)], s / c, m, w)
       }
       END { h = n / 2; printf "first visit of each page (%d clicks): %s; second visit (%d clicks): %s", h, stats(1, h), h, stats(h + 1, n) }' "$f"
}
row "Renderer" "" cell_renderer
row "Idle CPU, counter" "~0%: nothing asked for while nothing is scheduled" cell_idle
row "Gallery animation: frame rate" "the display's refresh rate with no dropped frames" cell_anim_fps
row "UI thread per frame, animation only" "< 4 ms" cell_anim_cpu
row "UI thread per frame, full rebuild of a gallery page" "< 16 ms" cell_rebuild
row "Start to first frame, counter, app.wasm cached" "< 1 s" cell_start
row "Start to first frame, gallery, app.wasm cached" "< 1 s" cell_gallery_start
row "Gallery navigation: from the release of the click to the end of the next frame" "(no target)" cell_clicks
echo
echo "Raw output of every measurement:"
for f in "$tmp"/*.txt; do
  echo "--- $(basename "$f" .txt)"
  cat "$f"
done
