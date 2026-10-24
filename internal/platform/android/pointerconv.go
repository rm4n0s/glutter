package android

import "github.com/rm4n0s/glutter/ui"

// pointerState is what the converter knows about one device.
//
// Flutter: lib/ui/window/pointer_data_packet_converter.h PointerState
type pointerState struct {
	pointerIdentifier int
	isDown            bool
	isPanZoomActive   bool
	physicalX         float64
	physicalY         float64
	// buttons are the buttons of the last down, up or hover that was stored (a move does not store
	// them, as in C++).
	buttons  int
	panX     float64
	panY     float64
	scale    float64
	rotation float64
	viewID   ui.ViewID
}

// pointerConverter turns the raw pointer data of the Android input (change, kind, device, position,
// no identifier and no delta) into the data the framework needs: it gives each contact a pointer
// identifier, works out the deltas, and keeps the order the framework checks by synthesizing the
// add before a hover or a down, the hover before a down at another place, the move before an up at
// another place, the cancel and remove before a second add, and the hover or move before a signal
// at another place.
//
//	down -> move -> up   becomes   add(synthesized) -> down -> move -> up
//
// Deviation: Flutter's converter is built over a delegate that tells which views exist; here the
// converter has the function viewExists, which is nil (every view exists) as long as the platform
// has only the implicit view. The converter does not log the FML_DCHECKs: a change for a pointer
// that is in a state the checks forbid is handled as the release build of C++ handles it, and a
// move, up, pan update or pan end for a device the converter has not seen is dropped.
//
// Flutter: lib/ui/window/pointer_data_packet_converter.cc PointerDataPacketConverter
type pointerConverter struct {
	states map[int]pointerState
	// pointer is the last pointer identifier given; the identifiers start at 1.
	pointer int
	// viewExists tells whether a view exists; nil means that every view does.
	viewExists func(ui.ViewID) bool
}

// newPointerConverter makes a converter that has seen no pointer.
//
// Flutter: lib/ui/window/pointer_data_packet_converter.cc PointerDataPacketConverter.PointerDataPacketConverter
func newPointerConverter() *pointerConverter {
	return &pointerConverter{states: map[int]pointerState{}}
}

func (c *pointerConverter) hasView(id ui.ViewID) bool {
	return c.viewExists == nil || c.viewExists(id)
}

// Convert converts a packet of raw data to the data for the framework, in order.
//
// Flutter: lib/ui/window/pointer_data_packet_converter.cc PointerDataPacketConverter.Convert
func (c *pointerConverter) Convert(in []ui.PointerData) []ui.PointerData {
	out := make([]ui.PointerData, 0, len(in))
	for _, d := range in {
		out = c.convertPointerData(d, out)
	}
	return out
}

