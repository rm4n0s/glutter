//go:build js && wasm

package main

import (
	m "github.com/rm4n0s/glutter/material"
)

// The layout, in CSS pixels of the 480x640 viewport the tests give: the single-line field is the first in a
// column with a 16 pixel padding and is 56 pixels high.
const singleX, singleY = 240.0, 44.0

// probe is what the app tells the driver, and the way the driver reaches the UI goroutine.
type probe struct {
	// ready is closed when the app has been built.
	ready chan struct{}
	// post runs a function on the UI goroutine.
	post func(func())
	// binding is the widgets binding, found when the app is built.
	binding *m.WidgetsBinding

	single, secret, multi                *m.TextEditingController
	singleFocus, secretFocus, multiFocus *m.FocusNode
}

func newProbe() *probe {
	return &probe{
		ready:  make(chan struct{}),
		single: m.NewTextEditingController(), secret: m.NewTextEditingController(), multi: m.NewTextEditingController(),
		singleFocus: m.NewFocusNode(), secretFocus: m.NewFocusNode(), multiFocus: m.NewFocusNode(),
	}
}

// onUI runs fn on the UI goroutine and waits for it, which the driver (another goroutine) needs to read the
// state of the app.
func (p *probe) onUI(fn func()) {
	done := make(chan struct{})
	p.post(func() {
		fn()
		close(done)
	})
	<-done
}

type textApp struct {
	m.StatelessWidget
	p *probe
}

func newTextApp(p *probe) *textApp { return &textApp{p: p} }

func (a *textApp) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Text test",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       newTextPage(a.p),
	})
}

type textPage struct {
	m.StatefulWidget
	p *probe
}

func newTextPage(p *probe) *textPage {
	return &textPage{StatefulWidget: m.NewStatefulWidgetBase(), p: p}
}

func (w *textPage) CreateState() m.State { return &textPageState{} }

type textPageState struct {
	m.StateOf[*textPage]
}

func (s *textPageState) Build(bc m.BuildContext) m.Widget {
	p := s.Widget().p
	if p.binding == nil {
		p.binding = m.WidgetsBindingOf(bc)
		p.post = p.binding.Post
		close(p.ready)
	}
	gap := m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(16.0)})
	return m.NewScaffold(m.ScaffoldOpts{
		Body: m.NewPadding(m.NewEdgeInsetsAll(16), m.NewColumn([]m.Widget{
			m.NewTextField(m.TextFieldOpts{Controller: p.single, FocusNode: p.singleFocus}),
			gap,
			m.NewTextField(m.TextFieldOpts{Controller: p.secret, FocusNode: p.secretFocus, ObscureText: true}),
			gap,
			m.NewTextField(m.TextFieldOpts{Controller: p.multi, FocusNode: p.multiFocus, MaxLines: m.Some(3),
				KeyboardType: m.Some(m.TextInputTypeMultiline)}),
		})),
	})
}
