package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// RouteTransitionRecord is a route and the decision that the TransitionDelegate
// of the Navigator makes about how it transitions in or out when the pages
// change.
//
// Flutter: widgets/navigator.dart RouteTransitionRecord
type RouteTransitionRecord interface {
	// Route is the route that this record is about.
	//
	// Flutter: widgets/navigator.dart RouteTransitionRecord.route
	Route() Route
	// IsWaitingForEnteringDecision is whether the route is new and waits for the
	// delegate to decide how it enters: MarkForPush or MarkForAdd.
	//
	// Flutter: widgets/navigator.dart RouteTransitionRecord.isWaitingForEnteringDecision
	IsWaitingForEnteringDecision() bool
	// IsWaitingForExitingDecision is whether the route is leaving and waits for
	// the delegate to decide how it exits: MarkForPop or MarkForComplete.
	//
	// Flutter: widgets/navigator.dart RouteTransitionRecord.isWaitingForExitingDecision
	IsWaitingForExitingDecision() bool
	// MarkForPush marks the route to be pushed with a transition, as if
	// NavigatorState.Push was called.
	//
	// Flutter: widgets/navigator.dart RouteTransitionRecord.markForPush
	MarkForPush()
	// MarkForAdd marks the route to be added without a transition.
	//
	// Flutter: widgets/navigator.dart RouteTransitionRecord.markForAdd
	MarkForAdd()
	// MarkForPop marks the route to be popped with a transition and completes it
	// with the result, as if NavigatorState.Pop was called.
	//
	// Flutter: widgets/navigator.dart RouteTransitionRecord.markForPop
	MarkForPop(result ...any)
	// MarkForComplete marks the route to be completed with the result and removed
	// without a transition.
	//
	// Flutter: widgets/navigator.dart RouteTransitionRecord.markForComplete
	MarkForComplete(result ...any)
}

// TransitionDelegate decides how the routes of the pages that entered or left
// the Navigator.pages list transition in or out.
//
// Flutter: widgets/navigator.dart TransitionDelegate
type TransitionDelegate interface {
	// Resolve decides how the new page routes enter and the exiting ones leave:
	// it marks every route that waits for a decision and returns every route that
	// stays in the history, in order: the entering routes in the order of
	// newPageRouteHistory, with the exiting routes inserted anywhere.
	//
	// locationToExitingPageRoute maps a page route (nil for the bottom of the
	// history) to the exiting page route right above it, and pageRouteToPagelessRoutes
	// maps a page route (nil for the bottom) to the routes that were pushed above
	// it without a page.
	//
	// Flutter: widgets/navigator.dart TransitionDelegate.resolve
	Resolve(newPageRouteHistory []RouteTransitionRecord, locationToExitingPageRoute map[RouteTransitionRecord]RouteTransitionRecord, pageRouteToPagelessRoutes map[RouteTransitionRecord][]RouteTransitionRecord) []RouteTransitionRecord
}

// runTransitionDelegate calls the delegate and, in debug builds, checks that its
// result is consistent: every route got a decision, the entering routes keep their
// order and every exiting route is in the result.
//
// Flutter: widgets/navigator.dart TransitionDelegate._transition
func runTransitionDelegate(d TransitionDelegate, newPageRouteHistory []RouteTransitionRecord, locationToExitingPageRoute map[RouteTransitionRecord]RouteTransitionRecord, pageRouteToPagelessRoutes map[RouteTransitionRecord][]RouteTransitionRecord) []RouteTransitionRecord {
	results := d.Resolve(newPageRouteHistory, locationToExitingPageRoute, pageRouteToPagelessRoutes)
	if !foundation.DebugMode {
		return results
	}
	exitingPageRoutes := map[RouteTransitionRecord]bool{}
	for _, exiting := range locationToExitingPageRoute {
		exitingPageRoutes[exiting] = true
	}
	// Firstly, verifies that all exiting routes have been marked.
	for exiting := range exitingPageRoutes {
		if exiting.IsWaitingForExitingDecision() {
			panic(errors.New(tagTransitionDelegateExitingUnmarked, "the transition delegate did not decide how an exiting route leaves",
				"route", exiting.Route().String(), "delegate", foundation.RuntimeTypeName(d)))
		}
		for _, pageless := range pageRouteToPagelessRoutes[exiting] {
			if pageless.IsWaitingForExitingDecision() {
				panic(errors.New(tagTransitionDelegatePagelessUnmarked, "the transition delegate did not decide how a pageless route of an exiting route leaves",
					"route", pageless.Route().String(), "delegate", foundation.RuntimeTypeName(d)))
			}
		}
	}
	// Secondly, verifies that the order of the results matches newPageRouteHistory
	// and that they contain all the exiting routes.
	indexOfNextRouteInNewHistory := 0
	for _, record := range results {
		if record.IsWaitingForEnteringDecision() || record.IsWaitingForExitingDecision() {
			panic(errors.New(tagTransitionDelegateEnteringUnmarked, "the transition delegate returned a route that still waits for a decision",
				"route", record.Route().String(), "delegate", foundation.RuntimeTypeName(d)))
		}
		if indexOfNextRouteInNewHistory >= len(newPageRouteHistory) || record != newPageRouteHistory[indexOfNextRouteInNewHistory] {
			if !exitingPageRoutes[record] {
				panic(errors.New(tagTransitionDelegateResultInvalid, "the transition delegate returned a route that is neither an entering route in order nor an exiting route",
					"route", record.Route().String(), "delegate", foundation.RuntimeTypeName(d)))
			}
			delete(exitingPageRoutes, record)
		} else {
			indexOfNextRouteInNewHistory++
		}
	}
	if indexOfNextRouteInNewHistory != len(newPageRouteHistory) || len(exitingPageRoutes) != 0 {
		panic(errors.New(tagTransitionDelegateResultIncomplete,
			"the merged result from "+foundation.RuntimeTypeName(d)+".Resolve does not include all required routes; do you remember to merge all exiting routes?",
			"delegate", foundation.RuntimeTypeName(d)))
	}
	return results
}

