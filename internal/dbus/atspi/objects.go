package atspi

import (
	"strconv"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/glutter/ui"
)

// The names of the AT-SPI2 interfaces and paths.
const (
	ifaceAccessible  = "org.a11y.atspi.Accessible"
	ifaceApplication = "org.a11y.atspi.Application"
	ifaceComponent   = "org.a11y.atspi.Component"
	ifaceAction      = "org.a11y.atspi.Action"
	ifaceProperties  = "org.freedesktop.DBus.Properties"

	pathPrefix = "/org/a11y/atspi/accessible/"
	rootPath   = godbus.ObjectPath(pathPrefix + "root")

	toolkitName  = "Glutter"
	atspiVersion = "2.1"
)

// nodePath is the object path of a node, or of the application root.
func nodePath(ref NodeRef) godbus.ObjectPath {
	if ref.IsApp() {
		return rootPath
	}
	return godbus.ObjectPath(pathPrefix + strconv.FormatInt(int64(ref.View), 10) + "_" + strconv.Itoa(ref.ID))
}

// objRef is the type (so) of a reference to an accessible object: the bus name of the
// app and the path.
type objRef struct {
	Name string
	Path godbus.ObjectPath
}

// relation is the type (ua(so)) of an entry of the relation set.
type relation struct {
	Type    uint32
	Targets []objRef
}

// actionDesc is the type (sss) of an action of GetActions.
type actionDesc struct {
	Name, Description, KeyBinding string
}

// rect is the type (iiii) of the extents of a component.
type rect struct{ X, Y, Width, Height int32 }

func invalidArgs(msg string) *godbus.Error {
	return &godbus.Error{Name: "org.freedesktop.DBus.Error.InvalidArgs", Body: []any{msg}}
}

func unknownObject(ref NodeRef) *godbus.Error {
	return &godbus.Error{Name: "org.freedesktop.DBus.Error.UnknownObject",
		Body: []any{"no accessible object " + string(nodePath(ref))}}
}

// actionInfo is an action of a node as the Action interface shows it.
type actionInfo struct {
	name, description string
	typ               ui.SemanticsAction
	custom            int
}

// actionTable lists the actions of a node that the Action interface offers, in the
// order they are numbered; the name is what a screen reader uses ("click" for the
// primary action). The custom actions of the node follow. The focus and show-on-screen
// actions are not here: they are the methods GrabFocus and ScrollTo of Component.
var actionTable = []actionInfo{
	{"click", "Clicks the item", ui.SemanticsActionTap, 0},
	{"longpress", "Presses the item for a long time", ui.SemanticsActionLongPress, 0},
	{"increase", "Increases the value", ui.SemanticsActionIncrease, 0},
	{"decrease", "Decreases the value", ui.SemanticsActionDecrease, 0},
	{"dismiss", "Dismisses the item", ui.SemanticsActionDismiss, 0},
	{"expand", "Expands the item", ui.SemanticsActionExpand, 0},
	{"collapse", "Collapses the item", ui.SemanticsActionCollapse, 0},
	{"copy", "Copies the selection", ui.SemanticsActionCopy, 0},
	{"cut", "Cuts the selection", ui.SemanticsActionCut, 0},
	{"paste", "Pastes the clipboard", ui.SemanticsActionPaste, 0},
	{"scroll up", "Scrolls the content up", ui.SemanticsActionScrollUp, 0},
	{"scroll down", "Scrolls the content down", ui.SemanticsActionScrollDown, 0},
	{"scroll left", "Scrolls the content left", ui.SemanticsActionScrollLeft, 0},
	{"scroll right", "Scrolls the content right", ui.SemanticsActionScrollRight, 0},
}

// actionsOf are the actions of a node.
func (b *Bridge) actionsOf(n *Node) []actionInfo {
	var out []actionInfo
	for _, a := range actionTable {
		if n.Actions&a.typ != 0 {
			out = append(out, a)
		}
	}
	if n.Actions&ui.SemanticsActionCustomAction != 0 {
		vt := b.model.views[n.View]
		for _, id := range n.AdditionalActions {
			label := strconv.Itoa(id)
			if ca, ok := vt.customActions[id]; ok && ca.Label != "" {
				label = ca.Label
			}
			out = append(out, actionInfo{name: label, description: label, typ: ui.SemanticsActionCustomAction, custom: id})
		}
	}
	return out
}

