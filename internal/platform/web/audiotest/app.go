//go:build js && wasm

package main

import (
	m "github.com/rm4n0s/glutter/material"
)

// probe hands the binding of the app to the driver.
type probe struct {
	ready   chan struct{}
	binding *m.WidgetsBinding
}

func newProbe() *probe { return &probe{ready: make(chan struct{})} }

type audioApp struct {
	m.StatelessWidget
	p *probe
}

func newAudioApp(p *probe) *audioApp { return &audioApp{p: p} }

func (a *audioApp) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Audio test",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       newAudioPage(a.p),
	})
}

type audioPage struct {
	m.StatefulWidget
	p *probe
}

func newAudioPage(p *probe) *audioPage {
	return &audioPage{StatefulWidget: m.NewStatefulWidgetBase(), p: p}
}

func (w *audioPage) CreateState() m.State { return &audioPageState{} }

type audioPageState struct {
	m.StateOf[*audioPage]
}

func (s *audioPageState) Build(bc m.BuildContext) m.Widget {
	p := s.Widget().p
	if p.binding == nil {
		p.binding = m.WidgetsBindingOf(bc)
		close(p.ready)
	}
	return m.NewScaffold(m.ScaffoldOpts{Body: m.NewCenter(m.NewText("Audio test"))})
}
