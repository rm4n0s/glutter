package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/widgets"
)

// Route tests of the failure tags of the Pages API: the transition delegate
// checks, the decisions of the route entries, and the pages themselves.

// funcDelegate is a transition delegate that runs a function.
type funcDelegate func(newHistory []RouteTransitionRecord, exiting map[RouteTransitionRecord]RouteTransitionRecord, pageless map[RouteTransitionRecord][]RouteTransitionRecord) []RouteTransitionRecord

func (f funcDelegate) Resolve(n []RouteTransitionRecord, e map[RouteTransitionRecord]RouteTransitionRecord, p map[RouteTransitionRecord][]RouteTransitionRecord) []RouteTransitionRecord {
	return f(n, e, p)
}

// afterDefault runs the default delegate and changes its result.
func afterDefault(change func([]RouteTransitionRecord) []RouteTransitionRecord) funcDelegate {
	return func(n []RouteTransitionRecord, e map[RouteTransitionRecord]RouteTransitionRecord, p map[RouteTransitionRecord][]RouteTransitionRecord) []RouteTransitionRecord {
		return change(DefaultTransitionDelegate{}.Resolve(n, e, p))
	}
}

// updateTo pumps the first pages, then the second ones, and returns the first
// error that the second update reported.
func updateTo(t *testing.T, delegate TransitionDelegate, first, second []Page) error {
	t.Helper()
	ts := gluttertest.New(t)
	onPop := func(Route, any) bool { return false }
	app := func(pages []Page) Widget {
		return pagesApp(pagesAppOpts{pages: pages, onPopPage: onPop, delegate: delegate})
	}
	ts.PumpWidget(app(first))
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Fatalf("the first pages reported %v", errs)
	}
	ts.PumpWidget(app(second))
	return takeFirst(t, ts)
}

func pagesNamed(names ...string) []Page {
	var out []Page
	for _, n := range names {
		out = append(out, newTestPage(n, n))
	}
	return out
}

// stubbornRoute is a route that handles pops internally forever.
type stubbornRoute struct {
	*navTestRoute
	popResult bool
}

func (r *stubbornRoute) WillHandlePopInternally() bool { return true }
func (r *stubbornRoute) DidPop(any) bool               { return r.popResult }

func stubbornPage(key string, popResult bool) *testPage {
	return newPage(key, key, func(p *testPage) Route {
		return &stubbornRoute{newNavTestRoute(navTestRouteOpts{builder: func(BuildContext) Widget { return NewText(key) }, settings: p}), popResult}
	})
}

func TestTransitionDelegateTags(t *testing.T) {
	skipUnlessDebug(t)
	one, oneTwo, two := pagesNamed("1"), pagesNamed("1", "2"), pagesNamed("2")
	tests := []struct {
		name          string
		tag           string
		delegate      TransitionDelegate
		first, second []Page
	}{
		{"exiting route not marked", "TransitionDelegateExitingUnmarked",
			funcDelegate(func(n []RouteTransitionRecord, e map[RouteTransitionRecord]RouteTransitionRecord, _ map[RouteTransitionRecord][]RouteTransitionRecord) []RouteTransitionRecord {
				for _, r := range n {
					r.MarkForPush()
				}
				return append(n, e[nil])
			}), one, two},
		{"entering route not marked", "TransitionDelegateEnteringUnmarked",
			funcDelegate(func(n []RouteTransitionRecord, _ map[RouteTransitionRecord]RouteTransitionRecord, _ map[RouteTransitionRecord][]RouteTransitionRecord) []RouteTransitionRecord {
				return n
			}), one, oneTwo},
		{"result with a route that is neither entering nor exiting", "TransitionDelegateResultInvalid",
			afterDefault(func(r []RouteTransitionRecord) []RouteTransitionRecord { return append(r, r[0]) }), one, oneTwo},
		{"result without all the routes", "TransitionDelegateResultIncomplete",
			afterDefault(func(r []RouteTransitionRecord) []RouteTransitionRecord { return r[:len(r)-1] }), one, oneTwo},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_ = requireErrorTag(t, updateTo(t, tc.delegate, tc.first, tc.second), tc.tag)
		})
	}
}

func TestTransitionDelegatePagelessRoutesNotMarked(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	delegate := funcDelegate(func(n []RouteTransitionRecord, e map[RouteTransitionRecord]RouteTransitionRecord, _ map[RouteTransitionRecord][]RouteTransitionRecord) []RouteTransitionRecord {
		for _, r := range n {
			r.MarkForPush()
		}
		// Marks the exiting page route but not the pageless routes above it.
		exiting := e[nil]
		exiting.MarkForComplete()
		return append([]RouteTransitionRecord{exiting}, n...)
	})
	app := func(pages ...string) Widget {
		return pagesApp(pagesAppOpts{key: key, pages: pagesNamed(pages...), onPopPage: func(Route, any) bool { return false }, delegate: delegate})
	}
	ts.PumpWidget(app("1"))
	navPush(key, heroPageRoute(func(BuildContext) Widget { return NewText("pageless") }))
	ts.PumpAndSettle()
	ts.PumpWidget(app("2"))
	_ = requireErrorTag(t, takeFirst(t, ts), "TransitionDelegatePagelessUnmarked")
}

