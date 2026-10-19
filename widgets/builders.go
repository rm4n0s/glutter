package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
)

// WidgetBuilder is a function that builds a widget.
//
// Flutter: widgets/framework.dart WidgetBuilder
type WidgetBuilder func(bc BuildContext) Widget

// StateSetter is the function that StatefulBuilder gives its builder to rebuild
// the subtree: the function it is given runs, and then the builder runs again.
//
// Flutter: widgets/framework.dart StateSetter
type StateSetter func(fn func())

// Builder is a stateless utility widget whose build method uses its builder
// callback to create the widget's child. It gives the callback a context that
// is below the widget, so that lookups such as ThemeOf find what the widget
// above it provides.
//
// Flutter: widgets/basic.dart Builder
type Builder struct {
	StatelessWidget
	builder WidgetBuilder
}

// NewBuilder creates a widget that calls the builder to build its child.
//
// Flutter: widgets/basic.dart Builder.new
func NewBuilder(builder WidgetBuilder, opts ...WidgetOpts) *Builder {
	return &Builder{NewStatelessWidgetBase(firstOpts(opts)), builder}
}

// Builder is the function that builds the child.
//
// Flutter: widgets/basic.dart Builder.builder
func (w *Builder) Builder() WidgetBuilder { return w.builder }

// Build calls the builder.
//
// Flutter: widgets/basic.dart Builder.build
func (w *Builder) Build(bc BuildContext) Widget { return w.builder(bc) }

// StatefulBuilderFunc is the builder of a StatefulBuilder.
//
// Flutter: widgets/basic.dart StatefulWidgetBuilder
type StatefulBuilderFunc func(bc BuildContext, setState StateSetter) Widget

// StatefulBuilder is a platonic widget that calls a closure to obtain its child
// widget, and lets that closure rebuild it with a setState function.
//
// Flutter: widgets/basic.dart StatefulBuilder
type StatefulBuilder struct {
	StatefulWidget
	builder StatefulBuilderFunc
}

// NewStatefulBuilder creates a widget that has the state that the builder
// changes with setState.
//
// Flutter: widgets/basic.dart StatefulBuilder.new
func NewStatefulBuilder(builder StatefulBuilderFunc, opts ...WidgetOpts) *StatefulBuilder {
	return &StatefulBuilder{NewStatefulWidgetBase(firstOpts(opts)), builder}
}

// Builder is the function that builds the child.
//
// Flutter: widgets/basic.dart StatefulBuilder.builder
func (w *StatefulBuilder) Builder() StatefulBuilderFunc { return w.builder }

// CreateState creates the state.
//
// Flutter: widgets/basic.dart StatefulBuilder.createState
func (w *StatefulBuilder) CreateState() State { return &statefulBuilderState{} }

type statefulBuilderState struct{ StateOf[*StatefulBuilder] }

func (s *statefulBuilderState) Build(bc BuildContext) Widget {
	return s.Widget().builder(bc, s.SetState)
}

// ValueWidgetBuilder builds a widget for the value of a ValueListenable.
//
// Flutter: widgets/value_listenable_builder.dart ValueWidgetBuilder
type ValueWidgetBuilder[T any] func(bc BuildContext, value T, child Widget) Widget

// ValueListenableBuilderOpts are the optional parameters of
// NewValueListenableBuilder.
//
// Flutter: widgets/value_listenable_builder.dart ValueListenableBuilder.new
type ValueListenableBuilderOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// ValueListenableBuilder rebuilds its subtree when the value of a
// ValueListenable changes.
//
// Flutter: widgets/value_listenable_builder.dart ValueListenableBuilder
type ValueListenableBuilder[T any] struct {
	StatefulWidget
	valueListenable foundation.ValueListenable[T]
	builder         ValueWidgetBuilder[T]
	child           Widget
}

// NewValueListenableBuilder creates a widget that builds with the value of the
// listenable and builds again each time it changes. The child, which may be nil,
// is the part of the subtree that does not depend on the value: it is given to
// the builder, which puts it in what it builds.
//
// Flutter: widgets/value_listenable_builder.dart ValueListenableBuilder.new
func NewValueListenableBuilder[T any](valueListenable foundation.ValueListenable[T], builder ValueWidgetBuilder[T], child Widget,
	opts ...ValueListenableBuilderOpts) *ValueListenableBuilder[T] {
	return &ValueListenableBuilder[T]{NewStatefulWidgetBase(WidgetOpts(firstOpts(opts))), valueListenable, builder, child}
}

// ValueListenable is the listenable whose value is built.
//
// Flutter: widgets/value_listenable_builder.dart ValueListenableBuilder.valueListenable
func (w *ValueListenableBuilder[T]) ValueListenable() foundation.ValueListenable[T] {
	return w.valueListenable
}

// Builder is the function that builds the widget for the value.
//
// Flutter: widgets/value_listenable_builder.dart ValueListenableBuilder.builder
func (w *ValueListenableBuilder[T]) Builder() ValueWidgetBuilder[T] { return w.builder }

// Child is the part of the subtree that does not depend on the value.
//
// Flutter: widgets/value_listenable_builder.dart ValueListenableBuilder.child
func (w *ValueListenableBuilder[T]) Child() Widget { return w.child }

// CreateState creates the state.
//
// Flutter: widgets/value_listenable_builder.dart ValueListenableBuilder.createState
func (w *ValueListenableBuilder[T]) CreateState() State { return &valueListenableBuilderState[T]{} }

