package widgets

import (
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/services"
)

// Intent is a description of an action that the user wants to do, such as moving
// the focus to the next widget or dismissing a dialog. Shortcuts map key events
// to intents, and Actions map the types of intents to the Action that carries
// them out.
//
// An intent is a value: embed IntentBase in a struct, and pass the struct itself
// (not a pointer) to Actions and Shortcuts. The type of the value, as
// IntentTypeOf gives it, is the key that finds the Action.
//
// Deviation: Flutter's Intent is an abstract class; Glutter's is this interface
// with the embeddable IntentBase (SPEC §12.10).
//
// Flutter: widgets/actions.dart Intent
type Intent interface {
	// IsIntent marks the type as an intent.
	IsIntent()
}

// IntentBase is embedded in a struct to make it an Intent.
//
// Flutter: widgets/actions.dart Intent
type IntentBase struct{}

// IsIntent marks the type as an intent.
//
// Flutter: widgets/actions.dart Intent
func (IntentBase) IsIntent() {}

// DebugFillProperties adds nothing: an intent has no properties of its own.
//
// Flutter: widgets/actions.dart Intent
func (IntentBase) DebugFillProperties(*foundation.DiagnosticPropertiesBuilder) {}

// IntentTypeOf is the key of an intent type in an ActionMap: Flutter's Type
// literal, such as DismissIntent.
//
// Flutter: widgets/actions.dart Actions.actions
func IntentTypeOf[T Intent]() reflect.Type { return reflect.TypeFor[T]() }

// ActionMap maps the type of an intent to the action that handles it, like
// Flutter's Map<Type, Action<Intent>>. Make the keys with IntentTypeOf.
//
// Flutter: widgets/actions.dart Actions.actions
type ActionMap map[reflect.Type]Action

// ActionListenerCallback is the type of a function that listens to an action.
//
// Flutter: widgets/actions.dart ActionListenerCallback
type ActionListenerCallback func(action Action)

// ActionListenerHandle identifies a listener added to an action, to remove it.
//
// Flutter: no counterpart (Go functions are not comparable)
type ActionListenerHandle int

// Action is a handler of an Intent: a discrete piece of work that the user asks
// for. An action is found for an intent by the Actions widgets above a context,
// and Shortcuts invoke it for a key.
//
// Implement it by embedding ActionBase (or ContextActionBase), setting the base
// up with InitActionBase, and giving the methods that differ from the defaults:
// IntentType, Invoke (or InvokeWithContext), and optionally IsActionEnabled,
// IsEnabled, ConsumesKey and ToKeyEventResult. Generic code asks the intent
// for its concrete type with a type assertion.
//
// Deviation: Flutter's Action<T extends Intent> is a generic abstract class; the
// type argument is the value of IntentType (SPEC §12.10).
//
// Flutter: widgets/actions.dart Action
type Action interface {
	// IntentType is the type of the intents the action handles. An interface type
	// (such as the Intent interface) accepts every intent that implements it.
	//
	// Flutter: widgets/actions.dart Action.intentType
	IntentType() reflect.Type
	// IsEnabled is whether the action is enabled for the intent: it can be
	// invoked. The default is IsActionEnabled.
	//
	// Flutter: widgets/actions.dart Action.isEnabled
	IsEnabled(intent Intent) bool
	// IsActionEnabled is whether the action is enabled for every intent.
	//
	// Flutter: widgets/actions.dart Action.isActionEnabled
	IsActionEnabled() bool
	// ConsumesKey is whether the key event that triggered the intent is consumed:
	// handled by the action and not passed on. The default is true.
	//
	// Flutter: widgets/actions.dart Action.consumesKey
	ConsumesKey(intent Intent) bool
	// ToKeyEventResult is what a key event that triggered the intent results in,
	// given the value that Invoke returned. The default is Handled when ConsumesKey,
	// else SkipRemainingHandlers.
	//
	// Flutter: widgets/actions.dart Action.toKeyEventResult
	ToKeyEventResult(intent Intent, invokeResult any) KeyEventResult
	// Invoke carries out the action and returns its result, which is nil for most
	// actions.
	//
	// Flutter: widgets/actions.dart Action.invoke
	Invoke(intent Intent) any
	// AddActionListener adds a function that is called when the action notifies
	// that its enabled state changed.
	//
	// Flutter: widgets/actions.dart Action.addActionListener
	AddActionListener(listener ActionListenerCallback) ActionListenerHandle
	// RemoveActionListener removes a listener.
	//
	// Flutter: widgets/actions.dart Action.removeActionListener
	RemoveActionListener(handle ActionListenerHandle)
	// CallingAction is the action that this one overrides while it is invoked as
	// an override, or nil.
	//
	// Flutter: widgets/actions.dart Action.callingAction
	CallingAction() Action

	actionBase() *ActionBase
}

// ContextAction is an action that is invoked with the context it was found from.
//
// Flutter: widgets/actions.dart ContextAction
type ContextAction interface {
	Action
	// IsEnabledWithContext is IsEnabled with the context, which is nil when the
	// action is asked without one.
	//
	// Flutter: widgets/actions.dart ContextAction.isEnabled
	IsEnabledWithContext(intent Intent, bc BuildContext) bool
	// InvokeWithContext is Invoke with the context, which is nil when the action
	// is invoked without one.
	//
	// Flutter: widgets/actions.dart ContextAction.invoke
	InvokeWithContext(intent Intent, bc BuildContext) any
}

type actionListenerEntry struct {
	handle ActionListenerHandle
	fn     ActionListenerCallback
}

// ActionBase implements the parts of Action that do not depend on the intent.
// Embed it, and call InitActionBase with the outer value.
//
// Flutter: widgets/actions.dart Action
type ActionBase struct {
	self          Action
	listeners     []actionListenerEntry
	nextHandle    ActionListenerHandle
	callingAction Action
	reporter      foundation.ErrorReporter
}

// InitActionBase sets the outer value of the action.
//
// Flutter: widgets/actions.dart Action.new
func InitActionBase(b *ActionBase, self Action) { b.self = self }

func (b *ActionBase) actionBase() *ActionBase { return b }

