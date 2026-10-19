package widgets_test

import (
	"context"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
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

// fakeTextClient is a text field that records the values the input method asks for.
type fakeTextClient struct {
	services.TextInputClientDefaults
	values []services.TextEditingValue
}

func (c *fakeTextClient) CurrentTextEditingValue() (services.TextEditingValue, bool) {
	return services.TextEditingValueEmpty, true
}
func (c *fakeTextClient) CurrentAutofillScope() services.AutofillScope { return nil }
func (c *fakeTextClient) UpdateEditingValue(v services.TextEditingValue) {
	c.values = append(c.values, v)
}
func (c *fakeTextClient) PerformAction(services.TextInputAction)               {}
func (c *fakeTextClient) PerformPrivateCommand(string, map[string]any)         {}
func (c *fakeTextClient) UpdateFloatingCursor(services.RawFloatingCursorPoint) {}
func (c *fakeTextClient) ShowAutocorrectionPromptRect(int, int)                {}
func (c *fakeTextClient) ConnectionClosed()                                    {}

func TestTextInputThroughTheBinding(t *testing.T) {
	ts := gluttertest.New(t)
	ti := ts.Binding.TextInput()
	client := &fakeTextClient{}
	conn := ti.Attach(client, services.NewTextInputConfiguration())
	conn.SetEditingState(services.NewTextEditingValue(services.TextEditingValueOpts{
		Text: "abc", Selection: foundation.Some(services.NewTextSelectionCollapsed(3)),
	}))
	conn.Show()
	states := ts.TextInputStates()
	if len(states) == 0 || !states[len(states)-1].Enabled || states[len(states)-1].Text != "abc" {
		t.Fatalf("states = %+v, want the input method on with abc", states)
	}

	ts.SendTextInputEvent(ui.TextInputEvent{Commit: "d", PreeditCursorBegin: -1, PreeditCursorEnd: -1})
	if len(client.values) != 1 || client.values[0].Text != "abcd" || client.values[0].Selection.BaseOffset != 4 {
		t.Errorf("client values = %v", client.values)
	}

	conn.Close()
	ts.FlushPosted()
	states = ts.TextInputStates()
	if states[len(states)-1].Enabled {
		t.Error("the input method is on after the client closed")
	}
}
