package cupertino

import (
	"math"
	"reflect"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoActionSheetOpts are the optional parameters of NewCupertinoActionSheet.
//
// Flutter: cupertino/dialog.dart CupertinoActionSheet.new
type CupertinoActionSheetOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Title is the title, usually a Text.
	Title widgets.Widget
	// Message is the message below the title, usually a Text.
	Message widgets.Widget
	// Actions are the buttons: usually CupertinoActionSheetActions. An empty non-nil slice is a sheet that has
	// actions, with none.
	Actions []widgets.Widget
	// MessageScrollController scrolls the title and the message.
	MessageScrollController widgets.ScrollController
	// ActionScrollController scrolls the actions.
	ActionScrollController widgets.ScrollController
	// CancelButton is the button apart below the sheet, usually a CupertinoActionSheetAction.
	CancelButton widgets.Widget
}

// CupertinoActionSheet is an iOS style action sheet: a title, a message and actions in a rounded translucent
// surface above the screen's bottom edge, and a cancel button in a surface of its own.
//
// Flutter: cupertino/dialog.dart CupertinoActionSheet
type CupertinoActionSheet struct {
	widgets.StatefulWidget
	o CupertinoActionSheetOpts
}

// NewCupertinoActionSheet creates an action sheet. In debug builds it panics with the tag
// CupertinoActionSheetEmpty when it has no actions (not even an empty slice), no title, no message and no cancel
// button.
//
// Flutter: cupertino/dialog.dart CupertinoActionSheet.new
func NewCupertinoActionSheet(opts ...CupertinoActionSheetOpts) *CupertinoActionSheet {
	o := firstOf(opts)
	if foundation.DebugMode && o.Actions == nil && o.Title == nil && o.Message == nil && o.CancelButton == nil {
		panic(errors.New(tagCupertinoActionSheetEmpty, "An action sheet must have a non-null value for at least one of the following arguments: actions, title, message, or cancelButton"))
	}
	return &CupertinoActionSheet{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Title is the title of the sheet.
//
// Flutter: cupertino/dialog.dart CupertinoActionSheet.title
func (w *CupertinoActionSheet) Title() widgets.Widget { return w.o.Title }

// Message is the message of the sheet.
//
// Flutter: cupertino/dialog.dart CupertinoActionSheet.message
func (w *CupertinoActionSheet) Message() widgets.Widget { return w.o.Message }

// Actions are the buttons of the sheet.
//
// Flutter: cupertino/dialog.dart CupertinoActionSheet.actions
func (w *CupertinoActionSheet) Actions() []widgets.Widget { return w.o.Actions }

// CancelButton is the button below the sheet.
//
// Flutter: cupertino/dialog.dart CupertinoActionSheet.cancelButton
func (w *CupertinoActionSheet) CancelButton() widgets.Widget { return w.o.CancelButton }

// CreateState creates the state.
//
// Flutter: cupertino/dialog.dart CupertinoActionSheet.createState
func (w *CupertinoActionSheet) CreateState() widgets.State { return &cupertinoActionSheetState{} }

const actionSheetCancelButtonIndex = -1

type cupertinoActionSheetState struct {
	widgets.StateOf[*CupertinoActionSheet]
	pressedIndex                  foundation.Opt[int]
	backupMessageScrollController widgets.ScrollController
	backupActionScrollController  widgets.ScrollController
}

func (s *cupertinoActionSheetState) effectiveMessageScrollController() widgets.ScrollController {
	if c := s.Widget().o.MessageScrollController; c != nil {
		return c
	}
	if s.backupMessageScrollController == nil {
		s.backupMessageScrollController = widgets.NewScrollController()
	}
	return s.backupMessageScrollController
}

func (s *cupertinoActionSheetState) effectiveActionScrollController() widgets.ScrollController {
	if c := s.Widget().o.ActionScrollController; c != nil {
		return c
	}
	if s.backupActionScrollController == nil {
		s.backupActionScrollController = widgets.NewScrollController()
	}
	return s.backupActionScrollController
}

func (s *cupertinoActionSheetState) Dispose() {
	if s.backupMessageScrollController != nil {
		s.backupMessageScrollController.Dispose()
	}
	if s.backupActionScrollController != nil {
		s.backupActionScrollController.Dispose()
	}
}

func (s *cupertinoActionSheetState) hasContent() bool {
	w := s.Widget()
	return w.o.Title != nil || w.o.Message != nil
}

func (s *cupertinoActionSheetState) buildContent(bc widgets.BuildContext) widgets.Widget {
	if !s.hasContent() {
		return nil
	}
	w := s.Widget()
	textStyle := actionSheetContentStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(CupertinoDynamicColorResolve(actionSheetContentTextColor, bc))})
	bold := textStyle.CopyWith(painting.TextStyleCopy{FontWeight: ui.FontWeightW600})
	titleBottom, messageTop := 0.0, 0.0
	if w.o.Message == nil {
		titleBottom = actionSheetContentVerticalPadding
	}
	if w.o.Title == nil {
		messageTop = actionSheetContentVerticalPadding
	}
	titleStyle, messageStyle := textStyle, textStyle
	if w.o.Message != nil {
		titleStyle = bold
	}
	if w.o.Title == nil {
		messageStyle = bold
	}
	return widgets.NewColoredBox(CupertinoDynamicColorResolve(actionSheetBackgroundColor, bc), &alertContentSection{
		StatelessWidget:  widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}),
		title:            w.o.Title,
		message:          w.o.Message,
		scrollController: s.effectiveMessageScrollController(),
		titlePadding: foundation.Some(painting.EdgeInsets{Left: actionSheetContentHorizontalPadding, Right: actionSheetContentHorizontalPadding,
			Bottom: titleBottom, Top: actionSheetContentVerticalPadding}),
		messagePadding: foundation.Some(painting.EdgeInsets{Left: actionSheetContentHorizontalPadding, Right: actionSheetContentHorizontalPadding,
			Bottom: actionSheetContentVerticalPadding, Top: messageTop}),
		titleStyle:                       titleStyle,
		messageStyle:                     messageStyle,
		additionalPaddingTitleAndMessage: foundation.Some(painting.EdgeInsets{Top: 4}),
	})
}

