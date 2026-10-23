// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// defaultDialogInsetPadding is the distance between the edges of the screen and
// the dialog.
//
// Flutter: material/dialog.dart _defaultInsetPadding
var defaultDialogInsetPadding = painting.NewEdgeInsetsSymmetric(24, 40)

// DialogOpts are the optional parameters of NewDialog.
//
// Flutter: material/dialog.dart Dialog.new
type DialogOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// BackgroundColor is the background color of the surface; the default is the
	// one of the dialog theme.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the dialog; it must not be negative.
	Elevation foundation.Opt[float64]
	// ShadowColor is the color of the shadow below the dialog.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the dialog.
	SurfaceTintColor foundation.Opt[ui.Color]
	// InsetAnimationDuration is the duration of the animation that follows the
	// view insets, such as the keyboard; the default is 100 ms.
	InsetAnimationDuration foundation.Opt[time.Duration]
	// InsetAnimationCurve is the curve of that animation; the default is
	// Curves.Decelerate.
	InsetAnimationCurve animation.Curve
	// InsetPadding is the distance between the edges of the screen and the
	// dialog; the default is 40 horizontally and 24 vertically.
	InsetPadding *painting.EdgeInsets
	// ClipBehavior is how the content is clipped; the default is none.
	ClipBehavior ui.Clip
	// Shape is the shape of the dialog.
	Shape painting.ShapeBorder
	// Alignment is where the dialog is in the space left by the inset padding.
	Alignment painting.AlignmentGeometry
	// SemanticsRole is the role of the dialog for accessibility; the default is
	// dialog.
	SemanticsRole foundation.Opt[ui.SemanticsRole]
	// Constraints are the size constraints of the dialog; the default is a
	// minimum width of 280.
	Constraints foundation.Opt[rendering.BoxConstraints]
}

// Dialog is a Material Design dialog: a surface above the app, with its child
// inside. It is used by AlertDialog and SimpleDialog.
//
// Flutter: material/dialog.dart Dialog
type Dialog struct {
	widgets.StatelessWidget
	child      widgets.Widget
	o          DialogOpts
	fullscreen bool
}

// NewDialog creates a dialog around the child, which can be nil. In debug builds
// it panics with the tag DialogElevationNegative when the elevation is negative.
//
// Flutter: material/dialog.dart Dialog.new
func NewDialog(child widgets.Widget, opts ...DialogOpts) *Dialog {
	o := firstOpts(opts)
	if foundation.DebugMode && o.Elevation.IsSome() && o.Elevation.Value() < 0 {
		panic(errors.New(tagDialogElevationNegative, "the elevation of a dialog must not be negative", "elevation", o.Elevation.Value()))
	}
	return &Dialog{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child, o: o}
}

// DialogFullscreenOpts are the optional parameters of NewDialogFullscreen.
//
// Flutter: material/dialog.dart Dialog.fullscreen
type DialogFullscreenOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// BackgroundColor is the background color of the surface; the default is the
	// surface color of the color scheme.
	BackgroundColor foundation.Opt[ui.Color]
	// InsetAnimationDuration is the duration of the animation that follows the
	// view insets; the default is zero.
	InsetAnimationDuration foundation.Opt[time.Duration]
	// InsetAnimationCurve is the curve of that animation; the default is
	// Curves.Decelerate.
	InsetAnimationCurve animation.Curve
	// SemanticsRole is the role of the dialog for accessibility; the default is
	// dialog.
	SemanticsRole foundation.Opt[ui.SemanticsRole]
}

// NewDialogFullscreen creates a dialog that fills the whole screen.
//
// Flutter: material/dialog.dart Dialog.fullscreen
func NewDialogFullscreen(child widgets.Widget, opts ...DialogFullscreenOpts) *Dialog {
	f := firstOpts(opts)
	d := &Dialog{
		StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: f.Key}),
		child:           child,
		fullscreen:      true,
		o: DialogOpts{
			BackgroundColor:        f.BackgroundColor,
			InsetAnimationDuration: foundation.Some(f.InsetAnimationDuration.OrElse(0)),
			InsetAnimationCurve:    f.InsetAnimationCurve,
			SemanticsRole:          f.SemanticsRole,
			Elevation:              foundation.Some(0.0),
			InsetPadding:           &painting.EdgeInsets{},
			ClipBehavior:           ui.ClipNone,
		},
	}
	return d
}

