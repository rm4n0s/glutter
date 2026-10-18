# Scripts

Helper scripts for developing Glutter. Run them from the repository root.

Available now:

| Script | Purpose |
|---|---|
| `fetch-reference.sh` | Fetch the pinned Flutter sources used for porting into `reference/` (git-ignored). The first run records the commits in `reference/VERSION`; later runs check out exactly those commits |
| `headless-wayland.sh` | Run a command inside a headless Wayland compositor (sway, else weston) with `WAYLAND_DISPLAY` set, for display tests and proof screenshots: `scripts/headless-wayland.sh -- go test -tags glutter_display ./...` |
| `screenshot-headless.sh` | Run a command in headless sway, wait until it maps a window, and save a screenshot of the output with `grim` (needs `sway grim jq`): `scripts/screenshot-headless.sh out.png -- ./binary` |
| `headless-x11.sh` | Run a command under a headless X server (`Xvfb`, on a free display number, no window manager) with `DISPLAY` set: `scripts/headless-x11.sh -- go test -tags glutter_display ./internal/platform/linux/x11/` (needs `xorg-x11-server-Xvfb`) |
| `fetch-wayland-protocols.sh` | Download the Wayland protocol XML files into `internal/platform/linux/wayland/protocols/` and record the release tags in its `VERSIONS` file. `--latest` moves to the newest releases; then run `go generate ./internal/platform/linux/wayland/...` |

To come, in the order of `docs/IMPLEMENTATION.md`:

| Script | Step | Purpose |
|---|---|---|
| `fetch-fonts.sh` | 4.1 | Download Roboto and Material Icons at pinned commits |
