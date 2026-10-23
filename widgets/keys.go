package widgets

import (
	"reflect"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// AnyGlobalKey is a key that is unique across the whole app: the GlobalKey and
// GlobalObjectKey types. A widget with a global key keeps its element, and its
// state, when it moves to another place in the tree.
//
// Flutter: widgets/framework.dart GlobalKey
type AnyGlobalKey interface {
	foundation.Key
	// CurrentElement is the element of the widget that has this key, or nil.
	CurrentElement() Element
	globalKeyMarker()
	// bind tells the key which owner registered it.
	bind(owner *BuildOwner)
	currentElementIn(owner *BuildOwner) Element
}

// GlobalKey is a key that is unique across the app, and from which the element,
// the widget and the state of the widget that has it can be reached. S is the
// type of the state of the widget, or State when it does not matter.
//
// Deviation: the key remembers the BuildOwner that registered it and looks the
// element up there, where Flutter reaches the owner through
// WidgetsBinding.instance (SPEC §12.10).
//
// Flutter: widgets/framework.dart GlobalKey, LabeledGlobalKey
type GlobalKey[S State] struct {
	foundation.KeyBase
	debugLabel string
	owner      *BuildOwner
}

// GlobalKeyOpts are the optional parameters of NewGlobalKey.
//
// Flutter: widgets/framework.dart GlobalKey.new
type GlobalKeyOpts struct {
	// DebugLabel is a label for debug output.
	DebugLabel string
}

// NewGlobalKey creates a key that is equal only to itself.
//
// Flutter: widgets/framework.dart GlobalKey.new
func NewGlobalKey[S State](opts ...GlobalKeyOpts) *GlobalKey[S] {
	return &GlobalKey[S]{debugLabel: firstOpts(opts).DebugLabel}
}

func (k *GlobalKey[S]) globalKeyMarker()       {}
func (k *GlobalKey[S]) bind(owner *BuildOwner) { k.owner = owner }

func (k *GlobalKey[S]) currentElementIn(owner *BuildOwner) Element {
	if owner == nil {
		return nil
	}
	return owner.globalKeyRegistry[k]
}

// CurrentElement is the element of the widget that has this key, or nil.
//
// Flutter: widgets/framework.dart GlobalKey._currentElement
func (k *GlobalKey[S]) CurrentElement() Element { return k.currentElementIn(k.owner) }

// CurrentContext is the build context of the widget that has this key, or nil
// when no widget in the tree has it.
//
// Flutter: widgets/framework.dart GlobalKey.currentContext
func (k *GlobalKey[S]) CurrentContext() BuildContext {
	if e := k.CurrentElement(); e != nil {
		return e
	}
	return nil
}

// CurrentWidget is the widget that has this key, or nil.
//
// Flutter: widgets/framework.dart GlobalKey.currentWidget
func (k *GlobalKey[S]) CurrentWidget() Widget {
	if e := k.CurrentElement(); e != nil {
		return e.AsElement().widget
	}
	return nil
}

// CurrentState is the state of the stateful widget that has this key. It
// returns false when no such widget is in the tree or its state is not an S.
//
// Flutter: widgets/framework.dart GlobalKey.currentState
func (k *GlobalKey[S]) CurrentState() (S, bool) {
	return currentStateOf[S](k.CurrentElement())
}

func currentStateOf[S State](e Element) (S, bool) {
	var zero S
	se, ok := e.(*StatefulElement)
	if !ok || se.state == nil {
		return zero, false
	}
	s, ok := se.state.(S)
	return s, ok
}

// String describes the key, such as "[GlobalKey#1a2b3 label]".
//
// Flutter: widgets/framework.dart LabeledGlobalKey.toString
func (k *GlobalKey[S]) String() string {
	label := ""
	if k.debugLabel != "" {
		label = " " + k.debugLabel
	}
	return "[GlobalKey#" + foundation.ShortHash(k) + label + "]"
}

// GlobalObjectKey is a global key that takes its identity from an object: two
// GlobalObjectKeys are equal when they hold the same object (the same pointer,
// or equal values of a type that is not a pointer).
//
// Deviation: it has no CurrentState; look the element up with
// CurrentElementIn, because the registry is per BuildOwner and a key that is a
// plain value cannot remember its owner (SPEC §12.10).
//
// Flutter: widgets/framework.dart GlobalObjectKey
type GlobalObjectKey[S State] struct {
	foundation.KeyBase
	value any
}

// NewGlobalObjectKey creates a key from an object. The object's type must be
// comparable, or it panics with the tag GlobalObjectKeyNotComparable.
//
// Flutter: widgets/framework.dart GlobalObjectKey.new
func NewGlobalObjectKey[S State](value any) GlobalObjectKey[S] {
	if value != nil && !reflect.TypeOf(value).Comparable() {
		panic(errors.New(tagGlobalObjectKeyNotComparable, "the value of a GlobalObjectKey must be comparable",
			"type", foundation.RuntimeTypeName(value)))
	}
	return GlobalObjectKey[S]{value: value}
}

func (k GlobalObjectKey[S]) globalKeyMarker() {}
func (k GlobalObjectKey[S]) bind(*BuildOwner) {}

// Value is the object the key was made from.
//
// Flutter: widgets/framework.dart GlobalObjectKey.value
func (k GlobalObjectKey[S]) Value() any { return k.value }

func (k GlobalObjectKey[S]) currentElementIn(owner *BuildOwner) Element {
	if owner == nil {
		return nil
	}
	return owner.globalKeyRegistry[k]
}

// CurrentElement is always nil: a GlobalObjectKey does not know its owner. Use
// CurrentElementIn.
//
// Flutter: widgets/framework.dart GlobalKey._currentElement
func (k GlobalObjectKey[S]) CurrentElement() Element { return nil }

// CurrentElementIn is the element of the widget that has this key in the tree
// of the owner, or nil.
//
// Flutter: widgets/framework.dart GlobalKey._currentElement
func (k GlobalObjectKey[S]) CurrentElementIn(owner *BuildOwner) Element {
	return k.currentElementIn(owner)
}

// CurrentStateIn is the state of the stateful widget that has this key in the
// tree of the owner.
//
// Flutter: widgets/framework.dart GlobalKey.currentState
func (k GlobalObjectKey[S]) CurrentStateIn(owner *BuildOwner) (S, bool) {
	return currentStateOf[S](k.currentElementIn(owner))
}

// String describes the key, such as "[GlobalObjectKey<FooState> int#00003]".
//
// Flutter: widgets/framework.dart GlobalObjectKey.toString
func (k GlobalObjectKey[S]) String() string {
	name := "GlobalObjectKey"
	if t := reflect.TypeFor[S](); t != reflect.TypeFor[State]() {
		name += "<" + foundation.RuntimeTypeName(reflect.Zero(t).Interface()) + ">"
	}
	return "[" + name + " " + foundation.DescribeIdentity(k.value) + "]"
}

// typeKey is a string that is the same for widgets of the same Go type.
func typeKey(w Widget) string { return reflect.TypeOf(w).String() }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagGlobalObjectKeyNotComparable = "GlobalObjectKeyNotComparable"
)
