#!/usr/bin/env bash
# Runs a command with a private audio server and stops the server afterwards. It is how the Linux
# audio tests run without touching the sound of the machine (docs/SPEC.md §28.8).
#
# Usage: scripts/headless-audio.sh -- <command> [args...]
#
# The server is PipeWire with its PulseAudio protocol (pipewire, wireplumber and pipewire-pulse, which
# is what current desktops run), started in a private runtime directory with a null sink, whose
# monitor is the recording of what is played, and a null source; no sound card is opened and the
# session bus is not used. Inside the command:
#   PULSE_SERVER            unix:<runtime>/pulse/native, the server of the run
#   XDG_RUNTIME_DIR         the private runtime directory (PIPEWIRE_RUNTIME_DIR is the same)
#   DBUS_SESSION_BUS_ADDRESS  a bus that does not exist, so nothing reaches the user's session
#   GLUTTER_AUDIO_SERVER_RUNTIME  the runtime directory, for tests that stop and start the server
#                           again (see below)
# The names of the devices are glutter_sink (its monitor is glutter_sink.monitor) and glutter_source.
# HEADLESS_AUDIO_LOGS=<dir> copies the logs to <dir> when the run ends.
# The output of the three programs goes to <runtime>/*.log, which is printed when the server does not
# start. Every program runs in its own process group (setsid) and the groups are killed at the end.
#
# The server can be stopped and started again from inside the command, for the tests of a server
# that is lost:
#   $GLUTTER_AUDIO_CONTROL stop      stops pipewire-pulse (the socket goes)
#   $GLUTTER_AUDIO_CONTROL start     starts it again
#
# Exit status: the command's own; 2 when a program is missing; 3 when the server did not start; 64
# for a usage mistake.
set -euo pipefail

usage() {
  echo "usage: $0 -- <command> [args...]" >&2
  exit 64
}