// Child is the widget in the dialog.
//
// Flutter: material/dialog.dart Dialog.child
func (w *Dialog) Child() widgets.Widget { return w.child }

// Build builds the dialog: an animated padding that follows the view insets
// around an aligned, constrained Material.
//
// Flutter: material/dialog.dart Dialog.build
func (w *Dialog) Build(bc widgets.BuildContext) widgets.Widget {
	theme := ThemeOf(bc)
	dialogTheme := DialogThemeOf(bc)
	o := w.o
	insets := widgets.MediaQueryViewInsetsOf(bc)
	inset := defaultDialogInsetPadding
	if o.InsetPadding != nil {
		inset = *o.InsetPadding
	} else if dialogTheme.InsetPadding != nil {
		inset = *dialogTheme.InsetPadding
	}
	effectivePadding := painting.NewEdgeInsetsFromLTRB(insets.Left+inset.Left, insets.Top+inset.Top, insets.Right+inset.Right, insets.Bottom+inset.Bottom)
	var defaults DialogThemeData
	switch {
	case !theme.UseMaterial3:
		defaults = dialogDefaultsM2(theme)
	case w.fullscreen:
		defaults = dialogFullscreenDefaultsM3(theme)
	default:
		defaults = dialogDefaultsM3(theme)
	}
	boxConstraints := rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinWidth: foundation.Some(280.0)})
	if c, ok := firstOpt(o.Constraints, dialogTheme.Constraints).Get(); ok {
		boxConstraints = c
	}
	backgroundColor := firstOpt(o.BackgroundColor, dialogTheme.BackgroundColor, defaults.BackgroundColor)

	var dialogChild widgets.Widget
	if w.fullscreen {
		dialogChild = NewMaterial(w.child, MaterialOpts{Color: backgroundColor})
	} else {
		alignment := firstNonNil(o.Alignment, dialogTheme.Alignment, defaults.Alignment)
		shape := firstNonNil(o.Shape, dialogTheme.Shape, defaults.Shape)
		clip := firstNonZero(o.ClipBehavior, dialogTheme.ClipBehavior, defaults.ClipBehavior)
		dialogChild = widgets.NewAlign(widgets.NewConstrainedBox(boxConstraints, NewMaterial(w.child, MaterialOpts{
			Color:            backgroundColor,
			Elevation:        firstOpt(o.Elevation, dialogTheme.Elevation, defaults.Elevation).Value(),
			ShadowColor:      firstOpt(o.ShadowColor, dialogTheme.ShadowColor, defaults.ShadowColor),
			SurfaceTintColor: firstOpt(o.SurfaceTintColor, dialogTheme.SurfaceTintColor, defaults.SurfaceTintColor),
			Shape:            shape,
			Type:             MaterialTypeCard,
			ClipBehavior:     clip,
		})), widgets.AlignOpts{Alignment: alignment})
	}
	curve := o.InsetAnimationCurve
	if curve == nil {
		curve = animation.Curves.Decelerate
	}
	duration := o.InsetAnimationDuration.OrElse(100 * time.Millisecond)
	role := o.SemanticsRole.OrElse(ui.SemanticsRoleDialog)
	return widgets.NewSemantics(
		widgets.NewAnimatedPadding(effectivePadding, duration,
			widgets.NewMediaQueryRemoveViewInsets(bc, widgets.RemoveSides{Left: true, Top: true, Right: true, Bottom: true}, dialogChild),
			widgets.AnimatedPaddingOpts{Curve: curve}),
		widgets.SemanticsOpts{Role: foundation.Some(role)})
}

// firstNonNil returns the first value that is not nil.
func firstNonNil[T comparable](values ...T) T {
	var zero T
	for _, v := range values {
		if v != zero {
			return v
		}
	}
	return zero
}

// firstNonZero returns the first value that is not the zero value.
func firstNonZero[T comparable](values ...T) T { return firstNonNil(values...) }

// dialogScalePadding is the factor that shrinks the padding of a dialog when
// the text scale grows: 1 at a scale of 1, a third at a scale of 2.
//
// Flutter: material/dialog.dart _scalePadding
func dialogScalePadding(textScaleFactor float64) float64 {
	clamped := ui.ClampDouble(textScaleFactor, 1, 2)
	return 1 + (1.0/3.0-1)*(clamped-1)
}

