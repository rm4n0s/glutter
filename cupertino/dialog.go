package cupertino

import (
	"math"
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

func dialogTextStyle(family string, size, weight float64, height, letterSpacing foundation.Opt[float64]) *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{FontFamily: family, FontFamilyFallback: cupertinoFontFamilyFallback, Inherit: foundation.Some(false), FontSize: foundation.Some(size),
		FontWeight: ui.FontWeight(int(weight)), Height: height, LetterSpacing: letterSpacing, TextBaseline: ui.TextBaselineAlphabetic})
}

var (
	dialogTitleStyle   = dialogTextStyle(CupertinoSystemTextFamily, 17, 600, foundation.Some(1.3), foundation.Some(-0.5))
	dialogContentStyle = dialogTextStyle(CupertinoSystemTextFamily, 13, 400, foundation.Some(1.35), foundation.Some(-0.2))
	dialogActionStyle  = dialogTextStyle(CupertinoSystemTextFamily, 16.8, 400, foundation.None[float64](), foundation.None[float64]())
	// actionSheetActionStyle is the style of the text of an action of an action sheet.
	actionSheetActionStyle  = dialogTextStyle(CupertinoSystemDisplayFamily, 17, 400, foundation.None[float64](), foundation.None[float64]())
	actionSheetContentStyle = dialogTextStyle(CupertinoSystemTextFamily, 13, 400, foundation.None[float64](), foundation.None[float64]())
)

const (
	dialogCornerRadius                    = 14.0
	dialogDividerThickness                = 0.3
	dialogWidth                           = 270.0
	dialogAccessibilityWidth              = 310.0
	dialogEdgePadding                     = 20.0
	dialogMinButtonHeight                 = 45.0
	dialogMinButtonFontSize               = 10.0
	dialogActionsSectionMinHeight         = 67.8
	actionSheetEdgePadding                = 8.0
	actionSheetCancelButtonPadding        = 8.0
	actionSheetContentHorizontalPadding   = 16.0
	actionSheetContentVerticalPadding     = 13.5
	actionSheetActionsSectionMinHeight    = 84.0
	actionSheetButtonHorizontalPadding    = 10.0
	actionSheetButtonMinHeight            = 57.17
	actionSheetButtonVerticalPaddingFacto = 0.4
	actionSheetButtonVerticalPaddingBase  = 1.8
	maxRegularTextScaleFactor             = 1.4
)

var (
	dialogColor                    = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xCCF2F2F2), ui.NewColor(0xCC2D2D2D))
	dialogPressedColor             = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFFE1E1E1), ui.NewColor(0xFF404040))
	actionSheetPressedColor        = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xCAE0E0E0), ui.NewColor(0xC1515151))
	actionSheetCancelColor         = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF2C2C2C))
	actionSheetCancelPressedColor  = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFFECECEC), ui.NewColor(0xFF494949))
	actionSheetBackgroundColor     = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xC8FCFCFC), ui.NewColor(0xBE292929))
	actionSheetContentTextColor    = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0x851D1D1D), ui.NewColor(0x96F1F1F1))
	actionSheetButtonDividerColor  = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xD4C9C9C9), ui.NewColor(0xD57D7D7D))
	popupSurfaceLightSaturationMat = [20]float64{1.74, -0.40, -0.17, 0, 0, -0.26, 1.60, -0.17, 0, 0, -0.26, -0.40, 1.83, 0, 0, 0, 0, 0, 1, 0}
	popupSurfaceDarkSaturationMat  = [20]float64{1.39, -0.56, -0.11, 0, 0.30, -0.32, 1.14, -0.11, 0, 0.30, -0.32, -0.56, 1.59, 0, 0.30, 0, 0, 0, 1, 0}
)

// isInAccessibilityMode is whether the text scale is the one of the accessibility sizes: more than 1.4.
//
// Flutter: cupertino/dialog.dart _isInAccessibilityMode
func isInAccessibilityMode(bc widgets.BuildContext) bool {
	const defaultFontSize = 14.0
	scaler, ok := widgets.MediaQueryMaybeTextScalerOf(bc)
	return ok && scaler.Scale(defaultFontSize) > defaultFontSize*maxRegularTextScaleFactor
}