func (b *ActionBase) outer() Action {
	if b.self == nil {
		panic(errors.New(tagActionNotInitialized, "an action must call InitActionBase with its outer value"))
	}
	return b.self
}

// IntentType accepts every intent; actions for one kind of intent override it.
//
// Flutter: widgets/actions.dart Action.intentType
func (b *ActionBase) IntentType() reflect.Type { return reflect.TypeFor[Intent]() }

// IsEnabled is IsActionEnabled.
//
// Flutter: widgets/actions.dart Action.isEnabled
func (b *ActionBase) IsEnabled(Intent) bool { return b.outer().IsActionEnabled() }

// IsActionEnabled is true.
//
// Flutter: widgets/actions.dart Action.isActionEnabled
func (b *ActionBase) IsActionEnabled() bool { return true }

// ConsumesKey is true.
//
// Flutter: widgets/actions.dart Action.consumesKey
func (b *ActionBase) ConsumesKey(Intent) bool { return true }

// ToKeyEventResult is Handled when the action consumes the key, else
// SkipRemainingHandlers.
//
// Flutter: widgets/actions.dart Action.toKeyEventResult
func (b *ActionBase) ToKeyEventResult(intent Intent, _ any) KeyEventResult {
	if b.outer().ConsumesKey(intent) {
		return KeyEventResultHandled
	}
	return KeyEventResultSkipRemainingHandlers
}

// Invoke panics: an action must implement it.
//
// Flutter: widgets/actions.dart Action.invoke
func (b *ActionBase) Invoke(Intent) any {
	panic(errors.New(tagActionInvokeMissing, "the action does not implement Invoke", "action", foundation.DescribeIdentity(b.self)))
}

// AddActionListener adds a listener of the action's changes.
//
// Flutter: widgets/actions.dart Action.addActionListener
func (b *ActionBase) AddActionListener(listener ActionListenerCallback) ActionListenerHandle {
	b.nextHandle++
	b.listeners = append(b.listeners, actionListenerEntry{b.nextHandle, listener})
	return b.nextHandle
}

// RemoveActionListener removes a listener.
//
// Flutter: widgets/actions.dart Action.removeActionListener
func (b *ActionBase) RemoveActionListener(handle ActionListenerHandle) {
	b.listeners = slices.DeleteFunc(b.listeners, func(e actionListenerEntry) bool { return e.handle == handle })
}

// SetErrorReporter sets the function that receives the panics of the listeners.
// Without one, the first panic goes on after the last listener was called.
//
// Flutter: no counterpart (FlutterError.reportError is global)
func (b *ActionBase) SetErrorReporter(r foundation.ErrorReporter) { b.reporter = r }

// CallingAction is the action that this one overrides while it runs as an
// override.
//
// Flutter: widgets/actions.dart Action.callingAction
func (b *ActionBase) CallingAction() Action { return b.callingAction }

// NotifyActionListeners tells the listeners that the action changed. A listener
// that panics does not stop the others.
//
// Flutter: widgets/actions.dart Action.notifyActionListeners
func (b *ActionBase) NotifyActionListeners() {
	if len(b.listeners) == 0 {
		return
	}
	var first any
	for _, entry := range slices.Clone(b.listeners) {
		if !slices.ContainsFunc(b.listeners, func(e actionListenerEntry) bool { return e.handle == entry.handle }) {
			continue
		}
		func() {
			defer func() {
				r := recover()
				if r == nil {
					return
				}
				if b.reporter == nil {
					if first == nil {
						first = r
					}
					return
				}
				b.reporter(foundation.PanicError(foundation.CallbackPanicked,
					"while dispatching notifications for "+foundation.DescribeIdentity(b.self), r))
			}()
			entry.fn(b.self)
		}()
	}
	if first != nil {
		panic(first)
	}
}

// DebugFillProperties adds nothing.
//
// Flutter: widgets/actions.dart Action
func (b *ActionBase) DebugFillProperties(*foundation.DiagnosticPropertiesBuilder) {}

// ContextActionBase implements ContextAction on top of ActionBase: Invoke and
// IsEnabled are the context variants with no context. Embed it, call
// InitContextActionBase, and implement InvokeWithContext.
//
// Flutter: widgets/actions.dart ContextAction
type ContextActionBase struct {
	ActionBase
	cself ContextAction
}

// InitContextActionBase sets the outer value of the action.
//
// Flutter: widgets/actions.dart ContextAction.new
func InitContextActionBase(b *ContextActionBase, self ContextAction) {
	InitActionBase(&b.ActionBase, self)
	b.cself = self
}

// IsEnabled is IsEnabledWithContext with no context.
//
// Flutter: widgets/actions.dart ContextAction.isEnabled
func (b *ContextActionBase) IsEnabled(intent Intent) bool {
	return b.cself.IsEnabledWithContext(intent, nil)
}

// IsEnabledWithContext is IsActionEnabled.
//
// Flutter: widgets/actions.dart ContextAction.isEnabled
func (b *ContextActionBase) IsEnabledWithContext(Intent, BuildContext) bool {
	return b.cself.IsActionEnabled()
}

// Invoke is InvokeWithContext with no context.
//
// Flutter: widgets/actions.dart ContextAction.invoke
func (b *ContextActionBase) Invoke(intent Intent) any { return b.cself.InvokeWithContext(intent, nil) }

// InvokeWithContext panics: an action must implement it.
//
// Flutter: widgets/actions.dart ContextAction.invoke
func (b *ContextActionBase) InvokeWithContext(Intent, BuildContext) any {
	panic(errors.New(tagActionInvokeMissing, "the action does not implement InvokeWithContext", "action", foundation.DescribeIdentity(b.cself)))
}

// actionIsEnabled is Action._isEnabled: the context variant for a ContextAction.
func actionIsEnabled(a Action, intent Intent, bc BuildContext) bool {
	if ca, ok := a.(ContextAction); ok {
		return ca.IsEnabledWithContext(intent, bc)
	}
	return a.IsEnabled(intent)
}

// actionInvoke is Action._invoke: the context variant for a ContextAction.
func actionInvoke(a Action, intent Intent, bc BuildContext) any {
	if ca, ok := a.(ContextAction); ok {
		return ca.InvokeWithContext(intent, bc)
	}
	return a.Invoke(intent)
}

