package ui

import (
	"strconv"

	"github.com/rm4n0s/errors"
)

// FontStyle is whether to use italic or upright glyphs.
//
// Flutter: ui/text.dart FontStyle
type FontStyle int

// The values of FontStyle follow Flutter's declaration order.
const (
	// FontStyleUnspecified is the zero value: the default (upright).
	//
	// Flutter: no counterpart
	FontStyleUnspecified FontStyle = iota
	// FontStyleNormal uses the upright ("Roman") glyphs.
	//
	// Flutter: ui/text.dart FontStyle.normal
	FontStyleNormal
	// FontStyleItalic uses glyphs that have a more pronounced angle and typically a cursive style.
	//
	// Flutter: ui/text.dart FontStyle.italic
	FontStyleItalic
)

// String returns the name Dart prints for the value, such as "FontStyle.italic".
func (e FontStyle) String() string {
	return dartEnumName([]string{"unspecified", "normal", "italic"}, int(e), "FontStyle")
}

// TextAlign is how the text is aligned horizontally.
//
// Flutter: ui/text.dart TextAlign
type TextAlign int

// The values of TextAlign follow Flutter's declaration order.
const (
	// TextAlignUnspecified is the zero value: the default (start).
	//
	// Flutter: no counterpart
	TextAlignUnspecified TextAlign = iota
	// TextAlignLeft aligns the text on the left edge of the container.
	//
	// Flutter: ui/text.dart TextAlign.left
	TextAlignLeft
	// TextAlignRight aligns the text on the right edge of the container.
	//
	// Flutter: ui/text.dart TextAlign.right
	TextAlignRight
	// TextAlignCenter aligns the text in the center of the container.
	//
	// Flutter: ui/text.dart TextAlign.center
	TextAlignCenter
	// TextAlignJustify stretches the lines that end in a soft line break to fill the width of the container.
	//
	// Flutter: ui/text.dart TextAlign.justify
	TextAlignJustify
	// TextAlignStart aligns the text on the leading edge of the container.
	//
	// Flutter: ui/text.dart TextAlign.start
	TextAlignStart
	// TextAlignEnd aligns the text on the trailing edge of the container.
	//
	// Flutter: ui/text.dart TextAlign.end
	TextAlignEnd
)

// String returns the name Dart prints for the value, such as "TextAlign.left".
func (e TextAlign) String() string {
	return dartEnumName([]string{"unspecified", "left", "right", "center", "justify", "start", "end"}, int(e), "TextAlign")
}

// TextBaseline is a horizontal line used for aligning text.
//
// Flutter: ui/text.dart TextBaseline
type TextBaseline int

// The values of TextBaseline follow Flutter's declaration order.
const (
	// TextBaselineUnspecified is the zero value: the default (alphabetic).
	//
	// Flutter: no counterpart
	TextBaselineUnspecified TextBaseline = iota
	// TextBaselineAlphabetic is the horizontal line used to align the bottom of glyphs for alphabetic characters.
	//
	// Flutter: ui/text.dart TextBaseline.alphabetic
	TextBaselineAlphabetic
	// TextBaselineIdeographic is the horizontal line used to align ideographic characters.
	//
	// Flutter: ui/text.dart TextBaseline.ideographic
	TextBaselineIdeographic
)

// String returns the name Dart prints for the value, such as "TextBaseline.alphabetic".
func (e TextBaseline) String() string {
	return dartEnumName([]string{"unspecified", "alphabetic", "ideographic"}, int(e), "TextBaseline")
}

// TextDecorationStyle is the style in which to draw a text decoration.
//
// Flutter: ui/text.dart TextDecorationStyle
type TextDecorationStyle int

