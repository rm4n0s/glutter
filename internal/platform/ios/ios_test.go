//go:build !ios

package ios_test

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/ios"
)

// TestMainFailsOffIOS: a program that calls Main builds everywhere, and Main says it needs iOS.
func TestMainFailsOffIOS(t *testing.T) {
	err := ios.Main(ios.MainOptions{})
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("the error is a %T: %v", err, err)
	}
	if !appErr.HasRoute("Main." + ios.TagIOSUnavailable) {
		t.Errorf("route = %s", appErr.Route())
	}
}

// TestNoPackageOfAnotherPlatformIsLinkedForIOS lists what a program of the module links for GOOS=ios: no platform package of
// another operating system (macOS's AppKit, Linux's Wayland, Windows, Android, the web), which would load a library that iOS lacks.
func TestNoPackageOfAnotherPlatformIsLinkedForIOS(t *testing.T) {
	if testing.Short() || runtime.GOOS == "js" || runtime.GOOS == "windows" {
		t.Skip("runs go list, which a test under Node or Wine cannot")
	}
	for _, pkg := range []string{"./examples/counter", "./examples/gallery", "./examples/cupertino", "./internal/iostest/hello"} {
		t.Run(pkg, func(t *testing.T) {
			cmd := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", pkg)
			cmd.Dir = "../../.."
			cmd.Env = append(cmd.Environ(), "GOOS=ios", "GOARCH=arm64", "CGO_ENABLED=1")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go list: %v\n%s", err, out)
			}
			for _, dep := range strings.Fields(string(out)) {
				for _, other := range []string{"internal/platform/macos", "internal/platform/linux", "internal/platform/windows", "internal/platform/android", "internal/platform/web", "internal/gpu/vulkan", "internal/gpu/gles", "internal/gpu/d3d12"} {
					if strings.HasSuffix(dep, "glutter/"+other) || strings.Contains(dep, "glutter/"+other+"/") {
						t.Errorf("%s is linked for iOS", dep)
					}
				}
			}
		})
	}
}
