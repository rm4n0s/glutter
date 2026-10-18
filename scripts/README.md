# Scripts

Helper scripts for developing Glutter. Run them from the repository root.

Available now:

| Script | Purpose |
|---|---|
| `fetch-reference.sh` | Fetch the pinned Flutter sources used for porting into `reference/` (git-ignored). The first run records the commits in `reference/VERSION`; later runs check out exactly those commits |
| `fetch-wayland-protocols.sh` | Download the Wayland protocol XML files into `internal/platform/linux/wayland/protocols/` and record the release tags in its `VERSIONS` file. `--latest` moves to the newest releases; then run `go generate ./internal/platform/linux/wayland/...` |

To come, in the order of `docs/IMPLEMENTATION.md`:

| Script | Step | Purpose |
|---|---|---|
| `headless-wayland.sh` | 1.4 | Run a command under a headless Wayland compositor |
| `screenshot-headless.sh` | 1.5 | Run a command in headless sway and capture the output with `grim` |
| `headless-x11.sh` | 1.9 | Run a command under `Xvfb` |
| `fetch-fonts.sh` | 4.1 | Download Roboto and Material Icons at pinned commits |
