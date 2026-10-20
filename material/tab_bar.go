// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	// Flutter: material/tabs.dart _kTabHeight
	tabHeight = 46.0
	// Flutter: material/tabs.dart _kTextAndIconTabHeight
	tabTextAndIconHeight = 72.0
	// Flutter: material/tabs.dart _kStartOffset
	tabStartOffset = 52.0
)

// TabOpts are the optional parameters of NewTab.
//
// Flutter: material/tabs.dart Tab.new
type TabOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Text is the text of the label; it must not be given with Child.
	Text string
	// Icon is the icon, above the label.
	Icon widgets.Widget
	// IconMargin is the margin around the icon when the tab has a label too; the
	// default is 2 at the bottom (Material 3) or 10 (Material 2).
	IconMargin painting.EdgeInsetsGeometry
	// Height is the height of the tab; the default is 46, or 72 with a label and
	// an icon.
	Height foundation.Opt[float64]
	// Child is the label, instead of Text.
	Child widgets.Widget
}

// Tab is a Material Design TabBar tab: it has a text label and an icon, or a
// child, or both.
//
// Flutter: material/tabs.dart Tab
type Tab struct {
	widgets.StatelessWidget
	o TabOpts
}

// NewTab creates a tab. In debug builds it panics with the tag TabNoContent when
// there is no text, no child and no icon, and with TabTextAndChild when there are
// both text and child.
//
// Flutter: material/tabs.dart Tab.new
func NewTab(opts ...TabOpts) *Tab {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if o.Text == "" && o.Child == nil && o.Icon == nil {
			panic(errors.New("TabNoContent", "Tab requires at least one of text, child, or icon to be non-null."))
		}
		if o.Text != "" && o.Child != nil {
			panic(errors.New("TabTextAndChild", "Provide either text or child, not both, when creating a Tab."))
		}
	}
	return &Tab{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Text is the text of the label.
//
// Flutter: material/tabs.dart Tab.text
func (w *Tab) Text() string { return w.o.Text }

func (w *Tab) buildLabelText() widgets.Widget {
	if w.o.Child != nil {
		return w.o.Child
	}
	return widgets.NewText(w.o.Text, widgets.TextOpts{SoftWrap: foundation.Some(false), Overflow: painting.TextOverflowFade})
}

// Build builds the tab.
//
// Flutter: material/tabs.dart Tab.build
func (w *Tab) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterial(bc)

	var calculatedHeight float64
	var label widgets.Widget
	switch {
	case w.o.Icon == nil:
		calculatedHeight = tabHeight
		label = w.buildLabelText()
	case w.o.Text == "" && w.o.Child == nil:
		calculatedHeight = tabHeight
		label = w.o.Icon
	default:
		calculatedHeight = tabTextAndIconHeight
		effectiveIconMargin := w.o.IconMargin
		if effectiveIconMargin == nil {
			if ThemeOf(bc).UseMaterial3 {
				effectiveIconMargin = tabsPrimaryIconMargin
			} else {
				effectiveIconMargin = tabsM2IconMargin
			}
		}
		label = widgets.NewColumn([]widgets.Widget{
			widgets.NewPadding(effectiveIconMargin, w.o.Icon),
			w.buildLabelText(),
		}, widgets.ColumnOpts{MainAxisAlignment: rendering.MainAxisAlignmentCenter})
	}

	return widgets.NewSizedBox(
		widgets.NewCenter(label, widgets.CenterOpts{WidthFactor: foundation.Some(1.0)}),
		widgets.SizedBoxOpts{Height: foundation.Some(w.o.Height.OrElse(calculatedHeight))})
}

// PreferredSize is the height of the tab.
//
// Flutter: material/tabs.dart Tab.preferredSize
func (w *Tab) PreferredSize() ui.Size {
	if h, ok := w.o.Height.Get(); ok {
		return ui.Size{Height: h}
	}
	if (w.o.Text != "" || w.o.Child != nil) && w.o.Icon != nil {
		return ui.Size{Height: tabTextAndIconHeight}
	}
	return ui.Size{Height: tabHeight}
}

// DebugFillProperties adds the text.
//
// Flutter: material/tabs.dart Tab.debugFillProperties
func (w *Tab) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("text", nilIfZero(w.o.Text)))
}

// tabStyle gives its child the style of the label of a tab, which changes with
// the animation.
//
// Flutter: material/tabs.dart _TabStyle
type tabStyle struct {
	widgets.AnimatedWidget
	animation            animation.Animation[float64]
	isSelected           bool
	isPrimary            bool
	labelColor           foundation.Opt[ui.Color]
	unselectedLabelColor foundation.Opt[ui.Color]
	labelStyle           *painting.TextStyle
	unselectedLabelStyle *painting.TextStyle
	defaults             TabBarThemeData
	child                widgets.Widget
}

