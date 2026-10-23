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
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// tooltipVisibilityScope tells the tooltips below it whether to show.
//
// Flutter: material/tooltip_visibility.dart _TooltipVisibilityScope
type tooltipVisibilityScope struct {
	widgets.InheritedWidget
	visible bool
}

func (w *tooltipVisibilityScope) UpdateShouldNotify(old widgets.Widget) bool {
	return old.(*tooltipVisibilityScope).visible != w.visible
}

// TooltipVisibility is a widget that configures whether the Tooltips below it
// show.
//
// Flutter: material/tooltip_visibility.dart TooltipVisibility
type TooltipVisibility struct {
	widgets.StatelessWidget
	visible bool
	child   widgets.Widget
}

// NewTooltipVisibility creates the widget.
//
// Flutter: material/tooltip_visibility.dart TooltipVisibility.new
func NewTooltipVisibility(visible bool, child widgets.Widget, opts ...widgets.WidgetOpts) *TooltipVisibility {
	return &TooltipVisibility{StatelessWidget: widgets.NewStatelessWidgetBase(opts...), visible: visible, child: child}
}

// Build builds the scope.
//
// Flutter: material/tooltip_visibility.dart TooltipVisibility.build
func (w *TooltipVisibility) Build(widgets.BuildContext) widgets.Widget {
	return &tooltipVisibilityScope{InheritedWidget: widgets.NewInheritedWidgetBase(w.child), visible: w.visible}
}

// TooltipVisibilityOf is whether the tooltips below the context show: true when
// there is no TooltipVisibility above it. The context depends on it.
//
// Flutter: material/tooltip_visibility.dart TooltipVisibility.of
func TooltipVisibilityOf(bc widgets.BuildContext) bool {
	if s, ok := widgets.DependOn[*tooltipVisibilityScope](bc); ok {
		return s.visible
	}
	return true
}

// TooltipOpts are the optional parameters of NewTooltip.
//
// Deviation: the deprecated height is not ported (SPEC §12.10).
//
// Flutter: material/tooltip.dart Tooltip.new
type TooltipOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Message is the text to display in the tooltip. Not both of Message and
	// RichMessage are set.
	Message string
	// RichMessage is the formatted text to display in the tooltip.
	RichMessage painting.InlineSpan
	// Constraints are the size constraints of the tooltip; the default is a
	// minimum height of 24 on the desktop and 32 on the mobile platforms.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// Padding is the padding of the message in the tooltip.
	Padding painting.EdgeInsetsGeometry
	// Margin is the empty space around the tooltip.
	Margin painting.EdgeInsetsGeometry
	// VerticalOffset is the vertical gap between the widget and the tooltip; the
	// default is 24.
	VerticalOffset foundation.Opt[float64]
	// PreferBelow is whether the tooltip is below the widget when there is room;
	// the default is true.
	PreferBelow foundation.Opt[bool]
	// ExcludeFromSemantics is whether the message is left out of the semantics
	// tree; the default is false.
	ExcludeFromSemantics foundation.Opt[bool]
	// Decoration is the decoration of the tooltip.
	Decoration painting.Decoration
	// TextStyle is the style of the message.
	TextStyle *painting.TextStyle
	// TextAlign is the alignment of the message; the default is start.
	TextAlign ui.TextAlign
	// WaitDuration is how long a pointer hovers over the widget before the
	// tooltip shows; the default is zero.
	WaitDuration foundation.Opt[time.Duration]
	// ShowDuration is how long the tooltip shows after a touch trigger; the
	// default is 1.5 seconds.
	ShowDuration foundation.Opt[time.Duration]
	// ExitDuration is how long the tooltip stays after the pointer leaves; the
	// default is 100 milliseconds.
	ExitDuration foundation.Opt[time.Duration]
	// DisableTapToDismiss stops a tap on the tooltip from dismissing it: Flutter's
	// enableTapToDismiss is false.
	DisableTapToDismiss bool
	// TriggerMode is the gesture that shows the tooltip; the default is long press.
	TriggerMode widgets.TooltipTriggerMode
	// EnableFeedback is whether the gesture gives feedback; the default is true.
	EnableFeedback foundation.Opt[bool]
	// OnTriggered is called when the tooltip is triggered by a gesture.
	OnTriggered foundation.VoidCallback
	// MouseCursor is the cursor of a pointer over the widget.
	MouseCursor services.MouseCursor
	// IgnorePointer is whether the tooltip ignores the pointer; the default is
	// true for a message and false for a rich message.
	IgnorePointer foundation.Opt[bool]
	// PositionDelegate gives the position of the tooltip.
	PositionDelegate widgets.TooltipPositionDelegate
}

// Tooltip provides text labels that help explain the function of a button or
// another user interface action.
//
// Flutter: material/tooltip.dart Tooltip
type Tooltip struct {
	widgets.StatefulWidget
	child widgets.Widget
	o     TooltipOpts
}