// CupertinoAlertDialogOpts are the optional parameters of NewCupertinoAlertDialog.
//
// Flutter: cupertino/dialog.dart CupertinoAlertDialog.new
type CupertinoAlertDialogOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Title is the title of the dialog, usually a Text.
	Title widgets.Widget
	// Content is the content below the title, usually a Text.
	Content widgets.Widget
	// Actions are the buttons at the bottom: usually CupertinoDialogActions.
	Actions []widgets.Widget
	// ScrollController scrolls the title and the content.
	ScrollController widgets.ScrollController
	// ActionScrollController scrolls the actions.
	ActionScrollController widgets.ScrollController
	// InsetAnimationDuration is how long the dialog takes to move away from the keyboard; the default is 100 ms.
	InsetAnimationDuration foundation.Opt[time.Duration]
	// InsetAnimationCurve is the curve of that animation; the default is Curves.Decelerate.
	InsetAnimationCurve animation.Curve
}

// CupertinoAlertDialog is an iOS style alert dialog: a title, a content and actions, in a translucent rounded
// surface. Two actions that fit are side by side.
//
// Flutter: cupertino/dialog.dart CupertinoAlertDialog
type CupertinoAlertDialog struct {
	widgets.StatefulWidget
	o CupertinoAlertDialogOpts
}

// NewCupertinoAlertDialog creates an alert dialog.
//
// Flutter: cupertino/dialog.dart CupertinoAlertDialog.new
func NewCupertinoAlertDialog(opts ...CupertinoAlertDialogOpts) *CupertinoAlertDialog {
	o := firstOf(opts)
	if o.InsetAnimationCurve == nil {
		o.InsetAnimationCurve = animation.Curves.Decelerate
	}
	return &CupertinoAlertDialog{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Title is the title of the dialog.
//
// Flutter: cupertino/dialog.dart CupertinoAlertDialog.title
func (w *CupertinoAlertDialog) Title() widgets.Widget { return w.o.Title }

// Content is the content of the dialog.
//
// Flutter: cupertino/dialog.dart CupertinoAlertDialog.content
func (w *CupertinoAlertDialog) Content() widgets.Widget { return w.o.Content }

// Actions are the buttons of the dialog.
//
// Flutter: cupertino/dialog.dart CupertinoAlertDialog.actions
func (w *CupertinoAlertDialog) Actions() []widgets.Widget { return w.o.Actions }

// CreateState creates the state.
//
// Flutter: cupertino/dialog.dart CupertinoAlertDialog.createState
func (w *CupertinoAlertDialog) CreateState() widgets.State { return &cupertinoAlertDialogState{} }

type cupertinoAlertDialogState struct {
	widgets.StateOf[*CupertinoAlertDialog]
	pressedIndex                 foundation.Opt[int]
	backupScrollController       widgets.ScrollController
	backupActionScrollController widgets.ScrollController
}

func (s *cupertinoAlertDialogState) effectiveScrollController() widgets.ScrollController {
	if c := s.Widget().o.ScrollController; c != nil {
		return c
	}
	if s.backupScrollController == nil {
		s.backupScrollController = widgets.NewScrollController()
	}
	return s.backupScrollController
}

func (s *cupertinoAlertDialogState) effectiveActionScrollController() widgets.ScrollController {
	if c := s.Widget().o.ActionScrollController; c != nil {
		return c
	}
	if s.backupActionScrollController == nil {
		s.backupActionScrollController = widgets.NewScrollController()
	}
	return s.backupActionScrollController
}

func (s *cupertinoAlertDialogState) Dispose() {
	if s.backupScrollController != nil {
		s.backupScrollController.Dispose()
	}
	if s.backupActionScrollController != nil {
		s.backupActionScrollController.Dispose()
	}
}

func (s *cupertinoAlertDialogState) buildContent(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	if w.o.Title == nil && w.o.Content == nil {
		return nil
	}
	const defaultFontSize = 14.0
	effectiveTextScaleFactor := widgets.MediaQueryTextScalerOf(bc).Scale(defaultFontSize) / defaultFontSize
	label := CupertinoDynamicColorResolve(CupertinoColors.Label, bc)
	titleBottom, messageTop := 1.0, 1.0
	if w.o.Content == nil {
		titleBottom = dialogEdgePadding
	}
	if w.o.Title == nil {
		messageTop = dialogEdgePadding
	}
	child := &alertContentSection{
		StatelessWidget:  widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}),
		title:            w.o.Title,
		message:          w.o.Content,
		scrollController: s.effectiveScrollController(),
		titlePadding:     foundation.Some(painting.EdgeInsets{Left: dialogEdgePadding, Right: dialogEdgePadding, Bottom: titleBottom, Top: dialogEdgePadding * effectiveTextScaleFactor}),
		messagePadding:   foundation.Some(painting.EdgeInsets{Left: dialogEdgePadding, Right: dialogEdgePadding, Bottom: dialogEdgePadding * effectiveTextScaleFactor, Top: messageTop}),
		titleStyle:       dialogTitleStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(label)}),
		messageStyle:     dialogContentStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(label)}),
	}
	return widgets.NewColoredBox(CupertinoDynamicColorResolve(dialogColor, bc), child)
}

