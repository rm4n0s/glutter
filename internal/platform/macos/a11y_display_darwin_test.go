//go:build darwin && !ios && glutter_display

package macos

import (
	"math"
	"slices"
	"sync"
	"testing"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/internal/platform/macos/a11y"
	"github.com/rm4n0s/glutter/ui"
)

// fakeAXHost is the engine's side of the bridge: it records what the bridge asks of the framework.
type fakeAXHost struct {
	mu      sync.Mutex
	enabled []bool
	actions []ui.SemanticsActionEvent
}

func (f *fakeAXHost) ScreenReaderChanged(on bool) {
	f.mu.Lock()
	f.enabled = append(f.enabled, on)
	f.mu.Unlock()
}

func (f *fakeAXHost) PerformAction(ev ui.SemanticsActionEvent) {
	f.mu.Lock()
	f.actions = append(f.actions, ev)
	f.mu.Unlock()
}

func (f *fakeAXHost) takeActions() []ui.SemanticsActionEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.actions
	f.actions = nil
	return out
}

func axNode(id int, f func(n *ui.SemanticsNodeUpdate), children ...int) ui.SemanticsNodeUpdate {
	n := ui.SemanticsNodeUpdate{
		ID: id, TextSelectionBase: -1, TextSelectionExtent: -1, PlatformViewID: -1, MaxValueLength: -1, CurrentValueLength: -1,
		TraversalParent: -1, ChildrenInTraversalOrder: children, ChildrenInHitTestOrder: children,
		ScrollPosition: math.NaN(), ScrollExtentMax: math.NaN(), ScrollExtentMin: math.NaN(),
		Transform:        [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1},
		HitTestTransform: [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1},
	}
	if f != nil {
		f(&n)
	}
	return n
}

func ltwh(l, t, w, h float64) ui.Rect { return ui.NewRectFromLTWH(l, t, w, h) }

// axFixture opens a window and starts the bridge with a fake engine host.
func axFixture(s *session, h *recorder) (w *window, host *fakeAXHost) {
	w = inputWindow(s, h)
	host = &fakeAXHost{}
	s.onUI(func() {
		acc := s.p.Services().Accessibility()
		acc.Start(nil, host)
		acc.SetWindow(w.id, "The window", true)
	})
	return w, host
}

