package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/selectable_region_test.dart.

// selectionTextStyle matches the big text of the Dart tests: the drag distance
// must exceed the slop of the gesture recognizers.
func selectionTextStyle() *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Roboto", FontSize: foundation.Some(48.0), Color: foundation.Some(ui.NewColor(0xFF000000))})
}

// paragraphOffsetToPosition is textOffsetToPosition of the Dart test.
func paragraphOffsetToPosition(p *rendering.RenderParagraph, offset int) ui.Offset {
	local := p.GetOffsetForCaret(ui.NewTextPosition(offset), ui.NewRectFromLTWH(0, 0, 2, 20)).Add(ui.Offset{Dy: p.PreferredLineHeight()})
	return p.LocalToGlobal(local, nil).Add(ui.Offset{Dy: -2})
}

func emptyContextMenu(BuildContext, *SelectableRegionState) Widget { return NewSizedBoxShrink(nil) }

type selectableRegionTestOpts struct {
	focusNode          *FocusNode
	onSelectionChanged func(*rendering.SelectedContent)
	contextMenu        SelectableRegionContextMenuBuilder
	noContextMenu      bool
	controls           TextSelectionControls
}

// selectableRegion is _selectableRegion of the Dart test.
func selectableRegion(child Widget, opts ...selectableRegionTestOpts) *SelectableRegion {
	var o selectableRegionTestOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.contextMenu == nil && !o.noContextMenu {
		o.contextMenu = emptyContextMenu
	}
	if o.controls == nil {
		o.controls = testTextSelectionControls{}
	}
	return NewSelectableRegion(o.controls, child, SelectableRegionOpts{
		ContextMenuBuilder: o.contextMenu, FocusNode: o.focusNode, OnSelectionChanged: o.onSelectionChanged,
	})
}

func pumpSelectable(ts *gluttertest.Tester, child Widget, opts ...selectableRegionTestOpts) {
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{textStyle: selectionTextStyle(), home: selectableRegion(child, opts...)}))
}

func paragraphOf(ts *gluttertest.Tester, text string) *rendering.RenderParagraph {
	return ts.RenderObject(gluttertest.Descendant(gluttertest.ByText(text), gluttertest.ByType[*RichText]())).(*rendering.RenderParagraph)
}

func requireSelection(t *testing.T, p *rendering.RenderParagraph, want services.TextSelection) {
	t.Helper()
	sel := p.Selections()
	if len(sel) == 0 {
		t.Fatalf("no selection, want %v", want)
	}
	if sel[0] != want {
		t.Fatalf("selection = %v, want %v", sel[0], want)
	}
}

func requireNoSelection(t *testing.T, p *rendering.RenderParagraph) {
	t.Helper()
	if sel := p.Selections(); len(sel) != 0 {
		t.Fatalf("selection = %v, want none", sel)
	}
}

func sel(base, extent int) services.TextSelection {
	return services.NewTextSelection(base, extent)
}

var (
	desktopPlatforms = []foundation.TargetPlatform{foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows}
	mobilePlatforms  = []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS, foundation.TargetPlatformFuchsia}
)

func forPlatforms(t *testing.T, platforms []foundation.TargetPlatform, fn func(t *testing.T, ts *gluttertest.Tester)) {
	for _, p := range platforms {
		t.Run(p.String(), func(t *testing.T) {
			usePlatform(t, p)
			fn(t, gluttertest.New(t))
		})
	}
}

var mouseKind = gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse}

func TestSelectableRegionMouseCanSelectSingleTextOnDesktopPlatforms(t *testing.T) {
	forPlatforms(t, desktopPlatforms, func(t *testing.T, ts *gluttertest.Tester) {
		pumpSelectable(ts, NewCenter(ltrText("How are you")))
		p := paragraphOf(ts, "How are you")
		g := ts.StartGesture(paragraphOffsetToPosition(p, 2), mouseKind)
		ts.Pump()
		g.MoveTo(paragraphOffsetToPosition(p, 4))
		ts.Pump()
		requireSelection(t, p, sel(2, 4))
		g.MoveTo(paragraphOffsetToPosition(p, 6))
		ts.Pump()
		requireSelection(t, p, sel(2, 6))
		// Check backward selection.
		g.MoveTo(paragraphOffsetToPosition(p, 1))
		ts.Pump()
		requireSelection(t, p, sel(2, 1))
		// Start a new drag.
		g.Up()
		ts.PumpAndSettle()
		g.Down(paragraphOffsetToPosition(p, 5))
		ts.PumpAndSettle()
		requireSelection(t, p, services.NewTextSelectionCollapsed(5))
		// Selecting across line should select to the end.
		g.MoveTo(paragraphOffsetToPosition(p, 5).Add(ui.Offset{Dy: 200}))
		ts.Pump()
		requireSelection(t, p, sel(5, 11))
		g.Up()
	})
}

