package material_test

import (
	"image/color"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

var (
	red   = ui.NewColor(0xFFFF0000)
	blue  = ui.NewColor(0xFF0000FF)
	white = ui.NewColor(0xFFFFFFFF)
)

// box wraps a Material in a white page: the Material is a 100x100 square in the
// middle of the 800x600 view, at x 350..450 and y 250..350.
func box(m widgets.Widget) widgets.Widget {
	return materialApp(NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(false)}),
		widgets.NewColoredBox(white, widgets.NewCenter(widgets.NewSizedBox(m, widgets.SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}))))
}

func rgba(c ui.Color) color.RGBA {
	return color.RGBA{R: uint8(c.Red()), G: uint8(c.Green()), B: uint8(c.Blue()), A: uint8(c.Alpha())}
}

func TestMaterialPaintsItsColor(t *testing.T) {
	tests := []struct {
		name   string
		m      *Material
		inside color.RGBA
		corner color.RGBA
	}{
		{"canvas", NewMaterial(nil, MaterialOpts{Color: foundation.Some(red)}), rgba(red), rgba(red)},
		{"transparency draws nothing", NewMaterial(nil, MaterialOpts{Type: MaterialTypeTransparency, Color: foundation.Some(red)}), rgba(white), rgba(white)},
		{"circle clips the corners", NewMaterial(nil, MaterialOpts{Type: MaterialTypeCircle, Color: foundation.Some(red)}), rgba(red), rgba(white)},
		{"border radius clips the corners", NewMaterial(nil, MaterialOpts{BorderRadius: painting.NewBorderRadiusCircular(40), Color: foundation.Some(red)}), rgba(red), rgba(white)},
		{"shape clips the corners", NewMaterial(nil, MaterialOpts{Shape: painting.NewCircleBorder(), Color: foundation.Some(red)}), rgba(red), rgba(white)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(box(tt.m))
			ts.PumpAndSettle()
			if got := pixel(ts, 400, 300); got != tt.inside {
				t.Errorf("center = %v, want %v", got, tt.inside)
			}
			if got := pixel(ts, 352, 252); got != tt.corner {
				t.Errorf("corner = %v, want %v", got, tt.corner)
			}
			if got := pixel(ts, 100, 100); got != rgba(white) {
				t.Errorf("outside = %v, want the page", got)
			}
		})
	}
}

func TestMaterialShadowFollowsElevation(t *testing.T) {
	tests := []struct {
		name      string
		elevation float64
		shadow    bool
	}{
		{"flat", 0, false},
		{"raised", 8, true},
		{"high", 24, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(box(NewMaterial(nil, MaterialOpts{Color: foundation.Some(red), Elevation: tt.elevation, ShadowColor: foundation.Some(ui.NewColor(0xFF000000))})))
			ts.PumpAndSettle()
			below := pixel(ts, 400, 356)
			if shadowed := below != rgba(white); shadowed != tt.shadow {
				t.Errorf("below the material = %v, shadow = %v, want %v", below, shadowed, tt.shadow)
			}
		})
	}
}

func TestMaterialDarkThemeElevationOverlay(t *testing.T) {
	// A dark Material 2 theme lightens a surface-colored material with its elevation.
	theme := NewThemeData(ThemeDataOpts{Brightness: ui.BrightnessDark, UseMaterial3: foundation.Some(false), ApplyElevationOverlayColor: foundation.Some(true)})
	surface := theme.ColorScheme.Surface()
	got := map[float64]color.RGBA{}
	for _, elevation := range []float64{0, 1, 12} {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(materialApp(theme, widgets.NewCenter(widgets.NewSizedBox(
			NewMaterial(nil, MaterialOpts{Color: foundation.Some(surface), Elevation: elevation}),
			widgets.SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}))))
		ts.PumpAndSettle()
		got[elevation] = pixel(ts, 400, 300)
	}
	if got[0] != rgba(surface) {
		t.Errorf("flat = %v, want the surface %v", got[0], rgba(surface))
	}
	if !(got[0].R < got[1].R && got[1].R < got[12].R) {
		t.Errorf("the overlay does not grow with the elevation: %v %v %v", got[0], got[1], got[12])
	}
}

