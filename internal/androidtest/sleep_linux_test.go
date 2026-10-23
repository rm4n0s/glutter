//go:build linux

package androidtest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// execSleep is a `sleep` that has the environment of the harness for root.
func execSleep(root string) *exec.Cmd {
	cmd := exec.Command("sleep", "30")
	cmd.Env = append(os.Environ(), "ANDROID_AVD_HOME="+filepath.Join(root, "avd"))
	return cmd
}

func TestLeftoversSeesAProcessWithTheHarnessEnvironment(t *testing.T) {
	root := t.TempDir()
	if got := Leftovers(root); len(got) != 0 {
		t.Fatalf("leftovers before: %v", got)
	}
	if _, err := os.Stat("/proc/self/environ"); err != nil {
		t.Skip("no /proc")
	}
	cmd := execSleep(root)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() }() // the sleep is ours
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && len(Leftovers(root)) == 0 {
		time.Sleep(20 * time.Millisecond)
	}
	if got := Leftovers(root); len(got) != 1 || !strings.Contains(got[0], "sleep") {
		t.Errorf("leftovers: %v", got)
	}
}