func TestSelectableRegionMouseCanSelectSingleTextOnMobilePlatforms(t *testing.T) {
	forPlatforms(t, mobilePlatforms, func(t *testing.T, ts *gluttertest.Tester) {
		pumpSelectable(ts, NewCenter(ltrText("How are you")))
		p := paragraphOf(ts, "How are you")
		g := ts.StartGesture(paragraphOffsetToPosition(p, 2), mouseKind)
		ts.Pump()
		g.MoveTo(paragraphOffsetToPosition(p, 4))
		ts.Pump()
		requireSelection(t, p, sel(2, 4))
		g.MoveTo(paragraphOffsetToPosition(p, 1))
		ts.Pump()
		requireSelection(t, p, sel(2, 1))
		g.Up()
		ts.PumpAndSettle()
		g.Down(paragraphOffsetToPosition(p, 5))
		ts.PumpAndSettle()
		g.MoveTo(paragraphOffsetToPosition(p, 6))
		ts.Pump()
		requireSelection(t, p, sel(5, 6))
		g.MoveTo(paragraphOffsetToPosition(p, 5).Add(ui.Offset{Dy: 200}))
		ts.Pump()
		requireSelection(t, p, sel(5, 11))
		g.Up()
	})
}

func TestSelectableRegionMouseDragFinalizesTheSelection(t *testing.T) {
	forPlatforms(t, append(append([]foundation.TargetPlatform{}, desktopPlatforms...), mobilePlatforms...), func(t *testing.T, ts *gluttertest.Tester) {
		textKey := NewGlobalKey[State]()
		pumpSelectable(ts, NewCenter(NewText("How are you", TextOpts{Key: textKey, TextDirection: ui.TextDirectionLTR})))
		ts.PumpAndSettle()
		notifier := SelectableRegionSelectionStatusScopeMaybeOf(textKey.CurrentContext())
		if notifier == nil {
			t.Fatal("no selection status scope")
		}
		var status SelectableRegionSelectionStatus
		notifier.AddListener(func() { status = notifier.Value() })
		p := paragraphOf(ts, "How are you")
		g := ts.StartGesture(paragraphOffsetToPosition(p, 2), mouseKind)
		ts.Pump()
		g.MoveTo(paragraphOffsetToPosition(p, 4))
		ts.Pump()
		if status != SelectableRegionSelectionStatusChanging {
			t.Fatalf("status during the drag = %v", status)
		}
		g.Up()
		ts.Pump()
		if len(p.Selections()) != 1 || status != SelectableRegionSelectionStatusFinalized {
			t.Fatalf("after the drag: selections=%v status=%v", p.Selections(), status)
		}
	})
}

func TestSelectableRegionTouchDragDoesNotSelectOnMobilePlatforms(t *testing.T) {
	forPlatforms(t, mobilePlatforms, func(t *testing.T, ts *gluttertest.Tester) {
		pumpSelectable(ts, NewCenter(ltrText("How are you")))
		p := paragraphOf(ts, "How are you")
		g := ts.StartGesture(paragraphOffsetToPosition(p, 2))
		ts.Pump()
		g.MoveTo(paragraphOffsetToPosition(p, 4))
		ts.Pump()
		g.Up()
		ts.Pump()
		requireNoSelection(t, p)
	})
}

