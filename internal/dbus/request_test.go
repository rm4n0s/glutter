package dbus_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/fakebus"
)

const askMethod = "org.freedesktop.portal.Test.Ask"

// tokenOf returns the handle token that a call passes in its options.
func tokenOf(t *testing.T, call fakebus.Call) string {
	t.Helper()
	opts, ok := call.Args[0].(map[string]godbus.Variant)
	if !ok {
		t.Fatalf("args = %#v", call.Args)
	}
	token, ok := opts["handle_token"].Value().(string)
	if !ok || token == "" {
		t.Fatalf("no handle_token in %#v", opts)
	}
	return token
}

func ask(token string) []any {
	return []any{map[string]godbus.Variant{"handle_token": godbus.MakeVariant(token)}}
}

func response(path godbus.ObjectPath, body ...any) *godbus.Signal {
	return &godbus.Signal{Path: path, Name: "org.freedesktop.portal.Request.Response", Body: body}
}

func TestPortalRequestPath(t *testing.T) {
	got := dbus.PortalRequestPath(":1.42", "tok")
	if want := godbus.ObjectPath("/org/freedesktop/portal/desktop/request/1_42/tok"); got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
}

func TestPortalRequestAnswers(t *testing.T) {
	tests := []struct {
		name     string
		body     func() []any
		wantCode dbus.PortalResponseCode
		wantKeys []string
	}{
		{"success", func() []any {
			return []any{uint32(0), map[string]godbus.Variant{"uris": godbus.MakeVariant([]string{"file:///a"})}}
		}, dbus.PortalResponseSuccess, []string{"uris"}},
		{"cancelled is a result", func() []any {
			return []any{uint32(1), map[string]godbus.Variant{}}
		}, dbus.PortalResponseCancelled, nil},
		{"ended is a result", func() []any {
			return []any{uint32(2), map[string]godbus.Variant{}}
		}, dbus.PortalResponseEnded, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := fakebus.New(":1.42")
			b.Handle(dbus.PortalDest, dbus.PortalPath, askMethod, func(c fakebus.Call) ([]any, error) {
				path := dbus.PortalRequestPath(":1.42", tokenOf(t, c))
				// The answer comes before the call returns: it must not be lost.
				b.Emit(response(path, tc.body()...))
				return []any{path}, nil
			})
			got, err := dbus.PortalRequest(context.Background(), b, askMethod, ask)
			if err != nil {
				t.Fatal(err)
			}
			if got.Code != tc.wantCode {
				t.Errorf("code = %v, want %v", got.Code, tc.wantCode)
			}
			var keys []string
			for k := range got.Results {
				keys = append(keys, k)
			}
			if !reflect.DeepEqual(keys, tc.wantKeys) {
				t.Errorf("result keys = %v, want %v", keys, tc.wantKeys)
			}
		})
	}
}

func TestPortalRequestIgnoresOtherRequests(t *testing.T) {
	b := fakebus.New(":1.42")
	b.Handle(dbus.PortalDest, dbus.PortalPath, askMethod, func(c fakebus.Call) ([]any, error) {
		path := dbus.PortalRequestPath(":1.42", tokenOf(t, c))
		b.Emit(response(dbus.PortalRequestPath(":1.42", "other"), uint32(1), map[string]godbus.Variant{}))
		b.Emit(response(path, uint32(0), map[string]godbus.Variant{}))
		return []any{path}, nil
	})
	got, err := dbus.PortalRequest(context.Background(), b, askMethod, ask)
	if err != nil {
		t.Fatal(err)
	}
	if got.Code != dbus.PortalResponseSuccess {
		t.Errorf("code = %v, want success (the answer of the other request must be ignored)", got.Code)
	}
}

