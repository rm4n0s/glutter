package services

import "github.com/rm4n0s/glutter/ui"

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

// SystemSoundBackend is what SystemSound asks of the engine. *ui.PlatformDispatcher is one; it
// forwards the call when the engine can play the sounds of the system (Android) and ignores it
// otherwise.
//
// Flutter: no counterpart (the flutter/platform channel of SystemChannels.platform)
type SystemSoundBackend interface {
	// PlaySystemSound plays the sound.
	PlaySystemSound(kind ui.SystemSoundKind)
}

// SystemSound plays the sounds of the system. The binding makes one
// (WidgetsBinding.SystemSound). The desktop has no sound for them, so playing does nothing
// there unless OnPlay is set (a test sets it to see the sounds). The zero value is usable and
// has no backend.
//
// Deviation: Flutter's SystemSound is a class of static methods that call the flutter/platform
// channel and return a Future<void> (SPEC §12.10).
//
// Flutter: services/system_sound.dart SystemSound
type SystemSound struct {
	// OnPlay is called with the type of each sound that is played.
	OnPlay func(SystemSoundType)

	backend SystemSoundBackend
}

// NewSystemSound creates a SystemSound that sends each sound to the backend, which may be nil.
//
// Flutter: services/system_sound.dart SystemSound
func NewSystemSound(backend SystemSoundBackend) *SystemSound {
	return &SystemSound{backend: backend}
}

// Play plays the sound.
//
// Flutter: services/system_sound.dart SystemSound.play
func (s *SystemSound) Play(t SystemSoundType) {
	if s == nil {
		return
	}
	if s.OnPlay != nil {
		s.OnPlay(t)
	}
	if s.backend == nil {
		return
	}
	switch t {
	case SystemSoundTypeClick:
		s.backend.PlaySystemSound(ui.SystemSoundClick)
	case SystemSoundTypeTick:
		s.backend.PlaySystemSound(ui.SystemSoundTick)
	case SystemSoundTypeAlert:
		s.backend.PlaySystemSound(ui.SystemSoundAlert)
	}
}
