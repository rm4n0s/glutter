//go:build js && wasm

package main

import (
	"strconv"

	"github.com/rm4n0s/glutter/foundation"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
)

// probe is what the app tells the driver, and the way the driver reaches the UI goroutine.
type probe struct {
	ready   chan struct{}
	post    func(func())
	binding *m.WidgetsBinding
	// counts are the activations of the two buttons, and focused which of them has the focus (0 or 1, -1 for neither).
	counts  [2]int
	nodes   [2]*m.FocusNode
	focused func() int
}

func newProbe() *probe { return &probe{ready: make(chan struct{})} }

// onUI runs fn on the UI goroutine and waits for it.
func (p *probe) onUI(fn func()) {
	done := make(chan struct{})
	p.post(func() {
		fn()
		close(done)
	})
	<-done
}

type padApp struct {
	m.StatelessWidget
	p *probe
}

func newPadApp(p *probe) *padApp { return &padApp{p: p} }

func (a *padApp) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Pad test",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       newPadPage(a.p),
	})
}

type padPage struct {
	m.StatefulWidget
	p *probe
}

func newPadPage(p *probe) *padPage { return &padPage{StatefulWidget: m.NewStatefulWidgetBase(), p: p} }

func (w *padPage) CreateState() m.State { return &padPageState{} }

type padPageState struct {
	m.StateOf[*padPage]
}

func (s *padPageState) InitState() {
	p := s.Widget().p
	for i := range p.nodes {
		p.nodes[i] = m.NewFocusNode()
	}
	p.focused = func() int {
		for i, n := range p.nodes {
			if n.HasPrimaryFocus() {
				return i
			}
		}
		return -1
	}
}

func (s *padPageState) Dispose() {
	for _, n := range s.Widget().p.nodes {
		n.Dispose()
	}
}

func (s *padPageState) Build(bc m.BuildContext) m.Widget {
	p := s.Widget().p
	if p.binding == nil {
		p.binding = m.WidgetsBindingOf(bc)
		p.post = p.binding.Post
		close(p.ready)
	}
	button := func(i int) m.Widget {
		return m.NewElevatedButton(func() { s.SetState(func() { p.counts[i]++ }) },
			m.NewText("button "+strconv.Itoa(i)+": "+strconv.Itoa(p.counts[i])),
			m.ElevatedButtonOpts{Autofocus: i == 0, FocusNode: p.nodes[i]})
	}
	return m.NewScaffold(m.ScaffoldOpts{
		// The web's default shortcuts give the arrow keys to scrolling, not to the focus: an app that wants the D-pad to move the
		// focus says so, as this one does.
		Body: m.NewShortcuts(m.ShortcutMap{
			{Activator: m.NewSingleActivator(services.LogicalKeyboardKeyArrowDown), Intent: m.NextFocusIntent{}},
			{Activator: m.NewSingleActivator(services.LogicalKeyboardKeyArrowUp), Intent: m.PreviousFocusIntent{}},
		}, m.NewColumn([]m.Widget{
			m.NewSizedBox(button(0), m.SizedBoxOpts{Height: foundation.Some(80.0)}),
			m.NewSizedBox(button(1), m.SizedBoxOpts{Height: foundation.Some(80.0)}),
		})),
	})
}
