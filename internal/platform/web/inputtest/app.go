//go:build js && wasm

package main

import (
	"strconv"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
)

// The layout, in CSS pixels of the 480x640 viewport the tests give: the button is in a band
// 100 pixels high at the top that does not scroll, and the list fills the rest.
const (
	bandHeight = 100.0
	// buttonX and buttonY are the center of the button.
	buttonX, buttonY = 240.0, 50.0
	// listX and listY are a point inside the list.
	listX, listY = 240.0, 400.0
	rowHeight    = 48.0
	rows         = 200
)

// probe is what the app tells the driver, and the way the driver reaches the UI goroutine.
type probe struct {
	// ready is closed when the app has been built.
	ready chan struct{}
	// post runs a function on the UI goroutine.
	post func(func())
	// binding is the widgets binding, found when the app is built.
	binding *m.WidgetsBinding

	count int
	// downs are the pointer-down events the framework saw, as "x,y".
	downs    []string
	keyCount int
	offset   float64
	ctrl     *m.ScrollControllerBase
}

func newProbe() *probe { return &probe{ready: make(chan struct{})} }

// onUI runs fn on the UI goroutine and waits for it, which the driver (another goroutine) needs to
// read the state of the app.
func (p *probe) onUI(fn func()) {
	done := make(chan struct{})
	p.post(func() {
		fn()
		close(done)
	})
	<-done
}

// incrementIntent is what the K key asks for.
type incrementIntent struct{ m.IntentBase }

type inputApp struct {
	m.StatelessWidget
	p *probe
}

func newInputApp(p *probe) *inputApp { return &inputApp{p: p} }

func (a *inputApp) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Input test",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       newInputPage(a.p),
	})
}

type inputPage struct {
	m.StatefulWidget
	p *probe
}

func newInputPage(p *probe) *inputPage {
	return &inputPage{StatefulWidget: m.NewStatefulWidgetBase(), p: p}
}

func (w *inputPage) CreateState() m.State { return &inputPageState{} }

type inputPageState struct {
	m.StateOf[*inputPage]
}

func (s *inputPageState) InitState() {
	p := s.Widget().p
	p.ctrl = m.NewScrollController()
	p.ctrl.AddListener(func() { p.offset = p.ctrl.Offset() })
}

func (s *inputPageState) Build(bc m.BuildContext) m.Widget {
	p := s.Widget().p
	if p.binding == nil {
		p.binding = m.WidgetsBindingOf(bc)
		p.post = p.binding.Post
		close(p.ready)
	}
	list := m.NewListViewBuilder(func(bc m.BuildContext, i int) m.Widget {
		if i >= rows {
			return nil
		}
		return m.NewSizedBox(m.NewText("Item "+strconv.Itoa(i)), m.SizedBoxOpts{Height: foundation.Some(rowHeight)})
	}, m.ListViewOpts{Controller: p.ctrl})
	body := m.NewColumn([]m.Widget{
		m.NewSizedBox(m.NewCenter(
			m.NewElevatedButton(func() { s.SetState(func() { p.count++ }) }, m.NewText("count "+strconv.Itoa(p.count))),
		), m.SizedBoxOpts{Height: foundation.Some(bandHeight)}),
		m.NewExpanded(list),
	})
	listener := m.NewListener(body, m.ListenerOpts{
		Behavior: m.HitTestBehaviorTranslucent,
		OnPointerDown: func(e *gestures.PointerDownEvent) {
			p.downs = append(p.downs, strconv.FormatFloat(e.Position().Dx, 'f', 0, 64)+","+strconv.FormatFloat(e.Position().Dy, 'f', 0, 64))
		},
	})
	return m.NewScaffold(m.ScaffoldOpts{
		Body: m.NewActions(m.ActionMap{
			m.IntentTypeOf[incrementIntent](): m.NewCallbackAction(func(incrementIntent) any {
				p.keyCount++
				return nil
			}),
		}, m.NewShortcuts(m.ShortcutMap{
			{Activator: m.NewSingleActivator(services.LogicalKeyboardKeyKeyK), Intent: incrementIntent{}},
		}, m.NewFocus(listener, m.FocusOpts{Autofocus: true}))),
	})
}
