#!/usr/bin/env bash
# Fetches the fonts that Glutter bundles (docs/SPEC.md §10) into material/fonts/.
#
# Usage: scripts/fetch-fonts.sh   (from anywhere; works relative to the repo root)
#
# Downloads Roboto from its official GitHub repository and the Material Icons font
# from the fonts archive that the Flutter SDK downloads (the font of Flutter has its
# own code points: Icons.add is 0xe047, not the 0xe145 of Google's repository), at
# the pins recorded in material/fonts/VERSIONS, and checks every file against the
# SHA-256 recorded there.
# To move to a newer font, change the pins in this script, delete the font files and
# VERSIONS, run it again, and review the new VERSIONS.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/material/fonts"

ROBOTO_REPO="googlefonts/roboto-3-classic"
ROBOTO_TAG="v3.016"
ROBOTO_COMMIT="5166f3d07889bf7d3732fb72e09623d7e52f862b"
ROBOTO_ZIP_FILE="android/Roboto[ital,wdth,wght].ttf"
# The archive of Flutter's fonts is named in bin/internal/material_fonts.version of the
# Flutter commit that reference/VERSION pins.
ICONS_FLUTTER_COMMIT="abaf9c523780a608bd46686fd5e53740a07077f8"
ICONS_ARCHIVE="flutter_infra_release/flutter/fonts/3012db47f3130e62f7cc0beabff968a33cbec8d8/fonts.zip"

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

log "Material Icons ($ICONS_ARCHIVE)"
fetch "https://storage.googleapis.com/$ICONS_ARCHIVE" "$TMP/fonts.zip"
unzip -q -o -j "$TMP/fonts.zip" MaterialIcons-Regular.otf MaterialIcons_LICENSE.txt -d "$TMP"
mv "$TMP/MaterialIcons-Regular.otf" "$OUT/MaterialIcons-Regular.otf"
mv "$TMP/MaterialIcons_LICENSE.txt" "$OUT/MaterialIcons-LICENSE.txt"

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
    echo "icons_flutter_repository=https://github.com/flutter/flutter"
    echo "icons_flutter_commit=$ICONS_FLUTTER_COMMIT"
    echo "icons_archive=https://storage.googleapis.com/$ICONS_ARCHIVE (files MaterialIcons-Regular.otf and MaterialIcons_LICENSE.txt, renamed MaterialIcons-LICENSE.txt)"
    (cd "$OUT" && sha256sum Roboto-Variable.ttf Roboto-LICENSE.txt MaterialIcons-Regular.otf MaterialIcons-LICENSE.txt)
  } >"$VERSIONS"
  log "wrote $VERSIONS"
fi