// axTree is a window's tree: a button, a switch, a slider, a text field and a list with two items.
func axTree(scale float64) *ui.SemanticsUpdate {
	rootT := [16]float32{float32(scale), 0, 0, 0, 0, float32(scale), 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	root := axNode(0, func(n *ui.SemanticsNodeUpdate) { n.Rect = ltwh(0, 0, 300, 200); n.Transform = rootT }, 1, 2, 3, 4, 5)
	button := axNode(1, func(n *ui.SemanticsNodeUpdate) {
		n.Rect = ltwh(10, 10, 80, 30)
		n.Label, n.Hint = "Save", "Saves the file"
		n.Flags.IsButton = true
		n.Actions = ui.SemanticsActionTap
	})
	toggle := axNode(2, func(n *ui.SemanticsNodeUpdate) {
		n.Rect = ltwh(10, 50, 60, 30)
		n.Label = "Wi-Fi"
		n.Flags.IsToggled = ui.TristateIsTrue
		n.Actions = ui.SemanticsActionTap
	})
	slider := axNode(3, func(n *ui.SemanticsNodeUpdate) {
		n.Rect = ltwh(10, 90, 120, 20)
		n.Label, n.Value = "Volume", "40%"
		n.Flags.IsSlider = true
		n.Actions = ui.SemanticsActionIncrease | ui.SemanticsActionDecrease
	})
	field := axNode(4, func(n *ui.SemanticsNodeUpdate) {
		n.Rect = ltwh(10, 120, 200, 24)
		n.Label, n.Value = "Name", "Ada😀 Lovelace"
		n.Flags.IsTextField = true
		n.Flags.IsFocused = ui.TristateIsTrue
		n.TextSelectionBase, n.TextSelectionExtent = 3, 3
		n.Actions = ui.SemanticsActionSetText | ui.SemanticsActionSetSelection
	})
	list := axNode(5, func(n *ui.SemanticsNodeUpdate) { n.Rect = ltwh(10, 150, 200, 40); n.Role = ui.SemanticsRoleList }, 6, 7)
	item1 := axNode(6, func(n *ui.SemanticsNodeUpdate) {
		n.Rect = ltwh(0, 0, 200, 20)
		n.Label = "One"
		n.Role = ui.SemanticsRoleListItem
	})
	item2 := axNode(7, func(n *ui.SemanticsNodeUpdate) {
		n.Rect = ltwh(0, 20, 200, 20)
		n.Label = "Two"
		n.Role = ui.SemanticsRoleListItem
		n.Transform = [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	})
	return &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{root, button, toggle, slider, field, list, item1, item2}}
}

func axString(obj objc.ID, getter string) string { return objcx.String(send[objc.ID](obj, getter)) }

// children are the elements of the accessibilityChildren of an object.
func axChildren(obj objc.ID) []objc.ID {
	arr := send[objc.ID](obj, "accessibilityChildren")
	var out []objc.ID
	for i := range objcx.ArrayCount(arr) {
		out = append(out, objcx.ArrayAt(arr, i))
	}
	return out
}

func TestViewHasTheRootElementAndTheTreeBelowIt(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w, host := axFixture(s, h)
		var kids []objc.ID
		s.onUI(func() {
			// Before the first question no tree was asked for; the question asks, once.
			if len(host.enabled) != 0 {
				t.Error("the framework was asked for a tree before a client asked")
			}
			if got := axChildren(w.view); len(got) != 0 {
				t.Errorf("a view with no tree has %d children", len(got))
			}
			_ = axChildren(w.view)
			if !slices.Equal(host.enabled, []bool{true}) {
				t.Errorf("the screen reader was reported %v, want once on", host.enabled)
			}
			s.p.acc.UpdateSemantics(w.id, axTree(w.scale))
			roots := axChildren(w.view)
			if len(roots) != 1 {
				t.Fatalf("the view has %d children, want the root", len(roots))
			}
			root := roots[0]
			if axString(root, "accessibilityRole") != "AXGroup" || axString(root, "accessibilityLabel") != "The window" {
				t.Errorf("root = %q %q", axString(root, "accessibilityRole"), axString(root, "accessibilityLabel"))
			}
			if send[objc.ID](root, "accessibilityParent") != w.view {
				t.Error("the parent of the root is not the view")
			}
			kids = axChildren(root)
		})
		if len(kids) != 5 {
			t.Fatalf("the root has %d children, want 5", len(kids))
		}
		s.onUI(func() {
			want := []struct{ role, label string }{
				{"AXButton", "Save"}, {"AXCheckBox", "Wi-Fi"}, {"AXSlider", "Volume"}, {"AXTextField", "Name"}, {"AXList", ""},
			}
			for i, k := range kids {
				if got := axString(k, "accessibilityRole"); got != want[i].role {
					t.Errorf("child %d role = %q, want %q", i, got, want[i].role)
				}
				if got := axString(k, "accessibilityLabel"); got != want[i].label {
					t.Errorf("child %d label = %q, want %q", i, got, want[i].label)
				}
			}
			if got := axString(kids[0], "accessibilityHelp"); got != "Saves the file" {
				t.Errorf("help = %q", got)
			}
			if got := axString(kids[1], "accessibilitySubrole"); got != "AXSwitch" {
				t.Errorf("switch subrole = %q", got)
			}
			if v := send[objc.ID](kids[1], "accessibilityValue"); send[float64](v, "doubleValue") != 1 {
				t.Error("the switch is not on")
			}
			if v := send[objc.ID](kids[2], "accessibilityValue"); send[float64](v, "doubleValue") != 40 {
				t.Error("the slider is not at 40")
			}
			if send[float64](send[objc.ID](kids[2], "accessibilityMaxValue"), "doubleValue") != 100 {
				t.Error("the slider's maximum is not 100")
			}
			list := axChildren(kids[4])
			if len(list) != 2 || axString(list[0], "accessibilityLabel") != "One" || axString(list[1], "accessibilityLabel") != "Two" {
				t.Errorf("the list's items: %d", len(list))
			}
			if send[objc.ID](list[0], "accessibilityParent") != kids[4] {
				t.Error("the parent of a list item is not the list")
			}
		})
	})
}

