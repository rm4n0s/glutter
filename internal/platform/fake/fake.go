package fake

import (
	"context"
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// Platform is a scripted platform.Platform for tests. Run replays a queue of
// events and posted functions on the goroutine that called it (the fake UI
// thread), and everything the engine asks of the platform is recorded in a
// log. It needs no display.
//
// Flutter: no counterpart
type Platform struct {
	mu       sync.Mutex
	queue    []func(h platform.Handler)
	wake     chan struct{} // capacity 1: a pending wakeup for Run
	running  bool
	nextView ui.ViewID
	windows  []*Window
	log      []Entry
	displays []platform.Display
	settings platform.Settings

	clipboardText string
	hasClipboard  bool
	clipboardPNG  []byte
	clipboardURIs []string
	primary       ui.ClipboardContent
}

var _ platform.Platform = (*Platform)(nil)

// New returns a fake platform that has no windows and is not running.
//
// Flutter: no counterpart
func New() *Platform {
	return &Platform{wake: make(chan struct{}, 1)}
}

// Run delivers queued events and posted functions to h, in the order they were
// queued, until ctx is cancelled. It returns nil on cancellation. Events
// queued before Run starts are delivered once it does. A second call while Run
// is active fails with PlatformAlreadyRunning.
//
// Flutter: no counterpart
func (p *Platform) Run(ctx context.Context, h platform.Handler) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return errors.New("PlatformAlreadyRunning", "Run was called while the platform is already running")
	}
	p.running = true
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
	}()

	for {
		if ctx.Err() != nil {
			return nil
		}
		if item, ok := p.pop(); ok {
			item(h)
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		case <-p.wake:
		}
	}
}

// pop removes and returns the oldest queued item.
func (p *Platform) pop() (func(h platform.Handler), bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.queue) == 0 {
		return nil, false
	}
	item := p.queue[0]
	p.queue[0] = nil
	p.queue = p.queue[1:]
	return item, true
}

// enqueue appends an item and wakes Run.
func (p *Platform) enqueue(item func(h platform.Handler)) {
	p.mu.Lock()
	p.queue = append(p.queue, item)
	p.mu.Unlock()
	select {
	case p.wake <- struct{}{}:
	default: // a wakeup is already pending
	}
}

// Post queues fn to run inside Run. It is safe to call from any goroutine.
// Functions posted by one goroutine run in the order it posted them, and in
// order with events added by Inject. A function posted after Run has returned
// waits for the next Run.
//
// Flutter: no counterpart
func (p *Platform) Post(fn func()) {
	p.enqueue(func(platform.Handler) { fn() })
}

// Inject queues an input event for delivery to the handler inside Run. It is
// safe to call from any goroutine.
//
// Flutter: no counterpart
func (p *Platform) Inject(event Event) {
	p.enqueue(func(h platform.Handler) {
		p.record(Entry{Kind: EntryEvent, View: event.view(), Value: event})
		event.dispatch(p, h)
	})
}

// NewWindow creates a fake window and records its config. The first window
// gets the implicit view ID; later windows get the next IDs.
//
// Flutter: no counterpart
func (p *Platform) NewWindow(cfg platform.WindowConfig) (platform.Window, error) {
	p.mu.Lock()
	w := &Window{p: p, id: p.nextView, config: cfg, title: cfg.Title, size: cfg.Size, minSize: cfg.MinSize, maxSize: cfg.MaxSize, state: cfg.State}
	p.nextView++
	p.windows = append(p.windows, w)
	p.log = append(p.log, Entry{Kind: EntryNewWindow, View: w.id, Value: cfg})
	p.mu.Unlock()
	return w, nil
}

// Windows returns every window created so far, closed ones included, in
// creation order.
//
// Flutter: no counterpart
func (p *Platform) Windows() []*Window {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]*Window(nil), p.windows...)
}

// Displays returns the displays set by SetDisplays; none by default.
//
// Flutter: no counterpart
func (p *Platform) Displays() []platform.Display {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]platform.Display(nil), p.displays...)
}

