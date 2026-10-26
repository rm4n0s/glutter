package wintool

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rm4n0s/errors"
)

// BuildOptions are the parameters of Build (the flags of `glutter-windows build`, SPEC §25.1).
//
// Flutter: no counterpart
type BuildOptions struct {
	// Package is the main package to build, as `go build` takes it: "./examples/counter".
	Package string
	// Dir is the directory to run the go command in, which Package is relative to. The default is the
	// current directory.
	Dir string
	// Out is the .exe to write; "<base name of Package>.exe" when empty. With more than one
	// architecture the architecture is added before the extension: "app_amd64.exe".
	Out string
	// Archs are the architectures to build, by name ("amd64", "arm64"); amd64 when empty.
	Archs []string
	// Release builds with -tags glutter_release and -ldflags=-s -w.
	Release bool
	// Console builds a console program (no -H=windowsgui): it opens a console window when started
	// from Explorer, and prints to the terminal it is started from.
	Console bool
	// AppID, Name and Version go into the manifest and the version information; they have defaults
	// (DefaultAppID, DefaultName, DefaultVersion).
	AppID, Name, Version string
	// Icon is a PNG for the icon; Glutter's icon when empty.
	Icon string
	// Env is added to the environment of the go command.
	Env []string
}

// Built is one .exe that Build wrote.
//
// Flutter: no counterpart
type Built struct {
	// Arch is the architecture's name.
	Arch string
	// Path is the .exe, Size its size in bytes.
	Path string
	Size int64
}

// BuildResult says what Build wrote.
//
// Flutter: no counterpart
type BuildResult struct {
	// Exes are the files written, one per architecture, in the order asked for.
	Exes []Built
	// AppID is the application ID of the manifest.
	AppID string
}

// SysoFile is the name, in the package's directory, of the file that holds the resources of an
// architecture: "rsrc_windows_amd64.syso". The go command links every .syso whose name's GOOS and
// GOARCH match the build.
//
// Flutter: no counterpart
func SysoFile(arch string) string { return "rsrc_windows_" + arch + ".syso" }

// CopyPrefix starts the name of the temporary directory that Build makes next to the package's
// directory and removes at the end (SPEC §25.1). The underscore makes the go command's "./..." skip
// it.
//
// Flutter: no counterpart
const CopyPrefix = "_glutterwin-"