func newTabStyle(a animation.Animation[float64], isSelected, isPrimary bool, labelColor, unselectedLabelColor foundation.Opt[ui.Color],
	labelStyle, unselectedLabelStyle *painting.TextStyle, defaults TabBarThemeData, child widgets.Widget) *tabStyle {
	return &tabStyle{AnimatedWidget: widgets.NewAnimatedWidgetBase(a), animation: a, isSelected: isSelected, isPrimary: isPrimary,
		labelColor: labelColor, unselectedLabelColor: unselectedLabelColor, labelStyle: labelStyle,
		unselectedLabelStyle: unselectedLabelStyle, defaults: defaults, child: child}
}

func (w *tabStyle) resolveWithLabelColor(bc widgets.BuildContext, iconTheme *widgets.IconThemeData) widgets.WidgetStateProperty[ui.Color] {
	themeData := ThemeOf(bc)
	tabBarTheme := TabBarThemeOf(bc)

	// labelStyle.color (and tabBarTheme.labelStyle.color) is not considered as
	// it'll be a breaking change without a possible migration plan.
	//
	// Deviation: Flutter's labelColor can be a WidgetStateColor, which resolves
	// to a selected and an unselected color; here it is a plain color, and the
	// unselected color comes from the other properties (SPEC §12.10).
	selectedColor := firstOpt(w.labelColor, tabBarTheme.LabelColor, styleColor(w.labelStyle), styleColor(tabBarTheme.LabelStyle), w.defaults.LabelColor).Value()

	// unselectedLabelColor and tabBarTheme.unselectedLabelColor are ignored when
	// labelColor is a WidgetStateColor.
	var iconColor foundation.Opt[ui.Color]
	if iconTheme != nil {
		iconColor = iconTheme.Color()
	}
	fallback := selectedColor.WithAlpha(0xB2) // 70% alpha
	if themeData.UseMaterial3 {
		fallback = w.defaults.UnselectedLabelColor.Value()
	}
	unselectedColor := firstOpt(w.unselectedLabelColor, tabBarTheme.UnselectedLabelColor, styleColor(w.unselectedLabelStyle),
		styleColor(tabBarTheme.UnselectedLabelStyle), iconColor, foundation.Some(fallback)).Value()

	a := w.animation
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) ui.Color {
		if states.Has(widgets.WidgetStateSelected) {
			return *ui.ColorLerp(&selectedColor, &unselectedColor, a.Value())
		}
		return *ui.ColorLerp(&unselectedColor, &selectedColor, a.Value())
	})
}

func styleColor(s *painting.TextStyle) foundation.Opt[ui.Color] {
	if s == nil {
		return foundation.None[ui.Color]()
	}
	return s.Color
}

func (w *tabStyle) Build(bc widgets.BuildContext) widgets.Widget {
	theme := ThemeOf(bc)
	tabBarTheme := TabBarThemeOf(bc)

	states := unselectedState
	if w.isSelected {
		states = selectedState
	}

	// To enable TextStyle.lerp(style1, style2, value), both styles must have the
	// same value of inherit. Force that to be inherit=true here.
	inherit := painting.TextStyleCopy{Inherit: foundation.Some(true)}
	selectedStyle := w.defaults.LabelStyle.Merge(firstTextStyle(w.labelStyle, tabBarTheme.LabelStyle)).CopyWith(inherit)
	unselectedStyle := w.defaults.UnselectedLabelStyle.Merge(firstTextStyle(w.unselectedLabelStyle, tabBarTheme.UnselectedLabelStyle, w.labelStyle)).CopyWith(inherit)
	var textStyle *painting.TextStyle
	if w.isSelected {
		textStyle = painting.TextStyleLerp(selectedStyle, unselectedStyle, w.animation.Value())
	} else {
		textStyle = painting.TextStyleLerp(unselectedStyle, selectedStyle, w.animation.Value())
	}
	defaultIconColor := DefaultIconLightColor
	if theme.ColorScheme.Brightness() == ui.BrightnessLight {
		defaultIconColor = DefaultIconDarkColor
	}
	var customIconTheme *widgets.IconThemeData
	if it := widgets.IconThemeOf(bc); it.Color() != foundation.Some(defaultIconColor) {
		customIconTheme = &it
	}
	iconColor := w.resolveWithLabelColor(bc, customIconTheme).Resolve(states)
	labelColor := w.resolveWithLabelColor(bc, nil).Resolve(states)

	iconSize := 24.0
	if customIconTheme != nil {
		iconSize = customIconTheme.Size().OrElse(24.0)
	}
	return widgets.NewDefaultTextStyle(textStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(labelColor)}),
		widgets.NewIconThemeMerge(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Size: foundation.Some(iconSize), Color: foundation.Some(iconColor)}), w.child))
}

