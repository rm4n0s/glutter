// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package foundation

import (
	"reflect"
	"testing"
)

// Ported from packages/flutter/test/foundation/change_notifier_test.dart.

// testNotifier is Flutter's TestNotifier.
type testNotifier struct{ ChangeNotifier }

func (n *testNotifier) notify() { n.NotifyListeners() }

// counter is Flutter's Counter, a ChangeNotifier used as a mixin.
type counter struct {
	ChangeNotifier
	value int
}

func (c *counter) setValue(v int) {
	if c.value != v {
		c.value = v
		c.NotifyListeners()
	}
}

func TestChangeNotifierOrder(t *testing.T) {
	// The first test of the Flutter file, without badListener (a panic in a
	// listener is tested in error_report_test.go).
	n := &testNotifier{}
	var log []string
	logger := func(s string) VoidCallback { return func() { log = append(log, s) } }
	listener, listener1, listener2 := logger("listener"), logger("listener1"), logger("listener2")

	var h, h1, h2 []ListenerHandle
	steps := []struct {
		name string
		do   func()
		want []string
	}{
		{"add listener twice", func() { h = append(h, n.AddListener(listener), n.AddListener(listener)) }, []string{"listener", "listener"}},
		{"remove one", func() { n.RemoveListener(h[0]) }, []string{"listener"}},
		{"remove the other", func() { n.RemoveListener(h[1]) }, nil},
		{"remove a removed handle", func() { n.RemoveListener(h[1]) }, nil},
		{"add again", func() { h = append(h[:0], n.AddListener(listener)) }, []string{"listener"}},
		{"add listener1", func() { h1 = append(h1, n.AddListener(listener1)) }, []string{"listener", "listener1"}},
		{"add listener2", func() { h2 = append(h2, n.AddListener(listener2)) }, []string{"listener", "listener1", "listener2"}},
		{"remove listener1", func() { n.RemoveListener(h1[0]) }, []string{"listener", "listener2"}},
		{"add listener1 again", func() { h1 = append(h1[:0], n.AddListener(listener1)) }, []string{"listener", "listener2", "listener1"}},
		{"remove all and add some", func() {
			n.AddListener(listener1)
			n.RemoveListener(h[0])
			n.RemoveListener(h1[0])
			n.RemoveListener(h2[0])
			n.AddListener(listener2)
		}, []string{"listener1", "listener2"}},
	}
	for _, s := range steps {
		log = nil
		s.do()
		n.notify()
		if !reflect.DeepEqual(log, s.want) {
			t.Errorf("%s: log %v, want %v", s.name, log, s.want)
		}
	}
	n.Dispose()
}

func TestChangeNotifierZeroHandle(t *testing.T) {
	n := &testNotifier{}
	calls := 0
	n.AddListener(func() { calls++ })
	n.RemoveListener(ListenerHandle{})
	n.notify()
	if calls != 1 {
		t.Errorf("a zero handle removed a listener: %d calls", calls)
	}
}

func TestChangeNotifierMutatingListener(t *testing.T) {
	n := &testNotifier{}
	var log []string
	var h1, h3 ListenerHandle
	listener4 := func() { log = append(log, "listener4") }
	listener2 := func() {
		log = append(log, "listener2")
		n.RemoveListener(h1)
		n.RemoveListener(h3)
		n.AddListener(listener4)
	}
	h1 = n.AddListener(func() { log = append(log, "listener1") })
	n.AddListener(listener2)
	h3 = n.AddListener(func() { log = append(log, "listener3") })
	tests := [][]string{
		{"listener1", "listener2"},
		{"listener2", "listener4"},
		{"listener2", "listener4", "listener4"},
	}
	for i, want := range tests {
		log = nil
		n.notify()
		if !reflect.DeepEqual(log, want) {
			t.Errorf("notification %d: log %v, want %v", i+1, log, want)
		}
	}
}

func TestChangeNotifierAddedAndRemovedDuringNotify(t *testing.T) {
	n := &testNotifier{}
	var log []string
	n.AddListener(func() {
		log = append(log, "listener1")
		h := n.AddListener(func() { log = append(log, "listener2") })
		n.RemoveListener(h)
		n.AddListener(func() { log = append(log, "listener3") })
	})
	n.notify()
	if want := []string{"listener1"}; !reflect.DeepEqual(log, want) {
		t.Errorf("log %v, want %v", log, want)
	}
}

