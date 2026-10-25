package windows

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// balloonLog records the calls of the callbacks of the notifications.
type balloonLog struct {
	actions []string
	closed  map[string]mobile.CloseReason
	order   []string
}

func newBalloonLog() *balloonLog { return &balloonLog{closed: map[string]mobile.CloseReason{}} }

// spec makes a notification named name whose callbacks write to the log.
func (l *balloonLog) spec(name string, mods ...func(*mobile.NotificationSpec)) mobile.NotificationSpec {
	s := mobile.NotificationSpec{
		Title: name, Body: "body of " + name,
		Actions: []mobile.Action{{ID: mobile.DefaultAction, Label: "Open"}},
		OnAction: func(id string) {
			l.actions = append(l.actions, name+":"+id)
		},
		OnClose: func(r mobile.CloseReason) {
			l.closed[name] = r
			l.order = append(l.order, "closed:"+name)
		},
	}
	for _, m := range mods {
		m(&s)
	}
	return s
}

// mustShow shows the notification and fails the test when the call does.
func mustShow(t *testing.T, d *Desktop, spec mobile.NotificationSpec) uint32 {
	t.Helper()
	id, err := d.ShowNotification(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func shown(sh *fakeShell) iconData {
	for i := len(sh.notify) - 1; i >= 0; i-- {
		if sh.notify[i].Command == win32.NIMModify && sh.notify[i].Data.Flags&win32.NIFInfo != 0 {
			return sh.notify[i].Data
		}
	}
	return iconData{}
}

func TestBalloonShowsOneAtATimeAndQueuesTheRest(t *testing.T) {
	d, sh := newTestDesktop()
	l := newBalloonLog()
	ctx := t.Context()
	id1, err := d.ShowNotification(ctx, l.spec("one"))
	if err != nil {
		t.Fatal(err)
	}
	if got := shown(sh); got.InfoTitle != "one" || got.Info != "body of one" || got.ID != balloonIconID {
		t.Fatalf("the first balloon = %+v", got)
	}
	// The hidden icon of their own was added, hidden, with version 4.
	if got, want := sh.commands()[:3], []string{"add#2", "version#2", "modify#2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("commands = %v, want %v", got, want)
	}
	if add := sh.notify[0].Data; add.State != win32.NISHidden || add.StateMask != win32.NISHidden || add.Flags&win32.NIFState == 0 {
		t.Errorf("the owner icon = %+v: it is hidden", add)
	}
	id2, err := d.ShowNotification(ctx, l.spec("two"))
	if err != nil {
		t.Fatal(err)
	}
	id3, err := d.ShowNotification(ctx, l.spec("three"))
	if err != nil {
		t.Fatal(err)
	}
	if id1 == 0 || id1 == id2 || id2 == id3 || id1 == id3 {
		t.Errorf("IDs %d %d %d are not three different numbers", id1, id2, id3)
	}
	if got := shown(sh); got.InfoTitle != "one" {
		t.Errorf("a second notification replaced the first on the screen: %+v", got)
	}

	// The click on the shown one calls "default", closes it as dismissed, and starts the next.
	d.onIconMessage(0, uintptr(win32.NINBalloonUserClick)|uintptr(balloonIconID)<<16)
	if !reflect.DeepEqual(l.actions, []string{"one:default"}) || l.closed["one"] != mobile.CloseDismissed {
		t.Errorf("actions %v, closed %v", l.actions, l.closed)
	}
	if got := shown(sh); got.InfoTitle != "two" {
		t.Errorf("after the click the balloon is %+v, want two", got)
	}
	// The echo of the click (a NIN_BALLOONHIDE) does not end the next one.
	d.onIconMessage(0, uintptr(win32.NINBalloonHide)|uintptr(balloonIconID)<<16)
	if _, ended := l.closed["two"]; ended {
		t.Error("the echo of a click ended the next balloon")
	}
	// The shell's time-out ends the second as expired.
	d.onIconMessage(0, uintptr(win32.NINBalloonTimeout)|uintptr(balloonIconID)<<16)
	if l.closed["two"] != mobile.CloseExpired {
		t.Errorf("two closed as %v", l.closed["two"])
	}
	if got := shown(sh); got.InfoTitle != "three" {
		t.Errorf("the balloon is %+v, want three", got)
	}
	// A hide that nobody caused is the user dismissing the balloon.
	d.onIconMessage(0, uintptr(win32.NINBalloonHide)|uintptr(balloonIconID)<<16)
	if l.closed["three"] != mobile.CloseDismissed {
		t.Errorf("three closed as %v", l.closed["three"])
	}
	// With nothing left, the hidden icon goes.
	if last := sh.commands(); last[len(last)-1] != "delete#2" {
		t.Errorf("commands end with %v: the owner icon stays", last[len(last)-3:])
	}
	if sh.armed() != 0 {
		t.Errorf("%d timers are still armed", sh.armed())
	}
	if !reflect.DeepEqual(l.order, []string{"closed:one", "closed:two", "closed:three"}) {
		t.Errorf("the order of the closes: %v", l.order)
	}
}

func TestBalloonReplaceInPlaceAndInTheQueue(t *testing.T) {
	d, sh := newTestDesktop()
	l := newBalloonLog()
	ctx := t.Context()
	id1, _ := d.ShowNotification(ctx, l.spec("one"))
	id2, _ := d.ShowNotification(ctx, l.spec("two"))

	// The shown one is replaced in place: same ID, new text, no close of the old one.
	got, err := d.ShowNotification(ctx, l.spec("one again", func(s *mobile.NotificationSpec) { s.ReplacesID = id1 }))
	if err != nil || got != id1 {
		t.Fatalf("replace the shown one: id %d, %v; want %d", got, err, id1)
	}
	if s := shown(sh); s.InfoTitle != "one again" {
		t.Errorf("the balloon is %+v", s)
	}
	if len(l.closed) != 0 {
		t.Errorf("a replaced notification was closed: %v", l.closed)
	}
	// The queued one is replaced in the queue, in its place.
	got, err = d.ShowNotification(ctx, l.spec("two again", func(s *mobile.NotificationSpec) { s.ReplacesID = id2 }))
	if err != nil || got != id2 {
		t.Fatalf("replace the queued one: id %d, %v; want %d", got, err, id2)
	}
	d.onIconMessage(0, uintptr(win32.NINBalloonTimeout)|uintptr(balloonIconID)<<16)
	if s := shown(sh); s.InfoTitle != "two again" {
		t.Errorf("the next balloon is %+v, want the replaced text", s)
	}
	if l.closed["one again"] != mobile.CloseExpired {
		t.Errorf("closed: %v", l.closed)
	}
	// A replace of an ID that is gone is a new notification.
	got, err = d.ShowNotification(ctx, l.spec("late", func(s *mobile.NotificationSpec) { s.ReplacesID = 999 }))
	if err != nil || got == 999 || got == 0 {
		t.Errorf("replace of an unknown ID: id %d, %v", got, err)
	}
}

func TestBalloonClose(t *testing.T) {
	d, sh := newTestDesktop()
	l := newBalloonLog()
	ctx := t.Context()
	id1, _ := d.ShowNotification(ctx, l.spec("one"))
	id2, _ := d.ShowNotification(ctx, l.spec("two"))
	id3, _ := d.ShowNotification(ctx, l.spec("three"))

	// A queued notification leaves the queue.
	if err := d.CloseNotification(ctx, id2); err != nil {
		t.Fatal(err)
	}
	if l.closed["two"] != mobile.CloseByApp {
		t.Errorf("two closed as %v", l.closed["two"])
	}
	// The shown one is hidden and the next starts.
	n := len(sh.notify)
	if err := d.CloseNotification(ctx, id1); err != nil {
		t.Fatal(err)
	}
	hide := sh.notify[n]
	if hide.Command != win32.NIMModify || hide.Data.Flags != win32.NIFInfo || hide.Data.Info != "" {
		t.Errorf("the hide = %+v", hide)
	}
	if l.closed["one"] != mobile.CloseByApp || shown(sh).InfoTitle != "three" {
		t.Errorf("one closed as %v, shown %+v", l.closed["one"], shown(sh))
	}
	// The echo of our own hide ends nothing.
	d.onIconMessage(0, uintptr(win32.NINBalloonHide)|uintptr(balloonIconID)<<16)
	if _, ended := l.closed["three"]; ended {
		t.Error("the echo of our hide ended the next balloon")
	}
	// A notification that is gone is not an error.
	if err := d.CloseNotification(ctx, id1); err != nil {
		t.Errorf("closing a closed notification: %v", err)
	}
	if err := d.CloseNotification(ctx, id3); err != nil {
		t.Fatal(err)
	}
	if err := d.CloseNotification(ctx, 12345); err != nil {
		t.Errorf("closing an unknown ID: %v", err)
	}
}

func TestBalloonExpiry(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		after   time.Duration
		expired bool
	}{
		{"the app's time-out ends the balloon", 3 * time.Second, 3 * time.Second, true},
		{"not before the time-out", 3 * time.Second, 2900 * time.Millisecond, false},
		{"no time-out: the shell's own time is waited for", 0, 14 * time.Second, false},
		{"no time-out: a shell that stays silent is given up on", 0, 15 * time.Second, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, sh := newTestDesktop()
			l := newBalloonLog()
			if _, err := d.ShowNotification(t.Context(), l.spec("one", func(s *mobile.NotificationSpec) { s.Timeout = tt.timeout })); err != nil {
				t.Fatal(err)
			}
			if _, err := d.ShowNotification(t.Context(), l.spec("two")); err != nil {
				t.Fatal(err)
			}
			sh.advance(tt.after)
			got, ended := l.closed["one"]
			if ended != tt.expired || (ended && got != mobile.CloseExpired) {
				t.Errorf("closed = %v (%v), want expired %v", got, ended, tt.expired)
			}
			if tt.expired && shown(sh).InfoTitle != "two" {
				t.Errorf("the next balloon is %+v", shown(sh))
			}
		})
	}
}