// DefaultTransitionDelegate is the transition delegate of a Navigator that has
// none: it pushes the last new route with a transition and adds the others, and
// pops the last exiting route (when it has no pageless routes above) with a
// transition and completes the others.
//
// Flutter: widgets/navigator.dart DefaultTransitionDelegate
type DefaultTransitionDelegate struct{}

// Resolve decides the transitions as the type describes.
//
// Flutter: widgets/navigator.dart DefaultTransitionDelegate.resolve
func (DefaultTransitionDelegate) Resolve(newPageRouteHistory []RouteTransitionRecord, locationToExitingPageRoute map[RouteTransitionRecord]RouteTransitionRecord, pageRouteToPagelessRoutes map[RouteTransitionRecord][]RouteTransitionRecord) []RouteTransitionRecord {
	var results []RouteTransitionRecord
	// This function handles the exiting route and its corresponding pageless
	// routes at this location. It also recursively checks whether there are other
	// exiting routes above it and handles them accordingly.
	var handleExitingRoute func(location RouteTransitionRecord, isLast bool)
	handleExitingRoute = func(location RouteTransitionRecord, isLast bool) {
		exitingPageRoute, ok := locationToExitingPageRoute[location]
		if !ok {
			return
		}
		if exitingPageRoute.IsWaitingForExitingDecision() {
			pagelessRoutes, hasPagelessRoute := pageRouteToPagelessRoutes[exitingPageRoute]
			_, exitingAbove := locationToExitingPageRoute[exitingPageRoute]
			isLastExitingPageRoute := isLast && !exitingAbove
			if isLastExitingPageRoute && !hasPagelessRoute {
				exitingPageRoute.MarkForPop(exitingPageRoute.Route().CurrentResult())
			} else {
				exitingPageRoute.MarkForComplete(exitingPageRoute.Route().CurrentResult())
			}
			if hasPagelessRoute {
				for i, pagelessRoute := range pagelessRoutes {
					// A pageless route that belongs to an exiting page-based route may
					// not require an exiting decision: this happens if the page list is
					// updated right after a Navigator.pop.
					if pagelessRoute.IsWaitingForExitingDecision() {
						if isLastExitingPageRoute && i == len(pagelessRoutes)-1 {
							pagelessRoute.MarkForPop(pagelessRoute.Route().CurrentResult())
						} else {
							pagelessRoute.MarkForComplete(pagelessRoute.Route().CurrentResult())
						}
					}
				}
			}
		}
		results = append(results, exitingPageRoute)
		// There may be another exiting route above this one.
		handleExitingRoute(exitingPageRoute, isLast)
	}

	// Handles the exiting route at the beginning of the list.
	handleExitingRoute(nil, len(newPageRouteHistory) == 0)

	for i, pageRoute := range newPageRouteHistory {
		isLastIteration := i == len(newPageRouteHistory)-1
		if pageRoute.IsWaitingForEnteringDecision() {
			if _, exiting := locationToExitingPageRoute[pageRoute]; !exiting && isLastIteration {
				pageRoute.MarkForPush()
			} else {
				pageRoute.MarkForAdd()
			}
		}
		results = append(results, pageRoute)
		handleExitingRoute(pageRoute, isLastIteration)
	}
	return results
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTransitionDelegateEnteringUnmarked = "TransitionDelegateEnteringUnmarked"
	tagTransitionDelegateExitingUnmarked  = "TransitionDelegateExitingUnmarked"
	tagTransitionDelegatePagelessUnmarked = "TransitionDelegatePagelessUnmarked"
	tagTransitionDelegateResultIncomplete = "TransitionDelegateResultIncomplete"
	tagTransitionDelegateResultInvalid    = "TransitionDelegateResultInvalid"
)
