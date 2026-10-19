package opts

import (
	"testing"

	"github.com/rm4n0s/errors"
)

type testOpts struct{ N int }

func TestFirst(t *testing.T) {
	tests := []struct {
		name string
		in   []testOpts
		want testOpts
	}{
		{"none", nil, testOpts{}},
		{"empty slice", []testOpts{}, testOpts{}},
		{"one", []testOpts{{N: 3}}, testOpts{N: 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := First(tt.in); got != tt.want {
				t.Errorf("First(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestFirstMultiplePanics(t *testing.T) {
	tests := []struct {
		name string
		in   []testOpts
	}{
		{"two", []testOpts{{}, {}}},
		{"three", []testOpts{{N: 1}, {N: 2}, {N: 3}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				err, ok := r.(error)
				if !ok {
					t.Fatalf("expected an error panic, got %T (%v)", r, r)
				}
				appErr, ok := errors.FromError(err)
				if !ok {
					t.Fatalf("expected *errors.Error, got %T", err)
				}
				if !appErr.HasRoute("First[...].MultipleOptions") {
					t.Errorf("unexpected failure path: %s", appErr.Route())
				}
			}()
			First(tt.in)
			t.Fatal("expected a panic")
		})
	}
}