// layoutCallback is called with the offsets of the leading edges of the tabs and
// of the trailing edge of the last tab, the text direction and the width.
//
// Flutter: material/tabs.dart _LayoutCallback
type layoutCallback func(xOffsets []float64, textDirection ui.TextDirection, width float64)

// tabLabelBarRenderer reports the offsets of the tabs when it is laid out.
//
// Flutter: material/tabs.dart _TabLabelBarRenderer
type tabLabelBarRenderer struct {
	rendering.RenderFlex
	onPerformLayout layoutCallback
}

func (r *tabLabelBarRenderer) PerformLayout() {
	r.RenderFlex.PerformLayout()
	// xOffsets will contain childCount+1 values, giving the offsets of the leading
	// edge of the first tab as the first value, of the leading edge of the each
	// subsequent tab as each subsequent value, and of the trailing edge of the last
	// tab as the last value.
	var xOffsets []float64
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		xOffsets = append(xOffsets, child.ParentData().(*rendering.FlexParentData).Offset.Dx)
	}
	textDirection := r.TextDirection()
	switch textDirection {
	case ui.TextDirectionRTL:
		xOffsets = append([]float64{r.Size().Width}, xOffsets...)
	default:
		xOffsets = append(xOffsets, r.Size().Width)
	}
	r.onPerformLayout(xOffsets, textDirection, r.Size().Width)
}

// tabLabelBar only exists, with its renderer, to report the widths of the tabs
// upon layout. The tab widths are only used at paint time (see indicatorPainter)
// or in response to input.
//
// Flutter: material/tabs.dart _TabLabelBar
type tabLabelBar struct {
	widgets.Flex
	onPerformLayout layoutCallback
}

func newTabLabelBar(children []widgets.Widget, mainAxisSize rendering.MainAxisSize, onPerformLayout layoutCallback) *tabLabelBar {
	return &tabLabelBar{
		Flex: *widgets.NewFlex(painting.AxisHorizontal, children, widgets.FlexOpts{
			MainAxisAlignment: rendering.MainAxisAlignmentStart, MainAxisSize: mainAxisSize,
			CrossAxisAlignment: rendering.CrossAxisAlignmentCenter, VerticalDirection: painting.VerticalDirectionDown}),
		onPerformLayout: onPerformLayout,
	}
}

func (w *tabLabelBar) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	r := &tabLabelBarRenderer{onPerformLayout: w.onPerformLayout}
	rendering.InitRenderFlex(&r.RenderFlex, r, nil, rendering.RenderFlexOpts{
		Direction: w.Direction(), MainAxisAlignment: w.MainAxisAlignment(), MainAxisSize: w.MainAxisSize(),
		CrossAxisAlignment: w.CrossAxisAlignment(), TextDirection: widgets.DirectionalityOf(bc), VerticalDirection: w.VerticalDirection()})
	return r
}

func (w *tabLabelBar) UpdateRenderObject(bc widgets.BuildContext, renderObject rendering.RenderObject) {
	w.Flex.UpdateRenderObject(bc, &renderObject.(*tabLabelBarRenderer).RenderFlex)
	renderObject.(*tabLabelBarRenderer).onPerformLayout = w.onPerformLayout
}

// indexChangeProgress is how far the index change of the controller got, from 0
// to 1.
//
// Flutter: material/tabs.dart _indexChangeProgress
func indexChangeProgress(controller *TabController) float64 {
	controllerValue := controller.Animation().Value()
	previousIndex := float64(controller.PreviousIndex())
	currentIndex := float64(controller.Index())

	// The controller's offset is changing because the user is dragging the
	// TabBarView's PageView to the left or right.
	if !controller.IndexIsChanging() {
		return ui.ClampDouble(math.Abs(currentIndex-controllerValue), 0.0, 1.0)
	}

	// The TabController animation's value is changing from previousIndex to
	// currentIndex.
	return math.Abs(controllerValue-currentIndex) / math.Abs(currentIndex-previousIndex)
}

// dividerPainter paints the divider below a scrollable tab bar.
//
// Flutter: material/tabs.dart _DividerPainter
type dividerPainter struct {
	rendering.CustomPainterBase
	dividerColor  ui.Color
	dividerHeight float64
}

func newDividerPainter(dividerColor ui.Color, dividerHeight float64) *dividerPainter {
	return &dividerPainter{CustomPainterBase: rendering.NewCustomPainterBase(nil), dividerColor: dividerColor, dividerHeight: dividerHeight}
}

func (p *dividerPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	if p.dividerHeight <= 0.0 {
		return
	}
	paint := ui.NewPaint()
	paint.SetColor(p.dividerColor)
	paint.SetStrokeWidth(p.dividerHeight)
	canvas.DrawLine(ui.Offset{Dx: 0, Dy: size.Height - p.dividerHeight/2}, ui.Offset{Dx: size.Width, Dy: size.Height - p.dividerHeight/2}, paint)
}

