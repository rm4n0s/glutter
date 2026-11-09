package macos

import (
	"bytes"
	"context"
	"image/png"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
	"github.com/rm4n0s/glutter/ui"
)

func TestPlanFilters(t *testing.T) {
	tests := []struct {
		name    string
		filters []platform.FileFilter
		want    filterPlan
		open    []string
		save    []string
	}{
		{"none", nil, filterPlan{}, nil, nil},
		{"images and text", []platform.FileFilter{
			{Name: "Images", Patterns: []string{"*.png", "*.JPG", "*.png"}},
			{Name: "Text", Patterns: []string{"*.txt", "readme.md"}}},
			filterPlan{Groups: []filterGroup{{"Images", []string{"png", "jpg"}}, {"Text", []string{"txt", "md"}}}},
			[]string{"png", "jpg", "txt", "md"}, []string{"png", "jpg"}},
		{"all files", []platform.FileFilter{{Name: "Everything", Patterns: []string{"*"}}, {Name: "PNG", Patterns: []string{"*.png"}}},
			filterPlan{AllFiles: true, Groups: []filterGroup{{"Everything", nil}, {"PNG", []string{"png"}}}}, nil, nil},
		{"star dot star", []platform.FileFilter{{Name: "Any", Patterns: []string{"*.*"}}},
			filterPlan{AllFiles: true, Groups: []filterGroup{{"Any", nil}}}, nil, nil},
		{"a name with no extension and a wildcard in it name no type", []platform.FileFilter{{Name: "Odd", Patterns: []string{"Makefile", "img.*x"}}},
			filterPlan{Groups: []filterGroup{{"Odd", nil}}}, nil, nil},
		{"the last extension of a double one", []platform.FileFilter{{Name: "Archives", Patterns: []string{"*.tar.gz"}}},
			filterPlan{Groups: []filterGroup{{"Archives", []string{"gz"}}}}, []string{"gz"}, []string{"gz"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := planFilters(tc.filters)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("plan = %+v, want %+v", got, tc.want)
			}
			if o := got.extensions(); !reflect.DeepEqual(o, tc.open) {
				t.Errorf("extensions = %v, want %v", o, tc.open)
			}
			if s := got.saveExtensions(); !reflect.DeepEqual(s, tc.save) {
				t.Errorf("saveExtensions = %v, want %v", s, tc.save)
			}
		})
	}
	plan := planFilters([]platform.FileFilter{{Name: "Images", Patterns: []string{"*.png"}}, {Name: "Text", Patterns: []string{"*.txt"}}})
	for file, want := range map[string]string{"/a/b.PNG": "Images", "c.txt": "Text", "d.bin": "", "noext": ""} {
		if got := plan.filterNameFor(file); got != want {
			t.Errorf("filterNameFor(%q) = %q, want %q", file, got, want)
		}
	}
}

func TestPlanTrayMenu(t *testing.T) {
	var pressed []string
	press := func(s string) func() { return func() { pressed = append(pressed, s) } }
	items := []platform.TrayMenuItem{
		{Separator: true}, // first: left out
		{Label: "Open", OnPressed: press("open")},
		{Label: "Hidden", Hidden: true, OnPressed: press("hidden")},
		{Separator: true},
		{Separator: true}, // next to another: left out
		{Label: "Sound", Checkable: true, Checked: true, OnPressed: press("sound")},
		{Label: "Off", Disabled: true, OnPressed: press("off")},
		{Label: "Mode", Children: []platform.TrayMenuItem{
			{Label: "A", Radio: true, Checked: true, OnPressed: press("a")},
			{Label: "B", Radio: true, OnPressed: press("b")},
		}, OnPressed: press("mode")},
		{Separator: true}, // last: left out
	}
	plan, handlers := planTrayMenu(items)
	want := []trayEntry{
		{ID: 1, Label: "Open"},
		{ID: 2, Separator: true},
		{ID: 3, Label: "Sound", Checked: true},
		{ID: 4, Label: "Off", Disabled: true},
		{ID: 5, Label: "Mode", Children: []trayEntry{{ID: 6, Label: "A", Checked: true}, {ID: 7, Label: "B"}}},
	}
	if !reflect.DeepEqual(plan, want) {
		t.Errorf("plan = %+v\nwant   %+v", plan, want)
	}
	for _, id := range []int{1, 3, 6, 7} {
		h, ok := handlers[id]
		if !ok {
			t.Errorf("no handler for %d", id)
			continue
		}
		h()
	}
	for _, id := range []int{2, 4, 5} {
		if _, ok := handlers[id]; ok {
			t.Errorf("an entry that is a separator, disabled or a submenu has a handler: %d", id)
		}
	}
	if !reflect.DeepEqual(pressed, []string{"open", "sound", "a", "b"}) {
		t.Errorf("pressed = %v", pressed)
	}
}

