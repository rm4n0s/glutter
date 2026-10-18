# Glutter: Implementation guide

This file breaks the milestones in `docs/SPEC.md` §18 into small steps. Each step is one unit of work for one coding agent session. Work goes strictly top to bottom. Every finished step is recorded in `docs/IMPLEMENTED.md`.

**Who this is for**: coding agents (and humans) implementing Glutter. You don't need to read the whole file; you need the Rules below and the one step you're doing.

---

## Rules

### Before you start a step

1. **Find your step.** It's the first step in this file that has no entry in `docs/IMPLEMENTED.md`. Do only that step. Never work on two steps at once, and never skip ahead.
2. **Read, in this order:**
   1. `CLAUDE.md`, all of it.
   2. The whole step in this file.
   3. Every section of `docs/SPEC.md` listed under **Read**.
   4. The Flutter files listed under **Port from**. They're under `reference/` (created by step 0.2). If `reference/` is missing, run `scripts/fetch-reference.sh`.
3. **Create the branch.**
   ```sh
   git switch main && git pull --ff-only
   git switch -c feat/step-<id>-<slug>     # e.g. feat/step-6.5-multi-child-render-objects
   ```

### While you work

4. **Do exactly the step's Tasks.** Don't do work that belongs to a later step, even if it looks easy. If a later step will need a hook, leave only a comment: `// TODO(step 9.2): ...`.
5. **Follow CLAUDE.md.** In particular: `rm4n0s/errors` for every error, the Version 2 constructors (SPEC §12), no cgo and the purego rules (SPEC §8.1), and doc comments ending in `// Flutter: <file> <Symbol>`.
6. **Port, don't invent.** When Flutter has the behavior, port Flutter's algorithm, defaults and edge cases from `reference/flutter`. Port the relevant cases from the test files listed under **Tests** as table-driven Go tests. In `docs/IMPLEMENTED.md`, say how many test cases you ported.
7. **Stop and ask the user** when any of these happens:
   - the step is ambiguous
   - the spec looks wrong or contradicts Flutter
   - a tool is missing (give the exact install command, e.g. `sudo dnf install sway grim`)
   - a new dependency seems necessary
   - something needs `sudo`
   - a check still fails after a real attempt to fix it

   Never weaken or delete a test, skip a check, or mark a step done just to get unblocked.

### Finishing a step

8. **The standard checks must all pass.** Run them from the repo root:
   ```sh
   test -z "$(gofmt -l .)"
   go vet ./...
   golangci-lint run
   go test -race ./...
   CGO_ENABLED=0 go build ./...
   CGO_ENABLED=0 go test ./...
   ```
   If the step lists **Display tests**, run them too: `go test -race -tags glutter_display <packages>`. They run under a headless compositor (step 1.5) unless the step says otherwise.
9. **Proof images.** Only when the step has a **Proof** item:
   - Produce exactly the images it lists, in `docs/implemented-proof-images/`, named `<id>-<name>.png` (e.g. `6.6-rendering-tour.png`).
   - Use the method it names (A, B or C, see below).
   - **Open every image with your image-viewing tool** (the Read tool shows images) and check that it shows what **Proof** describes. If it doesn't, fix the code. The step isn't done.
   - Never draw, edit, crop or generate a proof image any other way, and never reuse an older image.
10. **Record the step.** On the step branch, add an entry to `docs/IMPLEMENTED.md`, using the format at the top of that file. Embed the proof images.
11. **Commit on the step branch.** Use one commit per step, with the message `Step <id>: <imperative summary>`:
    ```sh
    git add -A && git commit -m "Step 6.5: Add multi-child render objects"
    ```
12. **Publish every finished step to `main`, right away.**
    - A step counts as finished only when its commit is on `origin/main`.
    - Publish before you start the next step. Never collect several steps into one publish, and never leave a finished step only on its branch.
    - Publish only after rules 8–11 are complete: checks green, proof verified, `IMPLEMENTED.md` entry committed.
    ```sh
    git switch main && git pull --ff-only
    git merge --ff-only feat/step-6.5-multi-child-render-objects
    git push origin main
    git branch -d feat/step-6.5-multi-child-render-objects
    ```
    If `--ff-only` fails, rebase the step branch onto `main`, run the standard checks again, and retry. If the push is rejected or fails (authentication, network), stop and tell the user. Don't start the next step until the push succeeded.
13. **Report to the user.** Give the step id, a two-line summary, the check results, the proof image paths, and the commit hash now on `main`.

**Default "Done when".** A step without a **Done when** line is done when its Tasks are complete, its **Tests** are ported and passing, its **Display tests** pass, its **Proof** images are verified, and the standard checks pass.

### Proof methods

| Method | Use it for | How |
|---|---|---|
| **A. Software render** | drawing and layout before real apps exist | `go run ./cmd/glutter-proof -scene <name> -o docs/implemented-proof-images/<file>.png`. The tool comes from step 2.6. Scenes are registered in `internal/proof/scenes.go`, and a step that uses method A adds its scene there |
| **B. App self-capture** | anything drawn inside an app window | Run the example with `GLUTTER_SCREENSHOT=docs/implemented-proof-images/<file>.png`, and optionally `GLUTTER_SCREENSHOT_AFTER=500ms` for mid-animation shots. The app saves its own window content and exits (from steps 1.6 and 7.4). If `WAYLAND_DISPLAY` is set, run it in the user's session; otherwise use `scripts/headless-wayland.sh -- <command>` (step 1.5) |
| **C. Desktop capture** | things outside the app's surface: several windows, server-side decorations | `scripts/screenshot-headless.sh <file.png> -- <command...>` runs the command in a headless sway session and captures the whole output with `grim` (step 1.5) |
| **C (user desktop)** | things only a real desktop shows: GNOME/libdecor title bar, tray icon in a panel, notification bubble, portal dialog | Ask the user to run the command given in the step and save a screenshot at the given path. Then view and verify the image yourself |

### Examples

- Public examples live in `examples/<name>/main.go` and use **only public packages**, the way an app would.
- Early development examples (milestones 1–4, before the public API exists) live in `examples/dev/<name>/` and may use `internal/` packages.
- Every example starts with a comment saying what it shows and how to run it (`go run ./examples/<name>`).
- An example step is done only when the example runs, its proof image is verified, and the standard checks pass.
- To reach a specific state for a proof screenshot (a page, a tab, an open menu, a number of taps), an example may read `GLUTTER_DEV_*` environment variables. It reads them **only in debug builds** (`foundation.DebugMode`), and lists them in its header comment.

---

## Milestone overview

| Milestone (SPEC §18) | Steps | Examples |
|---|---|---|
| M0 Skeleton | 0.1–0.3 | |
| M1 Platform core | 1.1–1.10 | `dev/window`, `dev/input` |
| M2 `ui` engine + software renderer | 2.1–2.10 | |
| M3 GLES renderer | 3.1–3.7 | `dev/canvas-tour` |
| M4 Text | 4.1–4.4 | `dev/text-tour` |
| M5 foundation, scheduler, physics, gestures | 5.1–5.6 | |
| M6 painting, rendering, semantics | 6.1–6.7 | |
| M7 widgets core, `gluttertest` | 7.1–7.9 | `layout`, `custompaint` |
| M8 animation | 8.1–8.4 | `animations` |
| M9 scrolling | 9.1–9.4 | `scrolling` |
| M10 navigation, services, focus | 10.1–10.7 | `navigation` |
| M11 text editing, IME, clipboard | 11.1–11.7 | `editor` |
| M12 Material foundation | 12.1–12.5 | `counter` |
| M13 Material components | 13.1–13.11 | `gallery` |
| M14 desktop windowing, Wayland quality | 14.1–14.8 | `multiwindow` |
| M15 D-Bus services | 15.1–15.5 | `tray` |
| M16 accessibility | 16.1–16.2 | |
| M17 effects and performance | 17.1–17.4 | |
| M18 1.0 | 18.1–18.3 | |

---

## M0: Skeleton

