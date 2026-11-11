package iostool

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rm4n0s/errors"
)

// BuildOptions are the parameters of Build (the flags of `glutter-ios build`, SPEC §30.1).
//
// Flutter: no counterpart
type BuildOptions struct {
	// Package is the main package to build, as `go build` takes it: "./examples/counter".
	Package string
	// Dir is the directory to run the go command in, which Package is relative to. The default is the current directory.
	Dir string
	// Out is the bundle to write, which ends in ".app"; "<Name>.app" in the current directory when empty.
	Out string
	// Simulator builds for the iOS Simulator (arm64, signed ad hoc) and not for a device.
	Simulator bool
	// Debug builds with -tags debug_mode (foundation.DebugMode) and does not strip the program; without it the build
	// is a release, stripped with -ldflags=-s -w.
	Debug bool
	// Team and Profile are the developer team (its ten characters) and the provisioning profile file of a build for a device; both or
	// neither. The owner has them; the tool only signs with them.
	Team, Profile string
	// Identity is the signing identity of a build for a device; "Apple Development" when empty.
	Identity string
	// ID, Name and Version go into the Info.plist; the defaults are DefaultID, the name of the package, and DefaultVersion.
	ID, Name, Version string
	// Icon is a PNG of at least 167 pixels, square, for the icon; Glutter's icon when empty.
	Icon string
	// Usage are usage strings added to those of DefaultUsage (a key of the plist and its text).
	Usage map[string]string
	// Orientations are the orientations the app supports (the Orientation constants); the default of InfoPlist when empty.
	Orientations []string
	// Env is added to the environment of the go command.
	Env []string
	// Runner runs the programs of Xcode; ExecRunner when nil.
	Runner Runner
}

// BuildResult says what Build wrote.
//
// Flutter: no counterpart
type BuildResult struct {
	// Layout has the paths of the bundle.
	Layout Layout
	// Info is what went into the plist, with the defaults filled in.
	Info Info
	// Size is the size of the executable in bytes.
	Size int64
}

// DefaultName is the name of an app that has none: the last element of its package path, or of the directory for ".".
//
// Flutter: no counterpart
func DefaultName(pkg, dir string) string {
	if pkg == "" || pkg == "." {
		abs, err := filepath.Abs(dir)
		if err != nil || dir == "" {
			abs, _ = filepath.Abs(".")
		}
		return filepath.Base(abs)
	}
	return filepath.Base(filepath.ToSlash(pkg))
}

// Env is the environment that makes `go build` build for iOS with Xcode's compiler: GOOS=ios, GOARCH=arm64, cgo on (Go's iOS port
// links externally, and purego needs cgo there, D35), and the compiler, the SDK and the target of the Simulator or of a device in
// the compiler and linker flags. The target says "-simulator" for the Simulator, which makes the executable one of the platform
// iOS Simulator, which a device refuses and a Simulator needs.
//
// Flutter: no counterpart
func Env(t Toolchain, simulator bool) []string {
	sdk, target := t.DeviceSDK, "arm64-apple-ios"+MinOSVersion
	if simulator {
		sdk, target = t.SimulatorSDK, "arm64-apple-ios"+MinOSVersion+"-simulator"
	}
	flags := "-isysroot " + sdk + " -target " + target
	return []string{"GOOS=ios", "GOARCH=arm64", "CGO_ENABLED=1", "CC=" + t.Clang + " " + flags, "CGO_CFLAGS=" + flags, "CGO_LDFLAGS=" + flags}
}

// Build compiles the package for iOS, makes the icon files and the bundle, and signs it (ad hoc for the Simulator, with the team,
// the profile and the identity for a device). It fails with IOSXcodeMissing when Xcode is not there, IOSBuildFailed, with the output
// of the go command, when the package does not build, and with the tags of the pieces (IOSIconInvalid, IOSBundleInvalid,
// IOSSignFailed) otherwise.
//
// Flutter: no counterpart
func Build(ctx context.Context, o BuildOptions) (BuildResult, error) {
	var res BuildResult
	if o.Package == "" {
		return res, errors.New(tagIOSBuildFailed, "build needs a package")
	}
	if (o.Team == "") != (o.Profile == "") {
		return res, errors.New(tagIOSBuildFailed, "a build for a device needs the team and the provisioning profile together", "team", o.Team, "profile", o.Profile)
	}
	if o.Simulator && o.Team != "" {
		return res, errors.New(tagIOSBuildFailed, "a build for the Simulator is signed ad hoc and has no team", "team", o.Team)
	}
	name := o.Name
	if name == "" {
		name = DefaultName(o.Package, o.Dir)
	}
	info := Info{Executable: name, ID: o.ID, Name: name, Version: o.Version, Simulator: o.Simulator, Orientations: o.Orientations, Usage: DefaultUsage(name)}
	if info.ID == "" {
		info.ID = DefaultID(name)
	}
	if info.Version == "" {
		info.Version = DefaultVersion
	}
	for k, v := range o.Usage {
		info.Usage[k] = v
	}
	if _, err := InfoPlist(info); err != nil { // the identity is checked before the compiler runs
		return res, err
	}
	iconPNG, err := o.iconPNG()
	if err != nil {
		return res, err
	}
	icons, err := Icons(iconPNG)
	if err != nil {
		return res, err
	}
	var profile []byte
	if o.Profile != "" {
		if profile, err = os.ReadFile(o.Profile); err != nil {
			return res, errors.NewErr(tagIOSBuildFailed, err).SetMetadata("profile", o.Profile)
		}
	}
	tc, err := CheckXcode(ctx, o.Runner)
	if err != nil {
		return res, err
	}
	tmp, err := os.MkdirTemp("", "glutter-ios-build-")
	if err != nil {
		return res, errors.NewErr(tagIOSBuildFailed, err)
	}
	defer func() { _ = os.RemoveAll(tmp) }() // best effort: a temporary directory
	exe, err := o.compile(ctx, tc, filepath.Join(tmp, "app"))
	if err != nil {
		return res, err
	}
	out := o.Out
	if out == "" {
		out = name + ".app"
	}
	layout, err := Write(out, Bundle{Info: info, Executable: exe, Icons: icons, Profile: profile})
	if err != nil {
		return res, err
	}
	if err := Sign(ctx, layout.App, SignOptions{Team: o.Team, ID: info.ID, Identity: o.Identity, Runner: o.Runner}); err != nil {
		return res, err
	}
	info.Icons = true
	res.Layout, res.Info, res.Size = layout, info, int64(len(exe))
	return res, nil
}

