package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

func TestTitleSetsTheApplicationDescription(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	red, blue := ui.NewColor(0xFFFF0000), ui.NewColor(0xFF0000FF)
	pumpLoose(ts, NewTitle(red, newBox(5, 5), TitleOpts{Title: "One"}), ui.TextDirectionLTR)
	want := ApplicationSwitcherDescription{Label: "One", PrimaryColor: red}
	if got := ts.Binding.ApplicationSwitcherDescription(); got != want {
		t.Errorf("description = %+v, want %+v", got, want)
	}
	if s := ts.Size(gluttertest.ByType[*box]()); s != (ui.Size{Width: 5, Height: 5}) {
		t.Errorf("the title does not build its child: %v", s)
	}
	pumpLoose(ts, NewTitle(blue, newBox(5, 5), TitleOpts{Title: "Two"}), ui.TextDirectionLTR)
	want = ApplicationSwitcherDescription{Label: "Two", PrimaryColor: blue}
	if got := ts.Binding.ApplicationSwitcherDescription(); got != want {
		t.Errorf("description after the update = %+v, want %+v", got, want)
	}
}

func TestTitleColorMustBeOpaque(t *testing.T) {
	skipUnlessDebug(t)
	err := panicOf(func() { NewTitle(ui.NewColor(0x80FF0000), nil) })
	if err == nil {
		t.Fatal("no panic")
	}
	_ = requireErrorTag(t, err.(error), "TitleColorNotOpaque")
}

func TestSafeArea(t *testing.T) {
	padding := painting.EdgeInsets{Left: 1, Top: 20, Right: 3, Bottom: 40}
	tests := []struct {
		name     string
		opts     SafeAreaOpts
		viewPad  painting.EdgeInsets
		wantRect ui.Rect // the child's rect in the 800x600 view
	}{
		{"all sides", SafeAreaOpts{}, painting.EdgeInsets{}, ui.Rect{Left: 1, Top: 20, Right: 797, Bottom: 560}},
		{"no top", SafeAreaOpts{Top: foundation.Some(false)}, painting.EdgeInsets{}, ui.Rect{Left: 1, Top: 0, Right: 797, Bottom: 560}},
		{"no left or bottom", SafeAreaOpts{Left: foundation.Some(false), Bottom: foundation.Some(false)}, painting.EdgeInsets{},
			ui.Rect{Left: 0, Top: 20, Right: 797, Bottom: 600}},
		{"minimum", SafeAreaOpts{Minimum: painting.EdgeInsets{Left: 10, Top: 5}}, painting.EdgeInsets{}, ui.Rect{Left: 10, Top: 20, Right: 797, Bottom: 560}},
		{"maintain the bottom view padding", SafeAreaOpts{MaintainBottomViewPadding: true}, painting.EdgeInsets{Bottom: 100},
			ui.Rect{Left: 1, Top: 20, Right: 797, Bottom: 500}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			data := NewMediaQueryData(MediaQueryDataOpts{Size: ui.Size{Width: 800, Height: 600}, Padding: padding, ViewPadding: tt.viewPad})
			ts.PumpWidget(NewMediaQuery(data, NewDirectionality(ui.TextDirectionLTR, NewSafeArea(NewSizedBoxExpand(newLeaf()), tt.opts))))
			p := ts.TopLeft(gluttertest.ByType[*SizedBox]())
			s := ts.Size(gluttertest.ByType[*SizedBox]())
			rect := ui.Rect{Left: p.Dx, Top: p.Dy, Right: p.Dx + s.Width, Bottom: p.Dy + s.Height}
			if rect != tt.wantRect {
				t.Errorf("child rect = %v, want %v", rect, tt.wantRect)
			}
		})
	}
}

func TestSafeAreaRemovesThePaddingBelow(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var inner painting.EdgeInsets
	data := NewMediaQueryData(MediaQueryDataOpts{Size: ui.Size{Width: 800, Height: 600}, Padding: painting.EdgeInsets{Top: 20, Bottom: 10}})
	ts.PumpWidget(NewMediaQuery(data, NewDirectionality(ui.TextDirectionLTR,
		NewSafeArea(NewBuilder(func(bc BuildContext) Widget { inner = MediaQueryPaddingOf(bc); return newLeaf() }), SafeAreaOpts{Bottom: foundation.Some(false)}))))
	if want := (painting.EdgeInsets{Bottom: 10}); inner != want {
		t.Errorf("padding below = %v, want %v", inner, want)
	}
}

func TestSafeAreaNeedsAMediaQuery(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	pumpRaw(ts, NewDirectionality(ui.TextDirectionLTR, NewSafeArea(nil)))
	_ = requireErrorTag(t, ts.TakeError(), "MediaQueryMissing")
}
