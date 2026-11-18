//go:build glutter_ios

package iostest_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/iostest"
	"github.com/rm4n0s/glutter/internal/issues"
)

// runIssue installs internal/iostest/issueapp on an iPhone, launches it with GLUTTER_ISSUE naming the issue and returns the lines it
// printed once it printed issues.DoneLine; it fails the test with the app's error when the app printed issues.FailedLine.
func runIssue(t *testing.T, issue string) []string {
	t.Helper()
	h := iostest.New(t)
	dev, res := installed(t, h, "./internal/iostest/issueapp")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID, issues.Env+"="+issue)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		out := app.Output()
		for _, line := range out {
			if strings.Contains(line, issues.FailedLine) {
				t.Fatalf("the app failed: %s\noutput:\n%s", line, strings.Join(out, "\n"))
			}
			if strings.Contains(line, issues.DoneLine) {
				return out
			}
		}
		if app.Ended() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("the app did not end its report; output:\n%s", strings.Join(app.Output(), "\n"))
	return nil
}

// TestIOSDrawsAFrameThatFillsTheAtlas runs the reproduction of the issue of step 46.1 (internal/issues, atlas-full) on the Metal of
// the Simulator, in an app that draws its frames with the explicit renderer, strict, into a texture of 1920x1080 (step 47.1). The
// check is the one of every other platform.
func TestIOSDrawsAFrameThatFillsTheAtlas(t *testing.T) {
	out := runIssue(t, issues.AtlasFullName)
	rep, err := issues.ParseAtlasFull(out)
	if err != nil {
		t.Fatalf("the report: %+v", err)
	}
	for _, line := range rep.Lines() {
		t.Log(line)
	}
	for _, problem := range rep.Problems() {
		t.Error(problem)
	}
}
