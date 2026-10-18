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