func (s *cupertinoAlertDialogState) onPressedUpdate(actionIndex int, isPressed bool) {
	if isPressed {
		s.SetState(func() { s.pressedIndex = foundation.Some(actionIndex) })
	} else if p, ok := s.pressedIndex.Get(); ok && p == actionIndex {
		s.SetState(func() { s.pressedIndex = foundation.None[int]() })
	}
}

func (s *cupertinoAlertDialogState) buildActions() widgets.Widget {
	w := s.Widget()
	if len(w.o.Actions) == 0 {
		return nil
	}
	return newAlertActionSection(w.o.Actions, s.pressedIndex, s.onPressedUpdate, s.effectiveActionScrollController())
}

func (s *cupertinoAlertDialogState) buildBody(bc widgets.BuildContext) widgets.Widget {
	backgroundColor := CupertinoDynamicColorResolve(dialogColor, bc)
	dividerColor := CupertinoDynamicColorResolve(CupertinoColors.Separator, bc)
	return widgets.NewMediaQueryRemovePadding(bc, widgets.RemoveSides{Left: true, Top: true, Right: true, Bottom: true},
		widgets.NewLayoutBuilder(func(bc widgets.BuildContext, constraints rendering.BoxConstraints) widgets.Widget {
			contentSection := s.buildContent(bc)
			actionsSection := s.buildActions()
			if actionsSection == nil {
				if contentSection != nil {
					return contentSection
				}
				return widgets.NewLimitedBox(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(math.Inf(1)), Height: foundation.Some(0.0)}),
					widgets.LimitedBoxOpts{MaxWidth: foundation.Some(0.0)})
			}
			scrolledActionsSection := newOverscrollBackground(backgroundColor, actionsSection)
			if contentSection == nil {
				return scrolledActionsSection
			}
			actionsMinHeight := dialogActionsSectionMinHeight + dialogDividerThickness
			if isInAccessibilityMode(bc) {
				actionsMinHeight = constraints.MaxHeight/2 + dialogDividerThickness
			}
			return newPriorityColumn(contentSection, widgets.NewColumn([]widgets.Widget{
				widgets.NewSizedBox(newDialogDivider(dividerColor, backgroundColor, false), widgets.SizedBoxOpts{Width: foundation.Some(math.Inf(1))}),
				widgets.NewFlexible(scrolledActionsSection),
			}), actionsMinHeight)
		}))
}

