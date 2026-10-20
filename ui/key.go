// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package ui

import (
	"fmt"
	"strings"
	"time"
)

// KeyEventType is the type of a key event. The zero value,
// KeyEventTypeUnspecified, is never produced by a platform backend.
//
// Flutter: ui/key.dart KeyEventType
type KeyEventType int

// The values of KeyEventType follow Flutter's declaration order.
const (
	// KeyEventTypeUnspecified is the zero value.
	//
	// Flutter: no counterpart
	KeyEventTypeUnspecified KeyEventType = iota
	// KeyEventTypeDown means a key was pressed.
	//
	// Flutter: ui/key.dart KeyEventType.down
	KeyEventTypeDown
	// KeyEventTypeUp means a key was released.
	//
	// Flutter: ui/key.dart KeyEventType.up
	KeyEventTypeUp
	// KeyEventTypeRepeat means a key is held down and repeats.
	//
	// Flutter: ui/key.dart KeyEventType.repeat
	KeyEventTypeRepeat
)

var keyEventTypeLabels = [...]string{"Unspecified", "Key Down", "Key Up", "Key Repeat"}

// Label returns a human-readable name, such as "Key Down".
//
// Flutter: ui/key.dart KeyEventType.label
func (t KeyEventType) Label() string {
	return enumName(keyEventTypeLabels[:], int(t), "KeyEventType")
}

// String returns the same text as Label.
//
// Flutter: ui/key.dart KeyEventType
func (t KeyEventType) String() string { return t.Label() }

// KeyEventDeviceType is the type of device that produced a key event. The zero
// value, KeyEventDeviceTypeUnspecified, is never produced by a platform
// backend.
//
// Flutter: ui/key.dart KeyEventDeviceType
type KeyEventDeviceType int

// The values of KeyEventDeviceType follow Flutter's declaration order.
const (
	// KeyEventDeviceTypeUnspecified is the zero value.
	//
	// Flutter: no counterpart
	KeyEventDeviceTypeUnspecified KeyEventDeviceType = iota
	// KeyEventDeviceTypeKeyboard is a keyboard.
	//
	// Flutter: ui/key.dart KeyEventDeviceType.keyboard
	KeyEventDeviceTypeKeyboard
	// KeyEventDeviceTypeDirectionalPad is a directional pad.
	//
	// Flutter: ui/key.dart KeyEventDeviceType.directionalPad
	KeyEventDeviceTypeDirectionalPad
	// KeyEventDeviceTypeGamepad is a gamepad.
	//
	// Flutter: ui/key.dart KeyEventDeviceType.gamepad
	KeyEventDeviceTypeGamepad
	// KeyEventDeviceTypeJoystick is a joystick.
	//
	// Flutter: ui/key.dart KeyEventDeviceType.joystick
	KeyEventDeviceTypeJoystick
	// KeyEventDeviceTypeHDMI is an HDMI remote.
	//
	// Flutter: ui/key.dart KeyEventDeviceType.hdmi
	KeyEventDeviceTypeHDMI
)

var keyEventDeviceTypeLabels = [...]string{
	"Unspecified", "Keyboard", "Directional Pad", "Gamepad", "Joystick", "HDMI",
}

// Label returns a human-readable name, such as "Directional Pad".
//
// Flutter: ui/key.dart KeyEventDeviceType.label
func (t KeyEventDeviceType) Label() string {
	return enumName(keyEventDeviceTypeLabels[:], int(t), "KeyEventDeviceType")
}

// String returns the same text as Label.
//
// Flutter: ui/key.dart KeyEventDeviceType
func (t KeyEventDeviceType) String() string { return t.Label() }

// KeyData is information about a key event: which physical key, which logical
// key and which character it produced.
//
// Flutter: ui/key.dart KeyData
type KeyData struct {
	// TimeStamp is the time at which the event was generated, since an
	// arbitrary fixed point in the past.
	TimeStamp time.Duration
	// Type is the type of the event.
	Type KeyEventType
	// DeviceType is the type of the device that produced the event.
	DeviceType KeyEventDeviceType
	// Physical is the ID of the physical key (Flutter's PhysicalKeyboardKey),
	// which does not depend on the keyboard layout.
	Physical uint64
	// Logical is the ID of the logical key (Flutter's LogicalKeyboardKey),
	// which depends on the keyboard layout.
	Logical uint64
	// Character is the text the key produced. It is empty where Flutter has
	// null: for key releases and for keys that produce no text.
	Character string
	// Synthesized is whether the event was created by the engine to keep the
	// pressed-key state consistent, rather than by the user.
	Synthesized bool
}

// keyPlaneShift is the number of low bits of a key ID that hold the value; the
// bits above them are the plane (Unicode, GTK, ...).
const keyPlaneShift = 32

var keyPlaneNames = map[uint64]string{
	0x000: " (Unicode)",
	0x001: " (Unprintable)",
	0x002: " (Flutter)",
	0x011: " (Android)",
	0x012: " (Fuchsia)",
	0x013: " (iOS)",
	0x014: " (macOS)",
	0x015: " (GTK)",
	0x016: " (Windows)",
	0x017: " (Web)",
	0x018: " (GLFW)",
}

func (d KeyData) logicalString() string {
	plane := (d.Logical >> keyPlaneShift) & 0x0FF
	return fmt.Sprintf("0x%x%s", d.Logical, keyPlaneNames[plane])
}

func (d KeyData) escapedCharacter() string {
	if d.Character == "" {
		return "<none>"
	}
	switch d.Character {
	case "\n":
		return `"\n"`
	case "\t":
		return `"\t"`
	case "\r":
		return `"\r"`
	case "\b":
		return `"\b"`
	case "\f":
		return `"\f"`
	}
	return `"` + d.Character + `"`
}

// quotedCharCode lists the UTF-16 code units of the character, like Dart's
// String.codeUnits.
func (d KeyData) quotedCharCode() string {
	if d.Character == "" {
		return ""
	}
	var hex []string
	for _, r := range d.Character {
		if r >= 0x10000 {
			r -= 0x10000
			hex = append(hex, fmt.Sprintf("%04x", 0xD800+(r>>10)), fmt.Sprintf("%04x", 0xDC00+(r&0x3FF)))
			continue
		}
		hex = append(hex, fmt.Sprintf("%02x", r))
	}
	return " (0x" + strings.Join(hex, " ") + ")"
}

// String describes the event for logs and test failures. Like Flutter's
// toString, the text has an unmatched opening parenthesis.
//
// Flutter: ui/key.dart KeyData.toString
func (d KeyData) String() string {
	synthesized := ""
	if d.Synthesized {
		synthesized = ", synthesized"
	}
	return fmt.Sprintf("KeyData(%s, physical: 0x%x, logical: %s, character: %s%s%s",
		d.Type.Label(), d.Physical, d.logicalString(), d.escapedCharacter(), d.quotedCharCode(), synthesized)
}
