// Command android-services is a development example (step 25.12): a Material page with buttons that
// open a URI and post a notification with an action button, and log what the desktop services answer,
// so that the device tests can start the actions without taps, read the results and check the
// notification in dumpsys. It uses the public API and package desktop only and runs on every platform
// (off Android the services are those of the desktop, or of the web).
//
// GLUTTER_DEMO is a list of words separated by commas; each one runs once, 3 seconds after the start:
//
//	open           opens https://example.com
//	openbad        opens glutter-nothing://x, which no app handles
//	notify         shows a notification with the action "Reply" (ID reply)
//	notifyclose    shows a notification and closes it 2 seconds later
//	notifyreplace  shows a notification, then another one with ReplacesID
//	notifytimeout  shows a notification with a Timeout of 3 seconds
//
// The log lines (the default logger's, so "INFO " comes first):
//
//	open result uri=<uri> err=<tag or none>
//	notify result id=<n> err=<tag or none>
//	notification action id=<action ID>
//	notification closed reason=<n>
//	close result id=<n> err=<tag or none>
//
// Run it on a device or the emulator with:
//
//	go run ./cmd/glutter-android run ./examples/dev/android-services
package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/desktop"
	m "github.com/rm4n0s/glutter/material"
)

func main() {
	err := m.RunApp(newApp(), m.RunAppOpts{
		AppID:  "dev.glutter.androidservices",
		Window: m.WindowConfig{Title: "Android services", Size: m.Size{Width: 480, Height: 640}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func newApp() m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Android services",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       &page{StatefulWidget: m.NewStatefulWidgetBase()},
	})
}

type page struct{ m.StatefulWidget }

func (p *page) CreateState() m.State { return &pageState{} }

type pageState struct {
	m.StateOf[*page]
	// mu guards last, which the goroutines of the actions set and read.
	mu   sync.Mutex
	last desktop.NotificationID
}

// startDelay is how long the demo words wait, so that the first frame and the window are there.
const startDelay = 3 * time.Second

func (s *pageState) InitState() {
	demo := map[string]bool{}
	for _, word := range strings.Split(os.Getenv("GLUTTER_DEMO"), ",") {
		demo[strings.TrimSpace(word)] = true
	}
	run := func(word string, fn func(ctx context.Context)) {
		if !demo[word] {
			return
		}
		s.Go(func(ctx context.Context) {
			select {
			case <-time.After(startDelay):
			case <-ctx.Done():
				return
			}
			fn(ctx)
		})
	}
	bc := s.Context()
	run("open", func(ctx context.Context) { s.open(ctx, bc, "https://example.com") })
	run("openbad", func(ctx context.Context) { s.open(ctx, bc, "glutter-nothing://x") })
	run("notify", func(ctx context.Context) { s.notify(ctx, bc, desktop.NotificationOpts{}) })
	run("notifyclose", func(ctx context.Context) {
		s.notify(ctx, bc, desktop.NotificationOpts{})
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return
		}
		s.closeLast(ctx, bc)
	})
	run("notifyreplace", func(ctx context.Context) {
		s.notify(ctx, bc, desktop.NotificationOpts{Body: "first"})
		s.notify(ctx, bc, desktop.NotificationOpts{Body: "second", ReplacesID: s.lastID()})
	})
	run("notifytimeout", func(ctx context.Context) { s.notify(ctx, bc, desktop.NotificationOpts{Timeout: 3 * time.Second}) })
}

// tagOf is the tag of an error, or "none".
func tagOf(err error) string {
	if err == nil {
		return "none"
	}
	if appErr, ok := errors.FromError(err); ok {
		return appErr.Tag
	}
	return "unknown"
}

func (s *pageState) lastID() desktop.NotificationID {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last
}

// open opens the URI and logs the result. It runs off the UI thread.
func (s *pageState) open(ctx context.Context, bc m.BuildContext, uri string) {
	err := desktop.OpenURI(ctx, bc, uri)
	slog.Info("open result", "uri", uri, "err", tagOf(err))
}

// notify shows a notification with a "Reply" button and logs the result and what comes back of it. It
// runs off the UI thread.
func (s *pageState) notify(ctx context.Context, bc m.BuildContext, o desktop.NotificationOpts) {
	if o.Body == "" {
		o.Body = "Hello from Glutter"
	}
	o.Actions = []desktop.NotificationAction{{ID: "reply", Label: "Reply"}}
	o.OnAction = func(id string) { slog.Info("notification action", "id", id) }
	o.OnClosed = func(r desktop.NotificationCloseReason) { slog.Info("notification closed", "reason", int(r)) }
	id, err := desktop.ShowNotification(ctx, bc, "Glutter", o)
	slog.Info("notify result", "id", int(id), "err", tagOf(err))
	if err == nil {
		s.mu.Lock()
		s.last = id
		s.mu.Unlock()
	}
}

// closeLast closes the notification that was shown last and logs the result.
func (s *pageState) closeLast(ctx context.Context, bc m.BuildContext) {
	id := s.lastID()
	err := desktop.CloseNotification(ctx, bc, id)
	slog.Info("close result", "id", int(id), "err", tagOf(err))
}

func (s *pageState) Build(m.BuildContext) m.Widget {
	bc := s.Context()
	button := func(label string, fn func(ctx context.Context)) m.Widget {
		return m.NewPadding(m.NewEdgeInsetsAll(8), m.NewElevatedButton(func() { s.Go(fn) }, m.NewText(label)))
	}
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("Android services")}),
		Body: m.NewColumn([]m.Widget{
			button("Open link", func(ctx context.Context) { s.open(ctx, bc, "https://example.com") }),
			button("Open mail", func(ctx context.Context) { s.open(ctx, bc, "mailto:someone@example.com") }),
			button("Open bad", func(ctx context.Context) { s.open(ctx, bc, "glutter-nothing://x") }),
			button("Notify", func(ctx context.Context) { s.notify(ctx, bc, desktop.NotificationOpts{}) }),
			button("Close", func(ctx context.Context) { s.closeLast(ctx, bc) }),
		}),
	})
}