// The values of TextDecorationStyle follow Flutter's declaration order.
const (
	// TextDecorationStyleUnspecified is the zero value: the default (solid).
	//
	// Flutter: no counterpart
	TextDecorationStyleUnspecified TextDecorationStyle = iota
	// TextDecorationStyleSolid draws a solid line.
	//
	// Flutter: ui/text.dart TextDecorationStyle.solid
	TextDecorationStyleSolid
	// TextDecorationStyleDouble draws two lines.
	//
	// Flutter: ui/text.dart TextDecorationStyle.double
	TextDecorationStyleDouble
	// TextDecorationStyleDotted draws a dotted line.
	//
	// Flutter: ui/text.dart TextDecorationStyle.dotted
	TextDecorationStyleDotted
	// TextDecorationStyleDashed draws a dashed line.
	//
	// Flutter: ui/text.dart TextDecorationStyle.dashed
	TextDecorationStyleDashed
	// TextDecorationStyleWavy draws a sinusoidal line.
	//
	// Flutter: ui/text.dart TextDecorationStyle.wavy
	TextDecorationStyleWavy
)

// String returns the name Dart prints for the value, such as "TextDecorationStyle.wavy".
func (e TextDecorationStyle) String() string {
	return dartEnumName([]string{"unspecified", "solid", "double", "dotted", "dashed", "wavy"}, int(e), "TextDecorationStyle")
}

// TextLeadingDistribution is how the extra vertical space of a line (the line height minus the font's own height) is divided.
//
// Flutter: ui/text.dart TextLeadingDistribution
type TextLeadingDistribution int

// The values of TextLeadingDistribution follow Flutter's declaration order.
const (
	// TextLeadingDistributionUnspecified is the zero value: the default (proportional).
	//
	// Flutter: no counterpart
	TextLeadingDistributionUnspecified TextLeadingDistribution = iota
	// TextLeadingDistributionProportional distributes the leading in proportion to the font's ascent and descent.
	//
	// Flutter: ui/text.dart TextLeadingDistribution.proportional
	TextLeadingDistributionProportional
	// TextLeadingDistributionEven puts half of the leading above the ascent and half below the descent.
	//
	// Flutter: ui/text.dart TextLeadingDistribution.even
	TextLeadingDistributionEven
)

// String returns the name Dart prints for the value, such as "TextLeadingDistribution.even".
func (e TextLeadingDistribution) String() string {
	return dartEnumName([]string{"unspecified", "proportional", "even"}, int(e), "TextLeadingDistribution")
}

// TextDirection is a direction in which text flows.
//
// Flutter: ui/text.dart TextDirection
type TextDirection int

// The values of TextDirection follow Flutter's declaration order (rtl first).
const (
	// TextDirectionUnspecified is the zero value: the default (left to right).
	//
	// Flutter: no counterpart
	TextDirectionUnspecified TextDirection = iota
	// TextDirectionRTL is the text flows from right to left (e.g. Arabic, Hebrew).
	//
	// Flutter: ui/text.dart TextDirection.rtl
	TextDirectionRTL
	// TextDirectionLTR is the text flows from left to right (e.g., English, French).
	//
	// Flutter: ui/text.dart TextDirection.ltr
	TextDirectionLTR
)

// String returns the name Dart prints for the value, such as "TextDirection.rtl".
func (e TextDirection) String() string {
	return dartEnumName([]string{"unspecified", "rtl", "ltr"}, int(e), "TextDirection")
}

// IsRTL reports whether the direction is right to left. The unspecified direction is left to right.
//
// Flutter: no counterpart
func (e TextDirection) IsRTL() bool { return e == TextDirectionRTL }

// FontWeight is the thickness of the glyphs used to draw the text, from 1 to 1000. The zero value is unspecified: the default (400).
//
// Flutter: ui/text.dart FontWeight
type FontWeight int

