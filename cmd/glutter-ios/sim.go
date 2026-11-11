package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/iostool"
)

// simFlags are what `run` and `shot` need to find the Simulator device.
type simFlags struct {
	device  string
	set     string
	env     listFlags
	timeout time.Duration
}

func (s *simFlags) add(fs *flag.FlagSet) {
	s.env.what = "-env"
	fs.StringVar(&s.device, "device", "iphone", "the device to use: iphone or ipad (devices named glutter-iphone and glutter-ipad, made on first use)")
	fs.StringVar(&s.set, "set", "", "the directory of the device set; the user's own set when empty (the tests use .ios-test/devices)")
	fs.Var(&s.env, "env", "an environment variable for the app, NAME=value (repeatable)")
	fs.DurationVar(&s.timeout, "timeout", 3*time.Minute, "how long the whole command may take")
}

// started is an app that is installed and running on a booted device.
type started struct {
	sim     iostool.Simulator
	device  iostool.Device
	id      string
	wasUp   bool
	release func()
}

// prepare builds the app for the Simulator, boots the device (it remembers whether it was up already) and installs the app.
func prepare(ctx context.Context, bf *buildFlags, sf *simFlags, pkg string) (*started, iostool.BuildResult, error) {
	var res iostool.BuildResult
	o, err := bf.options(pkg, true)
	if err != nil {
		return nil, res, err
	}
	if o.Out == "" {
		dir, err := os.MkdirTemp("", "glutter-ios-run-")
		if err != nil {
			return nil, res, errors.NewErr(tagIOSUsage, err)
		}
		o.Out = filepath.Join(dir, iostool.DefaultName(pkg, "")+".app")
	}
	if res, err = iostool.Build(ctx, o); err != nil {
		return nil, res, err
	}
	sim, err := iostool.Discover(ctx, iostool.Simctl{Set: sf.set})
	if err != nil {
		return nil, res, err
	}
	dev, err := sim.Device(ctx, sf.device)
	if err != nil {
		return nil, res, err
	}
	wasUp, err := sim.Booted(ctx, dev.UDID)
	if err != nil {
		return nil, res, err
	}
	if err := sim.Simctl.Boot(ctx, dev.UDID); err != nil {
		return nil, res, err
	}
	s := &started{sim: sim, device: dev, id: res.Info.ID, wasUp: wasUp}
	if err := sim.Simctl.Install(ctx, dev.UDID, res.Layout.App); err != nil {
		return nil, res, err
	}
	return s, res, nil
}

// launch starts the app with its console on the writer and the environment of the flags; the returned command ends when the app does.
func (s *started) launch(ctx context.Context, env []string, out io.Writer) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, "xcrun", s.sim.Simctl.LaunchArgs(s.device.UDID, s.id, true)...)
	cmd.Env = os.Environ()
	for _, e := range env {
		cmd.Env = append(cmd.Env, "SIMCTL_CHILD_"+e)
	}
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Start(); err != nil {
		return nil, errors.NewErr(iostool.TagIOSCommandFailed, err).SetMetadata("program", "xcrun simctl launch")
	}
	return cmd, nil
}

// stop terminates the app, and shuts the device down when this command was the one that booted it.
func (s *started) stop(ctx context.Context) {
	// The app may be gone already, and the device may be in use by another command: both are fine, and nothing is left to report.
	_ = s.sim.Simctl.Terminate(ctx, s.device.UDID, s.id)
	if !s.wasUp {
		_ = s.sim.Simctl.Shutdown(ctx, s.device.UDID)
	}
}

// runRun is `glutter-ios run`: it builds, installs and launches the app on a Simulator device, opens the Simulator's window, and
// prints the console of the app until it ends or the command is interrupted.
func runRun(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	var bf buildFlags
	var sf simFlags
	bf.add(fs, false)
	sf.add(fs)
	pkg, err := parseWith(fs, args, stderr)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 24*time.Hour)
	defer cancel()
	s, _, err := prepare(ctx, &bf, &sf, pkg)
	if err != nil {
		return err
	}
	cleanup, cancelCleanup := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
	defer cancelCleanup()
	defer s.stop(cleanup)
	if sf.set == "" {
		// The window of the Simulator shows the booted devices of the user's own set only.
		if err := exec.CommandContext(ctx, "open", "-a", "Simulator").Run(); err != nil {
			return errors.NewErr(iostool.TagIOSCommandFailed, err).SetMetadata("program", "open -a Simulator")
		}
	}
	cmd, err := s.launch(ctx, sf.env.values, stdout)
	if err != nil {
		return err
	}
	// The command ends with the app, or with the interrupt that cancels the context; the exit code is the app's own.
	_ = cmd.Wait()
	return nil
}

