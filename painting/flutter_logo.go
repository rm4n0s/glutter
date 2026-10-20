// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package painting

import (
	"math"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// FlutterLogoStyle is an immutable style description of how to paint the
// Flutter logo. The zero value is FlutterLogoStyleUnspecified: markOnly.
//
// Flutter: painting/flutter_logo.dart FlutterLogoStyle
type FlutterLogoStyle int

// The values of FlutterLogoStyle follow Flutter's declaration order.
const (
	// FlutterLogoStyleUnspecified is the zero value: the default (mark only).
	FlutterLogoStyleUnspecified FlutterLogoStyle = iota
	// FlutterLogoStyleMarkOnly is only the Flutter logo, no text.
	//
	// Flutter: painting/flutter_logo.dart FlutterLogoStyle.markOnly
	FlutterLogoStyleMarkOnly
	// FlutterLogoStyleHorizontal is the mark and the text next to it.
	//
	// Flutter: painting/flutter_logo.dart FlutterLogoStyle.horizontal
	FlutterLogoStyleHorizontal
	// FlutterLogoStyleStacked is the mark with the text below it.
	//
	// Flutter: painting/flutter_logo.dart FlutterLogoStyle.stacked
	FlutterLogoStyleStacked
)

// Name returns the Dart enum name, such as "markOnly".
//
// Flutter: painting/flutter_logo.dart FlutterLogoStyle.name
func (s FlutterLogoStyle) Name() string {
	switch s {
	case FlutterLogoStyleHorizontal:
		return "horizontal"
	case FlutterLogoStyleStacked:
		return "stacked"
	case FlutterLogoStyleMarkOnly:
		return "markOnly"
	}
	return "unspecified"
}

// String returns the name Dart prints, such as "FlutterLogoStyle.stacked".
//
// Flutter: painting/flutter_logo.dart FlutterLogoStyle.toString
func (s FlutterLogoStyle) String() string { return "FlutterLogoStyle." + s.Name() }

// FlutterLogoDecoration is an immutable description of the Flutter logo.
//
// Flutter: painting/flutter_logo.dart FlutterLogoDecoration
type FlutterLogoDecoration struct {
	DecorationDefaults
	// TextColor is the color used to paint the "Flutter" text.
	TextColor ui.Color
	// Style is what that logo should look like.
	Style FlutterLogoStyle
	// Margin is how far to inset the logo from the edge of the container.
	Margin EdgeInsets
	// TextEngine makes the paragraph of the "Flutter" text. Deviation: Dart
	// reaches a global font collection; here the engine is a parameter, and
	// with none the text is not painted.
	TextEngine ui.ParagraphFactory

	position float64
	opacity  float64
}

// FlutterLogoDecorationOpts are the optional parameters of
// NewFlutterLogoDecoration.
//
// Flutter: painting/flutter_logo.dart FlutterLogoDecoration.new
type FlutterLogoDecorationOpts struct {
	// TextColor defaults to 0xFF757575.
	TextColor foundation.Opt[ui.Color]
	// Style defaults to FlutterLogoStyleMarkOnly.
	Style FlutterLogoStyle
	// Margin defaults to zero.
	Margin EdgeInsets
	// TextEngine makes the paragraph of the "Flutter" text; with none the text
	// is not painted.
	TextEngine ui.ParagraphFactory
}

// NewFlutterLogoDecoration creates a decoration that paints the Flutter logo.
//
// Flutter: painting/flutter_logo.dart FlutterLogoDecoration.new
func NewFlutterLogoDecoration(opts ...FlutterLogoDecorationOpts) *FlutterLogoDecoration {
	o := firstOpts(opts)
	d := &FlutterLogoDecoration{
		TextColor: o.TextColor.OrElse(ui.NewColor(0xFF757575)), Style: o.Style, Margin: o.Margin, TextEngine: o.TextEngine, opacity: 1.0,
	}
	switch d.Style {
	case FlutterLogoStyleHorizontal:
		d.position = 1.0
	case FlutterLogoStyleStacked:
		d.position = -1.0
	default:
		d.Style = FlutterLogoStyleMarkOnly
		d.position = 0.0
	}
	return d
}

func (d *FlutterLogoDecoration) inTransition() bool {
	return d.opacity != 1.0 || (d.position != -1.0 && d.position != 0.0 && d.position != 1.0)
}

// DebugAssertIsValid panics with the tag FlutterLogoInvalid, in debug builds,
// if the position isn't finite or the opacity is out of range.
//
// Flutter: painting/flutter_logo.dart FlutterLogoDecoration.debugAssertIsValid
func (d *FlutterLogoDecoration) DebugAssertIsValid() bool {
	if foundation.DebugMode && !(!math.IsInf(d.position, 0) && !math.IsNaN(d.position) && d.opacity >= 0.0 && d.opacity <= 1.0) {
		panic(errors.New("FlutterLogoInvalid", "the position of a FlutterLogoDecoration must be finite and its opacity from 0 to 1",
			"position", d.position, "opacity", d.opacity))
	}
	return true
}

// IsComplex is whether the logo is not in a transition.
//
// Flutter: painting/flutter_logo.dart FlutterLogoDecoration.isComplex
func (d *FlutterLogoDecoration) IsComplex() bool { return !d.inTransition() }

// FlutterLogoDecorationLerp linearly interpolates between two Flutter logo
// decorations. Either may be nil.
//
// Flutter: painting/flutter_logo.dart FlutterLogoDecoration.lerp
func FlutterLogoDecorationLerp(a, b *FlutterLogoDecoration, t float64) *FlutterLogoDecoration {
	if a != nil {
		a.DebugAssertIsValid()
	}
	if b != nil {
		b.DebugAssertIsValid()
	}
	if a == b {
		return a
	}
	if a == nil {
		return &FlutterLogoDecoration{
			TextColor: b.TextColor, Style: b.Style, Margin: b.Margin.Mul(t), TextEngine: b.TextEngine,
			position: b.position, opacity: b.opacity * ui.ClampDouble(t, 0.0, 1.0),
		}
	}
	if b == nil {
		return &FlutterLogoDecoration{
			TextColor: a.TextColor, Style: a.Style, Margin: a.Margin.Mul(t), TextEngine: a.TextEngine,
			position: a.position, opacity: a.opacity * ui.ClampDouble(1.0-t, 0.0, 1.0),
		}
	}
	if t == 0.0 {
		return a
	}
	if t == 1.0 {
		return b
	}
	style := b.Style
	if t < 0.5 {
		style = a.Style
	}
	return &FlutterLogoDecoration{
		TextColor: a.TextColor.Lerp(b.TextColor, t), Style: style, Margin: *EdgeInsetsLerp(&a.Margin, &b.Margin, t), TextEngine: b.TextEngine,
		position: a.position + (b.position-a.position)*t, opacity: ui.ClampDouble(a.opacity+(b.opacity-a.opacity)*t, 0.0, 1.0),
	}
}

// LerpFrom interpolates from a FlutterLogoDecoration or nil.
//
// Flutter: painting/flutter_logo.dart FlutterLogoDecoration.lerpFrom
func (d *FlutterLogoDecoration) LerpFrom(a Decoration, t float64) Decoration {
	d.DebugAssertIsValid()
	switch x := a.(type) {
	case nil:
		return FlutterLogoDecorationLerp(nil, d, t)
	case *FlutterLogoDecoration:
		return FlutterLogoDecorationLerp(x, d, t)
	}
	return nil
}

// LerpTo interpolates to a FlutterLogoDecoration or nil.
//
// Flutter: painting/flutter_logo.dart FlutterLogoDecoration.lerpTo
func (d *FlutterLogoDecoration) LerpTo(b Decoration, t float64) Decoration {
	d.DebugAssertIsValid()
	switch x := b.(type) {
	case nil:
		return FlutterLogoDecorationLerp(d, nil, t)
	case *FlutterLogoDecoration:
		return FlutterLogoDecorationLerp(d, x, t)
	}
	return nil
}

// HitTest is always true.
//
// Flutter: painting/flutter_logo.dart FlutterLogoDecoration.hitTest
func (d *FlutterLogoDecoration) HitTest(ui.Size, ui.Offset, ui.TextDirection) bool { return true }

// GetClipPath is the rect.
//
// Flutter: painting/flutter_logo.dart FlutterLogoDecoration.getClipPath
func (d *FlutterLogoDecoration) GetClipPath(rect ui.Rect, _ ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRect(rect)
	return p
}

// Equal reports whether the other decoration is equal: same text color,
// position and opacity.
//
// Flutter: painting/flutter_logo.dart FlutterLogoDecoration.operator ==
func (d *FlutterLogoDecoration) Equal(o *FlutterLogoDecoration) bool {
	if d == o {
		return true
	}
	if d == nil || o == nil {
		return false
	}
	d.DebugAssertIsValid()
	return o.TextColor == d.TextColor && o.position == d.position && o.opacity == d.opacity
}

// ToStringShort names the decoration.
//
// Flutter: painting/decoration.dart Decoration.toStringShort
func (d *FlutterLogoDecoration) ToStringShort() string { return "FlutterLogoDecoration" }

// String describes the decoration and its properties.
//
// Flutter: painting/flutter_logo.dart FlutterLogoDecoration.toString
func (d *FlutterLogoDecoration) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the text color, the style and the transition.
//
// Flutter: painting/flutter_logo.dart FlutterLogoDecoration.debugFillProperties
func (d *FlutterLogoDecoration) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(NewColorProperty("textColor", foundation.Some(d.TextColor)))
	p.Add(foundation.NewEnumProperty("style", foundation.Some(d.Style)))
	if d.inTransition() {
		p.Add(foundation.NewDiagnosticsNodeMessage("transition " + foundation.DebugFormatDouble(foundation.Some(d.position)) + ":" +
			foundation.DebugFormatDouble(foundation.Some(d.opacity))))
	}
}

