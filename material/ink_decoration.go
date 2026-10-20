// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// InkOpts are the optional parameters of NewInk.
//
// Flutter: material/ink_decoration.dart Ink.new
type InkOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Padding is the amount of space by which to inset the child inside the
	// decoration.
	Padding painting.EdgeInsetsGeometry
	// Color is the shorthand for a BoxDecoration with this color; it must not be
	// given with Decoration.
	Color foundation.Opt[ui.Color]
	// Decoration is the painted decoration.
	Decoration painting.Decoration
	// Width, if given, makes the ink that wide.
	Width foundation.Opt[float64]
	// Height, if given, makes the ink that high.
	Height foundation.Opt[float64]
}

// Ink is a convenience widget for drawing images and other decorations on
// Material widgets, so that InkWell and InkResponse splashes will render over
// them.
//
// Flutter: material/ink_decoration.dart Ink
type Ink struct {
	widgets.StatefulWidget
	child widgets.Widget
	o     InkOpts
}

// NewInk creates a widget that paints the decoration on the closest Material.
// In debug builds it panics with the tag InkColorAndDecoration when both a color
// and a decoration are given, and with InkPaddingNegative for a negative padding.
//
// Flutter: material/ink_decoration.dart Ink.new
func NewInk(child widgets.Widget, opts ...InkOpts) *Ink {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if o.Padding != nil && !o.Padding.IsNonNegative() {
			panic(errors.New("InkPaddingNegative", "the padding of an Ink must not be negative"))
		}
		if o.Decoration != nil {
			o.Decoration.DebugAssertIsValid()
		}
		if o.Color.IsSome() && o.Decoration != nil {
			panic(errors.New("InkColorAndDecoration", "Cannot provide both a color and a decoration\nThe color argument is just a shorthand for \"decoration: BoxDecoration(color: color)\"."))
		}
	}
	if o.Decoration == nil && o.Color.IsSome() {
		o.Decoration = painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: o.Color})
		o.Color = foundation.None[ui.Color]()
	}
	return &Ink{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), child, o}
}

// InkImageOpts are the optional parameters of NewInkImage.
//
// Flutter: material/ink_decoration.dart Ink.image
type InkImageOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Padding is the amount of space by which to inset the child inside the
	// decoration.
	Padding painting.EdgeInsetsGeometry
	// OnImageError is called when the image fails to load.
	OnImageError painting.ImageErrorListener
	// ColorFilter is applied to the image before painting it.
	ColorFilter *ui.ColorFilter
	// Fit is how the image is inscribed into the box.
	Fit painting.BoxFit
	// Alignment defaults to center.
	Alignment painting.AlignmentGeometry
	// CenterSlice is the center slice of a nine-patch image.
	CenterSlice foundation.Opt[ui.Rect]
	// Repeat defaults to noRepeat.
	Repeat painting.ImageRepeat
	// MatchTextDirection says whether to paint the image in the direction of the
	// text.
	MatchTextDirection bool
	// Width, if given, makes the ink that wide.
	Width foundation.Opt[float64]
	// Height, if given, makes the ink that high.
	Height foundation.Opt[float64]
}

// NewInkImage creates a widget that paints an image on the closest Material.
//
// Flutter: material/ink_decoration.dart Ink.image
func NewInkImage(image painting.ImageProvider, child widgets.Widget, opts ...InkImageOpts) *Ink {
	o := firstOpts(opts)
	if foundation.DebugMode && o.Padding != nil && !o.Padding.IsNonNegative() {
		panic(errors.New("InkPaddingNegative", "the padding of an Ink must not be negative"))
	}
	return NewInk(child, InkOpts{Key: o.Key, Padding: o.Padding, Width: o.Width, Height: o.Height,
		Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{Image: painting.NewDecorationImage(image, painting.DecorationImageOpts{
			OnError: o.OnImageError, ColorFilter: o.ColorFilter, Fit: o.Fit, Alignment: o.Alignment, CenterSlice: o.CenterSlice,
			Repeat: o.Repeat, MatchTextDirection: o.MatchTextDirection})})})
}

// Child is the widget below this widget in the tree.
//
// Flutter: material/ink_decoration.dart Ink.child
func (w *Ink) Child() widgets.Widget { return w.child }