// textDirectionOf is the direction of the closest Directionality, or left to
// right without one.
func textDirectionOf(bc widgets.BuildContext) ui.TextDirection {
	if d, ok := widgets.MaybeDirectionalityOf(bc); ok {
		return d
	}
	return ui.TextDirectionLTR
}

// AlertDialogOpts are the optional parameters of NewAlertDialog.
//
// Flutter: material/dialog.dart AlertDialog.new
type AlertDialogOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Icon is the optional icon at the top, above the title.
	Icon widgets.Widget
	// IconPadding is the padding around the icon.
	IconPadding painting.EdgeInsetsGeometry
	// IconColor is the color of the icon; the default is the one of the dialog
	// theme.
	IconColor foundation.Opt[ui.Color]
	// Title is the optional title.
	Title widgets.Widget
	// TitlePadding is the padding around the title.
	TitlePadding painting.EdgeInsetsGeometry
	// TitleTextStyle is the style of the title.
	TitleTextStyle *painting.TextStyle
	// Content is the optional content, below the title.
	Content widgets.Widget
	// ContentPadding is the padding around the content.
	ContentPadding painting.EdgeInsetsGeometry
	// ContentTextStyle is the style of the content.
	ContentTextStyle *painting.TextStyle
	// Actions are the buttons at the bottom. Nil means there is no row of
	// actions.
	Actions []widgets.Widget
	// ActionsPadding is the padding around the actions.
	ActionsPadding painting.EdgeInsetsGeometry
	// ActionsAlignment is how the actions are aligned in a row; the default is
	// end.
	ActionsAlignment foundation.Opt[rendering.MainAxisAlignment]
	// ActionsOverflowAlignment is how the actions are aligned in a column when
	// they overflow; the default is end.
	ActionsOverflowAlignment rendering.OverflowBarAlignment
	// ActionsOverflowDirection is the direction of the column of overflowing
	// actions; the default is down.
	ActionsOverflowDirection painting.VerticalDirection
	// ActionsOverflowButtonSpacing is the vertical space between overflowing
	// actions; the default is 0.
	ActionsOverflowButtonSpacing foundation.Opt[float64]
	// ButtonPadding is the padding around each action; only its horizontal
	// extent is used, as spacing between the actions. The default is 16.
	ButtonPadding painting.EdgeInsetsGeometry
	// BackgroundColor is the background color of the dialog.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the dialog.
	Elevation foundation.Opt[float64]
	// ShadowColor is the color of the shadow below the dialog.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the dialog.
	SurfaceTintColor foundation.Opt[ui.Color]
	// SemanticLabel is the label that screen readers announce for the dialog;
	// the default is the alert dialog label of the localizations.
	SemanticLabel foundation.Opt[string]
	// InsetPadding is the distance between the edges of the screen and the
	// dialog.
	InsetPadding *painting.EdgeInsets
	// ClipBehavior is how the content is clipped.
	ClipBehavior ui.Clip
	// Shape is the shape of the dialog.
	Shape painting.ShapeBorder
	// Alignment is where the dialog is in the space left by the inset padding.
	Alignment painting.AlignmentGeometry
	// Constraints are the size constraints of the dialog.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// Scrollable makes the title and the content scroll together, with the
	// actions fixed at the bottom.
	Scrollable bool
}

// AlertDialog is a Material Design alert dialog: it informs the user about
// situations that require acknowledgement, with an optional icon, title,
// content and a row of actions.
//
// Flutter: material/dialog.dart AlertDialog
type AlertDialog struct {
	widgets.StatelessWidget
	o AlertDialogOpts
}

