package widgets_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/editable_text_shortcuts_test.dart.
//
// Deviations of the port:
//   - The text offsets are UTF-8 bytes (SPEC §12.10): a family emoji (testCluster)
//     is 18 bytes, and the offsets of the tests that are inside or at the
//     boundary of a cluster are those bytes.
//   - Glutter has only the Linux bindings for every target platform (SPEC §12.10),
//     so the tests of the other platforms' own shortcuts (the Apple Option/Command
//     keys) and the web tests are not ported, and the tests that run on several
//     platforms run on Linux.
//   - There is no Ahem font: the soft wrap tests measure a 20 character line with
//     the bundled font and make the field that wide.

const (
	testText = "Now is the time for\n" + // 20
		"all good people\n" + // 36
		"to come to the aid\n" + // 55
		"of their country." // 72
	testCluster      = "👨‍👩‍👦👨‍👩‍👦👨‍👩‍👦" // 3 × 18 bytes
	testSoftwrapText = "0123456789ABCDEFGHIJ0123456789ABCDEFGHIJ0123456789ABCDEFGHIJ0123456789ABCDEFGHIJ"
	testVerticalText = "1\n2\n3\n4\n5\n6\n7\n8\n9"
)

type fieldOpts struct {
	readOnly, obscured, noInteractive bool
	textAlign                         ui.TextAlign
	width                             float64
	fontSize                          float64
	height                            float64
}

type editableFixture struct {
	ts         *gluttertest.Tester
	controller *TextEditingController
	focus      *FocusNode
	scroll     ScrollController
	opts       fieldOpts
}

func (f *editableFixture) build() Widget {
	o := f.opts
	if o.width == 0 {
		o.width = 201
	}
	if o.height == 0 {
		o.height = 200
	}
	if o.fontSize == 0 {
		o.fontSize = 10
	}
	eo := EditableTextOpts{
		Autofocus: true, CursorWidth: foundation.Some(0.0), KeyboardType: foundation.Some(services.TextInputTypeText),
		ReadOnly: o.readOnly, ObscureText: o.obscured, TextAlign: o.textAlign, ScrollController: f.scroll,
		EnableInteractiveSelection: foundation.Some(!o.noInteractive), TextScaler: painting.TextScalerNoScaling,
		MaxLines: foundation.Some(rendering0MaxLinesUnlimited),
	}
	if o.obscured {
		eo.MaxLines = foundation.Some(1)
	}
	return testWidgetsApp(testWidgetsAppOpts{home: NewAlign(NewSizedBox(NewEditableText(f.controller, f.focus,
		painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Roboto", FontSize: foundation.Some(o.fontSize), Color: foundation.Some(ui.NewColor(0xFF000000))}),
		ui.NewColor(0xFF2196F3), ui.NewColor(0xFF9E9E9E), eo), SizedBoxOpts{Width: foundation.Some(o.width), Height: foundation.Some(o.height)}),
		AlignOpts{Alignment: painting.AlignmentTopLeft})})
}

// rendering0MaxLinesUnlimited is rendering.MaxLinesUnlimited, Flutter's null maxLines.
const rendering0MaxLinesUnlimited = 0

func newEditableFixture(t *testing.T, text string, sel services.TextSelection, opts ...fieldOpts) *editableFixture {
	t.Helper()
	f := &editableFixture{ts: gluttertest.New(t), controller: NewTextEditingController(text), focus: NewFocusNode(FocusNodeOpts{DebugLabel: "editable"}),
		scroll: NewScrollController()}
	if len(opts) > 0 {
		f.opts = opts[0]
	}
	t.Cleanup(func() { f.focus.Dispose(); f.scroll.Dispose(); f.controller.Dispose() })
	f.controller.SetSelection(sel)
	f.ts.PumpWidget(f.build())
	f.ts.Pump()
	return f
}

func collapsed(offset int) services.TextSelection { return services.NewTextSelectionCollapsed(offset) }
func collapsedUp(offset int) services.TextSelection {
	return services.NewTextSelectionCollapsed(offset, services.TextSelectionOpts{Affinity: ui.TextAffinityUpstream})
}
func rangeSel(base, extent int) services.TextSelection {
	return services.NewTextSelection(base, extent)
}

type mods struct{ shift, control, alt, meta bool }

func (m mods) String() string {
	return fmt.Sprintf("shift=%v control=%v alt=%v meta=%v", m.shift, m.control, m.alt, m.meta)
}

