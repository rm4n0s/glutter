//go:build glutter_android

package androidtest

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// servicesApp installs examples/dev/android-services and starts it with the renderer soft and the demo
// words (GLUTTER_DEMO), which the app runs 3 seconds after it started. The notification permission is
// granted first when grant is true (Android 13 and later asks for it otherwise).
func servicesApp(ctx context.Context, t *testing.T, e *Emulator, demo string, grant bool) string {
	t.Helper()
	appID := installApp(ctx, t, e, "./examples/dev/android-services")
	t.Cleanup(func() {
		_, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID)
		_, _ = e.ADB.Output(context.Background(), "shell", "cmd", "notification", "cancel_all", appID)
	})
	// A browser that an earlier run opened can still sit on top of the app's task, and Android would
	// bring that task back with its first intent: clearing the app removes its tasks.
	_, _ = e.ADB.Output(ctx, "shell", "pm", "clear", appID)
	if grant {
		// Below Android 13 the permission does not exist and pm says so: it is not an error here.
		_, _ = e.ADB.Output(ctx, "shell", "pm", "grant", appID, "android.permission.POST_NOTIFICATIONS")
	}
	if err := e.Launch(ctx, appID, map[string]string{"GLUTTER_RENDERER": "soft", "GLUTTER_DEMO": demo}); err != nil {
		t.Fatal(Describe(err))
	}
	if _, err := e.WaitLog(ctx, appID, "first frame presented", time.Minute); err != nil {
		t.Fatal(Describe(err))
	}
	return appID
}

// TestOpenURIStartsAnActivityForTheLink opens an https URI: Android starts a VIEW intent for it, which
// the activity manager shows.
func TestOpenURIStartsAnActivityForTheLink(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := servicesApp(ctx, t, e, "open", false)
		if _, err := e.WaitLog(ctx, appID, "open result uri=https://example.com err=none", 40*time.Second); err != nil {
			t.Fatal(Describe(err))
		}
		for end := time.Now().Add(30 * time.Second); ; time.Sleep(time.Second) {
			out := shell(ctx, t, e, "dumpsys", "activity", "activities")
			if strings.Contains(out, "dat=https://example.com") {
				return
			}
			if time.Now().After(end) {
				t.Fatalf("the activity manager shows no VIEW intent for the link:\n%s", grep(out, "example.com", "VIEW"))
			}
		}
	})
}

// TestOpenURIOfASchemeNoAppHandlesIsUnavailable opens a URI of a scheme that nothing handles.
func TestOpenURIOfASchemeNoAppHandlesIsUnavailable(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := servicesApp(ctx, t, e, "openbad", false)
		if _, err := e.WaitLog(ctx, appID, "err=OpenURIUnavailable", 40*time.Second); err != nil {
			t.Fatal(Describe(err))
		}
	})
}

// notificationDump is the dump of the notification service, with the texts shown.
func notificationDump(ctx context.Context, t *testing.T, e *Emulator) string {
	t.Helper()
	return shell(ctx, t, e, "dumpsys", "notification", "--noredact")
}

// TestANotificationIsPostedWithItsActionButton shows a notification: the notification service has its
// title and its action button.
func TestANotificationIsPostedWithItsActionButton(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := servicesApp(ctx, t, e, "notify", true)
		log, err := e.WaitLog(ctx, appID, "notify result id=", 40*time.Second)
		if err != nil {
			t.Fatal(Describe(err))
		}
		if got := lines(log, "notify result"); len(got) != 1 || !strings.Contains(got[0], "err=none") {
			t.Fatalf("the notification was not posted:\n%s", strings.Join(got, "\n"))
		}
		for end := time.Now().Add(30 * time.Second); ; time.Sleep(time.Second) {
			out := notificationDump(ctx, t, e)
			if i := strings.Index(out, "pkg="+appID); i >= 0 && strings.Contains(out[i:], "Glutter") && strings.Contains(out[i:], "Reply") {
				return
			}
			if time.Now().After(end) {
				t.Fatalf("the notification service has no notification of the app with its action:\n%s", grep(out, appID, "Reply", "Glutter"))
			}
		}
	})
}