// NewAlertDialog creates an alert dialog.
//
// Flutter: material/dialog.dart AlertDialog.new
func NewAlertDialog(opts ...AlertDialogOpts) *AlertDialog {
	o := firstOpts(opts)
	return &AlertDialog{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// NewAlertDialogAdaptive creates an alert dialog that adapts to the platform.
//
// Deviation: Glutter has no Cupertino library, so the dialog is the Material
// one on every platform, including iOS and macOS (SPEC §12.10).
//
// Flutter: material/dialog.dart AlertDialog.adaptive
func NewAlertDialogAdaptive(opts ...AlertDialogOpts) *AlertDialog { return NewAlertDialog(opts...) }

// Build builds the dialog.
//
// Flutter: material/dialog.dart AlertDialog.build
func (w *AlertDialog) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterialLocalizations(bc)
	o := w.o
	theme := ThemeOf(bc)
	dialogTheme := DialogThemeOf(bc)
	defaults := dialogDefaultsM2(theme)
	if theme.UseMaterial3 {
		defaults = dialogDefaultsM3(theme)
	}
	label := o.SemanticLabel
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
	default:
		if label.IsNone() {
			label = foundation.Some(MaterialLocalizationsOf(bc).AlertDialogLabel())
		}
	}
	const fontSizeToScale = 14.0
	effectiveTextScale := widgets.MediaQueryTextScalerOf(bc).Scale(fontSizeToScale) / fontSizeToScale
	paddingScaleFactor := dialogScalePadding(effectiveTextScale)
	textDirection := textDirectionOf(bc)
	namesRoute := foundation.Some(label.IsNone() && foundation.DefaultTargetPlatform() != foundation.TargetPlatformIOS)

	var iconWidget, titleWidget, contentWidget, actionsWidget widgets.Widget
	if o.Icon != nil {
		belowIsTitle := o.Title != nil
		belowIsContent := !belowIsTitle && o.Content != nil
		bottom := 24.0
		if belowIsTitle {
			bottom = 16
		} else if belowIsContent {
			bottom = 0
		}
		effective := painting.NewEdgeInsetsFromLTRB(24, 24, 24, bottom)
		if o.IconPadding != nil {
			effective = o.IconPadding.Resolve(textDirection)
		}
		iconWidget = widgets.NewPadding(
			painting.NewEdgeInsetsFromLTRB(effective.Left*paddingScaleFactor, effective.Top*paddingScaleFactor, effective.Right*paddingScaleFactor, effective.Bottom),
			widgets.NewIconTheme(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: firstOpt(o.IconColor, dialogTheme.IconColor, defaults.IconColor)}), o.Icon))
	}
	if o.Title != nil {
		top := 24.0
		if o.Icon != nil {
			top = 0
		}
		bottom := 0.0
		if o.Content == nil {
			bottom = 20
		}
		effective := painting.NewEdgeInsetsFromLTRB(24, top, 24, bottom)
		if o.TitlePadding != nil {
			effective = o.TitlePadding.Resolve(textDirection)
		}
		topPad := effective.Top * paddingScaleFactor
		if o.Icon != nil {
			topPad = effective.Top
		}
		align := ui.TextAlignStart
		if o.Icon != nil {
			align = ui.TextAlignCenter
		}
		titleWidget = widgets.NewPadding(
			painting.NewEdgeInsetsFromLTRB(effective.Left*paddingScaleFactor, topPad, effective.Right*paddingScaleFactor, effective.Bottom),
			widgets.NewDefaultTextStyle(firstNonNil(o.TitleTextStyle, dialogTheme.TitleTextStyle, defaults.TitleTextStyle),
				widgets.NewSemantics(o.Title, widgets.SemanticsOpts{NamesRoute: namesRoute, Container: true}),
				widgets.DefaultTextStyleOpts{TextAlign: align}))
	}
	if o.Content != nil {
		top := 20.0
		if theme.UseMaterial3 {
			top = 16
		}
		effective := painting.NewEdgeInsetsFromLTRB(24, top, 24, 24)
		if o.ContentPadding != nil {
			effective = o.ContentPadding.Resolve(textDirection)
		}
		topPad := effective.Top
		if o.Title == nil && o.Icon == nil {
			topPad = effective.Top * paddingScaleFactor
		}
		contentWidget = widgets.NewPadding(
			painting.NewEdgeInsetsFromLTRB(effective.Left*paddingScaleFactor, topPad, effective.Right*paddingScaleFactor, effective.Bottom),
			widgets.NewDefaultTextStyle(firstNonNil(o.ContentTextStyle, dialogTheme.ContentTextStyle, defaults.ContentTextStyle),
				widgets.NewSemantics(o.Content, widgets.SemanticsOpts{Container: true, ExplicitChildNodes: true})))
	}
	if o.Actions != nil {
		spacing := 8.0
		if o.ButtonPadding != nil {
			spacing = o.ButtonPadding.Horizontal() / 2
		}
		var padding painting.EdgeInsetsGeometry
		switch {
		case o.ActionsPadding != nil:
			padding = o.ActionsPadding
		case dialogTheme.ActionsPadding != nil:
			padding = dialogTheme.ActionsPadding
		case theme.UseMaterial3:
			padding = defaults.ActionsPadding
		default:
			padding = defaults.ActionsPadding.Add(painting.NewEdgeInsetsAll(spacing))
		}
		overflowAlignment := o.ActionsOverflowAlignment
		if overflowAlignment == rendering.OverflowBarAlignmentUnspecified {
			overflowAlignment = rendering.OverflowBarAlignmentEnd
		}
		overflowDirection := o.ActionsOverflowDirection
		if overflowDirection == painting.VerticalDirectionUnspecified {
			overflowDirection = painting.VerticalDirectionDown
		}
		actionsWidget = widgets.NewPadding(padding, widgets.NewOverflowBar(o.Actions, widgets.OverflowBarOpts{
			Alignment:         foundation.Some(o.ActionsAlignment.OrElse(rendering.MainAxisAlignmentEnd)),
			Spacing:           spacing,
			OverflowAlignment: overflowAlignment,
			OverflowDirection: overflowDirection,
			OverflowSpacing:   o.ActionsOverflowButtonSpacing.OrElse(0),
		}))
	}

	var columnChildren []widgets.Widget
	add := func(ws ...widgets.Widget) {
		for _, c := range ws {
			if c != nil {
				columnChildren = append(columnChildren, c)
			}
		}
	}
	if o.Scrollable {
		if o.Title != nil || o.Content != nil {
			var inner []widgets.Widget
			for _, c := range []widgets.Widget{iconWidget, titleWidget, contentWidget} {
				if c != nil {
					inner = append(inner, c)
				}
			}
			add(widgets.NewFlexible(widgets.NewSingleChildScrollView(widgets.NewColumn(inner, widgets.ColumnOpts{
				MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch,
			}))))
		}
		add(actionsWidget)
	} else {
		add(iconWidget, titleWidget)
		if contentWidget != nil {
			add(widgets.NewFlexible(contentWidget))
		}
		add(actionsWidget)
	}
	var dialogChild widgets.Widget = widgets.NewIntrinsicWidth(widgets.NewColumn(columnChildren, widgets.ColumnOpts{
		MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch,
	}))
	if l, ok := label.Get(); ok {
		dialogChild = widgets.NewSemantics(dialogChild, widgets.SemanticsOpts{
			ScopesRoute: foundation.Some(true), ExplicitChildNodes: true, NamesRoute: foundation.Some(true), Label: foundation.Some(l),
		})
	}
	return NewDialog(dialogChild, DialogOpts{
		BackgroundColor: o.BackgroundColor, Elevation: o.Elevation, ShadowColor: o.ShadowColor, SurfaceTintColor: o.SurfaceTintColor,
		InsetPadding: o.InsetPadding, ClipBehavior: o.ClipBehavior, Shape: o.Shape, Alignment: o.Alignment, Constraints: o.Constraints,
		SemanticsRole: foundation.Some(ui.SemanticsRoleAlertDialog),
	})
}

