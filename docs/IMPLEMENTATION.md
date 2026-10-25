# Glutter: Implementation guide

This file breaks the milestones in `docs/SPEC.md` §18 into small steps. Each step is one unit of work for one coding agent session. Work goes strictly top to bottom. Every finished step is recorded in `docs/IMPLEMENTED.md`.

**Who this is for**: coding agents (and humans) implementing Glutter. You don't need to read the whole file; you need the Rules below and the one step you're doing.

---

## Rules

### Who does the work

The owner's choice (2026-10-25): the main session is the **manager** (Opus 5.5, high effort) and **workers** write the code (Sonnet 5.5 agents, launched with the Agent tool, `model: "sonnet"`, `effort: "high"`). The rules below apply to the step as a whole; this part says who does what.

- **The manager** does rules 1–3 (finds the step, reads it, the SPEC sections and the Flutter sources it names, creates the branch), then writes a brief for each worker in `.claude/briefs/step-<id>[-<part>].md` (git-ignored; never committed). A brief is the only thing the worker knows: the goal, the tasks it owns, the files and SPEC sections to read, the Flutter files to port, the tests to write, the decisions already made and those it must bring back instead of making, and the rules it must follow (CLAUDE.md, the "Errors" rules, the purego rules, doc comments, `// Deviation:` and SPEC §12.10, `go generate ./material`, processes it starts are stopped, display tests only under `scripts/headless-wayland.sh` or `scripts/headless-x11.sh`, heavy Go runs with the memory cap the manager names in the brief, e.g. a `ulimit -v` wrapper given to `-toolexec`).
- **Workers run in the shared checkout, on the step branch, without worktree isolation**: an isolated worktree starts from `main`, not from the step branch. A worker never runs a git command that changes state (commit, merge, rebase, stash, push, tag, branch, switch, reset, checkout of files); the manager does all of them.
- **One worker per step**, or several at once only when their parts touch disjoint directories and do not depend on each other; at most three at a time, since parallel `-race` builds can run the machine out of memory. A part that needs another part's API waits for it.
- **The manager reviews the work, not the summary**: it reads the diff, runs the tests the worker says pass and the checks of rule 8 itself, and looks at every proof image. What is wrong, missing or outside the step goes back to the same worker (SendMessage, so it keeps its context) with the concrete failure. The manager writes only small fixes itself (a few lines, a doc comment, a test table row).
- **Questions.** A worker that hits a case of rule 7 stops and reports it; the manager decides what the plan and the SPEC already decide, and asks the user otherwise (rule 7). The manager answers "what is going on" questions of the user while workers run, and never reports a worker's result before it arrived.
- **The manager finishes the step**: the `IMPLEMENTED.md` entry (rule 10; it may have a worker draft it, and checks every claim in it), the commit, the publish and the tag (rules 11–12), the report (rule 13), and deletes the step's briefs.

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
5. **Follow CLAUDE.md.** In particular: `rm4n0s/errors` v0.2.0 for every error, used the way its `AI_GUIDE.md` says (the "Errors" rules below), the Version 2 constructors (SPEC §12), no cgo and the purego rules (SPEC §8.1), and doc comments ending in `// Flutter: <file> <Symbol>`.
6. **Port, don't invent.** When Flutter has the behavior, port Flutter's algorithm, defaults and edge cases from `reference/flutter`. Port the relevant cases from the test files listed under **Tests** as table-driven Go tests. In `docs/IMPLEMENTED.md`, say how many test cases you ported.
7. **Stop and ask the user** when any of these happens:
   - the step is ambiguous
   - the spec looks wrong or contradicts Flutter
   - a tool is missing (give the exact install command, e.g. `sudo dnf install sway grim`)
   - a new dependency seems necessary
   - something needs `sudo`
   - a check still fails after a real attempt to fix it

   Never weaken or delete a test, skip a check, or mark a step done just to get unblocked.

**Processes you start.** Every compositor, browser, Wine instance or app you start for a test, a proof or a check is stopped when you are done with it: run it under `timeout -k`, in its own process group, and stop it from `t.Cleanup` or `defer` in Go code. Before you finish a step, check with `pgrep -a` that nothing you started is still running. The user works on this desktop and sees every stray window.