func (s *cupertinoAlertDialogState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	localizations := CupertinoLocalizationsOf(bc)
	inAccessibilityMode := isInAccessibilityMode(bc)
	width := dialogWidth
	if inAccessibilityMode {
		width = dialogAccessibilityWidth
	}
	return NewCupertinoUserInterfaceLevel(CupertinoUserInterfaceLevelElevated,
		widgets.NewMediaQueryWithClampedTextScaling(
			widgets.NewScrollConfiguration(widgets.ScrollConfigurationOf(bc).CopyWith(widgets.ScrollBehaviorCopy{Scrollbars: foundation.Some(false)}),
				widgets.NewLayoutBuilder(func(bc widgets.BuildContext, _ rendering.BoxConstraints) widgets.Widget {
					insets := widgets.MediaQueryViewInsetsOf(bc).Plus(painting.EdgeInsets{Left: 40, Right: 40, Top: 24, Bottom: 24})
					duration := w.o.InsetAnimationDuration.OrElse(100 * time.Millisecond)
					return widgets.NewAnimatedPadding(insets, duration,
						widgets.NewMediaQueryRemoveViewInsets(bc, widgets.RemoveSides{Left: true, Top: true, Right: true, Bottom: true},
							widgets.NewCenter(widgets.NewPadding(painting.NewEdgeInsetsSymmetric(dialogEdgePadding, 0),
								widgets.NewSizedBox(newActionSheetGestureDetector(
									NewCupertinoPopupSurface(widgets.NewSemantics(s.buildBody(bc), widgets.SemanticsOpts{
										Role: foundation.Some(ui.SemanticsRoleAlertDialog), NamesRoute: foundation.Some(true), ScopesRoute: foundation.Some(true),
										ExplicitChildNodes: true, Label: foundation.Some(localizations.AlertDialogLabel())}),
										CupertinoPopupSurfaceOpts{NotSurfacePainted: true})),
									widgets.SizedBoxOpts{Width: foundation.Some(width)})))),
						widgets.AnimatedPaddingOpts{Curve: w.o.InsetAnimationCurve})
				})), widgets.MediaQueryClampedTextScalingOpts{MinScaleFactor: 1.0}))
}

// CupertinoPopupSurfaceDefaultBlurSigma is the blur of the surface behind a popup: 30.
//
// Flutter: cupertino/dialog.dart CupertinoPopupSurface.defaultBlurSigma
const CupertinoPopupSurfaceDefaultBlurSigma = 30.0

// CupertinoPopupSurfaceOpts are the optional parameters of NewCupertinoPopupSurface.
//
// Flutter: cupertino/dialog.dart CupertinoPopupSurface.new
type CupertinoPopupSurfaceOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// BlurSigma is the blur of what is behind the surface; the default is CupertinoPopupSurfaceDefaultBlurSigma,
	// and zero is no blur.
	BlurSigma foundation.Opt[float64]
	// NotSurfacePainted leaves the surface with no color of its own (Flutter's isSurfacePainted is true by
	// default): the child paints it.
	NotSurfacePainted bool
}

// CupertinoPopupSurface is the surface of a popup: it blurs and boosts the saturation of what is behind it, and
// has a color (unless NotSurfacePainted) and the rounded corners of iOS.
//
// Flutter: cupertino/dialog.dart CupertinoPopupSurface
type CupertinoPopupSurface struct {
	widgets.StatelessWidget
	child widgets.Widget
	o     CupertinoPopupSurfaceOpts
}

// NewCupertinoPopupSurface creates a surface around the child. In debug builds it panics with the tag
// CupertinoPopupSurfaceBlurSigmaNegative when the blur is negative.
//
// Deviation: Flutter's static debugIsVibrancePainted, which the tests set to turn the saturation off, is not ported
// (no global mutable state, SPEC §12.10).
//
// Flutter: cupertino/dialog.dart CupertinoPopupSurface.new
func NewCupertinoPopupSurface(child widgets.Widget, opts ...CupertinoPopupSurfaceOpts) *CupertinoPopupSurface {
	o := firstOf(opts)
	if foundation.DebugMode && o.BlurSigma.OrElse(CupertinoPopupSurfaceDefaultBlurSigma) < 0 {
		panic(errors.New(tagCupertinoPopupSurfaceBlurSigmaNegative, "CupertinoPopupSurface requires a non-negative blur sigma.", "blurSigma", o.BlurSigma.Value()))
	}
	return &CupertinoPopupSurface{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child, o: o}
}

// BlurSigma is the blur of what is behind the surface.
//
// Flutter: cupertino/dialog.dart CupertinoPopupSurface.blurSigma
func (w *CupertinoPopupSurface) BlurSigma() float64 {
	return w.o.BlurSigma.OrElse(CupertinoPopupSurfaceDefaultBlurSigma)
}

