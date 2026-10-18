package wayland

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
)

// Global is one global object that the compositor announced.
//
// Flutter: no counterpart
type Global struct {
	// Name is the numeric name that identifies the global in the registry.
	Name uint32
	// Version is the highest version of the interface the compositor offers.
	Version uint32
}

// requiredGlobal is a global without which the backend cannot run (SPEC §8.2).
type requiredGlobal struct {
	iface *wl.Interface
	// max is the highest version this backend uses; it binds the lower of that
	// and the version the compositor offers.
	max uint32
}

// The required globals, with the versions the backend uses. wl_compositor v6
// brings wl_surface.preferred_buffer_scale, wl_seat v9 the last pointer axis
// events, wl_output v4 the output name, and xdg_wm_base v6 the suspended state
// and wm_capabilities (SPEC §8.2).
var required = []requiredGlobal{
	{wl.WlCompositorInterface, 6},
	{wl.WlSubcompositorInterface, 1},
	{wl.WlShmInterface, 1},
	{wl.WlSeatInterface, 9},
	{wl.WlOutputInterface, 4},
	{wl.XdgWmBaseInterface, 6},
}

// optional are the globals the backend uses when they are present (SPEC §8.2).
// They are recorded but not bound here: the step that uses one binds it.
var optional = []*wl.Interface{
	wl.ZxdgDecorationManagerV1Interface,
	wl.WpFractionalScaleManagerV1Interface,
	wl.WpViewporterInterface,
	wl.WpCursorShapeManagerV1Interface,
	wl.ZwpTextInputManagerV3Interface,
	wl.WlDataDeviceManagerInterface,
	wl.ZwpPrimarySelectionDeviceManagerV1Interface,
	wl.ZwpPointerGesturesV1Interface,
	wl.XdgActivationV1Interface,
	wl.WpPresentationInterface,
	wl.XdgToplevelIconManagerV1Interface,
	wl.ZxdgExporterV2Interface,
	wl.XdgWmDialogV1Interface,
	wl.ZxdgOutputManagerV1Interface,
	wl.ZwpTabletManagerV2Interface,
	wl.ZwpIdleInhibitManagerV1Interface,
	wl.WpContentTypeManagerV1Interface,
}

// session is a connection with the registry listed and the required globals
// bound.
type session struct {
	conn     *wl.Conn
	registry *wl.WlRegistry

	// advertised lists every global the compositor announced, by interface name.
	// A compositor may announce an interface more than once (wl_output).
	advertised map[string][]Global

	compositor    *wl.WlCompositor
	subcompositor *wl.WlSubcompositor
	shm           *wl.WlShm
	seat          *wl.WlSeat
	outputs       []*wl.WlOutput
	wmBase        *wl.XdgWmBase

	// decorationManager is nil when the compositor has no
	// zxdg_decoration_manager_v1.
	decorationManager *wl.ZxdgDecorationManagerV1

	// optionalPresent holds the optional globals that the compositor offers.
	optionalPresent map[string]Global
}

// newSession lists the registry of conn and binds the required globals. On
// failure it closes conn. A required global that is missing fails with
// WaylandProtocolMissing (metadata "interface"). Errors from the connection
// itself are returned unchanged.
func newSession(conn *wl.Conn) (*session, error) {
	s := &session{
		conn:            conn,
		advertised:      map[string][]Global{},
		optionalPresent: map[string]Global{},
	}
	s.registry = conn.Display().GetRegistry()
	s.registry.SetEvents(&wl.WlRegistryEvents{
		OnGlobal: func(name uint32, iface string, version uint32) {
			s.advertised[iface] = append(s.advertised[iface], Global{name, version})
		},
	})
	if err := conn.Roundtrip(); err != nil {
		conn.Close()
		return nil, err
	}
	if err := s.bindRequired(); err != nil {
		conn.Close()
		return nil, err
	}
	s.recordOptional()
	// The bound outputs send their description right after they are bound.
	if err := conn.Roundtrip(); err != nil {
		conn.Close()
		return nil, err
	}
	return s, nil
}

// bindVersion returns the version to bind for g: the version the compositor
// offers, capped at max.
func bindVersion(g Global, max uint32) uint32 { return min(g.Version, max) }

func (s *session) bindRequired() error {
	for _, r := range required {
		globals := s.advertised[r.iface.Name]
		if len(globals) == 0 {
			return errors.New("WaylandProtocolMissing", "the compositor does not offer a required Wayland protocol",
				"interface", r.iface.Name)
		}
		g, v := globals[0], bindVersion(globals[0], r.max)
		switch r.iface {
		case wl.WlCompositorInterface:
			s.compositor = wl.BindWlCompositor(s.registry, g.Name, v)
		case wl.WlSubcompositorInterface:
			s.subcompositor = wl.BindWlSubcompositor(s.registry, g.Name, v)
		case wl.WlShmInterface:
			s.shm = wl.BindWlShm(s.registry, g.Name, v)
		case wl.WlSeatInterface:
			s.seat = wl.BindWlSeat(s.registry, g.Name, v)
		case wl.WlOutputInterface:
			for _, og := range globals {
				s.outputs = append(s.outputs, wl.BindWlOutput(s.registry, og.Name, bindVersion(og, r.max)))
			}
		case wl.XdgWmBaseInterface:
			s.wmBase = wl.BindXdgWmBase(s.registry, g.Name, v)
			// The compositor pings to find out whether the app still responds;
			// without a pong it marks the window as not responding.
			wmBase := s.wmBase
			s.wmBase.SetEvents(&wl.XdgWmBaseEvents{OnPing: func(serial uint32) { wmBase.Pong(serial) }})
		}
	}
	return nil
}

func (s *session) recordOptional() {
	for _, iface := range optional {
		if globals := s.advertised[iface.Name]; len(globals) > 0 {
			s.optionalPresent[iface.Name] = globals[0]
		}
	}
	// The optional globals that the code uses already are bound; the others are
	// bound by the step that needs them.
	if g, ok := s.optionalPresent[wl.ZxdgDecorationManagerV1Interface.Name]; ok {
		s.decorationManager = wl.BindZxdgDecorationManagerV1(s.registry, g.Name, bindVersion(g, 1))
	}
}

// close destroys the connection and with it every object.
func (s *session) close() { s.conn.Close() }
