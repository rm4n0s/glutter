package ui

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func tagOf(t *testing.T, err error) string {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	return appErr.Tag
}

func TestWindowIconValidate(t *testing.T) {
	tests := []struct {
		name string
		icon WindowIcon
		want string // the tag, empty for valid
	}{
		{"empty", WindowIcon{}, ""},
		{"a name", WindowIcon{Name: "text-editor"}, ""},
		{"a picture", WindowIcon{Images: []WindowIconImage{{Size: 2, RGBA: make([]byte, 16)}}}, ""},
		{"a picture with a scale", WindowIcon{Images: []WindowIconImage{{Size: 2, Scale: 2, RGBA: make([]byte, 16)}}}, ""},
		{"too few bytes", WindowIcon{Images: []WindowIconImage{{Size: 2, RGBA: make([]byte, 15)}}}, "WindowIconInvalid"},
		{"too many bytes", WindowIcon{Images: []WindowIconImage{{Size: 2, RGBA: make([]byte, 17)}}}, "WindowIconInvalid"},
		{"no size", WindowIcon{Images: []WindowIconImage{{}}}, "WindowIconInvalid"},
		{"negative scale", WindowIcon{Images: []WindowIconImage{{Size: 1, Scale: -1, RGBA: make([]byte, 4)}}}, "WindowIconInvalid"},
		{"the second picture is wrong", WindowIcon{Images: []WindowIconImage{{Size: 1, RGBA: make([]byte, 4)}, {Size: 3}}}, "WindowIconInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.icon.Validate()
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("no error")
			}
			if got := tagOf(t, err); got != tt.want {
				t.Errorf("tag = %s, want %s", got, tt.want)
			}
		})
	}
	if !(WindowIcon{}).IsZero() || (WindowIcon{Name: "x"}).IsZero() || (WindowIcon{Images: []WindowIconImage{{}}}).IsZero() {
		t.Error("IsZero is wrong")
	}
}

type appearanceHost struct {
	fakeHost
	icons    map[ViewID]WindowIcon
	contents map[ViewID]ContentType
	screen   map[ViewID]bool
	err      error
}

func (h *appearanceHost) SetWindowIcon(v ViewID, i WindowIcon) error {
	if h.icons == nil {
		h.icons = map[ViewID]WindowIcon{}
	}
	h.icons[v] = i
	return h.err
}

func (h *appearanceHost) SetWindowContentType(v ViewID, c ContentType) {
	if h.contents == nil {
		h.contents = map[ViewID]ContentType{}
	}
	h.contents[v] = c
}

func (h *appearanceHost) KeepScreenOn(v ViewID, on bool) error {
	if h.screen == nil {
		h.screen = map[ViewID]bool{}
	}
	h.screen[v] = on
	return h.err
}

func TestWindowAppearanceThroughTheDispatcher(t *testing.T) {
	host := &appearanceHost{}
	d := NewPlatformDispatcher(host)
	v := d.AddView(3, ViewConfiguration{})
	if err := v.SetIcon(WindowIcon{Name: "a"}); err != nil {
		t.Fatal(err)
	}
	v.SetContentType(ContentTypeVideo)
	if err := v.KeepScreenOn(true); err != nil {
		t.Fatal(err)
	}
	if host.icons[3].Name != "a" || host.contents[3] != ContentTypeVideo || !host.screen[3] {
		t.Errorf("host got icons %v, contents %v, screen %v", host.icons, host.contents, host.screen)
	}
	// A picture that is wrong never reaches the host.
	err := v.SetIcon(WindowIcon{Images: []WindowIconImage{{Size: 2}}})
	if got := tagOf(t, err); got != "WindowIconInvalid" {
		t.Errorf("tag = %s", got)
	}
	if host.icons[3].Name != "a" {
		t.Errorf("an invalid icon reached the host: %+v", host.icons[3])
	}
	// The host's own error comes back.
	host.err = errors.New("IconRefused", "no")
	if got := tagOf(t, v.SetIcon(WindowIcon{Name: "b"})); got != "IconRefused" {
		t.Errorf("tag = %s", got)
	}
}

func TestWindowAppearanceWithAHostThatCannot(t *testing.T) {
	d := NewPlatformDispatcher(&fakeHost{})
	v := d.AddView(1, ViewConfiguration{})
	if err := v.SetIcon(WindowIcon{Name: "a"}); err != nil {
		t.Errorf("SetIcon: %v", err)
	}
	v.SetContentType(ContentTypeGame)
	if got := tagOf(t, v.KeepScreenOn(true)); got != "IdleInhibitUnavailable" {
		t.Errorf("tag = %s, want IdleInhibitUnavailable", got)
	}
}

type portalHost struct {
	fakeHost
}

func (portalHost) PortalParentHandle(v ViewID, done func(string)) {
	done("host:" + string(rune('0'+v)))
}
func (portalHost) ActivationToken(_ ViewID, done func(string)) { done("tok") }
func (portalHost) AppID() string                               { return "org.example.App" }

func TestPortalThroughTheDispatcher(t *testing.T) {
	var handle, token string
	d := NewPlatformDispatcher(&portalHost{})
	d.PortalParentHandle(3, func(h string) { handle = h })
	d.ActivationToken(3, func(tk string) { token = tk })
	if handle != "host:3" || token != "tok" || d.AppID() != "org.example.App" {
		t.Errorf("handle %q, token %q", handle, token)
	}
	// A host without portals gives nothing, at once.
	handle, token = "x", "y"
	d = NewPlatformDispatcher(&fakeHost{})
	d.PortalParentHandle(3, func(h string) { handle = h })
	d.ActivationToken(3, func(tk string) { token = tk })
	if handle != "" || token != "" || d.AppID() != "" {
		t.Errorf("handle %q, token %q, app id %q, want none", handle, token, d.AppID())
	}
}