// IsSurfacePainted is whether the surface has a color.
//
// Flutter: cupertino/dialog.dart CupertinoPopupSurface.isSurfacePainted
func (w *CupertinoPopupSurface) IsSurfacePainted() bool { return !w.o.NotSurfacePainted }

func (w *CupertinoPopupSurface) buildFilter(brightness ui.Brightness, hasBrightness bool) ui.ImageFilter {
	matrix := popupSurfaceLightSaturationMat
	if hasBrightness && brightness == ui.BrightnessDark {
		matrix = popupSurfaceDarkSaturationMat
	}
	colorFilter := ui.NewColorFilterMatrix(matrix)
	if w.BlurSigma() == 0 {
		return colorFilter
	}
	return ui.NewImageFilterCompose(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: w.BlurSigma(), SigmaY: w.BlurSigma()}), colorFilter)
}

func (w *CupertinoPopupSurface) Build(bc widgets.BuildContext) widgets.Widget {
	brightness, ok := CupertinoThemeMaybeBrightnessOf(bc)
	filter := w.buildFilter(brightness, ok)
	contents := w.child
	if w.IsSurfacePainted() {
		contents = widgets.NewColoredBox(CupertinoDynamicColorResolve(dialogColor, bc), contents)
	}
	return widgets.NewClipRSuperellipse(widgets.NewBackdropFilter(filter, contents),
		widgets.ClipRSuperellipseOpts{BorderRadius: painting.NewBorderRadiusCircular(13)})
}

// dialogDivider is a thin line between the buttons, which is hidden (it has the color of the background) next to
// a pressed button.
//
// Flutter: cupertino/dialog.dart _Divider
func newDialogDivider(dividerColor, hiddenColor ui.Color, hidden bool) widgets.Widget {
	color := dividerColor
	if hidden {
		color = hiddenColor
	}
	return widgets.NewLimitedBox(widgets.NewConstrainedBox(
		rendering.BoxConstraints{MinWidth: dialogDividerThickness, MinHeight: dialogDividerThickness, MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)},
		widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(color)}), nil)),
		widgets.LimitedBoxOpts{MaxHeight: foundation.Some(dialogDividerThickness), MaxWidth: foundation.Some(dialogDividerThickness)})
}

// overscrollBackground paints the background color behind the overscroll of the actions, so the overscroll does
// not show what is behind the dialog.
//
// Flutter: cupertino/dialog.dart _OverscrollBackground
type overscrollBackground struct {
	widgets.StatefulWidget
	color ui.Color
	child widgets.Widget
}

func newOverscrollBackground(color ui.Color, child widgets.Widget) *overscrollBackground {
	return &overscrollBackground{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{}), color: color, child: child}
}

func (w *overscrollBackground) CreateState() widgets.State { return &overscrollBackgroundState{} }

type overscrollBackgroundState struct {
	widgets.StateOf[*overscrollBackground]
	topOverscroll, bottomOverscroll float64
}

func (s *overscrollBackgroundState) onScrollUpdate(n *widgets.ScrollUpdateNotification) bool {
	m := n.Metrics()
	s.SetState(func() {
		s.topOverscroll = math.Min(math.Max(m.MinScrollExtent()-m.Pixels(), 0), m.ViewportDimension())
		s.bottomOverscroll = math.Min(math.Max(m.Pixels()-m.MaxScrollExtent(), 0), m.ViewportDimension())
	})
	return false
}

