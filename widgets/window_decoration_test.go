package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// windowDecorationApp is a decoration around a plain widget, in a view that has
// the window info.
func windowDecorationApp(t *testing.T, info ui.WindowInfo) (*gluttertest.Tester, Widget) {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.SetViewConfiguration(func(c *ui.ViewConfiguration) { c.Window = info })
	body := NewDirectionality(ui.TextDirectionLTR, NewText("the app"))
	return ts, body
}

func appInfo() ui.WindowInfo {
	return ui.WindowInfo{Decoration: ui.DecorationApp, Capabilities: ui.AllWindowCapabilities, Title: "My window", Activated: true}
}

func TestWindowDecorationOnlyWhenTheAppDrawsIt(t *testing.T) {
	tests := []struct {
		mode      ui.DecorationMode
		wantTitle bool
	}{
		{ui.DecorationApp, true},
		{ui.DecorationServerSide, false},
		{ui.DecorationLibdecor, false},
		{ui.DecorationNone, false},
	}
	for _, tt := range tests {
		t.Run(tt.mode.Name(), func(t *testing.T) {
			info := appInfo()
			info.Decoration = tt.mode
			ts, body := windowDecorationApp(t, info)
			ts.PumpWidget(NewWindowDecoration(body))
			if got := ts.Exists(gluttertest.ByText("My window")); got != tt.wantTitle {
				t.Errorf("title shown = %v, want %v", got, tt.wantTitle)
			}
			if !ts.Exists(gluttertest.ByText("the app")) {
				t.Error("the child is not shown")
			}
		})
	}
}

func TestWindowDecorationButtons(t *testing.T) {
	const closeX, y = 800 - 23.0, 20.0
	tests := []struct {
		name   string
		mutate func(*ui.WindowInfo)
		x      float64
		want   ui.WindowActionKind
	}{
		{"close", func(*ui.WindowInfo) {}, closeX, ui.WindowActionClose},
		{"maximize", func(*ui.WindowInfo) {}, closeX - 46, ui.WindowActionMaximize},
		{"minimize", func(*ui.WindowInfo) {}, closeX - 92, ui.WindowActionMinimize},
		{"restore when maximized", func(i *ui.WindowInfo) { i.Maximized = true }, closeX - 46, ui.WindowActionRestore},
		{"close without the other buttons", func(i *ui.WindowInfo) { i.Capabilities = ui.WindowCapabilities{} }, closeX, ui.WindowActionClose},
		{"maximize moves over when there is no minimize", func(i *ui.WindowInfo) { i.Capabilities.Minimize = false }, closeX - 46, ui.WindowActionMaximize},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := appInfo()
			tt.mutate(&info)
			ts, body := windowDecorationApp(t, info)
			ts.PumpWidget(NewWindowDecoration(body))
			ts.TapAt(ui.Offset{Dx: tt.x, Dy: y})
			got := ts.WindowActions()
			if len(got) != 1 || got[0].Kind != tt.want {
				t.Errorf("actions = %+v, want one %v", got, tt.want)
			}
		})
	}
}

func TestWindowDecorationButtonsFollowTheCapabilities(t *testing.T) {
	info := appInfo()
	info.Capabilities = ui.WindowCapabilities{}
	ts, body := windowDecorationApp(t, info)
	ts.PumpWidget(NewWindowDecoration(body))
	// Only the close button is there, so a tap where maximize would be does nothing.
	ts.TapAt(ui.Offset{Dx: 800 - 23 - 46, Dy: 20})
	if got := ts.WindowActions(); len(got) != 0 {
		t.Errorf("actions = %+v, want none", got)
	}
}

func TestWindowDecorationTitleBarGestures(t *testing.T) {
	t.Run("a drag of the bar moves the window", func(t *testing.T) {
		ts, body := windowDecorationApp(t, appInfo())
		ts.PumpWidget(NewWindowDecoration(body))
		ts.DragFrom(ui.Offset{Dx: 300, Dy: 20}, ui.Offset{Dx: 50, Dy: 10})
		got := ts.WindowActions()
		if len(got) != 1 || got[0].Kind != ui.WindowActionMove {
			t.Errorf("actions = %+v, want one move", got)
		}
	})
	tests := []struct {
		name      string
		maximized bool
		want      ui.WindowActionKind
	}{
		{"a double tap maximizes", false, ui.WindowActionMaximize},
		{"a double tap restores a maximized window", true, ui.WindowActionRestore},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := appInfo()
			info.Maximized = tt.maximized
			ts, body := windowDecorationApp(t, info)
			ts.PumpWidget(NewWindowDecoration(body))
			ts.TapAt(ui.Offset{Dx: 300, Dy: 20})
			ts.Pump(50 * 1000 * 1000)
			ts.TapAt(ui.Offset{Dx: 300, Dy: 20})
			ts.PumpAndSettle()
			got := ts.WindowActions()
			if len(got) != 1 || got[0].Kind != tt.want {
				t.Errorf("actions = %+v, want one %v", got, tt.want)
			}
		})
	}
	t.Run("a right click opens the window menu where it was made", func(t *testing.T) {
		ts, body := windowDecorationApp(t, appInfo())
		ts.PumpWidget(NewWindowDecoration(body))
		ts.TapAt(ui.Offset{Dx: 300, Dy: 20}, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse, Buttons: foundation.Some(2)})
		ts.PumpAndSettle()
		got := ts.WindowActions()
		if len(got) != 1 || got[0].Kind != ui.WindowActionShowMenu || got[0].Position != (ui.Offset{Dx: 300, Dy: 20}) {
			t.Errorf("actions = %+v, want the window menu at 300,20", got)
		}
	})
}

