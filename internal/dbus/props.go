package dbus

import (
	"reflect"
	"sync"

	godbus "github.com/godbus/dbus/v5"
)

// PropertiesIface is the interface of the standard properties of an object.
const PropertiesIface = "org.freedesktop.DBus.Properties"

// Properties serves the properties of one object path: the methods Get, GetAll
// and Set of org.freedesktop.DBus.Properties, and the signal PropertiesChanged
// when a value is updated. The values are set and read from any goroutine.
//
// Flutter: no counterpart
type Properties struct {
	bus  Bus
	path godbus.ObjectPath

	mu       sync.Mutex
	values   map[string]map[string]godbus.Variant
	exported bool
}

// NewProperties creates the properties of path. Nothing is served until Export.
//
// Flutter: no counterpart
func NewProperties(bus Bus, path godbus.ObjectPath) *Properties {
	return &Properties{bus: bus, path: path, values: map[string]map[string]godbus.Variant{}}
}

// Export starts serving the properties on the bus.
//
// Flutter: no counterpart
func (p *Properties) Export() error {
	if err := p.bus.Export(propertiesObject{p}, p.path, PropertiesIface); err != nil {
		return err
	}
	p.mu.Lock()
	p.exported = true
	p.mu.Unlock()
	return nil
}

// Unexport stops serving the properties.
//
// Flutter: no counterpart
func (p *Properties) Unexport() {
	p.mu.Lock()
	p.exported = false
	p.mu.Unlock()
	p.bus.Unexport(p.path, PropertiesIface)
}

// Store sets the value of a property without telling anybody, for the values
// that exist before the object is exported. It reports whether the value changed.
//
// Flutter: no counterpart
func (p *Properties) Store(iface, name string, value any) (changed bool) {
	v := godbus.MakeVariant(value)
	p.mu.Lock()
	defer p.mu.Unlock()
	m := p.values[iface]
	if m == nil {
		m = map[string]godbus.Variant{}
		p.values[iface] = m
	}
	if old, ok := m[name]; ok && reflect.DeepEqual(old.Value(), v.Value()) {
		return false
	}
	m[name] = v
	return true
}

// Update sets the value of a property and, when it changed and the object is
// exported, emits PropertiesChanged with the new value.
//
// Flutter: no counterpart
func (p *Properties) Update(iface, name string, value any) error {
	if !p.Store(iface, name, value) {
		return nil
	}
	p.mu.Lock()
	exported := p.exported
	p.mu.Unlock()
	if !exported {
		return nil
	}
	changed := map[string]godbus.Variant{name: godbus.MakeVariant(value)}
	return p.bus.EmitSignal(p.path, PropertiesIface+".PropertiesChanged", iface, changed, []string{})
}

// Value is the current value of a property.
//
// Flutter: no counterpart
func (p *Properties) Value(iface, name string) (any, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.values[iface][name]
	if !ok {
		return nil, false
	}
	return v.Value(), true
}

// propertiesObject is what is exported: the methods of the interface, and
// nothing else, since godbus exports every method of the receiver.
type propertiesObject struct{ p *Properties }

func unknownProperty(iface, name string) *godbus.Error {
	return &godbus.Error{Name: "org.freedesktop.DBus.Error.UnknownProperty",
		Body: []any{"no property " + name + " on " + iface}}
}

// Get is org.freedesktop.DBus.Properties.Get.
func (o propertiesObject) Get(iface, name string) (godbus.Variant, *godbus.Error) {
	o.p.mu.Lock()
	defer o.p.mu.Unlock()
	v, ok := o.p.values[iface][name]
	if !ok {
		return godbus.Variant{}, unknownProperty(iface, name)
	}
	return v, nil
}

// GetAll is org.freedesktop.DBus.Properties.GetAll.
func (o propertiesObject) GetAll(iface string) (map[string]godbus.Variant, *godbus.Error) {
	o.p.mu.Lock()
	defer o.p.mu.Unlock()
	all := make(map[string]godbus.Variant, len(o.p.values[iface]))
	for k, v := range o.p.values[iface] {
		all[k] = v
	}
	return all, nil
}

// Set is org.freedesktop.DBus.Properties.Set; every property is read-only.
func (o propertiesObject) Set(iface, name string, _ godbus.Variant) *godbus.Error {
	return &godbus.Error{Name: "org.freedesktop.DBus.Error.PropertyReadOnly",
		Body: []any{name + " of " + iface + " is read-only"}}
}