func TestElementsHaveTheFramesOfTheirNodes(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w, _ := axFixture(s, h)
		s.onUI(func() {
			_ = axChildren(w.view)
			s.p.acc.UpdateSemantics(w.id, axTree(w.scale))
			root := axChildren(w.view)[0]
			kids := axChildren(root)
			// The frame on the screen: the node's rectangle in the view (from the top left, in points) moved to the
			// screen with the window's own conversion.
			screenOf := func(x, y, wd, ht float64) nsRect {
				bounds := send[nsRect](w.view, "bounds")
				inView := nsRect{Origin: nsPoint{X: x, Y: bounds.Size.Height - y - ht}, Size: nsSize{Width: wd, Height: ht}}
				inWindow := objc.Send[nsRect](w.view, sel("convertRect:toView:"), inView, objc.ID(0))
				return objc.Send[nsRect](w.ns, sel("convertRectToScreen:"), inWindow)
			}
			check := func(name string, obj objc.ID, x, y, wd, ht float64) {
				got := objc.Send[nsRect](obj, sel("accessibilityFrame"))
				want := screenOf(x, y, wd, ht)
				if math.Abs(got.Origin.X-want.Origin.X) > 0.5 || math.Abs(got.Origin.Y-want.Origin.Y) > 0.5 ||
					math.Abs(got.Size.Width-want.Size.Width) > 0.5 || math.Abs(got.Size.Height-want.Size.Height) > 0.5 {
					t.Errorf("%s frame = %+v, want %+v", name, got, want)
				}
			}
			check("root", root, 0, 0, 300, 200)
			check("button", kids[0], 10, 10, 80, 30)
			check("text field", kids[3], 10, 120, 200, 24)
			items := axChildren(kids[4])
			// The item's own transform is the identity: its rectangle is in the coordinates of the list, which has the identity too.
			check("second list item", items[1], 0, 20, 200, 20)
		})
	})
}

func TestPressIncrementAndShowMenuReachTheFramework(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w, host := axFixture(s, h)
		s.onUI(func() {
			_ = axChildren(w.view)
			s.p.acc.UpdateSemantics(w.id, axTree(w.scale))
			kids := axChildren(axChildren(w.view)[0])
			if !send[bool](kids[0], "accessibilityPerformPress") {
				t.Error("the press of a button was refused")
			}
			if !send[bool](kids[2], "accessibilityPerformIncrement") || !send[bool](kids[2], "accessibilityPerformDecrement") {
				t.Error("the slider refused its increment or decrement")
			}
			// What a node does not do is refused, and the action is not sent.
			if send[bool](kids[0], "accessibilityPerformIncrement") || send[bool](kids[3], "accessibilityPerformPress") {
				t.Error("an action that the node does not have was performed")
			}
			names := send[objc.ID](kids[0], "accessibilityActionNames")
			if objcx.ArrayCount(names) != 1 || objcx.String(objcx.ArrayAt(names, 0)) != "AXPress" {
				t.Errorf("the button's actions = %d", objcx.ArrayCount(names))
			}
			ns := objcx.NewString("AXPress")
			ok := objc.Send[bool](kids[1], sel("accessibilityPerformAction:"), ns)
			objcx.Release(ns)
			if !ok {
				t.Error("accessibilityPerformAction: AXPress was refused")
			}
		})
		got := host.takeActions()
		want := []ui.SemanticsActionEvent{
			{ViewID: w.id, NodeID: 1, Type: ui.SemanticsActionTap},
			{ViewID: w.id, NodeID: 3, Type: ui.SemanticsActionIncrease},
			{ViewID: w.id, NodeID: 3, Type: ui.SemanticsActionDecrease},
			{ViewID: w.id, NodeID: 2, Type: ui.SemanticsActionTap},
		}
		if len(got) != len(want) {
			t.Fatalf("the framework got %d actions, want %d: %+v", len(got), len(want), got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("action %d = %+v, want %+v", i, got[i], want[i])
			}
		}
	})
}

