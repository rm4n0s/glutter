package ui

import (
	"fmt"
	"strings"
)

// Paint is a description of the style to use when drawing on a Canvas.
//
// Most APIs on Canvas take a Paint object to describe the style to use for that
// operation. The zero value is Flutter's `Paint()`: opaque black, filled, with
// anti-aliasing, blend mode srcOver, and no filters or shader. Copy a Paint by
// assigning it; every part of it is immutable or a value.
//
// Flutter's properties become a getter and a Set method, such as Color and
// SetColor, so that the zero value can be the default.
//
// Flutter: ui/painting.dart Paint
type Paint struct {
	noAntiAlias bool
	// color is the color, unless colorSet is false, which means black.
	color            Color
	colorSet         bool
	blendMode        BlendMode // BlendModeUnspecified is srcOver
	style            PaintingStyle
	strokeWidth      float32
	strokeCap        StrokeCap
	strokeJoin       StrokeJoin
	strokeMiterDelta float32 // the miter limit minus its default, 4
	filterQuality    FilterQuality
	maskFilter       *MaskFilter
	shader           Shader
	colorFilter      *ColorFilter
	imageFilter      ImageFilter
	invertColors     bool
}

// strokeMiterLimitDefault is the default miter limit.
const strokeMiterLimitDefault = 4.0

// NewPaint creates a Paint with Flutter's defaults. It is the same as the zero
// value.
//
// Flutter: ui/painting.dart Paint
func NewPaint() *Paint { return &Paint{} }

// NewPaintFrom creates a copy of other.
//
// Flutter: ui/painting.dart Paint.from
func NewPaintFrom(other *Paint) *Paint {
	c := *other
	return &c
}

// IsAntiAlias reports whether to apply anti-aliasing to lines and images drawn
// on the canvas. It defaults to true.
//
// Flutter: ui/painting.dart Paint.isAntiAlias
func (p *Paint) IsAntiAlias() bool { return !p.noAntiAlias }

// SetIsAntiAlias sets whether to apply anti-aliasing.
//
// Flutter: ui/painting.dart Paint.isAntiAlias
func (p *Paint) SetIsAntiAlias(v bool) { p.noAntiAlias = !v }

// paintColorDefault is Flutter's default paint color, 0xFF000000.
var paintColorDefault = Color{A: 1}

// Color is the color to use when stroking or filling a shape. It defaults to
// opaque black.
//
// Flutter: ui/painting.dart Paint.color
func (p *Paint) Color() Color {
	if !p.colorSet {
		return paintColorDefault
	}
	return p.color
}

// SetColor sets the color to use when stroking or filling a shape.
//
// Flutter: ui/painting.dart Paint.color
func (p *Paint) SetColor(c Color) {
	if c == paintColorDefault {
		p.color, p.colorSet = Color{}, false
		return
	}
	p.color, p.colorSet = c, true
}

// BlendMode is the blend mode to apply when a shape is drawn or a layer is
// composited. It defaults to BlendModeSrcOver.
//
// Flutter: ui/painting.dart Paint.blendMode
func (p *Paint) BlendMode() BlendMode {
	if p.blendMode == BlendModeUnspecified {
		return BlendModeSrcOver
	}
	return p.blendMode
}

// SetBlendMode sets the blend mode. BlendModeUnspecified is BlendModeSrcOver.
//
// Flutter: ui/painting.dart Paint.blendMode
func (p *Paint) SetBlendMode(m BlendMode) {
	if m == BlendModeSrcOver {
		m = BlendModeUnspecified
	}
	p.blendMode = m
}

// Style is whether to paint inside shapes, the edges of shapes, or both. It
// defaults to PaintingStyleFill.
//
// Flutter: ui/painting.dart Paint.style
func (p *Paint) Style() PaintingStyle {
	if p.style == PaintingStyleUnspecified {
		return PaintingStyleFill
	}
	return p.style
}

// SetStyle sets whether to paint inside shapes, the edges of shapes, or both.
// PaintingStyleUnspecified is PaintingStyleFill.
//
// Flutter: ui/painting.dart Paint.style
func (p *Paint) SetStyle(s PaintingStyle) {
	if s == PaintingStyleFill {
		s = PaintingStyleUnspecified
	}
	p.style = s
}