// SimpleDialogOptionOpts are the optional parameters of NewSimpleDialogOption.
//
// Flutter: material/dialog.dart SimpleDialogOption.new
type SimpleDialogOptionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Padding is the padding around the child; the default is 8 vertically and 24
	// horizontally.
	Padding *painting.EdgeInsets
}

// SimpleDialogOption is an option in a SimpleDialog: an ink well with padding.
//
// Flutter: material/dialog.dart SimpleDialogOption
type SimpleDialogOption struct {
	widgets.StatelessWidget
	onPressed foundation.VoidCallback
	child     widgets.Widget
	o         SimpleDialogOptionOpts
}

// NewSimpleDialogOption creates an option. A nil onPressed disables it.
//
// Flutter: material/dialog.dart SimpleDialogOption.new
func NewSimpleDialogOption(onPressed foundation.VoidCallback, child widgets.Widget, opts ...SimpleDialogOptionOpts) *SimpleDialogOption {
	o := firstOpts(opts)
	return &SimpleDialogOption{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), onPressed: onPressed, child: child, o: o}
}

// Build builds the option.
//
// Flutter: material/dialog.dart SimpleDialogOption.build
func (w *SimpleDialogOption) Build(widgets.BuildContext) widgets.Widget {
	padding := painting.NewEdgeInsetsSymmetric(8, 24)
	if w.o.Padding != nil {
		padding = *w.o.Padding
	}
	return NewInkWell(widgets.NewPadding(padding, w.child), InkWellOpts{OnTap: gestures.GestureTapCallback(w.onPressed)})
}