// convertPointerData appends the converted data of one pointer data to out. Each case works on a
// copy of the state, as the C++ does, and stores it back only where the C++ does.
//
// Flutter: lib/ui/window/pointer_data_packet_converter.cc PointerDataPacketConverter.ConvertPointerData
func (c *pointerConverter) convertPointerData(d ui.PointerData, out []ui.PointerData) []ui.PointerData {
	// Ignores pointer events with an invalid view ID.
	if !c.hasView(d.ViewID) {
		return out
	}
	if d.SignalKind != ui.PointerSignalKindUnspecified && d.SignalKind != ui.PointerSignalKindNone {
		return c.convertSignal(d, out)
	}
	switch d.Change {
	case ui.PointerChangeCancel:
		// Android's three finger gesture sends a cancel to a non-existing pointer: the cancel is
		// dropped if the pointer was not added.
		state, ok := c.states[d.Device]
		if !ok {
			return out
		}
		c.updatePointerIdentifier(&d, &state, false)
		if locationNeedsUpdate(d, state) {
			// Synthesizes a move event if the location does not match.
			move := d
			move.Change = ui.PointerChangeMove
			move.Synthesized = true
			c.updateDeltaAndState(&move, &state)
			out = append(out, move)
		}
		state.isDown = false
		c.states[d.Device] = state
		out = append(out, d)
	case ui.PointerChangeAdd:
		if state, ok := c.states[d.Device]; ok {
			// Synthesizes a remove event if the pointer was not previously removed.
			synthesized := d
			synthesized.PhysicalX = state.physicalX
			synthesized.PhysicalY = state.physicalY
			synthesized.PanX = state.panX
			synthesized.PanY = state.panY
			synthesized.Scale = state.scale
			synthesized.Rotation = state.rotation
			synthesized.Buttons = state.buttons
			synthesized.Synthesized = true

			// The framework expects an add event to always follow a remove event, and remove events
			// with invalid views are ignored: the view of the add event is used for the remove
			// event if the old view has been destroyed.
			if c.hasView(state.viewID) {
				synthesized.ViewID = state.viewID
				if state.isDown {
					// Synthesizes a cancel event if the pointer is down.
					cancel := synthesized
					cancel.Change = ui.PointerChangeCancel
					c.updatePointerIdentifier(&cancel, &state, false)
					out = append(out, cancel)
				}
			}
			remove := synthesized
			remove.Change = ui.PointerChangeRemove
			out = append(out, remove)
		}
		c.ensurePointerState(d)
		out = append(out, d)
	case ui.PointerChangeRemove:
		state, ok := c.states[d.Device]
		if !ok {
			return out
		}
		if state.viewID != d.ViewID {
			// Ignores a remove if the pointer was added to a different view.
			return out
		}
		if state.isDown {
			// Synthesizes a cancel event if the pointer is down.
			cancel := d
			cancel.Change = ui.PointerChangeCancel
			cancel.Synthesized = true
			c.updatePointerIdentifier(&cancel, &state, false)
			state.isDown = false
			c.states[cancel.Device] = state
			out = append(out, cancel)
		}
		if locationNeedsUpdate(d, state) {
			// Synthesizes a hover event if the location does not match.
			hover := d
			hover.Change = ui.PointerChangeHover
			hover.Synthesized = true
			c.updateDeltaAndState(&hover, &state)
			out = append(out, hover)
		}
		delete(c.states, d.Device)
		out = append(out, d)
	case ui.PointerChangeHover:
		state, out2 := c.stateOrSynthesizeAdd(d, out)
		out = out2
		state.buttons = d.Buttons
		if locationNeedsUpdate(d, state) {
			c.updateDeltaAndState(&d, &state)
			out = append(out, d)
		}
	case ui.PointerChangeDown:
		state, out2 := c.stateOrSynthesizeAdd(d, out)
		out = out2
		if locationNeedsUpdate(d, state) {
			// Synthesizes a hover event if the location does not match.
			hover := d
			hover.Change = ui.PointerChangeHover
			hover.Synthesized = true
			hover.Buttons = 0
			c.updateDeltaAndState(&hover, &state)
			out = append(out, hover)
		}
		c.updatePointerIdentifier(&d, &state, true)
		state.isDown = true
		state.buttons = d.Buttons
		c.states[d.Device] = state
		out = append(out, d)
	case ui.PointerChangeMove:
		state, ok := c.states[d.Device]
		if !ok {
			return out
		}
		c.updatePointerIdentifier(&d, &state, false)
		c.updateDeltaAndState(&d, &state)
		out = append(out, d)
	case ui.PointerChangeUp:
		state, ok := c.states[d.Device]
		if !ok {
			return out
		}
		c.updatePointerIdentifier(&d, &state, false)
		if locationNeedsUpdate(d, state) {
			// Synthesizes a move event if the location does not match.
			move := d
			move.Change = ui.PointerChangeMove
			move.Buttons = state.buttons
			move.Synthesized = true
			c.updateDeltaAndState(&move, &state)
			out = append(out, move)
		}
		state.isDown = false
		state.buttons = d.Buttons
		c.states[d.Device] = state
		out = append(out, d)
	case ui.PointerChangePanZoomStart:
		state, out2 := c.stateOrSynthesizeAdd(d, out)
		out = out2
		if locationNeedsUpdate(d, state) {
			// Synthesizes a hover event if the location does not match.
			hover := d
			hover.Change = ui.PointerChangeHover
			hover.Synthesized = true
			hover.Buttons = 0
			c.updateDeltaAndState(&hover, &state)
			out = append(out, hover)
		}
		c.updatePointerIdentifier(&d, &state, true)
		state.isPanZoomActive = true
		state.panX = 0
		state.panY = 0
		state.scale = 1
		state.rotation = 0
		c.states[d.Device] = state
		out = append(out, d)
	case ui.PointerChangePanZoomUpdate:
		state, ok := c.states[d.Device]
		if !ok {
			return out
		}
		c.updatePointerIdentifier(&d, &state, false)
		c.updateDeltaAndState(&d, &state)
		out = append(out, d)
	case ui.PointerChangePanZoomEnd:
		state, ok := c.states[d.Device]
		if !ok {
			return out
		}
		c.updatePointerIdentifier(&d, &state, false)
		if locationNeedsUpdate(d, state) {
			// Synthesizes an update event if the location does not match.
			update := d
			update.Change = ui.PointerChangePanZoomUpdate
			update.PanX = state.panX
			update.PanY = state.panY
			update.PanDeltaX = 0
			update.PanDeltaY = 0
			update.Scale = state.scale
			update.Rotation = state.rotation
			update.Synthesized = true
			c.updateDeltaAndState(&update, &state)
			out = append(out, update)
		}
		state.isPanZoomActive = false
		c.states[d.Device] = state
		out = append(out, d)
	default:
		out = append(out, d)
	}
	return out
}

