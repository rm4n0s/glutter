package wayland

import (
	"context"
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
)

// Platform is the Wayland backend of platform.Platform. It is complete so far
// as far as the connection goes: Run drives the event loop, and Post wakes it.
//
// TODO(step 1.5): NewWindow, TODO(step 14.3): Displays, TODO(step 10.4):
// Clipboard, TODO(step 14.5): Settings.
//
// Flutter: no counterpart
type Platform struct {
	loop *linux.Loop
	sess *session
	src  *connSource

	mu      sync.Mutex
	running bool
}

var _ platform.Platform = (*Platform)(nil)

// New connects to the compositor with the given socket name (empty means
// $WAYLAND_DISPLAY) and binds the required globals. It fails with
// LibraryLoadFailed when libwayland-client is missing, WaylandConnectFailed when
// there is no compositor, and WaylandProtocolMissing when the compositor lacks a
// required protocol. Close releases the connection.
//
// Flutter: no counterpart
func New(name string) (*Platform, error) {
	conn, err := wl.Connect(name)
	if err != nil {
		return nil, err
	}
	return newPlatform(conn)
}

// newPlatform builds the platform on a connection that is already open.
func newPlatform(conn *wl.Conn) (*Platform, error) {
	sess, err := newSession(conn)
	if err != nil {
		return nil, err
	}
	loop, err := linux.NewLoop()
	if err != nil {
		sess.close()
		return nil, err
	}
	return &Platform{loop: loop, sess: sess, src: &connSource{conn: conn}}, nil
}

// Close releases the event loop and the connection. Do not use the platform
// afterwards, and do not call it while Run is active.
//
// Flutter: no counterpart
func (p *Platform) Close() error {
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

// Run runs the event loop on the calling goroutine, which must be the UI thread.
// It returns nil when ctx is cancelled, and the error of the connection when it
// fails: WaylandProtocolError (metadata "interface", "id", "code") when the
// compositor reports a protocol error, WaylandConnectionLost when the socket
// fails. A second call while Run is active fails with PlatformAlreadyRunning.
//
// Flutter: no counterpart
func (p *Platform) Run(ctx context.Context, h platform.Handler) error {
	_ = h // TODO(step 1.5): deliver window events to the handler.
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

	if err := p.loop.AddSource(p.src); err != nil {
		return err
	}
	runErr := p.loop.Run(ctx)
	if err := p.loop.RemoveSource(p.src); err != nil && runErr == nil {
		return err
	}
	return runErr
}

// Post runs fn on the UI thread inside Run. It is safe from any goroutine and
// wakes the loop.
//
// Flutter: no counterpart
func (p *Platform) Post(fn func()) { p.loop.Post(fn) }

// NewWindow is not implemented yet.
//
// TODO(step 1.5): create the surface, xdg_surface and xdg_toplevel.
//
// Flutter: no counterpart
func (p *Platform) NewWindow(platform.WindowConfig) (platform.Window, error) {
	return nil, errors.New("NotImplemented", "Wayland windows are not implemented yet")
}

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
