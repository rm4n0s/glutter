package widgets_test

import (
	"sync"
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/widgets"
)

// Route tests of the failure tags of the Router and its helpers, and tests of
// Future.

type unimplementedParser struct {
	RouteInformationParserBase[*routeInfo]
}

func TestRouterTags(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("RouteInformationURIMissing", func(t *testing.T) {
		requireTag(t, "RouteInformationURIMissing", func() { NewRouteInformation(RouteInformationOpts{}) })
	})
	t.Run("RouteInformationParserUnimplemented", func(t *testing.T) {
		p := &unimplementedParser{}
		InitRouteInformationParserBase[*routeInfo](&p.RouteInformationParserBase, nil)
		requireTag(t, "RouteInformationParserUnimplemented", func() { p.ParseRouteInformation(uriInfo("a")) })
	})
	t.Run("RouteInformationProviderNoListeners", func(t *testing.T) {
		ts := gluttertest.New(t)
		provider := NewPlatformRouteInformationProvider(ts.Binding, uriInfo("initial"))
		requireTag(t, "RouteInformationProviderNoListeners", func() { provider.DidPushRouteInformation(uriInfo("a")) })
	})
	t.Run("BackButtonDispatcherEmpty", func(t *testing.T) {
		ts := gluttertest.New(t)
		root := NewRootBackButtonDispatcher(ts.Binding)
		requireTag(t, "BackButtonDispatcherEmpty", func() { root.DeferTo(NewChildBackButtonDispatcher(root)) })
	})
	t.Run("ChildBackButtonDispatcherEmpty", func(t *testing.T) {
		ts := gluttertest.New(t)
		root := NewRootBackButtonDispatcher(ts.Binding)
		root.AddCallback(func() bool { return false })
		child := NewChildBackButtonDispatcher(root)
		requireTag(t, "ChildBackButtonDispatcherEmpty", func() { child.DeferTo(NewChildBackButtonDispatcher(child)) })
	})
	t.Run("BackButtonDispatcherCallbacksInvalid", func(t *testing.T) {
		ts := gluttertest.New(t)
		root := NewRootBackButtonDispatcher(ts.Binding)
		root.AddCallback(func() bool { return true })
		root.AddCallback(func() bool { return true })
		if root.InvokeCallback(false) {
			t.Error("a dispatcher with two callbacks should return the default")
		}
		_ = requireErrorTag(t, takeFirst(t, ts), "BackButtonDispatcherCallbacksInvalid")
	})
	t.Run("BackButtonDispatcherMissing", func(t *testing.T) {
		ts := gluttertest.New(t)
		delegate := newSimpleDelegate(func(BuildContext, *routeInfo) Widget {
			return NewBackButtonListener(func() bool { return true }, NewContainer(nil))
		}, nil)
		ts.PumpWidget(routerOf(delegate, RouterOpts[*routeInfo]{}))
		_ = requireErrorTag(t, takeFirst(t, ts), "BackButtonDispatcherMissing")
	})
	t.Run("RouterScopeMissing", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(ltr(NewText("x")))
		bc := ts.Element(gluttertest.ByText("x"))
		requireTag(t, "RouterScopeMissing", func() { RouterNavigate(bc, func() {}) })
		requireTag(t, "RouterScopeMissing", func() { RouterNeglect(bc, func() {}) })
	})
}

func TestFuture(t *testing.T) {
	var posted []func()
	post := func(fn func()) { posted = append(posted, fn) }
	run := func() {
		for len(posted) > 0 {
			fn := posted[0]
			posted = posted[1:]
			fn()
		}
	}
	t.Run("synchronous runs at once", func(t *testing.T) {
		got := 0
		NewSynchronousFuture(3).Then(post, func(v int) { got = v })
		if got != 3 || len(posted) != 0 {
			t.Errorf("got %d with %d posted", got, len(posted))
		}
	})
	t.Run("value runs in a later turn", func(t *testing.T) {
		got := 0
		NewFutureValue(4).Then(post, func(v int) { got = v })
		if got != 0 {
			t.Fatal("the callback ran at once")
		}
		run()
		if got != 4 {
			t.Errorf("got %d", got)
		}
	})
	t.Run("pending runs on completion, once", func(t *testing.T) {
		f := NewFuture[string]()
		var got []string
		f.Then(post, func(v string) { got = append(got, v) })
		f.Then(post, func(v string) { got = append(got, v+"!") })
		if f.IsCompleted() {
			t.Fatal("completed too early")
		}
		f.Complete("a")
		f.Complete("b")
		run()
		if len(got) != 2 || got[0] != "a" || got[1] != "a!" || !f.IsCompleted() {
			t.Errorf("got %v", got)
		}
	})
	t.Run("map keeps the kind", func(t *testing.T) {
		sync := FutureMap(NewSynchronousFuture(2), post, func(v int) int { return v * 2 })
		got := 0
		sync.Then(post, func(v int) { got = v })
		if got != 4 {
			t.Errorf("synchronous map gave %d", got)
		}
		f := NewFuture[int]()
		mapped := FutureMap(f, post, func(v int) string { return "n" })
		var s string
		mapped.Then(post, func(v string) { s = v })
		f.Complete(1)
		run()
		if s != "n" {
			t.Errorf("pending map gave %q", s)
		}
	})
	t.Run("completes from another goroutine", func(t *testing.T) {
		f := NewFuture[int]()
		var wg sync.WaitGroup
		wg.Add(1)
		go func() { defer wg.Done(); f.Complete(7) }()
		wg.Wait()
		got := 0
		f.Then(post, func(v int) { got = v })
		run()
		if got != 7 {
			t.Errorf("got %d", got)
		}
	})
}