// interfacesOf are the AT-SPI2 interfaces that the object of the node implements.
func (b *Bridge) interfacesOf(ref NodeRef) []string {
	if ref.IsApp() {
		return []string{ifaceAccessible, ifaceApplication}
	}
	n := b.model.Node(ref)
	out := []string{ifaceAccessible, ifaceComponent}
	if n != nil && len(b.actionsOf(n)) > 0 {
		out = append(out, ifaceAction)
	}
	return out
}

// object is what is exported for a node (and for the application root) under each of its
// interfaces. godbus serves every method of the receiver under every interface it is
// exported as; the names of the methods of the interfaces do not clash.
type object struct {
	b   *Bridge
	ref NodeRef
}

// node returns the node of the object, or the error to answer with.
func (o *object) node() (*Node, *godbus.Error) {
	n := o.b.model.Node(o.ref)
	if n == nil {
		return nil, unknownObject(o.ref)
	}
	return n, nil
}

func (o *object) objRef(ref NodeRef) objRef {
	return objRef{Name: o.b.busName(), Path: nodePath(ref)}
}

// parentRef is the parent of the object: the frame's parent is the application and
// the application's is what the registry said.
func (o *object) parentRef() objRef {
	if o.ref.IsApp() {
		return o.b.registryRoot
	}
	n := o.b.model.Node(o.ref)
	switch {
	case n == nil:
		return objRef{Name: o.b.busName(), Path: "/org/a11y/atspi/null"}
	case n.Parent < 0:
		return o.objRef(appRef)
	}
	return o.objRef(NodeRef{o.ref.View, n.Parent})
}

func (o *object) children() []NodeRef {
	b := o.b
	if o.ref.IsApp() {
		var out []NodeRef
		for _, v := range b.model.Views() {
			out = append(out, NodeRef{v, rootID})
		}
		return out
	}
	if n := b.model.Node(o.ref); n != nil {
		return b.model.Children(n)
	}
	return nil
}

// GetChildAtIndex is Accessible.GetChildAtIndex.
func (o *object) GetChildAtIndex(index int32) (objRef, *godbus.Error) {
	o.b.mu.Lock()
	defer o.b.mu.Unlock()
	kids := o.children()
	if index < 0 || int(index) >= len(kids) {
		return objRef{}, invalidArgs("no child " + strconv.Itoa(int(index)))
	}
	return o.objRef(kids[index]), nil
}

// GetChildren is Accessible.GetChildren.
func (o *object) GetChildren() ([]objRef, *godbus.Error) {
	o.b.mu.Lock()
	defer o.b.mu.Unlock()
	out := []objRef{}
	for _, k := range o.children() {
		out = append(out, o.objRef(k))
	}
	return out, nil
}

// GetIndexInParent is Accessible.GetIndexInParent.
func (o *object) GetIndexInParent() (int32, *godbus.Error) {
	b := o.b
	b.mu.Lock()
	defer b.mu.Unlock()
	if o.ref.IsApp() {
		return -1, nil
	}
	n, err := o.node()
	if err != nil {
		return -1, err
	}
	if n.Parent < 0 {
		return int32(b.model.frameIndex(o.ref.View)), nil
	}
	for i, c := range b.model.Children(b.model.Node(NodeRef{o.ref.View, n.Parent})) {
		if c == o.ref {
			return int32(i), nil
		}
	}
	return -1, nil
}

// GetRelationSet is Accessible.GetRelationSet: no relations.
func (o *object) GetRelationSet() ([]relation, *godbus.Error) { return []relation{}, nil }

// GetRole is Accessible.GetRole.
func (o *object) GetRole() (uint32, *godbus.Error) {
	b := o.b
	b.mu.Lock()
	defer b.mu.Unlock()
	return uint32(b.roleOfRef(o.ref)), nil
}

func (b *Bridge) roleOfRef(ref NodeRef) Role {
	if ref.IsApp() {
		return RoleApplication
	}
	if n := b.model.Node(ref); n != nil {
		return b.model.Role(n)
	}
	return RoleInvalid
}

