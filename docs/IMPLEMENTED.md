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
