package fakebus

import (
	"context"
	"testing"

	godbus "github.com/godbus/dbus/v5"
)

// TestNameHasOwner: the bus answers who owns a name, as a real bus does, unless the test gave the method a handler.
func TestNameHasOwner(t *testing.T) {
	const method = "org.freedesktop.DBus.NameHasOwner"
	b := New(":1.5")
	b.Handle("org.example.Served", "/a", "org.example.Served.Do", func(Call) ([]any, error) { return nil, nil })
	b.HandleAnyPath("org.example.AnyPath", "org.example.AnyPath.Do", func(Call) ([]any, error) { return nil, nil })
	if err := b.RequestName("org.example.Mine"); err != nil {
		t.Fatal(err)
	}
	b.Take("org.example.Other")
	tests := []struct {
		name string
		want bool
	}{
		{"org.freedesktop.DBus", true},
		{":1.5", true},
		{"org.example.Served", true},
		{"org.example.AnyPath", true},
		{"org.example.Mine", true},
		{"org.example.Other", true},
		{"org.example.Nobody", false},
		{"org.example", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.Call(context.Background(), "org.freedesktop.DBus", "/org/freedesktop/DBus", method, tt.name)
			if err != nil || len(got) != 1 || got[0] != tt.want {
				t.Errorf("NameHasOwner(%s) = %v, %v; want %v", tt.name, got, err, tt.want)
			}
		})
	}
	t.Run("a name that is no longer served", func(t *testing.T) {
		b.Unhandle("org.example.Served", "/a", "org.example.Served.Do")
		if got, _ := b.Call(context.Background(), "org.freedesktop.DBus", "/org/freedesktop/DBus", method, "org.example.Served"); got[0] != false {
			t.Errorf("NameHasOwner = %v after Unhandle", got)
		}
	})
	t.Run("a handler of the test answers instead", func(t *testing.T) {
		b.Handle("org.freedesktop.DBus", godbus.ObjectPath("/org/freedesktop/DBus"), method, func(Call) ([]any, error) { return []any{true}, nil })
		if got, _ := b.Call(context.Background(), "org.freedesktop.DBus", "/org/freedesktop/DBus", method, "org.example.Nobody"); got[0] != true {
			t.Errorf("NameHasOwner = %v, want the handler's answer", got)
		}
	})
}
