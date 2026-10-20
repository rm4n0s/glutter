package desktop

import (
	"slices"
	"strings"

	godbus "github.com/godbus/dbus/v5"
)

// TrayMenuEntry is an entry of the menu of a system tray icon: a TrayMenuItem or a
// separator.
//
// Flutter: no counterpart
type TrayMenuEntry interface {
	trayMenuEntry()
}

// TrayMenuItemOpts are the optional parameters of NewTrayMenuItem.
//
// Flutter: no counterpart
type TrayMenuItemOpts struct {
	// Disabled greys the item out and makes it do nothing.
	Disabled bool
	// Hidden leaves the item out of the menu.
	Hidden bool
	// Checkable gives the item a check mark; Checked says whether it is set. The
	// item does not change by itself when the user chooses it: the app rebuilds the
	// tray with the new value.
	Checkable bool
	// Radio makes a checkable item a radio button instead of a check box.
	Radio bool
	// Checked is the state of a checkable item.
	Checked bool
	// Icon is the name of an icon of the icon theme.
	Icon string
}

// TrayMenuItem is an item of the menu of a system tray icon: a label that the user
// chooses to call a function, or, with children, a submenu.
//
// Flutter: no counterpart
type TrayMenuItem struct {
	label     string
	onPressed func()
	children  []TrayMenuEntry
	o         TrayMenuItemOpts
}

func (*TrayMenuItem) trayMenuEntry() {}

// NewTrayMenuItem creates an item of the menu of a system tray. onPressed is called on
// the UI thread when the user chooses the item; it may be nil. A single underscore in
// the label is kept as it is (the menu protocol's mnemonic is not used).
//
// Flutter: no counterpart
func NewTrayMenuItem(label string, onPressed func(), opts ...TrayMenuItemOpts) *TrayMenuItem {
	return &TrayMenuItem{label: label, onPressed: onPressed, o: firstOpts(opts)}
}

// NewTrayMenuSubmenu creates an item that opens a submenu of the entries.
//
// Flutter: no counterpart
func NewTrayMenuSubmenu(label string, children []TrayMenuEntry, opts ...TrayMenuItemOpts) *TrayMenuItem {
	return &TrayMenuItem{label: label, children: children, o: firstOpts(opts)}
}

// Label is the text of the item.
//
// Flutter: no counterpart
func (i *TrayMenuItem) Label() string { return i.label }

// Children are the entries of the submenu, none for an item that is not one.
//
// Flutter: no counterpart
func (i *TrayMenuItem) Children() []TrayMenuEntry { return i.children }

// TrayMenuSeparator is a line between two groups of items.
//
// Flutter: no counterpart
type TrayMenuSeparator struct{}

func (*TrayMenuSeparator) trayMenuEntry() {}

// NewTrayMenuSeparator creates a separator.
//
// Flutter: no counterpart
func NewTrayMenuSeparator() *TrayMenuSeparator { return &TrayMenuSeparator{} }

// menuNode is an item of the menu in the shape of com.canonical.dbusmenu: its ID,
// its properties and its children. The root has the ID 0.
type menuNode struct {
	ID       int32
	Props    map[string]godbus.Variant
	Children []*menuNode
}

// menuTree is the menu as it was built from the entries.
type menuTree struct {
	root *menuNode
	// nodes find a node by ID, and pressed the function of an item.
	nodes   map[int32]*menuNode
	pressed map[int32]func()
}

// buildMenu numbers the entries depth first from 1, as the menu protocol needs, and
// makes their properties. The labels are escaped: the protocol takes a single
// underscore as the mark of a mnemonic.
func buildMenu(entries []TrayMenuEntry) *menuTree {
	t := &menuTree{nodes: map[int32]*menuNode{}, pressed: map[int32]func(){}}
	t.root = &menuNode{Props: map[string]godbus.Variant{"children-display": godbus.MakeVariant("submenu")}}
	t.nodes[0] = t.root
	next := int32(1)
	var add func(parent *menuNode, entries []TrayMenuEntry)
	add = func(parent *menuNode, entries []TrayMenuEntry) {
		for _, e := range entries {
			n := &menuNode{ID: next, Props: map[string]godbus.Variant{}}
			next++
			switch e := e.(type) {
			case *TrayMenuSeparator:
				n.Props["type"] = godbus.MakeVariant("separator")
			case *TrayMenuItem:
				if e == nil {
					continue
				}
				n.Props["label"] = godbus.MakeVariant(strings.ReplaceAll(e.label, "_", "__"))
				n.Props["enabled"] = godbus.MakeVariant(!e.o.Disabled)
				if e.o.Hidden {
					n.Props["visible"] = godbus.MakeVariant(false)
				}
				if e.o.Icon != "" {
					n.Props["icon-name"] = godbus.MakeVariant(e.o.Icon)
				}
				if e.o.Checkable || e.o.Radio {
					toggle := "checkmark"
					if e.o.Radio {
						toggle = "radio"
					}
					state := int32(0)
					if e.o.Checked {
						state = 1
					}
					n.Props["toggle-type"] = godbus.MakeVariant(toggle)
					n.Props["toggle-state"] = godbus.MakeVariant(state)
				}
				if e.onPressed != nil {
					t.pressed[n.ID] = e.onPressed
				}
				if len(e.children) > 0 {
					n.Props["children-display"] = godbus.MakeVariant("submenu")
				}
			default:
				continue
			}
			t.nodes[n.ID] = n
			parent.Children = append(parent.Children, n)
			if item, ok := e.(*TrayMenuItem); ok && item != nil {
				add(n, item.children)
			}
		}
	}
	add(t.root, entries)
	return t
}

// menuLayout is the type (ia{sv}av) of GetLayout: the ID, the properties and the
// children, each of them a variant that holds a menuLayout.
type menuLayout struct {
	ID       int32
	Props    map[string]godbus.Variant
	Children []godbus.Variant
}

// layout makes the layout of the node down to depth levels of children (all of them
// for a negative depth), with the properties named, or all of them for no names.
func (n *menuNode) layout(depth int32, names []string) menuLayout {
	l := menuLayout{ID: n.ID, Props: filterProps(n.Props, names), Children: []godbus.Variant{}}
	if depth == 0 {
		return l
	}
	for _, c := range n.Children {
		l.Children = append(l.Children, godbus.MakeVariant(c.layout(depth-1, names)))
	}
	return l
}

func filterProps(props map[string]godbus.Variant, names []string) map[string]godbus.Variant {
	out := make(map[string]godbus.Variant, len(props))
	for k, v := range props {
		if len(names) == 0 || slices.Contains(names, k) {
			out[k] = v
		}
	}
	return out
}
