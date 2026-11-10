//go:build ios

package ios

import (
	"context"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin"
	"github.com/rm4n0s/glutter/internal/platform/darwin/a11ymodel"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/internal/platform/ios/a11y"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the accessibility bridge of iOS (SPEC §30.7): the semantics tree as UIAccessibilityElements for VoiceOver, Switch Control and
// Voice Control, ported from Flutter's SemanticsObject.mm and accessibility_bridge.mm. The model of the trees is a11ymodel (shared with
// macOS) and the rules of what a node is to UIKit are package a11y (plain Go).
//
// Deviation: Flutter makes a tree of containers (a SemanticsObjectContainer for each node that has children, and a UIScrollView that
// mirrors each scrollable so that VoiceOver scrolls it natively). Glutter gives the view a flat list of the elements in traversal order,
// with their frames in screen space; the three-finger scroll reaches the framework through accessibilityScroll: (SPEC §12.10, §30.7).

// The notifications that the bridge posts (UIAccessibilityNotifications).
const (
	notificationScreenChanged = 1000
	notificationLayoutChanged = 1001
	notificationAnnouncement  = 1008
)

// A11yEnvEnv forces the semantics tree on ("on") or off ("off") in debug builds, whatever the system's switches say.
//
// Flutter: no counterpart
const A11yEnv = "GLUTTER_IOS_A11Y"

// a11yBridge is platform.Accessibility of iOS. All its methods but Close belong to the UI thread.
type a11yBridge struct {
	p    *Platform
	host platform.AccessibilityHost

	model *a11ymodel.Model
	elems map[a11ymodel.NodeRef]*a11yElement
	// custom are the labels of the custom actions, by ID, from the updates of the tree.
	custom map[int]string
	// enabled says that the framework was asked for the tree.
	enabled bool
	// focused is the element that has the accessibility focus.
	focused    a11ymodel.NodeRef
	hasFocused bool
	closed     bool
}

var _ platform.Accessibility = (*a11yBridge)(nil)

// a11yElement is the UIAccessibilityElement of a node.
type a11yElement struct {
	b      *a11yBridge
	ref    a11ymodel.NodeRef
	obj    objc.ID
	handle objcx.Handle
	// actions is the NSArray of the custom actions that the element has now (retained), 0 for none.
	actions objc.ID
}

// a11yClasses are what the bridge loads once: the class of the elements and the values of the traits.
type a11yClasses struct {
	element *objcx.RegisteredClass
	traits  map[a11y.Trait]uint64
	post    func(notification uint32, argument uintptr)
}

// loadA11y registers the class of the elements and reads the traits' values and the posting function from UIKit.
func loadA11y() (*a11yClasses, error) {
	uikit, err := dl.OpenFramework("UIKit")
	if err != nil {
		return nil, err
	}
	c := &a11yClasses{traits: map[a11y.Trait]uint64{}}
	for tr, name := range a11y.TraitNames {
		addr, err := uikit.Symbol(name)
		if err != nil {
			return nil, err
		}
		c.traits[tr] = *(*uint64)(unsafe.Pointer(dl.Pointer(addr)))
	}
	if err := uikit.Bind(&c.post, "UIAccessibilityPostNotification"); err != nil {
		return nil, err
	}
	if c.element, err = registerA11yElement(); err != nil {
		return nil, err
	}
	return c, nil
}

// elementOf is the element of the object UIKit called, nil when the bridge is gone.
func elementOf(self objc.ID) *a11yElement {
	e, _ := objcx.LookupAs[*a11yElement](objcx.HandleOf(self))
	if e == nil || e.b.closed {
		return nil
	}
	return e
}

// registerA11yElement registers the class of the elements: a UIAccessibilityElement whose actions are methods of Go.
//
// Flutter: shell/platform/darwin/ios/framework/Source/SemanticsObject.mm SemanticsObject
func registerA11yElement() (*objcx.RegisteredClass, error) {
	// do runs fn for the element under the platform's guard, and answers no when there is none.
	do := func(self objc.ID, fn func(e *a11yElement) bool) (ok bool) {
		if e := elementOf(self); e != nil {
			e.b.p.guard(func() { ok = fn(e) })
		}
		return ok
	}
	return objcx.RegisterClass(objcx.ClassDef{
		Name: "SemanticsElement", Super: "UIAccessibilityElement",
		Methods: []objcx.Method{
			{Name: "accessibilityActivate", Fn: func(self objc.ID, _ objc.SEL) bool {
				return do(self, func(e *a11yElement) bool { return e.activate() })
			}},
			{Name: "accessibilityIncrement", Fn: func(self objc.ID, _ objc.SEL) {
				do(self, func(e *a11yElement) bool { e.adjust(true); return true })
			}},
			{Name: "accessibilityDecrement", Fn: func(self objc.ID, _ objc.SEL) {
				do(self, func(e *a11yElement) bool { e.adjust(false); return true })
			}},
			{Name: "accessibilityScroll:", Fn: func(self objc.ID, _ objc.SEL, direction int64) bool {
				return do(self, func(e *a11yElement) bool { return e.scroll(a11y.ScrollDirection(direction), false) })
			}},
			{Name: "accessibilityPerformEscape", Fn: func(self objc.ID, _ objc.SEL) bool {
				return do(self, func(e *a11yElement) bool { return e.dispatchIfHas(ui.SemanticsActionDismiss, nil) })
			}},
			{Name: "accessibilityElementDidBecomeFocused", Fn: func(self objc.ID, _ objc.SEL) {
				do(self, func(e *a11yElement) bool { e.focusChanged(true); return true })
			}},
			{Name: "accessibilityElementDidLoseFocus", Fn: func(self objc.ID, _ objc.SEL) {
				do(self, func(e *a11yElement) bool { e.focusChanged(false); return true })
			}},
		},
	})
}

// newA11yBridge makes the bridge of the platform.
func newA11yBridge(p *Platform) *a11yBridge {
	return &a11yBridge{p: p, model: a11ymodel.NewModel(a11y.Describe), elems: map[a11ymodel.NodeRef]*a11yElement{}, custom: map[int]string{}}
}

// Start remembers the host and turns the framework's tree on when it is wanted: always in the Simulator (there is no way to tell that
// the Accessibility Inspector is on, so the tree is made for it, as Flutter does) and, on a device, while VoiceOver, Switch Control or
// Speak Screen is running; statusChanged follows the switches.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterViewController.mm onAccessibilityStatusChanged:
func (b *a11yBridge) Start(_ context.Context, host platform.AccessibilityHost) {
	b.host = host
	b.statusChanged()
}

// wanted says whether the framework should make the semantics tree.
func (b *a11yBridge) wanted() bool {
	if foundation.DebugMode {
		switch b.p.getenv(A11yEnv) {
		case "on":
			return true
		case "off":
			return false
		}
	}
	if b.p.getenv("SIMULATOR_DEVICE_NAME") != "" {
		return true
	}
	in := b.p.settingsLast
	if !b.p.settingsHave {
		in = b.p.readSettings()
	}
	return in.VoiceOver || in.SwitchControl || in.SpeakScreen
}

// statusChanged tells the host when the wish for the tree changed.
func (b *a11yBridge) statusChanged() {
	if b.host == nil || b.closed {
		return
	}
	if w := b.wanted(); w != b.enabled {
		b.enabled = w
		b.host.ScreenReaderChanged(w)
	}
}

// SetWindow tells the bridge the title of the window and whether it is active; the elements do not use them.
func (b *a11yBridge) SetWindow(view ui.ViewID, title string, active bool) {
	b.model.SetWindow(view, title, active)
}

// RemoveView forgets the elements of the view.
func (b *a11yBridge) RemoveView(view ui.ViewID) {
	for _, c := range b.model.RemoveView(view) {
		if c.Kind == a11ymodel.ChangeNodeRemoved {
			b.dropElement(c.Ref)
		}
	}
}

// Close drops every element. It is for the UI thread here (the engine calls it there).
func (b *a11yBridge) Close() {
	if b.closed {
		return
	}
	b.closed = true
	for ref := range b.elems {
		b.dropElement(ref)
	}
}

// UpdateSemantics applies the update to the model and makes the elements follow it, then posts what changed: a layout change when the
// elements came or went, a screen change when a route appeared, an announcement for a live region that changed.
//
// Flutter: shell/platform/darwin/ios/framework/Source/accessibility_bridge.mm AccessibilityBridge::UpdateSemantics
func (b *a11yBridge) UpdateSemantics(view ui.ViewID, update *ui.SemanticsUpdate) {
	if b.closed || update == nil {
		return
	}
	for _, ca := range update.CustomActions {
		b.custom[ca.ID] = ca.Label
	}
	changes := b.model.Apply(view, update)
	layout, screen := false, false
	var announce []string
	for _, c := range changes {
		switch c.Kind {
		case a11ymodel.ChangeNodeRemoved:
			b.dropElement(c.Ref)
			layout = true
		case a11ymodel.ChangeNodeAdded:
			b.syncElement(c.Ref)
			layout = true
			if n := b.model.Node(c.Ref); n != nil && n.Flags.ScopesRoute {
				screen = true
			}
		case a11ymodel.ChangeChildren, a11ymodel.ChangeUpdated:
			b.syncElement(c.Ref)
			layout = layout || c.Kind == a11ymodel.ChangeChildren
		case a11ymodel.ChangeAnnounce:
			if n := b.model.Node(c.Ref); n != nil && a11y.LabelOf(n) != "" {
				announce = append(announce, a11y.LabelOf(n))
			}
		}
	}
	switch {
	case screen:
		b.post(notificationScreenChanged, 0)
	case layout:
		b.post(notificationLayoutChanged, 0)
	}
	for _, text := range announce {
		objcx.WithPool(func() {
			s := objcx.NewString(text)
			b.p.s.a11y.post(notificationAnnouncement, uintptr(s))
			objcx.Release(s)
		})
	}
}

// post posts a notification of UIAccessibility with an argument (an object or 0).
func (b *a11yBridge) post(n uint32, arg uintptr) { b.p.s.a11y.post(n, arg) }

// syncElement makes the element of the node the way the node is now: the node that is an element gets one with its properties, the
// node that is not loses it.
func (b *a11yBridge) syncElement(ref a11ymodel.NodeRef) {
	n := b.model.Node(ref)
	if n == nil || !a11y.IsElement(n) {
		b.dropElement(ref)
		return
	}
	w := b.p.windows[ref.View]
	if w == nil || w.closed {
		return
	}
	e := b.elems[ref]
	if e == nil {
		e = &a11yElement{b: b, ref: ref}
		e.handle = objcx.NewHandle(e)
		objcx.WithPool(func() {
			e.obj = b.p.s.a11y.element.New(e.handle)
			e.obj.Send(sel("initWithAccessibilityContainer:"), w.view)
			e.obj.Send(sel("setIsAccessibilityElement:"), true)
		})
		b.elems[ref] = e
	}
	objcx.WithPool(func() { e.apply(n, w) })
}

// dropElement releases the element of the node, if there is one.
func (b *a11yBridge) dropElement(ref a11ymodel.NodeRef) {
	e := b.elems[ref]
	if e == nil {
		return
	}
	delete(b.elems, ref)
	if b.hasFocused && b.focused == ref {
		b.hasFocused = false
	}
	objcx.DropHandle(e.handle)
	objcx.Release(e.actions)
	objcx.Release(e.obj)
	e.obj, e.actions = 0, 0
}

// refreshFrames makes the frames of the elements of the view follow the view (a rotation moves it on the screen).
func (b *a11yBridge) refreshFrames(w *window) {
	if b.closed {
		return
	}
	objcx.WithPool(func() {
		for ref, e := range b.elems {
			if ref.View != w.id {
				continue
			}
			if n := b.model.Node(ref); n != nil {
				e.setFrame(n, w)
			}
		}
	})
}

// elementList is the elements of the view in traversal order: the nodes that are elements and are not hidden (a scrollable that is hidden
// but still a target of navigation is kept).
func (b *a11yBridge) elementList(view ui.ViewID) []objc.ID {
	var out []objc.ID
	var walk func(n *a11ymodel.Node)
	walk = func(n *a11ymodel.Node) {
		if n == nil {
			return
		}
		if e := b.elems[n.Ref()]; e != nil && (!n.Flags.IsHidden || n.Flags.HasImplicitScrolling) {
			out = append(out, e.obj)
		}
		for _, c := range b.model.Children(n) {
			walk(b.model.Node(c))
		}
	}
	walk(b.model.Root(view))
	return out
}

// apply gives the element the properties of the node.
func (e *a11yElement) apply(n *a11ymodel.Node, w *window) {
	set := func(setter, value string) {
		if value == "" {
			e.obj.Send(sel(setter), objc.ID(0))
			return
		}
		s := objcx.NewString(value)
		e.obj.Send(sel(setter), s)
		objcx.Release(s)
	}
	set("setAccessibilityLabel:", a11y.LabelOf(n))
	set("setAccessibilityValue:", a11y.ValueOf(n))
	set("setAccessibilityHint:", a11y.HintOf(n))
	set("setAccessibilityIdentifier:", n.Identifier)
	e.obj.Send(sel("setAccessibilityTraits:"), e.traits(n))
	if send[bool](e.obj, "respondsToSelector:", sel("setAccessibilityRespondsToUserInteraction:")) {
		e.obj.Send(sel("setAccessibilityRespondsToUserInteraction:"), a11y.RespondsToUserInteraction(n))
	}
	e.setFrame(n, w)
	e.setCustomActions(n)
}

// traits are the UIKit traits for the node, from the values that UIKit's symbols hold.
func (e *a11yElement) traits(n *a11ymodel.Node) uint64 {
	var out uint64
	have := e.b.model.Children(n)
	for tr, v := range e.b.p.s.a11y.traits {
		if a11y.TraitsOf(n, len(have) > 0)&tr != 0 {
			out |= v
		}
	}
	return out
}

// setFrame puts the frame of the node on the element, in screen coordinates (points).
func (e *a11yElement) setFrame(n *a11ymodel.Node, w *window) {
	px := e.b.model.Extents(n)
	s := w.screenScale()
	local := darwin.CGRect{Origin: darwin.CGPoint{X: px.X / s, Y: px.Y / s}, Size: darwin.CGSize{Width: px.Width / s, Height: px.Height / s}}
	space := send[objc.ID](class("UIScreen").Send(sel("mainScreen")), "coordinateSpace")
	frame := send[darwin.CGRect](w.view, "convertRect:toCoordinateSpace:", local, space)
	e.obj.Send(sel("setAccessibilityFrame:"), frame)
}

// setCustomActions gives the element the custom actions of the node: a UIAccessibilityCustomAction for each, whose handler asks the
// framework for it.
func (e *a11yElement) setCustomActions(n *a11ymodel.Node) {
	objcx.Release(e.actions)
	e.actions = 0
	if len(n.AdditionalActions) == 0 {
		e.obj.Send(sel("setAccessibilityCustomActions:"), objc.ID(0))
		return
	}
	var list []objc.ID
	for _, id := range n.AdditionalActions {
		label := e.b.custom[id]
		if label == "" {
			continue
		}
		name := objcx.NewString(label)
		blk := objcx.NewBlock(func(_ objc.Block, _ objc.ID) bool {
			if e.b.closed || e.obj == 0 {
				return false
			}
			ok := false
			e.b.p.guard(func() { ok = e.dispatch(ui.SemanticsActionCustomAction, id) })
			return ok
		})
		kept := blk.Copy()
		blk.Release()
		act := class("UIAccessibilityCustomAction").Send(sel("alloc")).Send(sel("initWithName:actionHandler:"), name, kept)
		objcx.Release(name)
		list = append(list, act)
	}
	arr := objcx.NewArray(list...)
	for _, a := range list {
		objcx.Release(a)
	}
	e.actions = arr
	e.obj.Send(sel("setAccessibilityCustomActions:"), arr)
}

// node is the node of the element as the model has it now.
func (e *a11yElement) node() *a11ymodel.Node { return e.b.model.Node(e.ref) }

// dispatch asks the framework for the action on the node; it is false when there is no host to ask.
func (e *a11yElement) dispatch(action ui.SemanticsAction, args any) bool {
	if e.b.host == nil {
		return false
	}
	e.b.host.PerformAction(ui.SemanticsActionEvent{ViewID: e.ref.View, NodeID: e.ref.ID, Type: action, Arguments: args})
	return true
}

// dispatchIfHas is dispatch for a node that has the action; a node that has not answers no.
func (e *a11yElement) dispatchIfHas(action ui.SemanticsAction, args any) bool {
	if n := e.node(); n != nil && a11y.Has(n, action) {
		return e.dispatch(action, args)
	}
	return false
}

// activate is a double tap: a tap on the node.
//
// Flutter: SemanticsObject.mm accessibilityActivate
func (e *a11yElement) activate() bool {
	n := e.node()
	if n == nil {
		return false
	}
	action, send, ok := a11y.ActivateAction(n)
	if send {
		return e.dispatch(action, nil)
	}
	return ok
}

// adjust is a swipe up or down on an adjustable element: the element shows the new value at once, and the framework is asked to change it.
//
// Flutter: SemanticsObject.mm accessibilityIncrement, accessibilityDecrement
func (e *a11yElement) adjust(up bool) {
	n := e.node()
	if n == nil {
		return
	}
	action, value := ui.SemanticsActionDecrease, n.DecreasedValue
	if up {
		action, value = ui.SemanticsActionIncrease, n.IncreasedValue
	}
	if !a11y.Has(n, action) {
		return
	}
	if value != "" {
		objcx.WithPool(func() {
			s := objcx.NewString(value)
			e.obj.Send(sel("setAccessibilityValue:"), s)
			objcx.Release(s)
		})
	}
	e.dispatch(action, nil)
}

// scroll is a three-finger scroll. The element answers when its own node scrolls that way; fromAncestors makes it look up the tree for a
// node that does (the view's answer, for the element that has the focus).
//
// Flutter: SemanticsObject.mm accessibilityScroll:
func (e *a11yElement) scroll(d a11y.ScrollDirection, fromAncestors bool) bool {
	action, ok := a11y.ScrollActionOf(d)
	if !ok {
		return false
	}
	for n := e.node(); n != nil; n = e.b.model.ParentOf(n) {
		if a11y.Has(n, action) {
			return (&a11yElement{b: e.b, ref: n.Ref()}).dispatch(action, nil)
		}
		if !fromAncestors {
			break
		}
	}
	return false
}

// focusChanged is VoiceOver moving its focus to the element or away from it: the framework hears of it, and a hidden element or a heading
// is shown on the screen.
//
// Flutter: SemanticsObject.mm accessibilityElementDidBecomeFocused, accessibilityElementDidLoseFocus
func (e *a11yElement) focusChanged(gained bool) {
	n := e.node()
	if n == nil {
		return
	}
	if gained {
		e.b.focused, e.b.hasFocused = e.ref, true
		if n.Flags.IsHidden || n.HeadingLevel > 0 {
			e.dispatchIfHas(ui.SemanticsActionShowOnScreen, nil)
		}
		e.dispatchIfHas(ui.SemanticsActionDidGainAccessibilityFocus, nil)
		return
	}
	if e.b.hasFocused && e.b.focused == e.ref {
		e.b.hasFocused = false
	}
	e.dispatchIfHas(ui.SemanticsActionDidLoseAccessibilityFocus, nil)
}

// scrollFromFocus is the view's answer to a three-finger scroll: the element that has the focus, or its nearest ancestor that scrolls that
// way, scrolls.
func (b *a11yBridge) scrollFromFocus(d a11y.ScrollDirection) bool {
	if !b.hasFocused {
		return false
	}
	return (&a11yElement{b: b, ref: b.focused}).scroll(d, true)
}

// a11yServices is platform.Services of iOS: the accessibility bridge; the gamepads, the audio and the sensors come with step 35.8.
type a11yServices struct {
	acc platform.Accessibility
}

var _ platform.Services = a11yServices{}

func (s a11yServices) Accessibility() platform.Accessibility { return s.acc }
func (s a11yServices) Gamepads() platform.Gamepads           { return nil }
func (s a11yServices) Audio() platform.Audio                 { return nil }
func (s a11yServices) Sensors() platform.Sensors             { return nil }

// a11yState is the platform's field for the bridge.
type a11yState struct {
	accOnce sync.Once
	acc     *a11yBridge
}

// Services returns the services of the platform: the accessibility bridge (SPEC §30.7). It is made at the first call, which is the
// engine's.
//
// Flutter: no counterpart
func (p *Platform) Services() platform.Services {
	p.accOnce.Do(func() { p.acc = newA11yBridge(p) })
	return a11yServices{acc: p.acc}
}

// accessibilityMethods are the methods of the view that make it the container of the elements.
func accessibilityMethods() []objcx.Method {
	return []objcx.Method{
		{Name: "isAccessibilityElement", Fn: func(objc.ID, objc.SEL) bool { return false }},
		{Name: "accessibilityElements", Fn: func(self objc.ID, _ objc.SEL) objc.ID {
			w := windowOf(self)
			if w == nil || w.closed || w.p.acc == nil {
				return 0
			}
			return objcx.Autorelease(objcx.NewArray(w.p.acc.elementList(w.id)...))
		}},
		{Name: "accessibilityScroll:", Fn: func(self objc.ID, _ objc.SEL, direction int64) bool {
			w := windowOf(self)
			if w == nil || w.closed || w.p.acc == nil {
				return false
			}
			return w.p.guardBool(func() bool { return w.p.acc.scrollFromFocus(a11y.ScrollDirection(direction)) })
		}},
	}
}

// logA11y logs the elements of the window as UIKit reads them back from the objects: the count, and for each its label, value, hint, traits
// and frame (the dev events step "a11y"; the Simulator tests read the lines).
func (p *Platform) logA11y(w *window) {
	if p.acc == nil {
		p.log.Info(A11yElementsLogLine, "count", 0)
		return
	}
	objcx.WithPool(func() {
		list := w.view.Send(sel("accessibilityElements"))
		n := 0
		if list != 0 {
			n = objcx.ArrayCount(list)
		}
		p.log.Info(A11yElementsLogLine, "count", n)
		for i := 0; i < n; i++ {
			e := objcx.ArrayAt(list, i)
			str := func(name string) string { return objcx.String(send[objc.ID](e, name)) }
			f := send[darwin.CGRect](e, "accessibilityFrame")
			p.log.Info(A11yElementLogLine, "label", str("accessibilityLabel"), "value", str("accessibilityValue"), "hint", str("accessibilityHint"),
				"traits", send[uint64](e, "accessibilityTraits"), "x", f.Origin.X, "y", f.Origin.Y, "w", f.Size.Width, "h", f.Size.Height)
		}
	})
}

// activateA11y activates the element with the label through UIKit's own method, as VoiceOver's double tap does, and logs the answer.
func (p *Platform) activateA11y(w *window, label string) {
	objcx.WithPool(func() {
		list := w.view.Send(sel("accessibilityElements"))
		found, ok := false, false
		if list != 0 {
			for i := 0; i < objcx.ArrayCount(list); i++ {
				e := objcx.ArrayAt(list, i)
				if objcx.String(send[objc.ID](e, "accessibilityLabel")) == label {
					found, ok = true, send[bool](e, "accessibilityActivate")
					break
				}
			}
		}
		p.log.Info(A11yActivateLogLine, "label", label, "found", found, "activated", ok)
	})
}
