package gestures

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// PointerRoute is a function that receives the pointer events of a route.
//
// Flutter: gestures/pointer_router.dart PointerRoute
type PointerRoute func(event PointerEvent)

// PointerRouteHandle identifies a route that was added to a PointerRouter. It
// removes the route.
//
// Deviation: Flutter removes a route by passing the same function again; Go
// functions can't be compared, so AddRoute returns a handle (SPEC §12.10).
type PointerRouteHandle struct{ id uint64 }

type routeEntry struct {
	id        uint64
	route     PointerRoute
	transform *ui.Matrix4
}

// PointerRouter routes pointer events to the routes registered for the
// pointer, and to the global routes. Per-pointer routes run before the global
// ones, each group in the order it was added.
//
// Flutter: gestures/pointer_router.dart PointerRouter
type PointerRouter struct {
	nextID       uint64
	routes       map[int][]routeEntry
	globalRoutes []routeEntry
}

// NewPointerRouter creates a router without routes.
//
// Flutter: gestures/pointer_router.dart PointerRouter.new
func NewPointerRouter() *PointerRouter {
	return &PointerRouter{routes: map[int][]routeEntry{}}
}

// AddRoute registers a route for the events of the pointer. The route receives
// the events transformed by transform, if it is not nil.
//
// Flutter: gestures/pointer_router.dart PointerRouter.addRoute
func (r *PointerRouter) AddRoute(pointer int, route PointerRoute, transform *ui.Matrix4) PointerRouteHandle {
	r.nextID++
	r.routes[pointer] = append(r.routes[pointer], routeEntry{r.nextID, route, transform})
	return PointerRouteHandle{r.nextID}
}

// RemoveRoute removes a route added for the pointer. Removing a route that is
// not registered panics with the tag PointerRouteNotFound in debug builds.
//
// Flutter: gestures/pointer_router.dart PointerRouter.removeRoute
func (r *PointerRouter) RemoveRoute(pointer int, handle PointerRouteHandle) {
	entries, ok := r.routes[pointer]
	if foundation.DebugMode && !ok {
		panic(errors.New("PointerRouteNotFound", "no route is registered for the pointer", "pointer", pointer))
	}
	i := indexOfRoute(entries, handle.id)
	if i < 0 {
		if foundation.DebugMode {
			panic(errors.New("PointerRouteNotFound", "the route is not registered for the pointer", "pointer", pointer))
		}
		return
	}
	entries = append(entries[:i:i], entries[i+1:]...)
	if len(entries) == 0 {
		delete(r.routes, pointer)
	} else {
		r.routes[pointer] = entries
	}
}

// AddGlobalRoute registers a route that receives every event, whatever its
// pointer. The route receives the events transformed by transform, if it is
// not nil.
//
// Flutter: gestures/pointer_router.dart PointerRouter.addGlobalRoute
func (r *PointerRouter) AddGlobalRoute(route PointerRoute, transform *ui.Matrix4) PointerRouteHandle {
	r.nextID++
	r.globalRoutes = append(r.globalRoutes, routeEntry{r.nextID, route, transform})
	return PointerRouteHandle{r.nextID}
}

// RemoveGlobalRoute removes a global route. Removing a route that is not
// registered panics with the tag PointerRouteNotFound in debug builds.
//
// Flutter: gestures/pointer_router.dart PointerRouter.removeGlobalRoute
func (r *PointerRouter) RemoveGlobalRoute(handle PointerRouteHandle) {
	i := indexOfRoute(r.globalRoutes, handle.id)
	if i < 0 {
		if foundation.DebugMode {
			panic(errors.New("PointerRouteNotFound", "the global route is not registered"))
		}
		return
	}
	r.globalRoutes = append(r.globalRoutes[:i:i], r.globalRoutes[i+1:]...)
}

func indexOfRoute(entries []routeEntry, id uint64) int {
	for i, e := range entries {
		if e.id == id {
			return i
		}
	}
	return -1
}

// DebugGlobalRouteCount is the number of global routes. It is meant for tests
// that look for leaks, and panics with the tag DebugOnly in release builds.
//
// Flutter: gestures/pointer_router.dart PointerRouter.debugGlobalRouteCount
func (r *PointerRouter) DebugGlobalRouteCount() int {
	if !foundation.DebugMode {
		panic(errors.New("DebugOnly", "DebugGlobalRouteCount is not supported in release builds"))
	}
	return len(r.globalRoutes)
}

func dispatchToRoute(event PointerEvent, route PointerRoute, transform *ui.Matrix4) {
	// TODO(step 7.3): recover a panic in the route and report it, as
	// Flutter's PointerRouter does with FlutterError.reportError, then go on
	// with the next route.
	route(event.Transformed(transform))
}

// Route calls the routes of the event's pointer, then the global routes. A
// route that was removed by a route called earlier for the same event is not
// called, and one that was added is not called until the next event.
//
// Flutter: gestures/pointer_router.dart PointerRouter.route
func (r *PointerRouter) Route(event PointerEvent) {
	var pointerRoutes []routeEntry
	if entries, ok := r.routes[event.Pointer()]; ok {
		pointerRoutes = append(pointerRoutes, entries...)
	}
	globalCopy := append([]routeEntry(nil), r.globalRoutes...)
	r.dispatchToRoutes(event, func() []routeEntry { return r.routes[event.Pointer()] }, pointerRoutes)
	r.dispatchToRoutes(event, func() []routeEntry { return r.globalRoutes }, globalCopy)
}

func (r *PointerRouter) dispatchToRoutes(event PointerEvent, current func() []routeEntry, copied []routeEntry) {
	for _, e := range copied {
		if indexOfRoute(current(), e.id) >= 0 {
			dispatchToRoute(event, e.route, e.transform)
		}
	}
}