func (s *cupertinoActionSheetState) onPressedUpdate(actionIndex int, state bool) {
	if !state {
		if p, ok := s.pressedIndex.Get(); ok && p == actionIndex {
			s.SetState(func() { s.pressedIndex = foundation.None[int]() })
		}
		return
	}
	s.SetState(func() { s.pressedIndex = foundation.Some(actionIndex) })
}

func (s *cupertinoActionSheetState) buildCancelButton() widgets.Widget {
	w := s.Widget()
	cancelPadding := 0.0
	if w.o.Actions != nil || w.o.Message != nil || w.o.Title != nil {
		cancelPadding = actionSheetCancelButtonPadding
	}
	p, pressed := s.pressedIndex.Get()
	return widgets.NewPadding(painting.EdgeInsets{Top: cancelPadding},
		NewCupertinoFocusHaloWithRRect(CupertinoButtonSizeBorderRadius(CupertinoButtonSizeLarge),
			&actionSheetButtonBackground{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{}), isCancel: true,
				pressed: pressed && p == actionSheetCancelButtonIndex, child: w.o.CancelButton,
				onPressStateChange: func(state bool) { s.onPressedUpdate(actionSheetCancelButtonIndex, state) }}))
}

// actionSheetLerp is the linear interpolation of y between (x1, y1) and (x2, y2), clamped to them.
//
// Flutter: cupertino/dialog.dart _CupertinoActionSheetState._lerp
func actionSheetLerp(x, x1, y1, x2, y2 float64) float64 {
	switch {
	case x <= x1:
		return y1
	case x >= x2:
		return y2
	}
	return ui.LerpDouble(y1, y2, (x-x1)/(x2-x1))
}

// topPadding is the space above the sheet: the status bar's, scaled as iOS does.
//
// Flutter: cupertino/dialog.dart _CupertinoActionSheetState._topPadding
func actionSheetTopPadding(bc widgets.BuildContext) float64 {
	if widgets.MediaQueryOrientationOf(bc) == widgets.OrientationLandscape {
		return actionSheetEdgePadding
	}
	const (
		viewPaddingData1  = 47.0
		paddingRatioData1 = 1.0
		viewPaddingData2  = 59.0
		paddingRatioData2 = 54.0 / 59.0
	)
	currentViewPadding := widgets.MediaQueryViewPaddingOf(bc).Top
	currentPaddingRatio := actionSheetLerp(currentViewPadding, viewPaddingData1, paddingRatioData1, viewPaddingData2, paddingRatioData2)
	padding := math.Round(currentPaddingRatio * currentViewPadding)
	return math.Max(padding, dialogEdgePadding)
}

