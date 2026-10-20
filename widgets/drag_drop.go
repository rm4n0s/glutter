package widgets

import (
	"net/url"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/ui"
)

// The bridge between DragTarget and Draggable and the drag and drop of the platform
// (SPEC §8.11 "Data exchange"): the drags of other applications reach the
// DragTargets as *ExternalDragData, and a Draggable with an Export starts a drag of
// the platform.

// dropState is the state of the drags of the platform that go through the binding.
type dropState struct {
	// sessions are the drags of the platform that are over a view.
	sessions map[ui.ViewID]*dragAvatar
	// exports are the drags this app started, by the ID of the request.
	exports map[uint64]*exportedDrag
	nextID  uint64
}

func newDropState() *dropState {
	return &dropState{sessions: map[ui.ViewID]*dragAvatar{}, exports: map[uint64]*exportedDrag{}}
}

// installDrop connects the dispatcher's drop events to the binding.
func (b *WidgetsBinding) installDrop() {
	b.drops = newDropState()
	b.d.OnDrop = b.handleDrop
	b.d.OnDragEnd = b.handleDragEnd
}

// exportedDrag is the Drag of a Draggable whose drag the platform runs: the
// pointer events no longer reach the app, the platform tells how it ended.
type exportedDrag struct {
	id        uint64
	data      any
	offset    ui.Offset
	onDragEnd func(velocity gestures.Velocity, offset ui.Offset, wasAccepted bool)
}

// Update ignores the move: the platform runs the drag.
func (e *exportedDrag) Update(gestures.DragUpdateDetails) {}

// End ignores the pointer being lifted: the platform runs the drag.
func (e *exportedDrag) End(gestures.DragEndDetails) {}

// Cancel ignores the cancel of the pointer, which is what the platform's drag does
// to the pointer events of the app.
func (e *exportedDrag) Cancel() {}

// startExportedDrag starts the drag of the platform for a Draggable with an Export.
// It returns nil when the platform cannot, and the Draggable then drags inside its
// window.
func startExportedDrag(b *WidgetsBinding, view ui.ViewID, position, anchor ui.Offset, export *DragExport, data any,
	onDragEnd func(gestures.Velocity, ui.Offset, bool)) *exportedDrag {
	if export == nil || b.drops == nil {
		return nil
	}
	b.drops.nextID++
	id := b.drops.nextID
	err := b.d.StartDrag(view, ui.DragRequest{ID: id, Text: export.Text, URIs: export.URIs, Actions: export.Actions})
	if err != nil {
		return nil
	}
	e := &exportedDrag{id: id, data: data, offset: position.Sub(anchor), onDragEnd: onDragEnd}
	b.drops.exports[id] = e
	return e
}

// handleDragEnd is the end of a drag that the app started.
func (b *WidgetsBinding) handleDragEnd(result ui.DragResult) {
	e, ok := b.drops.exports[result.ID]
	if !ok {
		return
	}
	delete(b.drops.exports, result.ID)
	e.onDragEnd(gestures.Velocity{}, e.offset, result.Action != ui.DropActionNone)
}

// handleDrop is a drag of the platform over a view: it plays the part of the drag
// avatar of a Draggable, with the data the platform gives.
func (b *WidgetsBinding) handleDrop(event ui.DropEvent) ui.DropActions {
	view := b.d.View(event.ViewID)
	if view == nil {
		return ui.DropActionNone
	}
	position := event.Position.Div(max(view.DevicePixelRatio(), 1))
	avatar := b.drops.sessions[event.ViewID]
	switch event.Phase {
	case ui.DropPhaseLeave:
		if avatar != nil {
			delete(b.drops.sessions, event.ViewID)
			avatar.finishDrag(dragEndCanceled, foundation.None[gestures.Velocity]())
		}
		return ui.DropActionNone
	case ui.DropPhaseEnter, ui.DropPhaseMove, ui.DropPhaseDrop:
		if avatar == nil {
			avatar = b.newExternalAvatar(event, position)
			b.drops.sessions[event.ViewID] = avatar
		}
	}
	if ext, ok := avatar.data.(*ExternalDragData); ok {
		ext.Types, ext.Actions = event.Types, event.Actions
		if event.Phase == ui.DropPhaseDrop {
			ext.Text, ext.URIs, ext.Dropped = event.Text, event.URIs, true
		}
	}
	avatar.position = position
	avatar.updateDrag(position)
	accepted := avatar.activeTarget != nil
	if event.Phase == ui.DropPhaseDrop {
		delete(b.drops.sessions, event.ViewID)
		if accepted {
			avatar.finishDrag(dragEndDropped, foundation.None[gestures.Velocity]())
		} else {
			avatar.finishDrag(dragEndCanceled, foundation.None[gestures.Velocity]())
		}
	}
	if !accepted {
		return ui.DropActionNone
	}
	return chooseDropAction(event.Actions)
}

// newExternalAvatar starts the avatar of a drag of the platform. When the drag is
// one that this app started, its data is the Draggable's own.
func (b *WidgetsBinding) newExternalAvatar(event ui.DropEvent, position ui.Offset) *dragAvatar {
	var data any
	if local, ok := b.drops.exports[event.LocalID]; ok && event.LocalID != 0 && local.data != nil {
		data = local.data
	} else {
		data = &ExternalDragData{Types: event.Types, Actions: event.Actions}
	}
	return &dragAvatar{dragAvatarArgs: dragAvatarArgs{binding: b, data: data, initialPosition: position, viewID: event.ViewID}, position: position}
}

// chooseDropAction picks the action to perform among those the source allows: copy
// if it can, else move, else link.
func chooseDropAction(offered ui.DropActions) ui.DropActions {
	switch {
	case offered == ui.DropActionNone, offered.Has(ui.DropActionCopy):
		return ui.DropActionCopy
	case offered.Has(ui.DropActionMove):
		return ui.DropActionMove
	case offered.Has(ui.DropActionLink):
		return ui.DropActionLink
	}
	return ui.DropActionNone
}

// fileURIPath is the path of a file:// URI.
func fileURIPath(uri string) (string, bool) {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" || (u.Host != "" && u.Host != "localhost") || u.Path == "" {
		return "", false
	}
	return u.Path, true
}
