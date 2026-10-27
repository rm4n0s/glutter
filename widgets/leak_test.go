package widgets_test

import (
	"image/color"
	"runtime"
	"testing"
	"time"
	"weak"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
	"golang.org/x/image/font/gofont/gomono"
)

// TestTapRegionSurfaceKeepsNoHitTestOfARemovedChild checks that the hit test results that a tap
// region surface remembers for its entries do not keep the render objects of a removed child
// alive: the mouse tracker hit tests after every frame, and none of those entries is ever
// dispatched (step 29.21, SPEC §15 "Memory over time").
func TestTapRegionSurfaceKeepsNoHitTestOfARemovedChild(t *testing.T) {
	tests := []struct {
		name   string
		hovers int
	}{
		{"one hover", 1},
		{"many hovers", 300},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{DiscardScenes: true})
			show := true
			var setShow StateSetter
			ts.PumpWidget(ltr(NewTapRegionSurface(NewStatefulBuilder(func(_ BuildContext, setState StateSetter) Widget {
				setShow = setState
				if !show {
					return NewSizedBox(nil)
				}
				return NewSizedBox(NewMouseRegion(nil, MouseRegionOpts{Key: foundation.NewValueKey("inner")}),
					SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)})
			}))))
			inner := weak.Make(ts.RenderObject(gluttertest.ByKey(foundation.NewValueKey("inner"))).(*rendering.RenderMouseRegion))
			for i := 0; i < tt.hovers; i++ {
				ts.Hover(ui.Offset{Dx: 10 + float64(i%2), Dy: 10})
				ts.Pump(time.Millisecond)
			}
			setShow(func() { show = false })
			ts.Pump()
			runtime.GC()
			runtime.GC()
			if inner.Value() != nil {
				t.Error("the render object of the removed child is still alive")
			}
		})
	}
}

// TestOverlayPortalLeavesNoLocationWhenItIsRemoved checks that a portal that is removed from the
// tree takes its location out of the child model of its overlay entry: the list is read by every
// paint of the theater, and it grew by one location for each portal that was ever shown and
// removed, as long as the entry lived (a slider in a list that scrolls in and out of view).
func TestOverlayPortalLeavesNoLocationWhenItIsRemoved(t *testing.T) {
	tests := []struct {
		name      string
		portals   int
		showFirst bool
	}{
		{"one portal", 1, true},
		{"three portals", 3, true},
		{"one portal that is never shown", 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{DiscardScenes: true})
			controllers := make([]*OverlayPortalController, tt.portals)
			for i := range controllers {
				controllers[i] = NewOverlayPortalController()
				if tt.showFirst {
					controllers[i].Show()
				}
			}
			show := true
			var setShow StateSetter
			var bc BuildContext
			entry := NewOverlayEntry(func(BuildContext) Widget {
				return NewStatefulBuilder(func(c BuildContext, setState StateSetter) Widget {
					setShow, bc = setState, c
					if !show {
						return NewSizedBox(nil)
					}
					children := make([]Widget, len(controllers))
					for i, c := range controllers {
						children[i] = NewOverlayPortal(c, emptyBox, NewSizedBox(nil))
					}
					return NewColumn(children)
				})
			})
			defer removeEntry(entry)
			ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
			for round := 0; round < 5; round++ {
				if n, want := OverlayChildModelLen(bc), map[bool]int{true: tt.portals, false: 0}[tt.showFirst && show]; n != want {
					t.Fatalf("round %d, shown: %d locations in the child model, want %d", round, n, want)
				}
				setShow(func() { show = false })
				ts.Pump()
				if n := OverlayChildModelLen(bc); n != 0 {
					t.Fatalf("round %d, removed: %d locations stay in the child model, want 0", round, n)
				}
				for _, c := range controllers {
					if tt.showFirst {
						c.Show()
					}
				}
				setShow(func() { show = true })
				ts.Pump()
			}
		})
	}
}

