//go:build windows && glutter_wine && glutter_audio

package wasapi_test

// The program that the test of the host starts under Wine (wine_host_test.go): it plays a tone through WASAPI and talks to the host
// through files in the directory of GLUTTER_WASAPI_SYNC: it makes "ready" when the sound plays, ends when "stop" appears, and writes
// the states that the device went through to "states". Without GLUTTER_WASAPI_CHILD it is skipped.

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/wasapi"
)

func TestWineChild(t *testing.T) {
	role := os.Getenv("GLUTTER_WASAPI_CHILD")
	if role == "" {
		t.Skip("this is the program of the test of the host")
	}
	dir := os.Getenv("GLUTTER_WASAPI_SYNC")
	if dir == "" {
		t.Fatal("no GLUTTER_WASAPI_SYNC")
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	s := wasapi.New(logger)
	defer s.Close()
	opts := platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 2}}
	if name := os.Getenv("GLUTTER_WASAPI_DEVICE"); name != "" {
		list, err := s.Devices(platform.AudioPlayback)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range list {
			if strings.Contains(d.Name, name) {
				opts.ID = d.ID
			}
		}
		if opts.ID == "" {
			t.Fatalf("no device named %q in %+v", name, list)
		}
	}
	h := newCounter(48000, 2)
	dev, err := s.Open(context.Background(), platform.AudioPlayback, opts, h)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := dev.Close(); err != nil {
			t.Error(err)
		}
	}()
	eventually(t, "the first fills", 10*time.Second, func() bool { f, _ := h.filled(); return f > 3 })
	if err := os.WriteFile(filepath.Join(dir, "ready"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(dir, "stop")); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	var names []string
	for _, st := range h.stateList() {
		switch st {
		case platform.AudioLost:
			names = append(names, "lost")
		case platform.AudioPaused:
			names = append(names, "paused")
		default:
			names = append(names, "other")
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "states"), []byte(strings.Join(names, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
}
