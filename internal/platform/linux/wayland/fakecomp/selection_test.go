//go:build linux && !android

package fakecomp_test

import (
	"syscall"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
)

func TestReadingASourceTheClientDoesNotHaveFails(t *testing.T) {
	tests := []struct {
		name  string
		read  func(*fakecomp.Server) error
		route string
		key   string
	}{
		{"drag source", func(s *fakecomp.Server) error { _, err := s.ReadDragSource("text/plain"); return err },
			"Server.ReadDragSource.FakeCompNoDragSource", ""},
		{"selection", func(s *fakecomp.Server) error { _, err := s.ReadClientSelection(false, "text/plain"); return err },
			"Server.ReadClientSelection.FakeCompNoSelection", "primary"},
		{"primary selection", func(s *fakecomp.Server) error { _, err := s.ReadClientSelection(true, "text/plain"); return err },
			"Server.ReadClientSelection.FakeCompNoSelection", "primary"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, fd := fakecomp.New(t, fakecomp.Config{})
			defer srv.Wait()
			defer func() { _ = syscall.Close(fd) }()

			err := tt.read(srv)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), tt.route)
			}
			if tt.key != "" {
				if _, has := appErr.Metadata[tt.key]; !has {
					t.Errorf("metadata = %v, want the key %q", appErr.Metadata, tt.key)
				}
			}
		})
	}
}