**Errors.** `github.com/rm4n0s/errors` is used to the fullest, as its `AI_GUIDE.md` says. This is a requirement of the project owner, not a style choice. It is in the module cache, so reading it is allowed: `$(go env GOMODCACHE)/github.com/rm4n0s/errors@v0.2.0/AI_GUIDE.md` (277 lines). Read all of it before you write or change code that makes, returns, logs or tests an error, and again after the dependency moves to a new version (then also read the library's `errors.go` for changed signatures). The rules, with the guide's numbers (SPEC §14.4 has the same list with the reasons):
- **R1.** `errors.New(tag, message, metadata...)` always has a tag and a message. A tag names one thing a function cannot do.
- **R2.** Metadata is given in one call, as alternating `"key", value` pairs, with string keys: to `errors.New` itself, and after `NewErr` or `NewErrf` (which take no metadata) in exactly one `SetMetadata`. A later `SetMetadata` is only for data that becomes known in another function. `RemoveMetadata` and `SetMessage` exist for the rest.
- **R3.** An error is tagged once, where it first appears. An error from a function of this module is returned as it is (`return err`): no `NewErr`, no `NewErrf("... %w")`, no `fmt.Errorf`, no new tag on the way up. An error from outside the module (the standard library, purego calls, godbus, go-text, `x/image`) is tagged where it first appears with `errors.NewErr`, so a function of this module never returns an untagged foreign error. A failure the code detects itself is `errors.New`.
- **R4.** The constructor is called in the function that fails, never in a helper that builds and returns the error (the helper would be a frame in the route). A table that turns a C error code into a tag with a switch (`vkError`) is the one accepted exception, listed with its reason in `cmd/glutter-gen/errcheck.allow`; an exported function or method is not a helper.
- **R5.** A tag is a constant declared next to the code that produces it (`const tagX = "X"` at the end of the file that makes the error, exported as `TagX` with a doc comment when another package matches on it). Code branches on `appErr.Tag == pkg.TagX` or `errors.Is(err, sentinel)`, never on a string literal or on `Error()` or `Message`. A sentinel is a struct literal, `&errors.Error{Tag: TagX}`, never a package-level `errors.New`. A tag built at run time from a prefix (a widget's name in a debug check) stays an expression and is listed in `errcheck.allow`.
- **R6.** Import only `github.com/rm4n0s/errors` (it re-exports `Is`, `As` and `Unwrap`). When something else of the standard library's `errors` is needed (`Join`), import it as `stdErrors "errors"`.
- **Logging.** An error is logged with `slog` as the attribute `"err"`: `logger.Error("...", "err", err)`. `*errors.Error` is an `slog.LogValuer`, so the entry has `Tag`, `Message`, `Route`, `OriginalErr`, `Metadata` and `StackFrames`, and a test replays it through `errors.ErrorJson` and `errors.FromJson`. Code does not call `ToJson` or `json.Marshal` on an error to log it.
- **Tests.** Every tag has a test that asserts the path with `HasRoute("Outer->Inner.Tag")` (the entry function you call, down to the tag), and the metadata when it matters. Function names in a route are written the way the guide says: `Type.Method`, no package names.
- `go run ./cmd/glutter-gen errcheck -allow cmd/glutter-gen/errcheck.allow` checks R2 to R6 and the logging rule over the whole module (SPEC §14.4). A new violation is fixed, or accepted in that file with a reason.

### Finishing a step

8. **The standard checks must all pass.** Run them from the repo root:
   ```sh
   test -z "$(gofmt -l .)"
   go vet ./...
   golangci-lint run
   go test -race ./...
   CGO_ENABLED=0 go build ./...
   CGO_ENABLED=0 go test ./...
   go run ./cmd/glutter-gen errcheck -allow cmd/glutter-gen/errcheck.allow   # from step 24.1 on
   GOOS=js GOARCH=wasm go build ./...    # from step 19.1 on
   GOOS=js GOARCH=wasm go vet ./...      # from step 19.1 on
   scripts/test-wasm.sh                  # from step 19.1 on
   scripts/check-android.sh              # from step 25.1 on: build and vet for android/arm64 and android/amd64
   ```
   If the step lists **Display tests**, run them too: `go test -race -tags glutter_display <packages>`. They run under a headless compositor (step 1.5) unless the step says otherwise.
   If the step lists **Browser tests**, run them too: `go test -tags glutter_browser <packages>` (SPEC §17).
   If the step lists **Device tests**, run them too: `go test -tags glutter_android <packages>` (SPEC §24.13). They start the emulator themselves and stop it; run them one package at a time (`-p 1`), since one emulator needs 2–4 GB of memory.
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
    git tag -a step-6.5 -m "Step 6.5: Add multi-child render objects"
    git push origin step-6.5
    git branch -d feat/step-6.5-multi-child-render-objects
    ```
    - **Tag every published step** with an annotated tag `step-<id>` (the step's id as in this file) on the commit now on `main`, and push it right after the push of `main`. The step is finished only when the tag is on `origin` too. If the tag already exists, stop and tell the user; never move or delete a tag.
    - The release tags `v<major>.<minor>.<patch>` are a separate thing: the release-checklist steps (`Release 1.x`) end with asking the user, and only the user's yes makes the `v…` tag.
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
| **D. Browser capture** (from step 21.1) | anything drawn by an app on the web | `go run ./cmd/glutter-web shot -browser chromium\|firefox [-renderer r] -o docs/implemented-proof-images/<file>.png <package>`: builds the app for js/wasm, opens it in a headless browser with a temporary profile, and saves the frame the app reads back and posts (SPEC §23.8). `GLUTTER_DEV_*` values are given as URL query parameters with `-query` |
| **E. Android capture** (from step 25.3) | anything drawn by an app on Android | `go run ./cmd/glutter-android shot [-avd glutter-37] [-renderer r] [-env GLUTTER_X=v]... -o docs/implemented-proof-images/<file>.png <package>`: builds the APK, starts the emulator if no device is given, runs the app with `GLUTTER_SCREENSHOT=1` and pulls the frame the app saved (SPEC §24.13). `-screen` captures the device's whole screen instead (status bar, keyboard, notifications). The emulator is stopped afterwards |

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
| M19 Portability groundwork (1.1) | 19.1–19.3 | |
| M20 Vulkan renderer | 20.1–20.7 | |
| M21 Web | 21.1–21.8 | the examples on the web |
| M22 Fonts and trees for code editors (1.2, 1.4) | 22.1–22.11 | |
| M23 Windows for tray apps | 23.1 | `tray` |
| M24 Errors to the fullest (`rm4n0s/errors` v0.2.0) | 24.1 | |
| M25 Android platform (1.6) | 25.1–25.12 | `dev/android-hello`, `dev/android-services`; the examples on the emulator |
| M26 Android behaviors of the framework, and 1.6 | 26.1–26.6 | every example but `multiwindow` and `tray` on Android |
| M27 Fixes for Emetos (1.7) | 27.1–27.11 | `multiwindow` |

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
**Tasks:** every simulation in SPEC §11.1's `physics` row, plus `Tolerance`. That includes `BouncingScrollSimulation` and `ClampingScrollSimulation`, which Flutter keeps in `widgets/scroll_simulation.dart`: port that file here (step 9.2 does not).

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
**Port from:** `widgets/scrollable.dart`, `scroll_controller.dart`, `scroll_position.dart`, `scroll_position_with_single_context.dart`, `scroll_physics.dart`, `scroll_activity.dart`, `scroll_configuration.dart`, `scroll_notification.dart`, `scroll_metrics.dart`, `scrollbar.dart`, `scroll_delegate.dart`, `scroll_context.dart`, `scrollable_helpers.dart`.
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
**Port from:** `material/menu_anchor.dart`, `menu_style.dart`, `menu_theme.dart`, `menu_bar_theme.dart`, `menu_button_theme.dart`, `popup_menu.dart`, `popup_menu_theme.dart`, `dropdown.dart`.
**Tasks:** the rest of SPEC §11.2 "Actions", plus `ShowMenu`. Menus open inside the window's `Overlay` (SPEC §8.7). Include keyboard navigation and nested submenus. New page: "Menus", with a `MenuBar` (File/Edit/View, with shortcuts shown), a popup menu and dropdowns.
**Proof:** `13.4-gallery-menus.png`, with the File menu and one submenu open (`GLUTTER_DEV_OPEN_MENU=file/recent`).

### 13.5 Text fields
**Port from:** `material/input_decorator.dart`, `text_field.dart`, `text_form_field.dart`, `adaptive_text_selection_toolbar.dart`, `desktop_text_selection.dart`, `desktop_text_selection_toolbar.dart`, `desktop_text_selection_toolbar_button.dart`, `text_selection.dart`, `selection_area.dart`, `text_selection_theme.dart`.
**Tasks:** `InputDecoration`/`InputDecorator`, `TextField`, `TextFormField` (with `Form` validation), `DropdownButtonFormField`, `DropdownMenu` with `DropdownMenuTheme` and `DropdownMenuEntry` (from `material/dropdown_menu.dart` and `dropdown_menu_theme.dart`, moved from 13.4: they are text fields with a menu), the desktop context menu on right-click, and the material `SelectionArea`. New page: "Text fields" (filled, outlined, with errors, prefix/suffix, multi-line, password).
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

---

## Tools for M19–M21

Install a missing tool only by asking the user (rule 7), with the command from this table.

| Tool | Needed from | Check | Install (Fedora) |
|---|---|---|---|
| Node.js | 19.1 (`scripts/test-wasm.sh`) | `node --version` | `sudo dnf install nodejs` |
| naga | 19.3 (WGSL → SPIR-V) | `naga --version` | `cargo install naga-cli --locked` |
| Mesa lavapipe | 20.1 | `ls /usr/share/vulkan/icd.d/lvp_icd.*.json` | `sudo dnf install mesa-vulkan-drivers` |
| Vulkan validation layers | 20.1 | `ls /usr/share/vulkan/explicit_layer.d/` lists `VkLayer_khronos_validation.json` | `sudo dnf install vulkan-validation-layers` |
| Vulkan headers | 20.1 (layout tests) | `ls /usr/include/vulkan/vulkan_core.h` | `sudo dnf install vulkan-headers` |
| Chromium | 21.1 | `chromium-browser --version` or `chromium --version` | `sudo dnf install chromium` |
| Firefox | 21.1 | `firefox --version` | `sudo dnf install firefox` |

---

## M19: Portability groundwork (1.1)

### 19.1 Per-OS backends and the js/wasm build
**Goal:** `GOOS=js GOARCH=wasm go build ./...`, `go vet` and the wasm unit tests pass, and nothing changes on Linux.
**Read:** SPEC §6.2, §6.3 (rules 1, 2, 7 and 8), §17 "Web unit tests", §23.1, D18.
**Files:** `internal/engine/backends_linux.go`, `backends_js.go`, `backends_other.go`; `foundation/web_js.go`, `foundation/web_other.go`; `scripts/test-wasm.sh`; build tags in `internal/platform/linux/...` (including `dl` and `wayland/fakecomp`) and in the purego files of `internal/gpu/gles`; `widgets/run_app.go`.
**Tasks:**
1. Put `//go:build linux` on every file of `internal/platform/linux/...` and on the files of `internal/gpu/gles` that use purego or EGL. Each package whose files are all tagged keeps an untagged `doc.go` (SPEC §6.3 rule 2).
2. Move the choice of the platform, the renderer and the font matcher out of `internal/engine/engine.go` and `widgets/run_app.go` (which imports `internal/text/fontconfig` today) into `internal/engine/backends_<goos>.go`. Linux keeps today's behavior exactly. On js and on other OSes, `Run` fails with `BackendUnavailable` for now (step 21.1 adds the web). No public package imports an OS-specific package.
3. `foundation.IsWeb`, true only under `js && wasm` (`// Flutter: foundation/constants.dart kIsWeb`).
4. `scripts/test-wasm.sh`: runs `GOOS=js GOARCH=wasm go test` for every package that has no OS tag, with `$(go env GOROOT)/lib/wasm` on `PATH` and a minimal environment (`env -i` with `HOME`, `PATH`, `GOCACHE`, `GOMODCACHE`, `GOFLAGS`): Go's wasm loader fails when the environment is larger than its limit. The tests of `fontscan` (system fonts) skip on js, with the reason.
5. The three wasm checks of rule 8 are standard from this step on (CLAUDE.md "Definition of done" lists them).
**Tests:** the js and other-OS backends return `BackendUnavailable` (route test, run by `scripts/test-wasm.sh`); `foundation.IsWeb` is false in the normal test run.
**Done when:** the default "Done when", and the display tests of steps 1.10 and 7.4 still pass.

### 19.2 Choosing the renderer
**Goal:** the engine chooses a renderer by `RunAppOpts.Renderer` or `GLUTTER_RENDERER`, with fallback, while GLES is still the only renderer on Linux.
**Read:** SPEC §9.6, §6.3 (the `SurfaceHandle` sketch), §14.1.
**Files:** `internal/engine/renderers.go`, `internal/engine/backends_linux.go`, `internal/gpu/surface.go`, `internal/platform/linux/wayland/window.go`, `internal/platform/linux/x11/window.go`, `widgets/run_app.go`, the `material` re-exports (regenerate with `go generate ./...`).
**Tasks:**
1. `RunAppOpts.Renderer string` and the environment variable `GLUTTER_RENDERER`, with the values of SPEC §9.6; `RendererUnknown` and `RendererUnavailable`. Until step 20.2, `vulkan` is `RendererUnavailable`.
2. The engine keeps an ordered list of renderer candidates per OS and chooses at the first window, as SPEC §9.6 says: a candidate is kept once it has presented its first frame; otherwise its error is logged once at Info level and the next one is tried. A renderer asked for by name never falls back. The Linux list is `gles` for now; step 20.7 puts `vulkan` before it.
3. `gpu.SurfaceHandle` gains `Surface uintptr`: the `wl_surface` on Wayland, the window again on X11. `Window` stays the `wl_egl_window`, which `gles` uses.
4. The startup debug log and the `GLUTTER_FRAME_STATS` lines name the renderer and its device.
**Tests:** a table test of the choice with fake candidates: asked for by name, `auto` with the first candidate failing (falls back, logs once), every candidate failing (the last error is returned), an unknown value, a known but unavailable value; route tests for `RendererUnknown` and `RendererUnavailable`.
**Display tests:** `examples/dev/window` runs with `GLUTTER_RENDERER=gles` and with `auto`.

### 19.3 WGSL shaders and the explicit device interface
**Goal:** the shader toolchain of D20 and the `explicit.Device` interface exist, with a fake device for tests.
**Read:** SPEC §9.3, D19, D20, §20 (the naga row).
**Files:** `cmd/glutter-gen/internal/shaders/`, `internal/gpu/explicit/device.go`, `internal/gpu/explicit/shaders/*.wgsl`, `internal/gpu/explicit/shaders_gen.go` and `*.spv`, `internal/gpu/explicit/fakedevice/`.
**Tasks:**
1. `glutter-gen shaders` runs `naga` on each `shaders/*.wgsl`: it validates the shader, writes `<name>.spv`, and writes `shaders_gen.go` with the WGSL text, the SPIR-V (through `embed`) and the bindings of each entry point (group, binding, kind), read from the WGSL declarations. The generated header records naga's version. `go generate ./internal/gpu/explicit` runs it.
2. A test that the generated files are up to date: it generates into a temporary directory and compares. Without `naga` on `PATH` it skips with a message; builds always use the checked-in files.
3. `explicit.Device`, `Encoder`, `Pass` and `Swapchain` as sketched in SPEC §9.3, with the design limits as `Limits`. In debug builds a call that breaks a limit panics with `GPULimitExceeded` (SPEC §14.2).
4. `fakedevice`: records every call (resources, passes, draws, copies) for unit tests, and checks the design limits.
5. The first shaders, ported from the GLES `blit` and `solid` shaders: `blit.wgsl` (frame texture → swapchain image) and `solid.wgsl` (instanced solid rectangles).
**Tests:** table tests of the generator's binding parser; the fake device rejects each broken limit (route tests for `GPULimitExceeded`).

---

## M20: Vulkan renderer

### 20.1 Vulkan loader, instance and device
**Goal:** Vulkan is loaded through purego and draws into an offscreen texture.
**Read:** SPEC §9.4, §8.1 (all rules), D17.
**Files:** `internal/gpu/vulkan/` (`lib.go`, `vk.go` with the function table, `structs.go`, `structs_layout_test.go`, `arena.go`, `instance.go`, `device.go`, `memory.go`).
**Tasks:**
1. Load `libvulkan.so.1` (`GLUTTER_VULKAN_LIBRARY` overrides it) and `vkGetInstanceProcAddr`; the function table with the functions SPEC §9.4 needs; `SymbolMissing` for a missing one. A test checks that no function in the table takes a struct by value.
2. Go mirrors of the Vulkan structs used, each with a layout test against `/usr/include/vulkan/vulkan_core.h`; the C-memory arena for create infos and the arrays they point to.
3. The instance (with `GLUTTER_VULKAN_VALIDATION=1`: the validation layer and the debug-utils messenger, the one callback, logging through the engine's logger), the choice of physical device of SPEC §9.4 (including `GLUTTER_VULKAN_DEVICE` and the CPU rule), the logical device and its queue, the memory allocator, the staging ring, fences, and command buffers for at most 2 frames in flight.
4. `explicit.Device` for Vulkan, offscreen only: buffers, textures, samplers, pipelines from the SPIR-V of step 19.3, bind groups (descriptor sets from a pool per frame), passes with dynamic rendering, copies, and `ReadTexture` (it waits for the fence).
5. Errors `VulkanLoadFailed`, `VulkanUnsupported`, `VulkanNoDevice` and `GPUOutOfMemory`, each with a route test.
**Tests:** the layout tests; the arena; the device choice as a table test over fake lists of physical devices.
**Display tests:** with lavapipe (`VK_DRIVER_FILES` pointing at `lvp_icd.*.json`) and validation on; they need Vulkan but no display: clear an offscreen texture, draw `solid` rectangles, read back and compare the pixels; no validation message.

### 20.2 Vulkan windows: surfaces and swapchains
**Goal:** a Glutter window presents through Vulkan, on Wayland and on X11.
**Read:** SPEC §9.4 (swapchain, damage), §9.3 "Frames", §8.11 "Frame pacing" and "Resizing and scaling", §9.6.
**Files:** `internal/gpu/vulkan/surface.go`, `swapchain.go`; `internal/gpu/explicit/renderer.go` (skeleton); `internal/engine/backends_linux.go`.
**Tasks:**
1. A `VkSurfaceKHR` from the `SurfaceHandle` (Wayland: `Display` and `Surface`; X11: the Xlib display and the window) and the swapchain of SPEC §9.4: format, the order of present modes, image count, recreation on `VK_ERROR_OUT_OF_DATE_KHR`, `VK_SUBOPTIMAL_KHR` and a new size. Errors `VulkanSurfaceFailed` and `VulkanSwapchainFailed`.
2. The skeleton of `explicit.Renderer`, implementing the engine's renderer: the frame texture, the clear to the background, the blit, and presenting with damage rectangles through `VK_KHR_incremental_present` when the device has it.
3. `GLUTTER_RENDERER=vulkan` works, with no fallback yet. `GLUTTER_SCREENSHOT` reads the frame texture back.
4. On Wayland, the order of SPEC §8.11: `ack_configure`, then the swapchain at the new size, then the present. A suspended window presents nothing. Several windows share the device.
**Display tests:** under headless sway with lavapipe and validation on: a window opens and presents its background; the resize test of step 3.7, run with the Vulkan renderer, never sees a buffer of the old size after `ack_configure`; on X11 (the server step 1.9's tests use), a window presents.

### 20.3 Shapes and rect clips on Vulkan
**Goal:** the explicit renderer draws every analytic shape.
**Read:** SPEC §9.1 table rows "Rect, RRect…", "Clips" (the scissor part) and "State changes"; §9.3.
**Port from:** `internal/gpu/gles/shapes.go` and `internal/gpu/gles/shaders/` (`shape.*.glsl`, `sdf.glsl`, `common.glsl`, `paint.glsl`). `gles` is the reference for these algorithms.
**Files:** `internal/gpu/explicit/renderer.go`, `shapes.go`, `batch.go`; `shaders/shape.wgsl`; the code moved from `gles` into `internal/gpu`.
**Tasks:**
1. Move the code that does not depend on GL (instance encoding, paint parameters and uniform layouts, batching keys) from `gles` into `internal/gpu`, so that both renderers use it. The `gles` tests pass unchanged.
2. From `gpu.Flatten`'s draw list to instanced draws batched by pipeline, texture and blend mode; the uniform ring with dynamic offsets aligned to 256 bytes; rect clips through the scissor.
3. Every analytic shape in WGSL (rect, rrect, rsuperellipse, oval, circle, DRRect, lines, points), drawing what the GLES shaders draw.
4. `examples/dev/canvas-tour` honors `GLUTTER_RENDERER`.
**Display tests:** the shape cells of `shapes-clips`, drawn through Vulkan, match the `soft` goldens within the tolerance `gles` uses; no validation message.
**Proof:** B, `20.3-vulkan-canvas-tour-shapes.png`: the shapes page of `examples/dev/canvas-tour` with `GLUTTER_RENDERER=vulkan`.

### 20.4 Paths, clip masks, images and text on Vulkan
**Goal:** general paths, antialiased clips, images and text through the explicit renderer.
**Read:** SPEC §9.1 table rows "General paths", "Text", "Images" and "Clips"; §9.3 (design limits, mipmaps).
**Port from:** `internal/gpu/gles/paths.go`, `clipmask.go`, `images.go`, and the text drawing of step 4.4.
**Tasks:**
1. The coverage atlas (`internal/gpu/atlas.go`, shared) in r8unorm textures, uploaded with rows aligned to 256 bytes.
2. Clip masks, as `gles` makes them.
3. Image textures with mipmaps made by the downsampling pass; bicubic sampling; nine-patch, atlas and vertices draws.
4. Glyphs through the path atlas, as `gles` draws them.
**Display tests:** the `shapes-basic`, `shapes-clips`, image and text scenes match the `soft` goldens.
**Proof:** B, `20.4-vulkan-canvas-tour-paths.png` (the paths page of `examples/dev/canvas-tour`) and `20.4-vulkan-text-tour.png` (`examples/dev/text-tour`), both with `GLUTTER_RENDERER=vulkan`.

### 20.5 Gradients, blend modes and layers on Vulkan
**Goal:** paint effects through the explicit renderer.
**Read:** SPEC §9.1 table rows "Gradients", "Blend modes" and "Layers"; §9.3 "Frames".
**Port from:** `internal/gpu/gles/layers.go`, `filters.go`, `shaders/blend.glsl`, `layer.*.glsl`, `filter.*.glsl`.
**Tasks:**
1. Gradient shaders (linear, radial, two-point conical, sweep, every tile mode) and image shaders.
2. The Porter-Duff modes through the pipeline's blend state; every other mode in a shader that reads a copy of the frame texture's region.
3. `SaveLayer`, opacity, color filters and layer pushes, in pooled textures.
**Display tests:** the `gradients` and `blend-modes` scenes match the `soft` goldens.
**Proof:** B, `20.5-vulkan-canvas-tour-blend.png` (the blend-mode page, `GLUTTER_RENDERER=vulkan`).

### 20.6 Effects, raster cache, snapshots and full parity on Vulkan
**Goal:** Vulkan draws everything `gles` draws.
**Read:** SPEC §9.1 table rows "BoxShadow / drawShadow", "Blur", "Layers", "Raster cache" and "Damage"; §9.3; §12.10 (the raster-cache and damage rows).
**Port from:** `internal/gpu/gles/effects.go`, `rastercache.go`, `renderer.go` (`Snapshot`).
**Tasks:**
1. The analytic shadow, the separable blur with downsampling, backdrop filters, shader masks and image filters.
2. The raster cache, with the rule of step 17.2.
3. `Snapshot` and `RasterizeScene` (`ui.Rasterizer`) through Vulkan.
4. Damage tracking (`gpu.DamageTracker`) into `VK_KHR_incremental_present`.
5. The parity test of step 3.6 over every golden scene, for Vulkan.
**Display tests:** the full parity test; the raster-cache and damage tests of `gles`, run for Vulkan.
**Proof:** B, `20.6-vulkan-gallery.png`: `examples/gallery` with `GLUTTER_RENDERER=vulkan`.

### 20.7 Vulkan by default, with GLES as the fallback
**Goal:** D17 holds: an app uses Vulkan where it works and GLES elsewhere, without being told.
**Read:** SPEC §9.6, §9.4 (device choice, device lost), §15, §17 (the Wayland matrix).
**Tasks:**
1. The Linux candidates for `auto` become `vulkan`, then `gles`. A Vulkan whose only device is a CPU gives GLES. Each fallback is logged once at Info level with its tagged error.
2. Device loss as SPEC §9.4 says (`VulkanDeviceLost`), tested with a fake device that reports a loss.
3. Run the benchmarks of step 17.4 and `scripts/perf.sh` with both renderers and record them in `docs/PERFORMANCE.md`. Vulkan meets SPEC §15 and is not more than 10% slower than GLES on any metric; otherwise list the exceptions for the user.
4. `README.md`: the runtime requirements and `GLUTTER_RENDERER`.
**Tests:** the choice with fake candidates: no Vulkan gives GLES; a CPU device gives GLES; `vulkan` asked for by name without Vulkan returns `VulkanLoadFailed` from `RunApp`; route tests.
**Display tests:** under headless sway: with `VK_DRIVER_FILES` pointing at a file that does not exist, an app falls back to GLES and runs; with lavapipe and `auto` it uses GLES (the CPU rule); with lavapipe and `GLUTTER_RENDERER=vulkan` it uses Vulkan.
**Manual checks:** ask the user to run `go run ./examples/gallery` on their desktop with no environment variables set, and to confirm that the log names `vulkan` and their GPU, and that resizing, moving between monitors and scrolling work; then once with `GLUTTER_RENDERER=gles`, to compare. Record the answers in `IMPLEMENTED.md`.
**Done when:** the default "Done when", the performance targets are met or the user approved the exceptions, and the manual checks are recorded.

---

## M21: Web

### 21.1 Web platform skeleton and glutter-web
**Goal:** `examples/counter` opens in a browser, drawn by `soft` into a 2D canvas.
**Read:** SPEC §23.1, §23.2, §23.3, §23.6 (settings), §23.8, §9.6 (the web candidates), §13, §17 "Browser tests".
**Port from:** `engine/src/flutter/lib/web_ui/lib/src/engine/view_embedder/`, `engine/src/flutter/lib/web_ui/lib/src/engine/browser_detection.dart`, `packages/flutter/lib/src/foundation/_platform_web.dart`.
**Files:** `scripts/fetch-reference.sh`, `internal/platform/web/`, `internal/gpu/canvas2d/`, `internal/engine/backends_js.go`, `cmd/glutter-web/`, `internal/webtest/`.
**Tasks:**
1. Add `engine/src/flutter/lib/web_ui` to the sparse paths of `scripts/fetch-reference.sh`, at the pinned commit, and run it.
2. The web `Platform`: `Run` and `Post` as SPEC §23.3 says; one window (a canvas in the host element, or the full page); size and device pixel ratio; frames through `requestAnimationFrame`; lifecycle (focus, `visibilitychange`); settings (SPEC §23.6); `Displays`; `TargetPlatform` from the browser; `HostElementNotFound`, `WindowUnsupported`.
3. `canvas2d`: draws with `soft` and puts the image into the canvas with `putImageData`, one `js.CopyBytesToJS` per frame (the web's last fallback, SPEC §9.6).
4. `backends_js.go`: the web platform, the renderer list (`soft` only for now) and the bundled fonts only.
5. `cmd/glutter-web` with `build`, `serve` and `shot` (SPEC §23.1, §23.8); the engine's capture on idle (step 7.4) posts the PNG on the web.
6. `internal/webtest`: the harness of SPEC §17 (an `httptest` server, the launcher for Chromium and Firefox with temporary profiles, killed by process group, and the endpoint the page posts its results to).
**Tests:** the parts of the platform that are plain Go (size math, device pixel ratio, config parsing, query parameters → environment), under `scripts/test-wasm.sh`; `glutter-web` (the `index.html` it makes, gzip serving).
**Browser tests:** in Firefox and Chromium, the first frame of `examples/counter` matches its `soft` golden; resizing the browser window gives a frame of the new size.
**Proof:** D, `21.1-web-counter-soft.png`: `examples/counter` in headless Firefox, drawn by `soft`.

### 21.2 Web input
**Goal:** mouse, touch, wheel and keyboard input on the web, as Flutter web delivers it.
**Read:** SPEC §23.4, §23.3.
**Port from:** `lib/web_ui/lib/src/engine/pointer_binding.dart`, `pointer_converter.dart`, `keyboard_binding.dart`, `key_map.g.dart`, `mouse/cursor.dart` (or the files that hold this code at the pinned commit); `packages/flutter/lib/src/services/browser_context_menu.dart`; the web names in `dev/tools/gen_keycodes/data`.
**Tests:** `lib/web_ui/test/engine/pointer_binding_test.dart` and `keyboard_binding_test.dart` (the relevant cases), as table tests of the conversion functions. These take plain Go values, not JavaScript objects, so they run under `scripts/test-wasm.sh`.
**Files:** `internal/platform/web/pointer.go`, `keyboard.go`, `keymap_gen.go`, `cursor.go`; `cmd/glutter-gen/internal/keys` (the web tables); `services` (`BrowserContextMenu`).
**Tasks:**
1. Pointer, wheel and touch input as SPEC §23.4 says.
2. The keyboard, with the generated tables and the synthesized modifier events; `preventDefault` from the framework's answer (SPEC §23.3).
3. Cursors, focus handling, and `BrowserContextMenu`.
**Browser tests:** the test program dispatches synthetic DOM events: three taps on the counter's button show 3; a key that a `Shortcuts` widget handles has `defaultPrevented`; wheel events scroll a `ListView`.
**Proof:** D, `21.2-web-scrolling.png`: `examples/scrolling` after a scripted wheel scroll.

### 21.3 Web text input, IME and clipboard
**Goal:** text editing on the web, with IMEs, virtual keyboards and the clipboard.
**Read:** SPEC §23.5, §23.3.
**Port from:** `lib/web_ui/lib/src/engine/text_editing/` (`text_editing.dart`, `composition_aware_mixin.dart`, `input_type.dart`, `input_action.dart`), `lib/web_ui/lib/src/engine/clipboard.dart`.
**Tests:** `lib/web_ui/test/engine/text_editing_test.dart` and `composition_test.dart` (the relevant cases).
**Tasks:**
1. The hidden text field, kept in sync with the editing state; composition; its placement at the caret; `inputmode` and `enterkeyhint`.
2. The clipboard as SPEC §23.5 says.
**Browser tests:** synthetic `beforeinput`/`input` events type into a `TextField`; a `compositionstart`/`compositionupdate`/`compositionend` sequence commits its text; paste through the `paste` event; keys sent while a clipboard read is pending do not deadlock (SPEC §23.3).
**Proof:** D, `21.3-web-editor.png`: `examples/editor` with text typed by the test script.
**Manual checks:** ask the user to run `go run ./cmd/glutter-web serve ./examples/editor` and to try an IME (for example IBus or Fcitx5 with a Chinese or Japanese input method) in Chrome and in Firefox. Record the answers.

### 21.4 WebGL 2 through gles
**Goal:** `gles` draws on the web through WebGL 2, with the same shaders.
**Read:** SPEC §9.1, §9.6, §6.3 rule 8, §22 (the risk about calls into JavaScript).
**Files:** `internal/gpu/gles/gl_purego.go` (Linux), `gl_js.go` and `context_js.go` (js).
**Tasks:**
1. Split the GL function table into a purego binding (Linux) and a WebGL 2 binding through `syscall/js`: a `js.Value` per GL object, and reused typed arrays for uploads.
2. The context from the canvas: `getContext("webgl2", {alpha, premultipliedAlpha, antialias: false, preserveDrawingBuffer: false, ...})`, matching SPEC §9.1. Handle what WebGL 2 lacks (BGRA, some `readPixels` formats, swap with damage). `webglcontextlost` and `webglcontextrestored` drop and recreate every resource, and the next frame is drawn whole. `WebGL2Unavailable` when there is no context.
3. The web renderer list becomes `webgl`, then `soft`.
**Tests:** both bindings have the same set of functions.
**Browser tests:** the parity test over every golden scene, in Chromium and in Firefox with WebGL 2, within the `gles` tolerance.
**Proof:** D, `21.4-web-gallery-webgl.png`: `examples/gallery` in Firefox with `GLUTTER_RENDERER=webgl`.

### 21.5 WebGPU
**Goal:** the explicit renderer draws on the web through WebGPU.
**Read:** SPEC §9.3, §9.5, §23.3.
**Files:** `internal/gpu/webgpu/`.
**Tasks:**
1. `explicit.Device` over `navigator.gpu`, as SPEC §9.5 says: adapter and device awaited at startup, before the listeners; the canvas context as the swapchain; `writeBuffer` and `writeTexture`; `ReadTexture` that finishes later; `Snapshot` through `soft`.
2. Errors `WebGPUUnavailable`, `WebGPUDeviceFailed` and `ShaderModuleInvalid`, with route tests.
3. The web renderer list becomes `webgpu`, `webgl`, `soft`.
**Browser tests:** the parity test over every golden scene, in Chromium with WebGPU (the flags it needs are kept in `internal/webtest`), and in Firefox when its WebGPU is on.
**Proof:** D, `21.5-web-gallery-webgpu.png`: `examples/gallery` in Chromium with WebGPU.

### 21.6 Web fallback, GPU loss and services
**Goal:** the web chooses its renderer as SPEC §9.6 says, survives a lost GPU, and has the services of SPEC §23.7.
**Read:** SPEC §9.6, §9.5 (device lost), §23.7.
**Tasks:**
1. `auto` tries `webgpu`, `webgl`, then `soft` at the first frame, with a new canvas element before each try.
2. A lost WebGPU device or WebGL context is recovered as SPEC §9.5 and step 21.4 say; WebGPU's last resort switches the page to WebGL 2.
3. `desktop.OpenURI` and `desktop.ShowNotification` on the web (SPEC §23.7). The other services return their `Unavailable` tags.
**Tests:** the choice with fake candidates on js (under `scripts/test-wasm.sh`); route tests.
**Browser tests:** Chromium with WebGPU turned off uses WebGL 2; with WebGL turned off too, `soft`; `GLUTTER_RENDERER=webgpu` without WebGPU fails with `WebGPUUnavailable`, which the loader shows; a refused notification permission gives `NotificationsUnavailable`.

### 21.7 Web examples, size and performance
**Goal:** the examples run on the web, within the web targets of SPEC §15.
**Read:** SPEC §15 (the web table), §19, §23.1, §22.
**Tasks:**
1. `examples/examples_test.go` builds every example except `multiwindow` and `tray` for js/wasm.
2. `scripts/perf-web.sh` measures the sizes, the time to first frame, idle CPU and frame times in Chromium (WebGPU) and Firefox (WebGL 2), with release builds.
3. Profile and fix until the targets are met. If calls into JavaScript dominate, use the remedy SPEC §22 names.
4. `docs/PERFORMANCE.md`: the web table. `README.md`: a "Web" section (building, serving, browsers, what the web lacks). `docs/GUIDE.md`: a note for Flutter web developers (renderers, fonts, `IsWeb`).
**Manual checks:** ask the user to open `go run ./cmd/glutter-web serve ./examples/gallery` in their Chrome and Firefox and try scrolling, the theme switch and a text field. Record the answers.
**Done when:** the default "Done when", the web targets are met or the user approved the exceptions, and the manual checks are recorded.

### 21.8 Release checklist for 1.1
**Tasks:** confirm and record in `IMPLEMENTED.md`:
- the standard checks, the display tests and the browser tests
- the manual checks of 20.7, 21.3 and 21.7
- the performance targets (20.7, 21.7)
- the API check (`go run ./cmd/glutter-gen apicheck -allow cmd/glutter-gen/apicheck.allow`)

Then **ask the user** before tagging `v1.1.0` and pushing the tag.

---

## M22: Fonts and trees for code editors (1.2)

These steps come from a request by the Emetos code editor, which is built on Glutter. Steps 22.1 to 22.3 are what it needs first; 22.4 releases them. Steps 22.5 to 22.8 follow, each on its own branch. The tests that need a monospace font use Go Mono from `golang.org/x/image/font/gofont/gomono` (already a dependency), so that they give the same result on every machine.

### 22.1 Fonts loaded at run time
**Goal:** an app loads TrueType and OpenType fonts under a family name while it runs, and the family is usable in `TextStyle`.
**Read:** SPEC §7, §10 ("Loaded fonts"), §12.10 (the rows on `LoadFontFromList`, `FontLoader`, `RunAppOpts.Fonts` and `RelayoutWhenSystemFontsChangeMixin`), §13, §14.1.
**Port from:** `packages/flutter/lib/src/services/font_loader.dart`, `packages/flutter/lib/src/painting/binding.dart` (`handleSystemMessage`, `systemFonts`), `packages/flutter/lib/src/rendering/object.dart` (`RelayoutWhenSystemFontsChangeMixin`). `dart:ui`'s `loadFontFromList` is not in `reference/`; SPEC §12.10 says what it does here.
**Files:** `internal/text/fonts.go`, `internal/text/layout/engine.go`, `ui/platform_dispatcher.go`, `services/font_loader.go`, `painting/binding.go`, `widgets/binding.go`, `widgets/run_app.go`, `docs/GUIDE.md`.
**Tasks:**
1. `text.Collection.AddFont(family string, data []byte) error`: a family can hold several fonts. Read each font's weight and slant (OS/2, and the `wght` and `ital` axes of a variable font). `Match` puts the loaded fonts of a family first, the closest to the requested weight and slant first, then the bundled font, then the system fonts. The same bytes under the same family are loaded once. `Collection.Generation()` grows with every change of the loaded fonts. An empty family takes the family name of the font (`FontFamilyMissing` when it has none); data that is not a font is `FontDataInvalid`.
2. `layout.Engine.LoadFontFromList(list []byte, family string) error`; the engine drops what it resolved for a style (`byStyle`, the faces per character) when the generation changes, so a family that was missing a moment ago now resolves to the loaded font.
3. `ui.PlatformDispatcher.LoadFontFromList(list, family) error` (`FontLoadUnsupported` when the text engine cannot load fonts) and `OnFontsChanged`, called after a font was added.
4. The widgets binding sets `OnFontsChanged` to `PaintingBinding.HandleSystemFontsChange`, and listens to `SystemFonts()`: it calls `SystemFontsDidChange` on every render object that has the method, in every view's render tree, and schedules a frame.
5. `services.FontLoader` as SPEC §12.10 says (`NewFontLoader`, `AddFont`, `AddFontBytes`, `AddFontAsset`, `Load`), with `FontLoaderReused`, `FontLoadCanceled`.
6. `widgets.RunAppOpts.Fonts []FontSource{Family, Data, Asset}`, loaded in `RunApp` before the first frame with the `AssetBundle` of `Assets`; `FontSourceInvalid` for a source with both or neither, `AssetNotFound` for a missing asset (the loader's own tag). The failure is returned by `RunApp`. `systemTextEngine` moves its work to a function that takes the sources and that `gluttertest` does not need.
7. `docs/GUIDE.md`: a section "Fonts" (a code editor's monospace font: loading, `FontFamily`, `FontFamilyFallback`, weights and italics).
**Tests:** table tests for `AddFont` (one font, a family of regular + bold + italic and which one `Match` puts first for each request, a variable font, the same bytes twice, a family from the font, invalid data); the generation; a paragraph that uses a family before and after it is loaded (widths of Go Mono: every glyph the same advance); `FontLoader` (order, reuse, a failing source, cancel); `RunApp` with `Fonts` on the fake platform (bytes and asset; the error of a bad source); a `RenderParagraph` laid out again after `LoadFontFromList` through the binding; route tests for every new tag.
**Display tests:** none. **Browser tests:** a font loaded from bytes is used on the web (the Go Mono widths), run with `GOOS=js` under `scripts/test-wasm.sh` for the pure parts.

### 22.2 Fonts in `gluttertest`
**Goal:** a widget test can show text in a font that is not Roboto, the same on every machine.
**Read:** SPEC §17, §12.10 (the `gluttertest` row).
**Files:** `gluttertest/tester.go`, `internal/renderingtest/renderingtest.go`, `gluttertest/doc.go`.
**Tasks:**
1. `gluttertest.Font{Family string; Data []byte}` and `Opts.Fonts []Font`. `New` builds a text engine with Roboto, Material Icons and these fonts (no system fonts) with `renderingtest.NewTextEngine(fonts ...)`; the engine of a test without `Fonts` stays the shared one. A font that fails to load fails the test with its tag; `Fonts` together with `TextEngine` fails the test (`tb.Fatalf`).
2. A golden test with monospace text: a `Text` in Go Mono at several sizes and weights (Go Mono has regular, bold, italic and bold italic: `gomono`, `gomonobold`, `gomonoitalic`, `gomonobolditalic`).
**Tests:** the engine of `Fonts` has the family (widths); the shared engine is untouched by another test's fonts; `Fonts` with `TextEngine`; a bad font. Golden: `gluttertest/testdata/goldens/monospace_text.png`, looked at.
**Proof:** none.

### 22.3 Fallback to the listed families first
**Goal:** `FontFamily`, then each `FontFamilyFallback` entry in order, then the system default.
**Read:** SPEC §10 ("Font fallback order"), §12.10 (the fallback row).
**Files:** `internal/text/fonts.go`, `internal/text/scan.go`, `internal/text/layout/engine.go`, `internal/platform/linux/fontconfig/fontconfig.go`, `internal/engine/backends_linux.go`.
**Tasks:**
1. `text.Aliaser` (optional on a `Matcher`): the family names that a name stands for. fontconfig: the `FC_FAMILY` values of the pattern after `FcConfigSubstitute` that a name that matches nothing (a probe name) does not get as well. fontscan: the family of the font it substitutes, when it differs from the one it gives for the probe name.
2. `Collection.Match` gives, for a family that is not generic, the loaded and bundled fonts, and the system fonts that belong to the family (or to one of its aliases); for a generic family (`sans-serif`, `serif`, `monospace`, `cursive`, `fantasy`, `emoji`, `math`) the whole list as before.
3. The engine's candidates are the families of the style, in order, then `sans-serif`; the rule of the last-resort `Fallback` by character stays.
**Tests:** a fake matcher that, like fontconfig, answers a missing family with the default sans font first: `FontFamily: "Missing", FontFamilyFallback: ["monospace"]` draws a Latin glyph with the monospace face; `FontFamily: "Go Mono"` (loaded) with a fallback `Roboto`: a glyph Go Mono lacks comes from Roboto; a system font that is an alias (`Arial` → `Liberation Sans`) is used; a missing family with no fallback uses the default; order of the candidates; the fontconfig and fontscan `Aliases` (skipped with the reason when the library or fonts are not there). A paragraph in the soft renderer: the same widths as Go Mono's advance.
**Display tests:** none.

### 22.4 Release 1.2
**Tasks:**
1. Run every check of the "Finishing a step" list and the API check; record them in `IMPLEMENTED.md`.
2. `README.md` and `docs/GUIDE.md` mention loading fonts; the SPEC version line says 1.2.
3. Tag `v1.2.0` on `main` and push the tag. The owner asked for this release in the request that added M22, so no further question is asked.

### 22.5 Font queries and the desktop's font settings
**Goal:** an app can ask which face a family resolves to, which families exist, and which fonts the desktop uses.
**Read:** SPEC §10 ("Font queries"), §8.8.
**Files:** `ui/platform_dispatcher.go`, `internal/text/`, `internal/text/layout/engine.go`, `internal/platform/linux/` (portal settings), `internal/platform/fake`, `internal/engine`, `ui/platform_configuration.go`.
**Tasks:**
1. `ui.ResolvedFont{Family, Path string; Index int; Origin FontOrigin; Weight int; Italic bool}` (`FontOriginLoaded`, `FontOriginBundled`, `FontOriginSystem`; the zero value `FontOriginUnspecified`) and `PlatformDispatcher.ResolveFont(family string, weight FontWeight, style FontStyle) (ResolvedFont, bool)`: the first face of the family's candidates (what a `TextStyle` with only that family draws a Latin letter with); false when the family has no face. `Collection.Families(ctx) ([]string, error)`, `PlatformDispatcher.InstalledFontFamilies() []string`: sorted, without duplicates, from fontconfig's `FcFontList` (or the footprints of fontscan), the bundled and the loaded fonts.
2. `PlatformConfiguration.UIFontFamily/UIFontSize/MonospaceFontFamily/MonospaceFontSize`, read from the settings portal's `org.gnome.desktop.interface` `font-name` and `monospace-font-name`, followed live; a Pango description (`Family [style] size`) parsed by a function with its own table test (styles, no size, quoted names).
3. The fake platform sets them for tests (`gluttertest` `SetPlatformConfiguration`-like helper that exists for the other settings).
**Tests:** the parser (table); `ResolveFont` for a loaded, a bundled and a missing family; `InstalledFontFamilies` with a fake matcher; the settings on the fake portal; fontconfig's list (skipped with the reason when the library is missing).

### 22.6 Text benchmarks and an LRU glyph cache
**Goal:** the cost of code-like paragraphs is measured, and the glyph path cache drops the oldest entries instead of everything.
**Read:** SPEC §10 ("Glyph outline cache"), §15, `docs/PERFORMANCE.md`.
**Files:** `internal/text/layout/paint.go`, `internal/text/layout/bench_test.go`, `painting/` or `gluttertest` benchmarks, `docs/PERFORMANCE.md`.
**Tasks:**
1. Benchmarks for building (`ParagraphBuilder`), laying out and painting (`Canvas.DrawParagraph` into a recorder, and through `soft`) a paragraph of 100 columns of monospace text with 5, 10 and 20 styled spans, with Go Mono; plus a 50-line page. Record `ns/op` and allocations in `docs/PERFORMANCE.md`.
2. The cache of glyph outlines (`maxCachedGlyphs = 1<<14`) evicts the least recently used entry (a list and a map), not everything; the same for the color glyph pictures if they are cleared the same way. A cache hit moves the entry to the front.
3. Profile the benchmarks once; fix what is the biggest avoidable cost if it is small and local, otherwise write it under "Pending" in `docs/PERFORMANCE.md`.
**Tests:** the cache (size bound, the oldest goes first, a hit protects an entry, the same outline after eviction), table-driven.

### 22.7 `TreeSliver`
**Goal:** Flutter's tree view.
**Read:** SPEC §11.3.
**Port from:** `packages/flutter/lib/src/widgets/sliver_tree.dart`, `packages/flutter/lib/src/rendering/sliver_tree.dart` if present, `packages/flutter/test/widgets/sliver_tree_test.dart`.
**Files:** `widgets/sliver_tree.go`, `rendering/sliver_tree.go` (if Flutter has the render object in `rendering`), `widgets/sliver_tree_test.go`, the generated `material` re-exports, `examples/scrolling` (a tree section).
**Tasks:** port `TreeSliver`, `TreeSliverNode`, `TreeSliverController`, `TreeSliverIndentationType`, `TreeSliverNodeCallback`, `TreeSliverNodeBuilder`, `RenderTreeSliver` and its parent data, with the toggle animation. Update SPEC §12.10 (the "not ported" row becomes the list of deviations).
**Tests:** every case of `sliver_tree_test.dart`, as table tests.
**Proof:** B, `22.7-tree-sliver.png`: `examples/scrolling` with a tree that has some nodes expanded.

### 22.8 `ReorderableListView`
**Goal:** Material's reorderable list.
**Read:** SPEC §11.3.
**Port from:** `packages/flutter/lib/src/material/reorderable_list.dart`, `packages/flutter/test/material/reorderable_list_test.dart`.
**Files:** `material/reorderable_list.go`, `material/reorderable_list_test.go`, the generated re-exports.
**Tasks:** port `ReorderableListView` (default and `builder`), its default drag handles for the platform (`ReorderableDragStartListener` on desktop), the proxy decorator and `buildDefaultDragHandles`, on `widgets.ReorderableList`. Update SPEC §11.2 and §12.10.
**Tests:** every case of `reorderable_list_test.dart` that does not need what Glutter lacks, as table tests; say which ones were skipped and why.
**Proof:** B, `22.8-reorderable-list-view.png`: `examples/scrolling` with a `ReorderableListView` mid-drag.

### 22.9 Tab stops in paragraph layout
**Goal:** a code editor's paragraph puts each `\t` on the next tab stop, and every query sees the tab's whole advance.
**Read:** SPEC §10 ("Tab stops", "Paragraph layout"), §12.10 (the tab stops row), §14.1 (the tags).
**Files:** `ui/text_style.go` (`ParagraphStyleOpts.TabSize`, `ParagraphStyle.TabSize`), `painting/text_painter.go`, `painting/text_style.go` (`GetParagraphStyleOpts`), `internal/text/layout/` (`paragraph.go`, `lines.go`, `paint.go`, a new `tabs.go`), tests in `internal/text/layout/tabs_test.go`, `painting/text_painter_test.go`, `ui/`; `docs/GUIDE.md`.
**Tasks:**
1. `ui.ParagraphStyleOpts.TabSize float64`: the distance between tab stops as a count of the advance of U+0020 in the paragraph's own font (SPEC §10). 0 means no tab stops. `NewParagraphStyle` panics with the tag `TabSizeInvalid` for a negative, `NaN` or infinite value. `(*ParagraphStyle).TabSize() float64`.
2. `painting.TextPainterOpts.TabSize`, `TextPainter.TabSize()` and `SetTabSize` (which marks the layout stale when the value changes), `TextPainterComputeWidthOpts.TabSize` and `GetParagraphStyleOpts.TabSize`, passed to `ui.ParagraphStyleOpts`.
3. The text engine: the tab width is `TabSize` times the advance of a space shaped with the paragraph's base style (zero, so that nothing changes, when the paragraph has no tab or no font). The advance of the runes is measured from the start of the line: line breaking (`breakLines`), the intrinsic widths, the width of a line, `ellipsize` and the hanging space at the end all use a function that gives the width of a rune range that starts at a given distance from the line start. A tab at the end of a line hangs. When a line is built, its tab glyphs get the advance of their stop (plus the letter and word spacing of their style), so that boxes, positions, glyph info, carets, justification and `LongestLine` follow from the glyphs. A tab draws nothing.
4. `docs/GUIDE.md` says how to turn it on.
**Tests:** tables, with Go Mono (the same on every machine): tabs at columns 0 to 9 of a line (a tab on a stop advances a whole width); a tab size of 1, 2, 4, 8 and 2.5; mixed with CJK (wide glyphs) and emoji; inside styled spans (bold, a bigger size, letter and word spacing); a right-to-left paragraph and a right-to-left run inside a left-to-right paragraph; under wrapping (a tab at the end of a line hangs, the next line starts its stops again, a word wider than the line); with `maxLines` and an ellipsis; justified; and, for each of these cases, `GetBoxesForRange`, `GetPositionForOffset`, `GetGlyphInfoAt`, `GetClosestGlyphInfoAt`, the line metrics, the intrinsic widths and the caret (`TextPainter.GetOffsetForCaret` after a tab is at the stop). `TabSize` 0 gives the layout of today for text with tabs (byte for byte, compared with a paragraph made without the option); the painter passes the option; `TabSizeInvalid` has a route test. A fuzz test builds random text with tabs and checks that every query is consistent (boxes tile each line, caret positions are monotonic).

### 22.10 A cache of shaped runs
**Goal:** building a paragraph again, when its spans mostly did not change, shapes only what changed.
**Read:** SPEC §10 ("Shaped run cache"), §12.10 (the cache row), §13, `docs/PERFORMANCE.md` (step 22.6 and "Pending").
**Files:** `internal/text/layout/` (`shape.go`, `engine.go`, a new `shapecache.go`, `lru.go`), `internal/text/layout/bench_test.go`, `docs/PERFORMANCE.md`.
**Tasks:**
1. Profile `CodeLineBuild` first and write the numbers in `docs/PERFORMANCE.md`: the cost of the shaping, and of what is around it (bidi, script segmentation, `SplitByFace`, `measure`).
2. The cache: the engine keeps the shaped glyphs of an itemized run (`shapeRun`), keyed as SPEC §10 says (text, face, features, direction, script, language, and the context HarfBuzz reads). A hit copies the glyphs with their clusters moved to the place of the run and scaled to the size; letter and word spacing are applied after it, as now. Bounded by a number of entries (4,096) and by bytes (8 MiB) with a least recently used order (the generic `lruCache` of step 22.6 gets a weight), UI thread only, emptied when the fonts change (`LoadFontFromList` and the `SystemFontsDidChange` path, through the font generation the engine already checks).
3. Make the rest of the build cheap enough that an unchanged line is at least 5 times faster than the 65 to 96 µs of step 22.6: a second cache of what is found out about a text (levels, pieces by script, line breaks, clusters, words), shortcuts for ASCII and left-to-right text, an array of the faces of the ASCII characters, a memory of the last styles looked up, one `ResolvedTextStyle` for each pushed style instead of copies, and no layout at build time (a paragraph is laid out when something asks). Every shortcut is checked against the general path by a test.
4. Benchmarks, in `bench_test.go` and `docs/PERFORMANCE.md`, with times and allocations: `CodeLineBuild` of an unchanged line (the same spans again), a line with one span changed (only the runs of that span shape: the test counts the calls of the shaper), and a page of 50 lines with one line changed (about the cost of one line). Keep the numbers of step 22.6 beside them.
**Tests:** the cache returns what fresh shaping returns, fuzzed text (Latin, Arabic and Hebrew, CJK, emoji, combining marks, tabs, newlines) and styles (sizes, weights, features, letter spacing), compared glyph by glyph with an engine whose cache is off; two runs with different contexts do not share an entry (Arabic joining at a run's edge; a combining mark at the start of a run); eviction respects the entry bound and the byte bound, and the oldest goes first; a hit protects an entry; `LoadFontFromList` and a system fonts change empty the cache; the sizes differ and the entry is shared; a run with a missing face is not cached.

### 22.11 Release 1.4
**Tasks:**
1. Run every check of the "Finishing a step" list and the API check; record them in `IMPLEMENTED.md`.
2. `docs/GUIDE.md` and `README.md` mention tab stops; the SPEC version line says 1.4.
3. Tag `v1.4.0` on `main` and push the tag. The owner asked for this release in the request that added steps 22.9 and 22.10 ("Release a new Glutter version tag when both are done"), so no further question is asked.

## M23: Windows for tray apps

This step comes from a request for an app built on Glutter (English Tutor) that has a system tray icon: pressing the window's ✕ should take the window off the screen and the task bar while the process and the tray icon keep running, and a click on the icon should bring it back. The owner chose the API. It was done on the branch `feat/window-hide` before it was written here.

### 23.1 Hiding a window (close to the tray)
**Goal:** an app hides a window for real (no surface, no task-bar entry, no input, no focus) and shows it again as it was, while the app keeps running.
**Read:** SPEC §8.2 ("Hiding a window"), §8.3 ("Hidden windows"), §8.7 ("Hiding", "Closing"), §8.11 ("Frame pacing"), §13 (the lifecycle), §12.10 (the hiding row), §23.2.
**Files:** `widgets/window_controller.go`, `ui/window_control.go`, `internal/platform/platform.go`, `internal/engine/engine.go`, `internal/platform/linux/x11/{lib,window,wmstate}.go`, `internal/platform/linux/wayland/{window,decoration,extras}.go`, `internal/platform/linux/wayland/decor/decor.go`, `internal/platform/linux/wayland/fakecomp/windowing.go`, `internal/platform/fake/`, `internal/platform/web/window_js.go`, `internal/renderhost/renderhost.go`, `gluttertest/{tester,windows}.go`, `examples/tray/`, `docs/GUIDE.md`.
**Tasks:**
1. `RegularWindowController.SetHidden(hidden bool)` and `IsHidden() bool`; `Activate` (and `SetMinimized(false)`) shows a hidden window first. The view, the widget tree and its state stay. Doc comments end with `// Flutter: no counterpart`, with a `// Deviation:` and a row in SPEC §12.10.
2. The request goes through `ui.PlatformDispatcher.SetWindowHidden` (`ui.WindowControlHost`) to `platform.Window.SetHidden(hidden) bool`, which says whether the platform can hide windows.
3. X11: withdraw the window per ICCCM (`XWithdrawWindow`, loaded through purego); show it by restoring `_NET_WM_STATE`, `XMapWindow` and `_NET_ACTIVE_WINDOW`. A withdrawn window is not minimized; the frame timer stops and the requests that would map it again do nothing.
4. Wayland: unmap with a null buffer and destroy the role objects (decoration, dialog, export, `xdg_toplevel`, `xdg_surface`, or the libdecor frame); show by making a new role on the same `wl_surface`, with the initial commit, and draw nothing before its configure. The idle inhibitor and what else hangs on the surface stay; a frame callback in flight is forgotten. Both the plain `xdg_toplevel` path and the libdecor path.
5. The web ignores it (returns false); the fake platform records it.
6. The engine asks a hidden window for no frame (the one wanted is asked for on show, presented whole) and derives the desktop lifecycle from the windows: `Hidden` when every window is hidden, `Resumed` when one has the focus, `Inactive` otherwise, after the view focus events of a change. A hidden window still counts as open: `RunApp` goes on, and closing, `ExitApplication` and `RequestAppExit` work as before.
7. `examples/tray` hides its window on close and shows it from the tray; SPEC and `docs/GUIDE.md` say how.
**Tests:** table tests in `widgets` (hide, show, `Activate` on a hidden window, state kept, lifecycle `Hidden` and frames, exiting with the only window hidden); the engine over the fake platform (lifecycle, frames, `Run` going on and ending on exit, focus between windows); Wayland against the fake compositor (the requests, both decoration paths, the EGL path); `glutter_display` tests that hide and show a real window on X11 (Xvfb, with the test window manager) and Wayland (headless sway, with libdecor too); the tray example's tests.
**Proof:** none (a hidden window shows nothing); a manual check on a real desktop of `go run ./examples/tray`: ✕ takes the window off the task bar, a click on the tray icon brings it back, "Quit" exits.

## M24: Errors to the fullest

The owner asked that `github.com/rm4n0s/errors` v0.2.0 is used to the fullest, the way its `AI_GUIDE.md` says, and that the requirement is written into this file. The rules are in "While you work" ("Errors") above, in CLAUDE.md and in SPEC §14.4. This step brings the code that existed before the requirement in line with them.

### 24.1 Errors the way `AI_GUIDE.md` says
**Goal:** every error of the module is made, returned, logged and tested as the v0.2.0 guide says, and a checker keeps it so.
**Read:** `AI_GUIDE.md` of `rm4n0s/errors` v0.2.0 (all of it), SPEC §14.1, §14.2, §14.4, CLAUDE.md ("Error handling").
**Files:** `go.mod`, `go.sum`, `cmd/glutter-gen/` (`run.go`, `errcheck.allow`, `internal/errcheck/`), every package that makes or returns an error, `widgets/binding.go` (the default reporter), `docs/`.
**Tasks:**
1. Move the dependency to v0.2.0 (`ToJson` returns one value and `FromJson` returns an `*Error`; `RemoveMetadata`, `SetMessage` and the `slog.LogValuer` are new). Fix what no longer builds.
2. `glutter-gen errcheck` (SPEC §14.4): the checker reads the module for Linux, `js/wasm` and the display and browser tags, tests included, and reports `std-errors`, `fmt-errorf`, `metadata-chain`, `metadata-pairs`, `sentinel-ctor`, `error-helper`, `error-rewrap`, `error-untagged`, `tag-branch`, `tag-literal` and `error-tojson`. `cmd/glutter-gen/errcheck.allow` lists the accepted places with a reason; `TestTheModuleKeepsTheErrorRules` runs it on the module. The check is part of the standard checks from this step on.
3. R4: the functions that only built and returned an error are gone; the constructor is called in the function that fails, and the route tests say so. Only the tables that turn a C error code into a tag stay (`errcheck.allow`).
4. R3: an error is tagged where it first appears (`NewErr` on the foreign call) and goes up as it is. The wraps of errors of this module (`NewErr`, `NewErrf` with `%w`, `fmt.Errorf`) are gone. The public tags of SPEC §14.1 are the same as before.
5. R5: every tag that `errors.New`, `NewErr` or `NewErrf` is given is a constant declared in the file that produces it (`tagX`; exported `TagX` where another package matches on it); code compares tags with constants or `errors.Is`; the one package-level `errors.New` is a struct-literal sentinel. A tag built at run time from a prefix stays an expression and is listed in `errcheck.allow`.
6. Logging: errors are logged as the `slog` attribute `"err"` (the default error reporter, `glutter-gen`, the run-time logs of the engine, the platform and the D-Bus services); no `ToJson` outside tests. A test replays a logged entry through `errors.ErrorJson` and `errors.FromJson`.
7. Update CLAUDE.md, SPEC §14.2, §14.4 and §18 (M24), and `docs/GUIDE.md` where it shows error handling.
**Tests:** the checker (a fixture module per rule, a clean fixture, the allow file, the flags, every tag of the checker has a route test); the default reporter's log entry replays; every route assertion that went through a removed helper or a removed wrap is updated; every tag still has a route test.
**Proof:** none. **Done when:** `go run ./cmd/glutter-gen errcheck -allow cmd/glutter-gen/errcheck.allow` reports nothing and the standard checks pass.

## Tools for M25–M26

Install a missing tool only by asking the user (rule 7), with the command from this table. `ANDROID_HOME` is the SDK (`$HOME/Android/Sdk` on the owner's machine, with `cmdline-tools/latest/bin`, `platform-tools` and `emulator` on `PATH`). The owner allowed reading and running what is inside `ANDROID_HOME`; everything a test writes goes into `.android-test/` at the module root. On 2026-10-23 every row but the last three was installed and checked.

| Tool | Needed from | Check | Install (Fedora) |
|---|---|---|---|
| NDK 30.0.16248370 | 25.1 | `ls $ANDROID_HOME/ndk/` | `sdkmanager "ndk;30.0.16248370"` |
| SDK command-line tools | 25.2 | `sdkmanager --version` | "Command line tools only" from developer.android.com, unzipped to `$ANDROID_HOME/cmdline-tools/latest` |
| platform-tools, build-tools 37.0.0, platform 37.2 | 25.2 | `ls $ANDROID_HOME/build-tools/37.0.0/aapt2 $ANDROID_HOME/platforms/android-37.2/android.jar` | `sdkmanager "platform-tools" "build-tools;37.0.0" "platforms;android-37.2"` |
| JDK 17+ (`javac`) | 25.2 (only to change the Java) | `javac -version` | `sudo dnf install java-25-openjdk-devel` |
| Emulator and the API 37 images | 25.2 | `emulator -accel-check` says KVM is usable; `ls $ANDROID_HOME/system-images/android-37.0 $ANDROID_HOME/system-images/android-37.2` | `sdkmanager "emulator" "system-images;android-37.0;google_apis;x86_64" "system-images;android-37.2;google_apis_ps16k;x86_64"` |
| `adb` | 25.2 | `adb version` | `sudo dnf install android-tools` (or the SDK's `platform-tools`) |
| API 26 image (the minimum API level) | 25.7 | `ls $ANDROID_HOME/system-images/android-26` | `sdkmanager "system-images;android-26;google_apis;x86_64"` |
| Vulkan validation layer for Android | 25.5 | `ls .android-test/vvl/x86_64/libVkLayer_khronos_validation.so` | the `android-binaries-*.zip` of the newest release at github.com/KhronosGroup/Vulkan-ValidationLayers/releases, unzipped into `.android-test/vvl/` |
| The owner's phone with USB debugging | 25.4 (manual checks) | `adb devices` lists it | Settings → About phone → tap "Build number" 7 times; Developer options → USB debugging |

---

## M25: Android platform (1.6)

The owner asked for Android support "like Flutter" on 2026-10-23 and chose: cgo only on Android (D21), a small Java embedding (D22), Vulkan with GLES as the fallback (D23), and the platform plus the Android behaviors of the framework (M26). SPEC §24 is the design; each step adds a "Decided in step" paragraph to the section it settles, as the web's steps did in §23.

### 25.1 Build tags and the Android check
**Goal:** the module builds and vets for `android/arm64` and `android/amd64`, an Android build has none of the desktop backends, and nothing changes on Linux or the web.
**Read:** SPEC §24 (the introduction, "Build tags"), §6.3 (rules 2, 7 and 9), §24.13 (the first bullet), D21.
**Files:** the build tags of `internal/platform/linux/...` and `internal/engine/backends_linux.go`; `internal/engine/backends_android.go`; `foundation/host_android.go`, `foundation/web_other.go`; `scripts/check-android.sh`; `scripts/fetch-reference.sh`; `cmd/glutter-gen/internal/errcheck` (and `apicheck` if it reads the module per OS).
**Tasks:**
1. `linux && !android` on every file that only the Linux desktop has: `internal/platform/linux/...` (Wayland, X11, xkb, the IME, fontconfig; `dl` keeps `linux` if `gles` and `vulkan` use it) and `internal/engine/backends_linux.go` with its tests. Each package whose files are then all tagged keeps its untagged `doc.go`. The purego files of `gles` and `vulkan` keep `linux`. The pure-Go packages (`internal/dbus`, `desktop`) get no OS tag (SPEC §6.3 rule 2); if an Android build reaches `internal/platform/linux` through another import, the import is cut where it starts.
2. `internal/engine/backends_android.go`: `Run` fails with `BackendUnavailable` for now (step 25.3 adds the platform), no renderer candidates, no system font matcher (step 25.10).
3. `foundation/host_android.go`: the host platform is `TargetPlatformAndroid`; `web_other.go` becomes `!(js && wasm) && !android`.
4. `scripts/check-android.sh`: finds the NDK (`ANDROID_NDK_HOME`, else the newest `$ANDROID_HOME/ndk/*`) and runs `go build ./...` and `go vet ./...` for `android/arm64` and `android/amd64` with `CGO_ENABLED=1` and the NDK's clang for API 26 (`aarch64-linux-android26-clang`, `x86_64-linux-android26-clang`). Without an NDK it fails and prints the install command. It is a standard check from this step on (rule 8, CLAUDE.md).
5. `glutter-gen errcheck`: an `android/arm64` pass with `CGO_ENABLED=1` and the NDK's clang, so that the `*_android.go` files are checked; without an NDK it fails with `ErrCheckAndroidNDKNotFound` and the install command. The same for `apicheck` if it reads the module per OS.
6. `scripts/fetch-reference.sh`: add `engine/src/flutter/shell/platform/android` (the Java embedding and its tests) and `engine/src/flutter/impeller/renderer/backend/vulkan` (the driver deny list and the swapchain), at the pinned commit, and run it.
**Tests:** `go list -deps` of `examples/counter` for `GOOS=android` has no package of `internal/platform/linux` (it needs no NDK); the Linux and js lists are unchanged; the route test of `ErrCheckAndroidNDKNotFound`.
**Done when:** the default "Done when", `scripts/check-android.sh` passes, and the display tests of steps 1.10 and 7.4 still pass.

### 25.2 The Java embedding and `glutter-android`
**Goal:** `go run ./cmd/glutter-android run ./examples/dev/android-hello` builds an APK, installs it on the emulator and shows the program's log lines; `main` runs on its own thread and the activity finishes when it returns.
**Read:** SPEC §24.1, §24.2, §24.3 ("Two threads", "`main`"), §24.9 ("Directories", "Logging"), §24.13 ("Device tests"), D22, D24, CLAUDE.md ("Don't": processes).
**Port from:** `shell/platform/android/io/flutter/embedding/android/FlutterActivity.java` (`onCreate`, the launch flow) and `FlutterActivityAndFragmentDelegate.java` (the order of the setup), only the parts this step needs.
**Files:** `internal/androidtool/` (`sdk.go`, `manifest.go`, `apk.go`, `sign.go`, `java/io/github/rm4n0s/glutter/{GlutterActivity,Native}.java`, `classes.dex`), `cmd/glutter-android/`, `cmd/glutter-gen/` (`android-dex`), `internal/platform/android/` (`doc.go`, `jni_android.go`, `main_android.go`, `log_android.go`, `env_android.go`, and the plain-Go files), `internal/androidtest/`, `examples/dev/android-hello/`, `.gitignore` (`.android-test/`).
**Tasks:**
1. The Java classes `GlutterActivity` and `Native` (SPEC §24.2): `onCreate` copies the `GLUTTER_` extras and the app's directories into the environment (`Native.setenv`), loads `libapp.so` and calls `Native.start`; the activity finishes when Go reports that `main` ended. The other classes come in the steps that need them.
2. `glutter-gen android-dex` (SPEC §24.2) and a test that the checked-in `classes.dex` is up to date (skips without `javac`).
3. `internal/androidtool`: the SDK and NDK finder with its tags (SPEC §24.1), the generated manifest (with Flutter's `configChanges`: `orientation|keyboardHidden|keyboard|screenSize|smallestScreenSize|locale|layoutDirection|fontScale|screenLayout|density|uiMode`), the `go build -buildmode=c-shared` per ABI, `aapt2 link`, the dex and the libraries, `zipalign -P 16`, the debug key made with Go's `crypto`, `apksigner`.
4. `internal/platform/android`: `JNI_OnLoad`, `Native.start` running `main.main` on a goroutine locked to a new OS thread, the end of `main` (finish, exit code), standard output and error to `logcat` (SPEC §24.9), the directories and the extras.
5. `cmd/glutter-android` with `build`, `install` and `run` (SPEC §24.1); `-device` picks the device, else the only one connected, else an error that lists them.
6. `internal/androidtest`: the harness of SPEC §24.13: it creates `glutter-37` and `glutter-37-16k` in `.android-test/`, starts the emulator headless in its own process group under `timeout -k` with its own `adb` server port and `HOME`, waits for `sys.boot_completed`, installs, starts with extras, waits for the app to end, reads its `logcat` lines, and stops the emulator and the `adb` server. A test that nothing it started is still running.
7. `examples/dev/android-hello`: logs a line, the `GLUTTER_` variables it got and its directories, and exits with the code `GLUTTER_DEV_EXIT` names (debug builds only).
**Tests:** table tests of the finder (temporary SDK trees: none, no build-tools, no platform, several versions), the manifest (golden text), the command lines the builder runs (a fake runner), the zip layout (the libraries stored and 16 KB-aligned, the dex, the manifest), the key (made once, reused); route tests of every tag.
**Device tests:** on `glutter-37` and on `glutter-37-16k`: `android-hello` logs its line, sees an extra, has `HOME` in its files directory, exits with code 3 and its activity is gone; `llvm-readelf` shows 16 KB-aligned segments in both libraries.
**Proof:** none (nothing is drawn yet).

### 25.3 The Android platform: looper, surface, frames and `soft`
**Goal:** `examples/counter` shows its first frame on the emulator, drawn by `soft` into the window.
**Read:** SPEC §24.2 ("One view"), §24.3, §24.4 (`SurfaceKindAndroid`, `soft`), §24.13 (proof method E), §13, §9.6.
**Port from:** `FlutterView.java` and `FlutterSurfaceView.java` (the surface's life), `shell/platform/android/vsync_waiter_android.cc` (frames from `AChoreographer`).
**Files:** `internal/platform/android/` (`platform_android.go`, `loop_android.go`, `window_android.go`, `surface_android.go`, and plain-Go `metrics.go`, `frames.go`, `surfacestate.go`), `java/.../GlutterView.java`, `internal/gpu/surface.go` (`SurfaceKindAndroid`), `internal/gpu/nativewindow/`, `internal/engine/backends_android.go`, `internal/engine/screenshot.go`, `cmd/glutter-android` (`shot`).
**Tasks:**
1. The UI thread's `ALooper`, the `eventfd` behind `Post`, and the frames from `AChoreographer` (SPEC §24.3).
2. `GlutterView` (a `SurfaceView`) and the surface handshake: `surfaceCreated`, `surfaceChanged`, and `surfaceDestroyed` waiting for the renderer to let go (SPEC §24.3 "The one wait").
3. The platform: `Run`, `Post`, one window (`WindowUnsupported` for a second), `RequestFrame`, the size and the device pixel ratio, `Displays` (density and refresh rate), a first lifecycle (`Resumed` while the activity is resumed, `Hidden` while it is stopped; step 25.9 completes it), settings with their defaults (step 25.7), a clipboard that fails with `ClipboardUnavailable` (step 25.8), no services.
4. `internal/gpu/nativewindow` (SPEC §24.4 `soft`); the Android renderer list is `soft` for now, and `vulkan` and `gles` are `RendererUnavailable` until steps 25.4 and 25.5.
5. Self-capture on Android: the PNG goes to the app's files directory; `glutter-android shot` (proof method E, both modes).
**Tests:** the plain-Go parts: metrics, merging frame requests, the surface state machine (created, changed, destroyed while a frame is drawn, destroyed with no renderer) with fakes; route tests.
**Device tests:** the first frame of `examples/counter` equals its `soft` golden; turning the emulator (`adb emu rotate`) gives a frame of the new size; Home and back to the app draws again; the surface destroyed during a frame neither crashes nor blocks longer than the limit.
**Proof:** E, `25.3-android-counter-soft.png`.

### 25.4 GLES on Android
**Goal:** `gles` draws on Android through EGL, with the same shaders, at parity with `soft`.
**Read:** SPEC §24.4 (GLES, "A lost surface"), §9.1, §9.6.
**Files:** `internal/gpu/gles/egl.go` (`SurfaceKindAndroid`, the Android library names), `internal/engine/backends_android.go`, a parity test program for the device tests.
**Tasks:**
1. EGL on the `ANativeWindow`: `libEGL.so` and `libGLESv3.so`, the config, `eglCreateWindowSurface`; the context outlives the surfaces (SPEC §24.4). `EGLLoadFailed` and the other EGL tags as on Linux.
2. The Android renderer list becomes `gles`, `soft`.
3. A parity program that draws every golden scene with the chosen renderer and writes the images for the device test.
**Tests:** the choice of the EGL config (plain Go, as on Linux); the surface kinds.
**Device tests:** parity over every golden scene with `gles` on `glutter-37`, within the `gles` tolerance; Home and back keeps the textures (no new uploads of the atlases); `GLUTTER_RENDERER=gles` given as an extra is used.
**Proof:** E, `25.4-android-gallery-gles.png`: `examples/gallery` with `-renderer gles`.
**Manual checks:** ask the user to run `go run ./cmd/glutter-android run -env GLUTTER_RENDERER=gles ./examples/gallery` on their phone and to scroll and switch the theme. Record the answers and the renderer line of the log.

### 25.5 Vulkan on Android, and choosing the renderer
**Goal:** the explicit renderer draws on Android through Vulkan, `auto` chooses as SPEC §24.4 says, and the validation layer finds nothing.
**Read:** SPEC §24.4, §9.3, §9.4, §9.6, D23, §24.13 ("Vulkan validation").
**Port from:** `impeller/renderer/backend/vulkan/driver_info_vk.cc` (the deny list), the pre-rotation of `impeller/renderer/backend/vulkan/swapchain/` (or the files that hold it at the pinned commit).
**Files:** `internal/gpu/vulkan/surface.go`, `swapchain.go`, `instance.go`; `internal/gpu/explicit` (load and store operations); `internal/engine/backends_android.go`; `internal/androidtool` (`-vulkan-validation`).
**Tasks:**
1. `VK_KHR_android_surface` and `libvulkan.so`; the swapchain's `preTransform` from `currentTransform`, with the frame drawn rotated (SPEC §24.4).
2. The render passes of `explicit` checked for tile-based GPUs (SPEC §24.4), on every platform: the goldens and the Linux parity must not change.
3. `auto` on Android: API 29, Vulkan 1.1, the deny list (ported as a table, tested), the first frame presented; the list becomes `vulkan`, `gles`, `soft`.
4. `glutter-android build -vulkan-validation <dir>` packs the layer for the device tests (SPEC §24.13).
**Tests:** the deny list (table: driver, version, API level → allowed); the choice with fake candidates on Android; the pre-rotation transform for the four rotations; route tests.
**Device tests:** parity over every golden scene with `vulkan` and validation on, in portrait and landscape; `auto` on the emulator logs its choice and why; `GLUTTER_RENDERER=vulkan` on an emulator started without Vulkan (`-feature -Vulkan`) fails with its tag and `auto` uses `gles`.
**Proof:** E, `25.5-android-gallery-vulkan.png`, and with `-screen` in landscape, `25.5-android-gallery-landscape.png`.
**Manual checks:** ask the user to run `go run ./cmd/glutter-android run ./examples/gallery` on their phone, read the renderer and device from the log, turn the phone, and scroll. Record the answers.

### 25.6 Touch, mouse, stylus and keys
**Goal:** input on Android as Flutter's embedding delivers it.
**Read:** SPEC §24.5, §24.3 ("Events"), §24.13 ("Debug-only scripted events").
**Port from:** `AndroidTouchProcessor.java`, `KeyboardManager.java`, `KeyEmbedderResponder.java`, `KeyboardMap.java` (generated in Flutter from `android_keyboard_map_java.tmpl`), `MouseCursorPlugin.java`; the Android names in `dev/tools/gen_keycodes/data`.
**Tests:** `AndroidTouchProcessorTest.java`, `KeyboardManagerTest.java` and `KeyEmbedderResponderTest.java` (the relevant cases), as table tests of the plain-Go conversions.
**Files:** `internal/platform/android/` (`pointer.go`, `keyboard.go`, `keymap_gen.go`, `cursor.go`, and the `_android.go` glue), `java/.../GlutterView.java`, `cmd/glutter-gen/internal/keys` (the Android tables).
**Tasks:**
1. Pointers, hover, scroll and the mouse buttons (SPEC §24.5), with the historical samples; the Java side copies the values out of the `MotionEvent`.
2. Keys with the generated tables, the synthesized modifier and lock events, and the dispatch again of what the framework did not handle (SPEC §24.5).
3. Mouse cursors through `PointerIcon`.
4. `GLUTTER_DEV_EVENTS` for Android (SPEC §24.13) with `tap`, `swipe`, `key` and `wait`; the later steps add theirs.
**Device tests:** three `adb shell input tap` on the counter's button show 3; a swipe scrolls `examples/scrolling`; a key that a `Shortcuts` widget handles is taken, and Back, which no widget handles, reaches the activity.
**Proof:** E, `25.6-android-scrolling.png`: `examples/scrolling` after a scripted swipe.
**Manual checks:** ask the user to try on their phone: a fling in `examples/scrolling`, a two-finger gesture in the gallery, and, if they have one, a mouse or a stylus. Record the answers.

### 25.7 Insets, system UI, feedback and settings
**Goal:** the app draws edge to edge with the right insets, controls the system bars and the orientation, gives haptic feedback, and follows the system settings.
**Read:** SPEC §24.7, §24.13 (`glutter-26`).
**Port from:** `PlatformPlugin.java` (system chrome, haptics, sounds), `SettingsChannel.java` and its callers, the viewport metrics of `FlutterView.java` (`onApplyWindowInsets`); the framework side: `services/system_chrome.dart`, `services/haptic_feedback.dart`, and the system UI style of `rendering/view.dart` (`_updateSystemChrome`) where it is not ported yet.
**Tests:** `services/system_chrome_test.dart`, `services/haptic_feedback_test.dart`, and the system chrome cases of `rendering/view_test.dart` and `material/app_bar_test.dart` (the relevant cases).
**Files:** `services/system_chrome.go`, `services/haptic_feedback.go`, `widgets/binding.go` (the hooks), `rendering/view.go`, `ui` (the host interfaces), `internal/engine`, `internal/platform/android/` (`insets.go`, `settings.go`, `systemui.go` and their glue), `java/...`.
**Tasks:**
1. Edge to edge, and `ViewPadding`, `ViewInsets` (the keyboard, completed in step 25.8) and `SystemGestureInsets` from the insets, with the calls below API 30.
2. The framework's `SystemChrome` (SPEC §24.7), reached through a host interface of `ui.PlatformDispatcher` as the other platform services are; the overlay style of the `AnnotatedRegion` sent after each frame. On Linux and the web the calls do nothing.
3. `HapticFeedback` and `SystemSound` on Android; nothing elsewhere.
4. The settings (SPEC §24.7) and their changes; `Displays` from `DisplayManager`; `KeepScreenOn`; fullscreen as immersive mode; the requested orientation.
5. The harness creates `glutter-26`; the device tests of this step run on it too.
**Device tests:** on `glutter-37` and `glutter-26`: the app bar of the gallery sits below the status bar (`MediaQuery.padding.top` equals the status bar height `dumpsys window` reports); `cmd uimode night yes` makes the theme dark; `settings put system font_scale 1.5` scales the text; `settings put global animator_duration_scale 0` turns animations off; `SetPreferredOrientations` landscape turns the activity; immersive mode hides the bars.
**Proof:** E with `-screen`, `25.7-android-gallery-dark-insets.png`: the dark gallery under the status bar, with light status bar icons.
**Manual checks:** on the user's phone (with a display cutout and gesture navigation, if it has them): the app bar below the cutout, nothing under the gesture bar, a haptic tick from a long press. Record the answers.

### 25.8 Text input, the soft keyboard and the clipboard
**Goal:** text fields work with the soft keyboard and IMEs as in a Flutter app, the `Scaffold` moves with the keyboard, and copy and paste work with other apps.
**Read:** SPEC §24.6, §23.5 ("Decided in step 21.3", the contract), §24.3.
**Port from:** `TextInputPlugin.java`, `InputConnectionAdaptor.java`, `ListenableEditingState.java`, `ImeSyncDeferringInsetsCallback.java`, the clipboard of `PlatformPlugin.java`.
**Tests:** `InputConnectionAdaptorTest.java`, `ListenableEditingStateTest.java` and `TextInputPluginTest.java` (the relevant cases, as Go tables of the plain-Go model and as device tests where they need the Java class).
**Files:** `java/.../GlutterInputConnection.java`, `GlutterView.java`; `internal/platform/android/` (`textinput.go`, `clipboard.go` and their glue); `ui` (additions to the contract), `services/text_input.go` if the contract grows.
**Tasks:**
1. The input connection and its copy of the editing state (SPEC §24.6); what the contract needs beyond Linux and the web, added without changing the old fields.
2. The soft keyboard, `EditorInfo` from the configuration, the editor action.
3. The keyboard's insets and their animation (API 30 and later).
4. The clipboard (SPEC §24.6).
5. `GLUTTER_DEV_EVENTS`: `text:` and `compose:` through the input connection.
**Device tests:** scripted `text:` types into a `TextField` of `examples/editor`; `compose:` commits its text; the action of the keyboard submits a single-line field; a password field asks for a password keyboard (`dumpsys input_method` shows the `inputType`); copy in the app and `cmd clipboard` (or a second app) read the text, and paste brings it back; the bottom `ViewInsets` equals the keyboard's height.
**Proof:** E with `-screen`, `25.8-android-editor-keyboard.png`: `examples/editor` with typed text and the soft keyboard up, the editor above it.
**Manual checks:** ask the user to try on their phone with Gboard (or their keyboard): typing with autocorrect, swipe typing, a Japanese or Chinese input method, an emoji from the keyboard, a password field, a multiline field, and copy and paste with another app. Record the answers.

### 25.9 Lifecycle, back navigation and exit
**Goal:** the app's lifecycle, the back button and gesture (with predictive back) and exiting work as in a Flutter app.
**Read:** SPEC §24.8, §24.2 ("`main`"), §13.
**Port from:** `LifecycleChannel.java`, the lifecycle and back parts of `FlutterActivityAndFragmentDelegate.java` and `FlutterActivity.java` (`OnBackInvokedCallback`, `OnBackAnimationCallback`), `BackGestureChannel.java`; on the framework side, `services/predictive_back_event.dart` and the back gesture of `widgets/binding.dart` where they are not ported yet.
**Tests:** `LifecycleChannelTest.java` and the back cases of `FlutterActivityTest.java` (as tables of the plain-Go state machine), and the predictive back cases of `widgets/binding_test.dart` and `widgets/pop_scope_test.dart` that §12.10 left out (`setFrameworkHandlesBack`).
**Files:** `java/.../GlutterActivity.java`, `internal/platform/android/` (`lifecycle.go`, `back.go` and their glue), `internal/engine` (following the platform's lifecycle, `Paused`), `services/system_navigator.go`, `widgets/binding.go`.
**Tasks:**
1. The lifecycle of SPEC §24.8; the engine follows it, with `Paused`; no frames while paused or without a surface.
2. The back callbacks, `SetFrameworkHandlesBack`, and the predictive back events into the framework's back gesture handlers.
3. `SystemNavigatorPop`, `RequestAppExit`, the end of `main`, and a new activity in a living process (SPEC §24.2).
4. `onTrimMemory` (SPEC §24.8, §24.4).
5. `GLUTTER_DEV_EVENTS`: `back:`.
**Device tests:** Home makes the app `Hidden` then `Paused`, and coming back `Resumed`; Back pops a route of `examples/navigation`, and Back on the first route finishes the activity and the process exits; a `PopScope` that cannot pop keeps the activity; a scripted predictive back gesture gives the framework start, progress and commit (on `glutter-37`) and a plain pop (on `glutter-26`); `am kill` of a backgrounded app and a new start runs `main` again.
**Proof:** none (the transitions come in step 26.3).
**Manual checks:** ask the user to try on their phone: the back gesture in `examples/navigation` (with the system's predictive back animation on the first route), Home and back, and the recent-apps screen. Record the answers.

### 25.10 Fonts on Android
**Goal:** text uses the system's fonts and fallback as an Android app does, including CJK by locale and emoji on every supported version.
**Read:** SPEC §24.9 ("Fonts"), §10, the color glyph rows of §9 and §12.10.
**Port from:** Skia's `src/ports/SkFontMgr_android_parser.cpp` and `SkFontMgr_android.cpp`: add a sparse clone of `https://skia.googlesource.com/skia` at the revision Flutter's `DEPS` pins at the pinned commit to `scripts/fetch-reference.sh` (`reference/skia`), and record it in `reference/VERSION`.
**Files:** `internal/text/androidfonts/` (the parser and the matcher, plain Go), `internal/engine/backends_android.go` (`SystemFontMatcher`, the cache directory of `fontscan`), the glyph drawing of `internal/gpu` and `internal/text` (CBDT bitmaps), `scripts/fetch-reference.sh`.
**Tasks:**
1. The parser of `fonts.xml` (the formats of API 26 to 37) and of `fonts_customization.xml`: families, names, aliases, languages, variants, weights and styles.
2. The matcher (`text.Matcher`): a family by name or alias, and the fallback chain by language, as Skia's Android manager chooses.
3. `fontscan` with its cache directory in the app's cache directory.
4. CBDT color glyphs (go-text's bitmap glyph data), drawn scaled as Skia draws them, so the emoji font of Android 8 to 12 shows (SPEC §24.9). On Linux this also draws CBDT fonts that fontconfig picks.
**Tests:** table tests of the parser over fixture files written for the tests (the elements of each format version), and of the matcher's choices (`sans-serif`, `serif`, `monospace`, `zh-Hans`, `ja`, `ko`, an emoji, an unknown family); a golden of CBDT glyphs from a small test font (a subset of Noto Color Emoji, with its OFL license next to it; ask the user if a tool is needed to make the subset).
**Device tests:** on `glutter-37` and `glutter-26`: a text page with Latin, Greek, Arabic, Hebrew, Chinese, Japanese and emoji draws every character with a system font (no missing-glyph box) in both locales `en-US` and `ja-JP`.
**Proof:** E, `25.10-android-text.png` (on `glutter-37`), and `25.10-android-text-api26.png` (on `glutter-26`, CBDT emoji).

### 25.11 TalkBack
**Goal:** apps are usable with TalkBack.
**Read:** SPEC §24.10, §8.10 (the bridge on Linux), §7 (the semantics updates of `ui`).
**Port from:** `AccessibilityBridge.java` (and what it reads of the semantics updates).
**Tests:** `AccessibilityBridgeTest.java` (the relevant cases, as tables of the plain-Go parts and as device tests).
**Files:** `java/.../GlutterAccessibility.java`, `GlutterView.java`; `internal/platform/android/` (`accessibility.go` and its glue); `internal/engine` (the Android bridge as `Services().Accessibility()`).
**Tasks:**
1. The bridge (SPEC §24.10): the screen reader's state, the copy of the tree on the main thread, the node infos, the actions, the events.
2. `GLUTTER_DEV_EVENTS`: `a11y:` (an action on the node with a label).
**Device tests:** with the bridge forced on (a debug extra; the `google_apis` images have no TalkBack), `uiautomator dump` lists the gallery's buttons, texts and text fields with their labels and bounds; a scripted `a11y:click` on the counter's button increments it; a scroll action scrolls a list; a live region's change is announced (`logcat` of the event).
**Proof:** none.
**Manual checks:** ask the user to turn on TalkBack on their phone and try the gallery: explore by touch, swipe to the next element, double-tap a button, scroll a list, type in a text field. Record the answers.

### 25.12 Services on Android
**Goal:** opening URIs and notifications work on Android; the other services say they are unavailable.
**Read:** SPEC §24.11, §23.7 ("Decided in step 21.6", how the web plugs into `desktop`), §8.9.
**Files:** `internal/platform/android/` (`desktop.go` plain Go, and its glue), `java/...` (intents, the notification channel, the permission request, the pending intents of the actions), `desktop/` (the Android services, as the web's are plugged in), `examples/dev/android-services/`.
**Tasks:**
1. `desktop.OpenURI` through `ACTION_VIEW` (SPEC §24.11).
2. `desktop.ShowNotification` and `CloseNotification`: the channels, the permission, the actions, the closes.
3. `TrayUnavailable` and `FileChooserUnavailable`; D-Bus is never tried.
4. `examples/dev/android-services`: buttons that open a URI and show a notification with an action.
**Tests:** the plain-Go parts over fakes (validation, the notification's fields, the permission states, replace by ID); route tests of every tag.
**Device tests:** `OpenURI` of an `https:` URI starts an activity for it (`dumpsys activity` shows the intent); a notification appears in `dumpsys notification` with its title, body and action; with the permission revoked (`pm revoke`) and denied, `NotificationsUnavailable`.
**Proof:** E with `-screen`, `25.12-android-notification.png`: the notification shade with the example's notification and its action button.
**Manual checks:** ask the user to try `examples/dev/android-services` on their phone: the link opens the browser, the permission prompt appears once, the action button calls the app. Record the answers.

---

## M26: Android behaviors of the framework, and 1.6

These steps port what earlier steps left out because only Android (or a touch screen) uses it (SPEC §24.12). They are framework steps: their tests run with `gluttertest` and `TargetPlatformAndroid`, and their proofs come from the emulator.

### 26.1 Overscroll indicators
**Goal:** scrolling past the end stretches the content on Android, as Material 3 does in Flutter.
**Read:** SPEC §24.12, §12.10 (the rows of the overscroll indicators).
**Port from:** `widgets/overscroll_indicator.dart`, the `buildOverscrollIndicator` of `widgets/scroll_configuration.dart` and of `material/app.dart` (`MaterialScrollBehavior`).
**Tests:** `widgets/overscroll_indicator_test.dart`, `widgets/overscroll_stretch_indicator_test.dart`, and the overscroll cases of `widgets/scroll_behavior_test.dart` and `material/app_test.dart` (including the case that was skipped).
**Files:** `widgets/overscroll_indicator.go`, `widgets/scroll_configuration.go`, `material/scroll_behavior.go` (or where `MaterialScrollBehavior` lives), the `material` re-exports.
**Tasks:**
1. `GlowingOverscrollIndicator`, `StretchingOverscrollIndicator` and `OverscrollIndicatorNotification`, as at the pinned commit. If the stretch needs a shader image filter that `ui` lacks, port the form without it and record the deviation.
2. The scroll behaviors build them on Android and Fuchsia, as Flutter does; the §12.10 rows that said they are not ported are updated.
**Proof:** E, `26.1-android-stretch.png`: `examples/scrolling` held past its top with a scripted swipe (`GLUTTER_SCREENSHOT_AFTER`).

### 26.2 The Material text selection toolbar and the magnifier
**Goal:** selecting text on Android shows Material's toolbar, handles and magnifier.
**Read:** SPEC §24.12, §12.10 (the `TextSelectionThemeData` row and its deviation).
**Port from:** `material/text_selection_toolbar.dart`, `material/text_selection_toolbar_text_button.dart`, the Android branch of `material/adaptive_text_selection_toolbar.dart`, `material/magnifier.dart`, `widgets/magnifier.dart`, and the magnifier parts of `widgets/text_selection.dart` and `widgets/editable_text.dart`.
**Tests:** `material/text_selection_toolbar_test.dart`, `material/text_selection_toolbar_text_button_test.dart`, the Android cases of `material/adaptive_text_selection_toolbar_test.dart` and `material/text_field_test.dart` (toolbar and handles), `material/magnifier_test.dart`, `widgets/magnifier_test.dart`.
**Files:** `material/text_selection_toolbar.go`, `material/text_selection_toolbar_text_button.go`, `material/adaptive_text_selection_toolbar.go`, `material/magnifier.go`, `widgets/magnifier.go`, `widgets/text_selection.go`, the re-exports.
**Tasks:**
1. The toolbar with its overflow menu and its buttons; the adaptive toolbar's Android branch; the §12.10 deviation ("the toolbar of every platform is the desktop one") is narrowed to iOS and macOS.
2. `RawMagnifier`, `MagnifierDecoration`, `TextMagnifier` and their use while a handle is dragged.
**Proof:** E with `-screen`, `26.2-android-selection-toolbar.png` (a long-pressed word in `examples/editor`, with its handles and toolbar) and `26.2-android-magnifier.png` (a handle being dragged).

### 26.3 Predictive back transitions
**Goal:** a route follows the back gesture on Android as in Flutter.
**Read:** SPEC §24.8 ("Back"), §24.12, §12.10 (the `PageTransitionsTheme` row).
**Port from:** `material/predictive_back_page_transitions_builder.dart`, the Android defaults of `material/page_transitions_theme.dart` at the pinned commit.
**Tests:** `material/predictive_back_page_transitions_builder_test.dart`, the Android cases of `material/page_transitions_theme_test.dart`.
**Files:** `material/predictive_back_page_transitions_builder.go`, `material/page_transitions_theme.go`, `gluttertest` (a back gesture: start, progress, commit, cancel), the re-exports.
**Tasks:**
1. `PredictiveBackPageTransitionsBuilder` and the Android default of `PageTransitionsTheme`; the §12.10 row is updated.
2. `gluttertest` drives a back gesture through the same framework entry points the platform calls (step 25.9).
**Proof:** E, `26.3-android-predictive-back.png`: `examples/navigation` halfway through a scripted back gesture.
**Manual checks:** ask the user to try the back gesture in `examples/navigation` on their phone and compare with a Flutter or system app. Record the answers.

### 26.4 Spell check
**Goal:** misspelled words in text fields are marked on Android and the suggestions toolbar replaces them, as in Flutter.
**Read:** SPEC §24.6 ("Spell check"), §24.12.
**Port from:** `services/spell_check.dart`, `widgets/spell_check.dart`, `material/spell_check_suggestions_toolbar.dart`, `material/spell_check_suggestions_toolbar_layout_delegate.dart`, the spell check parts of `widgets/editable_text.dart`; `SpellCheckPlugin.java` for the platform side.
**Tests:** `services/spell_check_test.dart`, `widgets/spell_check_test.dart`, `material/spell_check_suggestions_toolbar_test.dart`, the spell check cases of `widgets/editable_text_test.dart`.
**Files:** `services/spell_check.go`, `widgets/spell_check.go`, `material/spell_check_suggestions_toolbar.go`, `widgets/editable_text.go`, `internal/platform/android/` (`spellcheck.go` and its glue), `java/...`.
**Tasks:**
1. The framework's spell check (`SpellCheckConfiguration`, `SpellCheckService`, the suggestion spans, the toolbar).
2. The Android service through `TextServicesManager` (SPEC §24.6). Linux and the web have none, so spell check stays off there, as in Flutter.
**Device tests:** a misspelled word typed with `text:` is underlined (the framework's spell check results, read in the test program).
**Proof:** E with `-screen`, `26.4-android-spell-check.png`: a misspelled word underlined, with the suggestions toolbar open.

### 26.5 The examples on Android, size and performance
**Goal:** every example except `multiwindow` and `tray` runs on Android, and the Android targets of SPEC §15 are set and met.
**Read:** SPEC §15, §19, §24.1, §24.13.
**Files:** `examples/examples_test.go`, the examples where Android needs a change, `cmd/glutter-android` (`perf`), `scripts/perf-android.sh`, `docs/PERFORMANCE.md`, `README.md`, `docs/GUIDE.md`.
**Tasks:**
1. `examples/examples_test.go` builds every example except `multiwindow` and `tray` for `android/arm64` (skipped without an NDK, and under `-short`); a device test draws the first frame of each.
2. `glutter-android perf` and `scripts/perf-android.sh`: cold start to the first frame (`am start -W` and the engine's own timing), idle CPU, frame times with `GLUTTER_FRAME_STATS=1`, memory, and the APK size per ABI, with release builds, on the emulator and on the owner's phone.
3. Propose the Android targets for §15 from the measurements and Flutter's numbers on the same phone if the owner can give them; profile and fix until they are met, or the owner approves the exceptions.
4. `docs/PERFORMANCE.md`: the Android table. `README.md`: an "Android" section (what to install, building, running, what Android lacks). `docs/GUIDE.md`: notes for Flutter Android developers (the APK tool, intent extras, files and logs, no plugins).
**Manual checks:** ask the user to run `examples/gallery` (release build) on their phone and judge scrolling, the theme switch and a page change. Record the answers.
**Done when:** the default "Done when", the Android targets are in §15 and met or approved, and the manual checks are recorded.

### 26.6 Release 1.6
**Tasks:** confirm and record in `IMPLEMENTED.md`:
- the standard checks, the display tests, the browser tests and the device tests
- the manual checks of 25.4–25.12 and 26.3–26.5
- the performance targets (26.5)
- the API check and the error rules check

Then update the SPEC's version paragraph and the README, and **ask the user** before tagging `v1.6.0` and pushing the tag.

## M27: Fixes for Emetos (1.7)

These steps come from the issues that the Emetos code editor (built on Glutter, see M22) found and worked around. Each issue names the Emetos workaround, which is a `// Glutter:` comment in Emetos; the steps here only change Glutter, and never read the Emetos sources (they are outside this project folder). 27.1 and 27.2 are bugs (Glutter differs from Flutter and something breaks), 27.3–27.5 are gaps of the desktop windows, 27.6–27.9 are additions to Flutter's API, each with a `// Deviation:` comment and a row in SPEC §12.10, and 27.10 is the cold start. 27.11 releases them.

### 27.1 Scroll positions after their Scrollable is gone
**Goal:** a `Scrollable` that leaves the tree in the frame after its metrics changed no longer makes the program panic.
**Read:** SPEC §11.1 (scrolling), §13 (`Post`), §14.2.
**Port from:** `packages/flutter/lib/src/widgets/scroll_position.dart` (`didUpdateScrollMetrics`, `dispose`, the `scheduleMicrotask` calls), `scrollable.dart` (`notificationContext`, `storageContext`).
**Files:** `widgets/scroll_position.go`, `widgets/scroll_position_with_single_context.go`, `widgets/scrollable.go`, `internal/engine/` (the test).
**Tasks:**
1. `ScrollPositionBase.DidUpdateScrollMetrics` panics in debug builds when the position was disposed before the function it posted ran: it reaches the binding through `widgetsBinding()`, which calls `StorageContext()`, the context of an unmounted state. Flutter asserts on the global scheduler and never touches the context there. Keep a `disposed` flag (set by `Dispose`); a function that a position posted does nothing once it is set. The debug checks read the binding the position kept when it was attached (no context lookup).
2. Look at every other function a position, an activity or a `ScrollableState` posts or schedules (`post`, timers, `AddPostFrameCallback`), and at every use of `StorageContext()` or `Context()` that can run after `Dispose`; give each the same guard.
**Tests:** `DidUpdateScrollMetrics` after `Dispose` does nothing (table: each posted function); a widget test where a `TextField` inside a `Scrollable` is removed in the frame after its metrics changed. `gluttertest` runs the posted functions inside `Pump`, in the same frame, which hides the bug, so also add a test that runs the real loop (the engine over the fake platform): a window whose width drops under the size where a `Scrollable` is taken out of the tree, then one more frame, and no panic.
**Proof:** none.

### 27.2 `gluttertest`: unmount the tree at the end of a test, and a fake desktop without a file chooser
**Goal:** the states of a widget test are disposed when the test ends, and a test can see the failure of a desktop that has no file chooser portal.
**Read:** SPEC §17, §12.10 (the `gluttertest` rows), §8.9 (file dialogs).
**Port from:** flutter_test's `TestWidgetsFlutterBinding._runTestBody` (not in `reference/`): after the test body, when no error is pending, it runs `runApp(Container(key: UniqueKey()))` and pumps, so every state is disposed.
**Files:** `gluttertest/tester.go`, `gluttertest/desktop.go`, `gluttertest/doc.go`, `desktop/` (only if the tag is not reachable).
**Tasks:**
1. The `Tester`'s cleanup pumps an empty widget before it cancels the binding's context, unless the test failed already (as flutter_test does), so `State.Dispose` runs and every `State.Lifetime()` is cancelled. An error that the unmounting reports fails the test like any other. Cleanups run last in, first out, so a cleanup that the test registered after `New` still sees the tree.
2. Fix the tests of this module that break because their states now get disposed (each one is a real disposal bug, or a test that relied on the tree outliving it); never weaken a test.
3. `FakeDesktop.DisableFileChooser()`: the file chooser calls then fail with the tag the real desktop gives when there is no portal (`FileChooserUnavailable`), until the test ends.
4. Measure `go test -race ./...` of the module before and after, and say both in `IMPLEMENTED.md`.
**Tests:** a state's `Dispose` runs and its lifetime is cancelled at the end of a test (a `t.Run` subtest that checks afterwards); a test that already failed is not unmounted; an error while unmounting is reported; `DisableFileChooser` with open, save and directory dialogs (route tests of the failure).
**Proof:** none.

### 27.3 The focus of a window
**Goal:** an app knows whether each window has the focus from the moment it is created, also on a compositor whose seat has no keyboard.
**Read:** SPEC §8.2, §8.3, §8.7, §12.10 (the `ViewFocusEvent` row), §13, §15 (idle and unfocused).
**Port from:** Flutter's GTK embedder follows `GtkWindow`'s `is-active` (the compositor's activation) for the view focus (`shell/platform/linux/fl_view.cc`, not in `reference/`; SPEC §12.10 says what is done here).
**Files:** `internal/platform/linux/wayland/{window,decoration,keyboard}.go`, `internal/platform/linux/wayland/decor/`, `internal/platform/linux/wayland/fakecomp/`, `internal/platform/linux/x11/`, `internal/engine/engine.go`, `internal/platform/fake/`, `gluttertest/`.
**Tasks:**
1. Wayland: the view focus follows the toplevel's `activated` state (the states of `xdg_toplevel.configure`, and libdecor's `LIBDECOR_WINDOW_STATE_ACTIVE`), not `wl_keyboard.enter` and `leave`, which keep routing the keys. A seat without a keyboard (the headless sway of the tests) then has focused windows too.
2. A new window gets a `ViewFocusEvent` as soon as its focus is known (Wayland: its first configure; X11: its first focus event, or the active window of `_NET_ACTIVE_WINDOW` when it is mapped), so `DidChangeViewFocus` is called once for it; until then it counts as focused, as before.
3. Update the SPEC §12.10 row and §8.2/§8.3, and the idle-and-unfocused measurement of `scripts/perf.sh` (it can now run headless: a second window takes the focus).
**Tests:** Wayland against the fake compositor (activated with and without a keyboard, with libdecor, the first configure sends the event, keys still follow `wl_keyboard`); X11 on its fake or under Xvfb; the engine queues the initial event with the first frame's input.
**Display tests:** headless sway (no keyboard): two windows, the focus moves to the one the compositor activates; X11 under Xvfb with the test window manager.
**Proof:** none. A manual check on a real desktop: two windows of `examples/multiwindow`, the focus follows the clicks and `Alt+Tab`.

### 27.4 What a window controller can do, and the first window in tests
**Goal:** the title, activation and full screen of any window, the first one included, are reachable without an interface assertion, a window can open maximized or full screen, and a test sees the first window like the others.
**Read:** SPEC §8.7, §12.10 (the window controller rows), §17.
**Port from:** `packages/flutter/lib/src/widgets/_window.dart` (`BaseWindowController`, `RegularWindowController`, `DialogWindowController`).
**Files:** `widgets/window_controller.go`, `widgets/window_scope.go`, `ui/window_control.go`, `internal/platform/` (initial state), `internal/engine/`, `gluttertest/windows.go`, `docs/GUIDE.md`.
**Tasks:**
1. An exported interface, beside `BaseWindowController`, with what both regular and dialog windows can do: `Title`, `SetTitle`, `Activate`, `IsActivated`, `SetMinimized`, `IsMinimized`, and for regular windows `SetMaximized`, `IsMaximized`, `SetFullscreen`, `IsFullscreen` (in a second interface or with a "can" query; decide and write it in SPEC §12.10). `WindowScopeOf` stays Flutter's (`BaseWindowController`); a new `WindowScopeControllerOf`-style accessor, or the interface on the value it returns, reaches the rest. `// Flutter: no counterpart`, with a `// Deviation:`.
2. `RegularWindowControllerOpts.State` (normal, maximized, full screen; the zero value is normal), as `WindowConfig.State` of the first window: the platform creates the window in that state, so it never shows at its size first. No position (Wayland gives clients none).
3. `Tester.Windows()` lists the implicit view too (its title, size and state), or the `Tester` gets `ViewTitle(view)`; choose the one that keeps the existing tests meaningful, and say which in SPEC §12.10.
**Tests:** table tests over the regular, dialog and implicit controllers (each method, through `WindowScope`); a window created maximized and full screen on the fake platform; Wayland and X11 send the state before the first commit or map (fake compositor, X11 fake); the implicit view's title in `Tester`.
**Display tests:** headless sway and Xvfb: a window opened maximized has the output's size in its first frame.
**Proof:** none.

### 27.5 Closing and destroying the first window
**Goal:** an app with several windows handles the close request of the first window like any other window's, and can destroy the first window while others stay.
**Read:** SPEC §8.7 ("Closing"), §12.10 (the rows on `WindowingOwner.ImplicitWindowController`, `WindowScope` and closing), §13 (the lifecycle and `RunApp`).
**Port from:** `packages/flutter/lib/src/widgets/_window.dart` (`RegularWindowControllerDelegate.onWindowCloseRequested`, `WindowingOwner`), `widgets/app_lifecycle_listener.dart`.
**Files:** `widgets/window_controller.go`, `widgets/binding.go`, `widgets/run_app.go`, `widgets/app_lifecycle_listener.go`, `internal/engine/`, `gluttertest/windows.go`, `examples/multiwindow/`, `docs/GUIDE.md`.
**Tasks:**
1. The close request of the implicit window goes to a delegate the app can give it (`WindowingOwner.SetImplicitWindowDelegate`, or a `RunAppOpts` field; decide and record it), with the view; without one it stays an exit request, as now, so a single-window app does not change.
2. `ImplicitWindowController().Destroy()` when other windows exist removes the implicit view: the `View` that `RunApp` wrapped the app in is taken out of the tree (its states are disposed) and the platform window is closed. `RunApp` ends when the last window is destroyed, as for the other windows. Destroying it when it is the only window is an exit, as closing it is.
3. `examples/multiwindow`: closing the first window while a second one is open keeps the app running.
**Tests:** the close request of the first window reaches the delegate with its view, and an exit request without one; destroying the first window with others open (its states disposed, the view gone from the dispatcher, frames go on for the others, `RunApp` goes on, then ends with the last); destroying it alone; the multiwindow example's test.
**Display tests:** headless sway and Xvfb: two windows, the first is closed, the second keeps drawing and receiving input.
**Proof:** none.

### 27.6 `RawTooltip`: a delay read on enter, show and hide callbacks, a controller, dismissal by a key
**Goal:** a hover can follow the rules of an editor (shown at once when another hover hid less than a moment ago, shown for the item that got the keyboard focus, hidden by Escape) with `RawTooltip` alone.
**Read:** SPEC §11.1 and §11.2 (tooltips), §12.10 (the `RawTooltip` row).
**Port from:** `packages/flutter/lib/src/widgets/raw_tooltip.dart` (Glutter's port is `widgets/raw_tooltip.go`).
**Files:** `widgets/raw_tooltip.go`, `material/tooltip.go` (pass the options through when it builds a `RawTooltip`), `material/widgets_reexport_gen.go` (regenerated).
**Tasks:**
1. `RawTooltipOpts.HoverDelayFunc func() time.Duration`, read when the pointer enters; it wins over `HoverDelay`.
2. `OnShown` and `OnHidden` callbacks, called when the overlay's animation starts showing and when it is dismissed (whatever hid it: exit, tap, `RawTooltipDismissAll`, the controller, the key).
3. `RawTooltipController` (`Show`, `Hide`, `IsShowing`), given in the options; it works on the state that it is attached to and does nothing when none is.
4. `DismissOnKey bool`: a key down while the tooltip shows hides it, and the key is not handled (a `HardwareKeyboard` handler that returns false), as Flutter's `Tooltip` does for Escape on desktop.
5. Every addition gets `// Deviation:` and a row in SPEC §12.10.
**Tests:** table tests for each option (the delay function read at each enter, the callbacks for every way of hiding, the controller on a mounted and an unmounted tooltip, a key hides it and still reaches the focused widget); the Flutter test cases of `raw_tooltip_test.dart` that the port has still pass.
**Proof:** none.

### 27.7 The cursor of a pressed pointer
**Goal:** while a button is held after a press in a region, the cursor stays the one of that region, whatever the pointer moves over, until the button is released.
**Read:** SPEC §8.5 (cursors), §11.1 (`MouseRegion`), §12.10.
**Port from:** `packages/flutter/lib/src/rendering/mouse_tracker.dart`, `services/mouse_cursor.dart` (`MouseCursorManager.handleDeviceCursorUpdate`), `rendering/proxy_box.dart` (`RenderMouseRegion`).
**Files:** `widgets/pointer_widgets.go`, `rendering/mouse_tracker.go`, `rendering/proxy_box*.go` (`RenderMouseRegion`), `services/mouse_cursor.go`, `material/widgets_reexport_gen.go`.
**Tasks:**
1. `MouseRegionOpts.HoldCursorWhilePressed bool`: when a pointer goes down inside such a region, the mouse tracker keeps that region as the cursor's source for the device until the pointer goes up or is cancelled; the cursor changes if the region's own cursor changes meanwhile, and the region being removed ends the hold. The hover events (`OnEnter`, `OnExit`, `OnHover`) stay as they are.
2. `// Deviation:` and a row in SPEC §12.10.
**Tests:** table tests: press in a sash-like region and move over a region with its own cursor and over empty space, release; the region's cursor changing during the hold; the region removed during the hold; a touch pointer (no cursor); several mice.
**Proof:** none. A manual check: `examples/gallery`'s split pane (or a small `examples/dev` sash) keeps the resize cursor while the pointer outruns it.

### 27.8 `Draggable` and `DragTarget` for lists and tabs
**Goal:** a click on a draggable row stays a click, a running drag can be cancelled with Escape, and a target can scroll or expand while the pointer rests over it.
**Read:** SPEC §11.1 (drag and drop), §12.10 (the `Draggable` rows), §8.5 (input).
**Port from:** `packages/flutter/lib/src/widgets/drag_target.dart`, `gestures/multidrag.dart` (`computeHitSlop`), `rendering/proxy_box.dart` (`RenderMouseRegion.hitTest`, `RenderProxyBoxWithHitTestBehavior`).
**Files:** `widgets/drag_target.go`, `gestures/multidrag.go`, `rendering/`, `material/widgets_reexport_gen.go`.
**Tasks:**
1. `MultiDragGestureRecognizerOpts.Slop` and `DraggableOpts.Slop` (`foundation.Opt[float64]`): the distance before a drag starts, for every pointer kind; the default stays Flutter's (`computeHitSlop`: 1 px for a mouse).
2. `DraggableOpts.CancelOnEscape bool`: while its drag runs, Escape (a `HardwareKeyboard` handler) cancels it as a cancelled pointer would: `OnDraggableCanceled`, the targets' `OnLeave`, the avatar removed.
3. `DragTargetOpts.RepeatMove bool`: while a drag is over the target, `OnMove` is called once per frame even when the pointer does not move (a frame is scheduled for it only while such a drag is over such a target, so idle CPU stays 0 otherwise).
4. Check the report that the non-opaque `MouseRegion` of `ScrollableElement` makes the rows' hit test report "not hit" to what is above it, so a deferring `Listener` above misses the press. Flutter's `RenderMouseRegion` reports a hit when a child was hit. If Glutter differs, fix it as a bug with a test from `proxy_box_test.dart` or `mouse_region_test.dart`; if not, write a test that shows the case works.
5. `// Deviation:` and rows in SPEC §12.10 for 1–3.
**Tests:** table tests: a mouse moving 3 px with a slop of 4 px is a tap, 5 px is a drag; Escape during a drag and outside one; `RepeatMove` calls per frame and no frames when not over the target; the hit test of a `Listener` above a list.
**Proof:** none.

### 27.9 `TreeSliver`: changing children, lazy nodes, node identity
**Goal:** a tree can keep its nodes between changes: remove and replace children, show a chevron on a folder whose children are not read yet, and find nodes by a key.
**Read:** SPEC §11.3 (`TreeSliver`, step 22.7), §12.10 (the `TreeSliver` rows).
**Port from:** `packages/flutter/lib/src/widgets/sliver_tree.dart` (`TreeSliverNode.children` is a mutable list in Flutter; `toggleNode`, `getNodeFor`, `onNodeToggle`).
**Files:** `widgets/sliver_tree.go`, `material/widgets_reexport_gen.go`.
**Tasks:**
1. `TreeSliverNode.SetChildren`, `RemoveChild` and `InsertChild` (Flutter's children list is mutable, so these are a port, not a deviation): the parent and depth of the nodes are kept right, and the sliver is told to rebuild its active nodes, as Flutter's "modifying the children ... will cause the tree to be rebuilt".
2. `TreeSliverNodeOpts.HasChildren bool` (a lazy node): it can be toggled with no children; toggling it calls `OnNodeToggle`, where the app adds the children, and the node then shows them (with the animation of the sliver).
3. `TreeSliverOpts.NodeKey func(T) any` (comparable keys): `GetNodeFor`, `IsExpanded` and keeping the expansion when the tree is rebuilt with new nodes use it; without it, `reflect.DeepEqual` as now.
4. `// Deviation:` and rows in SPEC §12.10 for 2 and 3.
**Tests:** the cases of `sliver_tree_test.dart` that touch children and the controller, still passing; table tests for each new function (depth and parent after a move, removal of an expanded node, a lazy node toggled and filled, keys across a rebuild with new node values).
**Proof:** none.

### 27.10 Cold start: what the first draw costs, and a shader cache
**Goal:** the first draw is profiled part by part, a second start compiles no shader, and the cold start is closer to SPEC §15.
**Read:** SPEC §9.1, §9.4, §9.6, §15, `docs/PERFORMANCE.md`.
**Files:** `internal/engine/{engine,renderers}.go`, `internal/gpu/gles/device.go` (and a new cache file), `internal/gpu/vulkan/` (`VkPipelineCache`), `scripts/perf.sh`, `docs/PERFORMANCE.md`.
**Tasks:**
1. With `GLUTTER_FRAME_STATS=1`, the startup log splits "first draw" into each renderer tried (with its error and how long it took), the context or device, the shaders or pipelines, and the first frame.
2. GLES: a program cache with `glGetProgramBinary`/`glProgramBinary` (GLES 3.0) in `$XDG_CACHE_HOME/glutter/shaders/`, keyed by the hash of the sources and the driver (`GL_VENDOR`, `GL_RENDERER`, `GL_VERSION`); a binary the driver refuses is compiled again and replaced; a cache that cannot be written is logged once and ignored. Not on the web.
3. Vulkan: a `VkPipelineCache` saved in the same directory, checked by the header's vendor, device and UUID.
4. Compile only the programs the first frame needs before it; the others when first used (or in the idle time after the first frame), so a rare path does not delay the start.
5. Measure the cold start with `scripts/perf.sh` (warm and empty caches), and with the Vulkan try that fails on the headless compositor; record the numbers in `docs/PERFORMANCE.md`. If the failed Vulkan try costs a large part, say so and ask the user whether `auto` should try GLES first where Vulkan presents to no output.
**Tests:** the cache over the fake GL (hit, miss, a refused binary, a changed driver, an unwritable directory, route tests for new tags); lazy compilation (a program compiled once, at first use); the startup log's parts.
**Display tests:** headless sway: a second start finds every program in the cache (counted in the log).
**Proof:** none. A manual check: the cold start of `examples/counter` with the GPU, `scripts/perf.sh` in the user's session, recorded in `docs/PERFORMANCE.md`.

### 27.11 Release 1.7
**Tasks:** confirm and record in `IMPLEMENTED.md`:
- the standard checks, the display tests, the browser tests and the device tests
- the manual checks of 27.3, 27.7 and 27.10
- the API check and the error rules check

Then update the SPEC's version paragraph and the README, and **ask the user** before tagging `v1.7.0` and pushing the tag.