func TestTrayTip(t *testing.T) {
	tests := []struct {
		spec platform.TraySpec
		want string
	}{
		{platform.TraySpec{}, ""},
		{platform.TraySpec{Tooltip: "Title"}, "Title"},
		{platform.TraySpec{Tooltip: "Title", TooltipBody: "Body"}, "Title\nBody"},
		{platform.TraySpec{Title: "App"}, "App"},
		{platform.TraySpec{TooltipBody: "Only body"}, "Only body"},
	}
	for _, tc := range tests {
		if got := trayTip(tc.spec); got != tc.want {
			t.Errorf("trayTip(%+v) = %q, want %q", tc.spec, got, tc.want)
		}
	}
}

func TestCheckOpenURI(t *testing.T) {
	tests := []struct {
		uri string
		tag string
	}{
		{"https://example.com/a?b=c", ""},
		{"mailto:me@example.com", ""},
		{"file:///Users/a/picture.png", ""},
		{"file:///Users/a/My%20Folder/", ""},
		{"x-apple.systempreferences:com.apple.preference.security", ""},
		{"", tagOpenURIInvalid},
		{"no scheme", tagOpenURIInvalid},
		{"https://example.com/\nx", tagOpenURIInvalid},
		{"https://example.com/\x00", tagOpenURIInvalid},
		{"javascript:alert(1)", tagOpenURISchemeBlocked},
		{"vbscript:run", tagOpenURISchemeBlocked},
		{"file:///Applications/Safari.app", tagOpenURISchemeBlocked},
		{"file:///Applications/Safari.app/", tagOpenURISchemeBlocked},
		{"file:///Users/a/run.COMMAND", tagOpenURISchemeBlocked},
		{"file:///tmp/install.pkg", tagOpenURISchemeBlocked},
		{"file:///tmp/s%2Esh", tagOpenURISchemeBlocked},
	}
	for _, tc := range tests {
		err := checkOpenURI(tc.uri)
		if tc.tag == "" {
			if err != nil {
				t.Errorf("checkOpenURI(%q) = %v, want nil", tc.uri, err)
			}
			continue
		}
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tc.tag || !appErr.HasRoute("checkOpenURI."+tc.tag) {
			t.Errorf("checkOpenURI(%q) = %v, want %s (route %v)", tc.uri, err, tc.tag, err)
		}
	}
}

// fakeCenter is a notifyBackend that records what it is asked.
type fakeCenter struct {
	mu        sync.Mutex
	notReady  string
	grant     bool
	authErr   error
	authCalls int
	cats      [][]notifyCategory
	added     []notifyRequest
	removed   []string
	addErr    error
}

func (f *fakeCenter) Ready() (bool, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.notReady == "", f.notReady
}

func (f *fakeCenter) Authorize(context.Context) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.authCalls++
	return f.grant, f.authErr
}

func (f *fakeCenter) SetCategories(c []notifyCategory) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cats = append(f.cats, c)
	return nil
}

func (f *fakeCenter) Add(_ context.Context, r notifyRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.addErr != nil {
		return f.addErr
	}
	f.added = append(f.added, r)
	return nil
}

func (f *fakeCenter) Remove(id string) {
	f.mu.Lock()
	f.removed = append(f.removed, id)
	f.mu.Unlock()
}

// fakeClock is the timer of a notifier that the test fires.
type fakeClock struct {
	mu     sync.Mutex
	timers []*fakeTimer
}

type fakeTimer struct {
	d       time.Duration
	fn      func()
	stopped bool
}

func (c *fakeClock) after(d time.Duration, fn func()) func() bool {
	t := &fakeTimer{d: d, fn: fn}
	c.mu.Lock()
	c.timers = append(c.timers, t)
	c.mu.Unlock()
	return func() bool { c.mu.Lock(); defer c.mu.Unlock(); was := !t.stopped; t.stopped = true; return was }
}

func (c *fakeClock) fireAll() {
	c.mu.Lock()
	ts := append([]*fakeTimer(nil), c.timers...)
	c.mu.Unlock()
	for _, t := range ts {
		if !t.stopped {
			t.fn()
		}
	}
}

func newTestNotifier(b notifyBackend) (*notifier, *fakeClock) {
	n := newNotifier(b)
	c := &fakeClock{}
	n.after = c.after
	return n, c
}

