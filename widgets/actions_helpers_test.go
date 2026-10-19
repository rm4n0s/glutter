package widgets_test

import (
	"reflect"

	. "github.com/rm4n0s/glutter/widgets"
)

// The helpers of actions_test.dart and shortcuts_test.dart.

// testIntent is TestIntent: an interface, so that secondTestIntent and
// thirdTestIntent, which extend it in Dart, are accepted by an action made for it.
type testIntent interface {
	Intent
	isTestIntent()
}

type testIntentV struct{ IntentBase }

func (testIntentV) isTestIntent() {}

type secondTestIntent struct{ testIntentV }

type thirdTestIntent struct{ secondTestIntent }

// testAction is TestAction: a CallbackAction<TestIntent> that can be disabled and
// that counts its listeners.
type testAction struct {
	ActionBase
	onInvoke  func(intent Intent) any
	enabled   bool
	listeners map[ActionListenerHandle]bool
}

func newTestAction(onInvoke func(intent Intent) any) *testAction {
	a := &testAction{onInvoke: onInvoke, enabled: true, listeners: map[ActionListenerHandle]bool{}}
	InitActionBase(&a.ActionBase, a)
	return a
}

func (a *testAction) IntentType() reflect.Type { return reflect.TypeFor[testIntent]() }
func (a *testAction) IsEnabled(Intent) bool    { return a.enabled }
func (a *testAction) Invoke(intent Intent) any { return a.onInvoke(intent) }

func (a *testAction) setEnabled(value bool) {
	if a.enabled == value {
		return
	}
	a.enabled = value
	a.NotifyActionListeners()
}

func (a *testAction) AddActionListener(l ActionListenerCallback) ActionListenerHandle {
	h := a.ActionBase.AddActionListener(l)
	a.listeners[h] = true
	return h
}

func (a *testAction) RemoveActionListener(h ActionListenerHandle) {
	a.ActionBase.RemoveActionListener(h)
	delete(a.listeners, h)
}

// postInvokeCallback is PostInvokeCallback.
type postInvokeCallback func(action Action, intent Intent, dispatcher ActionDispatcher)

// testDispatcher is TestDispatcher: it tells postInvoke what it invoked.
type testDispatcher struct {
	ActionDispatcherBase
	postInvoke postInvokeCallback
}

func newTestDispatcher(postInvoke postInvokeCallback) *testDispatcher {
	d := &testDispatcher{postInvoke: postInvoke}
	InitActionDispatcherBase(&d.ActionDispatcherBase, d)
	return d
}

func (d *testDispatcher) InvokeAction(action Action, intent Intent, bc BuildContext) any {
	result := d.ActionDispatcherBase.InvokeAction(action, intent, bc)
	if d.postInvoke != nil {
		d.postInvoke(action, intent, d)
	}
	return result
}

// testDispatcher1 is TestDispatcher1: another type of dispatcher, to tell them
// apart.
type testDispatcher1 struct{ testDispatcher }

func newTestDispatcher1(postInvoke postInvokeCallback) *testDispatcher1 {
	d := &testDispatcher1{}
	d.postInvoke = postInvoke
	InitActionDispatcherBase(&d.ActionDispatcherBase, d)
	return d
}

func (d *testDispatcher1) InvokeAction(action Action, intent Intent, bc BuildContext) any {
	result := d.ActionDispatcherBase.InvokeAction(action, intent, bc)
	if d.postInvoke != nil {
		d.postInvoke(action, intent, d)
	}
	return result
}

// testContextAction is TestContextAction.
type testContextAction struct {
	ContextActionBase
	capturedContexts []BuildContext
}

func newTestContextAction() *testContextAction {
	a := &testContextAction{}
	InitContextActionBase(&a.ContextActionBase, a)
	return a
}

func (a *testContextAction) IntentType() reflect.Type { return reflect.TypeFor[testIntent]() }

func (a *testContextAction) InvokeWithContext(_ Intent, bc BuildContext) any {
	a.capturedContexts = append(a.capturedContexts, bc)
	return nil
}

// logIntent is LogIntent.
type logIntent struct {
	IntentBase
	log *[]string
}

