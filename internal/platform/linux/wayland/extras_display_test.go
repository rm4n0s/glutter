//go:build linux && glutter_display

package wayland

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// The optional protocols of step 14.7 against a real compositor: each one that it
// offers must work without a protocol error, and each one that it lacks must fail
// the way SPEC §8.11 says. Which ones exist depends on the compositor, so the test
// logs them.
func TestDisplayOptionalProtocolsOfTheCompositor(t *testing.T) {
	p := connectToCompositor(t)
	has := func(iface string) bool { _, ok := p.Globals(iface); return ok }
	t.Logf("xdg_activation_v1 %v, zwp_idle_inhibit_manager_v1 %v, wp_content_type_manager_v1 %v, xdg_toplevel_icon_manager_v1 %v, zwp_tablet_manager_v2 %v",
		has("xdg_activation_v1"), has("zwp_idle_inhibit_manager_v1"), has("wp_content_type_manager_v1"),
		has("xdg_toplevel_icon_manager_v1"), has("zwp_tablet_manager_v2"))

	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter optional protocols"
	cfg.AppID = "dev.glutter.displaytest"
	cfg.Size = ui.Size{Width: 320, Height: 200}
	cfg.ContentType = ui.ContentTypeVideo
	px := make([]byte, 16*16*4)
	for i := 0; i < len(px); i += 4 {
		px[i], px[i+3] = 200, 255
	}
	cfg.Icon = ui.WindowIcon{Name: "applications-graphics", Images: []ui.WindowIconImage{{Size: 16, RGBA: px}}}
	if _, err := p.NewWindow(cfg); err != nil {
		t.Fatal(err)
	}
	h := &recHandler{allowClose: true}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, h) }()
	waitUntil(t, func() bool { return len(h.metricsSeen()) > 0 }, "the window was never configured")

	var keepErr, iconErr error
	tokens := make(chan string, 1)
	handles := make(chan string, 1)
	finished := make(chan struct{})
	p.Post(func() {
		w := p.windows[0]
		keepErr = w.KeepScreenOn(true)
		w.SetContentType(ui.ContentTypeGame)
		iconErr = w.SetIcon(ui.WindowIcon{Name: "text-editor", Images: []ui.WindowIconImage{{Size: 16, Scale: 1, RGBA: px}}})
		p.RequestActivationToken(w.id, func(token string) { tokens <- token })
		w.PortalHandle(func(handle string) { handles <- handle })
		close(finished)
	})
	<-finished
	if has("zwp_idle_inhibit_manager_v1") != (keepErr == nil) {
		t.Errorf("KeepScreenOn(true) = %v with the protocol offered = %v", keepErr, has("zwp_idle_inhibit_manager_v1"))
	}
	if iconErr != nil {
		t.Errorf("SetIcon: %v", iconErr)
	}
	select {
	case token := <-tokens:
		if has("xdg_activation_v1") && token == "" {
			t.Error("the compositor offers xdg_activation_v1 but gave no token")
		}
		t.Logf("activation token: %d bytes", len(token))
	case <-time.After(10 * time.Second):
		t.Fatal("the activation token request was never answered")
	}
	select {
	case handle := <-handles:
		if has("zxdg_exporter_v2") != strings.HasPrefix(handle, "wayland:") {
			t.Errorf("the portal handle is %q with zxdg_exporter_v2 offered = %v", handle, has("zxdg_exporter_v2"))
		}
		t.Logf("portal handle: %q", handle)
	case <-time.After(10 * time.Second):
		t.Fatal("the portal handle was never answered")
	}
	if t.Failed() {
		return
	}
	// A protocol error would end Run; give the compositor a moment to send one.
	p.Post(func() { _ = p.windows[0].KeepScreenOn(false) })
	select {
	case err := <-done:
		t.Fatalf("Run ended early: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
	p.Post(func() { p.windows[0].onClose() })
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after the window closed")
	}
}
