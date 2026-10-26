//go:build windows && glutter_wine

package windows

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/uia"
	"github.com/rm4n0s/glutter/internal/platform/windows/uia/semtree"
	"github.com/rm4n0s/glutter/internal/platform/windows/uia/uiatest"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests check the UI Automation bridge under Wine (SPEC §25.11, §25.13): a window of the platform
// answers WM_GETOBJECT, and the UI Automation client of package uiatest, on a thread of its own, walks the
// tree of a semantics update, reads properties, calls patterns and listens to events. The client and the
// provider are in one process; Wine implements part of uiautomationcore, and a case it cannot do is
// skipped with a message that says what is missing.

// accTestHost plays the engine: when the bridge turns on it gives the bridge the tree, and it records the
// actions the patterns ask for.
type accTestHost struct {
	p    *Platform
	acc  platform.Accessibility
	view ui.ViewID
	tree func() *ui.SemanticsUpdate

	mu      sync.Mutex
	changes []bool
	actions []ui.SemanticsActionEvent
}

func (h *accTestHost) ScreenReaderChanged(on bool) {
	h.mu.Lock()
	h.changes = append(h.changes, on)
	h.mu.Unlock()
	if on {
		h.p.Post(func() { h.acc.UpdateSemantics(h.view, h.tree()) })
	}
}

func (h *accTestHost) PerformAction(ev ui.SemanticsActionEvent) {
	h.mu.Lock()
	h.actions = append(h.actions, ev)
	h.mu.Unlock()
}

func (h *accTestHost) takeActions() []ui.SemanticsActionEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := h.actions
	h.actions = nil
	return out
}

// accTree is the tree of the tests: a root with a button, a check box, a text field, a slider, a list, a
// tree item and a radio button.
func accTree() *ui.SemanticsUpdate {
	return accTreeWith(func(*accNodes) {})
}

// accNodes are the nodes of accTree, which a test edits before they are put together.
type accNodes struct {
	root, button, check, field, slider, list, item1, item2, branch, radio ui.SemanticsNodeUpdate
}

func accTreeWith(edit func(n *accNodes)) *ui.SemanticsUpdate {
	n := &accNodes{
		root:   semtree.N(0, "", 0, 0, 400, 300, 1, 2, 3, 4, 5, 8, 9),
		button: semtree.N(1, "Press", 10, 10, 80, 30),
		check:  semtree.N(2, "Agree", 10, 50, 80, 30),
		field:  semtree.N(3, "Name", 10, 90, 200, 30),
		slider: semtree.N(4, "Level", 10, 130, 200, 30),
		list:   semtree.N(5, "Items", 10, 170, 200, 100, 6, 7),
		item1:  semtree.N(6, "One", 10, 170, 200, 30),
		item2:  semtree.N(7, "Two", 10, 200, 200, 30),
		branch: semtree.N(8, "Branch", 220, 10, 100, 30),
		radio:  semtree.N(9, "Option", 220, 50, 100, 30),
	}
	n.button.Flags.IsButton = true
	n.button.Actions = ui.SemanticsActionTap
	n.check.Flags.IsChecked = ui.CheckedStateIsFalse
	n.check.Actions = ui.SemanticsActionTap
	n.field.Flags.IsTextField = true
	n.field.Flags.IsFocused = ui.TristateIsFalse
	n.field.Value = "héllo"
	n.field.Actions = ui.SemanticsActionFocus
	n.slider.Flags.IsSlider = true
	n.slider.Value = "50%"
	n.slider.Actions = ui.SemanticsActionIncrease | ui.SemanticsActionDecrease
	n.list.Actions = ui.SemanticsActionScrollUp
	n.list.ScrollPosition, n.list.ScrollExtentMin, n.list.ScrollExtentMax = 0, 0, 100
	n.branch.Flags.IsExpanded = ui.TristateIsFalse
	n.branch.Actions = ui.SemanticsActionExpand
	n.radio.Flags.IsChecked = ui.CheckedStateIsFalse
	n.radio.Flags.IsInMutuallyExclusiveGroup = true
	n.radio.Actions = ui.SemanticsActionTap
	edit(n)
	return semtree.Update(n.root, n.button, n.check, n.field, n.slider, n.list, n.item1, n.item2, n.branch, n.radio)
}

