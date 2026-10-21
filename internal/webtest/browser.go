package webtest

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
)

// Browser is a browser the harness can launch.
//
// Flutter: no counterpart
type Browser string

// The browsers of the harness (SPEC §17).
const (
	// Chromium is Chromium, or Chrome.
	//
	// Flutter: no counterpart
	Chromium Browser = "chromium"
	// Firefox is Firefox.
	//
	// Flutter: no counterpart
	Firefox Browser = "firefox"
)

// ParseBrowser reads the name of a browser: "chromium" or "firefox". It fails with
// BrowserUnknown for another name.
//
// Flutter: no counterpart
func ParseBrowser(name string) (Browser, error) {
	switch Browser(name) {
	case Chromium, Firefox:
		return Browser(name), nil
	}
	return "", errors.New("BrowserUnknown", "the browser is not chromium or firefox", "value", name)
}

// executables are the names a browser's program goes by, best first.
var executables = map[Browser][]string{
	Chromium: {"chromium-browser", "chromium", "google-chrome-stable", "google-chrome"},
	Firefox:  {"firefox"},
}

// Locate finds the program of the browser with lookPath (exec.LookPath when nil). It fails
// with BrowserNotFound, naming the programs it looked for.
//
// Flutter: no counterpart
func Locate(b Browser, lookPath func(string) (string, error)) (string, error) {
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	names := executables[b]
	for _, name := range names {
		if path, err := lookPath(name); err == nil {
			return path, nil
		}
	}
	return "", errors.New("BrowserNotFound", "the browser is not installed (sudo dnf install chromium firefox)",
		"browser", string(b), "looked for", strings.Join(names, ", "))
}

// ExtraFlagsEnv names an environment variable whose value (flags separated by blanks) is
// added to the command line of Chromium, for an environment that needs a flag the harness
// does not set, such as --no-sandbox in a container without user namespaces.
//
// Flutter: no counterpart
const ExtraFlagsEnv = "GLUTTER_WEBTEST_FLAGS"

// flag is a command-line flag of a browser with the reason it is there.
type flag struct {
	arg    string
	reason string
}

// chromiumFlags are the flags every Chromium gets. The reason of each is the reason it is
// kept; none is here for convenience.
var chromiumFlags = []flag{
	{"--headless=new", "no window: the new headless mode is the browser itself, so the page behaves as in a real one (SPEC §23.8)"},
	{"--no-first-run", "skip the first-run page and the welcome flow, which would take the first load"},
	{"--no-default-browser-check", "no question about the default browser"},
	{"--disable-extensions", "nothing but the page under test"},
	{"--disable-background-networking", "no update, sync, safe-browsing or field-trial traffic during a test"},
	{"--disable-component-update", "no download of components while a test runs"},
	{"--disable-sync", "no Google account sync"},
	{"--disable-default-apps", "no first-run apps"},
	{"--hide-scrollbars", "scroll bars would change the viewport that the frame is measured against"},
	{"--mute-audio", "tests make no sound"},
	{"--enable-unsafe-swiftshader", "headless Chromium has no GPU here, and its WebGL 2 then comes from SwiftShader, the software Vulkan that it allows for trusted content only with this flag (without it a future version answers null to getContext(\"webgl2\"), and today it logs that the fallback is deprecated); the WebGL tests of step 21.4 and the pages that use WebGL need it"},
	{"--enable-unsafe-webgpu", "WebGPU is off in Chromium on Linux; this turns it on, so that navigator.gpu gives an adapter (without it requestAdapter answers null, \"No available adapters\"). It is not enough by itself: with only this flag the adapter is SwiftShader's, and the canvas's webgpu context cannot be presented in headless mode (the GPU process logs \"Could not find SharedImageBackingFactory\" for the WebgpuSwapChainTexture, and the device is destroyed at the first getCurrentTexture), so the two flags below come with it"},
	{"--enable-features=Vulkan", "makes the GPU process use Vulkan, whose shared images can back the canvas's WebGPU swap chain; headless mode has no other factory that can (tried: --use-angle=swiftshader, --use-vulkan=swiftshader, --ignore-gpu-blocklist, --ozone-platform=x11 under headless: all fail as above)"},
	{"--use-angle=vulkan", "ANGLE on Vulkan, the other half of the pair: with both flags the adapter is the machine's Vulkan GPU (RADV on the machine where step 21.5 was done) and a canvas is presented; with either alone it is not. WebGL 2 then also runs on Vulkan through ANGLE instead of SwiftShader, which --enable-unsafe-swiftshader above still allows as the fallback where the machine has no Vulkan GPU"},
	// --no-sandbox is not here: Chromium starts without it where these tests were developed.
	// Where it cannot (a container without user namespaces), set GLUTTER_WEBTEST_FLAGS=--no-sandbox.
}

