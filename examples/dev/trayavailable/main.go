// Command trayavailable runs the reproduction of the issue "The tray's availability" (step 60.1,
// internal/issues TrayAvailableName) in a real app: it asks desktop.TrayAvailable from a goroutine,
// places a SystemTray, waits for what the tray reports, prints the report's line and
// issues.DoneLine, and exits. The Wine test (examples, TestWineTellsWhetherTheDesktopShowsTrayIcons)
// and the emulator test (internal/androidtest, TestAndroidTellsWhetherTheDesktopShowsTrayIcons) run
// it and read the line from its output and its log. It is a development example: it uses
// internal/issues.
//
// Run it with: go run ./examples/dev/trayavailable
//
// The report names the platform (the target platform) and what its desktop is expected to answer:
// yes on Windows and macOS, no on Android and iOS; on Linux it depends on the desktop, and nothing is
// expected.
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/internal/issues/trayrun"
	m "github.com/rm4n0s/glutter/material"
)

func main() {
	r := trayrun.New()
	logger := m.DefaultErrorReporter(nil)
	platform, want := expected()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		rep := r.Run(ctx, platform, want, 5*time.Second)
		for _, line := range rep.Lines() {
			log.Print(line)
		}
		for _, p := range rep.Problems() {
			log.Print(issues.TrayAvailableName + ": problem: " + p)
		}
		log.Print(issues.DoneLine)
		os.Exit(0)
	}()
	err := m.RunApp(page(r), m.RunAppOpts{
		AppID:  "dev.glutter.trayavailable",
		Window: m.WindowConfig{Title: "Tray available", Size: m.Size{Width: 360, Height: 200}},
		ErrorReporter: func(err error) {
			r.Report(err)
			logger(err)
		},
	})
	if err != nil {
		log.Print(issues.FailedLine)
		log.Fatal(err)
	}
}

// expected names the platform and what its desktop is expected to answer.
func expected() (platform, want string) {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformWindows:
		return "windows", issues.TrayAnswerYes
	case foundation.TargetPlatformMacOS:
		return "macos", issues.TrayAnswerYes
	case foundation.TargetPlatformAndroid:
		return "android", issues.TrayAnswerNo
	case foundation.TargetPlatformIOS:
		return "ios", issues.TrayAnswerNo
	default:
		return "linux", ""
	}
}

func page(r *trayrun.Runner) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		DebugShowCheckedModeBanner: m.Some(false),
		Home: m.NewScaffold(m.ScaffoldOpts{
			Body: r.Wrap(m.NewCenter(m.NewText("Does the desktop show tray icons?"))),
		}),
	})
}
