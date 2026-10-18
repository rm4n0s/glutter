#!/usr/bin/env bash
# Fetches the fonts that Glutter bundles (docs/SPEC.md §10) into material/fonts/.
#
# Usage: scripts/fetch-fonts.sh   (from anywhere; works relative to the repo root)
#
# Downloads from Google's official GitHub repositories, at the pins recorded in
# material/fonts/VERSIONS, and checks every file against the SHA-256 recorded there.
# To move to a newer font, change the pins in this script, delete the font files and
# VERSIONS, run it again, and review the new VERSIONS.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/material/fonts"

ROBOTO_REPO="googlefonts/roboto-3-classic"
ROBOTO_TAG="v3.016"
ROBOTO_COMMIT="5166f3d07889bf7d3732fb72e09623d7e52f862b"
ROBOTO_ZIP_FILE="android/Roboto[ital,wdth,wght].ttf"
ICONS_REPO="google/material-design-icons"
ICONS_COMMIT="49d4db35df873165d6bd6ba09b063c7dafbac2f4"

log() { printf '%s\n' "$*" >&2; }

mkdir -p "$OUT"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

fetch() { # fetch URL DEST
  curl --fail --silent --show-error --location --output "$2" "$1"
}

log "Roboto $ROBOTO_TAG"
fetch "https://github.com/$ROBOTO_REPO/releases/download/$ROBOTO_TAG/Roboto_$ROBOTO_TAG.zip" "$TMP/roboto.zip"
unzip -q -o -j "$TMP/roboto.zip" "${ROBOTO_ZIP_FILE//[/\\[}" -d "$TMP"
mv "$TMP/Roboto[ital,wdth,wght].ttf" "$OUT/Roboto-Variable.ttf"
fetch "https://raw.githubusercontent.com/$ROBOTO_REPO/$ROBOTO_COMMIT/OFL.txt" "$OUT/Roboto-LICENSE.txt"

log "Material Icons $ICONS_COMMIT"
fetch "https://raw.githubusercontent.com/$ICONS_REPO/$ICONS_COMMIT/font/MaterialIcons-Regular.ttf" "$OUT/MaterialIcons-Regular.ttf"
fetch "https://raw.githubusercontent.com/$ICONS_REPO/$ICONS_COMMIT/LICENSE" "$OUT/MaterialIcons-LICENSE.txt"

VERSIONS="$OUT/VERSIONS"
if [ -f "$VERSIONS" ]; then
  # Verify the files against the recorded hashes.
  (cd "$OUT" && grep -E '^[0-9a-f]{64}  ' "$VERSIONS" | sha256sum --check --quiet)
  log "fonts match $VERSIONS"
else
  {
    echo "# Pins of the bundled fonts. scripts/fetch-fonts.sh downloads them and checks the hashes."
    echo "roboto_repo=https://github.com/$ROBOTO_REPO"
    echo "roboto_release=$ROBOTO_TAG (file $ROBOTO_ZIP_FILE, renamed Roboto-Variable.ttf)"
    echo "roboto_license_commit=$ROBOTO_COMMIT"
    echo "icons_repo=https://github.com/$ICONS_REPO"
    echo "icons_commit=$ICONS_COMMIT"
    (cd "$OUT" && sha256sum Roboto-Variable.ttf Roboto-LICENSE.txt MaterialIcons-Regular.ttf MaterialIcons-LICENSE.txt)
  } >"$VERSIONS"
  log "wrote $VERSIONS"
fi
