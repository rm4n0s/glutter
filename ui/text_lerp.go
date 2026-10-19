package ui

import "math"

// TextHeightNone is the sentinel for TextStyle heights: it asks the line to use the font's own height.
//
// Flutter: ui/text.dart kTextHeightNone
const TextHeightNone = 0.0

// FontWeightLerp linearly interpolates between two font weights, where FontWeight(0) is Flutter's null: it counts as normal when
// the other is set. It returns 0 for two nulls. The result is rounded and clamped to 100 to 900.
//
// Flutter: ui/text.dart FontWeight.lerp
func FontWeightLerp(a, b FontWeight, t float64) FontWeight {
	if a == 0 && b == 0 {
		return 0
	}
	av, bv := FontWeightNormal, FontWeightNormal
	if a != 0 {
		av = a
	}
	if b != 0 {
		bv = b
	}
	v := math.Round(float64(av) + float64(bv-av)*t)
	return FontWeight(ClampDouble(v, 100, 900))
}

// FontVariationLerp linearly interpolates between two font variations. Either may be nil. When the axes differ, or both are nil, the
// result is a for t < 0.5 and b otherwise.
//
// Flutter: ui/text.dart FontVariation.lerp
func FontVariationLerp(a, b *FontVariation, t float64) *FontVariation {
	if (a == nil) != (b == nil) || a == nil || a.Axis != b.Axis {
		if t < 0.5 {
			return a
		}
		return b
	}
	return &FontVariation{a.Axis, ClampDouble(LerpDouble(a.Value, b.Value, t), -32768.0, 32768.0-1.0/65536.0)}
}

// Scale returns a shadow with its offset and blur radius scaled by the factor.
//
// Flutter: ui/painting.dart Shadow.scale
func (s Shadow) Scale(factor float64) Shadow {
	return Shadow{Color: s.Color, Offset: s.Offset.Mul(factor), BlurRadius: s.BlurRadius * factor}
}

// ShadowLerp linearly interpolates between two shadows. Either may be nil; two nils give nil.
//
// Flutter: ui/painting.dart Shadow.lerp
func ShadowLerp(a, b *Shadow, t float64) *Shadow {
	switch {
	case b == nil && a == nil:
		return nil
	case b == nil:
		r := a.Scale(1.0 - t)
		return &r
	case a == nil:
		r := b.Scale(t)
		return &r
	}
	return &Shadow{Color: a.Color.Lerp(b.Color, t), Offset: a.Offset.Lerp(b.Offset, t), BlurRadius: LerpDouble(a.BlurRadius, b.BlurRadius, t)}
}

// ShadowLerpList linearly interpolates between two lists of shadows; the shorter list is padded with transparent shadows. Two nil lists
// give nil.
//
// Flutter: ui/painting.dart Shadow.lerpList
func ShadowLerpList(a, b []Shadow, t float64) []Shadow {
	if a == nil && b == nil {
		return nil
	}
	common := min(len(a), len(b))
	out := make([]Shadow, 0, max(len(a), len(b)))
	for i := 0; i < common; i++ {
		out = append(out, *ShadowLerp(&a[i], &b[i], t))
	}
	for i := common; i < len(a); i++ {
		out = append(out, a[i].Scale(1.0-t))
	}
	for i := common; i < len(b); i++ {
		out = append(out, b[i].Scale(t))
	}
	return out
}
