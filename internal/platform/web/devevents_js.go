//go:build js && wasm

package web

import (
	"time"

	"github.com/rm4n0s/glutter/internal/platform/web/synthetic"
)

// startDevEvents replays the script of GLUTTER_DEV_EVENTS (debug builds only), once, from a
// goroutine of its own: the events are dispatched as the browser would deliver them, and the
// listeners that wait for the UI goroutine need it free (SPEC §23.3). It is started after the
// first frame, so that the app has laid out and the canvas is in the page.
func (p *Platform) startDevEvents() {
	if len(p.devEvents) == 0 || p.devStarted {
		return
	}
	p.devStarted = true
	events := p.devEvents
	go func() {
		canvas, err := synthetic.Canvas()
		if err != nil {
			p.log.Error("glutter: GLUTTER_DEV_EVENTS cannot start", "error", err)
			return
		}
		for _, e := range events {
			switch e.Kind {
			case "wait":
				time.Sleep(e.Wait)
			case "wheel":
				synthetic.DispatchWheel(canvas, e.X, e.Y, e.DX, e.DY)
			case "move":
				synthetic.DispatchPointer(canvas, synthetic.Pointer{Type: "pointermove", X: e.X, Y: e.Y})
			case "tap":
				synthetic.DispatchPointer(canvas, synthetic.Pointer{Type: "pointerdown", X: e.X, Y: e.Y, Buttons: 1, Pressure: 0.5})
				time.Sleep(50 * time.Millisecond)
				synthetic.DispatchPointer(canvas, synthetic.Pointer{Type: "pointerup", X: e.X, Y: e.Y})
			case "input", "compose", "paste":
				field, err := synthetic.TextField()
				if err != nil {
					p.log.Error("glutter: GLUTTER_DEV_EVENTS cannot find the text field", "error", err)
					continue
				}
				switch e.Kind {
				case "input":
					synthetic.TypeText(field, e.Text)
				case "compose":
					synthetic.Compose(field, e.Steps...)
				case "paste":
					synthetic.Paste(field, e.Text)
				}
			case "key":
				synthetic.DispatchKey(canvas, synthetic.Key{Type: "keydown", Code: e.Code, Key: e.Key})
				time.Sleep(20 * time.Millisecond)
				synthetic.DispatchKey(canvas, synthetic.Key{Type: "keyup", Code: e.Code, Key: e.Key})
			}
		}
	}()
}