func (s *cupertinoActionSheetState) Build(bc widgets.BuildContext) widgets.Widget {
	widgets.DebugCheckHasMediaQuery(bc)
	w := s.Widget()
	var actions []widgets.Widget
	actions = append(actions, w.o.Actions...)
	children := []widgets.Widget{
		widgets.NewFlexible(widgets.NewClipRSuperellipse(
			widgets.NewBackdropFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: CupertinoPopupSurfaceDefaultBlurSigma, SigmaY: CupertinoPopupSurfaceDefaultBlurSigma}),
				&actionSheetMainSheet{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}), pressedIndex: s.pressedIndex,
					onPressedUpdate: s.onPressedUpdate, scrollController: s.effectiveActionScrollController(), contentSection: s.buildContent(bc),
					actions: actions, dividerColor: CupertinoDynamicColorResolve(actionSheetButtonDividerColor, bc)}),
			widgets.ClipRSuperellipseOpts{BorderRadius: painting.NewBorderRadiusCircular(12)})),
	}
	if w.o.CancelButton != nil {
		children = append(children, s.buildCancelButton())
	}
	actionSheetWidth := widgets.MediaQueryWidthOf(bc)
	if widgets.MediaQueryOrientationOf(bc) == widgets.OrientationLandscape {
		actionSheetWidth = widgets.MediaQueryHeightOf(bc)
	}
	return widgets.NewSafeArea(
		widgets.NewScrollConfiguration(widgets.ScrollConfigurationOf(bc).CopyWith(widgets.ScrollBehaviorCopy{Scrollbars: foundation.Some(false)}),
			widgets.NewSemantics(
				NewCupertinoUserInterfaceLevel(CupertinoUserInterfaceLevelElevated,
					widgets.NewPadding(painting.EdgeInsets{Left: actionSheetEdgePadding, Right: actionSheetEdgePadding, Top: actionSheetTopPadding(bc)},
						widgets.NewSizedBox(newActionSheetGestureDetector(widgets.NewSemantics(
							widgets.NewColumn(children, widgets.ColumnOpts{MainAxisAlignment: rendering.MainAxisAlignmentEnd,
								MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch}),
							widgets.SemanticsOpts{ExplicitChildNodes: true})),
							widgets.SizedBoxOpts{Width: foundation.Some(actionSheetWidth - actionSheetEdgePadding*2)}))),
				widgets.SemanticsOpts{NamesRoute: foundation.Some(true), ScopesRoute: foundation.Some(true), ExplicitChildNodes: true,
					Role: foundation.Some(ui.SemanticsRoleDialog), Label: foundation.Some("Alert")})),
		widgets.SafeAreaOpts{Minimum: painting.EdgeInsets{Bottom: actionSheetEdgePadding}})
}

// CupertinoActionSheetActionOpts are the optional parameters of NewCupertinoActionSheetAction.
//
// Flutter: cupertino/dialog.dart CupertinoActionSheetAction.new
type CupertinoActionSheetActionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// IsDefaultAction makes the text bold.
	IsDefaultAction bool
	// IsDestructiveAction makes the text red.
	IsDestructiveAction bool
	// MouseCursor is the cursor over the action; the default is the click cursor on the web.
	MouseCursor services.MouseCursor
	// FocusNode is the focus node of the action.
	FocusNode *widgets.FocusNode
	// FocusColor is the color of the focus highlight; the default is the active blue.
	FocusColor ui.ColorValue
}

// CupertinoActionSheetAction is a button of a CupertinoActionSheet.
//
// Flutter: cupertino/dialog.dart CupertinoActionSheetAction
type CupertinoActionSheetAction struct {
	widgets.StatefulWidget
	onPressed func()
	child     widgets.Widget
	o         CupertinoActionSheetActionOpts
}

// NewCupertinoActionSheetAction creates a button that calls onPressed (nil does nothing) and shows the child.
//
// Flutter: cupertino/dialog.dart CupertinoActionSheetAction.new
func NewCupertinoActionSheetAction(onPressed func(), child widgets.Widget, opts ...CupertinoActionSheetActionOpts) *CupertinoActionSheetAction {
	o := firstOf(opts)
	return &CupertinoActionSheetAction{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), onPressed: onPressed, child: child, o: o}
}

