package androidtool

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
)

// Device is a device or emulator that adb lists.
//
// Flutter: no counterpart
type Device struct {
	// Serial is what adb calls it, such as "emulator-5554" or "R58M123ABC".
	Serial string
	// State is adb's state: "device" when it is usable, else "offline" or "unauthorized".
	State string
}

// ADB runs adb for one device (or for the server, with no serial).
//
// Flutter: no counterpart
type ADB struct {
	// Path is the adb program ("adb" when empty).
	Path string
	// Serial is the device every command is for; empty for commands of the server.
	Serial string
	// Env is the environment of adb, for a private server (ANDROID_ADB_SERVER_PORT); the
	// current one when nil.
	Env []string
	// Run runs the commands.
	Run Runner
}

func (a ADB) command(args ...string) Command {
	path := a.Path
	if path == "" {
		path = "adb"
	}
	if a.Serial != "" {
		args = append([]string{"-s", a.Serial}, args...)
	}
	return Command{Env: a.Env, Name: path, Args: args}
}

// Output runs `adb [-s serial] args...` and returns what it prints.
func (a ADB) Output(ctx context.Context, args ...string) (string, error) {
	return a.Run.Run(ctx, a.command(args...))
}

// Devices lists the devices adb knows, sorted by serial.
func (a ADB) Devices(ctx context.Context) ([]Device, error) {
	out, err := a.Output(ctx, "devices")
	if err != nil {
		return nil, err
	}
	return ParseDevices(out), nil
}

// ParseDevices reads the output of `adb devices`.
//
// Flutter: no counterpart
func ParseDevices(out string) []Device {
	var devs []Device
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) < 2 || f[0] == "List" || strings.HasPrefix(f[0], "*") {
			continue
		}
		devs = append(devs, Device{Serial: f[0], State: f[1]})
	}
	sort.Slice(devs, func(i, j int) bool { return devs[i].Serial < devs[j].Serial })
	return devs
}

// PickDevice chooses the device a command is for: the one named by want, else the only one
// that is connected. It fails with AndroidNoDevice, AndroidDeviceNotFound or
// AndroidManyDevices, which lists the serials.
//
// Flutter: no counterpart
func PickDevice(devs []Device, want string) (Device, error) {
	if want != "" {
		for _, d := range devs {
			if d.Serial == want {
				if d.State != "device" {
					return Device{}, errors.New(TagAndroidDeviceNotFound, "the device is "+d.State+", not ready", "device", want, "state", d.State)
				}
				return d, nil
			}
		}
		return Device{}, errors.New(TagAndroidDeviceNotFound, "the device is not connected", "device", want, "devices", describeDevices(devs))
	}
	var ready []Device
	for _, d := range devs {
		if d.State == "device" {
			ready = append(ready, d)
		}
	}
	switch len(ready) {
	case 0:
		return Device{}, errors.New(TagAndroidNoDevice, "no device is connected: connect a phone with USB debugging, or start an emulator", "devices", describeDevices(devs))
	case 1:
		return ready[0], nil
	}
	return Device{}, errors.New(TagAndroidManyDevices, "more than one device is connected: pick one with -device", "devices", describeDevices(ready))
}

func describeDevices(devs []Device) string {
	var parts []string
	for _, d := range devs {
		parts = append(parts, d.Serial+" ("+d.State+")")
	}
	return strings.Join(parts, ", ")
}

// ABI is the ABI of the device that adb is for, chosen among the ones glutter-android builds.
// It fails with AndroidABIUnsupported for a device that has none of them (32-bit ARM or x86).
func (a ADB) ABI(ctx context.Context) (ABI, error) {
	out, err := a.Output(ctx, "shell", "getprop", "ro.product.cpu.abilist")
	if err != nil {
		return ABI{}, err
	}
	list := strings.Split(strings.TrimSpace(out), ",")
	for _, name := range list {
		for _, abi := range ABIs {
			if abi.Name == name {
				return abi, nil
			}
		}
	}
	return ABI{}, errors.New(TagAndroidABIUnsupported, "the device runs none of the supported ABIs (arm64-v8a, x86_64)", "abis", strings.TrimSpace(out))
}

// Install installs the APK over any earlier version (`adb install -r -t`).
func (a ADB) Install(ctx context.Context, apk string) error {
	_, err := a.Output(ctx, "install", "-r", "-t", apk)
	return err
}

