package dbus

import (
	"context"
	"crypto/rand"
	"strings"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
)

const (
	// PortalDest is the name of the xdg-desktop-portal service.
	//
	// Flutter: no counterpart
	PortalDest = "org.freedesktop.portal.Desktop"
	// PortalPath is the object path of the portal frontend.
	//
	// Flutter: no counterpart
	PortalPath godbus.ObjectPath = "/org/freedesktop/portal/desktop"

	portalRequestInterface = "org.freedesktop.portal.Request"
	portalRequestPrefix    = "/org/freedesktop/portal/desktop/request/"
	portalCloseTimeout     = 2 * time.Second
)

// PortalResponseCode is the first value of the Response signal of a portal
// request.
//
// Flutter: no counterpart
type PortalResponseCode uint32

// The values of PortalResponseCode are the codes of the portal protocol.
const (
	// PortalResponseSuccess means the request finished and has results.
	//
	// Flutter: no counterpart
	PortalResponseSuccess PortalResponseCode = 0
	// PortalResponseCancelled means the user cancelled the interaction. It is a
	// result, not an error.
	//
	// Flutter: no counterpart
	PortalResponseCancelled PortalResponseCode = 1
	// PortalResponseEnded means the interaction ended in some other way.
	//
	// Flutter: no counterpart
	PortalResponseEnded PortalResponseCode = 2
)

// PortalResponse is the answer to a portal request.
//
// Flutter: no counterpart
type PortalResponse struct {
	// Code says how the interaction ended.
	Code PortalResponseCode
	// Results are the values of the answer; empty unless Code is
	// PortalResponseSuccess.
	Results map[string]godbus.Variant
}

// PortalRequest calls a portal method that answers through a
// org.freedesktop.portal.Request object, and waits for the answer.
//
// The method is the full name, such as "org.freedesktop.portal.FileChooser.OpenFile".
// args builds the arguments of the call from the handle token it must put in
// its options dictionary as "handle_token". The Response signal is subscribed
// to before the call, at the path the portal derives from the token, so an
// answer that comes at once is not lost. When ctx is done before the answer,
// the request is closed and the error is PortalRequestAborted.
//
// Flutter: no counterpart
func PortalRequest(ctx context.Context, bus Bus, method string, args func(handleToken string) []any) (PortalResponse, error) {
	token := newHandleToken()
	predicted := PortalRequestPath(bus.UniqueName(), token)

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	sigs, err := bus.Subscribe(subCtx, SignalMatch{Path: predicted, Interface: portalRequestInterface, Member: "Response"})
	if err != nil {
		return PortalResponse{}, err
	}

	reply, err := bus.Call(ctx, PortalDest, PortalPath, method, args(token)...)
	if err != nil {
		return PortalResponse{}, err
	}
	handle := predicted
	if len(reply) > 0 {
		h, ok := reply[0].(godbus.ObjectPath)
		if !ok {
			return PortalResponse{}, errors.New(tagPortalReplyInvalid, "the reply of the portal method has no request handle",
				"method", method, "reply", reply)
		}
		handle = h
	}
	if handle != predicted {
		// Portals older than version 0.9 do not derive the handle from the
		// token: follow the signal at the handle they returned.
		more, err := bus.Subscribe(subCtx, SignalMatch{Path: handle, Interface: portalRequestInterface, Member: "Response"})
		if err != nil {
			return PortalResponse{}, err
		}
		sigs = mergeSignals(subCtx, sigs, more)
	}

	select {
	case sig, ok := <-sigs:
		if !ok {
			return PortalResponse{}, errors.New(tagPortalRequestClosed, "the connection closed before the portal answered",
				"method", method)
		}
		return parsePortalResponse(sig, method)
	case <-ctx.Done():
		appErr := errors.NewErr(tagPortalRequestAborted, ctx.Err()).SetMetadata("method", method)
		closeCtx, closeCancel := context.WithTimeout(context.WithoutCancel(ctx), portalCloseTimeout)
		defer closeCancel()
		if _, err := bus.Call(closeCtx, PortalDest, handle, portalRequestInterface+".Close"); err != nil {
			appErr = appErr.SetMetadata("closeError", err.Error())
		}
		return PortalResponse{}, appErr
	}
}

// PortalRequestPath is the object path of the request that the portal makes
// for the connection named unique and the handle token token.
//
// Flutter: no counterpart
func PortalRequestPath(unique, token string) godbus.ObjectPath {
	sender := strings.ReplaceAll(strings.TrimPrefix(unique, ":"), ".", "_")
	return godbus.ObjectPath(portalRequestPrefix + sender + "/" + token)
}

func newHandleToken() string {
	return "glutter" + rand.Text()[:16]
}

func parsePortalResponse(sig *godbus.Signal, method string) (PortalResponse, error) {
	if len(sig.Body) != 2 {
		return PortalResponse{}, errors.New(tagPortalResponseInvalid, "the Response signal does not have two values",
			"method", method, "values", len(sig.Body))
	}
	code, ok := sig.Body[0].(uint32)
	if !ok {
		return PortalResponse{}, errors.New(tagPortalResponseInvalid, "the response code is not a uint32",
			"method", method, "type", typeName(sig.Body[0]))
	}
	results, ok := sig.Body[1].(map[string]godbus.Variant)
	if !ok {
		return PortalResponse{}, errors.New(tagPortalResponseInvalid, "the results are not a dictionary of variants",
			"method", method, "type", typeName(sig.Body[1]))
	}
	return PortalResponse{Code: PortalResponseCode(code), Results: results}, nil
}

// mergeSignals forwards the signals of both channels until ctx is done or both
// are closed.
func mergeSignals(ctx context.Context, a, b <-chan *godbus.Signal) <-chan *godbus.Signal {
	out := make(chan *godbus.Signal, 1)
	go func() {
		defer close(out)
		for a != nil || b != nil {
			var sig *godbus.Signal
			var ok bool
			select {
			case sig, ok = <-a:
				if !ok {
					a = nil
					continue
				}
			case sig, ok = <-b:
				if !ok {
					b = nil
					continue
				}
			case <-ctx.Done():
				return
			}
			select {
			case out <- sig:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagPortalReplyInvalid    = "PortalReplyInvalid"
	tagPortalRequestAborted  = "PortalRequestAborted"
	tagPortalRequestClosed   = "PortalRequestClosed"
	tagPortalResponseInvalid = "PortalResponseInvalid"
)
