package androidtool

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

// fakeRunner stands in for the programs of the build: it records the command lines and makes the
// files each program would make.
type fakeRunner struct {
	cmds []Command
	fail string // a program name (base) that fails
}

func (f *fakeRunner) Run(_ context.Context, c Command) (string, error) {
	f.cmds = append(f.cmds, c)
	name := filepath.Base(c.Name)
	if name == f.fail {
		return "boom", errors.New(TagAndroidCommandFailed, "failed", "command", c.String())
	}
	arg := func(flag string) string {
		for i, a := range c.Args {
			if a == flag && i+1 < len(c.Args) {
				return c.Args[i+1]
			}
		}
		return ""
	}
	switch name {
	case "go":
		return "", os.WriteFile(arg("-o"), []byte("ELF "+arg("-o")), 0o644)
	case "aapt2":
		if c.Args[0] == "link" {
			zw, err := os.Create(arg("-o"))
			if err != nil {
				return "", err
			}
			w := zip.NewWriter(zw)
			for _, n := range []string{"AndroidManifest.xml", "resources.arsc"} {
				e, _ := w.Create(n)
				_, _ = e.Write([]byte(n))
			}
			if err := w.Close(); err != nil {
				return "", err
			}
			return "", zw.Close()
		}
		return "", os.WriteFile(arg("-o"), nil, 0o644)
	case "zipalign":
		b, err := os.ReadFile(c.Args[len(c.Args)-2])
		if err != nil {
			return "", err
		}
		return "", os.WriteFile(c.Args[len(c.Args)-1], b, 0o644)
	case "apksigner":
		b, err := os.ReadFile(c.Args[len(c.Args)-1])
		if err != nil {
			return "", err
		}
		return "", os.WriteFile(arg("--out"), b, 0o644)
	}
	return "", nil
}

func testSDK() *SDK {
	return &SDK{Home: "/sdk", NDK: "/ndk", BuildTools: "/bt", Platform: "/plat"}
}

func TestBuildCommandLines(t *testing.T) {
	tests := []struct {
		name    string
		opts    BuildOptions
		abis    []string // ABI names built, in order
		goArgs  []string // the words every go command has
		noGoArg []string
	}{
		{name: "debug, both ABIs", opts: BuildOptions{Package: "./examples/counter"}, abis: []string{"arm64-v8a", "x86_64"},
			goArgs: []string{"build", "-buildmode=c-shared"}, noGoArg: []string{"glutter_release"}},
		{name: "release, one ABI", opts: BuildOptions{Package: "./examples/counter", ABIs: []string{"amd64"}, Release: true, Key: "k", Cert: "c"},
			abis: []string{"x86_64"}, goArgs: []string{"-tags", "glutter_release", "-ldflags=-s -w"}},
		{name: "ABI by its Android name", opts: BuildOptions{Package: "./examples/counter", ABIs: []string{"arm64-v8a"}}, abis: []string{"arm64-v8a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.opts.Out = filepath.Join(dir, "app.apk")
			tt.opts.ConfigDir = filepath.Join(dir, "key")
			tt.opts.Dex = []byte("dex")
			run := &fakeRunner{}
			res, err := Build(context.Background(), testSDK(), run, tt.opts)
			if err != nil {
				t.Fatal(err)
			}
			var built []string
			for _, c := range run.cmds {
				if c.Name != "go" {
					continue
				}
				joined := " " + strings.Join(c.Args, " ") + " "
				for _, w := range tt.goArgs {
					if !strings.Contains(joined, " "+w+" ") {
						t.Errorf("go %s has no %q", joined, w)
					}
				}
				for _, w := range tt.noGoArg {
					if strings.Contains(joined, w) {
						t.Errorf("go %s has %q", joined, w)
					}
				}
				env := strings.Join(c.Env, "\n")
				var abi string
				for _, a := range ABIs {
					if strings.Contains(env, "GOARCH="+a.GOARCH+"\n") || strings.HasSuffix(env, "GOARCH="+a.GOARCH) {
						abi = a.Name
					}
				}
				if !strings.Contains(env, "GOOS=android") || !strings.Contains(env, "CGO_ENABLED=1") ||
					!strings.Contains(env, "CC=/ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/"+ABITriple(abi)+"26-clang") {
					t.Errorf("the environment of go for %s is wrong:\n%s", abi, env)
				}
				built = append(built, abi)
			}
			if strings.Join(built, ",") != strings.Join(tt.abis, ",") {
				t.Errorf("built ABIs %v, want %v", built, tt.abis)
			}
			// The rest of the pipeline, in order.
			var rest []string
			for _, c := range run.cmds {
				if c.Name != "go" {
					rest = append(rest, filepath.Base(c.Name)+" "+c.Args[0])
				}
			}
			want := "aapt2 compile,aapt2 link,zipalign -f,apksigner sign"
			if strings.Join(rest, ",") != want {
				t.Errorf("commands after go: %v, want %s", rest, want)
			}
			last := run.cmds[len(run.cmds)-1]
			if joined := strings.Join(last.Args, " "); !strings.Contains(joined, "--min-sdk-version 26") || !strings.Contains(joined, "--out "+tt.opts.Out) {
				t.Errorf("apksigner: %s", joined)
			}
			for _, c := range run.cmds {
				if filepath.Base(c.Name) == "zipalign" && strings.Join(c.Args[:4], " ") != "-f -P 16 4" {
					t.Errorf("zipalign: %v", c.Args)
				}
				if filepath.Base(c.Name) == "aapt2" && c.Args[0] == "link" {
					if j := strings.Join(c.Args, " "); !strings.Contains(j, "--min-sdk-version 26 --target-sdk-version 37") || !strings.Contains(j, "-I /plat/android.jar") {
						t.Errorf("aapt2 link: %s", j)
					}
				}
			}
			if res.AppID != DefaultAppID("./examples/counter") || len(res.LibSizes) != len(tt.abis) {
				t.Errorf("result %+v", res)
			}
		})
	}
}

