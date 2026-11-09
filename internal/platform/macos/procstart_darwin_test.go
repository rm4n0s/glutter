//go:build darwin && !ios

package macos

import (
	"os/exec"
	"testing"
	"time"
)

func TestProcessStartIsWhenTheProcessStarted(t *testing.T) {
	got, ok := ProcessStart()
	if !ok {
		t.Fatal("the kernel gave no start time")
	}
	// This test binary started before the test, and not long ago.
	if got.After(time.Now()) || time.Since(got) > 30*time.Minute {
		t.Errorf("ProcessStart = %v (now %v)", got, time.Now())
	}
	// A process that is started now says that it started now.
	before := time.Now().Add(-2 * time.Second)
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	child, ok := processStartOf(cmd.Process.Pid)
	if !ok {
		t.Fatal("the kernel gave no start time of the child")
	}
	if child.Before(before) || child.After(time.Now().Add(2*time.Second)) {
		t.Errorf("the child started at %v, between %v and now", child, before)
	}
	if _, ok := processStartOf(1 << 30); ok {
		t.Error("a process that is not there has a start time")
	}
}
