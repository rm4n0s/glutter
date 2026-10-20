// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package animation

import (
	"github.com/rm4n0s/glutter/foundation"
)

// LazyListeners counts the listeners of an animation so that it can attach to
// its parents only while someone listens. Embed it, set the two functions and
// call DidRegisterListener and DidUnregisterListener from the listener methods.
//
// Flutter: animation/listener_helpers.dart AnimationLazyListenerMixin
type LazyListeners struct {
	counter int
	// OnStartListening is called when the first listener is added.
	OnStartListening func()
	// OnStopListening is called when the last listener is removed.
	OnStopListening func()
}

// DidRegisterListener counts a listener that was added.
//
// Flutter: animation/listener_helpers.dart AnimationLazyListenerMixin.didRegisterListener
func (l *LazyListeners) DidRegisterListener() {
	if l.counter == 0 && l.OnStartListening != nil {
		l.OnStartListening()
	}
	l.counter++
}

// DidUnregisterListener counts a listener that was removed.
//
// Flutter: animation/listener_helpers.dart AnimationLazyListenerMixin.didUnregisterListener
func (l *LazyListeners) DidUnregisterListener() {
	l.counter--
	if l.counter == 0 && l.OnStopListening != nil {
		l.OnStopListening()
	}
}

// IsListening reports whether there are listeners.
//
// Flutter: animation/listener_helpers.dart AnimationLazyListenerMixin.isListening
func (l *LazyListeners) IsListening() bool { return l.counter > 0 }

// LocalListeners keeps the listeners of an animation and calls them. Embed it
// in an animation: it has AddListener, RemoveListener and NotifyListeners. The
// listeners are called in the order they were added, and a listener that is
// removed during the notification is not called; an error that one of them
// panics with is reported (see SetErrorReporter) and the others are still
// called.
//
// Flutter: animation/listener_helpers.dart AnimationLocalListenersMixin
type LocalListeners struct {
	notifier *foundation.ChangeNotifier
	active   map[foundation.ListenerHandle]struct{}
	reporter foundation.ErrorReporter
	// OnRegister is called before a listener is added.
	OnRegister func()
	// OnUnregister is called after a listener that was there is removed.
	OnUnregister func()
}

func (l *LocalListeners) init() {
	if l.notifier == nil {
		l.notifier = foundation.NewChangeNotifier()
		l.notifier.SetErrorReporter(l.reporter)
		l.active = map[foundation.ListenerHandle]struct{}{}
	}
}

// SetErrorReporter sets where the errors of the listeners go; without one the
// first error is thrown again after all the listeners were called.
//
// Flutter: no counterpart (Flutter reports with FlutterError.reportError)
func (l *LocalListeners) SetErrorReporter(r foundation.ErrorReporter) {
	l.reporter = r
	if l.notifier != nil {
		l.notifier.SetErrorReporter(r)
	}
}

// AddListener calls the listener whenever the value changes.
//
// Flutter: animation/listener_helpers.dart AnimationLocalListenersMixin.addListener
func (l *LocalListeners) AddListener(listener foundation.VoidCallback) foundation.ListenerHandle {
	l.init()
	if l.OnRegister != nil {
		l.OnRegister()
	}
	h := l.notifier.AddListener(listener)
	l.active[h] = struct{}{}
	return h
}

// RemoveListener stops calling the listener; one that is not there is ignored.
//
// Flutter: animation/listener_helpers.dart AnimationLocalListenersMixin.removeListener
func (l *LocalListeners) RemoveListener(handle foundation.ListenerHandle) {
	if l.notifier == nil {
		return
	}
	if _, ok := l.active[handle]; !ok {
		return
	}
	delete(l.active, handle)
	l.notifier.RemoveListener(handle)
	if l.OnUnregister != nil {
		l.OnUnregister()
	}
}

// ClearListeners removes every listener, without calling OnUnregister.
//
// Flutter: animation/listener_helpers.dart AnimationLocalListenersMixin.clearListeners
func (l *LocalListeners) ClearListeners() {
	l.notifier = nil
	l.active = nil
	l.init()
}

// NotifyListeners calls the listeners.
//
// Flutter: animation/listener_helpers.dart AnimationLocalListenersMixin.notifyListeners
func (l *LocalListeners) NotifyListeners() {
	if l.notifier == nil {
		return
	}
	l.notifier.NotifyListeners()
}

// LocalStatusListeners keeps the status listeners of an animation and calls
// them, like LocalListeners.
//
// Flutter: animation/listener_helpers.dart AnimationLocalStatusListenersMixin
type LocalStatusListeners struct {
	entries  []statusEntry
	nextID   uint64
	reporter foundation.ErrorReporter
	// OnRegister is called before a listener is added.
	OnRegister func()
	// OnUnregister is called after a listener that was there is removed.
	OnUnregister func()
}

type statusEntry struct {
	id uint64
	fn AnimationStatusListener
}

// SetErrorReporter sets where the errors of the listeners go.
//
// Flutter: no counterpart (Flutter reports with FlutterError.reportError)
func (l *LocalStatusListeners) SetErrorReporter(r foundation.ErrorReporter) { l.reporter = r }

// AddStatusListener calls the listener whenever the status changes.
//
// Flutter: animation/listener_helpers.dart AnimationLocalStatusListenersMixin.addStatusListener
func (l *LocalStatusListeners) AddStatusListener(listener AnimationStatusListener) StatusListenerHandle {
	if l.OnRegister != nil {
		l.OnRegister()
	}
	l.nextID++
	l.entries = append(l.entries, statusEntry{l.nextID, listener})
	return StatusListenerHandle{l.nextID}
}

// RemoveStatusListener stops calling the listener.
//
// Flutter: animation/listener_helpers.dart AnimationLocalStatusListenersMixin.removeStatusListener
func (l *LocalStatusListeners) RemoveStatusListener(handle StatusListenerHandle) {
	for i, e := range l.entries {
		if e.id == handle.id && handle.id != 0 {
			l.entries = append(l.entries[:i:i], l.entries[i+1:]...)
			if l.OnUnregister != nil {
				l.OnUnregister()
			}
			return
		}
	}
}

// ClearStatusListeners removes every status listener, without calling
// OnUnregister.
//
// Flutter: animation/listener_helpers.dart AnimationLocalStatusListenersMixin.clearStatusListeners
func (l *LocalStatusListeners) ClearStatusListeners() { l.entries = nil }

// NotifyStatusListeners calls the status listeners with the status.
//
// Flutter: animation/listener_helpers.dart AnimationLocalStatusListenersMixin.notifyStatusListeners
func (l *LocalStatusListeners) NotifyStatusListeners(status AnimationStatus) {
	local := append([]statusEntry(nil), l.entries...)
	var first any
	for _, e := range local {
		if !l.has(e.id) {
			continue
		}
		func() {
			defer func() {
				r := recover()
				if r == nil {
					return
				}
				if l.reporter == nil {
					if first == nil {
						first = r
					}
					return
				}
				l.reporter(foundation.PanicError(foundation.CallbackPanicked, "while notifying status listeners", r))
			}()
			e.fn(status)
		}()
	}
	if first != nil {
		panic(first)
	}
}

func (l *LocalStatusListeners) has(id uint64) bool {
	for _, e := range l.entries {
		if e.id == id {
			return true
		}
	}
	return false
}