func TestBuildZipLayout(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out", "app.apk")
	_, err := Build(context.Background(), testSDK(), &fakeRunner{}, BuildOptions{Package: "./x", Out: out, ConfigDir: filepath.Join(dir, "key"), Dex: []byte("DEXDATA")})
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	methods := map[string]uint16{}
	var names []string
	for _, f := range zr.File {
		methods[f.Name] = f.Method
		names = append(names, f.Name)
	}
	for _, n := range []string{"AndroidManifest.xml", "resources.arsc", "classes.dex", "lib/arm64-v8a/libapp.so", "lib/x86_64/libapp.so"} {
		if _, ok := methods[n]; !ok {
			t.Errorf("the APK has no %s: %v", n, names)
		}
	}
	for _, n := range []string{"lib/arm64-v8a/libapp.so", "lib/x86_64/libapp.so"} {
		if methods[n] != zip.Store {
			t.Errorf("%s is compressed (method %d); the system maps it from the APK", n, methods[n])
		}
	}
}

func TestBuildUsesTheCheckedInDex(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "app.apk")
	if _, err := Build(context.Background(), testSDK(), &fakeRunner{}, BuildOptions{Package: "./x", Out: out, ABIs: []string{"amd64"}, ConfigDir: filepath.Join(dir, "k")}); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	for _, f := range zr.File {
		if f.Name == "classes.dex" && int(f.UncompressedSize64) != len(Dex) {
			t.Errorf("classes.dex has %d bytes, the embedded one %d", f.UncompressedSize64, len(Dex))
		}
	}
	if len(Dex) < 1000 {
		t.Errorf("the embedded classes.dex has %d bytes: run go run ./cmd/glutter-gen android-dex", len(Dex))
	}
}

func TestBuildErrors(t *testing.T) {
	good := func() BuildOptions {
		d := t.TempDir()
		return BuildOptions{Package: "./x", Out: filepath.Join(d, "a.apk"), ConfigDir: filepath.Join(d, "k"), Dex: []byte("d")}
	}
	tests := []struct {
		name  string
		opts  func() BuildOptions
		fail  string
		route string
	}{
		{name: "no package", opts: func() BuildOptions { o := good(); o.Package = ""; return o }, route: "Build." + TagAndroidBuildInvalid},
		{name: "unknown ABI", opts: func() BuildOptions { o := good(); o.ABIs = []string{"mips"}; return o }, route: "Build->selectABIs." + TagAndroidBuildInvalid},
		{name: "release without a key", opts: func() BuildOptions { o := good(); o.Release = true; return o }, route: "Build." + TagAndroidBuildInvalid},
		{name: "bad application ID", opts: func() BuildOptions { o := good(); o.AppID = "nodots"; return o }, route: "Build." + TagAndroidBuildInvalid},
		{name: "missing icon", opts: func() BuildOptions { o := good(); o.Icon = "/no/such/icon.png"; return o }, route: "Build." + TagAndroidWriteFailed},
		{name: "go fails", opts: good, fail: "go", route: "Build->fakeRunner.Run." + TagAndroidCommandFailed},
		{name: "aapt2 fails", opts: good, fail: "aapt2", route: "Build->fakeRunner.Run." + TagAndroidCommandFailed},
		{name: "zipalign fails", opts: good, fail: "zipalign", route: "Build->fakeRunner.Run." + TagAndroidCommandFailed},
		{name: "apksigner fails", opts: good, fail: "apksigner", route: "Build->fakeRunner.Run." + TagAndroidCommandFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Build(context.Background(), testSDK(), &fakeRunner{fail: tt.fail}, tt.opts())
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("route %s, want %s", appErr.Route(), tt.route)
			}
		})
	}
}

