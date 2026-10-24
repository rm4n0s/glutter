//go:build glutter_android

package androidtest

import (
	"context"
	"strings"
	"testing"
	"time"
)

// spellApp installs examples/dev/android-spell, which logs the answers of the spell checker, and starts
// it with the renderer soft and the dev events script. It returns once the app has presented its first
// frame.
func spellApp(ctx context.Context, t *testing.T, e *Emulator, script string) string {
	t.Helper()
	appID := installApp(ctx, t, e, "./examples/dev/android-spell")
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })
	extras := map[string]string{"GLUTTER_RENDERER": "soft"}
	if script != "" {
		extras["GLUTTER_DEV_EVENTS"] = script
	}
	if err := e.Launch(ctx, appID, extras); err != nil {
		t.Fatal(Describe(err))
	}
	if _, err := e.WaitLog(ctx, appID, "first frame presented", time.Minute); err != nil {
		t.Fatal(Describe(err))
	}
	return appID
}

// TestAMisspelledWordIsFoundBySystemsSpellChecker types misspelled words with the dev events and reads
// what the framework's spell check got from TextServicesManager. On the API 26 image Gboard's spell checker
// has its dictionary and finds the word with suggestions; on the API 37 image it has none that works
// offline and says that every word is in the dictionary, so there the test checks that the request went
// through the system's service and came back without an error (SPEC §24.6).
func TestAMisspelledWordIsFoundBySystemsSpellChecker(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := spellApp(ctx, t, e, "wait:3s;text:helo wrld")
		log, err := e.WaitLog(ctx, appID, `text="helo wrld" ok=true`, 60*time.Second)
		if err != nil {
			full, _ := e.FullLog(context.Background())
			t.Fatal(Describe(err), "\n", grep(full, "spellcheck", "SpellCheck", "GlutterJava"))
		}
		var line string
		for _, l := range lines(log, `spellcheck supported=`) {
			line = l
		}
		if !strings.Contains(line, "supported=true") || !strings.Contains(line, "err=<nil>") {
			t.Fatalf("the system's spell checker was not asked, or failed:\n%s", line)
		}
		if e.AVD.API == "26" && !strings.Contains(line, "5-9[world") {
			t.Errorf("the spell checker did not find wrld, with world as the first suggestion:\n%s", line)
		}
	})
}

// TestTheSuggestionsToolbarReplacesAMisspelledWord taps the underlined word, which opens the toolbar with
// the suggestions, and taps the first one: the word is replaced. It runs on the API 26 image, whose
// spell checker has suggestions (see the test above).
func TestTheSuggestionsToolbarReplacesAMisspelledWord(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		if e.AVD.API != "26" {
			t.Skip("the spell checker of this image has no dictionary that works offline")
		}
		// The word is at (160, 330) on the 1080 x 2400 screen, and the first suggestion at (120, 420).
		appID := spellApp(ctx, t, e, "wait:3s;text:helo wrld ok;wait:3s;tap:160,330;wait:2s;tap:120,420")
		if _, err := e.WaitLog(ctx, appID, `field changed text="helo world ok"`, 60*time.Second); err != nil {
			full, _ := e.FullLog(context.Background())
			t.Fatal(Describe(err), "\n", grep(full, "spellcheck", "field changed"))
		}
	})
}
