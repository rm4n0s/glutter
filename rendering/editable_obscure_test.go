package rendering_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// obscureCases are texts that the obscured editable has to place the caret in:
// ASCII, a precomposed and a combining accent, and an emoji joined by ZWJs.
var obscureCases = []struct {
	name, text string
}{
	{"ascii", "secret"},
	{"two byte letters", "pässwörd"},
	{"a combining accent", "née"},
	{"a ZWJ emoji", "a👨‍👩‍👧b"},
	{"cjk", "中文测试"},
}

var obscuringChars = []string{"•", "*"}

// obscuredPair is an obscured editable on text, and the editable that shows the
// same string of obscuring characters as plain text, which is where the caret
// of the first one has to be.
type obscuredPair struct {
	obscured, plain *rendering.RenderEditable
	delegate        *fakeEditableTextState
	harness         *renderingtest.Harness
	bounds          []int // the byte offsets of the graphemes of the text
	char            string
}

func newObscuredPair(t *testing.T, text, char string) *obscuredPair {
	t.Helper()
	bounds := foundation.GraphemeBoundaries(text)
	shown := strings.Repeat(char, len(bounds)-1)
	delegate := newFakeState()
	delegate.value = services.NewTextEditingValue(services.TextEditingValueOpts{Text: text})
	constraints := withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 100})
	obscured := newEditableOffset(editableSpan(shown, 14), rendering.NewViewportOffsetZero(), delegate, rendering.RenderEditableOpts{
		ObscureText: true, ObscuringCharacter: char, ObscuredText: foundation.Some(text), HasFocus: true,
		EnableInteractiveSelection: foundation.Some(true),
		Selection:                  collapsedAt(0), CursorColor: foundation.Some(red), ShowCursor: foundation.NewValueNotifier(true),
	})
	h := newHarness(t)
	layout(t, h, obscured, atPhase(constraints, phaseCompositingBits))
	plain := newEditable(editableSpan(shown, 14), rendering.RenderEditableOpts{Selection: collapsedAt(0)})
	layout(t, newHarness(t), plain, atPhase(constraints, phaseCompositingBits))
	return &obscuredPair{obscured: obscured, plain: plain, delegate: delegate, harness: h, bounds: bounds, char: char}
}

// shownOffset is the offset in the string of obscuring characters of the
// boundary of the grapheme with the index.
func (p *obscuredPair) shownOffset(grapheme int) int { return grapheme * len(p.char) }

func TestEditableObscuredCaret(t *testing.T) {
	for _, char := range obscuringChars {
		for _, tc := range obscureCases {
			t.Run(tc.name+" "+char, func(t *testing.T) {
				p := newObscuredPair(t, tc.text, char)
				last := len(p.bounds) - 1
				first := p.plain.GetLocalRectForCaret(ui.NewTextPosition(0))
				for g := 0; g <= last; g++ {
					want := p.plain.GetLocalRectForCaret(ui.NewTextPosition(p.shownOffset(g)))
					if got := p.obscured.GetLocalRectForCaret(ui.NewTextPosition(p.bounds[g])); got != want {
						t.Errorf("caret before grapheme %d (offset %d) = %v, want %v", g, p.bounds[g], got, want)
					}
					if g > 0 && want.Left <= first.Left {
						t.Fatalf("the reference caret of grapheme %d is not to the right of the first: %v", g, want)
					}
					// An offset inside a grapheme is the start of the grapheme.
					for inside := p.bounds[g] + 1; g < last && inside < p.bounds[g+1]; inside++ {
						if got := p.obscured.GetLocalRectForCaret(ui.NewTextPosition(inside)); got != want {
							t.Errorf("caret at offset %d, inside grapheme %d = %v, want the one at its start %v", inside, g, got, want)
						}
					}
				}
			})
		}
	}
}

func TestEditableObscuredSelectionBoxes(t *testing.T) {
	for _, char := range obscuringChars {
		for _, tc := range obscureCases {
			t.Run(tc.name+" "+char, func(t *testing.T) {
				p := newObscuredPair(t, tc.text, char)
				last := len(p.bounds) - 1
				for a := 0; a < last; a++ {
					for b := a + 1; b <= last; b++ {
						want := p.plain.GetBoxesForSelection(services.NewTextSelection(p.shownOffset(a), p.shownOffset(b)))
						if len(want) == 0 {
							t.Fatalf("the reference has no boxes for %d..%d", a, b)
						}
						// The base after the extent selects the same boxes.
						for _, sel := range []services.TextSelection{
							services.NewTextSelection(p.bounds[a], p.bounds[b]), services.NewTextSelection(p.bounds[b], p.bounds[a]),
						} {
							if got := p.obscured.GetBoxesForSelection(sel); !reflect.DeepEqual(got, want) {
								t.Errorf("boxes of %v (graphemes %d..%d) = %v, want %v", sel, a, b, got, want)
							}
						}
						gotPoints := p.obscured.GetEndpointsForSelection(services.NewTextSelection(p.bounds[a], p.bounds[b]))
						wantPoints := p.plain.GetEndpointsForSelection(services.NewTextSelection(p.shownOffset(a), p.shownOffset(b)))
						if !reflect.DeepEqual(gotPoints, wantPoints) {
							t.Errorf("endpoints of graphemes %d..%d = %v, want %v", a, b, gotPoints, wantPoints)
						}
					}
				}
				// The composing range is the same map.
				got, ok := p.obscured.GetRectForComposingRange(ui.NewTextRange(p.bounds[0], p.bounds[last]))
				want, wantOK := p.plain.GetRectForComposingRange(ui.NewTextRange(0, p.shownOffset(last)))
				if ok != wantOK || got != want {
					t.Errorf("composing rect = %v %v, want %v %v", got, ok, want, wantOK)
				}
			})
		}
	}
}