// runShot is `glutter-ios shot`: it builds, installs and launches the app, waits for the frame that the app saves in its container
// (the app is asked with GLUTTER_SCREENSHOT) and copies it to -o; with -screen it takes the screen of the device instead, which has
// what is outside the app's view (the status bar, the keyboard).
func runShot(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("shot", flag.ContinueOnError)
	var bf buildFlags
	var sf simFlags
	var screen bool
	var settle time.Duration
	bf.add(fs, false)
	sf.add(fs)
	// shot makes a debug build unless told otherwise: the hooks it drives (GLUTTER_DEV_*) are read only by debug builds.
	bf.debug = true
	fs.Lookup("debug").DefValue = "true"
	fs.BoolVar(&screen, "screen", false, "take the screen of the device (simctl io screenshot) instead of the frame the app saves")
	fs.DurationVar(&settle, "settle", 3*time.Second, "with -screen, how long to wait after the launch before the screenshot")
	pkg, err := parseWith(fs, args, stderr)
	if err != nil {
		return err
	}
	file := bf.out
	if file == "" || !strings.HasSuffix(file, ".png") {
		return errors.New(tagIOSUsage, "shot needs -o file.png and one package\n"+usage)
	}
	// -o of shot is the picture, not the bundle: the bundle goes to a temporary directory.
	shotOut := file
	bf.out = ""
	ctx, cancel := context.WithTimeout(ctx, sf.timeout)
	defer cancel()
	s, _, err := prepare(ctx, &bf, &sf, pkg)
	if err != nil {
		return err
	}
	cleanup, cancelCleanup := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
	defer cancelCleanup()
	var console strings.Builder
	env := append([]string{}, sf.env.values...)
	var inContainer string
	if !screen {
		container, err := s.sim.Simctl.Container(ctx, s.device.UDID, s.id)
		if err != nil {
			return err
		}
		inContainer = filepath.Join(container, "Documents", "glutter-shot.png")
		if err := os.MkdirAll(filepath.Dir(inContainer), 0o755); err != nil {
			return errors.NewErr(tagIOSUsage, err).SetMetadata("dir", filepath.Dir(inContainer))
		}
		env = append(env, "GLUTTER_SCREENSHOT="+inContainer)
	}
	cmd, err := s.launch(ctx, env, &console)
	if err != nil {
		return err
	}
	ended := make(chan struct{})
	go func() { _ = cmd.Wait(); close(ended) }() // the exit code is the app's own; the picture is what matters
	// The app is terminated, and the device shut down if this command booted it, before the command that holds the console is waited
	// for: that command ends with the app.
	defer func() {
		s.stop(cleanup)
		<-ended
	}()
	if screen {
		select {
		case <-time.After(settle):
		case <-ctx.Done():
			return errors.NewErr(tagIOSUsage, ctx.Err())
		}
		if err := s.sim.Simctl.Screenshot(ctx, s.device.UDID, shotOut); err != nil {
			return err
		}
	} else {
		if err := waitForFile(ctx, inContainer, ended); err != nil {
			return errors.New(tagIOSUsage, "the app did not save its frame", "file", inContainer, "console", console.String())
		}
		data, err := os.ReadFile(inContainer)
		if err != nil {
			return errors.NewErr(tagIOSUsage, err).SetMetadata("file", inContainer)
		}
		if err := os.WriteFile(shotOut, data, 0o644); err != nil {
			return errors.NewErr(tagIOSUsage, err).SetMetadata("file", shotOut)
		}
	}
	if _, err := fmt.Fprintln(stdout, shotOut); err != nil {
		return errors.NewErr(tagIOSUsage, err)
	}
	return nil
}

// waitForFile waits until the file exists and its size has stopped changing, until the app ends or the context is done.
func waitForFile(ctx context.Context, path string, ended <-chan struct{}) error {
	var last int64 = -1
	for {
		if st, err := os.Stat(path); err == nil && st.Size() > 0 {
			if st.Size() == last {
				return nil
			}
			last = st.Size()
		}
		select {
		case <-ctx.Done():
			return errors.NewErr(tagIOSUsage, ctx.Err())
		case <-ended:
			if st, err := os.Stat(path); err == nil && st.Size() > 0 {
				return nil
			}
			return errors.New(tagIOSUsage, "the app ended before it saved its frame", "file", path)
		case <-time.After(200 * time.Millisecond):
		}
	}
}