### 0.1 Module and package layout
**Goal:** an empty but complete repository skeleton that passes all checks.
**Read:** SPEC §6.2, §20; CLAUDE.md "Layout".
**Tasks:**
1. `go mod init github.com/rm4n0s/glutter`, set `go 1.27`.
2. Create every package directory in SPEC §6.2: the public packages, `internal/...`, `cmd/glutter-gen`, `examples/`. Give each one a `doc.go` whose package comment states its purpose and the Flutter library it mirrors.
3. Add `third_party/flutter/LICENSE` (Flutter's BSD-3-Clause text; download it from `https://raw.githubusercontent.com/flutter/flutter/master/LICENSE`) and a `NOTICE` file at the root listing every row in SPEC §20.
4. Add `.golangci.yml` enabling the default linters plus `gofmt`, `govet`, `errcheck`, `staticcheck`, `revive` (exported doc comments).
5. Create `scripts/` (empty except a `README.md` listing the scripts to come).
**Done when:** the standard checks pass.

### 0.2 Reference sources
**Goal:** a pinned local copy of Flutter's sources for porting (SPEC §20).
**Read:** SPEC §20 "Reference sources"; CLAUDE.md "Porting from Flutter".
**Files:** `scripts/fetch-reference.sh`, `reference/VERSION` (git-ignored, written by the script).
**Tasks:**
1. Write `scripts/fetch-reference.sh` (bash, `set -euo pipefail`). It must:
   1. If `reference/VERSION` exists, check out exactly the commits recorded there. Otherwise clone the `stable` branch of `https://github.com/flutter/flutter`, using `--depth 1 --filter=blob:none --sparse`.
   2. Sparse-checkout these paths: `packages/flutter/lib`, `packages/flutter/test`, `engine/src/flutter/lib/ui` (the `dart:ui` sources), `dev/tools/gen_keycodes/data`, `LICENSE`, `version` (if present).
   3. Clone `https://github.com/material-foundation/material-color-utilities` (sparse: `dart`, `LICENSE`) into `reference/material-color-utilities`.
   4. Write `reference/VERSION` with both repositories' commit hashes and dates, plus Flutter's version string.
2. Run it. Then check that every sparse path exists. If one is missing (Flutter reorganizes sometimes), find it with `git -C reference/flutter ls-tree -r --name-only HEAD | grep <name>` and **ask the user** before changing the paths.
3. Confirm with `git status` that `reference/` is ignored.
**Done when:** the script runs twice in a row without errors (the second run uses `VERSION`), and the standard checks pass.

### 0.3 Code generator skeleton
**Goal:** `cmd/glutter-gen` with subcommands, wired to `go generate`.
**Read:** SPEC §6.2, §12.11; CLAUDE.md "Error handling".
**Tasks:**
1. `cmd/glutter-gen/main.go`: dispatch on the first argument to the subcommands `wayland`, `keys`, `reexport`, `copywith`, `icons`. Each one returns a tagged error `NotImplemented` for now. Keep `main.go` thin; put the subcommands in `cmd/glutter-gen/internal/<name>/`.
2. Add `github.com/rm4n0s/errors` to `go.mod` (its first use). Errors are printed with `ToJson()`, and the exit code is 1.
3. Add a table test for the dispatch (an unknown command gives the tagged error `UnknownCommand`, plus a route test).
**Done when:** `go run ./cmd/glutter-gen wayland` prints the `NotImplemented` error; the standard checks pass.

---

## M1: Platform core (Linux)

### 1.1 Platform interfaces and the fake platform
**Goal:** the internal contract between the engine and OS backends, plus a scriptable fake for tests.
**Read:** SPEC §6.3, §7 (only the geometry, pointer data and key data rows), §8.4.
**Files:** `ui/geometry.go` (`Offset`, `Size`, `Rect` only, the rest is in 2.1), `ui/pointer.go`, `ui/key.go` (data types only), `internal/platform/platform.go`, `internal/platform/fake/`.
**Tasks:**
1. Define the `ui` value types needed by the interface: `Offset`, `Size`, `Rect`, `ViewID`, `PointerData` (all fields from `engine/src/flutter/lib/ui/pointer.dart`), `KeyData` (from `key.dart`), `AppLifecycleState`.
2. Define `Platform`, `Window`, `Handler`, `WindowConfig`, `Display`, `Settings`, `Services`, `WindowState`, exactly as sketched in SPEC §6.3. Use placeholder types for things that come later (e.g. `TextInputState`) and mark them with a `// TODO(step X.Y)` comment.
3. Implement `internal/platform/fake`:
   - `Run` drives scripted events from a queue
   - `Post` works from any goroutine
   - `NewWindow` records configs
   - `Inject(event)` adds input
   - it has an inspectable log
4. Tests: `Post` from 100 goroutines arrives in order per goroutine; `Run` stops when `ctx` is cancelled.
**Done when:** the standard checks pass.

### 1.2 Library loader (purego)
**Goal:** one safe way to load C libraries and bind functions, used by every backend.
**Read:** SPEC §8.1 (all of it), §22; CLAUDE.md purego rules.
**Files:** `internal/platform/linux/dl/`.
**Tasks:**
1. Add `github.com/ebitengine/purego` to `go.mod`.
2. `dl.Open(names ...string) (*Lib, error)`: tries each name with `purego.Dlopen(RTLD_NOW|RTLD_LOCAL)`. If all fail, it returns `LibraryLoadFailed` with metadata `names` and `dlerror`.
3. `dl.OpenOverride(env string, names ...string)`: an absolute path in the environment variable wins (for `GLUTTER_EGL_LIBRARY`, `GLUTTER_GLES_LIBRARY`).
4. `(*Lib).Bind(fptr any, symbol string) error` wraps `purego.RegisterLibFunc`. A missing symbol gives `SymbolMissing` (metadata: library, symbol). `(*Lib).Has(symbol) bool` checks without failing.
5. Bind libc (`libc.so.6`): `malloc`, `calloc`, `free`, `strlen`. Add helpers `CString(string) uintptr`, `GoString(uintptr) string`, `Free(uintptr)`.
6. Add the callback registry `dl.Callback(fn any) uintptr` around `purego.NewCallback`. It counts callbacks and panics with tag `CallbackLimit` past 1,500, leaving a safety margin under purego's 2,000.
7. Add `dl.AssertLayout(t, name string, size uintptr, offsets map[string]uintptr, got any)`, a test helper for struct layout tests (SPEC §8.1 rule 3).
**Tests:** load libc; bind `strlen` and call it; route test for `LibraryLoadFailed` and `SymbolMissing`; the callback counter.
**Done when:** the standard checks pass (including `CGO_ENABLED=0 go test`).

### 1.3 Wayland protocol bindings generator
**Goal:** Go bindings for every Wayland protocol in SPEC §8.2, generated from XML.
**Read:** SPEC §8.1 "Wayland protocol bindings" and "Rules for calling C through purego", §8.2.
**Files:** `scripts/fetch-wayland-protocols.sh`, `internal/platform/linux/wayland/protocols/*.xml` + `VERSIONS`, `cmd/glutter-gen/internal/wayland/`, `internal/platform/linux/wayland/wl/` (generated output + runtime).
**Tasks:**
1. Write `scripts/fetch-wayland-protocols.sh`:
   - Find the latest release tags of `https://gitlab.freedesktop.org/wayland/wayland` and `.../wayland/wayland-protocols` with `git ls-remote --tags`.
   - Download `protocol/wayland.xml`, plus every protocol in the SPEC §8.2 tables: xdg-shell, xdg-decoration-v1, fractional-scale-v1, viewporter, cursor-shape-v1, tablet-v2, text-input-v3, primary-selection-v1, pointer-gestures-v1, xdg-activation-v1, presentation-time, xdg-toplevel-icon-v1, xdg-foreign-v2, xdg-dialog-v1, xdg-output-v1, idle-inhibit-v1, content-type-v1.
   - Write the tags and file list into `VERSIONS`.
2. In the generated **runtime** (hand-written, in `wl/`):
   - Bind `libwayland-client.so.0`: `wl_display_connect`, `wl_display_disconnect`, `wl_display_get_fd`, `wl_display_flush`, `wl_display_prepare_read`, `wl_display_read_events`, `wl_display_cancel_read`, `wl_display_dispatch_pending`, `wl_display_get_error`, `wl_display_get_protocol_error`, `wl_proxy_marshal_array_flags`, `wl_proxy_add_dispatcher`, `wl_proxy_destroy`, `wl_proxy_set_user_data`, `wl_proxy_get_user_data`, `wl_proxy_get_version`, `wl_proxy_get_id`.
   - Mirror `wl_interface`, `wl_message` and `wl_argument` in Go, with layout tests.
   - Install **one** dispatcher callback for all proxies. Its user data is an integer handle into a Go object table, **never a Go pointer**.
3. The **generator** (`glutter-gen wayland`) reads all the XML files and writes one `*_gen.go` per protocol:
   - a Go type per interface
   - typed request methods, which build a `[]wl_argument` in C memory and call `wl_proxy_marshal_array_flags`
   - an event handler struct of typed func fields (`OnConfigure func(width, height int32, states []uint32)`)
   - enum constants
   - the `wl_interface` tables, built in C memory at `init`
4. Add `//go:generate go run ../../../cmd/glutter-gen wayland` and run it.
**Tests:** generator golden test on a small test XML (in `testdata/`); layout tests; the generated code compiles under `CGO_ENABLED=0`.
**Done when:** the standard checks pass, and the generated files carry the "Code generated ... DO NOT EDIT." header.

### 1.4 Wayland connection and event loop
**Goal:** connect to a compositor, bind globals, and run the UI thread's event loop (SPEC §8.4).
**Read:** SPEC §8.2 (required protocols), §8.4.
**Files:** `internal/platform/linux/loop.go` (shared with X11), `internal/platform/linux/wayland/conn.go`, `scripts/headless-wayland.sh`.
**Tasks:**
1. Event loop (`linux/loop.go`):
   - `epoll` on the display fd, an `eventfd` (for `Post`) and a `timerfd` (for timers), using the `syscall` package
   - `runtime.LockOSThread()` in an `init` of `internal/platform`
   - `Post(fn)` is safe from any goroutine and wakes the loop
2. Wayland reading follows the `prepare_read` / `poll` / `read_events` / `dispatch_pending` pattern, with a flush before every wait.
3. Registry: bind the required globals at the versions listed in SPEC §8.2. A missing required global gives `WaylandProtocolMissing` (metadata: interface). Record the optional globals that are present.
4. Errors: a failed connect gives `WaylandConnectFailed`. A protocol error ends `Run` with `WaylandProtocolError` (interface, id, code).
5. `scripts/headless-wayland.sh -- <cmd>` starts a headless compositor and runs the command with `WAYLAND_DISPLAY` set to it, then stops it:
   - prefer `sway` with `WLR_BACKENDS=headless WLR_LIBINPUT_NO_DEVICES=1`
   - fall back to `weston --backend=headless`
   - if neither exists, print the install command (`sudo dnf install sway grim weston`) and exit 2
**Tests:** unit tests for the loop (`Post` ordering, timers); **Display tests**: connect to the headless compositor, list globals, assert that the required ones are present.
**Done when:** the standard checks pass; the display tests pass under `scripts/headless-wayland.sh`.

### 1.5 First Wayland window (shared-memory buffer)
**Goal:** a mapped `xdg_toplevel` filled with a solid color through `wl_shm`, before any GPU code exists.
**Read:** SPEC §8.2, §8.7 (properties), §8.11 "Lifecycle and correctness".
**Files:** `internal/platform/linux/wayland/window.go`, `scripts/screenshot-headless.sh`, `examples/dev/window/main.go`.
**Tasks:**
1. Create `wl_surface` → `xdg_surface` → `xdg_toplevel`. Set the title and the app id **before** the first commit.
2. Handle `configure`: store the size and states (maximized, fullscreen, activated, suspended, ...), then `ack_configure`. `close` calls `Handler.OnCloseRequest`.
3. Request server-side decorations through `zxdg_decoration_manager_v1` when it's available.
4. Draw a solid-color buffer: `memfd_create` + `ftruncate` + `mmap` (via `syscall`), then `wl_shm_pool` → `wl_buffer`. Re-create it on resize.
5. Write `scripts/screenshot-headless.sh <out.png> -- <cmd...>`: start headless sway (as in 1.4), run the command, wait until a window is mapped (poll `swaymsg -t get_tree`, max 10 s), run `grim <out.png>`, then stop everything.
6. `examples/dev/window`: open a 640×400 window titled "Glutter dev window", filled with `#6750A4`, which closes on the close request.
**Display tests:** the window maps, a configure is received, and the close request ends `Run`.
**Proof:** C, `1.5-shm-window.png`: a purple 640×400 rectangle (the window) on sway's background.
**Example:** `examples/dev/window` (shm version).

### 1.6 EGL and GLES on Wayland, and self-capture
**Goal:** a GLES 3.0 context rendering into the window, paced by frame callbacks; plus the `GLUTTER_SCREENSHOT` capture used by proof method B.
**Read:** SPEC §8.1 (EGL rows, lookup rules), §8.11 "Frame pacing", "GPU drivers", §9.1 first bullets.
**Files:** `internal/gpu/gles/egl.go`, `internal/gpu/gles/gl.go` (only the functions needed now), `internal/platform/linux/wayland/egl.go`, `internal/engine/screenshot.go`.
**Tasks:**
1. Load `libEGL` and `libGLESv2` with `dl.OpenOverride`.
   - `eglGetPlatformDisplay(EGL_PLATFORM_WAYLAND_KHR, wl_display)`, then `eglInitialize`.
   - Choose a config with RGBA8, `EGL_OPENGL_ES3_BIT` and a window surface. None gives `EGLNoConfig`.
   - Create an ES 3.0 context (failure gives `GLES3Unsupported`) and query `GL_VERSION`.
2. Bind `libwayland-egl.so.1`: `wl_egl_window_create`, `_resize`, `_destroy`. Create the EGL window surface.
3. `eglSwapInterval(0)`. Render only on `wl_surface.frame` callbacks: request the next callback before each swap.
4. Render: `glClearColor(#6750A4)`, `glClear`, `eglSwapBuffers`. On configure, resize the egl window and render **before** `ack_configure` + commit (§8.11 "Resizing").
5. Self-capture: if `GLUTTER_SCREENSHOT` is set, after the first presented frame call `glReadPixels` (RGBA, then flip rows), encode a PNG to that path, log it, and exit with code 0. `GLUTTER_SCREENSHOT_AFTER=<duration>` delays the capture. Idle detection comes in 7.4.
6. Switch `examples/dev/window` to EGL; keep the shm path only behind `GLUTTER_DEV_SHM=1`.
**Tests:** layout tests for any EGL structs; route tests for `EGLLoadFailed` (using an override pointing to a missing file) and `EGLNoConfig` (with a fake that returns 0 configs).
**Display tests:** context creation succeeds under headless sway.
**Proof:** B, `1.6-egl-window.png`: an evenly purple image of the window size.
**Example:** `examples/dev/window` (EGL version).

### 1.7 Keyboard (xkbcommon) and key tables
**Goal:** correct key events for any layout, on Wayland, mapped to Flutter's physical and logical keys.
**Read:** SPEC §8.5 "Keyboard", §8.11 "Input".
**Port from:** `reference/flutter/dev/tools/gen_keycodes/data/physical_key_data.g.json`, `logical_key_data.g.json` (the Linux/GTK columns).
**Files:** `internal/platform/linux/xkb/`, `cmd/glutter-gen/internal/keys/`, `internal/platform/linux/keymap_gen.go`, `internal/platform/linux/wayland/keyboard.go`.
**Tasks:**
1. Bind `libxkbcommon.so.0`:
   - context, keymap from string, state, `key_get_syms`, `key_get_utf8`, `state_update_mask`, `mod_name_is_active`, `keymap_key_repeats`
   - the compose table and compose state, for dead keys
2. `glutter-gen keys`: generate `keymap_gen.go`, with an XKB keycode → Flutter physical key id map and a keysym → Flutter logical key id map. Unknown keysyms become Flutter's "unprintable" plane ids (follow Flutter's GTK embedder rules in the JSON comments).
3. `wl_keyboard`: `keymap` (mmap the fd read-only), `enter`/`leave` (synthesize key up for keys still down on leave), `key`, `modifiers`, `repeat_info`. Key repeat runs on the loop's `timerfd`, producing `KeyData` with type repeat.
4. Deliver `ui.KeyData` (with physical, logical and character) to `Handler.OnKey`.
**Tests:** xkb tests **without a display**, using a keymap string built from the "us" and "de" layouts (`xkb_keymap_new_from_names`). Cover: `a`, `Shift+a`, `AltGr+q` on de (`@`), the dead key `^` + `e` = `ê`, and repeat timing with the fake clock.
**Done when:** the standard checks pass.

### 1.8 Pointer, touch and cursors (Wayland)
**Goal:** complete pointer and touch input as `ui.PointerData`, plus basic cursors.
**Read:** SPEC §8.5 "Pointer", §8.11 "Input".
**Files:** `internal/platform/linux/wayland/pointer.go`, `touch.go`, `cursor.go`, `examples/dev/input/main.go`.
**Tasks:**
1. `wl_pointer`: collect `enter`, `leave`, `motion`, `button`, `axis`, `axis_value120`, `axis_source`, `axis_stop` and `axis_relative_direction` until `frame`, then emit one batch of `PointerData`. Use device kind mouse, or trackpad when the axis source is finger. Map Linux button codes to Flutter's button bits.
2. `zwp_pointer_gestures_v1` pinch/swipe → `PointerData` pan/zoom start/update/end.
3. `wl_touch` → touch `PointerData` (down, motion, up, cancel, frame).
4. Cursors: `wp_cursor_shape_v1` when present, else `libwayland-cursor.so.0` loading the theme from `XCURSOR_THEME`/`XCURSOR_SIZE`. Wire `Window.SetCursor` for `basic`, `click` (pointer), `text`, `grab`, `grabbing`, `resize*`, `wait`, `forbidden` and `none`.
5. `examples/dev/input`: a window that logs every key and pointer event (with `slog`) and changes cursor when the mouse is in the left vs right half.
**Tests:** unit tests feeding recorded event sequences into the batcher (a high-resolution wheel, a kinetic touchpad scroll with stop, a pinch).
**Done when:** the standard checks pass. Manually, run `go run ./examples/dev/input` in the user's session if one is available; otherwise note in `IMPLEMENTED.md` that the manual check is pending.
**Example:** `examples/dev/input`.

### 1.9 X11 backend
**Goal:** the same window, rendering, keyboard and pointer support on X11.
**Read:** SPEC §8.3, §8.4.
**Files:** `internal/platform/linux/x11/`, `scripts/headless-x11.sh`.
**Tasks:**
1. Bind `libX11.so.6`:
   - `XOpenDisplay`, `XCloseDisplay`, `ConnectionNumber`, `XPending`, `XNextEvent`, `XCreateWindow`, `XMapWindow`, `XDestroyWindow`, `XInternAtom`, `XChangeProperty`, `XSetWMProtocols`, `XSelectInput`, `XFlush`, `XSetErrorHandler` (one callback)
   - plus `libX11-xcb.so.1` `XGetXCBConnection`
   - mirror `XEvent` (192 bytes) and the event structs used, with layout tests
2. EGL with `EGL_PLATFORM_X11_KHR` on the Xlib `Display`. Frame pacing: a `timerfd` at the monitor's refresh rate (from XRandR, default 60 Hz), with swap interval 0.
3. Window properties: `_NET_WM_NAME`, `WM_CLASS` (the app id), `WM_DELETE_WINDOW` → close request, `_NET_WM_PING`.
4. Keyboard: `libxkbcommon-x11.so.0` (`xkb_x11_keymap_new_from_device`, `xkb_x11_state_new_from_device`) with the same key tables as 1.7.
5. Pointer: XInput2 (`libXi.so.6`: `XIQueryVersion`, `XISelectEvents`, `XGetEventData`) for motion, buttons, smooth scroll valuators and touch. Cursors with `libXcursor.so.1`.
6. Self-capture (`GLUTTER_SCREENSHOT`) works here too.
7. `scripts/headless-x11.sh -- <cmd>` runs the command under `Xvfb :99` (if missing, print `sudo dnf install xorg-x11-server-Xvfb` and exit 2).
**Display tests (X11):** window and EGL context under `scripts/headless-x11.sh`.
**Proof:** B under `scripts/headless-x11.sh`, `1.9-x11-window.png`: a purple image (the dev window running on X11).

### 1.10 Backend selection and engine skeleton
**Goal:** `internal/engine` chooses Wayland or X11 per SPEC §6.3 rule 5 and owns the run loop.
**Read:** SPEC §6.1, §6.3 rules 5–6, §8.4, §14.1.
**Files:** `internal/engine/engine.go`, `internal/platform/linux/select.go`.
**Tasks:**
1. Selection order: `GLUTTER_BACKEND`, then Wayland when `WAYLAND_DISPLAY` connects **and** the Wayland libraries load, then X11 via `DISPLAY`, then `NoDisplay`. Log which backend was chosen and why.
2. `engine.Run(ctx, cfg)` creates the platform, the first window and the renderer, and runs the loop. Errors are returned unchanged (route tests).
3. Make both dev examples go through `engine.Run`.
**Tests:** table tests for the selection with a fake environment and fake loaders, plus route tests for `NoDisplay`.
**Done when:** the standard checks pass; `examples/dev/window` runs under both headless scripts.

---

## M2: `ui` engine and software renderer

