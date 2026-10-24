//go:build glutter_android

package androidtest

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"
)

// textApp installs examples/dev/android-text, which logs what the framework hears of its three text
// fields, and starts it with the renderer soft, the field that has the focus (GLUTTER_DEMO) and the dev
// events script. It returns when the app has presented its first frame.
func textApp(ctx context.Context, t *testing.T, e *Emulator, demo, script string) string {
	t.Helper()
	appID := installApp(ctx, t, e, "./examples/dev/android-text")
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })
	startText(ctx, t, e, appID, demo, script)
	return appID
}

// startText starts the installed app again.
func startText(ctx context.Context, t *testing.T, e *Emulator, appID, demo, script string) {
	t.Helper()
	extras := map[string]string{"GLUTTER_RENDERER": "soft", "GLUTTER_DEMO": demo}
	if script != "" {
		extras["GLUTTER_DEV_EVENTS"] = script
	}
	if err := e.Launch(ctx, appID, extras); err != nil {
		t.Fatal(Describe(err))
	}
	if _, err := e.WaitLog(ctx, appID, "first frame presented", time.Minute); err != nil {
		t.Fatal(Describe(err))
	}
}

// TestScriptedTextIsTypedAndComposed has the dev events type through the input connection, as an input
// method does: committed text (with an accent and an emoji, which is two UTF-16 units), and a
// composition that ends committed.
func TestScriptedTextIsTypedAndComposed(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := textApp(ctx, t, e, "single", "wait:3s;text:héllo😀;wait:500ms;compose:k,ka,か")
		if _, err := e.WaitLog(ctx, appID, "name=single text=héllo😀", 40*time.Second); err != nil {
			t.Fatal(Describe(err))
		}
		if _, err := e.WaitLog(ctx, appID, "name=single text=héllo😀か", 40*time.Second); err != nil {
			t.Fatal(Describe(err))
		}
	})
}

// TestTheKeyboardsActionSubmitsASingleLineField types a word and has the input method do the action of
// its Enter key.
func TestTheKeyboardsActionSubmitsASingleLineField(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := textApp(ctx, t, e, "single", "wait:3s;text:go;wait:300ms;action:enter")
		if _, err := e.WaitLog(ctx, appID, "field submitted name=single text=go", 40*time.Second); err != nil {
			t.Fatal(Describe(err))
		}
	})
}

// TestAMultilineFieldTakesANewLineFromTheKeyboard does the keyboard's action in a multiline field: a line
// break, not a submit.
func TestAMultilineFieldTakesANewLineFromTheKeyboard(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := textApp(ctx, t, e, "multi", "wait:3s;text:ab;wait:300ms;action:enter;wait:300ms;text:cd")
		if _, err := e.WaitLog(ctx, appID, `name=multi text="ab\ncd"`, 40*time.Second); err != nil {
			full, _ := e.FullLog(context.Background())
			t.Fatal(Describe(err), "\n", grep(full, "GlutterJava"))
		}
		log, err := e.Log(ctx, appID)
		if err != nil {
			t.Fatal(Describe(err))
		}
		if strings.Contains(log, "field submitted") {
			t.Errorf("the multiline field was submitted:\n%s", grep(log, "field "))
		}
	})
}

// TestAPasswordFieldAsksForAPasswordKeyboard looks at what the input method was told: the input type of a
// text password, which has no suggestions, in the dump of the input method service.
func TestAPasswordFieldAsksForAPasswordKeyboard(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := textApp(ctx, t, e, "password", "")
		time.Sleep(4 * time.Second)
		_ = appID
		dump := shell(ctx, t, e, "dumpsys", "input_method")
		// "inputType=0x80081": TYPE_CLASS_TEXT | TYPE_TEXT_VARIATION_PASSWORD | TYPE_TEXT_FLAG_NO_SUGGESTIONS.
		if !strings.Contains(dump, "inputType=0x80081") {
			t.Errorf("the input method was not asked for a password:\n%s", grep(dump, "inputType", "EditorInfo", "mCurAttribute"))
		}
	})
}

// TestTheKeyboardsHeightIsTheBottomViewInset shows the soft keyboard (the field has the focus from the start) and
// compares the inset that the framework gets with the keyboard that the system reports.
func TestTheKeyboardsHeightIsTheBottomViewInset(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := textApp(ctx, t, e, "single", "")
		var inset, dpr float64
		// The keyboard's height is the last value of its animation: it is read until it has not changed
		// for two reads (a slow emulator takes more than half a second to slide it in).
		var previous float64
		for end := time.Now().Add(40 * time.Second); inset == 0 || inset != previous; time.Sleep(700 * time.Millisecond) {
			previous = inset
			log, err := e.Log(ctx, appID)
			if err != nil {
				t.Fatal(Describe(err))
			}
			for _, l := range strings.Split(log, "\n") {
				if i := strings.Index(l, "INFO keyboard viewInsetsBottom="); i >= 0 {
					f := strings.Fields(l[i:])
					inset, _ = strconv.ParseFloat(strings.TrimPrefix(f[2], "viewInsetsBottom="), 64)
					dpr, _ = strconv.ParseFloat(strings.TrimPrefix(f[3], "dpr="), 64)
				}
			}
			if time.Now().After(end) {
				t.Fatalf("the keyboard's height never reached the framework:\n%s", grep(log, "keyboard"))
			}
		}
		pixels := inset * dpr
		if pixels < 300 || pixels > 1200 {
			t.Errorf("the keyboard is %v pixels high", pixels)
		}
		if e.AVD.API != "26" {
			out := shell(ctx, t, e, "dumpsys", "window")
			ime := barHeight(out, "ime")
			if ime == 0 {
				t.Fatalf("dumpsys window has no keyboard:\n%s", grep(out, "type=ime"))
			}
			if d := pixels - float64(ime); d < -2 || d > 2 {
				t.Errorf("the bottom inset is %v pixels and the keyboard %d", pixels, ime)
			}
		}
	})
}

// TestTheClipboardIsTheSystems copies text in one run of the app and pastes it in the next: the text
// went through the ClipboardManager of Android.
func TestTheClipboardIsTheSystems(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := textApp(ctx, t, e, "single,copy", "")
		if _, err := e.WaitLog(ctx, appID, "clipboard copied err=<nil>", 40*time.Second); err != nil {
			t.Fatal(Describe(err))
		}
		shell(ctx, t, e, "am", "force-stop", appID)
		startText(ctx, t, e, appID, "single,paste", "")
		if _, err := e.WaitLog(ctx, appID, `clipboard pasted text="from the app"`, 40*time.Second); err != nil {
			t.Fatal(Describe(err))
		}
	})
}
