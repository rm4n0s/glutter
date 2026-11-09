#!/usr/bin/env bash
# Fetches the pinned Flutter sources that Glutter ports from (docs/SPEC.md §20).
#
# Usage: scripts/fetch-reference.sh   (from anywhere; works relative to the repo root)
#
# First run: clones the `stable` branch of flutter/flutter (or, when GLUTTER_FLUTTER_COMMIT is set to a full commit hash,
# that commit: the pinned Flutter of SPEC §25 is 3.47.7, abaf9c523780a608bd46686fd5e53740a07077f8, and a machine that
# starts from nothing sets it so that it ports from the same sources as the others), the
# material-color-utilities repository, Skia (at Flutter's pinned revision), flutter_svg (at the commit pinned
# below) and SDL 3 (at the release tag pinned below) into reference/, then records the exact commits in reference/VERSION.
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
  engine/src/flutter/shell/platform/android
  engine/src/flutter/shell/platform/windows
  engine/src/flutter/shell/platform/common
  engine/src/flutter/shell/platform/darwin
  engine/src/flutter/impeller/renderer/backend/vulkan
)
# Directories that Windows steps port from if the pinned commit has them (SPEC §25): the C++
# accessibility library of the engine (ui/accessibility, in third_party). Not an error when absent.
FLUTTER_OPTIONAL_PATHS=(
  engine/src/flutter/third_party/accessibility
)

# Skia, at the revision that Flutter's DEPS pins (skia_revision) at the pinned Flutter commit: the
# Android font manager (SPEC §24.9) is ported from it.
SKIA_URL="https://skia.googlesource.com/skia"
SKIA_DIR="$REF/skia"
SKIA_PATHS=(src/ports include/ports)

# flutter_svg (MIT), for the API of package svg (SPEC §26.1, step 29.4): names, defaults and sizing only; the parser
# and the painter of internal/svg follow the SVG specifications. The repository stopped tagging at v2.0.7 (2023); its
# master (packages/flutter_svg, version 2.0.10+1) is newer, so the pin is that commit.
FLUTTER_SVG_URL="https://github.com/dnfield/flutter_svg"
FLUTTER_SVG_DIR="$REF/flutter_svg"
FLUTTER_SVG_COMMIT="6ed5a9d2831ac88869a6490aac701366acad1b59"
FLUTTER_SVG_PATHS=(packages/flutter_svg/lib packages/flutter_svg/test)

# SDL 3 (zlib), the reference of the packages gpu, gamepad, audio and sensors (SPEC §27, §28, D32): the shape of the GPU
# descriptors, the gamepad mappings, the audio backends and the sensors. Nothing is imported or vendored from it.
SDL_URL="https://github.com/libsdl-org/SDL"
SDL_DIR="$REF/sdl"
SDL_TAG="release-3.4.20"
SDL_PATHS=(include src/gpu src/joystick src/audio src/sensor test)

# meshoptimizer (MIT), the reference of the decoder of EXT_meshopt_compression in model/gltf (SPEC §28.5, step 31.8): the vertex and
# index codecs and the filters, ported to Go in internal/meshopt, with the test vectors of its own tests.
MESHOPT_URL="https://github.com/zeux/meshoptimizer"
MESHOPT_DIR="$REF/meshoptimizer"
MESHOPT_TAG="v1.3"
MESHOPT_PATHS=(src demo)

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
  # ${arr[@]+"${arr[@]}"}: the empty array is "unbound" under set -u in bash 3.2, which is the bash of macOS.
  git clone --quiet --depth 1 --filter=blob:none --sparse ${branch_args[@]+"${branch_args[@]}"} "$url" "$dir"
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

# write_sdl_version appends the SDL 3 lines to reference/VERSION.
write_sdl_version() {
  {
    echo "sdl_repository=$SDL_URL"
    echo "sdl_tag=$SDL_TAG"
    echo "sdl_commit=$(git -C "$SDL_DIR" rev-parse HEAD)"
    echo "sdl_date=$(commit_date "$SDL_DIR")"
  } >>"$VERSION_FILE"
}

