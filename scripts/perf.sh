#!/usr/bin/env bash
# Measures the numbers of SPEC §15 that need a running app: the idle CPU, the time to the
# first frame, the memory of the counter example, and the frames of the gallery, on this
# machine, under a headless Sway. The benchmarks of the framework alone are
# `go test -bench` (docs/PERFORMANCE.md lists the commands).
#
# Usage: scripts/perf.sh [-n runs] [-s seconds]
#
#   -n runs      how many times the counter is started to measure the cold start (default 5)
#   -s seconds   how long the idle CPU and the gallery's frames are measured (default 10)
#
# Environment:
#   PERF_TAGS    build tags of the examples (default glutter_release: no debug checks)
#   HEADLESS_SIZE, HEADLESS_SCALE   the output (default 1920x1080 at 1.25, the display of
#                SPEC §15); the windows are tiled over it
#
# Needs sway (see scripts/headless-wayland.sh). The software renderer of Mesa draws the
# frames here, so the raster times are not those of a GPU; the other numbers are.
set -euo pipefail

cd "$(dirname "$0")/.."
runs=5
seconds=10
inner=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    -n) runs="$2"; shift 2 ;;
    -s) seconds="$2"; shift 2 ;;
    --inner) inner=1; shift ;;
    *) echo "usage: $0 [-n runs] [-s seconds]" >&2; exit 64 ;;
  esac
done
tags="${PERF_TAGS:-glutter_release}"

if [[ $inner -eq 0 ]]; then
  bin="$(mktemp -d)"
  trap 'rm -rf "$bin"' EXIT
  go build -tags "$tags" -o "$bin/counter" ./examples/counter
  go build -tags "$tags" -o "$bin/gallery" ./examples/gallery
  export PERF_BIN="$bin" PERF_RUNS="$runs" PERF_SECONDS="$seconds" PERF_TAGS="$tags"
  export HEADLESS_SIZE="${HEADLESS_SIZE:-1920x1080}" HEADLESS_SCALE="${HEADLESS_SCALE:-1.25}" HEADLESS_FLOATING=0
  scripts/headless-wayland.sh -- "$0" --inner
  exit $?
fi

runs="$PERF_RUNS"
seconds="$PERF_SECONDS"
bin="$PERF_BIN"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# duration_ms converts a Go duration such as 1.5s, 120ms or 85.2µs to milliseconds.
duration_ms() {
  awk -v d="$1" 'BEGIN {
    if (d ~ /µs$/) { sub(/µs$/, "", d); v = d / 1000 }
    else if (d ~ /ms$/) { sub(/ms$/, "", d); v = d + 0 }
    else if (d ~ /s$/) { sub(/s$/, "", d); v = d * 1000 }
    else v = d
    printf "%.2f", v }'
}

# field extracts key=value from a log line.
field() { sed -n "s/.* $2=\\([^ ]*\\).*/\\1/p" <<<"$1" | head -1; }

# start_app starts a binary in the background with the log in $tmp/$1.log, waits for its
# first frame, and sets pid.
start_app() {
  local name="$1"
  shift
  env GLUTTER_FRAME_STATS=1 "$@" "$bin/$name" >"$tmp/$name.log" 2>&1 &
  pid=$!
  for _ in $(seq 1 300); do
    grep -q "first frame presented" "$tmp/$name.log" 2>/dev/null && return 0
    kill -0 "$pid" 2>/dev/null || { cat "$tmp/$name.log" >&2; echo "$name stopped before its first frame" >&2; exit 1; }
    sleep 0.1
  done
  echo "$name drew no frame in 30 seconds" >&2
  exit 1
}

stop_app() {
  kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
}

# cpu_ticks is the user and system time of the process in clock ticks.
cpu_ticks() { awk '{ sub(/^.*\) /, ""); print $12 + $13 }' "/proc/$pid/stat"; }

# Cold start: the counter, several times; the median of the time from the start of the
# process to its first frame, and of the time from the start of Run. Once with an empty
# shader cache (a new directory for each start: every program is compiled) and once with a
# warm one (a first start fills it, SPEC §9.7). The parts of the first draw come from the log
# of the last start (GLUTTER_FRAME_STATS=1): the renderers tried and failed, the GPU context,
# the shaders, and the rest of the first frame.
median() { printf '%s\n' "$@" | sort -n | awk '{ v[NR] = $1 } END { print v[int((NR + 1) / 2)] }'; }

