// Command android-system is a development example (step 25.7): a Material page that logs what the
// framework knows of the system on every build (the insets, the brightness, the text scale, the
// clock, the animations, the accessibility settings), and does what the variable GLUTTER_DEMO asks
// of the system's interface once it has started, so that the device tests can see the effect. It
// uses the public API only and runs on every platform.
//
// GLUTTER_DEMO is a list of words separated by commas:
//
//	landscape, portrait    SystemChrome.setPreferredOrientations
//	immersive, leanback    SystemChrome.setEnabledSystemUIMode, the bars hidden
//	edge                   the mode edgeToEdge, edge-late the same three seconds later
//	style-dark             an overlay style with dark icons on a light bar, style-light the other
//	haptic, sound          HapticFeedback and SystemSound, one of each
//	keepon                 the screen stays on (the window's KeepScreenOn)
//
// Run it on a device or the emulator with:
//
//	go run ./cmd/glutter-android run ./examples/dev/android-system
package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

func main() {
	err := m.RunApp(newApp(), m.RunAppOpts{
		AppID:  "dev.glutter.androidsystem",
		Window: m.WindowConfig{Title: "Android system", Size: m.Size{Width: 480, Height: 640}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func newApp() m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Android system",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       &page{StatefulWidget: m.NewStatefulWidgetBase()},
	})
}

type page struct{ m.StatefulWidget }

func (p *page) CreateState() m.State { return &pageState{} }

type pageState struct{ m.StateOf[*page] }

func (s *pageState) InitState() {
	b := m.WidgetsBindingOf(s.Context())
	b.SystemChrome().SetSystemUIChangeCallback(func(visible bool) { slog.Info("systemui", "visible", visible) })
	for _, word := range strings.Split(os.Getenv("GLUTTER_DEMO"), ",") {
		switch strings.TrimSpace(word) {
		case "landscape":
			b.SystemChrome().SetPreferredOrientations([]services.DeviceOrientation{
				services.DeviceOrientationLandscapeLeft, services.DeviceOrientationLandscapeRight})
		case "portrait":
			b.SystemChrome().SetPreferredOrientations([]services.DeviceOrientation{services.DeviceOrientationPortraitUp})
		case "immersive":
			b.SystemChrome().SetEnabledSystemUIMode(services.SystemUiModeImmersiveSticky, nil)
		case "leanback":
			b.SystemChrome().SetEnabledSystemUIMode(services.SystemUiModeLeanBack, nil)
		case "edge":
			b.SystemChrome().SetEnabledSystemUIMode(services.SystemUiModeEdgeToEdge, nil)
		case "edge-late":
			// Three seconds later: the bars come back, and the callback hears of it.
			time.AfterFunc(3*time.Second, func() {
				b.PlatformDispatcher().Post(func() {
					b.SystemChrome().SetEnabledSystemUIMode(services.SystemUiModeEdgeToEdge, nil)
				})
			})
		case "style-dark":
			b.SystemChrome().SetSystemUIOverlayStyle(services.SystemUiOverlayStyleDark)
		case "style-light":
			b.SystemChrome().SetSystemUIOverlayStyle(services.SystemUiOverlayStyleLight)
		case "haptic":
			b.HapticFeedback().HeavyImpact()
		case "keepon":
			if err := b.PlatformDispatcher().KeepScreenOn(ui.ImplicitViewID, true); err != nil {
				slog.Error("the screen cannot be kept on", "err", err)
			}
		case "sound":
			b.SystemSound().Play(services.SystemSoundTypeClick)
		}
	}
}

func (s *pageState) Build(bc m.BuildContext) m.Widget {
	mq := m.MediaQueryOf(bc)
	edges := func(e m.EdgeInsets) string {
		return fmt.Sprintf("%.3f,%.3f,%.3f,%.3f", e.Left, e.Top, e.Right, e.Bottom)
	}
	locale := ""
	if l, ok := m.LocalizationsMaybeLocaleOf(bc); ok {
		locale = l.String()
	}
	slog.Info("mq",
		"size", fmt.Sprintf("%.0fx%.0f", mq.Size.Width, mq.Size.Height),
		"dpr", mq.DevicePixelRatio,
		"padding", edges(mq.Padding), "viewPadding", edges(mq.ViewPadding), "viewInsets", edges(mq.ViewInsets),
		"gestures", edges(mq.SystemGestureInsets),
		"brightness", mq.PlatformBrightness.String(),
		"textScale", mq.TextScaler.TextScaleFactor(),
		"scale10", mq.TextScaler.Scale(10), "scale100", mq.TextScaler.Scale(100),
		"h24", mq.AlwaysUse24HourFormat, "animationsOff", mq.DisableAnimations,
		"bold", mq.BoldText, "highContrast", mq.HighContrast, "locale", locale)
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("Android system")}),
		Body:   m.NewCenter(m.NewText(fmt.Sprintf("top %.0f", mq.Padding.Top))),
	})
}
