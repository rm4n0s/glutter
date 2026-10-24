package ui

// ServiceHost is what the engine implements to hand out the optional services that its platform
// has by itself, and that package desktop uses instead of D-Bus: on Android the services of
// opening a URI and of notifications (SPEC §24.11). A host without it has none. The services are
// named, and the value is a plain Go interface that the caller asserts, because package ui cannot
// name the platform's types.
//
// Flutter: no counterpart
type ServiceHost interface {
	// PlatformService returns the service with the name, or nil when the platform has none. It may be
	// called from any goroutine.
	PlatformService(name string) any
}

// PlatformService returns the optional service of the platform with the name (see ServiceHost), or
// nil when the host or the platform has none. It may be called from any goroutine.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) PlatformService(name string) any {
	if h, ok := d.host.(ServiceHost); ok {
		return h.PlatformService(name)
	}
	return nil
}
