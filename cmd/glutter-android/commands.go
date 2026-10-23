package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/androidtool"
)

// listFlag is a flag that can be given more than once.
type listFlag []string

func (l *listFlag) String() string     { return strings.Join(*l, ",") }
func (l *listFlag) Set(v string) error { *l = append(*l, v); return nil }

// buildFlags are the flags of the commands that build an APK.
type buildFlags struct {
	out, abi, appID, label, icon, key, cert string
	validation                              string
	release                                 bool
	permissions                             listFlag
}

func (b *buildFlags) add(fs *flag.FlagSet, withOut bool) {
	if withOut {
		fs.StringVar(&b.out, "o", "", "the APK to write (default: <package>.apk)")
	}
	fs.StringVar(&b.abi, "abi", "", "the ABIs to build, comma separated: arm64, amd64 (default: all; install and run build the device's)")
	fs.BoolVar(&b.release, "release", false, "build with -tags glutter_release and -ldflags=-s -w, not debuggable")
	fs.StringVar(&b.appID, "app-id", "", "the application ID (default: org.glutter.<name of the package>)")
	fs.StringVar(&b.label, "label", "", "the name under the icon (default: the name of the package)")
	fs.StringVar(&b.icon, "icon", "", "a PNG for the launcher icon (default: Glutter's)")
	fs.StringVar(&b.key, "key", "", "the PKCS#8 private key that signs a release build")
	fs.StringVar(&b.cert, "cert", "", "the X.509 certificate of the key")
	fs.StringVar(&b.validation, "vulkan-validation", "", "a directory with <abi>/libVkLayer_khronos_validation.so to pack, for the device tests (SPEC §24.13)")
	fs.Var(&b.permissions, "permission", "an Android permission the app asks for, such as android.permission.INTERNET (repeatable)")
}

func (b *buildFlags) options(pkg string) androidtool.BuildOptions {
	var abis []string
	for _, a := range strings.Split(b.abi, ",") {
		if a = strings.TrimSpace(a); a != "" {
			abis = append(abis, a)
		}
	}
	return androidtool.BuildOptions{Package: pkg, Out: b.out, ABIs: abis, Release: b.release, AppID: b.appID, Label: b.label,
		Icon: b.icon, Key: b.key, Cert: b.cert, Permissions: b.permissions, ValidationLayerDir: b.validation}
}

// parse parses the flags and returns the one package that is left.
func parse(fs *flag.FlagSet, args []string) (string, error) {
	if err := fs.Parse(args); err != nil {
		return "", errors.NewErr(tagAndroidUsage, err)
	}
	if fs.NArg() != 1 {
		return "", errors.New(tagAndroidUsage, "one package is needed, such as ./examples/counter\n"+usage)
	}
	return fs.Arg(0), nil
}

// say prints to the command's output.
func say(w io.Writer, format string, a ...any) error {
	if _, err := fmt.Fprintf(w, format, a...); err != nil {
		return errors.NewErr(tagAndroidUsage, err)
	}
	return nil
}

// runBuild is `glutter-android build`.
func runBuild(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var bf buildFlags
	bf.add(fs, true)
	pkg, err := parse(fs, args)
	if err != nil {
		return err
	}
	sdk, err := androidtool.FindSDK(os.Getenv)
	if err != nil {
		return err
	}
	res, err := androidtool.Build(ctx, sdk, androidtool.ExecRunner{}, bf.options(pkg))
	if err != nil {
		return err
	}
	return sayBuilt(stdout, pkg, res)
}

func sayBuilt(w io.Writer, pkg string, res *androidtool.BuildResult) error {
	var libs []string
	for _, a := range androidtool.ABIs {
		if n, ok := res.LibSizes[a.Name]; ok {
			libs = append(libs, fmt.Sprintf("%s %s", a.Name, size(n)))
		}
	}
	return say(w, "built %s into %s (%s, %s; libapp.so: %s)\n", pkg, res.APK, res.AppID, size(res.Size), strings.Join(libs, ", "))
}

func size(n int64) string { return fmt.Sprintf("%.1f MB", float64(n)/(1<<20)) }

// deviceFlags are the flags of the commands that talk to a device.
type deviceFlags struct{ device string }

