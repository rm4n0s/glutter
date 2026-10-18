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
