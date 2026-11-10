package iokit

import (
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
)

// This file is the rumble of the pads that are read through their HID elements, played with the Force Feedback framework (the
// rules, the effect and the order of the calls are SDL 3's src/joystick/darwin/SDL_iokitjoystick.c DARWIN_JoystickRumble and
// CreateRumbleEffectData; this is a new Go version of them, and changed). The calls into the framework are ff_darwin.go; here is
// what is plain Go, so that it is tried with a fake framework on every machine.

// ForceFeedback is the rumble of one pad.
//
// Flutter: no counterpart
type ForceFeedback interface {
	// Play turns the motors at the strengths for d; a duration of 0 or no strength stops them. The two motors of the standard
	// layout are one actuator here, which turns at the mean of the two strengths (SDL's choice).
	Play(low, high uint16, d time.Duration) error
	// Close lets go of the effect and the device.
	Close()
}

// ForceFeedbacker is implemented by a Device that the system may give force feedback to.
//
// Flutter: no counterpart
type ForceFeedbacker interface {
	// HasForceFeedback says whether the device has force feedback (FFIsForceFeedback of its IOService).
	HasForceFeedback() bool
	// OpenForceFeedback makes the device's rumble: the device is created, reset and its actuators are turned on. The caller closes it.
	OpenForceFeedback() (ForceFeedback, error)
}

// The values of the framework (ForceFeedback.h and ForceFeedbackConstants.h).
const (
	ffOK                  = 0
	ffsffcReset           = 0x00000001
	ffsffcSetActuatorsOn  = 0x00000010
	ffeffObjectOffsets    = 0x00000002
	ffeffCartesian        = 0x00000010
	ffebNoTrigger         = 0xFFFFFFFF
	ffepDuration          = 0x00000001
	ffepTypeSpecific      = 0x00000100
	ffGain                = 10000
	ffSinePeriodMicros    = 1000000 // SDL's period: one second
	ffMaxMagnitude        = 10000   // the framework's top magnitude
	ffMaxDurationMicros   = 65535 * 1000
	ffSineEffectUUID      = "E559C463-C5CD-11D6-8A1C-00039353BD00" // kFFEffectType_Sine_ID
	ffMagnitudeOfFullTurn = 0x7FFF
)

// ffEffect is the framework's FFEFFECT (80 bytes on a 64-bit machine); ffPeriodic is its FFPERIODIC (16 bytes), which an effect of
// the sine type points to. The test of the layout checks the offsets that the C compiler gave (the probe is in the step's notes).
type ffEffect struct {
	Size, Flags, Duration, SamplePeriod, Gain, TriggerButton, TriggerRepeatInterval uint32
	Axes                                                                            uint32         // cAxes
	AxesList                                                                        unsafe.Pointer // rgdwAxes
	Directions                                                                      unsafe.Pointer // rglDirection
	Envelope                                                                        unsafe.Pointer // lpEnvelope
	TypeSpecificSize                                                                uint32         // cbTypeSpecificParams
	_                                                                               uint32
	TypeSpecific                                                                    unsafe.Pointer // lpvTypeSpecificParams
	StartDelay                                                                      uint32
	_                                                                               uint32
}

type ffPeriodic struct {
	Magnitude uint32
	Offset    int32
	Phase     uint32
	Period    uint32
}

// ffAPI is what the rumble asks of the framework: the calls of one device and its effect. The real one is ff_darwin.go's.
type ffAPI interface {
	// Command sends a command to the device (FFDeviceSendForceFeedbackCommand) and returns the HRESULT.
	Command(cmd uint32) int32
	// CreateEffect makes the sine effect (FFDeviceCreateEffect).
	CreateEffect(e *ffEffect) (handle uintptr, result int32)
	// SetParameters changes an effect (FFEffectSetParameters).
	SetParameters(handle uintptr, e *ffEffect, flags uint32) int32
	// Start plays the effect once (FFEffectStart), and Stop ends it (FFEffectStop).
	Start(handle uintptr) int32
	Stop(handle uintptr) int32
	// ReleaseEffect and ReleaseDevice let go.
	ReleaseEffect(handle uintptr)
	ReleaseDevice()
}

// ffMagnitude is the strength of the actuator for the two motors' strengths, in the framework's 0 to 10000: the mean of the two
// halves, as SDL does, scaled from the 15 bits of a signed 16.
func ffMagnitude(low, high uint16) uint32 {
	m := (uint32(low/2) + uint32(high/2)) / 2
	return m * ffMaxMagnitude / ffMagnitudeOfFullTurn
}