// OnPressed is called when the action is tapped.
//
// Flutter: cupertino/dialog.dart CupertinoActionSheetAction.onPressed
func (w *CupertinoActionSheetAction) OnPressed() func() { return w.onPressed }

// IsDefaultAction is whether the text is bold.
//
// Flutter: cupertino/dialog.dart CupertinoActionSheetAction.isDefaultAction
func (w *CupertinoActionSheetAction) IsDefaultAction() bool { return w.o.IsDefaultAction }

// IsDestructiveAction is whether the text is red.
//
// Flutter: cupertino/dialog.dart CupertinoActionSheetAction.isDestructiveAction
func (w *CupertinoActionSheetAction) IsDestructiveAction() bool { return w.o.IsDestructiveAction }

// Child is the widget that the action shows.
//
// Flutter: cupertino/dialog.dart CupertinoActionSheetAction.child
func (w *CupertinoActionSheetAction) Child() widgets.Widget { return w.child }

// CreateState creates the state.
//
// Flutter: cupertino/dialog.dart CupertinoActionSheetAction.createState
func (w *CupertinoActionSheetAction) CreateState() widgets.State {
	return &cupertinoActionSheetActionState{}
}

type cupertinoActionSheetActionState struct {
	widgets.StateOf[*CupertinoActionSheetAction]
	showHighlight bool
}

func (s *cupertinoActionSheetActionState) didEnter(_, innerEnabled bool) bool { return innerEnabled }
func (s *cupertinoActionSheetActionState) didLeave()                          {}
func (s *cupertinoActionSheetActionState) didConfirm()                        { s.press() }

// press calls onPressed. Deviation: Flutter's onPressed is required and never null; a nil one here does nothing.
func (s *cupertinoActionSheetActionState) press() {
	if f := s.Widget().onPressed; f != nil {
		f()
	}
}

func (s *cupertinoActionSheetActionState) handleTap() {
	s.press()
	s.Context().FindRenderObject().SendSemanticsEvent(semantics.NewTapSemanticEvent())
}

func (s *cupertinoActionSheetActionState) effectiveFocusBackgroundColor(bc widgets.BuildContext) ui.Color {
	focus := ui.ColorValue(CupertinoColors.ActiveBlue)
	if c := s.Widget().o.FocusColor; c != nil {
		focus = c
	}
	opacity := CupertinoButtonTintedOpacityDark
	if CupertinoThemeBrightnessOf(bc) == ui.BrightnessLight {
		opacity = CupertinoButtonTintedOpacityLight
	}
	return painting.NewHSLColorFromColor(focus.EffectiveColor().WithOpacity(opacity)).ToColor()
}

func (s *cupertinoActionSheetActionState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	cursor := w.o.MouseCursor
	if cursor == nil {
		cursor = services.MouseCursorDefer
		if foundation.IsWeb {
			cursor = services.SystemMouseCursors.Click
		}
	}
	var content widgets.Widget = &actionSheetActionContent{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}),
		isDestructiveAction: w.o.IsDestructiveAction, isDefaultAction: w.o.IsDefaultAction, child: w.child}
	if s.showHighlight {
		content = widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(s.effectiveFocusBackgroundColor(bc))}), content)
	}
	return widgets.NewMouseRegion(widgets.NewMetaData(widgets.NewConstrainedBox(
		rendering.BoxConstraints{MinHeight: actionSheetButtonMinHeight, MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)},
		widgets.NewFocusableActionDetector(widgets.NewSemantics(content, widgets.SemanticsOpts{Button: foundation.Some(true), OnTap: w.onPressed}),
			widgets.FocusableActionDetectorOpts{
				Actions:              widgets.ActionMap{reflect.TypeFor[widgets.ActivateIntent](): widgets.NewCallbackAction(func(widgets.ActivateIntent) any { s.handleTap(); return nil })},
				FocusNode:            w.o.FocusNode,
				OnShowFocusHighlight: func(show bool) { s.SetState(func() { s.showHighlight = show }) },
			})),
		widgets.MetaDataOpts{MetaData: s, Behavior: rendering.HitTestBehaviorOpaque}),
		widgets.MouseRegionOpts{Cursor: cursor})
}