func TestSelectableRegionMouseCanSelectWordByWordOnDoubleClickDrag(t *testing.T) {
	ts := gluttertest.New(t)
	pumpSelectable(ts, NewCenter(ltrText("How are you")))
	p := paragraphOf(ts, "How are you")
	g := ts.StartGesture(paragraphOffsetToPosition(p, 2), mouseKind)
	ts.Pump()
	g.Up()
	ts.Pump()
	g.Down(paragraphOffsetToPosition(p, 2))
	ts.PumpAndSettle()
	requireSelection(t, p, sel(0, 3))
	steps := []struct {
		to   int
		want services.TextSelection
	}{
		{3, sel(0, 4)}, {4, sel(0, 7)}, {7, sel(0, 8)}, {8, sel(0, 11)}, {1, sel(0, 3)},
	}
	for _, s := range steps {
		g.MoveTo(paragraphOffsetToPosition(p, s.to))
		ts.Pump()
		requireSelection(t, p, s.want)
	}
	// Start a new double-click drag.
	g.Up()
	ts.Pump()
	g.Down(paragraphOffsetToPosition(p, 5))
	ts.Pump()
	g.Up()
	requireSelection(t, p, services.NewTextSelectionCollapsed(5))
	ts.Pump(gestures.DoubleTapTimeout)
	g.Down(paragraphOffsetToPosition(p, 5))
	ts.Pump()
	g.Up()
	ts.Pump()
	g.Down(paragraphOffsetToPosition(p, 5))
	ts.PumpAndSettle()
	requireSelection(t, p, sel(4, 7))
	// Selecting across line should select to the end.
	g.MoveTo(paragraphOffsetToPosition(p, 5).Add(ui.Offset{Dy: 200}))
	ts.Pump()
	requireSelection(t, p, sel(4, 11))
	g.Up()
}

func threeParagraphs() Widget {
	return NewColumn([]Widget{ltrText("How are you?"), ltrText("Good, and you?"), ltrText("Fine, thank you.")})
}

func TestSelectableRegionMouseCanSelectMultipleWidgets(t *testing.T) {
	ts := gluttertest.New(t)
	pumpSelectable(ts, threeParagraphs())
	p1, p2, p3 := paragraphOf(ts, "How are you?"), paragraphOf(ts, "Good, and you?"), paragraphOf(ts, "Fine, thank you.")
	g := ts.StartGesture(paragraphOffsetToPosition(p1, 2), mouseKind)
	ts.Pump()
	g.MoveTo(paragraphOffsetToPosition(p1, 4))
	ts.Pump()
	requireSelection(t, p1, sel(2, 4))
	g.MoveTo(paragraphOffsetToPosition(p2, 5))
	requireSelection(t, p1, sel(2, 12))
	requireSelection(t, p2, sel(0, 5))
	g.MoveTo(paragraphOffsetToPosition(p3, 6))
	requireSelection(t, p1, sel(2, 12))
	requireSelection(t, p2, sel(0, 14))
	requireSelection(t, p3, sel(0, 6))
	g.Up()
}

func TestSelectableRegionMouseCanSelectMultipleWidgetsOnDoubleClickDrag(t *testing.T) {
	ts := gluttertest.New(t)
	pumpSelectable(ts, threeParagraphs())
	p1, p2, p3 := paragraphOf(ts, "How are you?"), paragraphOf(ts, "Good, and you?"), paragraphOf(ts, "Fine, thank you.")
	g := ts.StartGesture(paragraphOffsetToPosition(p1, 2), mouseKind)
	ts.Pump()
	g.Up()
	ts.Pump()
	g.Down(paragraphOffsetToPosition(p1, 2))
	ts.PumpAndSettle()
	requireSelection(t, p1, sel(0, 3))
	g.MoveTo(paragraphOffsetToPosition(p1, 4))
	ts.Pump()
	requireSelection(t, p1, sel(0, 7))
	g.MoveTo(paragraphOffsetToPosition(p2, 5))
	requireSelection(t, p1, sel(0, 12))
	requireSelection(t, p2, sel(0, 6))
	g.MoveTo(paragraphOffsetToPosition(p3, 6))
	requireSelection(t, p1, sel(0, 12))
	requireSelection(t, p2, sel(0, 14))
	requireSelection(t, p3, sel(0, 11))
	g.Up()
}

func TestSelectableRegionMouseCanWorkWithDisabledContainer(t *testing.T) {
	ts := gluttertest.New(t)
	pumpSelectable(ts, NewColumn([]Widget{
		ltrText("How are you?"),
		NewSelectionContainerDisabled(ltrText("Good, and you?")),
		ltrText("Fine, thank you."),
	}))
	p1, p2, p3 := paragraphOf(ts, "How are you?"), paragraphOf(ts, "Good, and you?"), paragraphOf(ts, "Fine, thank you.")
	g := ts.StartGesture(paragraphOffsetToPosition(p1, 2), mouseKind)
	ts.Pump()
	g.MoveTo(paragraphOffsetToPosition(p1, 4))
	ts.Pump()
	requireSelection(t, p1, sel(2, 4))
	g.MoveTo(paragraphOffsetToPosition(p2, 5))
	requireSelection(t, p1, sel(2, 12))
	requireNoSelection(t, p2)
	g.MoveTo(paragraphOffsetToPosition(p3, 6))
	requireSelection(t, p1, sel(2, 12))
	requireNoSelection(t, p2)
	requireSelection(t, p3, sel(0, 6))
	g.Up()
}

