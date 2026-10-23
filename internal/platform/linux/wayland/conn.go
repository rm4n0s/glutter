//go:build linux

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
	outputs       []*output
	// xdgOutputs is nil without zxdg_output_manager_v1.
	xdgOutputs *wl.ZxdgOutputManagerV1
	// onOutputsChanged, when set, is called when the outputs or what they say change.
	onOutputsChanged func()
	// outputMax is the version of wl_output that the backend binds, at most.
	outputMax uint32
	wmBase    *wl.XdgWmBase

	// seatCaps is what the seat can do now (the last wl_seat.capabilities).
	// onSeatCaps, when set, is called when it changes.
	seatCaps    wl.WlSeatCapability
	seatVersion uint32
	onSeatCaps  func(wl.WlSeatCapability)

	// decorationManager is nil when the compositor has no
	// zxdg_decoration_manager_v1.
	decorationManager *wl.ZxdgDecorationManagerV1
	// dialogManager is nil without xdg_wm_dialog_v1; a modal dialog is then only a
	// transient window of its parent.
	dialogManager *wl.XdgWmDialogV1
	// gestures is nil without zwp_pointer_gestures_v1 (touchpad pinch and swipe).
	gestures *wl.ZwpPointerGesturesV1
	// cursorShape is nil without wp_cursor_shape_manager_v1; the cursors then come
	// from the XCursor theme.
	cursorShape *wl.WpCursorShapeManagerV1
	// fractionalScale and viewporter are nil without wp_fractional_scale_manager_v1
	// and wp_viewporter; the windows then use the integer scale (SPEC §8.11).
	fractionalScale *wl.WpFractionalScaleManagerV1
	viewporter      *wl.WpViewporter
	// activation is nil without xdg_activation_v1: the focus requests are then
	// ignored.
	activation *wl.XdgActivationV1
	// idleInhibit is nil without zwp_idle_inhibit_manager_v1; KeepScreenOn then
	// fails with IdleInhibitUnavailable.
	idleInhibit *wl.ZwpIdleInhibitManagerV1
	// contentType is nil without wp_content_type_manager_v1.
	contentType *wl.WpContentTypeManagerV1
	// exporter is nil without zxdg_exporter_v2: the portals then get no parent window
	// handle for a dialog (SPEC §8.9).
	exporter *wl.ZxdgExporterV2
	// iconManager is nil without xdg_toplevel_icon_manager_v1; the window icon then
	// comes only from the .desktop file.
	iconManager *wl.XdgToplevelIconManagerV1
	// iconSizes are the icon sizes the compositor prefers (icon_size), in pixels.
	iconSizes []int32
	// tabletManager is nil without zwp_tablet_manager_v2: no stylus input.
	tabletManager *wl.ZwpTabletManagerV2
	// outputScale is the wl_output.scale of each output, 1 until it says;
	// onOutputScale, when set, is called when one changes.
	outputScale   map[*wl.WlOutput]int32
	onOutputScale func()

	// live is set when the session is set up: from then on the registry's news of
	// outputs are acted on.
	live bool

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
		outputScale:     map[*wl.WlOutput]int32{},
	}
	s.registry = conn.Display().GetRegistry()
	s.registry.SetEvents(&wl.WlRegistryEvents{
		OnGlobal: func(name uint32, iface string, version uint32) {
			s.advertised[iface] = append(s.advertised[iface], Global{name, version})
			if s.live && iface == wl.WlOutputInterface.Name {
				// A monitor that was plugged in.
				s.newOutput(Global{name, version}, s.outputMax)
			}
		},
		OnGlobalRemove: func(name uint32) {
			if s.live {
				s.removeOutput(name)
			}
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
	s.live = true
	return s, nil
}

// bindVersion returns the version to bind for g: the version the compositor
// offers, capped at max.
func bindVersion(g Global, max uint32) uint32 { return min(g.Version, max) }

func (s *session) bindRequired() error {
	for _, r := range required {
		globals := s.advertised[r.iface.Name]
		if len(globals) == 0 {
			return errors.New(tagWaylandProtocolMissing, "the compositor does not offer a required Wayland protocol",
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
			s.seatVersion = v
			// The capabilities arrive in the roundtrip after the binding, so
			// the handler is set now.
			s.seat.SetEvents(&wl.WlSeatEvents{OnCapabilities: func(caps wl.WlSeatCapability) {
				s.seatCaps = caps
				if s.onSeatCaps != nil {
					s.onSeatCaps(caps)
				}
			}})
		case wl.WlOutputInterface:
			s.outputMax = r.max
			for _, og := range globals {
				s.newOutput(og, r.max)
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
	if g, ok := s.optionalPresent[wl.ZxdgOutputManagerV1Interface.Name]; ok {
		s.xdgOutputs = wl.BindZxdgOutputManagerV1(s.registry, g.Name, bindVersion(g, 3))
		for _, o := range s.outputs {
			s.attachXdgOutput(o)
		}
	}
	if g, ok := s.optionalPresent[wl.XdgWmDialogV1Interface.Name]; ok {
		s.dialogManager = wl.BindXdgWmDialogV1(s.registry, g.Name, bindVersion(g, 1))
	}
	if g, ok := s.optionalPresent[wl.ZwpPointerGesturesV1Interface.Name]; ok {
		s.gestures = wl.BindZwpPointerGesturesV1(s.registry, g.Name, bindVersion(g, 1))
	}
	if g, ok := s.optionalPresent[wl.WpCursorShapeManagerV1Interface.Name]; ok {
		s.cursorShape = wl.BindWpCursorShapeManagerV1(s.registry, g.Name, bindVersion(g, 1))
	}
	if g, ok := s.optionalPresent[wl.XdgActivationV1Interface.Name]; ok {
		s.activation = wl.BindXdgActivationV1(s.registry, g.Name, bindVersion(g, 1))
	}
	if g, ok := s.optionalPresent[wl.ZwpIdleInhibitManagerV1Interface.Name]; ok {
		s.idleInhibit = wl.BindZwpIdleInhibitManagerV1(s.registry, g.Name, bindVersion(g, 1))
	}
	if g, ok := s.optionalPresent[wl.WpContentTypeManagerV1Interface.Name]; ok {
		s.contentType = wl.BindWpContentTypeManagerV1(s.registry, g.Name, bindVersion(g, 1))
	}
	if g, ok := s.optionalPresent[wl.ZxdgExporterV2Interface.Name]; ok {
		s.exporter = wl.BindZxdgExporterV2(s.registry, g.Name, bindVersion(g, 1))
	}
	if g, ok := s.optionalPresent[wl.XdgToplevelIconManagerV1Interface.Name]; ok {
		s.iconManager = wl.BindXdgToplevelIconManagerV1(s.registry, g.Name, bindVersion(g, 1))
		s.iconManager.SetEvents(&wl.XdgToplevelIconManagerV1Events{
			OnIconSize: func(size int32) { s.iconSizes = append(s.iconSizes, size) },
		})
	}
	if g, ok := s.optionalPresent[wl.ZwpTabletManagerV2Interface.Name]; ok {
		s.tabletManager = wl.BindZwpTabletManagerV2(s.registry, g.Name, bindVersion(g, 2))
	}
	// Fractional scale works through a viewport, so one without the other is no use.
	fs, hasFS := s.optionalPresent[wl.WpFractionalScaleManagerV1Interface.Name]
	vp, hasVP := s.optionalPresent[wl.WpViewporterInterface.Name]
	if hasVP {
		s.viewporter = wl.BindWpViewporter(s.registry, vp.Name, bindVersion(vp, 1))
	}
	if hasFS && hasVP {
		s.fractionalScale = wl.BindWpFractionalScaleManagerV1(s.registry, fs.Name, bindVersion(fs, 1))
	}
}

// close destroys the connection and with it every object.
func (s *session) close() { s.conn.Close() }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWaylandProtocolMissing = "WaylandProtocolMissing"
)
