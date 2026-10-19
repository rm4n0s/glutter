package x11

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/gles"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/ui"
)

// defaultRefreshRate is the frame rate used when XRandR is missing or reports
// none.
const defaultRefreshRate = 60

// atoms are the atoms the backend uses, interned once at startup.
type atoms struct {
	wmProtocols, wmDeleteWindow, netWMPing                        uint64
	netWMName, utf8String, netWMPID                               uint64
	netWMState, netWMStateMaxVert, netWMStateMaxHorz              uint64
	netWMStateFullscreen, netWMStateModal                         uint64
	netWMWindowType, netWMWindowTypeNormal, netWMWindowTypeDialog uint64
	motifWMHints                                                  uint64
}

// Platform is the X11 backend of platform.Platform: Xlib and XInput2 through
// purego, with EGL for drawing (SPEC §8.3). It belongs to the UI thread.
//
// TODO(step 14.3): Displays, TODO(step 14.5):
// Settings.
//
// Flutter: no counterpart
type Platform struct {
	x       *xlib
	display uintptr
	root    uint64
	screen  int32
	atoms   atoms

	loop *linux.Loop
	src  *connSource

	mu      sync.Mutex
	running bool

	start    time.Time     // the origin of the frame times given to Handler.OnFrame
	interval time.Duration // the time between frames: the monitor's refresh period

	// event is C memory for one XEvent, which XNextEvent fills.
	event uintptr

	// The fields below belong to the UI thread.
	handler   platform.Handler
	windows   map[uint64]*window // by X window ID
	nextView  ui.ViewID
	hadWindow bool
	failure   error
	// xErrors counts the errors the server reported for our requests.
	xErrors int

	input   *inputState // nil when XInput2 is not available
	cursors *cursors
	clip    *clipboard // nil when its window could not be created

	// visualFor returns the X visual that EGL wants for windows on the display;
	// tests replace it.
	visualFor func(display uintptr) (uint32, error)
}

var _ platform.Platform = (*Platform)(nil)

// New connects to the X server named by name (empty means $DISPLAY) and sets up
// the keyboard and pointer input. It fails with X11LoadFailed when libX11 is
// missing and X11OpenDisplayFailed when the display cannot be opened. Without libXi or
// without XInput2 on the server there is no keyboard and pointer input (a warning
// is logged). Close releases the connection.
//
// Flutter: no counterpart
func New(name string) (*Platform, error) {
	x, err := loadXlibOnce()
	if err != nil {
		return nil, err
	}
	var cname uintptr
	if name != "" {
		cname = dl.CString(name)
		defer dl.Free(cname)
	}
	display := x.openDisplay(cname)
	if display == 0 {
		return nil, errors.New("X11OpenDisplayFailed", "could not open the X display", "display", name)
	}
	loop, err := linux.NewLoop()
	if err != nil {
		x.closeDisplay(display)
		return nil, err
	}
	p := &Platform{
		x:       x,
		display: display,
		root:    x.defaultRootWindow(display),
		screen:  x.defaultScreen(display),
		loop:    loop,
		start:   time.Now(),
		event:   dl.Calloc(1, xEventSize),
		windows: map[uint64]*window{},

		visualFor: gles.X11Visual,
	}
	p.src = &connSource{p: p}
	p.atoms = p.internAtoms()
	p.interval = p.refreshInterval()
	p.cursors = newCursors(p)
	registerErrorHandler(p)
	p.input = newInputState(p)
	p.clip = newClipboard(p)
	return p, nil
}

// internAtoms interns the atoms the backend uses.
func (p *Platform) internAtoms() atoms {
	intern := func(name string) uint64 {
		c := dl.CString(name)
		defer dl.Free(c)
		return p.x.internAtom(p.display, c, 0)
	}
	return atoms{
		wmProtocols: intern("WM_PROTOCOLS"), wmDeleteWindow: intern("WM_DELETE_WINDOW"), netWMPing: intern("_NET_WM_PING"),
		netWMName: intern("_NET_WM_NAME"), utf8String: intern("UTF8_STRING"), netWMPID: intern("_NET_WM_PID"),
		netWMState: intern("_NET_WM_STATE"), netWMStateMaxVert: intern("_NET_WM_STATE_MAXIMIZED_VERT"),
		netWMStateMaxHorz: intern("_NET_WM_STATE_MAXIMIZED_HORZ"), netWMStateFullscreen: intern("_NET_WM_STATE_FULLSCREEN"),
		netWMStateModal: intern("_NET_WM_STATE_MODAL"), netWMWindowType: intern("_NET_WM_WINDOW_TYPE"),
		netWMWindowTypeNormal: intern("_NET_WM_WINDOW_TYPE_NORMAL"), netWMWindowTypeDialog: intern("_NET_WM_WINDOW_TYPE_DIALOG"),
		motifWMHints: intern("_MOTIF_WM_HINTS"),
	}
}

