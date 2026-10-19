# CLAUDE.md

Guidance for Claude Code when working in this repository (Go, hosted on GitHub).

## Project
Glutter is a cgo-free GUI toolkit for Go that reproduces Flutter's framework API (widgets, rendering, gestures, animation, Material 3) on its own engine: a Wayland/X11 platform layer and an OpenGL ES 3.0 renderer (system libraries called through purego), and text layout on go-text. Linux desktop comes first; every OS- and GPU-specific piece sits behind internal interfaces so Windows, macOS, web and mobile can be added later.

**`docs/SPEC.md` is the source of truth.** Read the sections that cover a task before starting it. If the code and the spec disagree, or a task needs a decision the spec doesn't make, ask instead of guessing. When a decision changes, update the spec on the same branch.

## Commands
- Build: `go build ./...`. Glutter is cgo-free; `CGO_ENABLED=0 go build ./...` must always work
- Test all: `go test -race ./...` (the race detector needs cgo, so this one needs a C compiler)
- Test the cgo-free path: `CGO_ENABLED=0 go test ./...`
- Test one package: `go test ./widgets -run TestName -v`
- Tests that need a real display/GPU: `go test -race -tags glutter_display ./...`
- Update golden images: `go test ./... -run Golden -update-goldens`, then review every changed PNG
- Vet: `go vet ./...`
- Lint: `golangci-lint run`
- Format: `gofmt -w .` (or `goimports -w .`)
- Generate (Wayland protocol bindings, key tables, material re-exports): `go generate ./...`
- Run an example: `go run ./examples/counter`
- After changing dependencies: `go mod tidy`

## Definition of done
Before reporting a task as finished, run `gofmt`, `go vet ./...`, `golangci-lint run` and `go test -race ./...`, and fix anything they report. Say so explicitly if any step was skipped or failed. Also:
- `CGO_ENABLED=0 go build ./...` and `CGO_ENABLED=0 go test ./...` pass (proves the library stays cgo-free)
- Every new exported identifier has a doc comment that ends with its Flutter counterpart (see "Porting from Flutter")
- Every changed golden image has been looked at, not just regenerated

