# Scripts

Helper scripts for developing Glutter. Run them from the repository root.

Available now:

| Script | Purpose |
|---|---|
| `fetch-reference.sh` | Fetch the pinned Flutter sources used for porting into `reference/` (git-ignored). The first run records the commits in `reference/VERSION`; later runs check out exactly those commits |

To come, in the order of `docs/IMPLEMENTATION.md`:

| Script | Step | Purpose |
|---|---|---|
| `fetch-wayland-protocols.sh` | 1.3 | Download the Wayland protocol XML files |
| `headless-wayland.sh` | 1.4 | Run a command under a headless Wayland compositor |
| `screenshot-headless.sh` | 1.5 | Run a command in headless sway and capture the output with `grim` |
| `headless-x11.sh` | 1.9 | Run a command under `Xvfb` |
| `fetch-fonts.sh` | 4.1 | Download Roboto and Material Icons at pinned commits |
