package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

var testIcon = NewIconData(0xE145, IconDataOpts{FontFamily: "MaterialIcons"})

func TestIconDataValue(t *testing.T) {
	tests := []struct {
		name      string
		a, b      IconData
		wantEqual bool
	}{
		{"same", testIcon, NewIconData(0xE145, IconDataOpts{FontFamily: "MaterialIcons"}), true},
		{"code point", testIcon, NewIconData(0xE146, IconDataOpts{FontFamily: "MaterialIcons"}), false},
		{"fallback", NewIconData(1, IconDataOpts{FontFamilyFallback: []string{"a", "b"}}), NewIconData(1, IconDataOpts{FontFamilyFallback: []string{"a", "b"}}), true},
		{"other fallback", NewIconData(1, IconDataOpts{FontFamilyFallback: []string{"a"}}), NewIconData(1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a == tt.b; got != tt.wantEqual {
				t.Errorf("== = %v, want %v", got, tt.wantEqual)
			}
		})
	}
	if got := testIcon.String(); got != "IconData(U+0E145)" {
		t.Errorf("String = %q", got)
	}
	if got := NewIconData(1, IconDataOpts{FontFamilyFallback: []string{"a", "b"}}).FontFamilyFallback(); len(got) != 2 || got[1] != "b" {
		t.Errorf("FontFamilyFallback = %v", got)
	}
}