// Padding is the amount of space by which to inset the child inside the
// decoration, or nil.
//
// Flutter: material/ink_decoration.dart Ink.padding
func (w *Ink) Padding() painting.EdgeInsetsGeometry { return w.o.Padding }

// Decoration is the decoration to paint on the Material, or nil.
//
// Flutter: material/ink_decoration.dart Ink.decoration
func (w *Ink) Decoration() painting.Decoration { return w.o.Decoration }

// Width is the width of the ink, if given.
//
// Flutter: material/ink_decoration.dart Ink.width
func (w *Ink) Width() foundation.Opt[float64] { return w.o.Width }

// Height is the height of the ink, if given.
//
// Flutter: material/ink_decoration.dart Ink.height
func (w *Ink) Height() foundation.Opt[float64] { return w.o.Height }

func (w *Ink) paddingIncludingDecoration() painting.EdgeInsetsGeometry {
	var decorationPadding painting.EdgeInsetsGeometry
	if w.o.Decoration != nil {
		decorationPadding = w.o.Decoration.Padding()
	}
	switch {
	case w.o.Padding == nil && decorationPadding == nil:
		return painting.EdgeInsetsZero
	case w.o.Padding == nil:
		return decorationPadding
	case decorationPadding == nil:
		return w.o.Padding
	}
	return w.o.Padding.Add(decorationPadding)
}

// DebugFillProperties adds the padding and the decoration.
//
// Flutter: material/ink_decoration.dart Ink.debugFillProperties
func (w *Ink) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsPropertyOpt("padding", optEdgeInsets(w.o.Padding), foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
	var deco foundation.Opt[painting.Decoration]
	if w.o.Decoration != nil {
		deco = foundation.Some(w.o.Decoration)
	}
	p.Add(foundation.NewDiagnosticsPropertyOpt("bg", deco, foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
}

func optEdgeInsets(e painting.EdgeInsetsGeometry) foundation.Opt[painting.EdgeInsetsGeometry] {
	if e == nil {
		return foundation.None[painting.EdgeInsetsGeometry]()
	}
	return foundation.Some(e)
}

// CreateState creates the state.
//
// Flutter: material/ink_decoration.dart Ink.createState
func (w *Ink) CreateState() widgets.State {
	return &inkState{boxKey: widgets.NewGlobalKey[widgets.State]()}
}

type inkState struct {
	widgets.StateOf[*Ink]
	boxKey *widgets.GlobalKey[widgets.State]
	ink    *InkDecoration
}

func (s *inkState) handleRemoved() { s.ink = nil }

func (s *inkState) Deactivate() {
	if s.ink != nil {
		s.ink.Dispose()
	}
}

func (s *inkState) build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	// By creating the InkDecoration from within a Builder widget, we can use the
	// RenderBox of the Padding widget.
	if s.ink == nil {
		s.ink = NewInkDecoration(w.o.Decoration, MaterialOf(bc), s.boxKey.CurrentContext().FindRenderObject().(rendering.Box),
			widgets.CreateLocalImageConfiguration(bc, foundation.None[ui.Size]()), InkDecorationOpts{
				IsVisible: foundation.Some(widgets.VisibilityOf(bc)), OnRemoved: s.handleRemoved})
	} else {
		s.ink.SetDecoration(w.o.Decoration)
		s.ink.SetVisible(widgets.VisibilityOf(bc))
		s.ink.SetConfiguration(widgets.CreateLocalImageConfiguration(bc, foundation.None[ui.Size]()))
	}
	if w.child != nil {
		return w.child
	}
	return widgets.NewConstrainedBox(rendering.NewBoxConstraintsExpand(), nil)
}

func (s *inkState) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterial(bc)
	w := s.Widget()
	var result widgets.Widget = widgets.NewPadding(w.paddingIncludingDecoration(), widgets.NewBuilder(s.build), widgets.PaddingOpts{Key: s.boxKey})
	if w.o.Width.IsSome() || w.o.Height.IsSome() {
		result = widgets.NewSizedBox(result, widgets.SizedBoxOpts{Width: w.o.Width, Height: w.o.Height})
	}
	return result
}