func TestChangeNotifierSelfRemoving(t *testing.T) {
	tests := []struct {
		name  string
		order []string // "self" removes itself, "1" logs listener1
		want  []string
	}{
		{"in the middle", []string{"1", "self", "1"}, []string{"listener1", "selfRemovingListener", "listener1"}},
		{"first", []string{"self", "1"}, []string{"selfRemovingListener", "listener1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &testNotifier{}
			var log []string
			var self ListenerHandle
			for _, o := range tt.order {
				if o == "1" {
					n.AddListener(func() { log = append(log, "listener1") })
				} else {
					self = n.AddListener(func() {
						log = append(log, "selfRemovingListener")
						n.RemoveListener(self)
					})
				}
			}
			n.notify()
			if !reflect.DeepEqual(log, tt.want) {
				t.Errorf("log %v, want %v", log, tt.want)
			}
		})
	}
}

func TestMergeListenables(t *testing.T) {
	s1, s2, s3 := &testNotifier{}, &testNotifier{}, &testNotifier{}
	var log []string
	merged := MergeListenables(s1, s2)
	l1 := func() { log = append(log, "listener1") }
	l2 := func() { log = append(log, "listener2") }
	all := func() { s1.notify(); s2.notify(); s3.notify() }

	h1 := merged.AddListener(l1)
	all()
	if want := []string{"listener1", "listener1"}; !reflect.DeepEqual(log, want) {
		t.Errorf("after add: %v, want %v", log, want)
	}
	log = nil
	merged.RemoveListener(h1)
	all()
	if len(log) != 0 {
		t.Errorf("after remove: %v", log)
	}
	merged.AddListener(l1)
	merged.AddListener(l2)
	all()
	if want := []string{"listener1", "listener2", "listener1", "listener2"}; !reflect.DeepEqual(log, want) {
		t.Errorf("after add both: %v, want %v", log, want)
	}
	// Removing a handle twice, or one that is not the merged object's, does
	// nothing.
	merged.RemoveListener(h1)
	merged.RemoveListener(ListenerHandle{})
}

func TestMergeListenablesIgnoresNil(t *testing.T) {
	s1, s2 := &testNotifier{}, &testNotifier{}
	var nilNotifier *testNotifier
	merged := MergeListenables(nil, s1, nilNotifier, s2, nil)
	calls := 0
	h := merged.AddListener(func() { calls++ })
	s1.notify()
	s2.notify()
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
	merged.RemoveListener(h)
	s1.notify()
	if calls != 2 {
		t.Errorf("a removed listener was called: %d", calls)
	}
}

