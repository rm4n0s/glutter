package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/{adaptive_text_selection_toolbar,
// spell_check_suggestions_toolbar}_test.dart. The platforms of Flutter's TargetPlatformVariant.all() are the
// subtests.

var allPlatforms = []struct {
	name    string
	p       foundation.TargetPlatform
	desktop bool
}{
	{"android", foundation.TargetPlatformAndroid, false},
	{"fuchsia", foundation.TargetPlatformFuchsia, false},
	{"iOS", foundation.TargetPlatformIOS, false},
	{"macOS", foundation.TargetPlatformMacOS, true},
	{"linux", foundation.TargetPlatformLinux, true},
	{"windows", foundation.TargetPlatformWindows, true},
}

func TestCupertinoAdaptiveTextSelectionToolbar(t *testing.T) {
	mobileToolbar := gluttertest.ByType[*cupertino.CupertinoTextSelectionToolbar]()
	desktopToolbar := gluttertest.ByType[*cupertino.CupertinoDesktopTextSelectionToolbar]()
	mobileButton := gluttertest.ByType[*cupertino.CupertinoTextSelectionToolbarButton]()
	desktopButton := gluttertest.ByType[*cupertino.CupertinoDesktopTextSelectionToolbarButton]()
	anchors := widgets.TextSelectionToolbarAnchors{PrimaryAnchor: ui.Offset{}}
	for _, pl := range allPlatforms {
		t.Run(pl.name+": it builds the right toolbar and shows the button items", func(t *testing.T) {
			usePlatform(t, pl.p)
			ts := gluttertest.New(t)
			homeApp(ts, cupertino.NewCupertinoAdaptiveTextSelectionToolbarButtonItems(
				[]widgets.ContextMenuButtonItem{{Label: foundation.Some("Click me"), OnPressed: func() {}}}, anchors))
			if !ts.Exists(gluttertest.ByText("Click me")) {
				t.Fatal("no button")
			}
			if got := ts.Exists(mobileToolbar); got == pl.desktop {
				t.Errorf("mobile toolbar %v on a desktop %v platform", got, pl.desktop)
			}
			if got := ts.Exists(desktopToolbar); got != pl.desktop {
				t.Errorf("desktop toolbar %v on a desktop %v platform", got, pl.desktop)
			}
		})
		t.Run(pl.name+": it builds the right button", func(t *testing.T) {
			usePlatform(t, pl.p)
			ts := gluttertest.New(t)
			homeApp(ts, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				return widgets.NewColumn(cupertino.CupertinoAdaptiveTextSelectionToolbarGetAdaptiveButtons(bc,
					[]widgets.ContextMenuButtonItem{{Label: foundation.Some("Click me"), OnPressed: func() {}}}))
			}))
			if !ts.Exists(gluttertest.ByText("Click me")) {
				t.Fatal("no button")
			}
			if ts.Exists(mobileButton) == pl.desktop || ts.Exists(desktopButton) != pl.desktop {
				t.Errorf("mobile %v, desktop %v on a desktop %v platform", ts.Exists(mobileButton), ts.Exists(desktopButton), pl.desktop)
			}
		})
		t.Run(pl.name+": it builds the buttons of an editable field", func(t *testing.T) {
			usePlatform(t, pl.p)
			ts := gluttertest.New(t)
			noop := func() {}
			ts.SetViewSize(ui.Size{Width: 330, Height: 600})
			homeApp(ts, cupertino.NewCupertinoAdaptiveTextSelectionToolbarEditable(widgets.ClipboardStatusPasteable,
				widgets.EditableButtonItemsCallbacks{OnCopy: noop, OnCut: noop, OnPaste: noop, OnSelectAll: noop, OnLiveTextInput: noop,
					OnLookUp: noop, OnSearchWeb: noop, OnShare: noop}, anchors))
			liveText := gluttertest.ByPredicate("the Live Text icon", func(w widgets.Widget) bool {
				c, ok := w.(*widgets.CustomPaint)
				return ok && c.Painter() != nil && typeName(c.Painter()) == "*cupertino.liveTextIconPainter"
			})
			if pl.desktop {
				if n := ts.Count(desktopButton); n != 8 {
					t.Errorf("%d desktop buttons, want 8", n)
				}
				for _, label := range []string{"Cut", "Copy", "Paste", "Select All", "Share...", "Look Up", "Search Web"} {
					if !ts.Exists(gluttertest.ByText(label)) {
						t.Errorf("no %q", label)
					}
				}
				// The Live Text button of the desktop is a text with an empty label.
				if !ts.Exists(gluttertest.ByText("")) {
					t.Error("no Live Text button")
				}
				return
			}
			// The labels of the Roboto of the tests are narrower than the Ahem ones of Flutter's tests, so the
			// view is narrow for the buttons to overflow onto a second page.
			seen := map[string]bool{}
			collect := func() {
				for _, label := range []string{"Cut", "Copy", "Paste", "Select All", "Look Up", "Search Web", "Share..."} {
					if ts.Exists(gluttertest.ByText(label)) {
						seen[label] = true
					}
				}
				if ts.Exists(liveText) {
					seen["live"] = true
				}
			}
			collect()
			next := gluttertest.ByPredicate("the next chevron", isChevron(false))
			if !ts.Exists(next) {
				t.Fatal("the buttons do not overflow onto a second page")
			}
			for ts.Exists(next) {
				ts.TapAt(ts.Center(next))
				ts.PumpAndSettle()
				collect()
			}
			if len(seen) != 8 {
				t.Errorf("the pages show %v, want the seven labels and the Live Text button", seen)
			}
		})
	}
	t.Run("it builds children directly", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := foundation.NewUniqueKey()
		homeApp(ts, cupertino.NewCupertinoAdaptiveTextSelectionToolbar(anchors, []widgets.Widget{widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Key: key})}))
		if !ts.Exists(gluttertest.ByKey(key)) {
			t.Error("the child is not built")
		}
	})
	t.Run("it builds an empty toolbar with no children and no items", func(t *testing.T) {
		ts := gluttertest.New(t)
		homeApp(ts, cupertino.NewCupertinoAdaptiveTextSelectionToolbar(anchors, nil))
		if ts.Exists(mobileToolbar) || ts.Exists(desktopToolbar) {
			t.Error("a toolbar is built")
		}
	})
	t.Run("it builds from an EditableTextState and shows only Paste in an empty field", func(t *testing.T) {
		for _, pl := range allPlatforms {
			usePlatformSub(t, pl.name, pl.p, func(t *testing.T) {
				ts := gluttertest.New(t)
				key := foundation.NewUniqueKey()
				controller := widgets.NewTextEditingController()
				defer controller.Dispose()
				node := widgets.NewFocusNode(widgets.FocusNodeOpts{})
				defer node.Dispose()
				ts.SetClipboardText("clip")
				ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewAlign(widgets.NewSizedBox(widgets.NewEditableText(controller, node,
					painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Roboto"}), ui.NewColor(0xFF00FFFF), ui.NewColor(0xFF00FFFF), widgets.EditableTextOpts{
						SelectionControls: cupertino.CupertinoTextSelectionHandleControls,
						ContextMenuBuilder: func(_ widgets.BuildContext, s *widgets.EditableTextState) widgets.Widget {
							return cupertino.NewCupertinoAdaptiveTextSelectionToolbarEditableText(s, widgets.WidgetOpts{Key: key})
						}}), widgets.SizedBoxOpts{Width: foundation.Some(400.0)}),
					widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})}))
				ts.Pump()
				if ts.Exists(gluttertest.ByKey(key)) {
					t.Fatal("the menu is shown at first")
				}
				ts.LongPress(gluttertest.ByType[*widgets.EditableText]())
				gluttertest.StateOf[*widgets.EditableTextState](ts, gluttertest.ByType[*widgets.EditableText]()).ShowToolbar()
				ts.PumpAndSettle()
				if !ts.Exists(gluttertest.ByKey(key)) {
					t.Fatal("the menu is not shown")
				}
				for _, label := range []string{"Copy", "Cut", "Select All"} {
					if ts.Exists(gluttertest.ByText(label)) {
						t.Errorf("%q is shown in an empty field", label)
					}
				}
				if !ts.Exists(gluttertest.ByText("Paste")) {
					t.Error("no Paste")
				}
				if pl.desktop && ts.Count(desktopButton) != 1 || !pl.desktop && ts.Count(mobileButton) != 1 {
					t.Errorf("mobile buttons %d, desktop buttons %d", ts.Count(mobileButton), ts.Count(desktopButton))
				}
			})
		}
	})
}