// TestADeniedNotificationPermissionIsNotificationsUnavailable revokes the permission and makes the
// decision the user's (fixed), so that the system answers the request without a dialog: Android 13 and
// later only.
func TestADeniedNotificationPermissionIsNotificationsUnavailable(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	appID := installApp(ctx, t, e, "./examples/dev/android-services")
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })
	_, _ = e.ADB.Output(ctx, "shell", "pm", "clear", appID)
	shell(ctx, t, e, "pm", "revoke", appID, "android.permission.POST_NOTIFICATIONS")
	shell(ctx, t, e, "pm", "set-permission-flags", appID, "android.permission.POST_NOTIFICATIONS", "user-fixed")
	if err := e.Launch(ctx, appID, map[string]string{"GLUTTER_RENDERER": "soft", "GLUTTER_DEMO": "notify"}); err != nil {
		t.Fatal(Describe(err))
	}
	if _, err := e.WaitLog(ctx, appID, "err=NotificationsUnavailable", 60*time.Second); err != nil {
		t.Fatal(Describe(err))
	}
}

// TestACloseByTheAppRemovesTheNotification closes the notification from the app: it is gone from the
// notification service and the app is told it closed by the app (reason 3).
func TestACloseByTheAppRemovesTheNotification(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := servicesApp(ctx, t, e, "notifyclose", true)
		if _, err := e.WaitLog(ctx, appID, "notification closed reason=3", 60*time.Second); err != nil {
			t.Fatal(Describe(err))
		}
		time.Sleep(2 * time.Second)
		out := notificationDump(ctx, t, e)
		if i := strings.Index(out, "pkg="+appID); i >= 0 && strings.Contains(out[i:], "Reply") {
			t.Errorf("the notification is still shown:\n%s", grep(out, appID, "Reply"))
		}
	})
}

// TestTheActionButtonCallsTheApp taps the notification's action button (the notification shade is
// opened with the status bar service and the button is found with uiautomator): the app is told the id of
// the action. With GLUTTER_PROOF_OUT set to a file name the shade's screen is written there.
func TestTheActionButtonCallsTheApp(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	appID := servicesApp(ctx, t, e, "notify", true)
	if _, err := e.WaitLog(ctx, appID, "notify result id=", 40*time.Second); err != nil {
		t.Fatal(Describe(err))
	}
	time.Sleep(2 * time.Second)
	shell(ctx, t, e, "cmd", "statusbar", "expand-notifications")
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "cmd", "statusbar", "collapse") })
	var reply uiNode
	var found bool
	for end := time.Now().Add(40 * time.Second); ; time.Sleep(2 * time.Second) {
		_, _ = e.ADB.Output(ctx, "shell", "uiautomator", "dump", "/sdcard/glutter-ui.xml")
		b, err := e.ADB.Output(ctx, "shell", "cat", "/sdcard/glutter-ui.xml")
		if err == nil {
			var root struct {
				Nodes []uiNode `xml:"node"`
			}
			if xml.Unmarshal([]byte(b), &root) == nil {
				for _, n := range root.Nodes {
					for _, c := range n.flat() {
						if strings.EqualFold(c.Text, "Reply") && c.Bounds != "" {
							reply, found = c, true
						}
					}
				}
			}
		}
		if found || time.Now().After(end) {
			break
		}
	}
	if !found {
		t.Fatal("the notification shade has no Reply button")
	}
	if out := os.Getenv("GLUTTER_PROOF_OUT"); out != "" {
		b, err := e.ADB.Screencap(ctx)
		if err != nil {
			t.Fatal(Describe(err))
		}
		if err := os.WriteFile(filepath.Clean(out), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	x, y := boundsCenter(t, reply.Bounds)
	shell(ctx, t, e, "input", "tap", x, y)
	if _, err := e.WaitLog(ctx, appID, "notification action id=reply", 40*time.Second); err != nil {
		t.Fatal(Describe(err))
	}
}

// boundsCenter is the center of "[x0,y0][x1,y1]" as the arguments of input tap.
func boundsCenter(t *testing.T, bounds string) (x, y string) {
	t.Helper()
	var x0, y0, x1, y1 int
	if _, err := fmt.Sscanf(bounds, "[%d,%d][%d,%d]", &x0, &y0, &x1, &y1); err != nil {
		t.Fatalf("the bounds %q: %v", bounds, err)
	}
	return strconv.Itoa((x0 + x1) / 2), strconv.Itoa((y0 + y1) / 2)
}