// refreshInterval returns the time between frames: the refresh period that
// XRandR reports for the screen, or that of 60 Hz when libXrandr is missing or
// says nothing useful.
func (p *Platform) refreshInterval() time.Duration {
	return refreshIntervalOf(p.xrandrRate())
}

// refreshIntervalOf converts a refresh rate in Hz to a frame interval, with the
// default for a rate that is not positive or not plausible.
func refreshIntervalOf(rate int) time.Duration {
	if rate < 1 || rate > 1000 {
		rate = defaultRefreshRate
	}
	return time.Second / time.Duration(rate)
}

// xrandrRate asks XRandR for the current refresh rate of the screen, or 0.
func (p *Platform) xrandrRate() int {
	r, err := loadXrandrOnce()
	if err != nil {
		slog.Debug("XRandR is not available: using 60 Hz", "error", err)
		return 0
	}
	config := r.getScreenInfo(p.display, p.root)
	if config == 0 {
		return 0
	}
	defer r.freeScreenInfo(config)
	return int(r.currentRate(config))
}

// now returns the time for Handler.OnFrame and the key and pointer events:
// monotonic, from the start of the platform.
func (p *Platform) now() time.Duration { return time.Since(p.start) }

// Close closes the windows that are still open and the connection. Do not use the
// platform afterwards, and do not call it while Run is active.
//
// Flutter: no counterpart
func (p *Platform) Close() error {
	for _, w := range p.windows {
		w.Close()
	}
	if p.input != nil {
		p.input.close()
		p.input = nil
	}
	if p.clip != nil {
		p.clip.close()
		p.clip = nil
	}
	p.cursors.close()
	unregisterErrorHandler(p)
	p.x.flush(p.display)
	p.x.closeDisplay(p.display)
	p.display = 0
	dl.Free(p.event)
	p.event = 0
	return p.loop.Close()
}

// Run runs the event loop on the calling goroutine, which must be the UI thread,
// and delivers events to h. It returns nil when ctx is cancelled or when the last
// window has been closed. A second call while Run is active fails with
// PlatformAlreadyRunning.
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
	p.x.flush(p.display)
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
// TODO(step 14.3): the monitors from XRandR.
//
// Flutter: no counterpart
func (p *Platform) Displays() []platform.Display { return nil }

// Clipboard returns the CLIPBOARD selection.
//
// Flutter: no counterpart
func (p *Platform) Clipboard() platform.Clipboard {
	if p.clip == nil {
		return nil
	}
	return p.clip
}

// Settings returns the zero Settings for now.
//
// Flutter: no counterpart
func (p *Platform) Settings() platform.Settings { return platform.Settings{} }

// Services returns nil: the X11 backend adds no desktop services itself.
//
// Flutter: no counterpart
func (p *Platform) Services() platform.Services { return nil }

// connSource plugs the X connection into the event loop: before each wait it
// handles what Xlib has queued and flushes the output, after it it reads and
// handles what arrived.
type connSource struct{ p *Platform }

var _ linux.Source = (*connSource)(nil)

func (c *connSource) FD() int          { return int(c.p.x.connectionNumber(c.p.display)) }
func (c *connSource) WantsWrite() bool { return false }

func (c *connSource) BeforeWait() error {
	// Events may already sit in Xlib's queue without the socket being readable,
	// and a handler may have made requests that wait in its buffer.
	c.p.dispatchEvents()
	c.p.x.flush(c.p.display)
	return nil
}

func (c *connSource) AfterWait(linux.Events) error {
	c.p.dispatchEvents()
	return nil
}

// dispatchEvents handles every event that is available now.
func (p *Platform) dispatchEvents() {
	for p.display != 0 && p.x.pending(p.display) > 0 {
		p.x.nextEvent(p.display, p.event)
		p.handleEvent()
	}
}