func TestBalloonFieldsAndLimits(t *testing.T) {
	tests := []struct {
		name      string
		mod       func(*mobile.NotificationSpec)
		wantTitle string
		wantBody  string
		wantFlags uint32
	}{
		{"plain", func(*mobile.NotificationSpec) {}, "T", "B", win32.NIIFInfo},
		{"critical is the warning icon", func(s *mobile.NotificationSpec) { s.Urgency = mobile.UrgencyCritical }, "T", "B", win32.NIIFWarning},
		{"low is silent", func(s *mobile.NotificationSpec) { s.Urgency = mobile.UrgencyLow }, "T", "B", win32.NIIFInfo | win32.NIIFNoSound},
		{"a title of 63 stays", func(s *mobile.NotificationSpec) { s.Title = strings.Repeat("t", 63) }, strings.Repeat("t", 63), "B", win32.NIIFInfo},
		{"a title of 64 is cut with an ellipsis", func(s *mobile.NotificationSpec) { s.Title = strings.Repeat("t", 64) }, strings.Repeat("t", 62) + "…", "B", win32.NIIFInfo},
		{"a body of 255 stays", func(s *mobile.NotificationSpec) { s.Body = strings.Repeat("b", 255) }, "T", strings.Repeat("b", 255), win32.NIIFInfo},
		{"a body of 256 is cut with an ellipsis", func(s *mobile.NotificationSpec) { s.Body = strings.Repeat("b", 256) }, "T", strings.Repeat("b", 254) + "…", win32.NIIFInfo},
		{"an empty body would hide the balloon", func(s *mobile.NotificationSpec) { s.Body = "" }, "T", " ", win32.NIIFInfo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, sh := newTestDesktop()
			spec := mobile.NotificationSpec{Title: "T", Body: "B"}
			tt.mod(&spec)
			if _, err := d.ShowNotification(t.Context(), spec); err != nil {
				t.Fatal(err)
			}
			got := shown(sh)
			if got.InfoTitle != tt.wantTitle || got.Info != tt.wantBody || got.InfoFlags != tt.wantFlags {
				t.Errorf("balloon = title %q body %q flags %#x; want title %q body %q flags %#x",
					got.InfoTitle, got.Info, got.InfoFlags, tt.wantTitle, tt.wantBody, tt.wantFlags)
			}
		})
	}
}