func usePlatformSub(t *testing.T, name string, p foundation.TargetPlatform, f func(t *testing.T)) {
	t.Run(name, func(t *testing.T) {
		usePlatform(t, p)
		f(t)
	})
}

func isChevron(left bool) func(widgets.Widget) bool {
	return func(w widgets.Widget) bool {
		c, ok := w.(*widgets.CustomPaint)
		return ok && c.Painter() != nil && typeName(c.Painter()) == "*cupertino.cupertinoChevronPainter" && chevronIsLeft(c) == left
	}
}

func TestCupertinoSpellCheckSuggestionsToolbar(t *testing.T) {
	toolbarOf := func(items ...string) widgets.Widget {
		var b []widgets.ContextMenuButtonItem
		for _, s := range items {
			b = append(b, widgets.ContextMenuButtonItem{OnPressed: func() {}, Label: foundation.Some(s)})
		}
		return cupertino.NewCupertinoSpellCheckSuggestionsToolbar(widgets.TextSelectionToolbarAnchors{PrimaryAnchor: ui.Offset{}}, b)
	}
	t.Run("more than three suggestions is an error", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t)
		homeApp(ts, toolbarOf("hello", "yellow", "yell"))
		err := recoverError(func() { toolbarOf("hello", "yellow", "yell", "yeller") })
		requireErrorTag(t, err, "CupertinoSpellCheckSuggestionsToolbarTooManyButtons")
	})
	t.Run("a toolbar of zero area does not crash", func(t *testing.T) {
		ts := gluttertest.New(t)
		homeApp(ts, widgets.NewSizedBoxShrink(cupertino.NewCupertinoSpellCheckSuggestionsToolbar(widgets.TextSelectionToolbarAnchors{PrimaryAnchor: ui.Offset{Dx: 1, Dy: 1}}, nil)))
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoSpellCheckSuggestionsToolbar]()); got != (ui.Size{}) {
			t.Errorf("size %v", got)
		}
	})
	t.Run("the buttons are the suggestions", func(t *testing.T) {
		ts := gluttertest.New(t)
		homeApp(ts, toolbarOf("hello", "yellow", "yell"))
		for _, s := range []string{"hello", "yellow", "yell"} {
			if !ts.Exists(gluttertest.ByText(s)) {
				t.Errorf("no %q", s)
			}
		}
	})
}