### 2.1 Geometry
**Goal:** all of `dart:ui`'s geometry in Go.
**Read:** SPEC §7 "Geometry"; CLAUDE.md "Flutter API conventions".
**Port from:** `reference/flutter/engine/src/flutter/lib/ui/geometry.dart`, `painting.dart` (RRect, RSuperellipse, Radius); the `Matrix4` operations Flutter uses from `vector_math` (`translate`, `scale`, `rotateZ`, `multiply`, `invert`, `transformPoint`, `transformRect`).
**Files:** `ui/geometry.go` (extend), `ui/rrect.go`, `ui/matrix4.go`.
**Tasks:**
1. Port `Offset`, `Size`, `Rect`, `Radius`, `RRect`, `RSuperellipse` with every method and lerp. Operators become methods (SPEC §12.8).
2. `Matrix4`: float32, column-major, with the methods listed above plus `IsIdentity`, `Determinant` and `MatrixUtils`-style helpers.
**Tests:** port the cases from Flutter's `dart:ui` geometry tests (if absent in `reference/`, write cases from the Dart doc comments); fuzz test `Rect` intersect/union invariants.

### 2.2 Color, Paint, shaders and filters
**Goal:** the paint-describing value types (no drawing yet).
**Read:** SPEC §7 "Color", "Paint", "Shaders", "Filters".
**Port from:** `engine/src/flutter/lib/ui/painting.dart`.
**Files:** `ui/color.go`, `ui/paint.go`, `ui/shader.go`, `ui/filter.go`.
**Tasks:**
1. `Color` with float32 ARGB components plus `ColorSpace`; `FromARGB32`, `WithAlpha`, `WithValues`, `Lerp`, `ComputeLuminance`, and `AlphaBlend` exactly as in Dart.
2. `BlendMode` (all 29, same order as Dart), `FilterQuality`, `StrokeCap`, `StrokeJoin`, `PaintingStyle`, `TileMode`, `BlurStyle`.
3. `Paint` as a plain struct with Flutter's defaults (zero values per SPEC §12.3).
4. Shaders: `NewGradientLinear`, `NewGradientRadial` (with focal point), `NewGradientSweep`, `NewImageShader` (the image type is a stub until 2.10). Filters: `ColorFilterMode`, `ColorFilterMatrix`, `ColorFilterLinearToSrgbGamma`, `ColorFilterSrgbToLinearGamma`, `ImageFilterBlur`, `ImageFilterMatrix`, `ImageFilterCompose`, `ImageFilterDilate`, `ImageFilterErode`, `MaskFilterBlur`.
**Tests:** table tests for color math (lerp, luminance, alpha blend) against values computed from the Dart implementation.

### 2.3 Path and PathMetrics
**Goal:** a complete `Path` with flattening and measurement.
**Read:** SPEC §7 "Path".
**Port from:** `engine/src/flutter/lib/ui/painting.dart` (`Path`, `PathMetrics` API). The algorithms follow Skia's behavior (arc-to-conic conversion, `arcToPoint` radii scaling), so reproduce Skia's documented semantics.
**Files:** `ui/path.go`, `ui/path_flatten.go`, `ui/path_metrics.go`.
**Tasks:**
1. Store verbs and points (move, line, quad, conic, cubic, close). Add every builder method in SPEC §7, including the relative variants, `FillType`, `Transform`, `Shift`, `GetBounds` (tight) and `Contains` (non-zero and even-odd).
2. Flattening with a tolerance (`Flatten(transform, tolerance) [][]Offset`).
3. `PathMetrics`: length, `GetTangentForOffset`, `ExtractPath`. `PathCombine` comes in 17.1: leave it out entirely.
**Tests:** table tests (bounds of arcs and curves, contains, metrics length of a circle ≈ 2πr); fuzz flatten (no NaN, bounded output).

### 2.4 Canvas, PictureRecorder and Picture
**Goal:** record drawing commands into an immutable display list.
**Read:** SPEC §7 "Canvas".
**Port from:** `engine/src/flutter/lib/ui/painting.dart` (`Canvas`, `PictureRecorder`, `Picture`).
**Files:** `ui/canvas.go`, `ui/picture.go`, `internal/displaylist/` (the op encoding shared with the renderers).
**Tasks:**
1. Implement the complete `Canvas` API from SPEC §7, recording into a display list. Save/restore and the transform and clip stacks have Flutter's semantics (`GetSaveCount`, `RestoreToCount`, `GetTransform`, `GetLocalClipBounds`, `GetDestinationClipBounds`).
2. `Picture` is immutable, with `CullRect` and `ApproximateBytesUsed`. `ToImage` is a stub returning a tagged `NotImplemented` until 2.10.
**Tests:** recorded ops for a sequence of calls; save-count semantics; clip-bounds queries.

### 2.5 SceneBuilder and Scene
**Goal:** the compositing layer tree.
**Read:** SPEC §7 "Compositing".
**Port from:** `engine/src/flutter/lib/ui/compositing.dart`.
**Files:** `ui/scene.go`.
**Tasks:** implement `SceneBuilder` with every push method listed in SPEC §7, plus `AddPicture`, `AddRetained`, `Pop` and `Build`. A `Scene` is an immutable tree. Each `Push*` returns an `EngineLayer` handle that can be retained.
**Tests:** the tree shape for nested pushes; that a retained layer is reused.

### 2.6 Software rasterizer core, golden harness and proof tool
**Goal:** pixels. Fills and strokes with exact-area antialiasing, the golden-test harness, and `cmd/glutter-proof`.
**Read:** SPEC §9.2, §17 "Golden tests"; this file's "Proof methods".
**Files:** `internal/gpu/soft/raster.go`, `stroke.go`, `soft.go`, `internal/goldens/`, `internal/proof/scenes.go`, `cmd/glutter-proof/main.go`.
**Tasks:**
1. A scanline rasterizer with exact area coverage (accumulation buffer, as in font rasterizers), non-zero and even-odd fill, premultiplied RGBA8.
2. A stroker: width, caps (butt, round, square), joins (miter with limit, round, bevel), hairlines.
3. `soft.Render(scene, size) *image.RGBA` for picture layers containing paths with solid paint and transforms. Other ops return a tagged `NotImplemented` until 2.7–2.9.
4. `internal/goldens`: `goldens.Check(t, name, img)` compares against `testdata/goldens/<name>.png` with a per-pixel tolerance, writes `<name>.actual.png` and `<name>.diff.png` on mismatch, and rewrites the golden under `-update-goldens`.
5. `cmd/glutter-proof -scene <name> -o <file>` renders a scene registered in `internal/proof/scenes.go`. `-list` lists the scenes. Register the scene `shapes-basic`: filled star, stroked curves with every cap and join, an even-odd ring, a rotated rectangle.
**Tests:** goldens for each cap and join, the fill rules, and a transformed path.
**Proof:** A, `2.6-shapes-basic.png`: the star, the curves with visible cap and join differences, the ring with a hole, and the rotated rectangle, all with smooth antialiased edges.

### 2.7 Software renderer: shapes, clips and images
**Goal:** every geometric drawing op and clip.
**Read:** SPEC §9.1 table rows "Rect, RRect…", "Clips", "Images" (for the semantics), §9.2.
**Files:** `internal/gpu/soft/`.
**Tasks:**
1. Analytic coverage for rect, rrect, rsuperellipse, oval, circle, DRRect, arc (with and without center), lines and points (point modes).
2. Clips: rect, rrect, rsuperellipse and path, antialiased or not. Clip stacks intersect correctly with transforms.
3. Images: `DrawImage`, `DrawImageRect` and `DrawImageNine`, with nearest/linear/mipmapped filtering per `FilterQuality`; `DrawAtlas`; `DrawVertices` (triangles, strips, fans, per-vertex colors).
4. Register the proof scene `shapes-clips`.
**Tests:** goldens per op and per clip type, including a rotated rrect clip.
**Proof:** A, `2.7-shapes-clips.png`: a grid with one cell per shape type, plus a cell showing an image clipped by a rotated rounded rectangle.

### 2.8 Software renderer: gradients, blend modes, filters and layers
**Goal:** paint effects and layer compositing.
**Read:** SPEC §9.1 rows "Gradients", "Blend modes", "Layers"; SPEC §9.1 "Color" (sRGB blending).
**Files:** `internal/gpu/soft/`.
**Tasks:**
1. Linear, radial, two-point conical and sweep gradients with every `TileMode`; image shaders.
2. All 29 blend modes, with formulas matching Skia's definitions, blending in non-linear sRGB.
3. Color filters (mode, matrix, gamma); `SaveLayer` with paint (opacity, blend, color filter); scene layers for offset, transform, opacity, color filter and clip.
4. Register `gradients` and `blend-modes` (a 6×5 grid labelled by drawing order; labels come in M4, so use the grid position only).
**Tests:** goldens per gradient type and tile mode; a blend-mode grid golden; a SaveLayer opacity golden.
**Proof:** A, `2.8-gradients.png` (smooth, banding-free gradients of every type), `2.8-blend-modes.png` (29 visibly different cells).

### 2.9 Software renderer: blur, shadows, backdrop filter and shader mask
**Goal:** the remaining effects.
**Read:** SPEC §9.1 rows "BoxShadow / drawShadow", "Blur", "Layers".
**Port from:** Flutter's `drawShadow` semantics (`engine/src/flutter/lib/ui/painting.dart` docs); Material elevation shadows are tuned later (12.3).
**Files:** `internal/gpu/soft/`.
**Tasks:**
1. Gaussian blur (separable; downsampled for large sigma) for `MaskFilterBlur` (every `BlurStyle`) and `ImageFilterBlur` with tile modes. Add `ImageFilterMatrix`, `ImageFilterCompose`, dilate and erode.
2. Analytic rrect shadow (closed form); `DrawShadow` for arbitrary paths via blur.
3. `PushBackdropFilter` and `PushShaderMask` in scenes.
4. Register `effects`.
**Tests:** goldens for every blur style, the shadow at elevations 1, 4 and 12, a backdrop blur over stripes, and a gradient shader mask.
**Proof:** A, `2.9-effects.png`: soft shadows under cards, a frosted-glass rectangle over stripes, and text-less shapes faded by a gradient mask.