func TestMergeListenablesString(t *testing.T) {
	s1, s2 := &testNotifier{}, &testNotifier{}
	tests := []struct {
		name string
		in   []Listenable
		want string
	}{
		{"empty", nil, "Listenable.merge([])"},
		{"nil", []Listenable{nil}, "Listenable.merge([null])"},
		{"one", []Listenable{s1}, "Listenable.merge([Instance of 'testNotifier'])"},
		{"two", []Listenable{s1, s2}, "Listenable.merge([Instance of 'testNotifier', Instance of 'testNotifier'])"},
		{"nil and one", []Listenable{nil, s2}, "Listenable.merge([null, Instance of 'testNotifier'])"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MergeListenables(tt.in...).(interface{ String() string }).String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMergeListenablesDoesNotLeak(t *testing.T) {
	s1, s2 := &testNotifier{}, &testNotifier{}
	merged := MergeListenables(s1, s2)
	if s1.HasListeners() || s2.HasListeners() {
		t.Fatal("listened to before adding")
	}
	h := merged.AddListener(func() {})
	if !s1.HasListeners() || !s2.HasListeners() {
		t.Error("the children are not listened to")
	}
	merged.RemoveListener(h)
	if s1.HasListeners() || s2.HasListeners() {
		t.Error("the children are still listened to after removing")
	}
}

func TestChangeNotifierRemoveAfterDispose(t *testing.T) {
	n := &testNotifier{}
	h := n.AddListener(func() {})
	n.Dispose()
	n.RemoveListener(h)
	n.RemoveListener(ListenerHandle{})
	if n.HasListeners() {
		t.Error("a disposed notifier has no listeners")
	}
}

func TestValueNotifier(t *testing.T) {
	tests := []struct {
		name string
		set  []float64
		want []float64
	}{
		{"change", []float64{3.0}, []float64{3.0}},
		{"same value", []float64{3.0, 3.0}, []float64{3.0}},
		{"each change", []float64{3.0, 4.0, 4.0, 2.0}, []float64{3.0, 4.0, 2.0}},
		{"initial value", []float64{2.0}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NewValueNotifier(2.0)
			var log []float64
			n.AddListener(func() { log = append(log, n.Value()) })
			for _, v := range tt.set {
				n.SetValue(v)
			}
			if !reflect.DeepEqual(log, tt.want) {
				t.Errorf("log %v, want %v", log, tt.want)
			}
		})
	}
}

func TestValueNotifierEquality(t *testing.T) {
	a, b := []int{1}, []int{1}
	n := NewValueNotifier(a)
	calls := 0
	n.AddListener(func() { calls++ })
	n.SetValue(a) // the same slice
	if calls != 0 {
		t.Errorf("setting the same slice notified %d times", calls)
	}
	n.SetValue(b) // another slice with equal elements: Dart lists compare by identity
	if calls != 1 {
		t.Errorf("setting another slice notified %d times, want 1", calls)
	}
	if got := ignoringHashCodes(NewValueNotifier(5).String()); got != "ValueNotifier<int>#00000(5)" {
		t.Errorf("String() = %q", got)
	}
	var p *int
	if got := NewValueNotifier(p).String(); got == "" {
		t.Error("String() of a nil pointer notifier is empty")
	}
}

func TestHasListeners(t *testing.T) {
	n := NewValueNotifier(true)
	var h1a, h1b, h2 ListenerHandle
	steps := []struct {
		name string
		do   func()
		want bool
	}{
		{"none", func() {}, false},
		{"add", func() { h1a = n.AddListener(func() {}) }, true},
		{"add the same function again", func() { h1b = n.AddListener(func() {}) }, true},
		{"remove one", func() { n.RemoveListener(h1a) }, true},
		{"remove the other", func() { n.RemoveListener(h1b) }, false},
		{"add", func() { h1a = n.AddListener(func() {}) }, true},
		{"add a second", func() { h2 = n.AddListener(func() {}) }, true},
		{"remove first", func() { n.RemoveListener(h1a) }, true},
		{"remove second", func() { n.RemoveListener(h2) }, false},
	}
	for _, s := range steps {
		s.do()
		if got := n.HasListeners(); got != s.want {
			t.Errorf("%s: HasListeners = %v, want %v", s.name, got, s.want)
		}
	}
}

func TestChangeNotifierAsEmbeddedField(t *testing.T) {
	c := &counter{}
	var log []string
	c.AddListener(func() {
		log = append(log, "listener1")
		if c.value < 0 {
			c.setValue(0)
		}
	})
	steps := []struct {
		name string
		do   func()
		want []string
	}{
		{"notify", c.NotifyListeners, []string{"listener1"}},
		{"set 3", func() { c.setValue(3) }, []string{"listener1"}},
		{"set -2 recurses", func() { c.setValue(-2) }, []string{"listener1", "listener1"}},
	}
	for _, s := range steps {
		log = nil
		s.do()
		if !reflect.DeepEqual(log, s.want) {
			t.Errorf("%s: log %v, want %v", s.name, log, s.want)
		}
	}
}

func TestRemoveListenersWhileNotifyingOnListThatWillNotResize(t *testing.T) {
	n := &testNotifier{}
	var log []string
	var handles []ListenerHandle
	var autoRemove ListenerHandle
	autoRemove = n.AddListener(func() {
		// We remove 4 listeners. We will end up with 13-4 = 9 listeners.
		n.RemoveListener(handles[1])
		n.RemoveListener(handles[3])
		n.RemoveListener(handles[4])
		n.RemoveListener(autoRemove)
	})
	// We add 12 more listeners.
	for i := 0; i < 12; i++ {
		name := "listener" + itoa(i)
		handles = append(handles, n.AddListener(func() { log = append(log, name) }))
	}
	remaining := []int{0, 2, 5, 6, 7, 8, 9, 10, 11}
	var want []string
	for _, i := range remaining {
		want = append(want, "listener"+itoa(i))
	}
	n.notify()
	if !reflect.DeepEqual(log, want) {
		t.Errorf("first notification: %v, want %v", log, want)
	}
	log = nil
	// We expect the same result after the removal of the previous listeners.
	n.notify()
	if !reflect.DeepEqual(log, want) {
		t.Errorf("second notification: %v, want %v", log, want)
	}
	// We remove all the other listeners.
	for _, i := range remaining {
		n.RemoveListener(handles[i])
	}
	log = nil
	n.notify()
	if len(log) != 0 {
		t.Errorf("after removing all: %v", log)
	}
}

func TestChangeNotifierPanicKeepsState(t *testing.T) {
	// A listener that panics must not leave the notifier thinking it is still
	// notifying (without a reporter the panic goes on).
	n := &testNotifier{}
	var h ListenerHandle
	h = n.AddListener(func() { n.RemoveListener(h); panic("bad") })
	func() {
		defer func() { _ = recover() }()
		n.notify()
	}()
	if n.notifyDepth != 0 || n.HasListeners() {
		t.Errorf("state after a panic: depth %d, listeners %v", n.notifyDepth, n.HasListeners())
	}
}
