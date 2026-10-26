package wintool

import (
	"bytes"
	"context"
	"debug/pe"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

// fakeRunner records the commands of a build and answers them like the go command would: `go list`
// with the JSON of a package in dir, and `go build` with an .exe at the -o path. onBuild looks at the
// copy of the package while it exists.
type fakeRunner struct {
	dir     string // the package directory that go list reports
	listed  string // overrides the answer of go list
	cmds    []Command
	onBuild func(t *testing.T, copyDir string, c Command)
	t       *testing.T
	failOn  string // a command name+first argument that fails, such as "go build"
}

func (f *fakeRunner) Run(_ context.Context, c Command) (string, error) {
	f.cmds = append(f.cmds, c)
	if f.failOn != "" && f.failOn == c.Name+" "+c.Args[0] {
		return "boom", errors.New(TagWindowsCommandFailed, "the fake command failed", "command", c.String())
	}
	switch c.Args[0] {
	case "list":
		if f.listed != "" {
			return f.listed, nil
		}
		out, err := json.Marshal(map[string]any{"Dir": f.dir, "Name": "main", "EmbedFiles": []string{"assets/logo.txt"}})
		if err != nil {
			f.t.Fatal(err)
		}
		return string(out), nil
	case "build":
		copyDir := c.Args[len(c.Args)-1]
		if f.onBuild != nil {
			f.onBuild(f.t, copyDir, c)
		}
		out := c.Args[slices.Index(c.Args, "-o")+1]
		if !filepath.IsAbs(out) {
			out = filepath.Join(c.Dir, out)
		}
		if err := os.WriteFile(out, []byte("MZ exe"), 0o755); err != nil {
			f.t.Fatal(err)
		}
	}
	return "", nil
}

// newPackageDir makes a package directory with a main.go, a test, an embedded file in a subdirectory
// and a subdirectory that is not embedded.
func newPackageDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "mod", "app")
	for name, data := range map[string]string{
		"main.go":            "package main\n",
		"main_test.go":       "package main\n",
		"assets/logo.txt":    "logo",
		"other/ignored.txt":  "not embedded",
		"notes/also_not.txt": "not embedded",
	} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestBuildCommandLines(t *testing.T) {
	tests := []struct {
		name      string
		opts      BuildOptions
		wantBuild [][]string // the arguments of each go build, without the directory
		wantExes  []string
	}{
		{"a GUI build for amd64", BuildOptions{Package: "./app"},
			[][]string{{"build", "-o", "app.exe", "-ldflags=-H=windowsgui"}}, []string{"app.exe"}},
		{"a console build", BuildOptions{Package: "./app", Console: true},
			[][]string{{"build", "-o", "app.exe"}}, []string{"app.exe"}},
		{"a release GUI build", BuildOptions{Package: "./app", Release: true},
			[][]string{{"build", "-o", "app.exe", "-tags", "glutter_release", "-ldflags=-H=windowsgui -s -w"}}, []string{"app.exe"}},
		{"a release console build", BuildOptions{Package: "./app", Release: true, Console: true},
			[][]string{{"build", "-o", "app.exe", "-tags", "glutter_release", "-ldflags=-s -w"}}, []string{"app.exe"}},
		{"a name for the output", BuildOptions{Package: "./app", Out: "dist/My App.exe"},
			[][]string{{"build", "-o", "dist/My App.exe", "-ldflags=-H=windowsgui"}}, []string{"dist/My App.exe"}},
		{"two architectures", BuildOptions{Package: "./app", Archs: []string{"amd64", "arm64", "amd64"}},
			[][]string{{"build", "-o", "app_amd64.exe", "-ldflags=-H=windowsgui"}, {"build", "-o", "app_arm64.exe", "-ldflags=-H=windowsgui"}}, []string{"app_amd64.exe", "app_arm64.exe"}},
		{"arm64 alone keeps the name", BuildOptions{Package: "./app", Archs: []string{"arm64"}, Out: "x.exe"},
			[][]string{{"build", "-o", "x.exe", "-ldflags=-H=windowsgui"}}, []string{"x.exe"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgDir := newPackageDir(t)
			work := t.TempDir()
			f := &fakeRunner{dir: pkgDir, t: t}
			tt.opts.Dir = work
			if tt.opts.Out == "dist/My App.exe" {
				if err := os.MkdirAll(filepath.Join(work, "dist"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			res, err := Build(context.Background(), f, tt.opts)
			if err != nil {
				t.Fatal(err)
			}
			var builds [][]string
			for _, c := range f.cmds {
				if c.Name != "go" || c.Dir != work {
					t.Errorf("command %q in %q, want go in %q", c.String(), c.Dir, work)
				}
				if c.Args[0] == "build" {
					builds = append(builds, c.Args[:len(c.Args)-1]) // the last is the copy
				}
			}
			if len(builds) != len(tt.wantBuild) {
				t.Fatalf("%d builds, want %d: %v", len(builds), len(tt.wantBuild), f.cmds)
			}
			for i := range builds {
				if !slices.Equal(builds[i], tt.wantBuild[i]) {
					t.Errorf("build %d = %q, want %q", i, builds[i], tt.wantBuild[i])
				}
			}
			var exes []string
			for _, e := range res.Exes {
				exes = append(exes, strings.TrimPrefix(e.Path, work+string(filepath.Separator)))
				if e.Size != int64(len("MZ exe")) {
					t.Errorf("%s: size %d", e.Path, e.Size)
				}
			}
			if !slices.Equal(exes, tt.wantExes) {
				t.Errorf("exes = %v, want %v", exes, tt.wantExes)
			}
			// go list runs once, with the architecture of the first build and the release tag.
			list := f.cmds[0]
			if list.Args[0] != "list" || !slices.Contains(list.Env, "GOOS=windows") || !slices.Contains(list.Env, "CGO_ENABLED=0") {
				t.Errorf("the first command is %q with GOOS/CGO_ENABLED %v", list.String(), list.Env)
			}
			for _, c := range f.cmds {
				if !slices.Contains(c.Env, "GOOS=windows") || !slices.Contains(c.Env, "CGO_ENABLED=0") {
					t.Errorf("%q has no GOOS=windows and CGO_ENABLED=0", c.String())
				}
			}
		})
	}
}

func TestBuildPutsTheResourcesInATemporaryCopy(t *testing.T) {
	pkgDir := newPackageDir(t)
	iconPath := filepath.Join(t.TempDir(), "icon.png")
	if err := os.WriteFile(iconPath, squarePNG(t, 300, 300), 0o644); err != nil {
		t.Fatal(err)
	}
	var copyDirs []string
	f := &fakeRunner{dir: pkgDir, t: t}
	f.onBuild = func(t *testing.T, copyDir string, c Command) {
		copyDirs = append(copyDirs, copyDir)
		goarch := ""
		for _, kv := range c.Env {
			if v, ok := strings.CutPrefix(kv, "GOARCH="); ok {
				goarch = v
			}
		}
		// The copy is next to the package, named with the prefix, and has the Go file, the embedded
		// file and the .syso for the architecture; not the test, not the other directories.
		if filepath.Dir(copyDir) != filepath.Dir(pkgDir) || !strings.HasPrefix(filepath.Base(copyDir), CopyPrefix) {
			t.Errorf("the copy is %s, want a %s* directory next to %s", copyDir, CopyPrefix, pkgDir)
		}
		var files []string
		if err := filepath.WalkDir(copyDir, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				rel, _ := filepath.Rel(copyDir, path)
				files = append(files, filepath.ToSlash(rel))
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
		slices.Sort(files)
		want := []string{"assets/logo.txt", "main.go", "rsrc_windows_" + goarch + ".syso"}
		if !slices.Equal(files, want) {
			t.Errorf("the copy has %v, want %v", files, want)
		}
		// The .syso holds the manifest, 7 icons and their group, and the version.
		syso, err := os.ReadFile(filepath.Join(copyDir, "rsrc_windows_"+goarch+".syso"))
		if err != nil {
			t.Fatal(err)
		}
		obj, err := pe.NewFile(bytes.NewReader(syso))
		if err != nil {
			t.Fatal(err)
		}
		data, err := obj.Sections[0].Data()
		if err != nil {
			t.Fatal(err)
		}
		res := readResources(t, data, 0)
		var manifest, version string
		icons := 0
		for _, r := range res {
			switch r.Type {
			case RTManifest:
				manifest = string(r.Data)
			case RTVersion:
				version = string(r.Data)
			case RTIcon:
				icons++
			}
		}
		for _, want := range []string{`name="org.example.thing"`, `version="1.2.3.4"`, `processorArchitecture="` + goarch + `"`, "<description>Thing</description>"} {
			if !strings.Contains(manifest, want) {
				t.Errorf("manifest for %s has no %s:\n%s", goarch, want, manifest)
			}
		}
		if icons != len(IconSizes) {
			t.Errorf("%d icon images, want %d", icons, len(IconSizes))
		}
		// The version strings are UTF-16.
		for _, want := range []string{"Thing", "1.2.3.4", "app" + map[string]string{"amd64": "_amd64", "arm64": "_arm64"}[goarch] + ".exe"} {
			if !strings.Contains(version, utf16String(want)) {
				t.Errorf("version info for %s has no %q", goarch, want)
			}
		}
	}
	_, err := Build(context.Background(), f, BuildOptions{Package: "./app", Dir: t.TempDir(), Archs: []string{"amd64", "arm64"}, AppID: "org.example.thing", Name: "Thing", Version: "1.2.3.4", Icon: iconPath})
	if err != nil {
		t.Fatal(err)
	}
	if len(copyDirs) != 2 {
		t.Fatalf("%d copies, want 2", len(copyDirs))
	}
	// Afterwards no copy is left, and the package's directory has not been written.
	for _, dir := range copyDirs {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("the copy %s was not removed (%v)", dir, err)
		}
	}
	parent, err := os.ReadDir(filepath.Dir(pkgDir))
	if err != nil || len(parent) != 1 {
		t.Errorf("the parent of the package has %v (%v), want just the package", parent, err)
	}
	entries, err := os.ReadDir(pkgDir)
	if err != nil || len(entries) != 5 {
		t.Errorf("the package directory has %d entries (%v), want the 5 it started with", len(entries), err)
	}
}

// utf16String is the text as the UTF-16 bytes of a version resource, as a string.
func utf16String(s string) string {
	var b []byte
	for _, r := range s {
		b = append(b, byte(r), byte(r>>8))
	}
	return string(b)
}

func TestBuildRemovesTheCopyWhenTheBuildFails(t *testing.T) {
	pkgDir := newPackageDir(t)
	f := &fakeRunner{dir: pkgDir, t: t, failOn: "go build"}
	_, err := Build(context.Background(), f, BuildOptions{Package: "./app", Dir: t.TempDir()})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != TagWindowsCommandFailed {
		t.Fatalf("got %v, want WindowsCommandFailed", err)
	}
	if parent, _ := os.ReadDir(filepath.Dir(pkgDir)); len(parent) != 1 {
		t.Errorf("the parent of the package has %v, want just the package", parent)
	}
}

func TestBuildErrors(t *testing.T) {
	pkgDir := newPackageDir(t)
	pngPath := filepath.Join(t.TempDir(), "bad.png")
	if err := os.WriteFile(pngPath, []byte("not a png"), 0o644); err != nil {
		t.Fatal(err)
	}
	two := `{"Dir": "/x", "Name": "main"}` + "\n" + `{"Dir": "/y", "Name": "main"}`
	tests := []struct {
		name   string
		opts   BuildOptions
		listed string
		failOn string
		route  string
	}{
		{"no package", BuildOptions{}, "", "", "Build.WindowsBuildInvalid"},
		{"an unknown architecture", BuildOptions{Package: "./app", Archs: []string{"386"}}, "", "", "Build.WindowsBuildInvalid"},
		{"a bad version", BuildOptions{Package: "./app", Version: "1.x"}, "", "", "Build->ParseVersion.WindowsResourceInvalid"},
		{"an icon that is not there", BuildOptions{Package: "./app", Icon: "/nonexistent/icon.png"}, "", "", "Build.WindowsIconInvalid"},
		{"an icon that is not a PNG", BuildOptions{Package: "./app", Icon: pngPath}, "", "", "Build->NewIcon.WindowsIconInvalid"},
		{"a bad application ID", BuildOptions{Package: "./app", AppID: "a b"}, "", "", "Build->Manifest.WindowsManifestInvalid"},
		{"a package that is not main", BuildOptions{Package: "./app"}, `{"Dir": "/x", "Name": "lib"}`, "", "Build->listPackage.WindowsBuildInvalid"},
		{"a pattern of two packages", BuildOptions{Package: "./..."}, two, "", "Build->listPackage.WindowsBuildInvalid"},
		{"an answer that is not JSON", BuildOptions{Package: "./app"}, "no packages", "", "Build->listPackage.WindowsBuildInvalid"},
		{"a package with its own .syso", BuildOptions{Package: "./app"}, `{"Dir": "/x", "Name": "main", "SysoFiles": ["mine.syso"]}`, "", "Build->listPackage.WindowsBuildInvalid"},
		{"go list fails", BuildOptions{Package: "./app"}, "", "go list", "Build->listPackage->fakeRunner.Run.WindowsCommandFailed"},
		{"go build fails", BuildOptions{Package: "./app"}, "", "go build", "Build->fakeRunner.Run.WindowsCommandFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeRunner{dir: pkgDir, t: t, listed: tt.listed, failOn: tt.failOn}
			tt.opts.Dir = t.TempDir()
			_, err := Build(context.Background(), f, tt.opts)
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

func TestExecRunner(t *testing.T) {
	out, err := ExecRunner{}.Run(context.Background(), Command{Name: "sh", Args: []string{"-c", "echo $X; echo to-stderr >&2"}, Env: []string{"X=hello"}})
	if err != nil || out != "hello\nto-stderr\n" {
		t.Errorf("Run = %q, %v", out, err)
	}
	_, err = ExecRunner{}.Run(context.Background(), Command{Name: "sh", Args: []string{"-c", "echo boom; exit 3"}})
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("ExecRunner.Run.WindowsCommandFailed") {
		t.Fatalf("got %v, want WindowsCommandFailed from ExecRunner.Run", err)
	}
	if appErr.Metadata["output"] != "boom\n" || !strings.Contains(appErr.Metadata["command"].(string), "sh -c") {
		t.Errorf("metadata = %v", appErr.Metadata)
	}
}

func TestExeName(t *testing.T) {
	tests := []struct {
		out, arch string
		several   bool
		want      string
	}{
		{"app.exe", "amd64", false, "app.exe"},
		{"app.exe", "arm64", true, "app_arm64.exe"},
		{"dist/app", "amd64", true, "dist/app_amd64"},
		{"a.b.exe", "amd64", true, "a.b_amd64.exe"},
	}
	for _, tt := range tests {
		if got := exeName(tt.out, tt.arch, tt.several); got != tt.want {
			t.Errorf("exeName(%q, %q, %v) = %q, want %q", tt.out, tt.arch, tt.several, got, tt.want)
		}
	}
}