# stage_took prints the "took" of the log line of a startup stage, in milliseconds; 0 when
# there is none. stage_sum adds those of all the lines of the stage (several renderers tried).
stage_line() { grep "stage=$2 " "$tmp/$1.log" 2>/dev/null || grep "stage=\"$2\" " "$tmp/$1.log" 2>/dev/null || true; }
stage_sum() {
  local total=0 line
  while IFS= read -r line; do
    [[ -n "$line" ]] && total="$(awk -v a="$total" -v b="$(duration_ms "$(field "$line" took)")" 'BEGIN { printf "%.1f", a + b }')"
  done < <(stage_line "$1" "$2")
  echo "$total"
}

cache_root="$tmp/shader-cache"
# cold_start runs the counter $runs times and sets since_process and since_run. The mode says
# what the caches hold: "empty" is a machine that never ran the app (Glutter's cache and the
# driver's own cache, Mesa's, both empty), "driver" has the driver's cache warm and Glutter's
# empty (what a start after Glutter's cache was cleared sees), and "warm" has both warm. Mesa's
# cache is moved with MESA_SHADER_CACHE_DIR so that the modes do not touch the user's.
cold_start() {
  local mode="$1" dir mesa i line
  since_process=()
  since_run=()
  case "$mode" in
    warm) dir="$cache_root/warm"; mesa="$cache_root/warm-mesa" ;;
    driver) mesa="$cache_root/driver-mesa" ;;
  esac
  if [[ "$mode" == warm || "$mode" == driver ]]; then
    # A first start fills the caches that the runs below find.
    start_app counter XDG_CACHE_HOME="${dir:-$cache_root/prime}" MESA_SHADER_CACHE_DIR="$mesa"
    stop_app
  fi
  for i in $(seq 1 "$runs"); do
    [[ "$mode" != warm ]] && dir="$cache_root/$mode-$i"
    [[ "$mode" == empty ]] && mesa="$cache_root/empty-mesa-$i"
    start_app counter XDG_CACHE_HOME="$dir" MESA_SHADER_CACHE_DIR="$mesa"
    line="$(grep "first frame presented" "$tmp/counter.log" | head -1)"
    since_process+=("$(duration_ms "$(field "$line" since_process_start)")")
    since_run+=("$(duration_ms "$(field "$line" since_run)")")
    stop_app
  done
  cp "$tmp/counter.log" "$tmp/cold-$mode.log"
}
cold_start empty
cold_process="$(median "${since_process[@]}")"
cold_run="$(median "${since_run[@]}")"
cold_start driver
driver_process="$(median "${since_process[@]}")"
driver_run="$(median "${since_run[@]}")"
cold_start warm
warm_process="$(median "${since_process[@]}")"
warm_run="$(median "${since_run[@]}")"

# part prints a part of the first draw of the last start of a mode, in milliseconds.
part() {
  local mode="$1" what="$2" line
  cp "$tmp/cold-$mode.log" "$tmp/part.log"
  case "$what" in
    tried) stage_sum part "\"renderer tried\"" ;;
    context) stage_sum part "\"GPU context or device\"" ;;
    shaders) stage_sum part shaders ;;
    first) stage_sum part "\"frame without the above\"" ;;
    draw) line="$(grep "stage=\"first draw" "$tmp/part.log" | head -1)"; duration_ms "$(field "$line" took)" ;;
  esac
}
shader_counts() {
  local line
  cp "$tmp/cold-$1.log" "$tmp/part.log"
  line="$(grep "stage=shaders " "$tmp/part.log" | head -1)"
  echo "$(field "$line" built) built, $(field "$line" from_cache) from the cache"
}

# Idle CPU and memory: the counter, settled, with nothing happening.
start_app counter
sleep 3
go_heap="$(field "$(grep "first frame presented" "$tmp/counter.log" | head -1)" go_heap_mb)"
go_sys="$(field "$(grep "first frame presented" "$tmp/counter.log" | head -1)" go_sys_mb)"
rss_kb="$(awk '/^VmRSS:/ { print $2 }' "/proc/$pid/status")"
hwm_kb="$(awk '/^VmHWM:/ { print $2 }' "/proc/$pid/status")"
t0="$(cpu_ticks)"
sleep "$seconds"
t1="$(cpu_ticks)"
idle_cpu="$(awk -v a="$t0" -v b="$t1" -v s="$seconds" 'BEGIN { printf "%.2f", (b - a) / (s * 100) * 100 }')"
frames_idle="$(grep -c " INFO frames " "$tmp/counter.log" || true)"
stop_app

