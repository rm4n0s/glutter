package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
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

// TestWindowDecorationBarGolden draws the bar with widgets in Leading and
// Actions. The image was made before the drag area changed (step 28.18): the
// layout of the bar does not.
func TestWindowDecorationBarGolden(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 100}})
	ts.SetViewConfiguration(func(c *ui.ViewConfiguration) { c.Window = appInfo() })
	swatch := func(w, h float64, c uint32) Widget {
		return NewColoredBox(ui.NewColor(c), NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(w), Height: foundation.Some(h)}))
	}
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewWindowDecoration(NewColoredBox(ui.NewColor(0xFFFFFBFE), nil), WindowDecorationOpts{
		Title:   swatch(80, 14, 0xFF6750A4),
		Leading: []Widget{swatch(24, 24, 0xFFB3261E)},
		Actions: []Widget{swatch(30, 20, 0xFF0061A4), swatch(30, 20, 0xFF386A20)},
	})))
	ts.MatchesGolden("window_decoration_bar")
}

// TestWindowDecorationDragArea checks which part of the bar has the gestures of
// the window: the gap at the start, the title and the empty space around it,
// and not the Leading and Actions widgets (step 28.18).
func TestWindowDecorationDragArea(t *testing.T) {
	const y = 20.0
	// The bar, from the left: a gap of 12, Leading (62 wide), the title and the
	// space after it up to Actions (60 wide), and the three buttons (138).
	leading := NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(62.0), Height: foundation.Some(40.0)})
	actions := NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(60.0), Height: foundation.Some(40.0)})
	const (
		gapX, leadingX, emptyX, actionsX = 9.0, 40.0, 400.0, 800 - 138 - 30.0
	)
	build := func(t *testing.T, info ui.WindowInfo) *gluttertest.Tester {
		ts, body := windowDecorationApp(t, info)
		ts.PumpWidget(NewWindowDecoration(body, WindowDecorationOpts{Leading: []Widget{leading}, Actions: []Widget{actions}}))
		return ts
	}
	for _, tt := range []struct {
		name string
		x    float64 // -1: the center of the title
		move bool
	}{
		{"the gap before Leading", gapX, true},
		{"a Leading widget", leadingX, false},
		{"the title", -1, true},
		{"the empty space between the title and the actions", emptyX, true},
		{"an Actions widget", actionsX, false},
	} {
		at := func(ts *gluttertest.Tester) ui.Offset {
			if tt.x < 0 {
				return ui.Offset{Dx: ts.Center(gluttertest.ByText("My window")).Dx, Dy: y}
			}
			return ui.Offset{Dx: tt.x, Dy: y}
		}
		check := func(t *testing.T, ts *gluttertest.Tester, want ui.WindowActionKind) {
			t.Helper()
			got := ts.WindowActions()
			if !tt.move {
				if len(got) != 0 {
					t.Errorf("actions = %+v, want none", got)
				}
				return
			}
			if len(got) != 1 || got[0].Kind != want {
				t.Errorf("actions = %+v, want one %v", got, want)
			}
		}
		t.Run(tt.name+": a drag", func(t *testing.T) {
			ts := build(t, appInfo())
			ts.DragFrom(at(ts), ui.Offset{Dx: 30, Dy: 10})
			check(t, ts, ui.WindowActionMove)
		})
		t.Run(tt.name+": a double tap", func(t *testing.T) {
			ts := build(t, appInfo())
			ts.TapAt(at(ts))
			ts.Pump(50 * 1000 * 1000)
			ts.TapAt(at(ts))
			ts.PumpAndSettle()
			check(t, ts, ui.WindowActionMaximize)
		})
		t.Run(tt.name+": a right click", func(t *testing.T) {
			ts := build(t, appInfo())
			ts.TapAt(at(ts), gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse, Buttons: foundation.Some(2)})
			ts.PumpAndSettle()
			check(t, ts, ui.WindowActionShowMenu)
			if got := ts.WindowActions(); tt.move && len(got) == 1 && got[0].Position != at(ts) {
				t.Errorf("menu at %v, want %v", got[0].Position, at(ts))
			}
		})
	}
}