// Start stops the app, if it runs, and starts its activity with the given extras (strings,
// as `am start --es` gives them): the embedding copies the GLUTTER_ ones into the
// environment (SPEC §24.1).
func (a ADB) Start(ctx context.Context, appID string, extras map[string]string) error {
	if _, err := a.Output(ctx, "shell", "am", "force-stop", appID); err != nil {
		return err
	}
	args := []string{"shell", "am", "start", "-W", "-n", appID + "/" + ActivityClass}
	keys := make([]string, 0, len(extras))
	for k := range extras {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "--es", ShellQuote(k), ShellQuote(extras[k]))
	}
	out, err := a.Output(ctx, args...)
	if err != nil {
		return err
	}
	if strings.Contains(out, "Error:") {
		return errors.New(TagAndroidStartFailed, "the activity did not start", "output", out, "appID", appID)
	}
	return nil
}

// ShellQuote quotes s for the shell on the device, which `adb shell` hands the arguments to as
// one string.
//
// Flutter: no counterpart
func ShellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// Pid is the process ID of the app, or 0 when it does not run.
func (a ADB) Pid(ctx context.Context, appID string) (int, error) {
	out, err := a.Run.Run(ctx, a.command("shell", "pidof", "-s", appID))
	if err != nil {
		// pidof exits with 1 when there is no such process.
		return 0, nil
	}
	var pid int
	if _, scanErr := fmt.Sscanf(strings.TrimSpace(out), "%d", &pid); scanErr != nil {
		return 0, nil
	}
	return pid, nil
}

// StreamLog writes the log lines of the app to w until the app's process ends or ctx is done.
// It waits up to startWait for the process to appear. Lines are in logcat's "brief" format.
func (a ADB) StreamLog(ctx context.Context, w io.Writer, appID string, startWait time.Duration) error {
	var pid int
	deadline := time.Now().Add(startWait)
	for {
		var err error
		if pid, err = a.Pid(ctx, appID); err != nil {
			return err
		}
		if pid != 0 || time.Now().After(deadline) || ctx.Err() != nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if pid == 0 {
		return errors.New(TagAndroidStartFailed, "the app has no process", "appID", appID)
	}
	logCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	c := a.command("logcat", "-v", "brief", fmt.Sprintf("--pid=%d", pid))
	cmd := exec.CommandContext(logCtx, c.Name, c.Args...)
	cmd.Env = c.Env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return errors.NewErr(TagAndroidLogcatFailed, err)
	}
	if err := cmd.Start(); err != nil {
		return errors.NewErr(TagAndroidLogcatFailed, err).SetMetadata("command", c.String())
	}
	// When the process is gone, give logcat a moment for its last lines and stop it.
	go func() {
		for logCtx.Err() == nil {
			time.Sleep(500 * time.Millisecond)
			if p, _ := a.Pid(logCtx, appID); p != pid {
				time.Sleep(700 * time.Millisecond)
				cancel()
				return
			}
		}
	}()
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		if _, err := fmt.Fprintln(w, sc.Text()); err != nil {
			cancel()
			_ = cmd.Wait() // the output is already abandoned
			return errors.NewErr(TagAndroidLogcatFailed, err)
		}
	}
	_ = cmd.Wait() // killed on purpose when the app ended; its status says nothing
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagAndroidNoDevice is the tag of the error when no device is connected.
	TagAndroidNoDevice = "AndroidNoDevice"
	// TagAndroidManyDevices is the tag of the error when several devices are connected and
	// none was named; the metadata "devices" lists them.
	TagAndroidManyDevices = "AndroidManyDevices"
	// TagAndroidDeviceNotFound is the tag of the error for a named device that is not
	// connected, or not ready.
	TagAndroidDeviceNotFound = "AndroidDeviceNotFound"
	// TagAndroidABIUnsupported is the tag of the error for a device whose ABIs glutter-android
	// does not build.
	TagAndroidABIUnsupported = "AndroidABIUnsupported"
	// TagAndroidStartFailed is the tag of the error when the activity does not start.
	TagAndroidStartFailed = "AndroidStartFailed"
	// TagAndroidLogcatFailed is the tag of the error when logcat cannot be read.
	TagAndroidLogcatFailed = "AndroidLogcatFailed"
)
