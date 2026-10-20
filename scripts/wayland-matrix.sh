#!/usr/bin/env bash
# The automated part of the Wayland compatibility matrix (SPEC §17): runs the
# display tests and the examples under every headless compositor that is
# installed (sway, weston, `mutter --headless`, `kwin_wayland --virtual`), prints
# a pass/fail table and lists the compositors that were skipped because they are
# missing. The rest of the matrix (real desktops, GPU drivers, scales, two
# monitors) is the manual checklist in docs/wayland-checklist.md.
#
# Usage: scripts/wayland-matrix.sh [options]
#   --compositors LIST  comma-separated, from sway,weston,mutter,kwin (default: all)
#   --examples LIST     comma-separated example names (default: every example in
#                       examples/ except examples/dev)
#   --no-tests          skip the display tests
#   --no-examples       skip the examples
#   --out DIR           keep the screenshots and logs in DIR (default: a temporary
#                       directory that is removed at the end unless a run fails)
#
# For each compositor, started by scripts/headless-wayland.sh on a private session
# bus:
#   tests     go test -tags glutter_display for internal/platform/linux/wayland and
#             internal/gpu/gles (compiled once, run under each compositor)
#   examples  each example is built once and run with GLUTTER_SCREENSHOT=<png>; it
#             passes when it exits with 0 and leaves a PNG that is not empty
#
# weston's headless backend has no input devices and so no wl_seat, which Glutter
# requires: there the matrix checks that every example refuses to start with
# WaylandProtocolMissing (a clean error and exit status 1, not a crash), and the display
# tests, which need a seat, are skipped. The table says "seatless" in that row.
#
# Environment: MATRIX_TIMEOUT is the time limit of one test binary or example in
# seconds (default 120). HEADLESS_SIZE is passed on to headless-wayland.sh.
#
# Exit status: 0 when every row passes; 1 when something failed; 2 when no
# compositor is installed or a tool is missing; 64 for a usage mistake.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/.." && pwd)"
timeout_s="${MATRIX_TIMEOUT:-120}"
all_compositors=(sway weston mutter kwin)
test_packages=(./internal/platform/linux/wayland ./internal/gpu/gles)

binary_of() {
  case "$1" in
    kwin) echo kwin_wayland ;;
    *) echo "$1" ;;
  esac
}

# --- the inner mode: runs inside one compositor and writes its results ---------
# usage: wayland-matrix.sh --inner <compositor> <workdir> <examples,...> <tests 0|1>
if [[ "${1:-}" == "--inner" ]]; then
  comp="$2"
  work="$3"
  IFS=, read -r -a examples <<<"$4"
  run_tests="$5"
  results="$work/$comp.results"
  : >"$results"
  mkdir -p "$work/$comp"
  record() { printf '%s\t%s\t%s\n' "$1" "$2" "$3" >>"$results"; }

  if [[ "$run_tests" == 1 ]]; then
    for pkg in "${test_packages[@]}"; do
      name="$(basename "$pkg")"
      log="$work/$comp/test-$name.log"
      if [[ "$comp" == weston ]]; then
        record "test:$name" SKIP "needs a wl_seat"
        continue
      fi
      if (cd "$root/$pkg" && timeout "$timeout_s" "$work/tests/$name.test" -test.count=1 >"$log" 2>&1); then
        record "test:$name" PASS ""
      else
        record "test:$name" FAIL "see $log"
      fi
    done
  fi

  for ex in "${examples[@]}"; do
    [[ -n "$ex" ]] || continue
    png="$work/$comp/$ex.png"
    log="$work/$comp/$ex.log"
    set +e
    GLUTTER_BACKEND=wayland GLUTTER_SCREENSHOT="$png" GLUTTER_SCREENSHOT_AFTER=300ms \
      timeout "$timeout_s" "$work/bin/$ex" >"$log" 2>&1
    status=$?
    set -e
    if [[ "$comp" == weston ]]; then
      if [[ $status -eq 1 ]] && grep -q "does not offer a required Wayland protocol" "$log"; then
        record "example:$ex" PASS "refused: no wl_seat"
      else
        record "example:$ex" FAIL "expected the refusal for the missing wl_seat (exit 1), got exit $status, see $log"
      fi
      continue
    fi
    if [[ $status -eq 0 && -s "$png" ]] && head -c 8 "$png" | grep -q "PNG"; then
      record "example:$ex" PASS ""
    else
      record "example:$ex" FAIL "exit $status, see $log"
    fi
  done
  exit 0
fi

# --- the outer mode --------------------------------------------------------------
usage() {
  sed -n '2,/^set -euo/p' "${BASH_SOURCE[0]}" | sed '$d' | sed 's/^# \{0,1\}//' >&2
  exit 64
}

selected=("${all_compositors[@]}")
example_list=""
run_tests=1
run_examples=1
out=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --compositors) IFS=, read -r -a selected <<<"${2:?}"; shift 2 ;;
    --examples) example_list="${2:?}"; shift 2 ;;
    --no-tests) run_tests=0; shift ;;
    --no-examples) run_examples=0; shift ;;
    --out) out="${2:?}"; shift 2 ;;
    -h | --help) usage ;;
    *) echo "error: unknown option $1" >&2; usage ;;
  esac
