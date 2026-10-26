package androidtool

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
)

// BuildOptions are the parameters of Build (the flags of `glutter-android build`, SPEC §24.1).
//
// Flutter: no counterpart
type BuildOptions struct {
	// Package is the main package to build, as `go build` takes it: "./examples/counter".
	Package string
	// Dir is the directory to run the go command in, which Package is relative to. The
	// default is the current directory.
	Dir string
	// Out is the APK to write; "<base name of Package>.apk" when empty.
	Out string
	// ABIs are the ABIs to build, by their short names ("arm64", "amd64"); all when empty.
	ABIs []string
	// Release builds with -tags glutter_release and -ldflags=-s -w, without the debuggable
	// flag, and signs with Key and Cert.
	Release bool
	// AppID, Label and Permissions go into the manifest; AppID and Label have defaults.
	AppID, Label string
	Permissions  []string
	// Icon is a PNG for the launcher; Glutter's icon when empty.
	Icon string
	// Key and Cert (PKCS#8 and X.509, as apksigner takes them) sign a release build.
	Key, Cert string
	// ConfigDir holds the debug key; $XDG_CONFIG_HOME/glutter/android-debug when empty.
	ConfigDir string
	// Env is added to the environment of the go command.
	Env []string
	// Dex is the embedding's classes.dex; the checked-in Dex when nil.
	Dex []byte
	// ValidationLayerDir, when set, is a directory with the Khronos validation layer of Android,
	// <dir>/<abi>/libVkLayer_khronos_validation.so for each ABI built (the android-binaries zip of a
	// Vulkan-ValidationLayers release): it is packed into the APK next to libapp.so, where the
	// Vulkan loader finds the layer of a debuggable app (SPEC §24.13).
	ValidationLayerDir string
}

// ValidationLayerFile is the name of the Khronos validation layer's library.
//
// Flutter: no counterpart
const ValidationLayerFile = "libVkLayer_khronos_validation.so"

// BuildResult says what Build wrote.
//
// Flutter: no counterpart
type BuildResult struct {
	// APK is the signed APK, AppID the application ID.
	APK, AppID string
	// Size is the size of the APK in bytes, LibSizes the size of each ABI's libapp.so.
	Size     int64
	LibSizes map[string]int64
}