type callingActionUpdater interface{ updateCallingAction(value Action) }

// updateCallingAction is Action._updateCallingAction.
func updateCallingAction(a Action, value Action) {
	if u, ok := a.(callingActionUpdater); ok {
		u.updateCallingAction(value)
		return
	}
	a.actionBase().callingAction = value
}

// intentIsOfType reports whether the intent can be handled by an action of the
// intent type.
func intentIsOfType(intent Intent, t reflect.Type) bool {
	it := reflect.TypeOf(intent)
	if t.Kind() == reflect.Interface {
		return it.Implements(t)
	}
	return it == t
}

// debugCanHandleIntent panics in debug builds with the tag ActionIntentMismatch
// when the intent is not of the action's intent type.
func debugCanHandleIntent(a Action, intent Intent) {
	if !foundation.DebugMode || intent == nil {
		return
	}
	if !intentIsOfType(intent, a.IntentType()) {
		panic(errors.New(tagActionIntentMismatch,
			fmt.Sprintf("an Intent of type %v cannot be handled by %v: the Intent must be of the type %v", reflect.TypeOf(intent), foundation.DescribeIdentity(a), a.IntentType()),
			"intent", fmt.Sprint(intent), "action", foundation.DescribeIdentity(a)))
	}
}

// ActionListener is a widget that listens to an action and calls a function when
// the action notifies; it removes the listener when it goes away.
//
// Flutter: widgets/actions.dart ActionListener
type ActionListener struct {
	StatefulWidget
	listener ActionListenerCallback
	action   Action
	child    Widget
}

// NewActionListener creates the widget.
//
// Flutter: widgets/actions.dart ActionListener.new
func NewActionListener(listener ActionListenerCallback, action Action, child Widget, opts ...WidgetOpts) *ActionListener {
	return &ActionListener{NewStatefulWidgetBase(opts...), listener, action, child}
}

// Listener is the function that is called when the action notifies.
//
// Flutter: widgets/actions.dart ActionListener.listener
func (w *ActionListener) Listener() ActionListenerCallback { return w.listener }

// Action is the action that is listened to.
//
// Flutter: widgets/actions.dart ActionListener.action
func (w *ActionListener) Action() Action { return w.action }

// Child is the widget below.
//
// Flutter: widgets/actions.dart ActionListener.child
func (w *ActionListener) Child() Widget { return w.child }

// CreateState creates the state.
//
// Flutter: widgets/actions.dart ActionListener.createState
func (w *ActionListener) CreateState() State { return &actionListenerState{} }

type actionListenerState struct {
	StateOf[*ActionListener]
	handle ActionListenerHandle
}

func (s *actionListenerState) InitState() {
	s.handle = s.Widget().action.AddActionListener(s.Widget().listener)
}

func (s *actionListenerState) DidUpdateWidget(old *ActionListener) {
	// A function cannot be compared, so the listener is always replaced.
	old.action.RemoveActionListener(s.handle)
	s.handle = s.Widget().action.AddActionListener(s.Widget().listener)
}

func (s *actionListenerState) Dispose() { s.Widget().action.RemoveActionListener(s.handle) }

func (s *actionListenerState) Build(BuildContext) Widget { return s.Widget().child }

// CallbackAction is an action that calls a function with the intent.
//
// Flutter: widgets/actions.dart CallbackAction
type CallbackAction[T Intent] struct {
	ActionBase
	onInvoke func(intent T) any
}

// NewCallbackAction creates an action for intents of type T that calls onInvoke.
//
// Flutter: widgets/actions.dart CallbackAction.new
func NewCallbackAction[T Intent](onInvoke func(intent T) any) *CallbackAction[T] {
	a := &CallbackAction[T]{onInvoke: onInvoke}
	InitActionBase(&a.ActionBase, a)
	return a
}

// IntentType is the type T.
//
// Flutter: widgets/actions.dart Action.intentType
func (a *CallbackAction[T]) IntentType() reflect.Type { return reflect.TypeFor[T]() }

// Invoke calls the function.
//
// Flutter: widgets/actions.dart CallbackAction.invoke
func (a *CallbackAction[T]) Invoke(intent Intent) any { return a.onInvoke(intent.(T)) }

// ActionDispatcher invokes actions: it is the one place that runs them, so that
// it can be replaced to log or intercept. Actions widgets have one (the default,
// when none is given).
//
// Flutter: widgets/actions.dart ActionDispatcher
type ActionDispatcher interface {
	// InvokeAction invokes the action for the intent; the action must be enabled.
	// A nil context means the context of the primary focus.
	//
	// Flutter: widgets/actions.dart ActionDispatcher.invokeAction
	InvokeAction(action Action, intent Intent, bc BuildContext) any
	// InvokeActionIfEnabled invokes the action if it is enabled, and says whether
	// it was and what it returned.
	//
	// Flutter: widgets/actions.dart ActionDispatcher.invokeActionIfEnabled
	InvokeActionIfEnabled(action Action, intent Intent, bc BuildContext) (bool, any)
}

// ActionDispatcherBase is the default ActionDispatcher. Embed it to override a
// method and call the embedded one for the rest.
//
// Flutter: widgets/actions.dart ActionDispatcher
type ActionDispatcherBase struct {
	focusManager *FocusManager
	self         ActionDispatcher
}

// InitActionDispatcherBase tells the base of a dispatcher that embeds it which
// dispatcher it is part of, so that InvokeActionIfEnabled calls the InvokeAction
// the outer dispatcher overrides. Call it once, when the outer dispatcher is
// made.
//
// Flutter: widgets/actions.dart ActionDispatcher.invokeActionIfEnabled
func InitActionDispatcherBase(b *ActionDispatcherBase, self ActionDispatcher, focusManager ...*FocusManager) {
	b.self = self
	if len(focusManager) > 0 {
		b.focusManager = focusManager[0]
	}
}

func (d *ActionDispatcherBase) outer() ActionDispatcher {
	if d.self != nil {
		return d.self
	}
	return d
}