// sendKeyCombination presses the key with the modifiers and releases them, then
// pumps a frame.
func sendKeysWith(ts *gluttertest.Tester, key services.LogicalKeyboardKey, m mods) {
	if m.shift {
		ts.SendKeyDownEvent(services.LogicalKeyboardKeyShift)
	}
	if m.control {
		ts.SendKeyDownEvent(services.LogicalKeyboardKeyControl)
	}
	if m.alt {
		ts.SendKeyDownEvent(services.LogicalKeyboardKeyAlt)
	}
	if m.meta {
		ts.SendKeyDownEvent(services.LogicalKeyboardKeyMeta)
	}
	ts.SendKeyEvent(key)
	if m.meta {
		ts.SendKeyUpEvent(services.LogicalKeyboardKeyMeta)
	}
	if m.alt {
		ts.SendKeyUpEvent(services.LogicalKeyboardKeyAlt)
	}
	if m.control {
		ts.SendKeyUpEvent(services.LogicalKeyboardKeyControl)
	}
	if m.shift {
		ts.SendKeyUpEvent(services.LogicalKeyboardKeyShift)
	}
	ts.Pump()
}

func allModifierVariants() []mods {
	var out []mods
	for i := 0; i < 16; i++ {
		out = append(out, mods{i&1 != 0, i&2 != 0, i&4 != 0, i&8 != 0})
	}
	return out
}

func (f *editableFixture) press(key services.LogicalKeyboardKey, m ...mods) {
	var mm mods
	if len(m) > 0 {
		mm = m[0]
	}
	sendKeysWith(f.ts, key, mm)
}

func (f *editableFixture) expect(t *testing.T, text string, sel services.TextSelection) {
	t.Helper()
	if f.controller.Text() != text {
		t.Errorf("text = %q, want %q", f.controller.Text(), text)
	}
	if got := f.controller.Selection(); !got.Equal(sel) {
		t.Errorf("selection = %v, want %v", got, sel)
	}
}

func (f *editableFixture) expectSelection(t *testing.T, sel services.TextSelection, reason ...any) {
	t.Helper()
	if got := f.controller.Selection(); !got.Equal(sel) {
		t.Errorf("selection = %v, want %v %v", got, sel, reason)
	}
}

