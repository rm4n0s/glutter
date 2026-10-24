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
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// textSelectionToolbarItemPosition is where a button is in the toolbar.
//
// Flutter: material/text_selection_toolbar_text_button.dart _TextSelectionToolbarItemPosition
type textSelectionToolbarItemPosition int

const (
	// textSelectionToolbarItemPositionFirst is the first item among multiple in
	// the menu.
	textSelectionToolbarItemPositionFirst textSelectionToolbarItemPosition = iota
	// textSelectionToolbarItemPositionMiddle is one of several items, not the
	// first or last.
	textSelectionToolbarItemPositionMiddle
	// textSelectionToolbarItemPositionLast is the last item among multiple in
	// the menu.
	textSelectionToolbarItemPositionLast
	// textSelectionToolbarItemPositionOnly is the only item in the menu.
	textSelectionToolbarItemPositionOnly
)

const (
	// textSelectionToolbarMiddlePadding is the padding on the sides of a button
	// that face another button. These values were eyeballed to match the native
	// text selection menu on a Pixel 2 running Android 10.
	//
	// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton._kMiddlePadding
	textSelectionToolbarMiddlePadding = 9.5
	// textSelectionToolbarEndPadding is the padding on the sides of a button
	// that face the edge of the toolbar.
	//
	// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton._kEndPadding
	textSelectionToolbarEndPadding = 14.5
)

// TextSelectionToolbarTextButtonOpts are the optional parameters of
// NewTextSelectionToolbarTextButton.
//
// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton.new
type TextSelectionToolbarTextButtonOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnPressed is called when this button is pressed; nil disables the button.
	OnPressed foundation.VoidCallback
	// Alignment is the alignment of the button's child; the default is
	// Alignment.center.
	Alignment painting.AlignmentGeometry
}

// TextSelectionToolbarTextButton is a button styled like a Material native
// Android text selection menu button.
//
// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton
type TextSelectionToolbarTextButton struct {
	widgets.StatelessWidget
	child     widgets.Widget
	padding   painting.EdgeInsetsGeometry
	onPressed foundation.VoidCallback
	alignment painting.AlignmentGeometry
}

// NewTextSelectionToolbarTextButton creates a button with the given padding
// (TextSelectionToolbarTextButtonGetPadding gives the standard one) around the
// child, which is usually a Text.
//
// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton.new
func NewTextSelectionToolbarTextButton(padding painting.EdgeInsetsGeometry, child widgets.Widget, opts ...TextSelectionToolbarTextButtonOpts) *TextSelectionToolbarTextButton {
	o := firstOpts(opts)
	return &TextSelectionToolbarTextButton{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}),
		child: child, padding: padding, onPressed: o.OnPressed, alignment: o.Alignment}
}

// Child is the child of this button, usually a Text.
//
// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton.child
func (w *TextSelectionToolbarTextButton) Child() widgets.Widget { return w.child }

// OnPressed is called when this button is pressed.
//
// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton.onPressed
func (w *TextSelectionToolbarTextButton) OnPressed() foundation.VoidCallback { return w.onPressed }

// Padding is the padding between the button's edge and its child.
//
// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton.padding
func (w *TextSelectionToolbarTextButton) Padding() painting.EdgeInsetsGeometry { return w.padding }

// Alignment is the alignment of the button's child; nil is the center.
//
// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton.alignment
func (w *TextSelectionToolbarTextButton) Alignment() painting.AlignmentGeometry { return w.alignment }

// TextSelectionToolbarTextButtonGetPadding returns the standard padding for a
// button at index out of a total number of buttons. Standard Material
// TextSelectionToolbars have buttons with different padding depending on their
// position in the toolbar. It panics with the tag
// TextSelectionToolbarTextButtonPosition (in debug builds) when index is not
// below total or total is not above zero.
//
// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton.getPadding
func TextSelectionToolbarTextButtonGetPadding(index, total int) painting.EdgeInsetsGeometry {
	if foundation.DebugMode && (total <= 0 || index < 0 || index >= total) {
		panic(errors.New(tagTextSelectionToolbarTextButtonPosition, "the index of a button must be in [0, total) and the total above zero",
			"index", index, "total", total))
	}
	position := textSelectionToolbarTextButtonPosition(index, total)
	return painting.NewEdgeInsetsDirectionalFromSTEB(
		textSelectionToolbarTextButtonStartPadding(position), 0,
		textSelectionToolbarTextButtonEndPadding(position), 0)
}

// textSelectionToolbarTextButtonStartPadding is the padding of the start side.
//
// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton._getStartPadding
func textSelectionToolbarTextButtonStartPadding(position textSelectionToolbarItemPosition) float64 {
	if position == textSelectionToolbarItemPositionFirst || position == textSelectionToolbarItemPositionOnly {
		return textSelectionToolbarEndPadding
	}
	return textSelectionToolbarMiddlePadding
}

