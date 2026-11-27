package desktop_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/internal/issues/trayrun"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// TestLinuxTellsWhetherTheDesktopShowsTrayIcons is the issue "The tray's availability" (step 60.1) on Linux: the
// reproduction of internal/issues (trayrun: TrayAvailable from a goroutine, then a SystemTray) on a real
// dbus-daemon of its own, with a StatusNotifierWatcher on it, without one, and with no session bus at all (the
// real dbus.Connect to an address where nobody listens). The answer agrees with the tray, and the error of the
// tray matches desktop.TagTrayUnavailable. Before the step there was no TrayAvailable and no exported tag, so
// the test did not build.
func TestLinuxTellsWhetherTheDesktopShowsTrayIcons(t *testing.T) {
	tests := []struct {
		platform string
		want     string
		setup    func(t *testing.T, bc widgets.BuildContext) *watcher
	}{
		{"linux-watcher", issues.TrayAnswerYes, func(t *testing.T, bc widgets.BuildContext) *watcher {
			addr := startPrivateBus(t)
			host := dialBus(t, addr)
			w := &watcher{}
			if err := host.Export(w, watcherPath, watcherDest); err != nil {
				t.Fatal(err)
			}
			if reply, err := host.RequestName(watcherDest, godbus.NameFlagDoNotQueue); err != nil || reply != godbus.RequestNameReplyPrimaryOwner {
				t.Fatalf("watcher name: %v, %v", reply, err)
			}
			desktop.UseBus(bc, dbus.NewBus(dialBus(t, addr)))
			return w
		}},
		{"linux-no-watcher", issues.TrayAnswerNo, func(t *testing.T, bc widgets.BuildContext) *watcher {
			desktop.UseBus(bc, dbus.NewBus(dialBus(t, startPrivateBus(t))))
			return nil
		}},
		{"linux-no-session-bus", issues.TrayAnswerNo, func(t *testing.T, bc widgets.BuildContext) *watcher {
			t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+filepath.Join(t.TempDir(), "nobody"))
			desktop.UseBusOpener(bc, dbus.Connect)
			return nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 300}})
			r := trayrun.New()
			ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, r.Wrap(widgets.NewText("main"))))
			w := tt.setup(t, ts.Element(gluttertest.ByText("main")))
			done := make(chan issues.TrayAvailableReport, 1)
			go func() { done <- r.Run(context.Background(), tt.platform, tt.want, 2*time.Second) }()
			var rep issues.TrayAvailableReport
			deadline := time.After(30 * time.Second)
			for running := true; running; {
				select {
				case rep = <-done:
					running = false
				case <-deadline:
					t.Fatal("the reproduction did not end")
				default:
					ts.FlushPosted()
					ts.Pump()
					if err := ts.TakeError(); err != nil {
						r.Report(err)
					}
					time.Sleep(time.Millisecond)
				}
			}
			t.Logf("%s", strings.Join(rep.Lines(), "\n"))
			for _, p := range rep.Problems() {
				t.Error(p)
			}
			if w != nil && rep.Tray == issues.TrayShown {
				waitFor(t, ts, "the registration", func() bool { return len(w.registered()) == 1 })
			}
		})
	}
}
