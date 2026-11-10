package iostool_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/iostool"
)

// fakeRunner answers the programs of Xcode from a table: the key is the program and its arguments joined by blanks. It records the
// calls.
type fakeRunner struct {
	answers map[string]fakeAnswer
	calls   []string
}

type fakeAnswer struct {
	out  string
	exit int
	err  error
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, int, error) {
	key := strings.Join(append([]string{name}, args...), " ")
	f.calls = append(f.calls, key)
	a, ok := f.answers[key]
	if !ok {
		return nil, 127, nil
	}
	return []byte(a.out), a.exit, a.err
}

// xcodeAnswers are the answers of a machine that has the full Xcode.
func xcodeAnswers() map[string]fakeAnswer {
	return map[string]fakeAnswer{
		"xcode-select -p":                                                  {out: "/Applications/Xcode.app/Contents/Developer\n"},
		"xcrun --sdk iphoneos --find clang":                                {out: "/X/clang\n"},
		"xcrun --sdk iphoneos --show-sdk-path":                             {out: "/X/iPhoneOS.sdk\n"},
		"xcrun --sdk iphonesimulator --show-sdk-path":                      {out: "/X/iPhoneSimulator.sdk\n"},
		"/usr/bin/codesign --force --timestamp=none --sign - /a/Hello.app": {},
	}
}

func requireRoute(t *testing.T, err error, route string) {
	t.Helper()
	if err == nil {
		t.Fatalf("no error, want the route %s", route)
	}
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("the error is a %T, want *errors.Error: %v", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("route = %s, want it to contain %s", appErr.Route(), route)
	}
}

func TestInfoPlist(t *testing.T) {
	info := iostool.Info{Executable: "counter", ID: "dev.glutter.counter", Name: "Counter", Version: "1.2.3", Simulator: true, Icons: true,
		Usage: map[string]string{"NSMotionUsageDescription": "motion", "NSCameraUsageDescription": "a <b> & c"}}
	got, err := iostool.InfoPlist(info)
	if err != nil {
		t.Fatal(err)
	}
	// The file is well-formed XML, and the keys are in a fixed order, with the usage strings sorted.
	dec := xml.NewDecoder(bytes.NewReader(got))
	dec.Strict = false
	for {
		if _, err := dec.Token(); err != nil {
			if err != io.EOF {
				t.Fatalf("not XML: %v", err)
			}
			break
		}
	}
	text := string(got)
	for _, want := range []string{
		"<key>CFBundleIdentifier</key>\n\t<string>dev.glutter.counter</string>",
		"<key>CFBundleSupportedPlatforms</key>\n\t<array>\n\t\t<string>iPhoneSimulator</string>",
		"<key>MinimumOSVersion</key>\n\t<string>15.0</string>",
		"<key>UIDeviceFamily</key>\n\t<array>\n\t\t<integer>1</integer>\n\t\t<integer>2</integer>",
		"<key>UILaunchScreen</key>\n\t<dict/>",
		"<key>CADisableMinimumFrameDurationOnPhone</key>\n\t<true/>",
		"<key>UIApplicationSupportsMultipleScenes</key>\n\t<false/>",
		"<key>UISceneConfigurations</key>\n\t<dict/>",
		"<string>AppIcon60x60</string>",
		"<string>AppIcon83.5x83.5</string>",
		"a &lt;b&gt; &amp; c",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the plist lacks %q", want)
		}
	}
	if strings.Index(text, "NSCameraUsageDescription") > strings.Index(text, "NSMotionUsageDescription") {
		t.Error("the usage strings are not sorted by key")
	}
	again, err := iostool.InfoPlist(info)
	if err != nil || !bytes.Equal(got, again) {
		t.Error("the plist is not the same file every time")
	}
	// A device build says iPhoneOS, an app with no icons has no icon keys, and the default orientations differ by device.
	device, err := iostool.InfoPlist(iostool.Info{Executable: "a", ID: "dev.glutter.a", Name: "A", Version: "1"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(device), "<string>iPhoneOS</string>") || strings.Contains(string(device), "CFBundleIcons") {
		t.Error("a device build without icons has the wrong platform or icon keys")
	}
	phone := string(device)[strings.Index(string(device), "<key>UISupportedInterfaceOrientations</key>"):strings.Index(string(device), "<key>UISupportedInterfaceOrientations~ipad</key>")]
	if strings.Contains(phone, "UpsideDown") {
		t.Error("an iPhone does not support the upside down portrait by default")
	}
	if !strings.Contains(string(device), "UIInterfaceOrientationPortraitUpsideDown") {
		t.Error("an iPad supports the upside down portrait by default")
	}
}