func routeOf(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute(route) {
		t.Errorf("error %v, want the route %s", err, route)
	}
}

func TestNotificationNeedsABundleAndThePermission(t *testing.T) {
	ctx := context.Background()
	t.Run("no bundle", func(t *testing.T) {
		n, _ := newTestNotifier(&fakeCenter{notReady: "the app has no bundle identifier"})
		_, err := n.show(ctx, mobile.NotificationSpec{Title: "x"})
		routeOf(t, err, "notifier.show."+tagNotificationsUnavailable)
	})
	t.Run("refused, asked once", func(t *testing.T) {
		f := &fakeCenter{grant: false}
		n, _ := newTestNotifier(f)
		_, err := n.show(ctx, mobile.NotificationSpec{Title: "x"})
		routeOf(t, err, "notifier.authorize."+tagNotificationsUnavailable)
		_, err = n.show(ctx, mobile.NotificationSpec{Title: "y"})
		routeOf(t, err, "notifier.authorize."+tagNotificationsUnavailable)
		if f.authCalls != 1 {
			t.Errorf("the permission was asked %d times, want once", f.authCalls)
		}
	})
	t.Run("the permission cannot be asked", func(t *testing.T) {
		n, _ := newTestNotifier(&fakeCenter{authErr: errors.New(tagNotificationsUnavailable, "boom")})
		_, err := n.show(ctx, mobile.NotificationSpec{Title: "x"})
		routeOf(t, err, tagNotificationsUnavailable)
	})
	t.Run("the center refuses", func(t *testing.T) {
		n, _ := newTestNotifier(&fakeCenter{grant: true, addErr: errors.New(tagNotificationsUnavailable, "no")})
		_, err := n.show(ctx, mobile.NotificationSpec{Title: "x"})
		routeOf(t, err, tagNotificationsUnavailable)
	})
}

func TestNotificationLife(t *testing.T) {
	ctx := context.Background()
	f := &fakeCenter{grant: true}
	n, clock := newTestNotifier(f)
	var mu sync.Mutex
	var log []string
	rec := func(s string) { mu.Lock(); log = append(log, s); mu.Unlock() }
	spec := mobile.NotificationSpec{
		Title: "Hello", Body: "World", Urgency: mobile.UrgencyLow, Timeout: 5 * time.Second,
		Actions:  []mobile.Action{{ID: mobile.DefaultAction, Label: ""}, {ID: "reply", Label: "Reply"}},
		OnAction: func(id string) { rec("action:" + id) },
		OnClose:  func(r mobile.CloseReason) { rec("close:" + itoa(int(r))) },
	}
	id, err := n.show(ctx, spec)
	if err != nil || id != 1 {
		t.Fatalf("show = %d, %v", id, err)
	}
	if len(f.added) != 1 || f.added[0].Identifier != "glutter.1" || !f.added[0].Quiet || f.added[0].Title != "Hello" || f.added[0].Body != "World" {
		t.Errorf("request = %+v", f.added)
	}
	if len(f.cats) != 1 || len(f.cats[0]) != 1 || len(f.cats[0][0].Actions) != 1 || f.cats[0][0].Actions[0].ID != "reply" {
		t.Errorf("categories = %+v (the default action is no button)", f.cats)
	}
	// A second notification with the same buttons shares the category: the set is not set again.
	id2, _ := n.show(ctx, mobile.NotificationSpec{Title: "Two", Actions: spec.Actions})
	if id2 != 2 || len(f.cats) != 1 {
		t.Errorf("second notification: id %d, %d category sets", id2, len(f.cats))
	}
	// The click on the notification and on a button.
	n.response("glutter.2", mobile.DefaultAction) // not listed as an action of this notification? it is: "default"
	n.response("glutter.1", "reply")
	mu.Lock()
	got := append([]string(nil), log...)
	mu.Unlock()
	if !reflect.DeepEqual(got, []string{"action:reply", "close:2"}) {
		t.Errorf("log = %v (the second notification had no callbacks; the first: its action, then dismissed)", got)
	}
	if !contains(f.removed, "glutter.1") {
		t.Errorf("a button left the notification in the center: %v", f.removed)
	}
	// A response to a notification that is gone does nothing.
	n.response("glutter.1", "reply")
	n.response("other.identifier", "reply")
	mu.Lock()
	n2 := len(log)
	mu.Unlock()
	if n2 != 2 {
		t.Errorf("a response to a gone notification ran callbacks: %v", log)
	}
	// The timer of a notification that was answered does nothing.
	clock.fireAll()
	mu.Lock()
	n3 := len(log)
	mu.Unlock()
	if n3 != 2 {
		t.Errorf("the timer of an answered notification ran: %v", log)
	}
}

