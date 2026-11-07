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

type sensorApp struct {
	m.StatelessWidget
	p *probe
}

func newSensorApp(p *probe) *sensorApp { return &sensorApp{p: p} }

func (a *sensorApp) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Sensors test",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       newSensorPage(a.p),
	})
}

type sensorPage struct {
	m.StatefulWidget
	p *probe
}

func newSensorPage(p *probe) *sensorPage {
	return &sensorPage{StatefulWidget: m.NewStatefulWidgetBase(), p: p}
}

func (w *sensorPage) CreateState() m.State { return &sensorPageState{} }

type sensorPageState struct {
	m.StateOf[*sensorPage]
}

func (s *sensorPageState) Build(bc m.BuildContext) m.Widget {
	p := s.Widget().p
	if p.binding == nil {
		p.binding = m.WidgetsBindingOf(bc)
		close(p.ready)
	}
	return m.NewScaffold(m.ScaffoldOpts{Body: m.NewCenter(m.NewText("Sensors test"))})
}
