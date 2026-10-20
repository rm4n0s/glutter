package widgets

import (
	"reflect"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// Widget describes the configuration for an Element. Widgets are immutable
// after construction; the framework builds, updates and compares them by
// pointer, so a widget type is always used as a pointer (the constructors
// return *X).
//
// A widget type of your own embeds one of the base structs (StatelessWidget,
// StatefulWidget, and the ones of the proxy and render object widgets). The
// base carries the key; the framework chooses the element from the interfaces
// the outer type implements (SPEC §12.5).
//
// Flutter: widgets/framework.dart Widget
type Widget interface {
	// Key controls how this widget replaces another widget in the tree.
	//
	// Flutter: widgets/framework.dart Widget.key
	Key() foundation.Key
	widgetBase() *WidgetBase
}

// WidgetOpts are the optional parameters shared by the constructors of the
// widget base structs (NewStatelessWidgetBase and the like).
//
// Flutter: widgets/framework.dart Widget.new
type WidgetOpts struct {
	// Key controls how this widget replaces another widget in the tree.
	Key foundation.Key
}

// WidgetBase holds what every widget has: its key. It is embedded in the base
// structs of the widget kinds.
//
// Flutter: widgets/framework.dart Widget
type WidgetBase struct {
	key foundation.Key
}

// NewWidgetBase creates a base with the given key.
//
// Flutter: widgets/framework.dart Widget.new
func NewWidgetBase(opts ...WidgetOpts) WidgetBase {
	return WidgetBase{key: firstOpts(opts).Key}
}

// Key is the key of the widget, or nil.
//
// Flutter: widgets/framework.dart Widget.key
func (w *WidgetBase) Key() foundation.Key { return w.key }

func (w *WidgetBase) widgetBase() *WidgetBase { return w }

// ElementCreator is a widget that makes its own element. It takes priority
// over the element kinds the framework chooses from the other interfaces of
// the widget.
//
// Flutter: widgets/framework.dart Widget.createElement
type ElementCreator interface {
	Widget
	// CreateElement inflates this configuration to a concrete instance.
	//
	// Flutter: widgets/framework.dart Widget.createElement
	CreateElement() Element
}

// StatelessWidget is a widget that does not require mutable state. Embed it
// in a widget type and implement Build:
//
//	type Greeting struct {
//		widgets.StatelessWidget
//		name string
//	}
//	func (g *Greeting) Build(bc widgets.BuildContext) widgets.Widget { ... }
//
// Flutter: widgets/framework.dart StatelessWidget
type StatelessWidget struct{ WidgetBase }

// NewStatelessWidgetBase creates the embedded base of a stateless widget.
//
// Flutter: widgets/framework.dart StatelessWidget.new
func NewStatelessWidgetBase(opts ...WidgetOpts) StatelessWidget {
	return StatelessWidget{NewWidgetBase(opts...)}
}

func (*StatelessWidget) isStateless() {}

// statelessWidget is what the framework looks for to choose a StatelessElement.
type statelessWidget interface {
	Widget
	// Build describes the part of the user interface represented by this
	// widget.
	//
	// Flutter: widgets/framework.dart StatelessWidget.build
	Build(bc BuildContext) Widget
	isStateless()
}

// StatefulWidget is a widget that has mutable state. Embed it in a widget type
// and implement CreateState:
//
//	type Counter struct {
//		widgets.StatefulWidget
//	}
//	func (c *Counter) CreateState() widgets.State { return &counterState{} }
//
// Flutter: widgets/framework.dart StatefulWidget
type StatefulWidget struct{ WidgetBase }

// NewStatefulWidgetBase creates the embedded base of a stateful widget.
//
// Flutter: widgets/framework.dart StatefulWidget.new
func NewStatefulWidgetBase(opts ...WidgetOpts) StatefulWidget {
	return StatefulWidget{NewWidgetBase(opts...)}
}

func (*StatefulWidget) isStateful() {}

// statefulWidget is what the framework looks for to choose a StatefulElement.
type statefulWidget interface {
	Widget
	// CreateState creates the mutable state for this widget at a given
	// location in the tree. It must return a new State every time.
	//
	// Flutter: widgets/framework.dart StatefulWidget.createState
	CreateState() State
	isStateful()
}

// CreateElement inflates a widget to an element: its CreateElement method if
// it has one, otherwise the element of the kind of its embedded base: an
// InheritedElement, a ParentDataElement, a LeafRenderObjectElement, a
// SingleChildRenderObjectElement, a MultiChildRenderObjectElement, a
// StatefulElement (CreateState) or a StatelessElement (Build). A widget of any
// other kind panics with the tag WidgetKindUnknown.
//
// Flutter: widgets/framework.dart Widget.createElement
func CreateElement(w Widget) Element {
	switch x := w.(type) {
	case ElementCreator:
		return x.CreateElement()
	case modelWidget:
		return NewInheritedModelElement(x)
	case notifierWidget:
		return NewInheritedNotifierElement(x)
	case InheritedWidgetInterface:
		return NewInheritedElement(x)
	case parentDataWidget:
		return NewParentDataElement(x)
	case leafRenderObjectWidget:
		return NewLeafRenderObjectElement(x)
	case SlottedRenderObjectWidget:
		return NewSlottedRenderObjectElement(x)
	case singleChildRenderObjectWidget:
		return NewSingleChildRenderObjectElement(x)
	case multiChildRenderObjectWidget:
		return NewMultiChildRenderObjectElement(x)
	case statefulWidget:
		return NewStatefulElement(x)
	case statelessWidget:
		return NewStatelessElement(x)
	}
	panic(errors.New("WidgetKindUnknown",
		"the framework cannot tell which element this widget needs; embed StatelessWidget and implement Build, embed StatefulWidget and implement CreateState, or implement CreateElement",
		"widget", foundation.RuntimeTypeName(w)))
}

// CanUpdate is whether the new widget can be used to update an element that
// has the old widget as its configuration: the two have the same type and
// equal keys.
//
// Flutter: widgets/framework.dart Widget.canUpdate
func CanUpdate(oldWidget, newWidget Widget) bool {
	return reflect.TypeOf(oldWidget) == reflect.TypeOf(newWidget) && oldWidget.Key() == newWidget.Key()
}

// WidgetToStringShort describes a widget in a few words: its type, followed by
// its key if it has one ("Foo-[<'k'>]"). Unlike other objects it has no hash.
//
// Deviation: it is a function, because the embedded base struct of a widget
// cannot see the outer type (SPEC §12.10).
//
// Flutter: widgets/framework.dart Widget.toStringShort
func WidgetToStringShort(w Widget) string {
	if s, ok := w.(foundation.ShortDescriber); ok {
		return s.ToStringShort()
	}
	short := foundation.RuntimeTypeName(w)
	if k := w.Key(); k != nil {
		return short + "-" + k.String()
	}
	return short
}

// DescribeWidget returns the diagnostics node of a widget: its type with the
// key, and the properties of its DebugFillProperties method when it has one.
//
// Flutter: widgets/framework.dart Widget.toDiagnosticsNode
func DescribeWidget(w Widget, opts ...foundation.DiagnosticsNodeOpts) foundation.DiagnosticsNode {
	return foundation.NewDiagnosticableNode(&widgetDescription{w}, opts...)
}

// widgetDescription makes any widget a Diagnosticable that prints like Flutter's
// Widget.
type widgetDescription struct{ w Widget }

func (d *widgetDescription) ToStringShort() string { return WidgetToStringShort(d.w) }

func (d *widgetDescription) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	if dd, ok := d.w.(foundation.Diagnosticable); ok {
		dd.DebugFillProperties(p)
	}
}