// The conventional weights.
const (
	// FontWeightW100 is thin, the least thick.
	//
	// Flutter: ui/text.dart FontWeight.w100
	FontWeightW100 FontWeight = 100
	// FontWeightW200 is extra-light.
	//
	// Flutter: ui/text.dart FontWeight.w200
	FontWeightW200 FontWeight = 200
	// FontWeightW300 is light.
	//
	// Flutter: ui/text.dart FontWeight.w300
	FontWeightW300 FontWeight = 300
	// FontWeightW400 is normal / regular / plain.
	//
	// Flutter: ui/text.dart FontWeight.w400
	FontWeightW400 FontWeight = 400
	// FontWeightW500 is medium.
	//
	// Flutter: ui/text.dart FontWeight.w500
	FontWeightW500 FontWeight = 500
	// FontWeightW600 is semi-bold.
	//
	// Flutter: ui/text.dart FontWeight.w600
	FontWeightW600 FontWeight = 600
	// FontWeightW700 is bold.
	//
	// Flutter: ui/text.dart FontWeight.w700
	FontWeightW700 FontWeight = 700
	// FontWeightW800 is extra-bold.
	//
	// Flutter: ui/text.dart FontWeight.w800
	FontWeightW800 FontWeight = 800
	// FontWeightW900 is black, the most thick.
	//
	// Flutter: ui/text.dart FontWeight.w900
	FontWeightW900 FontWeight = 900
	// FontWeightNormal is the default font weight (w400).
	//
	// Flutter: ui/text.dart FontWeight.normal
	FontWeightNormal = FontWeightW400
	// FontWeightBold is a commonly used font weight that is heavier than normal (w700).
	//
	// Flutter: ui/text.dart FontWeight.bold
	FontWeightBold = FontWeightW700
)

// NewFontWeight returns the weight with the given value. A value outside 1 to 1000 panics with the tag FontWeightInvalid.
//
// Flutter: ui/text.dart FontWeight
func NewFontWeight(value int) FontWeight {
	if value < 1 || value > 1000 {
		panic(errors.New(tagFontWeightInvalid, "font weight must be between 1 and 1000", "value", value))
	}
	return FontWeight(value)
}

// Value is the weight as a number from 1 to 1000, or 0 when unspecified.
//
// Flutter: ui/text.dart FontWeight.value
func (w FontWeight) Value() int { return int(w) }

// String returns the name Dart prints for the value, such as "FontWeight.w400".
func (w FontWeight) String() string {
	if w >= 100 && w <= 900 && w%100 == 0 {
		return "FontWeight.w" + strconv.Itoa(int(w))
	}
	return "FontWeight(" + strconv.Itoa(int(w)) + ")"
}

// TextDecoration is a linear decoration to draw near the text: a bit mask of underline, overline and line through.
//
// Flutter: ui/text.dart TextDecoration
type TextDecoration int

// The decorations, combined with Combine or by or-ing.
const (
	// TextDecorationNone draws no decoration.
	//
	// Flutter: ui/text.dart TextDecoration.none
	TextDecorationNone TextDecoration = 0
	// TextDecorationUnderline draws a line underneath each line of text.
	//
	// Flutter: ui/text.dart TextDecoration.underline
	TextDecorationUnderline TextDecoration = 1
	// TextDecorationOverline draws a line above each line of text.
	//
	// Flutter: ui/text.dart TextDecoration.overline
	TextDecorationOverline TextDecoration = 2
	// TextDecorationLineThrough draws a line through each line of text.
	//
	// Flutter: ui/text.dart TextDecoration.lineThrough
	TextDecorationLineThrough TextDecoration = 4
)

// CombineTextDecorations creates a decoration that paints the union of all the given decorations.
//
// Flutter: ui/text.dart TextDecoration.combine
func CombineTextDecorations(decorations ...TextDecoration) TextDecoration {
	var mask TextDecoration
	for _, d := range decorations {
		mask |= d
	}
	return mask
}

// Contains reports whether this decoration will paint at least as much decoration as the other.
//
// Flutter: ui/text.dart TextDecoration.contains
func (d TextDecoration) Contains(other TextDecoration) bool { return d|other == d }

// String returns what Dart prints for the value, such as "TextDecoration.underline".
func (d TextDecoration) String() string {
	if d == 0 {
		return "TextDecoration.none"
	}
	var names []string
	for _, p := range []struct {
		bit  TextDecoration
		name string
	}{{TextDecorationUnderline, "underline"}, {TextDecorationOverline, "overline"}, {TextDecorationLineThrough, "lineThrough"}} {
		if d&p.bit != 0 {
			names = append(names, p.name)
		}
	}
	if len(names) == 1 {
		return "TextDecoration." + names[0]
	}
	out := "TextDecoration.combine(["
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out + "])"
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagFontWeightInvalid = "FontWeightInvalid"
)
