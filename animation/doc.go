// Package animation provides AnimationController, Tween, Curves and the other
// animation primitives.
//
// An Animation[T] is a value that changes over time, with a status
// (dismissed, forward, reverse, completed) and listeners. An
// AnimationController runs one from 0 to 1 (or between other bounds) on a
// ticker from a scheduler.TickerProvider. A Tween, a CurveTween or a
// TweenSequence maps the 0 to 1 value to another type, and a Curve shapes the
// progress. CurvedAnimation, ReverseAnimation, ProxyAnimation,
// TrainHoppingAnimation and the compound animations (AnimationMean,
// AnimationMax, AnimationMin) build one animation from others.
//
// Listeners are handles: AddListener returns a foundation.ListenerHandle (and
// AddStatusListener a StatusListenerHandle) that RemoveListener takes, because
// Go functions can't be compared. A panic in a listener is reported to the
// error reporter of the controller (see Environment) and the other listeners
// still run.
//
// The painting tweens (EdgeInsets, Decoration, TextStyle and so on) are in
// package widgets, with the implicit animations.
//
// Flutter: package:flutter/animation.dart.
package animation
