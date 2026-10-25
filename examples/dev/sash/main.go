// Command sash is the manual check of step 27.7 (the cursor of a pressed pointer): a split view
// whose sash keeps the resize cursor while the mouse button is held, whatever the mouse moves
// over. The right panel is a region with a cursor of its own (the hand), so that it is visible
// when the pointer outruns the sash.
//
// Run it with: go run ./examples/dev/sash
//
// Press on the sash (the thin bar between the panels) and drag fast to the left and to the right,
// and out of the window: the cursor stays the resize arrows until you let go, and the hand shows
// again only over the right panel after that.
package main

import (
	"log"

	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
)

func main() {
	err := m.RunApp(NewSashApp(), m.RunAppOpts{
		AppID:  "dev.glutter.sash",
		Window: m.WindowConfig{Title: "Sash", Size: m.Size{Width: 720, Height: 360}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// SashApp is the app.
type SashApp struct{ m.StatelessWidget }

// NewSashApp creates the app.
func NewSashApp() *SashApp { return &SashApp{} }

// Build builds the app.
func (a *SashApp) Build(m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Sash",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       m.NewScaffold(m.ScaffoldOpts{Body: NewSplit()}),
	})
}

// Split is a split view: the left panel has a width that the sash changes.
type Split struct{ m.StatefulWidget }

// NewSplit creates the split view.
func NewSplit() *Split { return &Split{StatefulWidget: m.NewStatefulWidgetBase()} }

// CreateState creates the state.
func (*Split) CreateState() m.State { return &splitState{width: 240} }

type splitState struct {
	m.StateOf[*Split]
	width float64
}

func (s *splitState) Build(m.BuildContext) m.Widget {
	panel := func(color m.Color, label string) m.Widget {
		return m.NewColoredBox(color, m.NewCenter(m.NewText(label)))
	}
	sash := m.NewMouseRegion(
		m.NewGestureDetector(
			m.NewSizedBox(m.NewColoredBox(m.Colors.Blue.Primary, nil), m.SizedBoxOpts{Width: m.Some(6.0)}),
			m.GestureDetectorOpts{OnHorizontalDragUpdate: func(d m.DragUpdateDetails) {
				s.SetState(func() { s.width = max(60, s.width+d.Delta.Dx) })
			}}),
		m.MouseRegionOpts{Cursor: services.SystemMouseCursors.ResizeLeftRight, HoldCursorWhilePressed: true})
	return m.NewRow([]m.Widget{
		m.NewSizedBox(panel(m.Colors.Grey.Shade200(), "left"), m.SizedBoxOpts{Width: m.Some(s.width)}),
		sash,
		m.NewExpanded(m.NewMouseRegion(panel(m.Colors.Amber.Shade100(), "right (hand cursor)"),
			m.MouseRegionOpts{Cursor: services.SystemMouseCursors.Click})),
	})
}