func TestBuildWriteFailures(t *testing.T) {
	// The output directory is a file, so it cannot be made.
	d := t.TempDir()
	blocker := filepath.Join(d, "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Build(context.Background(), testSDK(), &fakeRunner{}, BuildOptions{Package: "./x", Out: filepath.Join(blocker, "a.apk"), ConfigDir: filepath.Join(d, "k"), Dex: []byte("d")})
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Build."+TagAndroidWriteFailed) {
		t.Errorf("got %v", err)
	}
	// The debug key's directory is a file.
	_, err = Build(context.Background(), testSDK(), &fakeRunner{}, BuildOptions{Package: "./x", Out: filepath.Join(d, "a.apk"), ConfigDir: filepath.Join(blocker, "k"), Dex: []byte("d")})
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Build->EnsureDebugKey."+TagAndroidKeyFailed) {
		t.Errorf("got %v", err)
	}
}

func TestEnsureDebugKeyIsMadeOnceAndReused(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "android-debug")
	k1, err := EnsureDebugKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	b1, err := os.ReadFile(k1.Key)
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(k1.Key)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("the private key has mode %v", st.Mode().Perm())
	}
	k2, err := EnsureDebugKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := os.ReadFile(k2.Key)
	if err != nil {
		t.Fatal(err)
	}
	if string(b1) != string(b2) || k1.Cert != k2.Cert {
		t.Error("the second call made a new key")
	}
	cert, err := os.ReadFile(k1.Cert)
	if err != nil || !strings.HasPrefix(string(cert), "-----BEGIN CERTIFICATE-----") {
		t.Errorf("certificate: %v\n%s", err, cert)
	}
}

func TestDefaultIconIsAPNG(t *testing.T) {
	b, err := DefaultIcon()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "\x89PNG\r\n\x1a\n") {
		t.Errorf("not a PNG: % x", b[:8])
	}
}

// -vulkan-validation packs the layer of each ABI next to libapp.so, stored, where the Vulkan loader
// finds the layer of a debuggable app (SPEC §24.13).
func TestBuildPacksTheValidationLayer(t *testing.T) {
	dir := t.TempDir()
	layers := filepath.Join(dir, "vvl")
	for _, abi := range []string{"arm64-v8a", "x86_64"} {
		if err := os.MkdirAll(filepath.Join(layers, abi), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(layers, abi, ValidationLayerFile), []byte("LAYER "+abi), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out := filepath.Join(dir, "app.apk")
	_, err := Build(context.Background(), testSDK(), &fakeRunner{}, BuildOptions{Package: "./x", Out: out, ConfigDir: filepath.Join(dir, "k"), Dex: []byte("d"), ValidationLayerDir: layers})
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	found := map[string]*zip.File{}
	for _, f := range zr.File {
		found[f.Name] = f
	}
	for _, abi := range []string{"arm64-v8a", "x86_64"} {
		f, ok := found["lib/"+abi+"/"+ValidationLayerFile]
		if !ok {
			t.Errorf("the APK has no validation layer for %s", abi)
			continue
		}
		if f.Method != zip.Store {
			t.Errorf("the layer of %s is compressed", abi)
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b := make([]byte, 64)
		n, _ := rc.Read(b)
		_ = rc.Close()
		if string(b[:n]) != "LAYER "+abi {
			t.Errorf("the layer of %s holds %q", abi, b[:n])
		}
	}
	// Without the option the APK has none.
	out2 := filepath.Join(dir, "plain.apk")
	if _, err := Build(context.Background(), testSDK(), &fakeRunner{}, BuildOptions{Package: "./x", Out: out2, ConfigDir: filepath.Join(dir, "k"), Dex: []byte("d")}); err != nil {
		t.Fatal(err)
	}
	zr2, err := zip.OpenReader(out2)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr2.Close() }()
	for _, f := range zr2.File {
		if strings.Contains(f.Name, "VkLayer") {
			t.Errorf("a plain build has %s", f.Name)
		}
	}
}

func TestBuildFailsWhenTheValidationLayerIsMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := Build(context.Background(), testSDK(), &fakeRunner{}, BuildOptions{Package: "./x", Out: filepath.Join(dir, "a.apk"), ConfigDir: filepath.Join(dir, "k"), Dex: []byte("d"), ValidationLayerDir: filepath.Join(dir, "none")})
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute("Build." + TagAndroidBuildInvalid) {
		t.Errorf("route %s", appErr.Route())
	}
	if !strings.HasSuffix(appErr.Metadata["path"].(string), ValidationLayerFile) {
		t.Errorf("metadata: %v", appErr.Metadata)
	}
}
