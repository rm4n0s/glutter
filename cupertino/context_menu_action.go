package cupertino

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoContextMenuActionOpts are the optional parameters of NewCupertinoContextMenuAction.
//
// Flutter: cupertino/context_menu_action.dart CupertinoContextMenuAction.new
type CupertinoContextMenuActionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// IsDefaultAction makes the text bold.
	IsDefaultAction bool
	// IsDestructiveAction makes the text red.
	IsDestructiveAction bool
	// OnPressed is called when the action is tapped; nil makes the action do nothing.
	OnPressed func()
	// TrailingIcon is an icon at the end of the action.
	TrailingIcon foundation.Opt[widgets.IconData]
}

// CupertinoContextMenuAction is an action of a CupertinoContextMenu: a row of a label and an optional icon.
//
// Flutter: cupertino/context_menu_action.dart CupertinoContextMenuAction
type CupertinoContextMenuAction struct {
	widgets.StatefulWidget
	child widgets.Widget
	o     CupertinoContextMenuActionOpts
}

// NewCupertinoContextMenuAction creates an action that shows the child.
//
// Flutter: cupertino/context_menu_action.dart CupertinoContextMenuAction.new
func NewCupertinoContextMenuAction(child widgets.Widget, opts ...CupertinoContextMenuActionOpts) *CupertinoContextMenuAction {
	o := firstOf(opts)
	return &CupertinoContextMenuAction{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child, o: o}
}

// Child is the widget of the action, usually a Text.
//
// Flutter: cupertino/context_menu_action.dart CupertinoContextMenuAction.child
func (w *CupertinoContextMenuAction) Child() widgets.Widget { return w.child }

// IsDefaultAction is whether the text is bold.
//
// Flutter: cupertino/context_menu_action.dart CupertinoContextMenuAction.isDefaultAction
func (w *CupertinoContextMenuAction) IsDefaultAction() bool { return w.o.IsDefaultAction }

// IsDestructiveAction is whether the text is red.
//
// Flutter: cupertino/context_menu_action.dart CupertinoContextMenuAction.isDestructiveAction
func (w *CupertinoContextMenuAction) IsDestructiveAction() bool { return w.o.IsDestructiveAction }

// OnPressed is called when the action is tapped.
//
// Flutter: cupertino/context_menu_action.dart CupertinoContextMenuAction.onPressed
func (w *CupertinoContextMenuAction) OnPressed() func() { return w.o.OnPressed }

// TrailingIcon is the icon at the end of the action.
//
// Flutter: cupertino/context_menu_action.dart CupertinoContextMenuAction.trailingIcon
func (w *CupertinoContextMenuAction) TrailingIcon() foundation.Opt[widgets.IconData] {
	return w.o.TrailingIcon
}

// CreateState creates the state.
//
// Flutter: cupertino/context_menu_action.dart CupertinoContextMenuAction.createState
func (w *CupertinoContextMenuAction) CreateState() widgets.State {
	return &cupertinoContextMenuActionState{}
}

const contextMenuActionButtonHeight = 43.0

var (
	contextMenuActionPressedColor = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFFDDDDDD), ui.NewColor(0xFF3F3F40))
	contextMenuActionStyle        = painting.NewTextStyle(painting.TextStyleOpts{FontFamily: CupertinoSystemTextFamily, FontFamilyFallback: cupertinoFontFamilyFallback, Inherit: foundation.Some(false),
		FontSize: foundation.Some(16.0), FontWeight: ui.FontWeightW400, Color: foundation.Some(ui.Color(CupertinoColors.Black)), TextBaseline: ui.TextBaselineAlphabetic})
)

type cupertinoContextMenuActionState struct {
	widgets.StateOf[*CupertinoContextMenuAction]
	isPressed bool
}

func (s *cupertinoContextMenuActionState) setPressed(v bool) { s.SetState(func() { s.isPressed = v }) }

func (s *cupertinoContextMenuActionState) textStyle(bc widgets.BuildContext) *painting.TextStyle {
	w := s.Widget()
	label := CupertinoDynamicColorResolve(CupertinoColors.Label, bc)
	switch {
	case w.o.IsDefaultAction:
		return contextMenuActionStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(label), FontWeight: ui.FontWeightW600})
	case w.o.IsDestructiveAction:
		return contextMenuActionStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(CupertinoColors.DestructiveRed.EffectiveColor())})
	}
	return contextMenuActionStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(label)})
}

// Build builds the action.
//
// Flutter: cupertino/context_menu_action.dart _CupertinoContextMenuActionState.build
func (s *cupertinoContextMenuActionState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	style := s.textStyle(bc)
	var cursor services.MouseCursor = services.MouseCursorDefer
	if w.o.OnPressed != nil && foundation.IsWeb {
		cursor = services.SystemMouseCursors.Click
	}
	background := CupertinoDynamicColorResolve(CupertinoContextMenuKBackgroundColor, bc)
	if s.isPressed {
		background = CupertinoDynamicColorResolve(contextMenuActionPressedColor, bc)
	}
	row := []widgets.Widget{widgets.NewFlexible(w.child)}
	if icon, ok := w.o.TrailingIcon.Get(); ok {
		row = append(row, widgets.NewIcon(icon, widgets.IconOpts{Color: style.Color, Size: foundation.Some(21.0)}))
	}
	return widgets.NewMouseRegion(
		widgets.NewGestureDetector(
			widgets.NewConstrainedBox(rendering.BoxConstraints{MinHeight: contextMenuActionButtonHeight, MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)},
				widgets.NewSemantics(widgets.NewColoredBox(background,
					widgets.NewPadding(painting.NewEdgeInsetsFromLTRB(15.5, 8.0, 17.5, 8.0),
						widgets.NewDefaultTextStyle(style, widgets.NewRow(row, widgets.RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentSpaceBetween})))),
					widgets.SemanticsOpts{Button: foundation.Some(true)})),
			widgets.GestureDetectorOpts{
				OnTapDown:   func(gestures.TapDownDetails) { s.setPressed(true) },
				OnTapUp:     func(gestures.TapUpDetails) { s.setPressed(false) },
				OnTapCancel: func() { s.setPressed(false) },
				OnTap:       w.o.OnPressed,
				Behavior:    rendering.HitTestBehaviorOpaque,
			}),
		widgets.MouseRegionOpts{Cursor: cursor})
}
