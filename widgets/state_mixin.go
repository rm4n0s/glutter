package widgets

import "reflect"

// stateMixin is implemented by the structs that a State embeds to get what a
// Dart mixin on State gives it (SingleTickerProvider, TickerProvider,
// AutomaticKeepAliveClient). The element finds the embedded structs when it
// creates the state and calls these hooks around the state's own hooks:
// AttachStateMixin when the state is created, ActivateStateMixin before the
// state's Activate, and DisposeStateMixin after the state's Dispose.
//
// Flutter: no counterpart (Dart mixins override initState, activate and dispose
// and call super)
type stateMixin interface {
	attachStateMixin(s State)
	activateStateMixin()
	disposeStateMixin()
}

// stateInitMixin is a stateMixin that also runs before the state's InitState.
type stateInitMixin interface{ initStateMixin() }

// stateDeactivateMixin is a stateMixin that also runs before the state's
// Deactivate.
type stateDeactivateMixin interface{ deactivateStateMixin() }

// collectStateMixins finds the stateMixin structs embedded in the state, at any
// depth, in field order. A mixin embedded under an unexported name can't be
// reached and is ignored.
func collectStateMixins(state State) []stateMixin {
	v := reflect.ValueOf(state)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return nil
	}
	var found []stateMixin
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		if v.Kind() != reflect.Struct {
			return
		}
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			sf := v.Type().Field(i)
			if !sf.Anonymous || !sf.IsExported() {
				continue
			}
			if f.Kind() == reflect.Pointer {
				if f.IsNil() {
					continue
				}
				f = f.Elem()
			}
			if f.Kind() != reflect.Struct {
				continue
			}
			if f.CanAddr() {
				if m, ok := f.Addr().Interface().(stateMixin); ok {
					found = append(found, m)
					continue
				}
			}
			walk(f)
		}
	}
	walk(v.Elem())
	return found
}