// GetRoleName is Accessible.GetRoleName.
func (o *object) GetRoleName() (string, *godbus.Error) {
	o.b.mu.Lock()
	defer o.b.mu.Unlock()
	return o.b.roleOfRef(o.ref).Name(), nil
}

// GetLocalizedRoleName is Accessible.GetLocalizedRoleName: the English name.
func (o *object) GetLocalizedRoleName() (string, *godbus.Error) { return o.GetRoleName() }

// GetState is Accessible.GetState.
func (o *object) GetState() ([]uint32, *godbus.Error) {
	b := o.b
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stateOfRef(o.ref).Words(), nil
}

func (b *Bridge) stateOfRef(ref NodeRef) StateSet {
	if ref.IsApp() {
		return StateSet(0).With(StateEnabled).With(StateSensitive).With(StateShowing).With(StateVisible)
	}
	if n := b.model.Node(ref); n != nil {
		return b.model.States(n)
	}
	return 0
}

// GetAttributes is Accessible.GetAttributes.
func (o *object) GetAttributes() (map[string]string, *godbus.Error) {
	b := o.b
	b.mu.Lock()
	defer b.mu.Unlock()
	attrs := map[string]string{"toolkit": toolkitName}
	if o.ref.IsApp() {
		return attrs, nil
	}
	n, err := o.node()
	if err != nil {
		return nil, err
	}
	if n.Identifier != "" {
		attrs["id"] = n.Identifier
	}
	if n.HeadingLevel > 0 {
		attrs["level"] = strconv.Itoa(n.HeadingLevel)
	}
	if name, ok := landmarkNames[n.Role]; ok {
		attrs["xml-roles"] = name
	}
	if n.Flags.IsLiveRegion {
		attrs["live"] = "polite"
	}
	return attrs, nil
}

// GetApplication is Accessible.GetApplication.
func (o *object) GetApplication() (objRef, *godbus.Error) {
	return o.objRef(appRef), nil
}

// GetInterfaces is Accessible.GetInterfaces.
func (o *object) GetInterfaces() ([]string, *godbus.Error) {
	o.b.mu.Lock()
	defer o.b.mu.Unlock()
	return o.b.interfacesOf(o.ref), nil
}

// GetLocale is Application.GetLocale: the locale of the app, from the environment.
func (o *object) GetLocale(lctype uint32) (string, *godbus.Error) { return o.b.locale(), nil }

// --- Component ---

// extentsOf is the rectangle of the node in the pixels of its window, moved by the
// origin of the window when the coordinates are those of the screen (coordType 0).
func (o *object) extentsOf(n *Node, coordType uint32) Extents {
	e := o.b.model.Extents(n)
	if coordType == 0 && o.b.opts.WindowOrigin != nil {
		x, y := o.b.opts.WindowOrigin(n.View)
		e.X, e.Y = e.X+int32(x), e.Y+int32(y)
	}
	return e
}

// Contains is Component.Contains.
func (o *object) Contains(x, y int32, coordType uint32) (bool, *godbus.Error) {
	o.b.mu.Lock()
	defer o.b.mu.Unlock()
	n, err := o.node()
	if err != nil {
		return false, err
	}
	return o.extentsOf(n, coordType).Contains(x, y), nil
}

// GetAccessibleAtPoint is Component.GetAccessibleAtPoint: the deepest node of the
// object that is at the point, or the null reference.
func (o *object) GetAccessibleAtPoint(x, y int32, coordType uint32) (objRef, *godbus.Error) {
	b := o.b
	b.mu.Lock()
	defer b.mu.Unlock()
	n, err := o.node()
	if err != nil {
		return objRef{}, err
	}
	if coordType == 0 && b.opts.WindowOrigin != nil {
		ox, oy := b.opts.WindowOrigin(n.View)
		x, y = x-int32(ox), y-int32(oy)
	}
	if hit := b.model.HitTest(n, x, y); hit != nil {
		return o.objRef(hit.Ref()), nil
	}
	return objRef{Name: "", Path: "/org/a11y/atspi/null"}, nil
}

// GetExtents is Component.GetExtents.
func (o *object) GetExtents(coordType uint32) (rect, *godbus.Error) {
	o.b.mu.Lock()
	defer o.b.mu.Unlock()
	n, err := o.node()
	if err != nil {
		return rect{}, err
	}
	e := o.extentsOf(n, coordType)
	return rect(e), nil
}