func (p *dividerPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o := old.(*dividerPainter)
	return o.dividerColor != p.dividerColor || o.dividerHeight != p.dividerHeight
}

// indicatorPainter paints the indicator of the selected tab and the divider.
//
// Flutter: material/tabs.dart _IndicatorPainter
type indicatorPainter struct {
	rendering.CustomPainterBase
	controller         *TabController
	indicator          painting.Decoration
	indicatorSize      TabBarIndicatorSize
	indicatorPadding   painting.EdgeInsetsGeometry
	tabKeys            []*widgets.GlobalKey[widgets.State]
	labelPaddings      []painting.EdgeInsetsGeometry
	dividerColor       foundation.Opt[ui.Color]
	dividerHeight      foundation.Opt[float64]
	showDivider        bool
	devicePixelRatio   foundation.Opt[float64]
	indicatorAnimation TabIndicatorAnimation
	textDirection      ui.TextDirection
	repaint            *foundation.ChangeNotifier

	// currentTabOffsets and currentTextDirection are set each time TabBar layout
	// is completed. These values can be nil when TabBar contains no tabs, since
	// there are nothing to lay out.
	currentTabOffsets    []float64
	currentTextDirection ui.TextDirection
	currentRect          ui.Rect
	painter              painting.BoxPainter
	needsPaint           bool
}

type indicatorPainterOpts struct {
	controller         *TabController
	indicator          painting.Decoration
	indicatorSize      TabBarIndicatorSize
	tabKeys            []*widgets.GlobalKey[widgets.State]
	old                *indicatorPainter
	indicatorPadding   painting.EdgeInsetsGeometry
	labelPaddings      []painting.EdgeInsetsGeometry
	dividerColor       foundation.Opt[ui.Color]
	dividerHeight      foundation.Opt[float64]
	showDivider        bool
	devicePixelRatio   foundation.Opt[float64]
	indicatorAnimation TabIndicatorAnimation
	textDirection      ui.TextDirection
}

func newIndicatorPainter(o indicatorPainterOpts) *indicatorPainter {
	repaint := &foundation.ChangeNotifier{}
	p := &indicatorPainter{
		CustomPainterBase: rendering.NewCustomPainterBase(foundation.MergeListenables(o.controller.Animation(), repaint)),
		controller:        o.controller, indicator: o.indicator, indicatorSize: o.indicatorSize, indicatorPadding: o.indicatorPadding,
		tabKeys: o.tabKeys, labelPaddings: o.labelPaddings, dividerColor: o.dividerColor, dividerHeight: o.dividerHeight,
		showDivider: o.showDivider, devicePixelRatio: o.devicePixelRatio, indicatorAnimation: o.indicatorAnimation,
		textDirection: o.textDirection, repaint: repaint,
	}
	if o.old != nil {
		p.saveTabOffsets(o.old.currentTabOffsets, o.old.currentTextDirection)
	}
	return p
}

func (p *indicatorPainter) markNeedsPaint() {
	p.needsPaint = true
	p.repaint.NotifyListeners()
}

func (p *indicatorPainter) dispose() {
	if p.painter != nil {
		p.painter.Dispose()
	}
	p.repaint.Dispose()
}

func (p *indicatorPainter) saveTabOffsets(tabOffsets []float64, textDirection ui.TextDirection) {
	p.currentTabOffsets = tabOffsets
	p.currentTextDirection = textDirection
}

// maxTabIndex is the index of the last tab: currentTabOffsets[index] is the
// offset of the start edge of the tab at index, and
// currentTabOffsets[len(currentTabOffsets)-1] is the end edge of the last tab.
func (p *indicatorPainter) maxTabIndex() int { return len(p.currentTabOffsets) - 2 }

func (p *indicatorPainter) centerOf(tabIndex int) float64 {
	return (p.currentTabOffsets[tabIndex] + p.currentTabOffsets[tabIndex+1]) / 2.0
}

func (p *indicatorPainter) indicatorRect(tabBarSize ui.Size, tabIndex int) ui.Rect {
	var tabLeft, tabRight float64
	if p.currentTextDirection == ui.TextDirectionRTL {
		tabLeft, tabRight = p.currentTabOffsets[tabIndex+1], p.currentTabOffsets[tabIndex]
	} else {
		tabLeft, tabRight = p.currentTabOffsets[tabIndex], p.currentTabOffsets[tabIndex+1]
	}

	if p.indicatorSize == TabBarIndicatorSizeLabel {
		size, _ := p.tabKeys[tabIndex].CurrentContext().Size()
		tabWidth := size.Width
		insets := p.labelPaddings[tabIndex].Resolve(p.currentTextDirection)
		delta := ((tabRight - tabLeft) - (tabWidth + insets.Horizontal())) / 2.0
		tabLeft += delta + insets.Left
		tabRight = tabLeft + tabWidth
	}

	insets := p.indicatorPadding.Resolve(p.currentTextDirection)
	rect := ui.NewRectFromLTWH(tabLeft, 0.0, tabRight-tabLeft, tabBarSize.Height)

	if !rect.Size().Ge(insets.CollapsedSize()) {
		panic(errors.New("TabBarIndicatorPaddingTooBig", "indicatorPadding insets should be less than Tab Size",
			"rectSize", rect.Size(), "insets", insets))
	}
	return insets.DeflateRect(rect)
}

