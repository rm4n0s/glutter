package wintool

import (
	"context"
	"debug/pe"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildWithTheGoCommand builds a tiny Windows program with the real go command and reads the
// .exe back with debug/pe: the subsystem (GUI or console), the architecture, the resource directory
// that the linker resolved (the manifest, the icon images, their group and the version information)
// and that the source tree is as it was. It is the test that the temporary copy of SPEC §25.1 makes
// the Go linker take the .syso. It is skipped with -short, since it compiles.
func TestBuildWithTheGoCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles a Windows program")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	const pkg = "./internal/winetest/testdata/hello"
	before := listTree(t, filepath.Join(root, "internal", "winetest", "testdata"))

	tests := []struct {
		name      string
		opts      BuildOptions
		arch      string
		machine   uint16
		subsystem uint16
	}{
		{"a GUI program for amd64", BuildOptions{Package: pkg, Version: "3.1.4.1", Name: "Hello"}, "amd64", pe.IMAGE_FILE_MACHINE_AMD64, 2},
		{"a console program for amd64", BuildOptions{Package: pkg, Console: true}, "amd64", pe.IMAGE_FILE_MACHINE_AMD64, 3},
		{"a GUI program for arm64", BuildOptions{Package: pkg, Archs: []string{"arm64"}}, "arm64", pe.IMAGE_FILE_MACHINE_ARM64, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.opts.Dir = root
			tt.opts.Out = filepath.Join(t.TempDir(), "hello.exe")
			res, err := Build(context.Background(), ExecRunner{}, tt.opts)
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Exes) != 1 || res.Exes[0].Arch != tt.arch {
				t.Fatalf("built %+v", res.Exes)
			}
			f, err := pe.Open(res.Exes[0].Path)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = f.Close() }() // read-only
			if f.Machine != tt.machine {
				t.Errorf("machine = %#x, want %#x", f.Machine, tt.machine)
			}
			oh, ok := f.OptionalHeader.(*pe.OptionalHeader64)
			if !ok {
				t.Fatalf("optional header is %T", f.OptionalHeader)
			}
			if oh.Subsystem != tt.subsystem {
				t.Errorf("subsystem = %d, want %d", oh.Subsystem, tt.subsystem)
			}
			sec := f.Section(".rsrc")
			if sec == nil {
				t.Fatal("the .exe has no .rsrc section: the linker did not take the .syso")
			}
			if dir := oh.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE]; dir.VirtualAddress != sec.VirtualAddress || dir.Size == 0 {
				t.Errorf("resource data directory = %+v, want it at the .rsrc section %#x", dir, sec.VirtualAddress)
			}
			data, err := sec.Data()
			if err != nil {
				t.Fatal(err)
			}
			res2 := readResources(t, data, sec.VirtualAddress) // the relocations made the offsets RVAs
			var manifest string
			var icons, groups, versions int
			for _, r := range res2 {
				switch r.Type {
				case RTManifest:
					manifest = string(r.Data)
				case RTIcon:
					icons++
				case RTGroupIcon:
					groups++
				case RTVersion:
					versions++
					if want := "3.1.4.1"; tt.opts.Version != "" && !strings.Contains(string(r.Data), utf16String(want)) {
						t.Errorf("the version information has no %s", want)
					}
				}
			}
			if icons != len(IconSizes) || groups != 1 || versions != 1 {
				t.Errorf("icons %d groups %d versions %d", icons, groups, versions)
			}
			for _, want := range []string{"PerMonitorV2", "UTF-8", `processorArchitecture="` + tt.arch + `"`, "Microsoft.Windows.Common-Controls"} {
				if !strings.Contains(manifest, want) {
					t.Errorf("manifest has no %q:\n%s", want, manifest)
				}
			}
		})
	}
	if after := listTree(t, filepath.Join(root, "internal", "winetest", "testdata")); after != before {
		t.Errorf("the source tree changed:\nbefore: %s\nafter:  %s", before, after)
	}
	if entries, _ := filepath.Glob(filepath.Join(root, "internal", "winetest", "testdata", CopyPrefix+"*")); len(entries) != 0 {
		t.Errorf("temporary copies left: %v", entries)
	}
}

// listTree is the names of the files below dir, as one string.
func listTree(t *testing.T, dir string) string {
	t.Helper()
	var names []string
	if err := filepath.WalkDir(dir, func(path string, _ os.DirEntry, err error) error {
		names = append(names, path)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return strings.Join(names, "\n")
}