func TestWindowDecorationResize(t *testing.T) {
	tests := []struct {
		name string
		at   ui.Offset
		want ui.ResizeEdge
	}{
		{"left", ui.Offset{Dx: 2, Dy: 300}, ui.ResizeEdgeLeft},
		{"right", ui.Offset{Dx: 797, Dy: 300}, ui.ResizeEdgeRight},
		{"bottom", ui.Offset{Dx: 400, Dy: 597}, ui.ResizeEdgeBottom},
		{"top", ui.Offset{Dx: 400, Dy: 2}, ui.ResizeEdgeTop},
		{"top left corner", ui.Offset{Dx: 2, Dy: 2}, ui.ResizeEdgeTopLeft},
		{"top right corner", ui.Offset{Dx: 797, Dy: 2}, ui.ResizeEdgeTopRight},
		{"bottom left corner", ui.Offset{Dx: 2, Dy: 597}, ui.ResizeEdgeBottomLeft},
		{"bottom right corner", ui.Offset{Dx: 797, Dy: 597}, ui.ResizeEdgeBottomRight},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, body := windowDecorationApp(t, appInfo())
			ts.PumpWidget(NewWindowDecoration(body))
			g := ts.StartGesture(tt.at)
			g.Up()
			got := ts.WindowActions()
			if len(got) != 1 || got[0].Kind != ui.WindowActionResize || got[0].Edge != tt.want {
				t.Errorf("actions = %+v, want a resize from %v", got, tt.want)
			}
		})
	}
	for _, name := range []string{"maximized", "fullscreen", "tiled"} {
		t.Run("no resize when "+name, func(t *testing.T) {
			info := appInfo()
			info.Maximized, info.Fullscreen, info.Tiled = name == "maximized", name == "fullscreen", name == "tiled"
			ts, body := windowDecorationApp(t, info)
			ts.PumpWidget(NewWindowDecoration(body))
			ts.StartGesture(ui.Offset{Dx: 797, Dy: 597}).Up()
			if got := ts.WindowActions(); len(got) != 0 {
				t.Errorf("actions = %+v, want none", got)
			}
		})
	}
}

func TestWindowDecorationKeepsTheStateOfTheChild(t *testing.T) {
	// The compositor tells the mode after the first frame, so the child is built
	// before the title bar exists and must survive it.
	ts, _ := windowDecorationApp(t, ui.WindowInfo{Decoration: ui.DecorationNone})
	var increment StateSetter
	count := 0
	child := NewDirectionality(ui.TextDirectionLTR, NewStatefulBuilder(func(bc BuildContext, setState StateSetter) Widget {
		increment = setState
		return NewText("count " + string(rune('0'+count)))
	}))
	ts.PumpWidget(NewWindowDecoration(child))
	increment(func() { count = 3 })
	ts.Pump()
	if !ts.Exists(gluttertest.ByText("count 3")) {
		t.Fatal("the child did not rebuild")
	}
	built := 0
	_ = built
	before := ts.Element(gluttertest.ByType[*StatefulBuilder]())
	ts.SetViewConfiguration(func(c *ui.ViewConfiguration) { c.Window = appInfo() })
	ts.Pump()
	if !ts.Exists(gluttertest.ByText("My window")) {
		t.Fatal("the title bar did not appear")
	}
	after := ts.Element(gluttertest.ByType[*StatefulBuilder]())
	if before != after {
		t.Error("the element of the child was made again when the title bar appeared")
	}
}

func TestWindowDecorationOptions(t *testing.T) {
	ts, body := windowDecorationApp(t, appInfo())
	ts.PumpWidget(NewWindowDecoration(body, WindowDecorationOpts{
		Title:           NewText("custom"),
		Height:          foundation.Some(30.0),
		BackgroundColor: foundation.Some(ui.NewColor(0xFFFF0000)),
	}))
	if !ts.Exists(gluttertest.ByText("custom")) || ts.Exists(gluttertest.ByText("My window")) {
		t.Error("the custom title is not the one shown")
	}
	bar := ts.Size(gluttertest.First(gluttertest.Ancestor(gluttertest.ByText("custom"), gluttertest.ByType[*SizedBox]())))
	if bar.Height != 30 {
		t.Errorf("bar height = %v, want 30", bar.Height)
	}
	_ = painting.TextStyle{}
}
