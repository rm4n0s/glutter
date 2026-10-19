package widgets

import (
	"reflect"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// NewOverridableAction creates an action that does what the default action does
// unless an Actions widget above the lookup context has an action for the same
// intent: then that action, the override, does its work, and can call the
// default through its CallingAction. Widgets use it for the actions that an app
// can replace.
//
// Flutter: widgets/actions.dart Action.overridable
func NewOverridableAction(defaultAction Action, bc BuildContext) Action {
	if maker, ok := defaultAction.(interface{ makeOverridable(bc BuildContext) Action }); ok {
		return maker.makeOverridable(bc)
	}
	return newOverridableAction(defaultAction, bc)
}

func newOverridableAction(defaultAction Action, lookupContext BuildContext) *overridableAction {
	a := &overridableAction{defaultAction: defaultAction, lookupContext: lookupContext}
	_, a.contextVariant = defaultAction.(ContextAction)
	InitContextActionBase(&a.ContextActionBase, a)
	return a
}

// overridableAction is Flutter's _OverridableAction (for a plain default action)
// and _OverridableContextAction (for a ContextAction): both with
// _OverridableActionMixin.
//
// Flutter: widgets/actions.dart _OverridableActionMixin
type overridableAction struct {
	ContextActionBase
	defaultAction  Action
	lookupContext  BuildContext
	contextVariant bool

	invokeRecursive, isActionEnabledRecursive, isEnabledRecursive, consumesKeyRecursive bool
}

func (a *overridableAction) makeOverridable(bc BuildContext) Action {
	return newOverridableAction(a.defaultAction, bc)
}

// IntentType is the type of the default action.
func (a *overridableAction) IntentType() reflect.Type { return a.defaultAction.IntentType() }

func (a *overridableAction) updateCallingAction(value Action) {
	a.callingAction = value
	updateCallingAction(a.defaultAction, value)
}

func (a *overridableAction) getOverrideAction(intent Intent, declareDependency bool) Action {
	t := a.defaultAction.IntentType()
	var override Action
	if declareDependency {
		override = actionsMaybeFind(a.lookupContext, intent, t)
	} else {
		override = maybeFindWithoutDependingOn(a.lookupContext, intent, t)
	}
	if foundation.DebugMode && override == Action(a) {
		panic(errors.New("ActionOverrideSelf", "an overridable action found itself as its override"))
	}
	return override
}

func (a *overridableAction) guard(flag *bool, tag, what string) func() {
	if foundation.DebugMode {
		if *flag {
			panic(errors.New(tag, "the overridable action is mutually recursive in "+what+": an override that calls its calling action must not find the overridable action again"))
		}
		*flag = true
	}
	return func() {
		if foundation.DebugMode {
			*flag = false
		}
	}
}

func (a *overridableAction) invokeDefaultAction(intent Intent, bc BuildContext) any {
	if a.contextVariant {
		return actionInvoke(a.defaultAction, intent, bc)
	}
	return a.defaultAction.Invoke(intent)
}

func (a *overridableAction) invokeOverride(overrideAction Action, intent Intent, bc BuildContext) any {
	defer a.guard(&a.invokeRecursive, "ActionOverrideRecursion", "Invoke")()
	calling := a.defaultAction
	if a.contextVariant {
		if bc == nil {
			panic(errors.New("ActionContextMissing", "an overridable context action was invoked without a context"))
		}
		debugCanHandleIntent(overrideAction, intent)
		calling = &contextActionToActionAdapter{invokeContext: bc, action: a.defaultAction.(ContextAction)}
		InitActionBase(&calling.(*contextActionToActionAdapter).ActionBase, calling)
	}
	updateCallingAction(overrideAction, calling)
	returnValue := actionInvoke(overrideAction, intent, bc)
	updateCallingAction(overrideAction, nil)
	return returnValue
}

// InvokeWithContext invokes the override, or the default action.
func (a *overridableAction) InvokeWithContext(intent Intent, bc BuildContext) any {
	override := a.getOverrideAction(intent, false)
	if override == nil {
		return a.invokeDefaultAction(intent, bc)
	}
	return a.invokeOverride(override, intent, bc)
}

func (a *overridableAction) isOverrideActionEnabled(overrideAction Action) bool {
	defer a.guard(&a.isActionEnabledRecursive, "ActionOverrideRecursion", "IsActionEnabled")()
	updateCallingAction(overrideAction, a.defaultAction)
	enabled := overrideAction.IsActionEnabled()
	updateCallingAction(overrideAction, nil)
	return enabled
}

// IsActionEnabled is the override's, or the default action's.
func (a *overridableAction) IsActionEnabled() bool {
	if override := a.getOverrideAction(nil, true); override != nil {
		return a.isOverrideActionEnabled(override)
	}
	return a.defaultAction.IsActionEnabled()
}

// IsEnabledWithContext asks the override, or the default action.
func (a *overridableAction) IsEnabledWithContext(intent Intent, bc BuildContext) bool {
	defer a.guard(&a.isEnabledRecursive, "ActionOverrideRecursion", "IsEnabled")()
	override := a.getOverrideAction(intent, false)
	if override != nil {
		debugCanHandleIntent(override, intent)
		updateCallingAction(override, a.defaultAction)
	}
	target := a.defaultAction
	if override != nil {
		target = override
	}
	enabled := actionIsEnabled(target, intent, bc)
	if override != nil {
		updateCallingAction(override, nil)
	}
	return enabled
}

// ConsumesKey asks the override, or the default action.
func (a *overridableAction) ConsumesKey(intent Intent) bool {
	defer a.guard(&a.consumesKeyRecursive, "ActionOverrideRecursion", "ConsumesKey")()
	override := a.getOverrideAction(intent, false)
	if override != nil {
		updateCallingAction(override, a.defaultAction)
	}
	target := a.defaultAction
	if override != nil {
		target = override
	}
	consumes := target.ConsumesKey(intent)
	if override != nil {
		updateCallingAction(override, nil)
	}
	return consumes
}

// DebugFillProperties adds the default action.
func (a *overridableAction) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("defaultAction", foundation.DescribeIdentity(a.defaultAction)))
}