## Layout
- Public packages at the repo root mirror Flutter's libraries: `foundation`, `scheduler`, `animation`, `physics`, `gestures`, `painting`, `rendering`, `semantics`, `services`, `widgets`, `material`
- `ui/` is the engine API (Flutter's `dart:ui`): geometry, Canvas, Paint, Path, Scene, Paragraph, views, input events
- `desktop/` holds desktop services Flutter leaves to plugins: system tray, notifications, file dialogs, open-URI
- `gluttertest/` is the widget-testing package (Flutter's `flutter_test`)
- `internal/platform/` holds the platform interface and its backends (`linux/wayland`, `linux/x11`, `fake`)
- `internal/gpu/` holds the renderer interface and its backends (`gles`, `soft`)
- `internal/text/` holds paragraph layout on go-text; `internal/dbus/` holds the D-Bus services
- `cmd/glutter-gen/` holds the code generators; `examples/<app>/` holds runnable examples (public API only), and `examples/dev/<app>/` holds early development examples that may use `internal/`; `scripts/` holds helper scripts
- `docs/IMPLEMENTATION.md` is the step-by-step plan; `docs/IMPLEMENTED.md` logs finished steps; `docs/implemented-proof-images/` holds each step's proof screenshots
- `reference/` (git-ignored) holds the pinned Flutter sources used for porting (SPEC §20). Never import, vendor or commit it
- Tests live next to the code (`foo_test.go`); golden images live in `testdata/goldens/` of their package

## Go conventions
- Never ignore a returned error; handle it or return it
- Pass `context.Context` as the first parameter of anything that does I/O (assets, images, fonts, D-Bus, file dialogs). Name it `ctx`. Name a `BuildContext` `bc`
- Tests are table-driven
- No global mutable state. In particular, no singletons like Flutter's `WidgetsBinding.instance`: code reaches the binding through `BuildContext` or constructors. Package-level read-only values (`Colors`, `Icons`, `Curves`) are fine and are never reassigned
- Prefer the standard library. Allowed dependencies: `github.com/rm4n0s/errors`, `github.com/go-text/typesetting`, `golang.org/x/image`, `github.com/godbus/dbus/v5` and `github.com/ebitengine/purego`. Ask before adding any other
- **No cgo**: never write `import "C"`. C libraries are called only through purego, and only under `internal/platform/<os>/...` and `internal/gpu/gles`. OS build tags are allowed only there and in `internal/dbus`. C memory, Go mirrors of C structs and function pointers never cross those package boundaries
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
- Ported code is a derivative of Flutter (BSD-3-Clause). Keep `third_party/flutter/LICENSE` and the NOTICE entry (SPEC §20) intact

## Error handling
Use `github.com/rm4n0s/errors` for all errors. Never use `fmt.Errorf`, and never import the standard library `errors` package: this package re-exports `errors.Is`, `errors.As` and `errors.Unwrap`, so a single import covers everything.

**Creating errors**
- New failure with a fixed message: `errors.New("Tag", "message", "key", value, ...)`
  - Tag: short PascalCase name of the failure, e.g. `WaylandConnectFailed`
  - Trailing arguments are key/value metadata pairs, e.g. `"display", name`
- New failure with a formatted message: `errors.NewErrf("Tag", "format %s", arg)`
- Wrapping an error that is NOT already an `*errors.Error` (stdlib, C library error codes, third-party): `errors.NewErr("Tag", err)`. Add metadata by chaining `.SetMetadata("key", value)`
- Every `New`, `NewErr` and `NewErrf` call captures the stack at that point

**Propagating errors**
- If the error already came from this package, return it untouched: `return err`. Re-wrapping it would replace the Tag and the stack and break route tests
- Functions return the plain `error` interface, never `*errors.Error` (avoids the non-nil-interface-holding-nil-pointer bug)
- `errors.FromError(err)` is a plain type assertion. It fails if something else wrapped the error, so don't wrap these errors with anything

**Inspecting errors**
- `appErr, ok := errors.FromError(err)`, then use `appErr.Tag`, `appErr.Metadata`, `appErr.Message`
- Use `errors.Is` / `errors.As` / `errors.Unwrap` for the underlying cause stored in `OriginalErr`

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
- A route lists the call chain from outermost to innermost (`Type.Method` or `Func`, joined by `->`) and ends with the Tag. `HasRoute` matches by substring
- Log errors with `appErr.ToJson()` so they can be replayed in a test via `errors.FromJson`
- Panics in user code and programmer errors follow SPEC §14

## Git and GitHub
- Never commit or push directly to `main`; work on a branch named `feat/...` or `fix/...`
- **Exception:** if `docs/IMPLEMENTATION.md` exists, follow its Rules section for the steps it lists. Each step is done on its own branch, one at a time, and a finished step (all four Definition-of-done checks passing) is published to `main` by a fast-forward merge and push, without a PR, after its entry is added to `docs/IMPLEMENTED.md` on that branch. Never commit work for a step directly on `main`. Everything not covered by that file (other changes, fixes outside the listed steps) still follows the rule above
- Commit messages: imperative mood, e.g. "Add retry to client"
- PRs: short description, link the issue (`Closes #123`), keep them small
- No `.github/` directory and no GitHub Actions workflows: never add them. The four Definition-of-done checks are run locally

## Working through the steps
- Never stop to ask for confirmation, and never end a turn with a question such as "Should I start the next step?", until every step in `docs/IMPLEMENTATION.md` is finished (through the last one). After a step is published, report it briefly and start the next step in the same turn
- Stop early only for the cases in the Rules section of `docs/IMPLEMENTATION.md`: an ambiguous step, a spec that contradicts Flutter, a missing tool, a dependency not on the allowed list, a need for sudo, a check that still fails after a real fix, or a failed push (authentication or network)
- A request such as "publish to main" that is already satisfied gets a one-line answer, then work continues with the next step

## Don't
- Don't edit generated files (`*_gen.go`); regenerate them with `go generate ./...`
- Don't commit secrets or `.env` files
- Don't let a Linux, Wayland, X11 or GL type appear in a public package or in `internal/platform`'s or `internal/gpu`'s interfaces
- Never look for, list, open or read files outside this project folder (no home-directory databases, configs, logs or other projects). If something outside it is needed, ask the user to provide it. Exceptions: dependency source in the Go module cache, and system C headers (`/usr/include`) when writing or checking Go mirrors of C structs

## Testing notes
- Widget tests use `gluttertest` (`PumpWidget`, `Pump`, `Tap`, finders) with a fake clock and the fake platform; no display needed
- Golden tests render through the software renderer (`internal/gpu/soft`), so they are deterministic and need no GPU
- Tests that open real windows or GL contexts carry the `glutter_display` build tag and are skipped by the default test run
- Every error Tag has a route test