// Build makes the .exe files of a package (SPEC §25.1): for each architecture it writes the
// resources (the manifest, the icon, the version information) as a .syso, puts it into a temporary
// copy of the package's directory (next to the original, named CopyPrefix plus a number; the go
// command does not take a .syso from an overlay), and runs `go build` on the copy with GOOS=windows
// and CGO_ENABLED=0. The copy is removed afterwards. The programs run through run. It fails with
// WindowsBuildInvalid, WindowsManifestInvalid, WindowsResourceInvalid, WindowsIconInvalid and
// WindowsCommandFailed.
//
// Flutter: no counterpart (flutter build windows)
func Build(ctx context.Context, run Runner, o BuildOptions) (*BuildResult, error) {
	if o.Package == "" {
		return nil, errors.New(tagWindowsBuildInvalid, "there is no package to build")
	}
	var archs []Arch
	for _, name := range o.Archs {
		a, ok := FindArch(name)
		if !ok {
			return nil, errors.New(tagWindowsBuildInvalid, "the architecture is not amd64 or arm64", "arch", name)
		}
		if !hasArch(archs, a.Name) {
			archs = append(archs, a)
		}
	}
	if len(archs) == 0 {
		archs = []Arch{Archs[0]}
	}
	version, err := ParseVersion(o.Version)
	if err != nil {
		return nil, err
	}
	if o.AppID == "" {
		o.AppID = DefaultAppID(o.Package)
	}
	if o.Name == "" {
		o.Name = DefaultName(o.Package)
	}
	var iconPNG []byte
	if o.Icon != "" {
		if iconPNG, err = os.ReadFile(o.Icon); err != nil {
			return nil, errors.NewErr(tagWindowsIconInvalid, err).SetMetadata("path", o.Icon)
		}
	} else if iconPNG, err = DefaultIcon(); err != nil {
		return nil, err
	}
	icon, err := NewIcon(iconPNG)
	if err != nil {
		return nil, err
	}
	pkg, err := listPackage(ctx, run, o, archs[0])
	if err != nil {
		return nil, err
	}

	out := o.Out
	if out == "" {
		out = baseName(filepath.ToSlash(o.Package)) + ".exe"
		if o.Package == "." || strings.HasSuffix(o.Package, "/") {
			out = "app.exe"
		}
	}
	res := &BuildResult{AppID: o.AppID}
	for _, a := range archs {
		exe := exeName(out, a.Name, len(archs) > 1)
		manifest, err := Manifest(ManifestConfig{AppID: o.AppID, Name: o.Name, Version: version, Arch: a.Name})
		if err != nil {
			return nil, err
		}
		syso, err := BuildSyso(a.Name, Resources(manifest, icon, VersionInfo{Version: version, Name: o.Name, AppID: o.AppID, FileName: filepath.Base(exe)}))
		if err != nil {
			return nil, err
		}
		copyDir, err := copyPackage(pkg, syso, a.Name)
		if err != nil {
			return nil, err
		}
		args := []string{"build", "-o", exe}
		if o.Release {
			args = append(args, "-tags", "glutter_release")
		}
		if ld := ldflags(o); ld != "" {
			args = append(args, "-ldflags="+ld)
		}
		args = append(args, copyDir)
		env := append(os.Environ(), "GOOS=windows", "GOARCH="+a.GOARCH, "CGO_ENABLED=0")
		env = append(env, o.Env...)
		_, err = run.Run(ctx, Command{Dir: o.Dir, Env: env, Name: "go", Args: args})
		_ = os.RemoveAll(copyDir) // best effort: a temporary directory next to the package
		if err != nil {
			return nil, err
		}
		path := exe
		if !filepath.IsAbs(path) {
			path = filepath.Join(o.Dir, path)
		}
		st, err := os.Stat(path)
		if err != nil {
			return nil, errors.NewErr(tagWindowsBuildInvalid, err).SetMetadata("path", path)
		}
		res.Exes = append(res.Exes, Built{Arch: a.Name, Path: path, Size: st.Size()})
	}
	return res, nil
}

// listedPackage is the part of `go list -json` that Build reads.
type listedPackage struct {
	Dir        string
	Name       string
	SysoFiles  []string
	EmbedFiles []string
}

// listPackage runs `go list -json` for the package, for the first architecture and with the tags of
// the build, and checks that it is one main package without resources of its own. It fails with
// WindowsBuildInvalid.
func listPackage(ctx context.Context, run Runner, o BuildOptions, arch Arch) (*listedPackage, error) {
	env := append(os.Environ(), "GOOS=windows", "GOARCH="+arch.GOARCH, "CGO_ENABLED=0")
	env = append(env, o.Env...)
	args := []string{"list", "-json"}
	if o.Release {
		args = append(args, "-tags", "glutter_release")
	}
	listed, err := run.Run(ctx, Command{Dir: o.Dir, Env: env, Name: "go", Args: append(args, o.Package)})
	if err != nil {
		return nil, err
	}
	var pkg listedPackage
	dec := json.NewDecoder(strings.NewReader(listed))
	if err := dec.Decode(&pkg); err != nil {
		return nil, errors.NewErr(tagWindowsBuildInvalid, err).SetMetadata("package", o.Package, "listed", listed)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New(tagWindowsBuildInvalid, "the package must be one main package, not a pattern", "package", o.Package)
	}
	if pkg.Name != "main" || pkg.Dir == "" {
		return nil, errors.New(tagWindowsBuildInvalid, "the package is not a main package", "package", o.Package, "name", pkg.Name)
	}
	if len(pkg.SysoFiles) > 0 {
		return nil, errors.New(tagWindowsBuildInvalid, "the package has resources of its own (.syso files): build it with go build", "package", o.Package, "files", strings.Join(pkg.SysoFiles, ", "))
	}
	return &pkg, nil
}