// textSelectionToolbarTextButtonEndPadding is the padding of the end side.
//
// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton._getEndPadding
func textSelectionToolbarTextButtonEndPadding(position textSelectionToolbarItemPosition) float64 {
	if position == textSelectionToolbarItemPositionLast || position == textSelectionToolbarItemPositionOnly {
		return textSelectionToolbarEndPadding
	}
	return textSelectionToolbarMiddlePadding
}

// textSelectionToolbarTextButtonPosition is the position of the item.
//
// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton._getPosition
func textSelectionToolbarTextButtonPosition(index, total int) textSelectionToolbarItemPosition {
	if index == 0 {
		if total == 1 {
			return textSelectionToolbarItemPositionOnly
		}
		return textSelectionToolbarItemPositionFirst
	}
	if index == total-1 {
		return textSelectionToolbarItemPositionLast
	}
	return textSelectionToolbarItemPositionMiddle
}

// TextSelectionToolbarTextButtonCopy holds the values that
// TextSelectionToolbarTextButton.CopyWith replaces; the nil ones keep the
// value of the button.
//
// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton.copyWith
type TextSelectionToolbarTextButtonCopy struct {
	Child     widgets.Widget
	OnPressed foundation.VoidCallback
	Padding   painting.EdgeInsetsGeometry
	Alignment painting.AlignmentGeometry
}

// CopyWith returns a copy of this button with specific overrides.
//
// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton.copyWith
func (w *TextSelectionToolbarTextButton) CopyWith(c TextSelectionToolbarTextButtonCopy) *TextSelectionToolbarTextButton {
	out := &TextSelectionToolbarTextButton{StatelessWidget: widgets.NewStatelessWidgetBase(), child: w.child, padding: w.padding,
		onPressed: w.onPressed, alignment: w.alignment}
	if c.OnPressed != nil {
		out.onPressed = c.OnPressed
	}
	if c.Padding != nil {
		out.padding = c.Padding
	}
	if c.Alignment != nil {
		out.alignment = c.Alignment
	}
	if c.Child != nil {
		out.child = c.Child
	}
	return out
}

var (
	// These colors were taken from a screenshot of a Pixel 6 emulator running
	// Android API level 34.
	//
	// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton._defaultForegroundColorLight
	textSelectionToolbarDefaultForegroundColorLight = ui.NewColor(0xff000000)
	// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton._defaultForegroundColorDark
	textSelectionToolbarDefaultForegroundColorDark = ui.NewColor(0xffffffff)
	// The background color is hardcoded to transparent by default so the buttons
	// are the color of the container behind them. For example TextSelectionToolbar
	// hardcodes the color value, and TextSelectionToolbarTextButtons that are its
	// children become that color.
	//
	// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton._defaultBackgroundColorTransparent
	textSelectionToolbarDefaultBackgroundColorTransparent = ui.NewColor(0x00000000)
)

// textSelectionToolbarTextButtonForegroundColor is the color of the text: black
// or white, unless the scheme has an onSurface that is not the default one.
//
// Deviation: Flutter compares the onSurface to the one of ThemeData() with
// identical(), which holds for the colors of the default theme; Glutter compares
// the colors by value (SPEC §12.10).
//
// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton._getForegroundColor
func textSelectionToolbarTextButtonForegroundColor(colorScheme *ColorScheme) ui.Color {
	isDark := colorScheme.Brightness() == ui.BrightnessDark
	defaultScheme := colorSchemeLightM3
	if isDark {
		defaultScheme = colorSchemeDarkM3
	}
	if defaultScheme.OnSurface() != colorScheme.OnSurface() {
		return colorScheme.OnSurface()
	}
	if isDark {
		return textSelectionToolbarDefaultForegroundColorDark
	}
	return textSelectionToolbarDefaultForegroundColorLight
}

// Build builds a text button without a background, a square shape and the
// padding of the position of the button.
//
// Flutter: material/text_selection_toolbar_text_button.dart TextSelectionToolbarTextButton.build
func (w *TextSelectionToolbarTextButton) Build(bc widgets.BuildContext) widgets.Widget {
	colorScheme := ThemeOf(bc).ColorScheme
	return NewTextButton(w.onPressed, w.child, TextButtonOpts{Style: TextButtonStyleFrom(ButtonStyleFromOpts{
		BackgroundColor: foundation.Some(textSelectionToolbarDefaultBackgroundColorTransparent),
		ForegroundColor: foundation.Some(textSelectionToolbarTextButtonForegroundColor(colorScheme)),
		Shape:           painting.NewRoundedRectangleBorder(),
		MinimumSize:     foundation.Some(ui.Size{Width: MinInteractiveDimension, Height: MinInteractiveDimension}),
		Padding:         w.padding,
		Alignment:       w.alignment,
		// This value was eyeballed from a screenshot of a Pixel 6 emulator
		// running Android API level 34.
		TextStyle: painting.NewTextStyle(painting.TextStyleOpts{FontWeight: ui.FontWeightW400}),
	})})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTextSelectionToolbarTextButtonPosition = "TextSelectionToolbarTextButtonPosition"
)
