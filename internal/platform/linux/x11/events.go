package x11

// handleEvent handles the event that XNextEvent put in p.event.
func (p *Platform) handleEvent() {
	any := view[xAnyEvent](p.event)
	switch any.typ {
	case genericEvent:
		p.handleGenericEvent()
	case clientMessage:
		ev := view[xClientMessageEvent](p.event)
		if w := p.windows[ev.window]; w != nil && ev.format == 32 {
			w.onClientMessage(uint64(ev.data[0]), ev)
		}
	case configureNotify:
		ev := view[xConfigureEvent](p.event)
		if w := p.windows[ev.window]; w != nil {
			w.onConfigure(ev.width, ev.height)
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
		if p.clip != nil {
			p.clip.onSelectionRequest(*view[xSelectionRequestEvent](p.event))
		}
	case selectionNotify:
		if p.clip != nil {
			p.clip.onSelectionNotify(*view[xSelectionEvent](p.event))
		}
	case selectionClear:
		if p.clip != nil {
			p.clip.onSelectionClear(*view[xSelectionClearEvent](p.event))
		}
	case propertyNotify:
		if p.clip != nil {
			p.clip.onPropertyNotify(*view[xPropertyEvent](p.event))
		}
	case destroyNotify:
		ev := view[xDestroyWindowEvent](p.event)
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