func TestEditableTextShortcutsDeleteAndBackspace(t *testing.T) {
	type c struct {
		name    string
		text    string
		sel     services.TextSelection
		field   fieldOpts
		key     services.LogicalKeyboardKey
		mods    mods
		want    string
		wantSel services.TextSelection
	}
	remove := func(s string, from, to int) string { return s[:from] + s[to:] }
	cases := []c{
		{"backspace", testText, collapsedUp(20), fieldOpts{}, services.LogicalKeyboardKeyBackspace, mods{}, remove(testText, 19, 20), collapsed(19)},
		{"backspace readonly", testText, collapsedUp(20), fieldOpts{readOnly: true}, services.LogicalKeyboardKeyBackspace, mods{}, testText, collapsedUp(20)},
		{"backspace at start", testText, collapsed(0), fieldOpts{}, services.LogicalKeyboardKeyBackspace, mods{}, testText, collapsed(0)},
		{"backspace at end", testText, collapsedUp(72), fieldOpts{}, services.LogicalKeyboardKeyBackspace, mods{}, remove(testText, 71, 72), collapsed(71)},
		{"backspace inside of a cluster", testCluster, collapsedUp(4), fieldOpts{}, services.LogicalKeyboardKeyBackspace, mods{}, testCluster[:36], collapsed(0)},
		{"backspace at cluster boundary", testCluster, collapsedUp(18), fieldOpts{}, services.LogicalKeyboardKeyBackspace, mods{}, testCluster[:36], collapsed(0)},
		{"delete", testText, collapsedUp(20), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{}, remove(testText, 20, 21), collapsed(20)},
		{"delete readonly", testText, collapsedUp(20), fieldOpts{readOnly: true}, services.LogicalKeyboardKeyDelete, mods{}, testText, collapsedUp(20)},
		{"delete at start", testText, collapsed(0), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{}, testText[1:], collapsed(0)},
		{"delete at end", testText, collapsedUp(72), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{}, testText, collapsedUp(72)},
		{"delete inside of a cluster", testCluster, collapsedUp(4), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{}, testCluster[:36], collapsed(0)},
		{"delete at cluster boundary", testCluster, collapsedUp(18), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{}, testCluster[:36], collapsed(18)},
		{"non-collapsed delete inside of a cluster", testCluster, rangeSel(21, 24), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{}, testCluster[:36], collapsed(18)},
		{"non-collapsed delete at the boundaries of a cluster", testCluster, rangeSel(18, 36), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{}, testCluster[:36], collapsed(18)},
		{"non-collapsed delete cross-cluster", testCluster, rangeSel(2, 20), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{}, testCluster[:18], collapsed(0)},
		{"non-collapsed delete cross-cluster obscured text", testCluster, rangeSel(2, 20), fieldOpts{obscured: true}, services.LogicalKeyboardKeyDelete, mods{}, testCluster[22:], collapsed(0)},
		{"word-backspace", testText, collapsedUp(29), fieldOpts{}, services.LogicalKeyboardKeyBackspace, mods{control: true},
			"Now is the time for\nall people\nto come to the aid\nof their country.", collapsed(24)},
		{"word-backspace readonly", testText, collapsedUp(29), fieldOpts{readOnly: true}, services.LogicalKeyboardKeyBackspace, mods{control: true}, testText, collapsedUp(29)},
		{"word-backspace at start", testText, collapsed(0), fieldOpts{}, services.LogicalKeyboardKeyBackspace, mods{control: true}, testText, collapsed(0)},
		{"word-backspace at end", testText, collapsedUp(72), fieldOpts{}, services.LogicalKeyboardKeyBackspace, mods{control: true},
			"Now is the time for\nall good people\nto come to the aid\nof their ", collapsed(64)},
		{"word-backspace inside of a cluster", testCluster, collapsedUp(4), fieldOpts{}, services.LogicalKeyboardKeyBackspace, mods{control: true}, testCluster[:36], collapsed(0)},
		{"word-backspace at cluster boundary", testCluster, collapsedUp(18), fieldOpts{}, services.LogicalKeyboardKeyBackspace, mods{control: true}, testCluster[:36], collapsed(0)},
		{"word-delete", testText, collapsedUp(23), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{control: true},
			"Now is the time for\nall people\nto come to the aid\nof their country.", collapsed(23)},
		{"word-delete readonly", testText, collapsedUp(23), fieldOpts{readOnly: true}, services.LogicalKeyboardKeyDelete, mods{control: true}, testText, collapsedUp(23)},
		{"word-delete at start", testText, collapsed(0), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{control: true},
			" is the time for\nall good people\nto come to the aid\nof their country.", collapsed(0)},
		{"word-delete at end", testText, collapsedUp(72), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{control: true}, testText, collapsedUp(72)},
		{"word-delete inside of a cluster", testCluster, collapsedUp(4), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{control: true}, testCluster[:36], collapsed(0)},
		{"word-delete at cluster boundary", testCluster, collapsedUp(18), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{control: true}, testCluster[:36], collapsed(18)},
		{"line-backspace", testText, collapsedUp(29), fieldOpts{}, services.LogicalKeyboardKeyBackspace, mods{alt: true},
			"Now is the time for\npeople\nto come to the aid\nof their country.", collapsed(20)},
		{"line-backspace readonly", testText, collapsedUp(29), fieldOpts{readOnly: true}, services.LogicalKeyboardKeyBackspace, mods{alt: true}, testText, collapsedUp(29)},
		{"line-backspace at start", testText, collapsed(0), fieldOpts{}, services.LogicalKeyboardKeyBackspace, mods{alt: true}, testText, collapsed(0)},
		{"line-backspace at end", testText, collapsedUp(72), fieldOpts{}, services.LogicalKeyboardKeyBackspace, mods{alt: true},
			"Now is the time for\nall good people\nto come to the aid\n", collapsed(55)},
		{"line-backspace inside of a cluster", testCluster, collapsedUp(4), fieldOpts{}, services.LogicalKeyboardKeyBackspace, mods{alt: true}, testCluster[:36], collapsed(0)},
		{"line-backspace at cluster boundary", testCluster, collapsedUp(18), fieldOpts{}, services.LogicalKeyboardKeyBackspace, mods{alt: true}, testCluster[:36], collapsed(0)},
		{"line-delete", testText, collapsedUp(23), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{alt: true},
			"Now is the time for\nall\nto come to the aid\nof their country.", collapsed(23)},
		{"line-delete readonly", testText, collapsedUp(23), fieldOpts{readOnly: true}, services.LogicalKeyboardKeyDelete, mods{alt: true}, testText, collapsedUp(23)},
		{"line-delete at start", testText, collapsed(0), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{alt: true},
			"\nall good people\nto come to the aid\nof their country.", collapsed(0)},
		{"line-delete at end", testText, collapsedUp(72), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{alt: true}, testText, collapsedUp(72)},
		{"line-delete inside of a cluster", testCluster, collapsedUp(4), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{alt: true}, "", collapsed(0)},
		{"line-delete at cluster boundary", testCluster, collapsedUp(18), fieldOpts{}, services.LogicalKeyboardKeyDelete, mods{alt: true}, testCluster[:18], collapsed(18)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newEditableFixture(t, tc.text, tc.sel, tc.field)
			f.press(tc.key, tc.mods)
			f.expect(t, tc.want, tc.wantSel)
		})
	}
}

