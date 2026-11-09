package engine

import (
	"bytes"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform/fake"
)

// scriptConn is a connection that runs a script of events (platform.ScriptWaiter).
type scriptConn struct {
	*conn
	pending atomic.Bool
}

func (c *scriptConn) ScriptPending() bool { return c.pending.Load() }

// scriptHarness is a screenshot run whose platform has a script that ends when the test says so.
func scriptHarness(t *testing.T) (*harness, *scriptConn, string) {
	t.Helper()
	h, path := idleHarness(t)
	sc := &scriptConn{conn: h.c}
	sc.pending.Store(true)
	h.cfg.connectors[""] = func() (connection, error) { return sc, nil }
	return h, sc, path
}

func TestScreenshotWaitsForTheScriptOfEventsToEnd(t *testing.T) {
	h, sc, path := scriptHarness(t)
	h.c.Inject(metrics(1, 1))
	h.c.Inject(fake.FrameEvent{View: 0})
	done := h.run(t)
	time.Sleep(100 * time.Millisecond)
	if hasEntry(h.c, fake.EntryRequestFrame) {
		t.Fatal("the frame to capture was requested while the script of events was running")
	}
	sc.pending.Store(false)
	waitFor(t, func() bool { return hasEntry(h.c, fake.EntryRequestFrame) })
	h.c.Inject(fake.FrameEvent{View: 0})
	if err := result(t, done); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("no screenshot: %v", err)
	}
}

func TestScreenshotCapturesAnywayWhenTheScriptNeverEnds(t *testing.T) {
	h, _, path := scriptHarness(t)
	h.cfg.idleMax = 60 * time.Millisecond
	var logBuf bytes.Buffer
	h.cfg.Logger = slog.New(slog.NewTextHandler(&logBuf, nil))
	h.c.Inject(metrics(1, 1))
	h.c.Inject(fake.FrameEvent{View: 0})
	start := time.Now()
	done := h.run(t)
	waitFor(t, func() bool { return hasEntry(h.c, fake.EntryRequestFrame) })
	if elapsed := time.Since(start); elapsed < 60*time.Millisecond {
		t.Errorf("captured after %v, before the 60ms limit", elapsed)
	}
	h.c.Inject(fake.FrameEvent{View: 0})
	if err := result(t, done); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("no screenshot: %v", err)
	}
	if !bytes.Contains(logBuf.Bytes(), []byte("script of events had not ended")) {
		t.Errorf("log = %q, want a warning that the script had not ended", logBuf.String())
	}
}