// StrokeWidth is how wide to make edges drawn when the style is
// PaintingStyleStroke. A width of zero is a hairline. It defaults to 0.0. The
// value is held as a float32, as Flutter does.
//
// Flutter: ui/painting.dart Paint.strokeWidth
func (p *Paint) StrokeWidth() float64 { return float64(p.strokeWidth) }

// SetStrokeWidth sets how wide to make edges drawn when stroking.
//
// Flutter: ui/painting.dart Paint.strokeWidth
func (p *Paint) SetStrokeWidth(w float64) { p.strokeWidth = float32(w) }

// StrokeCap is the kind of finish to place on the end of lines drawn when the
// style is PaintingStyleStroke. It defaults to StrokeCapButt.
//
// Flutter: ui/painting.dart Paint.strokeCap
func (p *Paint) StrokeCap() StrokeCap {
	if p.strokeCap == StrokeCapUnspecified {
		return StrokeCapButt
	}
	return p.strokeCap
}

// SetStrokeCap sets the finish on the end of stroked lines.
// StrokeCapUnspecified is StrokeCapButt.
//
// Flutter: ui/painting.dart Paint.strokeCap
func (p *Paint) SetStrokeCap(c StrokeCap) {
	if c == StrokeCapButt {
		c = StrokeCapUnspecified
	}
	p.strokeCap = c
}

// StrokeJoin is the kind of finish to place on the joins between segments. It
// defaults to StrokeJoinMiter.
//
// Flutter: ui/painting.dart Paint.strokeJoin
func (p *Paint) StrokeJoin() StrokeJoin {
	if p.strokeJoin == StrokeJoinUnspecified {
		return StrokeJoinMiter
	}
	return p.strokeJoin
}

// SetStrokeJoin sets the finish on the joins between segments.
// StrokeJoinUnspecified is StrokeJoinMiter.
//
// Flutter: ui/painting.dart Paint.strokeJoin
func (p *Paint) SetStrokeJoin(j StrokeJoin) {
	if j == StrokeJoinMiter {
		j = StrokeJoinUnspecified
	}
	p.strokeJoin = j
}

// StrokeMiterLimit is the limit for miters to be drawn on segments when the join
// is set to StrokeJoinMiter. It defaults to 4.0.
//
// Flutter: ui/painting.dart Paint.strokeMiterLimit
func (p *Paint) StrokeMiterLimit() float64 {
	return float64(p.strokeMiterDelta) + strokeMiterLimitDefault
}

// SetStrokeMiterLimit sets the limit for miters.
//
// Flutter: ui/painting.dart Paint.strokeMiterLimit
func (p *Paint) SetStrokeMiterLimit(v float64) {
	p.strokeMiterDelta = float32(v - strokeMiterLimitDefault)
}

// MaskFilter is the MaskFilter to use when drawing, or nil for none.
//
// Flutter: ui/painting.dart Paint.maskFilter
func (p *Paint) MaskFilter() *MaskFilter { return p.maskFilter }

// SetMaskFilter sets the MaskFilter; nil removes it.
//
// Flutter: ui/painting.dart Paint.maskFilter
func (p *Paint) SetMaskFilter(f *MaskFilter) { p.maskFilter = f }

// FilterQuality is the quality of sampling used when scaling images. It defaults
// to FilterQualityNone.
//
// Flutter: ui/painting.dart Paint.filterQuality
func (p *Paint) FilterQuality() FilterQuality {
	if p.filterQuality == FilterQualityUnspecified {
		return FilterQualityNone
	}
	return p.filterQuality
}

// SetFilterQuality sets the quality of sampling when scaling images.
// FilterQualityUnspecified is FilterQualityNone.
//
// Flutter: ui/painting.dart Paint.filterQuality
func (p *Paint) SetFilterQuality(q FilterQuality) {
	if q == FilterQualityNone {
		q = FilterQualityUnspecified
	}
	p.filterQuality = q
}

