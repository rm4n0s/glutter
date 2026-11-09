package mactool

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/rm4n0s/errors"
)

// BuildOptions are the parameters of Build (the flags of `glutter-macos build`, SPEC §29.1).
//
// Flutter: no counterpart
type BuildOptions struct {
	// Package is the main package to build, as `go build` takes it: "./examples/counter".
	Package string
	// Dir is the directory to run the go command in, which Package is relative to. The default is the current directory.
	Dir string
	// Out is the bundle to write, which ends in ".app"; "<Name>.app" in the current directory when empty.
	Out string
	// Archs are the architectures to build, "arm64" and "amd64"; with both the executable is one universal Mach-O. The default is
	// the architecture of this machine (arm64 on a host that is not a Mac).
	Archs []string
	// Release builds with -tags glutter_release and -ldflags=-s -w.
	Release bool
	// Hardened signs with the hardened runtime and the entitlements of Entitlements().
	Hardened bool
	// ID, Name and Version go into the Info.plist; the defaults are DefaultID, the name of the package, and DefaultVersion.
	ID, Name, Version string
	// Icon is a PNG for the icon; Glutter's icon when empty.
	Icon string
	// Usage are usage strings added to those of DefaultUsage (a key of the plist and its text).
	Usage map[string]string
	// Env is added to the environment of the go command.
	Env []string
	// Sign says how to sign the bundle; the zero value signs it ad hoc.
	Sign SignOptions
}

// BuildResult says what Build wrote.
//
// Flutter: no counterpart
type BuildResult struct {
	// Layout has the paths of the bundle.
	Layout Layout
	// Archs are the architectures of the executable, in the order asked for.
	Archs []string
	// Info is what went into the plist, with the defaults filled in.
	Info Info
	// Size is the size of the executable in bytes.
	Size int64
	// Thin are the sizes of the thin files, by architecture.
	Thin map[string]int64
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

// Build compiles the package for each architecture without cgo, makes the executable (universal when there are two), the icon and
// the bundle, and signs it. It fails with MacOSBuildFailed, with the output of the go command, when the package does not build, and
// with the tags of the pieces (MacOSIconInvalid, MacOSMachOInvalid, MacOSBundleInvalid, MacOSSignFailed) otherwise.
//
// Flutter: no counterpart
func Build(ctx context.Context, o BuildOptions) (BuildResult, error) {
	var res BuildResult
	if o.Package == "" {
		return res, errors.New(tagMacOSBuildFailed, "build needs a package")
	}
	archs := o.Archs
	if len(archs) == 0 {
		// This machine's architecture; a host that is not a Mac (the tool is built and vetted everywhere) builds for arm64.
		archs = []string{"arm64"}
		if runtime.GOARCH == "amd64" {
			archs = []string{"amd64"}
		}
	}
	for _, a := range archs {
		if a != "arm64" && a != "amd64" {
			return res, errors.New(tagMacOSBuildFailed, "the architectures are arm64 and amd64", "arch", a)
		}
	}
	name := o.Name
	if name == "" {
		name = DefaultName(o.Package, o.Dir)
	}
	info := Info{Executable: name, ID: o.ID, Name: name, Version: o.Version, IconFile: name, Usage: DefaultUsage(name)}
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
	icns, err := ICNS(iconPNG)
	if err != nil {
		return res, err
	}
	tmp, err := os.MkdirTemp("", "glutter-macos-build-")
	if err != nil {
		return res, errors.NewErr(tagMacOSBuildFailed, err)
	}
	defer func() { _ = os.RemoveAll(tmp) }() // best effort: a temporary directory

	res.Thin = map[string]int64{}
	var thin []FatSlice
	for _, a := range archs {
		data, err := o.compile(ctx, filepath.Join(tmp, "app-"+a), a)
		if err != nil {
			return res, err
		}
		res.Thin[a] = int64(len(data))
		thin = append(thin, FatSlice{Name: a, Data: data})
	}
	exe := thin[0].Data
	if len(thin) > 1 {
		if exe, err = Fat(thin); err != nil {
			return res, err
		}
	}
	out := o.Out
	if out == "" {
		out = name + ".app"
	}
	layout, err := Write(out, Bundle{Info: info, Executable: exe, Icon: icns})
	if err != nil {
		return res, err
	}
	if err := Sign(ctx, layout.App, SignOptions{Hardened: o.Hardened, Identity: o.Sign.Identity, Tool: o.Sign.Tool}); err != nil {
		return res, err
	}
	res.Layout, res.Archs, res.Info, res.Size = layout, slices.Clone(archs), info, int64(len(exe))
	return res, nil
}

// iconPNG reads the icon the app asked for, or makes the default.
func (o BuildOptions) iconPNG() ([]byte, error) {
	if o.Icon == "" {
		return DefaultIcon()
	}
	data, err := os.ReadFile(o.Icon)
	if err != nil {
		return nil, errors.NewErr(tagMacOSIconInvalid, err).SetMetadata("file", o.Icon)
	}
	return data, nil
}

// compile runs `go build` for one architecture and reads the file it wrote.
func (o BuildOptions) compile(ctx context.Context, out, arch string) ([]byte, error) {
	args := []string{"build", "-o", out}
	if o.Release {
		args = append(args, "-tags", "glutter_release", "-ldflags=-s -w")
	}
	args = append(args, o.Package)
	var log bytes.Buffer
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = o.Dir
	cmd.Env = append(append(os.Environ(), "GOOS=darwin", "GOARCH="+arch, "CGO_ENABLED=0"), o.Env...)
	cmd.Stdout, cmd.Stderr = &log, &log
	if err := cmd.Run(); err != nil {
		return nil, errors.NewErr(tagMacOSBuildFailed, err).SetMetadata("package", o.Package, "arch", arch, "output", log.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		return nil, errors.NewErr(tagMacOSBuildFailed, err).SetMetadata("package", o.Package, "arch", arch)
	}
	return data, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMacOSBuildFailed = TagMacOSBuildFailed

	// TagMacOSBuildFailed is the tag of the error for a package that does not build for macOS, or a build that cannot start.
	//
	// Flutter: no counterpart
	TagMacOSBuildFailed = "MacOSBuildFailed"
)