// NewActionDispatcher creates the default dispatcher. The focus manager gives it
// the primary focus, whose context an action is invoked with when none is given;
// without one, such an action gets no context.
//
// Flutter: widgets/actions.dart ActionDispatcher.new
func NewActionDispatcher(focusManager ...*FocusManager) *ActionDispatcherBase {
	d := &ActionDispatcherBase{}
	InitActionDispatcherBase(d, d, focusManager...)
	return d
}

func (d *ActionDispatcherBase) target(bc BuildContext) BuildContext {
	if bc != nil {
		return bc
	}
	if d.focusManager != nil {
		if focus := d.focusManager.PrimaryFocus(); focus != nil && focus.Context() != nil {
			return focus.Context()
		}
	}
	return nil
}

// InvokeAction invokes the action. In debug builds it panics with the tag
// ActionNotEnabled when the action is not enabled.
//
// Flutter: widgets/actions.dart ActionDispatcher.invokeAction
func (d *ActionDispatcherBase) InvokeAction(action Action, intent Intent, bc BuildContext) any {
	target := d.target(bc)
	if foundation.DebugMode && !actionIsEnabled(action, intent, target) {
		panic(errors.New(tagActionNotEnabled, "the action must be enabled when calling InvokeAction", "action", foundation.DescribeIdentity(action)))
	}
	return actionInvoke(action, intent, target)
}

// InvokeActionIfEnabled invokes the action if it is enabled.
//
// Flutter: widgets/actions.dart ActionDispatcher.invokeActionIfEnabled
func (d *ActionDispatcherBase) InvokeActionIfEnabled(action Action, intent Intent, bc BuildContext) (bool, any) {
	target := d.target(bc)
	if actionIsEnabled(action, intent, target) {
		return true, d.outer().InvokeAction(action, intent, target)
	}
	return false, nil
}

// DebugFillProperties adds nothing.
//
// Flutter: widgets/actions.dart ActionDispatcher
func (d *ActionDispatcherBase) DebugFillProperties(*foundation.DiagnosticPropertiesBuilder) {}

// Actions is a widget that establishes a map of intent types to actions for its
// descendants. Use ActionsInvoke and the other functions to find an action for
// an intent and run it.
//
// Flutter: widgets/actions.dart Actions
type Actions struct {
	StatefulWidget
	dispatcher ActionDispatcher
	actions    ActionMap
	child      Widget
}

// ActionsOpts are the optional parameters of NewActions.
//
// Flutter: widgets/actions.dart Actions.new
type ActionsOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Dispatcher invokes the actions; the default is the one of the closest Actions
	// above that has one, else the default dispatcher.
	Dispatcher ActionDispatcher
}

// NewActions creates a widget that maps intent types to actions.
//
// Flutter: widgets/actions.dart Actions.new
func NewActions(actions ActionMap, child Widget, opts ...ActionsOpts) *Actions {
	o := firstOpts(opts)
	return &Actions{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), o.Dispatcher, actions, child}
}

// Dispatcher is the dispatcher of the widget, or nil.
//
// Flutter: widgets/actions.dart Actions.dispatcher
func (w *Actions) Dispatcher() ActionDispatcher { return w.dispatcher }

// Actions is the map of intent types to actions.
//
// Flutter: widgets/actions.dart Actions.actions
func (w *Actions) Actions() ActionMap { return w.actions }

// Child is the widget below.
//
// Flutter: widgets/actions.dart Actions.child
func (w *Actions) Child() Widget { return w.child }

// CreateState creates the state.
//
// Flutter: widgets/actions.dart Actions.createState
func (w *Actions) CreateState() State { return &actionsState{} }

// DebugFillProperties adds the dispatcher and the actions.
//
// Flutter: widgets/actions.dart Actions.debugFillProperties
func (w *Actions) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("dispatcher", w.dispatcher))
	p.Add(foundation.NewDiagnosticsProperty("actions", describeActionMap(w.actions)))
}

type describedActionMap string

func (d describedActionMap) String() string { return string(d) }

func describeActionMap(m ActionMap) describedActionMap {
	var parts []string
	for t, a := range m {
		parts = append(parts, fmt.Sprintf("%v: %s", typeName(t), foundation.DescribeIdentity(a)))
	}
	slices.Sort(parts)
	s := "{"
	for i, p := range parts {
		if i > 0 {
			s += ", "
		}
		s += p
	}
	return describedActionMap(s + "}")
}

func typeName(t reflect.Type) string { return t.Name() }

// actionsScope is the inherited widget that carries the map to the descendants.
//
// Flutter: widgets/actions.dart _ActionsScope
type actionsScope struct {
	InheritedWidget
	dispatcher ActionDispatcher
	actions    ActionMap
	rebuildKey int
}

func (w *actionsScope) UpdateShouldNotify(old Widget) bool {
	o := old.(*actionsScope)
	return w.rebuildKey != o.rebuildKey || o.dispatcher != w.dispatcher || !actionMapsEqual(o.actions, w.actions)
}

func actionMapsEqual(a, b ActionMap) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if other, ok := b[k]; !ok || other != v {
			return false
		}
	}
	return true
}

type actionsState struct {
	StateOf[*Actions]
	listened   map[Action]ActionListenerHandle
	rebuildKey int
}

func (s *actionsState) InitState() {
	s.listened = map[Action]ActionListenerHandle{}
	s.updateActionListeners()
}

func (s *actionsState) handleActionChanged(Action) {
	s.SetState(func() { s.rebuildKey++ })
}

func (s *actionsState) updateActionListeners() {
	widgetActions := map[Action]struct{}{}
	for _, a := range s.Widget().actions {
		widgetActions[a] = struct{}{}
	}
	for a, handle := range s.listened {
		if _, ok := widgetActions[a]; !ok {
			a.RemoveActionListener(handle)
			delete(s.listened, a)
		}
	}
	for a := range widgetActions {
		if _, ok := s.listened[a]; !ok {
			s.listened[a] = a.AddActionListener(s.handleActionChanged)
		}
	}
}

func (s *actionsState) DidUpdateWidget(*Actions) { s.updateActionListeners() }

func (s *actionsState) Dispose() {
	for a, handle := range s.listened {
		a.RemoveActionListener(handle)
	}
	s.listened = nil
}

