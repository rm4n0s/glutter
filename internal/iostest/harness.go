package iostest

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/iostool"
)

// Harness is the Simulator of a test: the device set, the runtime and the devices it has booted. It shuts them down and stops what it
// launched when the test ends (tb.Cleanup).
//
// Flutter: no counterpart
type Harness struct {
	tb      testing.TB
	Simctl  iostool.Simctl
	Runtime iostool.Runtime
	sim     iostool.Simulator
	mu      sync.Mutex
	booted  map[string]string // the UDIDs that this harness booted, by family
}

// ModuleRoot is the directory of the module: the first directory up from the working directory that has a go.mod. It fails with
// IOSSimulatorFailed when there is none.
//
// Flutter: no counterpart
func ModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", errors.NewErr(tagIOSSimulatorFailed, err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New(tagIOSSimulatorFailed, "no go.mod above the working directory")
		}
		dir = parent
	}
}

// New makes the harness of a test: it checks Xcode and the runtime, and skips the test, with the instruction, when either is
// missing, as the tests of the other platforms skip without their tools. The device set is `.ios-test/devices` of the module.
//
// Flutter: no counterpart
func New(tb testing.TB) *Harness {
	tb.Helper()
	ctx := context.Background()
	if _, err := iostool.CheckXcode(ctx, nil); err != nil {
		tb.Skipf("no iOS toolchain: %v", err)
	}
	root, err := ModuleRoot()
	if err != nil {
		tb.Fatal(err)
	}
	set := filepath.Join(root, ".ios-test", "devices")
	if err := os.MkdirAll(set, 0o755); err != nil {
		tb.Fatal(err)
	}
	sim, err := iostool.Discover(ctx, iostool.Simctl{Set: set})
	if err != nil {
		tb.Skipf("no iOS Simulator runtime: %v", err)
	}
	h := &Harness{tb: tb, Simctl: sim.Simctl, Runtime: sim.Runtime, sim: sim, booted: map[string]string{}}
	tb.Cleanup(h.shutdownAll)
	return h
}

// Device returns the booted device of the family, "iphone" or "ipad": it makes the device the first time (named glutter-iphone or
// glutter-ipad) and boots it, and the test's end shuts it down. It fails with IOSSimctlFailed when the device cannot be made or
// booted.
//
// Flutter: no counterpart
func (h *Harness) Device(ctx context.Context, family string) (iostool.Device, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	d, err := h.sim.Device(ctx, family)
	if err != nil {
		return d, err
	}
	if err := h.Simctl.Boot(ctx, d.UDID); err != nil {
		return d, err
	}
	h.booted[family] = d.UDID
	return d, nil
}

// shutdownAll shuts down the devices the harness booted. A device that is already down is not an error.
func (h *Harness) shutdownAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	up, err := h.Simctl.Devices(ctx, true)
	if err != nil {
		h.tb.Errorf("listing the booted devices: %v", err)
	}
	for family, udid := range h.booted {
		// A device that is down already (the test shut it down itself) needs no shutdown.
		if slices.ContainsFunc(up, func(d iostool.Device) bool { return d.UDID == udid }) {
			if err := h.Simctl.Shutdown(ctx, udid); err != nil {
				h.tb.Errorf("shutting down the %s: %v", family, err)
			}
		}
		delete(h.booted, family)
	}
}

// Leftovers lists the devices of the set that are booted: after a test it must be empty (SPEC §30.11).
//
// Flutter: no counterpart
func (h *Harness) Leftovers(ctx context.Context) ([]iostool.Device, error) {
	return h.Simctl.Devices(ctx, true)
}

// Launched is an app that was started with its console attached.
//
// Flutter: no counterpart
type Launched struct {
	h      *Harness
	udid   string
	id     string
	cmd    *exec.Cmd
	lines  chan string
	mu     sync.Mutex
	output []string
	done   chan struct{}
}

// Launch starts the installed app with its console attached; env is given to the app (as SIMCTL_CHILD_ variables). The test's end
// terminates the app and the command. It fails with IOSSimulatorFailed when the command cannot start.
//
// Flutter: no counterpart
func (h *Harness) Launch(ctx context.Context, udid, id string, env ...string) (*Launched, error) {
	l := &Launched{h: h, udid: udid, id: id, lines: make(chan string, 256), done: make(chan struct{})}
	cmd := exec.CommandContext(ctx, "xcrun", h.Simctl.LaunchArgs(udid, id, true)...)
	cmd.Env = os.Environ()
	for _, e := range env {
		cmd.Env = append(cmd.Env, "SIMCTL_CHILD_"+e)
	}
	ownProcessGroup(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.NewErr(tagIOSSimulatorFailed, err)
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return nil, errors.NewErr(tagIOSSimulatorFailed, err).SetMetadata("app", id)
	}
	l.cmd = cmd
	go func() {
		defer close(l.done)
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			line := sc.Text()
			l.mu.Lock()
			l.output = append(l.output, line)
			l.mu.Unlock()
			select {
			case l.lines <- line:
			default:
			}
		}
		_ = cmd.Wait() // the app ended or the command was killed; the lines are what the test reads
	}()
	h.tb.Cleanup(l.Stop)
	return l, nil
}

// Output is the lines the app has printed so far.
//
// Flutter: no counterpart
func (l *Launched) Output() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.output...)
}

// Ended says whether the app has ended (it exited, crashed or was terminated).
//
// Flutter: no counterpart
func (l *Launched) Ended() bool {
	select {
	case <-l.done:
		return true
	default:
		return false
	}
}

// WaitFor waits until the app has printed a line that contains the text, and returns the line; it fails with IOSSimulatorFailed,
// with the output so far, when the time is out or the app ends first.
//
// Flutter: no counterpart
func (l *Launched) WaitFor(text string, timeout time.Duration) (string, error) {
	for _, line := range l.Output() {
		if strings.Contains(line, text) {
			return line, nil
		}
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case line := <-l.lines:
			if strings.Contains(line, text) {
				return line, nil
			}
		case <-l.done:
			for _, line := range l.Output() {
				if strings.Contains(line, text) {
					return line, nil
				}
			}
			return "", errors.New(tagIOSSimulatorFailed, "the app ended before it printed the line", "wanted", text, "output", strings.Join(l.Output(), "\n"))
		case <-timer.C:
			return "", errors.New(tagIOSSimulatorFailed, "the app did not print the line in time", "wanted", text, "output", strings.Join(l.Output(), "\n"))
		}
	}
}

// Stop terminates the app and the command that holds its console. It is called at the end of the test; calling it again does nothing.
//
// Flutter: no counterpart
func (l *Launched) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// The app may be gone already, and then terminate says so.
	_ = l.h.Simctl.Terminate(ctx, l.udid, l.id)
	if l.cmd != nil && l.cmd.Process != nil {
		killGroup(l.cmd)
	}
	<-l.done
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagIOSSimulatorFailed = TagIOSSimulatorFailed

	// TagIOSSimulatorFailed is the tag of the error for a Simulator command of the harness that cannot run, a device that cannot be made
	// or booted, or an app that does not print what the test waits for.
	//
	// Flutter: no counterpart
	TagIOSSimulatorFailed = "IOSSimulatorFailed"
)
