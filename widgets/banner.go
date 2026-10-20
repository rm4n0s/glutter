// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

const (
	bannerOffset       = 40.0 // distance to the bottom of the banner, at a 45 degree angle inwards
	bannerHeight       = 12.0 // height of the banner
	bannerBottomOffset = bannerOffset + math.Sqrt2/2*bannerHeight
)

var (
	bannerRect   = ui.NewRectFromLTWH(-bannerOffset, bannerOffset-bannerHeight, bannerOffset*2.0, bannerHeight)
	bannerShadow = painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(ui.NewColor(0x7F000000)), BlurRadius: 6.0})
	bannerColor  = ui.NewColor(0xA0B71C1C)
)

func bannerTextStyle() *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{
		Color:      foundation.Some(ui.NewColor(0xFFFFFFFF)),
		FontSize:   foundation.Some(bannerHeight * 0.85),
		FontWeight: ui.FontWeightW900,
		Height:     foundation.Some(1.0),
	})
}

// BannerLocation is where to show a Banner.
//
// Flutter: widgets/banner.dart BannerLocation
type BannerLocation int

// The values of BannerLocation follow Flutter's declaration order.
const (
	// BannerLocationUnspecified is the zero value: no location.
	//
	// Flutter: no counterpart
	BannerLocationUnspecified BannerLocation = iota
	// BannerLocationTopStart is the top corner on the start side of the reading
	// direction.
	//
	// Flutter: widgets/banner.dart BannerLocation.topStart
	BannerLocationTopStart
	// BannerLocationTopEnd is the top corner on the end side.
	//
	// Flutter: widgets/banner.dart BannerLocation.topEnd
	BannerLocationTopEnd
	// BannerLocationBottomStart is the bottom corner on the start side.
	//
	// Flutter: widgets/banner.dart BannerLocation.bottomStart
	BannerLocationBottomStart
	// BannerLocationBottomEnd is the bottom corner on the end side.
	//
	// Flutter: widgets/banner.dart BannerLocation.bottomEnd
	BannerLocationBottomEnd
)

// Name is the name of the location, as Dart's Enum.name.
//
// Flutter: widgets/banner.dart BannerLocation
func (l BannerLocation) Name() string {
	switch l {
	case BannerLocationTopStart:
		return "topStart"
	case BannerLocationTopEnd:
		return "topEnd"
	case BannerLocationBottomStart:
		return "bottomStart"
	case BannerLocationBottomEnd:
		return "bottomEnd"
	}
	return "unspecified"
}

// BannerPainterOpts are the optional parameters of NewBannerPainter.
//
// Flutter: widgets/banner.dart BannerPainter.new
type BannerPainterOpts struct {
	// Color is the color of the banner; the default is a translucent dark red.
	Color foundation.Opt[ui.Color]
	// TextStyle is the style of the message; the default is white, bold and 10.2
	// logical pixels high.
	TextStyle *painting.TextStyle
	// Shadow is the shadow of the banner; the default is black at half opacity
	// with a blur radius of 6.
	Shadow foundation.Opt[painting.BoxShadow]
	// TextEngine lays out the message. Deviation: Flutter reaches the engine
	// through a global; the framework passes the engine of the PlatformDispatcher.
	TextEngine ui.ParagraphFactory
}

// BannerPainter paints a Banner: a diagonal ribbon with a message across a
// corner.
//
// Flutter: widgets/banner.dart BannerPainter
type BannerPainter struct {
	rendering.CustomPainterBase
	message         string
	textDirection   ui.TextDirection
	location        BannerLocation
	layoutDirection ui.TextDirection
	color           ui.Color
	textStyle       *painting.TextStyle
	shadow          painting.BoxShadow
	engine          ui.ParagraphFactory

	prepared    bool
	textPainter *painting.TextPainter
	paintShadow *ui.Paint
	paintBanner *ui.Paint
}

// NewBannerPainter creates the painter.
//
// Flutter: widgets/banner.dart BannerPainter.new
func NewBannerPainter(message string, textDirection ui.TextDirection, location BannerLocation, layoutDirection ui.TextDirection, opts ...BannerPainterOpts) *BannerPainter {
	o := firstOpts(opts)
	p := &BannerPainter{
		CustomPainterBase: rendering.NewCustomPainterBase(nil),
		message:           message, textDirection: textDirection, location: location, layoutDirection: layoutDirection,
		color: o.Color.OrElse(bannerColor), textStyle: o.TextStyle, shadow: o.Shadow.OrElse(bannerShadow), engine: o.TextEngine,
	}
	if p.textStyle == nil {
		p.textStyle = bannerTextStyle()
	}
	return p
}