func (p *indicatorPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	p.needsPaint = false
	if p.painter == nil {
		p.painter = p.indicator.CreateBoxPainter(p.markNeedsPaint)
	}

	value := p.controller.Animation().Value()

	switch p.indicatorAnimation {
	case TabIndicatorAnimationElastic:
		p.currentRect = p.applyElasticEffect(size, value)
	default:
		p.currentRect = p.applyLinearEffect(size, value)
	}

	configuration := painting.ImageConfiguration{
		Size: foundation.Some(p.currentRect.Size()), TextDirection: p.currentTextDirection, DevicePixelRatio: p.devicePixelRatio}
	if p.showDivider && p.dividerHeight.Value() > 0 {
		dividerPaint := ui.NewPaint()
		dividerPaint.SetColor(p.dividerColor.Value())
		dividerPaint.SetStrokeWidth(p.dividerHeight.Value())
		y := size.Height - p.dividerHeight.Value()/2
		canvas.DrawLine(ui.Offset{Dx: 0, Dy: y}, ui.Offset{Dx: size.Width, Dy: y}, dividerPaint)
	}
	p.painter.Paint(canvas, p.currentRect.TopLeft(), configuration)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (p *indicatorPainter) applyLinearEffect(size ui.Size, value float64) ui.Rect {
	index := float64(p.controller.Index())
	ltr := index > value
	var from int
	if ltr {
		from = clampInt(int(math.Floor(value)), 0, p.maxTabIndex())
	} else {
		from = clampInt(int(math.Ceil(value)), 0, p.maxTabIndex())
	}
	to := from - 1
	if ltr {
		to = from + 1
	}
	to = clampInt(to, 0, p.maxTabIndex())
	fromRect := p.indicatorRect(size, from)
	toRect := p.indicatorRect(size, to)
	return *ui.RectLerp(&fromRect, &toRect, math.Abs(value-float64(from)))
}

// decelerateInterpolation is ease out sine (decelerating).
func decelerateInterpolation(fraction float64) float64 { return math.Sin((fraction * math.Pi) / 2.0) }

// accelerateInterpolation is ease in sine (accelerating).
func accelerateInterpolation(fraction float64) float64 {
	return 1.0 - math.Cos((fraction*math.Pi)/2.0)
}

func (p *indicatorPainter) applyElasticEffect(size ui.Size, value float64) ui.Rect {
	index := float64(p.controller.Index())
	progressLeft := math.Abs(index - value)

	var to, from int
	if progressLeft == 0.0 || !p.controller.IndexIsChanging() {
		if p.textDirection == ui.TextDirectionRTL {
			to = clampInt(int(math.Floor(value)), 0, p.maxTabIndex())
			from = clampInt(to+1, 0, p.maxTabIndex())
		} else {
			to = clampInt(int(math.Ceil(value)), 0, p.maxTabIndex())
			from = clampInt(to-1, 0, p.maxTabIndex())
		}
	} else {
		to = p.controller.Index()
		from = p.controller.PreviousIndex()
	}
	toRect := p.indicatorRect(size, to)
	fromRect := p.indicatorRect(size, from)
	rect := *ui.RectLerp(&fromRect, &toRect, math.Abs(value-float64(from)))

	// If the tab animation is completed, there is no need to stretch the
	// indicator. This only works for the tab change animation via tab index, not
	// when dragging a TabBarView, but it's still ok, to avoid unnecessary
	// calculations.
	if animation.IsCompleted(p.controller.Animation()) {
		return rect
	}

	var tabChangeProgress float64
	if p.controller.IndexIsChanging() {
		tabsDelta := math.Abs(float64(p.controller.Index() - p.controller.PreviousIndex()))
		if tabsDelta != 0 {
			progressLeft /= tabsDelta
		}
		tabChangeProgress = 1 - ui.ClampDouble(progressLeft, 0.0, 1.0)
	} else {
		tabChangeProgress = math.Abs(index - value)
	}

	// If the animation has finished, there is no need to apply the stretch effect.
	if tabChangeProgress == 1.0 {
		return rect
	}

	var isMovingRight bool
	if p.textDirection == ui.TextDirectionRTL {
		isMovingRight = value > index
		if !p.controller.IndexIsChanging() {
			isMovingRight = index > value
		}
	} else {
		isMovingRight = value > index
		if p.controller.IndexIsChanging() {
			isMovingRight = index > value
		}
	}
	var leftFraction, rightFraction float64
	if isMovingRight {
		leftFraction = accelerateInterpolation(tabChangeProgress)
		rightFraction = decelerateInterpolation(tabChangeProgress)
	} else {
		leftFraction = decelerateInterpolation(tabChangeProgress)
		rightFraction = accelerateInterpolation(tabChangeProgress)
	}

	var lerpRectLeft, lerpRectRight float64
	// The controller.indexIsChanging is true when the Tab is pressed, instead of
	// swipe to change tabs. If the tab is pressed then only lerp between fromRect
	// and toRect.
	switch {
	case p.controller.IndexIsChanging():
		lerpRectLeft = ui.LerpDouble(fromRect.Left, toRect.Left, leftFraction)
		lerpRectRight = ui.LerpDouble(fromRect.Right, toRect.Right, rightFraction)
	case isMovingRight:
		// Switch the Rect left and right lerp order based on swipe direction.
		lerpRectLeft = ui.LerpDouble(fromRect.Left, toRect.Left, leftFraction)
		lerpRectRight = ui.LerpDouble(fromRect.Right, toRect.Right, rightFraction)
	default:
		lerpRectLeft = ui.LerpDouble(toRect.Left, fromRect.Left, leftFraction)
		lerpRectRight = ui.LerpDouble(toRect.Right, fromRect.Right, rightFraction)
	}

	return ui.NewRectFromLTRB(lerpRectLeft, rect.Top, lerpRectRight, rect.Bottom)
}

func (p *indicatorPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o := old.(*indicatorPainter)
	return p.needsPaint || p.controller != o.controller || !foundation.ValuesEqual(p.indicator, o.indicator) ||
		len(p.tabKeys) != len(o.tabKeys) || !floatsEqual(p.currentTabOffsets, o.currentTabOffsets) ||
		p.currentTextDirection != o.currentTextDirection
}

func floatsEqual(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// changeAnimation is the progress of an index change of the controller, from 0
// to 1.
//
// Flutter: material/tabs.dart _ChangeAnimation
type changeAnimation struct{ controller *TabController }

func (a *changeAnimation) parent() animation.Animation[float64] { return a.controller.Animation() }

func (a *changeAnimation) AddListener(l foundation.VoidCallback) foundation.ListenerHandle {
	return a.parent().AddListener(l)
}

func (a *changeAnimation) RemoveListener(h foundation.ListenerHandle) {
	if p := a.parent(); p != nil {
		p.RemoveListener(h)
	}
}

func (a *changeAnimation) AddStatusListener(l animation.AnimationStatusListener) animation.StatusListenerHandle {
	return a.parent().AddStatusListener(l)
}

func (a *changeAnimation) RemoveStatusListener(h animation.StatusListenerHandle) {
	if p := a.parent(); p != nil {
		p.RemoveStatusListener(h)
	}
}

func (a *changeAnimation) Status() animation.AnimationStatus { return a.parent().Status() }

func (a *changeAnimation) Value() float64 { return indexChangeProgress(a.controller) }

// dragAnimation is how far the drag of the TabBarView moved away from a tab,
// from 0 to 1.
//
// Flutter: material/tabs.dart _DragAnimation
type dragAnimation struct {
	changeAnimation
	index int
}

func (a *dragAnimation) Value() float64 {
	if foundation.DebugMode && a.controller.IndexIsChanging() {
		panic(errors.New("TabBarDragWhileChanging", "the drag animation of a tab cannot be read while the index is changing"))
	}
	controllerMaxValue := float64(a.controller.Length() - 1)
	controllerValue := ui.ClampDouble(a.controller.Animation().Value(), 0.0, controllerMaxValue)
	return ui.ClampDouble(math.Abs(controllerValue-float64(a.index)), 0.0, 1.0)
}

// tabBarScrollPosition only exists, with TabBarScrollController, to handle the
// case where a scrollable TabBar has a non-zero initialIndex. In that case we can
// only compute the scroll position's initial scroll offset (the "correct" pixels
// value) after the TabBar viewport width and scroll limits are known.
//
// Flutter: material/tabs.dart _TabBarScrollPosition
type tabBarScrollPosition struct {
	widgets.ScrollPositionWithSingleContext
	tabBar                      *tabBarState
	viewportDimensionWasNonZero bool
	// The scroll position should be adjusted at least once.
	needsPixelsCorrection bool
}

func (p *tabBarScrollPosition) ApplyContentDimensions(minScrollExtent, maxScrollExtent float64) bool {
	result := true
	if !p.viewportDimensionWasNonZero {
		p.viewportDimensionWasNonZero = p.ViewportDimension() != 0.0
	}
	// If the viewport never had a non-zero dimension, we just want to jump to the
	// initial scroll position to avoid strange scrolling effects in release mode:
	// the viewport temporarily may have a dimension of zero before the actual
	// dimension is calculated. In that scenario, setting the actual dimension would
	// cause a strange scroll effect without this guard because the super call
	// below would start a ballistic scroll activity.
	if !p.viewportDimensionWasNonZero || p.needsPixelsCorrection {
		p.needsPixelsCorrection = false
		p.CorrectPixels(p.tabBar.initialScrollOffset(p.ViewportDimension(), minScrollExtent, maxScrollExtent))
		result = false
	}
	return p.ScrollPositionWithSingleContext.ApplyContentDimensions(minScrollExtent, maxScrollExtent) && result
}

func (p *tabBarScrollPosition) markNeedsPixelsCorrection() { p.needsPixelsCorrection = true }

// TabBarScrollController is the ScrollController of a scrollable TabBar: it
// creates the scroll position that brings the selected tab into view at first.
//
// Flutter: material/tabs.dart TabBarScrollController
type TabBarScrollController struct {
	widgets.ScrollControllerBase
	tabBarState *tabBarState
}

// NewTabBarScrollController creates the controller.
//
// Flutter: material/tabs.dart TabBarScrollController.new
func NewTabBarScrollController() *TabBarScrollController {
	c := &TabBarScrollController{}
	widgets.InitScrollControllerBase(&c.ScrollControllerBase, c, widgets.ScrollControllerOpts{})
	return c
}

// CreateScrollPosition creates the position.
//
// Flutter: material/tabs.dart TabBarScrollController.createScrollPosition
func (c *TabBarScrollController) CreateScrollPosition(physics widgets.ScrollPhysics, context widgets.ScrollContext, oldPosition widgets.ScrollPosition) widgets.ScrollPosition {
	if foundation.DebugMode && c.tabBarState == nil {
		panic(errors.New("TabBarScrollControllerDetached", "This TabBarScrollController is not attached to any TabBar."))
	}
	p := &tabBarScrollPosition{tabBar: c.tabBarState, needsPixelsCorrection: true}
	widgets.InitScrollPositionWithSingleContext(&p.ScrollPositionWithSingleContext, p, physics, context,
		widgets.ScrollPositionWithSingleContextOpts{NoInitialPixels: true, OldPosition: oldPosition})
	return p
}

// Dispose detaches the controller from the tab bar.
//
// Flutter: material/tabs.dart TabBarScrollController.dispose
func (c *TabBarScrollController) Dispose() {
	c.tabBarState = nil
	c.ScrollControllerBase.Dispose()
}

// TabValueChanged is the signature of a callback that reports a change of a
// value of a tab, with its index.
//
// Flutter: material/tabs.dart TabValueChanged
type TabValueChanged[T any] func(value T, index int)

// TabBarOpts are the optional parameters of NewTabBar and NewTabBarSecondary.
//
// Flutter: material/tabs.dart TabBar.new
type TabBarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Controller coordinates the tab selection with the TabBarView; the default
	// is the one of the closest DefaultTabController.
	Controller *TabController
	// ScrollController controls the scroll of a scrollable bar.
	ScrollController *TabBarScrollController
	// IsScrollable says that the tabs take their natural width and scroll when
	// they do not fit.
	IsScrollable bool
	// Padding is the padding of the bar.
	Padding painting.EdgeInsetsGeometry
	// IndicatorColor is the color of the line below the selected tab.
	IndicatorColor foundation.Opt[ui.Color]
	// NoAutomaticIndicatorColorAdjustment turns off the change of the indicator
	// color to white when it is the color of the Material below the bar (Flutter's
	// automaticIndicatorColorAdjustment: false).
	NoAutomaticIndicatorColorAdjustment bool
	// IndicatorWeight is the thickness of the line below the selected tab; the
	// default is 2.
	IndicatorWeight foundation.Opt[float64]
	// IndicatorPadding is the padding of the indicator.
	IndicatorPadding painting.EdgeInsetsGeometry
	// Indicator defines the appearance of the selected tab indicator.
	Indicator painting.Decoration
	// IndicatorSize defines how the bounds of the indicator are computed.
	IndicatorSize TabBarIndicatorSize
	// DividerColor is the color of the divider below the bar.
	DividerColor foundation.Opt[ui.Color]
	// DividerHeight is the height of the divider below the bar.
	DividerHeight foundation.Opt[float64]
	// LabelColor is the color of the label of the selected tab.
	LabelColor foundation.Opt[ui.Color]
	// LabelStyle is the text style of the label of the selected tab.
	LabelStyle *painting.TextStyle
	// LabelPadding is the padding of each tab label; the default is 16 at each
	// side.
	LabelPadding painting.EdgeInsetsGeometry
	// UnselectedLabelColor is the color of the labels of the tabs that are not
	// selected.
	UnselectedLabelColor foundation.Opt[ui.Color]
	// UnselectedLabelStyle is the text style of the labels of the tabs that are
	// not selected.
	UnselectedLabelStyle *painting.TextStyle
	// DragStartBehavior is where a drag of a scrollable bar starts.
	DragStartBehavior gestures.DragStartBehavior
	// OverlayColor is the color of the highlight and the splash of a tab, in each
	// state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// MouseCursor is the cursor over a tab; it may be a WidgetStateMouseCursor.
	MouseCursor services.MouseCursor
	// EnableFeedback says whether taps give acoustic and haptic feedback; the
	// default is true.
	EnableFeedback foundation.Opt[bool]
	// OnTap is called when a tab is tapped, with its index.
	OnTap func(index int)
	// OnHover is called when a pointer enters or leaves a tab.
	OnHover TabValueChanged[bool]
	// OnFocusChange is called when the focus of a tab changes.
	OnFocusChange TabValueChanged[bool]
	// Physics is how a scrollable bar responds to the user.
	Physics widgets.ScrollPhysics
	// SplashFactory creates the splashes of the tabs.
	SplashFactory InteractiveInkFeatureFactory
	// SplashBorderRadius is the radius of the splash and the highlight of a tab.
	SplashBorderRadius foundation.Opt[painting.BorderRadius]
	// TabAlignment is how the tabs are aligned horizontally.
	TabAlignment TabAlignment
	// TextScaler scales the text of the labels.
	TextScaler painting.TextScaler
	// IndicatorAnimation is how the indicator moves when the selected tab changes.
	IndicatorAnimation TabIndicatorAnimation
}

// TabBar is a Material Design primary tab bar: a row of horizontal tabs, and an
// indicator that shows which one is selected.
//
// Flutter: material/tabs.dart TabBar
type TabBar struct {
	widgets.StatefulWidget
	tabs      []widgets.Widget
	o         TabBarOpts
	isPrimary bool
}

func newTabBar(tabs []widgets.Widget, o TabBarOpts, isPrimary bool) *TabBar {
	if o.IndicatorPadding == nil {
		o.IndicatorPadding = painting.EdgeInsetsZero
	}
	if !o.IndicatorWeight.IsSome() {
		o.IndicatorWeight = foundation.Some(2.0)
	}
	if foundation.DebugMode && o.Indicator == nil && !(o.IndicatorWeight.Value() > 0.0) {
		panic(errors.New("TabBarIndicatorWeight", "the indicator weight of a TabBar without an indicator must be greater than zero", "indicatorWeight", o.IndicatorWeight.Value()))
	}
	return &TabBar{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), tabs: tabs, o: o, isPrimary: isPrimary}
}

