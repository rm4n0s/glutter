package iostool

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

// Toolchain is where Xcode's pieces are on this machine.
//
// Flutter: no counterpart
type Toolchain struct {
	// Developer is the developer directory (`xcode-select -p`), inside Xcode.app.
	Developer string
	// Clang is the C compiler of Xcode, which Go's external linker and cgo use.
	Clang string
	// DeviceSDK and SimulatorSDK are the paths of the iPhoneOS and the iPhoneSimulator SDKs.
	DeviceSDK, SimulatorSDK string
}

// CheckXcode looks for the full Xcode: `xcode-select -p` must point into Xcode.app (the Command Line Tools have no iOS SDK) and
// `xcrun` must find a compiler and the two SDKs. It fails with IOSXcodeMissing, whose message says what to run, when one is missing.
//
// Flutter: no counterpart
func CheckXcode(ctx context.Context, r Runner) (Toolchain, error) {
	if r == nil {
		r = ExecRunner{}
	}
	var t Toolchain
	out, exit, err := r.Run(ctx, "xcode-select", "-p")
	if err != nil {
		return t, errors.New(tagIOSXcodeMissing, "iOS needs the full Xcode: run `xcode-select -s /Applications/Xcode.app`", "reason", err.Error())
	}
	t.Developer = strings.TrimSpace(string(out))
	if exit != 0 || !strings.Contains(t.Developer, "Xcode") || !strings.Contains(t.Developer, ".app") {
		return t, errors.New(tagIOSXcodeMissing, "iOS needs the full Xcode: run `xcode-select -s /Applications/Xcode.app`", "developer", t.Developer)
	}
	find := func(args ...string) (string, error) {
		out, exit, err := r.Run(ctx, "xcrun", args...)
		if err != nil {
			return "", errors.New(tagIOSXcodeMissing, "xcrun cannot be run", "reason", err.Error())
		}
		if exit != 0 {
			return "", errors.New(tagIOSXcodeMissing, "xcrun does not find what the build needs: run `xcode-select -s /Applications/Xcode.app`",
				"command", "xcrun "+strings.Join(args, " "), "exit", exit, "output", string(out))
		}
		return strings.TrimSpace(string(out)), nil
	}
	if t.Clang, err = find("--sdk", "iphoneos", "--find", "clang"); err != nil {
		return t, err
	}
	if t.DeviceSDK, err = find("--sdk", "iphoneos", "--show-sdk-path"); err != nil {
		return t, err
	}
	if t.SimulatorSDK, err = find("--sdk", "iphonesimulator", "--show-sdk-path"); err != nil {
		return t, err
	}
	return t, nil
}

// Runtime is an iOS Simulator runtime that is installed.
//
// Flutter: no counterpart
type Runtime struct {
	// ID is the identifier that `simctl create` takes ("com.apple.CoreSimulator.SimRuntime.iOS-27-2").
	ID string
	// Version is "27.2".
	Version string
	// DeviceTypes are the devices that the runtime can run.
	DeviceTypes []DeviceType
}

// DeviceType is a model of device that the Simulator can make.
//
// Flutter: no counterpart
type DeviceType struct {
	// ID is the identifier that `simctl create` takes.
	ID string
	// Name is "iPhone 17".
	Name string
	// Family is "iPhone" or "iPad".
	Family string
}

// ParseRuntimes reads the JSON of `simctl list runtimes -j` and returns the iOS runtimes that are available. It fails with
// IOSSimctlFailed when the text is not that.
//
// Flutter: no counterpart
func ParseRuntimes(data []byte) ([]Runtime, error) {
	var doc struct {
		Runtimes []struct {
			ID          string `json:"identifier"`
			Version     string `json:"version"`
			Platform    string `json:"platform"`
			IsAvailable bool   `json:"isAvailable"`
			Types       []struct {
				ID     string `json:"identifier"`
				Name   string `json:"name"`
				Family string `json:"productFamily"`
			} `json:"supportedDeviceTypes"`
		} `json:"runtimes"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, errors.NewErr(tagIOSSimctlFailed, err).SetMetadata("subcommand", "list runtimes", "output", string(data))
	}
	var out []Runtime
	for _, r := range doc.Runtimes {
		if r.Platform != "iOS" || !r.IsAvailable {
			continue
		}
		rt := Runtime{ID: r.ID, Version: r.Version}
		for _, t := range r.Types {
			rt.DeviceTypes = append(rt.DeviceTypes, DeviceType{ID: t.ID, Name: t.Name, Family: t.Family})
		}
		out = append(out, rt)
	}
	return out, nil
}

// versionLess compares two dotted versions ("27.2" and "15.4").
func versionLess(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < max(len(pa), len(pb)); i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i]) // a part that is not a number counts as zero
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			return x < y
		}
	}
	return false
}

// ChooseRuntime picks the runtime to test on: the newest iOS of at least 15 (D36), and among runtimes of one version the one that
// runs the most models (Xcode has an extra runtime for a single model). It fails with IOSRuntimeMissing, whose message says what to
// run, when there is none.
//
// Flutter: no counterpart
func ChooseRuntime(rs []Runtime) (Runtime, error) {
	var best Runtime
	found := false
	for _, r := range rs {
		if versionLess(r.Version, MinOSVersion) {
			continue
		}
		if !found || versionLess(best.Version, r.Version) || (r.Version == best.Version && len(r.DeviceTypes) > len(best.DeviceTypes)) {
			best, found = r, true
		}
	}
	if !found {
		return best, errors.New(tagIOSRuntimeMissing, "no iOS Simulator runtime of iOS 15 or later is installed: run `xcodebuild -downloadPlatform iOS`")
	}
	return best, nil
}

// ChooseDeviceType picks the model of the family ("iPhone" or "iPad") to test on: the current plain phone and the current plain
// iPad when the runtime has them, and the first of the family otherwise. It fails with IOSRuntimeMissing when the runtime has no
// model of the family.
//
// Flutter: no counterpart
func ChooseDeviceType(r Runtime, family string) (DeviceType, error) {
	prefer := map[string]string{"iPhone": "iPhone 17", "iPad": "iPad (A16)"}[family]
	var first DeviceType
	found := false
	for _, t := range r.DeviceTypes {
		if t.Family != family {
			continue
		}
		if t.Name == prefer {
			return t, nil
		}
		if !found {
			first, found = t, true
		}
	}
	if !found {
		return first, errors.New(tagIOSRuntimeMissing, "the runtime has no model of the family", "family", family, "runtime", r.ID)
	}
	return first, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagIOSXcodeMissing   = TagIOSXcodeMissing
	tagIOSRuntimeMissing = TagIOSRuntimeMissing

	// TagIOSXcodeMissing is the tag of the error for a machine without the full Xcode selected, or without its compiler or SDKs.
	//
	// Flutter: no counterpart
	TagIOSXcodeMissing = "IOSXcodeMissing"
	// TagIOSRuntimeMissing is the tag of the error for a machine without an iOS Simulator runtime of iOS 15 or later, or whose runtime
	// has no model of the family that was asked for.
	//
	// Flutter: no counterpart
	TagIOSRuntimeMissing = "IOSRuntimeMissing"
)