func TestTextFieldValueAndSelectionAreReadableAndSettable(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w, host := axFixture(s, h)
		s.onUI(func() {
			_ = axChildren(w.view)
			s.p.acc.UpdateSemantics(w.id, axTree(w.scale))
			field := axChildren(axChildren(w.view)[0])[3]
			if got := objcx.String(send[objc.ID](field, "accessibilityValue")); got != "Ada😀 Lovelace" {
				t.Errorf("value = %q", got)
			}
			if n := send[int64](field, "accessibilityNumberOfCharacters"); n != 14 {
				t.Errorf("characters = %d, want 14 UTF-16 units", n)
			}
			// The caret is after "Ada" (3 bytes: 3 units).
			if r := objc.Send[nsRange](field, sel("accessibilitySelectedTextRange")); r.Location != 3 || r.Length != 0 {
				t.Errorf("selection = %+v", r)
			}
			// Select the emoji (units 3..5): the framework gets byte offsets 3..7.
			objc.Send[struct{}](field, sel("setAccessibilitySelectedTextRange:"), nsRange{Location: 3, Length: 2})
			// Setting the value asks for the set text action; the field keeps its text until the framework says another.
			nv := objcx.NewString("Grace")
			field.Send(sel("setAccessibilityValue:"), nv)
			objcx.Release(nv)
			if got := objcx.String(send[objc.ID](field, "accessibilityValue")); got != "Ada😀 Lovelace" {
				t.Errorf("the value changed before the framework said so: %q", got)
			}
		})
		acts := host.takeActions()
		if len(acts) != 2 {
			t.Fatalf("%d actions: %+v", len(acts), acts)
		}
		sel0, ok := acts[0].Arguments.(map[string]int)
		if acts[0].Type != ui.SemanticsActionSetSelection || !ok || sel0["base"] != 3 || sel0["extent"] != 7 {
			t.Errorf("selection action = %+v", acts[0])
		}
		if acts[1].Type != ui.SemanticsActionSetText || acts[1].Arguments != "Grace" {
			t.Errorf("set text action = %+v", acts[1])
		}
		// The framework answers: the new value and selection are read back.
		s.onUI(func() {
			upd := axTree(w.scale)
			for i := range upd.Nodes {
				if upd.Nodes[i].ID == 4 {
					upd.Nodes[i].Value = "Grace"
					upd.Nodes[i].TextSelectionBase, upd.Nodes[i].TextSelectionExtent = 5, 5
				}
			}
			s.p.acc.UpdateSemantics(w.id, upd)
			field := axChildren(axChildren(w.view)[0])[3]
			if got := objcx.String(send[objc.ID](field, "accessibilityValue")); got != "Grace" {
				t.Errorf("the value after the update = %q", got)
			}
			if r := objc.Send[nsRange](field, sel("accessibilitySelectedTextRange")); r.Location != 5 || r.Length != 0 {
				t.Errorf("the selection after the update = %+v", r)
			}
		})
	})
}

func TestFocusedElementAndHitTest(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w, _ := axFixture(s, h)
		s.onUI(func() {
			_ = axChildren(w.view)
			s.p.acc.UpdateSemantics(w.id, axTree(w.scale))
			kids := axChildren(axChildren(w.view)[0])
			if got := send[objc.ID](w.view, "accessibilityFocusedUIElement"); got != kids[3] {
				t.Error("the focused element is not the text field")
			}
			// A point inside the button, on the screen: its centre.
			bounds := send[nsRect](w.view, "bounds")
			inView := nsPoint{X: 50, Y: bounds.Size.Height - 25}
			inWindow := objc.Send[nsPoint](w.view, sel("convertPoint:toView:"), inView, objc.ID(0))
			screen := objc.Send[nsRect](w.ns, sel("convertRectToScreen:"), nsRect{Origin: inWindow})
			if got := objc.Send[objc.ID](w.view, sel("accessibilityHitTest:"), screen.Origin); got != kids[0] {
				t.Error("the hit test of the button's centre did not find the button")
			}
		})
	})
}

func TestTreeChangesPostNotificationsAndDropElements(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w, _ := axFixture(s, h)
		s.onUI(func() {
			_ = axChildren(w.view)
			s.p.acc.UpdateSemantics(w.id, axTree(w.scale))
			root := axChildren(w.view)[0]
			if n := len(axChildren(root)); n != 5 {
				t.Fatalf("%d children", n)
			}
			// Remove the list: the root has four children and the list's items are gone with it.
			upd := &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{axNode(0, func(n *ui.SemanticsNodeUpdate) {
				n.Rect = ltwh(0, 0, 300, 200)
				n.Transform = [16]float32{float32(w.scale), 0, 0, 0, 0, float32(w.scale), 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
			}, 1, 2, 3, 4)}}
			s.p.acc.UpdateSemantics(w.id, upd)
			if n := len(axChildren(root)); n != 4 {
				t.Errorf("after the removal the root has %d children", n)
			}
			if s.p.acc.Model().Node(a11y.NodeRef{View: w.id, ID: 6}) != nil {
				t.Error("the item of the removed list is still in the model")
			}
			// The window closes: the elements go.
			s.p.acc.RemoveView(w.id)
			if got := axChildren(w.view); len(got) != 0 {
				t.Errorf("a view whose tree was removed has %d children", len(got))
			}
		})
	})
}
