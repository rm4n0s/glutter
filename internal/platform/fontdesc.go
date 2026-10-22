package platform

import (
	"strconv"
	"strings"
)

// fontStyleWords are the words of a Pango font description that name a style, a weight or a
// width, not the family: they come after the family and before the size.
var fontStyleWords = map[string]bool{
	"thin": true, "ultra-light": true, "ultralight": true, "extra-light": true, "extralight": true,
	"light": true, "semi-light": true, "semilight": true, "demi-light": true, "book": true,
	"regular": true, "normal": true, "medium": true, "semi-bold": true, "semibold": true,
	"demi-bold": true, "demibold": true, "bold": true, "ultra-bold": true, "ultrabold": true,
	"extra-bold": true, "extrabold": true, "heavy": true, "black": true, "ultra-heavy": true,
	"ultraheavy": true, "extra-heavy": true, "italic": true, "oblique": true, "roman": true,
	"small-caps": true, "all-small-caps": true, "petite-caps": true, "all-petite-caps": true,
	"unicase": true, "title-caps": true,
	"ultra-condensed": true, "extra-condensed": true, "condensed": true, "semi-condensed": true,
	"semi-expanded": true, "expanded": true, "extra-expanded": true, "ultra-expanded": true,
}

// ParseFontDescription reads a Pango font description, the form of GNOME's font-name and
// monospace-font-name settings ("Cantarell 11", "Source Code Pro Semi-Bold 10", "Noto Sans
// Mono Bold Italic 12"): a family (the first of a comma separated list), style words that
// are left out, and a size in points. A size in pixels ("12px") is converted to points at 96 dpi.
// The variations after "@" are ignored. size is 0 when the description has none, and ok is false
// when there is no family.
//
// Flutter: no counterpart
func ParseFontDescription(desc string) (family string, size float64, ok bool) {
	desc, _, _ = strings.Cut(desc, "@")
	fields := strings.Fields(desc)
	if n := len(fields); n > 1 {
		last := fields[n-1]
		if v, isPx := strings.CutSuffix(last, "px"); isPx {
			if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
				size = f * 72 / 96
				fields = fields[:n-1]
			}
		} else if f, err := strconv.ParseFloat(last, 64); err == nil && f > 0 {
			size = f
			fields = fields[:n-1]
		}
	}
	// The family is everything before the style words at the end.
	end := len(fields)
	for end > 1 && fontStyleWords[strings.ToLower(fields[end-1])] {
		end--
	}
	family = strings.Join(fields[:end], " ")
	// A list of families, the first is the one: "Noto Sans, Sans".
	if first, _, found := strings.Cut(family, ","); found {
		family = first
	}
	family = strings.Trim(strings.TrimSpace(family), `'"`)
	return family, size, family != ""
}