func TestEditableTextShortcutsSoftwrapLineBoundaries(t *testing.T) {
	// A line of 20 characters wraps the text of 80 characters in four lines.
	measure := gluttertest.New(t)
	measure.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewAlign(NewRichText(painting.NewTextSpan(painting.TextSpanOpts{
		Text: foundation.Some(testSoftwrapText[:20]), Style: painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Roboto", FontSize: foundation.Some(10.0)})})),
		AlignOpts{Alignment: painting.AlignmentTopLeft})}))
	width := measure.Size(gluttertest.ByType[*RichText]()).Width + 1
	field := fieldOpts{width: width}
	text := testSoftwrapText
	K := services.LogicalKeyboardKeyBackspace
	t.Run("line-backspace upstream", func(t *testing.T) {
		f := newEditableFixture(t, text, collapsedUp(40), field)
		f.press(K, mods{alt: true})
		f.expect(t, text[:20]+text[40:], collapsed(20))
	})
	t.Run("line-backspace downstream", func(t *testing.T) {
		f := newEditableFixture(t, text, collapsed(40), field)
		f.press(K, mods{alt: true})
		f.expect(t, text, collapsed(40))
	})
	t.Run("line-delete upstream", func(t *testing.T) {
		f := newEditableFixture(t, text, collapsedUp(40), field)
		f.press(services.LogicalKeyboardKeyDelete, mods{alt: true})
		f.expect(t, text, collapsedUp(40))
	})
	t.Run("line-delete downstream", func(t *testing.T) {
		f := newEditableFixture(t, text, collapsed(40), field)
		f.press(services.LogicalKeyboardKeyDelete, mods{alt: true})
		f.expect(t, text[:40]+text[60:], collapsed(40))
	})
}