func TestRouteEntryDecisionTags(t *testing.T) {
	skipUnlessDebug(t)
	one, oneTwo, two := pagesNamed("1"), pagesNamed("1", "2"), pagesNamed("2")
	mark := func(do func(entering, exiting RouteTransitionRecord)) funcDelegate {
		return func(n []RouteTransitionRecord, e map[RouteTransitionRecord]RouteTransitionRecord, p map[RouteTransitionRecord][]RouteTransitionRecord) []RouteTransitionRecord {
			var entering, exiting RouteTransitionRecord
			for _, r := range n {
				if r.IsWaitingForEnteringDecision() {
					entering = r
				}
			}
			exiting = e[nil]
			do(entering, exiting)
			return DefaultTransitionDelegate{}.Resolve(n, e, p)
		}
	}
	tests := []struct {
		name          string
		tag           string
		delegate      TransitionDelegate
		first, second []Page
	}{
		{"push twice", "RouteEntryPushDecisionInvalid", mark(func(en, _ RouteTransitionRecord) { en.MarkForPush(); en.MarkForPush() }), one, oneTwo},
		{"add twice", "RouteEntryAddDecisionInvalid", mark(func(en, _ RouteTransitionRecord) { en.MarkForAdd(); en.MarkForAdd() }), one, oneTwo},
		{"pop an entering route", "RouteEntryPopDecisionInvalid", mark(func(en, _ RouteTransitionRecord) { en.MarkForPop() }), one, oneTwo},
		{"complete an entering route", "RouteEntryCompleteDecisionInvalid", mark(func(en, _ RouteTransitionRecord) { en.MarkForComplete() }), one, oneTwo},
		{"push an exiting route", "RouteEntryPushDecisionInvalid", mark(func(_, ex RouteTransitionRecord) { ex.MarkForPush() }), one, two},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_ = requireErrorTag(t, updateTo(t, tc.delegate, tc.first, tc.second), tc.tag)
		})
	}
}

func TestRouteEntryPopTags(t *testing.T) {
	skipUnlessDebug(t)
	popAll := funcDelegate(func(n []RouteTransitionRecord, e map[RouteTransitionRecord]RouteTransitionRecord, p map[RouteTransitionRecord][]RouteTransitionRecord) []RouteTransitionRecord {
		for _, r := range n {
			r.MarkForPush()
		}
		exiting := e[nil]
		exiting.MarkForPop()
		return append([]RouteTransitionRecord{exiting}, n...)
	})
	t.Run("RouteEntryPopAttemptsExceeded", func(t *testing.T) {
		err := updateTo(t, popAll, []Page{stubbornPage("1", false)}, pagesNamed("2"))
		_ = requireErrorTag(t, err, "RouteEntryPopAttemptsExceeded")
	})
	t.Run("RouteEntryPopInvalid", func(t *testing.T) {
		err := updateTo(t, popAll, []Page{stubbornPage("1", true)}, pagesNamed("2"))
		_ = requireErrorTag(t, err, "RouteEntryPopInvalid")
	})
}

func TestPageTags(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("NavigatorPageKeyDuplicated", func(t *testing.T) {
		err := updateTo(t, nil, []Page{newTestPage("a", "x")}, []Page{newTestPage("a", "x"), newTestPage("a", "y")})
		_ = requireErrorTag(t, err, "NavigatorPageKeyDuplicated")
	})
	t.Run("PageRouteSettingsInvalid", func(t *testing.T) {
		wrong := newPage("b", "x", func(*testPage) Route {
			return heroPageRoute(func(BuildContext) Widget { return NewText("x") })
		})
		err := updateTo(t, nil, []Page{newTestPage("a", "x")}, []Page{newTestPage("a", "x"), wrong})
		_ = requireErrorTag(t, err, "PageRouteSettingsInvalid")
	})
	t.Run("NavigatorHistoryFlushReentrant", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := NewGlobalKey[*NavigatorState]()
		first := newTestPage("a", "a")
		pushing := newPage("b", "b", func(p *testPage) Route {
			// Pushing while the pages are being updated flushes the history
			// inside the update.
			if state, ok := key.CurrentState(); ok {
				state.Push(heroPageRoute(func(BuildContext) Widget { return NewText("pushed") }))
			}
			return heroPageRoute(func(BuildContext) Widget { return NewText("b") }, p)
		})
		onPop := func(Route, any) bool { return false }
		ts.PumpWidget(pagesApp(pagesAppOpts{key: key, pages: []Page{first}, onPopPage: onPop}))
		ts.PumpWidget(pagesApp(pagesAppOpts{key: key, pages: []Page{first, pushing}, onPopPage: onPop}))
		_ = requireErrorTag(t, takeFirst(t, ts), "NavigatorHistoryFlushReentrant")
	})
}
