package iostool

import (
	"context"
	"slices"
	"strings"

	"github.com/rm4n0s/errors"
)

// Simulator is a device set with the runtime that its devices run: the part of the harness of the tests and of `glutter-ios run` that
// makes the iPhone and the iPad of Glutter and boots them.
//
// Flutter: no counterpart
type Simulator struct {
	// Simctl is the device set.
	Simctl Simctl
	// Runtime is the runtime of the devices (ChooseRuntime).
	Runtime Runtime
}

// Discover reads the runtimes of the machine and picks the one to use. It fails with IOSRuntimeMissing, whose message says what to
// run, when there is none of iOS 15 or later.
//
// Flutter: no counterpart
func Discover(ctx context.Context, simctl Simctl) (Simulator, error) {
	out, err := simctl.run(ctx, "list", "runtimes", "-j")
	if err != nil {
		return Simulator{}, err
	}
	runtimes, err := ParseRuntimes(out)
	if err != nil {
		return Simulator{}, err
	}
	rt, err := ChooseRuntime(runtimes)
	if err != nil {
		return Simulator{}, err
	}
	return Simulator{Simctl: simctl, Runtime: rt}, nil
}

// Device returns the device of the family, "iphone" or "ipad": it makes it the first time (named glutter-iphone or glutter-ipad, of the
// model that ChooseDeviceType picks) and does not boot it. It fails with IOSSimctlFailed when the device cannot be made, and with
// IOSRuntimeMissing when the runtime has no model of the family.
//
// Flutter: no counterpart
func (s Simulator) Device(ctx context.Context, family string) (Device, error) {
	kind := map[string]string{"iphone": "iPhone", "ipad": "iPad"}[family]
	if kind == "" {
		return Device{}, errors.New(tagIOSSimctlFailed, "the family is iphone or ipad", "family", family)
	}
	name := "glutter-" + family
	devices, err := s.Simctl.Devices(ctx, false)
	if err != nil {
		return Device{}, err
	}
	// The runtime names of `simctl list` end with the same part as the identifier of the runtime ("iOS-27-2").
	end := s.Runtime.ID[strings.LastIndex(s.Runtime.ID, ".")+1:]
	for _, d := range devices {
		if d.Name == name && strings.HasSuffix(d.Runtime, end) {
			return d, nil
		}
	}
	dt, err := ChooseDeviceType(s.Runtime, kind)
	if err != nil {
		return Device{}, err
	}
	udid, err := s.Simctl.Create(ctx, name, dt.ID, s.Runtime.ID)
	if err != nil {
		return Device{}, err
	}
	return Device{Name: name, UDID: udid, Runtime: s.Runtime.ID, State: "Shutdown"}, nil
}

// Booted says whether the device is booted now.
//
// Flutter: no counterpart
func (s Simulator) Booted(ctx context.Context, udid string) (bool, error) {
	up, err := s.Simctl.Devices(ctx, true)
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(up, func(d Device) bool { return d.UDID == udid }), nil
}