// grapheme returns the box of one bullet of the plain editable.
func (p *obscuredPair) bullet(t *testing.T, grapheme int) ui.TextBox {
	t.Helper()
	boxes := p.plain.GetBoxesForSelection(services.NewTextSelection(p.shownOffset(grapheme), p.shownOffset(grapheme+1)))
	if len(boxes) != 1 {
		t.Fatalf("bullet %d has %d boxes", grapheme, len(boxes))
	}
	return boxes[0]
}

func TestEditableObscuredTap(t *testing.T) {
	for _, char := range obscuringChars {
		for _, tc := range obscureCases {
			t.Run(tc.name+" "+char, func(t *testing.T) {
				p := newObscuredPair(t, tc.text, char)
				last := len(p.bounds) - 1
				for g := 0; g < last; g++ {
					box := p.bullet(t, g)
					y := (box.Top + box.Bottom) / 2
					width := box.Right - box.Left
					// The left quarter of a bullet is the boundary before it, the right
					// quarter the boundary after it.
					for _, tap := range []struct {
						name string
						x    float64
						want int
					}{
						{"left", box.Left + width/4, p.bounds[g]},
						{"right", box.Left + 3*width/4, p.bounds[g+1]},
					} {
						pos := p.obscured.GetPositionForPoint(globalOffsetOf(p.obscured, ui.Offset{Dx: tap.x, Dy: y}))
						if pos.Offset != tap.want {
							t.Errorf("a tap on the %s of bullet %d = offset %d, want %d", tap.name, g, pos.Offset, tap.want)
						}
					}
				}
				// A tap past the last bullet is the end of the text.
				box := p.bullet(t, last-1)
				if pos := p.obscured.GetPositionForPoint(globalOffsetOf(p.obscured, ui.Offset{Dx: box.Right + 50, Dy: box.Top + 1})); pos.Offset != len(tc.text) {
					t.Errorf("a tap past the text = offset %d, want %d", pos.Offset, len(tc.text))
				}

				// SelectPositionAt hands the selection to the field in offsets of the text.
				first, lastBox := p.bullet(t, 0), p.bullet(t, last-1)
				y := (first.Top + first.Bottom) / 2
				p.obscured.SelectPositionAt(globalOffsetOf(p.obscured, ui.Offset{Dx: first.Left + 1, Dy: y}), foundation.Some(globalOffsetOf(p.obscured, ui.Offset{Dx: lastBox.Right - 1, Dy: y})), services.SelectionChangedCauseTap)
				if sel := p.delegate.selection.Value(); sel.BaseOffset != 0 || sel.ExtentOffset != len(tc.text) {
					t.Errorf("selection after a drag over all the bullets = %v, want 0..%d", sel, len(tc.text))
				}
			})
		}
	}
}

func TestEditableObscuredWordsAndLines(t *testing.T) {
	for _, tc := range obscureCases {
		t.Run(tc.name, func(t *testing.T) {
			p := newObscuredPair(t, tc.text, "•")
			box := p.bullet(t, 0)
			point := globalOffsetOf(p.obscured, ui.Offset{Dx: box.Left + 1, Dy: (box.Top + box.Bottom) / 2})
			p.obscured.SelectWordsInRange(point, foundation.None[ui.Offset](), services.SelectionChangedCauseLongPress)
			if sel := p.delegate.selection.Value(); sel.Start() != 0 || sel.End() != len(tc.text) {
				t.Errorf("the word of an obscured field = %v, want all the text 0..%d", sel, len(tc.text))
			}
			if line := p.obscured.GetLineAtOffset(ui.NewTextPosition(p.bounds[1])); line.Start() != 0 || line.End() != len(tc.text) {
				t.Errorf("line = %v, want all the text 0..%d", line, len(tc.text))
			}
			if word := p.obscured.GetWordAtOffset(ui.NewTextPosition(len(tc.text) + 5)); word.BaseOffset != len(tc.text) || !word.IsCollapsed() {
				t.Errorf("a word past the end = %v, want a caret at %d", word, len(tc.text))
			}
			// The boundaries are offsets of the text, on graphemes.
			word := p.obscured.GetWordBoundary(ui.NewTextPosition(p.bounds[1]))
			if !isBoundary(p.bounds, word.Start) || !isBoundary(p.bounds, word.End) {
				t.Errorf("word boundary %v is not on graphemes %v", word, p.bounds)
			}
		})
	}
}

