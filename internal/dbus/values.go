package dbus

import (
	"fmt"

	godbus "github.com/godbus/dbus/v5"
)

// typeName names the type of v for an error message.
func typeName(v any) string { return fmt.Sprintf("%T", v) }

// unwrapVariant returns the value inside v, however many variants wrap it.
// The Read method of the Settings portal wraps its value twice.
func unwrapVariant(v any) any {
	for {
		inner, ok := v.(godbus.Variant)
		if !ok {
			return v
		}
		v = inner.Value()
	}
}