// TestLeaksScrolling scrolls a long list back and forth (step 29.21, SPEC §15 "Memory over
// time"): the live heap and the goroutines after N and after 4N steps.
func TestLeaksScrolling(t *testing.T) {
	tests := []struct {
		name string
		cfg  estimateConfig
	}{
		{"estimate item offsets", estimateConfig{estimate: true}},
		{"maintain visible content position", estimateConfig{estimate: true, keep: true}},
		{"both options, reverse", estimateConfig{estimate: true, keep: true, reverse: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaktest.Run(t, "widgets/scrolling/"+tt.name, func() func() {
				cfg := tt.cfg
				cfg.discard = true
				l := newEstimateList(t, cfg)
				down := true
				nextID := estimateItems
				return func() {
					// A jump far away and back, and a few notches of a wheel.
					if down {
						l.ctl.JumpTo(40000)
					} else {
						l.ctl.JumpTo(1000)
					}
					down = !down
					l.settle()
					if tt.cfg.keep {
						// Two new rows replace the first two, as a feed that gets new posts does.
						l.setIDs(func() { l.ids = append([]int{nextID, nextID + 1}, l.ids[2:]...); nextID += 2 })
						l.ts.Pump(16 * time.Millisecond)
					}
					l.ts.ScrollAt(ui.Offset{Dx: 100, Dy: 100}, ui.Offset{Dx: 0, Dy: 120})
					l.ts.Pump(16 * time.Millisecond)
					l.ts.ClearLogs()
				}
			}, leaktest.Opts{N: 100})
		})
	}
}

// TestLeaksWindows opens and closes windows on the fake platform (step 29.21, SPEC §15 "Memory
// over time"): a window with its view, its tree and its controller, and a dialog window.
func TestLeaksWindows(t *testing.T) {
	tests := []struct {
		name   string
		dialog bool
	}{
		{"regular window", false},
		{"dialog window", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaktest.Run(t, "widgets/windows/"+tt.name, func() func() {
				ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}, DiscardScenes: true})
				ts.PumpWidget(text("main"))
				bc := ts.Element(gluttertest.ByText("main"))
				return func() {
					var w Widget
					var destroy func()
					if tt.dialog {
						c, err := NewDialogWindowController(bc, ui.Size{Width: 300, Height: 200}, DialogWindowControllerOpts{Title: "Dialog"})
						if err != nil {
							t.Fatal(err)
						}
						w, destroy = NewDialogWindow(c, text("dialog")), c.Destroy
					} else {
						c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300}, RegularWindowControllerOpts{Title: "Second"})
						if err != nil {
							t.Fatal(err)
						}
						w, destroy = NewRegularWindow(c, text("second")), c.Destroy
					}
					withWindows(ts, text("main"), w)
					ts.Pump(16 * time.Millisecond)
					destroy()
					withWindows(ts, text("main"))
					ts.ClearLogs()
				}
			}, leaktest.Opts{N: 100})
		})
	}
}

// TestLeaksImagesAndFonts loads images and fonts while the app runs (step 29.21, SPEC §15 "Memory
// over time"): the image cache stays within its bounds, and a font that is loaded again is not
// kept twice.
func TestLeaksImagesAndFonts(t *testing.T) {
	const imageLimit = 20
	t.Run("new images, a small image cache", func(t *testing.T) {
		leaktest.Run(t, "widgets/images/new images", func() func() {
			ts := gluttertest.New(t, gluttertest.Opts{DiscardScenes: true})
			cache := ts.Binding.Painting().ImageCache()
			cache.SetMaximumSize(imageLimit)
			show := true
			var setShow StateSetter
			var data []byte
			ts.PumpWidget(ltr(NewStatefulBuilder(func(_ BuildContext, setState StateSetter) Widget {
				setShow = setState
				if !show {
					return NewSizedBox(nil)
				}
				return NewImageMemory(data)
			})))
			i := 0
			return func() {
				i++
				data = pngBytes(t, 16, 16, color.RGBA{R: byte(i), G: byte(i >> 8), B: 7, A: 255})
				setShow(func() { show = true })
				waitFor(t, ts, func() bool {
					return ts.Exists(gluttertest.ByType[*RawImage]()) && imageRender(ts) != nil && imageRender(ts).Image() != nil
				})
				ts.Pump()
				setShow(func() { show = false })
				ts.Pump()
				if n := cache.CurrentSize(); n > imageLimit {
					t.Fatalf("the image cache holds %d images, its maximum is %d", n, imageLimit)
				}
			}
		}, leaktest.Opts{N: 100, Warmup: 3 * imageLimit})
	})
	t.Run("the same font again", func(t *testing.T) {
		leaktest.Run(t, "widgets/fonts/the same font again", func() func() {
			ts := gluttertest.New(t, gluttertest.Opts{DiscardScenes: true})
			style := painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Go Mono"})
			ts.PumpWidget(ltr(NewText("monospace", TextOpts{Style: style})))
			return func() {
				if err := ts.Dispatcher().LoadFontFromList(gomono.TTF, "Go Mono"); err != nil {
					t.Fatal(err)
				}
				ts.Pump(16 * time.Millisecond)
			}
		}, leaktest.Opts{N: 200})
	})
}