func TestMaterialSurfaceTintForMaterial3(t *testing.T) {
	theme := NewThemeData(ThemeDataOpts{ColorSchemeSeed: foundation.Some(ui.NewColor(0xFF6750A4))})
	surface := theme.ColorScheme.Surface()
	var flat, raised color.RGBA
	for i, elevation := range []float64{0, 12} {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(materialApp(theme, widgets.NewCenter(widgets.NewSizedBox(
			NewMaterial(nil, MaterialOpts{Color: foundation.Some(surface), SurfaceTintColor: foundation.Some(theme.ColorScheme.SurfaceTint()), Elevation: elevation}),
			widgets.SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}))))
		ts.PumpAndSettle()
		if i == 0 {
			flat = pixel(ts, 400, 300)
		} else {
			raised = pixel(ts, 400, 300)
		}
	}
	if flat != rgba(surface) {
		t.Errorf("flat = %v, want %v", flat, rgba(surface))
	}
	if raised == flat {
		t.Error("the surface tint did not change the color of the raised material")
	}
}

func TestMaterialAnimatesElevation(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(box(NewMaterial(nil, MaterialOpts{Color: foundation.Some(red), Elevation: 0, ShadowColor: foundation.Some(ui.NewColor(0xFF000000))})))
	if got := pixel(ts, 400, 356); got != rgba(white) {
		t.Fatalf("before = %v", got)
	}
	ts.PumpWidget(box(NewMaterial(nil, MaterialOpts{Color: foundation.Some(red), Elevation: 24, ShadowColor: foundation.Some(ui.NewColor(0xFF000000))})))
	ts.Pump(ThemeChangeDuration / 2)
	mid := pixel(ts, 400, 356)
	ts.PumpAndSettle()
	end := pixel(ts, 400, 356)
	if mid == rgba(white) || mid == end {
		t.Errorf("mid = %v, end = %v: the shadow should be growing", mid, end)
	}
}

func TestMaterialConstructorPanics(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name  string
		build func()
		route string
	}{
		{"negative elevation", func() { NewMaterial(nil, MaterialOpts{Elevation: -1}) }, "NewMaterial.MaterialElevationNegative"},
		{"shape and border radius", func() {
			NewMaterial(nil, MaterialOpts{Shape: painting.NewCircleBorder(), BorderRadius: painting.NewBorderRadiusCircular(3)})
		}, "NewMaterial.MaterialShapeAndBorderRadius"},
		{"circle with a shape", func() {
			NewMaterial(nil, MaterialOpts{Type: MaterialTypeCircle, Shape: painting.NewCircleBorder()})
		}, "NewMaterial.MaterialCircleWithShape"},
		{"circle with a border radius", func() {
			NewMaterial(nil, MaterialOpts{Type: MaterialTypeCircle, BorderRadius: painting.NewBorderRadiusCircular(3)})
		}, "NewMaterial.MaterialCircleWithShape"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantRoute(t, recoverError(tt.build), tt.route)
		})
	}
}

func TestMaterialOfAndMaybeOf(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	var inside MaterialInkController
	var insideFound, outsideFound bool
	var outsideErr error
	ts.PumpWidget(materialApp(nil, widgets.NewColumn([]widgets.Widget{
		widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			_, outsideFound = MaterialMaybeOf(bc)
			outsideErr = recoverError(func() { MaterialOf(bc) })
			return widgets.NewSizedBox(nil)
		}),
		NewMaterial(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			inside, insideFound = MaterialMaybeOf(bc)
			return widgets.NewSizedBox(nil)
		})),
	})))
	if outsideFound {
		t.Error("MaybeOf found a material outside of one")
	}
	wantRoute(t, outsideErr, "MaterialOf.MaterialNotFound")
	if !insideFound || inside == nil {
		t.Fatal("MaybeOf found no material inside of one")
	}
	if inside.Vsync() == nil {
		t.Error("the controller has no vsync")
	}
}

func TestDebugCheckHasMaterial(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	var err error
	var ok bool
	ts.PumpWidget(materialApp(nil, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		err = recoverError(func() { ok = DebugCheckHasMaterial(bc) })
		return widgets.NewSizedBox(nil)
	})))
	wantRoute(t, err, "DebugCheckHasMaterial.MaterialMissing")
	if ok {
		t.Error("the check passed without a Material")
	}
	ts.PumpWidget(materialApp(nil, NewMaterial(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		err = recoverError(func() { ok = DebugCheckHasMaterial(bc) })
		return widgets.NewSizedBox(nil)
	}))))
	if err != nil || !ok {
		t.Errorf("the check failed inside a Material: %v", err)
	}
}

func TestMaterialTextStyle(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var style painting.TextStyle
	ts.PumpWidget(materialApp(nil, NewMaterial(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		style = *widgets.DefaultTextStyleOf(bc).Style()
		return widgets.NewSizedBox(nil)
	}), MaterialOpts{TextStyle: painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(33.0)})})))
	if style.FontSize.OrElse(0) != 33 {
		t.Errorf("font size = %v, want 33", style.FontSize)
	}
}