// GetPosition is Component.GetPosition.
func (o *object) GetPosition(coordType uint32) (int32, int32, *godbus.Error) {
	o.b.mu.Lock()
	defer o.b.mu.Unlock()
	n, err := o.node()
	if err != nil {
		return 0, 0, err
	}
	e := o.extentsOf(n, coordType)
	return e.X, e.Y, nil
}

// GetSize is Component.GetSize.
func (o *object) GetSize() (int32, int32, *godbus.Error) {
	o.b.mu.Lock()
	defer o.b.mu.Unlock()
	n, err := o.node()
	if err != nil {
		return 0, 0, err
	}
	e := o.b.model.Extents(n)
	return e.Width, e.Height, nil
}

// GetLayer is Component.GetLayer: the widget layer (ATSPI_LAYER_WIDGET).
func (o *object) GetLayer() (uint32, *godbus.Error) { return 3, nil }

// GetMDIZOrder is Component.GetMDIZOrder.
func (o *object) GetMDIZOrder() (int16, *godbus.Error) { return 0, nil }

// GetAlpha is Component.GetAlpha.
func (o *object) GetAlpha() (float64, *godbus.Error) { return 1, nil }

// GrabFocus is Component.GrabFocus: it asks the framework to focus the node, which
// works for the nodes that can take the focus.
func (o *object) GrabFocus() (bool, *godbus.Error) {
	o.b.mu.Lock()
	defer o.b.mu.Unlock()
	n, err := o.node()
	if err != nil {
		return false, err
	}
	if n.Actions&ui.SemanticsActionFocus == 0 {
		return false, nil
	}
	o.b.perform(ui.SemanticsActionEvent{ViewID: n.View, NodeID: n.ID, Type: ui.SemanticsActionFocus})
	return true, nil
}

// ScrollTo is Component.ScrollTo: it asks the framework to show the node on the screen.
func (o *object) ScrollTo(scrollType uint32) (bool, *godbus.Error) {
	o.b.mu.Lock()
	defer o.b.mu.Unlock()
	n, err := o.node()
	if err != nil {
		return false, err
	}
	if n.Actions&ui.SemanticsActionShowOnScreen == 0 {
		return false, nil
	}
	o.b.perform(ui.SemanticsActionEvent{ViewID: n.View, NodeID: n.ID, Type: ui.SemanticsActionShowOnScreen})
	return true, nil
}

// SetExtents, SetPosition, SetSize and ScrollToPoint are Component methods that an
// app's accessibility tree does not support: the answer is false.
func (o *object) SetExtents(x, y, w, h int32, coordType uint32) (bool, *godbus.Error) {
	return false, nil
}
func (o *object) SetPosition(x, y int32, coordType uint32) (bool, *godbus.Error) { return false, nil }
func (o *object) SetSize(w, h int32) (bool, *godbus.Error)                       { return false, nil }
func (o *object) ScrollToPoint(scrollType uint32, x, y int32) (bool, *godbus.Error) {
	return false, nil
}

// --- Action ---

// GetActions is Action.GetActions.
func (o *object) GetActions() ([]actionDesc, *godbus.Error) {
	b := o.b
	b.mu.Lock()
	defer b.mu.Unlock()
	n, err := o.node()
	if err != nil {
		return nil, err
	}
	out := []actionDesc{}
	for _, a := range b.actionsOf(n) {
		out = append(out, actionDesc{Name: a.name, Description: a.description})
	}
	return out, nil
}

func (o *object) actionAt(index int32) (*Node, actionInfo, *godbus.Error) {
	n, err := o.node()
	if err != nil {
		return nil, actionInfo{}, err
	}
	acts := o.b.actionsOf(n)
	if index < 0 || int(index) >= len(acts) {
		return nil, actionInfo{}, invalidArgs("no action " + strconv.Itoa(int(index)))
	}
	return n, acts[index], nil
}

// GetName is Action.GetName.
func (o *object) GetName(index int32) (string, *godbus.Error) {
	o.b.mu.Lock()
	defer o.b.mu.Unlock()
	_, a, err := o.actionAt(index)
	return a.name, err
}