func (s *overscrollBackgroundState) Build(widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	box := func(h float64) widgets.Widget {
		return widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(w.color)}),
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(h)}))
	}
	overscroll := widgets.NewColumn([]widgets.Widget{box(s.topOverscroll), box(s.bottomOverscroll)}, widgets.ColumnOpts{
		MainAxisSize: rendering.MainAxisSizeMin, MainAxisAlignment: rendering.MainAxisAlignmentSpaceBetween, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
	return widgets.NewStack([]widgets.Widget{
		widgets.NewPositionedFill(overscroll),
		widgets.NewNotificationListener[*widgets.ScrollUpdateNotification](s.onScrollUpdate, w.child),
	})
}

// pressedUpdateHandler is told when a button is pressed (true) or released (false).
//
// Flutter: cupertino/dialog.dart _PressedUpdateHandler
type pressedUpdateHandler func(actionIndex int, state bool)

// alertContentSection is the title and the message of a dialog or an action sheet, in a scroll view.
//
// Flutter: cupertino/dialog.dart _CupertinoAlertContentSection
type alertContentSection struct {
	widgets.StatelessWidget
	title, message                   widgets.Widget
	scrollController                 widgets.ScrollController
	titlePadding, messagePadding     foundation.Opt[painting.EdgeInsets]
	titleStyle, messageStyle         *painting.TextStyle
	additionalPaddingTitleAndMessage foundation.Opt[painting.EdgeInsets]
}

func (w *alertContentSection) Build(widgets.BuildContext) widgets.Widget {
	if w.title == nil && w.message == nil {
		return widgets.NewSingleChildScrollView(widgets.NewSizedBoxShrink(nil), widgets.SingleChildScrollViewOpts{Controller: w.scrollController})
	}
	var group []widgets.Widget
	if w.title != nil {
		group = append(group, widgets.NewPadding(w.titlePadding.Value(),
			widgets.NewDefaultTextStyle(w.titleStyle, w.title, widgets.DefaultTextStyleOpts{TextAlign: ui.TextAlignCenter})))
	}
	if w.message != nil {
		group = append(group, widgets.NewPadding(w.messagePadding.Value(),
			widgets.NewDefaultTextStyle(w.messageStyle, w.message, widgets.DefaultTextStyleOpts{TextAlign: ui.TextAlignCenter})))
	}
	if extra, ok := w.additionalPaddingTitleAndMessage.Get(); ok && len(group) > 1 {
		group = append(group[:1], append([]widgets.Widget{widgets.NewPadding(extra, nil)}, group[1:]...)...)
	}
	return NewCupertinoScrollbar(widgets.NewSingleChildScrollView(
		widgets.NewColumn(group, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch}),
		widgets.SingleChildScrollViewOpts{Controller: w.scrollController}), CupertinoScrollbarOpts{Controller: w.scrollController})
}

// alertActionSection is the actions of an alert dialog, in a scroll view, with dividers between them.
//
// Flutter: cupertino/dialog.dart _CupertinoAlertActionSection
type alertActionSection struct {
	widgets.StatelessWidget
	actions          []widgets.Widget
	onPressedUpdate  pressedUpdateHandler
	pressedIndex     foundation.Opt[int]
	scrollController widgets.ScrollController
}

func newAlertActionSection(actions []widgets.Widget, pressedIndex foundation.Opt[int], onPressedUpdate pressedUpdateHandler, scrollController widgets.ScrollController) *alertActionSection {
	return &alertActionSection{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}), actions: actions, pressedIndex: pressedIndex,
		onPressedUpdate: onPressedUpdate, scrollController: scrollController}
}

func (w *alertActionSection) Build(bc widgets.BuildContext) widgets.Widget {
	idle := CupertinoDynamicColorResolve(dialogColor, bc)
	pressed := CupertinoDynamicColorResolve(dialogPressedColor, bc)
	dividerColor := CupertinoDynamicColorResolve(CupertinoColors.Separator, bc)
	var column []widgets.Widget
	isPressed := func(i int) bool { p, ok := w.pressedIndex.Get(); return ok && p == i }
	for i, action := range w.actions {
		if i != 0 {
			column = append(column, newDialogDivider(dividerColor, idle, isPressed(i-1) || isPressed(i)))
		}
		index := i
		column = append(column, &alertDialogButtonBackground{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{}),
			idleColor: idle, pressedColor: pressed, pressed: isPressed(i), child: action,
			onPressStateChange: func(state bool) { w.onPressedUpdate(index, state) }})
	}
	return NewCupertinoScrollbar(widgets.NewSingleChildScrollView(newAlertDialogActionsLayout(dialogDividerThickness, column),
		widgets.SingleChildScrollViewOpts{Controller: w.scrollController}), CupertinoScrollbarOpts{Controller: w.scrollController})
}

