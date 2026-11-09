//go:build darwin && !ios && glutter_display

package macos

import (
	"testing"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/ui"
)

// textWindow opens a window and gives it the state of a focused text field.
func textWindow(s *session, h *recorder, text string, anchor, cursor int) *window {
	w := inputWindow(s, h)
	h.takeKeys()
	h.takeTexts()
	s.onUI(func() {
		w.SetTextInput(ui.TextInputState{Enabled: true, View: w.id, Text: text, SelectionAnchor: anchor, SelectionCursor: cursor,
			CaretRect: ui.Rect{Left: 40, Top: 30, Right: 41, Bottom: 50}})
	})
	return w
}

func viewRange(w *window, name string) nsRangeVal { return objc.Send[nsRangeVal](w.view, sel(name)) }

func TestViewIsATextInputClient(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := textWindow(s, h, "", 0, 0)
		s.onUI(func() {
			proto := objc.GetProtocol("NSTextInputClient")
			if !send[bool](w.view, "conformsToProtocol:", proto) {
				t.Error("the view does not adopt NSTextInputClient")
			}
			if send[objc.ID](w.view, "inputContext") == 0 {
				t.Error("the view has no input context")
			}
			if send[bool](w.view, "hasMarkedText") {
				t.Error("a new field has marked text")
			}
			if got := viewRange(w, "markedRange"); got.Location != uint64(notFound) {
				t.Errorf("markedRange = %+v, want NSNotFound", got)
			}
			if got := viewRange(w, "selectedRange"); got != (nsRangeVal{0, 0}) {
				t.Errorf("selectedRange = %+v, want {0 0}", got)
			}
			w.Close()
		})
	})
}

func TestScriptedCompositionMakesTheEventsOfAnInputMethod(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := textWindow(s, h, "ab", 2, 2)
		s.onUI(func() { w.p.runDevText(DevEvent{Kind: "composing", Steps: []string{"に", "にほ"}}) })
		got := h.takeTexts()
		if len(got) != 2 || got[0].Preedit != "に" || got[1].Preedit != "にほ" {
			t.Fatalf("events = %+v, want the preedits に and にほ", got)
		}
		if got[0].PreeditCursorBegin != len("に") || got[0].PreeditCursorEnd != len("に") {
			t.Errorf("the cursor of the first preedit is %d-%d, want the end", got[0].PreeditCursorBegin, got[0].PreeditCursorEnd)
		}
		s.onUI(func() {
			if !send[bool](w.view, "hasMarkedText") {
				t.Error("hasMarkedText is false while composing")
			}
			if r := viewRange(w, "markedRange"); r != (nsRangeVal{2, 2}) {
				t.Errorf("markedRange = %+v, want {2 2}", r)
			}
			// The input method reads the text around the composition.
			var actual nsRangeVal
			attr := objc.Send[objc.ID](w.view, sel("attributedSubstringForProposedRange:actualRange:"), uint64(0), uint64(4), &actual)
			if attr == 0 || nsString(send[objc.ID](attr, "string")) != "abにほ" || actual != (nsRangeVal{0, 4}) {
				t.Errorf("substring = %q (%+v), want abにほ {0 4}", nsString(send[objc.ID](attr, "string")), actual)
			}
			w.p.runDevText(DevEvent{Kind: "compose", Steps: []string{"にほん", "日本"}})
			if send[bool](w.view, "hasMarkedText") {
				t.Error("the composition did not end")
			}
		})
		got = h.takeTexts()
		if len(got) != 3 || got[0].Preedit != "にほん" || got[1].Preedit != "日本" || got[2].Commit != "日本" || got[2].Preedit != "" {
			t.Errorf("events = %+v, want the preedits and then the commit of 日本", got)
		}
		s.onUI(w.Close)
	})
}

func TestScriptedTextIsTypedACharacterAtATime(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := textWindow(s, h, "", 0, 0)
		s.onUI(func() { w.p.runDevText(DevEvent{Kind: "text", Text: "a 😀\nb"}) })
		var commits []string
		for _, e := range h.takeTexts() {
			commits = append(commits, e.Commit)
		}
		want := []string{"a", " ", "😀", "b"}
		if len(commits) != len(want) {
			t.Fatalf("commits = %q, want %q (the line break types nothing)", commits, want)
		}
		for i := range want {
			if commits[i] != want[i] {
				t.Errorf("commits = %q, want %q", commits, want)
				break
			}
		}
		s.onUI(w.Close)
	})
}

func TestNoFieldNoInputMethod(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := inputWindow(s, h)
		h.takeTexts()
		s.onUI(func() {
			w.p.runDevText(DevEvent{Kind: "text", Text: "x"})
			w.p.runDevText(DevEvent{Kind: "composing", Steps: []string{"に"}})
		})
		if got := h.takeTexts(); len(got) != 0 {
			t.Errorf("events = %+v, want none: no text field has the focus", got)
		}
		s.onUI(w.Close)
	})
}