// NewTooltip creates a tooltip around the child (nil for none). The tag of its
// failure is TooltipMessageAndRichMessage (not both of them are set).
//
// Flutter: material/tooltip.dart Tooltip.new
func NewTooltip(child widgets.Widget, opts ...TooltipOpts) *Tooltip {
	o := firstOpts(opts)
	if foundation.DebugMode && o.Message != "" && o.RichMessage != nil {
		panic(errors.New(tagTooltipMessageAndRichMessage, "message and richMessage can't both be specified"))
	}
	return &Tooltip{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child, o: o}
}

// TooltipDismissAll dismisses all the tooltips that are showing.
//
// Flutter: material/tooltip.dart Tooltip.dismissAllToolTips
func TooltipDismissAll(bc widgets.BuildContext) bool { return widgets.RawTooltipDismissAll(bc) }

// TooltipMessage is the message of the tooltip, which gluttertest.ByTooltip finds
// it by; a tooltip with a rich message has the text of that message.
//
// Flutter: material/tooltip.dart Tooltip.message
func (w *Tooltip) TooltipMessage() string {
	if w.o.RichMessage != nil {
		return painting.InlineSpanToPlainText(w.o.RichMessage)
	}
	return w.o.Message
}

// CreateState creates the state.
//
// Flutter: material/tooltip.dart Tooltip.createState
func (w *Tooltip) CreateState() widgets.State { return &TooltipState{} }

// TooltipState is the state of a Tooltip.
//
// Flutter: material/tooltip.dart TooltipState
type TooltipState struct {
	widgets.StateOf[*Tooltip]
	tooltipKey   *widgets.GlobalKey[*widgets.RawTooltipState]
	visible      bool
	tooltipTheme TooltipThemeData
}

// InitState creates the key of the raw tooltip.
//
// Flutter: material/tooltip.dart TooltipState
func (s *TooltipState) InitState() { s.tooltipKey = widgets.NewGlobalKey[*widgets.RawTooltipState]() }

func (s *TooltipState) message() string {
	o := s.Widget().o
	if o.RichMessage != nil && o.Message == "" {
		return painting.InlineSpanToPlainText(o.RichMessage)
	}
	return o.Message
}

// EnsureTooltipVisible shows the tooltip if it is not already visible. It
// returns whether the tooltip was not visible before.
//
// Flutter: material/tooltip.dart TooltipState.ensureTooltipVisible
func (s *TooltipState) EnsureTooltipVisible() bool {
	if st, ok := s.tooltipKey.CurrentState(); ok {
		return st.EnsureTooltipVisible()
	}
	return false
}

// DidChangeDependencies reads the visibility and the theme.
//
// Flutter: material/tooltip.dart TooltipState.didChangeDependencies
func (s *TooltipState) DidChangeDependencies() {
	s.visible = TooltipVisibilityOf(s.Context())
	s.tooltipTheme = TooltipThemeOf(s.Context())
}

func isDesktopPlatform(p foundation.TargetPlatform) bool {
	return p == foundation.TargetPlatformMacOS || p == foundation.TargetPlatformLinux || p == foundation.TargetPlatformWindows
}

func (s *TooltipState) defaultPositionDelegate(c widgets.TooltipPositionContext) ui.Offset {
	o := s.Widget().o
	verticalOffset := firstOpt(o.VerticalOffset, s.tooltipTheme.VerticalOffset, foundation.Some(24.0)).Value()
	preferBelow := firstOpt(o.PreferBelow, s.tooltipTheme.PreferBelow, foundation.Some(true)).Value()
	resolved := widgets.TooltipPositionContext{
		Target: c.Target, TargetSize: c.TargetSize, TooltipSize: c.TooltipSize, OverlaySize: c.OverlaySize,
		VerticalOffset: verticalOffset, NoPreferBelow: !preferBelow,
	}
	if o.PositionDelegate != nil {
		return o.PositionDelegate(resolved)
	}
	return painting.PositionDependentBox(painting.PositionDependentBoxOpts{
		Size: c.OverlaySize, ChildSize: c.TooltipSize, Target: c.Target, VerticalOffset: verticalOffset, PreferBelow: preferBelow,
	})
}

