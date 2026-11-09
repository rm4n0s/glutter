package ui

// PlatformMenuItemData is one entry of the menus that the app asks the platform to show in its menu
// bar, in the form that the platform needs: the Go form of the maps that Flutter sends on its
// flutter/menu channel. A divider has IsDivider; an item that the platform provides (About, Quit, the
// Window menu's items) has HasProvided and the number of its type; any other has a label, a state and
// maybe a shortcut and children.
//
// Flutter: widgets/platform_menu_bar.dart (the map that PlatformMenuItem.toChannelRepresentation makes)
type PlatformMenuItemData struct {
	// ID names the item in the callbacks of PlatformMenuHandler.
	ID int
	// Label is the text of the item; "APP_NAME" in it stands for the name of the app.
	Label string
	// Tooltip is the tooltip of the item when HasTooltip.
	Tooltip    string
	HasTooltip bool
	// Enabled says the user can choose the item.
	Enabled bool
	// IsDivider makes the item a separator.
	IsDivider bool
	// HasProvided says the item is one that the platform provides; ProvidedType is the index of the
	// type (widgets.PlatformProvidedMenuItemType minus its unspecified zero value).
	HasProvided  bool
	ProvidedType int
	// ShortcutCharacter is the character of the shortcut when HasShortcutCharacter, and ShortcutTrigger
	// is the logical key id of the trigger key when HasShortcutTrigger; ShortcutModifiers has the
	// bits PlatformMenuShortcutModifier*.
	ShortcutCharacter    string
	HasShortcutCharacter bool
	ShortcutTrigger      uint64
	HasShortcutTrigger   bool
	ShortcutModifiers    int
	// Children are the items of the submenu of a menu; HasChildren says the item is a menu (it may have
	// no children).
	Children    []PlatformMenuItemData
	HasChildren bool
}

// The bits of PlatformMenuItemData.ShortcutModifiers, the values of Flutter's ShortcutSerialization.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization._shortcutModifierMeta
const (
	// PlatformMenuShortcutModifierMeta is the Meta key (Command on macOS).
	PlatformMenuShortcutModifierMeta = 1 << 0
	// PlatformMenuShortcutModifierShift is the Shift key.
	PlatformMenuShortcutModifierShift = 1 << 1
	// PlatformMenuShortcutModifierAlt is the Alt key (Option on macOS).
	PlatformMenuShortcutModifierAlt = 1 << 2
	// PlatformMenuShortcutModifierControl is the Control key.
	PlatformMenuShortcutModifierControl = 1 << 3
)

// PlatformMenuHandler is what the platform tells the app about its menus, on the UI thread: the id is
// that of a PlatformMenuItemData.
//
// Flutter: widgets/platform_menu_bar.dart DefaultPlatformMenuDelegate._methodCallHandler
type PlatformMenuHandler interface {
	// MenuSelected says the user chose the item.
	MenuSelected(id int)
	// MenuOpened says the menu opened.
	MenuOpened(id int)
	// MenuClosed says the menu closed.
	MenuClosed(id int)
}

// PlatformMenuHost is what the engine implements to put the menus of the app in the menu bar of the
// platform. The dispatcher uses it when the host has it; a host without it has no menu bar.
//
// Flutter: no counterpart (the engine's side of the flutter/menu channel)
type PlatformMenuHost interface {
	// SetPlatformMenus replaces the menus of the menu bar with the top-level menus; an empty list
	// removes them and leaves the platform's default menu bar. The handler gets the callbacks of the items
	// until the next call. It reports whether the platform has a menu bar.
	SetPlatformMenus(menus []PlatformMenuItemData, h PlatformMenuHandler) bool
}

// SetPlatformMenus puts the menus in the platform's menu bar (see PlatformMenuHost) and reports
// whether the platform has one. It does nothing, and reports false, on a host without a menu bar.
//
// Flutter: no counterpart (the flutter/menu channel's Menu.setMenus call)
func (d *PlatformDispatcher) SetPlatformMenus(menus []PlatformMenuItemData, h PlatformMenuHandler) bool {
	if host, ok := d.host.(PlatformMenuHost); ok {
		return host.SetPlatformMenus(menus, h)
	}
	return false
}