// actionSheetActionContent is the padded, centered text of an action: its size follows the text scale as the
// iOS sizes do.
//
// Flutter: cupertino/dialog.dart _ActionSheetActionContent
type actionSheetActionContent struct {
	widgets.StatelessWidget
	isDestructiveAction, isDefaultAction bool
	child                                widgets.Widget
}

func actionSheetButtonFontSize(contextBodySize float64) float64 {
	switch {
	case contextBodySize <= 17:
		return 21.0
	case contextBodySize <= 19:
		return ui.LerpDouble(21.0, 23.0, (contextBodySize-17.0)/(19.0-17.0))
	case contextBodySize <= 21:
		return ui.LerpDouble(23.0, 24.0, (contextBodySize-19.0)/(21.0-19.0))
	case contextBodySize <= 24:
		return 24.0
	}
	return contextBodySize
}

func (w *actionSheetActionContent) Build(bc widgets.BuildContext) widgets.Widget {
	const higLargeBodySize = 17.0
	contextBodySize := widgets.MediaQueryTextScalerOf(bc).Scale(higLargeBodySize)
	contextScaleFactor := contextBodySize / higLargeBodySize
	fontSize := actionSheetButtonFontSize(contextBodySize)
	var color ui.Color
	if w.isDestructiveAction {
		color = CupertinoDynamicColorResolve(CupertinoColors.SystemRed, bc)
	} else {
		color = CupertinoDynamicColorResolve(CupertinoThemeOf(bc).PrimaryColor(), bc)
	}
	style := actionSheetActionStyle.CopyWith(painting.TextStyleCopy{FontSize: foundation.Some(fontSize / contextScaleFactor), Color: foundation.Some(color)})
	if w.isDefaultAction {
		style = style.CopyWith(painting.TextStyleCopy{FontWeight: ui.FontWeightW600})
	}
	verticalPadding := actionSheetButtonVerticalPaddingBase + fontSize*actionSheetButtonVerticalPaddingFacto
	return widgets.NewPadding(painting.NewEdgeInsetsFromLTRB(actionSheetButtonHorizontalPadding, verticalPadding, actionSheetButtonHorizontalPadding, verticalPadding),
		widgets.NewDefaultTextStyle(style, widgets.NewCenter(w.child), widgets.DefaultTextStyleOpts{TextAlign: ui.TextAlignCenter}))
}

// actionSheetButtonBackground is the background of a button of an action sheet: it is a slide target.
//
// Flutter: cupertino/dialog.dart _ActionSheetButtonBackground
type actionSheetButtonBackground struct {
	widgets.StatefulWidget
	isCancel           bool
	pressed            bool
	onPressStateChange func(bool)
	child              widgets.Widget
}

func (w *actionSheetButtonBackground) CreateState() widgets.State {
	return &actionSheetButtonBackgroundState{}
}

type actionSheetButtonBackgroundState struct {
	widgets.StateOf[*actionSheetButtonBackground]
}

func (s *actionSheetButtonBackgroundState) didEnter(fromPointerDown, innerEnabled bool) bool {
	if f := s.Widget().onPressStateChange; f != nil {
		f(true)
	}
	if !fromPointerDown {
		emitSelectionClick(s.Context())
	}
	return innerEnabled
}

func (s *actionSheetButtonBackgroundState) didLeave() {
	if f := s.Widget().onPressStateChange; f != nil {
		f(false)
	}
}

func (s *actionSheetButtonBackgroundState) didConfirm() { s.didLeave() }

func (s *actionSheetButtonBackgroundState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	var child widgets.Widget
	if !w.isCancel {
		c := ui.ColorValue(actionSheetBackgroundColor)
		if w.pressed {
			c = actionSheetPressedColor
		}
		child = widgets.NewColoredBox(CupertinoDynamicColorResolve(c, bc), w.child)
	} else {
		c := ui.ColorValue(actionSheetCancelColor)
		if w.pressed {
			c = actionSheetCancelPressedColor
		}
		child = widgets.NewClipRSuperellipse(widgets.NewDecoratedBox(
			painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(CupertinoDynamicColorResolve(c, bc))}), w.child),
			widgets.ClipRSuperellipseOpts{BorderRadius: painting.NewBorderRadiusCircular(dialogCornerRadius)})
	}
	return widgets.NewMetaData(child, widgets.MetaDataOpts{MetaData: s})
}

