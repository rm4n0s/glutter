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
- **The manager finishes the step**: the `IMPLEMENTED.md` entry (rule 10; it may have a worker draft it, and checks every claim in it), the commit, the merge into the local `main` and the tag (rules 11–12), and the push of a finished milestone (rule 12), the report (rule 13), and deletes the step's briefs.

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

**Processes you start.** Every compositor, browser, Wine instance or app you start for a test, a proof or a check is stopped when you are done with it: run it under `timeout -k`, in its own process group, and stop it from `t.Cleanup` or `defer` in Go code. Wine is started under `timeout -k` with `ulimit -c 1` (so that a crash never reaches the system's crash reporter, ABRT), on a headless Xvfb, and stopped with `wineserver -k` (never `wineserver -w`: `winedevice` stays alive); `scripts/wine-exec.sh` and `internal/winetest` do this. Before you finish a step, check with `pgrep -a` that nothing you started is still running (no `wine`, `wineserver`, `winedevice`, `Xvfb`, `explorer.exe`). The user works on this desktop and sees every stray window.

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
   GOOS=windows GOARCH=amd64 go build ./...   # from step 28.1 on
   GOOS=windows GOARCH=amd64 go vet ./...     # from step 28.1 on
   GOOS=windows GOARCH=arm64 go build ./...   # from step 28.1 on (build only: there is no arm64 Wine here)
   scripts/test-windows.sh               # from step 28.1 on: the unit tests of the packages that build for Windows, under Wine
   ```
   If the step lists **Display tests**, run them too: `go test -race -tags glutter_display <packages>`. They run under a headless compositor (step 1.5) unless the step says otherwise.
   If the step lists **Browser tests**, run them too: `go test -tags glutter_browser <packages>` (SPEC §17).
   If the step lists **Device tests**, run them too: `go test -tags glutter_android <packages>` (SPEC §24.13). They start the emulator themselves and stop it; run them one package at a time (`-p 1`), since one emulator needs 2–4 GB of memory.
   If the step lists **Wine tests**, run them too (SPEC §25.13). Tests that run inside Wine: `GOOS=windows go test -tags glutter_wine -exec "$PWD/scripts/wine-exec.sh" -p 1 <packages>`; host-side tests that start Wine themselves (`internal/winetest`, `cmd/glutter-windows`, `examples`): `go test -tags glutter_wine -p 1 <packages>`. They start Xvfb and Wine themselves and stop them (`wineserver -k`); the tests of different packages share one prefix, so run one package at a time.
9. **Proof images.** Only when the step has a **Proof** item:
   - Produce exactly the images it lists, in `docs/implemented-proof-images/`, named `<id>-<name>.png` (e.g. `6.6-rendering-tour.png`).
   - Use the method it names (A to F, see below).
   - **Open every image with your image-viewing tool** (the Read tool shows images) and check that it shows what **Proof** describes. If it doesn't, fix the code. The step isn't done.
   - Never draw, edit, crop or generate a proof image any other way, and never reuse an older image.
10. **Record the step.** On the step branch, add an entry to `docs/IMPLEMENTED.md`, using the format at the top of that file. Embed the proof images.
11. **Commit on the step branch.** Use one commit per step, with the message `Step <id>: <imperative summary>`:
    ```sh
    git add -A && git commit -m "Step 6.5: Add multi-child render objects"
    ```
12. **Merge every finished step into the local `main` right away; publish to GitHub only a finished milestone** (the owner's rule of 2026-10-27; until then every step was pushed).
    - A step counts as finished when its commit is on the local `main` with its tag. Merge before you start the next step, and never leave a finished step only on its branch.
    - Merge only after rules 8–11 are complete: checks green, proof verified, `IMPLEMENTED.md` entry committed.
    ```sh
    git switch main
    git merge --ff-only feat/step-6.5-multi-child-render-objects
    git tag -a step-6.5 -m "Step 6.5: Add multi-child render objects"
    git branch -d feat/step-6.5-multi-child-render-objects
    ```
    - **Tag every finished step** with an annotated tag `step-<id>` (the step's id as in this file) on its commit on `main`. If the tag already exists, stop and tell the user; never move or delete a tag.
    - **Publish the milestone** when its last step is merged and tagged. Nothing is pushed before that: `git push --follow-tags origin main` pushes `main` with every annotated tag on it that `origin` lacks (the milestone's step tags and its release tag). Then check with `git ls-remote --tags origin` that every tag of the milestone is there. The milestone is finished only when `origin/main` and its tags are there.
    - The release tags `v<major>.<minor>.<patch>` are a separate thing. When a version is finished (its release step, or the fix step that closes a patch version), the annotated `v…` tag is made on its commit, without asking (the owner's rule of 2026-10-26), and published with the milestone. A patch version made of fix steps outside a milestone is published when the version is finished. The `Release 1.x` steps written before that rule say to ask the user; the rule replaces that question.
    If `--ff-only` fails, rebase the step branch onto `main`, run the standard checks again, and retry. If the push of a milestone is rejected or fails (authentication, network), retry; if it still fails, stop and tell the user.
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
| **F. Wine capture** (from step 28.2) | anything drawn by an app on Windows | `GOOS=windows go build -o <app>.exe <package>`, then `GLUTTER_SCREENSHOT=docs/implemented-proof-images/<file>.png scripts/wine-exec.sh ./<app>.exe`: the app saves its own frame under Wine on a headless Xvfb and exits (SPEC §25.13); from step 28.14 `go run ./cmd/glutter-windows shot -o <file.png> <package>` does both. For what lies outside the app's window (the tray, a dialog, several windows), `scripts/wine-screenshot.sh <file.png> -- <command...>` saves the root window of the Xvfb. Wine's fonts, its DXGI and its dialogs are not Windows', so a proof shows that the app draws, not what a PC's screen looks like; that is the manual checks |

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
| M28 Windows (1.8) | 28.1–28.19 | every example, `tray` and `multiwindow` included |

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

## Tools for M28

Install a missing tool only by asking the user (rule 7), with the command from this table. On 2026-10-25 `wine`, `Xvfb`, lavapipe, `naga` and ImageMagick were installed (Wine 11.0), and `wine-cms` and `mingw64-gcc` were not. Everything a test writes goes into `.wine-test/` at the module root (git-ignored); nothing of the user's `~/.wine` is read or written. Wine is LGPL and mingw-w64 is GPL: they are tools, nothing of them ships (SPEC §20).

| Tool | Needed from | Check | Install (Fedora) |
|---|---|---|---|
| Wine (`wine-core`) | 28.1 | `wine --version` says 11 or later; `wineserver --version` | `sudo dnf install wine-core` |
| `wine-cms` (without it a new Wine prefix hangs at its first boot, on `mscms.dll`; `scripts/wine-exec.sh` has a workaround, but the package is better) | 28.1 | `rpm -q wine-cms` | `sudo dnf install wine-cms` |
| Xvfb | 28.1 (`scripts/headless-x11.sh`) | `Xvfb -help` | `sudo dnf install xorg-x11-server-Xvfb` |
| Mesa lavapipe (Wine's Direct3D 12 runs on the host's Vulkan, and only lavapipe presents on Xvfb) | 28.8 | `ls /usr/share/vulkan/icd.d/lvp_icd.*.json` | `sudo dnf install mesa-vulkan-drivers` |
| mingw-w64 (`mingw64-gcc`, `mingw64-headers`): the layout tests of the Win32 and COM mirrors compile a C file against its headers | 28.1 | `x86_64-w64-mingw32-gcc --version`, and `ls /usr/x86_64-w64-mingw32/sys-root/mingw/include/d3d12.h` | `sudo dnf install mingw64-gcc mingw64-headers` |
| naga (it writes the HLSL too, `--shader-model 51`) | 28.7 | `naga --version`, and `naga --help` lists `--shader-model` | `cargo install naga-cli --locked` |
| ImageMagick (`import -window root`, the root-window captures of proof method F) | 28.1 (`scripts/wine-screenshot.sh`) | `import -version` | `sudo dnf install ImageMagick` |
| A Windows 10 (1809 or later) or 11 PC, with a second monitor, a touch screen and a pen if possible, and Narrator (built in) and NVDA (free) | 28.2 (manual checks) | `go run ./cmd/glutter-windows build` works on it, or the owner copies the `.exe` that the tool built here | none: the owner's machine |

---

## M28: Windows (1.8)

The owner chose on 2026-10-25 to port to Windows next, as release 1.8, "everything at once": every example that runs on Linux runs on Windows, `tray` and `multiwindow` included. That is the platform layer (windows, input, text input and IMEs, clipboard, cursors, DPI, displays, settings, fonts, lifecycle), the desktop services (tray, notifications, file dialogs, open URI), OLE drag and drop, UI Automation, the packaging of an `.exe`, and a Direct3D 12 renderer. The owner also decided: the renderer is Direct3D 12 at feature level 11_0 with shader model 5.1, as a new `explicit.Device` (D25, replacing the ANGLE plan of §6.4); the shaders are HLSL that naga writes and `D3DCompile` compiles at run time; Windows is called with `syscall`, not purego and not `golang.org/x/sys/windows`; and the tests run under Wine, with the rest as manual checks on a real PC. SPEC §25 and §9.8 are the design; each step adds a "Decided in step" paragraph to the section it settles, as the web and Android steps did.

Before 28.1 there is no Flutter Windows embedder in `reference/`: **step 28.1 adds `engine/src/flutter/shell/platform/windows` and `shell/platform/common` to the fetch and checks every "Port from" path below**, and corrects this plan where a file is named differently or is missing. Windows is tested under Wine (SPEC §25.13): the steps that open windows or use the GPU list **Wine tests**, the visible ones list a **Proof** of method F, and the things Wine cannot show are **Manual checks** on a Windows PC, which 28.15 collects in `docs/MANUAL-CHECKS.md`. Step order: the platform (28.2 to 28.6) draws with `soft`; the Direct3D 12 renderer (28.7 to 28.10) comes after it; the services, drag and drop and accessibility (28.11 to 28.13) need both. Steps 28.16 to 28.18 fix what Emetos asked for (its issues file, the owner's decision of 2026-10-25), so that they ship in 1.8.

### 28.1 Windows groundwork
**Goal:** `GOOS=windows` builds and vets, an `.exe` can call Windows (DLLs, COM, callbacks), Go's mirrors of Windows structures are checked against the mingw-w64 headers, and the Windows tests run under Wine with one command.
**Read:** SPEC §25 (the introduction, "Build tags"), §25.2, §25.13, §6.3 (rules 2, 7, 10, 11 and 12), §8.1 (the purego rules this mirrors), §14.4, §17, D25, D26, CLAUDE.md ("Don't": processes).
**Port from:** none (the plumbing is Glutter's).
**Files:** `internal/engine/backends_windows.go`, `backends_other.go`, `foundation/host_windows.go`, `foundation/web_other.go`, `internal/gpu/surface.go`, `internal/platform/windows/` (`doc.go`, `win32/`), `internal/winetest/`, `scripts/wine-exec.sh`, `scripts/wine-screenshot.sh`, `scripts/test-windows.sh`, `scripts/fetch-reference.sh`, `cmd/glutter-gen/internal/errcheck`, `.gitignore` (`.wine-test/`), every package that does not build for Windows yet.
**Tasks:**
1. Make `GOOS=windows GOARCH=amd64` and `arm64` build and vet: find the packages that do not (a file with a Unix-only call and no tag, a package whose files all say `linux`) and give each the tag or the split that SPEC §6.3 says; every package whose files all carry an OS tag keeps an untagged `doc.go`. `backends_windows.go`: `Run` fails with `BackendUnavailable`, no renderer candidates and no font matcher until steps 28.2, 28.6 and 28.10; `backends_other.go` becomes `!linux && !js && !windows`. `foundation/host_windows.go` (the host platform is `TargetPlatformWindows`, `IsWeb` is false) and the constraint of `web_other.go`. `gpu.SurfaceKindWin32` (`Window` is the `HWND`, `Display` the `HINSTANCE`).
2. `internal/platform/windows/win32` (SPEC §25.2): `LoadSystemDLL` (`LoadLibraryExW` with `LOAD_LIBRARY_SEARCH_SYSTEM32`; `LibraryLoadFailed` and `SymbolMissing` with the names), a proc table, UTF-16 helpers (a Go string to a `*uint16`, a fixed `[N]uint16` to a string), `GetLastError` errors, the `HRESULT` type with the failure check and the table of codes that become tags, and the first bindings that the later steps need to start (`kernel32`: `HeapAlloc`, `HeapFree`, `GetModuleHandleW`, `GlobalAlloc`, `Sleep`, events and waits). The Go mirrors of structures and the COM method indexes are in files with no OS tag.
3. COM: the `Object` type and `Call(index, args...)` over a vtable; GUID and IID parsing; the Go-implemented COM object of SPEC §25.2 (blocks from `HeapAlloc`, vtables filled once with `syscall.NewCallback` functions, `QueryInterface` by IID, atomic `AddRef`/`Release`, the table that finds the Go object by handle); a counter of the callbacks made so far (`CallbackCount`) and a test that fails above 1,500.
4. The layout-test harness (SPEC §25.2): it writes a C file of `_Static_assert`s from a Go mirror (`unsafe.Sizeof`, `Offsetof`, the vtable indexes) and compiles it with `x86_64-w64-mingw32-gcc -fsyntax-only` (`-DCOBJMACROS`); it skips with the install command when the compiler or the headers are missing. The first mirrors checked are those the harness itself needs (a `GUID`, a `RECT`, a `POINT`, `WNDCLASSEXW`, `MSG`) and the vtables of `IUnknown`.
5. `internal/winetest` (SPEC §25.13): the Linux-side harness (`linux` tag, a process group, as `internal/webtest` has it) that makes the prefix in `.wine-test/prefix` (the first-boot workaround when `wine-cms` is missing, under a lock), starts Xvfb through `scripts/headless-x11.sh` and a Windows program under `timeout -k`, in its own process group with the environment of SPEC §25.13, stops them with `wineserver -k`, and reports `Leftovers` (it reads `/proc/*/environ` for the prefix). And a `windows`-tagged helper that tells a test whether it runs under Wine (`ntdll`'s `wine_get_version`).
6. `scripts/wine-exec.sh`, `scripts/wine-screenshot.sh` and `scripts/test-windows.sh` as SPEC §25.13 describes them; `.wine-test/` in `.gitignore`.
7. `scripts/fetch-reference.sh`: add `engine/src/flutter/shell/platform/windows` and `shell/platform/common` (and `engine/src/flutter/third_party/accessibility` if it exists at the pinned commit), at the pinned commit, and run it. Then check every Flutter file named in SPEC §25 and in the "Port from" lines of M28; where one is missing or has another name, fix the SPEC and this plan in the same commit, and tell the manager.
8. `glutter-gen errcheck`: a `windows/amd64` pass, and the files with the `glutter_wine` tag, so that the Windows files are checked (as the Android pass does, step 25.1). The test of the whole module keeps passing on a machine that has no Wine.
9. Tell the manager the lines that CLAUDE.md needs (the commands, the definition of done, the layout, the build-tag list and the allowed `syscall` use, the Wine testing note); the manager edits CLAUDE.md.
**Tests:** `go list -deps` of `examples/counter` for `GOOS=windows` has no package of `internal/platform/linux`, `android` or `web`, and the Linux, js and android lists are unchanged (extend `TestBuildsHaveTheirOwnBackends`); the HRESULT table (code → tag); UTF-16 round trips (BMP, surrogate pairs, an embedded NUL, a string longer than a fixed buffer); the GUID parser; the layout harness over a fixture mirror, right and wrong (the wrong one fails with the structure's name); the `winetest` command lines (a fake runner); route tests of every new tag.
**Wine tests:** under `scripts/test-windows.sh` (they need no window): `LoadSystemDLL` finds `kernel32.dll` and `user32.dll` and not a DLL planted in the current directory or next to the test binary; a COM object made in Go is called back through its own vtable with `Call` (`QueryInterface` to another interface and to an unknown IID, `AddRef`/`Release` to zero frees the blocks); `CallbackCount` stays under the limit after the packages' init; `winetest` starts a tiny Windows program (built by the test), reads its exit code and output, and leaves nothing running.
**Proof:** none.
**Done when:** the default "Done when", the four Windows checks of rule 8 pass, and `pgrep -a` shows no `wine`, `wineserver` or `Xvfb`.

### 28.2 Windows and the message loop
**Goal:** `examples/counter` shows its first frame in a Win32 window under Wine, drawn by `soft` with GDI; windows open, resize, move between scales, minimize, maximize, go full screen, hide and close; and an idle app sleeps.
**Read:** SPEC §25.3, §25.4, §25.5 (`soft` into the window), §13, §8.4, §8.7, §8.11 ("Frame pacing", "Resizing and scaling"), §9.6, D26.
**Port from:** `shell/platform/windows/flutter_window.cc` (the window class, `WndProc`, `HandleMessage`, `OnResize`, `OnDpiScale`, activation; the pinned commit has no `win32_window.cc`, `FlutterWindow` holds what it held), `host_window.cc` (`HostWindow::HandleMessage`: `WM_DPICHANGED`, `WM_SIZE`, full screen), `window_binding_handler.h`, `task_runner_window.cc` (the window that wakes the loop), `dpi_utils.cc`.
**Files:** `internal/platform/windows/` (`platform_windows.go`, `loop_windows.go`, `class_windows.go`, `window_windows.go`, `displays_windows.go`, `titlebar_windows.go`, `console_windows.go`, plain-Go `timers.go`, `pacer.go`, `winstyle.go`, `frames.go`), `internal/platform/windows/win32` (user32, gdi32, dwmapi, shcore, shell32 bindings), `internal/gpu/gdiwindow/`, `internal/engine/backends_windows.go`.
**Tasks:**
1. Process start (SPEC §25.1, §25.3): the DPI awareness call when the manifest did not set it, `OleInitialize`, the explicit AppUserModelID, the console attach and `OutputDebugStringW` logging for a GUI program.
2. The service window and `Post` (SPEC §25.3); the loop with its timer heap and `MsgWaitForMultipleObjectsEx`; the modal-loop timers.
3. The vsync thread and the fallback pacer (SPEC §25.3), with the plain-Go state machine (`pacer.go`) separate from the `DwmFlush` call; `RequestFrame` with the merging rules of the other platforms.
4. `NewWindow` and the window procedure (SPEC §25.4): the class, the styles from `WindowConfig` (`winstyle.go`, plain Go), sizes in logical pixels, DPI and `WM_DPICHANGED`, the constraints, the states and full screen, hidden windows, the first show after the first frame, the title, the icon and the app id, `KeepScreenOn`, close requests and the end of `Run`, `DecorationsApp` and `PerformWindowAction`, the dark title bar (the setting itself comes in step 28.6; here it is read once with the registry call), `OnViewFocus` and `OnKeyboardFocus`.
5. `Displays()` and `OnDisplaysChanged` (SPEC §25.4).
6. `internal/gpu/gdiwindow`: `soft` plus a `Presenter` (SPEC §25.5, the design of `nativewindow` and `canvas2d`); the sequence is plain Go over a `windowAPI` interface, tested with a fake.
7. The engine's Windows list: the backend `win32` (`GLUTTER_BACKEND=win32` or empty), the renderer names `auto`, `d3d12` and `soft` with `d3d12` still `RendererUnavailable`, `auto` is `soft` for now; settings with defaults (step 28.6), a clipboard that fails with `ClipboardUnavailable` (step 28.6), no services (step 28.11).
8. `GLUTTER_DEV_EVENTS` for Windows (SPEC §25.6): the script reader, the replay from the first frame and `wait` (the later steps add `move`, `tap`, `wheel`, `key`, `text` and `compose`); `GLUTTER_SCREENSHOT` works through the engine, with the paths of SPEC §25.5.
**Tests:** table tests of the plain-Go parts: the pacer (a `DwmFlush` that returns at once twice, one that fails, one that is normal, a wanted frame while none is wanted), the timer heap (order, cancel, the next deadline), the frame-request merging, the style from the config (resizable or not, decorated or not, dialog, modal, borderless), the outer size from the client size at several scales and `WM_GETMINMAXINFO`, the full-screen rectangle and the restore, the `WM_SIZE` to window-info mapping, the `gdiwindow` presenter over a fake (a new size, damage rectangles, the kept frame and an exposure), the UTF-16 title; route tests of every new tag.
**Wine tests:** a window opens and the handler gets `OnMetrics` with the size and scale; `SetWindowPos` to a new size gives a new `OnMetrics` and a frame at that size before the handler returns; a prefix with `LogPixels` 144 gives a scale of 1.5 and a window of the right physical size (the harness writes the registry value); the constraints hold (`GetWindowRect` after `SetWindowPos` past them); minimize, maximize, restore and full screen report their states and the full-screen window has the monitor's rectangle; `SetHidden` removes and restores the window and asks for no frame while hidden; a close request that the handler refuses keeps the window, one it accepts ends `Run` when it was the last window; `DecorationsApp` has a client area as large as the window (`WM_NCCALCSIZE`) and the hit test answers the resize directions; `Post` from 100 goroutines runs every function once, in order for each goroutine; an idle app runs no loop turn for a second (counted by the platform) and the process uses no CPU ticks; the counter's first frame (`GLUTTER_SCREENSHOT`) equals its `soft` golden.
**Proof:** F, `28.2-windows-counter-soft.png`: `examples/counter`, drawn by `soft` into the window, saved by self-capture.
**Manual checks:** on the owner's PC: build the counter with `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags=-H=windowsgui -o counter.exe ./examples/counter` (on Linux, then copy the file; or on Windows itself) and run it; the window opens without a white flash; resize it by a corner and by the edges, the content follows the pointer; maximize, restore, full screen (the task bar is covered); drag it from a monitor at 100% to one at 150%, it stays sharp at once; change the scale in Settings while it runs; light and dark system mode change the title bar; Alt+Tab; minimize to the task bar and back; close it. Record the answers.

### 28.3 Mouse, touch, pen and cursors
**Goal:** pointer input on Windows as Flutter's embedder delivers it, and the cursors of `SystemMouseCursors`.
**Read:** SPEC §25.6 (mouse, wheel, touch and pen, cursors, scripted events), §8.5, §8.11 ("Input"), §12.10 (the rows on `PointerData` if any).
**Port from:** `flutter_window.cc` (`OnPointerMove`, `OnPointerDown`, `OnPointerUp`, `OnPointerLeave`, `OnScroll`), `flutter_windows_view.cc` (the pointer state and `SendPointerEventWithData`), `cursor_handler.cc`, `sequential_id_generator.cc` (the touch ids).
**Files:** `internal/platform/windows/` (`mouse.go`, `wheel.go`, `pointer.go`, `cursor.go`, `devevents.go`, and the `_windows.go` glue), `win32` (the input and cursor bindings), `cmd/glutter-gen` docs only if a table is added.
**Tasks:**
1. The mouse state machine (plain Go): buttons to Flutter's bits, the add, hover, down, move, up and remove events with the rules of SPEC §25.6, capture and the loss of it, `WM_MOUSELEAVE`, signed 16-bit coordinates.
2. The wheel (vertical and horizontal) with the amount per notch of Flutter's embedder and the system's lines setting.
3. Touch and pen from `WM_POINTER*`: the decoding of `POINTER_INFO`, `POINTER_TOUCH_INFO` and `POINTER_PEN_INFO` into plain values (pressure, tilt, rotation, the eraser end, the barrel button, the frame), the conversion to `PointerData`, the ids, no `DefWindowProc` for a handled message, the feedback setting.
4. Cursors: `WM_SETCURSOR`, the table of `SystemMouseCursors` (SPEC §25.6), the cursors with no stock counterpart in §12.10 with a `// Deviation:` comment, `MouseCursorNone`.
5. `GLUTTER_DEV_EVENTS`: `move`, `tap` and `wheel`, sent as the messages of a real mouse.
**Tests:** `flutter_window_unittests.cc` and the pointer cases of `flutter_windows_view_unittests.cc` (the relevant cases, as table tests of the plain-Go conversions); the state machine (a click, a drag outside the window with capture, leaving with a button down, two buttons, side buttons, the capture taken by another window); the wheel (one notch, a precision touchpad's small deltas, `WHEEL_PAGESCROLL`, horizontal); the pointer decoding (touch with several contacts in one frame, a pen with tilt and rotation, the eraser, hover, a lost capture); the cursor table covers every name of `SystemMouseCursors`.
**Wine tests:** `SendInput` (and the scripted events) give a click on the counter's button three times and the count is 3; a drag that leaves the window and is released outside still ends in an up; `WM_MOUSELEAVE` removes the pointer; a wheel scroll moves `examples/scrolling`; `GetCursorInfo` after a move over a text field and a button shows the cursors of the table (`IDC_IBEAM`, `IDC_HAND`).
**Proof:** F, `28.3-windows-scrolling.png`: `examples/scrolling` after a scripted wheel scroll.
**Manual checks:** on the owner's PC: all mouse buttons including the side ones (back and forward in the gallery's navigation, if mapped); a click-drag that goes outside the window; a wheel with notches and a free-spinning one; a precision touchpad (two-finger scroll, pinch in the gallery's `InteractiveViewer`); a touch screen (tap, drag, two fingers, long press) and a pen (pressure in a custom painting, the eraser end, the barrel button), if there are any; the cursors in the gallery (text field, buttons, a resize edge). Record the answers.

### 28.4 Keyboard
**Goal:** keys on Windows as Flutter's embedder reports them, on any layout, with the Windows text editing shortcuts.
**Read:** SPEC §25.6 (keyboard), §25.12, §8.5 (keyboard), §11.1 and §12.10 (the `DefaultTextEditingShortcuts` row).
**Port from:** `shell/platform/windows/keyboard_manager.cc`, `keyboard_key_embedder_handler.cc`, `keyboard_key_handler.cc`, `keyboard_utils.cc`, `flutter_window.cc` (`OnKey`, `OnText`); the data in `dev/tools/gen_keycodes/data` (`physical_key_data.g.json`, `logical_key_data.g.json`, `windows_scancode_logical_map.json`, `windows_logical_to_window_vk.json`, `layout_goals.json`, `windows_flutter_key_map_cc.tmpl`); `packages/flutter/lib/src/widgets/default_text_editing_shortcuts.dart` (`_windowsShortcuts`, `_commonShortcuts`, `_clipboardShortcuts`, the choice by platform).
**Files:** `cmd/glutter-gen/internal/keys` (`-windows-out`), `internal/platform/windows/` (`keyboard.go`, `keymap_gen.go`, `deadkeys.go`, and the glue), `widgets/default_text_editing_shortcuts.go`, `material/widgets_reexport_gen.go` if it changes, SPEC §11.1 and §12.10.
**Tasks:**
1. `glutter-gen keys -windows-out`: the tables of SPEC §25.6 from Flutter's data, with the layout goals; the header names the pinned commit.
2. The key converter (plain Go): scan code and extended flag to the physical key, character, virtual key and scan code to the logical key as Flutter's embedder does, the character from the queued `WM_CHAR` or `WM_DEADCHAR`, surrogate pairs, repeats, the AltGr fake Control, the synthesized modifier and lock events, the held keys released on `WM_KILLFOCUS`, `WM_INPUTLANGCHANGE`.
3. The window procedure's key messages: `OnKeySync` for the answer, `DefWindowProc` for unhandled system keys, `F10` and Alt alone consumed when handled (SPEC §25.6).
4. `DefaultTextEditingShortcutMap()` chooses the Windows map on Windows; the other platforms keep theirs; update the §12.10 row and the sentence in §11.1.
5. `GLUTTER_DEV_EVENTS`: `key:`.
**Tests:** the relevant cases of `keyboard_unittests.cc`, `keyboard_key_embedder_handler_unittests.cc` and `keyboard_key_handler_unittests.cc` as table tests of the converter (a plain letter, Shift, Control+letter, a dead key then a letter, AltGr on a German layout, a Cyrillic layout where the letter keys take the US character, the numpad with and without Num Lock, repeats, a lost focus with keys down, a surrogate pair); the generated tables against Flutter's data (spot checks of 40 keys, and every key of the data has a physical or a logical entry); `default_text_editing_shortcuts_test.dart` and `editable_text_shortcuts_test.dart` (the Windows cases).
**Wine tests:** `SendInput` keys type into the editor's field (letters, Shift, Backspace, Delete, arrows, Control+A, Control+C and Control+V with the clipboard, Home and End, Page Down); Wine's US layout and a loaded German layout (`LoadKeyboardLayoutW`) give the right logical keys for `Z` and `Y` (the layout goals); Alt alone does not open the system menu while Alt+F4 closes the window; Num Lock and Caps Lock states are seen after the focus returns.
**Proof:** F, `28.4-windows-editor.png`: `examples/editor` with typed text and a selection made by scripted keys.
**Manual checks:** on the owner's PC with at least a German or French layout and a Russian or Greek one: letters and shortcuts on each (Control+C works on every layout), AltGr characters, dead keys (`^`, `´`, `` ` `` then a letter), the numpad, Windows key combinations (they stay the system's), Alt+F4, Alt+Space, `F10`, key repeat while holding, switching layouts with Win+Space while the app runs. Record the answers.

### 28.5 Text input and IMEs
**Goal:** text fields work with the Windows input methods: composition is drawn in the field, the candidate window follows the caret.
**Read:** SPEC §25.7 (IMM32), §23.5 ("Decided in step 21.3", the contract), §24.6 (the Android model, for `TextEditor`), §8.6.
**Port from:** `shell/platform/windows/text_input_manager.cc` (`TextInputManager`: `GetComposingString`, `GetResultString`, `UpdateImeWindow`), `text_input_plugin.cc`, `flutter_window.cc` (`OnComposeBegin`, `OnComposeChange`, `OnComposeCommit`, `OnComposeEnd`, `UpdateCursorRect`).
**Files:** `internal/platform/windows/ime/` (plain-Go `composition.go` over an `imm` interface, `imm_windows.go`), `internal/platform/windows/textinput_windows.go`, `win32` (imm32 bindings), the window procedure, `GLUTTER_DEV_EVENTS` (`text:`, `compose:`).
**Tasks:**
1. The IMM32 bindings (`ImmGetContext`, `ImmReleaseContext`, `ImmGetCompositionStringW`, `ImmSetCandidateWindow`, `ImmSetCompositionWindow`, `ImmAssociateContext`, `ImmAssociateContextEx`).
2. The composition model (plain Go): `WM_IME_STARTCOMPOSITION`, `WM_IME_COMPOSITION` (`GCS_COMPSTR` and `GCS_CURSORPOS` to a preedit, `GCS_RESULTSTR` to a commit), `WM_IME_ENDCOMPOSITION`, `WM_IME_SETCONTEXT`, turned into `TextInputEvent`s by `textedit.TextEditor`; UTF-16 offsets to bytes; characters from `WM_CHAR` for a key with no IME, with surrogate pairs.
3. `SetTextInput`: turning the IME on and off with the focus, the candidate and composition windows at the caret rectangle (SPEC §25.7), moved when it changes.
4. `GLUTTER_DEV_EVENTS`: `text:` types as `WM_CHAR`, `compose:a,b,c` sends the composition messages of an IME through the `imm` interface's test double or Wine's own IME.
**Tests:** the composition model over a fake `imm` (a Japanese session: start, several updates with the caret inside, a conversion, the result; a Chinese session with a candidate commit; a cancelled composition; a commit with no preedit; a surrogate pair; a composition that crosses a framework state change; text with combining marks); the offsets; the candidate window math at scales 1, 1.25 and 1.5; a key with `VK_PROCESSKEY` is logical key "Process" and types nothing; the cases of `text_input_plugin_unittest.cc` and `window_unittests.cc` (it holds the `TextInputManager` cases; there is no `text_input_manager_unittests.cc` at the pinned commit) that concern composition.
**Wine tests:** `ImmGetContext` of the window works and `ImmAssociateContext(NULL)` turns the IME off and back on with the focus; the composition messages sent by the scripted events arrive as `Preedit` and `Commit` in an `EditableText` of `examples/editor`; the candidate window position is the caret rectangle's (`ImmGetCandidateWindow`); if Wine's default IME produces composition with `ImmSetCompositionStringW`, a test uses it, else it says so and the fake stands in.
**Proof:** F, `28.5-windows-editor-preedit.png`: `examples/editor` with a composition in progress (`compose:` events), the preedit underlined.
**Manual checks:** on the owner's PC with the Microsoft IMEs added (Settings → Time & language): Japanese (type, convert, choose a candidate), Chinese Pinyin, Korean; the candidate window opens next to the caret at 100%, 150% and on the second monitor; the emoji panel (Win+.) inserts an emoji; the touch keyboard and handwriting panel on a touch screen, if there is one (they may not work: record it). Record the answers.

### 28.6 Clipboard, settings, fonts and lifecycle
**Goal:** copy and paste work with other programs, the app follows the system's settings live, text uses the system's fonts, and the app's lifecycle is Windows'.
**Read:** SPEC §25.7 (clipboard), §25.8, §10, §13, §23.6 (the web's settings), §24.7 and §24.9 (Android's, for the shape), §12.10 (rows to add).
**Port from:** `shell/platform/windows/platform_handler.cc` (the clipboard), `settings_plugin.cc`, `system_utils.cc`, `windows_lifecycle_manager.cc`.
**Files:** `internal/platform/windows/` (`clipboard.go` plain-Go conversions, `clipboard_windows.go`, `settings.go`, `settings_windows.go`, `lifecycle.go`, `fonts_windows.go`), `internal/text/winfonts/` (plain Go), `internal/platform/settings.go` and `ui` (`CaretBlinkTime`, `DoubleClickTime`), `widgets/editable_text.go` (the blink), `internal/engine/backends_windows.go` (`SystemFontMatcher`), SPEC §12.10.
**Tasks:**
1. The clipboard (SPEC §25.7): text with the line-break conversion, `CF_HDROP` files, PNG and `CF_DIBV5` images, the retry on a busy clipboard, the read on a worker thread with the context, the errors.
2. The settings of SPEC §25.8 read from the registry and the system calls, followed on every `WM_SETTINGCHANGE`; `Settings` and `PlatformConfiguration` get the caret blink time and the double-click time (`// Deviation:` and a row in §12.10); `EditableText` blinks at that rate and does not blink when the system says never.
3. `internal/text/winfonts`: the family and file list from the directories and the registry (the reading is `_windows.go`, the matching plain Go), the generic families and the fallback chain of SPEC §25.8; `engine.SystemFontMatcher` on Windows; go-text's `fontscan` cache in `os.UserCacheDir()`.
4. The lifecycle of SPEC §25.8, the session end, and `Hidden` stopping the frames.
5. The dark title bar follows the setting (step 28.2 read it once); the title bars of all windows change live.
6. Self-capture under Wine with a Windows path and a relative one, and `GLUTTER_SCREENSHOT_AFTER`.
**Tests:** the clipboard conversions (CRLF, a lone `\r`, an embedded NUL, UTF-16 surrogates, a `DROPFILES` with several paths and one with a network path, PNG and DIB to image and back); the settings from fixture registry values (dark and light, absent keys, text scale 225, the blink time `INFINITE`, a locale list with a script tag, a 24-hour pattern); the font matcher over a list of families and files written for the tests as a stock Windows 10 and 11 have them (`sans-serif`, `serif`, `monospace`, an emoji, `zh-Hans`, `zh-Hant`, `ja`, `ko`, an Indic script, Thai, a missing family); the lifecycle table (visible, active, minimized, hidden, several windows, the session end); `EditableText` blink period.
**Wine tests:** a clipboard round trip of text, files and an image through the real clipboard (a second window, and a second process, stand in for another program); a busy clipboard (another thread holds it) is retried and then `ClipboardBusy`; a read cancelled by its context is `ClipboardReadTimeout`; writing `AppsUseLightTheme` in the prefix's registry and broadcasting `WM_SETTINGCHANGE` gives `OnSettings` with the new brightness; the matcher finds Wine's own fonts (Tahoma) and a generic family; minimize and restore give `Hidden` then `Resumed`; `GLUTTER_SCREENSHOT` with a Windows path writes the file.
**Proof:** F, `28.6-windows-text.png`: `examples/dev/text-tour` drawn with the fonts Wine has (Latin, Greek and Cyrillic; the missing glyphs of CJK and emoji are boxes there and a manual check on a PC).
**Manual checks:** on the owner's PC: copy and paste text, files and an image to and from Notepad, Explorer and Paint; switch Windows between dark and light and see the gallery change at once; change the accent color, the text size (Settings → Accessibility → Text size), turn on a high contrast theme and animation effects off; the CJK text and the color emoji of `examples/dev/text-tour`; the caret blink follows Control Panel's setting; log off or restart with the app open (it ends without a hang). Record the answers.

### 28.7 Shaders for Direct3D 12
**Goal:** every shader of the explicit renderer has HLSL for shader model 5.1 in `shaders_gen.go`, and `D3DCompile` turns each entry point into DXBC, with the bytecode kept in the shader cache.
**Read:** SPEC §9.3 ("Decided in step 19.3", the shader paragraphs), §9.8 ("Shaders", "Binding model"), §9.7 (Direct3D 12), D20, D25, §20 (naga).
**Port from:** none.
**Files:** `cmd/glutter-gen/internal/shaders` (the HLSL pass), `internal/gpu/explicit/shader.go` (`HLSL` and the register mapping), `internal/gpu/explicit/shaders_gen.go` (regenerated), `internal/gpu/d3d12/` (`compile_windows.go`, `hlslcache.go` plain Go, `doc.go`), `internal/gpu/shadercache`.
**Tasks:**
1. Find out what naga's command line does for HLSL (`naga in.wgsl out.hlsl --shader-model 51`, and the option that sets the register of a binding): the register of binding B in group G must be `register(<kind>B, spaceG)`. If the command line cannot say this, stop and ask (rule 7); the generator must not need a tool beyond naga.
   *Result (owner's decision, 2026-10-25):* naga 30.0.1's command line cannot say this for samplers (it writes a sampler heap and a buffer of indexes, `nagaSamplerHeap` and `nagaGroupNSamplerIndexArray` at `space255`, and has no option for the binding map or the special constants), and writes the registers of group 0 without a space. The generator therefore rewrites naga's HLSL in Go (`rewriteHLSL`, `checkHLSL`) and fails with a tagged error on any shape it does not know; it needs no tool beyond naga. Because the special constants cannot be asked for either, no shader may read `instance_index`: the pre-rotation of the blit moved from the first instance into a uniform (SPEC §9.8, "Decided in step 28.7").
2. `glutter-gen shaders` also writes, for each module, the HLSL, and puts it in `shaders_gen.go` as the new field `HLSL string` of `explicit.Shader` next to the WGSL and the SPIR-V; the entry points keep their stage and their HLSL names; the register mapping of each binding (kind, space, register) is recorded in `Shader`. The header records naga's version, and the test that compares a fresh generation with the checked-in files covers the HLSL.
3. Check every shader for what differs on Direct3D: `instance_index` and `vertex_index` (`SV_InstanceID` does not include the first instance: where a shader needs it, the generator uses naga's special constants binding and the register is recorded), derivatives in non-uniform control flow, the vertex semantics. Change a WGSL file only if the Vulkan and WebGPU goldens stay the same.
4. `d3d12.Compile(hlsl, entry, target, flags)` over `D3DCompile` (`win32` loads `d3dcompiler_47.dll` from System32): the blob, the messages, `D3D12ShaderCompileFailed` with them; flags per SPEC §9.8 (optimization level 3 in release builds, debugging information and no optimization with `GLUTTER_D3D12_DEBUG=1`).
5. The DXBC cache (SPEC §9.7): the key from the HLSL, entry point, target, flags and the file version of `d3dcompiler_47.dll` (`GetFileVersionInfoW`); a file with a bad `DXBC` header is dropped; no cache directory is not an error.
6. `D3DReflect` of each compiled entry point: the bound resources match the recorded mapping (the contract with the root signature of step 28.9).
**Tests:** the generator over fixture WGSL (bindings of every kind, two groups, an entry with a texture and a sampler, a vertex shader with `instance_index`) with the expected HLSL fragments and mapping; the cache key (each part changes it); the cache over a temporary directory (hit, miss, a bad header, an unwritable directory); route tests of `D3D12ShaderCompileFailed`; the regeneration test when naga is installed.
**Wine tests:** every entry point of every shader compiles to `vs_5_1` or `ps_5_1` and starts with `DXBC`; a second `Compile` with the cache is a hit and does not call `D3DCompile` (counted); a changed compiler version string is a miss; reflection reports the registers and spaces of the mapping; a deliberately broken HLSL gives `D3D12ShaderCompileFailed` with the compiler's line.
**Proof:** none.

### 28.8 The Direct3D 12 device: adapter, memory and resources
**Goal:** `explicit.Device` on Direct3D 12 offscreen: the adapter choice, the device at feature level 11_0, the queue and fences, buffers, textures, samplers, uploads and read-back.
**Read:** SPEC §9.3 (the device interface and every "Decided" paragraph about buffers, textures, limits and frames), §9.8 (all but "Binding model", "Shaders", "Pipelines" and "Swap chain"), §9.4 (the model), §9.7, D19, D25.
**Port from:** none (the Vulkan backend is the model: `internal/gpu/vulkan`, `memory.go`, `resources.go`, `frames.go`, `check.go`).
**Files:** `internal/gpu/d3d12/` (`dll_windows.go`, `types.go` and `vtables.go` with no OS tag, `adapter.go` plain Go, `device_windows.go`, `memory_windows.go`, `resources_windows.go`, `fence_windows.go`, `states.go` plain Go, `errors.go`, `check.go`), `internal/platform/windows/win32` (the COM helpers for DXGI and D3D12).
**Tasks:**
1. Loading and the entry points (SPEC §9.8 "Loading"); the interfaces and structures that the device calls (the mirrors and the vtable indexes, with their layout tests against `d3d12.h` and `dxgi1_6.h` of mingw-w64).
2. The adapter choice as a pure function over a list of adapter descriptions (hardware order, software skipped, WARP by name, an index, a part of a name), the DXGI enumeration, `D3D12CreateDevice` at 11_0, the optional debug layer and its messages, `D3D12Unsupported`, `D3D12NoAdapter`.
3. The queue, the fence and the event; the two frame slots; `Begin` and `Submit` without passes yet; waiting idle; device removal detection to `GPUDeviceLost` (`GetDeviceRemovedReason`); `d3d12Error` (the one function that turns an `HRESULT` into `GPUOutOfMemory`, `GPUDeviceLost` or `D3D12CallFailed`, listed in `errcheck.allow`).
4. Memory (SPEC §9.8): committed resources, the upload heaps of the ring buffers, the staging ring with its alignments, the readback buffers.
5. `NewBuffer`, `WriteBuffer`, `NewTexture` (rgba8, bgra8 and r8 with mip levels, `Level(i)`), `NewSampler`, `WriteTexture` through the staging ring, `ReadTexture` with `done` after the fence, `CopyTexture`; the state tracker and its batching of barriers (SPEC §9.8 "States"); `Limits` and the checks of §9.3.
6. The startup report lines of §9.7 for the device (adapter, feature level, binding tier).
**Tests:** the adapter choice (a table over fake adapter lists: the order, a software adapter skipped, WARP by name and by `auto`, an index out of range, a name that matches two, none); the state tracker (a texture drawn then sampled then copied: the barriers, batched; two levels of one texture in different states; a conflict is `D3D12StateConflict`); the staging ring (wrap, alignment of 256 and 512, a request larger than the ring, the release by frame); the row-pitch and size math against `explicit.Limits.AlignedRowPitch`; the `HRESULT` table; the limits checks; route tests of every tag.
**Wine tests:** a device is made at feature level 11_0 with the adapter `auto` chooses; `GLUTTER_D3D12_ADAPTER=` an index and a part of the name pick the same adapter, and a wrong one is `D3D12NoAdapter`; `GLUTTER_D3D12_DEBUG=1` logs that the layer is missing and goes on; a buffer is written and read back exactly; textures of each format are written and read back exactly (with a pitch that is not 256-aligned in the source), with mip levels and a view of one level; `CopyTexture` between textures; the fence wait returns once the GPU is done, and 100 frames in a row do not grow memory; `Close` leaves nothing; a second device in the same process.
**Proof:** none.

### 28.9 The Direct3D 12 device: pipelines, passes and parity
**Goal:** the explicit renderer draws every proof scene through Direct3D 12 offscreen and matches `soft`.
**Read:** SPEC §9.8 ("Binding model", "Shaders", "Pipelines", "Descriptors"), §9.3 (bind groups, passes, layers, "Frames"), §9.4 (the model), §17.
**Port from:** none (`internal/gpu/vulkan/resources.go`, `frames.go` and `parity_display_test.go` are the model).
**Files:** `internal/gpu/d3d12/` (`rootsig.go` plain Go, `rootsig_windows.go`, `descriptors_windows.go`, `pipeline_windows.go`, `bindgroup_windows.go`, `encoder_windows.go`, `mapping.go` plain Go, `parity_wine_test.go`), `internal/gpu/explicit` (only if a gap in the interface shows; any change is additive and shared by the other backends).
**Tasks:**
1. The root signature planner (plain Go): from the bindings of a pipeline's two entry points to root parameters (a root constant buffer view for each uniform binding, a table for the textures and one for the samplers of a group, spaces = groups), the DWORD count, the serialization with `D3D12SerializeRootSignature` version 1.0, the root signatures kept by the shape of the bindings. (No root constant for a first instance: step 28.7 moved the blit's rotation to a uniform and no shader reads `instance_index`.)
2. Descriptors (SPEC §9.8): the CPU heaps, the shader-visible heaps of a frame slot in blocks, `CopyDescriptorsSimple` into them when a bind group is made, the stand-ins for unused slots, `SetDescriptorHeaps`.
3. `NewPipeline`: the DXBC through the cache of step 28.7, the input layout from `VertexBufferLayout` (the semantics naga wrote), the blend state, the target format, the rasterizer without culling, the PSO; the pipeline library where `CreatePipelineLibrary` works, with its blob in the shader cache keyed as SPEC §9.8 says; PSOs are made at first use.
4. `NewBindGroup` and the encoder: `BeginPass` (clear, load and don't-care as `LoadOp` says), `SetPipeline`, `SetBindGroup` with the dynamic offsets as root CBV addresses, vertex and index buffers, the scissor and the viewport, `Draw` and `DrawIndexed`, `End`; `CopyTexture`; the barriers before each pass; the closing of the command list and `Submit` with the fence.
5. The parity test: the proof scenes of `gputest` through `explicit.Renderer` on the device, `Snapshot` against `soft` within `gputest.ProofTolerances`, as the Vulkan parity test does; the device loss rule of §9.3 with a device that reports removal on demand (a test hook).
**Tests:** the planner over every shader's bindings (each pipeline of the renderer: the parameter list, at most 20 DWORDs, spaces equal to groups, the offsets map to the right parameters); the blend, format, vertex-format, filter and address-mode mappings (tables); the load-op mapping; descriptor block allocation and reuse (plain-Go allocator); the pipeline cache key; the encoder over a fake COM layer (the order of barriers, heaps and root arguments of a frame that draws, copies and reads back); route tests of every tag.
**Wine tests:** every proof scene matches `soft` within the tolerance (the first run of each scene also compiles its pipelines: the time is logged); a second run with the cache compiles nothing (counted); a scene with more than one layer, a backdrop filter, a blend that reads the destination and a mask filter; a device lost on demand is recovered once and a second loss in 10 seconds ends with `GPUDeviceLost`; no leaks across 200 frames (the resource counts of the device).
**Proof:** none (offscreen).

### 28.10 Presentation and the renderer choice
**Goal:** windows are drawn by `d3d12` under Wine: the swap chain, resizing, damage, the choice of the renderer with its fallbacks, and the startup log.
**Read:** SPEC §9.8 ("Swap chain", "Synchronization", "Device removal"), §9.6, §9.7, §25.5, §25.3, §8.11 ("Frame pacing"), §22 (the GDI risk).
**Port from:** none.
**Files:** `internal/gpu/d3d12/` (`swapchain_windows.go`, `present.go` plain Go for the damage math), `internal/engine/backends_windows.go` and `renderers.go` (the candidates), `internal/gpu/gdiwindow` (the fallback checked), `internal/platform/windows` (the surface handle).
**Tasks:**
1. The swap chain (SPEC §9.8): creation on the command queue for the `HWND`, the format fall back, `MakeWindowAssociation`, `Configure` with `ResizeBuffers`, `Acquire` of the back buffer in the right state, `Present` with dirty rectangles and sync interval 0, `DXGI_STATUS_OCCLUDED` and the device-loss statuses; `D3D12SwapchainFailed`.
2. The engine's Windows candidates: `d3d12` on hardware, `d3d12` on WARP (`Options.Adapter` `warp`), `soft` (SPEC §9.6); by name `d3d12` has no WARP fallback; the chooser logs each failed try once; the first frame is the test of a candidate; `RendererUnavailable` for `gles` and `vulkan`.
3. The startup log lines of §9.7 for Direct3D 12 (`renderer chosen`, the adapter, the shaders built and from the cache, the pipeline library).
4. The window part: a swap chain that was released and `soft` drawing into the same window (SPEC §22): test it; if GDI shows nothing, make the surface a child window that `platform.SurfaceRenewer` makes again and record the decision in SPEC §25.5.
5. `GLUTTER_SCREENSHOT` reads the frame texture back; `Snapshot`; `GLUTTER_DAMAGE=0` and `Config.DisableDamage` work.
**Tests:** the candidate list and the choice with fake renderers (the first fails with each tag, WARP missing, all fail but `soft`, a name given); the damage rectangles to `DXGI_PRESENT_PARAMETERS` (clipping, from the top left, the empty case); the format fall back; the swap chain over a fake COM layer (resize while a frame is in flight waits for the GPU, a zero size does nothing); route tests of every tag.
**Wine tests:** the counter and the gallery draw their first frame with `d3d12` and the self-capture equals `soft`'s within the tolerance; resizing to three sizes presents each at its size; a damage frame equals a whole frame; `auto` with `GLUTTER_D3D12_ADAPTER` naming nothing logs the failure and ends on `soft` (and the frame is drawn); `GLUTTER_RENDERER=d3d12` with the same variable ends `RunApp` with `D3D12NoAdapter`; `GLUTTER_RENDERER=soft` works; an idle `d3d12` app asks for no frames; two windows share one device; scale 1.5 (the prefix with `LogPixels` 144).
**Proof:** F, `28.10-windows-gallery-d3d12.png`: `examples/gallery` drawn by `d3d12`.
**Manual checks:** on the owner's PC: run the gallery and read the renderer line of the log (which adapter?); on a laptop with two GPUs, which one is used; resize and drag between monitors without tearing or stretched frames; `GLUTTER_D3D12_ADAPTER=warp` and `GLUTTER_RENDERER=soft` both work; with the Graphics Tools feature installed, `GLUTTER_D3D12_DEBUG=1` reports no errors while scrolling the gallery; switch the user (fast user switching), lock the screen and sleep and wake the PC, and disable and enable the GPU in Device Manager: the app recovers or ends with the tag, never hangs; a Remote Desktop session. Record the answers.

### 28.11 Desktop services: tray, notifications, file dialogs, open URI
**Goal:** `examples/tray` works on Windows: the tray icon with its menu, balloons, the file dialogs and opening links.
**Read:** SPEC §25.9, §8.9 (the API of package `desktop`), §23.7 and §24.11 (how the web and Android plug in), §6.3 (rule 12), §21 (toasts).
**Port from:** none (Flutter has no counterpart; its apps use plugins).
**Files:** `internal/platform/windows/` (`tray_windows.go`, `menu_windows.go`, `balloon_windows.go`, `filedialog_windows.go`, `openuri_windows.go`, `services.go` plain Go, `worker_windows.go`), `desktop/` (the Windows plumbing, as `web.go` is the web's), `internal/platform/platform.go` (the optional interfaces), `gluttertest` if its fake desktop needs a Windows switch, `examples/tray`, SPEC §25.9.
**Tasks:**
1. The plumbing: the Windows value that `platform.DesktopProvider` returns, implementing Android's `Desktop` interface (open URI and notifications) and the new optional interfaces for a tray and for file dialogs; package `desktop` takes it in `servicesOf` and fails with `DBusUnavailableOnWindows` when it needs a bus; the worker thread (locked, `CoInitializeEx`, results posted).
2. The tray (SPEC §25.9): the notify icon, the callback message, the menu build and the modal show, the activation, `TaskbarCreated`, the diff of a rebuilt model, one tray at a time.
3. Balloons (SPEC §25.9): the queue, replace by ID, the fields and their limits, the owner icon, the close reasons, `NotificationsUnavailable`; check on Windows 10 and 11 whether a hidden icon shows its balloon (a manual check), and make the icon visible for the time of a notification if it does not.
4. File dialogs (SPEC §25.9): open (one, several, a folder), save, save-several, the filters, the owner, cancel as a result; `PortalHandle` as `win32:<hwnd>`.
5. Open URI (SPEC §25.9) with the Windows validation.
6. `examples/tray` runs: the tray, the menu, a notification and a file dialog; the example's header says what Windows lacks (buttons in balloons).
**Tests:** the tray model diff and the item numbering; the label escaping (`&`); the balloon queue (replace in place, replace queued, close of the shown one starts the next, the expiry); the limits of the fields (title 63, body 255, tooltip 127, with the ellipsis); the file-filter spec (`*.png;*.jpg`, no patterns, many filters); the URI validation table (`https`, `mailto`, `C:\x`, `javascript:`, a `file:` URI of an `.exe` and of a `.txt`, an empty string, a control character); the ERROR_CANCELLED to `Cancelled` mapping; route tests of every tag.
**Wine tests:** with a shell tray (`explorer /desktop=glutter,800x600` started by the harness) `Shell_NotifyIconW` adds, modifies and deletes the icon, and a click message that the test posts as the shell would reaches `OnActivate`; the menu built for a model has the items, separators, checks and submenus in order (`GetMenuItemInfoW`); a balloon is accepted by `NIM_MODIFY` and its click message calls `OnAction("default")`; `IFileOpenDialog` is made with the options and filters, shown on the worker thread, and closed by the test (`IDCANCEL` posted to the dialog window) as `Cancelled`; a custom URI scheme registered in the prefix's registry for a test program is opened by `OpenURI` and the program writes a file; the blocked schemes are refused before `ShellExecuteExW` is called.
**Proof:** F, `28.11-windows-tray.png` (root window: `examples/tray` with its icon in Wine's tray), and `28.11-windows-file-dialog.png` (the open dialog of `examples/tray`, with its filters), both by `scripts/wine-screenshot.sh`.
**Manual checks:** on the owner's PC (Windows 10 and, if possible, 11): the icon appears (in the overflow area at first: drag it to the bar), a left click shows the window, a right click shows the menu with its separators and checks, the tooltip; a click outside closes the menu; restart Explorer (`taskkill /f /im explorer.exe`, then `explorer`): the icon comes back; a notification appears as a toast, a click calls the app, a second one waits for the first; with Focus Assist on nothing appears and nothing breaks; the file dialogs (open one and several files, a folder, save with the overwrite prompt, the filters, cancel); open `https://` and `mailto:` links, and a `file:` link to an `.exe` is refused. Record the answers.

### 28.12 Drag and drop, and several windows
**Goal:** drag and drop with other programs and between the app's windows through OLE, and `examples/multiwindow` with dialogs and several windows.
**Read:** SPEC §25.10, §25.4 (dialogs and owners), §8.11 ("Drag and drop, how it works"), §8.7, §13, §12.10 (the rows on windows and drag).
**Port from:** none (Flutter has no drag and drop between applications).
**Files:** `internal/platform/windows/ole/` (`droptarget.go`, `dropsource.go`, `dataobject.go`, `formats.go` plain Go, the COM glue), `internal/platform/windows/drag_windows.go`, `window_windows.go` (owners, modal), `internal/engine` if the several windows need changes, `examples/multiwindow`.
**Tasks:**
1. The COM objects made in Go for `IDropTarget`, `IDropSource`, `IDataObject` and `IEnumFORMATETC` (SPEC §25.2); the formats model (plain Go): `CF_HDROP` with `DROPFILES`, `CF_UNICODETEXT`, `CF_TEXT`, the URL formats, the app's private format, the effects mapping.
2. Targets: `RegisterDragDrop` for each window, `DragEnter`/`Over`/`Leave`/`Drop` to `ui.DropEvent` and the returned effect, the data read at the drop with the 5 s limit.
3. Sources: `StartDrag` posts `DoDragDrop` with the loop timer, `QueryContinueDrag`, `GiveFeedback`, the data object from the `DragRequest`, the in-app recognition, `OnDragEnd`, the synthesized pointer up.
4. Several windows: dialog windows with an owner and the modal disable, the focus and activation between windows (`OnViewFocus` moves from one to the other), the task bar, closing the first window while others stay, one device and one renderer for all windows, `DestroyWindow` order.
5. `examples/multiwindow` runs with all of it.
**Tests:** the formats model (a `DROPFILES` with wide paths, relative and network paths, a text with `\r\n`, a URL, a data object that offers nothing we know, the enumerator's order and `Reset`/`Skip`/`Clone`); the effects mapping both ways; the drag state machine over fakes (enter, over with a changed key state, leave, drop, a drop that the app refuses, `DoDragDrop` cancelled by Escape, a drag that returns to the app's own window); the dialog and modal logic (a modal dialog disables its owner and enables it on close, a dialog whose parent is destroyed); route tests of every tag.
**Wine tests:** two windows of the test process: a drag from one (`DoDragDrop` driven by `SendInput`) to the other arrives as the app's own data (the private format); a data object made by the test with `CF_HDROP` and one with text, handed to the window's registered `IDropTarget` (found with `RegisterDragDrop`'s table through a probe), arrive as `ExternalDragData` with the files and the text; the effect the framework chooses is returned; Escape cancels a drag; a modal dialog's owner is disabled (`IsWindowEnabled`) until the dialog closes; closing the first window keeps the second; two windows draw with one `d3d12` device.
**Proof:** F, `28.12-windows-multiwindow.png`: two windows of `examples/multiwindow` and a dialog (root window, `scripts/wine-screenshot.sh`).
**Manual checks:** on the owner's PC: drag files from Explorer into the editor and the tray example, text from Notepad and from a web page, and the reverse (text and files out of the app to Notepad and Explorer, copy and move with Shift and Control); a drag between two windows of the app; Escape during a drag; a modal dialog blocks its owner and the owner's task-bar button flashes when clicked; Alt+Tab and the task bar with several windows; closing the first window of `multiwindow` while the second is open. Record the answers.

### 28.13 Accessibility: UI Automation
**Goal:** the semantics tree reaches Narrator, NVDA and the other UI Automation clients.
**Read:** SPEC §25.11, §25.2 (callbacks, floating-point thunks, COM objects), §8.10 (the AT-SPI2 bridge, whose model this follows), §7 (semantics updates), §22.
**Port from:** `shell/platform/windows/accessibility_bridge_windows.cc`, `flutter_platform_node_delegate_windows.cc`, `shell/platform/common/accessibility_bridge.cc` and `flutter_platform_node_delegate.cc`, and the Windows platform nodes of `engine/src/flutter/third_party/accessibility/ax/platform` (`ax_platform_node_win.cc`: the property and pattern mapping; `ax_fragment_root_win.cc`: the root of the tree, which the pinned commit has instead of `accessibility_root_node.cc`).
**Files:** `internal/platform/windows/uia/` (`model.go` plain Go, `controltype.go`, `props.go`, `patterns.go`, the COM provider objects, the events, the thunk files `thunk_amd64.s` and `thunk_arm64.s` with their Go declarations), `win32` (uiautomationcore bindings and the interface mirrors), `internal/platform/windows/window_windows.go` (`WM_GETOBJECT`), the engine's accessibility wiring.
**Tasks:**
1. The model: look at `internal/dbus/atspi` (`Model`, which mirrors one tree per view from `ui.SemanticsUpdate`); if it can move to a package with no bus in it without changing its behavior, move it and use it from both; else write the Windows one. The mapping from role and flags to control type, name, help text, states and the patterns a node supports (SPEC §25.11) is plain Go.
2. The floating-point thunks for `ElementProviderFromPoint` and `IRangeValueProvider::SetValue` in Go assembly for amd64 and arm64 (SPEC §25.2), with a test that proves the values arrive.
3. The providers as COM objects made in Go: the root fragment root, the node fragments, the pattern interfaces; the `ProviderOptions` for COM threading; `Navigate`, `GetRuntimeId`, `BoundingRectangle`, `SetFocus`, `GetFocus`, `ElementProviderFromPoint`.
4. `WM_GETOBJECT` and the on/off of SPEC §25.11 through `platform.Services.Accessibility()`; the actions of the patterns to `SemanticsActionEvent`s posted to the UI thread; the events with `UiaClientsAreListening`.
5. A test client in `internal/platform/windows/uia/uiatest` (`IUIAutomation`, from `uiautomationcore`/`UIAutomationClient`): walks the tree of a window, reads properties, calls patterns, listens to events.
**Tests:** the model over semantics updates (add, remove, move, reorder, a node unreachable from the root, several views); the control-type table; the pattern selection per node (flags and actions to patterns); the rectangle math at scales 1 and 1.5 and with the window not at the origin; the runtime id; the UTF-8 to UTF-16 offsets of the Text pattern; the enable/disable state machine; route tests of every tag.
**Wine tests:** `CUIAutomation::ElementFromHandle(hwnd)` returns the root; walking the gallery lists its buttons, texts and fields with their names, control types and bounding rectangles; `Invoke` on the counter's button increments it; `Toggle` on a checkbox, `Value.SetValue` on a text field, `RangeValue.SetValue` on a slider (the thunk), `Scroll` on a list; `SetFocus` moves the framework's focus; focus and property-change events are received by a client that subscribed. If Wine's `uiautomationcore` lacks something (it implements part of the client and of the core), the test skips that case with a message that says what is missing, and the case becomes a manual check.
**Proof:** none.
**Manual checks:** on the owner's PC: turn on Narrator (Win+Ctrl+Enter) and explore `examples/gallery`: moving through the elements reads their names and roles, Enter or Space activates a button, a checkbox says its state, a slider says its value and changes with the arrows, a text field reads its text and the caret while typing, a list scrolls; `examples/counter` says the new count; repeat with NVDA; inspect the tree with Accessibility Insights for Windows or Inspect (names, control types, patterns, bounds), and a high contrast theme. Record the answers.

### 28.14 The `.exe` and `glutter-windows`
**Goal:** `go run ./cmd/glutter-windows build ./examples/counter` makes an `.exe` with its manifest and icon that runs on Windows and under Wine; `run` and `shot` work on Linux.
**Read:** SPEC §25.1, §25.13 (proof method F), §16, D26.
**Port from:** none.
**Files:** `cmd/glutter-windows/`, `internal/wintool/` (`manifest.go`, `coff.go` and `rsrc.go` (the `.syso` writer), `ico.go`, `versioninfo.go`, `build.go`, `wine.go`), `.gitignore` (`*.syso`, if the fallback writes one), `docs/GUIDE.md` (a note), `internal/winetest` (use).
**Tasks:**
1. The manifest generator (SPEC §25.1) and its golden text.
2. The `.syso` writer in pure Go: a COFF object for amd64 and arm64 with a `.rsrc` section (the resource directory, the data entries, the relocations the linker needs), holding `RT_MANIFEST`, `RT_ICON` with `RT_GROUP_ICON` (16 to 256 pixels from the PNG, the 256 one stored as PNG) and `RT_VERSION`; the default icon (Glutter's) when none is given.
3. `build`: the overlay (or the temporary copy: find out which the go command accepts, SPEC §25.1), `-H=windowsgui` or `-console`, `-release`, the architectures, the output name; the version from `-version`.
4. `run`: builds and starts the `.exe` under Wine through `internal/winetest` (on the user's display, or `-headless` on Xvfb), shows its output, stops everything; on Windows it runs the `.exe` itself. `shot`: proof method F: builds, runs with `GLUTTER_SCREENSHOT` on Xvfb, reads the PNG; `-screen` takes the root window; `-env GLUTTER_X=v` is given to the app.
5. `docs/GUIDE.md`: how to build for Windows, the manifest, the icon, the GUI subsystem and where the log goes.
**Tests:** the manifest text (golden); the `.syso` written and read back with `debug/pe` and a small resource-directory reader (the entries, their sizes, the relocations); the icon sizes; the version info; the command lines the builder runs (a fake runner); the overlay file; route tests of every tag.
**Wine tests:** the built `.exe` of `examples/counter` runs under Wine; it reports `PerMonitorV2` awareness (`GetThreadDpiAwarenessContext`), the code page 65001, and loads its icon resource (`LoadImageW` of resource 1 has the right size); an `.exe` built with plain `go build` still runs and sets DPI awareness at run time; a GUI build has no console subsystem (`debug/pe` header) and a `-console` build has; `shot` of the counter equals the `soft` golden within the `d3d12` tolerance; `run` leaves nothing behind.
**Proof:** F, `28.14-windows-counter-exe.png`: the counter built by `glutter-windows` and captured with `shot`.
**Manual checks:** on the owner's PC: copy the `.exe` that `go run ./cmd/glutter-windows build ./examples/gallery` made (or build it on Windows with the same command) and double-click it: no console window opens, the icon shows in Explorer and the task bar, Properties → Details shows the version, the window is sharp at 150%, SmartScreen asks once (the file is unsigned); start it from a terminal and read the log lines there. Record the answers.

### 28.15 Every example, performance, and the manual checks
**Goal:** every example runs on Windows under Wine with Direct3D 12, the numbers are recorded, and the manual checks of the milestone are written for the owner.
**Read:** SPEC §15 (Windows), §19, §25.13, §17.
**Files:** `examples/examples_test.go` (and a Windows-tagged file), the examples where Windows needs a change, `cmd/glutter-windows` (`perf`), `scripts/perf-windows.sh`, `docs/PERFORMANCE.md`, `docs/MANUAL-CHECKS.md`, `README.md`, `docs/GUIDE.md`.
**Tasks:**
1. `examples/examples_test.go` builds every example, `tray` and `multiwindow` included, for `windows/amd64` and `windows/arm64` with `CGO_ENABLED=0` (skipped under `-short`); a Wine test (`TestEveryExampleDrawsItsFirstFrameUnderWine`, `glutter_wine`) starts each under Wine with `d3d12` and waits for its first frame; fix what an example needs (a path, a file name, a feature the platform lacks, with the example's header saying so).
2. `glutter-windows perf` and `scripts/perf-windows.sh` (the design of `glutter-android perf`, SPEC §24.13, and `internal/perfstats`): the cold start to the first frame (the engine's own timing, with an empty and with a warm shader cache), the idle CPU (the CPU time of the Wine processes of the prefix over a window), the engine's frame statistics (`GLUTTER_FRAME_STATS=1`), the memory (`/proc` of the Wine processes), the `.exe` size of the counter and the gallery. `docs/PERFORMANCE.md`: the Windows table, saying plainly that these are vkd3d on lavapipe under Wine and not a GPU on Windows.
3. `docs/MANUAL-CHECKS.md`: a section for each step 28.2 to 28.14 with the checks its **Manual checks** line lists, the command to run, numbered questions, "Status: pending" and "Answers: (not yet recorded)", as the phone sections have; and a section 28.15 for the whole sweep on a PC: every example, the gallery's scrolling and theme switch, the idle CPU in Task Manager, the cold start by eye, the memory of the counter. The introduction of the file says that it now also holds the checks of a Windows PC.
4. Propose the Windows performance targets for SPEC §15 from the Wine numbers and the owner's PC (the owner confirms them); `README.md`: a "Windows" section (what to install, building, running under Wine, what Windows lacks: toast buttons, snap layouts, `DirectManipulation`); `docs/GUIDE.md`: notes for Flutter Windows developers (no plugins, the `.exe` tool, the manifest, the log).
**Tests:** the examples build; the first-frame test; the plain-Go parts of `perf` (parsing, as the Android one).
**Wine tests:** as above.
**Proof:** none.
**Manual checks:** the sweep of task 3.
**Done when:** the default "Done when", every example draws its first frame under Wine, the Windows numbers are in `docs/PERFORMANCE.md`, and `docs/MANUAL-CHECKS.md` has a section for each step.

### 28.16 Font fallback that loads only the fonts that can help (Emetos)
**Goal:** a character that no font of the style has, a tab or another control character among them, no longer makes Glutter parse every font file of the style. A candidate is ruled out by its character map before its outlines are read, so a font that covers only another script costs nothing until that script is shown.
**Why:** Emetos found it (issues file, "A character no font has makes Glutter load every font of the style"). The first row with a tab made `styleFonts.ResolveFace` (`internal/text/layout/engine.go`) parse the 62 files that fontconfig lists for `monospace`: 116 MB on disk, 170 MB of Go heap and 300 MB more resident memory, on the UI thread in the first frame. Emetos works around it with a tiny font that has a glyph for U+0009.
**Read:** SPEC §10 (text, fonts, the fallback order of 1.2), §12.10 (the `Collection` and fallback rows), the 22.1–22.3 and 27.x entries of `docs/IMPLEMENTED.md` about fonts.
**Port from:** none. HarfBuzz and go-text's `shaping` take the default-ignorable characters out of the run and give the control characters no glyph of their own. `fontscan`'s footprints keep a font's coverage (`Runes`) without parsing it.
**Files:** `internal/text/layout/engine.go` (`styleFonts`), `internal/text/fonts.go` (`Collection`: coverage, lazy loading), `internal/text/scan.go` and the matchers if they must hand out coverage (`fontconfig`, `androidfonts`, `winfonts`, the web's), SPEC §10 and §12.10.
**Tasks:**
1. Control characters (general category `Cc`) and default-ignorable code points resolve to the first face of the style that loads, without the search and without the system fallback. The shaper keeps measuring a tab as before (tab stops of 1.4).
2. Each candidate's coverage is known before it is parsed: from the matcher where it has it (fontconfig's charset, `fontscan` footprints, the `winfonts` and `androidfonts` caches), else from the `cmap` table alone, read once and kept. A candidate whose coverage lacks the rune is skipped without `font.NewFont`.
3. A face is parsed only when it is chosen. If go-text can read glyph outlines lazily (a glyph the first time it is drawn), use that; if it cannot without changing go-text, say so and keep the full parse of the chosen face only.
4. A rune that no candidate covers goes to the system fallback (unchanged), then to the first face's missing-glyph box, still without parsing the rest.
**Tests:** a style of many fonts, written for the test, in which only the first and the last have Latin: a tab, `\r`, U+200B, U+FEFF and an unassigned code point parse at most the first file (count `Collection.load` calls); a Greek letter that only the third font has parses the first and the third; the result per rune is cached as before; the measures of a paragraph with tabs are unchanged (the tab-stop tests of 1.4 and a golden of a paragraph with tabs); the coverage from each matcher and from a `cmap` alone; a font whose `cmap` cannot be read is skipped with `FontLoadFailed` logged once. A benchmark: the first layout of a 10,000-line Go file with tabs against a style of 60 fonts, with its allocations, before and after (recorded in the entry).
**Display tests:** none.
**Proof:** none.
**Manual checks:** none; Emetos removes its `fonts.ControlsFont` workaround and checks its memory (its own issue).

### 28.17 `TreeSliverNode.SetExpanded` (Emetos)
**Goal:** an app can say which nodes of a `TreeSliver` are expanded without making the nodes again, also for nodes that are not shown.
**Why:** Emetos found it (issues file, "TreeSliver cannot set the expansion of a node that exists"). Its tree keeps the expansion in its model, by key, and must build every node again at each change, because a node's expansion can be given only when the node is made or toggled while it is shown.
**Read:** SPEC §11.3 (`TreeSliver`), §12.10 (the `sliver_tree.dart` rows), the 22.7 and 27.9 entries of `docs/IMPLEMENTED.md`.
**Port from:** none (Flutter's `TreeSliverNode._expanded` is private; the setter is a Glutter addition).
**Files:** `widgets/sliver_tree.go`, its tests, `material/widgets_reexport_gen.go` if it changes, SPEC §12.10.
**Tasks:**
1. `TreeSliverNode.SetExpanded(expanded bool)`: sets the expansion of a node whether it is shown or not; a node with no children and not lazy stays collapsed (as the constructor does). When the node belongs to a shown `TreeSliver`, the tree makes its active nodes again (the callback of 27.9), with no animation; `TreeSliverController.ExpandNode` and `CollapseNode` keep animating. Setting the value the node has does nothing.
2. The `// Deviation:` comment and a §12.10 row: a Glutter addition, because Go apps keep their tree's state in a model and Dart apps rebuild the nodes.
**Tests:** a hidden descendant expanded with `SetExpanded`, then its ancestor expanded: the descendant's children show; a shown node expanded and collapsed: the rows, `Parent()` and `Depth()` are right and no animation runs; a leaf stays collapsed; a lazy node expands; the same value twice rebuilds nothing; `NodeKey` with nodes made again keeps the model's expansion when the app sets it with `SetExpanded` (Emetos's case); the controller still animates.
**Proof:** none.
**Manual checks:** none.

### 28.18 The drag area of `WindowDecoration` (Emetos)
**Goal:** only the title and the empty part of `WindowDecoration`'s bar move the window, maximize it on a double click and open the window menu on a right click; the widgets an app puts in `Leading` and `Actions` get their own gestures only.
**Why:** Emetos found it (issues file, "WindowDecoration: the whole bar is the drag area"). Its menu button, workspace tabs and `+` button sit in the bar, so a drag of a tab moved the window, a double click maximized it and a right click asked for the window menu. Emetos works around it with a `GestureDetector` of no-op callbacks around its strip.
**Read:** SPEC §8.2 (step 3, `WindowDecoration`), §12.10 (the `WindowDecoration` row), the 27.4 entry of `docs/IMPLEMENTED.md`.
**Port from:** none (Glutter's own widget).
**Files:** `widgets/window_decoration.go`, its tests, SPEC §8.2 and §12.10.
**Tasks:**
1. The bar is a row of the `Leading` widgets, the draggable area and the `Actions` widgets; the gesture detector of the window (pan, double tap, secondary tap) covers the draggable area only: the title (default or `Title`) and the space around it, which takes the rest of the row. The window buttons stay as they are.
2. Update the §12.10 row and the sentence of §8.2.
**Tests:** a drag starting on a `Leading` widget and one on an `Actions` widget do not call `PerformWindowAction`; a drag on the title and one on the empty space between the title and the actions start a move; a double tap and a right click the same way; a `GestureDetector` with `OnTap` in `Actions` gets its tap at once (no wait for a double tap of the bar); the golden of the bar is unchanged.
**Proof:** none.
**Manual checks:** none; Emetos removes its no-op detector (its own issue).

### 28.19 Release 1.8
**Tasks:** confirm and record in `IMPLEMENTED.md`:
- the standard checks including the four Windows ones, the display tests, the browser tests, the device tests and the Wine tests (`GOOS=windows go test -tags glutter_wine -exec "$PWD/scripts/wine-exec.sh" -p 1 ./...` and the host-side Wine tests)
- the manual checks of 28.2 to 28.15 on a Windows PC, with the owner's answers in `docs/MANUAL-CHECKS.md`
- the performance numbers (28.15) and the Windows targets of SPEC §15, confirmed by the owner
- the API check and the error rules check
- the Emetos fixes of 28.16 to 28.18, listed in the release notes with what Emetos can remove (`fonts.ControlsFont`, the rebuilt tree nodes, the no-op detector of its title bar)

Then update the SPEC's version paragraph (the 1.8 sentence: Windows, and the fixes for Emetos) and the scope table, the README, and `third_party/flutter` NOTICE entries for the Windows embedder, and **ask the user** before tagging `v1.8.0` and pushing the tag.

### 28.20 An error boundary (Emetos, 1.8.1)
**Goal:** `widgets.NewErrorBoundary`, a widget that takes over the build errors of its subtree: when a widget under it fails to build, the boundary is told first and builds a fallback in place of its whole child, in the same frame.
**Why:** Emetos found it (issues file, "No error boundary: the host cannot tell whose widget failed"). Its views of extensions must show "<Name> failed" and turn the extension off in that tab when anything the extension built fails. Glutter catches a panic at the element that failed and tells only the one `OnError` of the application, so a panic deeper than the extension's own `Build` (in a widget it built, or in a `State` it wrote) is invisible to Emetos's `recover`. The owner chose (2026-10-26): the boundary replaces its whole subtree, and the step goes in 1.8.1.
**Read:** SPEC §14.2 (recovery points, `ErrorWidgetBuilder`, D15), §12.10, §12.13; `widgets/build_owner.go` (`errorWidgetFor`, `reportError`), `widgets/component_element.go` (`PerformRebuild`), `widgets/layout_builder.go`, `widgets/scroll_delegate.go`.
**Port from:** none (Flutter has no error boundary; a deviation, recorded in §12.10).
**Files:** `widgets/error_boundary.go` and its tests, `widgets/build_owner.go` and the places that call `errorWidgetFor`, the generated `material` re-exports (`go generate ./...`), SPEC §14.2 and §12.10.
**Tasks:**
1. `NewErrorBoundary(fallback func(bc BuildContext, err error) Widget, child Widget, opts ...ErrorBoundaryOpts) *ErrorBoundary`, with `ErrorBoundaryOpts{Key, OnError func(err error)}`. A `nil` fallback means the owner's error widget (`ErrorWidgetBuilder`, or `ErrorWidget`).
2. Every failure that now becomes an error widget (`errorWidgetFor`: a panic in `Build`, in `UpdateChild`, so in the `InitState`, `DidUpdateWidget` and `DidChangeDependencies` of a state below, in a `LayoutBuilder`'s builder, in a lazily built child of a sliver) finds the nearest `ErrorBoundary` above the failing element. If there is one:
   - its `OnError` is called with the error, once per error, before the owner's `OnError` (which is still called, as now);
   - the boundary becomes failed and is rebuilt in the same build pass: it builds `fallback(bc, err)` (the first error) in place of its child, so the failed subtree is unmounted (its states disposed) and never reaches a painted frame.

   Marking an ancestor dirty while a descendant builds is what Flutter forbids. Give the boundary its own way into the current build scope (the dirty list is re-sorted, as `dirtyElementsNeedsResorting` does), not a post-frame callback, which would paint the error widget for a frame.
3. A failed boundary stays failed: a rebuild of it, or a new child from its parent, builds the fallback again, and the old child is not tried again. A new `Key` makes a new boundary (the way to try again). Further errors while it is failed are only reported.
4. Boundaries nest: the nearest one takes the error. A panic in a fallback (or in the `OnError` callback) is an error of the boundary's own element, and goes to the next boundary up, or to the owner as now.
5. Without a boundary nothing changes. It works in release builds too.
6. SPEC: §14.2 says what a boundary does; a §12.10 row records the deviation (no Flutter counterpart). The doc comments end with `// Flutter: no counterpart`.
**Tests (table-driven):**
- a panic in a child's `Build`, in a grandchild state's `Build`, in a deeper `InitState`, in a `LayoutBuilder` and in an item of `ListView.builder` each show the fallback in the same pump, and the boundary's `OnError` is called before the owner's;
- the failed subtree's states are disposed;
- after the failure, a rebuild of the parent with a new child still shows the fallback; a new `Key` shows the child again;
- nested boundaries: the inner one takes it; a panic in the inner fallback goes to the outer one;
- a `nil` fallback shows `ErrorWidget`;
- without a boundary, the error widget takes the failed widget's place as before;
- a golden is not needed.
**Proof:** none.
**Manual checks:** none; Emetos replaces its `recover` in `ui.NewBuilder` and in its box with a boundary (its own issue).

### 28.21 The room of `WindowDecoration`'s `Leading` widgets (Emetos, 1.8.2)
**Goal:** the `Leading` widgets of `WindowDecoration` can take the room of the bar that is left after the gap, the `Actions` and the window buttons: they are laid out before the title, as one group with a bounded maximum width, and the title (the draggable area) gets what they leave.
**Why:** Emetos found it (issues file, "WindowDecoration: a Leading widget cannot take the room that is left"). Since 28.18 its menu and workspace tabs are `Leading` widgets, so that a drag, double click or right click on a tab is the tab's. But `Leading` widgets are children of the bar's row, next to the title, which is an `Expanded` with flex 1:
- a `Leading` widget with an unbounded width gets no limit;
- a `Flexible` one shares the row with the title, so the tabs get half of the bar at most.

Emetos works around it with a `ConstrainedBox` whose maximum is the window's width less Glutter's private constants (the 12-pixel gap and 46 pixels for each window button). If those constants change, the workaround breaks.
**Read:** SPEC §8.2 (step 3, `WindowDecoration`), §12.10 (the `WindowDecoration` row), the 28.18 entry of `docs/IMPLEMENTED.md`, `widgets/window_decoration.go`, and `widgets/navigation_toolbar.go` (Flutter's `NavigationToolbar` lays out leading and trailing first and gives the middle the rest, through a `MultiChildLayoutDelegate`).
**Port from:** none (Glutter's own widget); the layout follows `NavigationToolbar`'s delegate.
**Files:** `widgets/window_decoration.go`, its tests, SPEC §8.2 and §12.10.
**Tasks:**
1. Lay out the bar (the part before the window buttons) in this order, in the bar's height:
   - the `Actions` first, as one group with loose constraints;
   - then the `Leading` widgets, as one group (a row of minimum main-axis size) whose maximum width is what is left after the 12-pixel gap and the `Actions`. A loose `Flexible` among them takes what it needs, up to that room, and an `Expanded` one takes all of it;
   - then the title's draggable area, which gets exactly the rest (it may be 0 wide).

   The gap and the title area keep the window's gestures, as in 28.18. Right-to-left text direction mirrors the bar, as the `Row` did.
2. Nothing changes for a bar whose `Leading` and `Actions` have fixed widths: same positions, same golden.
3. Update the sentence of §8.2 and the §12.10 row. The doc comment of `WindowDecorationOpts.Leading` says what room the widgets get.
**Tests (table-driven):**
- a `Leading` group with a loose `Flexible` wider than the bar ends where the `Actions` begin, and the title area is 0 wide;
- a narrow one takes its own width, and the title gets the rest;
- an `Expanded` `Leading` takes all the room;
- a `Leading` widget with an unbounded width (for example a `Row` of max size) is limited and does not overflow;
- `Actions` keep their width; the window buttons are unchanged for each set of capabilities;
- right to left mirrors;
- the 28.18 gesture tests still pass (a drag on the gap and on the title moves; one on `Leading` does not);
- the existing golden is unchanged.
**Proof:** none.
**Manual checks:** none; Emetos removes its `ConstrainedBox` and the copied constants (its own issue).

### 28.22 The ellipsis cuts the last line at a character (Emetos, 1.8.3)
**Goal:** a line that ends with an ellipsis keeps every character that fits before the ellipsis, as Flutter does: the last line allowed by `MaxLines` is filled by grapheme cluster, not by word, before the ellipsis is added.
**Why:** Emetos found it (issues file, "Text with `TextOverflowEllipsis`: a line with no place to break gets only the ellipsis"). A one-line `RichText` with `Overflow: TextOverflowEllipsis` and a text with no place to break that is wider than the box (`fmt.Println(Shout(Greet("ev...`, after the tab of the indentation) shows only the ellipsis. Flutter shows the characters that fit, then the ellipsis.

The likely cause: `paragraph.Layout` (`internal/text/layout/lines.go`) breaks the lines by word first, keeps `MaxLines` of them, and only then calls `ellipsize` on the last kept line. When the word after the leading white space does not fit, it goes to a second line, which is dropped, so the last line is the white space and the ellipsis. A text with spaces shows the words that fit and then the ellipsis, where Flutter cuts inside the word that does not fit. With soft wrap on, a break after an ellipsis character `…` in the text leaves only that on the line.

Emetos works around it in its peek view: no soft wrap, an ellipsis only when the row has spaces, otherwise a clip, and the text before the match cut short.
**Read:** SPEC §10 (paragraph layout, `MaxLines`, ellipsis); `internal/text/layout/lines.go` (`Layout`, `breakLines`, `ellipsize`); `rendering/paragraph.go` (how `softWrap`, `MaxLines` and `Overflow` reach the paragraph); Flutter's `painting/text_painter.dart` (`ellipsis`, `maxLines`) and `rendering/paragraph.dart`.
**Port from:** the behavior of SkParagraph's line wrapper, which is not in `reference/`. When the line being built is the last one allowed (`maxLines` reached, or the only line when there is no limit) and the paragraph has an ellipsis, it takes clusters up to the width less the ellipsis, from the start of the line, without looking for a word break, and adds the ellipsis.
**Files:** `internal/text/layout/lines.go` and its tests, a widget-level test (`rendering` or `widgets`), and SPEC §10.
**Tasks:**
1. First write a test that shows the bug, at the paragraph level and through `RichText`. For each of these, give the result before and after the fix in the report:
   - a tab and then `fmt.Println(Shout(Greet("ev...`, wider than the box, with `MaxLines: 1`;
   - the same with `SoftWrap: false`;
   - a text with spaces;
   - a text that contains `…`.
2. When a line is the last one kept and the paragraph is ellipsized, the line runs from its start to the next hard break (or the end of the text), and `ellipsize` cuts it at the last grapheme cluster boundary where the text and the ellipsis fit. Trailing white space before the ellipsis is dropped, as now.
3. Nothing changes for text that fits, for text without an ellipsis, or for lines before the last.
4. If the box is narrower than the ellipsis itself, the ellipsis alone is shown, as in Flutter.
5. SPEC §10: say how the last line is cut.
**Tests (table-driven):**
- the four cases of task 1 keep the characters that fit, then `…`;
- the width of the line is at most the box's width;
- a right-to-left text;
- a line made of a combining sequence and an emoji with ZWJ is not cut inside a cluster;
- a box narrower than the ellipsis;
- `MaxLines: 2`: the first line still breaks at a word, and the second is cut at a character;
- `SoftWrap: false` without `MaxLines`.

A golden that changes because an ellipsized line now keeps more characters is looked at, and the reason is written in the report.
**Proof:** none.
**Manual checks:** none. Emetos removes its workaround in `peek_widget.go` (its own issue).

## M29: SVG images (1.9)

The owner asked on 2026-10-26 for Glutter to draw SVG images. They chose:
- `flutter_svg`'s API, in a package `svg`;
- the larger scope: text, masks, filters and CSS as well as shapes;
- a release, 1.9.

SPEC §26 and D27 are the design. The parser and the painter are Glutter's own, in plain Go, so the steps need no platform work. Each step is checked on every platform by the standard checks. The reference images of the tests are made once with `rsvg-convert` (SPEC §26.6), which is installed on the development machine.

`flutter_svg` is not in `reference/`. Step 29.1 adds `flutter_svg` (MIT) at a pinned version to `scripts/fetch-reference.sh` and to `reference/VERSION`, for the API's names, defaults and sizing only. The parser and the painter follow the SVG 1.1 and SVG 2 specifications, not `flutter_svg`'s compiler. Glutter's code ports none of `flutter_svg`'s code, so no license file is needed. If a step ports some, it adds the license and a NOTICE entry (SPEC §20).

### 29.1 The SVG parser
**Goal:** `internal/svg` parses an SVG document into a tree with the values resolved: XML, the CSS cascade, units, colors, transforms, path data, and `use`/`defs`/`symbol` references. Nothing is drawn yet.
**Read:** SPEC §26 (all), D27, §14.4, §12.
**Port from:** none (the SVG specifications). `flutter_svg` is fetched for the API of 29.4.
**Files:**
- `internal/svg/` (`doc.go`, the XML reader, `css.go`, `values.go` for lengths, colors, paints and transforms, `pathdata.go`, the tree and its resolution), its tests, and `internal/svg/testdata/` (the first test documents);
- `scripts/fetch-reference.sh`, `reference/VERSION`.
**Tasks:**
1. Fetch `flutter_svg` at its latest release into `reference/flutter_svg`, and pin it in the script and in `VERSION`.
2. The XML reader:
   - elements and attributes of the SVG and xlink namespaces;
   - `CDATA` and entities;
   - the text content of `text`, `tspan` and `style`;
   - every other element skipped with its children.
3. CSS:
   - the `style` attribute and `<style>` sheets, with the selectors of §26.2;
   - specificity, `!important` and order;
   - `@media` and `@import` skipped;
   - the properties that inherit, inherited down the tree.
4. Values, as SPEC §26.2 lists them: lengths and their units (a percentage resolved against the viewport, in its dimension), colors, paints, `transform`, `viewBox`, `preserveAspectRatio`, and number lists. An invalid value is ignored, except where §26.5 says it is an error.
5. Path data:
   - every command, absolute and relative;
   - the shorthand of numbers (`1.5.5`, `-1-2`) and the flags of arcs written without spaces;
   - the output is a `ui.Path`, with `ui.Path`'s SVG arc;
   - drawing stops at the first error, as browsers do.
6. The resolved tree:
   - each element with its computed style;
   - `use` expanded, with cycles cut and a depth limit;
   - `symbol` and nested `svg` as viewports;
   - `switch` (`requiredFeatures`, `systemLanguage` against the locale);
   - references by id (gradients, patterns, clip paths, masks, filters, markers ignored) kept for the painter.

   `Document.Size()` comes from `width`/`height` and the `viewBox`.
7. The errors of §26.5, each with a route test.
8. `BenchmarkParse` on a small icon and on a large document made by the test.
**Tests (table-driven):**
- every unit, color syntax, transform and path command;
- the CSS cascade: specificity, `!important`, inheritance, the `style` attribute over the sheet over the presentation attribute;
- `use` with a transform and an override, and a cycle;
- the size of a document with and without `viewBox`, `width` and `height`;
- the errors and their routes;
- a document of another namespace in the tree is skipped.
**Proof:** none.
**Manual checks:** none.

### 29.2 The SVG painter
**Goal:** a parsed document is recorded into a `ui.Picture` that draws its shapes, strokes, gradients, patterns, clip paths, group opacity, nested viewports and `data:` images (SPEC §26.3, except text, masks and filters), matching reference renders made by `rsvg-convert`.
**Read:** SPEC §26.3, §26.4, §26.6, §9 (what the renderers draw), `ui` (`Canvas`, `Path`, `Paint`, `Gradient`, `ImageShader`, `SaveLayer`).
**Port from:** none.
**Files:** `internal/svg/paint*.go`, its tests, `internal/svg/testdata/` (documents and their reference PNGs), the reference harness.
**Tasks:**
1. The harness (SPEC §26.6):
   - for each test document, a reference PNG made by `rsvg-convert` at a fixed size (`-update-references`, which skips with the install command when `rsvg-convert` is missing);
   - a test that renders each document with the software renderer and compares it with its reference within a tolerance.
2. The viewport: `viewBox` and `preserveAspectRatio` (every alignment, `meet`, `slice`, `none`), nested `svg` and `symbol` viewports, `overflow`.
3. The shapes of §26.3 (the `rx`/`ry` rules of `rect`), fill and stroke with every property, dashes, and `non-scaling-stroke`.
4. Paint servers:
   - linear and radial gradients with `href` inheritance, units, transforms and spread methods (focal radial gradients through the two-point conical gradient);
   - patterns, as an image shader of the tile, recorded at the device scale.
5. Group opacity through a save layer (only when the group has more than one child or overlaps itself), `clip-path` with its rules and units, and `display` and `visibility`.
6. `<image>` with a `data:` URL: raster formats through `ui`'s decoder, and SVG through this painter.
7. At least 40 test documents, one feature each, with their references.
**Tests:**
- the reference comparison of every document;
- unit tests of the viewport transform for every `preserveAspectRatio`;
- dashes along a curve.

A golden that differs from its reference beyond the tolerance is explained in the report, never widened silently.
**Proof:** none.
**Manual checks:** none.

### 29.3 Text, masks, filters and markers
**Goal:** SVG text, masks and the filters of SPEC §26.3 are drawn, matching their reference renders.
**Read:** SPEC §26.3 (text, masks, filters, color), §10 (paragraphs and fonts), `ui`'s image filters, color filters and save layers.
**Port from:** none.
**Files:** `internal/svg/text.go`, `mask.go`, `filter*.go`, their tests and test documents. Additions to `ui` if a filter needs one (an image filter run on the CPU), with their goldens.
**Tasks:**
1. Text:
   - `text`, `tspan` and `textPath`, as SPEC §26.3 says;
   - glyphs from Glutter's layout (shaped runs, the app's font collection);
   - placement by the per-character `x`, `y`, `dx` and `dy` lists, `text-anchor` per chunk, letter and word spacing, decorations, and fill and stroke of the glyphs;
   - the reference renders are made with the same font, a bundled one (Roboto), so that they can be compared.
2. Masks: a luminance or alpha mask through a save layer, a color matrix and `dstIn`, with the units of the mask.
3. Filters:
   - the filter region and its units;
   - the graph of primitives with named results;
   - each primitive of §26.3 on `ui.ImageFilter`s where they exist (blur, offset, color matrix, morphology, compose);
   - the rest on an image filter that runs on the CPU, added to `ui` with the soft renderer's support and, for the GPU renderers, a readback or an equivalent. Say which in the report.
   - `color-interpolation-filters` (linearRGB through the gamma color filters).
4. Markers (`marker-start`, `marker-mid`, `marker-end`, `orient`, `markerUnits`), which icons with arrows use.
5. Test documents for each, with their references.
**Tests:**
- the reference comparison;
- a filter graph with named results;
- a mask in a group with opacity;
- text with each anchor and in right-to-left text;
- the GPU renderers draw the filtered documents within the tolerance of `soft` (the display tests, `glutter_display`).
**Proof:** none.
**Manual checks:** none.

### 29.4 The `svg` package and its example
**Goal:** apps draw SVG with `flutter_svg`'s widgets. `svg.NewSvgPictureAsset`, `.String`, `.Memory`, `.File` and `.Network` load, cache and size the picture as SPEC §26.1 says, and `examples/svg` shows it on every platform.
**Read:** SPEC §26.1, §12, §13 (goroutines and `Post`), `reference/flutter_svg` (`svg.dart`, `SvgPicture`, `SvgTheme`, the loaders, the cache), `widgets/image.go` (how `Image` loads and sizes).
**Port from:** `flutter_svg`'s `SvgPicture` and its loaders: names, defaults and sizing only (MIT; if code is ported, the license goes in `third_party/flutter_svg` with a NOTICE entry).
**Files:**
- `svg/` (`svg.go`, `picture.go`, `loaders.go`, `cache.go`, `theme.go`) and its tests;
- `examples/svg/` with its assets and a golden;
- `examples` registry entries (the Windows, web and Android example lists);
- `cmd/glutter-gen` apicheck if the new package needs it.
**Tasks:**
1. The widget, its options and its sizing, as SPEC §26.1 says. `MatchTextDirection` mirrors in RTL. `ColorFilter` is applied to the picture. `SemanticsLabel` (or the document's `title`) goes to the semantics tree, unless `ExcludeFromSemantics`.
2. The loaders: assets through the bundle of the context (or `Bundle`), files, and network through `net/http` with `Headers`. Each runs on a goroutine with a context cancelled when the widget leaves the tree, and parses there. `PlaceholderBuilder` shows while it loads, and `ErrorBuilder` on failure (the error is also reported, §26.5).
3. `svg.Cache`: an LRU by source and theme with a byte limit, shared by the widgets of an app (no global state: it hangs off the binding, or is given in the options).
4. `svg.Parse`, `Document.Picture` and `Document.Size` (SPEC §26.1).
5. `examples/svg`: a grid of icons (`currentColor` from the theme, with a `ColorFilter`), a logo with gradients, a card with SVG text and a drop-shadow filter, and the same icon at three sizes. Add it to the lists of examples that run on the web, Android and Windows.
**Tests:**
- widget tests for each loader;
- the sizes: `Width` only, `Height` only, both, neither, under tight and loose constraints, and each `Fit`;
- the cache: one parse for two widgets of the same asset, and an eviction by size;
- `ErrorBuilder` and `PlaceholderBuilder`;
- semantics;
- the example's golden.
**Proof:** a screenshot of `examples/svg` on Linux in `docs/implemented-proof-images/`.
**Manual checks:** none.

### 29.5 Release 1.9
**Goal:** 1.9 is checked and published with the tag `v1.9.0` (CLAUDE.md, "Tags": a finished version is tagged without asking).
**Read:** the 28.19 entry of `docs/IMPLEMENTED.md` (the form of a release), SPEC §1, §3, §18.
**Tasks:**
1. Documents:
   - the SPEC's version paragraph gets a 1.9 sentence, and the scope of §3 a "Scope of 1.9" paragraph;
   - `README.md` gets a "What is new in 1.9" note and a short SVG example;
   - `docs/GUIDE.md` gets a section "13. SVG images", with `flutter_svg` in Dart beside `svg` in Go (checked by the docs test).
2. The checks:
   - the standard checks;
   - the display tests and the browser tests;
   - `examples/svg` under Wine, and on one Android emulator only (one AVD at a time, for memory);
   - the parse benchmark against the numbers of §26.4.
3. Merge into `main`, tag `step-29.5` and `v1.9.0`, and publish milestone M29 (rule 12): `git push --follow-tags origin main`, which takes steps 29.2 to 29.5 and their tags to GitHub.
**Proof:** none.
**Manual checks:** none.

### 29.6 A public software rasterizer that draws on any goroutine (Emetos, 1.9.1)
**Goal:** an app can turn a scene into pixels without `gluttertest`: package `offscreen` has a public software `ui.Rasterizer`, a `Surface` that redraws only what changed since its last frame, and a `Detacher` that makes a copy of a scene that any goroutine can draw. Full frames of the software renderer get faster.
**Why:** Emetos found it (issues file, "No public way to draw a view off screen and to read its pixels"). Its Remote Work view needs the pixels of a second view of a workspace, drawn when it changes and handed to a goroutine that sends them. Today it uses a `gluttertest.Tester` as its rasterizer, which puts `gluttertest` and the `testing` package in the editor binary. It draws on the UI thread, where a 1280x800 frame takes about 85 ms, so its caret does not blink. Drawing on another goroutine races with the next frame's layout (`go test -race`):
- a paragraph draws by asking the text engine that laid it out (`ui.ParagraphPainter`, through `textdraw.Expand`), whose caches have no lock;
- the paragraph object itself is laid out again by the next frame (`paragraph.Layout`).

The manager measured the software renderer on 2026-10-27: a 1280x800 `Scaffold` with no text takes about 50 ms, and one full of text about 45 ms. The time goes to compositing pixels one at a time (`composite`, `mix`, `store`), not to the glyphs. The owner chose (2026-10-27): 1.9.1, and the offscreen view on a binding of its own (step 29.7). The package and its names are SPEC D28.
**Read:** SPEC D28, §9.1 ("Damage"), §9.2, §9.9 (new, written with this step), §13, §12.10; `internal/gpu/soft/soft.go` (`Render`, `ImageRasterizer`, `composite`), `internal/gpu/damage.go` (`DamageTracker`), `internal/gpu/textdraw/textdraw.go`, `internal/text/layout/paint.go` and `lines.go`, `ui/picture.go` (`Rasterizer`), `ui/display_list.go`, `ui/scene.go`.
**Port from:** none (Flutter has no public software rasterizer; a deviation, recorded in §12.10).
**Files:** the new package `offscreen` (`rasterizer.go`, `surface.go`, `detach.go` and tests), `internal/gpu/soft` (fast paths, a region to draw), `ui` (what the detacher needs: the way to make a picture from operations, a paragraph that replays a recorded drawing), SPEC §9.2, §9.9 and §12.10.
**Tasks:**
1. `offscreen.NewRasterizer() *Rasterizer`, a `ui.Rasterizer` that draws with `internal/gpu/soft`. It keeps nothing after `RasterizeScene` returns, and two goroutines may use one at once. Apps use it in `ui.ToImageOpts{Rasterizer: ...}` (for `Scene.ToImage`, `Picture.ToImage`, `RenderRepaintBoundary.ToImage`).
2. `offscreen.NewDetacher() *Detacher` and `Detach(scene *ui.Scene) *ui.Scene`. Run it on the UI thread. It returns a scene that draws the same pixels and holds nothing that the framework changes later:
   - each `OpDrawParagraph` is replaced by what its paragraph draws now (`PaintParagraph`'s `ParagraphDraw`s, recorded and replayed), so the copy never reaches the text engine or the paragraph. Glyph snapping still happens when the copy is drawn, under its transform;
   - the pictures inside pictures are detached too;
   - a picture without a paragraph is kept as it is;
   - the copy of a picture is reused when the next scene has the same picture, so that the damage tracker still sees what did not change. The `Detacher` keeps only the copies of the last scene.

   Find anything else a scene reaches that the UI thread changes later (images that are disposed, retained layers, paths) and make the copy safe from it; say in the report what you found. Everything the renderer reads while drawing must also be safe for two goroutines at once (no lazy cache inside a `ui.Path` or a `ui.Image`).
3. `offscreen.NewSurface(width, height int) *Surface`, the pixels of one view across frames, used on one goroutine at a time:
   - `Draw(scene *ui.Scene) (damage []image.Rectangle, err error)` uses a `gpu.DamageTracker`. It redraws only the damaged rectangles over the last frame: each is cleared, then the scene is drawn with that rectangle as its clip, so the pixels equal those of a whole redraw;
   - the first frame, a new size (`Resize`) and a scene the tracker cannot bound redraw everything;
   - a scene that changes nothing returns no damage and draws nothing. This deviates from the tracker's rule for windows (the same scene again means draw everything): the surface's pixels are never lost;
   - `Image() *image.RGBA` is the last frame, valid until the next `Draw` or `Resize`.
4. The software renderer:
   - `soft` gets the way to draw a scene into a region of an existing image;
   - profile a full 1280x800 frame (a `Scaffold` with an `AppBar`, and the same full of 14-pixel text) and add fast paths where the time goes: an opaque solid color with `srcOver` (or `src`) over full coverage written without blending, rows of a mask that are full or empty, no allocation for each draw where one can be avoided;
   - pixels stay exactly the same: every golden test passes unchanged, and the display and parity tests that compare with `soft` keep their tolerances;
   - targets, in `BenchmarkFrame*` benchmarks in `offscreen`: a 1280x800 frame with no text under 10 ms, the frame full of text under 25 ms, a frame where only a caret (2x18 pixels) changed under 2 ms through a `Surface`. Report the numbers before and after. If a target cannot be met, report where the time goes.
5. `internal/gpu/soft.ImageRasterizer` and `gluttertest` keep working; `gluttertest.Tester.Screenshot` may use the new code.
6. SPEC: the new §9.9 gets the decisions of the step; §9.2 says what `soft` serves now; a §12.10 row records the public rasterizer (no Flutter counterpart). The doc comments end with `// Flutter: no counterpart`.
**Tests (table-driven):**
- `RasterizeScene` gives the same pixels as `soft.Render` for the `gputest` proof scenes, and as `Tester.Screenshot` for a widget tree;
- `Surface.Draw` after a change gives the same pixels as a whole redraw, for: text typed into a field, a moved caret, a scrolled list, a color change under an opacity layer, a backdrop blur (the tracker damages everything), a resize; and returns the damage rectangles it drew;
- a scene that changes nothing: no damage;
- the detached copy draws the same pixels as the scene. After the paragraphs of the source are laid out again with another width, and its images disposed, the copy still draws the old pixels;
- a race test (`go test -race`): the UI goroutine builds frames of a widget tree whose text changes and is laid out again, while another goroutine draws the detached scenes through a `Surface`. The race detector reports nothing;
- a picture that did not change keeps its detached copy (the damage of an unchanged frame is empty);
- the benchmarks of task 4.
**Proof:** none (no visible change); the numbers go in the report and `IMPLEMENTED.md`.
**Manual checks:** none.

### 29.7 An offscreen view, and release 1.9.1 (Emetos)
**Goal:** `offscreen.NewView` runs a widget tree in a view of its own, on the app's UI thread. It draws its frames on a goroutine of its own and hands their pixels and damage to the app. Then 1.9.1 is released.
**Why:** the rest of the same Emetos issue. Its Remote Work view makes a binding by hand today, over a `ui.PlatformHost` of its own, and paces its frames by the time the last picture took. The owner chose (2026-10-27) a view on a binding of its own, not a view in the app's binding. Glutter's binding composites every view in each frame, so a view in the app's binding would get a scene at the main window's rate, and none while that window is hidden. A binding of its own has its own frame rate, and its frames never wait for the main window.
**Read:** SPEC D28, §9.9, §13, §12.10; the 29.6 entry of `docs/IMPLEMENTED.md`; `ui/platform_dispatcher.go` (`PlatformHost`, `SetTextEngine`, the views), `widgets/binding.go` (`NewWidgetsBinding`, `WidgetsBindingOpts`, `AttachRootWidget`, `Post`), `widgets/run_app.go` (how `RunApp` attaches the root and the `View`), `internal/renderhost` (the fake platform's host).
**Port from:** none.
**Files:** `offscreen/view.go` and its tests, SPEC §9.9, §12.10, §1 and §3, `README.md`, `docs/GUIDE.md`.
**Tasks:**
1. `offscreen.NewView(parent *widgets.WidgetsBinding, size ui.Size, root widgets.Widget, onFrame func(Frame), opts ...ViewOpts) (*View, error)`, called on the parent's UI thread:
   - it makes a `ui.PlatformDispatcher` over a host of its own, with one view of `size` logical pixels times `ViewOpts.DevicePixelRatio` (default 1);
   - the dispatcher shares the parent dispatcher's text engine, so fonts loaded by the app work;
   - it makes a `widgets.WidgetsBinding` with the parent's `UIThread`, `Logger`, `ErrorReporter`, `ErrorWidgetBuilder` and `Debug`, and attaches `root` in a `widgets.View`, as `RunWidget` does;
   - `Post` and `ScheduleFrame` go through the parent's `Post`.
2. Frames:
   - `ScheduleFrame` asks for one frame, at most `ViewOpts.MaxFrameRate` per second (default 30; a timer, then a post to the UI thread). Begin-frame times come from a monotonic clock;
   - `RenderView` detaches the scene on the UI thread (29.6) and gives it to the view's raster goroutine. While that goroutine is drawing, a newer scene replaces the one waiting, so frames never queue up;
   - the goroutine draws it into a `Surface` and calls `onFrame(Frame{Image, Damage, Number})` there. `Image` is valid only during the call (the next frame draws into it), so the callback copies what it sends. A frame with no damage calls nothing;
   - a panic in `onFrame` is recovered and reported to the parent's error reporter; the view goes on.
3. `View.Binding()`, so the app can send input through its dispatcher; `View.SetSize(size)`; `View.Close()`, which detaches the root, stops the timer and the goroutine, and waits for it. It is safe to call twice. After `Close` nothing is posted, and nothing is kept: no scene list grows with the frames.
4. The tags of the errors (a nil parent or root, a size that is not positive, a view that is closed) with route tests.
5. Documents and release:
   - SPEC §9.9 (the view), §12.10, §1 (a 1.9.1 sentence) and §3;
   - `README.md` "What is new in 1.9.1";
   - `docs/GUIDE.md`: a short part on drawing a view off screen, in the section on windows or a new one (checked by the docs test).
6. The standard checks. Then merge, tag `step-29.7` and `v1.9.1`, and publish: `git push --follow-tags origin main` (CLAUDE.md, "Tags": a fix step that closes a patch version is published with its tag).
**Tests (table-driven):**
- a view of a counter: a tap through its dispatcher makes a frame whose image shows the new count (compared with `Tester.Screenshot` of the same tree);
- frames are paced: 100 `SetState` calls in a burst make at most a few frames;
- a blinking caret in a focused `TextField` makes small damage rectangles, and the callback gets them;
- a slow `onFrame`: the scenes in between are dropped, and the last one is drawn;
- `Close` while a frame is being drawn waits for it, and nothing runs after;
- 1000 frames: the memory the view holds does not grow (count the scenes kept, no timing);
- a panic in `onFrame` is reported and the next frame is still drawn;
- the error tags;
- `go test -race` with the parent on its UI thread.
**Proof:** a PNG of a frame of the view (the counter) in `docs/implemented-proof-images/29.7-offscreen.png`.
**Manual checks:** none. Emetos removes its host, its rasterizer made of a `Tester`, and `headlessPaceFactor` (its own issue).

### 29.8 A list keeps what the user reads in place when items are inserted before it (Emetos, 1.9.2)
**Goal:** a list with keyed items can keep its visible items where they are on the screen when items are inserted (or moved) before them, and it keeps following the start while the user is at the start. The new option is `MaintainVisibleContentPosition`, the name React Native's `FlatList` gives the same behavior.
**Why:** Emetos found it (issues file, "A reverse list does not keep what the user reads in place when items are added at its start"). The Events panel of its Parasite tab is a live log in a `ListView` with `Reverse: true`, so offset 0 is the newest row. When rows are added at item 0 while the user reads an older row, that row moves away by the height of the new rows. Emetos works around it with `ScrollController.JumpTo` by the added rows times their height, when the offset is not 0 (`keepPlace` in `extensions/parasite/clienttab.go`).

Flutter behaves the same way: `SliverMultiBoxAdaptorElement.performRebuild` sets the layout offset of a child that moved to a new index to null and gives the old offsets to the children now at the old indices (`widgets/sliver.dart`), even with keys and `findChildIndexCallback`. The manager checked on 2026-10-27 that Glutter matches it: in a reverse list of 50-pixel keyed rows, inserting 3 rows at item 0 moves a visible row by 150 pixels, with and without `FindChildIndexCallback`, at offset 0 and at offset 500. The owner chose (2026-10-27) the cheapest design that does what Emetos needs, in 1.9.2:
- the sliver keeps the layout offset of a child that moved, so the visible children stay put and the scroll offset does not change;
- the rows inserted before them are built only when they come into view, and the existing correction at item 0 (`ScrollOffsetCorrection`) fixes the offsets then;
- a correction of the scroll offset in every frame with new rows (anchoring in the viewport, as CSS scroll anchoring does) was rejected: it costs an extra layout pass for each insertion.

**Read:** SPEC §11 (scrolling, slivers), §12.10; `widgets/sliver.go` (`SliverMultiBoxAdaptorElement.PerformRebuild`, `indexToLayoutOffset`, the options structs `SliverListWidgetOpts`, `SliverBuilderOpts`, `SliverListOpts`, `SliverSeparatedOpts`), `widgets/sliver_prototype_extent_list.go`, `widgets/scroll_view.go` (`ListViewOpts`, how `ListView` picks its sliver), `widgets/scroll_delegate.go` (`FindIndexByKey`), `rendering/sliver_list.go` (`PerformLayout`: the leading children without a layout offset are collected and laid out again before the first child, and `ScrollOffsetCorrection` when item 0 does not land at 0), `rendering/sliver_fixed_extent_list.go`.
**Port from:** none (Flutter has no such option; a deviation, recorded in §12.10). Flutter's `widgets/sliver.dart` (`performRebuild`) and `rendering/sliver_list.dart` are the code it changes.
**Files:** `widgets/sliver.go`, `widgets/sliver_prototype_extent_list.go`, `widgets/scroll_view.go`, `rendering/sliver_list.go` and `rendering/sliver_fixed_extent_list.go` if they need it, their tests, the generated `material` re-exports (`go generate ./material`), SPEC §11 and §12.10, `docs/GUIDE.md` (the section on lists).
**Tasks:**
1. First write the test that shows the problem: a reverse `ListView.builder` of keyed rows of known heights with `FindChildIndexCallback`, rows inserted at item 0, at offset 0 and at an offset inside the list; the top-left of a visible row before and after. Give the numbers before and after the fix in the report.
2. Add `MaintainVisibleContentPosition bool` to `ListViewOpts` and to the options of `SliverList`, `SliverFixedExtentList`, `SliverVariedExtentList` and `SliverPrototypeExtentList` (every options struct that their constructors take). `ListView` passes it to the sliver it makes. It works only for children whose element moves to a new index by its key (keys and `FindChildIndexCallback`, or the keys of a list of children); without keys it does nothing, and the doc comment says so.
3. `SliverList` (the sliver whose children carry their layout offsets): when the option is on and the sliver is not at its leading edge (the `ScrollOffset` of its last constraints is above `precisionErrorTolerance`), `PerformRebuild` keeps the layout offset of a child that moved to a new index, and does not give the old offsets to the children that took its old indices. The new children then have no layout offset. `RenderSliverList.PerformLayout` already collects such leading children and lays them out before the first child that has an offset. Check that it does so for a child without an offset in the middle of the list, and fix it if it does not. When the first laid-out child is item 0 and does not land at 0, the existing `ScrollOffsetCorrection` fixes the offset, with no visible jump.
4. At the leading edge (offset 0 of the sliver: the newest row of a reverse log, the top of a feed), nothing changes from Flutter: the new items appear at the start and the rest moves, so the list follows the start.
5. The slivers whose layout offsets come from the index (`SliverFixedExtentList`, `SliverPrototypeExtentList`, `SliverVariedExtentList`): with the option on and not at the leading edge, the next layout returns a `ScrollOffsetCorrection` equal to the layout offset of the first visible child's new index less its offset before the rebuild. That is one correction pass, only in the frame where items were inserted before it.
6. With the option off, nothing changes: every existing test and golden passes as it is.
7. SPEC §11 says what the option does and why it is cheap. A §12.10 row records the deviation (Flutter has no such option; React Native's name). The doc comments of the new fields end with `// Flutter: no counterpart`, as other Glutter-only fields do, and `apicheck` passes.
8. `docs/GUIDE.md`: a short example of a live log (a reverse list that follows the newest row and keeps an older one in place while the user reads it), checked by the docs test.
**Tests (table-driven):**
- reverse and forward lists, at an offset inside the list: rows inserted at item 0, several at a time, in a run of frames; the visible rows keep their top-left to the pixel, and the scroll offset does not change;
- at offset 0: the new rows are shown at the start (the list follows), as without the option;
- after the insertion, scrolling to the start shows every new row in order, contiguous, with no gap or overlap, and the final offset is consistent (`ScrollController.Offset` at the start is 0);
- the fixed-extent, prototype and varied-extent lists: the same, through the correction;
- rows inserted after the visible ones: nothing moves;
- a visible row removed: the rest stays where Flutter puts it, with no panic;
- no keys: the option changes nothing;
- the option off: Flutter's behavior, as the check of 2026-10-27 measured (the row moves by the height of the new rows).
**Proof:** none.
**Manual checks:** none. Emetos removes `keepPlace` (its own issue).

### 29.9 The clipboard of an offscreen view (Emetos, 1.9.2)
**Goal:** an `offscreen.View` can have a clipboard of its own: `ViewOpts.Clipboard` takes a `ui.ClipboardHost`, and every clipboard path of the view's tree (copy, cut, paste, the paste check, the primary selection, rich content) goes to it and never to the desktop.
**Why:** Emetos found it (issues file, "An `offscreen.View` has no clipboard of its own"). The clipboard calls of a view go to its parent binding (`viewHost.SetClipboardText` and the others call `parent.PlatformDispatcher()`), which is the application's, and `WidgetsBinding.Clipboard()` cannot be replaced. A view that another machine types in (Emetos's Remote Work) must not read or replace the clipboard or the primary selection of the host's desktop. Its client must hear of the text its user copies, and give it the text to paste. Emetos works around it with a second binding, made only to be the view's parent, whose platform host is the view's clipboard (`internal/gui/app/headless_clipboard.go`).
**Read:** SPEC §9.9, D28, §12.10; `offscreen/view.go` (`viewHost`, its clipboard methods, `guardErr`), `ui/platform_dispatcher.go` (`ClipboardHost`), `ui/clipboard.go` (`RichClipboardHost`, `ClipboardKind`, the formats), `services` (the clipboard API the widgets use), `widgets/binding.go` (`Clipboard`), `widgets/editable_text.go` (copy, cut, paste, the primary selection, the paste check).
**Port from:** none.
**Files:** `offscreen/view.go` and its tests, SPEC §9.9.
**Tasks:**
1. `ViewOpts.Clipboard ui.ClipboardHost`. When it is nil, the view uses the app's clipboard, as in 1.9.1. When it is set:
   - `SetClipboardText` and `GetClipboardText` go to it;
   - if it also implements `ui.RichClipboardHost`, `SetClipboard` and `GetClipboard` (every kind, the primary selection included) go to it;
   - if it does not, a rich call for the clipboard kind and the text format goes to its text methods, and every other rich call (the primary selection, other formats) does nothing: an empty content, no error. The desktop is never reached.
2. Find every path from a widget of the view to a clipboard (the `services` clipboard, `EditableText`'s copy, cut and paste, the primary selection on a middle click and on a selection change, the paste check of the context menu, `SelectableRegion`, anything else the search finds) and check that each goes through the view's dispatcher. Say in the report what you found. A path that bypasses the dispatcher is fixed in this step.
3. The host's callbacks run on the UI thread: the app calls `done` there (or posts it). The view keeps its guard: after `Close` a late `done` does nothing. Say so in the doc comment.
4. SPEC §9.9: the clipboard of a view.
**Tests (table-driven):**
- a `TextField` in a view with a fake clipboard (text only, and rich): select all and copy, cut, paste through key events sent to the view's dispatcher; the fake gets and gives the text;
- a selection on a platform with a primary selection (Linux, through `foundation.TargetPlatform`): it goes to a rich fake, and to nothing with a text-only fake;
- the paste check (`HasStrings`) reads the fake;
- in every case, the app's clipboard (the parent's fake platform) is neither read nor written;
- nil: the app's clipboard, as in 1.9.1;
- a `done` called after `Close` does nothing.
**Proof:** none.
**Manual checks:** none. Emetos removes `clipboardParent` and its second binding (its own issue).

### 29.10 The platform configuration and the window of an offscreen view (Emetos, 1.9.2)
**Goal:** an `offscreen.View` starts with the app's platform configuration and follows its changes, can be given a `ui.WindowInfo`, and `SetSize` changes only the size.
**Why:** Emetos found it (issues file, "An `offscreen.View` starts with a default platform configuration and a blank window"). The view's platform configuration is the default one, not the app's (the 29.7 entry lists it as a follow-up). Its `ui.ViewConfiguration` has a blank `Window`: no decoration, not activated, not maximized. An app that draws its own title bar and its inactive state reads that window. `ViewOpts` sets neither. `View.SetSize` writes the view's own copy of the configuration, with a blank `Window` again, so a window set with `UpdateViewConfiguration` is lost at the next resize. The tree is mounted inside `NewView`, so a state that reads the configuration in `InitState` sees the default one. Emetos works around it with a root widget whose `InitState` sets both, and a call after each `SetSize` (`internal/gui/app/headless.go`). The owner chose (2026-10-27): the view follows the parent's configuration by default, since it needs no work from the app and mounts the tree once with the right values.
**Read:** SPEC §9.9, D28, §12.10; `offscreen/view.go` (`newView`, `SetSize`, `cfg`), `ui/platform_dispatcher.go` (`PlatformConfiguration`, `UpdatePlatformConfiguration`, `UpdateViewConfiguration`, the `On…Changed` callbacks), `ui/window_info.go`, `widgets/binding.go` (`WidgetsBindingObserver`, `AddObserver`), and how the binding hears of fonts and the caret blink time when they change.
**Port from:** none.
**Files:** `offscreen/view.go` and its tests, `ui` or `widgets` if the hook of task 2 needs it, SPEC §9.9.
**Tasks:**
1. Before the root is attached, the view's dispatcher gets the parent dispatcher's platform configuration (all of it: locales, text scale and its scaler, brightness, accessibility features, fonts, caret blink, the rest of the struct).
2. The view follows the parent's later changes of any field, until `Close`. The parent dispatcher's `OnPlatformConfigurationChanged` is one slot, owned by the parent binding, so do not take it. Use what the parent binding offers (a `WidgetsBindingObserver` on it, which `Close` removes), and check that every field is covered, including those with no observer method. If a field cannot be followed without a new public API, stop and report what you propose.
3. `ViewOpts.PlatformConfiguration foundation.Opt[ui.PlatformConfiguration]`: a fixed configuration instead of the parent's, which is not followed. The app may still change it through the view's dispatcher.
4. `ViewOpts.Window ui.WindowInfo` is the window of the view's configuration from the start; `View.SetWindow(info ui.WindowInfo) error` changes it (tag `ViewClosed` after `Close`) and the framework hears of it as a change of the view's metrics.
5. `View.SetSize` reads the view's current configuration from the dispatcher and changes only `PhysicalSize`, so the window and what the app set with `UpdateViewConfiguration` (padding, insets, display features) are kept.
6. SPEC §9.9: what the view's configuration is and when it changes. The 29.7 follow-up is closed.
**Tests (table-driven):**
- a state that reads `MediaQuery` (text scale, brightness, locale) and the window in `InitState` sees the app's values in the first frame, with no second build;
- a change of each field of the parent's configuration reaches the view (a rebuild, the new value);
- `ViewOpts.PlatformConfiguration`: the view keeps it and ignores the parent's changes;
- `ViewOpts.Window` and `SetWindow`: `WindowDecoration` draws the active and inactive state; `SetWindow` after `Close` returns `ViewClosed` (a route test);
- `SetSize` keeps the window and a padding set with `UpdateViewConfiguration`;
- after `Close`, a change of the parent's configuration does nothing (the observer is gone).
**Proof:** none.
**Manual checks:** none. Emetos removes `viewSetup`, `configure` and `applyWindow` (its own issue).

### 29.11 A default win after the arena is gone (Emetos, 1.9.2)
**Goal:** the gesture arena never panics when its default resolution runs after the arena it was scheduled for has resolved, been emptied, or been replaced by a new arena for the same pointer.
**Why:** Emetos found it (second issues list of 2026-10-27, "Crash (panic) in the gesture arena"): `gestures/arena.go` `resolveByDefault` panics with an index out of range, after the arena's members are gone, in a run with a real configuration. The manager's reading on 2026-10-27:
- `tryToResolveArena` schedules `resolveByDefault(pointer, state)` when one member is left. Flutter does it with `scheduleMicrotask`, which runs before the next event. Glutter's "microtask" is `GestureBinding.Post`, a post to the UI thread, so other pointer events can run first;
- `resolveByDefault` checks only that some arena exists for the pointer (`m.arenas[pointer]`), not that it is the arena it was scheduled for. If the last member rejected (the arena became empty and was removed) and a new down opened a new arena with the same pointer id before the post ran, the old, empty `state.members[0]` is read.
**Read:** SPEC §13 (the UI thread, `Post`), the gestures part of SPEC §11; `gestures/arena.go`, `gestures/binding.go` (`Post`, how pointer events are dispatched, `Close` and `Sweep`), how pointer ids are given to a mouse, touch and pen down on each platform (`gestures/converter.go` or where `PointerData.Pointer` is set); Flutter's `gestures/arena.dart`.
**Port from:** Flutter's `gestures/arena.dart` `_resolveByDefault` (the same check, with the identity of the arena added; a deviation, since Glutter's microtask is a post).
**Files:** `gestures/arena.go` and its tests, SPEC §12.10 if the deviation needs a row.
**Tasks:**
1. First write a test that panics on the current code: the sequence of Emetos's crash, reproduced with the arena manager and a scheduler that holds the posted function until the test runs it (down, close with one member, the member rejects, a new down for the same pointer id, then the post runs). Find any other sequence that reaches the panic (a hold and release, a sweep before the post, a pointer cancel) and add each.
2. `resolveByDefault` does nothing unless the arena of the pointer is still the one it was scheduled for and it still has exactly one member, open or closed as Flutter expects. Look at the other functions that read `state` after a post or a callback (`resolveInFavorOf` and `Sweep` call members, which may add or remove members of the same or another arena) and make them safe in the same way.
3. A member that is told it lost or won must still be told exactly once, as now: no gesture is accepted twice, and none is left without an answer.
4. If the cause is that pointer ids are reused on some platform where Flutter's are unique, say so in the report; do not change the ids in this step.
**Tests (table-driven):**
- every sequence of task 1, without a panic, and with the right member accepted or rejected once;
- the existing arena tests, ported from Flutter's `arena_test.dart`, unchanged;
- a widget test: two taps on a `GestureDetector` with a double-tap recognizer whose second down comes before the post of the first arena runs.
**Proof:** none.
**Manual checks:** none.

### 29.12 Shapes outside the clip are skipped before their mask is made (Emetos, 1.9.2)
**Goal:** the GPU renderers do no work for a shape that is outside the visible area. The geometry hash of a path is not made again in each frame.
**Why:** Emetos found it (second issues list of 2026-10-27, "Drawing cost grows with everything on the page, including what's off screen"). In its preview, 85% of the drawing time is spent hashing paths and masking them, so a frame takes over a second. The manager's reading on 2026-10-27:
- `explicit`'s `pathMask` (`internal/gpu/explicit/primitives.go`) calls `gpu.ShapeOf`, then asks the coverage atlas for tiles (`masks.Tiles`), for every path, arc and stroke the analytic pipeline cannot draw;
- `ShapeOf` hashes the path with `ui.Path.Fingerprint`, an FNV over each byte of each point (`ui/path_flatten.go`), in every frame;
- nothing checks first whether the shape's bounds, under its transform and widened by its stroke, meet the clip of the command (`s.outer`, the clip's bounds). The command is skipped only when its clip is empty;
- `gles/paths.go` has the same order, and `gpu.ClipKey` hashes the paths of clips the same way (`internal/gpu/clipmask.go`).
**Read:** SPEC §9.1 (GLES, the coverage atlas), §9.2, §9.3 (the explicit renderer), §9.9 (a detached scene is drawn on another goroutine: nothing the renderer reads may be a lazy cache that two goroutines write); `internal/gpu/mask.go` (`ShapeOf`, `MaskMargin`), `internal/gpu/clipmask.go`, `internal/gpu/explicit/primitives.go`, `frame.go` (`shape`), `scene.go`, `internal/gpu/gles/paths.go`, `internal/gpu/soft` (how `soft` bounds its work, for comparison), `ui/path.go` and `ui/path_flatten.go` (how a path is built and changed, `Fingerprint`), `ui/display_list.go` (what `OpDrawPath` keeps of a path).
**Port from:** none (Impeller culls by coverage, `Entity::GetCoverage`, but Glutter's renderers are its own).
**Files:** `internal/gpu` (`mask.go`, `clipmask.go`, a cull helper), `internal/gpu/explicit`, `internal/gpu/gles`, `ui` (the path's fingerprint), their tests and benchmarks, SPEC §9.1 and §9.3.
**Tasks:**
1. First a benchmark that shows the cost: a scene with many paths (icons, SVG-like shapes, strokes) in a tall scrolled list where most of them are outside the viewport, drawn through `explicit` on the fake device (`internal/gpu/explicit/fakedevice`) and through `gles` where a benchmark can run without a display. Report the time and the share of `Fingerprint` and `masks.Tiles` before and after (a CPU profile).
2. Cull early: before anything else of a shape command (the paint, `ShapeOf`, the atlas), compute the shape's device bounds (its bounds under the transform, widened by the stroke width, the joins and the anti-aliasing margin, as `MaskMargin` does) and skip the command when they do not meet the clip bounds of the command. The test is conservative: a shape that touches the visible area is drawn as now. Do the same for the analytic shapes, images, text runs and any other command whose bounds are cheap, if they are not culled already; say which were.
3. The fingerprint of a path is made once, not in every frame:
   - it is computed eagerly when the path changes (or when it is recorded into a display list), never lazily on first read, so two goroutines drawing the same detached scene never write it (SPEC §9.9);
   - a path that is not changed again keeps its fingerprint across frames, so a scene that redraws the same pictures hashes nothing;
   - pick the cheapest design that keeps that rule (for example, a hash updated as verbs and points are added, reset by the operations that rewrite the path), and say why in the report;
   - hash 64-bit words, not one byte at a time, if the atlas and clip-mask keys keep their quality.
4. Nothing else changes: every golden, parity and display test passes with its current tolerances; the atlas still finds a mask it made in an earlier frame.
5. SPEC §9.3 (and §9.1 for `gles`) says how commands are culled and how path keys are made.
**Tests (table-driven):**
- a shape fully outside the clip makes no atlas request and no instance (count them on the fake device), for each kind of shape command, under a translation, a scale and a rotation;
- a shape that only touches the clip with its stroke or its anti-aliased edge is still drawn;
- a path's fingerprint is the same as before for the same path, changes when the path changes, and is not written while the path is drawn (`go test -race` with two goroutines drawing one detached scene, as in 29.6);
- the benchmark of task 1, with the numbers before and after in the report; the goal is that the cost of a frame grows with what is visible, not with what is in the scene.
**Proof:** none.
**Manual checks:** none.

### 29.13 A keyboard without a keymap is not an error (Emetos, 1.9.2)
**Goal:** when a Wayland compositor sends `wl_keyboard.keymap` with the format `no_keymap` (when the last keyboard is unplugged), Glutter drops the layout without logging an error, and keys work again when a keyboard with a keymap returns.
**Why:** Emetos found it (second issues list of 2026-10-27, "Spurious error log"). The protocol allows `no_keymap` (format 0), but `keyboard.onKeymap` (`internal/platform/linux/wayland/keyboard.go`) logs every format other than `xkb_v1` with `slog.Error`.
**Read:** SPEC §8.2 and §8.5 (Wayland, input); `internal/platform/linux/wayland/keyboard.go`, the `wl_keyboard.keymap` event in the Wayland protocol (`wl/wayland_gen.go`, `WlKeyboardKeymapFormatNoKeymap`).
**Port from:** none.
**Files:** `internal/platform/linux/wayland/keyboard.go` and its tests.
**Tasks:**
1. `no_keymap`: drop the layout, stop the key repeat, release the keys that are held as the compositor's leave would, and log at `Debug` at most. A format that is neither `no_keymap` nor `xkb_v1` (one the protocol may add later) stays a `Warn`, not an `Error`, since the app goes on.
2. The keys sent while there is no keymap are ignored without a log for each key.
3. A later `xkb_v1` keymap makes the keyboard work again, as now.
**Tests (table-driven):** with the keyboard's handlers called directly, as the existing keyboard tests do: `no_keymap` logs nothing above `Debug` (a captured `slog` handler), drops the layout and the repeat; a key without a keymap is ignored; a new `xkb_v1` keymap restores the keys; an unknown format logs a `Warn`.
**Proof:** none.
**Manual checks:** none.

### 29.14 What an idle frame allocates (Emetos, 1.9.2)
**Goal:** a frame in which only a caret blinks allocates little, and nothing in proportion to the parts of the scene that did not change. Glutter's share of Emetos's idle allocations is found and cut.
**Why:** Emetos found it (second issues list of 2026-10-27, "while idle with only the cursor blinking, about 2 frames per second, the Go heap grows by about 11 MB/s until garbage collection runs"). That is about 5 MB for each idle frame. Emetos cannot yet tell whether it comes from Emetos or from Glutter; this step measures Glutter's part, with Glutter's own `examples/editor` and a large text, and fixes what is Glutter's.
**Read:** SPEC §9.1 (damage, retained layers), §9.2, §9.9, §13, §15 (performance targets); the 29.6 entry of `docs/IMPLEMENTED.md` (how `soft` was profiled); `widgets/editable_text_state.go` (the caret blink), `rendering` (repaint boundaries, the layer tree, `compositeFrame`), `ui/scene.go`, `internal/gpu/explicit/frame.go` and `scene.go`, `internal/gpu/damage.go`, `internal/engine` (the frame loop).
**Port from:** none.
**Files:** wherever the allocations are; benchmarks in the packages that own them; SPEC §15.
**Tasks:**
1. Measure first. Benchmarks with `b.ReportAllocs` (and a heap profile, `-memprofile`) of one caret-blink frame of a focused `TextField` in a large page (`examples/editor` with a long text, and a `Scaffold` with many widgets), split by stage:
   - the framework's frame (build, layout, paint, `compositeFrame`), through `gluttertest` or the fake platform;
   - the renderers: `explicit` on its fake device, `gles` where a benchmark can run, `soft`;
   - the damage tracker and, for `offscreen.View`, the detacher and the surface.
   Report the bytes and allocations of each stage, and the top allocation sites.
2. Cut the allocations that grow with the scene and not with the change: a repaint of the caret should not record, copy, hash into new memory or encode again the pictures that did not change. Reuse the frame's buffers across frames (instance arrays, command lists, maps, slices) where the code makes them new for each frame.
3. Target: a caret frame of `examples/editor` with a 5,000-line text allocates under 256 KB in each of the framework and the renderer, and the bytes do not grow with the length of the text. If a target is not met, report where the rest goes and why it cannot be cut in this step.
4. Nothing visible changes: every golden, parity and display test passes as it is.
5. SPEC §15: the new numbers. Say in the report what of Emetos's 5 MB per frame Glutter's own example reproduces, so that Emetos knows what is its own.
**Tests (table-driven):**
- the benchmarks of task 1, with the numbers before and after in the report and `IMPLEMENTED.md`;
- an allocation test (`testing.AllocsPerRun`, or a bytes count from `runtime.ReadMemStats` around the frame) for the caret frame of the framework and of `explicit` on the fake device, with a bound well above the measured value so that it is not flaky, and below the old one;
- `go test -race` of the buffers that are now reused across frames.
**Proof:** none.
**Manual checks:** none.

### 29.15 No `testing` in programs, and release 1.9.2 (Emetos)
**Goal:** no Glutter program links the `testing` package, and a test keeps it that way. Then 1.9.2 is released.
**Why:** Emetos found it (issues file, "`testing` is linked into every Glutter program"). `internal/platform/linux/dl/layout.go` is not a test file, and it imports `testing` for `AssertLayout`, so every program that links the Linux platform (Linux desktop and Android) links `testing`. The manager checked on 2026-10-27 with `go list -deps ./examples/counter`: it is the only cause on linux/amd64 and android/arm64; js/wasm and windows/amd64 do not link `testing`. It costs size, nothing else (Go registers its flags only in a test binary).
**Read:** SPEC §8.1 (rule 3, the layout tests), §25.2 (the Windows `layouttest` package, the model for this one); `internal/platform/linux/dl/layout.go` and every test that calls `dl.AssertLayout` (`xkb`, `fontconfig`, `x11`, `wayland`, `wayland/decor`, `wayland/wl`, `internal/gpu/nativewindow`, `dl` itself); `examples/examples_test.go`.
**Port from:** none.
**Files:** a new package `internal/platform/linux/dl/layouttest` (or the place SPEC §8.1 names, if it names one), the tests that call it, a guard test, SPEC §8.1, §1 and the scope paragraphs, `README.md`.
**Tasks:**
1. Move `AssertLayout` out of package `dl` into a package that only tests import, `internal/platform/linux/dl/layouttest`, as Windows has `win32/layouttest`. The tests call it from there. `dl` no longer imports `testing`.
2. A guard test (in `examples`, next to the build tests): `go list -deps` of every example, for linux/amd64, android/arm64 (`CGO_ENABLED=1` is not needed for the list), js/wasm, windows/amd64 and windows/arm64, and none of them lists `testing` (or `testing/...`) or `gluttertest`. When it fails, it names the package that imports it (`go list -f '{{.ImportPath}} {{.Imports}}'`).
3. Search for any other non-test file that imports `testing`, `net/http/httptest` or `gluttertest` outside the test helper packages, and say what you found.
4. Release 1.9.2:
   - SPEC: the version paragraph of §1 (a 1.9.2 sentence: lists that keep what the user reads in place, an offscreen view with its own clipboard and the app's configuration, a gesture arena that no longer panics, shapes outside the clip culled before their masks, fewer allocations in idle frames, no `testing` in programs), a "Scope of 1.9.2" paragraph next to the one of 1.9.1;
   - `README.md` "What is new in 1.9.2";
   - the standard checks of rule 8, and `go run ./cmd/glutter-gen apicheck -allow cmd/glutter-gen/apicheck.allow`;
   - merge, tag `step-29.15` and `v1.9.2`, and publish: `git push --follow-tags origin main`, which takes steps 29.8 to 29.15 and their tags to GitHub. Check with `git ls-remote --tags origin`.
**Tests (table-driven):**
- the guard test of task 2, which fails on the code before the move (say so in the report);
- every layout test passes from its new place.
**Proof:** none.
**Manual checks:** none.
