# CLAUDE.md

Guidance for Claude Code when working in this repository (Go, hosted on GitHub).

## Project
Glutter is a cgo-free GUI toolkit for Go (Android, from 1.6, is the one exception) that reproduces Flutter's framework API (widgets, rendering, gestures, animation, Material 3) on its own engine: a Wayland/X11 platform layer, a Vulkan renderer with an OpenGL ES 3.0 fallback (system libraries called through purego), and text layout on go-text. Linux desktop comes first; 1.1 adds the web (`GOOS=js GOARCH=wasm`, WebGPU with a WebGL 2 fallback, standard Go only, no TinyGo); 1.6 adds Android (`GOOS=android`, an APK built by `cmd/glutter-android`, a small Java embedding, Vulkan with a GLES fallback; SPEC §24); 1.8 adds Windows (`GOOS=windows`, Win32 through the standard library's `syscall`, Direct3D 12 at feature level 11_0; SPEC §25), tested under Wine. Every OS- and GPU-specific piece sits behind internal interfaces so Windows, macOS and mobile can be added later.

**`docs/SPEC.md` is the source of truth.** Read the sections that cover a task before starting it. If the code and the spec disagree, or a task needs a decision the spec doesn't make, ask instead of guessing. When a decision changes, update the spec on the same branch.

