// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import "sync"

// Future is the value of an asynchronous operation of the framework's
// navigation: a route information parser's result, or the end of a router
// delegate's route setting. It stands for the Future of dart:async where the
// Router needs one; the callbacks run on the UI thread, through the post
// function that Then gets, so a goroutine can complete a Future without touching
// the framework. The zero value is not usable: make one with NewFuture,
// NewFutureValue or NewSynchronousFuture.
//
// Flutter: no counterpart (dart:async Future and Completer; SynchronousFuture
// from foundation/synchronous_future.dart)
type Future[T any] struct {
	mu          sync.Mutex
	done        bool
	synchronous bool
	value       T
	waiters     []futureWaiter[T]
}

type futureWaiter[T any] struct {
	post func(func())
	fn   func(T)
}

// NewFuture makes a future that is not complete yet; Complete completes it, once.
// It is Completer.future in Dart.
//
// Flutter: no counterpart (dart:async Completer)
func NewFuture[T any]() *Future[T] { return &Future[T]{} }

// NewFutureValue makes a future that is complete with the value already; its
// callbacks run in a later turn of the UI thread, as the callbacks of Future.value
// do in a microtask.
//
// Flutter: no counterpart (dart:async Future.value)
func NewFutureValue[T any](value T) *Future[T] { return &Future[T]{done: true, value: value} }

// NewSynchronousFuture makes a future that is complete with the value, and whose
// callbacks run at once, when Then is called.
//
// Flutter: foundation/synchronous_future.dart SynchronousFuture
func NewSynchronousFuture[T any](value T) *Future[T] {
	return &Future[T]{done: true, synchronous: true, value: value}
}

// Complete completes the future with the value and posts the callbacks that wait
// for it. Completing a future twice is ignored. It is safe to call from any
// goroutine.
//
// Flutter: no counterpart (dart:async Completer.complete)
func (f *Future[T]) Complete(value T) {
	f.mu.Lock()
	if f.done {
		f.mu.Unlock()
		return
	}
	f.done = true
	f.value = value
	waiters := f.waiters
	f.waiters = nil
	f.mu.Unlock()
	for _, w := range waiters {
		w.post(func() { w.fn(value) })
	}
}

// IsCompleted is whether the future has a value.
//
// Flutter: no counterpart (dart:async Completer.isCompleted)
func (f *Future[T]) IsCompleted() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.done
}

// Then calls fn with the value of the future: at once for a synchronous future,
// through post (a later turn of the UI thread) for a complete one, and when the
// future completes for a pending one. post is how the callback gets to the UI
// thread, typically WidgetsBinding.Post.
//
// Flutter: no counterpart (dart:async Future.then)
func (f *Future[T]) Then(post func(func()), fn func(T)) {
	f.mu.Lock()
	switch {
	case f.done && f.synchronous:
		value := f.value
		f.mu.Unlock()
		fn(value)
	case f.done:
		value := f.value
		f.mu.Unlock()
		post(func() { fn(value) })
	default:
		f.waiters = append(f.waiters, futureWaiter[T]{post, fn})
		f.mu.Unlock()
	}
}

// FutureMap returns a future of fn applied to the value of this one, with the same
// kind of completion: a synchronous future gives a synchronous one.
//
// Flutter: no counterpart (dart:async Future.then)
func FutureMap[T, U any](f *Future[T], post func(func()), fn func(T) U) *Future[U] {
	f.mu.Lock()
	done, synchronous, value := f.done, f.synchronous, f.value
	f.mu.Unlock()
	if done && synchronous {
		return NewSynchronousFuture(fn(value))
	}
	out := NewFuture[U]()
	f.Then(post, func(v T) { out.Complete(fn(v)) })
	return out
}