func TestPortalRequestFollowsAnOldPortalHandle(t *testing.T) {
	b := fakebus.New(":1.42")
	old := godbus.ObjectPath("/org/freedesktop/portal/desktop/request/1_42/old")
	stop := make(chan struct{})
	defer close(stop)
	b.Handle(dbus.PortalDest, dbus.PortalPath, askMethod, func(fakebus.Call) ([]any, error) {
		// The portal answers at a path that does not come from the token, a
		// moment after the call returns: emit until the request listens there.
		go func() {
			for i := 0; i < 1000; i++ {
				select {
				case <-stop:
					return
				case <-time.After(time.Millisecond):
					b.Emit(response(old, uint32(0), map[string]godbus.Variant{}))
				}
			}
		}()
		return []any{old}, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	got, err := dbus.PortalRequest(ctx, b, askMethod, ask)
	if err != nil {
		t.Fatal(err)
	}
	if got.Code != dbus.PortalResponseSuccess {
		t.Errorf("code = %v", got.Code)
	}
}

func TestPortalRequestErrors(t *testing.T) {
	handle := func(h func(b *fakebus.Bus, c fakebus.Call) ([]any, error)) *fakebus.Bus {
		b := fakebus.New(":1.42")
		b.Handle(dbus.PortalDest, dbus.PortalPath, askMethod, func(c fakebus.Call) ([]any, error) { return h(b, c) })
		return b
	}
	tests := []struct {
		name  string
		bus   *fakebus.Bus
		route string
	}{
		{"portal missing", fakebus.New(":1.42"), "PortalRequest->Bus.Call.DBusCallFailed"},
		{"reply without a handle", handle(func(*fakebus.Bus, fakebus.Call) ([]any, error) { return []any{"x"}, nil }),
			"PortalRequest.PortalReplyInvalid"},
		{"response with one value", handle(func(b *fakebus.Bus, c fakebus.Call) ([]any, error) {
			path := dbus.PortalRequestPath(":1.42", tokenOf(t, c))
			b.Emit(response(path, uint32(0)))
			return []any{path}, nil
		}), "parsePortalResponse.PortalResponseInvalid"},
		{"response code of the wrong type", handle(func(b *fakebus.Bus, c fakebus.Call) ([]any, error) {
			path := dbus.PortalRequestPath(":1.42", tokenOf(t, c))
			b.Emit(response(path, "no", map[string]godbus.Variant{}))
			return []any{path}, nil
		}), "parsePortalResponse.PortalResponseInvalid"},
		{"results of the wrong type", handle(func(b *fakebus.Bus, c fakebus.Call) ([]any, error) {
			path := dbus.PortalRequestPath(":1.42", tokenOf(t, c))
			b.Emit(response(path, uint32(0), "no"))
			return []any{path}, nil
		}), "parsePortalResponse.PortalResponseInvalid"},
		{"connection closes", handle(func(b *fakebus.Bus, c fakebus.Call) ([]any, error) {
			if err := b.Close(); err != nil {
				t.Error(err)
			}
			return []any{dbus.PortalRequestPath(":1.42", tokenOf(t, c))}, nil
		}), "PortalRequest.PortalRequestClosed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := dbus.PortalRequest(context.Background(), tc.bus, askMethod, ask)
			if err == nil {
				t.Fatal("expected an error")
			}
			assertRoute(t, err, tc.route)
		})
	}
}

func TestPortalRequestSubscribeFails(t *testing.T) {
	b := fakebus.New(":1.42")
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	_, err := dbus.PortalRequest(context.Background(), b, askMethod, ask)
	assertRoute(t, err, "PortalRequest->Bus.Subscribe.FakeBusClosed")
}

func TestPortalRequestAbortClosesTheRequest(t *testing.T) {
	b := fakebus.New(":1.42")
	asked := make(chan godbus.ObjectPath, 1)
	b.Handle(dbus.PortalDest, dbus.PortalPath, askMethod, func(c fakebus.Call) ([]any, error) {
		path := dbus.PortalRequestPath(":1.42", tokenOf(t, c))
		asked <- path
		return []any{path}, nil
	})
	b.HandleAnyPath(dbus.PortalDest, "org.freedesktop.portal.Request.Close", func(fakebus.Call) ([]any, error) {
		return nil, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-asked
		cancel()
	}()
	_, err := dbus.PortalRequest(ctx, b, askMethod, ask)
	assertRoute(t, err, "PortalRequest.PortalRequestAborted")
	calls := b.Calls()
	last := calls[len(calls)-1]
	if last.Method != "org.freedesktop.portal.Request.Close" || !strings.HasPrefix(string(last.Path), "/org/freedesktop/portal/desktop/request/1_42/glutter") {
		t.Errorf("last call = %+v, want Close of the request", last)
	}
}

func TestPortalRequestAbortReportsAFailedClose(t *testing.T) {
	b := fakebus.New(":1.42")
	asked := make(chan struct{}, 1)
	b.Handle(dbus.PortalDest, dbus.PortalPath, askMethod, func(c fakebus.Call) ([]any, error) {
		asked <- struct{}{}
		return []any{dbus.PortalRequestPath(":1.42", tokenOf(t, c))}, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-asked
		cancel()
	}()
	_, err := dbus.PortalRequest(ctx, b, askMethod, ask)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if _, ok := appErr.Metadata["closeError"]; !ok {
		t.Errorf("metadata = %v, want closeError (nobody handles Close)", appErr.Metadata)
	}
}