func TestEditableTextShortcutsArrowMovement(t *testing.T) {
	left, right := services.LogicalKeyboardKeyArrowLeft, services.LogicalKeyboardKeyArrowRight
	t.Run("left at start", func(t *testing.T) {
		f := newEditableFixture(t, testText, collapsed(0))
		for _, m := range allModifierVariants() {
			f.press(left, m)
			f.expectSelection(t, collapsed(0), m)
		}
	})
	t.Run("right at end", func(t *testing.T) {
		f := newEditableFixture(t, testText, collapsed(72))
		for _, m := range allModifierVariants() {
			f.press(right, m)
			if s := f.controller.Selection(); !s.IsCollapsed() || s.BaseOffset != 72 {
				t.Errorf("%v: selection = %v", m, s)
			}
		}
	})
	cases := []struct {
		name string
		sel  services.TextSelection
		key  services.LogicalKeyboardKey
		m    mods
		want services.TextSelection
	}{
		{"left", collapsed(20), left, mods{}, collapsed(19)},
		{"right", collapsed(20), right, mods{}, collapsed(21)},
		{"word left", collapsed(7), left, mods{control: true}, collapsed(4)},
		{"word right", collapsed(7), right, mods{control: true}, collapsed(10)},
		{"line left", collapsed(24), left, mods{alt: true}, collapsed(20)},
		{"line right", collapsed(24), right, mods{alt: true}, collapsedUp(35)},
		// With a selection: the plain arrows collapse to its edge.
		{"left collapses to the start", rangeSel(20, 23), left, mods{}, collapsed(20)},
		{"left collapses a reversed selection to the start", rangeSel(23, 20), left, mods{}, collapsed(20)},
		{"right collapses to the end", rangeSel(20, 23), right, mods{}, collapsed(23)},
		{"right collapses a reversed selection to the end", rangeSel(23, 20), right, mods{}, collapsed(23)},
		// The word and line movements start from the extent of the selection.
		{"word left from a selection", rangeSel(24, 43), left, mods{control: true}, collapsed(39)},
		{"word left from a reversed selection", rangeSel(43, 24), left, mods{control: true}, collapsed(20)},
		{"word right from a selection", rangeSel(24, 43), right, mods{control: true}, collapsed(46)},
		{"word right from a reversed selection", rangeSel(43, 24), right, mods{control: true}, collapsed(28)},
		{"line left from a selection", rangeSel(24, 43), left, mods{alt: true}, collapsed(36)},
		{"line left from a reversed selection", rangeSel(43, 24), left, mods{alt: true}, collapsed(20)},
		{"line right from a selection", rangeSel(24, 43), right, mods{alt: true}, collapsedUp(54)},
		{"line right from a reversed selection", rangeSel(43, 24), right, mods{alt: true}, collapsedUp(35)},
		// Shift extends.
		{"select left", collapsed(5), left, mods{shift: true}, rangeSel(5, 4)},
		{"select right", collapsed(5), right, mods{shift: true}, rangeSel(5, 6)},
		{"select to the line start", collapsed(5), left, mods{shift: true, alt: true}, rangeSel(5, 0)},
		{"select to the line end", collapsed(5), right, mods{shift: true, alt: true}, services.NewTextSelection(5, 19, services.TextSelectionOpts{Affinity: ui.TextAffinityUpstream})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newEditableFixture(t, testText, tc.sel)
			f.press(tc.key, tc.m)
			f.expectSelection(t, tc.want)
		})
	}
	t.Run("select left and right with the selection disabled collapse", func(t *testing.T) {
		f := newEditableFixture(t, "testing", collapsed(5), fieldOpts{noInteractive: true})
		f.press(left, mods{shift: true})
		f.expect(t, "testing", collapsed(4))
		f2 := newEditableFixture(t, "testing", collapsed(5), fieldOpts{noInteractive: true})
		f2.press(right, mods{shift: true})
		f2.expect(t, "testing", collapsed(6))
	})
}

