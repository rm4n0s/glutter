// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package services

import (
	"context"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// ClipboardKTextPlain is the format of plain text, for Clipboard.GetData.
//
// Flutter: services/clipboard.dart Clipboard.kTextPlain
const ClipboardKTextPlain = "text/plain"

// ClipboardKImagePNG is the format of an image in the PNG format, for
// Clipboard.GetData.
//
// Flutter: no counterpart
const ClipboardKImagePNG = ui.ClipboardFormatPNG

// ClipboardKURIList is the format of a list of URIs (the files that a file
// manager copied), for Clipboard.GetData.
//
// Flutter: no counterpart
const ClipboardKURIList = ui.ClipboardFormatURIList

// ClipboardData is data on the clipboard: text, and Glutter's extension, an
// image and a list of URIs. SetData puts every format that is set; GetData
// returns the one that was asked for.
//
// Deviation: Flutter's ClipboardData has only the text (SPEC §12.10).
//
// Flutter: services/clipboard.dart ClipboardData
type ClipboardData struct {
	// Text is the text.
	Text string
	// PNG is an image in the PNG format; nil for none.
	PNG []byte
	// URIs are the URIs of files or other resources; nil for none.
	URIs []string
}

func (d ClipboardData) content() ui.ClipboardContent {
	return ui.ClipboardContent{Text: d.Text, HasText: d.Text != "" || (len(d.PNG) == 0 && len(d.URIs) == 0), PNG: d.PNG, URIs: d.URIs}
}

func dataOf(c ui.ClipboardContent) ClipboardData {
	return ClipboardData{Text: c.Text, PNG: c.PNG, URIs: c.URIs}
}

// RichClipboardBackend is what a ClipboardBackend also implements to put and
// read the primary selection, images and files: the platform dispatcher
// (ui.PlatformDispatcher) does. The functions it takes are called later on the
// UI thread, never before the method has returned.
//
// Flutter: no counterpart
type RichClipboardBackend interface {
	// SetClipboard puts the content on the selection of the kind.
	SetClipboard(ctx context.Context, kind ui.ClipboardKind, content ui.ClipboardContent, done func(err error))
	// GetClipboard reads the selection of the kind in the format.
	GetClipboard(ctx context.Context, kind ui.ClipboardKind, format string, done func(content ui.ClipboardContent, err error))
}

// NewClipboardData makes the data of a text.
//
// Flutter: services/clipboard.dart ClipboardData.ClipboardData
func NewClipboardData(text string) ClipboardData { return ClipboardData{Text: text} }

// ClipboardBackend is where a Clipboard gets and puts text: the platform
// dispatcher (ui.PlatformDispatcher) is one. The functions it takes are called
// later on the UI thread, never before the method has returned.
//
// Flutter: no counterpart (the flutter/platform channel)
type ClipboardBackend interface {
	// SetClipboardText puts text on the clipboard.
	SetClipboardText(ctx context.Context, text string, done func(err error))
	// GetClipboardText reads the text of the clipboard; ok is false when there is
	// none.
	GetClipboardText(ctx context.Context, done func(text string, ok bool, err error))
}

// Clipboard is the system clipboard. Reading it can take time (another
// application has to hand the data over), so each method takes a function that
// is called later, on the UI thread, with the result; ctx ends a read that takes
// too long. The binding has one.
//
// Deviation: Flutter's Clipboard has static methods that return a Future, and
// that call the platform through a channel; here the methods take a callback and
// go through the engine (SPEC §12.10).
//
// Flutter: services/clipboard.dart Clipboard
type Clipboard struct {
	backend ClipboardBackend
}

// NewClipboard makes a clipboard on a backend.
//
// Flutter: no counterpart
func NewClipboard(backend ClipboardBackend) *Clipboard { return &Clipboard{backend: backend} }

// SetData stores the data on the clipboard. done, when it is not nil, is called
// with the error if any. A clipboard without a backend calls it at once with
// ClipboardUnavailable.
//
// Flutter: services/clipboard.dart Clipboard.setData
func (c *Clipboard) SetData(ctx context.Context, data ClipboardData, done func(err error)) {
	if c == nil || c.backend == nil {
		if done != nil {
			done(errClipboardUnavailable())
		}
		return
	}
	if len(data.PNG) > 0 || len(data.URIs) > 0 {
		c.setRich(ctx, ui.ClipboardKindSystem, data, done)
		return
	}
	c.backend.SetClipboardText(ctx, data.Text, done)
}

func (c *Clipboard) setRich(ctx context.Context, kind ui.ClipboardKind, data ClipboardData, done func(err error)) {
	if c == nil || c.backend == nil {
		if done != nil {
			done(errClipboardUnavailable())
		}
		return
	}
	rich, ok := c.backend.(RichClipboardBackend)
	if !ok {
		if done != nil {
			done(errClipboardUnavailable())
		}
		return
	}
	rich.SetClipboard(ctx, kind, data.content(), done)
}

func (c *Clipboard) getRich(ctx context.Context, kind ui.ClipboardKind, format string, done func(data *ClipboardData, err error)) {
	if c == nil || c.backend == nil {
		done(nil, errClipboardUnavailable())
		return
	}
	rich, ok := c.backend.(RichClipboardBackend)
	if !ok {
		done(nil, errClipboardUnavailable())
		return
	}
	rich.GetClipboard(ctx, kind, format, func(content ui.ClipboardContent, err error) {
		if err != nil || !content.Has(format) {
			done(nil, err)
			return
		}
		d := dataOf(content)
		done(&d, nil)
	})
}

// SetPrimaryData stores the data on the primary selection, the selected text
// that a middle click pastes on Linux desktops. A platform without one calls done
// with ClipboardUnavailable.
//
// Flutter: no counterpart
func (c *Clipboard) SetPrimaryData(ctx context.Context, data ClipboardData, done func(err error)) {
	c.setRich(ctx, ui.ClipboardKindPrimary, data, done)
}

// GetPrimaryData retrieves the data of the format from the primary selection;
// data is nil when it has none of that format. A platform without one calls done
// with ClipboardUnavailable.
//
// Flutter: no counterpart
func (c *Clipboard) GetPrimaryData(ctx context.Context, format string, done func(data *ClipboardData, err error)) {
	if format != ClipboardKTextPlain && format != ClipboardKImagePNG && format != ClipboardKURIList {
		done(nil, nil)
		return
	}
	c.getRich(ctx, ui.ClipboardKindPrimary, format, done)
}

// GetData retrieves the data of the format from the clipboard: ClipboardKTextPlain,
// ClipboardKImagePNG or ClipboardKURIList. For another format, data is nil at
// once. The data is also nil when the clipboard has nothing in the format. A
// clipboard without a backend calls done at once with ClipboardUnavailable.
//
// Flutter: services/clipboard.dart Clipboard.getData
func (c *Clipboard) GetData(ctx context.Context, format string, done func(data *ClipboardData, err error)) {
	if c == nil || c.backend == nil {
		done(nil, errClipboardUnavailable())
		return
	}
	if format == ClipboardKImagePNG || format == ClipboardKURIList {
		c.getRich(ctx, ui.ClipboardKindSystem, format, done)
		return
	}
	if format != ClipboardKTextPlain {
		done(nil, nil)
		return
	}
	c.backend.GetClipboardText(ctx, func(text string, ok bool, err error) {
		if err != nil || !ok {
			done(nil, err)
			return
		}
		done(&ClipboardData{Text: text}, nil)
	})
}

// HasStrings reports whether the clipboard has text that can be pasted; empty
// text does not count.
//
// Flutter: services/clipboard.dart Clipboard.hasStrings
func (c *Clipboard) HasStrings(ctx context.Context, done func(has bool, err error)) {
	c.GetData(ctx, ClipboardKTextPlain, func(data *ClipboardData, err error) {
		done(data != nil && data.Text != "", err)
	})
}

func errClipboardUnavailable() error {
	return errors.New("ClipboardUnavailable", "the clipboard has no backend")
}