## Commands
- Build: `go build ./...`. Glutter is cgo-free; `CGO_ENABLED=0 go build ./...` must always work
- Test all: `go test -race ./...` (the race detector needs cgo, so this one needs a C compiler)
- Test the cgo-free path: `CGO_ENABLED=0 go test ./...`
- Test one package: `go test ./widgets -run TestName -v`
- Tests that need a real display/GPU: `go test -race -tags glutter_display ./...`
- Web build and checks (from step 19.1): `GOOS=js GOARCH=wasm go build ./...`, `GOOS=js GOARCH=wasm go vet ./...`, `scripts/test-wasm.sh` (unit tests under Node)
- Tests in a headless browser: `go test -tags glutter_browser ./...` (SPEC §17)
- Run an example on the web: `go run ./cmd/glutter-web serve ./examples/counter`
- Android check (from step 25.1): `scripts/check-android.sh` (build and vet for android/arm64 and android/amd64 with the NDK; needs `ANDROID_HOME`)
- Tests on the Android emulator (from step 25.2): `go test -p 1 -tags glutter_android ./...` (SPEC §24.13; they start and stop the emulator themselves)
- Run an example on Android (from step 25.2): `go run ./cmd/glutter-android run ./examples/counter` (a connected phone or a running emulator)
- Windows build and checks (from step 28.1): `GOOS=windows GOARCH=amd64 go build ./...`, `GOOS=windows GOARCH=amd64 go vet ./...`, `GOOS=windows GOARCH=arm64 go build ./...`, `scripts/test-windows.sh` (the unit tests of the packages that build for Windows, under Wine)
- Wine tests (from step 28.1): `GOOS=windows go test -tags glutter_wine -exec "$PWD/scripts/wine-exec.sh" -p 1 <packages>`; the host-side harness: `go test -tags glutter_wine -p 1 ./internal/winetest`
- Every example under Wine (from step 28.15): `go test -tags glutter_wine -p 1 -run TestEveryExampleDrawsItsFirstFrameUnderWine ./examples`; Windows performance: `scripts/perf-windows.sh -m` (Wine's numbers, vkd3d on lavapipe; one measurement at a time)
- Build a Windows `.exe` with its manifest and icon (from step 28.14): `go run ./cmd/glutter-windows build ./examples/counter`; run it under Wine: `go run ./cmd/glutter-windows run -headless ./examples/counter`; a proof screenshot: `go run ./cmd/glutter-windows shot -o out.png ./examples/counter`
- Run a Windows program under Wine, headless: `scripts/wine-exec.sh ./app.exe`; a screenshot of the X root: `scripts/wine-screenshot.sh out.png -- <command>`
- Update golden images: `go test ./... -run Golden -update-goldens`, then review every changed PNG
- Vet: `go vet ./...`
- Lint: `golangci-lint run`
- Format: `gofmt -w .` (or `goimports -w .`)
- Generate (Wayland protocol bindings, key tables, material re-exports, shaders): `go generate ./...`; the shaders need naga 30.0.1 (D20), and `glutter-gen shaders` also writes the HLSL, rewriting naga's sampler heap into plain registers. A shader never reads `instance_index` (SPEC §9.8)
- API conventions check (SPEC §12.13): `go run ./cmd/glutter-gen apicheck -allow cmd/glutter-gen/apicheck.allow`; a new violation is fixed, or accepted in that file with a reason and a row in SPEC §12.10
- Error rules check (SPEC §14.4): `go run ./cmd/glutter-gen errcheck -allow cmd/glutter-gen/errcheck.allow`; a new violation is fixed, or accepted in that file with a reason
- Pointer rule check (SPEC §25.2, from step 28.19): `go run ./cmd/glutter-gen uintptrcheck -allow cmd/glutter-gen/uintptrcheck.allow`: in the Windows code a Go pointer becomes a `uintptr` only in the arguments of a `//go:uintptrescapes` or `SyscallN` call (a stack move or the GC would make it stale); a new violation is fixed, or accepted in that file with a reason
- Run an example: `go run ./examples/counter`; SVG pictures: `go run ./examples/svg`
- After changing dependencies: `go mod tidy`

## Definition of done
Before reporting a task as finished, run `gofmt`, `go vet ./...`, `golangci-lint run` and `go test -race ./...`, and fix anything they report. Say so explicitly if any step was skipped or failed. Also:
- `CGO_ENABLED=0 go build ./...` and `CGO_ENABLED=0 go test ./...` pass (proves the library stays cgo-free)
- From step 24.1 on: `go run ./cmd/glutter-gen errcheck -allow cmd/glutter-gen/errcheck.allow` passes (the error rules of SPEC §14.4)
- From step 19.1 on: `GOOS=js GOARCH=wasm go build ./...`, `GOOS=js GOARCH=wasm go vet ./...` and `scripts/test-wasm.sh` pass
- From step 25.1 on: `scripts/check-android.sh` passes
- From step 28.1 on: `GOOS=windows GOARCH=amd64 go build ./...` and `go vet ./...`, `GOOS=windows GOARCH=arm64 go build ./...`, and `scripts/test-windows.sh` pass
- From step 28.19 on: `go run ./cmd/glutter-gen uintptrcheck -allow cmd/glutter-gen/uintptrcheck.allow` passes
- From step 28.15 on: `GOOS=windows golangci-lint run` (amd64 and arm64, with and without `--build-tags glutter_wine`) is clean
- Every new exported identifier has a doc comment that ends with its Flutter counterpart (see "Porting from Flutter")
- Every changed golden image has been looked at, not just regenerated

## Layout
- Public packages at the repo root mirror Flutter's libraries: `foundation`, `scheduler`, `animation`, `physics`, `gestures`, `painting`, `rendering`, `semantics`, `services`, `widgets`, `material`
- `ui/` is the engine API (Flutter's `dart:ui`): geometry, Canvas, Paint, Path, Scene, Paragraph, views, input events
- `desktop/` holds desktop services Flutter leaves to plugins: system tray, notifications, file dialogs, open-URI
- `gluttertest/` is the widget-testing package (Flutter's `flutter_test`)
- `internal/platform/` holds the platform interface and its backends (`linux/wayland`, `linux/x11`, `web`, `android`, `windows`, `fake`); `windows/win32` is the package that calls Windows (DLLs, COM, callbacks, the struct mirrors and their `layouttest`), and `windows/ole` holds the COM objects Go makes for OLE drag and drop and their plain-Go models; `windows/uia` is the UI Automation bridge (plain-Go model, COM providers, and the test helpers `callshim`, `semtree` and `uiatest`)
- `internal/gpu/` holds the renderer interface and its backends (`vulkan`, `gles`, `soft`, `webgpu`, `canvas2d`, `nativewindow`, `gdiwindow`, `d3d12`: the Direct3D 12 `explicit.Device` and its HLSL compiler), and `shadercache` keeps what they compile between runs; `explicit` is the renderer that `vulkan` and `webgpu` share, with its WGSL shaders
- `internal/winetest/` is the Wine harness; `.wine-test/` (git-ignored) holds its prefix
- `internal/text/` holds paragraph layout on go-text, with `androidfonts` and `winfonts` (the plain-Go matchers of the Android and Windows system fonts) and `fonttest` (test fonts made from the bundled fonts with another `cmap`); `internal/dbus/` holds the D-Bus services
- `internal/svg/` is the SVG parser and painter (SPEC §26, D27; plain Go, standard library only, no OS tags); `svg/` is the public package with `flutter_svg`'s widgets (`SvgPicture`, `Cache`, `SvgTheme`); `material` does not re-export it, and its doc comments end with `// flutter_svg: <file> <Name>`, which `apicheck` accepts. Doc comments in `internal/svg` cite `// SVG 1.1: <section>` or `// SVG 2: <section>` instead of "Flutter:". `reference/flutter_svg` (git-ignored, from `scripts/fetch-reference.sh`) is for the API names of package `svg` only
- `cmd/glutter-gen/` holds the code generators; `cmd/glutter-web/` builds, serves and screenshots apps for the web; `cmd/glutter-android/` builds, installs, runs and screenshots apps on Android, with `internal/androidtool` (packaging, the Java embedding and its checked-in `classes.dex`) and `internal/androidtest` (the emulator harness); `cmd/glutter-windows/` builds, runs, screenshots and measures (`perf`) apps for Windows, with `internal/wintool` (the manifest, the pure-Go `.syso` writer, the icon and version resources, the build in a temporary copy, the Wine launchers); `examples/<app>/` holds runnable examples (public API only), and `examples/dev/<app>/` holds early development examples that may use `internal/`; `scripts/` holds helper scripts
- `docs/IMPLEMENTATION.md` is the step-by-step plan; `docs/IMPLEMENTED.md` logs finished steps; `docs/implemented-proof-images/` holds each step's proof screenshots
- `reference/` (git-ignored) holds the pinned Flutter sources used for porting (SPEC §20). Never import, vendor or commit it
- Tests live next to the code (`foo_test.go`); golden images live in `testdata/goldens/` of their package

## Go conventions
- Never ignore a returned error; handle it or return it
- Pass `context.Context` as the first parameter of anything that does I/O (assets, images, fonts, D-Bus, file dialogs). Name it `ctx`. Name a `BuildContext` `bc`
- Tests are table-driven
- No global mutable state. In particular, no singletons like Flutter's `WidgetsBinding.instance`: code reaches the binding through `BuildContext` or constructors. Package-level read-only values (`Colors`, `Icons`, `Curves`) are fine and are never reassigned. The only process-wide mutable state is what Windows or Wine forces, each with its reason in the file: `ole.Shared` (COM callbacks are never freed) and the COM worker threads of `internal/platform/windows/worker_windows.go` (Wine never forgets an apartment's RPC endpoint)
- Prefer the standard library. Allowed dependencies: `github.com/rm4n0s/errors`, `github.com/go-text/typesetting`, `golang.org/x/image`, `github.com/godbus/dbus/v5` and `github.com/ebitengine/purego`. Ask before adding any other
- **No cgo**: never write `import "C"`, except in the `*_android.go` files of `internal/platform/android` (JNI and the NDK, D21; SPEC §24.2). C libraries are otherwise called only through purego, and only under `internal/platform/<os>/...`, `internal/gpu/gles`, `internal/gpu/vulkan` and `internal/gpu/nativewindow`. `GOOS=android` satisfies the `linux` tag: a file only for the Linux desktop says `linux && !android`. OS build tags are allowed only there, in `internal/dbus`, `internal/platform/web`, `internal/webtest` (the process group of the browsers it starts), `internal/gpu/webgpu`, `internal/gpu/canvas2d`, the `backends_<goos>.go` files and `gles_renderer.go` (`linux || js`) of `internal/engine`, `internal/androidtest` (the emulator's process group), `foundation/web_*.go` and `foundation/host_android.go` (SPEC §6.3). C memory, Go mirrors of C structs and function pointers never cross those package boundaries
- **Windows** is called only through the standard library's `syscall` (`SyscallN`, `NewCallback`), never purego and never `golang.org/x/sys/windows` (not an allowed dependency), and only in `internal/platform/windows/...`, `internal/gpu/d3d12`, `internal/gpu/gdiwindow` and the Windows files of `internal/engine`. The `windows` build tag is allowed there, in `internal/winetest` (with `linux` for the side that starts Wine), `internal/engine/backends_windows.go`, `foundation/host_windows.go` and `cmd/glutter-windows`. `GOOS=windows` satisfies neither `linux` nor `unix`. The Go mirrors of Win32 and COM structures live in files with no OS tag and each has a `layouttest` against the mingw-w64 headers (skipped without `mingw64-gcc`). Callbacks are made once, as with purego (SPEC §6.3 rules 10–11, §25.2). Go assembly is allowed only for the COM thunks that take floating-point arguments, in `internal/platform/windows/uia` and `uia/callshim` (`*_windows_amd64.s`, `*_windows_arm64.s`; SPEC §25.2). Under Wine, a UI Automation client never subscribes to `Text_TextChanged` or `LiveRegionChanged`: Wine 11 then blocks the thread that raises the event
- `syscall/js` is imported only in `internal/platform/web`, `internal/gpu/webgpu`, `internal/gpu/canvas2d` and the WebGL 2 files of `internal/gpu/gles`. Once the event listeners are installed, the UI goroutine never waits for a JavaScript promise (SPEC §23.3)
- purego rules (SPEC §8.1):
  - never call a C variadic function; use the non-variadic variant (e.g. `wl_proxy_marshal_array_flags`)
  - create C→Go callbacks once at startup (a process can have ~2,000) and dispatch by user data
  - every Go mirror of a C struct has a layout test
  - data C keeps lives in C memory
- No `runtime.GOOS` checks in the framework. Platform-dependent behavior goes through `foundation.TargetPlatform`, like Flutter's `defaultTargetPlatform`
- Only the UI thread touches the framework, GL or the display connection. Other goroutines hand work over through `SetState` or the binding's `Post` (SPEC §13)

## Flutter API conventions (summary of SPEC §12)
- Every widget and public class is created by a constructor `NewX(required..., opts ...XOpts) *X`
  - Flutter's `required` parameters become positional arguments, so the compiler enforces them
  - Optional parameters go in a trailing options struct; passing more than one options value panics
- Positional order follows Flutter's declaration order, except `child`/`children` always come last. A required-but-nullable parameter (e.g. `onPressed`) is still positional, and `nil` means what `null` means in Flutter
- Named constructors become `NewXName` (`ListView.builder` → `NewListViewBuilder`); static methods become `XName` functions (`Theme.of` → `ThemeOf`, `Navigator.push` → `NavigatorPush`)
- Names, defaults and semantics follow Flutter. In options structs, a nullable Flutter parameter whose `null` differs from Go's zero value becomes `foundation.Opt[T]` (`m.Some(v)`). An enum's zero value is `<Enum>Unspecified`, meaning "the widget's default"
- `material` re-exports everything in `widgets` through generated wrappers, so apps import one package

## Porting from Flutter
- Port behavior and algorithms from `reference/flutter`. Keep Flutter's names (in Go case), defaults, and the order of layout and paint operations
- End each exported identifier's doc comment with its source, e.g. `// Flutter: widgets/basic.dart Padding`
- A deliberate difference from Flutter goes in a `// Deviation:` comment and in SPEC §12.10
- Ported code is a derivative of Flutter (BSD-3-Clause). Keep `third_party/flutter/LICENSE` and the NOTICE entry (SPEC §20) intact. The UI Automation code ported from Chromium's accessibility code (via Flutter) is under Chromium's BSD-3-Clause license: keep `third_party/chromium/LICENSE` and its NOTICE entry intact

## Error handling
Use `github.com/rm4n0s/errors` (v0.2.0) for all errors, the way its `AI_GUIDE.md` says. This is a requirement of the project owner: read the whole guide (it is in the module cache, `$(go env GOMODCACHE)/github.com/rm4n0s/errors@v0.2.0/AI_GUIDE.md`) before you write or change code that makes, returns, logs or tests an error, and again when the dependency version changes. SPEC §14.4 lists the rules with the reasons, and `go run ./cmd/glutter-gen errcheck -allow cmd/glutter-gen/errcheck.allow` checks them. Never use `fmt.Errorf`, and never import the standard library `errors` package under the name `errors`: this package re-exports `errors.Is`, `errors.As` and `errors.Unwrap`, so a single import covers everything (if `Join` is needed, import the standard one as `stdErrors "errors"`).

**Creating errors**
- New failure with a fixed message: `errors.New("Tag", "message", "key", value, ...)`. It always has a tag and a message
  - Tag: short PascalCase name of the thing the function cannot do, e.g. `WaylandConnectFailed`. Declare it as a constant next to the code that produces it (`const tagWaylandConnectFailed = "WaylandConnectFailed"` at the end of the file; exported as `TagX`, with a doc comment, when another package matches on it)
  - Trailing arguments are key/value metadata pairs with string keys, e.g. `"display", name`. Give all of them in this one call; never chain `SetMetadata` after `errors.New`
- New failure with a formatted message: `errors.NewErrf("Tag", "format %s", arg)`
- Wrapping an error that comes from outside this module (stdlib, C library error codes, godbus, go-text, any third-party package): `errors.NewErr("Tag", err)`, once, where it first appears. `NewErr` and `NewErrf` take no metadata: add it with one `.SetMetadata("key", value, "key2", value2)` call. `SetMetadata` merges, so a later call in another function adds keys; `RemoveMetadata` and `SetMessage` change an error that is already made. `NewErr(tag, nil)` panics, so call it only inside `if err != nil`
- Call the constructor in the function that fails, never in a helper that builds and returns the error: the route is the call stack at the constructor, and a helper adds a frame that is not the failing function. (The one exception is a table that turns a C error code into a tag with a switch, such as `vkError`, listed in `errcheck.allow`; an exported function or method is not a helper)
- Every `New`, `NewErr` and `NewErrf` call captures the stack at that point
- A sentinel for `errors.Is` is a struct literal, `&errors.Error{Tag: TagX}`. Never write `var ErrX = errors.New(...)` at package level: its route is `init`

**Propagating errors**
- Every error is tagged exactly once. If the error came from a function of this module, return it untouched: `return err`. Re-wrapping it (`NewErr`, `NewErrf` with `%w`, `fmt.Errorf`) would replace the Tag and the stack and break route tests; the route already shows who called whom, so no context is added by wrapping
- Functions return the plain `error` interface, never `*errors.Error` (avoids the non-nil-interface-holding-nil-pointer bug)
- `errors.FromError(err)` is a plain type assertion. It fails if something else wrapped the error, so don't wrap these errors with anything

**Inspecting errors**
- `appErr, ok := errors.FromError(err)`, then use `appErr.Tag`, `appErr.Metadata`, `appErr.Message`
- Branch on the tag constant (`appErr.Tag == pkg.TagX`) or on `errors.Is(err, sentinel)` (it matches by Tag), never on a string literal, `Message` or `Error()`
- Use `errors.Is` / `errors.As` / `errors.Unwrap` for the underlying cause stored in `OriginalErr`

**Logging errors**
- Log with `slog` and pass the error as the attribute `"err"`: `logger.Error("what failed", "err", err)`. `*errors.Error` is an `slog.LogValuer`: the entry has `Tag`, `Message`, `Route`, `OriginalErr`, `Metadata` and `StackFrames`, and a test replays it with `json.Unmarshal` into `errors.ErrorJson` and `errors.FromJson`. Do not call `ToJson` or `json.Marshal` on an error to log it
- Metadata values must be JSON-serializable (strings, numbers, bools, simple structs); after a round trip numbers are `float64`

**Testing errors**
- Every new failure case gets its own unique Tag and a test that asserts the route, not just that an error occurred:
  ```go
  appErr, ok := errors.FromError(err)
  if !ok {
      t.Fatalf("expected *errors.Error, got %T", err)
  }
  if !appErr.HasRoute("Engine.Start->Wayland.Connect.WaylandConnectFailed") {
      t.Errorf("unexpected failure path: %s", appErr.Route())
  }
  ```
- A route lists the call chain from outermost to innermost (`Type.Method` or `Func`, joined by `->`, no package names) and ends with the Tag. `Route` and `HasRoute` stop at the function that called them, so in a test the route starts just below the test function. `HasRoute` matches by substring: give the path from the entry function you call down to `.Tag`, so the tag is checked too. Print `appErr.Route()` when an assertion fails
- Panics in user code and programmer errors follow SPEC §14

## Git and GitHub
- Never commit or push directly to `main`; work on a branch named `feat/...` or `fix/...`
- **Exception:** if `docs/IMPLEMENTATION.md` exists, follow its Rules section for the steps it lists. Each step is done on its own branch, one at a time, and a finished step (all four Definition-of-done checks passing) is merged into the local `main` by a fast-forward merge, without a PR, after its entry is added to `docs/IMPLEMENTED.md` on that branch. **Only a finished milestone is published to GitHub** (the owner's rule, 2026-10-27): after the last step of a milestone is merged, `main` is pushed with the step tags of the milestone and its release tag, all at once (`git push --follow-tags origin main`). Never commit work for a step directly on `main`. Everything not covered by that file (other changes, fixes outside the listed steps) still follows the rule above
- **Tags:** after a finished step is merged into the local `main`, tag its commit with an annotated tag `step-<id>` (e.g. `step-27.7`); the step counts as finished when the tag is on the local `main`. The tags go to `origin` with the milestone. Never move or delete a tag. **When a version is finished** (its release step, or a fix step that closes a patch version such as 1.8.2), make its annotated release tag (`v1.8.2` and the like) on that commit and publish it with the milestone, without asking (the owner's rule, 2026-10-26)
- Commit messages: imperative mood, e.g. "Add retry to client"
- PRs: short description, link the issue (`Closes #123`), keep them small
- No `.github/` directory and no GitHub Actions workflows: never add them. The four Definition-of-done checks are run locally

## Working through the steps
- Never stop to ask for confirmation, and never end a turn with a question such as "Should I start the next step?", until every step in `docs/IMPLEMENTATION.md` is finished (through the last one). After a step is finished (merged and tagged locally), report it briefly and start the next step in the same turn
- A request such as "publish to main" that is already satisfied gets a one-line answer, then work continues with the next step
- **Manager and workers** (the owner's choice, 2026-10-25): the main session (Opus 5.5, high effort) manages and Sonnet 5.5 agents at high effort write the code (Agent tool with `model: "sonnet"`, `effort: "high"`). The manager reads the step and the SPEC, writes the brief, launches and steers the workers, reviews their diffs, runs the checks, and does the git work; it writes only small fixes itself. The details are in `docs/IMPLEMENTATION.md`, Rules, "Who does the work"

## Don't
- Don't edit generated files (`*_gen.go`); regenerate them with `go generate ./...`
- Don't commit secrets or `.env` files
- Don't leave a `.syso` file or an `_glutterwin-*` directory in the tree (`glutter-windows build` makes and removes them)
- Don't let a Linux, Wayland, X11, GL, Vulkan, WebGPU, JavaScript, JNI, Android NDK, Win32, COM or Direct3D type appear in a public package or in `internal/platform`'s or `internal/gpu`'s interfaces
- Don't leave processes running: every compositor, browser, Wine instance (wineserver, explorer.exe, winedevice; Wine runs only on a headless Xvfb, with `ulimit -c 1`, its prefix in `.wine-test/prefix`, stopped with `wineserver -k`, never `-w`), Android emulator, `adb` server or app started for a test, a proof or a check is stopped when it is no longer needed (run it under `timeout -k`, in its own process group), and `pgrep -a` shows nothing left before a task is reported done
- Never look for, list, open or read files outside this project folder (no home-directory databases, configs, logs or other projects). If something outside it is needed, ask the user to provide it. Exceptions: dependency source in the Go module cache, system C headers (`/usr/include`) when writing or checking Go mirrors of C structs, and the Android SDK and NDK under `$ANDROID_HOME` (the owner asked for them to be used on 2026-10-23). The emulator's virtual devices and the test `adb` server live in `.android-test/` at the module root, never in `~/.android`

## Testing notes
- Widget tests use `gluttertest` (`PumpWidget`, `Pump`, `Tap`, finders) with a fake clock and the fake platform; no display needed
- Golden tests render through the software renderer (`internal/gpu/soft`), so they are deterministic and need no GPU
- Tests that open real windows or GL contexts carry the `glutter_display` build tag and are skipped by the default test run. Vulkan display tests use Mesa's lavapipe with validation on (SPEC §17)
- Tests that run in a headless browser carry the `glutter_browser` build tag; they use `internal/webtest` and temporary browser profiles
- Tests that run on the Android emulator carry the `glutter_android` build tag; they use `internal/androidtest`. What only a real phone shows is a manual check (SPEC §24.13)
- Tests that run inside Wine carry the `glutter_wine` build tag; `scripts/test-windows.sh` runs the unit tests of the packages that build for Windows under Wine
- A test that opens Windows windows keeps one OS thread from `New` to `Close` (`runtime.LockOSThread`). The Wine prefix gets copies of the host's Liberation Sans in `C:\windows\Fonts` (`scripts/wine-exec.sh`); some Wine tests are host-side (`go test -tags glutter_wine -p 1 ./internal/winetest ./cmd/glutter-windows ./internal/platform/windows ./examples/counter ./examples/scrolling ./examples/editor`): they start Wine themselves. The Wine clipboard tests start the test binary a second time as the other program (`TestWineClipboardHelper`); the settings tests write into the private prefix's registry and remove what they wrote. A pixel test never ends on a blinking caret (end the script with a selection), and a scripted key that works on the laid-out value waits for a frame after the edit before it. Under Wine `auto` draws with `d3d12`: a Wine test that compares frames pixel for pixel runs with `GLUTTER_RENDERER=soft` or `GLUTTER_RASTER_CACHE=0`, and one that compares `d3d12` with `soft` uses `gputest.Tolerance`. The tray Wine tests run in a child process on Wine's `explorer /desktop=glutter,800x600` (`TestWineShellServices`): threads of one process must not be on two Wine desktops. The `internal/gpu/d3d12` Wine suite takes 3 to 4 minutes and about 3 GB (vkd3d takes about 120 MB for each shader-visible descriptor heap): run it alone
- Every error Tag has a route test