func TestEditableTextShortcutsVerticalMovement(t *testing.T) {
	down, up := services.LogicalKeyboardKeyArrowDown, services.LogicalKeyboardKeyArrowUp
	t.Run("at start", func(t *testing.T) {
		f := newEditableFixture(t, testText, collapsed(0))
		for _, key := range []services.LogicalKeyboardKey{up, services.LogicalKeyboardKeyPageUp} {
			for _, m := range allModifierVariants() {
				f.press(key, m)
				f.expect(t, testText, collapsed(0))
			}
		}
	})
	t.Run("at end", func(t *testing.T) {
		f := newEditableFixture(t, testText, collapsed(72))
		for _, key := range []services.LogicalKeyboardKey{down, services.LogicalKeyboardKeyPageDown} {
			for _, m := range allModifierVariants() {
				f.press(key, m)
				if s := f.controller.Selection(); s.BaseOffset != 72 || s.ExtentOffset != 72 {
					t.Errorf("%v: selection = %v", m, s)
				}
			}
		}
	})
	t.Run("run", func(t *testing.T) {
		f := newEditableFixture(t, "aa\na\naa\naaa\naaaa", collapsed(2))
		steps := []struct {
			key  services.LogicalKeyboardKey
			want services.TextSelection
		}{
			{down, collapsedUp(4)}, {down, collapsedUp(7)}, {down, collapsed(10)}, {down, collapsed(14)}, {down, collapsed(16)},
			{up, collapsed(10)}, {up, collapsedUp(7)}, {up, collapsedUp(4)}, {up, collapsedUp(2)}, {up, collapsed(0)}, {down, collapsedUp(4)},
		}
		for i, s := range steps {
			f.press(s.key)
			f.expectSelection(t, s.want, "step", i)
		}
	})
	t.Run("run with page down and up", func(t *testing.T) {
		f := newEditableFixture(t, "aa\na\naa\naaa\n"+strings.Repeat("aaa\n", 50)+"aaaa", collapsed(2))
		f.press(down)
		f.expectSelection(t, collapsedUp(4))
		f.press(services.LogicalKeyboardKeyPageDown)
		if got := f.controller.Selection().BaseOffset; got <= 4 {
			t.Errorf("page down moved to %d", got)
		}
		f.press(services.LogicalKeyboardKeyPageUp)
		f.expectSelection(t, collapsedUp(4))
	})
	t.Run("run can be interrupted by layout changes", func(t *testing.T) {
		f := newEditableFixture(t, "aa\na\naa\naaa\naaaa", collapsed(2))
		f.press(up)
		f.expectSelection(t, collapsed(0))
		f.opts.textAlign = ui.TextAlignRight
		f.ts.PumpWidget(f.build())
		f.press(down)
		f.expectSelection(t, collapsed(3))
	})
	t.Run("run can be interrupted by selection changes", func(t *testing.T) {
		f := newEditableFixture(t, "aa\na\naa\naaa\naaaa", collapsed(2))
		f.press(up)
		f.expectSelection(t, collapsed(0))
		f.controller.SetSelection(collapsed(1))
		f.ts.Pump()
		f.controller.SetSelection(collapsed(0))
		f.ts.Pump()
		f.press(down)
		f.expectSelection(t, collapsed(3)) // Would have been 4 if the run wasn't interrupted.
	})
	t.Run("does not create an invalid selection", func(t *testing.T) {
		f := newEditableFixture(t, "A", collapsed(1))
		f.controller.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: "AA", Selection: foundation.Some(collapsed(2))}))
		f.press(down)
	})
	t.Run("select up and down", func(t *testing.T) {
		f := newEditableFixture(t, testVerticalText, collapsed(5))
		f.press(up, mods{shift: true})
		f.expectSelection(t, rangeSel(5, 3))
		g := newEditableFixture(t, testVerticalText, collapsed(5))
		g.press(down, mods{shift: true})
		g.expectSelection(t, rangeSel(5, 7))
		h := newEditableFixture(t, testVerticalText, collapsed(5))
		h.press(up, mods{shift: true, alt: true})
		h.expectSelection(t, rangeSel(5, 0))
		i := newEditableFixture(t, testVerticalText, collapsed(5))
		i.press(down, mods{shift: true, alt: true})
		i.expectSelection(t, rangeSel(5, 17))
	})
}

func TestEditableTextShortcutsHomeEndAndPages(t *testing.T) {
	longText := strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. "+
		"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. ", 2)
	field := fieldOpts{fontSize: 12, height: 80}
	home, end := services.LogicalKeyboardKeyHome, services.LogicalKeyboardKeyEnd
	t.Run("end moves the selection to the end of the line, no scroll", func(t *testing.T) {
		f := newEditableFixture(t, longText, collapsed(0), field)
		f.ts.PumpAndSettle()
		before := f.scroll.Offset()
		f.press(end)
		if s := f.controller.Selection(); !s.IsCollapsed() || s.BaseOffset <= 0 {
			t.Errorf("selection = %v", s)
		}
		if f.scroll.Offset() != before {
			t.Errorf("scrolled to %v", f.scroll.Offset())
		}
	})
	t.Run("end with shift highlights to the end of the line", func(t *testing.T) {
		f := newEditableFixture(t, longText, collapsed(0), field)
		f.press(end, mods{shift: true})
		if s := f.controller.Selection(); s.IsCollapsed() || s.BaseOffset != 0 || s.ExtentOffset <= 0 {
			t.Errorf("selection = %v", s)
		}
	})
	t.Run("home moves the selection to the beginning of the line, no scroll", func(t *testing.T) {
		f := newEditableFixture(t, longText, collapsed(len(longText)), field)
		f.ts.PumpAndSettle()
		before := f.scroll.Offset()
		f.press(home)
		if s := f.controller.Selection(); !s.IsCollapsed() || s.BaseOffset >= len(longText) {
			t.Errorf("selection = %v", s)
		}
		if f.scroll.Offset() != before {
			t.Errorf("scrolled to %v", f.scroll.Offset())
		}
	})
	t.Run("page down scrolls down and moves the selection", func(t *testing.T) {
		f := newEditableFixture(t, longText, collapsed(0), field)
		f.ts.PumpAndSettle()
		f.press(services.LogicalKeyboardKeyPageDown)
		if s := f.controller.Selection(); !s.IsCollapsed() || s.BaseOffset <= 0 {
			t.Errorf("selection = %v", s)
		}
		if f.scroll.Offset() <= 0 {
			t.Errorf("did not scroll: %v", f.scroll.Offset())
		}
	})
	t.Run("page up with shift scrolls up and extends the selection", func(t *testing.T) {
		f := newEditableFixture(t, longText, collapsed(len(longText)), field)
		f.ts.PumpAndSettle()
		before := f.scroll.Offset()
		f.press(services.LogicalKeyboardKeyPageUp, mods{shift: true})
		s := f.controller.Selection()
		if s.IsCollapsed() || s.BaseOffset != len(longText) || s.ExtentOffset >= len(longText) {
			t.Errorf("selection = %v", s)
		}
		if f.scroll.Offset() >= before {
			t.Errorf("did not scroll up: %v", f.scroll.Offset())
		}
	})
	t.Run("document boundaries with control", func(t *testing.T) {
		f := newEditableFixture(t, longText, collapsed(0), field)
		f.press(end, mods{control: true})
		f.expectSelection(t, collapsed(len(longText)))
		f.press(home, mods{control: true, shift: true})
		f.expectSelection(t, rangeSel(len(longText), 0))
	})
}