func TestIconThemeData(t *testing.T) {
	a := NewIconThemeData(IconThemeDataOpts{Size: foundation.Some(10.0), Opacity: foundation.Some(2.0)})
	b := NewIconThemeData(IconThemeDataOpts{Size: foundation.Some(20.0), Fill: foundation.Some(0.5), Color: foundation.Some(ui.NewColor(0xFF00FF00))})
	merged := a.Merge(&b)
	tests := []struct {
		name string
		got  foundation.Opt[float64]
		want foundation.Opt[float64]
	}{
		{"opacity is clamped", a.Opacity(), foundation.Some(1.0)},
		{"merge takes the other's size", merged.Size(), foundation.Some(20.0)},
		{"merge keeps its own opacity", merged.Opacity(), foundation.Some(1.0)},
		{"merge takes the fill", merged.Fill(), foundation.Some(0.5)},
		{"no weight", merged.Weight(), foundation.None[float64]()},
		{"lerp size", IconThemeDataLerp(&a, &b, 0.5).Size(), foundation.Some(15.0)},
		{"copy with", a.CopyWith(IconThemeDataOpts{Size: foundation.Some(3.0)}).Size(), foundation.Some(3.0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
	if a.IsConcrete() || !NewIconThemeDataFallback().IsConcrete() {
		t.Error("IsConcrete is wrong")
	}
	if !a.Equal(a.CopyWith(IconThemeDataOpts{})) || a.Equal(b) {
		t.Error("Equal is wrong")
	}
}

func TestIconThemeDataValidation(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		opts IconThemeDataOpts
	}{
		{"fill too large", IconThemeDataOpts{Fill: foundation.Some(1.5)}},
		{"weight zero", IconThemeDataOpts{Weight: foundation.Some(0.0)}},
		{"optical size zero", IconThemeDataOpts{OpticalSize: foundation.Some(0.0)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := panicOf(func() { NewIconThemeData(tt.opts) })
			if err == nil {
				t.Fatal("no panic")
			}
			_ = requireErrorTag(t, err.(error), "IconThemeDataInvalid")
		})
	}
	err := panicOf(func() { NewIcon(testIcon, IconOpts{Weight: foundation.Some(-1.0)}) })
	if err == nil {
		t.Fatal("no panic for the icon")
	}
	_ = requireErrorTag(t, err.(error), "IconInvalid")
}

func TestIconSizeAndStyle(t *testing.T) {
	tests := []struct {
		name      string
		w         Widget
		wantSize  float64
		wantColor ui.Color
	}{
		{"default theme", NewIcon(testIcon), 24, ui.NewColor(0xFF000000)},
		{"own size and color", NewIcon(testIcon, IconOpts{Size: foundation.Some(40.0), Color: foundation.Some(ui.NewColor(0xFF112233))}), 40, ui.NewColor(0xFF112233)},
		{"icon theme", NewIconTheme(NewIconThemeData(IconThemeDataOpts{Size: foundation.Some(32.0), Color: foundation.Some(ui.NewColor(0xFFAABBCC))}), NewIcon(testIcon)),
			32, ui.NewColor(0xFFAABBCC)},
		{"theme opacity", NewIconTheme(NewIconThemeData(IconThemeDataOpts{Opacity: foundation.Some(0.5), Color: foundation.Some(ui.NewColor(0xFF000000))}), NewIcon(testIcon)),
			24, ui.NewColor(0xFF000000).WithOpacity(0.5)},
		{"merged theme", NewIconThemeMerge(NewIconThemeData(IconThemeDataOpts{Size: foundation.Some(18.0)}),
			NewIconThemeMerge(NewIconThemeData(IconThemeDataOpts{Color: foundation.Some(ui.NewColor(0xFF010203))}), NewIcon(testIcon))), 18, ui.NewColor(0xFF010203)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			pumpLoose(ts, tt.w, ui.TextDirectionLTR)
			if got := ts.Size(gluttertest.ByType[*Icon]()); got != (ui.Size{Width: tt.wantSize, Height: tt.wantSize}) {
				t.Errorf("size = %v, want %v", got, tt.wantSize)
			}
			span := richOf(ts).Text().(*painting.TextSpan)
			if span.Text != foundation.Some(string(rune(0xE145))) {
				t.Errorf("text = %v, want the glyph", span.Text)
			}
			if c := span.Style().Color; c != foundation.Some(tt.wantColor) {
				t.Errorf("color = %v, want %v", c, tt.wantColor)
			}
			if span.Style().FontFamily != "MaterialIcons" {
				t.Errorf("family = %q", span.Style().FontFamily)
			}
			if span.Style().Inherit {
				t.Error("the icon style must not inherit")
			}
		})
	}
}

func TestIconFontVariations(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	pumpLoose(ts, NewIcon(testIcon, IconOpts{Fill: foundation.Some(1.0), Weight: foundation.Some(700.0)}), ui.TextDirectionLTR)
	got := richOf(ts).Text().(*painting.TextSpan).Style().FontVariations
	want := map[string]float64{"FILL": 1, "wght": 700, "GRAD": 0, "opsz": 48}
	if len(got) != len(want) {
		t.Fatalf("variations = %v", got)
	}
	for _, v := range got {
		if want[v.Axis] != v.Value {
			t.Errorf("axis %s = %v, want %v", v.Axis, v.Value, want[v.Axis])
		}
	}
}

func TestIconTextScaling(t *testing.T) {
	tests := []struct {
		name string
		opts IconOpts
		want float64
	}{
		{"not scaled by default", IconOpts{}, 24},
		{"scaled", IconOpts{ApplyTextScaling: foundation.Some(true)}, 48},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(NewMediaQuery(NewMediaQueryData(MediaQueryDataOpts{TextScaler: painting.NewTextScalerLinear(2)}),
				NewDirectionality(ui.TextDirectionLTR, NewAlign(NewIcon(testIcon, tt.opts), AlignOpts{Alignment: painting.AlignmentTopLeft}))))
			if got := ts.Size(gluttertest.ByType[*Icon]()).Width; got != tt.want {
				t.Errorf("width = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIconEmptyAndFinder(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	pumpLoose(ts, NewRow([]Widget{NewIcon(IconData{}, IconOpts{Size: foundation.Some(10.0)}), NewIcon(testIcon)}), ui.TextDirectionLTR)
	if got := ts.Count(gluttertest.ByType[*RichText]()); got != 1 {
		t.Errorf("%d paragraphs, want 1 (the empty icon draws nothing)", got)
	}
	if got := ts.Count(gluttertest.ByIcon(testIcon)); got != 1 {
		t.Errorf("ByIcon found %d, want 1", got)
	}
	if got := ts.Count(gluttertest.ByIcon(NewIconData(1))); got != 0 {
		t.Errorf("ByIcon found %d for another icon", got)
	}
}

func TestIconMirrorsInRightToLeft(t *testing.T) {
	mirrored := NewIconData(0xE145, IconDataOpts{FontFamily: "MaterialIcons", MatchTextDirection: true})
	tests := []struct {
		name string
		dir  ui.TextDirection
		icon IconData
		want int
	}{
		{"ltr", ui.TextDirectionLTR, mirrored, 0},
		{"rtl", ui.TextDirectionRTL, mirrored, 1},
		{"rtl, not mirrored", ui.TextDirectionRTL, testIcon, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			pumpLoose(ts, NewIcon(tt.icon), tt.dir)
			if got := ts.Count(gluttertest.ByType[*Transform]()); got != tt.want {
				t.Errorf("%d transforms, want %d", got, tt.want)
			}
		})
	}
}

func TestIconSemanticLabel(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.EnsureSemantics()
	pumpLoose(ts, NewIcon(testIcon, IconOpts{SemanticLabel: foundation.Some("add")}), ui.TextDirectionLTR)
	if !ts.Exists(gluttertest.BySemanticsLabel("add")) {
		t.Error("no semantics node with the label")
	}
}