// NewTabBar creates a primary tab bar. In debug builds it panics with the tag
// TabBarIndicatorWeight when there is no indicator and the weight is not greater
// than zero.
//
// Flutter: material/tabs.dart TabBar.new
func NewTabBar(tabs []widgets.Widget, opts ...TabBarOpts) *TabBar {
	return newTabBar(tabs, firstOpts(opts), true)
}

// NewTabBarSecondary creates a secondary tab bar, for the content of a page that
// a primary tab bar selected.
//
// Flutter: material/tabs.dart TabBar.secondary
func NewTabBarSecondary(tabs []widgets.Widget, opts ...TabBarOpts) *TabBar {
	return newTabBar(tabs, firstOpts(opts), false)
}

// Tabs are the tabs of the bar, usually Tab widgets.
//
// Flutter: material/tabs.dart TabBar.tabs
func (w *TabBar) Tabs() []widgets.Widget { return w.tabs }

// IsScrollable is whether the bar scrolls.
//
// Flutter: material/tabs.dart TabBar.isScrollable
func (w *TabBar) IsScrollable() bool { return w.o.IsScrollable }

// PreferredSize is the height of the tallest tab plus the indicator weight.
//
// Flutter: material/tabs.dart TabBar.preferredSize
func (w *TabBar) PreferredSize() ui.Size {
	maxHeight := tabHeight
	for _, item := range w.tabs {
		if p, ok := item.(widgets.PreferredSizeWidget); ok {
			maxHeight = math.Max(p.PreferredSize().Height, maxHeight)
		}
	}
	return ui.Size{Height: maxHeight + w.o.IndicatorWeight.Value()}
}

// TabHasTextAndIcon is whether any tab has both a text and an icon.
//
// Flutter: material/tabs.dart TabBar.tabHasTextAndIcon
func (w *TabBar) TabHasTextAndIcon() bool {
	for _, item := range w.tabs {
		if p, ok := item.(widgets.PreferredSizeWidget); ok && p.PreferredSize().Height == tabTextAndIconHeight {
			return true
		}
	}
	return false
}

// CreateState creates the state.
//
// Flutter: material/tabs.dart TabBar.createState
func (w *TabBar) CreateState() widgets.State { return &tabBarState{} }