// accSession is a window with the bridge, the platform running, and a function that runs the client.
type accSession struct {
	t    *testing.T
	p    *Platform
	r    *rec
	w    *window
	acc  platform.Accessibility
	host *accTestHost
}

func newAccSession(t *testing.T) *accSession {
	t.Helper()
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Title: "UIA test", Size: ui.Size{Width: 400, Height: 300}})
	svc := p.Services()
	if svc == nil {
		t.Fatal("the platform has no services: the COM objects of UI Automation could not be made")
	}
	acc := svc.Accessibility()
	h := &accTestHost{p: p, acc: acc, view: w.id, tree: accTree}
	acc.SetWindow(w.id, "UIA test", true)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	acc.Start(ctx, h)
	return &accSession{t: t, p: p, r: r, w: w, acc: acc, host: h}
}

// client runs fn on a thread of its own, in a multithreaded apartment, while the platform's loop runs on
// the test's thread, and ends the loop when fn returns.
func (s *accSession) client(limit time.Duration, fn func(c *uiatest.Client, root *uiatest.Element)) {
	s.t.Helper()
	done := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer close(done)
		uninit, err := uiatest.InitMTA()
		if err != nil {
			s.t.Errorf("CoInitializeEx: %v", err)
			return
		}
		defer uninit()
		c, err := uiatest.NewClient()
		if err != nil {
			s.t.Errorf("CUIAutomation: %v", err)
			return
		}
		defer c.Close()
		root, err := c.ElementFromHandle(s.w.hwnd)
		if err != nil {
			s.t.Errorf("ElementFromHandle: %v", err)
			return
		}
		defer root.Release()
		fn(c, root)
	}()
	runFor(s.t, s.p, s.r, limit, func(cancel context.CancelFunc) {
		go func() {
			<-done
			s.p.Post(cancel)
		}()
	})
}

// update gives the bridge a new tree, as the engine does after a frame: on the UI thread.
func (s *accSession) update(edit func(n *accNodes)) {
	tree := accTreeWith(edit)
	s.p.Post(func() { s.acc.UpdateSemantics(s.w.id, tree) })
}

// setWindow tells the bridge the window got or lost the focus.
func (s *accSession) setWindow(active bool) {
	s.p.Post(func() { s.acc.SetWindow(s.w.id, "UIA test", active) })
}