// contextActionToActionAdapter is Flutter's _ContextActionToActionAdapter: a
// ContextAction seen as a plain Action that invokes it with a fixed context.
//
// Flutter: widgets/actions.dart _ContextActionToActionAdapter
type contextActionToActionAdapter struct {
	ActionBase
	invokeContext BuildContext
	action        ContextAction
	inner         map[ActionListenerHandle]ActionListenerHandle
}

func (a *contextActionToActionAdapter) IntentType() reflect.Type { return a.action.IntentType() }

func (a *contextActionToActionAdapter) updateCallingAction(value Action) {
	updateCallingAction(a.action, value)
}

func (a *contextActionToActionAdapter) CallingAction() Action { return a.action.CallingAction() }

func (a *contextActionToActionAdapter) IsEnabled(intent Intent) bool {
	return a.action.IsEnabledWithContext(intent, a.invokeContext)
}

func (a *contextActionToActionAdapter) IsActionEnabled() bool { return a.action.IsActionEnabled() }

func (a *contextActionToActionAdapter) ConsumesKey(intent Intent) bool {
	return a.action.ConsumesKey(intent)
}

func (a *contextActionToActionAdapter) AddActionListener(listener ActionListenerCallback) ActionListenerHandle {
	handle := a.ActionBase.AddActionListener(listener)
	if a.inner == nil {
		a.inner = map[ActionListenerHandle]ActionListenerHandle{}
	}
	a.inner[handle] = a.action.AddActionListener(listener)
	return handle
}

func (a *contextActionToActionAdapter) RemoveActionListener(handle ActionListenerHandle) {
	a.ActionBase.RemoveActionListener(handle)
	if inner, ok := a.inner[handle]; ok {
		a.action.RemoveActionListener(inner)
		delete(a.inner, handle)
	}
}

func (a *contextActionToActionAdapter) Invoke(intent Intent) any {
	return a.action.InvokeWithContext(intent, a.invokeContext)
}
