// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	// desktopToolbarScreenPadding is the minimum space between the toolbar and the
	// edges of the screen.
	//
	// Flutter: material/desktop_text_selection_toolbar.dart _kToolbarScreenPadding
	desktopToolbarScreenPadding = 8.0
	// desktopToolbarWidth is the width of the toolbar.
	//
	// Flutter: material/desktop_text_selection_toolbar.dart _kToolbarWidth
	desktopToolbarWidth = 222.0
)

// DesktopTextSelectionToolbar is a Material-style desktop text selection
// toolbar: a menu in a card with one button for each child, placed at the anchor.
//
// Flutter: material/desktop_text_selection_toolbar.dart DesktopTextSelectionToolbar
type DesktopTextSelectionToolbar struct {
	widgets.StatelessWidget
	anchor   ui.Offset
	children []widgets.Widget
}

// NewDesktopTextSelectionToolbar creates the toolbar. It panics with the tag
// DesktopTextSelectionToolbarNoChildren (in debug builds) when there is no
// child.
//
// Flutter: material/desktop_text_selection_toolbar.dart DesktopTextSelectionToolbar.new
func NewDesktopTextSelectionToolbar(anchor ui.Offset, children []widgets.Widget, opts ...widgets.WidgetOpts) *DesktopTextSelectionToolbar {
	if foundation.DebugMode && len(children) == 0 {
		panic(errors.New("DesktopTextSelectionToolbarNoChildren", "a DesktopTextSelectionToolbar needs at least one child"))
	}
	return &DesktopTextSelectionToolbar{StatelessWidget: widgets.NewStatelessWidgetBase(firstOpts(opts)), anchor: anchor, children: children}
}

// Anchor is the point that the toolbar is placed at.
//
// Flutter: material/desktop_text_selection_toolbar.dart DesktopTextSelectionToolbar.anchor
func (w *DesktopTextSelectionToolbar) Anchor() ui.Offset { return w.anchor }

// Children are the buttons.
//
// Flutter: material/desktop_text_selection_toolbar.dart DesktopTextSelectionToolbar.children
func (w *DesktopTextSelectionToolbar) Children() []widgets.Widget { return w.children }

// Build puts the buttons in a card, 222 wide, and the card at the anchor.
//
// Flutter: material/desktop_text_selection_toolbar.dart DesktopTextSelectionToolbar.build
func (w *DesktopTextSelectionToolbar) Build(bc widgets.BuildContext) widgets.Widget {
	widgets.DebugCheckHasMediaQuery(bc)
	paddingAbove := widgets.MediaQueryPaddingOf(bc).Top + desktopToolbarScreenPadding
	localAdjustment := ui.Offset{Dx: desktopToolbarScreenPadding, Dy: paddingAbove}
	return widgets.NewPadding(
		painting.NewEdgeInsetsFromLTRB(desktopToolbarScreenPadding, paddingAbove, desktopToolbarScreenPadding, desktopToolbarScreenPadding),
		widgets.NewCustomSingleChildLayout(
			widgets.NewDesktopTextSelectionToolbarLayoutDelegate(w.anchor.Sub(localAdjustment)),
			desktopToolbarBuilder(bc, widgets.NewColumn(w.children, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin}))))
}

// desktopToolbarBuilder is the card of the toolbar.
//
// Flutter: material/desktop_text_selection_toolbar.dart DesktopTextSelectionToolbar._defaultToolbarBuilder
func desktopToolbarBuilder(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return widgets.NewSizedBox(
		NewMaterial(child, MaterialOpts{
			BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(7.0)),
			ClipBehavior: ui.ClipAntiAlias,
			Elevation:    1.0,
			Type:         MaterialTypeCard,
		}),
		widgets.SizedBoxOpts{Width: foundation.Some(desktopToolbarWidth)})
}

// desktopToolbarButtonFontStyle is the style of the text of a button.
//
// Flutter: material/desktop_text_selection_toolbar_button.dart _kToolbarButtonFontStyle
var desktopToolbarButtonFontStyle = painting.NewTextStyle(painting.TextStyleOpts{
	Inherit:       foundation.Some(false),
	FontSize:      foundation.Some(14.0),
	LetterSpacing: foundation.Some(-0.15),
	FontWeight:    ui.FontWeightW400,
})

