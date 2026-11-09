//go:build darwin

package coreaudio

import (
	"context"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// The values of AVAuthorizationStatus.
const (
	authNotDetermined = 0
	authRestricted    = 1
	authDenied        = 2
	authAuthorized    = 3
)

// avPermissions asks AVFoundation for the microphone. The first recording asks the user (the usage string of the bundle's
// Info.plist; an app that is not in a bundle is asked about the program that started it, which is the terminal).
type avPermissions struct{}

// NewPermissions returns the Permissions of the system's microphone.
//
// Flutter: no counterpart
func NewPermissions() Permissions { return avPermissions{} }

// Microphone implements Permissions.
func (avPermissions) Microphone(ctx context.Context) (bool, error) {
	if _, err := dl.OpenFramework("AVFoundation"); err != nil {
		return false, err
	}
	cls := objc.GetClass("AVCaptureDevice")
	if cls == 0 {
		return false, errors.New(platform.TagAudioCapturePermissionDenied, "AVFoundation has no capture devices", "class", "AVCaptureDevice")
	}
	var granted bool
	var err error
	objcx.WithPool(func() {
		media := objcx.NewString("soun") // AVMediaTypeAudio
		defer objcx.Release(media)
		switch objc.Send[int64](objc.ID(cls), objcx.Sel("authorizationStatusForMediaType:"), media) {
		case authAuthorized:
			granted = true
			return
		case authRestricted, authDenied:
			return
		}
		ch := make(chan bool, 1)
		blk := objcx.NewBlock(func(_ objc.Block, ok bool) { ch <- ok })
		kept := blk.Copy()
		blk.Release()
		objc.ID(cls).Send(objcx.Sel("requestAccessForMediaType:completionHandler:"), media, kept)
		select {
		case granted = <-ch:
			kept.Release()
		case <-ctx.Done():
			err = errors.NewErr(platform.TagAudioCapturePermissionDenied, ctx.Err()).SetMetadata("reason", "the context ended while the permission was asked")
		}
	})
	return granted, err
}