// TestWindowDecorationActionsTapAtOnce shows that the double-tap recognizer of
// the bar is not in the arena of a widget in Actions or Leading: its tap comes
// at once, not after the time a second tap could take (step 28.18).
func TestWindowDecorationActionsTapAtOnce(t *testing.T) {
	for _, tt := range []struct {
		name    string
		leading bool
		x       float64
	}{
		{"Actions", false, 800 - 138 - 30},
		{"Leading", true, 40},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts, body := windowDecorationApp(t, appInfo())
			taps := 0
			button := NewGestureDetector(NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(60.0), Height: foundation.Some(40.0)}),
				GestureDetectorOpts{Behavior: rendering.HitTestBehaviorOpaque, OnTap: func() { taps++ }})
			opts := WindowDecorationOpts{Actions: []Widget{button}}
			if tt.leading {
				opts = WindowDecorationOpts{Leading: []Widget{button}}
			}
			ts.PumpWidget(NewWindowDecoration(body, opts))
			ts.TapAt(ui.Offset{Dx: tt.x, Dy: 20})
			if taps != 1 {
				t.Errorf("taps right after the tap = %d, want 1 (no wait for a double tap)", taps)
			}
			ts.PumpAndSettle()
			if taps != 1 {
				t.Errorf("taps = %d, want 1", taps)
			}
			if got := ts.WindowActions(); len(got) != 0 {
				t.Errorf("actions = %+v, want none", got)
			}
		})
	}
}

// barBox is a colored box of a fixed size that a test finds by its key.
func barBox(key string, w, h float64) Widget {
	return NewColoredBox(ui.NewColor(0xFF808080), NewSizedBox(nil, SizedBoxOpts{Key: foundation.NewValueKey(key), Width: foundation.Some(w), Height: foundation.Some(h)}))
}

