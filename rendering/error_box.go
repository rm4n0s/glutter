// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

const (
	errorBoxMaxWidth  = 100000.0
	errorBoxMaxHeight = 100000.0
)

// RenderErrorBox is a render object that draws a red box with the text of an
// error, used by the ErrorWidget that replaces a widget that failed to build.
// It uses the low-level text API on purpose, to depend on as little as possible
// of a framework that may be in a bad state.
//
// Flutter: rendering/error.dart RenderErrorBox
type RenderErrorBox struct {
	RenderBox
	message   string
	paragraph *ui.Paragraph

	// Padding is the space around the text, used when the box is large enough.
	//
	// Deviation: Flutter has a mutable static; here it is a field set at
	// construction (SPEC §12.10).
	//
	// Flutter: rendering/error.dart RenderErrorBox.padding
	Padding painting.EdgeInsets
	// MinimumWidth is the width the text needs for the padding to be used.
	//
	// Flutter: rendering/error.dart RenderErrorBox.minimumWidth
	MinimumWidth float64
	// BackgroundColor is the color of the box.
	//
	// Flutter: rendering/error.dart RenderErrorBox.backgroundColor
	BackgroundColor ui.Color
}

// RenderErrorBoxOpts are the optional parameters of NewRenderErrorBox.
//
// Flutter: rendering/error.dart RenderErrorBox.new
type RenderErrorBoxOpts struct {
	// TextEngine makes the paragraph. Without it the box has a background and no
	// text.
	TextEngine ui.ParagraphFactory
}

// NewRenderErrorBox creates the box. In debug builds it is dark red with yellow
// monospace bold text; in release builds it is gray with dark text.
//
// Flutter: rendering/error.dart RenderErrorBox.new
func NewRenderErrorBox(message string, opts ...RenderErrorBoxOpts) *RenderErrorBox {
	o := firstOpts(opts)
	b := &RenderErrorBox{
		message:         message,
		Padding:         painting.EdgeInsets{Left: 64, Top: 96, Right: 64, Bottom: 12},
		MinimumWidth:    200,
		BackgroundColor: ui.NewColor(0xF0C0C0C0),
	}
	InitRenderBox(&b.RenderBox, b)
	textColor, family, size, weight := ui.NewColor(0xFF303030), "sans-serif", 18.0, ui.FontWeightNormal
	if foundation.DebugMode {
		b.BackgroundColor = ui.NewColor(0xF0900000)
		textColor, family, size, weight = ui.NewColor(0xFFFFFF66), "monospace", 14.0, ui.FontWeightBold
	}
	if message != "" && o.TextEngine != nil {
		// A failure here is dropped: the error has been reported already and
		// this box is what is shown for it.
		func() {
			defer func() { _ = recover() }()
			builder := ui.NewParagraphBuilder(o.TextEngine, ui.NewParagraphStyle(ui.ParagraphStyleOpts{
				TextDirection: ui.TextDirectionLTR, TextAlign: ui.TextAlignLeft,
			}))
			builder.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{
				Color: &textColor, FontFamily: family, FontSize: &size, FontWeight: weight,
			}))
			builder.AddText(message)
			b.paragraph = builder.Build()
		}()
	}
	return b
}

// Message is the text of the box.
//
// Flutter: rendering/error.dart RenderErrorBox.message
func (b *RenderErrorBox) Message() string { return b.message }

// ComputeMaxIntrinsicWidth is very large.
//
// Flutter: rendering/error.dart RenderErrorBox.computeMaxIntrinsicWidth
func (b *RenderErrorBox) ComputeMaxIntrinsicWidth(float64) float64 { return errorBoxMaxWidth }

// ComputeMaxIntrinsicHeight is very large.
//
// Flutter: rendering/error.dart RenderErrorBox.computeMaxIntrinsicHeight
func (b *RenderErrorBox) ComputeMaxIntrinsicHeight(float64) float64 { return errorBoxMaxHeight }

// SizedByParent is true: the box takes as much room as it is given.
//
// Flutter: rendering/error.dart RenderErrorBox.sizedByParent
func (b *RenderErrorBox) SizedByParent() bool { return true }

// HitTestSelf is true.
//
// Flutter: rendering/error.dart RenderErrorBox.hitTestSelf
func (b *RenderErrorBox) HitTestSelf(ui.Offset) bool { return true }

// ComputeDryLayout is the largest size the constraints allow, up to a very large
// size.
//
// Flutter: rendering/error.dart RenderErrorBox.computeDryLayout
func (b *RenderErrorBox) ComputeDryLayout(c BoxConstraints) ui.Size {
	return c.Constrain(ui.Size{Width: errorBoxMaxWidth, Height: errorBoxMaxHeight})
}

// Paint draws the background and the text.
//
// Flutter: rendering/error.dart RenderErrorBox.paint
func (b *RenderErrorBox) Paint(context *PaintingContext, offset ui.Offset) {
	// A failure while painting is dropped: the box is the report of another one.
	defer func() { _ = recover() }()
	size := b.Size()
	paint := ui.NewPaint()
	paint.SetColor(b.BackgroundColor)
	context.Canvas().DrawRect(offset.And(size), paint)
	if b.paragraph == nil {
		return
	}
	width, left, top := size.Width, 0.0, 0.0
	if width > b.Padding.Left+b.MinimumWidth+b.Padding.Right {
		width -= b.Padding.Left + b.Padding.Right
		left += b.Padding.Left
	}
	b.paragraph.Layout(ui.NewParagraphConstraints(width))
	if size.Height > b.Padding.Top+b.paragraph.Height()+b.Padding.Bottom {
		top += b.Padding.Top
	}
	context.Canvas().DrawParagraph(b.paragraph, offset.Add(ui.Offset{Dx: left, Dy: top}))
}
