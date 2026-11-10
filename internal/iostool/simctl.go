package iostool

import (
	"bytes"
	"context"
	"encoding/json"
	stdErrors "errors"
	"os/exec"
	"strings"

	"github.com/rm4n0s/errors"
)

// Runner runs a program and returns what it wrote to the standard output and the standard error together, and its exit code. The
// error is for a program that cannot be run at all (IOSCommandFailed); a program that runs and exits with a code other than zero is
// the caller's to judge. The tests of the command lines of simctl give a fake one.
//
// Flutter: no counterpart
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (out []byte, exit int, err error)
}

// ExecRunner runs the programs of the machine.
//
// Flutter: no counterpart
type ExecRunner struct{}

// Run runs name with args.
//
// Flutter: no counterpart
func (ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, int, error) {
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return out.Bytes(), 0, nil
	case stdErrors.As(err, &exit):
		return out.Bytes(), exit.ExitCode(), nil
	}
	return out.Bytes(), -1, errors.NewErr(tagIOSCommandFailed, err).SetMetadata("program", name, "arguments", strings.Join(args, " "))
}

// Simctl runs `xcrun simctl` for a set of Simulator devices. The set is a directory of its own (`.ios-test/devices` for the tests),
// so that Glutter's devices are not the ones of the owner's Xcode.
//
// Flutter: no counterpart
type Simctl struct {
	// Set is the directory of the device set; the default set of the user when empty.
	Set string
	// Runner runs xcrun; ExecRunner when nil.
	Runner Runner
}

func (s Simctl) runner() Runner {
	if s.Runner != nil {
		return s.Runner
	}
	return ExecRunner{}
}

// Args is the command line (the arguments of xcrun) of a simctl subcommand: the set goes before the subcommand, as simctl wants it.
//
// Flutter: no counterpart
func (s Simctl) Args(sub string, args ...string) []string {
	out := []string{"simctl"}
	if s.Set != "" {
		out = append(out, "--set", s.Set)
	}
	out = append(out, sub)
	return append(out, args...)
}

// run runs a subcommand and fails with IOSSimctlFailed, with its output, when it exits with a code other than zero; a command that
// cannot be run at all is IOSCommandFailed.
func (s Simctl) run(ctx context.Context, sub string, args ...string) ([]byte, error) {
	out, exit, err := s.runner().Run(ctx, "xcrun", s.Args(sub, args...)...)
	if err != nil {
		return out, err
	}
	if exit != 0 {
		return out, errors.New(tagIOSSimctlFailed, "simctl says no", "subcommand", sub, "arguments", strings.Join(args, " "), "exit", exit, "output", string(out))
	}
	return out, nil
}

// Create makes a device of the type and the runtime (identifiers from `simctl list`) and returns its UDID.
//
// Flutter: no counterpart
func (s Simctl) Create(ctx context.Context, name, deviceType, runtime string) (string, error) {
	out, err := s.run(ctx, "create", name, deviceType, runtime)
	return strings.TrimSpace(string(out)), err
}

// Boot starts the device and waits until it has finished booting (`bootstatus -b`, which boots it when it is not booted).
//
// Flutter: no counterpart
func (s Simctl) Boot(ctx context.Context, udid string) error {
	_, err := s.run(ctx, "bootstatus", udid, "-b")
	return err
}

// Shutdown stops the device.
//
// Flutter: no counterpart
func (s Simctl) Shutdown(ctx context.Context, udid string) error {
	_, err := s.run(ctx, "shutdown", udid)
	return err
}

// Install installs the bundle on the device.
//
// Flutter: no counterpart
func (s Simctl) Install(ctx context.Context, udid, app string) error {
	_, err := s.run(ctx, "install", udid, app)
	return err
}

// LaunchArgs is the command line of Launch: with the console, the output of the app is the output of the command; the environment
// of the app is given by SIMCTL_CHILD_ variables, which the caller puts in the environment of the command.
//
// Flutter: no counterpart
func (s Simctl) LaunchArgs(udid, id string, console bool) []string {
	args := []string{}
	if console {
		args = append(args, "--console-pty")
	}
	return s.Args("launch", append(args, udid, id)...)
}

// Terminate stops the app.
//
// Flutter: no counterpart
func (s Simctl) Terminate(ctx context.Context, udid, id string) error {
	_, err := s.run(ctx, "terminate", udid, id)
	return err
}

// Screenshot writes the screen of the device to a file.
//
// Flutter: no counterpart
func (s Simctl) Screenshot(ctx context.Context, udid, file string) error {
	_, err := s.run(ctx, "io", udid, "screenshot", file)
	return err
}

