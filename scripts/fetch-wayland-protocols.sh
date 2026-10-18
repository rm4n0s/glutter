#!/usr/bin/env bash
# Downloads the Wayland protocol XML files that Glutter generates bindings for
# (docs/SPEC.md §8.1 and §8.2) into
# internal/platform/linux/wayland/protocols/, and records the release tags and
# commits in the VERSIONS file there.
#
# Usage:
#   scripts/fetch-wayland-protocols.sh           use the tags recorded in VERSIONS
#                                                (the latest releases on the first run)
#   scripts/fetch-wayland-protocols.sh --latest  move to the latest releases
#
# Environment: WAYLAND_TAG and WAYLAND_PROTOCOLS_TAG pin a tag explicitly.
#
# After a change, run `go generate ./internal/platform/linux/wayland/...` and
# review the diff of the generated files.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/internal/platform/linux/wayland/protocols"
VERSIONS="$OUT/VERSIONS"

WAYLAND_URL="https://gitlab.freedesktop.org/wayland/wayland.git"
PROTOCOLS_URL="https://gitlab.freedesktop.org/wayland/wayland-protocols.git"

# The XML files to copy out of wayland-protocols: SPEC §8.2.
PROTOCOL_FILES=(
  xdg-shell.xml
  xdg-decoration-unstable-v1.xml
  fractional-scale-v1.xml
  viewporter.xml
  cursor-shape-v1.xml
  tablet-v2.xml
  text-input-unstable-v3.xml
  primary-selection-unstable-v1.xml
  pointer-gestures-unstable-v1.xml
  xdg-activation-v1.xml
  presentation-time.xml
  xdg-toplevel-icon-v1.xml
  xdg-foreign-unstable-v2.xml
  xdg-dialog-v1.xml
  xdg-output-unstable-v1.xml
  idle-inhibit-unstable-v1.xml
  content-type-v1.xml
)

log() { printf '%s\n' "$*" >&2; }

latest=0
if [[ "${1:-}" == "--latest" ]]; then
  latest=1
elif [[ $# -gt 0 ]]; then
  log "usage: $0 [--latest]"
  exit 2
fi

# latest_tag URL REGEX prints the highest release tag that matches REGEX.
latest_tag() {
  git ls-remote --tags --refs "$1" |
    sed 's#.*refs/tags/##' |
    grep -E "$2" | sort -V | tail -n 1
}

# recorded KEY prints a value from VERSIONS, or nothing.
recorded() {
  [[ -f "$VERSIONS" ]] && sed -n "s/^$1=//p" "$VERSIONS" | head -n 1 || true
}

wayland_tag="${WAYLAND_TAG:-}"
protocols_tag="${WAYLAND_PROTOCOLS_TAG:-}"
if [[ "$latest" -eq 0 ]]; then
  [[ -z "$wayland_tag" ]] && wayland_tag="$(recorded wayland_tag)"
  [[ -z "$protocols_tag" ]] && protocols_tag="$(recorded wayland_protocols_tag)"
fi
# Release candidates of wayland are x.y.9z; skip them.
[[ -z "$wayland_tag" ]] && wayland_tag="$(latest_tag "$WAYLAND_URL" '^[0-9]+\.[0-9]+\.([0-9]|[1-8][0-9])$')"
[[ -z "$protocols_tag" ]] && protocols_tag="$(latest_tag "$PROTOCOLS_URL" '^[0-9]+\.[0-9]+$')"
if [[ -z "$wayland_tag" || -z "$protocols_tag" ]]; then
  log "error: could not find the release tags (network problem?)"
  exit 1
fi
log "wayland $wayland_tag, wayland-protocols $protocols_tag"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

clone() { # URL TAG DIR
  git clone --quiet --depth 1 --filter=blob:none --sparse --branch "$2" "$1" "$3" 2>/dev/null
}

clone "$WAYLAND_URL" "$wayland_tag" "$work/wayland"
git -C "$work/wayland" sparse-checkout set protocol
wayland_commit="$(git -C "$work/wayland" rev-parse HEAD)"

clone "$PROTOCOLS_URL" "$protocols_tag" "$work/protocols"
git -C "$work/protocols" sparse-checkout set stable staging unstable
protocols_commit="$(git -C "$work/protocols" rev-parse HEAD)"

mkdir -p "$OUT"
rm -f "$OUT"/*.xml

cp "$work/wayland/protocol/wayland.xml" "$OUT/wayland.xml"
cp "$work/wayland/COPYING" "$OUT/COPYING.wayland"
cp "$work/protocols/COPYING" "$OUT/COPYING.wayland-protocols"

for name in "${PROTOCOL_FILES[@]}"; do
  src="$(find "$work/protocols/stable" "$work/protocols/staging" "$work/protocols/unstable" -name "$name" | head -n 1)"
  if [[ -z "$src" ]]; then
    log "error: $name is not in wayland-protocols $protocols_tag"
    log "       find it with: git ls-tree -r --name-only $protocols_tag | grep ${name%%.*}"
    exit 1
  fi
  cp "$src" "$OUT/$name"
done

{
  echo "# Written by scripts/fetch-wayland-protocols.sh. Do not edit."
  echo "wayland_repository=$WAYLAND_URL"
  echo "wayland_tag=$wayland_tag"
  echo "wayland_commit=$wayland_commit"
  echo "wayland_protocols_repository=$PROTOCOLS_URL"
  echo "wayland_protocols_tag=$protocols_tag"
  echo "wayland_protocols_commit=$protocols_commit"
  echo "# file sha256"
  (cd "$OUT" && sha256sum wayland.xml "${PROTOCOL_FILES[@]}" | sed 's/^/file=/')
} >"$VERSIONS"

log "Wrote $(ls "$OUT"/*.xml | wc -l) protocol files to ${OUT#"$ROOT"/}"
