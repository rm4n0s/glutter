package ios

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// fakeSessionAPI is a sessionAPI that records its calls.
type fakeSessionAPI struct {
	categories []string
	options    []uint64
	actives    []bool
	failSet    string
	failActive string
}

func (f *fakeSessionAPI) SetCategory(c string, o uint64) (string, bool) {
	if f.failSet != "" {
		return f.failSet, false
	}
	f.categories, f.options = append(f.categories, c), append(f.options, o)
	return "", true
}

func (f *fakeSessionAPI) SetActive(a bool) (string, bool) {
	if f.failActive != "" {
		return f.failActive, false
	}
	f.actives = append(f.actives, a)
	return "", true
}

func TestCategoryOf(t *testing.T) {
	if c, o := categoryOf(false); c != categoryPlayback || o != 0 {
		t.Errorf("playback: %q, %d", c, o)
	}
	c, o := categoryOf(true)
	if c != categoryPlayAndRecord || o&optionDefaultToSpeaker == 0 || o&optionAllowBluetooth == 0 {
		t.Errorf("recording: %q, %d", c, o)
	}
	// AVAudioSessionCategoryOptions' numbers: mix 1, allow Bluetooth 4, default to speaker 8, A2DP 32.
	if optionMixWithOthers != 1 || optionAllowBluetooth != 4 || optionDefaultToSpeaker != 8 || optionAllowBluetoothA2DP != 32 {
		t.Error("the options are not AVAudioSession's numbers")
	}
}

func TestSessionPolicyActivates(t *testing.T) {
	api := &fakeSessionAPI{}
	p := &sessionPolicy{api: api}
	if err := p.Activate(platform.AudioPlayback); err != nil {
		t.Fatal(err)
	}
	// The category is set once; the session is made active each time.
	if err := p.Activate(platform.AudioPlayback); err != nil {
		t.Fatal(err)
	}
	if len(api.categories) != 1 || api.categories[0] != categoryPlayback || len(api.actives) != 2 {
		t.Errorf("categories %v, actives %v", api.categories, api.actives)
	}
	// A stream that records changes the category, and it stays.
	if err := p.Activate(platform.AudioCapture); err != nil {
		t.Fatal(err)
	}
	if err := p.Activate(platform.AudioPlayback); err != nil {
		t.Fatal(err)
	}
	if len(api.categories) != 2 || api.categories[1] != categoryPlayAndRecord || len(api.actives) != 4 {
		t.Errorf("after recording: categories %v, actives %v", api.categories, api.actives)
	}
}

func TestSessionPolicyFailures(t *testing.T) {
	t.Run("the category", func(t *testing.T) {
		p := &sessionPolicy{api: &fakeSessionAPI{failSet: "no"}}
		err := p.Activate(platform.AudioPlayback)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tagAudioSessionFailed || !appErr.HasRoute("sessionPolicy.Activate."+tagAudioSessionFailed) {
			t.Errorf("error %v", err)
		}
		if p.category != "" {
			t.Errorf("the category %q was remembered after it failed", p.category)
		}
	})
	t.Run("the activation", func(t *testing.T) {
		p := &sessionPolicy{api: &fakeSessionAPI{failActive: "busy"}}
		err := p.Activate(platform.AudioPlayback)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tagAudioSessionFailed || !appErr.HasRoute("sessionPolicy.Activate."+tagAudioSessionFailed) {
			t.Errorf("error %v", err)
		}
	})
}

func TestInterruptionOf(t *testing.T) {
	tests := []struct {
		typ, options  uint64
		began, resume bool
	}{
		{1, 0, true, false},
		{1, 1, true, false},
		{0, 1, false, true},
		{0, 0, false, false},
	}
	for _, tc := range tests {
		if began, resume := interruptionOf(tc.typ, tc.options); began != tc.began || resume != tc.resume {
			t.Errorf("interruptionOf(%d, %d) = %v, %v; want %v, %v", tc.typ, tc.options, began, resume, tc.began, tc.resume)
		}
	}
	if !routeLostOf(2) || routeLostOf(1) || routeLostOf(0) {
		t.Error("only the reason 2, old device unavailable, is a lost route")
	}
}