func TestNotificationTimeoutReplaceAndClose(t *testing.T) {
	ctx := context.Background()
	f := &fakeCenter{grant: true}
	n, clock := newTestNotifier(f)
	var mu sync.Mutex
	var log []string
	rec := func(s string) { mu.Lock(); log = append(log, s); mu.Unlock() }
	mk := func(name string, timeout time.Duration, replaces uint32) mobile.NotificationSpec {
		return mobile.NotificationSpec{Title: name, Timeout: timeout, ReplacesID: replaces,
			OnClose: func(r mobile.CloseReason) { rec(name + ":" + itoa(int(r))) }}
	}
	id, _ := n.show(ctx, mk("a", time.Second, 0))
	// Replacing keeps the number, does not close the old one, and the old timer does nothing.
	id2, _ := n.show(ctx, mk("b", 0, id))
	if id2 != id {
		t.Errorf("a replacement has the number %d, want %d", id2, id)
	}
	if f.added[1].Identifier != f.added[0].Identifier {
		t.Errorf("a replacement has another identifier: %v", f.added)
	}
	clock.fireAll()
	mu.Lock()
	if len(log) != 0 {
		t.Errorf("the replaced notification's timer closed it: %v", log)
	}
	mu.Unlock()
	// The timeout of a notification that stays.
	id3, _ := n.show(ctx, mk("c", time.Second, 0))
	clock.fireAll()
	mu.Lock()
	if !reflect.DeepEqual(log, []string{"c:1"}) {
		t.Errorf("log = %v, want c expired", log)
	}
	mu.Unlock()
	if !contains(f.removed, identifierOf(id3)) {
		t.Errorf("an expired notification stays in the center: %v", f.removed)
	}
	// The app closes b, once.
	n.close(id)
	n.close(id)
	mu.Lock()
	if !reflect.DeepEqual(log, []string{"c:1", "b:3"}) {
		t.Errorf("log = %v, want b closed by the app once", log)
	}
	mu.Unlock()
	n.shutdown()
}

func TestIdentifiers(t *testing.T) {
	if idOfIdentifier(identifierOf(42)) != 42 || idOfIdentifier("glutter.x") != 0 || idOfIdentifier("x.5") != 0 || idOfIdentifier("glutter.99999999999") != 0 {
		t.Error("identifiers do not round trip")
	}
	a := categoryOf([]mobile.Action{{ID: "a", Label: "A"}})
	b := categoryOf([]mobile.Action{{ID: "a", Label: "A"}})
	c := categoryOf([]mobile.Action{{ID: "a", Label: "Other"}})
	d := categoryOf(nil)
	if a.ID != b.ID || a.ID == c.ID || a.ID == d.ID {
		t.Errorf("category ids: %s %s %s %s", a.ID, b.ID, c.ID, d.ID)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func itoa(i int) string {
	return string(rune('0' + i))
}

func TestTrayIconPNG(t *testing.T) {
	px := func(size int, c byte) ui.WindowIconImage {
		b := make([]byte, size*size*4)
		for i := 0; i < len(b); i += 4 {
			b[i], b[i+1], b[i+2], b[i+3] = c, 0, 0, 255
		}
		return ui.WindowIconImage{Size: size, RGBA: b}
	}
	if data, _, err := trayIconPNG(ui.WindowIcon{}); data != nil || err != nil {
		t.Errorf("no picture: %v, %v", data, err)
	}
	icon := ui.WindowIcon{Images: []ui.WindowIconImage{px(16, 1), px(32, 2), px(64, 3)}}
	icon.Images[1].Scale = 2 // 16 points at twice the pixels: as near to 18 as the 16
	data, points, err := trayIconPNG(icon)
	if err != nil || len(data) == 0 {
		t.Fatalf("trayIconPNG = %d bytes, %v", len(data), err)
	}
	if points != 16 {
		t.Errorf("points = %v, want 16 (the 32-pixel picture at scale 2, which has the most pixels of those nearest)", points)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil || img.Bounds().Dx() != 32 {
		t.Errorf("the picture is %v, %v; want the 32-pixel one", img.Bounds(), err)
	}
	if r, _, _, _ := img.At(0, 0).RGBA(); r>>8 != 2 {
		t.Errorf("the picture is not the second one (red %d)", r>>8)
	}
}
