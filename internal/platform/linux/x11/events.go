package x11

// handleEvent handles the event that XNextEvent put in p.event.
func (p *Platform) handleEvent() {
	any := view[xAnyEvent](p.event)
	if p.rrEventBase != 0 && any.typ == p.rrEventBase+rrScreenChange {
		if r, err := loadXrandrOnce(); err == nil {
			r.updateConfig(p.event)
		}
		p.monitorsChanged()
		return
	}
	switch any.typ {
	case genericEvent:
		p.handleGenericEvent()
	case clientMessage:
		ev := view[xClientMessageEvent](p.event)
		if ev.window == p.root && ev.messageType == p.atoms.manager && ev.format == 32 {
			// A client took a manager selection; if it is the XSETTINGS one, there is a
			// new daemon.
			if uint64(ev.data[1]) == p.atoms.xsettingsSelection {
				p.xsettingsChanged()
			}
			return
		}
		if w := p.windows[ev.window]; w != nil && ev.format == 32 {
			if p.xdnd.onClientMessage(w, ev.messageType, ev.data) {
				return
			}
			w.onClientMessage(uint64(ev.data[0]), ev)
		}
	case configureNotify:
		ev := view[xConfigureEvent](p.event)
		if w := p.windows[ev.window]; w != nil {
			w.onConfigure(ev.width, ev.height)
			p.textInputMoved(w)
		}
	case mapNotify:
		ev := view[xMapEvent](p.event)
		if w := p.windows[ev.window]; w != nil {
			w.onMap()
		}
	case exposeEvent:
		ev := view[xExposeEvent](p.event)
		if w := p.windows[ev.window]; w != nil && ev.count == 0 {
			w.onExpose()
		}
	case selectionRequest:
		if p.xdnd.onSelectionRequest(*view[xSelectionRequestEvent](p.event)) {
			return
		}
		if p.clip != nil {
			p.clip.onSelectionRequest(*view[xSelectionRequestEvent](p.event))
		}
	case selectionNotify:
		if p.xdnd.onSelectionNotify(*view[xSelectionEvent](p.event)) {
			return
		}
		if p.clip != nil {
			p.clip.onSelectionNotify(*view[xSelectionEvent](p.event))
		}
	case selectionClear:
		if p.clip != nil {
			p.clip.onSelectionClear(*view[xSelectionClearEvent](p.event))
		}
	case propertyNotify:
		if ev := view[xPropertyEvent](p.event); ev.window == p.root {
			if ev.atom == p.atoms.resourceManager {
				p.monitorsChanged()
			}
			return
		} else if p.xs != nil && ev.window == p.xs.owner && ev.atom == p.atoms.xsettingsProperty {
			p.xsettingsChanged()
			return
		}
		if p.clip != nil {
			p.clip.onPropertyNotify(*view[xPropertyEvent](p.event))
		}
	case destroyNotify:
		ev := view[xDestroyWindowEvent](p.event)
		if p.xs != nil && ev.window == p.xs.owner {
			p.xs.owner = 0 // the daemon went away; a new one announces itself with MANAGER
			return
		}
		if w := p.windows[ev.window]; w != nil {
			w.onDestroyedByServer()
		}
	}
}

// handleGenericEvent handles the event of an extension: XInput2's. The data of
// the event is fetched with XGetEventData and freed after the handler.
func (p *Platform) handleGenericEvent() {
	cookie := view[xGenericEventCookie](p.event)
	if p.input == nil || int(cookie.extension) != p.input.opcode {
		return
	}
	if p.x.getEventData(p.display, p.event) == 0 {
		return
	}
	defer p.x.freeEventData(p.display, p.event)
	p.input.handle(int(cookie.evtype), cookie.data)
}