func (s *actionsState) Build(BuildContext) Widget {
	w := s.Widget()
	return &actionsScope{NewInheritedWidgetBase(w.child), w.dispatcher, w.actions, s.rebuildKey}
}

// parentOf is the first ancestor of the context.
func parentOf(bc BuildContext) BuildContext {
	var parent BuildContext
	bc.VisitAncestorElements(func(ancestor Element) bool {
		parent = ancestor
		return false
	})
	return parent
}

// visitActionsAncestors calls the visitor with the elements of the Actions
// widgets above the context, nearest first, until it returns true. It returns
// whether it stopped at an element.
func visitActionsAncestors(bc BuildContext, visitor func(element InheritedElementSelf) bool) bool {
	if !bc.Mounted() {
		return false
	}
	element := GetElementForInheritedWidget[*actionsScope](bc)
	for element != nil {
		if visitor(element) {
			break
		}
		parent := parentOf(element)
		element = GetElementForInheritedWidget[*actionsScope](parent)
	}
	return element != nil
}

func findDispatcher(bc BuildContext) ActionDispatcher {
	var dispatcher ActionDispatcher
	visitActionsAncestors(bc, func(element InheritedElementSelf) bool {
		if found := element.Widget().(*actionsScope).dispatcher; found != nil {
			dispatcher = found
			return true
		}
		return false
	})
	if dispatcher != nil {
		return dispatcher
	}
	return NewActionDispatcher(bc.Owner().FocusManager())
}

// actionForIntent is Actions._getActionForIntent: the action mapped to the type of
// the intent (or to the given type when there is no intent).
func actionForIntent(scope *actionsScope, intent Intent, t reflect.Type) Action {
	if intent != nil {
		t = reflect.TypeOf(intent)
	}
	mapped := scope.actions[t]
	if mapped != nil {
		debugCanHandleIntent(mapped, intent)
	}
	return mapped
}

// ActionsOf is the dispatcher of the closest Actions above the context that has
// one, and the context depends on the closest Actions. Without a dispatcher it is
// the default one.
//
// Flutter: widgets/actions.dart Actions.of
func ActionsOf(bc BuildContext) ActionDispatcher {
	if marker, ok := DependOn[*actionsScope](bc); ok && marker.dispatcher != nil {
		return marker.dispatcher
	}
	return findDispatcher(bc)
}

// ActionsHandler returns a function that invokes the action bound to the intent,
// or nil when there is none or it is not enabled. A widget like a button uses it
// as its callback.
//
// Flutter: widgets/actions.dart Actions.handler
func ActionsHandler(bc BuildContext, intent Intent) func() {
	action := ActionsMaybeFindIntent(bc, intent)
	if action != nil && actionIsEnabled(action, intent, bc) {
		return func() {
			if actionIsEnabled(action, intent, bc) {
				ActionsOf(bc).InvokeAction(action, intent, bc)
			}
		}
	}
	return nil
}

// ActionsFind is the action bound to the intent type T above the context, and
// the context depends on the Actions that has it. In debug builds it panics with
// the tag ActionNotFound when there is none.
//
// Flutter: widgets/actions.dart Actions.find
func ActionsFind[T Intent](bc BuildContext) Action {
	return actionsFind(bc, nil, reflect.TypeFor[T]())
}

// ActionsFindIntent is ActionsFind for the type of the intent.
//
// Flutter: widgets/actions.dart Actions.find
func ActionsFindIntent(bc BuildContext, intent Intent) Action {
	return actionsFind(bc, intent, nil)
}

func actionsFind(bc BuildContext, intent Intent, t reflect.Type) Action {
	action := actionsMaybeFind(bc, intent, t)
	if foundation.DebugMode && action == nil {
		if intent != nil {
			t = reflect.TypeOf(intent)
		}
		panic(errors.New(tagActionNotFound,
			"unable to find an action for a "+t.String()+" in an Actions widget in the given context: Actions.find() was called on a context that doesn't contain an Actions widget with a mapping for the given intent type",
			"context", describeContext(bc), "intent", t.String()))
	}
	return action
}

// ActionsMaybeFind is ActionsFind that returns nil instead of panicking.
//
// Flutter: widgets/actions.dart Actions.maybeFind
func ActionsMaybeFind[T Intent](bc BuildContext) Action {
	return actionsMaybeFind(bc, nil, reflect.TypeFor[T]())
}

// ActionsMaybeFindIntent is ActionsMaybeFind for the type of the intent.
//
// Flutter: widgets/actions.dart Actions.maybeFind
func ActionsMaybeFindIntent(bc BuildContext, intent Intent) Action {
	return actionsMaybeFind(bc, intent, nil)
}

func actionsMaybeFind(bc BuildContext, intent Intent, t reflect.Type) Action {
	var action Action
	visitActionsAncestors(bc, func(element InheritedElementSelf) bool {
		result := actionForIntent(element.Widget().(*actionsScope), intent, t)
		if result != nil {
			bc.DependOnInheritedElement(element, nil)
			action = result
			return true
		}
		return false
	})
	return action
}

// maybeFindWithoutDependingOn is Actions._maybeFindWithoutDependingOn.
func maybeFindWithoutDependingOn(bc BuildContext, intent Intent, t reflect.Type) Action {
	var action Action
	visitActionsAncestors(bc, func(element InheritedElementSelf) bool {
		result := actionForIntent(element.Widget().(*actionsScope), intent, t)
		if result != nil {
			action = result
			return true
		}
		return false
	})
	return action
}

// ActionsInvoke finds the action bound to the intent above the context and
// invokes it if it is enabled, and returns what it returned (nil for an action
// that is not enabled). In debug builds it panics with the tag ActionNotFound
// when no Actions has a mapping for the intent.
//
// Flutter: widgets/actions.dart Actions.invoke
func ActionsInvoke(bc BuildContext, intent Intent) any {
	var returnValue any
	actionFound := visitActionsAncestors(bc, func(element InheritedElementSelf) bool {
		result := actionForIntent(element.Widget().(*actionsScope), intent, nil)
		if result != nil && actionIsEnabled(result, intent, bc) {
			returnValue = findDispatcher(element).InvokeAction(result, intent, bc)
		}
		return result != nil
	})
	if foundation.DebugMode && !actionFound {
		panic(errors.New(tagActionNotFound,
			"unable to find an action for an Intent with type "+reflect.TypeOf(intent).String()+" in an Actions widget in the given context: Actions.invoke() was unable to find an Actions widget that contained a mapping for the given intent",
			"context", describeContext(bc), "intent", reflect.TypeOf(intent).String()))
	}
	return returnValue
}