// firefoxPrefs are the preferences written to the user.js of a Firefox profile.
var firefoxPrefs = []struct {
	name, value, reason string
}{
	{"browser.shell.checkDefaultBrowser", "false", "no question about the default browser"},
	{"browser.startup.homepage_welcome_url", `""`, "no welcome page"},
	{"browser.startup.homepage_welcome_url.additional", `""`, "no second welcome page"},
	{"browser.aboutwelcome.enabled", "false", "no welcome flow"},
	{"startup.homepage_welcome_url", `""`, "no welcome page on an older version"},
	{"datareporting.policy.dataSubmissionEnabled", "false", "no telemetry"},
	{"toolkit.telemetry.reportingpolicy.firstRun", "false", "no telemetry notice"},
	{"app.update.auto", "false", "no browser updates during a test"},
	{"app.update.enabled", "false", "no browser updates during a test"},
	{"extensions.update.enabled", "false", "no extension updates during a test"},
	{"browser.safebrowsing.malware.enabled", "false", "no safe-browsing traffic"},
	{"browser.safebrowsing.phishing.enabled", "false", "no safe-browsing traffic"},
	{"network.captive-portal-service.enabled", "false", "no captive-portal probe"},
	{"network.connectivity-service.enabled", "false", "no connectivity probe"},
	{"app.normandy.enabled", "false", "no experiments"},
	{"browser.search.update", "false", "no search-engine updates"},
	{"browser.tabs.warnOnClose", "false", "no dialog when the browser is closed"},
	{"dom.webgpu.enabled", "true", "WebGPU is off in Firefox on Linux by default; with it on, navigator.gpu exists where the build has the implementation, and where no adapter comes the WebGPU tests skip Firefox with the reason (step 21.5)"},
}

// LaunchOptions are the optional parameters of Launch.
//
// Flutter: no counterpart
type LaunchOptions struct {
	// Path is the program of the browser; Locate finds it when empty.
	Path string
	// Width and Height are the size of the browser window in CSS pixels at the start;
	// 800x600 when zero.
	Width, Height int
	// Lifetime is how long the browser may run at most: it runs under `timeout -k 5s`, which
	// stops it when the time is up even if nothing else does. Two minutes when zero.
	Lifetime time.Duration
	// ScaleFactor is the device pixel ratio of the pages, 1 when zero. Frames are compared
	// pixel for pixel with the goldens, so the ratio is fixed instead of being the screen's
	// (Chromium: --force-device-scale-factor, Firefox: layout.css.devPixelsPerPx).
	ScaleFactor float64
	// Output receives what the browser writes to its standard output and error; nothing
	// when nil.
	Output io.Writer
}

// Process is a browser that Launch started.
//
// Flutter: no counterpart
type Process struct {
	cmd     *exec.Cmd
	profile string
	// display is the X server that Firefox draws on, nil for the other browser.
	display *privateDisplay
	done    chan struct{}
	once    sync.Once
	// Err is how the browser ended, valid once Done is closed.
	Err error
}

