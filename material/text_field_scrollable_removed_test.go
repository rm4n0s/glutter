package material_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// A TextField in a Scrollable that leaves the tree in the frame after its
// metrics changed (step 27.1): the app shows the list on a wide window and a
// box on a narrow one. gluttertest runs the posted functions inside Pump; the
// run of the real loop is TestScrollableTakenOutAfterItsMetricsChanged in
// internal/engine.
func TestTextFieldInScrollableTakenOutByTheWidth(t *testing.T) {
	tests := []struct {
		name  string
		sizes []ui.Size
	}{
		{"the window narrows", []ui.Size{{Width: 800, Height: 600}, {Width: 200, Height: 600}}},
		{"the window narrows and widens", []ui.Size{{Width: 800, Height: 600}, {Width: 200, Height: 600}, {Width: 800, Height: 600}}},
		{"the window changes height first", []ui.Size{{Width: 800, Height: 600}, {Width: 800, Height: 300}, {Width: 200, Height: 300}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.SetViewSize(tt.sizes[0])
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{
				Body: widgets.NewLayoutBuilder(func(_ widgets.BuildContext, c rendering.BoxConstraints) widgets.Widget {
					if c.MaxWidth < 400 {
						return widgets.NewSizedBox(nil)
					}
					return widgets.NewSingleChildScrollView(widgets.NewColumn([]widgets.Widget{
						widgets.NewSizedBox(NewTextField(), widgets.SizedBoxOpts{Width: foundation.Some(300.0)}),
						widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(2000.0)}),
					}))
				})})}))
			ts.Tap(findField())
			ts.Pump()
			for _, size := range tt.sizes[1:] {
				ts.SetViewSize(size)
				ts.Pump()
				ts.Pump()
			}
			wide := tt.sizes[len(tt.sizes)-1].Width >= 400
			if got := len(ts.Elements(findField())); wide != (got == 1) {
				t.Errorf("%d text fields at the end, want %v", got, wide)
			}
		})
	}
}
