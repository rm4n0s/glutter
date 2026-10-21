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

To come, in the order of `docs/IMPLEMENTATION.md`:

| Script | Step | Purpose |
|---|---|---|
| `fetch-fonts.sh` | 4.1 | Download Roboto and Material Icons at pinned commits |
