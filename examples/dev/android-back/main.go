// Command android-back is a development example (step 25.9): an app of three pages that logs what the
// framework hears of the activity's life and of the back button and the back gesture, so that the
// device tests can send the app Home, press Back and swipe back, and read what happened. It uses the
// public API only and runs on every platform.
//
// The home page pushes the second page, and the second page pushes a third page that cannot be popped
// (a PopScope with canPop false, which logs the attempt). GLUTTER_DEMO is a list of words separated by
// commas:
//
//	second   start on the second page
//	take     the observer of the example takes the back gesture, to hear its progress, commit and cancel
//	locked   start on the third page, whose pop is blocked (the history is /, /second, /second/locked)
//
// Run it on a device or the emulator with:
//
//	go run ./cmd/glutter-android run ./examples/dev/android-back
package main

import (
	"log"
	"log/slog"
	"os"
	"slices"
	"strings"

	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

func main() {
	err := m.RunApp(newApp(), m.RunAppOpts{
		AppID:  "dev.glutter.androidback",
		Window: m.WindowConfig{Title: "Android back", Size: m.Size{Width: 480, Height: 640}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func newApp() m.Widget {
	var initial string
	for _, word := range strings.Split(os.Getenv("GLUTTER_DEMO"), ",") {
		switch strings.TrimSpace(word) {
		case "second":
			initial = "/second"
		case "locked":
			initial = "/second/locked"
		}
	}
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Android back",
		DebugShowCheckedModeBanner: m.Some(false),
		InitialRoute:               initial,
		OnGenerateRoute:            func(settings m.RouteSettings) m.Route { return route(settings.Name()) },
		NavigatorObservers:         []m.NavigatorObserver{&navLog{}},
	})
}

// observed is the home page, which stays under the other pages for the life of the app and so can log
// the lifecycle and the back events.
type observed struct {
	m.StatefulWidget
}

func (o *observed) CreateState() m.State { return &observedState{} }

type observedState struct {
	m.StateOf[*observed]
	binding  *m.WidgetsBinding
	listener *m.AppLifecycleListener
	observer *backObserver
}

// backObserver logs the back button and the steps of a back gesture, and takes the gesture so that the
// framework's own handlers are not asked.
type backObserver struct {
	m.WidgetsBindingObserverBase
	// take is whether the observer takes the gesture (the demo word take), so that it hears the progress,
	// the commit and the cancel: the navigator's route takes it as well and pops on the commit.
	take bool
}

func (backObserver) DidPopRoute() bool {
	slog.Info("back", "event", "didPopRoute")
	return false
}

func (o backObserver) HandleStartBackGesture(e services.PredictiveBackEvent) bool {
	slog.Info("back", "event", "start", "progress", e.Progress(), "edge", e.SwipeEdge().String())
	return o.take
}

func (backObserver) HandleUpdateBackGestureProgress(e services.PredictiveBackEvent) {
	slog.Info("back", "event", "update", "progress", e.Progress(), "edge", e.SwipeEdge().String())
}

func (backObserver) HandleCommitBackGesture() { slog.Info("back", "event", "commit") }

func (backObserver) HandleCancelBackGesture() { slog.Info("back", "event", "cancel") }

// navLog logs the changes of the history of the navigator.
type navLog struct {
	m.NavigatorObserverBase
}

func (navLog) DidPush(route, _ m.Route) {
	slog.Info("navigator", "event", "push", "route", route.Settings().Name())
}
func (navLog) DidPop(route, _ m.Route) {
	slog.Info("navigator", "event", "pop", "route", route.Settings().Name())
}

func (s *observedState) InitState() {
	s.binding = m.WidgetsBindingOf(s.Context())
	s.listener = m.NewAppLifecycleListener(s.binding, m.AppLifecycleListenerOpts{
		OnStateChange: func(state ui.AppLifecycleState) { slog.Info("lifecycle", "state", state.String()) },
	})
	s.observer = &backObserver{take: slices.Contains(strings.Split(os.Getenv("GLUTTER_DEMO"), ","), "take")}
	s.binding.AddObserver(s.observer)
}

func (s *observedState) Dispose() {
	s.binding.RemoveObserver(s.observer)
	s.listener.Dispose()
}

func (s *observedState) Build(bc m.BuildContext) m.Widget {
	return page(bc, "Home", "/second", "Push the second page")
}

// route makes the page of a name.
func route(name string) m.Route {
	return m.NewMaterialPageRoute(func(bc m.BuildContext) m.Widget {
		switch name {
		case "/second":
			return page(bc, "Second", "/second/locked", "Push the page that cannot be popped")
		case "/second/locked":
			return m.NewPopScope(page(bc, "Locked", "", ""), m.PopScopeOpts{
				CanPop: m.Some(false),
				OnPopInvokedWithResult: func(didPop bool, _ any) {
					slog.Info("pop scope", "page", "locked", "didPop", didPop)
				},
			})
		}
		return &observed{StatefulWidget: m.NewStatefulWidgetBase()}
	}, m.MaterialPageRouteOpts{PageRouteOpts: m.PageRouteOpts{RouteOpts: m.RouteOpts{Settings: m.NewRouteSettings(m.RouteSettingsOpts{Name: name})}}})
}

func page(bc m.BuildContext, title, next, label string) m.Widget {
	slog.Info("page", "name", title)
	var body m.Widget = m.NewCenter(m.NewText(title))
	if next != "" {
		body = m.NewCenter(m.NewElevatedButton(func() {
			m.NavigatorPush(bc, route(next))
		}, m.NewText(label)))
	}
	// A page that is pushed has its own color, so that the page under it shows when the back gesture
	// shrinks it.
	var background m.Opt[m.Color]
	if title != "Home" {
		background = m.Some(m.Colors.Teal.Shade100())
	}
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar:          m.NewAppBar(m.AppBarOpts{Title: m.NewText(title)}),
		Body:            body,
		BackgroundColor: background,
	})
}
