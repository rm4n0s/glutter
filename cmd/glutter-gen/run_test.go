package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestDispatch(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantTag   string
		wantRoute string
		wantCmd   string
	}{
		{name: "no command", args: nil, wantTag: "MissingCommand", wantRoute: "dispatch.MissingCommand"},
		{name: "unknown command", args: []string{"bogus"}, wantTag: "UnknownCommand", wantRoute: "dispatch.UnknownCommand", wantCmd: "bogus"},
		{name: "wayland reaches the real subcommand", args: []string{"wayland", "-nope"}, wantTag: "WaylandBadFlags", wantRoute: "dispatch->Run.WaylandBadFlags"},
		{name: "keys reaches the real subcommand", args: []string{"keys", "-nope"}, wantTag: "KeysBadFlags", wantRoute: "dispatch->Run.KeysBadFlags"},
		{name: "reexport", args: []string{"reexport"}, wantTag: "NotImplemented", wantRoute: "dispatch->Run.NotImplemented", wantCmd: "reexport"},
		{name: "copywith reaches the real subcommand", args: []string{"copywith", "-nope"}, wantTag: "CopyWithBadFlags", wantRoute: "dispatch->Run.CopyWithBadFlags"},
		{name: "icons", args: []string{"icons"}, wantTag: "NotImplemented", wantRoute: "dispatch->Run.NotImplemented", wantCmd: "icons"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dispatch(tt.args)
			if err == nil {
				t.Fatal("expected an error")
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", err)
			}
			if appErr.Tag != tt.wantTag {
				t.Errorf("Tag = %q, want %q", appErr.Tag, tt.wantTag)
			}
			if !appErr.HasRoute(tt.wantRoute) {
				t.Errorf("unexpected failure path: %s, want it to contain %s", appErr.Route(), tt.wantRoute)
			}
			if got, _ := appErr.Metadata["command"].(string); got != tt.wantCmd {
				t.Errorf("Metadata[command] = %q, want %q", got, tt.wantCmd)
			}
		})
	}
}

func TestCommandNames(t *testing.T) {
	got := strings.Join(commandNames(), ",")
	want := "copywith,icons,keys,reexport,wayland"
	if got != want {
		t.Errorf("commandNames() = %q, want %q", got, want)
	}
}

type plainError struct{}

func (plainError) Error() string { return "plain failure" }

func TestFormatError(t *testing.T) {
	t.Run("tagged error is printed as replayable JSON", func(t *testing.T) {
		out := formatError(dispatch([]string{"reexport"}))

		var errJSON errors.ErrorJson
		if err := json.Unmarshal([]byte(out), &errJSON); err != nil {
			t.Fatalf("output is not JSON: %v\n%s", err, out)
		}
		replayed, err := errors.FromJson(errJSON)
		if err != nil {
			t.Fatalf("FromJson: %v", err)
		}
		if replayed.Tag != "NotImplemented" {
			t.Errorf("Tag = %q, want NotImplemented", replayed.Tag)
		}
		if !replayed.HasRoute("dispatch->Run.NotImplemented") {
			t.Errorf("unexpected failure path: %s", replayed.Route())
		}
	})
	t.Run("other error is printed as its message", func(t *testing.T) {
		if got, want := formatError(plainError{}), "glutter-gen: plain failure\n"; got != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})
}