func TestEditableTextShortcutsClipboard(t *testing.T) {
	t.Run("select all", func(t *testing.T) {
		f := newEditableFixture(t, testText, collapsed(0))
		f.press(services.LogicalKeyboardKeyKeyA, mods{control: true})
		f.expectSelection(t, rangeSel(0, len(testText)))
	})
	t.Run("copy", func(t *testing.T) {
		f := newEditableFixture(t, testText, rangeSel(0, 4))
		f.ts.SetClipboardText("empty")
		f.press(services.LogicalKeyboardKeyKeyC, mods{control: true})
		f.ts.FlushPosted()
		if got, _ := f.ts.ClipboardText(); got != "Now " {
			t.Errorf("clipboard = %q", got)
		}
		f.expect(t, testText, rangeSel(0, 4))
	})
	t.Run("copy of obscured text copies nothing", func(t *testing.T) {
		f := newEditableFixture(t, testText, rangeSel(0, 4), fieldOpts{obscured: true})
		f.ts.SetClipboardText("empty")
		f.press(services.LogicalKeyboardKeyKeyC, mods{control: true})
		f.ts.FlushPosted()
		if got, _ := f.ts.ClipboardText(); got != "empty" {
			t.Errorf("clipboard = %q", got)
		}
	})
	t.Run("cut", func(t *testing.T) {
		f := newEditableFixture(t, testText, rangeSel(0, 4))
		f.ts.SetClipboardText("empty")
		f.press(services.LogicalKeyboardKeyKeyX, mods{control: true})
		f.ts.FlushPosted()
		if got, _ := f.ts.ClipboardText(); got != "Now " {
			t.Errorf("clipboard = %q", got)
		}
		f.expect(t, testText[4:], collapsed(0))
	})
	t.Run("cut when read only does nothing", func(t *testing.T) {
		f := newEditableFixture(t, testText, rangeSel(0, 4), fieldOpts{readOnly: true})
		f.ts.SetClipboardText("empty")
		f.press(services.LogicalKeyboardKeyKeyX, mods{control: true})
		f.ts.FlushPosted()
		f.expect(t, testText, rangeSel(0, 4))
	})
	t.Run("paste", func(t *testing.T) {
		f := newEditableFixture(t, testText, collapsed(0))
		f.ts.SetClipboardText("some text")
		f.press(services.LogicalKeyboardKeyKeyV, mods{control: true})
		f.ts.FlushPosted()
		f.ts.Pump()
		f.expect(t, "some text"+testText, collapsed(9))
	})
	t.Run("paste replaces the selection", func(t *testing.T) {
		f := newEditableFixture(t, testText, rangeSel(0, 3))
		f.ts.SetClipboardText("Then")
		f.press(services.LogicalKeyboardKeyKeyV, mods{control: true})
		f.ts.FlushPosted()
		f.ts.Pump()
		f.expect(t, "Then"+testText[3:], collapsed(4))
	})
	t.Run("paste when read only does nothing", func(t *testing.T) {
		f := newEditableFixture(t, testText, collapsed(0), fieldOpts{readOnly: true})
		f.ts.SetClipboardText("some text")
		f.press(services.LogicalKeyboardKeyKeyV, mods{control: true})
		f.ts.FlushPosted()
		f.expect(t, testText, collapsed(0))
	})
}

var _ = time.Second