// actionSheetActionSection is the actions of an action sheet, in a scroll view, with dividers between them.
//
// Flutter: cupertino/dialog.dart _ActionSheetActionSection
type actionSheetActionSection struct {
	widgets.StatelessWidget
	actions          []widgets.Widget
	onPressedUpdate  pressedUpdateHandler
	pressedIndex     foundation.Opt[int]
	dividerColor     ui.Color
	scrollController widgets.ScrollController
}

func (w *actionSheetActionSection) Build(bc widgets.BuildContext) widgets.Widget {
	if len(w.actions) == 0 {
		return widgets.NewLimitedBox(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(math.Inf(1)), Height: foundation.Some(0.0)}),
			widgets.LimitedBoxOpts{MaxWidth: foundation.Some(0.0)})
	}
	hidden := CupertinoDynamicColorResolve(actionSheetBackgroundColor, bc)
	isPressed := func(i int) bool { p, ok := w.pressedIndex.Get(); return ok && p == i }
	var column []widgets.Widget
	for i, action := range w.actions {
		if i != 0 {
			column = append(column, newDialogDivider(w.dividerColor, hidden, isPressed(i-1) || isPressed(i)))
		}
		index := i
		column = append(column, &actionSheetButtonBackground{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{}),
			pressed: isPressed(i), child: action, onPressStateChange: func(state bool) { w.onPressedUpdate(index, state) }})
	}
	return NewCupertinoScrollbar(widgets.NewSingleChildScrollView(
		widgets.NewColumn(column, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch}),
		widgets.SingleChildScrollViewOpts{Controller: w.scrollController}), CupertinoScrollbarOpts{Controller: w.scrollController})
}

// actionSheetMainSheet is the content and the actions of an action sheet: the actions have the priority when the
// sheet is too high.
//
// Flutter: cupertino/dialog.dart _ActionSheetMainSheet
type actionSheetMainSheet struct {
	widgets.StatelessWidget
	pressedIndex     foundation.Opt[int]
	onPressedUpdate  pressedUpdateHandler
	scrollController widgets.ScrollController
	actions          []widgets.Widget
	contentSection   widgets.Widget
	dividerColor     ui.Color
}

func (w *actionSheetMainSheet) scrolledActionsSection(bc widgets.BuildContext) widgets.Widget {
	backgroundColor := CupertinoDynamicColorResolve(actionSheetBackgroundColor, bc)
	return newOverscrollBackground(backgroundColor, NewCupertinoFocusHaloWithRRect(
		CupertinoButtonSizeBorderRadius(CupertinoButtonSizeLarge).CopyWith(painting.BorderRadiusCopy{
			TopLeft: foundation.Some(ui.Radius{}), TopRight: foundation.Some(ui.Radius{})}),
		&actionSheetActionSection{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}), actions: w.actions,
			scrollController: w.scrollController, dividerColor: w.dividerColor, pressedIndex: w.pressedIndex, onPressedUpdate: w.onPressedUpdate}))
}

func (w *actionSheetMainSheet) dividerAndActionsSection(bc widgets.BuildContext) widgets.Widget {
	backgroundColor := CupertinoDynamicColorResolve(actionSheetBackgroundColor, bc)
	return widgets.NewColumn([]widgets.Widget{
		newDialogDivider(w.dividerColor, backgroundColor, false),
		widgets.NewFlexible(w.scrolledActionsSection(bc)),
	}, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
}

func (w *actionSheetMainSheet) Build(bc widgets.BuildContext) widgets.Widget {
	if len(w.actions) == 0 {
		if w.contentSection != nil {
			return w.contentSection
		}
		return widgets.NewLimitedBox(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(math.Inf(1)), Height: foundation.Some(0.0)}),
			widgets.LimitedBoxOpts{MaxWidth: foundation.Some(0.0)})
	}
	if w.contentSection == nil {
		return w.scrolledActionsSection(bc)
	}
	return newPriorityColumn(w.contentSection, w.dividerAndActionsSection(bc), actionSheetActionsSectionMinHeight+dialogDividerThickness)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagCupertinoActionSheetEmpty = "CupertinoActionSheetEmpty"