// Shader is the shader to use when drawing, or nil for none. When it is set, the
// color is ignored.
//
// Flutter: ui/painting.dart Paint.shader
func (p *Paint) Shader() Shader { return p.shader }

// SetShader sets the shader; nil removes it.
//
// Flutter: ui/painting.dart Paint.shader
func (p *Paint) SetShader(s Shader) { p.shader = s }

// ColorFilter is the ColorFilter to apply when drawing, or nil for none.
//
// Flutter: ui/painting.dart Paint.colorFilter
func (p *Paint) ColorFilter() *ColorFilter { return p.colorFilter }

// SetColorFilter sets the ColorFilter; nil removes it.
//
// Flutter: ui/painting.dart Paint.colorFilter
func (p *Paint) SetColorFilter(f *ColorFilter) { p.colorFilter = f }

// ImageFilter is the ImageFilter to apply to what is drawn, or nil for none.
//
// Flutter: ui/painting.dart Paint.imageFilter
func (p *Paint) ImageFilter() ImageFilter { return p.imageFilter }

// SetImageFilter sets the ImageFilter; nil removes it.
//
// Flutter: ui/painting.dart Paint.imageFilter
func (p *Paint) SetImageFilter(f ImageFilter) { p.imageFilter = f }

// InvertColors reports whether the colors are inverted when drawing. It defaults
// to false.
//
// Flutter: ui/painting.dart Paint.invertColors
func (p *Paint) InvertColors() bool { return p.invertColors }

// SetInvertColors sets whether the colors are inverted when drawing.
//
// Flutter: ui/painting.dart Paint.invertColors
func (p *Paint) SetInvertColors(v bool) { p.invertColors = v }

// String describes the paint, as Dart's toString does.
//
// Flutter: ui/painting.dart Paint.toString
func (p *Paint) String() string {
	var result strings.Builder
	semicolon := ""
	result.WriteString("Paint(")
	if p.Style() == PaintingStyleStroke {
		fmt.Fprintf(&result, "%s", p.Style())
		if p.StrokeWidth() != 0.0 {
			fmt.Fprintf(&result, " %s", toStringAsFixed1(p.StrokeWidth()))
		} else {
			result.WriteString(" hairline")
		}
		if p.StrokeCap() != StrokeCapButt {
			fmt.Fprintf(&result, " %s", p.StrokeCap())
		}
		if p.StrokeJoin() == StrokeJoinMiter {
			if p.StrokeMiterLimit() != strokeMiterLimitDefault {
				fmt.Fprintf(&result, " %s up to %s", p.StrokeJoin(), toStringAsFixed1(p.StrokeMiterLimit()))
			}
		} else {
			fmt.Fprintf(&result, " %s", p.StrokeJoin())
		}
		semicolon = "; "
	}
	if !p.IsAntiAlias() {
		result.WriteString(semicolon + "antialias off")
		semicolon = "; "
	}
	if p.Color() != paintColorDefault {
		result.WriteString(semicolon + p.Color().String())
		semicolon = "; "
	}
	if p.BlendMode() != BlendModeSrcOver {
		result.WriteString(semicolon + p.BlendMode().String())
		semicolon = "; "
	}
	if p.colorFilter != nil {
		result.WriteString(semicolon + "colorFilter: " + p.colorFilter.String())
		semicolon = "; "
	}
	if p.maskFilter != nil {
		result.WriteString(semicolon + "maskFilter: " + p.maskFilter.String())
		semicolon = "; "
	}
	if p.FilterQuality() != FilterQualityNone {
		result.WriteString(semicolon + "filterQuality: " + p.FilterQuality().String())
		semicolon = "; "
	}
	if p.shader != nil {
		fmt.Fprintf(&result, "%sshader: %v", semicolon, p.shader)
		semicolon = "; "
	}
	if p.imageFilter != nil {
		result.WriteString(semicolon + "imageFilter: " + p.imageFilter.String())
		semicolon = "; "
	}
	if p.invertColors {
		result.WriteString(semicolon + "invert: true")
	}
	result.WriteString(")")
	return result.String()
}
