# Setting up Fedora to work on Glutter

This is what a Fedora machine needs before you work on Glutter itself: build it, run its checks and tests, and make proof screenshots. An app that only *uses* Glutter needs Go 1.27 and nothing else (SPEC §16).

The commands were written for Fedora 44 Workstation. Install the core first. Add the other groups when you work on that part: each test or script that needs a missing tool skips or stops, and says what to install. The last section has a script that lists what is still missing.

## 1. Core: build, lint and the standard checks

Every change needs these (the Definition of done in `CLAUDE.md`).

```sh
sudo dnf install git gcc
```

- **Go 1.27 or newer.** Check with `go version`. If Fedora's `golang` package is older, install the release from https://go.dev/dl/ and put its `bin` directory on `PATH`.
- **gcc** is needed only because `go test -race` needs cgo. Glutter itself builds with `CGO_ENABLED=0`.
- **golangci-lint 2.15.0**:
  ```sh
  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
  ```
  It goes to `$(go env GOPATH)/bin`, which must be on `PATH`.
- **goimports** (optional; `gofmt` is enough):
  ```sh
  go install golang.org/x/tools/cmd/goimports@latest
  ```
- **A git identity**, needed to commit:
  ```sh
  git config --global user.name "Your Name"
  git config --global user.email "you@example.com"
  ```
- **The Flutter sources used for porting** (git-ignored, never committed): `scripts/fetch-reference.sh`.

## 2. Linux desktop: display tests, D-Bus and screenshots

Needed for `go test -tags glutter_display`, `scripts/headless-wayland.sh`, `scripts/headless-x11.sh`, `scripts/screenshot-headless.sh`, `scripts/wayland-matrix.sh` and the D-Bus tests.

```sh
sudo dnf install sway weston grim jq xorg-x11-server-Xvfb dbus-daemon
```

- `sway` is the headless compositor the scripts prefer; `weston` is the fallback.
- `grim` and `jq` take the screenshots in sway.
- `xorg-x11-server-Xvfb` is the headless X server, for the X11 tests and for Wine (group 5).
- `dbus-daemon` gives the tray, accessibility and D-Bus tests a private bus. Without it they are skipped.
- Optional, for the whole Wayland matrix: `sudo dnf install mutter kwin`. Mutter is already on Workstation.

The libraries Glutter loads at run time (SPEC §8.1: Wayland, xkbcommon, X11, EGL, GLES, Vulkan, libdecor, fontconfig) are on every Fedora Workstation. On a minimal install:

```sh
sudo dnf install libwayland-client libwayland-egl libwayland-cursor libxkbcommon libxkbcommon-x11 \
  libX11 libX11-xcb libXi libXrandr libXcursor libglvnd-egl libglvnd-gles mesa-libEGL \
  vulkan-loader libdecor fontconfig
```

## 3. Vulkan

Needed for the Vulkan display tests, which run on Mesa's software Vulkan driver (lavapipe) with validation on, and for the struct-layout tests of the Vulkan mirrors.

```sh
sudo dnf install mesa-vulkan-drivers vulkan-validation-layers vulkan-headers
```

## 4. Web

Needed for `scripts/test-wasm.sh` (the wasm unit tests run under Node) and the browser tests (`go test -tags glutter_browser`).

```sh
sudo dnf install nodejs chromium firefox
```

On Fedora 44 `nodejs` installs a versioned package (`nodejs22`); check with `node --version`.

## 5. Windows (tested under Wine)

Needed for `scripts/test-windows.sh`, the `glutter_wine` tests, `scripts/wine-exec.sh`, `scripts/wine-screenshot.sh` and the layout tests of the Win32 and COM mirrors.

```sh
sudo dnf install wine-core wine-cms xorg-x11-server-Xvfb mesa-vulkan-drivers \
  mingw64-gcc mingw64-headers ImageMagick
```

- Wine must be 11 or later (`wine --version`).
- Without `wine-cms`, a new Wine prefix hangs at its first boot.
- Wine's Direct3D 12 runs on the host's Vulkan, and on Xvfb only lavapipe (`mesa-vulkan-drivers`) can present.
- The mingw-w64 headers are only for the layout tests. Without them those tests are skipped.
- Every Wine test runs on a headless Xvfb, with its prefix in `.wine-test/prefix`. It never touches `~/.wine`.

## 6. Android

Needed for `scripts/check-android.sh`, `go run ./cmd/glutter-gen errcheck` (it also checks the Android files), the emulator tests (`go test -tags glutter_android`) and `cmd/glutter-android`.

1. **The SDK command-line tools.** Download "Command line tools only" for Linux from https://developer.android.com/studio#command-tools. Unzip it so that `sdkmanager` is at `~/Android/Sdk/cmdline-tools/latest/bin/sdkmanager`:
   ```sh
   mkdir -p ~/Android/Sdk/cmdline-tools
   unzip commandlinetools-linux-*_latest.zip -d ~/Android/Sdk/cmdline-tools
   mv ~/Android/Sdk/cmdline-tools/cmdline-tools ~/Android/Sdk/cmdline-tools/latest
   ```
