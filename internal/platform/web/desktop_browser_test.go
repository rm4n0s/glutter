//go:build glutter_browser

package web_test

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/webtest"
	"github.com/rm4n0s/glutter/internal/webtool"
)

// The desktop tests build internal/platform/web/desktoptest, a small app that replaces window.open and the
// Notification constructor of the page by stubs (a headless browser cannot answer a permission prompt, and
// opening real windows is not wanted), calls desktop.OpenURI and desktop.ShowNotification from a goroutine
// as an app does, and posts what it saw (SPEC §23.7). The code under test is the real one; the browser's
// objects are the stubs.

var (
	desktopOnce sync.Once
	desktopDir  string
	desktopErr  error
)

func desktopSite(t *testing.T) string {
	t.Helper()
	desktopOnce.Do(func() {
		if desktopDir, desktopErr = os.MkdirTemp("", "glutter-web-desktoptest-"); desktopErr != nil {
			return
		}
		_, desktopErr = webtool.Build(context.Background(), webtool.BuildOptions{
			Package: "./internal/platform/web/desktoptest", Dir: webtest.ModuleRoot(t), OutDir: desktopDir,
		})
	})
	if desktopErr != nil {
		t.Fatalf("building the desktop test app for the web: %v", desktopErr)
	}
	return desktopDir
}

func cleanupDesktopSite() {
	if desktopDir != "" {
		_ = os.RemoveAll(desktopDir) // a temporary directory: a leftover is harmless
	}
}

type noteOptions struct {
	Title              string `json:"title"`
	Body               string `json:"body"`
	Icon               string `json:"icon"`
	Tag                string `json:"tag"`
	RequireInteraction bool   `json:"requireInteraction"`
	Silent             bool   `json:"silent"`
}

type desktopObservations struct {
	Opened              [][]string  `json:"opened"`
	OpenTags            []string    `json:"openTags"`
	InvalidTag          string      `json:"invalidTag"`
	Refused             string      `json:"refused"`
	RefusedAgain        string      `json:"refusedAgain"`
	PromptsAfter        int         `json:"promptsAfter"`
	NotesWhileRefused   int         `json:"notesWhileRefused"`
	Granted             string      `json:"granted"`
	First               noteOptions `json:"first"`
	Actions             []string    `json:"actions"`
	Reasons             []int       `json:"reasons"`
	BrowserClosed       bool        `json:"browserClosed"`
	CloseTag            string      `json:"closeTag"`
	Replaced            noteOptions `json:"replaced"`
	SameID              bool        `json:"sameID"`
	NoAPI               string      `json:"noAPI"`
	CallbacksOnUIThread bool        `json:"callbacksOnUIThread"`
}

// TestBrowserDesktopServices: in Chromium and Firefox, OpenURI calls window.open(uri, "_blank", "noopener")
// and refuses a javascript: URI before it; a Notifications permission that the user refuses is
// NotificationsUnavailable (and the browser is not asked again, and no notification is made); a granted
// permission shows the notification with the title, the plain-text body, the icon URL, the tag and the
// flags of the critical urgency; a click calls OnAction("default") and the close of the browser OnClosed
// (dismissed); CloseNotification closes it in the browser with the reason of the app; ReplacesID keeps the
// tag and the ID; a page with no Notification is NotificationsUnavailable. Every callback runs on the UI
// thread.
func TestBrowserDesktopServices(t *testing.T) {
	dir := desktopSite(t)
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		res, err := webtest.Await(ctx, webtest.ShotOptions{Dir: dir, Browser: b, Width: 480, Height: 640}, "desktop")
		if err != nil {
			t.Fatalf("Await in %s: %v", b, err)
		}
		var o desktopObservations
		if err := json.Unmarshal(res.Data, &o); err != nil {
			t.Fatalf("the result is not the observations: %v\n%s", err, res.Data)
		}
		t.Logf("%+v", o)
		check := func(name string, got, want any) {
			t.Helper()
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s = %+v, want %+v", name, got, want)
			}
		}
		// OpenURI.
		check("the calls of window.open", o.Opened, [][]string{
			{"https://example.org/a?b=c", "_blank", "noopener"},
			{"https://throws.example/", "_blank", "noopener"},
		})
		check("the tags of OpenURI", o.OpenTags, []string{"", "OpenURISchemeBlocked", "OpenURIUnavailable"})
		check("a URI without a scheme", o.InvalidTag, "OpenURIInvalid")

		// A refused permission.
		check("a refused permission", o.Refused, "NotificationsUnavailable")
		check("a refused permission, again", o.RefusedAgain, "NotificationsUnavailable")
		check("the browser was asked for the permission once", o.PromptsAfter, 1)
		check("notifications made without the permission", o.NotesWhileRefused, 0)

		// A granted one.
		check("the granted call", o.Granted, "")
		check("the first notification", o.First, noteOptions{
			Title: "Build done", Body: "2 files", Icon: "/icons/done.png", Tag: "glutter-1", RequireInteraction: true,
		})
		check("the actions after the click", o.Actions, []string{"default"})
		check("the close reasons: dismissed after a click, then by the app", o.Reasons, []int{2, 3})
		check("CloseNotification closed it in the browser", o.BrowserClosed, true)
		check("CloseNotification's tag", o.CloseTag, "")
		check("a replacing notification keeps the tag", o.Replaced, noteOptions{Title: "Download 50%", Tag: "glutter-3"})
		check("a replacing notification keeps the ID", o.SameID, true)
		check("a page with no Notification", o.NoAPI, "NotificationsUnavailable")
		check("every callback ran on the UI thread", o.CallbacksOnUIThread, true)
	})
}
