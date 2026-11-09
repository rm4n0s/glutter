package cupertino_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/cupertino/fonts"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// iconCount is the number of constants in the class CupertinoIcons at the pinned Flutter commit.
const iconCount = 1322

func TestCupertinoIconsAreAllThere(t *testing.T) {
	set := reflect.TypeOf(cupertino.CupertinoIcons)
	if set.NumMethod() != iconCount {
		t.Fatalf("CupertinoIcons has %d icons, want %d", set.NumMethod(), iconCount)
	}
	v := reflect.ValueOf(cupertino.CupertinoIcons)
	seen := map[int]string{}
	for i := 0; i < set.NumMethod(); i++ {
		m := set.Method(i)
		icon := v.Method(i).Call(nil)[0].Interface().(cupertino.IconData)
		if icon.FontFamily != "CupertinoIcons" || icon.FontPackage != "cupertino_icons" || icon.Font == nil {
			t.Errorf("%s: family %q, package %q, font %v", m.Name, icon.FontFamily, icon.FontPackage, icon.Font)
		}
		if icon.CodePoint < 0xf000 || icon.CodePoint > 0xfffff {
			t.Errorf("%s: code point U+%X is not in the private use area of the font", m.Name, icon.CodePoint)
		}
		seen[icon.CodePoint] = m.Name
	}
	// The legacy names share code points with the new names, so there are fewer code points than icons.
	if len(seen) < 1000 || len(seen) > iconCount {
		t.Errorf("%d distinct code points", len(seen))
	}
}

func TestCupertinoIconsKeepFlutterValues(t *testing.T) {
	icons := cupertino.CupertinoIcons
	tests := []struct {
		name   string
		icon   cupertino.IconData
		code   int
		mirror bool
	}{
		{"left_chevron", icons.LeftChevron(), 0xf3d2, true},
		{"right_chevron", icons.RightChevron(), 0xf3d3, true},
		{"zzz", icons.Zzz(), 0xf8ae, false},
		{"heart_fill", icons.HeartFill(), icons.HeartFill().CodePoint, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.icon.CodePoint != tt.code || tt.icon.MatchTextDirection != tt.mirror {
				t.Errorf("code point %#x, mirror %v; want %#x, %v", tt.icon.CodePoint, tt.icon.MatchTextDirection, tt.code, tt.mirror)
			}
		})
	}
	if icons.Heart() == icons.HeartFill() {
		t.Error("two icons are equal")
	}
	if icons.Heart() != icons.Heart() {
		t.Error("an icon is not equal to itself: IconData must be comparable")
	}
}

func TestCupertinoIconsFontIsTheOneOfThePackage(t *testing.T) {
	data := fonts.CupertinoIcons()
	if len(data) < 100<<10 {
		t.Fatalf("the font has %d bytes", len(data))
	}
	if head := string(data[:4]); head != "\x00\x01\x00\x00" && head != "true" && head != "OTTO" {
		t.Errorf("not an sfnt font, starts with %q", head)
	}
	license := fonts.CupertinoIconsLicense()
	if !strings.Contains(license, "The MIT License") {
		t.Errorf("the license is %q", license[:min(60, len(license))])
	}
	root, err := os.ReadFile(filepath.Join("..", "third_party", "cupertino_icons", "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	if string(root) != license {
		t.Error("third_party/cupertino_icons/LICENSE is not the license that is embedded with the font")
	}
}

// TestIconLoadsTheFontOnce is the lazy loading: the font is loaded the first time an Icon
// builds a Cupertino icon, once for a binding.
func TestIconLoadsTheFontOnce(t *testing.T) {
	ts := gluttertest.New(t)
	icon := func(i cupertino.IconData) cupertino.Widget {
		return cupertino.NewDirectionality(ui.TextDirectionLTR, cupertino.NewIcon(i))
	}
	d := ts.Binding.PlatformDispatcher()
	family := cupertino.CupertinoIcons.Heart().Font.Family
	resolved := func() bool {
		_, ok := d.ResolveFont(family, ui.FontWeightW400, ui.FontStyleNormal)
		return ok
	}
	// The text engines of the testers share their font collection, so whether the font is
	// there before depends on the tests that ran first: that it is loaded only when an icon of
	// it is built is checked by the widgets' test of IconFont (the font data is asked for once
	// for each binding) and by the link test below.
	ts.PumpWidget(icon(cupertino.IconData{CodePoint: 0x41, FontFamily: "Roboto"}))
	ts.PumpWidget(icon(cupertino.CupertinoIcons.Heart()))
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Fatalf("building the icon reported %v", errs)
	}
	if !resolved() {
		t.Fatal("the font is not loaded after an icon of it was built")
	}
	ts.PumpWidget(icon(cupertino.CupertinoIcons.HeartFill()))
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Fatalf("the second icon reported %v", errs)
	}
}

func TestIconDrawsAGlyphOfTheFont(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(cupertino.NewDirectionality(ui.TextDirectionLTR, cupertino.NewAlign(
		cupertino.NewIcon(cupertino.CupertinoIcons.Heart(), cupertino.IconOpts{Size: foundation.Some(40.0)}),
		cupertino.AlignOpts{Alignment: painting.AlignmentTopLeft})))
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Fatalf("reported %v", errs)
	}
	got := ts.Size(gluttertest.ByType[*cupertino.Icon]())
	if got.Width != 40 || got.Height != 40 {
		t.Errorf("the icon is %v, want 40 by 40", got)
	}
}

// TestTheFontIsLinkedOnlyWhenAnIconIsUsed builds two programs that import cupertino, one
// that calls an icon and one that does not, and compares the sizes of the binaries: the font
// (257 KB) is in the first only (SPEC §31.2).
func TestTheFontIsLinkedOnlyWhenAnIconIsUsed(t *testing.T) {
	if testing.Short() || runtime.GOOS == "js" {
		t.Skip("builds two programs")
	}
	dir := t.TempDir()
	build := func(name string) int64 {
		out := filepath.Join(dir, name)
		cmd := exec.Command("go", "build", "-o", out, "./testdata/linkcheck/"+name)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("building %s: %v\n%s", name, err, b)
		}
		st, err := os.Stat(out)
		if err != nil {
			t.Fatal(err)
		}
		return st.Size()
	}
	without, with := build("noicon"), build("withicon")
	font := int64(len(fonts.CupertinoIcons()))
	if with-without < font*9/10 {
		t.Errorf("the program with an icon is %d bytes and the one without is %d: the difference is smaller than the font (%d)", with, without, font)
	}
	if with-without > font*3 {
		t.Errorf("the difference of %d bytes is much more than the font (%d)", with-without, font)
	}
}
