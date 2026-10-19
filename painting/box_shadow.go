package painting

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// BoxShadow is a shadow cast by a box. Unlike ui.Shadow it can spread, and it
// can be drawn with a blur style.
//
// Flutter: painting/box_shadow.dart BoxShadow
type BoxShadow struct {
	ui.Shadow
	// SpreadRadius is how much the shadow expands, in logical pixels, before
	// the blur is applied.
	SpreadRadius float64
	// BlurStyle is the style of the blur; ui.BlurStyleUnspecified means normal.
	BlurStyle ui.BlurStyle
}

// BoxShadowOpts are the optional parameters of NewBoxShadow.
//
// Flutter: painting/box_shadow.dart BoxShadow.new
type BoxShadowOpts struct {
	// Color defaults to opaque black.
	Color foundation.Opt[ui.Color]
	// Offset defaults to zero.
	Offset ui.Offset
	// BlurRadius defaults to 0.
	BlurRadius float64
	// SpreadRadius defaults to 0.
	SpreadRadius float64
	// BlurStyle defaults to ui.BlurStyleNormal.
	BlurStyle ui.BlurStyle
}

// NewBoxShadow creates a box shadow.
//
// Flutter: painting/box_shadow.dart BoxShadow.new
func NewBoxShadow(opts ...BoxShadowOpts) BoxShadow {
	o := firstOpts(opts)
	s := BoxShadow{
		Shadow:       ui.NewShadow(ui.ShadowOpts{Offset: o.Offset, BlurRadius: o.BlurRadius}),
		SpreadRadius: o.SpreadRadius,
		BlurStyle:    o.BlurStyle,
	}
	if c, ok := o.Color.Get(); ok {
		s.Color = c
	}
	if s.BlurStyle == ui.BlurStyleUnspecified {
		s.BlurStyle = ui.BlurStyleNormal
	}
	return s
}

// ToPaint creates a Paint that, when used to paint a box, draws the shadow.
//
// Flutter: painting/box_shadow.dart BoxShadow.toPaint
func (s BoxShadow) ToPaint() *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(s.Color)
	p.SetMaskFilter(ui.NewMaskFilterBlur(s.BlurStyle, s.BlurSigma()))
	return p
}

// Scale returns a new box shadow with its offset, blurRadius and spreadRadius
// scaled by the factor.
//
// Flutter: painting/box_shadow.dart BoxShadow.scale
func (s BoxShadow) Scale(factor float64) BoxShadow {
	return BoxShadow{
		Shadow:       ui.Shadow{Color: s.Color, Offset: s.Offset.Mul(factor), BlurRadius: s.BlurRadius * factor},
		SpreadRadius: s.SpreadRadius * factor,
		BlurStyle:    s.BlurStyle,
	}
}

// BoxShadowCopy holds the replacements for BoxShadow.CopyWith.
//
// Flutter: painting/box_shadow.dart BoxShadow.copyWith
type BoxShadowCopy struct {
	Color        foundation.Opt[ui.Color]
	Offset       foundation.Opt[ui.Offset]
	BlurRadius   foundation.Opt[float64]
	SpreadRadius foundation.Opt[float64]
	BlurStyle    foundation.Opt[ui.BlurStyle]
}

// CopyWith returns a copy of the shadow with the given fields replaced.
//
// Flutter: painting/box_shadow.dart BoxShadow.copyWith
func (s BoxShadow) CopyWith(c BoxShadowCopy) BoxShadow {
	return BoxShadow{
		Shadow: ui.Shadow{
			Color:      c.Color.OrElse(s.Color),
			Offset:     c.Offset.OrElse(s.Offset),
			BlurRadius: c.BlurRadius.OrElse(s.BlurRadius),
		},
		SpreadRadius: c.SpreadRadius.OrElse(s.SpreadRadius),
		BlurStyle:    c.BlurStyle.OrElse(s.BlurStyle),
	}
}

// BoxShadowLerp linearly interpolates between two box shadows. Either may be
// nil; the result is nil only for two nils.
//
// Flutter: painting/box_shadow.dart BoxShadow.lerp
func BoxShadowLerp(a, b *BoxShadow, t float64) *BoxShadow {
	if a == b {
		return a
	}
	if a == nil {
		r := b.Scale(t)
		return &r
	}
	if b == nil {
		r := a.Scale(1.0 - t)
		return &r
	}
	style := a.BlurStyle
	if style == ui.BlurStyleNormal {
		style = b.BlurStyle
	}
	return &BoxShadow{
		Shadow: ui.Shadow{
			Color:      a.Color.Lerp(b.Color, t),
			Offset:     a.Offset.Lerp(b.Offset, t),
			BlurRadius: ui.LerpDouble(a.BlurRadius, b.BlurRadius, t),
		},
		SpreadRadius: ui.LerpDouble(a.SpreadRadius, b.SpreadRadius, t),
		BlurStyle:    style,
	}
}

// BoxShadowLerpList linearly interpolates between two lists of box shadows. The
// shorter list is padded with transparent shadows.
//
// Flutter: painting/box_shadow.dart BoxShadow.lerpList
func BoxShadowLerpList(a, b []BoxShadow, t float64) []BoxShadow {
	if a == nil && b == nil {
		return nil
	}
	common := min(len(a), len(b))
	out := make([]BoxShadow, 0, max(len(a), len(b)))
	for i := 0; i < common; i++ {
		out = append(out, *BoxShadowLerp(&a[i], &b[i], t))
	}
	for i := common; i < len(a); i++ {
		out = append(out, a[i].Scale(1.0-t))
	}
	for i := common; i < len(b); i++ {
		out = append(out, b[i].Scale(t))
	}
	return out
}

// String describes the shadow, such as "BoxShadow(Color(0xff000000), Offset(0.0, 0.0), 0.0, 0.0, BlurStyle.normal)".
//
// Flutter: painting/box_shadow.dart BoxShadow.toString
func (s BoxShadow) String() string {
	return "BoxShadow(" + s.Color.String() + ", " + s.Offset.String() + ", " +
		foundation.DebugFormatDouble(foundation.Some(s.BlurRadius)) + ", " +
		foundation.DebugFormatDouble(foundation.Some(s.SpreadRadius)) + ", " + s.BlurStyle.String() + ")"
}
