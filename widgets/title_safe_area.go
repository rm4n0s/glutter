package widgets

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// Title describes the application to the system: the title that the task
// switcher shows and the primary color. It builds its child.
//
// Flutter: widgets/title.dart Title
type Title struct {
	StatefulWidget
	title string
	color ui.Color
	child Widget
}

// TitleOpts are the optional parameters of NewTitle.
//
// Flutter: widgets/title.dart Title.new
type TitleOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Title is the label of the application.
	Title string
}

// NewTitle creates the widget. In debug builds it panics with the tag
// TitleColorNotOpaque when the color is not opaque.
//
// Flutter: widgets/title.dart Title.new
func NewTitle(color ui.Color, child Widget, opts ...TitleOpts) *Title {
	o := firstOpts(opts)
	if foundation.DebugMode && color.Alpha() != 0xFF {
		panic(errors.New(tagTitleColorNotOpaque, "the color of a Title must be opaque", "color", color.String()))
	}
	return &Title{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), o.Title, color, child}
}

// Title is the label of the application.
//
// Flutter: widgets/title.dart Title.title
func (w *Title) Title() string { return w.title }

// Color is the primary color of the application.
//
// Flutter: widgets/title.dart Title.color
func (w *Title) Color() ui.Color { return w.color }

// CreateState creates the state.
//
// Flutter: widgets/title.dart Title.createState
func (w *Title) CreateState() State { return &titleState{} }

type titleState struct{ StateOf[*Title] }

func (s *titleState) InitState() { s.updateChrome() }

func (s *titleState) DidUpdateWidget(old *Title) {
	if old.title != s.Widget().title || old.color != s.Widget().color {
		s.updateChrome()
	}
}

func (s *titleState) updateChrome() {
	w := s.Widget()
	WidgetsBindingOf(s.Context()).SetApplicationSwitcherDescription(ApplicationSwitcherDescription{Label: w.title, PrimaryColor: w.color})
}

func (s *titleState) Build(BuildContext) Widget { return s.Widget().child }

// SafeAreaOpts are the optional parameters of NewSafeArea.
//
// Flutter: widgets/safe_area.dart SafeArea.new
type SafeAreaOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Left says whether to avoid system intrusions on the left; the default is
	// true.
	Left foundation.Opt[bool]
	// Top says whether to avoid system intrusions at the top; the default is true.
	Top foundation.Opt[bool]
	// Right says whether to avoid system intrusions on the right; the default is
	// true.
	Right foundation.Opt[bool]
	// Bottom says whether to avoid system intrusions at the bottom; the default
	// is true.
	Bottom foundation.Opt[bool]
	// Minimum is the minimum padding to apply.
	Minimum painting.EdgeInsets
	// MaintainBottomViewPadding says whether to use the view padding instead of
	// the padding at the bottom, which does not change when the keyboard shows.
	MaintainBottomViewPadding bool
}

// SafeArea is a widget that insets its child by sufficient padding to avoid
// intrusions by the operating system.
//
// Flutter: widgets/safe_area.dart SafeArea
type SafeArea struct {
	StatelessWidget
	child Widget
	o     SafeAreaOpts
}

// NewSafeArea creates the widget.
//
// Flutter: widgets/safe_area.dart SafeArea.new
func NewSafeArea(child Widget, opts ...SafeAreaOpts) *SafeArea {
	o := firstOpts(opts)
	return &SafeArea{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), child, o}
}

// Build pads the child and removes the padding from the MediaQuery below.
//
// Flutter: widgets/safe_area.dart SafeArea.build
func (w *SafeArea) Build(bc BuildContext) Widget {
	DebugCheckHasMediaQuery(bc)
	o := w.o
	left, top, right, bottom := o.Left.OrElse(true), o.Top.OrElse(true), o.Right.OrElse(true), o.Bottom.OrElse(true)
	padding := MediaQueryPaddingOf(bc)
	if o.MaintainBottomViewPadding {
		padding = padding.CopyWith(painting.EdgeInsetsCopy{Bottom: foundation.Some(MediaQueryViewPaddingOf(bc).Bottom)})
	}
	side := func(enabled bool, value, minimum float64) float64 {
		if !enabled {
			value = 0
		}
		return math.Max(value, minimum)
	}
	return NewPadding(painting.EdgeInsets{
		Left: side(left, padding.Left, o.Minimum.Left), Top: side(top, padding.Top, o.Minimum.Top),
		Right: side(right, padding.Right, o.Minimum.Right), Bottom: side(bottom, padding.Bottom, o.Minimum.Bottom),
	}, NewMediaQueryRemovePadding(bc, RemoveSides{Left: left, Top: top, Right: right, Bottom: bottom}, w.child))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTitleColorNotOpaque = "TitleColorNotOpaque"
)