// Build makes the APK of a package (SPEC §24.1): it builds libapp.so for each ABI with the
// NDK's clang and cgo, links the manifest and the icon with aapt2, adds classes.dex and the
// libraries (stored, not compressed), aligns the result to 16 KB with zipalign and signs it
// with apksigner. The programs run through run.
//
// Flutter: no counterpart (flutter build apk)
func Build(ctx context.Context, sdk *SDK, run Runner, o BuildOptions) (*BuildResult, error) {
	if o.Package == "" {
		return nil, errors.New(TagAndroidBuildInvalid, "there is no package to build")
	}
	abis, err := selectABIs(o.ABIs)
	if err != nil {
		return nil, err
	}
	if o.Release && (o.Key == "" || o.Cert == "") {
		return nil, errors.New(TagAndroidBuildInvalid, "a release build needs -key and -cert")
	}
	if o.AppID == "" {
		o.AppID = DefaultAppID(o.Package)
	}
	if !ApplicationIDValid(o.AppID) {
		return nil, errors.New(TagAndroidBuildInvalid, "the application ID is not valid: use segments of letters, digits and underscores, such as org.example.app", "appID", o.AppID)
	}
	if o.Label == "" {
		o.Label = DefaultLabel(o.Package)
	}
	if o.Out == "" {
		o.Out = filepath.Base(filepath.FromSlash(o.Package)) + ".apk"
	}
	dex := o.Dex
	if dex == nil {
		dex = Dex
	}
	key, cert := o.Key, o.Cert
	if !o.Release {
		dir := o.ConfigDir
		if dir == "" {
			cfg, err := os.UserConfigDir()
			if err != nil {
				return nil, errors.NewErr(TagAndroidKeyFailed, err).SetMetadata("what", "the config directory of the debug key")
			}
			dir = filepath.Join(cfg, "glutter", "android-debug")
		}
		dk, err := EnsureDebugKey(dir)
		if err != nil {
			return nil, err
		}
		key, cert = dk.Key, dk.Cert
	}
	icon := []byte(nil)
	if o.Icon != "" {
		if icon, err = os.ReadFile(o.Icon); err != nil {
			return nil, errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("path", o.Icon)
		}
	} else if icon, err = DefaultIcon(); err != nil {
		return nil, err
	}

	// Directories that a killed build left behind are removed first (tempdirs.go); this build removes its own
	// on every path that returns.
	sweepStaleTempDirs(os.TempDir(), time.Now(), staleTempAge, apkWorkPrefix, dexWorkPrefix)
	work, err := os.MkdirTemp("", apkWorkPrefix)
	if err != nil {
		return nil, errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("what", "the temporary directory of the build")
	}
	defer func() { _ = os.RemoveAll(work) }() // best effort: a temporary directory

	// 1. libapp.so for each ABI.
	libs, layers := map[string]string{}, map[string]string{}
	res := &BuildResult{AppID: o.AppID, LibSizes: map[string]int64{}}
	for _, abi := range abis {
		so := filepath.Join(work, "lib", abi.Name, "libapp.so")
		if err := os.MkdirAll(filepath.Dir(so), 0o755); err != nil {
			return nil, errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("dir", filepath.Dir(so))
		}
		args := []string{"build", "-buildmode=c-shared", "-o", so}
		if o.Release {
			args = append(args, "-tags", "glutter_release", "-ldflags=-s -w")
		}
		args = append(args, o.Package)
		env := append(os.Environ(), "GOOS=android", "GOARCH="+abi.GOARCH, "CGO_ENABLED=1", "CC="+sdk.Clang(abi.Name))
		env = append(env, o.Env...)
		if _, err := run.Run(ctx, Command{Dir: o.Dir, Env: env, Name: "go", Args: args}); err != nil {
			return nil, err
		}
		st, err := os.Stat(so)
		if err != nil {
			return nil, errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("path", so)
		}
		libs[abi.Name] = so
		res.LibSizes[abi.Name] = st.Size()
	}

	// 2. The manifest and the icon, linked with aapt2.
	manifest := filepath.Join(work, "AndroidManifest.xml")
	if err := os.WriteFile(manifest, Manifest(ManifestConfig{AppID: o.AppID, Label: o.Label, Permissions: o.Permissions, Debuggable: !o.Release, HasIcon: true}), 0o644); err != nil {
		return nil, errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("path", manifest)
	}
	resDir := filepath.Join(work, "res", "mipmap-xxxhdpi")
	if err := os.MkdirAll(resDir, 0o755); err != nil {
		return nil, errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("dir", resDir)
	}
	if err := os.WriteFile(filepath.Join(resDir, "ic_launcher.png"), icon, 0o644); err != nil {
		return nil, errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("dir", resDir)
	}
	resZip, base := filepath.Join(work, "res.zip"), filepath.Join(work, "base.apk")
	aapt2 := sdk.Tool("aapt2")
	if _, err := run.Run(ctx, Command{Name: aapt2, Args: []string{"compile", "--dir", filepath.Join(work, "res"), "-o", resZip}}); err != nil {
		return nil, err
	}
	link := []string{"link", "-o", base, "-I", sdk.AndroidJar(), "--manifest", manifest, "-R", resZip,
		"--min-sdk-version", strconv.Itoa(MinSDK), "--target-sdk-version", strconv.Itoa(TargetSDK), "--auto-add-overlay"}
	if _, err := run.Run(ctx, Command{Name: aapt2, Args: link}); err != nil {
		return nil, err
	}

	// 3. The APK: aapt2's output, classes.dex and the libraries.
	if o.ValidationLayerDir != "" {
		for _, abi := range abis {
			layer := filepath.Join(o.ValidationLayerDir, abi.Name, ValidationLayerFile)
			if _, err := os.Stat(layer); err != nil {
				return nil, errors.NewErr(TagAndroidBuildInvalid, err).SetMetadata("path", layer, "abi", abi.Name,
					"hint", "unzip the android-binaries zip of a Vulkan-ValidationLayers release so that this file exists")
			}
			layers[abi.Name] = layer
		}
	}
	packed := filepath.Join(work, "packed.apk")
	if err := packAPK(base, packed, dex, abis, libs, layers); err != nil {
		return nil, err
	}
	// The libraries are in the packed APK now: free their disk (this is memory under /tmp) while the
	// APK is aligned and signed.
	_ = os.RemoveAll(filepath.Join(work, "lib")) // best effort: a temporary directory

	// 4. Align to 16 KB, then sign.
	aligned := filepath.Join(work, "aligned.apk")
	if _, err := run.Run(ctx, Command{Name: sdk.Tool("zipalign"), Args: []string{"-f", "-P", "16", "4", packed, aligned}}); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(o.Out), 0o755); err != nil {
		return nil, errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("dir", filepath.Dir(o.Out))
	}
	sign := []string{"sign", "--key", key, "--cert", cert, "--min-sdk-version", strconv.Itoa(MinSDK), "--out", o.Out, aligned}
	if _, err := run.Run(ctx, Command{Name: sdk.Tool("apksigner"), Args: sign}); err != nil {
		return nil, err
	}
	st, err := os.Stat(o.Out)
	if err != nil {
		return nil, errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("path", o.Out)
	}
	res.APK, res.Size = o.Out, st.Size()
	return res, nil
}

