package gluttertest

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/ui"
)

func TestGPUDeviceOnlyWithTheOption(t *testing.T) {
	tests := []struct {
		name string
		opts Opts
		want bool
	}{
		{"by default the app has no GPU device", Opts{}, false},
		{"with the option", Opts{GPU: true}, true},
		{"with the option and a spell checker", Opts{GPU: true, SpellChecker: func(string, string) []ui.SpellCheckSpan { return nil }}, true},
		{"with a spell checker only", Opts{SpellChecker: func(string, string) []ui.SpellCheckSpan { return nil }}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := New(t, tt.opts)
			if got := ts.GPUDevice() != nil; got != tt.want {
				t.Fatalf("GPUDevice() != nil is %v, want %v", got, tt.want)
			}
			var host explicit.DeviceHost
			if got := ts.Dispatcher().HostAs(&host); got != tt.want {
				t.Fatalf("the dispatcher's host has a GPU device: %v, want %v", got, tt.want)
			}
			if !tt.want {
				return
			}
			dev, state := host.GPUDevice()
			if state != explicit.HostReady || dev != explicit.Device(ts.GPUDevice().Device()) {
				t.Errorf("GPUDevice() = %v, %v", dev, state)
			}
		})
	}
}

func TestGPURecorderEvents(t *testing.T) {
	ts := New(t, Opts{GPU: true})
	rec := ts.GPUDevice()
	var kinds []explicit.DeviceEventKind
	stop := rec.WatchGPUDevice(func(e explicit.DeviceEvent) { kinds = append(kinds, e.Kind) })
	first := rec.Device()
	rec.Lose()
	if rec.Device() == first {
		t.Error("Lose kept the device")
	}
	rec.EndApp()
	rec.EndApp() // twice does nothing more
	want := []explicit.DeviceEventKind{explicit.DeviceLost, explicit.DeviceOpened, explicit.DeviceClosed, explicit.HostEnded}
	if len(kinds) != len(want) {
		t.Fatalf("events = %v, want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Errorf("event %d = %v, want %v", i, kinds[i], want[i])
		}
	}
	stop()
}

func TestGPURecorderKeepsTheCallsAndTheLog(t *testing.T) {
	ts := New(t, Opts{GPU: true})
	rec := ts.GPUDevice()
	if _, err := rec.Device().NewBuffer(16, explicit.BufferVertex); err != nil {
		t.Fatal(err)
	}
	if ops := rec.Ops(); len(ops) != 1 || ops[0] != "NewBuffer" {
		t.Errorf("Ops = %v", ops)
	}
	if calls := rec.Calls(); len(calls) != 1 {
		t.Errorf("Calls = %v", calls)
	}
	rec.Reset()
	if len(rec.Ops()) != 0 {
		t.Errorf("Reset kept %v", rec.Ops())
	}
	rec.GPULogger().Info("hello", "n", 1)
	if !rec.LogContains("hello") {
		t.Errorf("Logs = %q", rec.Logs())
	}
	rec.ClearLogs()
	if rec.Logs() != "" {
		t.Errorf("ClearLogs kept %q", rec.Logs())
	}
}
