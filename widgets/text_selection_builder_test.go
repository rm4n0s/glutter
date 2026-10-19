package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// testSelectionField is a minimal text field: an EditableText inside the gesture
// detector of a TextSelectionGestureDetectorBuilder, which is what a TextField
// is (step 13.x). The tests of the mouse selection use it.
type testSelectionField struct {
	key *GlobalKey[*EditableTextState]
}

func (d *testSelectionField) EditableTextKey() *GlobalKey[*EditableTextState] { return d.key }
func (d *testSelectionField) ForcePressEnabled() bool                         { return false }
func (d *testSelectionField) SelectionEnabled() bool                          { return true }

type selectionFixture struct {
	ts         *gluttertest.Tester
	controller *TextEditingController
	focus      *FocusNode
	builder    *TextSelectionGestureDetectorBuilder
}

func newSelectionFixture(t *testing.T, text string) *selectionFixture {
	t.Helper()
	f := &selectionFixture{ts: gluttertest.New(t), controller: NewTextEditingController(text), focus: NewFocusNode(FocusNodeOpts{DebugLabel: "field"})}
	t.Cleanup(func() { f.focus.Dispose(); f.controller.Dispose() })
	key := NewGlobalKey[*EditableTextState]()
	f.builder = NewTextSelectionGestureDetectorBuilder(&testSelectionField{key: key})
	field := NewEditableText(f.controller, f.focus,
		painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Roboto", FontSize: foundation.Some(14.0), Color: foundation.Some(ui.NewColor(0xFF000000))}),
		ui.NewColor(0xFF2196F3), ui.NewColor(0xFF9E9E9E),
		EditableTextOpts{Key: key, MaxLines: foundation.Some(0), RendererIgnoresPointer: true, CursorWidth: foundation.Some(0.0),
			KeyboardType: foundation.Some(services.TextInputTypeMultiline)})
	f.ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewAlign(NewSizedBox(
		f.builder.BuildGestureDetector(field, TextSelectionGestureDetectorBuildOpts{Behavior: rendering.HitTestBehaviorTranslucent}),
		SizedBoxOpts{Width: foundation.Some(300.0), Height: foundation.Some(100.0)}), AlignOpts{Alignment: painting.AlignmentTopLeft})}))
	f.ts.Pump()
	return f
}

// pointAt is the global position of the middle of the character at the offset.
func (f *selectionFixture) pointAt(offset int) ui.Offset {
	r := findRenderEditable(f.ts)
	left := r.GetLocalRectForCaret(ui.NewTextPosition(offset))
	right := r.GetLocalRectForCaret(ui.NewTextPosition(offset + 1))
	origin := f.ts.TopLeft(gluttertest.ByType[*EditableText]())
	x := left.Left
	if right.Top == left.Top {
		x = (left.Left + right.Left) / 2
	}
	return ui.Offset{Dx: origin.Dx + x, Dy: origin.Dy + left.Top + left.Height()/2}
}

func (f *selectionFixture) click(at ui.Offset, times int) {
	for i := 0; i < times; i++ {
		f.ts.TapAt(at, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
		if i < times-1 {
			f.ts.Pump(50 * time.Millisecond)
		}
	}
	f.ts.Pump(500 * time.Millisecond)
}

func TestTextSelectionGestureDetectorBuilderMouseSelection(t *testing.T) {
	const text = "hello brave world\nsecond line here"
	tests := []struct {
		name   string
		clicks int
		at     int // the offset of the character that is clicked
		want   services.TextSelection
	}{
		{"a click puts the caret", 1, 8, services.NewTextSelectionCollapsed(8)},
		{"a double click selects a word", 2, 8, services.NewTextSelection(6, 11)},
		{"a double click on the first word", 2, 1, services.NewTextSelection(0, 5)},
		{"a double click on the second line", 2, 20, services.NewTextSelection(18, 24)},
		{"a triple click selects the line", 3, 8, services.NewTextSelection(0, 17)},
		{"a triple click on the second line", 3, 20, services.NewTextSelection(18, 34)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newSelectionFixture(t, text)
			f.click(f.pointAt(tc.at), tc.clicks)
			got := f.controller.Selection()
			if tc.want.IsCollapsed() {
				// The caret is put at the closer side of the character.
				if !got.IsCollapsed() || got.BaseOffset < tc.at || got.BaseOffset > tc.at+1 {
					t.Errorf("selection = %v, want a caret near %d", got, tc.at)
				}
			} else if got.Start() != tc.want.Start() || got.End() != tc.want.End() {
				t.Errorf("selection = %v, want %v", got, tc.want)
			}
			if !f.focus.HasFocus() {
				t.Error("the field does not have the focus")
			}
		})
	}
}

func TestTextSelectionGestureDetectorBuilderMouseDragSelects(t *testing.T) {
	f := newSelectionFixture(t, "hello brave world")
	g := f.ts.StartGesture(f.pointAt(0), gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
	f.ts.Pump()
	g.MoveTo(f.pointAt(10))
	f.ts.Pump()
	g.Up()
	f.ts.Pump(500 * time.Millisecond)
	got := f.controller.Selection()
	if got.Start() > 1 || got.End() < 10 || got.End() > 11 || got.IsCollapsed() {
		t.Errorf("selection = %v, want a range from the start to about offset 10", got)
	}
}

func TestTextSelectionGestureDetectorBuilderShiftClickExtends(t *testing.T) {
	f := newSelectionFixture(t, "hello brave world")
	f.click(f.pointAt(2), 1)
	f.ts.SendKeyDownEvent(services.LogicalKeyboardKeyShift)
	f.click(f.pointAt(10), 1)
	f.ts.SendKeyUpEvent(services.LogicalKeyboardKeyShift)
	got := f.controller.Selection()
	if got.IsCollapsed() || got.Start() > 3 || got.End() < 10 || got.End() > 11 {
		t.Errorf("selection = %v, want a range from about offset 2 to about offset 10", got)
	}
}