[[ "${1:-}" == "--" ]] || usage
shift
[[ $# -gt 0 ]] || usage

for program in pipewire wireplumber pipewire-pulse; do
  if ! command -v "$program" >/dev/null 2>&1; then
    echo "error: $program is not installed. install it with: sudo dnf install pipewire pipewire-pulseaudio wireplumber" >&2
    exit 2
  fi
done

runtime="$(mktemp -d "${TMPDIR:-/tmp}/glutter-au.XXXXXX")"
chmod 700 "$runtime"
pids=()

stop_group() {
  local pid="$1"
  if kill -0 "$pid" 2>/dev/null; then
    kill -TERM -- "-$pid" 2>/dev/null || kill -TERM "$pid" 2>/dev/null || true
    for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do
      kill -0 "$pid" 2>/dev/null || break
      sleep 0.1
    done
    kill -KILL -- "-$pid" 2>/dev/null || kill -KILL "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  fi
}

cleanup() {
  for pid in "${pids[@]:-}"; do
    [[ -n "$pid" ]] && stop_group "$pid"
  done
  [[ -n "${pulse_pid:-}" ]] && stop_group "$pulse_pid"
  # Every pipewire-pulse that the control script started (one pid on each line, the first is the one above).
  if [[ -s "$runtime/pulse.pids" ]]; then
    while read -r pid; do
      [[ -n "$pid" ]] && stop_group "$pid"
    done <"$runtime/pulse.pids"
  fi
  # HEADLESS_AUDIO_LOGS=<dir> keeps the logs of the three programs there (to look for an xrun after a long run).
  if [[ -n "${HEADLESS_AUDIO_LOGS:-}" ]]; then
    mkdir -p "$HEADLESS_AUDIO_LOGS" && cp "$runtime"/*.log "$HEADLESS_AUDIO_LOGS"/ 2>/dev/null || true
  fi
  rm -rf "$runtime"
}
trap cleanup EXIT

# The caches of the Go tool stay where they are: HOME is private below, and the command may be a Go test.
if command -v go >/dev/null 2>&1; then
  export GOCACHE="${GOCACHE:-$(go env GOCACHE)}"
  export GOMODCACHE="${GOMODCACHE:-$(go env GOMODCACHE)}"
  export GOPATH="${GOPATH:-$(go env GOPATH)}"
fi
export XDG_RUNTIME_DIR="$runtime"
export PIPEWIRE_RUNTIME_DIR="$runtime"
export PULSE_RUNTIME_PATH="$runtime/pulse"
export PULSE_SERVER="unix:$runtime/pulse/native"
export DBUS_SESSION_BUS_ADDRESS="unix:path=$runtime/no-bus"
unset DISPLAY WAYLAND_DISPLAY PIPEWIRE_REMOTE PULSE_COOKIE PULSE_SINK PULSE_SOURCE
# The configuration of the user is not read (HOME and the XDG directories are private); pipewire and pipewire-pulse read the
# files below, and wireplumber its system configuration with the fragment below.
export HOME="$runtime"
export XDG_CONFIG_HOME="$runtime/xdg-config"
export XDG_STATE_HOME="$runtime/xdg-state"
export XDG_CACHE_HOME="$runtime/xdg-cache"
mkdir -p "$runtime/conf" "$XDG_CONFIG_HOME/wireplumber/wireplumber.conf.d" "$XDG_STATE_HOME" "$XDG_CACHE_HOME"

cat >"$runtime/conf/pipewire.conf" <<'EOF'
context.properties = {
    core.daemon = true
    core.name = pipewire-0
    log.level = 2
    support.dbus = false
    link.max-buffers = 16
    mem.warn-mlock = false
    mem.allow-mlock = false
    default.clock.rate = 48000
    default.clock.quantum = 512
    default.clock.min-quantum = 32
    default.clock.max-quantum = 2048
}
context.spa-libs = {
    audio.convert.* = audioconvert/libspa-audioconvert
    support.*       = support/libspa-support
}
context.modules = [
    { name = libpipewire-module-protocol-native }
    { name = libpipewire-module-access args = { } }
    { name = libpipewire-module-client-node }
    { name = libpipewire-module-client-device }
    { name = libpipewire-module-adapter }
    { name = libpipewire-module-metadata }
    { name = libpipewire-module-spa-node-factory }
    { name = libpipewire-module-link-factory }
    { name = libpipewire-module-session-manager }
]
context.objects = [
    { factory = spa-node-factory
        args = { factory.name = support.node.driver
                 node.name = Dummy-Driver
                 node.group = pipewire.dummy
                 priority.driver = 20000 }
    }
    { factory = adapter
        args = { factory.name = support.null-audio-sink
                 node.name = glutter_sink
                 node.description = "Glutter test sink"
                 media.class = Audio/Sink
                 audio.position = [ FL FR ]
                 monitor.channel-volumes = true
                 monitor.passthrough = true
                 object.linger = true }
    }
    { factory = adapter
        args = { factory.name = support.null-audio-sink
                 node.name = glutter_source
                 node.description = "Glutter test source"
                 media.class = Audio/Source/Virtual
                 audio.position = [ FL FR ]
                 object.linger = true }
    }
]
EOF

cat >"$runtime/conf/pipewire-pulse.conf" <<'EOF'
context.properties = {
    log.level = 2
    support.dbus = false
    mem.allow-mlock = false
}
context.spa-libs = {
    audio.convert.* = audioconvert/libspa-audioconvert
    support.*       = support/libspa-support
}
context.modules = [
    { name = libpipewire-module-protocol-native }
    { name = libpipewire-module-client-node }
    { name = libpipewire-module-adapter }
    { name = libpipewire-module-metadata }
    { name = libpipewire-module-protocol-pulse
        args = { server.address = [ "unix:native" ]
                 pulse.min.req = 128/48000
                 pulse.default.req = 480/48000
                 pulse.min.quantum = 128/48000 }
    }
]
pulse.properties = {
    server.address = [ "unix:native" ]
    pulse.min.req = 128/48000
    pulse.default.req = 480/48000
    pulse.min.quantum = 128/48000
}
pulse.rules = [
    { matches = [ { } ] actions = { quirks = [ ] } }
]
stream.properties = { }
EOF

# WirePlumber links the streams of the clients to the sink. Its own configuration is read as it is, and this fragment turns off
# what looks for devices: no sound card, camera or Bluetooth device is opened, only the null devices of pipewire.conf.
cat >"$XDG_CONFIG_HOME/wireplumber/wireplumber.conf.d/90-glutter.conf" <<'EOF'
wireplumber.profiles = {
    main = {
        monitor.alsa = disabled
        monitor.alsa-midi = disabled
        monitor.bluez = disabled
        monitor.bluez-midi = disabled
        monitor.libcamera = disabled
        monitor.v4l2 = disabled
    }
}
EOF

log() { echo "$runtime/$1.log"; }

setsid pipewire -c "$runtime/conf/pipewire.conf" >"$(log pipewire)" 2>&1 &
pids+=($!)

for _ in $(seq 1 100); do
  [[ -S "$runtime/pipewire-0" ]] && break
  kill -0 "${pids[0]}" 2>/dev/null || break
  sleep 0.1
done
if [[ ! -S "$runtime/pipewire-0" ]]; then
  echo "error: pipewire did not start. Its output:" >&2
  sed 's/^/  /' "$(log pipewire)" >&2 || true
  exit 3
fi

setsid wireplumber >"$(log wireplumber)" 2>&1 &
pids+=($!)

start_pulse() {
  mkdir -p "$runtime/pulse"
  setsid pipewire-pulse -c "$runtime/conf/pipewire-pulse.conf" >>"$(log pipewire-pulse)" 2>&1 &
  pulse_pid=$!
  for _ in $(seq 1 100); do
    [[ -S "$runtime/pulse/native" ]] && return 0
    kill -0 "$pulse_pid" 2>/dev/null || return 1
    sleep 0.1
  done
  return 1
}

if ! start_pulse; then
  echo "error: pipewire-pulse did not start. Its output:" >&2
  sed 's/^/  /' "$(log pipewire-pulse)" >&2 || true
  sed 's/^/  /' "$(log pipewire)" >&2 || true
  exit 3
fi

# The control script that a test uses to stop and start the PulseAudio protocol again.
echo "$pulse_pid" >"$runtime/pulse.pid"
echo "$pulse_pid" >"$runtime/pulse.pids"
cat >"$runtime/control.sh" <<EOF
#!/usr/bin/env bash
# The pid of the running pipewire-pulse is in pulse.pid; its process group is its own (setsid).
pid=\$(cat "$runtime/pulse.pid" 2>/dev/null)
case "\$1" in
  stop)
    if [[ -n "\$pid" ]]; then
      kill -TERM -- "-\$pid" 2>/dev/null || kill -TERM "\$pid" 2>/dev/null
      for _ in \$(seq 1 50); do kill -0 "\$pid" 2>/dev/null || break; sleep 0.1; done
      kill -KILL -- "-\$pid" 2>/dev/null
    fi
    rm -f "$runtime/pulse/native"
    ;;
  start)
    # The redirections are on the whole subshell: otherwise the caller's pipe stays open for as long as the server runs.
    ( cd "$runtime" && exec setsid env XDG_RUNTIME_DIR="$runtime" PIPEWIRE_RUNTIME_DIR="$runtime" HOME="$runtime" pipewire-pulse -c "$runtime/conf/pipewire-pulse.conf" ) >>"$runtime/pipewire-pulse.log" 2>&1 </dev/null &
    echo \$! >"$runtime/pulse.pid"
    echo \$! >>"$runtime/pulse.pids"
    for _ in \$(seq 1 100); do [[ -S "$runtime/pulse/native" ]] && exit 0; sleep 0.1; done
    exit 1
    ;;
  *) exit 64 ;;
esac
EOF
chmod +x "$runtime/control.sh"
export GLUTTER_AUDIO_CONTROL="$runtime/control.sh"
export GLUTTER_AUDIO_SERVER_RUNTIME="$runtime"

# Wait until the sink is there for the clients to use (wireplumber picks it as the default).
ready=0
if command -v pactl >/dev/null 2>&1; then
  for _ in $(seq 1 50); do
    if timeout 2 pactl info >/dev/null 2>&1 && timeout 3 pactl list short sinks 2>/dev/null | grep -q glutter_sink; then
      ready=1
      break
    fi
    sleep 0.1
  done
else
  ready=1
fi
if [[ "$ready" != 1 ]]; then
  echo "error: the audio server has no glutter_sink. The output of the programs:" >&2
  for name in pipewire wireplumber pipewire-pulse; do
    echo "--- $name" >&2
    sed 's/^/  /' "$(log "$name")" >&2 || true
  done
  exit 3
fi

set +e
"$@"
status=$?
exit "$status"