2. **The environment.** Add to `~/.bashrc`, then open a new shell:
   ```sh
   export ANDROID_HOME=$HOME/Android/Sdk
   export PATH=$PATH:$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$ANDROID_HOME/emulator
   ```
3. **A Java runtime**, for `sdkmanager` and `apksigner`:
   ```sh
   sudo dnf install java-25-openjdk-headless
   ```
   A JDK (`java-25-openjdk-devel`) is needed only to change the Java embedding.
4. **The SDK packages**:
   ```sh
   sdkmanager --licenses
   sdkmanager "ndk;30.0.16248370" "platform-tools" "build-tools;37.0.0" "platforms;android-37.2"
   ```
   The NDK alone is enough for `check-android.sh` and `errcheck`.
5. **The emulator and its system images**, for the emulator tests:
   ```sh
   sdkmanager "emulator" "system-images;android-37.0;google_apis;x86_64" \
     "system-images;android-37.2;google_apis_ps16k;x86_64" "system-images;android-26;google_apis;x86_64"
   ```
   `emulator -accel-check` must say that KVM is usable. The test devices and the test `adb` server live in `.android-test/` at the module root, never in `~/.android`.
6. **The Vulkan validation layer for Android**, for the Vulkan device tests. Get the `android-binaries-*.zip` of the newest release at https://github.com/KhronosGroup/Vulkan-ValidationLayers/releases and unzip it into `.android-test/vvl/`, so that `.android-test/vvl/x86_64/libVkLayer_khronos_validation.so` exists.

## 7. Generators and reference images

Only needed to change shaders or to remake SVG reference images.

- **naga 30.0.1**, for `go generate` of the shaders (WGSL to SPIR-V, GLSL and HLSL; D20). The version is pinned:
  ```sh
  sudo dnf install cargo
  cargo install naga-cli --version 30.0.1 --locked
  ```
  It goes to `~/.cargo/bin`, which must be on `PATH`.
- **rsvg-convert**, for the SVG reference images (`-update-references`):
  ```sh
  sudo dnf install librsvg2-tools
  ```

## 8. Everything at once

```sh
sudo dnf install git gcc sway weston grim jq xorg-x11-server-Xvfb dbus-daemon \
  mesa-vulkan-drivers vulkan-validation-layers vulkan-headers nodejs chromium firefox \
  wine-core wine-cms mingw64-gcc mingw64-headers ImageMagick java-25-openjdk-headless \
  cargo librsvg2-tools
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
cargo install naga-cli --version 30.0.1 --locked
```

Then do the Android steps of group 6.

## 9. Check what is missing

Run this from the repository root. It prints only what is missing, and "nothing is missing" at the end if all is there.

```sh
missing=0
miss() { echo "missing: $*"; missing=1; }
for p in git gcc sway weston grim jq xorg-x11-server-Xvfb dbus-daemon mesa-vulkan-drivers \
  vulkan-validation-layers vulkan-headers chromium firefox wine-core wine-cms \
  mingw64-gcc mingw64-headers ImageMagick java-25-openjdk-headless cargo librsvg2-tools; do
  rpm -q "$p" >/dev/null 2>&1 || miss "package $p"
done
command -v node >/dev/null || miss "Node.js (node)"
v=$(go env GOVERSION 2>/dev/null | sed 's/^go//')
[ -n "$v" ] && [ "$(printf '%s\n1.27\n' "$v" | sort -V | head -1)" = "1.27" ] || miss "Go 1.27 or newer (have: ${v:-none})"
golangci-lint version 2>/dev/null | grep -q 'version 2\.15\.' || miss "golangci-lint 2.15 on PATH"
naga --version 2>/dev/null | grep -q '30\.0\.1' || miss "naga 30.0.1 on PATH"
w=$(wine --version 2>/dev/null | sed 's/^wine-//; s/[ .].*//')
[ "${w:-0}" -ge 11 ] 2>/dev/null || miss "Wine 11 or later (have: $(wine --version 2>/dev/null || echo none))"
git config user.name >/dev/null && git config user.email >/dev/null || miss "git user.name and user.email"
if [ -z "$ANDROID_HOME" ]; then
  miss "ANDROID_HOME"
else
  [ -x "$ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager" ] || miss "Android SDK command-line tools"
  [ -d "$ANDROID_HOME/ndk/30.0.16248370" ] || miss "Android NDK 30.0.16248370"
  [ -x "$ANDROID_HOME/build-tools/37.0.0/aapt2" ] || miss "Android build-tools 37.0.0"
  [ -f "$ANDROID_HOME/platforms/android-37.2/android.jar" ] || miss "Android platform 37.2"
  [ -x "$ANDROID_HOME/platform-tools/adb" ] || miss "Android platform-tools"
  [ -x "$ANDROID_HOME/emulator/emulator" ] || miss "Android emulator"
  for img in android-26 android-37.0 android-37.2; do
    [ -d "$ANDROID_HOME/system-images/$img" ] || miss "Android system image $img"
  done
fi
[ -f .android-test/vvl/x86_64/libVkLayer_khronos_validation.so ] || miss "Vulkan validation layer for Android in .android-test/vvl/"
[ "$missing" = 0 ] && echo "nothing is missing"
```