// alertDialogButtonBackground is the background of an action of an alert dialog: it is a slide target.
//
// Flutter: cupertino/dialog.dart _AlertDialogButtonBackground
type alertDialogButtonBackground struct {
	widgets.StatefulWidget
	pressed                 bool
	onPressStateChange      func(bool)
	idleColor, pressedColor ui.Color
	child                   widgets.Widget
}

func (w *alertDialogButtonBackground) CreateState() widgets.State {
	return &alertDialogButtonBackgroundState{}
}

type alertDialogButtonBackgroundState struct {
	widgets.StateOf[*alertDialogButtonBackground]
}

func emitSelectionClick(bc widgets.BuildContext) {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformAndroid:
		widgets.WidgetsBindingOf(bc).HapticFeedback().SelectionClick()
	}
}

func (s *alertDialogButtonBackgroundState) didEnter(fromPointerDown, innerEnabled bool) bool {
	if f := s.Widget().onPressStateChange; f != nil {
		f(innerEnabled)
	}
	if innerEnabled && !fromPointerDown {
		emitSelectionClick(s.Context())
	}
	return innerEnabled
}

func (s *alertDialogButtonBackgroundState) didLeave() {
	if f := s.Widget().onPressStateChange; f != nil {
		f(false)
	}
}

func (s *alertDialogButtonBackgroundState) didConfirm() { s.didLeave() }

func (s *alertDialogButtonBackgroundState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	background := w.idleColor
	if w.pressed {
		background = w.pressedColor
	}
	return widgets.NewMetaData(widgets.NewMergeSemantics(widgets.NewContainer(w.child, widgets.ContainerOpts{
		Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(CupertinoDynamicColorResolve(background, bc))})})),
		widgets.MetaDataOpts{MetaData: s})
}

// CupertinoDialogActionOpts are the optional parameters of NewCupertinoDialogAction.
//
// Flutter: cupertino/dialog.dart CupertinoDialogAction.new
type CupertinoDialogActionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnPressed is called when the action is tapped; nil disables the action.
	OnPressed func()
	// IsDefaultAction makes the text bold.
	IsDefaultAction bool
	// IsDestructiveAction makes the text red.
	IsDestructiveAction bool
	// TextStyle is merged into the style of the text.
	TextStyle *painting.TextStyle
	// MouseCursor is the cursor over the action; the default is the click cursor on the web when it is enabled.
	MouseCursor services.MouseCursor
}

// CupertinoDialogAction is a button of a CupertinoAlertDialog.
//
// Flutter: cupertino/dialog.dart CupertinoDialogAction
type CupertinoDialogAction struct {
	widgets.StatefulWidget
	child widgets.Widget
	o     CupertinoDialogActionOpts
}

// NewCupertinoDialogAction creates an action that shows the child.
//
// Flutter: cupertino/dialog.dart CupertinoDialogAction.new
func NewCupertinoDialogAction(child widgets.Widget, opts ...CupertinoDialogActionOpts) *CupertinoDialogAction {
	o := firstOf(opts)
	return &CupertinoDialogAction{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child, o: o}
}

// OnPressed is called when the action is tapped.
//
// Flutter: cupertino/dialog.dart CupertinoDialogAction.onPressed
func (w *CupertinoDialogAction) OnPressed() func() { return w.o.OnPressed }

// IsDefaultAction is whether the text is bold.
//
// Flutter: cupertino/dialog.dart CupertinoDialogAction.isDefaultAction
func (w *CupertinoDialogAction) IsDefaultAction() bool { return w.o.IsDefaultAction }

// IsDestructiveAction is whether the text is red.
//
// Flutter: cupertino/dialog.dart CupertinoDialogAction.isDestructiveAction
func (w *CupertinoDialogAction) IsDestructiveAction() bool { return w.o.IsDestructiveAction }

// Child is the widget that the action shows.
//
// Flutter: cupertino/dialog.dart CupertinoDialogAction.child
func (w *CupertinoDialogAction) Child() widgets.Widget { return w.child }

