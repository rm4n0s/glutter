package desktop

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/widgets"
)

// connectTimeout is how long the connection to the session bus may take.
const connectTimeout = 10 * time.Second

// services is the state of the desktop services of one app, kept in its
// WidgetsBinding (widgets.WidgetsBinding.Extension): the connection to the session
// bus, made on first use, and the seams that tests replace.
type services struct {
	mu  sync.Mutex
	bus dbus.Bus

	// open connects to the session bus; run starts a program and waits for it;
	// flatpak says whether the app runs inside a Flatpak sandbox.
	open    func(ctx context.Context) (dbus.Bus, error)
	run     func(ctx context.Context, env []string, name string, args ...string) error
	flatpak func() bool
}

type servicesKey struct{}

// servicesOf returns the services of the binding.
func servicesOf(b *widgets.WidgetsBinding) *services {
	return b.Extension(servicesKey{}, func() any {
		return &services{open: dbus.Connect, run: runProgram, flatpak: inFlatpak}
	}).(*services)
}

// connect returns the connection to the session bus, opening it when this is the
// first call. A failure is DBusConnectFailed, and is tried again by the next call.
func (s *services) connect(ctx context.Context) (dbus.Bus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bus != nil {
		return s.bus, nil
	}
	// The connection outlives the call that opened it: only the dial is bounded.
	dialCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), connectTimeout)
	defer cancel()
	b, err := s.open(dialCtx)
	if err != nil {
		return nil, err
	}
	s.bus = b
	return b, nil
}

// runProgram starts the program with the environment added to the app's and waits
// for it.
func runProgram(ctx context.Context, env []string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), env...)
	if err := cmd.Run(); err != nil {
		return errors.NewErr("ProgramFailed", err).SetMetadata("program", name)
	}
	return nil
}

// inFlatpak says whether the app runs inside a Flatpak sandbox: /.flatpak-info
// exists there.
func inFlatpak() bool {
	_, err := os.Stat("/.flatpak-info")
	return err == nil
}

// call is what a desktop call learns on the UI thread before it goes to the bus:
// the services of the app, the handle of the window as the parent of a dialog, and
// an activation token for the apps it starts.
type call struct {
	s      *services
	handle string
	token  string
}

// begin gathers what a call needs. It must not be called on the UI thread, which
// would wait for itself: a call is made from a goroutine of the app (SPEC §8.9,
// §13). It fails with DesktopCallOnUIThread then, and with DesktopWithoutBinding
// when bc does not belong to a WidgetsBinding. The window is the View above bc, when
// there is one; without it the call has no parent window.
func begin(ctx context.Context, bc widgets.BuildContext) (*call, error) {
	b := widgets.MaybeWidgetsBindingOf(bc)
	if b == nil {
		return nil, errors.New("DesktopWithoutBinding", "the context does not belong to a WidgetsBinding")
	}
	if b.UIThread().IsCurrent() {
		return nil, errors.New("DesktopCallOnUIThread",
			"a call of package desktop waits for the desktop, so it must not be made on the UI thread: call it from a goroutine")
	}
	c := &call{s: servicesOf(b)}
	done := make(chan struct{})
	b.Post(func() {
		v := widgets.MaybeViewOf(bc)
		if v == nil {
			close(done)
			return
		}
		d, id := v.PlatformDispatcher(), v.ViewID()
		// The platform answers on the UI thread, in turn: the handle first, then the
		// token, which ends the wait.
		d.PortalParentHandle(id, func(handle string) {
			c.handle = handle
			d.ActivationToken(id, func(token string) {
				c.token = token
				close(done)
			})
		})
	})
	select {
	case <-done:
	case <-ctx.Done():
		return nil, errors.NewErr("DesktopCallCancelled", ctx.Err())
	}
	return c, nil
}

// portalOptions are the options of a portal call: the handle token and, when there is
// one, the activation token.
func (c *call) portalOptions(token string, extra map[string]godbus.Variant) map[string]godbus.Variant {
	opts := map[string]godbus.Variant{"handle_token": godbus.MakeVariant(token)}
	if c.token != "" {
		opts["activation_token"] = godbus.MakeVariant(c.token)
	}
	for k, v := range extra {
		opts[k] = v
	}
	return opts
}
