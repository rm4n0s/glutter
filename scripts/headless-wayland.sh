#!/usr/bin/env bash
# Runs a command inside a headless Wayland compositor and stops the compositor
# afterwards. It is how the display tests and the proof screenshots run without
# a real screen.
#
# Usage: scripts/headless-wayland.sh -- <command> [args...]
#
# Inside the command:
#   WAYLAND_DISPLAY   the socket name of the headless compositor
#   XDG_RUNTIME_DIR   a private directory that holds the socket
#   SWAYSOCK          sway's IPC socket (only with sway), for swaymsg
#   DISPLAY           unset, so nothing falls back to the real X server
#
# The compositor is sway when it is installed, else weston. Prefer sway: weston's
# headless backend has no input devices and so no wl_seat, which Glutter requires
# (it fails with WaylandProtocolMissing), so weston only serves checks that do
# not need Glutter to start. The compositor's own output goes to
# $RUNTIME/compositor.log, which is printed if it fails to start. Environment:
#   HEADLESS_SIZE   output size, WIDTHxHEIGHT (default 1280x800)
#   HEADLESS_SCALE  output scale (default 1; sway only)
#   HEADLESS_FLOATING  1 (default) opens windows floating at the size they ask
#                   for; 0 lets sway tile them over the whole output (sway only)
#   WLR_RENDERER    wlroots renderer for sway (default pixman, needs no GPU)
#
# Exit status: the command's own; 2 when no compositor is installed; 3 when the
# compositor did not start; 64 for a usage mistake.
set -euo pipefail

usage() {
  echo "usage: $0 -- <command> [args...]" >&2
  exit 64
}

[[ "${1:-}" == "--" ]] || usage
shift
[[ $# -gt 0 ]] || usage

if command -v sway >/dev/null 2>&1; then
  compositor=sway
elif command -v weston >/dev/null 2>&1; then
  compositor=weston
else
  echo "error: neither sway nor weston is installed." >&2
  echo "       install them with: sudo dnf install sway grim weston" >&2
  exit 2
fi

size="${HEADLESS_SIZE:-1280x800}"
scale="${HEADLESS_SCALE:-1}"
floating="${HEADLESS_FLOATING:-1}"

# A private runtime directory keeps the socket away from the user's own session.
# Wayland socket paths are short-limited, so the directory is kept short too.
runtime="$(mktemp -d "${TMPDIR:-/tmp}/glutter-wl.XXXXXX")"
chmod 700 "$runtime"
log="$runtime/compositor.log"
pid=""

cleanup() {
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
    kill "$pid" 2>/dev/null || true
    # Give the compositor a moment, then insist.
    for _ in 1 2 3 4 5 6 7 8 9 10; do
      kill -0 "$pid" 2>/dev/null || break
      sleep 0.1
    done
    kill -9 "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  fi
  rm -rf "$runtime"
}
trap cleanup EXIT

export XDG_RUNTIME_DIR="$runtime"
unset WAYLAND_DISPLAY WAYLAND_SOCKET SWAYSOCK I3SOCK DISPLAY

case "$compositor" in
  sway)
    # An explicit config keeps the user's own sway config out of the run.
    cat >"$runtime/sway.conf" <<EOF
xwayland disable
output HEADLESS-1 resolution $size scale $scale
output * bg #3b3b3b solid_color
EOF
    if [[ "$floating" == 1 ]]; then
      # Every window floats, at the size it asks for.
      echo 'for_window [title=".*"] floating enable' >>"$runtime/sway.conf"
      echo 'for_window [app_id=".*"] floating enable' >>"$runtime/sway.conf"
    fi
    WLR_BACKENDS=headless WLR_LIBINPUT_NO_DEVICES=1 WLR_RENDERER="${WLR_RENDERER:-pixman}" \
      sway -c "$runtime/sway.conf" >"$log" 2>&1 &
    pid=$!
    ;;
  weston)
    : >"$runtime/weston.ini"
    weston --backend=headless --renderer=pixman --socket=glutter-headless \
      --width="${size%x*}" --height="${size#*x}" --idle-time=0 \
      --config="$runtime/weston.ini" >"$log" 2>&1 &
    pid=$!
    ;;
esac

# Wait for the socket, up to 10 seconds.
socket=""
for _ in $(seq 1 100); do
  if ! kill -0 "$pid" 2>/dev/null; then
    break
  fi
  for candidate in "$runtime"/wayland-* "$runtime"/glutter-headless; do
    # Skip lock files and anything that is not a socket.
    if [[ -S "$candidate" ]]; then
      socket="$(basename "$candidate")"
      break 2
    fi
  done
  sleep 0.1
done

if [[ -z "$socket" ]]; then
  echo "error: $compositor did not start a Wayland socket. Its output:" >&2
  sed 's/^/  /' "$log" >&2 || true
  exit 3
fi

export WAYLAND_DISPLAY="$socket"
if [[ "$compositor" == sway ]]; then
  for sock in "$runtime"/sway-ipc.*.sock; do
    [[ -S "$sock" ]] && export SWAYSOCK="$sock"
  done
fi

# Run the command and keep its exit status; the trap stops the compositor.
set +e
"$@"
status=$?
exit "$status"