type valueListenableBuilderState[T any] struct {
	StateOf[*ValueListenableBuilder[T]]
	value  T
	handle foundation.ListenerHandle
}

func (s *valueListenableBuilderState[T]) InitState() {
	s.value = s.Widget().valueListenable.Value()
	s.handle = s.Widget().valueListenable.AddListener(s.valueChanged)
}

func (s *valueListenableBuilderState[T]) DidUpdateWidget(old *ValueListenableBuilder[T]) {
	if old.valueListenable != s.Widget().valueListenable {
		old.valueListenable.RemoveListener(s.handle)
		s.value = s.Widget().valueListenable.Value()
		s.handle = s.Widget().valueListenable.AddListener(s.valueChanged)
	}
}

func (s *valueListenableBuilderState[T]) Dispose() {
	s.Widget().valueListenable.RemoveListener(s.handle)
}

func (s *valueListenableBuilderState[T]) valueChanged() {
	s.SetState(func() { s.value = s.Widget().valueListenable.Value() })
}

func (s *valueListenableBuilderState[T]) Build(bc BuildContext) Widget {
	return s.Widget().builder(bc, s.value, s.Widget().child)
}

// TransitionBuilder builds a widget for a ListenableBuilder or an
// AnimatedBuilder.
//
// Flutter: widgets/transitions.dart TransitionBuilder
type TransitionBuilder func(bc BuildContext, child Widget) Widget

// ListenableBuilderOpts are the optional parameters of NewListenableBuilder.
//
// Flutter: widgets/transitions.dart ListenableBuilder.new
type ListenableBuilderOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// ListenableBuilder is a widget that rebuilds when a Listenable notifies, such
// as a ChangeNotifier or an animation.
//
// Flutter: widgets/transitions.dart ListenableBuilder
type ListenableBuilder struct {
	StatefulWidget
	listenable foundation.Listenable
	builder    TransitionBuilder
	child      Widget
}

// NewListenableBuilder creates a widget that builds again each time the
// listenable notifies. The child, which may be nil, is the part of the subtree
// that does not depend on the listenable.
//
// Flutter: widgets/transitions.dart ListenableBuilder.new
func NewListenableBuilder(listenable foundation.Listenable, builder TransitionBuilder, child Widget, opts ...ListenableBuilderOpts) *ListenableBuilder {
	return &ListenableBuilder{NewStatefulWidgetBase(WidgetOpts(firstOpts(opts))), listenable, builder, child}
}

// Listenable is the object that the widget listens to.
//
// Flutter: widgets/transitions.dart AnimatedWidget.listenable
func (w *ListenableBuilder) Listenable() foundation.Listenable { return w.listenable }

// Builder is the function that builds the widget.
//
// Flutter: widgets/transitions.dart ListenableBuilder.builder
func (w *ListenableBuilder) Builder() TransitionBuilder { return w.builder }

// Child is the part of the subtree that does not depend on the listenable.
//
// Flutter: widgets/transitions.dart ListenableBuilder.child
func (w *ListenableBuilder) Child() Widget { return w.child }

// CreateState creates the state.
//
// Flutter: widgets/transitions.dart AnimatedWidget.createState
func (w *ListenableBuilder) CreateState() State { return &listenableBuilderState{} }

// listenableBuilderWidget is a ListenableBuilder or an AnimatedBuilder: the
// state works with both, as StateOf needs the exact type of the widget.
type listenableBuilderWidget interface {
	Widget
	Listenable() foundation.Listenable
	Builder() TransitionBuilder
	Child() Widget
}

type listenableBuilderState struct {
	StateOf[listenableBuilderWidget]
	handle foundation.ListenerHandle
}

func (s *listenableBuilderState) InitState() {
	s.handle = s.Widget().Listenable().AddListener(s.changed)
}

func (s *listenableBuilderState) DidUpdateWidget(old listenableBuilderWidget) {
	if old.Listenable() != s.Widget().Listenable() {
		old.Listenable().RemoveListener(s.handle)
		s.handle = s.Widget().Listenable().AddListener(s.changed)
	}
}

func (s *listenableBuilderState) Dispose() { s.Widget().Listenable().RemoveListener(s.handle) }

func (s *listenableBuilderState) changed() {
	// The state itself is not mutable; the call only marks it dirty.
	s.SetState(func() {})
}

func (s *listenableBuilderState) Build(bc BuildContext) Widget {
	return s.Widget().Builder()(bc, s.Widget().Child())
}

// AnimatedBuilderOpts are the optional parameters of NewAnimatedBuilder.
//
// Flutter: widgets/transitions.dart AnimatedBuilder.new
type AnimatedBuilderOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// AnimatedBuilder is a general-purpose widget for building animations: a
// ListenableBuilder that listens to an animation.
//
// Flutter: widgets/transitions.dart AnimatedBuilder
type AnimatedBuilder struct{ ListenableBuilder }

// NewAnimatedBuilder creates a widget that builds again each time the animation
// (or any listenable) notifies.
//
// Flutter: widgets/transitions.dart AnimatedBuilder.new
func NewAnimatedBuilder(animation foundation.Listenable, builder TransitionBuilder, child Widget, opts ...AnimatedBuilderOpts) *AnimatedBuilder {
	return &AnimatedBuilder{*NewListenableBuilder(animation, builder, child, ListenableBuilderOpts(firstOpts(opts)))}
}

// Animation is the animation (or listenable) that the widget builds with.
//
// Flutter: widgets/transitions.dart AnimatedBuilder.animation
func (w *AnimatedBuilder) Animation() foundation.Listenable { return w.listenable }
