//go:build linux && !android && glutter_display

package wayland

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// swayView is what sway's tree says of the window with an app id: whether it has the focus and whether
// it is urgent (it asked for the focus and sway marked it instead).
func swayView(t *testing.T, appID string) (found, focused, urgent bool) {
	t.Helper()
	out, err := exec.Command("swaymsg", "-t", "get_tree", "-r").Output()
	if err != nil {
		t.Fatalf("swaymsg -t get_tree: %v", err)
	}
	type node struct {
		AppID         string `json:"app_id"`
		Focused       bool   `json:"focused"`
		Urgent        bool   `json:"urgent"`
		Nodes         []node `json:"nodes"`
		FloatingNodes []node `json:"floating_nodes"`
	}
	var root node
	if err := json.Unmarshal(out, &root); err != nil {
		t.Fatalf("sway's tree: %v", err)
	}
	var walk func(n node) bool
	walk = func(n node) bool {
		if n.AppID == appID {
			found, focused, urgent = true, n.Focused, n.Urgent
			return true
		}
		for _, kids := range [][]node{n.Nodes, n.FloatingNodes} {
			for _, k := range kids {
				if walk(k) {
					return true
				}
			}
		}
		return false
	}
	walk(root)
	return found, focused, urgent
}

// TestDisplayActivatesWithATokenOfAnotherProcess activates a window on sway with a token that another
// client of the compositor got for a window of its own: the second instance of an app that hands its
// work to the running one (Theatro's issue of 1.17.9). sway takes the token for the window, and a token
// it never gave does nothing. The headless sway has no input device, so no token carries a user's input
// and sway marks the window urgent rather than giving it the focus, whatever its
// focus_on_window_activation; that the window uses the token it was given and asks for none of its own
// is the fake compositor's test (TestActivationWithATokenOfAnotherProcess).
func TestDisplayActivatesWithATokenOfAnotherProcess(t *testing.T) {
	if _, err := exec.LookPath("swaymsg"); err != nil || os.Getenv("SWAYSOCK") == "" {
		t.Skip("this test reads the windows of sway (run under scripts/headless-wayland.sh)")
	}
	const appID, otherID = "dev.glutter.tokentest.app", "dev.glutter.tokentest.other"
	start := func(id string) (*Platform, *recHandler) {
		p := connectToCompositor(t)
		cfg := platform.DefaultWindowConfig()
		cfg.Title, cfg.AppID, cfg.Size = id, id, ui.Size{Width: 300, Height: 200}
		if _, err := p.NewWindow(cfg); err != nil {
			t.Fatal(err)
		}
		h := &recHandler{allowClose: true}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		done := make(chan error, 1)
		go func() { done <- p.Run(ctx, h) }()
		t.Cleanup(func() {
			cancel()
			if err := <-done; err != nil {
				t.Errorf("Run returned %v", err)
			}
		})
		return p, h
	}
	onLoop := func(p *Platform, fn func()) {
		finished := make(chan struct{})
		p.Post(func() { fn(); close(finished) })
		select {
		case <-finished:
		case <-time.After(10 * time.Second):
			t.Fatal("a posted function did not run")
		}
	}
	app, appH := start(appID)       // the app that runs
	other, otherH := start(otherID) // the second instance, another client
	heard := func(h *recHandler) bool { return len(h.focusSeen()) > 0 }
	waitUntil(t, func() bool { return heard(appH) && heard(otherH) }, "a window never heard of its focus")
	if out, err := exec.Command("swaymsg", "[app_id="+otherID+"]", "focus").CombinedOutput(); err != nil {
		t.Fatalf("swaymsg: %v: %s", err, out)
	}
	waitUntil(t, func() bool { _, focused, _ := swayView(t, otherID); return focused }, "the other window never got the focus")

	activate := func(token string) {
		onLoop(app, func() {
			app.windows[0].PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionActivate, Token: token})
		})
	}
	// A token that sway never gave is refused.
	activate("not-a-token-of-sway")
	time.Sleep(300 * time.Millisecond)
	if found, focused, urgent := swayView(t, appID); !found || focused || urgent {
		t.Fatalf("after a token sway never gave: found %v, focused %v, urgent %v; want neither", found, focused, urgent)
	}

	tokens := make(chan string, 1)
	onLoop(other, func() { other.RequestActivationToken(other.windows[0].id, func(tok string) { tokens <- tok }) })
	var token string
	select {
	case token = <-tokens:
	case <-time.After(10 * time.Second):
		t.Fatal("sway never gave the other client a token")
	}
	if token == "" {
		t.Fatal("sway has no xdg_activation_v1")
	}
	activate(token)
	waitUntil(t, func() bool { _, focused, urgent := swayView(t, appID); return focused || urgent },
		"sway did not take the token of the other client for the window")
	_, focused, urgent := swayView(t, appID)
	t.Logf("with the other client's token the window is focused %v, urgent %v (focus events %s)",
		focused, urgent, strings.Join(appH.focusSeen(), ", "))
}
