package ui

import "github.com/rm4n0s/errors"

// ContentType is what a window shows, as a hint to the compositor (Wayland's
// wp_content_type_v1): a compositor can pick the refresh rate and the latency
// that suit a game or a video. The zero value is ContentTypeUnspecified.
//
// Flutter: no counterpart
type ContentType int

// The values of ContentType.
const (
	// ContentTypeUnspecified gives no hint.
	//
	// Flutter: no counterpart
	ContentTypeUnspecified ContentType = iota
	// ContentTypePhoto is a window that shows still pictures.
	//
	// Flutter: no counterpart
	ContentTypePhoto
	// ContentTypeVideo is a window that plays video.
	//
	// Flutter: no counterpart
	ContentTypeVideo
	// ContentTypeGame is a game: the compositor may favor a low latency.
	//
	// Flutter: no counterpart
	ContentTypeGame
)

// WindowIconImage is one picture of a window icon.
//
// Flutter: no counterpart
type WindowIconImage struct {
	// Size is the width and the height of the picture in pixels; the picture is
	// square.
	Size int
	// Scale is the scale of the picture; the icon is Size/Scale logical pixels
	// wide. The default, for zero, is 1.
	Scale int
	// RGBA holds Size*Size*4 bytes: the rows from the top, each pixel as red,
	// green, blue and alpha, with the color not premultiplied by the alpha.
	RGBA []byte
}

// WindowIcon is the icon of a window: the name of an icon of the icon theme,
// pictures, or both. The compositor uses the name first when it knows the icon.
// The zero value is no icon: the window gets its icon from the app's .desktop
// file.
//
// Flutter: no counterpart
type WindowIcon struct {
	// Name is the name of an icon of the icon theme, such as "text-editor".
	Name string
	// Images are the pictures of the icon, one for each size.
	Images []WindowIconImage
}

// IsZero is whether the icon is empty.
//
// Flutter: no counterpart
func (i WindowIcon) IsZero() bool { return i.Name == "" && len(i.Images) == 0 }

// Validate checks the pictures of the icon. It fails with WindowIconInvalid
// (metadata "index") when a picture is not square, has a negative scale, or does
// not have Size*Size*4 bytes.
//
// Flutter: no counterpart
func (i WindowIcon) Validate() error {
	for n, img := range i.Images {
		if img.Size <= 0 || img.Scale < 0 || len(img.RGBA) != img.Size*img.Size*4 {
			return errors.New(tagWindowIconInvalid, "an image of a window icon must have Size*Size*4 bytes and a positive size",
				"index", n, "size", img.Size, "bytes", len(img.RGBA))
		}
	}
	return nil
}

// WindowAppearanceHost is what the engine implements to change what a window
// looks like to the desktop. A host without it ignores the changes.
//
// Flutter: no counterpart
type WindowAppearanceHost interface {
	// SetWindowIcon sets the icon of the window of the view. An empty icon removes it.
	SetWindowIcon(view ViewID, icon WindowIcon) error
	// SetWindowContentType gives the compositor a hint about what the window shows.
	SetWindowContentType(view ViewID, contentType ContentType)
}

// IdleInhibitHost is what the engine implements to keep the screen from going
// idle (dimming, blanking and locking). A host without it cannot do it.
//
// Flutter: no counterpart
type IdleInhibitHost interface {
	// KeepScreenOn stops the screen from going idle while the window of the view
	// is visible (on is true), or lets it (false). It fails with
	// IdleInhibitUnavailable when the platform cannot.
	KeepScreenOn(view ViewID, on bool) error
}

// SetWindowIcon sets the icon of the window of the view. It fails with
// WindowIconInvalid when a picture is wrong, and with the platform's error when
// it cannot use the icon. A host that cannot set icons ignores it.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) SetWindowIcon(id ViewID, icon WindowIcon) error {
	if err := icon.Validate(); err != nil {
		return err
	}
	if h, ok := d.host.(WindowAppearanceHost); ok {
		return h.SetWindowIcon(id, icon)
	}
	return nil
}

// SetWindowContentType tells the compositor what the window of the view shows.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) SetWindowContentType(id ViewID, contentType ContentType) {
	if h, ok := d.host.(WindowAppearanceHost); ok {
		h.SetWindowContentType(id, contentType)
	}
}

// KeepScreenOn stops the screen from going idle while the window of the view is
// visible, or, with false, lets it. It fails with IdleInhibitUnavailable when the
// host or the platform cannot.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) KeepScreenOn(id ViewID, on bool) error {
	if h, ok := d.host.(IdleInhibitHost); ok {
		return h.KeepScreenOn(id, on)
	}
	return errors.New(tagIdleInhibitUnavailable, "the platform cannot keep the screen on")
}

// SetIcon sets the icon of the window of the view; see
// PlatformDispatcher.SetWindowIcon.
//
// Flutter: no counterpart
func (v *View) SetIcon(icon WindowIcon) error { return v.d.SetWindowIcon(v.id, icon) }

// SetContentType tells the compositor what the window of the view shows.
//
// Flutter: no counterpart
func (v *View) SetContentType(contentType ContentType) { v.d.SetWindowContentType(v.id, contentType) }

// KeepScreenOn stops the screen from going idle while the window is visible, or,
// with false, lets it; see PlatformDispatcher.KeepScreenOn.
//
// Flutter: no counterpart
func (v *View) KeepScreenOn(on bool) error { return v.d.KeepScreenOn(v.id, on) }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagIdleInhibitUnavailable = "IdleInhibitUnavailable"
	tagWindowIconInvalid      = "WindowIconInvalid"
)
