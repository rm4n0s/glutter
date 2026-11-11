package widgets_test

import (
	"testing"

	"golang.org/x/image/font/gofont/gomono"

	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/widgets"

	"github.com/rm4n0s/glutter/ui"
)

// IconFont is Glutter's addition (SPEC §12.10): the font of an icon is loaded the first
// time an Icon builds it, once for a binding, and a font that cannot be loaded is reported
// once.

func iconWidgetOf(data IconData) Widget {
	return NewDirectionality(ui.TextDirectionLTR, NewIcon(data))
}

func TestIconWithAFontThatCannotBeLoadedReportsItOnce(t *testing.T) {
	skipUnlessDebug(t)
	calls := 0
	font := &IconFont{Family: "packages/test/Broken", Data: func() []byte { calls++; return []byte("not a font") }}
	icon := IconData{CodePoint: 0xe000, FontFamily: "Broken", FontPackage: "test", Font: font}
	ts := gluttertest.New(t)
	ts.PumpWidget(iconWidgetOf(icon))
	_ = requireErrorTag(t, takeFirst(t, ts), "FontDataInvalid")
	ts.PumpWidget(iconWidgetOf(IconData{CodePoint: 0xe001, FontFamily: "Broken", FontPackage: "test", Font: font}))
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("the font was reported again: %v", errs)
	}
	if calls != 1 {
		t.Errorf("the font data was asked for %d times, want 1", calls)
	}
}

func TestIconFontIsLoadedOncePerBinding(t *testing.T) {
	calls := 0
	font := &IconFont{Family: "TestIcons", Data: func() []byte { calls++; return gomono.TTF }}
	icon := IconData{CodePoint: 0x41, FontFamily: "TestIcons", Font: font}
	for run := 1; run <= 2; run++ {
		ts := gluttertest.New(t)
		ts.PumpWidget(iconWidgetOf(icon))
		ts.PumpWidget(iconWidgetOf(IconData{CodePoint: 0x42, FontFamily: "TestIcons", Font: font}))
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Fatalf("run %d reported %v", run, errs)
		}
		if _, ok := ts.Binding.PlatformDispatcher().ResolveFont("TestIcons", ui.FontWeightW400, ui.FontStyleNormal); !ok {
			t.Errorf("run %d: the font is not loaded", run)
		}
		if calls != run {
			t.Errorf("run %d: the font data was asked for %d times in all, want %d (one for each binding)", run, calls, run)
		}
	}
}

func TestIconDataWithAFontIsComparable(t *testing.T) {
	font := &IconFont{Family: "X", Data: func() []byte { return nil }}
	a := IconData{CodePoint: 1, Font: font}
	b := IconData{CodePoint: 1, Font: font}
	if a != b || a == (IconData{CodePoint: 1, Font: &IconFont{Family: "X"}}) {
		t.Error("IconData does not compare by its font pointer")
	}
}

func TestResolveColorValue(t *testing.T) {
	ts := gluttertest.New(t)
	plain := ui.NewColor(0xFF112233)
	var gotPlain, gotNil ui.Color
	ts.PumpWidget(NewBuilder(func(bc BuildContext) Widget {
		gotPlain = ResolveColorValue(plain, bc)
		gotNil = ResolveColorValue(nil, bc)
		return NewSizedBox(nil)
	}))
	if gotPlain != plain || gotNil != (ui.Color{}) {
		t.Errorf("a plain color resolved to %v and nil to %v", gotPlain, gotNil)
	}
}
