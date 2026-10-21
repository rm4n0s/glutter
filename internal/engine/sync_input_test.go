package engine

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

var _ platform.SyncInputHandler = (*handler)(nil)

// TestSyncInputIsHandledAtOnceAfterWhatWaits: a key or a pointer batch that the platform needs an
// answer for (the web calls preventDefault with it) is handled when it arrives, after the input
// that waits for its frame, and the answer is there when the call returns.
func TestSyncInputIsHandledAtOnceAfterWhatWaits(t *testing.T) {
	var order []string
	d := ui.NewPlatformDispatcher(nil)
	d.OnKeyData = func(k ui.KeyData) bool {
		order = append(order, "key "+string(rune(k.Logical)))
		return k.Logical == 'b'
	}
	d.OnPointerDataPacket = func(p ui.PointerDataPacket) {
		order = append(order, "pointer")
		for _, data := range p.Data {
			data.Respond(true)
		}
	}
	h := &handler{disp: d, cfg: &Config{}}

	h.OnKey(ui.KeyData{Logical: 'a'}) // waits for its frame
	if len(order) != 0 {
		t.Fatalf("OnKey handled the key before a frame: %v", order)
	}
	if handled := h.OnKeySync(ui.KeyData{Logical: 'b'}); !handled {
		t.Error("OnKeySync = false, want what the dispatcher answered (true)")
	}
	if handled := h.OnKeySync(ui.KeyData{Logical: 'c'}); handled {
		t.Error("OnKeySync = true for a key the dispatcher did not handle")
	}
	answered := false
	h.OnPointerSync([]ui.PointerData{{Change: ui.PointerChangeHover, OnRespond: func(allow bool) { answered = allow }}})
	if !answered {
		t.Error("the answer of the framework is not there when OnPointerSync returns")
	}
	want := []string{"key a", "key b", "key c", "pointer"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v, want %v: what waited goes first", order, want)
	}
	if len(h.input.pending) != 0 {
		t.Errorf("%d inputs are still queued", len(h.input.pending))
	}
}

// TestSyncTextInputIsHandledAtOnceAfterWhatWaits: a text-input event that the platform needs the answer
// to (the web's text field must know the framework's new state before it reads the next event) goes to the
// dispatcher when it arrives, after the input that waits for its frame.
func TestSyncTextInputIsHandledAtOnceAfterWhatWaits(t *testing.T) {
	var order []string
	d := ui.NewPlatformDispatcher(nil)
	d.OnKeyData = func(k ui.KeyData) bool {
		order = append(order, "key "+string(rune(k.Logical)))
		return false
	}
	d.OnTextInput = func(e ui.TextInputEvent) { order = append(order, "text "+e.Commit) }
	h := &handler{disp: d, cfg: &Config{}}

	h.OnKey(ui.KeyData{Logical: 'a'}) // waits for its frame
	h.OnTextInputSync(ui.TextInputEvent{Commit: "b"})
	h.OnTextInput(ui.TextInputEvent{Commit: "c"}) // waits for its frame
	if want := []string{"key a", "text b"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("order = %v, want %v: what waited goes first, and what waits afterwards still waits", order, want)
	}
	h.OnTextInputSync(ui.TextInputEvent{Commit: "d"})
	if want := []string{"key a", "text b", "text c", "text d"}; !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
}

// TestBrowserContextMenuReachesTheWindowsThatHaveOne: the dispatcher's call goes to the windows that
// implement platform.ContextMenuController, and says whether any did.
func TestBrowserContextMenuReachesTheWindowsThatHaveOne(t *testing.T) {
	menu := &menuWindow{}
	// A window without a menu is not asked.
	h := &handler{windows: map[ui.ViewID]*window{0: {Window: menu}, 2: {Window: noMenuWindow{}}}}
	if !h.SetBrowserContextMenuEnabled(false) || !reflect.DeepEqual(menu.calls, []bool{false}) {
		t.Errorf("calls = %v", menu.calls)
	}
	delete(h.windows, 0)
	if h.SetBrowserContextMenuEnabled(true) {
		t.Error("it reported a change with no window that has a menu")
	}
}

type menuWindow struct {
	platform.Window
	calls []bool
}

func (m *menuWindow) SetBrowserContextMenuEnabled(enabled bool) { m.calls = append(m.calls, enabled) }

type noMenuWindow struct{ platform.Window }