// ActionsMaybeInvoke is ActionsInvoke that does nothing, and returns nil, when no
// action is found.
//
// Flutter: widgets/actions.dart Actions.maybeInvoke
func ActionsMaybeInvoke(bc BuildContext, intent Intent) any {
	var returnValue any
	visitActionsAncestors(bc, func(element InheritedElementSelf) bool {
		result := actionForIntent(element.Widget().(*actionsScope), intent, nil)
		if result != nil && actionIsEnabled(result, intent, bc) {
			returnValue = findDispatcher(element).InvokeAction(result, intent, bc)
		}
		return result != nil
	})
	return returnValue
}

// FocusableActionDetector is a widget that combines the functionality of Actions,
// Shortcuts, MouseRegion and a Focus widget to create a detector that defines
// actions and key bindings, and provides callbacks for handling focus and hover
// highlights.
//
// Flutter: widgets/actions.dart FocusableActionDetector
type FocusableActionDetector struct {
	StatefulWidget
	o     FocusableActionDetectorOpts
	child Widget
}

// FocusableActionDetectorOpts are the optional parameters of
// NewFocusableActionDetector.
//
// Flutter: widgets/actions.dart FocusableActionDetector.new
type FocusableActionDetectorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Disabled makes the widget unable to take the focus and ignore the actions and
	// shortcuts (Flutter's enabled is true by default).
	Disabled bool
	// FocusNode is the node of the Focus widget it makes.
	FocusNode *FocusNode
	// Autofocus asks for the focus when the widget is built.
	Autofocus bool
	// DescendantsAreNotFocusable makes the descendants unable to take the focus
	// (Flutter's descendantsAreFocusable is true by default).
	DescendantsAreNotFocusable bool
	// DescendantsAreNotTraversable makes traversal skip the descendants (Flutter's
	// descendantsAreTraversable is true by default).
	DescendantsAreNotTraversable bool
	// Shortcuts are the shortcuts that are active while the widget is enabled.
	Shortcuts ShortcutMap
	// Actions are the actions that are active while the widget is enabled.
	Actions ActionMap
	// OnShowFocusHighlight is called when the focus highlight should be shown or
	// hidden.
	OnShowFocusHighlight func(show bool)
	// OnShowHoverHighlight is called when the hover highlight should be shown or
	// hidden.
	OnShowHoverHighlight func(show bool)
	// OnFocusChange is called when the focus of the widget changes.
	OnFocusChange func(focused bool)
	// MouseCursor is the cursor over the widget; the default is MouseCursorDefer.
	MouseCursor services.MouseCursor
	// ExcludeFocusSemantics leaves out the Semantics node of the Focus widget
	// (Flutter's includeFocusSemantics is true by default).
	ExcludeFocusSemantics bool
}

// NewFocusableActionDetector creates the widget.
//
// Flutter: widgets/actions.dart FocusableActionDetector.new
func NewFocusableActionDetector(child Widget, opts ...FocusableActionDetectorOpts) *FocusableActionDetector {
	o := firstOpts(opts)
	if o.MouseCursor == nil {
		o.MouseCursor = services.MouseCursorDefer
	}
	return &FocusableActionDetector{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), o, child}
}

// Enabled is whether the widget handles the focus, actions and shortcuts.
//
// Flutter: widgets/actions.dart FocusableActionDetector.enabled
func (w *FocusableActionDetector) Enabled() bool { return !w.o.Disabled }

// FocusNode is the node of the widget, or nil.
//
// Flutter: widgets/actions.dart FocusableActionDetector.focusNode
func (w *FocusableActionDetector) FocusNode() *FocusNode { return w.o.FocusNode }

// Autofocus is whether the widget asks for the focus when it is built.
//
// Flutter: widgets/actions.dart FocusableActionDetector.autofocus
func (w *FocusableActionDetector) Autofocus() bool { return w.o.Autofocus }

// DescendantsAreFocusable is whether the descendants can take the focus.
//
// Flutter: widgets/actions.dart FocusableActionDetector.descendantsAreFocusable
func (w *FocusableActionDetector) DescendantsAreFocusable() bool {
	return !w.o.DescendantsAreNotFocusable
}

// DescendantsAreTraversable is whether traversal visits the descendants.
//
// Flutter: widgets/actions.dart FocusableActionDetector.descendantsAreTraversable
func (w *FocusableActionDetector) DescendantsAreTraversable() bool {
	return !w.o.DescendantsAreNotTraversable
}

// Child is the widget below.
//
// Flutter: widgets/actions.dart FocusableActionDetector.child
func (w *FocusableActionDetector) Child() Widget { return w.child }

// CreateState creates the state.
//
// Flutter: widgets/actions.dart FocusableActionDetector.createState
func (w *FocusableActionDetector) CreateState() State { return &focusableActionDetectorState{} }

type focusableActionDetectorState struct {
	StateOf[*FocusableActionDetector]
	canShowHighlight bool
	hovering         bool
	focused          bool
	highlightHandle  FocusHandlerHandle
	mouseRegionKey   *GlobalKey[State]
}

func (s *focusableActionDetectorState) InitState() {
	s.mouseRegionKey = NewGlobalKey[State]()
	binding := WidgetsBindingOf(s.Context())
	manager := binding.FocusManager()
	binding.Scheduler().AddPostFrameCallback(func(time_ time.Duration) {
		if !s.Mounted() {
			return
		}
		s.updateHighlightMode(manager.HighlightMode())
	})
	s.highlightHandle = manager.AddHighlightModeListener(s.handleFocusHighlightModeChange)
}

func (s *focusableActionDetectorState) Dispose() {
	s.Context().Owner().FocusManager().RemoveHighlightModeListener(s.highlightHandle)
}

