package mcu

// Variant is a type of dynamic scheme: how its palettes are made from the source
// color.
//
// Flutter: material-color-utilities dynamiccolor/variant.dart Variant
type Variant int

// The variants, in the order of Flutter's enum. VariantUnspecified is the zero
// value and is not a variant.
const (
	// VariantUnspecified is the zero value.
	//
	// Flutter: no counterpart
	VariantUnspecified Variant = iota
	// VariantMonochrome: all colors are grayscale, no chroma.
	//
	// Flutter: material-color-utilities dynamiccolor/variant.dart Variant.monochrome
	VariantMonochrome
	// VariantNeutral: close to grayscale, a hint of chroma.
	//
	// Flutter: material-color-utilities dynamiccolor/variant.dart Variant.neutral
	VariantNeutral
	// VariantTonalSpot: pastel tokens, low chroma palettes (32); the default
	// Material You theme at the 2021 launch.
	//
	// Flutter: material-color-utilities dynamiccolor/variant.dart Variant.tonalSpot
	VariantTonalSpot
	// VariantVibrant: pastel colors, high chroma palettes; the chroma of the
	// primary palette is at its maximum.
	//
	// Flutter: material-color-utilities dynamiccolor/variant.dart Variant.vibrant
	VariantVibrant
	// VariantExpressive: pastel colors, medium chroma palettes; the hue of the
	// primary palette is different from the source color, for variety.
	//
	// Flutter: material-color-utilities dynamiccolor/variant.dart Variant.expressive
	VariantExpressive
	// VariantContent: almost identical to fidelity; tokens and palettes match the
	// source color, and the tertiary palette is an analogue of it.
	//
	// Flutter: material-color-utilities dynamiccolor/variant.dart Variant.content
	VariantContent
	// VariantFidelity: tokens and palettes match the source color, and the tertiary
	// palette is its complement.
	//
	// Flutter: material-color-utilities dynamiccolor/variant.dart Variant.fidelity
	VariantFidelity
	// VariantRainbow: a playful theme; the hue of the source color does not appear
	// in the theme.
	//
	// Flutter: material-color-utilities dynamiccolor/variant.dart Variant.rainbow
	VariantRainbow
	// VariantFruitSalad: a playful theme; the hue of the source color does not
	// appear in the theme.
	//
	// Flutter: material-color-utilities dynamiccolor/variant.dart Variant.fruitSalad
	VariantFruitSalad
)

var variantLabels = [...]string{
	VariantUnspecified: "unspecified",
	VariantMonochrome:  "monochrome", VariantNeutral: "neutral", VariantTonalSpot: "tonal spot", VariantVibrant: "vibrant",
	VariantExpressive: "expressive", VariantContent: "content", VariantFidelity: "fidelity", VariantRainbow: "rainbow",
	VariantFruitSalad: "fruit salad",
}

var variantDescriptions = [...]string{
	VariantUnspecified: "",
	VariantMonochrome:  "All colors are grayscale, no chroma.",
	VariantNeutral:     "Close to grayscale, a hint of chroma.",
	VariantTonalSpot:   "Pastel tokens, low chroma palettes (32).\nDefault Material You theme at 2021 launch.",
	VariantVibrant: "Pastel colors, high chroma palettes. (max).\nThe primary palette's chroma is at maximum.\n" +
		"Use Fidelity instead if tokens should alter their tone to match the palette vibrancy.",
	VariantExpressive: "Pastel colors, medium chroma palettes.\nThe primary palette's hue is different from source color, for variety.",
	VariantContent: "Almost identical to Fidelity.\nTokens and palettes match source color.\n" +
		"Primary Container is source color, adjusted to ensure contrast with surfaces.\n\n" +
		"Tertiary palette is analogue of source color.\nFound by dividing color wheel by 6, then finding the 2 colors adjacent to source.\n" +
		"The one that increases hue is used.",
	VariantFidelity: "Tokens and palettes match source color.\nPrimary Container is source color, adjusted to ensure contrast with surfaces.\n" +
		"For example, if source color is black, it is lightened so it doesn't match surfaces in dark mode.\n\n" +
		"Tertiary palette is complement of source color.",
	VariantRainbow:    "A playful theme - the source color's hue does not appear in the theme.",
	VariantFruitSalad: "A playful theme - the source color's hue does not appear in the theme.",
}

// Label is the name of the variant.
//
// Flutter: material-color-utilities dynamiccolor/variant.dart Variant.label
func (v Variant) Label() string { return variantLabels[v] }

// Description says how the variant makes its palettes.
//
// Flutter: material-color-utilities dynamiccolor/variant.dart Variant.description
func (v Variant) Description() string { return variantDescriptions[v] }
