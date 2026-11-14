package offscreen_test

import (
	"fmt"
	"image"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/offscreen"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// picturesOf is the pictures of the layers of a scene.
func picturesOf(scene *ui.Scene) map[*ui.Picture]bool {
	out := map[*ui.Picture]bool{}
	var walk func(l ui.Layer)
	walk = func(l ui.Layer) {
		if p, ok := l.(*ui.PictureLayer); ok {
			out[p.Picture] = true
		}
		for _, c := range l.Children() {
			walk(c)
		}
	}
	walk(scene.Root())
	return out
}

// recordedAgain is how many pictures of the scene are not in the scene before it.
func recordedAgain(before, after *ui.Scene) int {
	old := picturesOf(before)
	n := 0
	for p := range picturesOf(after) {
		if !old[p] {
			n++
		}
	}
	return n
}

// TestAPictureRecordedAgainDamagesWhatItChanges is the workbench of step 45.2: rows of
// text in one layer, inside a Container with a decoration and a LayoutBuilder, which a
// rebuild records again as a whole (the LayoutBuilder lays out again, and every layout
// marks its render object for paint, as in Flutter). The pixels a frame redraws are those
// that the new picture draws differently: none for a rebuild that changes nothing, the row
// for a row whose text changed. Before the step the whole layer was redrawn in each case.
func TestAPictureRecordedAgainDamagesWhatItChanges(t *testing.T) {
	const rows, rowHeight = 14, 20.0
	type app struct {
		texts  []string
		border ui.Color
		set    widgets.StateSetter
	}
	build := func(a *app, layoutBuilder bool) widgets.Widget {
		return scaffoldApp(widgets.NewStatefulBuilder(func(_ widgets.BuildContext, set widgets.StateSetter) widgets.Widget {
			a.set = set
			column := func() widgets.Widget {
				lines := make([]widgets.Widget, rows)
				for i := range lines {
					lines[i] = widgets.NewSizedBox(widgets.NewText(a.texts[i]), widgets.SizedBoxOpts{Height: foundation.Some(rowHeight)})
				}
				// The rows are as wide as the body, so that the Container, and its border, keep
				// their size when a text grows.
				return widgets.NewColumn(lines, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
			}
			var body widgets.Widget = column()
			if layoutBuilder {
				body = widgets.NewLayoutBuilder(func(widgets.BuildContext, rendering.BoxConstraints) widgets.Widget { return column() })
			}
			// A new decoration in each build, equal to the one before it unless the border
			// changed.
			return widgets.NewContainer(body, widgets.ContainerOpts{Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{
				Color:  foundation.Some(ui.NewColor(0xFFFFF8E1)),
				Border: painting.NewBorderAll(painting.BorderSideOpts{Color: foundation.Some(a.border), Width: foundation.Some(2.0)}),
			})})
		}))
	}
	tests := []struct {
		name          string
		layoutBuilder bool
		change        func(a *app)
		// wantRecorded says whether the change records a picture again.
		wantRecorded bool
		// wantDamage says whether the change damages something, and maxDamage how many
		// pixels at most.
		wantDamage bool
		maxDamage  int
		// row is the row the damage must stay around, or -1.
		row int
	}{
		{name: "a rebuild of a Container that changes nothing", change: func(*app) {}, row: -1},
		{name: "a rebuild of a LayoutBuilder that changes nothing", layoutBuilder: true, change: func(*app) {}, wantRecorded: true, row: -1},
		{name: "a row whose text changed", change: func(a *app) { a.texts[6] = "row 6 was typed into" },
			wantRecorded: true, wantDamage: true, maxDamage: viewW * 128, row: 6},
		{name: "a row whose text changed under a LayoutBuilder", layoutBuilder: true, change: func(a *app) { a.texts[6] = "row 6 was typed into" },
			wantRecorded: true, wantDamage: true, maxDamage: viewW * 128, row: 6},
		{name: "a decoration that changed", change: func(a *app) { a.border = ui.NewColor(0xFFD32F2F) },
			wantRecorded: true, wantDamage: true, maxDamage: viewW * viewH, row: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &app{border: ui.NewColor(0xFF1976D2)}
			for i := 0; i < rows; i++ {
				a.texts = append(a.texts, fmt.Sprintf("row %d of the list", i))
			}
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: viewW, Height: viewH}})
			ts.PumpWidget(build(a, tt.layoutBuilder))
			ts.Pump()
			first := lastScene(ts)
			det := offscreen.NewDetacher()
			surf := offscreen.NewSurface(viewW, viewH)
			if _, err := surf.Draw(det.Detach(first)); err != nil {
				t.Fatal(err)
			}
			top := ts.TopLeft(gluttertest.ByType[*widgets.Column]()).Dy

			frames := len(ts.Scenes())
			a.set(func() { tt.change(a) })
			ts.Pump()
			scenes := ts.Scenes()[frames:]
			if len(scenes) == 0 {
				if tt.wantRecorded {
					t.Fatal("the change made no frame")
				}
				return
			}
			recorded, damaged := 0, 0
			before := first
			for i, next := range scenes {
				recorded += recordedAgain(before, next)
				before = next
				pixels := append([]byte(nil), surf.Image().Pix...)
				damage, err := surf.Draw(det.Detach(next))
				if err != nil {
					t.Fatal(err)
				}
				outsideUnchanged(t, pixels, surf.Image(), damage)
				want, err := soft.Render(next, viewW, viewH)
				if err != nil {
					t.Fatal(err)
				}
				samePixels(t, fmt.Sprintf("frame %d after the change", i+1), surf.Image(), want)
				damaged += area(damage)
				if tt.row >= 0 {
					// The damage is tiles of 64 pixels around the row.
					y0, y1 := int(top+float64(tt.row)*rowHeight), int(top+float64(tt.row+1)*rowHeight)
					band := image.Rect(0, y0/64*64, viewW, (y1+63)/64*64)
					for _, r := range damage {
						if !r.In(band) {
							t.Errorf("frame %d: the damage %v is outside the tiles of the row, %v", i+1, r, band)
						}
					}
				}
			}
			if (recorded > 0) != tt.wantRecorded {
				t.Errorf("%d pictures were recorded again, want some = %v", recorded, tt.wantRecorded)
			}
			if (damaged > 0) != tt.wantDamage || damaged > tt.maxDamage {
				t.Errorf("%d pixels were damaged, want some = %v and at most %d", damaged, tt.wantDamage, tt.maxDamage)
			}
		})
	}
}