# Idle CPU of a window that is not the active one: the counter, with a second window
# that the compositor activates (the newest one), so the counter has lost the focus. The
# view focus is the activated state of the toplevel, so this runs headless too (SPEC §8.2).
start_app counter
counter_pid="$pid"
start_app gallery
other_pid="$pid"
sleep 3
pid="$counter_pid"
t0="$(cpu_ticks)"
sleep "$seconds"
t1="$(cpu_ticks)"
unfocused_cpu="$(awk -v a="$t0" -v b="$t1" -v s="$seconds" 'BEGIN { printf "%.2f", (b - a) / (s * 100) * 100 }')"
frames_unfocused="$(grep -c " INFO frames " "$tmp/counter.log" || true)"
pid="$other_pid"
stop_app
pid="$counter_pid"
stop_app

# Frames: the gallery on the page with indeterminate progress indicators.
start_app gallery GLUTTER_GALLERY_PAGE=data GLUTTER_GALLERY_ANIMATE=1
sleep 2
mark="$(grep -c " INFO frames " "$tmp/gallery.log" || true)"
g0="$(cpu_ticks)"
sleep "$seconds"
g1="$(cpu_ticks)"
stop_app
gallery_cpu="$(awk -v a="$g0" -v b="$g1" -v s="$seconds" 'BEGIN { printf "%.1f", (b - a) / (s * 100) * 100 }')"
stats="$({ grep " INFO frames " "$tmp/gallery.log" || true; } | tail -n +"$((mark + 1))")"
summary="$(awk '
  function ms(d) { if (d ~ /µs$/) { sub(/µs$/, "", d); return d / 1000 } if (d ~ /ms$/) { sub(/ms$/, "", d); return d + 0 } sub(/s$/, "", d); return d * 1000 }
  { for (i = 1; i <= NF; i++) { split($i, kv, "="); v[kv[1]] = kv[2] }
    n++; fps += v["fps"]; slow += v["over_16ms"]
    b += ms(v["build_avg"]); r += ms(v["raster_avg"])
    if (ms(v["build_max"]) > bm) bm = ms(v["build_max"])
    if (ms(v["raster_max"]) > rm) rm = ms(v["raster_max"]) }
  END { if (n) printf "%.1f %d %.2f %.2f %.2f %.2f", fps / n, slow, b / n, bm, r / n, rm; else printf "0 0 0 0 0 0" }' <<<"$stats")"
read -r g_fps g_slow g_build g_build_max g_raster g_raster_max <<<"$summary"

cat <<TABLE
| Metric | Target (SPEC §15) | Measured |
|---|---|---|
| Idle CPU, counter ($seconds s, no input) | ~0% | ${idle_cpu}% (${frames_idle} frames logged while idle) |
| Idle CPU, counter without the focus (another window is active, $seconds s) | ~0% | ${unfocused_cpu}% (${frames_unfocused} frames logged since the start) |
| Cold start to first frame, counter, empty caches: Glutter's and the driver's (median of $runs, from the start of the process) | < 150 ms | ${cold_process} ms (from the start of Run: ${cold_run} ms) |
| Cold start to first frame, counter, the driver's cache warm and Glutter's empty (median of $runs) | < 150 ms | ${driver_process} ms (from the start of Run: ${driver_run} ms) |
| Cold start to first frame, counter, both caches warm (median of $runs) | < 150 ms | ${warm_process} ms (from the start of Run: ${warm_run} ms) |
| Resident memory, counter | < 60 MB | $((rss_kb / 1024)) MB (peak $((hwm_kb / 1024)) MB), of which the Go runtime asked the system for ${go_sys} MB (heap in use ${go_heap} MB) and the rest is the graphics driver and the executable |
| Gallery, data page with indeterminate indicators, $seconds s | the display's refresh rate, no dropped frames | ${g_fps} frames per second, ${g_slow} frames over 16.7 ms, CPU ${gallery_cpu}% |
| Gallery, build time per frame (average, worst) | UI thread < 2 ms for an animation-only frame | ${g_build} ms, ${g_build_max} ms |
| Gallery, raster time per frame (average, worst), Mesa software renderer | (not a SPEC §15 number: no GPU here) | ${g_raster} ms, ${g_raster_max} ms |
TABLE

cat <<PARTS

| Part of the first draw, last start (ms) | Empty caches | Driver's cache warm | Both warm |
|---|---|---|---|
| Renderers tried that failed (Vulkan without a presentable GPU) | $(part empty tried) | $(part driver tried) | $(part warm tried) |
| GPU context or device | $(part empty context) | $(part driver context) | $(part warm context) |
| Shaders ($(shader_counts empty); $(shader_counts driver); $(shader_counts warm)) | $(part empty shaders) | $(part driver shaders) | $(part warm shaders) |
| The frame itself | $(part empty first) | $(part driver first) | $(part warm first) |
| First draw, all of it | $(part empty draw) | $(part driver draw) | $(part warm draw) |
PARTS