# write_svg_version appends the flutter_svg lines to reference/VERSION.
write_svg_version() {
  {
    echo "flutter_svg_repository=$FLUTTER_SVG_URL"
    echo "flutter_svg_commit=$(git -C "$FLUTTER_SVG_DIR" rev-parse HEAD)"
    echo "flutter_svg_date=$(commit_date "$FLUTTER_SVG_DIR")"
    echo "flutter_svg_version=$(sed -n 's/^version: *//p' "$FLUTTER_SVG_DIR/packages/flutter_svg/pubspec.yaml" | head -n 1)"
  } >>"$VERSION_FILE"
}

# ensure_sdl COMMIT clones SDL 3 at SDL_TAG (a shallow clone of the tag) when it is missing, then pins to COMMIT when
# one is recorded.
ensure_sdl() {
  local commit="$1"
  if [[ ! -d "$SDL_DIR/.git" ]]; then
    log "Cloning $SDL_URL ($SDL_TAG) into $SDL_DIR"
    clone_sparse "$SDL_URL" "$SDL_DIR" "$SDL_TAG" "${SDL_PATHS[@]}"
  else
    git -C "$SDL_DIR" sparse-checkout set "${SDL_PATHS[@]}"
  fi
  if [[ -n "$commit" ]]; then
    pin_commit "$SDL_DIR" "$commit"
  fi
}

# write_meshopt_version appends the meshoptimizer lines to reference/VERSION.
write_meshopt_version() {
  {
    echo "meshoptimizer_repository=$MESHOPT_URL"
    echo "meshoptimizer_tag=$MESHOPT_TAG"
    echo "meshoptimizer_commit=$(git -C "$MESHOPT_DIR" rev-parse HEAD)"
    echo "meshoptimizer_date=$(commit_date "$MESHOPT_DIR")"
  } >>"$VERSION_FILE"
}

# ensure_meshopt COMMIT clones meshoptimizer at MESHOPT_TAG when it is missing, then pins to COMMIT when one is recorded.
ensure_meshopt() {
  local commit="$1"
  if [[ ! -d "$MESHOPT_DIR/.git" ]]; then
    log "Cloning $MESHOPT_URL ($MESHOPT_TAG) into $MESHOPT_DIR"
    clone_sparse "$MESHOPT_URL" "$MESHOPT_DIR" "$MESHOPT_TAG" "${MESHOPT_PATHS[@]}"
  else
    git -C "$MESHOPT_DIR" sparse-checkout set "${MESHOPT_PATHS[@]}"
  fi
  if [[ -n "$commit" ]]; then
    pin_commit "$MESHOPT_DIR" "$commit"
  fi
}

mkdir -p "$REF"

if [[ -f "$VERSION_FILE" ]]; then
  log "Using the commits recorded in $VERSION_FILE"
  flutter_commit="$(read_version flutter_commit)"
  mcu_commit="$(read_version material_color_utilities_commit)"
  skia_commit="$(read_version skia_commit)"
  if [[ -z "$flutter_commit" || -z "$mcu_commit" ]]; then
    log "error: $VERSION_FILE is incomplete; delete reference/ and run again"
    exit 1
  fi
  ensure_repo "$FLUTTER_URL" "$FLUTTER_DIR" "" "$flutter_commit" "${FLUTTER_PATHS[@]}" "${FLUTTER_OPTIONAL_PATHS[@]}"
  ensure_repo "$MCU_URL" "$MCU_DIR" "" "$mcu_commit" "${MCU_PATHS[@]}"
  if [[ -z "$skia_commit" ]]; then
    # An older reference/VERSION: the revision is in Flutter's DEPS.
    skia_commit="$(sed -n "s/^[[:space:]]*'skia_revision': '\([0-9a-f]*\)'.*/\1/p" "$FLUTTER_DIR/DEPS" | head -n 1)"
    echo "skia_repository=$SKIA_URL" >>"$VERSION_FILE"
    echo "skia_commit=$skia_commit" >>"$VERSION_FILE"
  fi
  ensure_repo "$SKIA_URL" "$SKIA_DIR" "" "$skia_commit" "${SKIA_PATHS[@]}"
  if ! grep -q '^skia_date=' "$VERSION_FILE"; then
    echo "skia_date=$(commit_date "$SKIA_DIR")" >>"$VERSION_FILE"
  fi
  svg_commit="$(read_version flutter_svg_commit)"
  ensure_repo "$FLUTTER_SVG_URL" "$FLUTTER_SVG_DIR" "" "${svg_commit:-$FLUTTER_SVG_COMMIT}" "${FLUTTER_SVG_PATHS[@]}"
  if [[ -z "$svg_commit" ]]; then
    # An older reference/VERSION: record the pinned flutter_svg.
    write_svg_version
  fi
  sdl_commit="$(read_version sdl_commit)"
  ensure_sdl "$sdl_commit"
  if [[ -z "$sdl_commit" ]]; then
    # An older reference/VERSION: record the pinned SDL 3.
    write_sdl_version
  fi
  meshopt_commit="$(read_version meshoptimizer_commit)"
  ensure_meshopt "$meshopt_commit"
  if [[ -z "$meshopt_commit" ]]; then
    # An older reference/VERSION: record the pinned meshoptimizer.
    write_meshopt_version
  fi
