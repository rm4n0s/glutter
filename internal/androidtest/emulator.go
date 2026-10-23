package androidtest

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/androidtool"
)

// Options say how an emulator is started.
//
// Flutter: no counterpart
type Options struct {
	// AVD is the name of the virtual device, a key of AVDs; "glutter-37" when empty.
	AVD string
	// Root is the directory of the harness's files (virtual devices, HOME of the emulator, its
	// log); <module root>/.android-test when empty.
	Root string
	// GPU is the emulator's -gpu mode; $GLUTTER_ANDROID_GPU, else "swangle_indirect" (ANGLE over SwiftShader: no host GPU and no display needed; swiftshader_indirect crashes the emulator).
	GPU string
	// Lifetime is how long `timeout -k` lets the emulator live at most; 30 minutes by default.
	Lifetime time.Duration
	// BootTimeout is how long to wait for sys.boot_completed; 5 minutes by default.
	BootTimeout time.Duration
	// Args are more arguments for the emulator, such as "-feature", "-Vulkan".
	Args []string
}

// Emulator is a running emulator with its own adb server.
//
// Flutter: no counterpart
type Emulator struct {
	// Serial is the emulator's adb serial, "emulator-5584".
	Serial string
	// AVD is the virtual device it runs.
	AVD AVD
	// Root is the directory of the harness's files.
	Root string
	// ADB runs adb for this emulator, on its private server.
	ADB androidtool.ADB
	// SDK is the SDK the emulator came from.
	SDK *androidtool.SDK

	cmd     *exec.Cmd
	done    chan struct{}
	adbPort int
	env     []string
	logPath string
}

// ModuleRoot is the directory of go.mod above the current directory.
func ModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", errors.NewErr(TagAndroidTestBootFailed, err).SetMetadata("what", "the working directory")
	}
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d, nil
		}
		if filepath.Dir(d) == d {
			return "", errors.New(TagAndroidTestBootFailed, "there is no go.mod above the working directory", "dir", dir)
		}
	}
}

// Start makes the virtual device if it is not there, starts the emulator headless in a process
// group of its own under `timeout -k`, waits for it to boot, and returns it. The caller stops it
// with Stop. It fails with AndroidTestImageMissing (the system image is not installed),
// AndroidTestEmulatorMissing and AndroidTestBootFailed.
//
// Flutter: no counterpart
func Start(ctx context.Context, o Options) (*Emulator, error) {
	if o.AVD == "" {
		o.AVD = "glutter-37"
	}
	avd, ok := AVDs[o.AVD]
	if !ok {
		return nil, errors.New(TagAndroidTestBootFailed, "there is no such virtual device", "avd", o.AVD)
	}
	if o.Root == "" {
		mod, err := ModuleRoot()
		if err != nil {
			return nil, err
		}
		o.Root = filepath.Join(mod, ".android-test")
	}
	if o.GPU == "" {
		o.GPU = os.Getenv("GLUTTER_ANDROID_GPU")
	}
	if o.GPU == "" {
		o.GPU = "swangle_indirect"
	}
	if o.Lifetime == 0 {
		o.Lifetime = 30 * time.Minute
	}
	if o.BootTimeout == 0 {
		o.BootTimeout = 5 * time.Minute
	}
	home, err := androidtool.FindHome(os.Getenv)
	if err != nil {
		return nil, err
	}
	sdk := &androidtool.SDK{Home: home}
	emulator := filepath.Join(home, "emulator", "emulator")
	if _, err := os.Stat(emulator); err != nil {
		return nil, errors.NewErr(TagAndroidTestEmulatorMissing, err).SetMetadata("path", emulator, "install", `sdkmanager "emulator"`)
	}
	if err := ensureAVD(home, o.Root, avd); err != nil {
		return nil, err
	}
	for _, dir := range []string{"home", "user"} {
		if err := os.MkdirAll(filepath.Join(o.Root, dir), 0o755); err != nil {
			return nil, errors.NewErr(TagAndroidTestAVDFailed, err).SetMetadata("dir", filepath.Join(o.Root, dir))
		}
	}
	adbPort, err := freePort()
	if err != nil {
		return nil, err
	}
	console, err := freeConsolePort()
	if err != nil {
		return nil, err
	}
	e := &Emulator{AVD: avd, Root: o.Root, SDK: sdk, done: make(chan struct{}), adbPort: adbPort,
		Serial: "emulator-" + strconv.Itoa(console), logPath: filepath.Join(o.Root, "emulator-"+avd.Name+".log")}
	e.env = harnessEnv(o.Root, home, adbPort)
	e.ADB = androidtool.ADB{Path: sdk.Adb(), Serial: e.Serial, Env: e.env, Run: androidtool.ExecRunner{}}

	server := androidtool.ADB{Path: sdk.Adb(), Env: e.env, Run: androidtool.ExecRunner{}}
	if _, err := server.Output(ctx, "start-server"); err != nil {
		return nil, err
	}
	logFile, err := os.Create(e.logPath)
	if err != nil {
		return nil, errors.NewErr(TagAndroidTestBootFailed, err).SetMetadata("path", e.logPath)
	}
	secs := strconv.Itoa(int(o.Lifetime.Seconds()))
	args := []string{"-k", "10s", secs + "s", emulator, "-avd", avd.Name, "-no-window", "-no-audio", "-no-boot-anim",
		"-no-snapshot", "-no-metrics", "-gpu", o.GPU, "-port", strconv.Itoa(console)}
	args = append(args, o.Args...)
	e.cmd = exec.Command("timeout", args...)
	e.cmd.Env = e.env
	e.cmd.Stdout, e.cmd.Stderr = logFile, logFile
	setProcessGroup(e.cmd)
	if err := e.cmd.Start(); err != nil {
		_ = logFile.Close() // nothing was written
		_, _ = server.Output(ctx, "kill-server")
		return nil, errors.NewErr(TagAndroidTestBootFailed, err).SetMetadata("command", "timeout "+strings.Join(args, " "))
	}
	go func() {
		_ = e.cmd.Wait() // the status says nothing: the emulator is stopped on purpose, or the boot check reports it
		_ = logFile.Close()
		close(e.done)
	}()
	if err := e.waitBoot(ctx, o.BootTimeout); err != nil {
		e.Stop()
		return nil, err
	}
	return e, nil
}

