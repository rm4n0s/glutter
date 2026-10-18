# Implemented steps

This is the log of finished steps from `docs/IMPLEMENTATION.md`, newest at the bottom. An agent adds an entry on the step's branch, as part of the step's commit (`docs/IMPLEMENTATION.md`, rules 10–12). The next step to do is the first step in `docs/IMPLEMENTATION.md` that has no entry here.

## Entry format

Copy this block, fill it in, and keep every heading. Write "none" where something doesn't apply.

```markdown
## <id> <step title>

- **Date:** YYYY-MM-DD
- **Branch:** feat/step-<id>-<slug>
- **Summary:** 2–4 sentences on what now exists and works.
- **Main files:** the packages or files created or changed.
- **Tests:** what was added; the number of Flutter test cases ported (and from which files).
- **Checks:** gofmt ✅ · go vet ✅ · golangci-lint ✅ · go test -race ✅ · CGO_ENABLED=0 build ✅ · CGO_ENABLED=0 test ✅ · display tests ✅/none
- **Proof:**
  ![<id> <name>](implemented-proof-images/<id>-<name>.png)
  One sentence describing what the image shows, as you verified it by viewing it.
- **Manual checks:** the user's answers to the step's manual checks, or "none".
- **Deviations and follow-ups:** differences from SPEC or Flutter (also recorded in SPEC §12.10), and TODOs left for later steps.
```

---

<!-- Entries start below this line. -->

## 0.1 Module and package layout

- **Date:** 2026-10-18
- **Branch:** feat/step-0.1-module-and-package-layout
- **Summary:** The Go module `github.com/rm4n0s/glutter` (go 1.27) exists with every package directory from SPEC §6.2, each with a `doc.go` stating its purpose and the Flutter library it mirrors. Flutter's LICENSE, a NOTICE covering every row of SPEC §20, the golangci-lint config and the scripts README are in place.
- **Main files:** `go.mod`, `NOTICE`, `.golangci.yml`, `third_party/flutter/LICENSE`, `scripts/README.md`, `*/doc.go` for the 14 public packages and the `internal/` packages, `cmd/glutter-gen/` (doc.go plus an empty `main`), `examples/doc.go`.
- **Tests:** none (no code yet). Checked by hand that revive flags an undocumented exported function.
- **Checks:** gofmt ✅ · go vet ✅ · golangci-lint ✅ · go test -race ✅ · CGO_ENABLED=0 build ✅ · CGO_ENABLED=0 test ✅ · display tests none
- **Proof:** none
- **Manual checks:** none
- **Deviations and follow-ups:** `cmd/glutter-gen/main.go` is an empty `main` so the package builds; step 0.3 fills it. `.golangci.yml` uses the golangci-lint v2 format and the `standard` default set plus `revive` (rules `exported`, `package-comments`); `errcheck`, `govet` and `staticcheck` are part of that set.

## 0.2 Reference sources

- **Date:** 2026-10-18
- **Branch:** feat/step-0.2-reference-sources
- **Summary:** `scripts/fetch-reference.sh` fetches shallow, blobless, sparse clones of Flutter (`stable`) and material-color-utilities into the git-ignored `reference/`, and records both commits in `reference/VERSION`. When `VERSION` exists, the script checks out exactly the recorded commits (it restores a deleted clone to the same commit). Pinned: Flutter 3.47.7 at `abaf9c52` (2026-10-08), material-color-utilities at `dd62982d` (2026-10-15).
- **Main files:** `scripts/fetch-reference.sh`, `scripts/README.md`.
- **Tests:** none (a shell script). Run by hand: first run, second run from `VERSION`, and a run after deleting `reference/flutter` (same commit restored). The script checks that every sparse path exists, and all of them did.
- **Checks:** gofmt ✅ · go vet ✅ · golangci-lint ✅ · go test -race ✅ · CGO_ENABLED=0 build ✅ · CGO_ENABLED=0 test ✅ · display tests none
- **Proof:** none
- **Manual checks:** none
- **Deviations and follow-ups:** the sparse checkout uses git's cone mode, which takes directories only and always includes root files, so `LICENSE` needs no entry. The `version` file named in the step does not exist in this checkout, so `flutter_version` comes from the release tag that points at the pinned commit (`git ls-remote --tags`). `shellcheck` is not installed, so the script was not linted.

## 0.3 Code generator skeleton

- **Date:** 2026-10-18
- **Branch:** feat/step-0.3-code-generator-skeleton
- **Summary:** `glutter-gen` dispatches on its first argument to the subcommands `wayland`, `keys`, `reexport`, `copywith` and `icons`; each returns the tagged error `NotImplemented`. A missing or unknown command gives `MissingCommand` or `UnknownCommand`. Errors are printed to stderr as `ToJson()` output and the exit code is 1. `github.com/rm4n0s/errors` is now in `go.mod`.
- **Main files:** `cmd/glutter-gen/main.go` (thin), `cmd/glutter-gen/run.go` (dispatch and error formatting), `cmd/glutter-gen/run_test.go`, `cmd/glutter-gen/internal/{wayland,keys,reexport,copywith,icons}/`.
- **Tests:** table test for the dispatch covering all seven cases (no command, unknown command, the five subcommands), each with a route assertion and the `command` metadata; a test that the JSON output replays through `errors.FromJson` with the same route; a test for non-tagged errors. No Flutter tests ported.
- **Checks:** gofmt ✅ · go vet ✅ · golangci-lint ✅ · go test -race ✅ · CGO_ENABLED=0 build ✅ · CGO_ENABLED=0 test ✅ · display tests none
- **Proof:** none (`go run ./cmd/glutter-gen wayland` prints the `NotImplemented` error and exits 1, as the step requires).
- **Manual checks:** none
- **Deviations and follow-ups:** a subcommand's route is `dispatch->Run.NotImplemented` for all five, because the errors package leaves package names out of routes; the `command` metadata tells them apart. `MissingCommand` is an addition to the step (no arguments needs some error). The subcommands' `// TODO(step X.Y)` comments point at 1.3 (wayland), 1.7 (keys), 12.2 (copywith), 12.3 (icons) and 12.4 (reexport). No `//go:generate` directives yet; they arrive with the steps that produce generated files (first 1.3).

