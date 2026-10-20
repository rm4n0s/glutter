// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package services

// SystemSoundType is a sound that the system can play.
//
// Flutter: services/system_sound.dart SystemSoundType
type SystemSoundType int

// The values of SystemSoundType follow Flutter's declaration order.
const (
	// SystemSoundTypeUnspecified is the zero value.
	//
	// Flutter: no counterpart
	SystemSoundTypeUnspecified SystemSoundType = iota
	// SystemSoundTypeClick is the sound of a click.
	//
	// Flutter: services/system_sound.dart SystemSoundType.click
	SystemSoundTypeClick
	// SystemSoundTypeTick is the sound of a tick.
	//
	// Flutter: services/system_sound.dart SystemSoundType.tick
	SystemSoundTypeTick
	// SystemSoundTypeAlert is the sound of an alert.
	//
	// Flutter: services/system_sound.dart SystemSoundType.alert
	SystemSoundTypeAlert
)

// String is the name of the sound type, like Dart's enum toString.
//
// Flutter: services/system_sound.dart SystemSoundType
func (t SystemSoundType) String() string {
	switch t {
	case SystemSoundTypeClick:
		return "SystemSoundType.click"
	case SystemSoundTypeTick:
		return "SystemSoundType.tick"
	case SystemSoundTypeAlert:
		return "SystemSoundType.alert"
	}
	return "SystemSoundType.unspecified"
}

// SystemSound plays the sounds of the system. The desktop has no sound for them,
// so playing does nothing unless OnPlay is set (a test sets it to see the
// sounds). The binding has one.
//
// Deviation: Flutter sends the sound to the platform through a channel
// (SPEC §12.10).
//
// Flutter: services/system_sound.dart SystemSound
type SystemSound struct {
	// OnPlay is called with the type of each sound that is played.
	OnPlay func(SystemSoundType)
}

// Play plays the sound.
//
// Flutter: services/system_sound.dart SystemSound.play
func (s *SystemSound) Play(t SystemSoundType) {
	if s != nil && s.OnPlay != nil {
		s.OnPlay(t)
	}
}