func (s *focusableActionDetectorState) updateHighlightMode(FocusHighlightMode) {
	s.mayTriggerCallback(func() {
		s.canShowHighlight = s.Context().Owner().FocusManager().HighlightMode() == FocusHighlightModeTraditional
	}, nil)
}

func (s *focusableActionDetectorState) handleFocusHighlightModeChange(mode FocusHighlightMode) {
	if !s.Mounted() {
		return
	}
	s.updateHighlightMode(mode)
}

func (s *focusableActionDetectorState) handleMouseEnter(*gestures.PointerEnterEvent) {
	if !s.hovering {
		s.mayTriggerCallback(func() { s.hovering = true }, nil)
	}
}

func (s *focusableActionDetectorState) handleMouseExit(*gestures.PointerExitEvent) {
	if s.hovering {
		s.mayTriggerCallback(func() { s.hovering = false }, nil)
	}
}

func (s *focusableActionDetectorState) handleFocusChange(focused bool) {
	if s.focused != focused {
		s.mayTriggerCallback(func() { s.focused = focused }, nil)
		if cb := s.Widget().o.OnFocusChange; cb != nil {
			cb(s.focused)
		}
	}
}

func (s *focusableActionDetectorState) canRequestFocus(target *FocusableActionDetector) bool {
	if mode, ok := MediaQueryMaybeNavigationModeOf(s.Context()); ok && mode == NavigationModeDirectional {
		return true
	}
	return target.Enabled()
}

// mayTriggerCallback runs the task and calls the show callbacks whose value
// changed because of it. oldWidget is the widget before an update.
func (s *focusableActionDetectorState) mayTriggerCallback(task func(), oldWidget *FocusableActionDetector) {
	shouldShowHoverHighlight := func(target *FocusableActionDetector) bool {
		return s.hovering && target.Enabled() && s.canShowHighlight
	}
	shouldShowFocusHighlight := func(target *FocusableActionDetector) bool {
		return s.focused && s.canShowHighlight && s.canRequestFocus(target)
	}
	if foundation.DebugMode && WidgetsBindingOf(s.Context()).Scheduler().SchedulerPhase() == scheduler.SchedulerPhasePersistentCallbacks {
		panic(errors.New(tagFocusableActionDetectorCallbackDuringBuild, "the highlight callbacks of a FocusableActionDetector must not run during the build phase"))
	}
	oldTarget := oldWidget
	if oldTarget == nil {
		oldTarget = s.Widget()
	}
	didShowHoverHighlight := shouldShowHoverHighlight(oldTarget)
	didShowFocusHighlight := shouldShowFocusHighlight(oldTarget)
	if task != nil {
		task()
	}
	w := s.Widget()
	doShowHoverHighlight := shouldShowHoverHighlight(w)
	doShowFocusHighlight := shouldShowFocusHighlight(w)
	if didShowFocusHighlight != doShowFocusHighlight && w.o.OnShowFocusHighlight != nil {
		w.o.OnShowFocusHighlight(doShowFocusHighlight)
	}
	if didShowHoverHighlight != doShowHoverHighlight && w.o.OnShowHoverHighlight != nil {
		w.o.OnShowHoverHighlight(doShowHoverHighlight)
	}
}

func (s *focusableActionDetectorState) DidUpdateWidget(old *FocusableActionDetector) {
	if s.Widget().Enabled() != old.Enabled() {
		WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
			if !s.Mounted() {
				return
			}
			s.mayTriggerCallback(nil, old)
		})
	}
}

func (s *focusableActionDetectorState) Build(bc BuildContext) Widget {
	w := s.Widget()
	var child Widget = NewMouseRegion(
		NewFocus(w.child, FocusOpts{
			FocusNode: w.o.FocusNode, Autofocus: w.o.Autofocus,
			DescendantsAreFocusable:   foundation.Some(w.DescendantsAreFocusable()),
			DescendantsAreTraversable: foundation.Some(w.DescendantsAreTraversable()),
			CanRequestFocus:           foundation.Some(s.canRequestFocus(w)),
			OnFocusChange:             s.handleFocusChange,
			ExcludeSemantics:          w.o.ExcludeFocusSemantics,
		}),
		MouseRegionOpts{Key: s.mouseRegionKey, OnEnter: s.handleMouseEnter, OnExit: s.handleMouseExit, Cursor: w.o.MouseCursor})
	if w.Enabled() && len(w.o.Actions) > 0 {
		child = NewActions(w.o.Actions, child)
	}
	if w.Enabled() && len(w.o.Shortcuts) > 0 {
		child = NewShortcuts(w.o.Shortcuts, child)
	}
	return child
}

// VoidCallbackIntent is an intent that carries a function to call.
//
// Flutter: widgets/actions.dart VoidCallbackIntent
type VoidCallbackIntent struct {
	IntentBase
	// Callback is the function to call.
	//
	// Flutter: widgets/actions.dart VoidCallbackIntent.callback
	Callback func()
}

// VoidCallbackAction is the action that calls the function of a
// VoidCallbackIntent.
//
// Flutter: widgets/actions.dart VoidCallbackAction
type VoidCallbackAction struct{ ActionBase }

// NewVoidCallbackAction creates the action.
//
// Flutter: widgets/actions.dart VoidCallbackAction.new
func NewVoidCallbackAction() *VoidCallbackAction {
	a := &VoidCallbackAction{}
	InitActionBase(&a.ActionBase, a)
	return a
}

// IntentType is VoidCallbackIntent.
//
// Flutter: widgets/actions.dart Action.intentType
func (a *VoidCallbackAction) IntentType() reflect.Type { return reflect.TypeFor[VoidCallbackIntent]() }

// Invoke calls the callback of the intent.
//
// Flutter: widgets/actions.dart VoidCallbackAction.invoke
func (a *VoidCallbackAction) Invoke(intent Intent) any {
	intent.(VoidCallbackIntent).Callback()
	return nil
}

// DoNothingIntent is an intent that does nothing, to disable a shortcut: map the
// key to it and bind it to a DoNothingAction.
//
// Flutter: widgets/actions.dart DoNothingIntent
type DoNothingIntent struct{ IntentBase }