// GetLocalizedName is Action.GetLocalizedName.
func (o *object) GetLocalizedName(index int32) (string, *godbus.Error) { return o.GetName(index) }

// GetDescription is Action.GetDescription.
func (o *object) GetDescription(index int32) (string, *godbus.Error) {
	o.b.mu.Lock()
	defer o.b.mu.Unlock()
	_, a, err := o.actionAt(index)
	return a.description, err
}

// GetKeyBinding is Action.GetKeyBinding: none.
func (o *object) GetKeyBinding(index int32) (string, *godbus.Error) {
	o.b.mu.Lock()
	defer o.b.mu.Unlock()
	_, _, err := o.actionAt(index)
	return "", err
}

// DoAction is Action.DoAction: it asks the framework to perform the action, on its UI thread.
func (o *object) DoAction(index int32) (bool, *godbus.Error) {
	b := o.b
	b.mu.Lock()
	defer b.mu.Unlock()
	n, a, err := o.actionAt(index)
	if err != nil {
		return false, err
	}
	ev := ui.SemanticsActionEvent{ViewID: n.View, NodeID: n.ID, Type: a.typ}
	if a.typ == ui.SemanticsActionCustomAction {
		ev.Arguments = a.custom
	}
	b.perform(ev)
	return true, nil
}

// --- Properties ---

// propertiesObject serves org.freedesktop.DBus.Properties of an object from the model, so
// that the values are never stale.
type propertiesObject struct{ o *object }

func (p propertiesObject) values(iface string) (map[string]godbus.Variant, *godbus.Error) {
	o := p.o
	b := o.b
	b.mu.Lock()
	defer b.mu.Unlock()
	v := godbus.MakeVariant
	switch iface {
	case ifaceAccessible:
		props := map[string]godbus.Variant{
			"Parent": v(o.parentRef()), "ChildCount": v(int32(len(o.children()))),
			"Locale": v(b.locale()), "AccessibleId": v(""), "HelpText": v(""),
		}
		if o.ref.IsApp() {
			props["Name"], props["Description"] = v(b.opts.AppName), v("")
			return props, nil
		}
		n, err := o.node()
		if err != nil {
			return nil, err
		}
		props["Name"], props["Description"] = v(b.model.Name(n)), v(n.Hint)
		props["AccessibleId"], props["HelpText"] = v(n.Identifier), v(n.Tooltip)
		return props, nil
	case ifaceApplication:
		if !o.ref.IsApp() {
			return map[string]godbus.Variant{}, nil
		}
		return map[string]godbus.Variant{
			"ToolkitName": v(toolkitName), "Version": v(b.opts.Version), "AtspiVersion": v(atspiVersion), "Id": v(b.appID),
		}, nil
	case ifaceAction:
		n, err := o.node()
		if err != nil {
			return nil, err
		}
		return map[string]godbus.Variant{"NActions": v(int32(len(b.actionsOf(n))))}, nil
	}
	return map[string]godbus.Variant{}, nil
}

// Get is Properties.Get.
func (p propertiesObject) Get(iface, name string) (godbus.Variant, *godbus.Error) {
	props, err := p.values(iface)
	if err != nil {
		return godbus.Variant{}, err
	}
	val, ok := props[name]
	if !ok {
		return godbus.Variant{}, &godbus.Error{Name: "org.freedesktop.DBus.Error.UnknownProperty", Body: []any{name + " of " + iface}}
	}
	return val, nil
}

// GetAll is Properties.GetAll.
func (p propertiesObject) GetAll(iface string) (map[string]godbus.Variant, *godbus.Error) {
	return p.values(iface)
}

// Set is Properties.Set: only the Id of the application can be set, by the registry.
func (p propertiesObject) Set(iface, name string, value godbus.Variant) *godbus.Error {
	if iface == ifaceApplication && name == "Id" && p.o.ref.IsApp() {
		if id, ok := value.Value().(int32); ok {
			p.o.b.mu.Lock()
			p.o.b.appID = id
			p.o.b.mu.Unlock()
			return nil
		}
		return invalidArgs("Id is an int32")
	}
	return &godbus.Error{Name: "org.freedesktop.DBus.Error.PropertyReadOnly", Body: []any{name + " of " + iface + " is read-only"}}
}
