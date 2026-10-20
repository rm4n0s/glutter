// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package foundation

import (
	"strconv"
	"sync/atomic"

	"github.com/rm4n0s/errors"
)

// TargetPlatform is the operating system the framework behaves like. Platform
// dependent framework behavior (scroll physics, shortcuts such as Ctrl or Cmd,
// menu conventions, typography) reads it instead of checking runtime.GOOS
// (SPEC §3).
//
// Flutter: foundation/platform.dart TargetPlatform
type TargetPlatform int

// The values of TargetPlatform follow Flutter's declaration order.
const (
	// TargetPlatformUnspecified is the zero value: the default target platform.
	//
	// Flutter: no counterpart
	TargetPlatformUnspecified TargetPlatform = iota
	// TargetPlatformAndroid is Android.
	//
	// Flutter: foundation/platform.dart TargetPlatform.android
	TargetPlatformAndroid
	// TargetPlatformFuchsia is Fuchsia.
	//
	// Flutter: foundation/platform.dart TargetPlatform.fuchsia
	TargetPlatformFuchsia
	// TargetPlatformIOS is iOS.
	//
	// Flutter: foundation/platform.dart TargetPlatform.iOS
	TargetPlatformIOS
	// TargetPlatformLinux is Linux.
	//
	// Flutter: foundation/platform.dart TargetPlatform.linux
	TargetPlatformLinux
	// TargetPlatformMacOS is macOS.
	//
	// Flutter: foundation/platform.dart TargetPlatform.macOS
	TargetPlatformMacOS
	// TargetPlatformWindows is Windows.
	//
	// Flutter: foundation/platform.dart TargetPlatform.windows
	TargetPlatformWindows
)

var targetPlatformNames = [...]string{"unspecified", "android", "fuchsia", "iOS", "linux", "macOS", "windows"}

// Name returns the name Dart's Enum.name gives the value, such as "linux".
func (p TargetPlatform) Name() string {
	if p >= 0 && int(p) < len(targetPlatformNames) {
		return targetPlatformNames[p]
	}
	return "TargetPlatform(" + strconv.Itoa(int(p)) + ")"
}

// String returns the name Dart prints for the value, such as
// "TargetPlatform.linux".
func (p TargetPlatform) String() string {
	if p >= 0 && int(p) < len(targetPlatformNames) {
		return "TargetPlatform." + targetPlatformNames[p]
	}
	return p.Name()
}

// hostPlatform is the platform the framework runs on. Linux is the only
// platform with a backend (SPEC §3); the engine supplies the value once
// another operating system is supported, so that foundation stays free of
// build tags.
const hostPlatform = TargetPlatformLinux

// debugPlatformOverride holds the override as a TargetPlatform; 0 is none.
var debugPlatformOverride atomic.Int32

// DefaultTargetPlatform is the platform the framework behaves like. Debug
// builds let a test change it with SetDebugDefaultTargetPlatformOverride.
//
// Flutter: foundation/platform.dart defaultTargetPlatform
func DefaultTargetPlatform() TargetPlatform {
	if DebugMode {
		if p := TargetPlatform(debugPlatformOverride.Load()); p != TargetPlatformUnspecified {
			return p
		}
	}
	return hostPlatform
}

// DebugDefaultTargetPlatformOverride returns the platform that replaces
// DefaultTargetPlatform, or None when there is no override.
//
// Flutter: foundation/platform.dart debugDefaultTargetPlatformOverride
func DebugDefaultTargetPlatformOverride() Opt[TargetPlatform] {
	if p := TargetPlatform(debugPlatformOverride.Load()); p != TargetPlatformUnspecified {
		return Some(p)
	}
	return None[TargetPlatform]()
}

// SetDebugDefaultTargetPlatformOverride makes DefaultTargetPlatform return p,
// or removes the override when p is None. Tests use it, and must remove the
// override when they are done. In a build that is not a debug build it panics
// with the tag TargetPlatformOverrideNotDebug.
//
// Flutter: foundation/platform.dart debugDefaultTargetPlatformOverride=
func SetDebugDefaultTargetPlatformOverride(p Opt[TargetPlatform]) {
	if !DebugMode {
		panic(errors.New("TargetPlatformOverrideNotDebug", "cannot modify the target platform override in non-debug builds"))
	}
	debugPlatformOverride.Store(int32(p.Value()))
}