### 2.10 Images and codecs
**Goal:** decode images, and make `Picture.ToImage` work.
**Read:** SPEC §7 "Images", D12.
**Files:** `ui/image.go`, `ui/codec.go`.
**Tasks:**
1. Add `golang.org/x/image` to `go.mod`.
2. `InstantiateImageCodec(ctx, data, opts)`: PNG, JPEG and GIF (including every frame with its duration) from the standard library; WebP and BMP from `x/image`. Decode into premultiplied RGBA. Support `TargetWidth`/`TargetHeight` resizing. Failure gives `ImageDecodeFailed`.
3. `DecodeImageFromPixels`; `Picture.ToImage` through the software renderer (later through the GPU, in 3.x).
**Tests:** decode the test images in `testdata/` (create small ones with Go's image encoders in a test helper); animated GIF frame count and durations; route test for `ImageDecodeFailed`.
**Done when:** the standard checks pass.

---

## M3: GLES renderer

### 3.1 GLES device layer
**Goal:** the infrastructure every GPU draw needs.
**Read:** SPEC §9.1 (first bullets), §6.3 rule 4, §8.1 rules.
**Files:** `internal/gpu/gles/gl.go` (complete function list), `device.go`, `shaders/*.glsl` (embedded with `embed`).
**Tasks:**
1. Bind the GLES 3.0 functions used in SPEC §9.1 through `eglGetProcAddress` + `purego.RegisterFunc`. Keep the list in one table, so a missing function is a clear `SymbolMissing`.
2. Shader programs: GLSL ES 3.00 sources compiled at startup. A compile or link failure gives `ShaderCompileFailed`, with the info log in its metadata.
3. Resource pools for buffers, textures and framebuffers; a GL state cache (avoids redundant calls); `GLES3Unsupported` if `GL_VERSION` is below 3.0.
4. A `gpu.Renderer` implementation skeleton (SPEC §6.3) whose `Render` clears to the scene's background.
**Tests:** shader sources compile under the display tag; pool reuse unit tests.
**Display tests:** create the device under headless sway.

### 3.2 Draw lists, batching and analytic shapes
**Goal:** the GLES backend draws solid shapes with SDF antialiasing.
**Read:** SPEC §9.1 "Pipeline" and table rows "Rect, RRect…", "Clips" (scissor part).
**Files:** `internal/gpu/flatten.go` (shared: `ui.Scene` → draw list), `internal/gpu/gles/shapes.go`.
**Tasks:**
1. Flatten a `ui.Scene` into draw commands with transforms, clip state and paint. This is shared code, also used by tests.
2. Batch by pipeline, texture and blend mode; instanced quads carrying SDF parameters for rect, rrect, rsuperellipse, oval, circle, DRRect, lines and points.
3. Rect clips through the scissor test.
4. `examples/dev/canvas-tour`: a window showing the proof scenes from 2.6–2.9 one per page; arrow keys switch pages, and `GLUTTER_DEV_PAGE=<n>` picks the start page.
**Display tests:** render the `shapes-clips` scene through `gles` (shape cells only) and compare with the software golden within tolerance.
**Proof:** B, `3.2-canvas-tour-shapes.png`: the shape grid from 2.7, drawn by the GPU.
**Example:** `examples/dev/canvas-tour` (first version).

### 3.3 Paths, clips and images on the GPU
**Goal:** general paths, antialiased clips and textures.
**Read:** SPEC §9.1 rows "General paths", "Clips", "Images".
**Files:** `internal/gpu/atlas.go` (coverage atlas shared with glyphs later), `internal/gpu/gles/paths.go`, `images.go`.
**Tasks:**
1. Coverage atlas: rasterize paths with the software rasterizer from 2.6 into 8-bit tiles, cached by path + transform, with LRU eviction; composite on the GPU. Paths larger than a tile use a tiled mask.
2. RRect/path clips with the stencil buffer, or antialiased through coverage masks.
3. Image textures with mipmaps; nine-patch, atlas and vertices draws.
**Display tests:** the `shapes-basic` and `shapes-clips` scenes match the software goldens.
**Proof:** B, `3.3-canvas-tour-paths.png` (page 1, the star and strokes on the GPU).

### 3.4 Gradients, blend modes and layers on the GPU
**Goal:** paint effects on the GPU.
**Read:** SPEC §9.1 rows "Gradients", "Blend modes", "Layers".
**Tasks:**
1. Gradient shaders (linear, radial, two-point conical, sweep, every tile mode) and image shaders.
2. Porter-Duff modes through `glBlendFunc`. The advanced modes use `KHR_blend_equation_advanced` when present, else a shader that reads a copy of the destination.
3. `SaveLayer`, opacity, color filters and layer pushes through pooled offscreen framebuffers.
**Display tests:** the `gradients` and `blend-modes` scenes match the software goldens.
**Proof:** B, `3.4-canvas-tour-blend.png` (the blend-mode page).

### 3.5 Blur, shadows, backdrop filter and shader mask on the GPU
**Read:** SPEC §9.1 rows "BoxShadow / drawShadow", "Blur", "Layers".
**Tasks:** an analytic rrect shadow shader; a separable blur with downsampling; backdrop filter (copy the destination, blur, composite); shader mask.
**Display tests:** the `effects` scene matches the software golden.
**Proof:** B, `3.5-canvas-tour-effects.png`.

### 3.6 GPU parity and the raster cache groundwork
**Goal:** `gles` and `soft` agree on every golden; `Picture.ToImage` on the GPU.
**Read:** SPEC §9.1 "Raster cache", §9.2.
**Tasks:**
1. A test that renders **every** golden scene registered so far through both backends and compares them within tolerance (display tag).
2. `Renderer.Snapshot` / `Picture.ToImage` through `gles`.
3. Retained `EngineLayer` textures (needed later by `RepaintBoundary`); the 3-frame caching rule stays off until 17.2.
**Display tests:** the parity test.

### 3.7 Frame pacing, resizing and fractional scale
**Goal:** the §8.11 frame-pacing and resizing requirements on Wayland.
**Read:** SPEC §8.11 "Frame pacing" and "Resizing and scaling", §8.2 rows `wp_fractional_scale_v1`, `wp_viewporter`, `xdg_wm_base` v6.
**Files:** `internal/platform/linux/wayland/` (frames, scale), `internal/engine/frames.go`.
**Tasks:**
1. At most one frame in flight per window; input is batched into the next frame.
2. Suspended windows don't render. When a window resumes, frame timestamps continue from "now".
3. On `configure`, render the new size before `ack_configure` + commit, without waiting for a frame callback.
4. Fractional scale: the buffer is `ceil(logical × scale)`, and the viewport destination is the logical size. Report `devicePixelRatio` to the handler. Fall back to `preferred_buffer_scale` (`wl_surface` v6), then integer output scale.
5. `examples/dev/canvas-tour`: add a page whose content depends on the window size (a grid of 1-px lines), to check scaling sharpness.
**Display tests:** a resize sequence never presents a buffer of the old size after `ack_configure`; under sway with `output HEADLESS-1 scale 1.5`, the buffer size is 1.5× the logical size.
**Proof:** C, `3.7-fractional-scale.png`: headless sway at scale 1.5 showing the 1-px grid page with crisp, evenly spaced lines.

---

## M4: Text

### 4.1 Fonts: collection, fontconfig and bundled fonts
**Goal:** find and load fonts the way Flutter's Linux engine does.
**Read:** SPEC §10 (first bullets and the fontconfig bullet), §8.1 (the libfontconfig row), D4, D16.
**Files:** `internal/text/fonts.go`, `internal/text/fontconfig/`, `material/fonts/` (font files + licenses), `scripts/fetch-fonts.sh`.
**Tasks:**
1. Add `github.com/go-text/typesetting` to `go.mod`.
2. Bind `libfontconfig.so.1` (non-variadic functions only: `FcInitLoadConfigAndFonts`, `FcPatternCreate`, `FcPatternAddString`, `FcPatternAddInteger`, `FcConfigSubstitute`, `FcDefaultSubstitute`, `FcFontSort`, `FcPatternGetString`, `FcPatternGetInteger`, `FcFontSetDestroy`, `FcPatternDestroy`). `Match(family, weight, slant, lang) []FontFile` returns the fallback order.
3. Without fontconfig, use `fontscan` (cache in `$XDG_CACHE_HOME/glutter/fonts`).
4. `scripts/fetch-fonts.sh`: download Roboto (variable TTF) and `MaterialIcons-Regular.ttf`, with their license files, from Google's official GitHub repositories, at pinned commits recorded in `material/fonts/VERSIONS`. Run it and commit the files. They're embedded with `embed` in `material/fonts/fonts.go`.
5. A face cache keyed by file + index + variations. Loading failures give `FontLoadFailed`.
**Tests:** fontconfig `Match("sans-serif")` returns at least one file (display-independent, but skipped with a logged reason if fontconfig is missing); `fontscan` fallback path; route test for `FontLoadFailed`.

### 4.2 Paragraph layout core
**Goal:** `ui.ParagraphBuilder` / `ui.Paragraph` with shaping, bidi and line breaking.
**Read:** SPEC §7 "Text", §10 "Paragraph layout" steps 1–5 and 8.
**Port from:** `engine/src/flutter/lib/ui/text.dart` (the API and its documented behavior).
**Files:** `internal/text/layout/`, `ui/text.go`.
**Tasks:**
1. `ui` types: `TextStyle`, `ParagraphStyle`, `StrutStyle`, `FontWeight`, `FontStyle`, `TextAlign`, `TextDirection`, `TextDecoration`, `TextBaseline`, `TextHeightBehavior`, `FontFeature`, `FontVariation`, `Locale`.
2. Layout pipeline: spans → runs (script, direction and font fallback through 4.1) → `go-text` shaping → UAX #14 line breaking honoring `softWrap` → `maxLines` and ellipsis → bidi reordering per line → alignment and justification → height, leading distribution and strut → metrics.
3. `Paragraph`: `Layout(width)`, `Width`, `Height`, `MinIntrinsicWidth`, `MaxIntrinsicWidth`, `AlphabeticBaseline`, `IdeographicBaseline`, `LongestLine`, `DidExceedMaxLines`, `ComputeLineMetrics`.
**Tests:** table tests: line breaks for Latin text at several widths; ellipsis with `maxLines=2`; RTL Arabic run order; mixed LTR/RTL; CJK breaking; intrinsic widths.

### 4.3 Paragraph queries, placeholders and decorations
**Goal:** everything editing and selection need, and inline widgets.
**Read:** SPEC §7 "Text" (`Paragraph` query methods), §10 steps 6–7 and "Queries for editing".
**Tasks:**
1. `GetBoxesForRange` (with `BoxHeightStyle` and `BoxWidthStyle`), `GetPositionForOffset`, `GetWordBoundary`, `GetLineBoundary`, `GetGlyphInfoAt`, `GetClosestGlyphInfoForOffset`, `GetBoxesForPlaceholders`.
2. Placeholders (`AddPlaceholder` with every `PlaceholderAlignment`).
3. Decorations: underline, overline and line-through in solid, double, dotted, dashed and wavy styles, with thickness and color; letter and word spacing; font features and variations passed to shaping.
**Tests:** caret positions inside ligatures and grapheme clusters (emoji with ZWJ); boxes across a bidi boundary; placeholder baseline alignment.

### 4.4 Drawing text (glyph atlas)
**Goal:** `Canvas.DrawParagraph` on both renderers.
**Read:** SPEC §9.1 row "Text".
**Files:** `internal/gpu/glyphs.go` (shared atlas), `internal/gpu/soft/text.go`, `internal/gpu/gles/text.go`, `examples/dev/text-tour/main.go`.
**Tasks:**
1. Rasterize glyph outlines with the 2.6 rasterizer into a grayscale atlas, at 4 horizontal subpixel positions; color glyphs (COLR, CBDT/sbix bitmaps) as far as go-text supports them.
2. `DrawParagraph` in `soft` and `gles`, including decorations and shadows (`TextStyle.Shadows`).
3. Register the proof scene `text-tour` covering:
   - Latin text in several sizes and weights
   - italic
   - Arabic and Hebrew (RTL)
   - a CJK line
   - an emoji line
   - every decoration style
   - an ellipsized paragraph
   - a justified paragraph
4. `examples/dev/text-tour` shows the same content in a window.
**Tests:** text goldens through `soft`; parity through `gles` (display tag).
**Proof:** A, `4.4-text-tour.png`; B, `4.4-text-tour-window.png`. Both show readable, correctly shaped text: Arabic letters joined and right-to-left, emoji in color, and an ellipsis on the clipped paragraph.
**Example:** `examples/dev/text-tour`.

---

## M5: foundation, scheduler, physics, gestures

### 5.1 foundation
**Read:** SPEC §11.1 (`foundation` row), §12.3 (`Opt`), §12.9, §14.3.
**Port from:** `packages/flutter/lib/src/foundation/` (`key.dart`, `change_notifier.dart`, `diagnostics.dart`, `platform.dart`, `constants.dart`).
**Tests:** `packages/flutter/test/foundation/change_notifier_test.dart`, `diagnostics_test.dart` (the relevant cases).
**Tasks:**
1. `Key`, `LocalKey`, `ValueKey[T]`, `ObjectKey`, `UniqueKey` (`GlobalKey` is declared in 7.2).
2. `Opt[T]` + `Some`, `None`; `Listenable`, `ChangeNotifier`, `ValueNotifier[T]` with Flutter's listener semantics (adding or removing during notification).
3. `TargetPlatform` (+ test override); `DebugMode`, `ProfileMode`, `ReleaseMode` constants chosen by the build tags `glutter_release` / `glutter_profile`.
4. `DiagnosticsNode`, `Diagnosticable` and the text tree dump.
5. The internal helper `opts.First[T](opts []T) T`, which panics with tag `MultipleOptions` when given more than one (SPEC §12.1).

### 5.2 `ui.PlatformDispatcher`, `ui.View` and engine wiring
**Goal:** the framework-facing engine API on top of `internal/engine`.
**Read:** SPEC §7 "Platform", §13.
**Port from:** `engine/src/flutter/lib/ui/platform_dispatcher.dart`, `window.dart`.
**Tasks:**
1. `PlatformDispatcher`: views, `OnBeginFrame`/`OnDrawFrame`, `ScheduleFrame`, the pointer data packet callback, the key data callback, locales, text scale factor, platform brightness, accessibility features, app lifecycle, `RequestAppExit`. `View`: id, physical size, DPR, padding/insets, display, `Render(scene)`.
2. `internal/engine` translates the platform `Handler` calls into dispatcher callbacks, and `View.Render` into `Renderer.Render`.
3. The dispatcher works over the fake platform (for tests) and the Linux platform.
**Tests:** with the fake platform: `ScheduleFrame` → begin/draw callbacks with monotonic timestamps; metrics changes reach the view.

### 5.3 scheduler
**Read:** SPEC §11.1 (`scheduler` row).
**Port from:** `packages/flutter/lib/src/scheduler/binding.dart`, `ticker.dart`, `priority.dart`.
**Tests:** `packages/flutter/test/scheduler/` (binding, ticker, scheduler tests).
**Tasks:** `SchedulerBinding` (transient, persistent and post-frame callbacks, `SchedulerPhase`, frame scheduling through the dispatcher, `TimeDilation`); `Ticker`, `TickerProvider`, `TickerFuture` (completion as a channel plus callbacks, SPEC §12.6). Add `internal/fakeclock` for tests.

### 5.4 physics
**Port from:** `packages/flutter/lib/src/physics/`.
**Tests:** `packages/flutter/test/physics/`.
**Tasks:** every simulation in SPEC §11.1's `physics` row, plus `Tolerance`.

### 5.5 gestures, part 1: events, hit testing, arena
**Read:** SPEC §11.1 (`gestures` row), §8.5.
**Port from:** `gestures/events.dart`, `converter.dart`, `hit_test.dart`, `pointer_router.dart`, `arena.dart`, `pointer_signal_resolver.dart`, `binding.dart`, `resampler.dart`.
**Tests:** `packages/flutter/test/gestures/` (events, converter, arena, pointer_router, hit_test).
**Tasks:** every `PointerEvent` type; converting `ui.PointerData` into events (`PointerEventConverter`); `HitTestResult`/`HitTestEntry`; `PointerRouter`; `GestureArenaManager`; `PointerSignalResolver`; `GestureBinding`, which dispatches through hit testing.

### 5.6 gestures, part 2: recognizers
**Port from:** `gestures/recognizer.dart`, `tap.dart`, `tap_and_drag.dart`, `long_press.dart`, `monodrag.dart`, `multidrag.dart`, `scale.dart`, `multitap.dart`, `eager.dart`, `velocity_tracker.dart`, `lsq_solver.dart`, `drag_details.dart`, `gesture_settings.dart`; `rendering/mouse_tracker.dart`.
**Tests:** the matching files in `packages/flutter/test/gestures/`, plus `rendering/mouse_tracker_test.dart`.
**Tasks:** every recognizer in SPEC §11.1; `VelocityTracker` (and the iOS/macOS variants); `MouseTracker` (enter, exit, hover; it lives in `rendering`). Recognizer callbacks are recovery points (SPEC §14.2), wired in 7.3: mark them `// TODO(step 7.3)`.

---

## M6: painting, rendering, semantics

### 6.1 painting: geometry, borders and decorations
**Read:** SPEC §11.1 (`painting` row), §12.3–12.4.
**Port from:** `painting/` `edge_insets.dart`, `alignment.dart`, `border_radius.dart`, `borders.dart`, `box_border.dart`, `box_decoration.dart`, `box_shadow.dart`, `gradient.dart`, `decoration.dart`, `shape_decoration.dart`, `rounded_rectangle_border.dart`, `rounded_superellipse_border.dart`, `stadium_border.dart`, `circle_border.dart`, `oval_border.dart`, `beveled_rectangle_border.dart`, `continuous_rectangle_border.dart`, `star_border.dart`, `linear_border.dart`, `colors.dart`, `matrix_utils.dart`, `basic_types.dart`, `flutter_logo.dart`, `clip.dart`.
**Tests:** the matching files in `packages/flutter/test/painting/`.
**Tasks:** port every type with its constructors (Version 2: `NewEdgeInsetsAll`, `NewBorderRadiusCircular`, …), lerps and painters (`BoxPainter`), using `ui.Canvas`. Register the proof scene `decorations`: boxes with every border type, gradients, shadows, and `FlutterLogoDecoration`.
**Proof:** A, `6.1-decorations.png`.

### 6.2 painting: text and images
**Port from:** `painting/` `text_style.dart`, `strut_style.dart`, `inline_span.dart`, `text_span.dart`, `placeholder_span.dart`, `text_painter.dart`, `text_scaler.dart`, `image_provider.dart`, `image_resolution.dart`, `image_cache.dart`, `image_stream.dart`, `decoration_image.dart`, `binding.dart` (`PaintingBinding`).
**Tests:** `painting/text_painter_test.dart`, `text_style_test.dart`, `text_span_test.dart`, `image_cache_test.dart`, `image_provider_test.dart`, `image_stream_test.dart`, `decoration_image_test.dart` (the relevant cases).
**Tasks:**
1. `TextStyle`, its lerp and `Merge`; spans; `TextPainter` on `ui.Paragraph`; `TextScaler`.
2. `ImageProvider` implementations: `AssetImage`/`ExactAssetImage` over `fs.FS` (resolution variants `2.0x/`), `FileImage`, `MemoryImage`, `NetworkImage` (`net/http` with a context), `ResizeImage`. `ImageCache` with size limits; `ImageStream` and its completers (multi-frame for GIF). Errors: `AssetNotFound`, `ImageDecodeFailed`.

### 6.3 rendering core
**Goal:** the render tree, layout protocol, paint, compositing and hit testing.
**Read:** SPEC §11.1 (`rendering` row), §12.5 (`InitRenderBox`).
**Port from:** `rendering/object.dart`, `box.dart`, `layer.dart`, `view.dart`, `binding.dart`, `debug.dart`, `layout_helper.dart`.
**Tests:** `packages/flutter/test/rendering/` `box_test.dart`, `object_test.dart`, `layer_test.dart`, `relayout_boundary_test.dart`, `repaint_boundary_test.dart`, `view_test.dart`, `box_constraints_test.dart`, `dynamic_intrinsics_test.dart`.
**Tasks:**
1. `RenderObject` and `RenderBox`, with the outer-pointer pattern `InitRenderBox(&r.RenderBox, r)`. `BoxConstraints` with every method; `ParentData`/`BoxParentData`/`ContainerBoxParentData`; the child-list mixin as an embedded helper.
2. `PipelineOwner`: layout, compositing bits, paint and semantics flush; relayout and repaint boundaries; `PaintingContext`.
3. The layer tree (every layer in SPEC §11.1) → `ui.SceneBuilder`; `RenderView` per `ui.View`; `RendererBinding`.
4. Hit testing (`BoxHitTestResult`, transforms). Layout and paint are recovery points (SPEC §14.2), wired in 7.3: mark them `// TODO(step 7.3)`.
5. A temporary test harness that pumps a render tree into a software image, used until `gluttertest` exists.

### 6.4 render objects: single child
**Port from:** `rendering/proxy_box.dart`, `shifted_box.dart`, `custom_paint.dart`, `rotated_box.dart`, `performance_overlay.dart` (skip; it's in 17.3).
**Tests:** `rendering/` `proxy_box_test.dart`, `shifted_box_test.dart`, `constraints_test.dart`, `aspect_ratio_test.dart`, `intrinsic_width_test.dart`, `limited_box_test.dart`, `fractionally_sized_box_test.dart`, `baseline_test.dart`, `transform_test.dart`, `custom_paint_test.dart`, `mouse_region_test.dart`, `annotated_region_test.dart`, `offstage_test.dart`, `rotated_box_test.dart`.
**Tasks:** every single-child render object listed in SPEC §11.1's `rendering` row.

### 6.5 render objects: multiple children
**Port from:** `rendering/flex.dart`, `stack.dart`, `wrap.dart`, `flow.dart`, `table.dart`, `table_border.dart`, `list_body.dart`, `custom_layout.dart`.
**Tests:** `rendering/flex_test.dart`, `flex_overflow_test.dart`, `stack_test.dart`, `wrap_test.dart`, `flow_test.dart`, `table_test.dart`, `table_border_test.dart`, `list_body_test.dart`.
**Tasks:** `RenderFlex` (including Flutter's overflow indicator in debug builds), `RenderStack`, `RenderIndexedStack`, `RenderWrap`, `RenderFlow`, `RenderTable`, `RenderListBody`, `RenderCustomMultiChildLayoutBox`.

### 6.6 RenderParagraph, RenderImage and the rendering tour
**Port from:** `rendering/paragraph.dart`, `image.dart`, `selection.dart`.
**Tests:** `rendering/paragraph_test.dart`, `paragraph_intrinsics_test.dart`, `image_test.dart`.
**Tasks:**
1. `RenderParagraph`, including placeholders for `WidgetSpan` children and `SelectionRegistrar` hooks; `RenderImage`.
2. Register the proof scene `rendering-tour`: a render tree with a `RenderFlex` row of three decorated boxes, a `RenderStack` with a positioned badge, a `RenderWrap` of chips, and a `RenderParagraph`. It's laid out by the pipeline, not drawn by hand.
**Proof:** A, `6.6-rendering-tour.png`.

### 6.7 semantics
**Read:** SPEC §11.1 (`semantics` row), §8.10 (first sentence).
**Port from:** `semantics/semantics.dart`, `semantics_event.dart`, `binding.dart`; the semantics parts of `rendering/object.dart` and `proxy_box.dart`.
**Tests:** `packages/flutter/test/semantics/semantics_test.dart`, `rendering/semantics_test.dart`.
**Tasks:** `SemanticsNode`, `SemanticsConfiguration`, `SemanticsOwner`, actions, flags, roles, `SemanticsProperties`, sort keys; the `PipelineOwner` semantics flush producing `ui.SemanticsUpdate` (add `ui.SemanticsUpdateBuilder` now). Semantics is enabled only on request (SPEC §8.10).

---

## M7: widgets core and `gluttertest`

### 7.1 Element tree, stateless and stateful widgets, keys
**Goal:** the heart of the framework: widgets build elements, and elements reconcile.
**Read:** SPEC §11.1 (`widgets` row, "Framework"), §12.1–12.5 (all of it), §12.6 (`SetState`).
**Port from:** `widgets/framework.dart` (`Widget`, `Element`, `ComponentElement`, `StatelessElement`, `StatefulElement`, `State`, `BuildOwner`, `BuildContext`, `updateChild`, `updateChildren`, the inactive-element lifecycle).
**Tests:** `packages/flutter/test/widgets/framework_test.dart`, `key_test.dart`, `reparent_state_test.dart`, `stateful_component_test.dart`, `build_scope_test.dart` (the relevant cases, run against a minimal internal harness until 7.5).
**Tasks:**
1. The `Widget` interface and base structs `StatelessWidget`/`StatefulWidget` carrying the key; `NewStatelessWidgetBase(opts ...WidgetOpts)`, `NewStatefulWidgetBase(...)`.
2. The element kind is chosen from the interfaces the outer type implements (SPEC §12.5).
3. `StateOf[W]`: `Widget()`, `Context()`, `Mounted()`, `SetState`, `Lifetime()`. The optional lifecycle interfaces are `InitState`, `DidChangeDependencies`, `DidUpdateWidget(old W)`, `Deactivate`, `Activate`, `Dispose`, `Reassemble`.
4. `BuildOwner` with dirty lists and build scopes. `updateChildren` must be Flutter's keyed reconciliation algorithm, ported exactly.
5. `SetState` on the UI thread runs immediately and marks the element dirty. Calling it after dispose panics with `SetStateAfterDispose`. The goroutine-safe path is in 7.3.

### 7.2 Inherited widgets, GlobalKey, ParentDataWidget, RenderObjectWidget
**Port from:** `widgets/framework.dart` (`InheritedWidget`, `InheritedElement`, `ProxyWidget`, `ParentDataWidget`, `RenderObjectWidget`, `LeafRenderObjectWidget`, `SingleChildRenderObjectWidget`, `MultiChildRenderObjectWidget`, `GlobalKey`); `widgets/inherited_notifier.dart`, `inherited_model.dart`, `inherited_theme.dart`.
**Tests:** `widgets/inherited_test.dart`, `inherited_model_test.dart`, `global_key_test.dart` (if present; otherwise the `GlobalKey` cases in `framework_test.dart`), `parent_data_test.dart`.
**Tasks:** every type above; `DependOn[T](bc)` and `GetInherited[T](bc)` (SPEC §12.5); `GlobalKey[S]` with `CurrentState()`, `CurrentContext()` and reparenting; `ErrorWidget` (the type only; display rules in 7.3).

### 7.3 Error containment, goroutines and thread checks (D15)
**Goal:** SPEC §14.2, completely.
**Read:** SPEC §14.2 (all of it), §13, §12.6.
**Port from:** `foundation/assertions.dart` (`FlutterError`, `FlutterErrorDetails`, `reportError`); the try/catch sites in `framework.dart`, `rendering/object.dart`, `gestures/recognizer.dart`, `foundation/change_notifier.dart`, `scheduler/binding.dart` and `painting/image_stream.dart`.
**Tasks:**
1. `recover` at every recovery point in the SPEC §14.2 table, replacing the `// TODO(step 7.3)` markers from 5.6 and 6.3. Each recovered panic becomes `BuildPanicked`, `LayoutPanicked`, `PaintPanicked` or `CallbackPanicked`, carrying the panic value, the stack and the element path.
2. `ErrorReporter` (default: `slog` + `ToJson()`) and `ErrorWidgetBuilder` (default: red box with the message in debug, gray box in release).
3. `SetState` from other goroutines is queued through the binding's `Post` and dropped if the state is unmounted. `State.Go(func(ctx))` runs under the same recovery.
4. Debug builds check the calling goroutine on the main entry points (`SetState` on the UI path, `MarkNeedsBuild`, `MarkNeedsLayout`, `ChangeNotifier.NotifyListeners`) and report `WrongThread`.
**Tests:** a panic in `Build` → the error widget shows and the siblings keep working; a panic in layout or paint; a panic in a tap callback; `SetState` from 50 goroutines; `State.Go` recovery; a route test for every tag.

### 7.4 WidgetsBinding, RunApp and capture-on-idle
**Read:** SPEC §13, §7 "Platform", §8.7 "Closing".
**Port from:** `widgets/binding.dart`, `widgets/view.dart`, `widgets/media_query.dart`, `widgets/basic.dart` (`Directionality`), `widgets/app_lifecycle_listener.dart`.
**Tests:** `widgets/binding_test.dart`, `media_query_test.dart`, `app_lifecycle_listener_test.dart`, `view_test.dart` (the relevant cases).
**Tasks:**
1. `WidgetsBinding` combines the gesture, scheduler, painting, semantics, renderer and widgets bindings, as an instance created by `RunApp` and reachable through `WidgetsBindingOf(bc)` (no global; SPEC §13).
2. `widgets.RunApp(app, opts ...RunAppOpts) error`. Its options are listed in SPEC §13 and §14.2. It creates the engine, the first window and the root view, and returns when the last window closes or after `SystemNavigatorPop`.
3. `View`, `MediaQuery` (every `MediaQueryData` field Glutter can provide), `Directionality`, `AppLifecycleListener` (including `OnExitRequested`), `DebugOptions` (SPEC §14.3 flags as fields).
4. Capture on idle: with `GLUTTER_SCREENSHOT` set and no `GLUTTER_SCREENSHOT_AFTER`, capture after the first frame at which no further frame is scheduled and no images are pending (max 5 s, then capture anyway and log a warning). With several windows, write `<name>-<viewID>.png` per window.
**Tests:** `RunApp` over the fake platform; the lifecycle states; route tests for environment errors coming from the engine.

### 7.5 `gluttertest`
**Goal:** Flutter-style widget tests, no display needed.
**Read:** SPEC §17 "Widget tests", "Golden tests", §14.2 "Tests".
**Port from:** `packages/flutter_test` is **not** in `reference/`. Port the behavior from Flutter's documentation of `WidgetTester`, `Finder` and `TestWidgetsFlutterBinding`, and ask the user if a behavior is unclear.
**Files:** `gluttertest/`.
**Tasks:**
1. `gluttertest.New(tb)` → `*Tester` over the fake platform, `internal/fakeclock` and the software renderer.
2. Methods: `PumpWidget`, `Pump(d)`, `PumpAndSettle`, `PumpFrames`, `Tap`, `LongPress`, `Drag`, `Fling`, `Hover`, `Scroll`, `EnterText` (stub until 11.4), `SendKeyEvent`, `SendKeyDownEvent`, `SendKeyUpEvent`, `TakeError`.
3. Finders: `ByText`, `ByKey`, `ByType[T]`, `ByIcon`, `ByTooltip`, `BySemanticsLabel`, `Descendant`, `Ancestor`.
4. `MatchesGolden(name)`, using `internal/goldens`.
5. Any reported error fails the test unless taken.
6. Move the tests from 7.1–7.4 off the internal harness onto `gluttertest`.
**Tests:** `gluttertest`'s own tests (a counter-like widget built from `GestureDetector`).

### 7.6 Basic widgets, part 1: layout and painting
**Read:** SPEC §11.1 (`widgets` row, "Basic", first half), §12.1–12.4.
**Port from:** `widgets/basic.dart`, `container.dart`, `visibility.dart`, `spacer.dart`, `placeholder.dart`.
**Tests:** `widgets/` `basic_test.dart`, `container_test.dart`, `align_test.dart`, `center_test.dart`, `sized_box_test.dart`, `flex_test.dart`, `row_test.dart`, `column_test.dart`, `stack_test.dart`, `wrap_test.dart`, `flow_test.dart`, `table_test.dart`, `opacity_test.dart`, `transform_test.dart`, `clip_test.dart`, `custom_paint_test.dart`, `visibility_test.dart`, `offstage_test.dart`, `placeholder_test.dart`, `fitted_box_test.dart`, `aspect_ratio_test.dart`.
**Tasks:** every widget from Container through RepaintBoundary in the SPEC §11.1 "Basic" list (layout, painting and clipping widgets), each with a Version 2 constructor (SPEC §12.1–12.2).

### 7.7 Basic widgets, part 2: text, images, input and builders
**Port from:** `widgets/text.dart`, `icon.dart`, `icon_theme.dart`, `icon_data.dart`, `image.dart`, `image_icon.dart`, `gesture_detector.dart`, `layout_builder.dart`, `value_listenable_builder.dart`, `async.dart`, `localizations.dart`, `title.dart`, `safe_area.dart`, `basic.dart` (`Builder`, `StatefulBuilder`, `Listener`, `MouseRegion`, `Semantics`, …), `transitions.dart` (`ListenableBuilder`, `AnimatedBuilder` only).
**Tests:** `widgets/` `text_test.dart`, `rich_text_test.dart`, `icon_test.dart`, `image_test.dart`, `gesture_detector_test.dart`, `listener_test.dart`, `mouse_region_test.dart`, `layout_builder_test.dart`, `value_listenable_builder_test.dart`, `async_test.dart`, `localizations_test.dart`, `semantics_test.dart` (the relevant cases).
**Tasks:**
1. The rest of the SPEC §11.1 "Basic" list.
2. `FutureBuilder` takes `func(ctx context.Context) (T, error)`, and `StreamBuilder` takes `<-chan T` (SPEC §12.6).
3. English `WidgetsLocalizations`.

### 7.8 `examples/layout`
**Goal:** the first public example, using only public packages.
**Read:** SPEC §19; this file's "Examples" rules.
**Tasks:** `examples/layout/main.go`, using `widgets.RunApp`, with five labelled sections:
- `Row` with `Expanded` and `Flexible` ratios
- `Column` with every `MainAxisAlignment`
- `Stack` with a `Positioned` badge
- `Wrap` of 12 colored tags
- a `Table`

It uses `DecoratedBox` borders and `Text` labels. Window title: "Glutter – layout".
**Tests:** a `gluttertest` golden of the example's root widget (`examples/layout/layout_test.go`).
**Proof:** B, `7.8-layout.png`: all five sections visible and labelled.
**Example:** `examples/layout`.

### 7.9 `examples/custompaint`
**Tasks:** `examples/custompaint/main.go`: a `CustomPaint` with a custom painter that draws an analog clock (ticks, hands, a gradient face, a shadow). A `StatefulWidget` updates it every second with `State.Go` + `SetState`. It repaints only through `ShouldRepaint`.
**Tests:** golden at a fixed time, with the clock injected through the widget's constructor.
**Proof:** B, `7.9-custompaint.png`: a smooth, antialiased clock face with the hands at the current time.
**Example:** `examples/custompaint`.

---

## M8: animation

### 8.1 Animation library
**Read:** SPEC §11.1 (`animation` row), §12.7.
**Port from:** `animation/animation.dart`, `animation_controller.dart`, `animations.dart`, `curves.dart`, `tween.dart`, `tween_sequence.dart`, `listener_helpers.dart`, `animation_style.dart`.
**Tests:** `packages/flutter/test/animation/` (every file, relevant cases).
**Tasks:** every type in SPEC §11.1's `animation` row. `Tween[T]` has built-in lerps for the types in SPEC §12.7, plus the `Lerp(T, float64) T` interface. `NewAnimationController(vsync, opts...)` (SPEC §12.1).

### 8.2 Ticker providers and implicit animations
**Port from:** `widgets/ticker_provider.dart`, `implicit_animations.dart`, `tween_animation_builder.dart`.
**Tests:** `widgets/ticker_provider_test.dart`, `implicit_animations_test.dart`, `tween_animation_builder_test.dart`.
**Tasks:** `SingleTickerProvider`, `TickerProvider` and `TickerMode` as embeddable structs (SPEC §12.5); every implicitly animated widget in SPEC §11.1 except `AnimatedTheme` (12.2); `TweenAnimationBuilder`. Animation listeners are recovery points (7.3).

### 8.3 Transitions and switchers
**Port from:** `widgets/transitions.dart`, `animated_switcher.dart`, `animated_cross_fade.dart`, `animated_size.dart`, `modal_barrier.dart` (`AnimatedModalBarrier`); `rendering/animated_size.dart`.
**Tests:** `widgets/transitions_test.dart`, `animated_switcher_test.dart`, `animated_cross_fade_test.dart`, `animated_size_test.dart`.
**Tasks:** every `XTransition`, `AnimatedSwitcher`, `AnimatedCrossFade`, `AnimatedSize`, `AnimatedModalBarrier`. (`AnimatedList` comes in 9.3, `Hero` in 10.2.)

### 8.4 `examples/animations`
**Tasks:** `examples/animations/main.go`, with four tiles:
1. an `AnimatedContainer` that changes size, color and border radius on tap
2. the `Pulse` widget from SPEC §12 (`ScaleTransition` with a repeating controller)
3. an `AnimatedSwitcher` cycling a counter
4. a `TweenAnimationBuilder` rotating an icon-like shape

A `Text` shows `Curves` names.
**Tests:** golden tests at t = 0, 150 ms and 300 ms using `Pump`.
**Proof:** B with `GLUTTER_SCREENSHOT_AFTER=400ms`, `8.4-animations.png`: all four tiles visible mid-animation (the pulse scaled, the container part-way between colors).
**Example:** `examples/animations`.

---

## M9: scrolling

### 9.1 Viewports and slivers (rendering)
**Port from:** `rendering/viewport.dart`, `viewport_offset.dart`, `sliver.dart`, `sliver_list.dart`, `sliver_fixed_extent_list.dart`, `sliver_varied_extent_list.dart` (if present), `sliver_grid.dart`, `sliver_multi_box_adaptor.dart`, `sliver_padding.dart`, `sliver_fill.dart`, `sliver_persistent_header.dart`, `sliver_group.dart`, `proxy_sliver.dart`, `sliver_tree.dart` (skip if it's experimental).
**Tests:** `rendering/viewport_test.dart`, `slivers_test.dart`, `sliver_cache_test.dart`, `slivers_block_test.dart`, `sliver_group_test.dart` (the relevant cases).
**Tasks:** the sliver protocol (`SliverConstraints`, `SliverGeometry`), `RenderViewport`, `RenderShrinkWrappingViewport`, `RenderAbstractViewport`, and every sliver render object in SPEC §11.1's `rendering` row, including the grid delegates.

### 9.2 Scrollable, physics and scrollbars
**Port from:** `widgets/scrollable.dart`, `scroll_controller.dart`, `scroll_position.dart`, `scroll_position_with_single_context.dart`, `scroll_physics.dart`, `scroll_simulation.dart`, `scroll_activity.dart`, `scroll_configuration.dart`, `scroll_notification.dart`, `scroll_metrics.dart`, `scrollbar.dart`, `scroll_delegate.dart`, `scroll_context.dart`, `scrollable_helpers.dart`.
**Tests:** `widgets/scrollable_test.dart`, `scroll_controller_test.dart`, `scroll_physics_test.dart`, `scroll_activity_test.dart`, `scroll_behavior_test.dart`, `scrollbar_test.dart`, `scroll_notification_test.dart` (the relevant cases).
**Tasks:**
1. `Scrollable`, `ScrollController`, `ScrollPosition` and the physics.
2. `ScrollConfiguration`/`ScrollBehavior` with the desktop defaults (SPEC §11.1): scrollbars visible, no mouse drag-scroll, high-resolution wheel, trackpad kinetic scrolling from `PointerPanZoom`.
3. Notifications; `RawScrollbar`.

### 9.3 Scroll views and lists
**Port from:** `widgets/scroll_view.dart`, `sliver.dart`, `single_child_scroll_view.dart`, `page_view.dart`, `nested_scroll_view.dart`, `reorderable_list.dart`, `list_wheel_scroll_view.dart`, `interactive_viewer.dart`, `animated_scroll_view.dart`, `sliver_fill.dart`, `sliver_persistent_header.dart`, `automatic_keep_alive.dart`, `keep_alive` (in `sliver.dart`), `two_dimensional_scroll_view.dart`, `two_dimensional_viewport.dart`.
**Tests:** the matching `widgets/*_test.dart` files (list_view, grid_view, custom_scroll_view, page_view, nested_scroll_view, reorderable_list, list_wheel_scroll_view, interactive_viewer, animated_list, animated_grid, keep_alive, two_dimensional_*).
**Tasks:** every widget in SPEC §11.1's "Scrolling" list, plus `AnimatedList`, `SliverAnimatedList` and `AnimatedGrid`. Named constructors follow SPEC §12.4 (`NewListViewBuilder`, `NewListViewSeparated`, `NewGridViewCount`, …).

### 9.4 `examples/scrolling`
**Tasks:** `examples/scrolling/main.go`, with a tab-like row of buttons (plain `GestureDetector`s; Material isn't available yet) switching between:
1. a `ListViewBuilder` with 10,000 rows of text
2. a `GridViewCount` of colored tiles
3. a `CustomScrollView` with a pinned `SliverPersistentHeader`, a list and a grid
4. a `PageView` of three pages
5. a `ReorderableListView`

Scrollbars are visible.
**Tests:** a golden of each tab; a fling test with `gluttertest`.
**Proof:** B, `9.4-scrolling.png` (the `CustomScrollView` tab, scrolled so the pinned header overlaps the list; use `GLUTTER_DEV_TAB=2` and `GLUTTER_DEV_SCROLL=600` read by the example in debug builds).
**Example:** `examples/scrolling`.

---

## M10: navigation, services, focus

### 10.1 Overlay and OverlayPortal
**Port from:** `widgets/overlay.dart`, `modal_barrier.dart`.
**Tests:** `widgets/overlay_test.dart`, `overlay_portal_test.dart`, `modal_barrier_test.dart`.
**Tasks:** `Overlay`, `OverlayEntry`, `OverlayPortal`, `ModalBarrier`.

### 10.2 Navigator (imperative API) and Hero
**Read:** SPEC §12.6 (`NavigatorPush` returns a channel), D13.
**Port from:** `widgets/navigator.dart`, `routes.dart`, `pages.dart`, `heroes.dart`, `pop_scope.dart`, `navigator_pop_handler.dart`, `transitions.dart` (route transitions).
**Tests:** `widgets/navigator_test.dart`, `routes_test.dart`, `heroes_test.dart`, `pop_scope_test.dart`, `page_route_builder_test.dart` (the relevant cases).
**Tasks:**
1. `Navigator`, `Route`, `OverlayRoute`, `TransitionRoute`, `ModalRoute`, `PageRoute`, `PopupRoute`, `RawDialogRoute`, `PageRouteBuilder`, `PopScope`, `BackButtonListener` (no back button on desktop; keep it for API parity), `Hero`.
2. `NavigatorOf(bc)` and `NavigatorPush(bc, route) <-chan any`, with every static method mapped per SPEC §12.4.

### 10.3 Navigator Pages API and Router
**Port from:** `widgets/navigator.dart` (`Page`, `pages:`, `onDidRemovePage`), `router.dart`, `route_notification_messages.dart`.
**Tests:** `widgets/router_test.dart`, `navigator_test.dart` (the Pages cases).
**Tasks:** `Page`, `MaterialPage`-independent `CustomPage`; `Router`, `RouterDelegate`, `RouteInformationParser`, `RouteInformationProvider` (`PlatformRouteInformationProvider` keeps routes in memory on desktop), `BackButtonDispatcher`.

### 10.4 services: keyboard, cursors, clipboard text, assets
**Read:** SPEC §11.1 (`services` row), §8.5.
**Port from:** `services/hardware_keyboard.dart`, `keyboard_key.g.dart` (generate it: extend `glutter-gen keys` to write `services/keyboard_key_gen.go` with every named constant), `keyboard_maps.g.dart` (Linux parts), `mouse_cursor.dart`, `mouse_tracking.dart`, `clipboard.dart`, `asset_bundle.dart`, `system_navigator.dart`, `system_sound.dart`.
**Tests:** `services/hardware_keyboard_test.dart`, `keyboard_key_test.dart`, `mouse_cursor_test.dart`, `asset_bundle_test.dart` (the relevant cases).
**Tasks:**
1. `HardwareKeyboard`, `KeyEvent` types, lock modes, focus-loss synthesis (SPEC §8.5).
2. `MouseCursor` / `SystemMouseCursors`, wired to `Window.SetCursor` for every cursor 1.8 supports.
3. `Clipboard.SetData`/`GetData` for text. The platform side uses `wl_data_device` and X11 `CLIPBOARD`, read and written through pipes on a goroutine (SPEC §8.11 "Data exchange").
4. `AssetBundle` over `RunAppOpts.Assets`. `SystemNavigatorPop` closes the window.

### 10.5 Focus system
**Port from:** `widgets/focus_manager.dart`, `focus_scope.dart`, `focus_traversal.dart`.
**Tests:** `widgets/focus_manager_test.dart`, `focus_scope_test.dart`, `focus_traversal_test.dart`.
**Tasks:** `FocusNode`, `FocusScopeNode`, `FocusManager`, `Focus`, `FocusScope`, `ExcludeFocus`, `FocusTraversalGroup`, every traversal policy, and highlight mode (keyboard vs pointer).

### 10.6 Shortcuts, Actions and Intents
**Port from:** `widgets/shortcuts.dart`, `actions.dart`, `text_editing_intents.dart` (the intent types only), `widgets/app.dart` (the default shortcuts map).
**Tests:** `widgets/shortcuts_test.dart`, `actions_test.dart`.
**Tasks:** `Shortcuts`, `ShortcutManager`, `SingleActivator`, `CharacterActivator`, `LogicalKeySet`, `Actions`, `Action[T]`, `CallbackAction`, `Intent`, `CallbackShortcuts`, `DefaultTextEditingShortcuts` (Linux bindings), and `WidgetsApp` with its default shortcuts and actions.

### 10.7 `examples/navigation`
**Tasks:** `examples/navigation/main.go` using `WidgetsApp` (no Material yet):
- Imperative navigation with a custom `PageRouteBuilder` slide transition.
- A detail page with a `Hero` image.
- A route returning a value read through the channel from `NavigatorPush`.
- A second tab using the Pages API.
- Keyboard: Tab/Shift+Tab move focus between links (visible focus highlight); Escape pops.
**Tests:** `gluttertest` tests for push, pop with a result, and keyboard focus traversal.
**Proof:** B, `10.7-navigation.png` (the detail page with a focused link highlighted).
**Example:** `examples/navigation`.

---

## M11: text editing, IME, clipboard

### 11.1 D-Bus connection and settings portal
**Read:** SPEC §8.8, §8.9 (the last paragraph: calls on a goroutine, results through `Post`), D14.
**Files:** `internal/dbus/`.
**Tasks:**
1. Add `github.com/godbus/dbus/v5`. Session bus connection with `DBusConnectFailed`; a helper for portal request/response (`org.freedesktop.portal.Request`).
2. Settings portal: read and watch the keys in SPEC §8.8 (color scheme, accent, contrast, text scaling, animations, cursor size and theme, font antialiasing and hinting).
3. Deliver settings through `Handler.OnSettings` → `PlatformDispatcher` (brightness, text scale, `DisableAnimations`, accessibility features).
**Tests:** against an in-process fake bus (SPEC §17): read, change signal, and portal missing (fallback values).

### 11.2 Text input service and Wayland text-input-v3
**Read:** SPEC §8.6, §8.11 "Input" (the IME bullet).
**Port from:** `services/text_input.dart`, `text_editing.dart`, `text_editing_delta.dart`, `text_formatter.dart`, `text_layout_metrics.dart`.
**Tests:** `services/text_input_test.dart`, `text_formatter_test.dart`, `text_editing_delta_test.dart` (the relevant cases).
**Tasks:**
1. `TextInput`, `TextInputConnection`, `TextInputClient`, `TextEditingValue`, `TextSelection`, `TextRange`, `TextEditingDelta`, the formatters.
2. Platform: `Window.SetTextInput` → `zwp_text_input_v3` (enable/disable, surrounding text, cursor rectangle, content type, preedit, commit, delete surrounding text, done serials) → `Handler.OnTextInput`.

### 11.3 IME over D-Bus (X11, and Wayland without text-input-v3)
**Read:** SPEC §8.3 (the IME bullet), §8.11 "Input".
**Port from:** nothing in `reference/`. Implement the IBus and Fcitx5 D-Bus protocols from their public documentation (the IBus `InputContext` interface, Fcitx5's `org.fcitx.Fcitx.InputContext1`). If a detail is unclear, ask the user instead of guessing.
**Tasks:**
1. IBus: `org.freedesktop.IBus` → `CreateInputContext`; `ProcessKeyEvent`, `SetCursorLocation`, `FocusIn`/`FocusOut`; the signals `CommitText`, `UpdatePreeditText`, `DeleteSurroundingText`.
2. Fcitx5: its D-Bus input-method frontend (`org.fcitx.Fcitx5`, `CreateInputContext`), with the same mapping.
3. Choose whichever is running; neither means plain keys plus xkb compose.
**Tests:** the fake bus scripts IBus and Fcitx5 sessions, checking that preedit and commit reach `OnTextInput`.

### 11.4 EditableText
**Port from:** `rendering/editable.dart`, `widgets/editable_text.dart`, `text_selection.dart`, `text_editing_intents.dart`, `default_text_editing_shortcuts.dart`, `undo_history.dart`, `spell_check.dart` (skip; no spell checking in 1.0).
**Tests:** `widgets/editable_text_test.dart`, `editable_text_shortcuts_test.dart`, `undo_history_test.dart`, `rendering/editable_test.dart` (the relevant cases; there are many, so prioritize keyboard navigation, selection, composing and undo).
**Tasks:**
1. `RenderEditable` (caret, selection painting, composing underline).
2. `EditableText`, `TextEditingController`, `UndoHistory`.
3. The Linux shortcuts (word and line movement, select all, cut/copy/paste through 10.4's clipboard).
4. Mouse selection: double-click selects a word, triple-click a line.
5. IME preedit display. `gluttertest`'s `EnterText` now works.

### 11.5 Selection, context menus, forms, autocomplete, menus, tooltip
**Port from:** `widgets/selectable_region.dart`, `selection_container.dart`, `context_menu_controller.dart`, `context_menu_button_item.dart`, `system_context_menu.dart` (skip if mobile-only), `form.dart`, `autocomplete.dart`, `raw_menu_anchor.dart`, `raw_tooltip.dart` (if present; otherwise the widgets-level tooltip basis).
**Tests:** the matching `widgets/*_test.dart` files.
**Tasks:** `SelectableRegion`, `SelectionContainer`, the context-menu infrastructure, `Form`/`FormField[T]`, `RawAutocomplete[T]`, `RawMenuAnchor`, and the tooltip basis.

### 11.6 Primary selection and rich clipboard
**Read:** SPEC §3 (the "Data exchange" row), §8.11 "Data exchange".
**Tasks:**
1. Primary selection on Wayland (`zwp_primary_selection_device_manager_v1`) and X11 (`PRIMARY`): selecting text sets it, and middle-click pastes into `EditableText`.
2. Clipboard images (`image/png`) and files (`text/uri-list`) through a `services` extension API.
3. Large X11 transfers with `INCR`.
**Tests:** fake-platform tests; display tests that copy text between two windows of one test process.

### 11.7 `examples/editor`
**Tasks:** `examples/editor/main.go` using `WidgetsApp`:
- A multi-line `EditableText` editor with a line count, word count and cursor position in a status bar.
- Undo/redo shortcuts.
- A `Shortcuts` binding for Ctrl+S that writes the text to `$XDG_DATA_HOME/glutter-editor/notes.txt` (with a `context.Context`; errors shown in the status bar).
- A read-only `SelectableRegion` panel showing the saved file.
**Tests:** `gluttertest`: type, select a word, undo, save (with a temp directory).
**Proof:** B, `11.7-editor.png`: text with a visible selection and a caret, and the status bar.
**Manual check (user):** ask the user to type with IBus or Fcitx5 (e.g. Pinyin or Japanese) on Wayland and X11 and confirm that the preedit underline shows and commit works. Record the answer in `IMPLEMENTED.md`.
**Example:** `examples/editor`.

---

## M12: Material foundation

### 12.1 Port material-color-utilities
**Read:** SPEC §11.2 (Theming), §20 (license row).
**Port from:** `reference/material-color-utilities/dart/lib/` (`hct/`, `scheme/`, `dynamiccolor/`, `palettes/`, `quantize/` (only if `ColorScheme.fromImageProvider` is needed; skip otherwise), `utils/`, `contrast/`, `blend/`).
**Tests:** `reference/material-color-utilities/dart/test/` (port the HCT, CAM16, tonal palette, dynamic scheme and contrast tests).
**Files:** `material/internal/mcu/`.
**Tasks:** port HCT, CAM16, `TonalPalette`, `DynamicScheme`, every scheme variant (tonal spot, neutral, vibrant, expressive, fidelity, content, monochrome, rainbow, fruit salad) and contrast levels. Keep a NOTICE entry (Apache-2.0).

### 12.2 Color schemes, typography and ThemeData
**Port from:** `material/color_scheme.dart`, `colors.dart`, `typography.dart`, `text_theme.dart`, `theme_data.dart`, `theme.dart`, `material_state.dart` / `widgets/widget_state.dart`, `visual_density` (in `theme_data.dart`), `constants.dart`.
**Tests:** `material/color_scheme_test.dart`, `theme_data_test.dart`, `theme_test.dart`, `typography_test.dart`, `text_theme_test.dart` (the relevant cases).
**Tasks:**
1. `ColorScheme` + `ColorSchemeFromSeed` (on 12.1), `Colors`, `Typography` (2021; Linux uses the "Helsinki" set, as in `typography.dart`), `TextTheme`, `VisualDensity` (compact on desktop), `MaterialTapTargetSize`, `WidgetState`/`WidgetStateProperty[T]`.
2. `ThemeData` (`NewThemeData`, `CopyWith`) and `Theme`/`ThemeOf`/`AnimatedTheme`.
3. Platform brightness from 11.1 picks between light and dark when `MaterialApp` is given both.
4. Implement `glutter-gen copywith`: from a struct annotated `//glutter:copywith`, it generates the `XCopy` struct (all fields `Opt[T]`) and the `CopyWith` method (SPEC §12.3). Use it for `ThemeData`, `ColorScheme`, `TextTheme` and `TextStyle`.

### 12.3 Material, ink, icons and localizations
**Port from:** `material/material.dart`, `elevation_overlay.dart`, `shadows.dart`, `ink_well.dart`, `ink_decoration.dart`, `ink_ripple.dart`, `ink_splash.dart`, `ink_highlight.dart`, `icons.dart`, `material_localizations.dart`.
**Tests:** `material/material_test.dart`, `ink_well_test.dart`, `ink_paint_test.dart`, `ink_splash_test.dart`, `localizations_test.dart` (the relevant cases).
**Tasks:**
1. `Material` (type, elevation, surface tint, shape, shadows tuned to match Material 3 shadows using 2.9/3.5).
2. `Ink`, `InkWell`, `InkResponse`, `InkRipple` (the default on Linux), `InkHighlight`, `InkSplash`.
3. Implement `glutter-gen icons`: generate `material/icons_gen.go` from the codepoints in `material/icons.dart`, wired to the embedded Material Icons font.
4. English `MaterialLocalizations`.
**Proof:** A, `12.3-material-ink.png`: Material surfaces at elevations 0, 1, 3, 6, 8 and 12 in light and dark, plus an `InkWell` frozen mid-ripple. Render it through `gluttertest` into the proof scene registry.

### 12.4 MaterialApp, re-exports and m.RunApp
**Read:** SPEC §12.11, §12.12, §13.
**Port from:** `material/app.dart`, `material/page.dart` (`MaterialPageRoute`, `MaterialPage`), `page_transitions_theme.dart`, `scrollbar.dart` (only `MaterialScrollBehavior`).
**Tests:** `material/app_test.dart`, `page_test.dart` (the relevant cases).
**Tasks:**
1. Implement `glutter-gen reexport`: it writes `material/widgets_reexport_gen.go`, with type aliases (generic too) and thin wrapper functions for every exported identifier of `widgets` (and of the `foundation`, `painting`, `animation`, `gestures`, `services` and `ui` identifiers that Flutter's `material.dart` exports).
2. `MaterialApp` (title, theme, darkTheme, themeMode, home, routes, the navigator options, routerConfig, localizations, scroll behavior, debug banner), `MaterialPageRoute`, `MaterialPage`, the page transitions theme (the Linux default), and `m.RunApp`.

### 12.5 `examples/counter`
**Goal:** the reference example from SPEC §12.12, **character for character**, except for anything the user approves changing.
**Tasks:** `examples/counter/main.go`. Before the Scaffold exists, the counter can't use `Scaffold`/`AppBar`/`FloatingActionButton`. So implement the **minimum** of those three widgets needed for the counter as part of this step: `Scaffold` with `AppBar`, `Body` and `FloatingActionButton` slots and Material 3 styling. They get completed in 13.1 and 13.3. Mark them `// TODO(step 13.1)` / `// TODO(step 13.3)`.
**Tests:** `gluttertest`: tap the FAB three times → the text shows `3`; a golden test.
**Proof:** B, `12.5-counter.png` (after tapping 3 times: use `GLUTTER_DEV_TAPS=3`, read by the example in debug builds), showing the AppBar title "Counter", the text "You have pushed the button this many times:", the number 3 and the FAB with a "+" icon.
**Example:** `examples/counter`.

---

## M13: Material components

For every step in M13:
- Port each widget **with its component theme** (`XThemeData`, `XTheme`), its Material 3 defaults (the `_XDefaultsM3` classes in Flutter's files), and its tests from `packages/flutter/test/material/<file>_test.dart` (the relevant cases).
- Add a page for the components to `examples/gallery` (created in 13.1, see below). Each page is reachable from the gallery's navigation rail.
- **Proof:** B, `13.N-gallery-<page>.png`, the gallery opened on the new page (`GLUTTER_DEV_PAGE=<page>`), in light mode.

### 13.1 App structure 1, and the gallery shell
**Port from:** `material/scaffold.dart`, `app_bar.dart`, `app_bar_theme.dart`, `bottom_app_bar.dart`, `snack_bar.dart`, `snack_bar_theme.dart`, `banner.dart`, `banner_theme.dart`, `flexible_space_bar.dart`.
**Tasks:** complete `Scaffold`, `AppBar`, `SliverAppBar` (+ medium, large), `BottomAppBar`, `ScaffoldMessenger`, `SnackBar`, `MaterialBanner`. Create `examples/gallery` with a `NavigationRail`-like side list (a plain `ListView` until 13.2), a theme toggle (light/dark) in the app bar, and the first page, "App bars & snack bars".
**Proof:** `13.1-gallery-app-bars.png`.

### 13.2 App structure 2
**Port from:** `material/navigation_bar.dart`, `navigation_rail.dart`, `navigation_drawer.dart`, `drawer.dart`, `drawer_header.dart`, `tabs.dart`, `tab_controller.dart`, `tab_indicator.dart`, `bottom_sheet.dart`, and their theme files.
**Tasks:** every widget in SPEC §11.2 "App structure" not done in 13.1, plus `ShowModalBottomSheet`. Switch the gallery's side list to a real `NavigationRail`. New page: "Navigation & tabs".
**Proof:** `13.2-gallery-navigation.png`.

### 13.3 Buttons
**Port from:** `material/button_style.dart`, `button_style_button.dart`, `elevated_button.dart`, `filled_button.dart`, `outlined_button.dart`, `text_button.dart`, `icon_button.dart`, `floating_action_button.dart`, `floating_action_button_location.dart`, `segmented_button.dart`, and their theme files.
**Tasks:** every button in SPEC §11.2 "Actions" before `MenuAnchor`, in enabled, disabled, hovered, focused and pressed states. New page: "Buttons", showing every variant and state.
**Proof:** `13.3-gallery-buttons.png`.

### 13.4 Menus
**Port from:** `material/menu_anchor.dart`, `menu_style.dart`, `menu_theme.dart`, `menu_bar_theme.dart`, `menu_button_theme.dart`, `popup_menu.dart`, `dropdown.dart`, `dropdown_menu.dart`, and their theme files.
**Tasks:** the rest of SPEC §11.2 "Actions", plus `ShowMenu`. Menus open inside the window's `Overlay` (SPEC §8.7). Include keyboard navigation and nested submenus. New page: "Menus", with a `MenuBar` (File/Edit/View, with shortcuts shown), a popup menu and dropdowns.
**Proof:** `13.4-gallery-menus.png`, with the File menu and one submenu open (`GLUTTER_DEV_OPEN_MENU=file/recent`).

### 13.5 Text fields
**Port from:** `material/input_decorator.dart`, `text_field.dart`, `text_form_field.dart`, `adaptive_text_selection_toolbar.dart`, `desktop_text_selection.dart`, `desktop_text_selection_toolbar.dart`, `desktop_text_selection_toolbar_button.dart`, `text_selection.dart`, `selection_area.dart`, `text_selection_theme.dart`.
**Tasks:** `InputDecoration`/`InputDecorator`, `TextField`, `TextFormField` (with `Form` validation), the desktop context menu on right-click, and the material `SelectionArea`. New page: "Text fields" (filled, outlined, with errors, prefix/suffix, multi-line, password).
**Proof:** `13.5-gallery-text-fields.png`, with one field focused and one showing a validation error.

### 13.6 Selection controls and chips
**Port from:** `material/checkbox.dart`, `radio.dart`, `radio_group` (in `widgets/radio_group.dart` or `material/radio.dart`, whichever has it), `switch.dart`, `slider.dart`, `range_slider.dart`, `slider_theme.dart`, `chip.dart`, `input_chip.dart`, `filter_chip.dart`, `choice_chip.dart`, `action_chip.dart`, `chip_theme.dart`, `circle_avatar.dart`.
**Tasks:** the rest of SPEC §11.2 "Inputs", except Autocomplete and Search. New page: "Selection & chips".
**Proof:** `13.6-gallery-selection.png`.

### 13.7 Search and autocomplete
**Port from:** `material/autocomplete.dart`, `search_anchor.dart`, `search_bar_theme.dart`, `search_view_theme.dart`.
**Tasks:** `Autocomplete`, `SearchAnchor`, `SearchBar`. New page: "Search", with a search bar filtering a list of countries and an autocomplete field.
**Proof:** `13.7-gallery-search.png`, with suggestions open (`GLUTTER_DEV_QUERY=gr`).

### 13.8 Dialogs and pickers
**Port from:** `material/dialog.dart`, `dialog_theme.dart`, `date_picker.dart`, `calendar_date_picker.dart`, `date_picker_theme.dart`, `time_picker.dart`, `time_picker_theme.dart`, `date.dart`, `input_date_picker_form_field.dart`.
**Tasks:** every widget and `Show*` function in SPEC §11.2 "Pickers and dialogs". `ShowDialog[T]` returns `<-chan T` (SPEC §12.6). New page: "Dialogs & pickers".
**Proof:** `13.8-gallery-date-picker.png`, with the date picker dialog open (`GLUTTER_DEV_OPEN=datepicker`).

### 13.9 Display, part 1
**Port from:** `material/card.dart`, `card_theme.dart`, `list_tile.dart`, `list_tile_theme.dart`, `checkbox_list_tile.dart`, `switch_list_tile.dart`, `radio_list_tile.dart`, `expansion_tile.dart`, `expansion_tile_theme.dart`, `expansion_panel.dart`, `divider.dart`, `divider_theme.dart`, `badge.dart`, `badge_theme.dart`, `tooltip.dart`, `tooltip_theme.dart`.
**Tasks:** those widgets. Tooltips show on hover after the Material delay. New page: "Lists & cards".
**Proof:** `13.9-gallery-lists.png`, with one tooltip visible (`GLUTTER_DEV_HOVER=tooltip-demo`).

### 13.10 Display, part 2
**Port from:** `material/data_table.dart`, `data_table_theme.dart`, `paginated_data_table.dart`, `progress_indicator.dart`, `progress_indicator_theme.dart`, `refresh_indicator.dart`, `scrollbar.dart`, `scrollbar_theme.dart`, `stepper.dart`, `carousel.dart`, `carousel_theme.dart`, `chip.dart` (the `Chip` display variant, if not done).
**Tasks:** the rest of SPEC §11.2 "Display". New page: "Data & progress".
**Proof:** `13.10-gallery-data.png`.

### 13.11 Gallery completion (light and dark)
**Tasks:**
1. The gallery covers **every** component in SPEC §11.2. Add a test that lists the widget constructors in `material` and fails if one isn't used on some page.
2. A home page with a component index.
3. Theme seed color picker; light/dark/system mode.
4. The window title "Glutter – Material 3 gallery".
**Proof:** B, `13.11-gallery-light.png` and `13.11-gallery-dark.png`: the home page in both modes (`GLUTTER_DEV_THEME=dark`).
**Example:** `examples/gallery`.

---

## M14: desktop windowing and Wayland quality

### 14.1 Window decorations
**Read:** SPEC §8.2 "Window decorations", §8.1 (the libdecor row), §8.11, D16, §22 (the libdecor risk).
**Files:** `internal/platform/linux/wayland/decor/` (libdecor binding), `widgets/window_decoration.go`.
**Tasks:**
1. Order of preference: server-side decorations → libdecor → `WindowDecoration` (SPEC §8.2). Add `WindowConfig.Decorations` with `DecorationsAuto` and `DecorationsApp`.
2. Bind `libdecor-0.so.0`: `libdecor_new` with an error callback, `libdecor_decorate`, `libdecor_frame_set_title`, `_set_app_id`, `_set_min_content_size`, `_set_max_content_size`, `_map`, `_commit`, `libdecor_configuration_get_content_size`, `_get_window_state`, `libdecor_state_new`/`_free`, `libdecor_dispatch` (the fd is integrated into the loop). The frame callbacks (configure, close, commit, dismiss popup) are created once (SPEC §8.1 rule 2).
3. `WindowDecoration` widget: title, buttons limited by `wm_capabilities`, double-click to maximize, right-click for the window menu, edge resize and move.
4. Measure memory and first-window time with and without libdecor's GTK plugin; record them in `IMPLEMENTED.md`.
**Display tests:** decorations mode chosen per compositor (sway → server-side).
**Proof:**
- C, `14.1-ssd-sway.png`: the counter example with sway's server-side title bar.
- C (user desktop), `14.1-libdecor-gnome.png`: ask the user to run `go run ./examples/counter` on GNOME and take a window screenshot (e.g. with the GNOME screenshot tool, "Window" mode) saved to that path. It must show a native GNOME (Adwaita) title bar.
- B, `14.1-app-decoration.png`: the counter with `GLUTTER_DECORATIONS=app` showing `WindowDecoration`.

### 14.2 Multiple windows
**Read:** SPEC §8.7 (all of it).
**Port from:** `widgets/_window.dart` and `widgets/_window_linux.dart` (the API shapes only; the Linux implementation there uses GTK, so Glutter's backend is its own), `widgets/view.dart` (`ViewCollection`).
**Tasks:**
1. `ViewCollection`; `WindowController` + `Window`; `DialogWindowController` + `DialogWindow` (parent, modal; `xdg_dialog_v1` on Wayland, `_NET_WM_WINDOW_TYPE_DIALOG` + `WM_TRANSIENT_FOR` on X11).
2. One shared EGL context, made current per surface.
3. The binding keeps one `RenderView` per window; focus and the keyboard follow the active window.
4. `examples/multiwindow`: a main window with "New window" and "Open dialog" buttons; each new window shows its own counter; the dialog is modal over its parent.
**Tests:** `gluttertest` with several views over the fake platform.
**Display tests:** create and close 3 windows; the dialog's parent relation is visible to the compositor.
**Proof:** C, `14.2-multiwindow.png`: headless sway showing the main window, a second window and the dialog. Run it with `GLUTTER_DEV_OPEN=window,dialog`, read by the example in debug builds.
**Example:** `examples/multiwindow`.

### 14.3 Outputs and scaling
**Read:** SPEC §8.11 "Resizing and scaling", §8.2 rows `zxdg_output_manager_v1`, `wl_output` v4.
**Tasks:**
1. Logical output geometry (`zxdg_output_v1` / `wl_output` v4 names), and `Displays()` with refresh rates.
2. Moving between monitors with different scales re-renders at the new scale with no blurry frame.
3. X11: per-monitor DPI from XRandR + `Xft.dpi`.
**Display tests:** sway headless with two outputs at scales 1 and 2; move the window and assert the buffer size changes in the next frame.

### 14.4 Cursors, complete
**Tasks:** every `SystemMouseCursors` value mapped to `wp_cursor_shape_v1` shapes and to XCursor names (with Flutter's GTK-embedder fallback names); cursor size and theme from the settings (11.1), scaled per output; X11 equivalents.
**Tests:** table test covering every cursor name.

### 14.5 Live system settings
**Read:** SPEC §8.8.
**Tasks:** changing dark mode, accent color, text scale or "reduce animations" in system settings updates the running app immediately (`MediaQuery`, `ThemeMode.system`); XSETTINGS fallback on X11; locale from `LC_ALL`/`LC_MESSAGES`/`LANG`.
**Tests:** a fake-bus signal changes `MediaQuery.PlatformBrightness`.
**Manual check (user):** ask the user to toggle dark mode in GNOME Settings while the gallery is running, and confirm it switches. Record the answer.

### 14.6 Drag and drop with other apps
**Read:** SPEC §3 (the "Data exchange" row), §8.11 "Data exchange".
**Port from:** `widgets/drag_target.dart` (`Draggable`, `LongPressDraggable`, `DragTarget`).
**Tasks:**
1. In-app drag and drop (`Draggable` / `DragTarget`).
2. Platform drag and drop:
   - Wayland `wl_data_device` as source and target: `start_drag` with an icon surface, the actions copy/move/ask, MIME negotiation
   - X11 XDND v5
3. A bridge so `DragTarget` accepts external drops (files as `text/uri-list`, text) and `Draggable` can export data.
4. `examples/multiwindow` gains "drag a file onto this window to see its path".
**Display tests:** drag between two windows of the test process.
**Manual check (user):** drag a file from GNOME Files or Dolphin onto the example, and drag text from the example into a text editor. Record the results.

### 14.7 The remaining Wayland quality items
**Read:** SPEC §8.11 (every item), §8.2 (the remaining rows), §17 (the Wayland compatibility matrix).
**Files:** `scripts/wayland-matrix.sh`, `docs/wayland-checklist.md`.
**Tasks:**
1. `XDG_ACTIVATION_TOKEN` consumed at startup; activation requests.
2. `desktop.KeepScreenOn(bc, on bool) error` through idle inhibit.
3. Tablet/stylus input (`zwp_tablet_v2` → stylus `PointerData`); `wp_content_type_v1`.
4. Window icon (`xdg_toplevel_icon_v1`).
5. `scripts/wayland-matrix.sh` runs the display tests and the examples (with screenshots into a temp directory) under every headless compositor that's installed (weston, sway, `mutter --headless`, `kwin_wayland --virtual`). It prints a pass/fail table and lists the compositors that were skipped because they're missing.
6. `docs/wayland-checklist.md` has one checkbox per §8.11 item, per compositor and GPU in the SPEC §17 matrix, for the user to fill in.
**Done when:** the standard checks pass; `scripts/wayland-matrix.sh` passes on every installed compositor (paste its table into `IMPLEMENTED.md`). Ask the user to fill in the checklist for their machine.

### 14.8 X11 parity
**Read:** SPEC §8.3 (every item).
**Tasks:** EWMH states (maximize, fullscreen, minimize), `_NET_WM_ICON`, `_MOTIF_WM_HINTS` for `DecorationsApp`, XI2 touch, XRandR hot-plug, selections with `INCR`, XDND (from 14.6), all checked under Xvfb.
**Display tests (X11):** each item above.
**Proof:** B under `scripts/headless-x11.sh`, `14.8-x11-gallery.png` (the gallery on X11).

---

## M15: D-Bus services

### 15.1 Portal plumbing and OpenURI
**Read:** SPEC §8.9 (the table rows "Open URI", "Settings"; the closing paragraph).
**Tasks:**
1. The parent-window handle for portals: `wayland:<handle>` through `zxdg_exporter_v2`, `x11:<window id hex>`.
2. `desktop.OpenURI(ctx, bc, uri) error` through `org.freedesktop.portal.OpenURI`, falling back to `xdg-open`, then `OpenURIUnavailable`.
**Tests:** fake-bus request/response; route tests.

### 15.2 File dialogs
**Tasks:** `desktop.OpenFiles`, `desktop.SaveFile`, `desktop.SaveFiles` through `org.freedesktop.portal.FileChooser`: filters, multiple selection, directory mode, current folder and name. Cancelling returns a result with `Cancelled: true`, not an error; a missing portal gives `FileChooserUnavailable`.
**Tests:** fake-bus scripted responses (selected, cancelled, unavailable).

### 15.3 Notifications
**Tasks:** `desktop.ShowNotification(ctx, bc, summary, opts) (id, error)` through `org.freedesktop.Notifications` (actions, icon, urgency, replaces-id, timeout). Action and close signals arrive as callbacks on the UI thread. Inside Flatpak (detected by `/.flatpak-info`), use the portal instead. `CloseNotification`.
**Tests:** fake-bus notify, action invoke and close signal.

### 15.4 System tray
**Read:** SPEC §8.9 (the tray row, and the API sketch).
**Tasks:**
1. Export a `StatusNotifierItem` object (properties, signals, `Activate`, `SecondaryActivate`, `Scroll`) and register it with `org.kde.StatusNotifierWatcher`. Export `com.canonical.dbusmenu` for the menu (layout, properties, events, `LayoutUpdated` on rebuild).
2. `desktop.NewSystemTray(icon, child, opts)`: a widget whose rebuilds update the tray (SPEC §8.9). The icon is rendered to ARGB32 pixmaps at 16, 22, 24, 32 and 48 px. No watcher gives `TrayUnavailable`, reported once through `ErrorReporter`.
**Tests:** fake bus: register, menu layout, a click on a menu item calls its callback on the UI thread, tooltip and icon updates.

### 15.5 `examples/tray`
**Tasks:** `examples/tray/main.go`:
- A window with a note field, plus a tray icon whose menu has "Show window", "New notification", "Open file…", a separator and "Quit".
- Closing the window hides it (the app keeps running in the tray).
- "New notification" shows a notification with an "Open" action that brings the window back.
- "Open file…" uses `desktop.OpenFiles` and shows the chosen path.
**Tests:** fake-bus integration tests for each menu item.
**Proof:** C (user desktop), `15.5-tray-menu.png`: ask the user to run `go run ./examples/tray` (on KDE, or GNOME with the AppIndicator extension), open the tray menu and save a screenshot to that path. Also C (user desktop), `15.5-notification.png`, with the notification visible.
**Example:** `examples/tray`.

---

## M16: Accessibility

### 16.1 AT-SPI2 bridge: tree, roles, actions
**Read:** SPEC §8.10.
**Tasks:**
1. Get the accessibility bus address (`org.a11y.Bus.GetAddress`) and connect.
2. Watch `org.a11y.Status.ScreenReaderEnabled`, and turn the framework's semantics on and off to match.
3. Export one AT-SPI2 object per semantics node, implementing `Accessible`, `Component` and `Action`, with roles and states mapped from `SemanticsFlag`/`SemanticsRole` (write the full mapping table in code, with comments).
4. Register the application with the registry.
**Tests:** fake bus: the tree for a small widget tree, `DoAction` triggers a tap, focus events.

### 16.2 Text, value, selection and events
**Tasks:** the `Text`, `EditableText`, `Value` and `Selection` interfaces; the events state-changed, text-changed, text-caret-moved, children-changed and visible-data-changed.
**Tests:** fake-bus tests for each interface.
**Manual check (user):** ask the user to run `examples/gallery` with Orca, navigate with Tab, and confirm that buttons, text fields and checkboxes are announced. Record the answer.

---

## M17: Effects and performance

### 17.1 Path boolean operations
**Read:** SPEC §7 (`PathCombine`).
**Tasks:** `ui.PathCombine(op, a, b)` for difference, intersect, union, xor and reverse difference. Use a robust polygon-clipping algorithm on flattened curves (document the choice), and keep curves where possible.
**Tests:** goldens for every operation on overlapping circles and stars; fuzz test for robustness (no panics, bounded output).

### 17.2 Raster cache and GPU efficiency
**Read:** SPEC §9.1 (the "Raster cache" row), §15.
**Tasks:** cache a `RepaintBoundary` subtree as a texture once it's unchanged for 3 frames, with eviction; `eglSwapBuffersWithDamageKHR` when available; checks for redundant state changes.
**Tests:** a frame counter proves that cached layers aren't re-rasterized; display tests for damage.

### 17.3 Debugging aids
**Read:** SPEC §14.3.
**Port from:** `rendering/debug.dart`, `widgets/debug.dart`, `rendering/performance_overlay.dart`, `widgets/performance_overlay.dart`.
**Tasks:** every flag in SPEC §14.3 as a `DebugOptions` field, `DebugDump*`, the performance overlay (`MaterialApp.ShowPerformanceOverlay`), and `runtime/trace` regions around the frame phases.
**Proof:** B, `17.3-debug-paint.png`: the layout example with `DebugPaintSizeEnabled` (`GLUTTER_DEV_DEBUG_PAINT=1`).

### 17.4 Benchmarks and the performance targets
**Read:** SPEC §15.
**Files:** `docs/PERFORMANCE.md`.
**Tasks:**
1. Benchmarks (`go test -bench`) for each SPEC §15 metric, in `rendering`, `widgets` and `internal/gpu/soft`, plus `scripts/perf.sh`, which measures idle CPU, startup time and memory of the counter and gallery examples on the current machine.
2. Profile and fix until the targets are met.
3. Record the machine and the numbers in `docs/PERFORMANCE.md`.
**Done when:** every target is met, or the user has approved the exceptions you list.

---

## M18: 1.0

### 18.1 API review
**Read:** SPEC §12 (all of it).
**Tasks:**
1. Add `cmd/glutter-gen apicheck`. For every exported constructor and type in the public packages, it checks:
   - the `NewX(required..., opts ...XOpts)` shape, with `child`/`children` last
   - the doc comment ends with a `Flutter:` reference or `Glutter-specific:`
   - the enum zero values are `Unspecified`
   - there's no exported mutable package variable
2. Run it, then fix the violations or list them for the user.
3. Update SPEC §12.10 with every deliberate deviation found.

### 18.2 Documentation and examples
**Tasks:**
1. A `README.md` with installation (only Go is needed), a quick start (the counter), the Flutter → Glutter translation table (link to SPEC §12), runtime requirements, and screenshots taken from `docs/implemented-proof-images/`.
2. Check that every example in SPEC §19 exists, runs, and has a proof image.
3. Write a `docs/GUIDE.md` "For Flutter developers" page with 10 side-by-side translations.

### 18.3 Release checklist
**Tasks:** confirm and record in `IMPLEMENTED.md`:
- the standard checks
- the Wayland matrix (14.7) and the checklist filled in by the user
- the Orca check (16.2)
- the performance targets (17.4)
- the API check (18.1)

Then **ask the user** before tagging `v1.0.0` and pushing the tag.