// InkDecoration is an ink feature that paints a Decoration on a Material.
//
// Flutter: material/ink_decoration.dart InkDecoration
type InkDecoration struct {
	InkFeatureBase
	painter       painting.BoxPainter
	decoration    painting.Decoration
	isVisible     bool
	configuration painting.ImageConfiguration
}

// InkDecorationOpts are the optional parameters of NewInkDecoration.
//
// Flutter: material/ink_decoration.dart InkDecoration.new
type InkDecorationOpts struct {
	// IsVisible is whether the decoration is painted; the default is true.
	IsVisible foundation.Opt[bool]
	// OnRemoved is called when the feature is disposed.
	OnRemoved func()
}

// NewInkDecoration begins a decoration on the material of the controller and
// adds it to it.
//
// Flutter: material/ink_decoration.dart InkDecoration.new
func NewInkDecoration(decoration painting.Decoration, controller MaterialInkController, referenceBox rendering.Box, configuration painting.ImageConfiguration, opts ...InkDecorationOpts) *InkDecoration {
	o := firstOpts(opts)
	d := &InkDecoration{configuration: configuration, isVisible: true}
	InitInkFeature(&d.InkFeatureBase, d, controller, referenceBox, InkFeatureOpts{OnRemoved: o.OnRemoved})
	d.SetDecoration(decoration)
	d.SetVisible(o.IsVisible.OrElse(true))
	controller.AddInkFeature(d)
	return d
}

// Decoration is the decoration to paint on the Material.
//
// Flutter: material/ink_decoration.dart InkDecoration.decoration
func (d *InkDecoration) Decoration() painting.Decoration { return d.decoration }

// SetDecoration changes the decoration and repaints when it changed.
//
// Flutter: material/ink_decoration.dart InkDecoration.decoration
func (d *InkDecoration) SetDecoration(value painting.Decoration) {
	if foundation.ValuesEqual(value, d.decoration) {
		return
	}
	d.decoration = value
	if d.painter != nil {
		d.painter.Dispose()
		d.painter = nil
	}
	if value != nil {
		d.painter = value.CreateBoxPainter(d.controller.MarkNeedsPaint)
	}
	d.controller.MarkNeedsPaint()
}

// IsVisible is whether the decoration is painted.
//
// Flutter: material/ink_decoration.dart InkDecoration.isVisible
func (d *InkDecoration) IsVisible() bool { return d.isVisible }

// SetVisible changes the visibility and repaints when it changed.
//
// Flutter: material/ink_decoration.dart InkDecoration.isVisible
func (d *InkDecoration) SetVisible(value bool) {
	if value == d.isVisible {
		return
	}
	d.isVisible = value
	d.controller.MarkNeedsPaint()
}

// Configuration is the settings to pass to the decoration when painting, so that
// it can resolve images appropriately.
//
// Flutter: material/ink_decoration.dart InkDecoration.configuration
func (d *InkDecoration) Configuration() painting.ImageConfiguration { return d.configuration }

// SetConfiguration changes the configuration and repaints when it changed.
//
// Flutter: material/ink_decoration.dart InkDecoration.configuration
func (d *InkDecoration) SetConfiguration(value painting.ImageConfiguration) {
	if value == d.configuration {
		return
	}
	d.configuration = value
	d.controller.MarkNeedsPaint()
}

// Dispose disposes the painter and removes the feature.
//
// Flutter: material/ink_decoration.dart InkDecoration.dispose
func (d *InkDecoration) Dispose() {
	if d.painter != nil {
		d.painter.Dispose()
	}
	d.InkFeatureBase.Dispose()
}

// PaintFeature paints the decoration.
func (d *InkDecoration) PaintFeature(canvas *ui.Canvas, transform *ui.Matrix4) {
	if d.painter == nil || !d.isVisible {
		return
	}
	originOffset, isTranslation := transform.AsTranslation()
	sizedConfiguration := d.configuration.CopyWith(painting.ImageConfigurationCopy{Size: foundation.Some(d.referenceBox.Size())})
	if !isTranslation {
		canvas.Save()
		canvas.Transform(transform)
		d.painter.Paint(canvas, ui.OffsetZero, sizedConfiguration)
		canvas.Restore()
	} else {
		d.painter.Paint(canvas, originOffset, sizedConfiguration)
	}
}