done
for c in "${selected[@]}"; do
  case "$c" in sway | weston | mutter | kwin) ;; *) echo "error: unknown compositor '$c'" >&2; exit 64 ;; esac
done

if [[ -z "$example_list" ]]; then
  example_list="$(cd "$root/examples" && for d in */; do d="${d%/}"; [[ "$d" == dev ]] && continue; [[ -f "$d/main.go" ]] && echo "$d"; done | paste -sd, -)"
fi
if [[ "$run_examples" == 0 ]]; then
  example_list=""
fi

present=()
skipped=()
for c in "${selected[@]}"; do
  if command -v "$(binary_of "$c")" >/dev/null 2>&1; then
    present+=("$c")
  else
    skipped+=("$c")
  fi
done
if [[ ${#present[@]} -eq 0 ]]; then
  echo "error: none of ${selected[*]} is installed." >&2
  echo "       install what is needed with: sudo dnf install sway weston mutter kwin" >&2
  exit 2
fi
if ! command -v go >/dev/null 2>&1; then
  echo "error: go is not installed." >&2
  exit 2
fi

keep=0
if [[ -n "$out" ]]; then
  mkdir -p "$out"
  work="$(cd "$out" && pwd)"
  keep=1
else
  work="$(mktemp -d "${TMPDIR:-/tmp}/glutter-matrix.XXXXXX")"
fi
mkdir -p "$work/bin" "$work/tests"

# Build everything once; the compositors only run the binaries.
cd "$root"
if [[ "$run_tests" == 1 ]]; then
  for pkg in "${test_packages[@]}"; do
    echo "building the display tests of $pkg ..." >&2
    go test -c -tags glutter_display -o "$work/tests/$(basename "$pkg").test" "$pkg"
  done
fi
if [[ -n "$example_list" ]]; then
  IFS=, read -r -a examples <<<"$example_list"
  for ex in "${examples[@]}"; do
    echo "building examples/$ex ..." >&2
    go build -o "$work/bin/$ex" "./examples/$ex"
  done
fi

for c in "${present[@]}"; do
  echo "running under $c ..." >&2
  set +e
  HEADLESS_COMPOSITOR="$c" HEADLESS_PRIVATE_BUS=1 "$here/headless-wayland.sh" -- \
    "${BASH_SOURCE[0]}" --inner "$c" "$work" "$example_list" "$run_tests" >"$work/$c.run.log" 2>&1
  status=$?
  set -e
  if [[ $status -ne 0 ]]; then
    printf '%s\t%s\t%s\n' "compositor" FAIL "did not run (exit $status), see $work/$c.run.log" >>"$work/$c.results"
  fi
done

# --- the table -------------------------------------------------------------------
failed=0
printf '\n%-9s %-9s %-9s %-14s %s\n' compositor "tests" examples notes ""
printf '%-9s %-9s %-9s %-14s\n' --------- --------- -------- --------------
for c in "${present[@]}"; do
  results="$work/$c.results"
  tp=0 tf=0 ts=0 ep=0 ef=0 other_fail=0
  details=()
  if [[ -f "$results" ]]; then
    while IFS=$'\t' read -r task status detail; do
      case "$task" in
        test:*) case "$status" in PASS) tp=$((tp + 1)) ;; SKIP) ts=$((ts + 1)) ;; *) tf=$((tf + 1)); details+=("$task: $detail") ;; esac ;;
        example:*) case "$status" in PASS) ep=$((ep + 1)) ;; *) ef=$((ef + 1)); details+=("$task: $detail") ;; esac ;;
        *) other_fail=$((other_fail + 1)); details+=("$task: $detail") ;;
      esac
    done <"$results"
  else
    other_fail=1
    details+=("no results, see $work/$c.run.log")
  fi
  tcol="$tp pass"
  [[ $tf -gt 0 ]] && tcol="$tf FAIL"
  [[ $((tp + tf)) -eq 0 && $ts -gt 0 ]] && tcol="skipped"
  [[ $((tp + tf + ts)) -eq 0 ]] && tcol="-"
  ecol="$ep pass"
  [[ $ef -gt 0 ]] && ecol="$ef FAIL"
  [[ $((ep + ef)) -eq 0 ]] && ecol="-"
  note=""
  [[ "$c" == weston ]] && note="seatless"
  row="ok"
  if [[ $((tf + ef + other_fail)) -gt 0 ]]; then
    row="FAIL"
    failed=1
  fi
  printf '%-9s %-9s %-9s %-14s %s\n' "$c" "$tcol" "$ecol" "$note" "$row"
  for d in "${details[@]+"${details[@]}"}"; do
    printf '          %s\n' "$d"
  done
done
if [[ ${#skipped[@]} -gt 0 ]]; then
  printf '\nskipped (not installed): %s\n' "${skipped[*]}"
fi
printf '\nexamples run: %s\n' "${example_list:-none}"
printf 'screenshots and logs: %s\n' "$work"

if [[ $failed -eq 0 && $keep -eq 0 ]]; then
  rm -rf "$work"
  echo "(removed; use --out DIR to keep them)"
fi
exit "$failed"
