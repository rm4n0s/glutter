//go:build !(js && wasm)

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

// TestMainOffTheWeb: the program says that it needs the web, as a logged error with its tag and
// route, and exits with a failure. The test runs the test binary again as the program.
func TestMainOffTheWeb(t *testing.T) {
	if os.Getenv("GPUWEBTEST_AS_PROGRAM") == "1" {
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestMainOffTheWeb$")
	cmd.Env = append(os.Environ(), "GPUWEBTEST_AS_PROGRAM=1")
	var out bytes.Buffer
	cmd.Stderr = &out
	if err := cmd.Run(); err == nil {
		t.Fatal("the program exited with success")
	}
	var entry struct {
		Err errors.ErrorJson `json:"err"`
	}
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.Contains(line, "GPUWebTestNeedsTheWeb") {
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				t.Fatalf("the log entry %q is not JSON: %v", line, err)
			}
		}
	}
	if entry.Err.Tag != "GPUWebTestNeedsTheWeb" || !entry.Err.HasRoute("main.GPUWebTestNeedsTheWeb") {
		t.Errorf("logged %q via %q (stderr: %s)", entry.Err.Tag, entry.Err.Route(), out.String())
	}
}