// CreateState creates the state.
//
// Flutter: cupertino/dialog.dart CupertinoDialogAction.createState
func (w *CupertinoDialogAction) CreateState() widgets.State { return &cupertinoDialogActionState{} }

type cupertinoDialogActionState struct {
	widgets.StateOf[*CupertinoDialogAction]
}

func (s *cupertinoDialogActionState) enabled() bool { return s.Widget().o.OnPressed != nil }

func (s *cupertinoDialogActionState) didEnter(_, _ bool) bool { return s.enabled() }
func (s *cupertinoDialogActionState) didLeave()               {}
func (s *cupertinoDialogActionState) didConfirm() {
	if f := s.Widget().o.OnPressed; f != nil {
		f()
	}
}

func (s *cupertinoDialogActionState) buildContentWithRegularSizingPolicy(bc widgets.BuildContext, textStyle *painting.TextStyle, content widgets.Widget, padding float64) widgets.Widget {
	width := dialogWidth
	if isInAccessibilityMode(bc) {
		width = dialogAccessibilityWidth
	}
	fontSizeRatio := widgets.MediaQueryTextScalerOf(bc).Scale(textStyle.FontSize.Value()) / dialogMinButtonFontSize
	return widgets.NewFittedBox(widgets.NewConstrainedBox(
		rendering.BoxConstraints{MaxWidth: fontSizeRatio * (width - 2*padding), MaxHeight: math.Inf(1)},
		widgets.NewSemantics(widgets.NewDefaultTextStyle(textStyle, content, widgets.DefaultTextStyleOpts{
			TextAlign: ui.TextAlignCenter, Overflow: painting.TextOverflowEllipsis, MaxLines: foundation.Some(1)}),
			widgets.SemanticsOpts{Button: foundation.Some(true), OnTap: s.Widget().o.OnPressed})),
		widgets.FittedBoxOpts{Fit: painting.BoxFitScaleDown})
}

func (s *cupertinoDialogActionState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	var base ui.ColorValue = CupertinoThemeOf(bc).PrimaryColor()
	if w.o.IsDestructiveAction {
		base = CupertinoColors.SystemRed
	}
	style := dialogActionStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(CupertinoDynamicColorResolve(base, bc))}).Merge(w.o.TextStyle)
	if w.o.IsDefaultAction {
		style = style.CopyWith(painting.TextStyleCopy{FontWeight: ui.FontWeightW600})
	}
	if !s.enabled() {
		style = style.CopyWith(painting.TextStyleCopy{Color: foundation.Some(style.Color.Value().WithOpacity(0.5))})
	}
	fontSize := style.FontSize.OrElse(painting.DefaultFontSize)
	if fontSize == 0 {
		fontSize = painting.DefaultFontSize
	}
	effectiveTextScale := widgets.MediaQueryTextScalerOf(bc).Scale(fontSize) / fontSize
	padding := 8.0 * effectiveTextScale
	var sizedContent widgets.Widget
	if isInAccessibilityMode(bc) {
		sizedContent = widgets.NewDefaultTextStyle(style, w.child, widgets.DefaultTextStyleOpts{TextAlign: ui.TextAlignCenter})
	} else {
		sizedContent = s.buildContentWithRegularSizingPolicy(bc, style, w.child, padding)
	}
	cursor := w.o.MouseCursor
	if cursor == nil {
		cursor = services.MouseCursorDefer
		if s.enabled() && foundation.IsWeb {
			cursor = services.SystemMouseCursors.Click
		}
	}
	return widgets.NewMouseRegion(widgets.NewMetaData(widgets.NewConstrainedBox(
		rendering.BoxConstraints{MinHeight: dialogMinButtonHeight, MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)},
		widgets.NewPadding(painting.NewEdgeInsetsAll(padding), widgets.NewCenter(sizedContent))),
		widgets.MetaDataOpts{MetaData: s, Behavior: rendering.HitTestBehaviorOpaque}),
		widgets.MouseRegionOpts{Cursor: cursor})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagCupertinoPopupSurfaceBlurSigmaNegative = "CupertinoPopupSurfaceBlurSigmaNegative"