func (d *deviceFlags) add(fs *flag.FlagSet) {
	fs.StringVar(&d.device, "device", "", "the serial of the device (default: the only one connected)")
}

// connect finds the SDK and the device, and returns adb for it.
func (d *deviceFlags) connect(ctx context.Context) (*androidtool.SDK, androidtool.ADB, error) {
	home, err := androidtool.FindHome(os.Getenv)
	if err != nil {
		return nil, androidtool.ADB{}, err
	}
	sdk := &androidtool.SDK{Home: home}
	adb := androidtool.ADB{Path: sdk.Adb(), Run: androidtool.ExecRunner{}}
	devs, err := adb.Devices(ctx)
	if err != nil {
		return nil, adb, err
	}
	dev, err := androidtool.PickDevice(devs, d.device)
	if err != nil {
		return nil, adb, err
	}
	adb.Serial = dev.Serial
	if sdk, err = androidtool.FindSDK(os.Getenv); err != nil {
		return nil, adb, err
	}
	return sdk, adb, nil
}

// buildForDevice builds the package for the device's ABI, or installs the APK that is given.
func buildForDevice(ctx context.Context, sdk *androidtool.SDK, adb androidtool.ADB, bf *buildFlags, target string, stdout io.Writer) (appID, apk string, cleanup func(), err error) {
	cleanup = func() {}
	if strings.HasSuffix(target, ".apk") {
		if bf.appID == "" {
			return "", "", cleanup, errors.New(tagAndroidUsage, "an .apk needs -app-id, to start it")
		}
		return bf.appID, target, cleanup, nil
	}
	opts := bf.options(target)
	if len(opts.ABIs) == 0 {
		abi, err := adb.ABI(ctx)
		if err != nil {
			return "", "", cleanup, err
		}
		opts.ABIs = []string{abi.Short}
	}
	if opts.Out == "" {
		dir, err := os.MkdirTemp("", "glutter-android-")
		if err != nil {
			return "", "", cleanup, errors.NewErr(tagAndroidUsage, err)
		}
		cleanup = func() { _ = os.RemoveAll(dir) } // best effort: a temporary directory
		opts.Out = filepath.Join(dir, "app.apk")
	}
	res, err := androidtool.Build(ctx, sdk, androidtool.ExecRunner{}, opts)
	if err != nil {
		cleanup()
		return "", "", func() {}, err
	}
	if err := sayBuilt(stdout, target, res); err != nil {
		cleanup()
		return "", "", func() {}, err
	}
	return res.AppID, res.APK, cleanup, nil
}

// runInstall is `glutter-android install`.
func runInstall(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var bf buildFlags
	var df deviceFlags
	bf.add(fs, false)
	df.add(fs)
	target, err := parse(fs, args)
	if err != nil {
		return err
	}
	sdk, adb, err := df.connect(ctx)
	if err != nil {
		return err
	}
	appID, apk, cleanup, err := buildForDevice(ctx, sdk, adb, &bf, target, stdout)
	if err != nil {
		return err
	}
	defer cleanup()
	if err := adb.Install(ctx, apk); err != nil {
		return err
	}
	return say(stdout, "installed %s on %s\n", appID, adb.Serial)
}

// runRun is `glutter-android run`.
func runRun(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var bf buildFlags
	var df deviceFlags
	var envs listFlag
	bf.add(fs, false)
	df.add(fs)
	fs.Var(&envs, "env", "an environment variable for the app, GLUTTER_X=v; it is given to the activity as an intent extra (repeatable)")
	target, err := parse(fs, args)
	if err != nil {
		return err
	}
	extras, err := parseEnvs(envs)
	if err != nil {
		return err
	}
	sdk, adb, err := df.connect(ctx)
	if err != nil {
		return err
	}
	appID, apk, cleanup, err := buildForDevice(ctx, sdk, adb, &bf, target, stdout)
	if err != nil {
		return err
	}
	defer cleanup()
	if err := adb.Install(ctx, apk); err != nil {
		return err
	}
	if err := adb.Start(ctx, appID, extras); err != nil {
		return err
	}
	if err := say(stdout, "started %s on %s; showing its log until it ends (Ctrl+C stops)\n", appID, adb.Serial); err != nil {
		return err
	}
	return adb.StreamLog(ctx, stdout, appID, 10*time.Second)
}