func TestCaretRectangleIsInScreenCoordinatesNearTheCaret(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := textWindow(s, h, "abc", 3, 3)
		s.onUI(func() {
			r := objc.Send[nsRect](w.view, sel("firstRectForCharacterRange:actualRange:"), uint64(0), uint64(0), uintptr(0))
			frame := send[nsRect](w.ns, "frame") // in screen coordinates
			content := objc.Send[nsRect](w.ns, sel("contentRectForFrameRect:"), frame)
			if r.Size.Width != 1 || r.Size.Height != 20 {
				t.Errorf("size = %v x %v, want 1 x 20", r.Size.Width, r.Size.Height)
			}
			// The caret is 40 points from the left of the view and its top 30 points from the top.
			if x := r.Origin.X - content.Origin.X; x < 39.5 || x > 40.5 {
				t.Errorf("x in the window = %v, want 40", x)
			}
			top := content.Origin.Y + content.Size.Height - (r.Origin.Y + r.Size.Height)
			if top < 29.5 || top > 30.5 {
				t.Errorf("top in the window = %v, want 30", top)
			}
			// No field: no rectangle.
			w.SetTextInput(ui.TextInputState{})
			if r := objc.Send[nsRect](w.view, sel("firstRectForCharacterRange:actualRange:"), uint64(0), uint64(0), uintptr(0)); r != (nsRect{}) {
				t.Errorf("without a field the rectangle is %+v", r)
			}
			w.Close()
		})
	})
}

func TestCommandsOfTheInputContextAreSelectorsForTheFramework(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := textWindow(s, h, "hello", 5, 5)
		s.onUI(func() {
			w.view.Send(sel("doCommandBySelector:"), sel("moveWordLeft:"))
			w.view.Send(sel("doCommandBySelector:"), sel("insertNewline:")) // the framework does Enter itself
			w.view.Send(sel("doCommandBySelector:"), sel("deleteWordBackward:"))
		})
		got := h.takeTexts()
		if len(got) != 2 || got[0].Selector != "moveWordLeft:" || got[1].Selector != "deleteWordBackward:" || got[0].View != w.id {
			t.Errorf("events = %+v, want the selectors moveWordLeft: and deleteWordBackward:", got)
		}
		// Enter ends a composition.
		s.onUI(func() {
			w.p.runDevText(DevEvent{Kind: "composing", Steps: []string{"かな"}})
			h.takeTexts()
			w.view.Send(sel("doCommandBySelector:"), sel("insertNewline:"))
		})
		if got := h.takeTexts(); len(got) != 1 || got[0].Commit != "かな" {
			t.Errorf("Enter during a composition made %+v, want the commit of かな", got)
		}
		s.onUI(w.Close)
	})
}

func TestAKeyThatNobodyTookGoesToTheInputContext(t *testing.T) {
	h := &recorder{keyHandled: false}
	runApp(t, Options{}, h, func(s *session) {
		w := textWindow(s, h, "", 0, 0)
		s.onUI(func() {
			ev := w.makeKeyEvent(eventTypeKeyDown, 0x00, devBaseFlags, "a", "a")
			w.keyEvent(ev, keyKindDown, true)
		})
		keys := h.takeKeys()
		if len(keys) != 1 || keys[0].Character != "" || keys[0].Physical != 0x70004 {
			t.Errorf("keys = %+v, want the key A without a character: the text comes from the input context", keys)
		}
		got := h.takeTexts()
		if len(got) != 1 || got[0].Commit != "a" {
			t.Errorf("text events = %+v, want the commit of a from the input context", got)
		}
		s.onUI(w.Close)
	})
}

func TestTextOfTheFieldReachesTheInputContextAfterTheAppChangesIt(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := textWindow(s, h, "", 0, 0)
		s.onUI(func() {
			w.p.runDevText(DevEvent{Kind: "composing", Steps: []string{"に"}})
			// The app replaces the text while the input method composes: the composition is dropped.
			w.SetTextInput(ui.TextInputState{Enabled: true, View: w.id, Text: "new", SelectionAnchor: 3, SelectionCursor: 3,
				CaretRect: ui.Rect{Left: 1, Top: 1, Right: 2, Bottom: 2}})
			if send[bool](w.view, "hasMarkedText") {
				t.Error("the composition survived a change of the text by the app")
			}
			if r := viewRange(w, "selectedRange"); r != (nsRangeVal{3, 0}) {
				t.Errorf("selectedRange = %+v, want {3 0}", r)
			}
			w.Close()
		})
	})
}
