//go:build js && wasm

package main

import (
	m "github.com/rm4n0s/glutter/material"
)

// probe is what the app tells the driver: the context of the page, from which package desktop is called.
type probe struct {
	// ready is closed when the app has been built.
	ready chan struct{}
	// bc is a context below the View, and binding the widgets binding, whose UI thread runs the callbacks.
	bc      m.BuildContext
	binding *m.WidgetsBinding
}

// isUIThread says whether the caller runs on the UI goroutine.
func (p *probe) isUIThread() bool { return p.binding.UIThread().IsCurrent() }

func newProbe() *probe { return &probe{ready: make(chan struct{})} }

type page struct {
	m.StatelessWidget
	p *probe
}

func (a *page) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Desktop test",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       &home{StatefulWidget: m.NewStatefulWidgetBase(), p: a.p},
	})
}

type home struct {
	m.StatefulWidget
	p *probe
}

func (w *home) CreateState() m.State { return &homeState{} }

type homeState struct{ m.StateOf[*home] }

func (s *homeState) Build(bc m.BuildContext) m.Widget {
	p := s.Widget().p
	if p.bc == nil {
		p.bc = bc
		p.binding = m.WidgetsBindingOf(bc)
		close(p.ready)
	}
	return m.NewScaffold(m.ScaffoldOpts{Body: m.NewCenter(m.NewText("Desktop services"))})
}