func isBoundary(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func TestEditableObscuredVerticalMovementAndSemantics(t *testing.T) {
	for _, tc := range obscureCases {
		t.Run(tc.name, func(t *testing.T) {
			p := newObscuredPair(t, tc.text, "•")
			last := len(p.bounds) - 1
			// The text is on one line: the position above and below the caret of a
			// grapheme is a boundary of the text, not of the bullets.
			for g := 0; g <= last; g++ {
				for _, pos := range []ui.TextPosition{
					p.obscured.GetTextPositionAbove(ui.NewTextPosition(p.bounds[g])),
					p.obscured.GetTextPositionBelow(ui.NewTextPosition(p.bounds[g])),
					p.obscured.StartVerticalCaretMovement(ui.NewTextPosition(p.bounds[g])).Current(),
				} {
					if !isBoundary(p.bounds, pos.Offset) {
						t.Errorf("a vertical movement from grapheme %d ended at %d, which is not a boundary of %v", g, pos.Offset, p.bounds)
					}
				}
			}

			// The semantics: the value is the bullets, so its selection is in their offsets.
			for g := 0; g <= last; g++ {
				p.obscured.SetSelection(foundation.Some(services.NewTextSelectionCollapsed(p.bounds[g])))
				config := semantics.NewSemanticsConfiguration()
				p.obscured.DescribeSemanticsConfiguration(config)
				if got, want := config.AttributedValue().String, strings.Repeat("•", last); got != want {
					t.Fatalf("value = %q, want %q", got, want)
				}
				if sel := config.TextSelection().Value(); sel.BaseOffset != g*len("•") || sel.ExtentOffset != g*len("•") {
					t.Errorf("semantics selection for the caret before grapheme %d = %v, want %d", g, sel, g*len("•"))
				}
				if g < last {
					config.OnMoveCursorForwardByCharacter()(false)
					if sel := p.delegate.selection.Value(); sel.ExtentOffset != p.bounds[g+1] {
						t.Errorf("forward by character from grapheme %d = %v, want %d", g, sel, p.bounds[g+1])
					}
				}
				if g > 0 {
					config.OnMoveCursorBackwardByCharacter()(false)
					if sel := p.delegate.selection.Value(); sel.ExtentOffset != p.bounds[g-1] {
						t.Errorf("backward by character from grapheme %d = %v, want %d", g, sel, p.bounds[g-1])
					}
				}
				// A selection that the screen reader sets is in the offsets of the bullets.
				config.OnSetSelection()(services.NewTextSelection(0, g*len("•")))
				if sel := p.delegate.selection.Value(); sel.BaseOffset != 0 || sel.ExtentOffset != p.bounds[g] {
					t.Errorf("set selection 0..%d = %v, want 0..%d", g*len("•"), sel, p.bounds[g])
				}
			}
		})
	}
}

func TestEditableObscuredTextChange(t *testing.T) {
	// The map follows the text of the field: it is worked out again when it changes.
	p := newObscuredPair(t, "ab", "•")
	p.obscured.SetText(editableSpan("•••", 14))
	p.obscured.SetObscuredText(foundation.Some("äöü"))
	p.delegate.value = services.NewTextEditingValue(services.TextEditingValueOpts{Text: "äöü"})
	pumpFrame(p.harness, phaseCompositingBits)
	plain := newEditable(editableSpan("•••", 14))
	layout(t, newHarness(t), plain, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 100}), phaseCompositingBits))
	for g, offset := range []int{0, 2, 4, 6} {
		want := plain.GetLocalRectForCaret(ui.NewTextPosition(g * len("•")))
		if got := p.obscured.GetLocalRectForCaret(ui.NewTextPosition(offset)); got != want {
			t.Errorf("caret at %d = %v, want %v", offset, got, want)
		}
	}
	// An obscured editable without the text of the field uses the offsets as they are.
	p.obscured.SetObscuredText(foundation.None[string]())
	for _, offset := range []int{0, 3, 6} {
		want := plain.GetLocalRectForCaret(ui.NewTextPosition(offset))
		if got := p.obscured.GetLocalRectForCaret(ui.NewTextPosition(offset)); got != want {
			t.Errorf("caret at %d without the text = %v, want %v", offset, got, want)
		}
	}
}