func TestSelectableRegionMouseCanReverseSelection(t *testing.T) {
	ts := gluttertest.New(t)
	pumpSelectable(ts, threeParagraphs())
	p1, p2, p3 := paragraphOf(ts, "How are you?"), paragraphOf(ts, "Good, and you?"), paragraphOf(ts, "Fine, thank you.")
	g := ts.StartGesture(paragraphOffsetToPosition(p3, 10), mouseKind)
	ts.Pump()
	g.MoveTo(paragraphOffsetToPosition(p3, 4))
	ts.Pump()
	requireSelection(t, p3, sel(10, 4))
	g.MoveTo(paragraphOffsetToPosition(p2, 5))
	requireSelection(t, p3, sel(10, 0))
	requireSelection(t, p2, sel(14, 5))
	g.MoveTo(paragraphOffsetToPosition(p1, 6))
	requireSelection(t, p3, sel(10, 0))
	requireSelection(t, p2, sel(14, 0))
	requireSelection(t, p1, sel(12, 6))
	g.Up()
}

func TestSelectableRegionTripleClickSelectsTheParagraph(t *testing.T) {
	const longText = "Hello world this is some long piece of text " +
		"that will represent a long paragraph, when triple clicking this block " +
		"of text all of it will be selected.\n" +
		"This will be the start of a new line. When triple clicking this block " +
		"of text all of it should be selected."
	ts := gluttertest.New(t)
	pumpSelectable(ts, NewCenter(ltrText(longText)))
	p := paragraphOf(ts, longText)
	g := ts.StartGesture(paragraphOffsetToPosition(p, 2), mouseKind)
	ts.Pump()
	g.Up()
	ts.Pump()
	g.Down(paragraphOffsetToPosition(p, 2))
	ts.Pump()
	g.Up()
	ts.Pump()
	g.Down(paragraphOffsetToPosition(p, 2))
	ts.PumpAndSettle()
	requireSelection(t, p, sel(0, 150))
	g.MoveTo(paragraphOffsetToPosition(p, 155))
	ts.PumpAndSettle()
	requireSelection(t, p, sel(0, 257))
	// Check backward selection.
	g.MoveTo(paragraphOffsetToPosition(p, 1))
	ts.Pump()
	requireSelection(t, p, sel(0, 150))
	g.Up()
}

func TestSelectableRegionSelectAllAndCopyWithTheKeyboard(t *testing.T) {
	forPlatforms(t, []foundation.TargetPlatform{foundation.TargetPlatformLinux, foundation.TargetPlatformWindows}, func(t *testing.T, ts *gluttertest.Tester) {
		focus := NewFocusNode()
		t.Cleanup(focus.Dispose)
		var content *rendering.SelectedContent
		pumpSelectable(ts, threeParagraphs(), selectableRegionTestOpts{focusNode: focus, onSelectionChanged: func(c *rendering.SelectedContent) { content = c }})
		focus.RequestFocus()
		ts.Pump()
		sendKeysWith(ts, services.LogicalKeyboardKeyKeyA, mods{control: true})
		ts.Pump()
		requireSelection(t, paragraphOf(ts, "How are you?"), sel(0, 12))
		requireSelection(t, paragraphOf(ts, "Good, and you?"), sel(0, 14))
		requireSelection(t, paragraphOf(ts, "Fine, thank you."), sel(0, 16))
		if content == nil || content.PlainText != "How are you?Good, and you?Fine, thank you." {
			t.Fatalf("OnSelectionChanged content = %+v", content)
		}
		sendKeysWith(ts, services.LogicalKeyboardKeyKeyC, mods{control: true})
		ts.FlushPosted()
		if text, ok := ts.ClipboardText(); !ok || text != "How are you?Good, and you?Fine, thank you." {
			t.Fatalf("clipboard = %q (%v)", text, ok)
		}
	})
}

func TestSelectableRegionCollapsingSelectionClearsTheOthers(t *testing.T) {
	ts := gluttertest.New(t)
	pumpSelectable(ts, threeParagraphs())
	p1, p2, p3 := paragraphOf(ts, "How are you?"), paragraphOf(ts, "Good, and you?"), paragraphOf(ts, "Fine, thank you.")
	g := ts.StartGesture(paragraphOffsetToPosition(p1, 2), mouseKind)
	ts.Pump()
	g.MoveTo(paragraphOffsetToPosition(p3, 6))
	ts.Pump()
	g.Up()
	ts.Pump()
	requireSelection(t, p2, sel(0, 14))
	ts.Pump(gestures.DoubleTapTimeout)
	ts.TapAt(paragraphOffsetToPosition(p2, 3), mouseKind)
	ts.Pump()
	requireSelection(t, p2, services.NewTextSelectionCollapsed(3))
	requireNoSelection(t, p1)
	requireNoSelection(t, p3)
}