// desktopToolbarButtonPadding is the padding of a button.
//
// Flutter: material/desktop_text_selection_toolbar_button.dart _kToolbarButtonPadding
var desktopToolbarButtonPadding = painting.NewEdgeInsetsFromLTRB(20.0, 0.0, 20.0, 3.0)

// DesktopTextSelectionToolbarButton is a button for a desktop text selection
// toolbar.
//
// Flutter: material/desktop_text_selection_toolbar_button.dart DesktopTextSelectionToolbarButton
type DesktopTextSelectionToolbarButton struct {
	widgets.StatelessWidget
	onPressed foundation.VoidCallback
	child     widgets.Widget
}

// NewDesktopTextSelectionToolbarButton creates a button with a child widget; a
// nil onPressed disables it.
//
// Flutter: material/desktop_text_selection_toolbar_button.dart DesktopTextSelectionToolbarButton.new
func NewDesktopTextSelectionToolbarButton(onPressed foundation.VoidCallback, child widgets.Widget, opts ...widgets.WidgetOpts) *DesktopTextSelectionToolbarButton {
	return &DesktopTextSelectionToolbarButton{StatelessWidget: widgets.NewStatelessWidgetBase(firstOpts(opts)), onPressed: onPressed, child: child}
}

// NewDesktopTextSelectionToolbarButtonText creates a button with a text.
//
// Flutter: material/desktop_text_selection_toolbar_button.dart DesktopTextSelectionToolbarButton.text
func NewDesktopTextSelectionToolbarButtonText(bc widgets.BuildContext, onPressed foundation.VoidCallback, text string, opts ...widgets.WidgetOpts) *DesktopTextSelectionToolbarButton {
	color := Colors.Black87
	if ThemeOf(bc).ColorScheme.Brightness() == ui.BrightnessDark {
		color = Colors.White
	}
	return NewDesktopTextSelectionToolbarButton(onPressed, widgets.NewText(text, widgets.TextOpts{
		Overflow: painting.TextOverflowEllipsis,
		Style:    desktopToolbarButtonFontStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(color)}),
	}), opts...)
}

// OnPressed is called when the button is pressed.
//
// Flutter: material/desktop_text_selection_toolbar_button.dart DesktopTextSelectionToolbarButton.onPressed
func (w *DesktopTextSelectionToolbarButton) OnPressed() foundation.VoidCallback { return w.onPressed }

// Child is the content of the button.
//
// Flutter: material/desktop_text_selection_toolbar_button.dart DesktopTextSelectionToolbarButton.child
func (w *DesktopTextSelectionToolbarButton) Child() widgets.Widget { return w.child }

// Build builds a text button as wide as the toolbar.
//
// Flutter: material/desktop_text_selection_toolbar_button.dart DesktopTextSelectionToolbarButton.build
func (w *DesktopTextSelectionToolbarButton) Build(bc widgets.BuildContext) widgets.Widget {
	theme := ThemeOf(bc)
	foreground := Colors.Black87
	if theme.ColorScheme.Brightness() == ui.BrightnessDark {
		foreground = Colors.White
	}
	return widgets.NewSizedBox(
		NewTextButton(w.onPressed, w.child, TextButtonOpts{Style: TextButtonStyleFrom(ButtonStyleFromOpts{
			Alignment:           painting.AlignmentCenterLeft,
			EnabledMouseCursor:  services.SystemMouseCursors.Basic,
			DisabledMouseCursor: services.SystemMouseCursors.Basic,
			ForegroundColor:     foundation.Some(foreground),
			Shape:               painting.NewRoundedRectangleBorder(),
			MinimumSize:         foundation.Some(ui.Size{Width: MinInteractiveDimension, Height: 36.0}),
			Padding:             desktopToolbarButtonPadding,
		})}),
		widgets.SizedBoxOpts{Width: foundation.Some(math.Inf(1))})
}
