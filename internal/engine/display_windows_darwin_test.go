//go:build darwin && !ios && glutter_display

package engine_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/mainthread"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// dialogRoot is the root of an app with a WindowManager (as examples/multiwindow has) whose page opens a window and
// a dialog, modal to the page, once it is on the screen.
type dialogRoot struct{ m.StatefulWidget }

func (r *dialogRoot) CreateState() m.State { return &dialogRootState{} }

type dialogRootState struct{ m.StateOf[*dialogRoot] }

func (s *dialogRootState) Build(bc m.BuildContext) m.Widget {
	b := m.WidgetsBindingOf(bc)
	return m.NewWindowManager(b.WrapWithDefaultView(b.WrapWithWindowScope(m.NewMaterialApp(m.MaterialAppOpts{Home: &dialogPage{StatefulWidget: m.NewStatefulWidgetBase()}}))))
}

type dialogPage struct{ m.StatefulWidget }

func (p *dialogPage) CreateState() m.State { return &dialogPageState{} }

type dialogPageState struct{ m.StateOf[*dialogPage] }

func (s *dialogPageState) InitState() {
	m.WidgetsBindingOf(s.Context()).RunPostFrame(func(time.Duration) {
		bc := s.Context()
		registry := m.WindowRegistryOf(bc)
		regular, err := m.NewRegularWindowController(bc, m.Size{Width: 240, Height: 160}, m.RegularWindowControllerOpts{Title: "Other"})
		if err == nil {
			registry.Register(m.NewWindowEntry(regular, func(m.BuildContext) m.Widget { return m.NewText("other") }))
		}
		dialog, err := m.NewDialogWindowController(bc, m.Size{Width: 240, Height: 160}, m.DialogWindowControllerOpts{Parent: m.WindowScopeOf(bc), Title: "Dialog"})
		if err == nil {
			registry.Register(m.NewWindowEntry(dialog, func(m.BuildContext) m.Widget { return m.NewText("dialog") }))
		}
	})
}

func (s *dialogPageState) Build(m.BuildContext) m.Widget { return m.NewText("page") }

// appWindows counts the windows of the app and those that cannot become the key window: the ones that a modal window
// covers. It only reads, from the goroutine of the test.
func appWindows() (all, blocked int) {
	list := objc.ID(objcx.MustClass("NSApplication")).Send(objcx.Sel("sharedApplication")).Send(objcx.Sel("windows"))
	n := int(objc.Send[uint64](list, objcx.Sel("count")))
	for i := 0; i < n; i++ {
		w := list.Send(objcx.Sel("objectAtIndex:"), uint64(i))
		if objcx.String(w.Send(objcx.Sel("title"))) == "" {
			continue // a window of the system (the input method's), not one of the app
		}
		all++
		if !objc.Send[bool](w, objcx.Sel("canBecomeKeyWindow")) {
			blocked++
		}
	}
	return all, blocked
}

// TestDisplayDialogFromTheFrameworkIsModal runs an app that opens a window and a dialog, as the framework's window
// controllers do on macOS: three windows exist, and the one under the dialog is the only one that is blocked
// (SPEC §29.4).
func TestDisplayDialogFromTheFrameworkIsModal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	var all, blocked int
	go func() {
		defer cancel()
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			time.Sleep(50 * time.Millisecond)
			a, b := appWindows()
			mu.Lock()
			all, blocked = a, b
			mu.Unlock()
			if a == 3 && b == 1 {
				return
			}
		}
	}()
	var err error
	mainthread.Do(func() {
		err = m.RunWidgetContext(ctx, &dialogRoot{m.NewStatefulWidgetBase()}, widgets.RunAppOpts{Window: widgets.WindowConfig{Title: "dialogs", Size: ui.Size{Width: 300, Height: 200}}})
	})
	if err != nil {
		t.Fatalf("RunWidget: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if all != 3 {
		t.Errorf("the app has %d windows, want 3 (the first, another one and the dialog)", all)
	}
	if blocked != 1 {
		t.Errorf("%d windows are blocked, want 1 (the first, under the dialog)", blocked)
	}
}
