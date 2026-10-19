package widgets_test

import (
	"context"
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
)

func TestSystemNavigatorPopEndsTheApp(t *testing.T) {
	ts := gluttertest.New(t)
	if ts.ExitRequests() != 0 {
		t.Fatal("the app asked to exit before it was told to")
	}
	ts.Binding.SystemNavigator().Pop()
	if ts.ExitRequests() != 1 {
		t.Errorf("exit requests = %d, want 1", ts.ExitRequests())
	}
	// An unhandled pop route pops the system navigator.
	ts.Binding.HandlePopRoute()
	if ts.ExitRequests() != 2 {
		t.Errorf("exit requests = %d, want 2", ts.ExitRequests())
	}
}

func TestSystemSoundDoesNothingByDefault(t *testing.T) {
	ts := gluttertest.New(t)
	ts.Binding.SystemSound().Play(services.SystemSoundTypeAlert)
	if err := ts.TakeError(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClipboardThroughTheBinding(t *testing.T) {
	ts := gluttertest.New(t)
	clip := ts.Binding.Clipboard()
	ctx := context.Background()

	var setDone bool
	clip.SetData(ctx, services.NewClipboardData("copied"), func(err error) {
		if err != nil {
			t.Errorf("SetData: %v", err)
		}
		setDone = true
	})
	ts.FlushPosted()
	if text, ok := ts.ClipboardText(); !setDone || !ok || text != "copied" {
		t.Errorf("clipboard = %q (has %v), set done %v", text, ok, setDone)
	}

	ts.SetClipboardText("from another app")
	var got *services.ClipboardData
	clip.GetData(ctx, services.ClipboardKTextPlain, func(d *services.ClipboardData, err error) {
		if err != nil {
			t.Errorf("GetData: %v", err)
		}
		got = d
	})
	if got != nil {
		t.Fatal("the result must arrive after the call returns")
	}
	ts.FlushPosted()
	if got == nil || got.Text != "from another app" {
		t.Errorf("GetData = %+v", got)
	}

	ts.ClearClipboard()
	has := true
	clip.HasStrings(ctx, func(h bool, err error) { has = h })
	ts.FlushPosted()
	if has {
		t.Error("HasStrings is true for an empty clipboard")
	}
}