// SetDisplays sets the displays that Displays reports.
//
// Flutter: no counterpart
func (p *Platform) SetDisplays(displays []platform.Display) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.displays = append([]platform.Display(nil), displays...)
}

// Clipboard returns the in-memory clipboard. Its results come back through Post,
// like the real ones.
//
// Flutter: no counterpart
func (p *Platform) Clipboard() platform.Clipboard {
	return &clipboard{p: p}
}

// ClipboardText is the text that the in-memory clipboard holds, and whether it
// holds any.
//
// Flutter: no counterpart
func (p *Platform) ClipboardText() (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.clipboardText, p.hasClipboard
}

// SetClipboardText sets the text of the in-memory clipboard, as another
// application would.
//
// Flutter: no counterpart
func (p *Platform) SetClipboardText(text string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clipboardText, p.hasClipboard = text, true
}

type clipboard struct{ p *Platform }

func (c *clipboard) SetText(_ context.Context, text string, done func(error)) {
	c.p.SetClipboardText(text)
	if done != nil {
		c.p.Post(func() { done(nil) })
	}
}

// SetClipboardContent sets the in-memory clipboard (kind system) or primary
// selection (kind primary), as another application would.
//
// Flutter: no counterpart
func (p *Platform) SetClipboardContent(kind ui.ClipboardKind, content ui.ClipboardContent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if kind == ui.ClipboardKindPrimary {
		p.primary = content
		return
	}
	p.clipboardText, p.hasClipboard, p.clipboardPNG, p.clipboardURIs = content.Text, content.HasText, content.PNG, content.URIs
}

// ClipboardContent is what the in-memory clipboard (kind system) or primary
// selection (kind primary) holds.
//
// Flutter: no counterpart
func (p *Platform) ClipboardContent(kind ui.ClipboardKind) ui.ClipboardContent {
	p.mu.Lock()
	defer p.mu.Unlock()
	if kind == ui.ClipboardKindPrimary {
		return p.primary
	}
	return ui.ClipboardContent{Text: p.clipboardText, HasText: p.hasClipboard, PNG: p.clipboardPNG, URIs: p.clipboardURIs}
}

func (c *clipboard) SetContent(_ context.Context, kind ui.ClipboardKind, content ui.ClipboardContent, done func(error)) {
	c.p.SetClipboardContent(kind, content)
	if done != nil {
		c.p.Post(func() { done(nil) })
	}
}

func (c *clipboard) GetContent(ctx context.Context, kind ui.ClipboardKind, format string, done func(ui.ClipboardContent, error)) {
	content := c.p.ClipboardContent(kind)
	var out ui.ClipboardContent
	switch format {
	case ui.ClipboardFormatText:
		out.Text, out.HasText = content.Text, content.HasText
	case ui.ClipboardFormatPNG:
		out.PNG = content.PNG
	case ui.ClipboardFormatURIList:
		out.URIs = content.URIs
	}
	c.p.Post(func() { done(out, ctx.Err()) })
}

func (c *clipboard) GetText(ctx context.Context, done func(string, bool, error)) {
	text, ok := c.p.ClipboardText()
	c.p.Post(func() { done(text, ok, ctx.Err()) })
}

// Settings returns the current settings: the zero value, or what the last
// SettingsEvent or SetSettings set.
//
// Flutter: no counterpart
func (p *Platform) Settings() platform.Settings {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.settings
}

// SetSettings sets the settings that Settings reports, without notifying the
// handler. Use a SettingsEvent to notify it.
//
// Flutter: no counterpart
func (p *Platform) SetSettings(settings platform.Settings) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.settings = settings
}

// Services returns nil: the fake has no optional capabilities.
//
// Flutter: no counterpart
func (p *Platform) Services() platform.Services {
	return nil
}

// record appends an entry to the log.
func (p *Platform) record(e Entry) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log = append(p.log, e)
}

// Log returns a copy of the log: window creation, calls on windows, and the
// events delivered to the handler, in the order they happened.
//
// Flutter: no counterpart
func (p *Platform) Log() []Entry {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]Entry(nil), p.log...)
}