func TestBalloonUsesTheTrayIconAsItsOwner(t *testing.T) {
	d, sh := newTestDesktop()
	if _, err := d.NewTray(platform.TraySpec{}); err != nil {
		t.Fatal(err)
	}
	n := len(sh.notify)
	if _, err := d.ShowNotification(t.Context(), mobile.NotificationSpec{Title: "T", Body: "B"}); err != nil {
		t.Fatal(err)
	}
	if got := sh.commands()[n:]; !reflect.DeepEqual(got, []string{"modify#1"}) {
		t.Errorf("a notification with a tray sent %v: no hidden icon is made", got)
	}
	d.onIconMessage(0, uintptr(win32.NINBalloonTimeout)|uintptr(trayIconID)<<16)
	if got := sh.commands()[n+1:]; len(got) != 0 {
		t.Errorf("the end of the balloon sent %v", got)
	}
}

func TestBalloonFailures(t *testing.T) {
	t.Run("an app with no notification area", func(t *testing.T) {
		d, sh := newTestDesktop()
		sh.notifyErr = func(c notifyCall) error { return errors.New(win32.TagWin32CallFailed, "no tray") }
		_, err := d.ShowNotification(t.Context(), mobile.NotificationSpec{Title: "T"})
		assertTagRoute(t, err, tagNotificationsUnavailable, "Desktop.ShowNotification")
		appErr, _ := errors.FromError(err)
		if !appErr.HasRoute("balloonQueue.show->balloonQueue.display->balloonQueue.ensureOwner." + tagNotificationsUnavailable) {
			t.Errorf("route %s", appErr.Route())
		}
	})
	t.Run("the icon of the owner cannot be made", func(t *testing.T) {
		d, sh := newTestDesktop()
		sh.iconErr = errors.New(tagWindowIconInvalid, "bad")
		_, err := d.ShowNotification(t.Context(), mobile.NotificationSpec{Title: "T"})
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tagNotificationsUnavailable || !appErr.HasRoute("balloonQueue.ensureOwner."+tagNotificationsUnavailable) {
			t.Errorf("got %v", err)
		}
	})
	t.Run("the shell refuses the balloon", func(t *testing.T) {
		d, sh := newTestDesktop()
		sh.notifyErr = func(c notifyCall) error {
			if c.Command == win32.NIMModify {
				return errors.New(win32.TagWin32CallFailed, "refused")
			}
			return nil
		}
		_, err := d.ShowNotification(t.Context(), mobile.NotificationSpec{Title: "T"})
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tagWindowsBalloonFailed || !appErr.HasRoute("balloonQueue.show->balloonQueue.display."+tagWindowsBalloonFailed) {
			t.Fatalf("got %v", err)
		}
		if appErr.Metadata["reason"] == nil {
			t.Errorf("metadata = %v", appErr.Metadata)
		}
		// The owner icon made for it was deleted again, and the failed notification is not queued.
		if last := sh.commands(); last[len(last)-1] != "delete#2" {
			t.Errorf("commands: %v", last)
		}
		if d.balloons.shown != nil || len(d.balloons.queue) != 0 {
			t.Error("the refused notification is kept")
		}
	})
	t.Run("a queued notification that the shell refuses later is closed as unknown", func(t *testing.T) {
		d, sh := newTestDesktop()
		l := newBalloonLog()
		mustShow(t, d, l.spec("one"))
		mustShow(t, d, l.spec("two"))
		mustShow(t, d, l.spec("three"))
		sh.notifyErr = func(c notifyCall) error {
			if c.Command == win32.NIMModify && c.Data.InfoTitle == "two" {
				return errors.New(win32.TagWin32CallFailed, "refused")
			}
			return nil
		}
		d.onIconMessage(0, uintptr(win32.NINBalloonTimeout)|uintptr(balloonIconID)<<16)
		if l.closed["two"] != mobile.CloseUnknown || shown(sh).InfoTitle != "three" {
			t.Errorf("two closed as %v, shown %+v", l.closed["two"], shown(sh))
		}
	})
}