// copyPackage makes the temporary directory next to the package's: it gets the files of the
// package's directory (not the tests and not the subdirectories) and the files that go:embed names,
// and the .syso for the architecture. It returns the directory, which the caller removes; the
// directory is removed here when the copy fails. It fails with WindowsBuildInvalid.
func copyPackage(pkg *listedPackage, syso []byte, arch string) (string, error) {
	dir, err := os.MkdirTemp(filepath.Dir(pkg.Dir), CopyPrefix+"*")
	if err != nil {
		return "", errors.NewErr(tagWindowsBuildInvalid, err).SetMetadata("what", "the temporary copy of the package", "parent", filepath.Dir(pkg.Dir))
	}
	files := slices.Clone(pkg.EmbedFiles)
	entries, err := os.ReadDir(pkg.Dir)
	if err != nil {
		_ = os.RemoveAll(dir) // best effort: the copy is not usable
		return "", errors.NewErr(tagWindowsBuildInvalid, err).SetMetadata("dir", pkg.Dir)
	}
	for _, e := range entries {
		if e.Type().IsRegular() && !strings.HasSuffix(e.Name(), "_test.go") {
			files = append(files, e.Name())
		}
	}
	for _, f := range slices.Compact(slices.Sorted(slices.Values(files))) {
		data, err := os.ReadFile(filepath.Join(pkg.Dir, f))
		if err != nil {
			_ = os.RemoveAll(dir) // best effort: the copy is not usable
			return "", errors.NewErr(tagWindowsBuildInvalid, err).SetMetadata("path", filepath.Join(pkg.Dir, f))
		}
		dst := filepath.Join(dir, f)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			_ = os.RemoveAll(dir) // best effort: the copy is not usable
			return "", errors.NewErr(tagWindowsBuildInvalid, err).SetMetadata("path", dst)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			_ = os.RemoveAll(dir) // best effort: the copy is not usable
			return "", errors.NewErr(tagWindowsBuildInvalid, err).SetMetadata("path", dst)
		}
	}
	sysoPath := filepath.Join(dir, SysoFile(arch))
	if err := os.WriteFile(sysoPath, syso, 0o644); err != nil {
		_ = os.RemoveAll(dir) // best effort: the copy is not usable
		return "", errors.NewErr(tagWindowsBuildInvalid, err).SetMetadata("path", sysoPath)
	}
	return dir, nil
}

// hasArch reports whether the list has the architecture.
func hasArch(list []Arch, name string) bool {
	for _, a := range list {
		if a.Name == name {
			return true
		}
	}
	return false
}

// ldflags are the linker flags: -H=windowsgui for a GUI program, -s -w for a release.
func ldflags(o BuildOptions) string {
	var flags []string
	if !o.Console {
		flags = append(flags, "-H=windowsgui")
	}
	if o.Release {
		flags = append(flags, "-s", "-w")
	}
	return strings.Join(flags, " ")
}

// exeName is the file for an architecture: out itself for one architecture, with "_amd64" before
// the extension for several.
func exeName(out, arch string, several bool) string {
	if !several {
		return out
	}
	ext := filepath.Ext(out)
	return strings.TrimSuffix(out, ext) + "_" + arch + ext
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWindowsBuildInvalid = TagWindowsBuildInvalid

	// TagWindowsBuildInvalid is the tag of the error for a build that cannot be done: no package, an
	// architecture that is not known, a package that is not one main package or that has resources
	// of its own, or a file of the build that cannot be written.
	TagWindowsBuildInvalid = "WindowsBuildInvalid"
)
