// Package macos is the platform backend of macOS: AppKit through purego and its objc package, with no cgo
// (SPEC §29). It implements internal/platform.Platform: the app and its run loop on the main thread, the
// windows, the displays, input, text input, the pasteboard, settings, menus, the desktop services, drag
// and drop and accessibility. It is filled in by the steps of M33; until step 33.2 it is empty.
//
// Its files are for darwin without ios (`darwin && !ios`); this file keeps the package listed on every OS.
// The helpers that macOS and iOS share are in internal/platform/darwin.
//
// Flutter: engine/src/flutter/shell/platform/darwin/macos/framework/Source
package macos