// Dispose releases the text painter.
//
// Flutter: widgets/banner.dart BannerPainter.dispose
func (p *BannerPainter) Dispose() {
	if p.textPainter != nil {
		p.textPainter.Dispose()
		p.textPainter = nil
	}
}

func (p *BannerPainter) prepare() {
	p.paintShadow = p.shadow.ToPaint()
	p.paintBanner = ui.NewPaint()
	p.paintBanner.SetColor(p.color)
	if p.textPainter != nil {
		p.textPainter.Dispose()
	}
	p.textPainter = painting.NewTextPainter(painting.TextPainterOpts{
		TextEngine: p.engine, Text: painting.NewTextSpan(painting.TextSpanOpts{Style: p.textStyle, Text: foundation.Some(p.message)}),
		TextAlign: ui.TextAlignCenter, TextDirection: p.textDirection,
	})
	p.prepared = true
}

// Paint draws the ribbon.
//
// Flutter: widgets/banner.dart BannerPainter.paint
func (p *BannerPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	if !p.prepared {
		p.prepare()
	}
	canvas.Translate(p.translationX(size.Width), p.translationY(size.Height))
	canvas.Rotate(p.rotation())
	canvas.DrawRect(bannerRect, p.paintShadow)
	canvas.DrawRect(bannerRect, p.paintBanner)
	const width = bannerOffset * 2.0
	p.textPainter.Layout(width, width)
	p.textPainter.Paint(canvas, bannerRect.TopLeft().Add(ui.Offset{Dx: 0.0, Dy: (bannerRect.Height() - p.textPainter.Height()) / 2.0}))
}

// ShouldRepaint is whether the message, location, color or text style changed.
//
// Flutter: widgets/banner.dart BannerPainter.shouldRepaint
func (p *BannerPainter) ShouldRepaint(oldDelegate rendering.CustomPainter) bool {
	old, ok := oldDelegate.(*BannerPainter)
	if !ok {
		return true
	}
	return p.message != old.message || p.location != old.location || p.color != old.color || !p.textStyle.Equal(old.textStyle)
}

// HitTest is false: the banner does not take pointer events.
//
// Flutter: widgets/banner.dart BannerPainter.hitTest
func (p *BannerPainter) HitTest(ui.Offset) foundation.Opt[bool] { return foundation.Some(false) }

func (p *BannerPainter) translationX(width float64) float64 {
	rtl := p.layoutDirection == ui.TextDirectionRTL
	switch p.location {
	case BannerLocationTopStart:
		if rtl {
			return width
		}
		return 0.0
	case BannerLocationTopEnd:
		if rtl {
			return 0.0
		}
		return width
	case BannerLocationBottomStart:
		if rtl {
			return width - bannerBottomOffset
		}
		return bannerBottomOffset
	default: // BannerLocationBottomEnd
		if rtl {
			return bannerBottomOffset
		}
		return width - bannerBottomOffset
	}
}

func (p *BannerPainter) translationY(height float64) float64 {
	switch p.location {
	case BannerLocationBottomStart, BannerLocationBottomEnd:
		return height - bannerBottomOffset
	}
	return 0.0
}

func (p *BannerPainter) rotation() float64 {
	rtl := p.layoutDirection == ui.TextDirectionRTL
	var sign float64
	switch p.location {
	case BannerLocationTopStart, BannerLocationBottomEnd:
		sign = -1
		if rtl {
			sign = 1
		}
	default: // BannerLocationBottomStart, BannerLocationTopEnd
		sign = 1
		if rtl {
			sign = -1
		}
	}
	return math.Pi / 4.0 * sign
}

// BannerOpts are the optional parameters of NewBanner.
//
// Flutter: widgets/banner.dart Banner.new
type BannerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// TextDirection is the direction of the message; the default is the one of
	// the context, which then must have a Directionality.
	TextDirection ui.TextDirection
	// LayoutDirection is the direction in which the corners are placed; the
	// default is the one of the context.
	LayoutDirection ui.TextDirection
	// Color is the color of the banner.
	Color foundation.Opt[ui.Color]
	// TextStyle is the style of the message.
	TextStyle *painting.TextStyle
	// Shadow is the shadow of the banner.
	Shadow foundation.Opt[painting.BoxShadow]
}

// Banner displays a diagonal message above the corner of another widget.
//
// Flutter: widgets/banner.dart Banner
type Banner struct {
	StatefulWidget
	child    Widget
	message  string
	location BannerLocation
	o        BannerOpts
}

// NewBanner creates the widget. The child, which may be nil, is the widget that
// the banner is over.
//
// Flutter: widgets/banner.dart Banner.new
func NewBanner(message string, location BannerLocation, child Widget, opts ...BannerOpts) *Banner {
	o := firstOpts(opts)
	return &Banner{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child: child, message: message, location: location, o: o}
}