func TestBalloonIconFromAFile(t *testing.T) {
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for i := 0; i < 8; i++ {
		img.Set(i, i, color.NRGBA{R: 255, A: 255})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	pngPath := filepath.Join(dir, "icon.png")
	if err := os.WriteFile(pngPath, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	textPath := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(textPath, []byte("not an image"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got, ok := readBalloonIcon(pngPath); !ok || got.Size != balloonIconSize || len(got.RGBA) != balloonIconSize*balloonIconSize*4 {
		t.Errorf("a PNG: ok %v, size %d, %d bytes", ok, got.Size, len(got.RGBA))
	}
	for _, name := range []string{"dialog-information", textPath, filepath.Join(dir, "missing.png"), ""} {
		if _, ok := readBalloonIcon(name); ok {
			t.Errorf("readBalloonIcon(%q) gave a picture", name)
		}
	}

	d, sh := newTestDesktop()
	l := newBalloonLog()
	mustShow(t, d, l.spec("with icon", func(s *mobile.NotificationSpec) { s.Icon = pngPath }))
	got := shown(sh)
	if got.BalloonIcon == 0 || got.InfoFlags&win32.NIIFUser == 0 || got.InfoFlags&win32.NIIFLargeIcon == 0 {
		t.Errorf("the balloon with an icon = %+v", got)
	}
	d.onIconMessage(0, uintptr(win32.NINBalloonTimeout)|uintptr(balloonIconID)<<16)
	if !slices.Contains(sh.freedIcons, got.BalloonIcon) {
		t.Errorf("the large icon %d was not freed: %v", got.BalloonIcon, sh.freedIcons)
	}
	// An icon that is a theme name gives a balloon with the icon of its urgency.
	mustShow(t, d, l.spec("theme", func(s *mobile.NotificationSpec) { s.Icon = "dialog-information" }))
	if got := shown(sh); got.BalloonIcon != 0 || got.InfoFlags != win32.NIIFInfo {
		t.Errorf("the balloon with a theme icon = %+v", got)
	}
}

func TestCloseDeletesTheIconsAndStopsTheTimers(t *testing.T) {
	d, sh := newTestDesktop()
	l := newBalloonLog()
	mustShow(t, d, l.spec("one"))
	mustShow(t, d, l.spec("two"))
	d.Close()
	if last := sh.commands(); last[len(last)-1] != "delete#2" {
		t.Errorf("commands: %v", last)
	}
	sh.advance(time.Hour)
	if len(l.closed) != 0 {
		t.Errorf("a timer ran after Close: %v", l.closed)
	}
}

func TestExplorerRestartAddsTheOwnerIconAgain(t *testing.T) {
	d, sh := newTestDesktop()
	mustShow(t, d, mobile.NotificationSpec{Title: "T", Body: "B"})
	n := len(sh.notify)
	d.onTaskbarCreated()
	if got := sh.commands()[n:]; !reflect.DeepEqual(got, []string{"add#2", "version#2"}) {
		t.Errorf("after TaskbarCreated: %v", got)
	}
	if add := sh.notify[n].Data; add.State != win32.NISHidden {
		t.Errorf("the owner icon came back visible: %+v", add)
	}
}
