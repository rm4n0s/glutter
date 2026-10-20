// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package painting

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
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