// iconPNG reads the icon the app asked for, or makes the default.
func (o BuildOptions) iconPNG() ([]byte, error) {
	if o.Icon == "" {
		return DefaultIcon()
	}
	data, err := os.ReadFile(o.Icon)
	if err != nil {
		return nil, errors.NewErr(tagIOSIconInvalid, err).SetMetadata("file", o.Icon)
	}
	return data, nil
}

// compile runs `go build` and reads the file it wrote.
func (o BuildOptions) compile(ctx context.Context, tc Toolchain, out string) ([]byte, error) {
	args := []string{"build", "-o", out}
	if o.Debug {
		args = append(args, "-tags", "debug_mode")
	} else {
		args = append(args, "-ldflags=-s -w")
	}
	args = append(args, o.Package)
	var log bytes.Buffer
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = o.Dir
	cmd.Env = append(append(os.Environ(), Env(tc, o.Simulator)...), o.Env...)
	cmd.Stdout, cmd.Stderr = &log, &log
	if err := cmd.Run(); err != nil {
		return nil, errors.NewErr(tagIOSBuildFailed, err).SetMetadata("package", o.Package, "simulator", o.Simulator, "output", log.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		return nil, errors.NewErr(tagIOSBuildFailed, err).SetMetadata("package", o.Package)
	}
	return data, nil
}

// SignOptions are what Sign is asked for.
//
// Flutter: no counterpart
type SignOptions struct {
	// Team is the developer team of a build for a device; empty signs ad hoc (the Simulator).
	Team string
	// ID is the bundle identifier, which goes into the entitlements of a build for a device.
	ID string
	// Identity is the signing identity of a build for a device; "Apple Development" when empty.
	Identity string
	// Runner runs codesign; ExecRunner when nil.
	Runner Runner
}

// Sign signs the bundle with codesign: ad hoc for the Simulator, and for a device with the identity and the entitlements of the team
// and the bundle identifier. It fails with IOSSignFailed, with codesign's output, when codesign cannot be run or says no.
//
// Flutter: no counterpart
func Sign(ctx context.Context, app string, o SignOptions) error {
	r := o.Runner
	if r == nil {
		r = ExecRunner{}
	}
	args := []string{"--force", "--timestamp=none"}
	if o.Team == "" {
		args = append(args, "--sign", "-")
	} else {
		identity := o.Identity
		if identity == "" {
			identity = "Apple Development"
		}
		dir, err := os.MkdirTemp("", "glutter-ios-sign-")
		if err != nil {
			return errors.NewErr(tagIOSSignFailed, err).SetMetadata("app", app)
		}
		defer func() { _ = os.RemoveAll(dir) }() // best effort: a temporary directory
		ent := filepath.Join(dir, "entitlements.plist")
		if err := os.WriteFile(ent, Entitlements(o.Team, o.ID), 0o600); err != nil {
			return errors.NewErr(tagIOSSignFailed, err).SetMetadata("app", app)
		}
		args = append(args, "--sign", identity, "--entitlements", ent)
	}
	args = append(args, app)
	out, exit, err := r.Run(ctx, "/usr/bin/codesign", args...)
	if err != nil {
		return err
	}
	if exit != 0 {
		return errors.New(tagIOSSignFailed, "codesign says no", "app", app, "exit", exit, "output", string(out))
	}
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagIOSBuildFailed = TagIOSBuildFailed
	tagIOSSignFailed  = TagIOSSignFailed

	// TagIOSBuildFailed is the tag of the error for a package that does not build for iOS, or a build that is asked for with
	// parameters that cannot go together.
	//
	// Flutter: no counterpart
	TagIOSBuildFailed = "IOSBuildFailed"
	// TagIOSSignFailed is the tag of the error for a bundle that codesign cannot sign.
	//
	// Flutter: no counterpart
	TagIOSSignFailed = "IOSSignFailed"
)