// waitAction waits for the framework to be asked for the action.
func (s *accSession) waitAction(want ui.SemanticsActionEvent) bool {
	s.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var got []ui.SemanticsActionEvent
	for time.Now().Before(deadline) {
		got = append(got, s.host.takeActions()...)
		for _, a := range got {
			if a.ViewID == want.ViewID && a.NodeID == want.NodeID && a.Type == want.Type && fmt.Sprint(a.Arguments) == fmt.Sprint(want.Arguments) {
				return true
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	s.t.Errorf("the framework was not asked for %+v; it got %+v", want, got)
	return false
}

// action is an event for a node of the test window.
func (s *accSession) action(node int, a ui.SemanticsAction, args any) ui.SemanticsActionEvent {
	return ui.SemanticsActionEvent{ViewID: s.w.id, NodeID: node, Type: a, Arguments: args}
}

func errText(err error) string {
	if err == nil {
		return "<nil>"
	}
	if appErr, ok := errors.FromError(err); ok {
		return fmt.Sprintf("%s %v", appErr.Route(), appErr.Metadata)
	}
	return err.Error()
}

// elementNamed finds the descendant of root with the name.
func elementNamed(t *testing.T, root *uiatest.Element, name string) *uiatest.Element {
	t.Helper()
	all, err := root.Descendants()
	if err != nil {
		t.Fatalf("Descendants: %s", errText(err))
	}
	var found *uiatest.Element
	for _, e := range all {
		n, _ := e.Name()
		if n == name && found == nil {
			found = e
			continue
		}
		e.Release()
	}
	if found == nil {
		t.Fatalf("no element named %q", name)
	}
	return found
}

// skipIfMissing skips the test, with what Wine lacks, when the call failed with E_NOTIMPL.
func skipIfMissing(t *testing.T, what string, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if appErr, ok := errors.FromError(err); ok && appErr.Metadata["hresult"] == "0x80004001" {
		t.Skipf("Wine's uiautomationcore lacks %s (E_NOTIMPL); a manual check on Windows (SPEC §25.13)", what)
	}
	t.Fatalf("%s: %s", what, errText(err))
}

func TestWineUIAElementFromHandleReturnsTheFragmentRoot(t *testing.T) {
	s := newAccSession(t)
	if len(s.host.changes) != 0 {
		t.Fatal("the bridge was on before a client asked")
	}
	s.client(60*time.Second, func(c *uiatest.Client, root *uiatest.Element) {
		if name, err := root.Name(); err != nil || name != "UIA test" {
			t.Errorf("the window's name = %q, %s; want its title (from Windows' own provider)", name, errText(err))
		}
		if ct, err := root.ControlType(); err != nil || ct != uia.ControlTypeWindow {
			t.Errorf("the window's control type = %d, %s; want a window", ct, errText(err))
		}
		kids, err := root.Children()
		if err != nil {
			t.Errorf("Children: %s", errText(err))
			return
		}
		defer func() {
			for _, k := range kids {
				k.Release()
			}
		}()
		if len(kids) != 1 {
			t.Errorf("the window has %d children, want the root of the semantics tree", len(kids))
			return
		}
		if ct, _ := kids[0].ControlType(); ct != uia.ControlTypePane {
			t.Errorf("the root of the tree is control type %d, want a pane", ct)
		}
	})
	s.host.mu.Lock()
	defer s.host.mu.Unlock()
	if !slices.Equal(s.host.changes, []bool{true}) {
		t.Errorf("the engine was told %v, want [true] (the first ask turns the tree on, once)", s.host.changes)
	}
}

func TestWineUIAWalkListsTheTreeWithNamesTypesAndRectangles(t *testing.T) {
	s := newAccSession(t)
	s.client(60*time.Second, func(c *uiatest.Client, root *uiatest.Element) {
		all, err := root.Descendants()
		if err != nil {
			t.Errorf("Descendants: %s", errText(err))
			return
		}
		defer func() {
			for _, e := range all {
				e.Release()
			}
		}()
		origin, ok := win32.ClientToScreen(s.w.hwnd, win32.Point{})
		if !ok {
			t.Error("ClientToScreen failed")
			return
		}
		type row struct {
			name string
			ct   uia.ControlType
			x, y int32 // the rectangle in the client area
			w, h int32
		}
		want := map[string]row{
			"UIA test": {"UIA test", uia.ControlTypePane, 0, 0, 400, 300},
			"Press":    {"Press", uia.ControlTypeButton, 10, 10, 80, 30},
			"Agree":    {"Agree", uia.ControlTypeCheckBox, 10, 50, 80, 30},
			"Name":     {"Name", uia.ControlTypeEdit, 10, 90, 200, 30},
			"Level":    {"Level", uia.ControlTypeSlider, 10, 130, 200, 30},
			"Items":    {"Items", uia.ControlTypeText, 10, 170, 200, 100},
			"One":      {"One", uia.ControlTypeText, 10, 170, 200, 30},
			"Two":      {"Two", uia.ControlTypeText, 10, 200, 200, 30},
			"Branch":   {"Branch", uia.ControlTypeText, 220, 10, 100, 30},
			"Option":   {"Option", uia.ControlTypeRadioButton, 220, 50, 100, 30},
		}
		if len(all) != len(want) {
			t.Errorf("the walk found %d elements, want %d", len(all), len(want))
		}
		for _, e := range all {
			name, err := e.Name()
			if err != nil {
				t.Errorf("Name: %s", errText(err))
				continue
			}
			w, ok := want[name]
			if !ok {
				t.Errorf("an element named %q that the tree does not have", name)
				continue
			}
			delete(want, name)
			if ct, err := e.ControlType(); err != nil || ct != w.ct {
				t.Errorf("%s: control type %d, %s; want %d", name, ct, errText(err), w.ct)
			}
			r, err := e.BoundingRectangle()
			if err != nil {
				t.Errorf("%s: BoundingRectangle: %s", name, errText(err))
				continue
			}
			got := win32.Rect{Left: r.Left - origin.X, Top: r.Top - origin.Y, Right: r.Right - origin.X, Bottom: r.Bottom - origin.Y}
			if wantRect := (win32.Rect{Left: w.x, Top: w.y, Right: w.x + w.w, Bottom: w.y + w.h}); got != wantRect {
				t.Errorf("%s: rectangle %+v in the client area, want %+v", name, got, wantRect)
			}
		}
		for name := range want {
			t.Errorf("the walk did not find %q", name)
		}
	})
}

// TestWineUIAPatternsThroughTheClient drives every pattern through the UI Automation client, the way a
// screen reader does. Wine's client cannot get a pattern (IUIAutomationElement::GetCurrentPattern is
// E_NOTIMPL, as are the typed getters and SetFocus), so under Wine this test skips, and the patterns are
// driven through their vtables by the tests of package uia (bridge_windows_test.go); on a Windows PC it
// is the first of the manual checks of SPEC §25.13.
func TestWineUIAPatternsThroughTheClient(t *testing.T) {
	s := newAccSession(t)
	s.client(90*time.Second, func(c *uiatest.Client, root *uiatest.Element) {
		probe := elementNamed(t, root, "Press")
		_, _, err := probe.Pattern(uia.PatternInvoke, uiatest.IfaceInvoke)
		probe.Release()
		if err != nil {
			if appErr, ok := errors.FromError(err); ok && appErr.Metadata["hresult"] == "0x80004001" {
				t.Skip("Wine's uiautomationcore lacks IUIAutomationElement::GetCurrentPattern (E_NOTIMPL), so no pattern can be driven through its client; the vtable tests of package uia drive them, and Narrator does on Windows (manual check)")
			}
			t.Errorf("GetCurrentPattern: %s", errText(err))
			return
		}
		fail := func(what string, err error) bool {
			if err != nil {
				t.Errorf("%s: %s", what, errText(err))
				return true
			}
			return false
		}
		// Invoke on a button taps it.
		button := elementNamed(t, root, "Press")
		defer button.Release()
		if !fail("Invoke", button.Invoke()) {
			s.waitAction(s.action(1, ui.SemanticsActionTap, nil))
		}
		// A button has no Toggle pattern.
		if err := button.Toggle(); err == nil {
			t.Error("Toggle of a button worked")
		} else if appErr, ok := errors.FromError(err); !ok || appErr.Tag != win32.TagCOMCallFailed || appErr.Metadata["hresult"] != "0x80040204" {
			t.Errorf("Toggle of a button: %s, want UIA_E_NOTSUPPORTED", errText(err))
		}
		// Toggle on a check box taps it, and the state follows the tree.
		check := elementNamed(t, root, "Agree")
		defer check.Release()
		if st, err := check.ToggleState(); fail("ToggleState", err) || st != uia.ToggleOff {
			t.Errorf("toggle state = %d, want off", st)
		}
		if !fail("Toggle", check.Toggle()) {
			s.waitAction(s.action(2, ui.SemanticsActionTap, nil))
		}
		s.update(func(n *accNodes) { n.check.Flags.IsChecked = ui.CheckedStateIsTrue })
		pollFor(t, "the check box to be on", func() bool { st, err := check.ToggleState(); return err == nil && st == uia.ToggleOn })
		// A text field: its value, SetValue (the field takes the focus first, then the text), SetFocus.
		field := elementNamed(t, root, "Name")
		defer field.Release()
		if v, err := field.Value(); fail("Value", err) || v != "héllo" {
			t.Errorf("value = %q, want héllo", v)
		}
		if ro, err := field.IsReadOnly(); fail("IsReadOnly", err) || ro {
			t.Errorf("read-only = %v", ro)
		}
		if !fail("SetFocus", field.SetFocus()) {
			s.waitAction(s.action(3, ui.SemanticsActionFocus, nil))
		}
		if !fail("SetValue", field.SetValue("abc")) {
			s.waitAction(s.action(3, ui.SemanticsActionFocus, nil))
			// The framework focuses the field and then offers the action to set its text.
			s.update(func(n *accNodes) {
				n.field.Flags.IsFocused = ui.TristateIsTrue
				n.field.Actions = ui.SemanticsActionFocus | ui.SemanticsActionSetText | ui.SemanticsActionSetSelection
			})
			s.waitAction(s.action(3, ui.SemanticsActionSetText, "abc"))
		}
		// A slider: its range and SetValue (the thunk), one step toward the value asked for.
		slider := elementNamed(t, root, "Level")
		defer slider.Release()
		if v, lo, hi, err := slider.RangeValue(); fail("RangeValue", err) || v != 50 || lo != 0 || hi != 100 {
			t.Errorf("range value = %v in [%v, %v], want 50 in [0, 100]", v, lo, hi)
		}
		if !fail("SetRangeValue up", slider.SetRangeValue(80)) {
			s.waitAction(s.action(4, ui.SemanticsActionIncrease, nil))
		}
		if !fail("SetRangeValue down", slider.SetRangeValue(20)) {
			s.waitAction(s.action(4, ui.SemanticsActionDecrease, nil))
		}
		if err := slider.SetRangeValue(101); err == nil {
			t.Error("a value over the maximum was taken")
		}
		if err := slider.SetRangeValue(math.NaN()); err == nil {
			t.Error("NaN was taken")
		}
		// A list: Scroll, and the scroll percentages.
		list := elementNamed(t, root, "Items")
		defer list.Release()
		if !fail("Scroll", list.Scroll(uia.ScrollNoAmount, uia.ScrollSmallIncrement)) {
			s.waitAction(s.action(5, ui.SemanticsActionScrollUp, nil))
		}
		if h, v, err := list.ScrollPercent(); fail("ScrollPercent", err) || h != uia.NoScroll || v != 0 {
			t.Errorf("scroll percent = %v, %v; want -1 and 0", h, v)
		}
		s.update(func(n *accNodes) { n.list.ScrollPosition = 25 })
		pollFor(t, "the list to be scrolled a quarter", func() bool { _, v, err := list.ScrollPercent(); return err == nil && v == 25 })
		// ExpandCollapse and SelectionItem.
		branch := elementNamed(t, root, "Branch")
		defer branch.Release()
		if st, err := branch.ExpandCollapseState(); fail("ExpandCollapseState", err) || st != uia.ExpandCollapseCollapsed {
			t.Errorf("expand state = %d, want collapsed", st)
		}
		if !fail("Expand", branch.Expand()) {
			s.waitAction(s.action(8, ui.SemanticsActionExpand, nil))
		}
		radio := elementNamed(t, root, "Option")
		defer radio.Release()
		if sel, err := radio.IsSelected(); fail("IsSelected", err) || sel {
			t.Errorf("selected = %v, want false", sel)
		}
		if !fail("Select", radio.Select()) {
			s.waitAction(s.action(9, ui.SemanticsActionTap, nil))
		}
	})
}

// pollFor polls cond for a few seconds, from a thread that is not the UI thread.
func pollFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Errorf("timed out waiting for %s", what)
}

// TestWineUIATextPatternThroughTheClient is the Text pattern through the client (see the patterns test
// for why it skips under Wine).
func TestWineUIATextPatternThroughTheClient(t *testing.T) {
	s := newAccSession(t)
	s.update(func(n *accNodes) {
		n.field.Actions = ui.SemanticsActionFocus | ui.SemanticsActionSetSelection
		n.field.TextSelectionBase, n.field.TextSelectionExtent = 1, 4 // "é" is two bytes: the units 1 to 2
	})
	s.client(60*time.Second, func(c *uiatest.Client, root *uiatest.Element) {
		field := elementNamed(t, root, "Name")
		defer field.Release()
		doc, err := field.DocumentRange()
		if err != nil {
			if appErr, ok := errors.FromError(err); ok && appErr.Metadata["hresult"] == "0x80004001" {
				t.Skip("Wine's uiautomationcore lacks IUIAutomationElement::GetCurrentPattern (E_NOTIMPL), so the Text pattern cannot be driven through its client; the vtable tests of package uia drive it")
			}
			t.Errorf("DocumentRange: %s", errText(err))
			return
		}
		defer doc.Release()
		if text, err := doc.Text(-1); err != nil || text != "héllo" {
			t.Errorf("the document = %q, %s; want héllo", text, errText(err))
		}
		if text, err := doc.Text(3); err != nil || text != "hél" {
			t.Errorf("the first 3 units = %q, %s", text, errText(err))
		}
		pollFor(t, "the selection", func() bool {
			sel, err := field.SelectionRanges()
			return err == nil && len(sel) == 1 && sel[0] == "é"
		})
		if err := doc.ExpandToEnclosingUnit(uia.TextUnitWord); err != nil {
			t.Errorf("ExpandToEnclosingUnit: %s", errText(err))
		}
		if moved, err := doc.MoveEndpointByUnit(uia.EndpointStart, uia.TextUnitCharacter, 1); err != nil || moved != 1 {
			t.Errorf("MoveEndpointByUnit = %d, %s", moved, errText(err))
		}
		if text, _ := doc.Text(-1); text != "éllo" {
			t.Errorf("after moving the start = %q, want éllo", text)
		}
		if err := doc.Select(); err != nil {
			t.Errorf("Select: %s", errText(err))
			return
		}
		// The range is units 1 to 5 of "héllo": bytes 1 to 6.
		s.waitAction(s.action(3, ui.SemanticsActionSetSelection, map[string]int{"base": 1, "extent": 6}))
	})
}

func TestWineUIAEvents(t *testing.T) {
	s := newAccSession(t)
	s.client(90*time.Second, func(c *uiatest.Client, root *uiatest.Element) {
		l, err := c.NewListener()
		if err != nil {
			t.Errorf("NewListener: %s", errText(err))
			return
		}
		defer func() {
			if err := l.Close(); err != nil {
				t.Logf("Close: %s", errText(err))
			}
		}()
		// The tree arrives with the first ask; the first events come after the handlers are in.
		time.Sleep(200 * time.Millisecond)
		t.Run("focus", func(t *testing.T) {
			if err := l.OnFocus(); err != nil {
				skipIfMissing(t, "AddFocusChangedEventHandler", err)
			}
			s.update(func(n *accNodes) { n.field.Flags.IsFocused = ui.TristateIsTrue })
			ev, ok := l.Wait(func(e uiatest.Event) bool { return e.Kind == uiatest.EventFocus && e.Name == "Name" }, 15*time.Second)
			if !ok {
				t.Fatal("no focus event for the text field")
			}
			ev.Sender.Release()
		})
		t.Run("window activation", func(t *testing.T) {
			// A window that is not active has no focused node, and activating it raises the focus event of
			// the field that has the focus.
			s.setWindow(false)
			time.Sleep(100 * time.Millisecond)
			for {
				ev, ok := l.Wait(func(uiatest.Event) bool { return true }, 100*time.Millisecond)
				if !ok {
					break
				}
				ev.Sender.Release()
			}
			s.setWindow(true)
			ev, ok := l.Wait(func(e uiatest.Event) bool { return e.Kind == uiatest.EventFocus && e.Name == "Name" }, 15*time.Second)
			if !ok {
				t.Fatal("activating the window raised no focus event for the field that has the focus")
			}
			ev.Sender.Release()
		})
		// The other events of the bridge are not driven through Wine's client. Its uiautomationcore refuses
		// AddPropertyChangedEventHandler and AddStructureChangedEventHandler (E_NOTIMPL); it takes
		// AddAutomationEventHandler for Text_TextChanged and LiveRegionChanged but delivers nothing, and
		// UiaRaiseAutomationEvent then blocks the raising thread while the client makes its next call
		// (a deadlock inside Wine, seen with the UI thread in the call and the client in the next
		// AddAutomationEventHandler), which would stop the UI thread. The bridge raises them (the Core
		// tests with a recorder, and TestEveryKindOfEventIsRaisedWithoutAnError in package uia with the
		// real functions), and a client receiving them is a manual check on Windows.
		t.Run("property structure text and live region", func(t *testing.T) {
			t.Skip("Wine's uiautomationcore lacks AddPropertyChangedEventHandler and AddStructureChangedEventHandler (E_NOTIMPL) and does not deliver Text_TextChanged or LiveRegionChanged (and deadlocks the raising thread); a manual check on Windows")
		})
	})
}
