package webtest

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

func TestAwaitScreenshot(t *testing.T) {
	closed := func() chan struct{} { c := make(chan struct{}); close(c); return c }
	tests := []struct {
		name        string
		shots       [][]byte
		results     []Result
		browserDone chan struct{}
		browserErr  error
		timeout     time.Duration
		wantPNG     bool
		wantTag     string
		wantMeta    string
	}{
		{"the PNG comes", [][]byte{tinyPNG}, nil, nil, nil, time.Second, true, "", ""},
		{"the PNG beats the exit that follows it", [][]byte{tinyPNG}, []Result{{Kind: "exit"}}, nil, nil, time.Second, true, "", ""},
		{"the PNG beats a browser that has ended", [][]byte{tinyPNG}, nil, closed(), nil, time.Second, true, "", ""},
		{"the app fails", nil, []Result{{Kind: "exit", ExitCode: 1, Log: "boom: HostElementNotFound\n"}}, nil, nil, time.Second, false, "WebAppFailed", "HostElementNotFound"},
		{"the app could not start", nil, []Result{{Kind: "exit", ExitCode: -1, Message: "CompileError"}}, nil, nil, time.Second, false, "WebAppFailed", "CompileError"},
		{"the app ends without a screenshot", nil, []Result{{Kind: "exit"}}, nil, nil, time.Second, false, "WebAppFailed", ""},
		{"other results are skipped", nil, []Result{{Kind: "note"}, {Kind: "exit", ExitCode: 3}}, nil, nil, time.Second, false, "WebAppFailed", ""},
		{"the browser ends", nil, nil, closed(), os.ErrClosed, time.Second, false, "BrowserExited", "file already closed"},
		{"nothing comes", nil, nil, nil, nil, 30 * time.Millisecond, false, "BrowserTimeout", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestServer(t)
			for _, s := range tc.shots {
				srv.shots <- s
			}
			for _, r := range tc.results {
				srv.results <- r
			}
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()
			png, err := awaitScreenshot(ctx, srv, tc.browserDone, func() error { return tc.browserErr })
			if tc.wantPNG {
				if err != nil || string(png) != string(tinyPNG) {
					t.Fatalf("awaitScreenshot = %q, %v; want the PNG", png, err)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tc.wantTag || !appErr.HasRoute("awaitScreenshot."+tc.wantTag) {
				t.Fatalf("awaitScreenshot = %v, want %s via awaitScreenshot", err, tc.wantTag)
			}
			if tc.wantMeta != "" {
				var all string
				for _, v := range appErr.Metadata {
					if s, ok := v.(string); ok {
						all += s + "\n"
					}
				}
				if !strings.Contains(all, tc.wantMeta) {
					t.Errorf("the error's details %q lack %q", all, tc.wantMeta)
				}
			}
		})
	}
}