// CreateBoxPainter returns a painter for the logo.
//
// Flutter: painting/flutter_logo.dart FlutterLogoDecoration.createBoxPainter
func (d *FlutterLogoDecoration) CreateBoxPainter(func()) BoxPainter {
	d.DebugAssertIsValid()
	return newFlutterLogoPainter(d)
}

const flutterLogoLabel = "Flutter"

type flutterLogoPainter struct {
	config *FlutterLogoDecoration

	text             *ui.Paragraph
	textBoundingRect ui.Rect
}

func newFlutterLogoPainter(config *FlutterLogoDecoration) *flutterLogoPainter {
	p := &flutterLogoPainter{config: config}
	p.prepareText()
	return p
}

func (p *flutterLogoPainter) prepareText() {
	if p.config.TextEngine == nil {
		return
	}
	size := 100.0 * 350.0 / 247.0
	b := ui.NewParagraphBuilder(p.config.TextEngine, ui.NewParagraphStyle(ui.ParagraphStyleOpts{TextDirection: ui.TextDirectionLTR}))
	color := p.config.TextColor
	b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{
		Color: &color, FontFamily: "Roboto", FontSize: &size, FontWeight: ui.FontWeightW300, TextBaseline: ui.TextBaselineAlphabetic,
	}))
	b.AddText(flutterLogoLabel)
	p.text = b.Build()
	p.text.Layout(ui.NewParagraphConstraints(math.Inf(1)))
	boxes := p.text.GetBoxesForRange(0, len(flutterLogoLabel))
	if len(boxes) == 1 {
		p.textBoundingRect = ui.NewRectFromLTRB(boxes[0].Left, boxes[0].Top, boxes[0].Right, boxes[0].Bottom)
	}
}

