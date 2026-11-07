// Command android-sensors is a development example (step 31.7): a Material page that opens the accelerometer and the gyroscope
// through package sensors, shows their latest readings, and logs them twice a second. The device tests start it on the emulator,
// set the emulator's virtual sensors (adb emu sensor set), and check that the readings follow with the right values and axes; on a
// phone it is the manual check (tilt the phone and the numbers move, and the arrow of the page turns with it). In a browser on a
// phone (go run ./cmd/glutter-web serve ./examples/dev/android-sensors) it shows the same, and a tap or a long press vibrates the phone. It uses the public API only.
//
// The log lines (the default logger's, so "INFO " comes first):
//
//	sensors open kind=<accelerometer|gyroscope> rate=<n>
//	sensors failed kind=<kind> err=<tag>
//	sensors accelerometer x=<n> y=<n> z=<n> readings=<n> frames=<n> dropped=<n>
//	sensors gyroscope x=<n> y=<n> z=<n> readings=<n> frames=<n> dropped=<n>
//
// where readings is how many readings the listener was given and frames how many calls (batches) it took.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/sensors"
)

func main() {
	err := m.RunApp(newApp(), m.RunAppOpts{
		AppID:  "dev.glutter.androidsensors",
		Window: m.WindowConfig{Title: "Sensors", Size: m.Size{Width: 480, Height: 640}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func newApp() m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Sensors",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       &page{StatefulWidget: m.NewStatefulWidgetBase()},
	})
}

type page struct{ m.StatefulWidget }

func (p *page) CreateState() m.State { return &pageState{} }

// reading is what the page keeps of a sensor.
type reading struct {
	values   [3]float64
	has      bool
	readings int
	frames   int
}

type pageState struct {
	m.StateOf[*page]
	once sync.Once
	mu   sync.Mutex
	last [2]reading // indexed by sensors.Kind
	open [2]*sensors.Sensor
	stop chan struct{}
}

func (s *pageState) DidChangeDependencies() { s.once.Do(func() { s.start() }) }

func (s *pageState) Dispose() {
	if s.stop != nil {
		close(s.stop)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, x := range s.open {
		if x != nil {
			if err := x.Close(); err != nil {
				slog.Error("sensors close failed", "err", tagOf(err))
			}
		}
	}
}

func tagOf(err error) string {
	if appErr, ok := errors.FromError(err); ok {
		return appErr.Tag
	}
	return "none"
}

func (s *pageState) start() {
	s.stop = make(chan struct{})
	sys := sensors.Of(s.Context())
	for _, kind := range []sensors.Kind{sensors.Accelerometer, sensors.Gyroscope} {
		// Open may wait for the user's answer (a browser on iOS), so it is not done on the UI goroutine.
		go func() {
			x, err := sys.Open(context.Background(), kind)
			if err != nil {
				slog.Error("sensors failed", "kind", kind.String(), "err", tagOf(err))
				return
			}
			slog.Info("sensors open", "kind", kind.String(), "rate", x.Rate())
			s.mu.Lock()
			s.open[kind] = x
			s.mu.Unlock()
			x.AddListener(func(batch []sensors.Reading) {
				s.mu.Lock()
				l := &s.last[kind]
				l.values, l.has = batch[len(batch)-1].Values, true
				l.readings += len(batch)
				l.frames++
				s.mu.Unlock()
				s.SetState(func() {})
			})
		}()
	}
	go func() {
		tick := time.NewTicker(500 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-tick.C:
			}
			s.mu.Lock()
			for _, kind := range []sensors.Kind{sensors.Accelerometer, sensors.Gyroscope} {
				l := s.last[kind]
				if !l.has {
					continue
				}
				dropped := 0
				if x := s.open[kind]; x != nil {
					dropped = x.Dropped()
				}
				slog.Info("sensors "+kind.String(), "x", f(l.values[0]), "y", f(l.values[1]), "z", f(l.values[2]),
					"readings", l.readings, "frames", l.frames, "dropped", dropped)
			}
			s.mu.Unlock()
		}
	}()
}

// f formats a value for the log, with two decimals.
func f(v float64) string { return fmt.Sprintf("%.2f", v) }

func (s *pageState) Build(bc m.BuildContext) m.Widget {
	s.mu.Lock()
	a, g := s.last[sensors.Accelerometer], s.last[sensors.Gyroscope]
	s.mu.Unlock()
	line := func(name string, r reading) m.Widget {
		if !r.has {
			return m.NewText(name + ": no reading yet")
		}
		return m.NewText(fmt.Sprintf("%s: x %.2f  y %.2f  z %.2f", name, r.values[0], r.values[1], r.values[2]))
	}
	// The arrow points where gravity pulls on the screen: a phone held upright points it down.
	angle := 0.0
	if a.has {
		angle = math.Atan2(a.values[0], a.values[1])
	}
	haptic := m.WidgetsBindingOf(bc).HapticFeedback()
	// A tap is a heavy impact and a long press a light one, to try the vibration (a phone's browser has navigator.vibrate).
	return m.NewScaffold(m.ScaffoldOpts{
		Body: m.NewGestureDetector(m.NewCenter(m.NewColumn([]m.Widget{
			line("accelerometer (m/s²)", a),
			line("gyroscope (rad/s)", g),
			m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(24.0)}),
			m.NewTransformRotate(angle, m.NewIcon(m.Icons.ArrowDownward(), m.IconOpts{Size: m.Some(96.0)})),
		}, m.ColumnOpts{MainAxisAlignment: m.MainAxisAlignmentCenter})), m.GestureDetectorOpts{
			OnTap:       haptic.HeavyImpact,
			OnLongPress: haptic.LightImpact,
		}),
	})
}