// TestWindowDecorationLeadingRoom checks the room of the bar: the Actions first,
// then the Leading widgets with what is left after the gap and the Actions, then
// the title area with the rest (step 28.21).
func TestWindowDecorationLeadingRoom(t *testing.T) {
	const gap, actionsW = 12.0, 60.0
	tests := []struct {
		name         string
		leading      []Widget
		noActions    bool
		capabilities ui.WindowCapabilities
		// Where the widget keyed "lead" is, and the title area, in the bar.
		leadWidth, titleX, titleWidth float64
	}{
		{
			name:    "a loose Flexible wider than the bar ends where the Actions begin",
			leading: []Widget{NewFlexible(barBox("lead", 5000, 20))},
			// The bar is 662 wide: 800 less the three buttons.
			capabilities: ui.AllWindowCapabilities,
			leadWidth:    662 - gap - actionsW, titleX: 662 - actionsW, titleWidth: 0,
		},
		{
			name:         "a narrow group takes its own width",
			leading:      []Widget{barBox("lead", 100, 20)},
			capabilities: ui.AllWindowCapabilities,
			leadWidth:    100, titleX: gap + 100, titleWidth: 662 - gap - 100 - actionsW,
		},
		{
			name:         "an Expanded widget takes all the room",
			leading:      []Widget{NewExpanded(barBox("lead", 10, 20))},
			capabilities: ui.AllWindowCapabilities,
			leadWidth:    662 - gap - actionsW, titleX: 662 - actionsW, titleWidth: 0,
		},
		{
			name:         "a widget with no width of its own is limited to the room",
			leading:      []Widget{NewRow([]Widget{NewExpanded(barBox("lead", 10, 20)), NewSpacer()})},
			capabilities: ui.AllWindowCapabilities,
			// The Expanded and the Spacer share the room.
			leadWidth: (662 - gap - actionsW) / 2, titleX: 662 - actionsW, titleWidth: 0,
		},
		{
			name:         "a Row of maximum size without a limit of its own fills the room",
			leading:      []Widget{NewRow([]Widget{NewExpanded(barBox("lead", 10, 20))}, RowOpts{MainAxisSize: rendering.MainAxisSizeMax})},
			capabilities: ui.AllWindowCapabilities,
			leadWidth:    662 - gap - actionsW, titleX: 662 - actionsW, titleWidth: 0,
		},
		{
			name:         "the room follows the window buttons: minimize and close",
			leading:      []Widget{NewExpanded(barBox("lead", 10, 20))},
			capabilities: ui.WindowCapabilities{Minimize: true},
			leadWidth:    800 - 92 - gap - actionsW, titleX: 800 - 92 - actionsW, titleWidth: 0,
		},
		{
			name:         "the room follows the window buttons: close only",
			leading:      []Widget{NewExpanded(barBox("lead", 10, 20))},
			capabilities: ui.WindowCapabilities{},
			leadWidth:    800 - 46 - gap - actionsW, titleX: 800 - 46 - actionsW, titleWidth: 0,
		},
		{
			name:         "an Expanded after a fixed widget takes what the fixed one leaves",
			leading:      []Widget{barBox("fixed", 50, 20), NewExpanded(barBox("lead", 10, 20))},
			capabilities: ui.AllWindowCapabilities,
			leadWidth:    662 - gap - actionsW - 50, titleX: 662 - actionsW, titleWidth: 0,
		},
		{
			name:         "without Actions the Leading widgets have the room up to the buttons",
			leading:      []Widget{NewExpanded(barBox("lead", 10, 20))},
			noActions:    true,
			capabilities: ui.AllWindowCapabilities,
			leadWidth:    662 - gap, titleX: 662, titleWidth: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := appInfo()
			info.Capabilities = tt.capabilities
			ts, body := windowDecorationApp(t, info)
			barWidth := 800.0
			switch tt.capabilities {
			case ui.AllWindowCapabilities:
				barWidth -= 138
			case ui.WindowCapabilities{Minimize: true}:
				barWidth -= 92
			default:
				barWidth -= 46
			}
			opts := WindowDecorationOpts{
				Title:   barBox("title", 80, 14),
				Leading: tt.leading,
			}
			if !tt.noActions {
				opts.Actions = []Widget{barBox("actions", actionsW, 20)}
			}
			ts.PumpWidget(NewWindowDecoration(body, opts))

			lead := gluttertest.ByKey(foundation.NewValueKey("lead"))
			if got := ts.Size(lead).Width; got != tt.leadWidth {
				t.Errorf("Leading widget width = %v, want %v", got, tt.leadWidth)
			}
			// The title area is the Align that holds the title.
			area := gluttertest.First(gluttertest.Ancestor(gluttertest.ByKey(foundation.NewValueKey("title")), gluttertest.ByType[*Align]()))
			if got := ts.TopLeft(area).Dx; got != tt.titleX {
				t.Errorf("title area at %v, want %v", got, tt.titleX)
			}
			if got := ts.Size(area).Width; got != tt.titleWidth {
				t.Errorf("title area width = %v, want %v", got, tt.titleWidth)
			}
			if !tt.noActions {
				actions := gluttertest.ByKey(foundation.NewValueKey("actions"))
				if got := ts.Size(actions).Width; got != actionsW {
					t.Errorf("Actions width = %v, want %v", got, actionsW)
				}
				if got := ts.TopLeft(actions).Dx; got != barWidth-actionsW {
					t.Errorf("Actions at %v, want %v", got, barWidth-actionsW)
				}
			}
			// The Leading group ends where the title area begins.
			if got := ts.TopLeft(lead).Dx; len(tt.leading) == 1 && got != gap {
				t.Errorf("Leading widget at %v, want %v", got, gap)
			}
			// The window buttons stay where they were: a tap on close closes.
			ts.TapAt(ui.Offset{Dx: 800 - 23, Dy: 20})
			if got := ts.WindowActions(); len(got) != 1 || got[0].Kind != ui.WindowActionClose {
				t.Errorf("actions after a tap on close = %+v", got)
			}
		})
	}
}