// Pbpaste is the text of the device's pasteboard: what `simctl pbpaste` prints.
//
// Flutter: no counterpart
func (s Simctl) Pbpaste(ctx context.Context, udid string) (string, error) {
	out, err := s.run(ctx, "pbpaste", udid)
	return string(out), err
}

// Pbcopy makes text the content of the device's pasteboard (`simctl pbcopy`, which reads its standard input). It is for a test that
// copies something as another program would; it fails with IOSCommandFailed when xcrun cannot run and with IOSSimctlFailed when simctl
// says no.
//
// Flutter: no counterpart
func (s Simctl) Pbcopy(ctx context.Context, udid, text string) error {
	cmd := exec.CommandContext(ctx, "xcrun", s.Args("pbcopy", udid)...)
	cmd.Stdin = strings.NewReader(text)
	out, err := cmd.CombinedOutput()
	if err != nil {
		var exit *exec.ExitError
		if stdErrors.As(err, &exit) {
			return errors.New(tagIOSSimctlFailed, "simctl says no", "subcommand", "pbcopy", "exit", exit.ExitCode(), "output", string(out))
		}
		return errors.NewErr(tagIOSCommandFailed, err).SetMetadata("program", "xcrun", "arguments", "simctl pbcopy")
	}
	return nil
}

// UI changes a setting of the device's interface with `simctl ui`: "appearance" with light or dark, "content_size" with the name of a
// Dynamic Type size (extra-small, small, medium, large, extra-large, extra-extra-large, extra-extra-extra-large, accessibility-medium,
// accessibility-large, accessibility-extra-large, accessibility-extra-extra-large, accessibility-extra-extra-extra-large), or
// "increase_contrast" with enabled or disabled. It fails with IOSSimctlFailed when simctl says no.
//
// Flutter: no counterpart
func (s Simctl) UI(ctx context.Context, udid string, args ...string) error {
	_, err := s.run(ctx, "ui", append([]string{udid}, args...)...)
	return err
}

// Container is the directory of the data of the app (what the app sees as its home).
//
// Flutter: no counterpart
func (s Simctl) Container(ctx context.Context, udid, id string) (string, error) {
	out, err := s.run(ctx, "get_app_container", udid, id, "data")
	return strings.TrimSpace(string(out)), err
}

// Delete removes the device.
//
// Flutter: no counterpart
func (s Simctl) Delete(ctx context.Context, udid string) error {
	_, err := s.run(ctx, "delete", udid)
	return err
}

// Device is a device of the set, as `simctl list devices -j` says.
//
// Flutter: no counterpart
type Device struct {
	Name, UDID, State, Runtime string
}

// Devices lists the devices of the set that have a state (all of them, or only the booted ones).
//
// Flutter: no counterpart
func (s Simctl) Devices(ctx context.Context, booted bool) ([]Device, error) {
	args := []string{"devices", "-j"}
	if booted {
		args = []string{"devices", "booted", "-j"}
	}
	out, err := s.run(ctx, "list", args...)
	if err != nil {
		return nil, err
	}
	return ParseDevices(out)
}

// ParseDevices reads the JSON of `simctl list devices -j`. It fails with IOSSimctlFailed when the text is not that.
//
// Flutter: no counterpart
func ParseDevices(data []byte) ([]Device, error) {
	var doc struct {
		Devices map[string][]struct {
			Name  string `json:"name"`
			UDID  string `json:"udid"`
			State string `json:"state"`
		} `json:"devices"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, errors.NewErr(tagIOSSimctlFailed, err).SetMetadata("subcommand", "list devices", "output", string(data))
	}
	var out []Device
	for runtime, ds := range doc.Devices {
		for _, d := range ds {
			out = append(out, Device{Name: d.Name, UDID: d.UDID, State: d.State, Runtime: runtime})
		}
	}
	return out, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagIOSSimctlFailed  = TagIOSSimctlFailed
	tagIOSCommandFailed = TagIOSCommandFailed

	// TagIOSCommandFailed is the tag of the error for a program of Xcode (xcrun, xcode-select, codesign) that cannot be run at all.
	//
	// Flutter: no counterpart
	TagIOSCommandFailed = "IOSCommandFailed"

	// TagIOSSimctlFailed is the tag of the error for a `simctl` command that cannot be run or says no, or whose list it cannot read.
	//
	// Flutter: no counterpart
	TagIOSSimctlFailed = "IOSSimctlFailed"
)
