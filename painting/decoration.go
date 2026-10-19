package painting

import (
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

// Decoration is a description of a painting to be done around a box. A nil
// Decoration is Flutter's null. The methods LerpFrom and LerpTo are protected
// in Flutter; they are exported so that other packages can implement a
// Decoration. Embed DecorationDefaults for the defaults.
//
// Flutter: painting/decoration.dart Decoration
type Decoration interface {
	foundation.Diagnosticable
	// DebugAssertIsValid asserts that the decoration is valid; it returns true
	// (so that it can be used in assertions).
	DebugAssertIsValid() bool
	// Padding is the insets to use when the decoration is shown around a child:
	// the area the decoration is guaranteed to paint around.
	Padding() EdgeInsetsGeometry
	// IsComplex is whether this decoration is complex enough to benefit from
	// caching its painting.
	IsComplex() bool
	// LerpFrom linearly interpolates from a (possibly nil) decoration to this
	// one; it returns nil if it can't.
	LerpFrom(a Decoration, t float64) Decoration
	// LerpTo linearly interpolates from this decoration to b (possibly nil);
	// it returns nil if it can't.
	LerpTo(b Decoration, t float64) Decoration
	// HitTest tests whether the given point, on a rectangle of the given size,
	// would be considered to hit the decoration.
	HitTest(size ui.Size, position ui.Offset, textDirection ui.TextDirection) bool
	// CreateBoxPainter returns a BoxPainter that will paint this decoration.
	CreateBoxPainter(onChanged func()) BoxPainter
	// GetClipPath returns a closed Path that describes the outer edge of the
	// decoration, for clipping.
	GetClipPath(rect ui.Rect, textDirection ui.TextDirection) *ui.Path
}

// DecorationDefaults gives the defaults of Decoration for embedding. Its
// LerpFrom and LerpTo return nil, its Padding is zero, HitTest is true, and
// GetClipPath panics.
//
// Flutter: painting/decoration.dart Decoration
type DecorationDefaults struct{}

// DebugAssertIsValid returns true.
//
// Flutter: painting/decoration.dart Decoration.debugAssertIsValid
func (DecorationDefaults) DebugAssertIsValid() bool { return true }

// Padding is zero.
//
// Flutter: painting/decoration.dart Decoration.padding
func (DecorationDefaults) Padding() EdgeInsetsGeometry { return EdgeInsetsZero }

// IsComplex is false.
//
// Flutter: painting/decoration.dart Decoration.isComplex
func (DecorationDefaults) IsComplex() bool { return false }

// LerpFrom returns nil.
//
// Flutter: painting/decoration.dart Decoration.lerpFrom
func (DecorationDefaults) LerpFrom(Decoration, float64) Decoration { return nil }

// LerpTo returns nil.
//
// Flutter: painting/decoration.dart Decoration.lerpTo
func (DecorationDefaults) LerpTo(Decoration, float64) Decoration { return nil }

// HitTest returns true.
//
// Flutter: painting/decoration.dart Decoration.hitTest
func (DecorationDefaults) HitTest(ui.Size, ui.Offset, ui.TextDirection) bool { return true }

// GetClipPath panics with the tag DecorationNoClipPath.
//
// Flutter: painting/decoration.dart Decoration.getClipPath
func (DecorationDefaults) GetClipPath(ui.Rect, ui.TextDirection) *ui.Path {
	panic(errors.New("DecorationNoClipPath", "this Decoration subclass does not expect to be used for clipping"))
}

// DebugFillProperties adds nothing.
//
// Flutter: painting/decoration.dart Decoration.debugFillProperties
func (DecorationDefaults) DebugFillProperties(*foundation.DiagnosticPropertiesBuilder) {}

// DecorationLerp linearly interpolates between two decorations. Either may be
// nil.
//
// Flutter: painting/decoration.dart Decoration.lerp
func DecorationLerp(a, b Decoration, t float64) Decoration {
	if a == nil && b == nil {
		return nil
	}
	if a != nil && b != nil && decorationIdentical(a, b) {
		return a
	}
	if a == nil {
		if r := b.LerpFrom(nil, t); r != nil {
			return r
		}
		return b
	}
	if b == nil {
		if r := a.LerpTo(nil, t); r != nil {
			return r
		}
		return a
	}
	if t == 0.0 {
		return a
	}
	if t == 1.0 {
		return b
	}
	if r := b.LerpFrom(a, t); r != nil {
		return r
	}
	if r := a.LerpTo(b, t); r != nil {
		return r
	}
	if t < 0.5 {
		if r := a.LerpTo(nil, t*2.0); r != nil {
			return r
		}
		return a
	}
	if r := b.LerpFrom(nil, (t-0.5)*2.0); r != nil {
		return r
	}
	return b
}

func decorationIdentical(a, b Decoration) bool { return sameValue(a, b) }

// BoxPainter is a stateful class that can paint a particular Decoration.
//
// Flutter: painting/decoration.dart BoxPainter
type BoxPainter interface {
	// Paint paints the decoration on the canvas at the offset; the
	// configuration says how big the box is.
	Paint(canvas *ui.Canvas, offset ui.Offset, configuration ImageConfiguration)
	// Dispose discards any resources used by the painter.
	Dispose()
}

// ImageConfiguration configures an image provider to the situation in which the
// image is used: the device pixel ratio, locale, size and so on.
//
// Flutter: painting/image_provider.dart ImageConfiguration
type ImageConfiguration struct {
	// DevicePixelRatio is the device pixel ratio where the image will be used.
	DevicePixelRatio foundation.Opt[float64]
	// Locale is the language and region for which to select the image.
	Locale foundation.Opt[ui.Locale]
	// TextDirection is the reading direction, for directional images;
	// ui.TextDirectionUnspecified for none.
	TextDirection ui.TextDirection
	// Size is the size at which the image will be rendered.
	Size foundation.Opt[ui.Size]
	// Platform is the platform the image is for.
	Platform foundation.Opt[foundation.TargetPlatform]
}

// ImageConfigurationEmpty is an image configuration that provides no
// additional information.
//
// Flutter: painting/image_provider.dart ImageConfiguration.empty
var ImageConfigurationEmpty = ImageConfiguration{}

// ImageConfigurationCopy holds the replacements for ImageConfiguration.CopyWith.
//
// Flutter: painting/image_provider.dart ImageConfiguration.copyWith
type ImageConfigurationCopy struct {
	DevicePixelRatio foundation.Opt[float64]
	Locale           foundation.Opt[ui.Locale]
	TextDirection    foundation.Opt[ui.TextDirection]
	Size             foundation.Opt[ui.Size]
	Platform         foundation.Opt[foundation.TargetPlatform]
}

// CopyWith returns a copy with the given fields replaced.
//
// Flutter: painting/image_provider.dart ImageConfiguration.copyWith
func (c ImageConfiguration) CopyWith(r ImageConfigurationCopy) ImageConfiguration {
	out := c
	if r.DevicePixelRatio.IsSome() {
		out.DevicePixelRatio = r.DevicePixelRatio
	}
	if r.Locale.IsSome() {
		out.Locale = r.Locale
	}
	if r.TextDirection.IsSome() {
		out.TextDirection = r.TextDirection.Value()
	}
	if r.Size.IsSome() {
		out.Size = r.Size
	}
	if r.Platform.IsSome() {
		out.Platform = r.Platform
	}
	return out
}

// String describes the configuration.
//
// Flutter: painting/image_provider.dart ImageConfiguration.toString
func (c ImageConfiguration) String() string {
	var parts []string
	if v, ok := c.DevicePixelRatio.Get(); ok {
		parts = append(parts, "devicePixelRatio: "+dartfmt.ToStringAsFixed(v, 1))
	}
	if v, ok := c.Locale.Get(); ok {
		parts = append(parts, "locale: "+v.String())
	}
	if c.TextDirection != ui.TextDirectionUnspecified {
		parts = append(parts, "textDirection: "+c.TextDirection.String())
	}
	if v, ok := c.Size.Get(); ok {
		parts = append(parts, "size: "+v.String())
	}
	if v, ok := c.Platform.Get(); ok {
		parts = append(parts, "platform: "+v.Name())
	}
	return "ImageConfiguration(" + strings.Join(parts, ", ") + ")"
}
