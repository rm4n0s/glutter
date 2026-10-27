package widgets_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// obscuredField pumps an EditableText (or the same one with the text replaced by
// the obscuring characters by hand) with the font size 24 and a visible caret.
type obscuredField struct {
	ts         *gluttertest.Tester
	controller *TextEditingController
	focus      *FocusNode
}

func newObscuredField(t *testing.T, text string, sel services.TextSelection, obscure bool, char string) *obscuredField {
	t.Helper()
	f := &obscuredField{ts: gluttertest.New(t), controller: NewTextEditingController(text), focus: NewFocusNode(FocusNodeOpts{DebugLabel: "editable"})}
	t.Cleanup(func() { f.focus.Dispose(); f.controller.Dispose() })
	f.controller.SetSelection(sel)
	o := EditableTextOpts{
		Autofocus: true, CursorWidth: foundation.Some(2.0), KeyboardType: foundation.Some(services.TextInputTypeText),
		ObscureText: obscure, MaxLines: foundation.Some(1), TextScaler: painting.TextScalerNoScaling,
		SelectionColor: foundation.Some(ui.NewColor(0x802196F3)), EnableInteractiveSelection: foundation.Some(true),
	}
	if obscure && char != "" {
		o.ObscuringCharacter = char
	}
	style := painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Roboto", FontSize: foundation.Some(24.0), Color: foundation.Some(ui.NewColor(0xFF000000))})
	f.ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewAlign(NewSizedBox(NewEditableText(f.controller, f.focus, style,
		ui.NewColor(0xFFE53935), ui.NewColor(0xFF9E9E9E), o), SizedBoxOpts{Width: foundation.Some(240.0), Height: foundation.Some(40.0)}),
		AlignOpts{Alignment: painting.AlignmentTopLeft})}))
	f.ts.Pump()
	return f
}

func (f *obscuredField) render() *EditableTextState {
	return gluttertest.StateOf[*EditableTextState](f.ts, gluttertest.ByType[*EditableText]())
}

var obscuredFieldTexts = []struct {
	name, text string
}{
	{"ascii", "secret"},
	{"two byte letters", "pässwörd"},
	{"a combining accent", "née!"},
	{"a ZWJ emoji", "a👨‍👩‍👧b"},
}

func TestEditableTextObscuredCaretAndTap(t *testing.T) {
	for _, char := range []string{"•", "*"} {
		for _, tc := range obscuredFieldTexts {
			t.Run(tc.name+" "+char, func(t *testing.T) {
				bounds := foundation.GraphemeBoundaries(tc.text)
				last := len(bounds) - 1
				// The reference shows the characters the field shows, as plain text.
				shown := strings.Repeat(char, last)
				ref := newObscuredField(t, shown, collapsed(0), false, "")
				refRender := ref.render().RenderEditable()

				for g := 0; g <= last; g++ {
					f := newObscuredField(t, tc.text, collapsed(bounds[g]), true, char)
					re := f.render().RenderEditable()
					got := re.GetLocalRectForCaret(f.controller.Selection().Extent())
					want := refRender.GetLocalRectForCaret(ui.NewTextPosition(g * len(char)))
					if got != want {
						t.Errorf("caret before grapheme %d (offset %d) = %v, want %v", g, bounds[g], got, want)
					}
				}

				f := newObscuredField(t, tc.text, collapsed(0), true, char)
				re := f.render().RenderEditable()
				// A tap on the left or the right half of each bullet puts the caret on
				// the boundary of the text before or after the grapheme.
				for g := 0; g < last; g++ {
					boxes := refRender.GetBoxesForSelection(services.NewTextSelection(g*len(char), (g+1)*len(char)))
					if len(boxes) != 1 {
						t.Fatalf("bullet %d has %d boxes", g, len(boxes))
					}
					box := boxes[0]
					width := box.Right - box.Left
					for _, tap := range []struct {
						side string
						x    float64
						want int
					}{{"left", box.Left + width/4, bounds[g]}, {"right", box.Left + 3*width/4, bounds[g+1]}} {
						at := re.LocalToGlobal(ui.Offset{Dx: tap.x, Dy: (box.Top + box.Bottom) / 2}, nil)
						f.ts.TapAt(at)
						f.ts.Pump()
						if sel := f.controller.Selection(); !sel.IsCollapsed() || sel.BaseOffset != tap.want {
							t.Errorf("a tap on the %s of bullet %d: selection = %v, want a caret at %d", tap.side, g, sel, tap.want)
						}
					}
				}
			})
		}
	}
}

func TestEditableTextObscuredSelectionBoxes(t *testing.T) {
	for _, tc := range obscuredFieldTexts {
		t.Run(tc.name, func(t *testing.T) {
			bounds := foundation.GraphemeBoundaries(tc.text)
			last := len(bounds) - 1
			shown := strings.Repeat("•", last)
			ref := newObscuredField(t, shown, collapsed(0), false, "")
			refRender := ref.render().RenderEditable()
			f := newObscuredField(t, tc.text, collapsed(0), true, "")
			re := f.render().RenderEditable()
			for a := 0; a < last; a++ {
				for b := a + 1; b <= last; b++ {
					want := refRender.GetBoxesForSelection(services.NewTextSelection(a*len("•"), b*len("•")))
					got := re.GetBoxesForSelection(services.NewTextSelection(bounds[a], bounds[b]))
					if len(got) != len(want) || (len(got) > 0 && got[0] != want[0]) {
						t.Errorf("boxes of graphemes %d..%d = %v, want %v", a, b, got, want)
					}
				}
			}
		})
	}
}

func TestEditableTextObscuredGoldens(t *testing.T) {
	// The caret is after the fourth bullet of eight, between "päss" and "wörd".
	t.Run("caret", func(t *testing.T) {
		f := newObscuredField(t, "pässwörd", collapsed(len("päss")), true, "")
		f.ts.MatchesGolden("editable_text_obscured_caret")
	})
	// The selection covers the bullets of "ssw".
	t.Run("selection", func(t *testing.T) {
		f := newObscuredField(t, "pässwörd", rangeSel(len("pä"), len("pässw")), true, "")
		f.ts.MatchesGolden("editable_text_obscured_selection")
	})
}