// Dispose releases the paragraph (Go's garbage collector does the work).
//
// Flutter: painting/flutter_logo.dart _FlutterLogoPainter.dispose
func (p *flutterLogoPainter) Dispose() { p.text = nil }

func paintFlutterLogoMark(canvas *ui.Canvas, rect ui.Rect) {
	canvas.Save()
	canvas.Translate(rect.Left, rect.Top)
	canvas.Scale(rect.Width()/202.0, rect.Height()/202.0)
	canvas.Translate((202.0-166.0)/2.0, 0.0)

	lightPaint := ui.NewPaint()
	lightPaint.SetColor(ui.NewColor(0xFF54C5F8))
	mediumPaint := ui.NewPaint()
	mediumPaint.SetColor(ui.NewColor(0xFF29B6F6))
	darkPaint := ui.NewPaint()
	darkPaint.SetColor(ui.NewColor(0xFF01579B))
	triangleGradient := ui.NewGradientLinear(
		ui.Offset{Dx: 87.2623 + 37.9092, Dy: 28.8384 + 123.4389},
		ui.Offset{Dx: 42.9205 + 37.9092, Dy: 35.0952 + 123.4389},
		[]ui.Color{ui.NewColor(0x001A237E), ui.NewColor(0x661A237E)})
	trianglePaint := ui.NewPaint()
	trianglePaint.SetShader(triangleGradient)

	topBeam := ui.NewPath()
	topBeam.MoveTo(37.7, 128.9)
	topBeam.LineTo(9.8, 101.0)
	topBeam.LineTo(100.4, 10.4)
	topBeam.LineTo(156.2, 10.4)
	canvas.DrawPath(topBeam, lightPaint)

	middleBeam := ui.NewPath()
	middleBeam.MoveTo(156.2, 94.0)
	middleBeam.LineTo(100.4, 94.0)
	middleBeam.LineTo(78.5, 115.9)
	middleBeam.LineTo(106.4, 143.8)
	canvas.DrawPath(middleBeam, lightPaint)

	bottomBeam := ui.NewPath()
	bottomBeam.MoveTo(79.5, 170.7)
	bottomBeam.LineTo(100.4, 191.6)
	bottomBeam.LineTo(156.2, 191.6)
	bottomBeam.LineTo(107.4, 142.8)
	canvas.DrawPath(bottomBeam, darkPaint)

	canvas.Save()
	canvas.Transform(ui.NewMatrix4FromList([16]float32{
		0.7071, -0.7071, 0.0, 0.0,
		0.7071, 0.7071, 0.0, 0.0,
		0.0, 0.0, 1.0, 0.0,
		-77.697, 98.057, 0.0, 1.0,
	}))
	canvas.DrawRect(ui.NewRectFromLTWH(59.8, 123.1, 39.4, 39.4), mediumPaint)
	canvas.Restore()

	triangle := ui.NewPath()
	triangle.MoveTo(79.5, 170.7)
	triangle.LineTo(120.9, 156.4)
	triangle.LineTo(107.4, 142.8)
	canvas.DrawPath(triangle, trianglePaint)
	canvas.Restore()
}