// Launch starts the browser, headless, with a temporary profile and a home directory of its
// own (so that it touches nothing of the user's), at the URL. Firefox also gets a private Xvfb
// display, which Stop ends (see the environment below). It runs under `timeout -k` in a
// process group of its own; Stop kills the group and removes the profile. It fails with
// BrowserNotFound when the program is not there and BrowserLaunchFailed when it does not
// start.
//
// Flutter: no counterpart
func Launch(b Browser, url string, o LaunchOptions) (*Process, error) {
	path := o.Path
	if path == "" {
		var err error
		if path, err = Locate(b, nil); err != nil {
			return nil, err
		}
	}
	width, height := o.Width, o.Height
	if width == 0 || height == 0 {
		width, height = 800, 600
	}
	lifetime := o.Lifetime
	if lifetime == 0 {
		lifetime = 2 * time.Minute
	}
	profile, err := os.MkdirTemp("", "glutter-browser-")
	if err != nil {
		return nil, errors.NewErr("BrowserLaunchFailed", err).SetMetadata("what", "the temporary profile")
	}
	home := filepath.Join(profile, "home")
	data := filepath.Join(profile, "profile")
	for _, d := range []string{home, data} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			_ = os.RemoveAll(profile) // the failure to make it is the one to report
			return nil, errors.NewErr("BrowserLaunchFailed", err).SetMetadata("what", "the temporary profile")
		}
	}
	scale := o.ScaleFactor
	if scale == 0 {
		scale = 1
	}
	scaleText := strconv.FormatFloat(scale, 'f', -1, 64)
	var args []string
	switch b {
	case Chromium:
		for _, f := range chromiumFlags {
			args = append(args, f.arg)
		}
		args = append(args, "--force-device-scale-factor="+scaleText, "--user-data-dir="+data, "--window-size="+strconv.Itoa(width)+","+strconv.Itoa(height))
		args = append(args, strings.Fields(os.Getenv(ExtraFlagsEnv))...)
	case Firefox:
		if err := writeFirefoxPrefs(data, scaleText); err != nil {
			_ = os.RemoveAll(profile) // the failure to write is the one to report
			return nil, err
		}
		args = []string{"--headless", "--no-remote", "--new-instance", "--profile", data,
			"--window-size=" + strconv.Itoa(width) + "," + strconv.Itoa(height)}
	default:
		_ = os.RemoveAll(profile)
		return nil, errors.New("BrowserUnknown", "the browser is not chromium or firefox", "value", string(b))
	}
	args = append(args, url)
	secs := strconv.Itoa(max(1, int(lifetime.Seconds())))
	cmd := exec.Command("timeout", append([]string{"-k", "5s", secs + "s", path}, args...)...)
	// Nothing of the user's desktop or home is reachable: the user's display is not passed (the
	// home, config and cache directories are in the profile too). Firefox gets an X server of
	// its own instead: on Linux it has no native GL to give a page without a display, so
	// getContext("webgl2") answers null (FEATURE_FAILURE_WEBGL_EXHAUSTED_DRIVERS), and with
	// one its headless mode draws WebGL with Mesa's software rasterizer. Its window is never
	// shown: the server is a virtual framebuffer.
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"), "HOME=" + home, "TMPDIR=" + profile,
		"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"), "XDG_CACHE_HOME=" + filepath.Join(home, ".cache"),
		"XDG_DATA_HOME=" + filepath.Join(home, ".local", "share"), "MOZ_HEADLESS=1", "LANG=C.UTF-8",
	}
	var display *privateDisplay
	if b == Firefox {
		var derr error
		if display, derr = startPrivateDisplay(lifetime); derr != nil {
			_ = os.RemoveAll(profile) // the failure of the display is the one to report
			return nil, derr
		}
		if display != nil {
			cmd.Env = append(cmd.Env, "DISPLAY="+display.name)
		}
	}
	cmd.Stdout, cmd.Stderr = o.Output, o.Output
	setProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		if display != nil {
			display.stop()
		}
		_ = os.RemoveAll(profile) // the failure to start is the one to report
		return nil, errors.NewErr("BrowserLaunchFailed", err).SetMetadata("browser", string(b), "path", path)
	}
	p := &Process{cmd: cmd, profile: profile, display: display, done: make(chan struct{})}
	go func() {
		p.Err = cmd.Wait()
		close(p.done)
	}()
	return p, nil
}

// writeFirefoxPrefs writes the user.js of the profile.
func writeFirefoxPrefs(profile, scale string) error {
	var sb strings.Builder
	sb.WriteString("// the device pixel ratio is fixed, not the screen's\nuser_pref(\"layout.css.devPixelsPerPx\", \"" + scale + "\");\n")
	for _, p := range firefoxPrefs {
		sb.WriteString("// " + p.reason + "\nuser_pref(\"" + p.name + "\", " + p.value + ");\n")
	}
	if err := os.WriteFile(filepath.Join(profile, "user.js"), []byte(sb.String()), 0o644); err != nil {
		return errors.NewErr("BrowserLaunchFailed", err).SetMetadata("what", "user.js of the Firefox profile")
	}
	return nil
}

// Done is closed when the browser has ended, by itself, by the lifetime or by Stop.
//
// Flutter: no counterpart
func (p *Process) Done() <-chan struct{} { return p.done }

// Stop kills the browser and every process it started, waits for it, and removes the
// profile. It is safe to call twice.
//
// Flutter: no counterpart
func (p *Process) Stop() {
	p.once.Do(func() {
		killGroup(p.cmd.Process.Pid)
		<-p.done
		if p.display != nil {
			p.display.stop()
		}
		_ = os.RemoveAll(p.profile) // a temporary directory: a leftover is harmless
	})
}