// waitBoot waits until sys.boot_completed is 1, the emulator exits, or the time is over.
func (e *Emulator) waitBoot(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		select {
		case <-e.done:
			return errors.New(TagAndroidTestBootFailed, "the emulator exited before it booted", "log", e.logTail())
		case <-ctx.Done():
			return errors.NewErr(TagAndroidTestBootFailed, ctx.Err()).SetMetadata("avd", e.AVD.Name)
		case <-time.After(2 * time.Second):
		}
		if out, err := e.ADB.Output(ctx, "shell", "getprop", "sys.boot_completed"); err == nil && strings.TrimSpace(out) == "1" {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New(TagAndroidTestBootFailed, "the emulator did not boot in time", "avd", e.AVD.Name, "timeout", timeout.String(), "log", e.logTail())
		}
	}
}

// logTail is the last lines of the emulator's output.
func (e *Emulator) logTail() string {
	b, err := os.ReadFile(e.logPath)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) > 25 {
		lines = lines[len(lines)-25:]
	}
	return strings.Join(lines, "\n")
}

// Stop stops the emulator and the adb server: the emulator is asked to quit (`adb emu kill`),
// and its process group is killed if it is still there after 20 seconds. Nothing of the harness
// runs afterwards; Leftovers says so.
func (e *Emulator) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, _ = e.ADB.Output(ctx, "emu", "kill") // it may be dead already
	select {
	case <-e.done:
	case <-time.After(20 * time.Second):
		if e.cmd != nil && e.cmd.Process != nil {
			killGroup(e.cmd.Process.Pid)
		}
		<-e.done
	}
	if e.cmd != nil && e.cmd.Process != nil {
		killGroup(e.cmd.Process.Pid) // netsimd and crashpad_handler can outlive the emulator
	}
	server := androidtool.ADB{Path: e.SDK.Adb(), Env: e.env, Run: androidtool.ExecRunner{}}
	_, _ = server.Output(ctx, "kill-server")
	waitForNoLeftovers(e.Root, 10*time.Second)
}

// waitForNoLeftovers waits until Leftovers has nothing, for at most d.
func waitForNoLeftovers(root string, d time.Duration) {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) && len(Leftovers(root)) > 0 {
		time.Sleep(200 * time.Millisecond)
	}
}

// harnessEnv is the environment of everything the harness starts: its own HOME and Android
// directories (so that ~/.android is never touched), its own adb server, and no display (so that
// no window opens on the user's desktop).
func harnessEnv(root, sdk string, adbPort int) []string {
	drop := map[string]bool{"HOME": true, "DISPLAY": true, "WAYLAND_DISPLAY": true, "ANDROID_USER_HOME": true,
		"ANDROID_AVD_HOME": true, "ANDROID_SDK_ROOT": true, "ANDROID_HOME": true, "ANDROID_ADB_SERVER_PORT": true,
		"ANDROID_EMULATOR_HOME": true, "ANDROID_SERIAL": true}
	var env []string
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if !drop[k] {
			env = append(env, kv)
		}
	}
	return append(env,
		"HOME="+filepath.Join(root, "home"),
		"ANDROID_USER_HOME="+filepath.Join(root, "user"),
		"ANDROID_AVD_HOME="+filepath.Join(root, "avd"),
		"ANDROID_SDK_ROOT="+sdk, "ANDROID_HOME="+sdk,
		"ANDROID_ADB_SERVER_PORT="+strconv.Itoa(adbPort))
}

// freePort is a TCP port nothing listens on now.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, errors.NewErr(TagAndroidTestBootFailed, err).SetMetadata("what", "a free port for the adb server")
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		return 0, errors.NewErr(TagAndroidTestBootFailed, err).SetMetadata("what", "a free port for the adb server")
	}
	return port, nil
}

// freeConsolePort is an even port from 5554 to 5584 whose neighbor (the adb port) is free too:
// the emulator's console port.
func freeConsolePort() (int, error) {
	for p := 5584; p >= 5554; p -= 2 {
		if portFree(p) && portFree(p+1) {
			return p, nil
		}
	}
	return 0, errors.New(TagAndroidTestBootFailed, "every console port from 5554 to 5584 is in use: another emulator runs")
}

func portFree(p int) bool {
	l, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(p))
	if err != nil {
		return false
	}
	return l.Close() == nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagAndroidTestEmulatorMissing is the tag of the error for an SDK without the emulator;
	// the metadata "install" has the sdkmanager command.
	TagAndroidTestEmulatorMissing = "AndroidTestEmulatorMissing"
	// TagAndroidTestBootFailed is the tag of the error for an emulator that did not start or
	// did not boot; the metadata "log" has the end of its output.
	TagAndroidTestBootFailed = "AndroidTestBootFailed"
)