// SimpleDialogOpts are the optional parameters of NewSimpleDialog.
//
// Flutter: material/dialog.dart SimpleDialog.new
type SimpleDialogOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Title is the optional title.
	Title widgets.Widget
	// TitlePadding is the padding around the title; the default is 24 on the top
	// and the sides.
	TitlePadding painting.EdgeInsetsGeometry
	// TitleTextStyle is the style of the title.
	TitleTextStyle *painting.TextStyle
	// ContentPadding is the padding around the children; the default is 12 on
	// the top and 16 on the bottom.
	ContentPadding painting.EdgeInsetsGeometry
	// ContentTextStyle is the style of the children.
	ContentTextStyle *painting.TextStyle
	// BackgroundColor is the background color of the dialog.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the dialog.
	Elevation foundation.Opt[float64]
	// ShadowColor is the color of the shadow below the dialog.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the dialog.
	SurfaceTintColor foundation.Opt[ui.Color]
	// SemanticLabel is the label that screen readers announce for the dialog.
	SemanticLabel foundation.Opt[string]
	// InsetPadding is the distance between the edges of the screen and the
	// dialog.
	InsetPadding *painting.EdgeInsets
	// ClipBehavior is how the content is clipped.
	ClipBehavior ui.Clip
	// Shape is the shape of the dialog.
	Shape painting.ShapeBorder
	// Alignment is where the dialog is in the space left by the inset padding.
	Alignment painting.AlignmentGeometry
	// Constraints are the size constraints of the dialog.
	Constraints foundation.Opt[rendering.BoxConstraints]
}

// SimpleDialog is a simple Material Design dialog: an optional title and a list
// of options, which scrolls when it does not fit.
//
// Flutter: material/dialog.dart SimpleDialog
type SimpleDialog struct {
	widgets.StatelessWidget
	children []widgets.Widget
	o        SimpleDialogOpts
}

// NewSimpleDialog creates a dialog with the options as its children. Nil children
// leave out the list.
//
// Flutter: material/dialog.dart SimpleDialog.new
func NewSimpleDialog(children []widgets.Widget, opts ...SimpleDialogOpts) *SimpleDialog {
	o := firstOpts(opts)
	return &SimpleDialog{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), children: children, o: o}
}

// Build builds the dialog.
//
// Flutter: material/dialog.dart SimpleDialog.build
func (w *SimpleDialog) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterialLocalizations(bc)
	o := w.o
	theme := ThemeOf(bc)
	dialogTheme := DialogThemeOf(bc)
	defaults := dialogDefaultsM2(theme)
	if theme.UseMaterial3 {
		defaults = dialogDefaultsM3(theme)
	}
	label := o.SemanticLabel
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
	default:
		if label.IsNone() {
			label = foundation.Some(MaterialLocalizationsOf(bc).DialogLabel())
		}
	}
	titleStyle := firstNonNil(o.TitleTextStyle, dialogTheme.TitleTextStyle, theme.TextTheme.TitleLarge)
	fontSize := titleStyle.FontSize.OrElse(painting.DefaultFontSize)
	fontSizeToScale := fontSize
	if fontSizeToScale == 0 {
		fontSizeToScale = painting.DefaultFontSize
	}
	effectiveTextScale := widgets.MediaQueryTextScalerOf(bc).Scale(fontSizeToScale) / fontSizeToScale
	paddingScaleFactor := dialogScalePadding(effectiveTextScale)
	textDirection := textDirectionOf(bc)
	hasChildren := w.children != nil

	var titleWidget, contentWidget widgets.Widget
	if o.Title != nil {
		effective := painting.NewEdgeInsetsFromLTRB(24, 24, 24, 0)
		if o.TitlePadding != nil {
			effective = o.TitlePadding.Resolve(textDirection)
		}
		bottom := effective.Bottom
		if !hasChildren {
			bottom *= paddingScaleFactor
		}
		titleWidget = widgets.NewPadding(
			painting.NewEdgeInsetsFromLTRB(effective.Left*paddingScaleFactor, effective.Top*paddingScaleFactor, effective.Right*paddingScaleFactor, bottom),
			widgets.NewDefaultTextStyle(titleStyle, widgets.NewSemantics(o.Title, widgets.SemanticsOpts{
				NamesRoute: foundation.Some(label.IsNone() && foundation.DefaultTargetPlatform() != foundation.TargetPlatformIOS), Container: true,
			})))
	}
	if hasChildren {
		effective := painting.NewEdgeInsetsFromLTRB(0, 12, 0, 16)
		if o.ContentPadding != nil {
			effective = o.ContentPadding.Resolve(textDirection)
		}
		top := effective.Top
		if o.Title == nil {
			top *= paddingScaleFactor
		}
		contentWidget = widgets.NewFlexible(widgets.NewSingleChildScrollView(
			widgets.NewDefaultTextStyle(firstNonNil(o.ContentTextStyle, dialogTheme.ContentTextStyle, defaults.ContentTextStyle), widgets.NewListBody(w.children)),
			widgets.SingleChildScrollViewOpts{Padding: painting.NewEdgeInsetsFromLTRB(effective.Left*paddingScaleFactor, top, effective.Right*paddingScaleFactor, effective.Bottom*paddingScaleFactor)}))
	}
	var column []widgets.Widget
	for _, c := range []widgets.Widget{titleWidget, contentWidget} {
		if c != nil {
			column = append(column, c)
		}
	}
	var dialogChild widgets.Widget = widgets.NewIntrinsicWidth(widgets.NewColumn(column, widgets.ColumnOpts{
		MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch,
	}), widgets.IntrinsicWidthOpts{StepWidth: foundation.Some(56.0)})
	if l, ok := label.Get(); ok {
		dialogChild = widgets.NewSemantics(dialogChild, widgets.SemanticsOpts{
			ScopesRoute: foundation.Some(true), ExplicitChildNodes: true, NamesRoute: foundation.Some(true), Label: foundation.Some(l),
		})
	}
	return NewDialog(dialogChild, DialogOpts{
		BackgroundColor: o.BackgroundColor, Elevation: o.Elevation, ShadowColor: o.ShadowColor, SurfaceTintColor: o.SurfaceTintColor,
		InsetPadding: o.InsetPadding, ClipBehavior: o.ClipBehavior, Shape: o.Shape, Alignment: o.Alignment, Constraints: o.Constraints,
	})
}

