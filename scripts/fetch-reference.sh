#!/usr/bin/env bash
# Fetches the pinned Flutter sources that Glutter ports from (docs/SPEC.md §20).
#
# Usage: scripts/fetch-reference.sh   (from anywhere; works relative to the repo root)
#
# First run: clones the `stable` branch of flutter/flutter and the
# material-color-utilities repository into reference/, then records the exact
# commits in reference/VERSION.
# Later runs: reference/VERSION exists, so exactly the recorded commits are
# checked out. To move to a newer Flutter, delete reference/ and run again.
#
# reference/ is git-ignored. Never import, vendor or commit it.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REF="$ROOT/reference"
VERSION_FILE="$REF/VERSION"

FLUTTER_URL="https://github.com/flutter/flutter"
FLUTTER_BRANCH="stable"
FLUTTER_DIR="$REF/flutter"
# Directories (cone mode). Root files such as LICENSE are always checked out.
FLUTTER_PATHS=(
  packages/flutter/lib
  packages/flutter/test
  engine/src/flutter/lib/ui
  engine/src/flutter/lib/web_ui
  dev/tools/gen_keycodes/data
)

MCU_URL="https://github.com/material-foundation/material-color-utilities"
MCU_DIR="$REF/material-color-utilities"
MCU_PATHS=(dart)

log() { printf '%s\n' "$*" >&2; }

# read_version KEY prints the value recorded in reference/VERSION.
read_version() {
  sed -n "s/^$1=//p" "$VERSION_FILE" | head -n 1
}

# clone_sparse URL DIR BRANCH PATH... clones a shallow, blobless, sparse checkout.
# BRANCH may be empty for the remote's default branch.
clone_sparse() {
  local url="$1" dir="$2" branch="$3"
  shift 3
  local branch_args=()
  [[ -n "$branch" ]] && branch_args=(--branch "$branch")
  git clone --quiet --depth 1 --filter=blob:none --sparse "${branch_args[@]}" "$url" "$dir"
  git -C "$dir" sparse-checkout set "$@"
}

# pin_commit DIR COMMIT makes the sparse clone in DIR match COMMIT exactly.
pin_commit() {
  local dir="$1" commit="$2"
  if [[ "$(git -C "$dir" rev-parse HEAD)" != "$commit" ]]; then
    git -C "$dir" fetch --quiet --depth 1 --filter=blob:none origin "$commit"
    git -C "$dir" checkout --quiet --detach "$commit"
  fi
}

# ensure_repo URL DIR BRANCH COMMIT PATH... clones if DIR is missing, then pins
# to COMMIT when it is non-empty.
ensure_repo() {
  local url="$1" dir="$2" branch="$3" commit="$4"
  shift 4
  if [[ ! -d "$dir/.git" ]]; then
    log "Cloning $url into $dir"
    clone_sparse "$url" "$dir" "$branch" "$@"
  else
    git -C "$dir" sparse-checkout set "$@"
  fi
  if [[ -n "$commit" ]]; then
    pin_commit "$dir" "$commit"
  fi
}

# commit_date DIR prints the committer date of HEAD in ISO 8601.
commit_date() {
  git -C "$1" log -1 --format=%cI HEAD
}

# flutter_version prints the release tag that points at the checked-out commit,
# or the contents of the `version` file when the checkout has one.
flutter_version() {
  if [[ -f "$FLUTTER_DIR/version" ]]; then
    tr -d '\n' <"$FLUTTER_DIR/version"
    return
  fi
  local head tag
  head="$(git -C "$FLUTTER_DIR" rev-parse HEAD)"
  # Annotated tags list their target commit on the peeled "^{}" line.
  tag="$(git ls-remote --tags "$FLUTTER_URL" 2>/dev/null |
    awk -v h="$head" '$1 == h { sub("refs/tags/", "", $2); sub(/\^\{\}$/, "", $2); print $2 }' |
    grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -n 1 || true)"
  printf '%s' "${tag:-unknown}"
}

mkdir -p "$REF"

if [[ -f "$VERSION_FILE" ]]; then
  log "Using the commits recorded in $VERSION_FILE"
  flutter_commit="$(read_version flutter_commit)"
  mcu_commit="$(read_version material_color_utilities_commit)"
  if [[ -z "$flutter_commit" || -z "$mcu_commit" ]]; then
    log "error: $VERSION_FILE is incomplete; delete reference/ and run again"
    exit 1
  fi
  ensure_repo "$FLUTTER_URL" "$FLUTTER_DIR" "" "$flutter_commit" "${FLUTTER_PATHS[@]}"
  ensure_repo "$MCU_URL" "$MCU_DIR" "" "$mcu_commit" "${MCU_PATHS[@]}"
else
  ensure_repo "$FLUTTER_URL" "$FLUTTER_DIR" "$FLUTTER_BRANCH" "" "${FLUTTER_PATHS[@]}"
  ensure_repo "$MCU_URL" "$MCU_DIR" "" "" "${MCU_PATHS[@]}"
  {
    echo "flutter_repository=$FLUTTER_URL"
    echo "flutter_branch=$FLUTTER_BRANCH"
    echo "flutter_commit=$(git -C "$FLUTTER_DIR" rev-parse HEAD)"
    echo "flutter_date=$(commit_date "$FLUTTER_DIR")"
    echo "flutter_version=$(flutter_version)"
    echo "material_color_utilities_repository=$MCU_URL"
    echo "material_color_utilities_commit=$(git -C "$MCU_DIR" rev-parse HEAD)"
    echo "material_color_utilities_date=$(commit_date "$MCU_DIR")"
  } >"$VERSION_FILE"
fi

# Every sparse path must exist; Flutter reorganizes its repository sometimes.
missing=0
for p in "${FLUTTER_PATHS[@]}" LICENSE; do
  if [[ ! -e "$FLUTTER_DIR/$p" ]]; then
    log "error: missing in reference/flutter: $p"
    missing=1
  fi
done
for p in "${MCU_PATHS[@]}" LICENSE; do
  if [[ ! -e "$MCU_DIR/$p" ]]; then
    log "error: missing in reference/material-color-utilities: $p"
    missing=1
  fi
done
if [[ "$missing" -ne 0 ]]; then
  log "Find the new location with: git -C reference/flutter ls-tree -r --name-only HEAD | grep <name>"
  exit 1
fi

log "Reference sources ready:"
sed 's/^/  /' "$VERSION_FILE" >&2