// Build builds the tooltip.
//
// Flutter: material/tooltip.dart TooltipState.build
func (s *TooltipState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	child := w.child
	if child == nil {
		child = widgets.NewSizedBox(nil)
	}
	if s.message() == "" {
		return child
	}
	theme := ThemeOf(bc)
	fontSize := 14.0
	height, hPad := 32.0, 16.0
	if isDesktopPlatform(theme.Platform) {
		fontSize, height, hPad = 12, 24, 8
	}
	var defaultTextStyle *painting.TextStyle
	var defaultDecoration painting.Decoration
	radius := painting.NewBorderRadiusAll(ui.NewRadiusCircular(4))
	if theme.Brightness() == ui.BrightnessDark {
		defaultTextStyle = theme.TextTheme.BodyMedium.CopyWith(painting.TextStyleCopy{Color: foundation.Some(Colors.Black), FontSize: foundation.Some(fontSize)})
		defaultDecoration = painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(Colors.White.WithOpacity(0.9)), BorderRadius: radius})
	} else {
		defaultTextStyle = theme.TextTheme.BodyMedium.CopyWith(painting.TextStyleCopy{Color: foundation.Some(Colors.White), FontSize: foundation.Some(fontSize)})
		defaultDecoration = painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(Colors.Grey.Shade700().WithOpacity(0.9)), BorderRadius: radius})
	}
	defaultConstraints := rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinHeight: foundation.Some(height)})
	constraints := firstOpt(o.Constraints, s.tooltipTheme.Constraints, foundation.Some(defaultConstraints)).Value()
	textStyle := firstNonNil(o.TextStyle, s.tooltipTheme.TextStyle, defaultTextStyle)
	textAlign := firstNonZero(o.TextAlign, s.tooltipTheme.TextAlign, ui.TextAlignStart)
	decoration := firstNonNil(o.Decoration, s.tooltipTheme.Decoration, defaultDecoration)
	padding := firstNonNil(o.Padding, s.tooltipTheme.Padding, painting.EdgeInsetsGeometry(painting.NewEdgeInsetsSymmetric(4, hPad)))
	margin := firstNonNil(o.Margin, s.tooltipTheme.Margin, painting.EdgeInsetsGeometry(painting.EdgeInsetsZero))
	rich := o.RichMessage
	if rich == nil {
		rich = painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(o.Message)})
	}
	tooltipBox := newTooltipBox(constraints, textStyle, textAlign, decoration, padding, margin, rich)

	cursor := o.MouseCursor
	if cursor == nil {
		cursor = services.MouseCursorDefer
	}
	var effectiveChild widgets.Widget = widgets.NewMouseRegion(child, widgets.MouseRegionOpts{Cursor: cursor})
	excludeFromSemantics := firstOpt(o.ExcludeFromSemantics, s.tooltipTheme.ExcludeFromSemantics).OrElse(false)
	if s.visible {
		semanticsTooltip := foundation.Some(s.message())
		if excludeFromSemantics {
			semanticsTooltip = foundation.None[string]()
		}
		effectiveChild = widgets.NewRawTooltip(semanticsTooltip, func(_ widgets.BuildContext, a animation.Animation[float64]) widgets.Widget {
			return widgets.NewFadeTransition(a, tooltipBox)
		}, effectiveChild, widgets.RawTooltipOpts{
			Key:                 s.tooltipKey,
			TouchDelay:          firstOpt(o.ShowDuration, s.tooltipTheme.ShowDuration, foundation.Some(1500*time.Millisecond)),
			TriggerMode:         firstNonZero(o.TriggerMode, s.tooltipTheme.TriggerMode, widgets.TooltipTriggerModeLongPress),
			DisableFeedback:     !firstOpt(o.EnableFeedback, s.tooltipTheme.EnableFeedback, foundation.Some(true)).Value(),
			HoverDelay:          firstOpt(o.WaitDuration, s.tooltipTheme.WaitDuration, foundation.Some(time.Duration(0))).Value(),
			DisableTapToDismiss: o.DisableTapToDismiss,
			OnTriggered:         o.OnTriggered,
			DismissDelay:        firstOpt(o.ExitDuration, s.tooltipTheme.ExitDuration, foundation.Some(100*time.Millisecond)),
			PositionDelegate:    s.defaultPositionDelegate,
			IgnorePointer:       o.IgnorePointer.OrElse(o.Message != ""),
		})
	}
	return effectiveChild
}

// tooltipBox is the box of the message of a tooltip.
//
// Flutter: material/tooltip.dart _TooltipBox
type tooltipBox struct {
	widgets.StatelessWidget
	constraints rendering.BoxConstraints
	textStyle   *painting.TextStyle
	textAlign   ui.TextAlign
	decoration  painting.Decoration
	padding     painting.EdgeInsetsGeometry
	margin      painting.EdgeInsetsGeometry
	richMessage painting.InlineSpan
}

func newTooltipBox(constraints rendering.BoxConstraints, textStyle *painting.TextStyle, textAlign ui.TextAlign, decoration painting.Decoration,
	padding, margin painting.EdgeInsetsGeometry, richMessage painting.InlineSpan) *tooltipBox {
	return &tooltipBox{StatelessWidget: widgets.NewStatelessWidgetBase(), constraints: constraints, textStyle: textStyle, textAlign: textAlign,
		decoration: decoration, padding: padding, margin: margin, richMessage: richMessage}
}

func (w *tooltipBox) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewConstrainedBox(w.constraints, widgets.NewDefaultTextStyle(w.textStyle,
		widgets.NewContainer(widgets.NewCenter(
			widgets.NewTextRich(w.richMessage, widgets.TextOpts{Style: w.textStyle, TextAlign: w.textAlign}),
			widgets.CenterOpts{WidthFactor: foundation.Some(1.0), HeightFactor: foundation.Some(1.0)}),
			widgets.ContainerOpts{Decoration: w.decoration, Padding: w.padding, Margin: w.margin}),
		widgets.DefaultTextStyleOpts{TextAlign: w.textAlign}))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTooltipMessageAndRichMessage = "TooltipMessageAndRichMessage"
)
