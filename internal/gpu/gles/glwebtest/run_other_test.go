//go:build !(js && wasm)

package main

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
)

// TestMainNeedsTheWeb runs the program (this test binary, with the variable that makes it call
// main) and reads the error that it logs: the entry has the tag and the route, which ends in main.
func TestMainNeedsTheWeb(t *testing.T) {
	if os.Getenv("GLWEBTEST_RUN_MAIN") == "1" {
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestMainNeedsTheWeb$")
	cmd.Env = append(os.Environ(), "GLWEBTEST_RUN_MAIN=1")
	var out bytes.Buffer
	cmd.Stderr = &out
	err := cmd.Run()
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
		t.Fatalf("exit = %v, want status 1", err)
	}
	for _, want := range []string{"GLWebTestNeedsTheWeb", "main.GLWebTestNeedsTheWeb"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Errorf("the log %q does not have %q", out.String(), want)
		}
	}
}
