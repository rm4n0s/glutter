package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
)

// selectionKeepAlive is Flutter's _SelectionKeepAlive: a widget that keeps the
// subtree below it alive in a lazy list while any selectable in it has a
// selection, so that the selection survives scrolling out of view.
//
// Flutter: widgets/scroll_delegate.dart _SelectionKeepAlive
type selectionKeepAlive struct {
	StatefulWidget
	child Widget
}

func newSelectionKeepAlive(child Widget) *selectionKeepAlive {
	return &selectionKeepAlive{StatefulWidget: NewStatefulWidgetBase(), child: child}
}

func (w *selectionKeepAlive) CreateState() State { return &selectionKeepAliveState{} }

type selectionKeepAliveState struct {
	StateOf[*selectionKeepAlive]
	AutomaticKeepAliveClient

	selectablesWithSelections map[rendering.Selectable]struct{}
	selectableAttachments     map[rendering.Selectable]foundation.ListenerHandle
	registrar                 rendering.SelectionRegistrar
	wantKeepAlive             bool
}

var _ rendering.SelectionRegistrar = (*selectionKeepAliveState)(nil)

// WantKeepAlive is whether any selectable below has a selection.
func (s *selectionKeepAliveState) WantKeepAlive() bool { return s.wantKeepAlive }

func (s *selectionKeepAliveState) setWantKeepAlive(value bool) {
	if s.wantKeepAlive != value {
		s.wantKeepAlive = value
		s.UpdateKeepAlive()
	}
}

func (s *selectionKeepAliveState) updateSelectablesWithSelections(selectable rendering.Selectable, add bool) {
	if add {
		if s.selectablesWithSelections == nil {
			s.selectablesWithSelections = map[rendering.Selectable]struct{}{}
		}
		s.selectablesWithSelections[selectable] = struct{}{}
	} else {
		delete(s.selectablesWithSelections, selectable)
	}
	s.setWantKeepAlive(len(s.selectablesWithSelections) > 0)
}

func (s *selectionKeepAliveState) DidChangeDependencies() {
	newRegistrar := SelectionContainerMaybeOf(s.Context())
	if s.registrar != newRegistrar {
		if s.registrar != nil {
			for selectable := range s.selectableAttachments {
				s.registrar.Remove(selectable)
			}
		}
		s.registrar = newRegistrar
		if s.registrar != nil {
			for selectable := range s.selectableAttachments {
				s.registrar.Add(selectable)
			}
		}
	}
}

// Add registers the selectable with the registrar above, and follows whether it
// has a selection.
func (s *selectionKeepAliveState) Add(selectable rendering.Selectable) {
	handle := selectable.AddListener(func() {
		s.updateSelectablesWithSelections(selectable, selectable.Value().HasSelection())
	})
	if s.selectableAttachments == nil {
		s.selectableAttachments = map[rendering.Selectable]foundation.ListenerHandle{}
	}
	s.selectableAttachments[selectable] = handle
	s.registrar.Add(selectable)
	if selectable.Value().HasSelection() {
		s.updateSelectablesWithSelections(selectable, true)
	}
}

// Remove unregisters the selectable.
func (s *selectionKeepAliveState) Remove(selectable rendering.Selectable) {
	if s.selectableAttachments == nil {
		return
	}
	handle := s.selectableAttachments[selectable]
	delete(s.selectableAttachments, selectable)
	selectable.RemoveListener(handle)
	s.registrar.Remove(selectable)
	s.updateSelectablesWithSelections(selectable, false)
}

func (s *selectionKeepAliveState) Dispose() {
	for selectable, handle := range s.selectableAttachments {
		s.registrar.Remove(selectable)
		selectable.RemoveListener(handle)
	}
	s.selectableAttachments = nil
	s.selectablesWithSelections = nil
}

func (s *selectionKeepAliveState) Build(BuildContext) Widget {
	s.KeepAliveBuild()
	w := s.Widget()
	if s.registrar == nil {
		return w.child
	}
	return NewSelectionRegistrarScope(s, w.child)
}