func TestInfoPlistRejectsWhatABundleCannotHave(t *testing.T) {
	ok := iostool.Info{Executable: "a", ID: "dev.glutter.a", Name: "A", Version: "1"}
	tests := []struct {
		name   string
		mutate func(*iostool.Info)
	}{
		{"an executable with a slash", func(i *iostool.Info) { i.Executable = "a/b" }},
		{"no executable", func(i *iostool.Info) { i.Executable = "" }},
		{"a name with a colon", func(i *iostool.Info) { i.Name = "a:b" }},
		{"an identifier with one part", func(i *iostool.Info) { i.ID = "counter" }},
		{"an identifier with an underscore", func(i *iostool.Info) { i.ID = "dev.glutter.my_app" }},
		{"a version with a bracket", func(i *iostool.Info) { i.Version = "1<2" }},
		{"an empty version", func(i *iostool.Info) { i.Version = "" }},
		{"an unknown orientation", func(i *iostool.Info) { i.Orientations = []string{"sideways"} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := ok
			tt.mutate(&i)
			_, err := iostool.InfoPlist(i)
			requireRoute(t, err, "InfoPlist."+iostool.TagIOSBundleInvalid)
		})
	}
}

func TestDefaultID(t *testing.T) {
	tests := map[string]string{"Counter": "dev.glutter.counter", "My App_2": "dev.glutter.my-app-2", "***": "dev.glutter.app", "a.b": "dev.glutter.a-b"}
	for name, want := range tests {
		if got := iostool.DefaultID(name); got != want {
			t.Errorf("DefaultID(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestEntitlements(t *testing.T) {
	got := string(iostool.Entitlements("ABCDE12345", "dev.glutter.counter"))
	for _, want := range []string{"<string>ABCDE12345.dev.glutter.counter</string>", "com.apple.developer.team-identifier", "<key>get-task-allow</key>\n\t<true/>"} {
		if !strings.Contains(got, want) {
			t.Errorf("the entitlements lack %q", want)
		}
	}
}

func TestIcons(t *testing.T) {
	src, err := iostool.DefaultIcon()
	if err != nil {
		t.Fatal(err)
	}
	icons, err := iostool.Icons(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(icons) != len(iostool.IconFiles()) {
		t.Fatalf("%d icons, want %d", len(icons), len(iostool.IconFiles()))
	}
	for _, f := range iostool.IconFiles() {
		data, ok := icons[f.Name]
		if !ok {
			t.Errorf("no file %s", f.Name)
			continue
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Errorf("%s: %v", f.Name, err)
			continue
		}
		if b := img.Bounds(); b.Dx() != f.Pixels || b.Dy() != f.Pixels {
			t.Errorf("%s is %v, want %d pixels", f.Name, b, f.Pixels)
		}
		if _, _, _, a := img.At(3, 3).RGBA(); a != 0xffff {
			t.Errorf("%s has alpha at its corner: iOS icons are opaque", f.Name)
		}
	}
}

func TestIconsRejectWhatIsNotAnIcon(t *testing.T) {
	square := func(n int) []byte {
		var b bytes.Buffer
		if err := png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, n, n))); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	var wide bytes.Buffer
	if err := png.Encode(&wide, image.NewNRGBA(image.Rect(0, 0, 400, 300))); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"not a PNG": []byte("nope"), "not square": wide.Bytes(), "too small": square(100)} {
		t.Run(name, func(t *testing.T) {
			_, err := iostool.Icons(data)
			requireRoute(t, err, "Icons."+iostool.TagIOSIconInvalid)
		})
	}
}

func TestWriteMakesAFlatBundle(t *testing.T) {
	icons, err := iostool.Icons(mustIcon(t))
	if err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(t.TempDir(), "Counter.app")
	l, err := iostool.Write(app, iostool.Bundle{Info: iostool.Info{Executable: "counter", ID: "dev.glutter.counter", Name: "Counter", Version: "1"},
		Executable: []byte("MACHO"), Icons: icons, Profile: []byte("profile")})
	if err != nil {
		t.Fatal(err)
	}
	if l.Executable != filepath.Join(app, "counter") || l.Plist != filepath.Join(app, "Info.plist") || l.Profile != filepath.Join(app, "embedded.mobileprovision") {
		t.Errorf("layout = %+v", l)
	}
	for _, name := range []string{"counter", "Info.plist", "embedded.mobileprovision", "AppIcon60x60@2x.png", "AppIcon83.5x83.5@2x~ipad.png"} {
		if _, err := os.Stat(filepath.Join(app, name)); err != nil {
			t.Errorf("%s is not in the bundle: %v", name, err)
		}
	}
	if st, err := os.Stat(l.Executable); err != nil || st.Mode()&0o111 == 0 {
		t.Errorf("the executable is not executable: %v", err)
	}
	plist, err := os.ReadFile(l.Plist)
	if err != nil || !strings.Contains(string(plist), "CFBundleIcons") {
		t.Error("the plist of a bundle that has icons names them")
	}
	// A bundle is written again over the old one: what was there is gone.
	if err := os.WriteFile(filepath.Join(app, "stale"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := iostool.Write(app, iostool.Bundle{Info: iostool.Info{Executable: "counter", ID: "dev.glutter.counter", Name: "Counter", Version: "1"}, Executable: []byte("x")}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(app, "stale")); err == nil {
		t.Error("an old file stayed in the bundle")
	}
	if _, err := os.Stat(filepath.Join(app, "AppIcon60x60@2x.png")); err == nil {
		t.Error("a bundle written without icons has icon files from the old one")
	}
}

func mustIcon(t *testing.T) []byte {
	t.Helper()
	data, err := iostool.DefaultIcon()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestWriteRejectsAPathThatIsNotABundle(t *testing.T) {
	_, err := iostool.Write(filepath.Join(t.TempDir(), "Counter"), iostool.Bundle{Info: iostool.Info{Executable: "a", ID: "dev.glutter.a", Name: "A", Version: "1"}})
	requireRoute(t, err, "Write."+iostool.TagIOSBundleInvalid)
	_, err = iostool.Write(filepath.Join(t.TempDir(), "A.app"), iostool.Bundle{Info: iostool.Info{Executable: "a", ID: "bad", Name: "A", Version: "1"}})
	requireRoute(t, err, "Write->InfoPlist."+iostool.TagIOSBundleInvalid)
}

func TestSimctlCommandLines(t *testing.T) {
	s := iostool.Simctl{Set: "/set", Runner: &fakeRunner{}}
	tests := []struct {
		name string
		got  []string
		want []string
	}{
		{"args with a set", s.Args("boot", "U"), []string{"simctl", "--set", "/set", "boot", "U"}},
		{"args without a set", iostool.Simctl{}.Args("boot", "U"), []string{"simctl", "boot", "U"}},
		{"launch with the console", s.LaunchArgs("U", "dev.x.y", true), []string{"simctl", "--set", "/set", "launch", "--console-pty", "U", "dev.x.y"}},
		{"launch without it", s.LaunchArgs("U", "dev.x.y", false), []string{"simctl", "--set", "/set", "launch", "U", "dev.x.y"}},
	}
	for _, tt := range tests {
		if !slices.Equal(tt.got, tt.want) {
			t.Errorf("%s: %v, want %v", tt.name, tt.got, tt.want)
		}
	}
	f := &fakeRunner{answers: map[string]fakeAnswer{}}
	for _, k := range []string{"create glutter-iphone T R", "bootstatus U -b", "shutdown U", "install U /a.app", "terminate U dev.x.y", "io U screenshot /s.png",
		"get_app_container U dev.x.y data", "delete U"} {
		f.answers["xcrun simctl --set /set "+k] = fakeAnswer{out: "UDID-OR-PATH\n"}
	}
	s = iostool.Simctl{Set: "/set", Runner: f}
	ctx := context.Background()
	if udid, err := s.Create(ctx, "glutter-iphone", "T", "R"); err != nil || udid != "UDID-OR-PATH" {
		t.Errorf("Create = %q, %v", udid, err)
	}
	if dir, err := s.Container(ctx, "U", "dev.x.y"); err != nil || dir != "UDID-OR-PATH" {
		t.Errorf("Container = %q, %v", dir, err)
	}
	for name, err := range map[string]error{"Boot": s.Boot(ctx, "U"), "Shutdown": s.Shutdown(ctx, "U"), "Install": s.Install(ctx, "U", "/a.app"),
		"Terminate": s.Terminate(ctx, "U", "dev.x.y"), "Screenshot": s.Screenshot(ctx, "U", "/s.png"), "Delete": s.Delete(ctx, "U")} {
		if err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	if len(f.calls) != 8 {
		t.Errorf("%d commands, want 8: %v", len(f.calls), f.calls)
	}
}

func TestSimctlErrors(t *testing.T) {
	ctx := context.Background()
	failing := &fakeRunner{answers: map[string]fakeAnswer{"xcrun simctl boot U": {out: "Unable to boot", exit: 164}}}
	err := iostool.Simctl{Runner: failing}.Boot(ctx, "U") // bootstatus is not in the table
	requireRoute(t, err, "Simctl.run."+iostool.TagIOSSimctlFailed)
	appErr, _ := errors.FromError(err)
	if appErr.Metadata["exit"] != 127 {
		t.Errorf("exit = %v, want 127 (the fake's answer for a command it does not know)", appErr.Metadata["exit"])
	}
	cannot := &fakeRunner{answers: map[string]fakeAnswer{"xcrun simctl shutdown U": {err: errors.New(iostool.TagIOSCommandFailed, "no xcrun")}}}
	err = iostool.Simctl{Runner: cannot}.Shutdown(ctx, "U")
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != iostool.TagIOSCommandFailed {
		t.Errorf("a command that cannot run keeps its own tag: %v", err)
	}
	_, err = iostool.ParseDevices([]byte("not json"))
	requireRoute(t, err, "ParseDevices."+iostool.TagIOSSimctlFailed)
	_, err = iostool.ParseRuntimes([]byte("not json"))
	requireRoute(t, err, "ParseRuntimes."+iostool.TagIOSSimctlFailed)
}

func TestExecRunner(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOOS == "windows" {
		t.Skip("runs sh, which Node and Wine do not have")
	}
	r := iostool.ExecRunner{}
	out, exit, err := r.Run(context.Background(), "sh", "-c", "echo out; echo err >&2; exit 3")
	if err != nil || exit != 3 || !strings.Contains(string(out), "out") || !strings.Contains(string(out), "err") {
		t.Errorf("Run = %q, %d, %v", out, exit, err)
	}
	_, _, err = r.Run(context.Background(), "/no/such/program-for-glutter")
	requireRoute(t, err, "ExecRunner.Run."+iostool.TagIOSCommandFailed)
}

const devicesJSON = `{"devices":{
 "com.apple.CoreSimulator.SimRuntime.iOS-27-2":[
  {"name":"glutter-iphone","udid":"U1","state":"Booted"},
  {"name":"glutter-ipad","udid":"U2","state":"Shutdown"}],
 "com.apple.CoreSimulator.SimRuntime.iOS-18-0":[]}}`

func TestParseDevices(t *testing.T) {
	ds, err := iostool.ParseDevices([]byte(devicesJSON))
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 2 {
		t.Fatalf("%d devices, want 2", len(ds))
	}
	slices.SortFunc(ds, func(a, b iostool.Device) int { return strings.Compare(a.Name, b.Name) })
	if ds[1].Name != "glutter-iphone" || ds[1].UDID != "U1" || ds[1].State != "Booted" || !strings.HasSuffix(ds[1].Runtime, "iOS-27-2") {
		t.Errorf("device = %+v", ds[1])
	}
}

const runtimesJSON = `{"runtimes":[
 {"identifier":"com.apple.CoreSimulator.SimRuntime.iOS-14-5","version":"14.5","platform":"iOS","isAvailable":true,"supportedDeviceTypes":[{"identifier":"t.iPhone-12","name":"iPhone 12","productFamily":"iPhone"}]},
 {"identifier":"com.apple.CoreSimulator.SimRuntime.iOS-17-0","version":"17.0","platform":"iOS","isAvailable":true,"supportedDeviceTypes":[{"identifier":"t.iPhone-15","name":"iPhone 15","productFamily":"iPhone"}]},
 {"identifier":"com.apple.CoreSimulator.SimRuntime.iOS-27-2-single","version":"27.2","platform":"iOS","isAvailable":true,"supportedDeviceTypes":[{"identifier":"t.Duo","name":"iPhone Duo","productFamily":"iPhone"}]},
 {"identifier":"com.apple.CoreSimulator.SimRuntime.iOS-27-2","version":"27.2","platform":"iOS","isAvailable":true,"supportedDeviceTypes":[
   {"identifier":"t.iPhone-18-Pro","name":"iPhone 18 Pro","productFamily":"iPhone"},
   {"identifier":"t.iPhone-17","name":"iPhone 17","productFamily":"iPhone"},
   {"identifier":"t.iPad-Pro","name":"iPad Pro 13-inch (M5)","productFamily":"iPad"},
   {"identifier":"t.iPad-A16","name":"iPad (A16)","productFamily":"iPad"}]},
 {"identifier":"com.apple.CoreSimulator.SimRuntime.iOS-28-0","version":"28.0","platform":"iOS","isAvailable":false,"supportedDeviceTypes":[]},
 {"identifier":"com.apple.CoreSimulator.SimRuntime.watchOS-11-0","version":"11.0","platform":"watchOS","isAvailable":true,"supportedDeviceTypes":[]}]}`

func TestChooseRuntimeAndDeviceType(t *testing.T) {
	rs, err := iostool.ParseRuntimes([]byte(runtimesJSON))
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 4 {
		t.Fatalf("%d runtimes, want the 4 iOS ones that are available", len(rs))
	}
	rt, err := iostool.ChooseRuntime(rs)
	if err != nil {
		t.Fatal(err)
	}
	if rt.ID != "com.apple.CoreSimulator.SimRuntime.iOS-27-2" {
		t.Errorf("runtime = %s, want the newest, the one that runs the most models", rt.ID)
	}
	for family, want := range map[string]string{"iPhone": "iPhone 17", "iPad": "iPad (A16)"} {
		dt, err := iostool.ChooseDeviceType(rt, family)
		if err != nil || dt.Name != want {
			t.Errorf("%s: %+v, %v, want %s", family, dt, err, want)
		}
	}
	// Without the preferred model the first of the family is used.
	rt.DeviceTypes = rt.DeviceTypes[:1]
	if dt, err := iostool.ChooseDeviceType(rt, "iPhone"); err != nil || dt.Name != "iPhone 18 Pro" {
		t.Errorf("fallback: %+v, %v", dt, err)
	}
	_, err = iostool.ChooseDeviceType(rt, "iPad")
	requireRoute(t, err, "ChooseDeviceType."+iostool.TagIOSRuntimeMissing)
	_, err = iostool.ChooseRuntime(rs[:1]) // iOS 14.5 is below the minimum
	requireRoute(t, err, "ChooseRuntime."+iostool.TagIOSRuntimeMissing)
	_, err = iostool.ChooseRuntime(nil)
	requireRoute(t, err, "ChooseRuntime."+iostool.TagIOSRuntimeMissing)
}

func TestCheckXcode(t *testing.T) {
	ctx := context.Background()
	tc, err := iostool.CheckXcode(ctx, &fakeRunner{answers: xcodeAnswers()})
	if err != nil {
		t.Fatal(err)
	}
	if tc.Clang != "/X/clang" || tc.DeviceSDK != "/X/iPhoneOS.sdk" || tc.SimulatorSDK != "/X/iPhoneSimulator.sdk" || !strings.Contains(tc.Developer, "Xcode.app") {
		t.Errorf("toolchain = %+v", tc)
	}
	tests := map[string]map[string]fakeAnswer{
		"the Command Line Tools":  {"xcode-select -p": {out: "/Library/Developer/CommandLineTools\n"}},
		"xcode-select fails":      {"xcode-select -p": {exit: 1}},
		"xcode-select is missing": {"xcode-select -p": {err: errors.New(iostool.TagIOSCommandFailed, "none")}},
		"no compiler": func() map[string]fakeAnswer {
			a := xcodeAnswers()
			a["xcrun --sdk iphoneos --find clang"] = fakeAnswer{exit: 1, out: "not found"}
			return a
		}(),
		"no simulator SDK": func() map[string]fakeAnswer {
			a := xcodeAnswers()
			delete(a, "xcrun --sdk iphonesimulator --show-sdk-path")
			return a
		}(),
		"xcrun cannot run": func() map[string]fakeAnswer {
			a := xcodeAnswers()
			a["xcrun --sdk iphoneos --show-sdk-path"] = fakeAnswer{err: errors.New(iostool.TagIOSCommandFailed, "none")}
			return a
		}(),
	}
	for name, answers := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := iostool.CheckXcode(ctx, &fakeRunner{answers: answers})
			// The failures of xcrun are made in the function inside CheckXcode that asks for each path.
			if appErr, ok := errors.FromError(err); !ok || appErr.Tag != iostool.TagIOSXcodeMissing || !appErr.HasRoute("CheckXcode") {
				t.Errorf("the error is %v, want IOSXcodeMissing from CheckXcode", err)
			}
		})
	}
}

func TestEnv(t *testing.T) {
	tc := iostool.Toolchain{Clang: "/X/clang", DeviceSDK: "/X/dev.sdk", SimulatorSDK: "/X/sim.sdk"}
	device := iostool.Env(tc, false)
	sim := iostool.Env(tc, true)
	for _, e := range []string{"GOOS=ios", "GOARCH=arm64", "CGO_ENABLED=1"} {
		if !slices.Contains(device, e) || !slices.Contains(sim, e) {
			t.Errorf("%s is not in both environments", e)
		}
	}
	if !slices.Contains(device, "CGO_LDFLAGS=-isysroot /X/dev.sdk -target arm64-apple-ios15.0") {
		t.Errorf("device = %v", device)
	}
	if !slices.Contains(sim, "CGO_LDFLAGS=-isysroot /X/sim.sdk -target arm64-apple-ios15.0-simulator") {
		t.Errorf("simulator = %v", sim)
	}
	if !slices.Contains(sim, "CC=/X/clang -isysroot /X/sim.sdk -target arm64-apple-ios15.0-simulator") {
		t.Errorf("simulator CC = %v", sim)
	}
}

func TestSign(t *testing.T) {
	ctx := context.Background()
	f := &fakeRunner{answers: map[string]fakeAnswer{"/usr/bin/codesign --force --timestamp=none --sign - /a/Hello.app": {}}}
	if err := iostool.Sign(ctx, "/a/Hello.app", iostool.SignOptions{Runner: f}); err != nil {
		t.Fatal(err)
	}
	// For a device the identity and the entitlements are given.
	f = &fakeRunner{answers: map[string]fakeAnswer{}}
	_ = iostool.Sign(ctx, "/a/Hello.app", iostool.SignOptions{Team: "ABCDE12345", ID: "dev.glutter.hello", Runner: f})
	if len(f.calls) != 1 || !strings.Contains(f.calls[0], "--sign Apple Development --entitlements ") || !strings.HasSuffix(f.calls[0], " /a/Hello.app") {
		t.Errorf("calls = %v", f.calls)
	}
	err := iostool.Sign(ctx, "/a/Hello.app", iostool.SignOptions{Runner: &fakeRunner{answers: map[string]fakeAnswer{
		"/usr/bin/codesign --force --timestamp=none --sign - /a/Hello.app": {exit: 1, out: "no identity"}}}})
	requireRoute(t, err, "Sign."+iostool.TagIOSSignFailed)
}

func TestBuildRefusesWhatCannotGoTogether(t *testing.T) {
	ctx := context.Background()
	tests := map[string]iostool.BuildOptions{
		"no package":                  {},
		"a team without a profile":    {Package: "./x", Team: "ABCDE12345"},
		"a profile without a team":    {Package: "./x", Profile: "p.mobileprovision"},
		"a team for the Simulator":    {Package: "./x", Simulator: true, Team: "ABCDE12345", Profile: "p.mobileprovision"},
		"a profile that is not there": {Package: "./x", Team: "ABCDE12345", Profile: "/no/such/profile"},
	}
	for name, o := range tests {
		t.Run(name, func(t *testing.T) {
			o.Runner = &fakeRunner{answers: xcodeAnswers()}
			_, err := iostool.Build(ctx, o)
			requireRoute(t, err, "Build."+iostool.TagIOSBuildFailed)
		})
	}
}

func TestBuildFailsForAPackageThatDoesNotBuild(t *testing.T) {
	_, err := iostool.Build(context.Background(), iostool.BuildOptions{Package: "./does/not/exist", Dir: t.TempDir(), Out: filepath.Join(t.TempDir(), "X.app"),
		Simulator: true, Runner: &fakeRunner{answers: xcodeAnswers()}})
	requireRoute(t, err, "Build->BuildOptions.compile."+iostool.TagIOSBuildFailed)
}

func TestBuildChecksTheIdentityAndTheIconBeforeTheCompiler(t *testing.T) {
	ctx := context.Background()
	r := &fakeRunner{answers: xcodeAnswers()}
	_, err := iostool.Build(ctx, iostool.BuildOptions{Package: "./x", ID: "bad id", Simulator: true, Runner: r})
	requireRoute(t, err, "Build->InfoPlist."+iostool.TagIOSBundleInvalid)
	_, err = iostool.Build(ctx, iostool.BuildOptions{Package: "./x", Icon: "/no/such/icon.png", Simulator: true, Runner: r})
	requireRoute(t, err, "BuildOptions.iconPNG."+iostool.TagIOSIconInvalid)
	bad := filepath.Join(t.TempDir(), "icon.png")
	if err := os.WriteFile(bad, []byte("not a png"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = iostool.Build(ctx, iostool.BuildOptions{Package: "./x", Icon: bad, Simulator: true, Runner: r})
	requireRoute(t, err, "Build->Icons."+iostool.TagIOSIconInvalid)
	if len(r.calls) != 0 {
		t.Errorf("Xcode was asked about before the identity and the icon were checked: %v", r.calls)
	}
}

func TestDefaultName(t *testing.T) {
	if got := iostool.DefaultName("./examples/counter", ""); got != "counter" {
		t.Errorf("DefaultName = %q", got)
	}
	dir := filepath.Join(t.TempDir(), "myapp")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := iostool.DefaultName(".", dir); got != "myapp" {
		t.Errorf("DefaultName for . = %q", got)
	}
}