func TestCupertinoSpellCheckSuggestionsToolbarButtonItems(t *testing.T) {
	// The only platform of the tests that has a native spell checker (the fake of gluttertest) is Android.
	usePlatform(t, foundation.TargetPlatformAndroid)
	checker := func(spans []services.SuggestionSpan) gluttertest.Opts {
		var out []ui.SpellCheckSpan
		for _, s := range spans {
			out = append(out, ui.SpellCheckSpan{Start: s.Range.Start, End: s.Range.End, Suggestions: s.Suggestions})
		}
		return gluttertest.Opts{SpellChecker: func(string, string) []ui.SpellCheckSpan { return out }}
	}
	stateWith := func(t *testing.T, text string, caret int, spans ...services.SuggestionSpan) (*widgets.EditableTextState, *widgets.TextEditingController) {
		ts := gluttertest.New(t, checker(spans))
		controller := widgets.NewTextEditingController(text)
		t.Cleanup(controller.Dispose)
		controller.SetSelection(services.NewTextSelectionCollapsed(caret))
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: controller, Autofocus: true,
			SpellCheckConfiguration: widgets.NewSpellCheckConfiguration()}))
		ts.PumpAndSettle()
		state := editableState(ts)
		state.SetSpellCheckResults(&services.SpellCheckResults{SpellCheckedText: text, SuggestionSpans: spans})
		return state, controller
	}
	span := func(start, end int, s ...string) services.SuggestionSpan {
		return services.NewSuggestionSpan(ui.TextRange{Start: start, End: end}, s)
	}
	t.Run("only the first three suggestions are considered", func(t *testing.T) {
		state, _ := stateWith(t, "helo", 2, span(0, 4, "hello", "yellow", "yell", "yeller"))
		items, ok := cupertino.CupertinoSpellCheckSuggestionsToolbarBuildButtonItems(state)
		if !ok || len(items) != 3 {
			t.Fatalf("items %v, ok %v", items, ok)
		}
		for i, want := range []string{"hello", "yellow", "yell"} {
			if items[i].Label.OrElse("") != want {
				t.Errorf("item %d is %q, want %q", i, items[i].Label.OrElse(""), want)
			}
		}
	})
	t.Run("no suggestions is a disabled 'No Replacements Found' button", func(t *testing.T) {
		state, _ := stateWith(t, "helo", 2, span(0, 4))
		items, ok := cupertino.CupertinoSpellCheckSuggestionsToolbarBuildButtonItems(state)
		if !ok || len(items) != 1 || items[0].Label.OrElse("") != "No Replacements Found" || items[0].OnPressed != nil {
			t.Errorf("items %v, ok %v", items, ok)
		}
	})
	t.Run("a cursor outside the misspelled words has no items", func(t *testing.T) {
		state, _ := stateWith(t, "helo world", 8, span(0, 4, "hello"))
		if items, ok := cupertino.CupertinoSpellCheckSuggestionsToolbarBuildButtonItems(state); ok || len(items) != 0 {
			t.Errorf("items %v, ok %v", items, ok)
		}
	})
	t.Run("replacing puts the cursor at the end of the word", func(t *testing.T) {
		state, controller := stateWith(t, "helo world", 2, span(0, 4, "hello"))
		items, _ := cupertino.CupertinoSpellCheckSuggestionsToolbarBuildButtonItems(state)
		items[0].OnPressed()
		if controller.Text() != "hello world" || controller.Selection() != services.NewTextSelectionCollapsed(5) {
			t.Errorf("text %q, selection %v", controller.Text(), controller.Selection())
		}
	})
	t.Run("a read only field cannot be replaced into", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t, checker([]services.SuggestionSpan{span(0, 4, "hello")}))
		controller := widgets.NewTextEditingController("helo")
		defer controller.Dispose()
		controller.SetSelection(services.NewTextSelectionCollapsed(2))
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: controller, Autofocus: true, ReadOnly: true,
			SpellCheckConfiguration: widgets.NewSpellCheckConfiguration()}))
		ts.PumpAndSettle()
		editableState(ts).SetSpellCheckResults(&services.SpellCheckResults{SpellCheckedText: "helo", SuggestionSpans: []services.SuggestionSpan{span(0, 4, "hello")}})
		items, _ := cupertino.CupertinoSpellCheckSuggestionsToolbarBuildButtonItems(editableState(ts))
		if len(items) == 0 {
			t.Skip("a read only field has no suggestions to replace")
		}
		err := recoverError(items[0].OnPressed)
		requireErrorTag(t, err, "CupertinoSpellCheckSuggestionsToolbarReplaceNotAllowed")
	})
}

// chevronIsLeft draws the chevron of the CustomPaint and says whether it points to the left: its first line goes
// from the right to the middle.
func chevronIsLeft(c *widgets.CustomPaint) bool {
	recorder := ui.NewPictureRecorder()
	c.Painter().Paint(ui.NewCanvas(recorder), ui.Size{Width: 10, Height: 10})
	for _, op := range recorder.EndRecording().DisplayList().Ops() {
		if l, ok := op.(ui.OpDrawLine); ok {
			return l.P1.Dx > l.P2.Dx
		}
	}
	return false
}