// selectABIs turns the short names of -abi into ABIs, all of them when none is given.
func selectABIs(names []string) ([]ABI, error) {
	if len(names) == 0 {
		return ABIs, nil
	}
	var out []ABI
	for _, n := range names {
		found := false
		for _, a := range ABIs {
			if a.Short == n || a.Name == n {
				out = append(out, a)
				found = true
				break
			}
		}
		if !found {
			return nil, errors.New(TagAndroidBuildInvalid, "unknown ABI "+strconv.Quote(n)+": use arm64 or amd64", "abi", n)
		}
	}
	return out, nil
}

// packAPK writes the APK: every entry of aapt2's output as it is, classes.dex, and each
// library stored without compression (the system maps it from the APK, SPEC §24.1).
func packAPK(base, out string, dex []byte, abis []ABI, libs, layers map[string]string) (err error) {
	r, err := zip.OpenReader(base)
	if err != nil {
		return errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("path", base)
	}
	defer func() { _ = r.Close() }() // read only
	f, err := os.Create(out)
	if err != nil {
		return errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("path", out)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = errors.NewErr(TagAndroidWriteFailed, cerr).SetMetadata("path", out)
		}
	}()
	w := zip.NewWriter(f)
	for _, e := range r.File {
		if err := w.Copy(e); err != nil {
			return errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("path", out, "entry", e.Name)
		}
	}
	dw, err := w.Create("classes.dex")
	if err != nil {
		return errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("path", out)
	}
	if _, err := dw.Write(dex); err != nil {
		return errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("path", out)
	}
	for _, abi := range abis {
		entries := []struct{ name, path string }{{"libapp.so", libs[abi.Name]}}
		if layers[abi.Name] != "" {
			entries = append(entries, struct{ name, path string }{ValidationLayerFile, layers[abi.Name]})
		}
		for _, e := range entries {
			src, err := os.Open(e.path)
			if err != nil {
				return errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("path", e.path)
			}
			lw, err := w.CreateHeader(&zip.FileHeader{Name: "lib/" + abi.Name + "/" + e.name, Method: zip.Store})
			if err != nil {
				_ = src.Close() // read only
				return errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("path", out)
			}
			_, err = io.Copy(lw, src)
			_ = src.Close() // read only
			if err != nil {
				return errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("path", e.path)
			}
		}
	}
	if err := w.Close(); err != nil {
		return errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("path", out)
	}
	return nil
}

// ApplicationIDValid says whether s is an application ID: at least two segments of letters,
// digits and underscores, each starting with a letter.
//
// Flutter: no counterpart
func ApplicationIDValid(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts {
		if p == "" || !isLetter(rune(p[0])) {
			return false
		}
		for _, r := range p {
			if !isLetter(r) && (r < '0' || r > '9') && r != '_' {
				return false
			}
		}
	}
	return true
}

func isLetter(r rune) bool { return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagAndroidBuildInvalid is the tag of the error for options that cannot make an APK: no
	// package, an unknown ABI, a release build without a key.
	TagAndroidBuildInvalid = "AndroidBuildInvalid"
	// TagAndroidWriteFailed is the tag of the error for a file of the build that cannot be
	// read or written; the metadata "path" or "dir" says which.
	TagAndroidWriteFailed = "AndroidWriteFailed"
)
