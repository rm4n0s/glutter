//go:build !js

package examples

import (
	"context"
	"debug/pe"
	"path/filepath"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/wintool"
)

// TestEveryExampleBuildsForWindows checks what step 28.15 asks: every example, `tray` and `multiwindow` included,
// builds for windows/amd64 and windows/arm64 with CGO_ENABLED=0 (SPEC §19, §25.1), through `glutter-windows build`'s
// builder: each .exe is a GUI program of the right machine type, with its manifest and icon. It is skipped by
// -short, because the first build after a change compiles the framework for Windows twice.
func TestEveryExampleBuildsForWindows(t *testing.T) {
	if testing.Short() {
		t.Skip("builds every example for windows/amd64 and windows/arm64; skipped by -short")
	}
	names := allExamples(t)
	if len(names) < 11 {
		t.Fatalf("found %d examples for Windows, want at least 11: %v", len(names), names)
	}
	machines := map[string]uint16{"amd64": pe.IMAGE_FILE_MACHINE_AMD64, "arm64": pe.IMAGE_FILE_MACHINE_ARM64}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()
			res, err := wintool.Build(ctx, wintool.ExecRunner{}, wintool.BuildOptions{
				Package: "./examples/" + name, Dir: "..", Out: filepath.Join(t.TempDir(), name+".exe"), Archs: []string{"amd64", "arm64"},
			})
			if err != nil {
				t.Fatalf("glutter-windows build ./examples/%s: %v", name, err)
			}
			if len(res.Exes) != 2 {
				t.Fatalf("the build made %d .exe files, want one for amd64 and one for arm64", len(res.Exes))
			}
			for _, exe := range res.Exes {
				f, err := pe.Open(exe.Path)
				if err != nil {
					t.Fatalf("%s: %v", exe.Path, err)
				}
				machine := f.Machine
				var subsystem uint16
				switch oh := f.OptionalHeader.(type) {
				case *pe.OptionalHeader64:
					subsystem = oh.Subsystem
				}
				if err := f.Close(); err != nil {
					t.Error(err)
				}
				if machine != machines[exe.Arch] {
					t.Errorf("%s: machine type %#x, want %#x", exe.Arch, machine, machines[exe.Arch])
				}
				if subsystem != pe.IMAGE_SUBSYSTEM_WINDOWS_GUI {
					t.Errorf("%s: subsystem %d, want the GUI subsystem (%d)", exe.Arch, subsystem, pe.IMAGE_SUBSYSTEM_WINDOWS_GUI)
				}
				if exe.Size < 5<<20 {
					t.Errorf("%s: the .exe is only %d bytes", exe.Arch, exe.Size)
				}
			}
		})
	}
}
