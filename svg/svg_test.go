package svg

import (
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestParseAndDraw(t *testing.T) {
	const doc = `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="10"><title> Red </title><desc>A red box</desc><rect width="20" height="10" fill="currentColor"/></svg>`
	tests := []struct {
		name      string
		opts      []ParseOpts
		wantColor ui.Color
	}{
		{"the default theme", nil, ui.NewColorFromARGB(255, 0, 0, 0)},
		{"a theme", []ParseOpts{{Theme: SvgTheme{CurrentColor: ui.NewColorFromARGB(255, 255, 0, 0)}}}, ui.NewColorFromARGB(255, 255, 0, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := Parse([]byte(doc), tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			if got := d.Size(); got != (ui.Size{Width: 20, Height: 10}) {
				t.Errorf("Size = %v", got)
			}
			if d.Title() != "Red" || d.Desc() != "A red box" {
				t.Errorf("Title = %q, Desc = %q", d.Title(), d.Desc())
			}
			for name, pic := range map[string]*ui.Picture{
				"Picture": d.Picture(), "PictureAt": d.PictureAt(2), "PictureWith": d.PictureWith(PictureOpts{Scale: 2}),
			} {
				img, err := pic.ToImageSync(20, 10, ui.ToImageOpts{Rasterizer: soft.ImageRasterizer{}})
				if err != nil {
					t.Fatalf("%s: %v", name, err)
				}
				px, err := img.ToByteData(ui.ImageByteFormatRawRGBA)
				if err != nil {
					t.Fatalf("%s: %v", name, err)
				}
				if int(px[0]) != tt.wantColor.Red() || int(px[1]) != tt.wantColor.Green() || int(px[2]) != tt.wantColor.Blue() || px[3] != 255 {
					t.Errorf("%s: pixel = %v, want %v", name, px[:4], tt.wantColor)
				}
			}
		})
	}
	t.Run("two options values panic", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("no panic")
			}
		}()
		_, _ = Parse([]byte(doc), ParseOpts{}, ParseOpts{})
	})
}

func TestSvgThemeDefaults(t *testing.T) {
	black := ui.NewColorFromARGB(255, 0, 0, 0)
	red := ui.NewColorFromARGB(255, 255, 0, 0)
	tests := []struct {
		name string
		in   SvgTheme
		want SvgTheme
	}{
		{"zero", SvgTheme{}, SvgTheme{CurrentColor: black, FontSize: 14, XHeight: 7}},
		{"a font size gives the x-height", SvgTheme{FontSize: 20}, SvgTheme{CurrentColor: black, FontSize: 20, XHeight: 10}},
		{"everything given", SvgTheme{CurrentColor: red, FontSize: 10, XHeight: 3}, SvgTheme{CurrentColor: red, FontSize: 10, XHeight: 3}},
		{"a negative size is the default", SvgTheme{FontSize: -4, XHeight: -1}, SvgTheme{CurrentColor: black, FontSize: 14, XHeight: 7}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.withDefaults(); got != tt.want {
				t.Errorf("withDefaults = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDefaultSvgThemeNotifiesWhenItChanges(t *testing.T) {
	red := SvgTheme{CurrentColor: ui.NewColorFromARGB(255, 255, 0, 0)}
	blue := SvgTheme{CurrentColor: ui.NewColorFromARGB(255, 0, 0, 255)}
	tests := []struct {
		name     string
		old, new SvgTheme
		want     bool
	}{
		{"the same", red, red, false},
		{"another color", red, blue, true},
		{"another font size", SvgTheme{FontSize: 10}, SvgTheme{FontSize: 12}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := NewDefaultSvgTheme(tt.old, widgets.NewSizedBox(nil))
			cur := NewDefaultSvgTheme(tt.new, widgets.NewSizedBox(nil))
			if got := cur.UpdateShouldNotify(old); got != tt.want {
				t.Errorf("UpdateShouldNotify = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultSvgThemeOfAndWrap(t *testing.T) {
	red := SvgTheme{CurrentColor: ui.NewColorFromARGB(255, 255, 0, 0)}
	ts := newTester(t)
	var inside, outside widgets.BuildContext
	ts.PumpWidget(widgets.NewColumn([]widgets.Widget{
		widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget { outside = bc; return widgets.NewSizedBox(nil) }),
		NewDefaultSvgTheme(red, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget { inside = bc; return widgets.NewSizedBox(nil) })),
	}))
	if _, ok := DefaultSvgThemeOf(outside); ok {
		t.Error("a theme was found outside of the widget")
	}
	d, ok := DefaultSvgThemeOf(inside)
	if !ok || d.Theme() != red {
		t.Fatalf("theme inside = %+v, %v", d, ok)
	}
	// Captured by CaptureThemes, a theme is copied to another place of the tree.
	wrapped := widgets.CaptureThemes(inside, nil).Wrap(widgets.NewSizedBox(nil))
	ts.PumpWidget(wrapped)
	if n := ts.Count(gluttertest.ByType[*DefaultSvgTheme]()); n != 1 {
		t.Errorf("%d copies of the theme, want 1", n)
	}
}
