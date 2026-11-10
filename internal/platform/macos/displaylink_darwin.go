//go:build darwin && !ios

package macos

import (
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/loop"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// frameLink is what paces the frames of one window (SPEC §29.3): a display link that ticks while it
// runs, on the main thread, and that follows the window to another display. Two implementations are
// behind it: the view's CADisplayLink on macOS 14 and later, and a CVDisplayLink on macOS 12 and 13,
// whose callback is on a thread of its own and posts to the loop. The pacing decisions are plain Go
// (pacer.go).
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterDisplayLink.mm FlutterDisplayLink
type frameLink interface {
	// setRunning starts or stops the ticks. A stopped link costs nothing.
	setRunning(run bool)
	// screenChanged tells the link that the window is now on another display.
	screenChanged()
	// close releases the link.
	close()
}

// linkEnv is the environment variable that picks the display link, for the tests: "cv" is CVDisplayLink
// even where CADisplayLink exists, "ca" is the view's CADisplayLink even for a self-capture.
const linkEnv = "GLUTTER_DISPLAYLINK"

// newFrameLink makes the link of the window, with fire called on the main thread with the target time
// of the tick in seconds of the clock of CACurrentMediaTime.
func (w *window) newFrameLink(fire func(target float64)) (frameLink, error) {
	s, err := sys()
	if err != nil {
		return nil, err
	}
	// A self-capture (GLUTTER_SCREENSHOT) uses CVDisplayLink as well: it ticks whether or not the window is
	// visible, so a proof can be taken while the screen is asleep or locked (SPEC §29.13).
	useCV := w.p.getenv(linkEnv) == "cv" || (w.p.getenv(linkEnv) == "" && w.p.getenv("GLUTTER_SCREENSHOT") != "")
	if !useCV && objc.Send[bool](w.view, sel("respondsToSelector:"), sel("displayLinkWithTarget:selector:")) {
		return newCADisplayLink(w, fire), nil
	}
	return newCVDisplayLink(s, w, fire)
}

// caLink is a CADisplayLink of the view (macOS 14 and later): it follows the window to another display
// and to a ProMotion refresh rate by itself.
type caLink struct {
	link objc.ID
}

func newCADisplayLink(w *window, fire func(float64)) *caLink {
	// The view is the target: its method displayLinkFired: finds the window by the handle and calls fire.
	l := objc.ID(w.view).Send(sel("displayLinkWithTarget:selector:"), w.view, sel("displayLinkFired:"))
	objcx.Retain(l)
	l.Send(sel("setPaused:"), true)
	loop := class("NSRunLoop").Send(sel("mainRunLoop"))
	mode := objcx.NewString("kCFRunLoopCommonModes")
	l.Send(sel("addToRunLoop:forMode:"), loop, mode)
	objcx.Release(mode)
	w.onLinkFire = fire
	return &caLink{link: l}
}

func (c *caLink) setRunning(run bool) { c.link.Send(sel("setPaused:"), !run) }
func (c *caLink) screenChanged()      {}
func (c *caLink) close() {
	if c.link != 0 {
		c.link.Send(sel("invalidate"))
		objcx.Release(c.link)
		c.link = 0
	}
}

// cvFuncs are the CVDisplayLink functions.
type cvFuncs struct {
	create       func(out *uintptr) int32
	setCallback  func(link uintptr, cb uintptr, userInfo uintptr) int32
	start        func(link uintptr) int32
	stop         func(link uintptr) int32
	release      func(link uintptr)
	setDisplay   func(link uintptr, display uint32) int32
	callbackOnce uintptr
}

func (f *cvFuncs) bind() error {
	lib, err := dl.OpenFramework("CoreVideo")
	if err != nil {
		return err
	}
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&f.create, "CVDisplayLinkCreateWithActiveCGDisplays"}, {&f.setCallback, "CVDisplayLinkSetOutputCallback"},
		{&f.start, "CVDisplayLinkStart"}, {&f.stop, "CVDisplayLinkStop"}, {&f.release, "CVDisplayLinkRelease"},
		{&f.setDisplay, "CVDisplayLinkSetCurrentCGDisplay"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			return err
		}
	}
	return nil
}

// cvOutput is the one callback of CVDisplayLink of the process (a purego callback is made once, SPEC
// §6.3 rule 11): it runs on the link's own thread, finds the window by the handle in userInfo and offers
// a tick to the main thread. The link thread does nothing else: no Objective-C, no handler.
var cvOutput = sync.OnceValue(func() uintptr {
	return dl.Callback(func(link, now, out uintptr, flagsIn uint64, flagsOut uintptr, userInfo uintptr) int32 {
		w, ok := objcx.LookupAs[*window](objcx.Handle(userInfo))
		if !ok {
			return 0
		}
		w.cvOffer()
		return 0
	})
})

// cvLink is a CVDisplayLink of the window (macOS 12 and 13).
type cvLink struct {
	s       *system
	w       *window
	link    uintptr
	running bool
	gate    loop.TickGate
	fire    func(float64)
}

func newCVDisplayLink(s *system, w *window, fire func(float64)) (*cvLink, error) {
	c := &cvLink{s: s, w: w, fire: fire}
	var link uintptr
	if rc := s.cv.create(&link); rc != 0 || link == 0 {
		return nil, errors.New(tagCVDisplayLinkFailed, "CVDisplayLinkCreateWithActiveCGDisplays failed", "cvreturn", int(rc))
	}
	c.link = link
	if rc := s.cv.setCallback(link, cvOutput(), uintptr(w.handle)); rc != 0 {
		s.cv.release(link)
		return nil, errors.New(tagCVDisplayLinkFailed, "CVDisplayLinkSetOutputCallback failed", "cvreturn", int(rc))
	}
	w.cv = c
	c.screenChanged()
	return c, nil
}

func (c *cvLink) setRunning(run bool) {
	if run == c.running {
		return
	}
	c.running = run
	if run {
		c.s.cv.start(c.link)
	} else {
		c.s.cv.stop(c.link)
	}
}

// screenChanged points the link at the display of the window.
func (c *cvLink) screenChanged() {
	if id, ok := c.w.displayID(); ok {
		c.s.cv.setDisplay(c.link, id)
	}
}

func (c *cvLink) close() {
	if c.link != 0 {
		c.s.cv.stop(c.link)
		c.s.cv.release(c.link)
		c.link = 0
	}
}

// offer is called on the link's thread: a tick goes to the main thread unless one waits there.
func (c *cvLink) offer() {
	if !c.gate.Offer() {
		return
	}
	c.w.p.Post(func() {
		c.gate.Done()
		if c.link != 0 {
			c.fire(c.s.mediaTime() + c.w.frameInterval())
		}
	})
}

const tagCVDisplayLinkFailed = "CVDisplayLinkFailed"