// convertSignal converts the data of a signal (scroll, scroll inertia cancel, scale): the pointer
// is added, and hovered or moved to the place of the signal, before it. An unknown signal is
// ignored.
//
// Flutter: lib/ui/window/pointer_data_packet_converter.cc PointerDataPacketConverter.ConvertPointerData
func (c *pointerConverter) convertSignal(d ui.PointerData, out []ui.PointerData) []ui.PointerData {
	switch d.SignalKind {
	case ui.PointerSignalKindScroll, ui.PointerSignalKindScrollInertiaCancel, ui.PointerSignalKindScale:
	default:
		return out
	}
	state, ok := c.states[d.Device]
	if !ok {
		// Synthesizes an add event if the pointer was not previously added.
		add := d
		add.SignalKind = ui.PointerSignalKindNone
		add.Change = ui.PointerChangeAdd
		add.Synthesized = true
		add.Buttons = 0
		state = c.ensurePointerState(add)
		out = append(out, add)
	}
	if locationNeedsUpdate(d, state) {
		if state.isDown {
			// Synthesizes a move event if the pointer is down.
			move := d
			move.SignalKind = ui.PointerSignalKindNone
			move.Change = ui.PointerChangeMove
			move.Buttons = state.buttons
			move.Synthesized = true
			c.updateDeltaAndState(&move, &state)
			out = append(out, move)
		} else {
			// Synthesizes a hover event if the pointer is up.
			hover := d
			hover.SignalKind = ui.PointerSignalKindNone
			hover.Change = ui.PointerChangeHover
			hover.Buttons = 0
			hover.Synthesized = true
			c.updateDeltaAndState(&hover, &state)
			out = append(out, hover)
		}
	}
	return append(out, d)
}

// stateOrSynthesizeAdd returns the state of the device of d; if there is none, it adds the device
// with a synthesized add event, appended to out.
func (c *pointerConverter) stateOrSynthesizeAdd(d ui.PointerData, out []ui.PointerData) (pointerState, []ui.PointerData) {
	if state, ok := c.states[d.Device]; ok {
		return state, out
	}
	add := d
	add.Change = ui.PointerChangeAdd
	add.Synthesized = true
	add.Buttons = 0
	state := c.ensurePointerState(add)
	return state, append(out, add)
}

// ensurePointerState starts the state of the device of d, at its position.
//
// Flutter: lib/ui/window/pointer_data_packet_converter.cc PointerDataPacketConverter.EnsurePointerState
func (c *pointerConverter) ensurePointerState(d ui.PointerData) pointerState {
	state := pointerState{physicalX: d.PhysicalX, physicalY: d.PhysicalY, viewID: d.ViewID}
	c.states[d.Device] = state
	return state
}

// updateDeltaAndState sets the deltas of d from the state and moves the state to d.
//
// Flutter: lib/ui/window/pointer_data_packet_converter.cc PointerDataPacketConverter.UpdateDeltaAndState
func (c *pointerConverter) updateDeltaAndState(d *ui.PointerData, state *pointerState) {
	d.PhysicalDeltaX = d.PhysicalX - state.physicalX
	d.PhysicalDeltaY = d.PhysicalY - state.physicalY
	d.PanDeltaX = d.PanX - state.panX
	d.PanDeltaY = d.PanY - state.panY
	state.physicalX = d.PhysicalX
	state.physicalY = d.PhysicalY
	state.panX = d.PanX
	state.panY = d.PanY
	state.scale = d.Scale
	state.rotation = d.Rotation
	c.states[d.Device] = *state
}

// locationNeedsUpdate reports whether the state is at another place than d.
//
// Flutter: lib/ui/window/pointer_data_packet_converter.cc PointerDataPacketConverter.LocationNeedsUpdate
func locationNeedsUpdate(d ui.PointerData, state pointerState) bool {
	return state.physicalX != d.PhysicalX || state.physicalY != d.PhysicalY
}

// updatePointerIdentifier gives d the identifier of the state; when startNewPointer is true, the
// state first gets a new one.
//
// Flutter: lib/ui/window/pointer_data_packet_converter.cc PointerDataPacketConverter.UpdatePointerIdentifier
func (c *pointerConverter) updatePointerIdentifier(d *ui.PointerData, state *pointerState, startNewPointer bool) {
	if startNewPointer {
		c.pointer++
		state.pointerIdentifier = c.pointer
		c.states[d.Device] = *state
	}
	d.PointerIdentifier = state.pointerIdentifier
}