// TestWindowDecorationBarRightToLeft checks that a right-to-left direction
// mirrors the bar: the gap at the end, then the Leading widgets, the title area,
// and the Actions at the start (step 28.21).
func TestWindowDecorationBarRightToLeft(t *testing.T) {
	const barWidth, height = 400.0, 40.0
	tests := []struct {
		name      string
		direction ui.TextDirection
		// The left edge of the gap, the Leading widget, the title area and the Actions.
		gapX, leadX, titleX, actionsX float64
	}{
		{"left to right", ui.TextDirectionLTR, 0, 12, 12 + 100, 400 - 60},
		{"right to left", ui.TextDirectionRTL, 400 - 12, 400 - 12 - 100, 60, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: barWidth, Height: 100}})
			ts.PumpWidget(NewDirectionality(tt.direction, NewAlign(NewSizedBox(NewWindowBarForTest(
				barBox("gap", 12, height),
				NewAlign(barBox("title", 20, 14), AlignOpts{Alignment: painting.AlignmentCenterLeft}),
				[]Widget{barBox("lead", 100, 20)},
				[]Widget{barBox("actions", 60, 20)},
				height), SizedBoxOpts{Width: foundation.Some(barWidth), Height: foundation.Some(height)}), AlignOpts{Alignment: painting.AlignmentTopLeft})))
			x := func(key string) float64 { return ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey(key))).Dx }
			if got := x("gap"); got != tt.gapX {
				t.Errorf("gap at %v, want %v", got, tt.gapX)
			}
			if got := x("lead"); got != tt.leadX {
				t.Errorf("Leading widget at %v, want %v", got, tt.leadX)
			}
			if got := x("actions"); got != tt.actionsX {
				t.Errorf("Actions at %v, want %v", got, tt.actionsX)
			}
			area := gluttertest.First(gluttertest.Ancestor(gluttertest.ByKey(foundation.NewValueKey("title")), gluttertest.ByType[*Align]()))
			if got := ts.TopLeft(area).Dx; got != tt.titleX {
				t.Errorf("title area at %v, want %v", got, tt.titleX)
			}
			if got := ts.Size(area).Width; got != 400-12-100-60 {
				t.Errorf("title area width = %v, want %v", got, 400-12-100-60)
			}
		})
	}
}

// barTab is a keyed stateful widget that counts the states it gets.
type barTab struct {
	StatefulWidget
	inits *int
}

func (w *barTab) CreateState() State { return &barTabState{} }

type barTabState struct{ StateOf[*barTab] }

func (s *barTabState) InitState() { *s.Widget().inits++ }
func (s *barTabState) Build(BuildContext) Widget {
	return NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(30.0), Height: foundation.Some(20.0)})
}

// TestWindowDecorationLeadingKeepsKeyedState checks that a keyed Leading widget
// keeps its state when the widgets are reordered, although the bar wraps each
// one to limit its width (step 28.21).
func TestWindowDecorationLeadingKeepsKeyedState(t *testing.T) {
	ts, body := windowDecorationApp(t, appInfo())
	inits := 0
	tab := func(name string) Widget {
		return &barTab{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: foundation.NewValueKey(name)}), inits: &inits}
	}
	ts.PumpWidget(NewWindowDecoration(body, WindowDecorationOpts{Leading: []Widget{tab("a"), tab("b")}}))
	ax := ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey("a"))).Dx
	ts.PumpWidget(NewWindowDecoration(body, WindowDecorationOpts{Leading: []Widget{tab("b"), tab("a")}}))
	if got := ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey("a"))).Dx; got <= ax {
		t.Errorf("a is at %v after the swap, was at %v: want it after b", got, ax)
	}
	if inits != 2 {
		t.Errorf("states made = %d, want 2 (a reorder keeps them)", inits)
	}
}
