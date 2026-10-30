package widgets_test

import (
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// scrollRangeField is an EditableText of 200 pixels with a scroll controller.
type scrollRangeField struct {
	t          *testing.T
	ts         *gluttertest.Tester
	controller *TextEditingController
	scroll     ScrollController
}

func newScrollRangeField(t *testing.T, text string, multiline bool, physics ScrollPhysics) *scrollRangeField {
	t.Helper()
	f := &scrollRangeField{t: t, ts: gluttertest.New(t), controller: NewTextEditingController(text), scroll: NewScrollController()}
	focus := NewFocusNode(FocusNodeOpts{DebugLabel: "field"})
	opts := EditableTextOpts{ScrollController: f.scroll, ScrollPhysics: physics}
	if multiline {
		opts.MaxLines = foundation.Some(3)
	}
	f.ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCenter(NewSizedBox(
		NewEditableText(f.controller, focus, plainEditableStyle(), ui.NewColor(0xFF000000), ui.NewColor(0xFF888888), opts),
		SizedBoxOpts{Width: foundation.Some(200.0)}))}))
	focus.RequestFocus()
	f.ts.Pump()
	return f
}

func (f *scrollRangeField) pixels() float64 { return f.scroll.Offset() }
func (f *scrollRangeField) max() float64    { return f.scroll.Position().MaxScrollExtent() }

// scrollToEnd moves the caret and the offset to the end of the text.
func (f *scrollRangeField) scrollToEnd() {
	f.controller.SetSelection(services.NewTextSelectionCollapsed(len(f.controller.Text())))
	f.ts.Pump()
	f.scroll.JumpTo(f.max())
	f.ts.Pump()
}

func TestEditableTextOffsetIsInRangeAfterTheTextGetsShorter(t *testing.T) {
	oneLine := strings.Repeat("word ", 80)
	manyLines := strings.Repeat("a line of text\n", 10) + "last"
	tests := []struct {
		name      string
		text      string
		multiline bool
		// shrink changes the text and returns the number of frames to check.
		shrink func(f *scrollRangeField) int
		// scrolled is whether the field is scrolled to the end before the change.
		scrolled bool
	}{
		{"single line, replaced by the controller", oneLine, false, func(f *scrollRangeField) int {
			f.controller.SetText("short")
			return 1
		}, true},
		{"multi line, replaced by the controller", manyLines, true, func(f *scrollRangeField) int {
			f.controller.SetText("short")
			return 1
		}, true},
		{"single line, Backspace held", oneLine, false, func(f *scrollRangeField) int {
			f.ts.SendKeyDownEvent(services.LogicalKeyboardKeyBackspace)
			for i := 0; i < 12; i++ {
				f.ts.SendKeyRepeatEvent(services.LogicalKeyboardKeyBackspace)
				f.ts.Pump(16 * time.Millisecond)
				f.check(i)
			}
			f.ts.SendKeyUpEvent(services.LogicalKeyboardKeyBackspace)
			return 0
		}, true},
		{"multi line, Backspace held", manyLines, true, func(f *scrollRangeField) int {
			f.ts.SendKeyDownEvent(services.LogicalKeyboardKeyBackspace)
			for i := 0; i < 30; i++ {
				f.ts.SendKeyRepeatEvent(services.LogicalKeyboardKeyBackspace)
				f.ts.Pump(16 * time.Millisecond)
				f.check(i)
			}
			f.ts.SendKeyUpEvent(services.LogicalKeyboardKeyBackspace)
			return 0
		}, true},
		{"single line, not scrolled", oneLine, false, func(f *scrollRangeField) int {
			f.controller.SetText("short")
			return 1
		}, false},
		{"multi line, not scrolled", manyLines, true, func(f *scrollRangeField) int {
			f.controller.SetText("short")
			return 1
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newScrollRangeField(t, tt.text, tt.multiline, nil)
			if tt.scrolled {
				f.scrollToEnd()
				if f.max() <= 0 || f.pixels() != f.max() {
					t.Fatalf("the field is not scrolled to the end: pixels %v, max %v", f.pixels(), f.max())
				}
			} else if f.pixels() != 0 {
				t.Fatalf("pixels = %v before the change, want 0", f.pixels())
			}
			frames := tt.shrink(f)
			for i := 0; i < frames; i++ {
				f.ts.Pump(16 * time.Millisecond)
				f.check(i)
			}
			// The offset stays in range while the idle activity settles.
			for i := 0; i < 10; i++ {
				f.ts.Pump(100 * time.Millisecond)
				f.check(100 + i)
			}
		})
	}
}

// check reports a frame in which the offset is out of range.
func (f *scrollRangeField) check(frame int) {
	f.t.Helper()
	if f.pixels() > f.max()+1e-6 || f.pixels() < -1e-6 {
		f.t.Errorf("frame %d: pixels = %v, want within [0, %v]", frame, f.pixels(), f.max())
	}
}

func TestEditableTextOffsetOfADragIsNotCutShort(t *testing.T) {
	tests := []struct {
		name      string
		multiline bool
		text      string
		by        ui.Offset
	}{
		{"single line", false, strings.Repeat("word ", 80), ui.Offset{Dx: -400}},
		{"multi line", true, strings.Repeat("a line of text\n", 10) + "last", ui.Offset{Dy: -400}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newScrollRangeField(t, tt.text, tt.multiline, NewBouncingScrollPhysics())
			f.scrollToEnd()
			g := f.ts.StartGesture(f.ts.Center(gluttertest.ByType[*EditableText]()))
			// The user pulls the text past its end; the physics let it go past.
			for i := 0; i < 5; i++ {
				g.MoveBy(ui.Offset{Dx: tt.by.Dx / 5, Dy: tt.by.Dy / 5})
				f.ts.Pump(16 * time.Millisecond)
			}
			over := f.pixels() - f.max()
			if over <= 0 {
				t.Fatalf("the drag did not overscroll: pixels %v, max %v", f.pixels(), f.max())
			}
			// The text gets shorter under the finger.
			f.controller.SetText(tt.text[:len(tt.text)-1])
			f.ts.Pump(16 * time.Millisecond)
			if f.pixels() <= f.max() {
				t.Errorf("a drag was cut short: pixels %v, max %v", f.pixels(), f.max())
			}
			g.Up()
		})
	}
}
