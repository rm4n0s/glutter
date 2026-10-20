package mcu

import (
	"fmt"
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// CorePalettes is the five palettes that a dynamic scheme is made from.
//
// Flutter: material-color-utilities palettes/core_palettes.dart CorePalettes
type CorePalettes struct {
	Primary, Secondary, Tertiary, Neutral, NeutralVariant *TonalPalette
}

// CorePalette is the palettes of the first version of the Material 3 color
// system: key colors that make the primary, secondary, tertiary, neutral,
// neutral variant and error palettes. DynamicScheme replaces it.
//
// Flutter: material-color-utilities palettes/core_palette.dart CorePalette
type CorePalette struct {
	Primary, Secondary, Tertiary, Neutral, NeutralVariant, Error *TonalPalette
}

// CorePaletteSize is the number of the palettes that a list of colors holds.
//
// Flutter: material-color-utilities palettes/core_palette.dart CorePalette.size
const CorePaletteSize = 5

// CorePaletteOf makes the palettes of a color.
//
// Flutter: material-color-utilities palettes/core_palette.dart CorePalette.of
func CorePaletteOf(argb uint32) *CorePalette {
	cam := Cam16FromInt(argb)
	hue, chroma := cam.Hue, cam.Chroma
	return &CorePalette{
		Primary:        TonalPaletteOf(hue, math.Max(48, chroma)),
		Secondary:      TonalPaletteOf(hue, 16),
		Tertiary:       TonalPaletteOf(hue+60, 24),
		Neutral:        TonalPaletteOf(hue, 4),
		NeutralVariant: TonalPaletteOf(hue, 8),
		Error:          TonalPaletteOf(25, 84),
	}
}

// CorePaletteContentOf makes the palettes of a color that keep its chroma.
//
// Flutter: material-color-utilities palettes/core_palette.dart CorePalette.contentOf
func CorePaletteContentOf(argb uint32) *CorePalette {
	cam := Cam16FromInt(argb)
	hue, chroma := cam.Hue, cam.Chroma
	return &CorePalette{
		Primary:        TonalPaletteOf(hue, chroma),
		Secondary:      TonalPaletteOf(hue, chroma/3),
		Tertiary:       TonalPaletteOf(hue+60, chroma/2),
		Neutral:        TonalPaletteOf(hue, math.Min(chroma/12, 4)),
		NeutralVariant: TonalPaletteOf(hue, math.Min(chroma/6, 8)),
		Error:          TonalPaletteOf(25, 84),
	}
}

// CorePaletteFromList makes the palettes from the colors of their common tones,
// the primary palette first. In debug builds it panics with the tag
// CorePaletteListSize when the list does not have the right size.
//
// Flutter: material-color-utilities palettes/core_palette.dart CorePalette.fromList
func CorePaletteFromList(colors []uint32) *CorePalette {
	if foundation.DebugMode && len(colors) != CorePaletteSize*CommonSize {
		panic(errors.New("CorePaletteListSize", "a core palette needs one color for each common tone of each palette", "colors", len(colors), "want", CorePaletteSize*CommonSize))
	}
	part := func(n int) *TonalPalette { return TonalPaletteFromList(colors[n*CommonSize : (n+1)*CommonSize]) }
	return &CorePalette{
		Primary: part(0), Secondary: part(1), Tertiary: part(2), Neutral: part(3), NeutralVariant: part(4),
		Error: TonalPaletteOf(25, 84),
	}
}

// AsList is the colors of the common tones of the five palettes (not the error
// palette), the primary palette first.
//
// Flutter: material-color-utilities palettes/core_palette.dart CorePalette.asList
func (c *CorePalette) AsList() []uint32 {
	var out []uint32
	for _, p := range []*TonalPalette{c.Primary, c.Secondary, c.Tertiary, c.Neutral, c.NeutralVariant} {
		out = append(out, p.AsList()...)
	}
	return out
}

// Equal reports whether the palettes are the same.
//
// Flutter: material-color-utilities palettes/core_palette.dart CorePalette.==
func (c *CorePalette) Equal(o *CorePalette) bool {
	return o != nil && c.Primary.Equal(o.Primary) && c.Secondary.Equal(o.Secondary) && c.Tertiary.Equal(o.Tertiary) &&
		c.Neutral.Equal(o.Neutral) && c.NeutralVariant.Equal(o.NeutralVariant) && c.Error.Equal(o.Error)
}

// String lists the palettes.
//
// Flutter: material-color-utilities palettes/core_palette.dart CorePalette.toString
func (c *CorePalette) String() string {
	return fmt.Sprintf("primary: %v\nsecondary: %v\ntertiary: %v\nneutral: %v\nneutralVariant: %v\nerror: %v\n",
		c.Primary, c.Secondary, c.Tertiary, c.Neutral, c.NeutralVariant, c.Error)
}
