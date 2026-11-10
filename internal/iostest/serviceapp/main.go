//go:build ios

// Command serviceapp is the app of the Simulator tests of step 35.6: it runs the iOS platform with a handler of its own and, after the
// first frame, the steps that the environment gives it (serviceslines.StepsEnv): the system UI, the clipboard, the spell check, the
// haptics and the sounds, opening a URI and a notification. Each step prints a line that the test reads.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/iostest/serviceslines"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/ios"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
	"github.com/rm4n0s/glutter/ui"
)

type handler struct {
	platform.NopHandler
	p       *ios.Platform
	w       platform.Window
	started bool
	// pending counts the steps that finish on a goroutine of their own (open, notify, paste): the last line waits for them.
	pending sync.WaitGroup
}

// OnSettings prints the settings that the system changed to.
func (h *handler) OnSettings(s platform.Settings) { h.printSettings(s) }

func (h *handler) printSettings(s platform.Settings) {
	brightness := "light"
	if s.PlatformBrightness == ui.BrightnessDark {
		brightness = "dark"
	}
	font17 := 17.0
	if s.ScaleFontSize != nil {
		font17 = s.ScaleFontSize(17)
	}
	locale := ""
	if len(s.Locales) > 0 {
		locale = s.Locales[0].String()
	}
	fmt.Printf("%s brightness=%s scale=%.4f font17=%.2f bold=%v motion=%v contrast=%v voiceover=%v h24=%v locale=%s\n", serviceslines.Settings,
		brightness, s.TextScaleFactor, font17, s.BoldText, s.DisableAnimations, s.HighContrast, s.AccessibleNavigation, s.AlwaysUse24HourFormat, locale)
}

func (h *handler) OnFrame(_ ui.ViewID, _ time.Duration) {
	if h.started {
		return
	}
	h.started = true
	steps := strings.Split(os.Getenv(serviceslines.StepsEnv), ";")
	h.printSettings(h.p.Settings())
	fmt.Println(serviceslines.Ready)
	go func() {
		for _, s := range steps {
			time.Sleep(400 * time.Millisecond)
			if s = strings.TrimSpace(s); s != "" {
				h.p.Post(func() { h.run(s) })
			}
		}
		time.Sleep(400 * time.Millisecond)
		h.pending.Wait()
		fmt.Println(serviceslines.Done)
	}()
}

// tagOf is the tag of the error, "nil" for none.
func tagOf(err error) string {
	if err == nil {
		return "nil"
	}
	if appErr, ok := errors.FromError(err); ok {
		return appErr.Tag
	}
	return "untagged:" + err.Error()
}

func say(step, result string) { fmt.Printf("%s %s %s\n", serviceslines.Step, step, result) }

// run does one step on the UI thread.
func (h *handler) run(step string) {
	name, arg, _ := strings.Cut(step, ":")
	chrome, _ := h.w.(ui.SystemChromeHost)
	switch name {
	case "style":
		b := ui.BrightnessLight
		if arg == "dark" {
			b = ui.BrightnessDark
		}
		chrome.SetSystemUIOverlayStyle(ui.SystemUIOverlayStyle{StatusBarBrightness: b})
		say(step, "done")
	case "mode":
		switch arg {
		case "edge":
			chrome.SetEnabledSystemUIMode(ui.SystemUIModeEdgeToEdge, nil)
		case "hidden":
			chrome.SetEnabledSystemUIMode(ui.SystemUIModeImmersiveSticky, nil)
		case "top":
			chrome.SetEnabledSystemUIMode(ui.SystemUIModeManual, []ui.SystemUIOverlay{ui.SystemUIOverlayTop})
		case "bottom":
			chrome.SetEnabledSystemUIMode(ui.SystemUIModeManual, []ui.SystemUIOverlay{ui.SystemUIOverlayBottom})
		}
		say(step, "done")
	case "orient":
		switch arg {
		case "portrait":
			chrome.SetPreferredOrientations([]ui.DeviceOrientation{ui.DeviceOrientationPortraitUp})
		case "landscape":
			chrome.SetPreferredOrientations([]ui.DeviceOrientation{ui.DeviceOrientationLandscapeLeft, ui.DeviceOrientationLandscapeRight})
		default:
			chrome.SetPreferredOrientations(nil)
		}
		say(step, "done")
	case "settings":
		h.printSettings(h.p.Settings())
		say(step, "done")
	case "report":
		h.p.ReportChrome()
	case "copy":
		h.pending.Add(1)
		h.p.Clipboard().SetText(context.Background(), arg, func(err error) { defer h.pending.Done(); say(step, "err="+tagOf(err)) })
	case "paste":
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		h.pending.Add(1)
		h.p.Clipboard().GetText(ctx, func(text string, ok bool, err error) {
			defer h.pending.Done()
			cancel()
			say(step, fmt.Sprintf("ok=%v text=%q err=%s", ok, text, tagOf(err)))
		})
	case "spell":
		locale, text, _ := strings.Cut(arg, ":")
		sc, ok := h.w.(ui.SpellCheckHost)
		if !ok {
			say(step, "err=nohost")
			return
		}
		h.pending.Add(1)
		sc.SpellCheck(context.Background(), locale, text, func(spans []ui.SpellCheckSpan, err error) {
			defer h.pending.Done()
			var parts []string
			for _, s := range spans {
				first := ""
				if len(s.Suggestions) > 0 {
					first = s.Suggestions[0]
				}
				parts = append(parts, fmt.Sprintf("%d,%d,%s", s.Start, s.End, first))
			}
			say(step, fmt.Sprintf("spans=%s err=%s", strings.Join(parts, ";"), tagOf(err)))
		})
	case "haptic":
		for _, k := range []ui.HapticFeedbackKind{ui.HapticFeedbackLightImpact, ui.HapticFeedbackMediumImpact, ui.HapticFeedbackHeavyImpact,
			ui.HapticFeedbackSelectionClick, ui.HapticFeedbackVibrate} {
			chrome.PerformHapticFeedback(k)
		}
		say(step, "done")
	case "sound":
		for _, k := range []ui.SystemSoundKind{ui.SystemSoundClick, ui.SystemSoundTick, ui.SystemSoundAlert} {
			chrome.PlaySystemSound(k)
		}
		say(step, "done")
	case "open":
		h.pending.Add(1)
		go func() {
			defer h.pending.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			say(step, "err="+tagOf(h.p.Desktop().OpenURI(ctx, arg)))
		}()
	case "notify":
		h.pending.Add(1)
		go func() {
			defer h.pending.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_, err := h.p.Desktop().ShowNotification(ctx, mobile.NotificationSpec{Title: "serviceapp", Body: "a notification"})
			say(step, "err="+tagOf(err))
		}()
	default:
		say(step, "unknown")
	}
}

func main() {
	p, err := ios.New(ios.Options{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "serviceapp:", err)
		os.Exit(1)
	}
	w, err := p.NewWindow(platform.WindowConfig{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "serviceapp:", err)
		os.Exit(1)
	}
	go func() {
		for range time.Tick(100 * time.Millisecond) {
			p.Post(w.RequestFrame)
		}
	}()
	h := &handler{p: p, w: w}
	if err := p.Run(context.Background(), h); err != nil {
		fmt.Fprintln(os.Stderr, "serviceapp:", err)
		os.Exit(1)
	}
}
