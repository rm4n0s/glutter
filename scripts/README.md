# Scripts

Helper scripts for developing Glutter. Run them from the repository root.

Available now:

| Script | Purpose |
|---|---|
| `fetch-reference.sh` | Fetch the pinned Flutter sources used for porting into `reference/` (git-ignored). The first run records the commits in `reference/VERSION`; later runs check out exactly those commits |
| `headless-wayland.sh` | Run a command inside a headless Wayland compositor (sway, else weston; `HEADLESS_COMPOSITOR=sway\|weston\|mutter\|kwin` chooses) with `WAYLAND_DISPLAY` set, for display tests and proof screenshots: `scripts/headless-wayland.sh -- go test -tags glutter_display ./...` |
| `wayland-matrix.sh` | The automated Wayland matrix: the display tests and every example (with a screenshot) under each installed headless compositor (sway, weston, mutter, kwin), a pass/fail table, and the compositors that were skipped. The manual part is `docs/wayland-checklist.md`: `scripts/wayland-matrix.sh` (options: `--compositors`, `--examples`, `--no-tests`, `--no-examples`, `--out DIR`) |
| `screenshot-headless.sh` | Run a command in headless sway, wait until it maps a window, and save a screenshot of the output with `grim` (needs `sway grim jq`): `scripts/screenshot-headless.sh out.png -- ./binary` |
| `headless-x11.sh` | Run a command under a headless X server (`Xvfb`, on a free display number, no window manager) with `DISPLAY` set: `scripts/headless-x11.sh -- go test -tags glutter_display ./internal/platform/linux/x11/` (needs `xorg-x11-server-Xvfb`) |
| `fetch-wayland-protocols.sh` | Download the Wayland protocol XML files into `internal/platform/linux/wayland/protocols/` and record the release tags in its `VERSIONS` file. `--latest` moves to the newest releases; then run `go generate ./internal/platform/linux/wayland/...` |
| `test-wasm.sh` | Run the unit tests of every package that builds for the web under Node (`GOOS=js GOARCH=wasm go test`, with the minimal environment that Go's wasm loader needs): `scripts/test-wasm.sh` or `scripts/test-wasm.sh ./foundation/...` (needs `nodejs`) |
| `check-android.sh` | Build and vet the module for android/arm64 and android/amd64 with the NDK (needs `ANDROID_HOME`, no device): `scripts/check-android.sh` |
| `perf-android.sh` | The Android numbers of SPEC §15 with release builds: APK sizes, cold start, idle CPU, frame times, memory, on the emulator or a phone (`glutter-android perf`): `scripts/perf-android.sh -m -o report.md`, `-d serial` for a phone, `-t counter,gallery,animation,navigation` |
| `wine-exec.sh` | Run a Windows program under Wine on a headless X server, with a private prefix in `.wine-test/prefix` (`GLUTTER_WINEPREFIX` moves it), `ulimit -c 1`, a timeout and `wineserver -k` at the end; it is the runner of `go test -exec`: `scripts/wine-exec.sh ./app.exe` (needs `wine-core` and `xorg-x11-server-Xvfb`) |
| `wine-screenshot.sh` | Save the root window of the headless X server while a command runs (proof method F, for the tray, dialogs and several windows): `scripts/wine-screenshot.sh out.png -- scripts/wine-exec.sh ./app.exe` (needs `ImageMagick`; `GLUTTER_SHOT_DELAY` seconds, 3 by default) |
| `test-windows.sh` | Run the unit tests of every package that builds for Windows, under Wine (`GOOS=windows go test -exec scripts/wine-exec.sh -p 1`): `scripts/test-windows.sh` or `scripts/test-windows.sh ./internal/platform/windows/...` |

To come, in the order of `docs/IMPLEMENTATION.md`:

| Script | Step | Purpose |
|---|---|---|
| `fetch-fonts.sh` | 4.1 | Download Roboto and Material Icons at pinned commits |
