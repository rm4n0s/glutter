package platform

import "github.com/rm4n0s/glutter/ui"

// MenuBar is implemented by a Platform whose apps have a menu bar that the app fills (macOS): the engine
// finds it with a type assertion and gives it the menus of a PlatformMenuBar. Called on the UI thread.
//
// Flutter: no counterpart (the plugin of the flutter/menu channel, shell/platform/darwin/macos/framework/Source/FlutterMenuPlugin.mm)
type MenuBar interface {
	// SetMenus replaces the menus of the menu bar with the top-level menus of the app, or with the
	// platform's default menu bar when the list is empty. The handler gets the callbacks of the items
	// until the next call.
	SetMenus(menus []ui.PlatformMenuItemData, h ui.PlatformMenuHandler)
}