// Paint paints the logo. In debug builds it panics with the tag
// FlutterLogoNoSize if the configuration has no size.
//
// Flutter: painting/flutter_logo.dart _FlutterLogoPainter.paint
func (p *flutterLogoPainter) Paint(canvas *ui.Canvas, offset ui.Offset, configuration ImageConfiguration) {
	cfg := p.config
	size, ok := configuration.Size.Get()
	if !ok {
		if foundation.DebugMode {
			panic(errors.New("FlutterLogoNoSize", "the ImageConfiguration given to a FlutterLogoDecoration painter has no size"))
		}
		return
	}
	offset = offset.Add(cfg.Margin.TopLeft())
	canvasSize := cfg.Margin.DeflateSize(size)
	if canvasSize.IsEmpty() {
		return
	}
	var logoSize ui.Size
	switch {
	case cfg.position > 0.0:
		logoSize = ui.Size{Width: 820.0, Height: 232.0}
	case cfg.position < 0.0:
		logoSize = ui.Size{Width: 252.0, Height: 306.0}
	default:
		logoSize = ui.Size{Width: 202.0, Height: 202.0}
	}
	fitted := ApplyBoxFit(BoxFitContain, logoSize, canvasSize)
	rect := AlignmentCenter.Inscribe(fitted.Destination, offset.And(canvasSize))
	centerSquareHeight := canvasSize.ShortestSide()
	centerSquare := ui.NewRectFromLTWH(
		offset.Dx+(canvasSize.Width-centerSquareHeight)/2.0, offset.Dy+(canvasSize.Height-centerSquareHeight)/2.0,
		centerSquareHeight, centerSquareHeight)
	var logoTargetSquare ui.Rect
	switch {
	case cfg.position > 0.0:
		logoTargetSquare = ui.NewRectFromLTWH(rect.Left, rect.Top, rect.Height(), rect.Height())
	case cfg.position < 0.0:
		logoHeight := rect.Height() * 191.0 / 306.0
		logoTargetSquare = ui.NewRectFromLTWH(rect.Left+(rect.Width()-logoHeight)/2.0, rect.Top, logoHeight, logoHeight)
	default:
		logoTargetSquare = centerSquare
	}
	logoSquare := centerSquare.Lerp(logoTargetSquare, math.Abs(cfg.position))

	if cfg.opacity < 1.0 {
		layerPaint := ui.NewPaint()
		layerPaint.SetColorFilter(ui.NewColorFilterMode(ui.NewColor(0xFFFFFFFF).WithOpacity(cfg.opacity), ui.BlendModeModulate))
		bounds := offset.And(canvasSize)
		canvas.SaveLayer(&bounds, layerPaint)
	}
	if cfg.position != 0.0 && p.text != nil {
		if cfg.position > 0.0 {
			fontSize := 2.0 / 3.0 * logoSquare.Height() * (1 - (10.4*2.0)/202.0)
			scale := fontSize / 100.0
			finalLeftTextPosition := (256.4/820.0)*rect.Width() - (32.0/350.0)*fontSize
			initialLeftTextPosition := rect.Width()/2.0 - p.textBoundingRect.Width()*scale
			textOffset := ui.Offset{
				Dx: rect.Left + ui.LerpDouble(initialLeftTextPosition, finalLeftTextPosition, cfg.position),
				Dy: rect.Top + (rect.Height()-p.textBoundingRect.Height()*scale)/2.0,
			}
			canvas.Save()
			if cfg.position < 1.0 {
				center := logoSquare.Center()
				path := ui.NewPath()
				path.MoveTo(center.Dx, center.Dy)
				path.LineTo(center.Dx+rect.Width(), center.Dy-rect.Width())
				path.LineTo(center.Dx+rect.Width(), center.Dy+rect.Width())
				path.Close()
				canvas.ClipPath(path)
			}
			canvas.Translate(textOffset.Dx, textOffset.Dy)
			canvas.Scale(scale, scale)
			canvas.DrawParagraph(p.text, ui.Offset{})
			canvas.Restore()
		} else {
			fontSize := 0.35 * logoTargetSquare.Height() * (1 - (10.4*2.0)/202.0)
			scale := fontSize / 100.0
			if cfg.position > -1.0 {
				bounds := p.textBoundingRect
				canvas.SaveLayer(&bounds, ui.NewPaint())
			} else {
				canvas.Save()
			}
			canvas.Translate(logoTargetSquare.Center().Dx-(p.textBoundingRect.Width()*scale/2.0), logoTargetSquare.Bottom)
			canvas.Scale(scale, scale)
			canvas.DrawParagraph(p.text, ui.Offset{})
			if cfg.position > -1.0 {
				fade := ui.NewPaint()
				fade.SetBlendMode(ui.BlendModeModulate)
				w := p.textBoundingRect.Width()
				fade.SetShader(ui.NewGradientLinear(ui.Offset{Dx: w * -0.5}, ui.Offset{Dx: w * 1.5},
					[]ui.Color{ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFFFFFFFF), ui.NewColor(0x00FFFFFF), ui.NewColor(0x00FFFFFF)},
					ui.GradientLinearOpts{ColorStops: []float64{
						0.0, math.Max(0.0, math.Abs(cfg.position)-0.1), math.Min(math.Abs(cfg.position)+0.1, 1.0), 1.0,
					}}))
				canvas.DrawRect(p.textBoundingRect.Inflate(p.textBoundingRect.Width()*0.5), fade)
			}
			canvas.Restore()
		}
	}
	paintFlutterLogoMark(canvas, logoSquare)
	if cfg.opacity < 1.0 {
		canvas.Restore()
	}
}

var _ Decoration = (*FlutterLogoDecoration)(nil)
