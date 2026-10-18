package wayland

import (
	"context"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
	"github.com/rm4n0s/glutter/ui"
)

// Options configures a Platform.
//
// Flutter: no counterpart
type Options struct {
	// ShmFill makes every window draw a solid color itself, through wl_shm, and
	// never call Handler.OnFrame. It is a development and test aid for
	// machines and tests without EGL (GLUTTER_DEV_SHM=1 in the dev examples).
	ShmFill bool
}

// Platform is the Wayland backend of platform.Platform. So far it has the
// connection, the event loop and shared-memory windows.
//
// TODO(step 14.3): Displays, TODO(step 10.4): Clipboard, TODO(step 14.5):
// Settings.
//
// Flutter: no counterpart
type Platform struct {
	loop *linux.Loop
	sess *session
	src  *connSource

	mu      sync.Mutex
	running bool

	opts  Options
	start time.Time // the origin of the frame times given to Handler.OnFrame

	// The fields below belong to the UI thread.
	handler   platform.Handler
	windows   map[ui.ViewID]*window
	nextView  ui.ViewID
	hadWindow bool
	// failure is the first error that a callback could not return; Run returns
	// it.
	failure error
}

var _ platform.Platform = (*Platform)(nil)

// New connects to the compositor with the given socket name (empty means
// $WAYLAND_DISPLAY) and binds the required globals. It fails with
// LibraryLoadFailed when libwayland-client is missing, WaylandConnectFailed when
// there is no compositor, and WaylandProtocolMissing when the compositor lacks a
// required protocol. Close releases the connection.
//
// Flutter: no counterpart
func New(name string, opts ...Options) (*Platform, error) {
	if len(opts) > 1 {
		panic(errors.New("MultipleOptions", "New takes at most one Options value", "count", len(opts)))
	}
	conn, err := wl.Connect(name)
	if err != nil {
		return nil, err
	}
	var o Options
	if len(opts) == 1 {
		o = opts[0]
	}
	return newPlatform(conn, o)
}

// now returns the time for Handler.OnFrame: monotonic, from the start of the
// platform.
//
// TODO(step 3.7): use the timestamp of the frame callback when there is one.
func (p *Platform) now() time.Duration { return time.Since(p.start) }

// newPlatform builds the platform on a connection that is already open.
func newPlatform(conn *wl.Conn, opts Options) (*Platform, error) {
	sess, err := newSession(conn)
	if err != nil {
		return nil, err
	}
	loop, err := linux.NewLoop()
	if err != nil {
		sess.close()
		return nil, err
	}
	return &Platform{
		opts:    opts,
		start:   time.Now(),
		loop:    loop,
		sess:    sess,
		src:     &connSource{conn: conn},
		windows: map[ui.ViewID]*window{},
	}, nil
}

// Close closes the windows that are still open, then releases the event loop and
// the connection. Do not use the platform afterwards, and do not call it while
// Run is active.
//
// Flutter: no counterpart
func (p *Platform) Close() error {
	// Windows that are still open go first, as their native windows need the
	// connection (and the renderer must have destroyed its surfaces already).
	for _, w := range p.windows {
		w.Close()
	}
	p.sess.close()
	return p.loop.Close()
}

// Globals returns every global the compositor announced for an interface, such
// as "wl_output" or "wp_viewporter", and whether there was any.
//
// Flutter: no counterpart
func (p *Platform) Globals(iface string) ([]Global, bool) {
	g := p.sess.advertised[iface]
	return append([]Global(nil), g...), len(g) > 0
}

// Run runs the event loop on the calling goroutine, which must be the UI thread,
// and delivers events to h. It returns nil when ctx is cancelled or when the
// last window has been closed, and the error of the connection when it fails: WaylandProtocolError (metadata "interface", "id", "code") when the
// compositor reports a protocol error, WaylandConnectionLost when the socket
// fails. A second call while Run is active fails with PlatformAlreadyRunning.
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

	p.handler = h
	defer func() { p.handler = nil }()
	if err := p.loop.AddSource(p.src); err != nil {
		return err
	}
	runErr := p.loop.Run(ctx)
	if runErr == nil {
		runErr = p.failure
		p.failure = nil
	}
	// Send what is still buffered, such as the requests that destroyed the
	// windows; a full socket is not worth waiting for here.
	if _, err := p.sess.conn.Flush(); err != nil && runErr == nil {
		runErr = err
	}
	if err := p.loop.RemoveSource(p.src); err != nil && runErr == nil {
		runErr = err
	}
	return runErr
}

// fail records an error that happened inside a callback, where it cannot be
// returned, and stops the loop so that Run returns it.
func (p *Platform) fail(err error) {
	if p.failure == nil {
		p.failure = err
	}
	p.loop.Stop()
}

// Post runs fn on the UI thread inside Run. It is safe from any goroutine and
// wakes the loop.
//
// Flutter: no counterpart
func (p *Platform) Post(fn func()) { p.loop.Post(fn) }

// Displays returns nil for now.
//
// TODO(step 14.3): the outputs with their scale and size.
//
// Flutter: no counterpart
func (p *Platform) Displays() []platform.Display { return nil }

// Clipboard returns nil for now.
//
// Flutter: no counterpart
func (p *Platform) Clipboard() platform.Clipboard { return nil }

// Settings returns the zero Settings for now.
//
// Flutter: no counterpart
func (p *Platform) Settings() platform.Settings { return platform.Settings{} }

// Services returns nil: the Wayland backend adds no desktop services itself.
//
// Flutter: no counterpart
func (p *Platform) Services() platform.Services { return nil }