// ffDuration is a duration in microseconds, at least one and at most what the other platforms play (65.535 s).
func ffDuration(d time.Duration) uint32 {
	us := d.Microseconds()
	switch {
	case us < 1:
		return 1
	case us > ffMaxDurationMicros:
		return ffMaxDurationMicros
	}
	return uint32(us)
}

// rumbler is the ForceFeedback of a device over an ffAPI. Its calls come from the UI thread.
type rumbler struct {
	api     ffAPI
	effect  ffEffect
	sine    ffPeriodic
	axes    [2]uint32
	dirs    [2]int32
	handle  uintptr
	created bool
	closed  bool
}

// newRumbler resets the device and turns its actuators on, as SDL does before it makes the effect.
func newRumbler(api ffAPI) (*rumbler, error) {
	if hr := api.Command(ffsffcReset); hr != ffOK {
		api.ReleaseDevice()
		return nil, errors.New(tagForceFeedbackReset, "the force feedback device cannot be reset", "result", int(hr))
	}
	if hr := api.Command(ffsffcSetActuatorsOn); hr != ffOK {
		api.ReleaseDevice()
		return nil, errors.New(tagForceFeedbackActuators, "the actuators of the force feedback device cannot be turned on", "result", int(hr))
	}
	return &rumbler{api: api}, nil
}

// fill makes the effect for a strength and a duration: SDL's CreateRumbleEffectData, with a duration that the caller gives.
func (r *rumbler) fill(low, high uint16, d time.Duration) {
	r.sine = ffPeriodic{Magnitude: ffMagnitude(low, high), Period: ffSinePeriodMicros}
	r.effect = ffEffect{
		Size: uint32(unsafe.Sizeof(ffEffect{})), Flags: ffeffObjectOffsets | ffeffCartesian, Duration: ffDuration(d), Gain: ffGain,
		TriggerButton: ffebNoTrigger, Axes: 2, AxesList: unsafe.Pointer(&r.axes[0]), Directions: unsafe.Pointer(&r.dirs[0]),
		TypeSpecificSize: uint32(unsafe.Sizeof(ffPeriodic{})), TypeSpecific: unsafe.Pointer(&r.sine),
	}
}

// Play implements ForceFeedback.
func (r *rumbler) Play(low, high uint16, d time.Duration) error {
	if r.closed {
		return errors.New(tagForceFeedbackClosed, "the force feedback of the pad is closed")
	}
	if d <= 0 || ffMagnitude(low, high) == 0 {
		if !r.created {
			return nil
		}
		if hr := r.api.Stop(r.handle); hr != ffOK {
			return errors.New(tagForceFeedbackStop, "the rumble effect cannot be stopped", "result", int(hr))
		}
		return nil
	}
	r.fill(low, high, d)
	if !r.created {
		handle, hr := r.api.CreateEffect(&r.effect)
		if hr != ffOK {
			return errors.New(tagForceFeedbackEffect, "the rumble effect cannot be made", "result", int(hr))
		}
		r.handle, r.created = handle, true
	} else if hr := r.api.SetParameters(r.handle, &r.effect, ffepDuration|ffepTypeSpecific); hr != ffOK {
		return errors.New(tagForceFeedbackUpdate, "the rumble effect cannot be changed", "result", int(hr))
	}
	if hr := r.api.Start(r.handle); hr != ffOK {
		return errors.New(tagForceFeedbackStart, "the rumble effect cannot be played", "result", int(hr))
	}
	return nil
}

// Close implements ForceFeedback; closing twice does nothing.
func (r *rumbler) Close() {
	if r.closed {
		return
	}
	r.closed = true
	if r.created {
		r.api.ReleaseEffect(r.handle)
	}
	r.api.ReleaseDevice()
}

// The tags of the errors of the rumble. Each failing call has its own.
const (
	tagForceFeedbackReset     = "ForceFeedbackResetFailed"
	tagForceFeedbackActuators = "ForceFeedbackActuatorsFailed"
	tagForceFeedbackEffect    = "ForceFeedbackEffectFailed"
	tagForceFeedbackUpdate    = "ForceFeedbackUpdateFailed"
	tagForceFeedbackStart     = "ForceFeedbackStartFailed"
	tagForceFeedbackStop      = "ForceFeedbackStopFailed"
	tagForceFeedbackClosed    = "ForceFeedbackClosed"
	tagForceFeedbackDevice    = "ForceFeedbackDeviceFailed"
	tagForceFeedbackGone      = "ForceFeedbackDeviceGone"
)