func TestSelectableRegionSelectionStatusIsFinalizedOnlyOnce(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	textKey := NewGlobalKey[State]()
	pumpSelectable(ts, NewCenter(NewText("How are you", TextOpts{Key: textKey, TextDirection: ui.TextDirectionLTR})))
	if SelectableRegionSelectionStatusScopeMaybeOf(textKey.CurrentContext()) == nil {
		t.Fatal("no selection status scope")
	}
	state := gluttertest.StateOf[*SelectableRegionState](ts, gluttertest.ByType[*SelectableRegion]())
	state.SelectAll()
	ts.Pump()
	state.ClearSelection()
	ts.Pump()
}

func TestSelectableRegionAddAndRemoveGuards(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	pumpSelectable(ts, NewCenter(ltrText("How are you")))
	state := gluttertest.StateOf[*SelectableRegionState](ts, gluttertest.ByType[*SelectableRegion]())
	stranger := &struct{ rendering.Selectable }{}
	requireTag(t, "SelectableRegionMultipleSelectables", func() { state.Add(stranger) })
	requireTag(t, "SelectableRegionUnknownSelectable", func() { state.Remove(stranger) })
}

func TestSelectableRegionRightClickShowsTheContextMenu(t *testing.T) {
	for _, p := range []foundation.TargetPlatform{foundation.TargetPlatformLinux, foundation.TargetPlatformWindows} {
		t.Run(p.String(), func(t *testing.T) {
			usePlatform(t, p)
			ts := gluttertest.New(t)
			menus := 0
			pumpSelectable(ts, NewCenter(ltrText("How are you")), selectableRegionTestOpts{
				contextMenu: func(BuildContext, *SelectableRegionState) Widget {
					menus++
					// An overlay entry gets tight constraints: align the menu.
					return NewAlign(NewSizedBox(ltrText("menu"), SizedBoxOpts{Width: foundation.Some(10.0), Height: foundation.Some(10.0)}), AlignOpts{Alignment: painting.AlignmentTopLeft})
				},
			})
			para := paragraphOf(ts, "How are you")
			at := paragraphOffsetToPosition(para, 3)
			g := ts.StartGesture(at, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse, Buttons: foundation.Some(gestures.SecondaryButton)})
			ts.Pump()
			g.Up()
			ts.Pump()
			if menus == 0 || !ts.Exists(gluttertest.ByText("menu")) {
				t.Fatalf("the context menu was not shown (builder called %d times)", menus)
			}
			// A tap elsewhere hides it.
			ts.TapAt(ui.Offset{Dx: 100, Dy: 50}, mouseKind)
			ts.Pump(time.Second)
			if ts.Exists(gluttertest.ByText("menu")) {
				t.Fatal("the context menu stayed after a tap outside")
			}
		})
	}
}

func TestSelectableRegionSelectAllInRichTextWithAWidgetSpan(t *testing.T) {
	ts := gluttertest.New(t)
	var content *rendering.SelectedContent
	focus := NewFocusNode()
	t.Cleanup(focus.Dispose)
	span := painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("Hello "), Children: []painting.InlineSpan{
		NewWidgetSpan(NewSizedBox(ltrText("W"), SizedBoxOpts{Width: foundation.Some(60.0), Height: foundation.Some(60.0)})),
		painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(" world")}),
	}})
	pumpSelectable(ts, NewCenter(NewTextRich(span, TextOpts{TextDirection: ui.TextDirectionLTR})),
		selectableRegionTestOpts{focusNode: focus, onSelectionChanged: func(c *rendering.SelectedContent) { content = c }})
	state := gluttertest.StateOf[*SelectableRegionState](ts, gluttertest.ByType[*SelectableRegion]())
	state.SelectAll()
	ts.Pump()
	if content == nil || content.PlainText != "Hello W world" {
		t.Fatalf("selected content = %+v, want %q", content, "Hello W world")
	}
	state.ClearSelection()
	ts.Pump()
	if content != nil {
		t.Fatalf("content after clearing = %+v, want nil", content)
	}
}