// dialogDefaultsM2 are the Material 2 defaults of the dialogs.
//
// Flutter: material/dialog.dart _DialogDefaultsM2
func dialogDefaultsM2(theme *ThemeData) DialogThemeData {
	background := Colors.White
	if theme.Brightness() == ui.BrightnessDark {
		background = Colors.Grey.Shade800()
	}
	return DialogThemeData{
		Alignment:        painting.AlignmentCenter,
		Elevation:        foundation.Some(24.0),
		Shape:            painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(4))}),
		ClipBehavior:     ui.ClipNone,
		IconColor:        theme.IconTheme.Color(),
		BackgroundColor:  foundation.Some(background),
		ShadowColor:      foundation.Some(theme.ShadowColor),
		TitleTextStyle:   theme.TextTheme.TitleLarge,
		ContentTextStyle: theme.TextTheme.TitleMedium,
		ActionsPadding:   painting.EdgeInsetsZero,
	}
}

// dialogFullscreenDefaultsM3 are the Material 3 defaults of the full screen
// dialogs.
//
// Flutter: material/dialog.dart _DialogFullscreenDefaultsM3
func dialogFullscreenDefaultsM3(theme *ThemeData) DialogThemeData {
	return DialogThemeData{
		ClipBehavior:    ui.ClipNone,
		BackgroundColor: foundation.Some(theme.ColorScheme.Surface()),
	}
}

// dialogDefaultsM3 are the Material 3 defaults of the dialogs (generated from
// the Material token database in Flutter).
//
// Flutter: material/dialog.dart _DialogDefaultsM3
func dialogDefaultsM3(theme *ThemeData) DialogThemeData {
	colors := theme.ColorScheme
	return DialogThemeData{
		Alignment:        painting.AlignmentCenter,
		Elevation:        foundation.Some(6.0),
		Shape:            painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(28))}),
		ClipBehavior:     ui.ClipNone,
		IconColor:        foundation.Some(colors.Secondary()),
		BackgroundColor:  foundation.Some(colors.SurfaceContainerHigh()),
		ShadowColor:      foundation.Some(Colors.Transparent),
		SurfaceTintColor: foundation.Some(Colors.Transparent),
		TitleTextStyle:   theme.TextTheme.HeadlineSmall,
		ContentTextStyle: theme.TextTheme.BodyMedium,
		ActionsPadding:   painting.NewEdgeInsetsFromLTRB(24, 0, 24, 24),
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDialogElevationNegative = "DialogElevationNegative"
)
