package mactool

import (
	"bytes"
	"context"
	"debug/macho"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

func routeOf(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("error = %T %v", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("route = %s, want %s", appErr.Route(), route)
	}
}

func TestInfoPlist(t *testing.T) {
	i := Info{Executable: "counter", ID: "dev.glutter.counter", Name: "Tom & Jerry <1>", Version: "1.2.3", IconFile: "counter",
		Usage: map[string]string{"NSMicrophoneUsageDescription": `Records "sound"`, "NSCameraUsageDescription": "Takes pictures"}}
	data, err := InfoPlist(i)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>`, `<plist version="1.0">`,
		"<key>CFBundleExecutable</key>\n\t<string>counter</string>",
		"<key>CFBundleIdentifier</key>\n\t<string>dev.glutter.counter</string>",
		"<key>CFBundlePackageType</key>\n\t<string>APPL</string>",
		"<key>CFBundleShortVersionString</key>\n\t<string>1.2.3</string>",
		"<key>CFBundleVersion</key>\n\t<string>1.2.3</string>",
		"<key>CFBundleIconFile</key>\n\t<string>counter</string>",
		"<key>LSMinimumSystemVersion</key>\n\t<string>12.0</string>",
		"<key>NSHighResolutionCapable</key>\n\t<true/>",
		"<key>NSPrincipalClass</key>\n\t<string>NSApplication</string>",
		// The name and the usage strings are escaped.
		"<string>Tom &amp; Jerry &lt;1&gt;</string>", "&#34;sound&#34;",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the plist has no %q:\n%s", want, got)
		}
	}
	// The usage strings are sorted by key, after the fixed ones.
	if c, m := strings.Index(got, "NSCameraUsageDescription"), strings.Index(got, "NSMicrophoneUsageDescription"); c < 0 || m < c {
		t.Errorf("usage keys: camera at %d, microphone at %d", c, m)
	}
	again, _ := InfoPlist(i)
	if !bytes.Equal(data, again) {
		t.Error("the plist is not the same on a second call")
	}
	// Without an icon the key is not there.
	i.IconFile = ""
	data, _ = InfoPlist(i)
	if strings.Contains(string(data), "CFBundleIconFile") {
		t.Error("an icon key without an icon")
	}
}

func TestInfoPlistRefusesWhatABundleCannotHave(t *testing.T) {
	good := Info{Executable: "app", ID: "dev.glutter.app", Name: "App", Version: "1.0.0"}
	tests := []struct {
		name string
		mod  func(i *Info)
	}{
		{"no executable", func(i *Info) { i.Executable = "" }},
		{"a slash in the executable", func(i *Info) { i.Executable = "a/b" }},
		{"no name", func(i *Info) { i.Name = "" }},
		{"a colon in the name", func(i *Info) { i.Name = "a:b" }},
		{"an id of one part", func(i *Info) { i.ID = "app" }},
		{"an id with a space", func(i *Info) { i.ID = "dev.glutter.my app" }},
		{"an id with a slash", func(i *Info) { i.ID = "dev/glutter.app" }},
		{"no version", func(i *Info) { i.Version = "" }},
		{"a version with a bracket", func(i *Info) { i.Version = "<1>" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := good
			tt.mod(&i)
			_, err := InfoPlist(i)
			routeOf(t, err, "InfoPlist.MacOSBundleInvalid")
		})
	}
	if _, err := InfoPlist(good); err != nil {
		t.Errorf("the good identity: %v", err)
	}
}

func TestDefaults(t *testing.T) {
	ids := map[string]string{"counter": "dev.glutter.counter", "My App_2": "dev.glutter.my-app-2", "été": "dev.glutter.t", "日本語": "dev.glutter.app", "": "dev.glutter.app", "a.b": "dev.glutter.a-b"}
	for name, want := range ids {
		if got := DefaultID(name); got != want {
			t.Errorf("DefaultID(%q) = %q, want %q", name, got, want)
		}
	}
	if u := DefaultUsage("Counter"); !strings.Contains(u["NSMicrophoneUsageDescription"], "Counter") {
		t.Errorf("usage = %v", u)
	}
	if got := DefaultName("./examples/counter", ""); got != "counter" {
		t.Errorf("DefaultName = %q", got)
	}
	dir := filepath.Join(t.TempDir(), "myapp")
	if got := DefaultName(".", dir); got != "myapp" {
		t.Errorf("DefaultName(., dir) = %q", got)
	}
	if string(Entitlements()) == "" || !strings.Contains(string(Entitlements()), "com.apple.security.device.audio-input") {
		t.Error("entitlements")
	}
}

// squarePNG makes a PNG of a size with a red half and a blue half.
func squarePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			if x < w/2 {
				img.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
			} else {
				img.SetNRGBA(x, y, color.NRGBA{B: 255, A: 255})
			}
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestICNSReadsBack(t *testing.T) {
	data, err := ICNS(squarePNG(t, 256, 256))
	if err != nil {
		t.Fatal(err)
	}
	if string(data[:4]) != "icns" || int(binary.BigEndian.Uint32(data[4:])) != len(data) {
		t.Fatalf("header %q length %d of %d", data[:4], binary.BigEndian.Uint32(data[4:]), len(data))
	}
	var got []icnsEntry
	for off := 8; off < len(data); {
		typ := string(data[off : off+4])
		n := int(binary.BigEndian.Uint32(data[off+4:]))
		img, err := png.Decode(bytes.NewReader(data[off+8 : off+n]))
		if err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
		b := img.Bounds()
		if b.Dx() != b.Dy() {
			t.Errorf("%s is %dx%d", typ, b.Dx(), b.Dy())
		}
		got = append(got, icnsEntry{typ, b.Dx()})
		// The picture is kept: red on the left, blue on the right.
		l, r := color.NRGBAModel.Convert(img.At(b.Dx()/8, b.Dy()/2)).(color.NRGBA), color.NRGBAModel.Convert(img.At(b.Dx()*7/8, b.Dy()/2)).(color.NRGBA)
		if l.R < 200 || l.B > 60 || r.B < 200 || r.R > 60 {
			t.Errorf("%s: left %v right %v", typ, l, r)
		}
		off += n
	}
	if len(got) != len(icnsEntries) {
		t.Fatalf("entries = %v", got)
	}
	for i := range got {
		if got[i] != icnsEntries[i] {
			t.Errorf("entry %d = %v, want %v", i, got[i], icnsEntries[i])
		}
	}
}

func TestICNSRefuses(t *testing.T) {
	_, err := ICNS([]byte("not a picture"))
	routeOf(t, err, "ICNS.MacOSIconInvalid")
	_, err = ICNS(squarePNG(t, 64, 32))
	routeOf(t, err, "ICNS.MacOSIconInvalid")
	_, err = ICNS(squarePNG(t, 8, 8))
	routeOf(t, err, "ICNS.MacOSIconInvalid")
	if _, err = ICNS(squarePNG(t, 16, 16)); err != nil {
		t.Errorf("a 16 pixel icon: %v", err)
	}
}

func TestDefaultIconIsAnIcon(t *testing.T) {
	data, err := DefaultIcon()
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 1024 || img.Bounds().Dy() != 1024 {
		t.Errorf("size = %v", img.Bounds())
	}
	// The corners are clear (the margin and the rounding), the centre of the left of the ring is white, the field is blue.
	if _, _, _, a := img.At(2, 2).RGBA(); a != 0 {
		t.Errorf("the corner is not clear: alpha %d", a)
	}
	if c := color.NRGBAModel.Convert(img.At(200, 200)).(color.NRGBA); c.A != 255 || c.B < 200 || c.R > 60 {
		t.Errorf("the field = %v", c)
	}
	if _, err := ICNS(data); err != nil {
		t.Errorf("the default icon makes no .icns: %v", err)
	}
}

// thinMachO makes the smallest file that debug/macho reads: the header of a 64-bit Mach-O with no commands.
func thinMachO(cpu macho.Cpu, sub uint32, pad int) []byte {
	b := make([]byte, 32, 32+pad)
	binary.LittleEndian.PutUint32(b[0:], macho.Magic64)
	binary.LittleEndian.PutUint32(b[4:], uint32(cpu))
	binary.LittleEndian.PutUint32(b[8:], sub)
	binary.LittleEndian.PutUint32(b[12:], uint32(macho.TypeExec))
	return append(b, bytes.Repeat([]byte{0xAB}, pad)...)
}

func TestFatReadsBack(t *testing.T) {
	arm := thinMachO(macho.CpuArm64, 0, 5000)
	amd := thinMachO(macho.CpuAmd64, 3, 100)
	data, err := Fat([]FatSlice{{"arm64", arm}, {"amd64", amd}})
	if err != nil {
		t.Fatal(err)
	}
	ff, err := macho.NewFatFile(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(ff.Arches) != 2 || ff.Arches[0].Cpu != macho.CpuArm64 || ff.Arches[1].Cpu != macho.CpuAmd64 || ff.Arches[1].SubCpu != 3 {
		t.Fatalf("arches = %+v", ff.Arches)
	}
	// The slices start where their alignment says (16 KB for arm64, 4 KB for amd64) and hold the bytes that were given.
	off0 := binary.BigEndian.Uint32(data[8+8:])
	off1 := binary.BigEndian.Uint32(data[8+fatArchSize+8:])
	if off0%(1<<14) != 0 || off1%(1<<12) != 0 || off1 < off0+uint32(len(arm)) {
		t.Errorf("offsets %d, %d", off0, off1)
	}
	if !bytes.Equal(data[off0:off0+uint32(len(arm))], arm) || !bytes.Equal(data[off1:off1+uint32(len(amd))], amd) {
		t.Error("the slices are not what was given")
	}
	if binary.BigEndian.Uint32(data[8+16:]) != 14 || binary.BigEndian.Uint32(data[8+fatArchSize+16:]) != 12 {
		t.Error("the alignments are not 2^14 and 2^12")
	}
	for i, a := range ff.Arches {
		if a.Cpu == macho.CpuArm64 && a.Size != uint32(len(arm)) || a.Cpu == macho.CpuAmd64 && a.Size != uint32(len(amd)) {
			t.Errorf("slice %d size %d", i, a.Size)
		}
	}
}

func TestFatRefuses(t *testing.T) {
	_, err := Fat(nil)
	routeOf(t, err, "Fat.MacOSMachOInvalid")
	_, err = Fat([]FatSlice{{"junk", []byte("this is not a Mach-O file at all, but it is long")}})
	routeOf(t, err, "Fat.MacOSMachOInvalid")
	arm := thinMachO(macho.CpuArm64, 0, 8)
	_, err = Fat([]FatSlice{{"one", arm}, {"two", arm}})
	routeOf(t, err, "Fat.MacOSMachOInvalid")
}

func TestBundleLayoutAndReplacement(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "Demo.app")
	info := Info{Executable: "demo", ID: "dev.glutter.demo", Name: "Demo", Version: "2.0.0", IconFile: "demo"}
	l, err := Write(app, Bundle{Info: info, Executable: []byte("exe"), Icon: []byte("icns")})
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{l.Executable: "exe", l.Icon: "icns", filepath.Join(l.Contents, "PkgInfo"): "APPL????"} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v", path, got, err)
		}
	}
	if st, err := os.Stat(l.Executable); err != nil || st.Mode().Perm() != 0o755 {
		t.Errorf("the executable's mode: %v %v", st, err)
	}
	if l.Executable != filepath.Join(app, "Contents", "MacOS", "demo") || l.Icon != filepath.Join(app, "Contents", "Resources", "demo.icns") || l.Plist != filepath.Join(app, "Contents", "Info.plist") {
		t.Errorf("layout = %+v", l)
	}
	plist, err := os.ReadFile(l.Plist)
	if err != nil || !strings.Contains(string(plist), "<string>2.0.0</string>") {
		t.Errorf("plist: %v", err)
	}
	// A second bundle replaces the first: what was in the old one and not in the new is gone.
	if err := os.WriteFile(filepath.Join(app, "Contents", "stale"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Write(app, Bundle{Info: info, Executable: []byte("exe2")}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(app, "Contents", "stale")); err == nil {
		t.Error("the old bundle's file is still there")
	}
	if got, _ := os.ReadFile(l.Executable); string(got) != "exe2" {
		t.Errorf("executable = %q", got)
	}
}

func TestBundleRefuses(t *testing.T) {
	dir := t.TempDir()
	info := Info{Executable: "demo", ID: "dev.glutter.demo", Name: "Demo", Version: "1.0.0", IconFile: "demo"}
	tests := []struct {
		name  string
		path  string
		b     Bundle
		route string
	}{
		{"a name that is not .app", filepath.Join(dir, "Demo"), Bundle{Info: info, Executable: []byte("x")}, "Write.MacOSBundleInvalid"},
		{"no executable", filepath.Join(dir, "A.app"), Bundle{Info: info}, "Write.MacOSBundleInvalid"},
		{"an icon without a name", filepath.Join(dir, "B.app"), Bundle{Info: Info{Executable: "d", ID: "dev.glutter.d", Name: "D", Version: "1"}, Executable: []byte("x"), Icon: []byte("i")}, "Write.MacOSBundleInvalid"},
		{"a bad identity", filepath.Join(dir, "C.app"), Bundle{Info: Info{Executable: "d", ID: "x", Name: "D", Version: "1"}, Executable: []byte("x")}, "Write->InfoPlist.MacOSBundleInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Write(tt.path, tt.b)
			routeOf(t, err, tt.route)
		})
	}
	// A directory that is called .app and is not a bundle is not removed.
	notBundle := filepath.Join(dir, "Mine.app")
	if err := os.MkdirAll(filepath.Join(notBundle, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Write(notBundle, Bundle{Info: info, Executable: []byte("x")})
	routeOf(t, err, "Write.MacOSBundleInvalid")
	if _, err := os.Stat(filepath.Join(notBundle, "docs")); err != nil {
		t.Error("a directory that is not a bundle was removed")
	}
	// A file at the path is not replaced either.
	file := filepath.Join(dir, "File.app")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Write(file, Bundle{Info: info, Executable: []byte("x")})
	routeOf(t, err, "Write.MacOSBundleInvalid")
}

// tinyModule writes a module with a program that prints a line.
func tinyModule(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tiny\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const tinyMain = "package main\n\nfunc main() { println(\"hello\") }\n"

func TestBuildMakesAUniversalSignedBundle(t *testing.T) {
	needGo(t)
	dir := tinyModule(t, tinyMain)
	out := filepath.Join(t.TempDir(), "Tiny.app")
	o := BuildOptions{Package: ".", Dir: dir, Out: out, Archs: []string{"arm64", "amd64"}, Version: "3.1.4"}
	if runtime.GOOS != "darwin" {
		// codesign is a Mac's (SPEC §29.1): elsewhere a program that does nothing stands in for it, and the rest of the bundle is
		// checked.
		tool, err := exec.LookPath("true")
		if err != nil {
			t.Skip("no codesign and no true to stand in for it")
		}
		o.Sign.Tool = tool
	}
	res, err := Build(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Info.Name != filepath.Base(dir) || res.Info.ID != DefaultID(filepath.Base(dir)) || res.Info.Version != "3.1.4" {
		t.Errorf("info = %+v", res.Info)
	}
	if len(res.Archs) != 2 || res.Thin["arm64"] == 0 || res.Thin["amd64"] == 0 || res.Size < res.Thin["arm64"]+res.Thin["amd64"] {
		t.Errorf("result = %+v", res)
	}
	ff, err := macho.OpenFat(res.Layout.Executable)
	if err != nil {
		t.Fatalf("the executable is not a universal Mach-O: %v", err)
	}
	defer ff.Close()
	if len(ff.Arches) != 2 || ff.Arches[0].Cpu != macho.CpuArm64 || ff.Arches[1].Cpu != macho.CpuAmd64 {
		t.Errorf("arches = %+v", ff.Arches)
	}
	icns, err := os.ReadFile(res.Layout.Icon)
	if err != nil || string(icns[:4]) != "icns" {
		t.Errorf("the icon: %v", err)
	}
	// The bundle is signed: on a Mac codesign says it holds.
	if runtime.GOOS == "darwin" {
		if err := Verify(context.Background(), res.Layout.App, SignOptions{}); err != nil {
			t.Errorf("the bundle does not verify: %v", err)
		}
	}
}

// needGo skips a test that runs the go command, which a host such as js/wasm cannot.
func needGo(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "js" || runtime.GOOS == "wasip1" {
		t.Skip("this host cannot run the go command")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go command")
	}
}

func TestBuildThinRunsOnThisMac(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("runs the program: a Mac only")
	}
	dir := tinyModule(t, tinyMain)
	out := filepath.Join(t.TempDir(), "Hello.app")
	res, err := Build(context.Background(), BuildOptions{Package: ".", Dir: dir, Out: out, Name: "Hello", ID: "dev.glutter.hello", Hardened: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Archs) != 1 || res.Archs[0] != runtime.GOARCH {
		t.Errorf("archs = %v", res.Archs)
	}
	got, err := exec.Command(res.Layout.Executable).CombinedOutput()
	if err != nil || strings.TrimSpace(string(got)) != "hello" {
		t.Errorf("the bundled program printed %q, %v", got, err)
	}
	if err := Verify(context.Background(), res.Layout.App, SignOptions{}); err != nil {
		t.Errorf("verify: %v", err)
	}
	// A change after signing breaks the signature.
	if err := os.WriteFile(filepath.Join(res.Layout.Resources, "extra.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	routeOf(t, Verify(context.Background(), res.Layout.App, SignOptions{}), "Verify.MacOSSignFailed")
}

func TestBuildErrors(t *testing.T) {
	needGo(t)
	ctx := context.Background()
	dir := tinyModule(t, "package main\n\nfunc main() { undefined() }\n")
	tests := []struct {
		name  string
		o     BuildOptions
		route string
	}{
		{"no package", BuildOptions{}, "Build.MacOSBuildFailed"},
		{"an architecture that is not one", BuildOptions{Package: ".", Dir: dir, Archs: []string{"ppc"}}, "Build.MacOSBuildFailed"},
		{"a bad identity", BuildOptions{Package: ".", Dir: dir, ID: "nodots"}, "InfoPlist.MacOSBundleInvalid"},
		{"an icon that is not there", BuildOptions{Package: ".", Dir: dir, Icon: filepath.Join(dir, "nope.png")}, "BuildOptions.iconPNG.MacOSIconInvalid"},
		{"an icon that is not a picture", BuildOptions{Package: ".", Dir: dir, Icon: filepath.Join(dir, "main.go")}, "ICNS.MacOSIconInvalid"},
		{"a program that does not compile", BuildOptions{Package: ".", Dir: dir, Out: filepath.Join(t.TempDir(), "X.app")}, "BuildOptions.compile.MacOSBuildFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Build(ctx, tt.o)
			routeOf(t, err, tt.route)
		})
	}
	// The compiler's output is in the error.
	_, err := Build(ctx, BuildOptions{Package: ".", Dir: dir, Out: filepath.Join(t.TempDir(), "X.app")})
	if appErr, _ := errors.FromError(err); appErr == nil || !strings.Contains(appErr.Metadata["output"].(string), "undefined") {
		t.Errorf("error = %v", err)
	}
}

func TestSignRefusesWhatCodesignRefuses(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("codesign is a Mac's")
	}
	err := Sign(context.Background(), filepath.Join(t.TempDir(), "Nope.app"), SignOptions{})
	routeOf(t, err, "Sign.MacOSSignFailed")
	err = Sign(context.Background(), "x.app", SignOptions{Tool: filepath.Join(t.TempDir(), "no-such-codesign")})
	routeOf(t, err, "Sign.MacOSSignFailed")
	routeOf(t, Verify(context.Background(), filepath.Join(t.TempDir(), "Nope.app"), SignOptions{}), "Verify.MacOSSignFailed")
}