else
  ensure_repo "$FLUTTER_URL" "$FLUTTER_DIR" "$FLUTTER_BRANCH" "${GLUTTER_FLUTTER_COMMIT:-}" "${FLUTTER_PATHS[@]}" "${FLUTTER_OPTIONAL_PATHS[@]}"
  ensure_repo "$MCU_URL" "$MCU_DIR" "" "" "${MCU_PATHS[@]}"
  skia_commit="$(sed -n "s/^[[:space:]]*'skia_revision': '\([0-9a-f]*\)'.*/\1/p" "$FLUTTER_DIR/DEPS" | head -n 1)"
  ensure_repo "$SKIA_URL" "$SKIA_DIR" "" "$skia_commit" "${SKIA_PATHS[@]}"
  ensure_repo "$FLUTTER_SVG_URL" "$FLUTTER_SVG_DIR" "" "$FLUTTER_SVG_COMMIT" "${FLUTTER_SVG_PATHS[@]}"
  {
    echo "flutter_repository=$FLUTTER_URL"
    echo "flutter_branch=$FLUTTER_BRANCH"
    echo "flutter_commit=$(git -C "$FLUTTER_DIR" rev-parse HEAD)"
    echo "flutter_date=$(commit_date "$FLUTTER_DIR")"
    echo "flutter_version=$(flutter_version)"
    echo "material_color_utilities_repository=$MCU_URL"
    echo "material_color_utilities_commit=$(git -C "$MCU_DIR" rev-parse HEAD)"
    echo "material_color_utilities_date=$(commit_date "$MCU_DIR")"
    echo "skia_repository=$SKIA_URL"
    echo "skia_commit=$(git -C "$SKIA_DIR" rev-parse HEAD)"
    echo "skia_date=$(commit_date "$SKIA_DIR")"
  } >"$VERSION_FILE"
  write_svg_version
  ensure_sdl ""
  write_sdl_version
  ensure_meshopt ""
  write_meshopt_version
fi

# Every sparse path must exist; Flutter reorganizes its repository sometimes.
missing=0
for p in "${FLUTTER_PATHS[@]}" LICENSE; do
  if [[ ! -e "$FLUTTER_DIR/$p" ]]; then
    log "error: missing in reference/flutter: $p"
    missing=1
  fi
done
for p in "${FLUTTER_OPTIONAL_PATHS[@]}"; do
  if [[ ! -e "$FLUTTER_DIR/$p" ]]; then
    log "note: not in reference/flutter at the pinned commit (optional): $p"
  fi
done
for p in "${SKIA_PATHS[@]}" LICENSE; do
  if [[ ! -e "$SKIA_DIR/$p" ]]; then
    log "error: missing in reference/skia: $p"
    missing=1
  fi
done
for p in "${FLUTTER_SVG_PATHS[@]}" LICENSE; do
  if [[ ! -e "$FLUTTER_SVG_DIR/$p" ]]; then
    log "error: missing in reference/flutter_svg: $p"
    missing=1
  fi
done
for p in "${SDL_PATHS[@]}" LICENSE.txt; do
  if [[ ! -e "$SDL_DIR/$p" ]]; then
    log "error: missing in reference/sdl: $p"
    missing=1
  fi
done
for p in "${MESHOPT_PATHS[@]}" LICENSE.md; do
  if [[ ! -e "$MESHOPT_DIR/$p" ]]; then
    log "error: missing in reference/meshoptimizer: $p"
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