// Child is the widget that the banner is over.
//
// Flutter: widgets/banner.dart Banner.child
func (w *Banner) Child() Widget { return w.child }

// Message is the text of the banner.
//
// Flutter: widgets/banner.dart Banner.message
func (w *Banner) Message() string { return w.message }

// Location is the corner of the child where the banner is.
//
// Flutter: widgets/banner.dart Banner.location
func (w *Banner) Location() BannerLocation { return w.location }

// CreateState creates the state.
//
// Flutter: widgets/banner.dart Banner.createState
func (w *Banner) CreateState() State { return &bannerState{} }

// DebugFillProperties adds the message, directions, location, color and text style.
//
// Flutter: widgets/banner.dart Banner.debugFillProperties
func (w *Banner) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewStringProperty("message", foundation.Some(w.message), foundation.StringPropertyOpts{ShowName: foundation.Some(false)}))
	if w.o.TextDirection != ui.TextDirectionUnspecified {
		p.Add(foundation.NewEnumProperty("textDirection", foundation.Some(w.o.TextDirection)))
	}
	p.Add(foundation.NewEnumProperty("location", foundation.Some(w.location)))
	if w.o.LayoutDirection != ui.TextDirectionUnspecified {
		p.Add(foundation.NewEnumProperty("layoutDirection", foundation.Some(w.o.LayoutDirection)))
	}
	p.Add(painting.NewColorProperty("color", w.o.Color, foundation.DiagnosticsPropertyOpts{ShowName: foundation.Some(false)}))
	if w.o.TextStyle != nil {
		w.o.TextStyle.DebugFillPropertiesWithPrefix(p, "text ")
	}
}

type bannerState struct {
	StateOf[*Banner]
	painter *BannerPainter
}

func (s *bannerState) Dispose() {
	if s.painter != nil {
		s.painter.Dispose()
	}
}

func (s *bannerState) Build(bc BuildContext) Widget {
	w := s.Widget()
	if foundation.DebugMode && (w.o.TextDirection == ui.TextDirectionUnspecified || w.o.LayoutDirection == ui.TextDirectionUnspecified) {
		DebugCheckHasDirectionality(bc)
	}
	if s.painter != nil {
		s.painter.Dispose()
	}
	textDirection, layoutDirection := w.o.TextDirection, w.o.LayoutDirection
	if textDirection == ui.TextDirectionUnspecified {
		textDirection = DirectionalityOf(bc)
	}
	if layoutDirection == ui.TextDirectionUnspecified {
		layoutDirection = DirectionalityOf(bc)
	}
	var engine ui.ParagraphFactory
	if b := MaybeWidgetsBindingOf(bc); b != nil {
		engine = b.PlatformDispatcher().TextEngine()
	}
	s.painter = NewBannerPainter(w.message, textDirection, w.location, layoutDirection,
		BannerPainterOpts{Color: w.o.Color, TextStyle: w.o.TextStyle, Shadow: w.o.Shadow, TextEngine: engine})
	return NewCustomPaint(w.child, CustomPaintOpts{ForegroundPainter: s.painter})
}

// CheckedModeBanner displays a "DEBUG" banner in the top end corner of its
// child in debug builds; in release and profile builds it shows only the child.
//
// Flutter: widgets/banner.dart CheckedModeBanner
type CheckedModeBanner struct {
	StatelessWidget
	child Widget
}

// NewCheckedModeBanner creates the widget.
//
// Flutter: widgets/banner.dart CheckedModeBanner.new
func NewCheckedModeBanner(child Widget, opts ...WidgetOpts) *CheckedModeBanner {
	return &CheckedModeBanner{StatelessWidget: NewStatelessWidgetBase(opts...), child: child}
}

// Child is the widget that the banner is over.
//
// Flutter: widgets/banner.dart CheckedModeBanner.child
func (w *CheckedModeBanner) Child() Widget { return w.child }

// Build shows the banner in debug builds.
//
// Flutter: widgets/banner.dart CheckedModeBanner.build
func (w *CheckedModeBanner) Build(BuildContext) Widget {
	if foundation.DebugMode {
		return NewBanner("DEBUG", BannerLocationTopEnd, w.child, BannerOpts{TextDirection: ui.TextDirectionLTR})
	}
	return w.child
}

// DebugFillProperties adds the message of the banner, or "disabled".
//
// Flutter: widgets/banner.dart CheckedModeBanner.debugFillProperties
func (w *CheckedModeBanner) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	message := "disabled"
	if foundation.DebugMode {
		message = `"DEBUG"`
	}
	p.Add(foundation.NewDiagnosticsNodeMessage(message))
}
