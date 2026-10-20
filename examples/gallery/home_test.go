package main

import (
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
)

func TestHome(t *testing.T) {
	ts := newTester(t)
	for _, text := range []string{"Glutter gallery", "Material 3 in Glutter", "Mode", "Seed color", "Components", "Light", "System", "Dark"} {
		if !ts.Exists(gluttertest.ByText(text)) {
			t.Errorf("the text %q is not shown", text)
		}
	}
	// Every page except the home page has a card in the index, besides its entry in the rail.
	for _, p := range galleryPages[1:] {
		if n := ts.Count(gluttertest.ByText(p.Name)); n != 2 {
			t.Errorf("the name %q is shown %d times, want 2 (the rail and the index)", p.Name, n)
		}
	}
	if appTitle != "Glutter – Material 3 gallery" {
		t.Errorf("the title is %q", appTitle)
	}
}

// TestHomeIndexOpensPages taps the card of every page in the index.
func TestHomeIndexOpensPages(t *testing.T) {
	useLinux(t)
	for _, p := range galleryPages[1:] {
		t.Run(p.Slug, func(t *testing.T) {
			ts := newTester(t)
			card := gluttertest.ByText(p.Components)
			scrollIntoView(ts, card)
			ts.Tap(card)
			ts.PumpAndSettle()
			if ts.Exists(gluttertest.ByText("Material 3 in Glutter")) {
				t.Error("the home page is still open")
			}
			if ts.Exists(gluttertest.ByText(p.Components)) {
				t.Error("the index card is still shown")
			}
		})
	}
}

// scrollIntoView drags the page up until the center of the widget is in the window.
func scrollIntoView(ts *gluttertest.Tester, f gluttertest.Finder) {
	for i := 0; i < 20 && ts.Center(f).Dy > 680; i++ {
		ts.Drag(gluttertest.First(gluttertest.ByType[*m.ListView]()), m.Offset{Dx: 0, Dy: -300})
		ts.PumpAndSettle()
	}
}

func brightnessOf(ts *gluttertest.Tester) m.Brightness {
	return m.ThemeBrightnessOf(ts.Element(gluttertest.First(gluttertest.ByType[*m.AppBar]())))
}

func TestHomeModePicker(t *testing.T) {
	ts := newTester(t)
	tests := []struct {
		tap  string
		want m.Brightness
	}{
		{"Dark", m.BrightnessDark},
		// The test platform is in the light mode.
		{"System", m.BrightnessLight},
		{"Dark", m.BrightnessDark},
		{"Light", m.BrightnessLight},
	}
	for _, tc := range tests {
		ts.Tap(gluttertest.ByText(tc.tap))
		ts.PumpAndSettle()
		if got := brightnessOf(ts); got != tc.want {
			t.Errorf("after %q the brightness is %v, want %v", tc.tap, got, tc.want)
		}
	}
}

func TestHomeSeedPicker(t *testing.T) {
	ts := newTester(t)
	primary := func() m.Color {
		return m.ThemeOf(ts.Element(gluttertest.First(gluttertest.ByType[*m.AppBar]()))).ColorScheme.Primary()
	}
	seen := map[m.Color]string{}
	for _, option := range seedOptions {
		ts.Tap(gluttertest.ByTooltip(option.name))
		ts.PumpAndSettle()
		got := primary()
		if other, dup := seen[got]; dup {
			t.Errorf("%s gives the same primary color as %s", option.name, other)
		}
		seen[got] = option.name
	}
	// The mode and the seed work together: the dark theme has the seed too.
	ts.Tap(gluttertest.ByText("Dark"))
	ts.PumpAndSettle()
	if _, ok := seen[primary()]; ok {
		t.Error("the dark theme has the same primary color as the light one")
	}
}

func TestHomeGolden(t *testing.T) {
	ts := newTester(t)
	ts.Pump(time.Second)
	ts.MatchesGolden("gallery-home")
}

func TestHomeDarkGolden(t *testing.T) {
	ts := newTester(t)
	ts.Tap(gluttertest.ByText("Dark"))
	ts.PumpAndSettle()
	ts.Pump(time.Second)
	ts.MatchesGolden("gallery-home-dark")
}

func openThemesPage(t *testing.T) *gluttertest.Tester {
	t.Helper()
	ts := newTester(t)
	ts.Tap(gluttertest.First(gluttertest.ByText("Component themes")))
	ts.PumpAndSettle()
	return ts
}

func TestThemesPage(t *testing.T) {
	ts := openThemesPage(t)
	ts.Pump(time.Second)
	ts.MatchesGolden("gallery-themes")
}

// TestThemeSamplesBuild builds every sample alone, so that an error in one shows
// with its name.
func TestThemeSamplesBuild(t *testing.T) {
	useLinux(t)
	for _, s := range themeSamples {
		t.Run(s.name, func(t *testing.T) {
			if !strings.HasSuffix(s.name, "Theme") {
				t.Fatalf("the sample %q is not named after a theme", s.name)
			}
			ts := newTester(t)
			ts.PumpWidget(m.NewMaterialApp(m.MaterialAppOpts{Home: m.NewScaffold(m.ScaffoldOpts{Body: m.NewBuilder(func(bc m.BuildContext) m.Widget {
				return m.NewSingleChildScrollView(m.NewPadding(m.NewEdgeInsetsAll(16), s.build(bc, m.ThemeOf(bc).ColorScheme.TertiaryContainer())))
			})})}))
			ts.Pump(300 * time.Millisecond)
		})
	}
}

func TestThemesApplyToTheirComponent(t *testing.T) {
	ts := openThemesPage(t)
	scrollIntoView(ts, gluttertest.ByText("Show a snack bar"))
	ts.Tap(gluttertest.ByText("Show a snack bar"))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("A themed snack bar")) {
		t.Error("the snack bar of the SnackBarTheme sample is not shown")
	}
}

func openMorePage(t *testing.T) *gluttertest.Tester {
	t.Helper()
	ts := newTester(t)
	ts.Tap(gluttertest.First(gluttertest.ByText("More components")))
	ts.PumpAndSettle()
	return ts
}

func TestMorePage(t *testing.T) {
	ts := openMorePage(t)
	ts.Pump(time.Second)
	ts.MatchesGolden("gallery-more")
}

func TestMoreInk(t *testing.T) {
	ts := openMorePage(t)
	if !ts.Exists(gluttertest.ByText("Taps: 0")) {
		t.Fatal("the counter is not shown")
	}
	ts.Tap(gluttertest.ByText("RawMaterialButton"))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Taps: 1")) {
		t.Error("the raw material button did not count the tap")
	}
}
