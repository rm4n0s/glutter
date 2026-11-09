//go:build darwin && !ios

package a11y

import (
	"context"
	"log/slog"
	"sort"
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// Host is what the bridge asks of the platform. Both methods are called on the UI thread.
//
// Flutter: no counterpart
type Host interface {
	// ViewObjects are the NSView and the NSWindow of the view, and the scale of the window, ok is false for a view that
	// does not exist (any more).
	ViewObjects(view ui.ViewID) (nsView, nsWindow objc.ID, scale float64, ok bool)
	// Guard runs fn so that a panic of the handler is not raised through AppKit's frames.
	Guard(fn func())
}

// The notifications of NSAccessibility that the bridge posts (the values of the constants).
const (
	noteValueChanged   = "AXValueChanged"
	noteTitleChanged   = "AXTitleChanged"
	noteLayoutChanged  = "AXLayoutChanged"
	noteFocusChanged   = "AXFocusedUIElementChanged"
	noteSelectedText   = "AXSelectedTextChanged"
	noteAnnouncement   = "AXAnnouncementRequested"
	noteMoved          = "AXMoved"
	noteResized        = "AXResized"
	noteSelectedChild  = "AXSelectedChildrenChanged"
	noteElementDeleted = "AXUIElementDestroyed"
)

// The announcement's keys and priorities.
const (
	announcementKey = "AXAnnouncementKey"
	priorityKey     = "AXPriorityKey"
	priorityHigh    = 90
)

// element is the NSAccessibilityElement of a node.
type element struct {
	b      *Bridge
	ref    NodeRef
	obj    objc.ID
	handle objcx.Handle
	// internal is set while the bridge sets a property, so that the setters that the clients can call are told apart
	// from the bridge's own (a client that sets the value of a text field asks for the set text action).
	internal bool
}

// Bridge is the accessibility bridge of macOS: it implements platform.Accessibility over the model and the elements.
// Its methods belong to the UI thread but Close, which may be called from any goroutine.
//
// Flutter: shell/platform/darwin/macos/framework/Source/AccessibilityBridgeMac.mm AccessibilityBridgeMac
type Bridge struct {
	host Host
	log  *slog.Logger

	model *Model
	elems map[NodeRef]*element
	// roots are the elements of the roots of the views.
	acc platform.AccessibilityHost
	// enabled says that an assistive technology asked for the tree: the framework makes the semantics tree from then on.
	enabled bool
	closed  bool
}

var _ platform.Accessibility = (*Bridge)(nil)

// NewBridge makes the bridge. It needs the host of the platform.
//
// Flutter: no counterpart
func NewBridge(host Host, log *slog.Logger) *Bridge {
	if log == nil {
		log = slog.Default()
	}
	return &Bridge{host: host, log: log, model: NewModel(), elems: map[NodeRef]*element{}}
}

// Start remembers the host of the engine. The bridge turns the semantics tree on when an assistive technology
// first asks for the elements of a view (Touch), and not before: an app that no screen reader looks at makes no tree.
func (b *Bridge) Start(_ context.Context, acc platform.AccessibilityHost) {
	b.acc = acc
}

// Touch is the first question of an assistive technology: the semantics tree is turned on, once. It stays on for the
// life of the process.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterViewController.mm onAccessibilityStatusChanged:
func (b *Bridge) Touch() {
	if b.enabled || b.closed || b.acc == nil {
		return
	}
	b.enabled = true
	b.acc.ScreenReaderChanged(true)
}

// Enabled says whether an assistive technology asked for the tree.
func (b *Bridge) Enabled() bool { return b.enabled }

// UpdateSemantics applies the update to the model and tells the assistive technologies what changed.
func (b *Bridge) UpdateSemantics(view ui.ViewID, update *ui.SemanticsUpdate) {
	if b.closed || update == nil {
		return
	}
	b.process(b.model.Apply(view, update))
}

// SetWindow sets the title and the focus of the window of the view.
func (b *Bridge) SetWindow(view ui.ViewID, title string, active bool) {
	if b.closed {
		return
	}
	b.process(b.model.SetWindow(view, title, active))
}

// RemoveView forgets the tree of a view that closed.
func (b *Bridge) RemoveView(view ui.ViewID) {
	b.process(b.model.RemoveView(view))
}

// Close stops the bridge and drops the elements. It may be called twice.
func (b *Bridge) Close() {
	if b.closed {
		return
	}
	b.closed = true
	for ref, e := range b.elems {
		b.drop(e)
		delete(b.elems, ref)
	}
}

// RootElement is the element of the root of the view's tree, 0 while the view has none: it is what the view's
// accessibilityChildren has.
func (b *Bridge) RootElement(view ui.ViewID) objc.ID {
	if e := b.elems[NodeRef{view, RootID}]; e != nil {
		return e.obj
	}
	return 0
}

// FocusedElement is the element that has the keyboard focus in the view, or its root.
func (b *Bridge) FocusedElement(view ui.ViewID) objc.ID {
	if n := b.model.Focused(view); n != nil {
		if e := b.elems[n.Ref()]; e != nil {
			return e.obj
		}
	}
	return b.RootElement(view)
}

// ElementAt is the element under the point, in the pixels of the view from its top left.
func (b *Bridge) ElementAt(view ui.ViewID, x, y float64) objc.ID {
	root := b.model.Root(view)
	if root == nil {
		return 0
	}
	if n := b.model.HitTest(root, x, y); n != nil {
		if e := b.elems[n.Ref()]; e != nil {
			return e.obj
		}
	}
	return 0
}

// Model is the mirror of the trees, for the tests.
func (b *Bridge) Model() *Model { return b.model }

// post runs fn under the platform's guard.
func (b *Bridge) guard(fn func()) { b.host.Guard(fn) }

// process makes the changes of the model true in the elements and posts the notifications. The elements are made
// first, then given their properties and children, then the notifications are posted, so that a client that
// answers a notification finds the whole tree.
func (b *Bridge) process(changes []Change) {
	if len(changes) == 0 {
		return
	}
	objcx.WithPool(func() {
		for _, c := range changes {
			if c.Kind == ChangeNodeAdded {
				b.add(c.Ref)
			}
		}
		// The properties of the nodes that were added or changed, and the children of those whose children changed.
		touched := map[NodeRef]bool{}
		for _, c := range changes {
			switch c.Kind {
			case ChangeNodeAdded, ChangeUpdated, ChangeChildren, ChangeTextSelection:
				touched[c.Ref] = true
			}
		}
		refs := make([]NodeRef, 0, len(touched))
		for r := range touched {
			refs = append(refs, r)
		}
		sort.Slice(refs, func(i, j int) bool {
			if refs[i].View != refs[j].View {
				return refs[i].View < refs[j].View
			}
			return refs[i].ID < refs[j].ID
		})
		for _, r := range refs {
			b.sync(r)
		}
		for _, c := range changes {
			b.notify(c)
		}
		for _, c := range changes {
			if c.Kind == ChangeNodeRemoved {
				b.remove(c.Ref)
			}
		}
	})
}

// add makes the element of a node.
func (b *Bridge) add(ref NodeRef) {
	if b.elems[ref] != nil {
		return
	}
	cls, err := elementClass()
	if err != nil {
		b.log.Warn("the accessibility element class could not be made", "err", err)
		return
	}
	e := &element{b: b, ref: ref}
	e.handle = objcx.NewHandle(e)
	e.obj = cls.New(e.handle)
	b.elems[ref] = e
}

// remove drops the element of a node that is gone.
func (b *Bridge) remove(ref NodeRef) {
	if e := b.elems[ref]; e != nil {
		b.drop(e)
		delete(b.elems, ref)
	}
}

// drop cuts the element from the tree and releases it.
func (b *Bridge) drop(e *element) {
	if e.obj == 0 {
		return
	}
	e.internal = true
	e.obj.Send(objcx.Sel("setAccessibilityParent:"), objc.ID(0))
	e.obj.Send(objcx.Sel("setAccessibilityChildren:"), objc.ID(0))
	e.internal = false
	objcx.DropHandle(e.handle)
	objcx.Release(e.obj)
	e.obj = 0
}

func str(s string) objc.ID { return objcx.NewString(s) }

// setString sets a string property, nil for an empty string.
func setString(obj objc.ID, setter, v string) {
	if v == "" {
		obj.Send(objcx.Sel(setter), objc.ID(0))
		return
	}
	s := str(v)
	obj.Send(objcx.Sel(setter), s)
	objcx.Release(s)
}

// number makes an NSNumber of a double (+1).
func number(f float64) objc.ID {
	return objc.ID(objcx.MustClass("NSNumber")).Send(objcx.Sel("numberWithDouble:"), f)
}

// sync gives the element the properties of its node: the role, the label, the value, the state, the frame and the
// children.
func (b *Bridge) sync(ref NodeRef) {
	e, n := b.elems[ref], b.model.Node(ref)
	if e == nil || n == nil || e.obj == 0 {
		return
	}
	nsView, _, scale, ok := b.host.ViewObjects(ref.View)
	if !ok || scale <= 0 {
		return
	}
	obj := e.obj
	e.internal = true
	defer func() { e.internal = false }()
	sel := objcx.Sel

	setString(obj, "setAccessibilityRole:", Role(n))
	setString(obj, "setAccessibilitySubrole:", Subrole(n))
	setString(obj, "setAccessibilityLabel:", b.model.Label(n))
	setString(obj, "setAccessibilityHelp:", Help(n))
	setString(obj, "setAccessibilityIdentifier:", Identifier(n))
	obj.Send(sel("setAccessibilityEnabled:"), Enabled(n))
	obj.Send(sel("setAccessibilitySelected:"), Selected(n))
	obj.Send(sel("setAccessibilityElement:"), isElement(n))
	focused := false
	if f := b.model.Focused(ref.View); f != nil && f.ID == n.ID {
		focused = true
	}
	obj.Send(sel("setAccessibilityFocused:"), focused)
	if x := ExpandedState(n); x >= 0 {
		obj.Send(sel("setAccessibilityExpanded:"), x == 1)
	}
	b.syncValue(obj, n)

	// The frame is in the space of the parent: the view for the root, else the parent's element, with the origin at the
	// bottom left, in points.
	viewBounds := objc.Send[nsRect](nsView, sel("bounds"))
	viewH := viewBounds.H
	ext := b.model.Extents(n)
	topLeft := rect{ext.X / scale, ext.Y / scale, ext.Width / scale, ext.Height / scale}
	frame := rect{X: topLeft.X, Y: viewH - (topLeft.Y + topLeft.H), W: topLeft.W, H: topLeft.H}
	parent := b.model.ParentOf(n)
	if parent == nil {
		obj.Send(sel("setAccessibilityParent:"), nsView)
	} else if pe := b.elems[parent.Ref()]; pe != nil {
		pext := b.model.Extents(parent)
		pframe := rect{pext.X / scale, viewH - (pext.Y/scale + pext.Height/scale), pext.Width / scale, pext.Height / scale}
		frame = rect{X: frame.X - pframe.X, Y: frame.Y - pframe.Y, W: frame.W, H: frame.H}
		obj.Send(sel("setAccessibilityParent:"), pe.obj)
	}
	obj.Send(sel("setAccessibilityFrameInParentSpace:"), nsRect(frame))

	kids := b.model.Children(n)
	objs := make([]objc.ID, 0, len(kids))
	for _, k := range kids {
		if ke := b.elems[k]; ke != nil && ke.obj != 0 && !Hidden(b.model.Node(k)) {
			objs = append(objs, ke.obj)
		}
	}
	arr := objcx.NewArray(objs...)
	obj.Send(sel("setAccessibilityChildren:"), arr)
	objcx.Release(arr)
}

// isElement says whether VoiceOver stops on the node: a node with something to say or to do. A container with none of
// that is only a group that its children hang from (Chromium leaves out its ignored nodes the same way).
func isElement(n *Node) bool {
	if n.ID == RootID {
		return false
	}
	if n.Label != "" || n.Value != "" || n.Hint != "" || n.Tooltip != "" || n.Role != ui.SemanticsRoleNone {
		return true
	}
	if n.Flags != ui.SemanticsFlagsNone && n.Flags != (ui.SemanticsFlags{HasImplicitScrolling: true}) {
		return true
	}
	return n.Actions&^ui.SemanticsActionShowOnScreen != 0
}

// syncValue sets the value, the limits and the text properties of the element.
func (b *Bridge) syncValue(obj objc.ID, n *Node) {
	sel := objcx.Sel
	switch kind, text, num := Value(n); kind {
	case ValueString:
		s := str(text)
		obj.Send(sel("setAccessibilityValue:"), s)
		objcx.Release(s)
	case ValueNumber:
		v := number(num)
		obj.Send(sel("setAccessibilityValue:"), v)
	default:
		obj.Send(sel("setAccessibilityValue:"), objc.ID(0))
	}
	if r, ok := RangeOf(n); ok {
		obj.Send(sel("setAccessibilityMinValue:"), number(r.Min))
		obj.Send(sel("setAccessibilityMaxValue:"), number(r.Max))
	}
	if n.Flags.IsTextField {
		obj.Send(sel("setAccessibilityNumberOfCharacters:"), int64(NumberOfCharacters(n)))
		if loc, length, ok := SelectedRange(n); ok {
			obj.Send(sel("setAccessibilitySelectedTextRange:"), nsRange{Location: uint64(loc), Length: uint64(length)})
			text := n.Value
			from, to := ByteOffset(text, loc), ByteOffset(text, loc+length)
			setString(obj, "setAccessibilitySelectedText:", text[from:to])
		}
	}
}

// notify posts the notification that a change calls for.
func (b *Bridge) notify(c Change) {
	e := b.elems[c.Ref]
	switch c.Kind {
	case ChangeChildren:
		if e != nil {
			post(e.obj, noteLayoutChanged)
		}
	case ChangeWindowOpened:
		if e != nil {
			post(e.obj, noteLayoutChanged)
		}
	case ChangeUpdated:
		if e == nil {
			return
		}
		if c.Fields&(FieldValue|FieldExpanded) != 0 {
			post(e.obj, noteValueChanged)
		}
		if c.Fields&FieldLabel != 0 {
			post(e.obj, noteTitleChanged)
		}
		if c.Fields&FieldSelected != 0 {
			post(e.obj, noteSelectedChild)
		}
		if c.Fields&FieldFrame != 0 {
			post(e.obj, noteMoved)
			post(e.obj, noteResized)
		}
	case ChangeFocus:
		if e != nil {
			post(e.obj, noteFocusChanged)
		}
	case ChangeTextSelection:
		if e != nil {
			post(e.obj, noteSelectedText)
		}
	case ChangeAnnounce:
		if n := b.model.Node(c.Ref); n != nil {
			text := n.Label
			if text == "" {
				text = n.Value
			}
			if _, win, _, ok := b.host.ViewObjects(c.Ref.View); ok {
				announce(win, text)
			}
		}
	}
}

// The C functions of AppKit that post a notification.
var (
	postFuncs     sync.Once
	postNote      func(element, name uintptr)
	postNoteInfo  func(element, name, info uintptr)
	postFuncsErr  error
	postFuncsOpen = func() {
		lib, err := dl.OpenFramework("AppKit")
		if err != nil {
			postFuncsErr = err
			return
		}
		if err := lib.Bind(&postNote, "NSAccessibilityPostNotification"); err != nil {
			postFuncsErr = err
			return
		}
		postFuncsErr = lib.Bind(&postNoteInfo, "NSAccessibilityPostNotificationWithUserInfo")
	}
)

// post posts the notification for the element.
func post(obj objc.ID, name string) {
	postFuncs.Do(postFuncsOpen)
	if obj == 0 || postFuncsErr != nil {
		return
	}
	n := str(name)
	postNote(uintptr(obj), uintptr(n))
	objcx.Release(n)
}

// announce asks VoiceOver to say the text, at high priority.
func announce(window objc.ID, text string) {
	postFuncs.Do(postFuncsOpen)
	if window == 0 || text == "" || postFuncsErr != nil {
		return
	}
	n, k1, k2, t := str(noteAnnouncement), str(announcementKey), str(priorityKey), str(text)
	prio := number(priorityHigh)
	dict := objc.ID(objcx.MustClass("NSDictionary")).Send(objcx.Sel("dictionaryWithObjects:forKeys:count:"),
		&[]objc.ID{t, prio}[0], &[]objc.ID{k1, k2}[0], uint64(2))
	postNoteInfo(uintptr(window), uintptr(n), uintptr(dict))
	for _, o := range []objc.ID{n, k1, k2, t} {
		objcx.Release(o)
	}
}

// perform asks the framework for an action on a node.
func (b *Bridge) perform(ref NodeRef, action ui.SemanticsAction, args any) bool {
	if b.closed || b.acc == nil {
		return false
	}
	n := b.model.Node(ref)
	if n == nil || n.Actions&action == 0 {
		return false
	}
	b.acc.PerformAction(ui.SemanticsActionEvent{ViewID: ref.View, NodeID: ref.ID, Type: action, Arguments: args})
	return true
}
