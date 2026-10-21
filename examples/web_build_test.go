//go:build !js

package examples

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// buildForWeb builds ./examples/<name> for js/wasm and fails the test with the output of the go
// command when it does not build. The program is written to a temporary file.
func buildForWeb(t *testing.T, name string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", filepath.Join(t.TempDir(), name+".wasm"), "./examples/"+name)
	cmd.Dir = ".."
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm", "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("GOOS=js GOARCH=wasm go build ./examples/%s: %v\n%s", name, err, out)
	}
}