// DoNothingAndStopPropagationIntent is an intent that does nothing and stops the
// key event from going on: the key is not consumed, but no other handler gets
// it.
//
// Flutter: widgets/actions.dart DoNothingAndStopPropagationIntent
type DoNothingAndStopPropagationIntent struct{ IntentBase }

// DoNothingAction is an action that does nothing; it is bound to the
// DoNothingIntents.
//
// Flutter: widgets/actions.dart DoNothingAction
type DoNothingAction struct {
	ActionBase
	consumesKey bool
}

// DoNothingActionOpts are the optional parameters of NewDoNothingAction.
//
// Flutter: widgets/actions.dart DoNothingAction.new
type DoNothingActionOpts struct {
	// DoesNotConsumeKey makes the key event go on to the other handlers (Flutter's
	// consumesKey is true by default).
	DoesNotConsumeKey bool
}

// NewDoNothingAction creates the action.
//
// Flutter: widgets/actions.dart DoNothingAction.new
func NewDoNothingAction(opts ...DoNothingActionOpts) *DoNothingAction {
	a := &DoNothingAction{consumesKey: !firstOpts(opts).DoesNotConsumeKey}
	InitActionBase(&a.ActionBase, a)
	return a
}

// ConsumesKey is whether the action consumes the key.
//
// Flutter: widgets/actions.dart DoNothingAction.consumesKey
func (a *DoNothingAction) ConsumesKey(Intent) bool { return a.consumesKey }

// Invoke does nothing.
//
// Flutter: widgets/actions.dart DoNothingAction.invoke
func (a *DoNothingAction) Invoke(Intent) any { return nil }

// ActivateIntent is an intent that activates the focused control.
//
// Flutter: widgets/actions.dart ActivateIntent
type ActivateIntent struct{ IntentBase }

// ButtonActivateIntent is an intent that activates the focused button, on
// platforms where buttons are activated by Enter (and not by Space).
//
// Flutter: widgets/actions.dart ButtonActivateIntent
type ButtonActivateIntent struct{ IntentBase }

// ActivateAction is an action for ActivateIntent; embed it in the action of a
// control.
//
// Flutter: widgets/actions.dart ActivateAction
type ActivateAction struct{ ActionBase }

// IntentType is ActivateIntent.
//
// Flutter: widgets/actions.dart Action.intentType
func (a *ActivateAction) IntentType() reflect.Type { return reflect.TypeFor[ActivateIntent]() }

// SelectIntent is an intent that selects the focused control.
//
// Flutter: widgets/actions.dart SelectIntent
type SelectIntent struct{ IntentBase }

// SelectAction is an action for SelectIntent; embed it in the action of a
// control.
//
// Flutter: widgets/actions.dart SelectAction
type SelectAction struct{ ActionBase }

// IntentType is SelectIntent.
//
// Flutter: widgets/actions.dart Action.intentType
func (a *SelectAction) IntentType() reflect.Type { return reflect.TypeFor[SelectIntent]() }

// DismissIntent is an intent that dismisses the current route, dialog or popup
// (Escape by default).
//
// Flutter: widgets/actions.dart DismissIntent
type DismissIntent struct{ IntentBase }

// DismissAction is an action for DismissIntent; embed it in the action of a
// dismissable widget.
//
// Flutter: widgets/actions.dart DismissAction
type DismissAction struct{ ActionBase }

// IntentType is DismissIntent.
//
// Flutter: widgets/actions.dart Action.intentType
func (a *DismissAction) IntentType() reflect.Type { return reflect.TypeFor[DismissIntent]() }

// PrioritizedIntents is an intent that holds a list of intents, in priority
// order. The first of them whose action is enabled runs.
//
// Flutter: widgets/actions.dart PrioritizedIntents
type PrioritizedIntents struct {
	IntentBase
	// OrderedIntents are the intents, the one to try first first.
	//
	// Flutter: widgets/actions.dart PrioritizedIntents.orderedIntents
	OrderedIntents []Intent
}

// PrioritizedAction is the action for PrioritizedIntents: it finds the first
// intent that has an enabled action at the primary focus and invokes it.
//
// Flutter: widgets/actions.dart PrioritizedAction
type PrioritizedAction struct {
	ContextActionBase
	selectedAction Action
	selectedIntent Intent
}

// NewPrioritizedAction creates the action.
//
// Flutter: widgets/actions.dart PrioritizedAction.new
func NewPrioritizedAction() *PrioritizedAction {
	a := &PrioritizedAction{}
	InitContextActionBase(&a.ContextActionBase, a)
	return a
}

// IntentType is PrioritizedIntents.
//
// Flutter: widgets/actions.dart Action.intentType
func (a *PrioritizedAction) IntentType() reflect.Type { return reflect.TypeFor[PrioritizedIntents]() }

// IsEnabledWithContext finds the action to invoke: it is enabled when there is
// one.
//
// Flutter: widgets/actions.dart PrioritizedAction.isEnabled
func (a *PrioritizedAction) IsEnabledWithContext(intent Intent, bc BuildContext) bool {
	if bc == nil {
		return false
	}
	focus := PrimaryFocus(bc)
	if focus == nil || focus.Context() == nil {
		return false
	}
	for _, candidate := range intent.(PrioritizedIntents).OrderedIntents {
		candidateAction := ActionsMaybeFindIntent(focus.Context(), candidate)
		if candidateAction != nil && actionIsEnabled(candidateAction, candidate, bc) {
			a.selectedAction, a.selectedIntent = candidateAction, candidate
			return true
		}
	}
	return false
}

// InvokeWithContext invokes the action that IsEnabledWithContext found.
//
// Flutter: widgets/actions.dart PrioritizedAction.invoke
func (a *PrioritizedAction) InvokeWithContext(_ Intent, bc BuildContext) any {
	actionInvoke(a.selectedAction, a.selectedIntent, bc)
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagActionIntentMismatch                       = "ActionIntentMismatch"
	tagActionInvokeMissing                        = "ActionInvokeMissing"
	tagActionNotEnabled                           = "ActionNotEnabled"
	tagActionNotFound                             = "ActionNotFound"
	tagActionNotInitialized                       = "ActionNotInitialized"
	tagFocusableActionDetectorCallbackDuringBuild = "FocusableActionDetectorCallbackDuringBuild"
)