## 1.1 Platform interfaces and the fake platform

- **Date:** 2026-10-18
- **Branch:** feat/step-1.1-platform-interfaces-and-fake
- **Summary:** `internal/platform` defines `Platform`, `Window`, `Handler`, `WindowConfig`, `WindowState`, `Display`, `Settings`, `ViewMetrics`, `Clipboard` and `Services` as sketched in SPEC §6.3. `ui` has `Offset`, `Size`, `Rect`, `ViewID`, `AppLifecycleState`, `PointerData` (all fields, plus its enums) and `KeyData` (plus its enums). `internal/platform/fake` is a scriptable platform: `Run` replays a FIFO queue of posted functions and injected events, `Post` works from any goroutine, `NewWindow` records configs and hands out view IDs, and an inspectable log records everything.
- **Main files:** `ui/{geometry,view,lifecycle,pointer,key,enum}.go`, `internal/gpu/surface.go`, `internal/platform/{platform,window,settings,placeholders}.go`, `internal/platform/fake/{fake,events,window}.go`, `docs/SPEC.md` (§7 note).
- **Tests:** `ui`: `KeyData.String` (7 cases), enum names and order, `PointerData.Respond` and `String`. `fake`: `Post` from 100 goroutines × 50 posts arrives in order per goroutine; `Run` stops on cancel (3 ways); `PlatformAlreadyRunning` route test; every event type reaches the handler, and posts and events interleave in queue order; close requests close or keep the window; window IDs, configs and recorded calls. Ported from Flutter: the `KeyData.toString` cases are written from `key.dart`; `reference/` has no engine tests for `dart:ui` data classes, so 0 Flutter test cases were ported.
- **Checks:** gofmt ✅ · go vet ✅ · golangci-lint ✅ · go test -race ✅ · CGO_ENABLED=0 build ✅ · CGO_ENABLED=0 test ✅ · display tests none
- **Proof:** none
- **Manual checks:** none
- **Deviations and follow-ups:**
  - Geometry uses `float64` (Dart's `double`); only `Matrix4` and `Color` are `float32`. SPEC §6.3's sketch has `dpr float32`, which should become `float64` at step 3.1. SPEC §7 now states the rule.
  - The step names `Offset`, `Size`, `Rect` as types only; their methods are left to 2.1.
  - Flutter's `PointerData` and `KeyData` have Dart defaults; here every enum starts with `<Enum>Unspecified` (SPEC §12.3), so `PointerData{}` has `Change == PointerChangeUnspecified` where Dart's default is `cancel`. Backends always set it. `KeyData.Character` is `""` where Dart has `null`.
  - `KeyData.String` copies Flutter's `toString`, which lacks its closing parenthesis.
  - The placeholder types the sketch calls `ui.MouseCursor`, `ui.TextInputState`, `ui.TextInputEvent` and `ui.ViewMetrics` live in `internal/platform` (`MouseCursor`, `TextInputState`, `TextInputEvent`, `ViewMetrics`), together with `DragData`, `DropEvent`, `Brightness`, `Clipboard` and `Services`, so the public `ui` package gets no stubs. Each carries a `TODO(step X.Y)` for the step that defines it.
  - `gpu.SurfaceHandle` is `any` until step 1.6. `WindowConfig.Resizable` and `.Decorated` are plain `bool`s with `DefaultWindowConfig()` giving `true`; step 5.1 turns them into `Opt[bool]`.
  - `Platform.Run` returns nil on cancel and is not reentrant. The first window gets view ID 0 (Flutter's implicit view).
  - `runtime.LockOSThread()` is not in this step; the plan assigns it to 1.4.

## 1.2 Library loader (purego)

- **Date:** 2026-10-18
- **Branch:** feat/step-1.2-library-loader
- **Summary:** `internal/platform/linux/dl` is the one way to load C libraries and call into them: `Open`, `OpenOverride`, `(*Lib).Bind` and `Has`, libc helpers (`Malloc`, `Calloc`, `Free`, `Strlen`, `CString`, `GoString`), the callback registry (`Callback`, `CallbackCount`, limit 1,500) and the layout test helper `AssertLayout`. purego v0.11.2 is pinned in `go.mod` (SPEC §22).
- **Main files:** `internal/platform/linux/dl/{doc,dl,libc,callback,layout}.go` and their tests, `go.mod`, `go.sum`.
- **Tests:** `Open` (5 cases, route test for `LibraryLoadFailed` with `names` and `dlerror` metadata); `OpenOverride` (4 cases: unset, existing absolute override, missing override with no fallback, relative override); `Bind` strlen called from Go, `Bind`/`Has` on a missing symbol (route test for `SymbolMissing`); C strings (empty, ASCII, Unicode, 100 kB); `Malloc`/`Calloc`; the callback counter and the `CallbackLimit` panic (route test); a real C→Go callback, sorting an array with libc's `qsort` and a Go comparator; `AssertLayout` (9 cases, including every kind of mismatch). No Flutter tests apply to this step.
- **Checks:** gofmt ✅ · go vet ✅ · golangci-lint ✅ · go test -race ✅ · CGO_ENABLED=0 build ✅ · CGO_ENABLED=0 test ✅ · display tests none
- **Proof:** none
- **Manual checks:** none
- **Deviations and follow-ups:**
  - Two tags beyond the step: `LibraryOverrideInvalid` (the override variable is set but not an absolute path) and `CStringAllocFailed` (a panic when `malloc` returns 0). Both are tested except `CStringAllocFailed`, which can't be triggered without exhausting memory.
  - A set override that fails to load is an error and does not fall back to the default names, because an explicit setting that is silently ignored is worse than an error.
  - libc is bound on first use (`sync.OnceValue`), not at import, and loading `libc.so.6` failing panics with `LibraryLoadFailed`. The `Callback` registry is the only package-level state; it only counts, because the purego limit is per process.
  - `AssertLayout` takes a `testing.TB` and additionally requires every named field of the struct to be listed in `offsets`, so a field added later can't escape the test. It is in a non-test file, as the step asks, so `testing` is linked into binaries that import `dl`.
  - Only `RTLD_NOW|RTLD_LOCAL` with glibc's `libc.so.6` is supported; musl is not.

## 1.3 Wayland protocol bindings generator

- **Date:** 2026-10-18
- **Branch:** feat/step-1.3-wayland-bindings-generator
- **Summary:** `glutter-gen wayland` reads the 18 protocol XML files (wayland 1.26.0, wayland-protocols 1.49, fetched by `scripts/fetch-wayland-protocols.sh`) and writes one `*_gen.go` per protocol into `internal/platform/linux/wayland/wl`: 73 interface types with typed request methods, a struct of typed event callbacks, enum types and constants, and the `wl_interface` tables built in C memory at init (about 8,500 generated lines). The hand-written runtime in the same package binds libwayland-client with purego, mirrors the C structs, installs one dispatcher for all proxies (user data is an integer handle, never a Go pointer) and provides `Conn` with the calls the event loop needs.
- **Main files:** `scripts/fetch-wayland-protocols.sh`, `internal/platform/linux/wayland/protocols/` (XML, licenses, `VERSIONS`), `cmd/glutter-gen/internal/wayland/{wayland,xml,gen,names}.go`, `internal/platform/linux/wayland/wl/{api,args,conn,doc,interface,proxy,structs}.go` and `*_gen.go`, `internal/platform/linux/dl` (new `Pointer` and `Bytes` helpers).
- **Tests:**
  - Generator: golden test on `testdata/small/small.xml` (requests and events with every argument kind, enums across interfaces, destructor, nullable, `since`, generic and typed `new_id`); determinism; a test that fails when the committed `*_gen.go` files differ from a fresh generation or lack the "Code generated … DO NOT EDIT." header; `goType` (15 cases) and naming (9 cases); 11 failure cases with route tests (`WaylandBadFlags`, `WaylandXMLReadFailed`, `WaylandXMLParseFailed`, `WaylandXMLInvalid`, `WaylandWriteFailed`); stale-file cleanup.
  - Runtime: layout tests for `wl_message`, `wl_interface`, `wl_argument` and `wl_array` (sizes and offsets from `wayland-util.h`, compiled with gcc); interface-table checks, including cross-protocol type references, `since` prefixes and nullable markers; argument and wl_fixed round trips.
  - End to end: the real libwayland-client against a hand-written fake compositor on a socket pair (it parses the wire format independently of the generated tables). It covers the registry and `bind`, `sync` callbacks, ints/uints/fixed/strings/objects/arrays in events, a file descriptor sent in a request and received in an event, a server-created object (`new_id` event) whose own events arrive in the same batch, destructors, requests on destroyed objects, handler panics re-raised outside C, protocol errors, a lost connection, and the prepare_read/read_events/dispatch_pending loop. Run 30 times under `-race` without a failure. No Flutter tests apply.
  - Cross-check (not committed): the interface tables of all 73 interfaces equal `wayland-scanner private-code` output (names, versions, signatures, type references).
- **Checks:** gofmt ✅ · go vet ✅ · golangci-lint ✅ · go test -race ✅ · CGO_ENABLED=0 build ✅ · CGO_ENABLED=0 test ✅ · display tests none (the end-to-end tests need `libwayland-client.so.0` but no display; they skip when it is missing)
- **Proof:** none
- **Manual checks:** none
- **Deviations and follow-ups:**
  - The `//go:generate` line is `go run ../../../../../cmd/glutter-gen wayland -protocols ../protocols -out .` because it sits in `wl/`; the step's relative path has the wrong depth.
  - Generated names: `Bind<Interface>(registry, name, version)` binds a global; the untyped `wl_registry.bind` is `BindRaw`. A `destroy`-type request keeps its XML name (`Destroy`, `Release`); every object also has `ProxyDestroy` for interfaces without one. An empty string for a nullable string argument is sent as NULL, and a NULL string in an event reads as `""`. The XML does not say what an array holds, so a table in the generator lists the six that hold `uint32` (`wl_keyboard.enter.keys`, `xdg_toplevel.configure.states` and `.wm_capabilities`, `zwp_tablet_pad_group_v2.buttons`, `zwp_text_input_v3.set_available_actions`); all others are `[]byte`.
  - Events of `wl_display` (`error`, `delete_id`) are handled inside libwayland and have no Go callbacks; errors surface through `Conn`.
  - `Conn` goes slightly beyond the step's function list, to make the layer testable and usable by 1.4: `ConnectFD`, `Roundtrip` and error mapping. New tags: `WaylandConnectFailed` (1.4 lists it too), `WaylandProtocolError` (interface, id, code), `WaylandConnectionLost`, `WaylandDispatcherFailed` and `WaylandObjectDestroyed` (panics, tested only for the latter). Step 1.4 still owns the event loop, the registry's required-globals check and `Run`.
  - `Flush` tells a full socket (EAGAIN) from a failure by asking libwayland whether it recorded an error, because purego does not expose `errno`.
  - The handle table behind the single dispatcher is the one piece of package-level mutable state in `wl`, like the callback counter in `dl`; both exist because the purego callback limit is per process.
  - `libwayland` logs protocol errors to stderr (the `wl_surface#N: error 2: bad scale` line in the protocol-error test output).

## 1.4 Wayland connection and event loop

- **Date:** 2026-10-18
- **Branch:** feat/step-1.4-wayland-connection-and-event-loop
- **Summary:** `internal/platform/linux` has the UI thread's event loop: epoll over sources, an eventfd for `Post` and a timerfd for timers (`syscall` only), and `internal/platform` pins the main goroutine to the main OS thread in an `init`. `internal/platform/linux/wayland` connects to a compositor, lists the registry, binds the six required globals at the SPEC §8.2 versions (capped at what the compositor offers), records the optional ones, answers `xdg_wm_base` pings, and runs the libwayland external-loop pattern (prepare_read, flush, wait, read_events, dispatch_pending). `scripts/headless-wayland.sh` runs a command inside headless sway (weston as fallback).
- **Main files:** `internal/platform/thread.go`, `internal/platform/linux/loop.go`, `internal/platform/linux/wayland/{conn,source,platform}.go`, `internal/platform/linux/wayland/fakecomp/` (the fake compositor, moved out of the `wl` tests so more packages can use it), `scripts/headless-wayland.sh`.
- **Tests:**
  - Loop: `Post` from 100 goroutines keeps order per goroutine; post before `Run` and from inside a posted function; four ways to stop; timers (deadline order, never early, cancel, nested, overdue); a source's readable, writable-interest and hang-up events; an idle loop does not spin; a source error ends `Run` and comes back unchanged; `RemoveSource`; syscall-failure route tests (`LoopSyscallFailed`).
  - Wayland, against the fake compositor (no display): required globals bound at `min(offered, ours)` including a second, older `wl_output`; optional globals recorded but not bound; each of the six required globals missing in turn gives `WaylandProtocolMissing` with the interface name and a route test; `WaylandConnectFailed`; ping answered with pong; a protocol error ends `Run` with `WaylandProtocolError` (interface, id, code); a hung-up socket ends it with `WaylandConnectionLost`; `Post` from 20 goroutines; `Run` stops on cancel; a second `Run` gives `PlatformAlreadyRunning`.
  - Display tests (`-tags glutter_display`, under `scripts/headless-wayland.sh`): connect to headless sway, list the globals, assert the required ones, `Run` and `Post` against the real compositor. They passed 15 times in a row. No Flutter tests apply.
- **Checks:** gofmt ✅ · go vet ✅ · golangci-lint ✅ · go test -race ✅ · CGO_ENABLED=0 build ✅ · CGO_ENABLED=0 test ✅ · display tests ✅ (headless sway; they also pass the lint and vet runs with `-tags glutter_display`)
- **Proof:** none
- **Manual checks:** none
- **Deviations and follow-ups:**
  - The loop is generic: a `Source` interface (`FD`, `WantsWrite`, `BeforeWait`, `AfterWait`) lets the Wayland connection do its prepare/read/cancel dance and lets X11 and D-Bus plug in later. The Wayland `Platform` implements `Run`, `Post` and `Close`; `NewWindow` returns `NotImplemented` until step 1.5, and `Displays`, `Clipboard` and `Settings` return empty values (TODOs name the steps).
  - Real sway offers `xdg_wm_base` only up to version 5, so the backend binds 5 there; the `suspended` state and `wm_capabilities` need 6 and are optional (SPEC §8.2).
  - Real sway has no `xdg_wm_dialog_v1` and no `xdg_toplevel_icon_manager_v1`; both are optional and get recorded as absent. It does offer 21 of the 23 optional globals the backend knows.
  - weston's headless backend has no `wl_seat`, so Glutter reports `WaylandProtocolMissing` there. The weston fallback of the script is therefore only useful for checks that don't start Glutter; this is documented in the script header. A seat is required by SPEC §8.2, so this is intended.
  - Timer tests use real time with coarse tolerances. The "fake clock" for key repeat (step 1.7) will need a clock the tests can control; the `Loop` has none yet.
  - `Loop.Close` and `wake` share the loop's mutex: `context.AfterFunc` can call `wake` after the loop was closed, and writing to a recycled descriptor would be a bug. The race detector found this.
  - `loop.go` is `//go:build linux`; `doc.go` of the package is not, so the package still lists on other systems.
  - `libwayland` prints protocol errors to stderr (`wl_shm#6: error 1: invalid format` in the test output).

## 1.5 First Wayland window (shared-memory buffer)

- **Date:** 2026-10-18
- **Branch:** feat/step-1.5-first-wayland-window
- **Summary:** `Platform.NewWindow` creates `wl_surface` → `xdg_surface` → `xdg_toplevel`, sets the title, app id, size limits, parent and state, and asks for server-side decorations (`zxdg_decoration_manager_v1`) when the compositor has them, all before the first commit. On each configure it draws an XRGB8888 `memfd`/`mmap` buffer filled with `#6750A4` at the new size, then sends `ack_configure`, attaches and commits; replaced buffers are freed when the compositor releases them. `xdg_toplevel.close` asks `Handler.OnCloseRequest`, and `Run` returns when the last window is closed. `scripts/screenshot-headless.sh` and the example `examples/dev/window` are in.
- **Main files:** `internal/platform/linux/wayland/{window,shm,memfd_amd64,memfd_arm64,memfd_other}.go`, `internal/platform/nop.go` (`NopHandler`), `internal/platform/linux/wayland/fakecomp/windowing.go`, `scripts/screenshot-headless.sh`, `scripts/headless-wayland.sh` (floating windows and a gray background by default), `examples/dev/window/main.go`.
- **Tests:**
  - Against the fake compositor, which now mimics the real ordering (a toplevel is configured only after its first commit; a new buffer releases the old one): the exact request order (title and app id and min/max size before the buffer-less first commit; pool, buffer, ack and the buffer commit after the configure); size from the config, the default 800×600, and the compositor's size; toplevel states (all nine decoded, a 0×0 configure keeps the size); resize draws the new buffer before the ack and frees the replaced one, with `OnMetrics`; decorations (server-side, client-side, protocol missing); close request agreed and refused; `SetTitle`/`SetState`/`SetSize`; two windows and `set_parent`; `StartDrag` `NotImplemented`; an oversized buffer ends `Run` with `ShmBufferFailed`; bad buffer sizes; `fill`; `decodeStates`. Stable over 25 runs under `-race`.
  - Display test under headless sway (`TestDisplayWindowMapsAndCloses`): the first configure arrives (640×400, activated), the window is mapped, and a close request ends `Run`. Passed 5 times in a row.
  - No Flutter tests apply.
- **Checks:** gofmt ✅ · go vet ✅ · golangci-lint ✅ · go test -race ✅ · CGO_ENABLED=0 build ✅ · CGO_ENABLED=0 test ✅ · display tests ✅ (headless sway)
- **Proof:**
  ![1.5 shm window](implemented-proof-images/1.5-shm-window.png)
  Headless sway's gray output with a floating window titled "Glutter dev window" (sway's server-side title bar) whose content is a solid purple 640×400 rectangle. Checked by pixel count too: exactly 256,000 pixels of `#6750A4` forming a 640×400 box.
- **Manual checks:** none
- **Deviations and follow-ups:**
  - `scripts/screenshot-headless.sh` needs `jq` besides `sway` and `grim`. It detects a mapped window as a sway tree node with a process id. The screenshot must be taken from a built binary, because the window has to appear within 10 seconds and `go run` compiles too slowly. A bug where the trap used an out-of-scope variable and never stopped the app was found by watching the app print an error at shutdown, and fixed.
  - `scripts/headless-wayland.sh` now makes windows float by default and paints the background `#3b3b3b` (`HEADLESS_FLOATING=0` tiles them). A tiled window would fill the whole output, hiding the 640×400 size.
  - `Run` returns when the last window closes, as the `Platform.Run` doc says; the engine (step 1.10) decides what that means for the app. Before `Run` starts, `NewWindow` only queues requests; the window is mapped once `Run` handles the first configure.
  - `OnMetrics` is called at the first configure and on every size change, with a device pixel ratio of 1 until step 3.7 (TODO in the code). `NativeSurface` returns nil until step 1.6; `SetCursor`, `SetTextInput` and `StartDrag` are stubs with TODOs for steps 1.8, 11.2 and 14.6. `Modal` and `Kind` are not used yet (they need `xdg_dialog_v1`, step 14.2).
  - `syscall` has no `SYS_MEMFD_CREATE`, so the number is defined per architecture (amd64 319, arm64 279; other architectures fail with `ENOSYS`).
  - The shm window draws only on configure; frame callbacks arrive with step 1.6.

## 1.6 EGL and GLES on Wayland, and self-capture

- **Date:** 2026-10-18
- **Branch:** feat/step-1.6-egl-gles-and-self-capture
- **Summary:** `internal/gpu/gles` loads `libEGL` and `libGLESv2` through purego (`GLUTTER_EGL_LIBRARY` / `GLUTTER_GLES_LIBRARY` override the paths), opens the display with `eglGetPlatformDisplay(EGL_PLATFORM_WAYLAND_KHR)`, picks an RGBA8 `EGL_OPENGL_ES3_BIT` window config, creates an ES 3 context and window surface, sets `eglSwapInterval(0)`, and offers `MakeCurrent`, `Viewport`, `Clear`, `SwapBuffers` and `ReadPixels` (rows flipped to top-first). The Wayland window creates a `wl_egl_window` through `libwayland-egl.so.1` and is paced by `wl_surface.frame` callbacks: at most one frame in flight, `Window.RequestFrame` merges requests, and a configure renders at once. `internal/engine` reads `GLUTTER_SCREENSHOT` / `GLUTTER_SCREENSHOT_AFTER` and saves the PNG. `examples/dev/window` now draws with EGL; the shm path stays behind `GLUTTER_DEV_SHM=1`.
- **Main files:** `internal/gpu/gles/{egl,gl,context}.go`, `internal/gpu/surface.go`, `internal/platform/linux/wayland/{egl,window,platform}.go`, `internal/platform/linux/wayland/fakecomp/{server,windowing}.go` (frame callbacks), `internal/engine/screenshot.go`, `examples/dev/window/main.go`.
- **Tests:**
  - `gles` against a fake EGL and GL: the success path and its call order, each failure with its route and the cleanup calls made (`EGLInitFailed`, `EGLNoConfig`, `GLES3Unsupported`, `EGLSurfaceFailed`, `EGLMakeCurrentFailed`, `EGLSwapFailed`, `GLReadPixelsFailed`); `EGLLoadFailed` through an override that points to a missing file (`Open->loadEGL->loadFailed.EGLLoadFailed`); `ReadPixels` flips rows.
  - Wayland EGL mode against the fake compositor (which now answers or holds frame callbacks): handshake order, a configure renders while a frame is in flight, one frame in flight and merged requests, close destroys the egl window before the surface, shm mode ignores `RequestFrame`, a second `Options` panics with `MultipleOptions`. Stable over 20 and then 15 runs under `-race`.
  - Screenshot: environment parsing (including bad and negative delays, route `ScreenshotAfterInvalid`), PNG round trip, write failures (`ScreenshotWriteFailed`).
  - Display test under headless sway (`TestDisplayClearsTheWindowToPurple`): context creation succeeds on Mesa (OpenGL ES 3.2), a frame is cleared to `#6750A4` and read back with every pixel exact.
  - No Flutter tests apply.
- **Checks:** gofmt ✅ · go vet ✅ · golangci-lint ✅ · go test -race ✅ · CGO_ENABLED=0 build ✅ · CGO_ENABLED=0 test ✅ · display tests ✅ (headless sway)
- **Proof:**
  ![1.6 EGL window](implemented-proof-images/1.6-egl-window.png)
  Self-capture of the EGL window (`GLUTTER_SCREENSHOT` under `scripts/headless-wayland.sh`): a 640×400 image that is evenly purple. Checked by pixel count too: all 256,000 pixels are exactly `#6750A4`. The delayed capture (`GLUTTER_SCREENSHOT_AFTER=300ms`) and the `GLUTTER_DEV_SHM=1` path were run as well.
- **Manual checks:** none
- **Deviations and follow-ups:**
  - Task 4 said to render before `ack_configure` + commit. With EGL the swap is the commit, so the backend resizes the `wl_egl_window`, sends `ack_configure`, then renders and swaps. The first commit after the ack has the new size, which is what SPEC §8.11 wants; the sentence in §8.11 now says so.
  - The screenshot reads the pixels before the swap, because the back buffer is undefined afterwards. With a delay, it requests a new frame after the delay and captures that one. Then the app ends `Run` and exits with code 0. A bug that saved the capture twice (the flag was never cleared) was found from the log and fixed.
  - `platform.Window` gained `RequestFrame()` (the engine needs it to ask for a frame when something changed). `gpu.SurfaceHandle` is now a struct (`Kind`, `Display`, `Window`) in place of the placeholder; the fake platform returns a `SurfaceKindFake` handle.
  - `wayland.New(name, ...Options)` takes `Options{ShmFill}` to keep the 1.5 shared-memory window for tests and `GLUTTER_DEV_SHM=1`; the shm tests use it.
  - `Platform.Close` now closes windows that are still open. The renderer must destroy its surface first (the example does, in `OnCloseRequest` and after `Run`).
  - Only the functions needed now are bound in `gl.go` (`GL_VERSION`, viewport, clear, read pixels). The rest comes with step 2.x.
  - Mesa prints `MESA-EGL: warning: failed to get driver name for fd -1` on the headless compositor (no DRM device); it is harmless.
  - `suspended` and occlusion handling (SPEC §8.11) are not done here: nothing drives frames yet. They belong to the engine (step 1.10).

## 1.7 Keyboard (xkbcommon) and key tables

- **Date:** 2026-10-18
- **Branch:** feat/step-1.7-keyboard-xkbcommon
- **Summary:** `internal/platform/linux/xkb` binds `libxkbcommon.so.0` through purego (context, keymaps from names, string and a compositor's file, state, `key_get_one_sym`/`utf8`, `key_get_layout`, `update_mask`, `mod_name_is_active`, `key_repeats`, `key_get_syms_by_level`, keysym to Unicode and lower case, compose table and state) and has `Layout`, which turns an evdev key into Flutter's physical key, logical key and character. `glutter-gen keys` reads Flutter's `physical_key_data.g.json`, `logical_key_data.g.json` and `layout_goals.json` and writes `internal/platform/linux/keymap_gen.go` (217 XKB keycodes to physical ids, 171 GTK keyvals to logical ids, the 49 layout goals); `go generate ./internal/platform/linux/` runs it. The Wayland backend binds `wl_keyboard` when the seat has one (and follows capability changes): it reads the keymap from the compositor's file, tracks modifiers and the layout group, gives focus on `enter`, repeats the held key on the loop's timer, and delivers `ui.KeyData` to `Handler.OnKey`.
- **Main files:** `internal/platform/linux/xkb/{lib,xkb,layout}.go`, `cmd/glutter-gen/internal/keys/keys.go`, `internal/platform/linux/{keymap.go,keymap_gen.go,doc.go}`, `internal/platform/linux/wayland/keyboard.go`, `internal/platform/linux/wayland/fakecomp/keyboard.go` (keyboard events for tests).
- **Tests:**
  - xkb, without a display, on keymaps made from the xkb data files (`us`, `de`, `ru`, `us,ru`): `a`; `Shift+a` (character `A`, logical key still `a`); Caps Lock; `Control+a` (character `a`); `Shift+1`; Enter, Escape, F1, Shift_L, the keypad with and without Num Lock; `de` swaps y and z; `AltGr+q` on `de` is `@`; the dead key `^` alone makes no text; `^` then `e` is `ê`; `^` then space is `^`; a cancelled sequence starts over; no compose table; Cyrillic keys keep the Latin logical key for letters (mandatory goals) and the layout's own for the comma key (not mandatory); `us,ru` in the second group; repeats per key (modifiers do not repeat); unknown keycodes go to the GTK plane; keymap round trip through string and through a file with a trailing NUL; `SymAt`; `KeyUTF8` follows modifiers; keysym helpers (8 cases); locale choice; the layout test of `xkb_rule_names`; route tests `XKBKeymapFailed` (four ways), `XKBComposeFailed`, `XKBLoadFailed` (a missing override and a relative one).
  - Generator: output on a small data set, a second run leaves the file alone, route tests for `KeysBadFlags`, `KeysDataMissing` (with the hint), `KeysDataInvalid` (two ways), `KeysDuplicate` (two ways), `KeysGoalUnknown` (two ways), `KeysWriteFailed`, and a staleness test of the committed tables against the Flutter data (skipped only when `reference/` is missing). The key table lookups have their own tests.
  - Wayland backend against the fake compositor with the real `de` and `us` keymaps, and a fake clock that the tests move by hand: press and release with modifiers (the key-up reuses the ids of the key-down), dead key and compose, keys without focus are ignored, held keys at `enter` are synthesized down and released in order at `leave`, closing the focused window releases the keys, duplicate downs and unknown ups are dropped, repeat timing (nothing at 399 ms, one at 400 ms, then every 40 ms, a pause is caught up, release stops it), another key ends the repeat, modifiers and rate 0 do not repeat, the focus loss stops it, the defaults (25 per second after 600 ms), a dead key repeats without feeding the compose state, a seat without a keyboard and plugging and unplugging one (`wl_keyboard.release` is sent), and a keymap that cannot be used leaves the keyboard silent. Stable over 30 runs with `-race`.
  - No Flutter tests apply.
- **Checks:** gofmt ✅ · go vet ✅ · golangci-lint ✅ · go test -race ✅ · CGO_ENABLED=0 build ✅ · CGO_ENABLED=0 test ✅ (the step lists no display tests; the display tests of the earlier steps still pass)
- **Proof:** none
- **Manual checks:** typing on a real keyboard has not been tried: the headless sway has no keyboard, and no tool to inject keys (`wtype`) is installed. It belongs with the manual checklist.
- **Deviations and follow-ups:**
  - The step says unknown keysyms become ids of the "unprintable" plane. Flutter's GTK embedder puts them in the GTK plane (`0x15_0000_0000 | keysym`), and the plane 1 ("unprintable") holds the keys in the table (Enter, Escape, F-keys). I followed the embedder. SPEC §8.5 now spells out the whole logical-key rule.
  - `reference/` has the framework and the data files but not the engine's `fl_keyboard_manager.cc` and `fl_key_embedder_responder.cc`. The layout-goal rule (first Latin character at level 0 in any layout, current group first; mandatory goals fall back to the US key) is rebuilt from `layout_goals.json`, the README of `gen_keycodes` and my reading of the embedder. It reproduces the documented behavior (shortcuts work on non-Latin layouts, Shift does not change the logical key), but it has not been compared line by line with the engine.
  - The character comes from the keysym (Control+a is `a`, Return is `\r`), as in the GTK embedder, not from `xkb_state_key_get_utf8`, which would give `\x01`. `State.KeyUTF8` is bound as the plan asks but the backend does not use it.
  - The synthesized events for lock modes and modifier bits (the GTK embedder's checked-key tables, `gtk_modifier_bit_mapping.json`, `gtk_lock_bit_mapping.json`) are not done here: only pressed keys are synchronized. TODO(step 8.x) when `HardwareKeyboard` exists.
  - Timestamps are the time the event reached the backend (`Platform.now`), not the compositor's millisecond `time`, so key and frame times share one clock.
  - A keymap that libxkbcommon rejects (or a keymap format other than `xkb_v1`) is logged and the keyboard is then silent; it does not end `Run`, because the app can still be used with the mouse. A locale with no compose table only turns dead keys and compose sequences off.
  - `wl_keyboard.key` with state `repeated` is ignored, since the repeats are made here (compositors do not send it).
  - The fake compositor now announces `wl_seat.capabilities` (keyboard by default), sends a valid one-key keymap by default, accepts `wl_keyboard.release`, and ignores EPIPE on a client that has gone. Its `get_keyboard` answers with an `enter` only if a surface exists.
  - libxkbcommon prints its errors to stderr (in the tests for bad keymaps). `go generate ./...` now needs `reference/` (run `scripts/fetch-reference.sh`); it fails with `KeysDataMissing` and says so otherwise.

## 1.8 Pointer, touch and cursors (Wayland)

- **Date:** 2026-10-18
- **Branch:** feat/step-1.8-pointer-touch-cursors
- **Summary:** The Wayland backend now binds `wl_pointer` and `wl_touch` when the seat has them (and follows capability changes) and sends `ui.PointerData` to `Handler.OnPointer`. A `pointerBatcher` collects enter, leave, motion, button, axis, axis_value120, axis_source, axis_stop and axis_relative_direction until `frame` and emits the batch: a wheel gives scroll signals, a touchpad a pan/zoom gesture, and `zwp_pointer_gestures_v1` pinch and swipe give pan/zoom gestures too. A `touchBatcher` turns `wl_touch` into touch pointers. Cursors: `Window.SetCursor` takes the new `platform.MouseCursor` (all 36 of Flutter's system cursors plus none) and uses `wp_cursor_shape_v1` when present, else the XCursor theme through `libwayland-cursor.so.0`. `examples/dev/input` logs every key and pointer event and changes the cursor between the left and right half.
- **Main files:** `internal/platform/linux/wayland/{pointer_batch,pointer,touch,cursor,cursor_lib,seat}.go`, `internal/platform/cursor.go` (`MouseCursor`, replaces the placeholder), `internal/platform/linux/cursor_names.go` (XCursor names, for X11 too), `internal/platform/linux/wayland/wl/foreign.go` (`Conn.WrapBuffer`), `internal/platform/linux/wayland/fakecomp/input.go`, `examples/dev/input/main.go`.
- **Tests:**
  - Batcher unit tests on recorded sequences (no display): enter, motion, buttons and leave with the exact events; a second button; leave with buttons down; entering while inside; button mapping (9 cases); events outside a window; scale 2; wheel scrolling (12 cases: one notch, back, half notch, several events, both axes, tilt, old clicks, pixels only, continuous, no source, a stop, no movement); a kinetic touchpad scroll with stop (start, updates, end, a new gesture); a lone stop and a stop in the last frame; pinch (pan, scale, rotation in radians); swipe; gestures ending one another and on leave; the axis direction; half-collected axes dropped on leave.
  - Touch: sequence with pressure and buttons, two fingers as two devices, cancel in id order, cancelling the fingers of one window, a repeated down, shape and orientation, scale.
  - Through the fake compositor with real libwayland: batches arrive at `frame`; without `frame` (version 4) each event is a batch; wheel and touchpad; pinch and swipe (the gesture objects are requested); touch down, move, up, cancel; events on a surface that is not a window are ignored; closing the window releases the button and cancels the finger; pointer and touch come and go with the seat (and `wl_pointer.release`, `wl_touch.release` are sent).
  - Cursors: shapes on enter and on `SetCursor` for ten cursors, none hides the cursor with and without shapes, the XCursor theme path with a generated theme (the old X names, hotspots, the cursor surface's buffer size), a missing cursor leaves the last one, route tests `CursorNotFound`, `CursorLibraryFailed` and `CursorThemeFailed`, `XCURSOR_SIZE` and `XCURSOR_THEME`, every cursor has a shape and names, the layout test of `wl_cursor` and `wl_cursor_image`. Stable over 15 runs with `-race`.
  - No Flutter tests apply.
- **Checks:** gofmt ✅ · go vet ✅ · golangci-lint ✅ · go test -race ✅ · CGO_ENABLED=0 build ✅ · CGO_ENABLED=0 test ✅ · display tests ✅ (the earlier steps' tests; this step lists none)
- **Proof:** none
- **Manual checks:** `examples/dev/input` was run for 4 seconds in the user's Wayland session (`wayland-0`) and under headless sway: it opens the window and logs the metrics (640×400). Nobody was at the keyboard or the mouse, and headless sway has no input devices, so typing, the pointer, the wheel, touchpad gestures, touch and the cursor change are **not** checked on a real device. Pending: run `go run ./examples/dev/input` and try them.
- **Deviations and follow-ups:**
  - The task text says `Window.SetCursor` takes `basic`, `click`, `text`, `grab`, `grabbing`, `resize*`, `wait`, `forbidden` and `none`; `platform.MouseCursor` has all of Flutter's `SystemMouseCursors` (36 plus none and the unspecified zero value), each with a shape and XCursor names.
  - The touchpad pan follows the fingers, so it is the negative of the axis values. I could not compare it with the engine's `fl_scrolling_manager` (not in `reference/`), so the sign rests on reading the protocol and Flutter's pan/zoom documentation; check it against a real touchpad in the manual run.
  - 53 logical pixels per wheel notch is Flutter's Linux embedder value (`kScrollOffsetMultiplier`) from memory, also not in `reference/`.
  - `axis_relative_direction` is collected (`pointerBatcher.inverted`) and unused: the axis values already follow the user's natural-scrolling setting.
  - Animated cursors (wait, progress) show their first image: TODO in `cursor_lib.go`. Device pixel ratio and buffer scale are 1 until step 3.7 (TODOs in the batchers and the cursors).
  - Cursor failures are logged (`slog`), never fatal: the fallback cursor path reports `CursorLibraryFailed`/`CursorThemeFailed` once, and `CursorNotFound` for each cursor the theme lacks.
  - The fake compositor now sends a pointer `enter` on `get_pointer` only if a surface exists, handles `get_touch`, the pointer, touch, cursor-shape and gesture requests, `wl_shm_pool.resize`, and has senders for every pointer, touch and gesture event. `WindowSurface()` returns the newest toplevel's surface, because `ID("wl_surface")` can be the cursor's.
  - `scripts/headless-wayland.sh` has no input devices, so an end-to-end test of real input needs `wtype`-like tooling (`sudo dnf install wtype`) or a virtual-keyboard client; not added.