func (i logIntent) add(s string) { *i.log = append(*i.log, s) }

// logInvocationAction is LogInvocationAction.
type logInvocationAction struct {
	ActionBase
	actionName      string
	enabled         bool
	shouldCallSuper bool
	// deferIsEnabled is LogInvocationButDeferIsEnabledAction.
	deferIsEnabled bool
	// newLog is RedirectOutputAction: the log to write to instead.
	newLog *[]string
}

type logOpts struct {
	disabled       bool
	dontCallSuper  bool
	deferIsEnabled bool
	newLog         *[]string
}

func newLogInvocationAction(name string, opts ...logOpts) *logInvocationAction {
	var o logOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	a := &logInvocationAction{actionName: name, enabled: !o.disabled, shouldCallSuper: !o.dontCallSuper,
		deferIsEnabled: o.deferIsEnabled, newLog: o.newLog}
	InitActionBase(&a.ActionBase, a)
	return a
}

func (a *logInvocationAction) IntentType() reflect.Type { return reflect.TypeFor[logIntent]() }

func (a *logInvocationAction) IsActionEnabled() bool {
	if a.deferIsEnabled {
		calling := a.CallingAction()
		return calling != nil && calling.IsActionEnabled()
	}
	return a.enabled
}

func (a *logInvocationAction) Invoke(i Intent) any {
	intent := i.(logIntent)
	if a.newLog != nil {
		intent = logIntent{log: a.newLog}
	}
	var calling Action
	if a.shouldCallSuper {
		calling = a.CallingAction()
	}
	if calling == nil {
		intent.add(a.actionName + ".invoke")
	} else {
		intent.add(a.actionName + ".invokeAsOverride-pre-super")
		calling.Invoke(intent)
		intent.add(a.actionName + ".invokeAsOverride-post-super")
	}
	return nil
}

// logInvocationContextAction is LogInvocationContextAction.
type logInvocationContextAction struct {
	ContextActionBase
	actionName    string
	enabled       bool
	invokeContext *BuildContext
}

func newLogInvocationContextAction(name string, invokeContext *BuildContext, disabled ...bool) *logInvocationContextAction {
	a := &logInvocationContextAction{actionName: name, enabled: len(disabled) == 0 || !disabled[0], invokeContext: invokeContext}
	InitContextActionBase(&a.ContextActionBase, a)
	return a
}

func (a *logInvocationContextAction) IntentType() reflect.Type { return reflect.TypeFor[logIntent]() }
func (a *logInvocationContextAction) IsActionEnabled() bool    { return a.enabled }

func (a *logInvocationContextAction) InvokeWithContext(i Intent, bc BuildContext) any {
	intent := i.(logIntent)
	*a.invokeContext = bc
	calling := a.CallingAction()
	if calling == nil {
		intent.add(a.actionName + ".invoke")
	} else {
		intent.add(a.actionName + ".invokeAsOverride-pre-super")
		actionInvokeFor(calling, intent, bc)
		intent.add(a.actionName + ".invokeAsOverride-post-super")
	}
	return nil
}

// actionInvokeFor invokes an action the way a calling action does: with the
// context when it is a context action.
func actionInvokeFor(a Action, intent Intent, bc BuildContext) any {
	if c, ok := a.(ContextAction); ok {
		return c.InvokeWithContext(intent, bc)
	}
	return a.Invoke(intent)
}

// defaultToKeyEventResultIntent and its action are DefaultToKeyEventResultIntent.
type defaultToKeyEventResultIntent struct{ IntentBase }

type defaultToKeyEventResultAction struct {
	ActionBase
	consumesKey bool
}

func newDefaultToKeyEventResultAction(consumesKey bool) *defaultToKeyEventResultAction {
	a := &defaultToKeyEventResultAction{consumesKey: consumesKey}
	InitActionBase(&a.ActionBase, a)
	return a
}

func (a *defaultToKeyEventResultAction) IntentType() reflect.Type {
	return reflect.TypeFor[defaultToKeyEventResultIntent]()
}
func (a *defaultToKeyEventResultAction) ConsumesKey(Intent) bool { return a.consumesKey }
func (a *defaultToKeyEventResultAction) Invoke(Intent) any       { return nil }
