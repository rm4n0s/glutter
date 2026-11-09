package cupertino

import (
	"fmt"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoUserInterfaceLevelData is the elevation of a part of the interface: the
// colors of a CupertinoDynamicColor change between the base level and the elevated
// one (a sheet or a popup above the rest of the app).
//
// Flutter: cupertino/interface_level.dart CupertinoUserInterfaceLevelData
type CupertinoUserInterfaceLevelData int

const (
	// CupertinoUserInterfaceLevelDataUnspecified is the base level: a CupertinoUserInterfaceLevel
	// widget needs a level, and the unspecified one stands for the base.
	//
	// Flutter: no counterpart
	CupertinoUserInterfaceLevelDataUnspecified CupertinoUserInterfaceLevelData = iota
	// CupertinoUserInterfaceLevelBase is the level of the interface that has nothing above
	// it: the color of a CupertinoDynamicColor that does not depend on the level.
	//
	// Flutter: cupertino/interface_level.dart CupertinoUserInterfaceLevelData.base
	CupertinoUserInterfaceLevelBase
	// CupertinoUserInterfaceLevelElevated is the level of the interface that is above the
	// base one, such as a sheet.
	//
	// Flutter: cupertino/interface_level.dart CupertinoUserInterfaceLevelData.elevated
	CupertinoUserInterfaceLevelElevated
)

var cupertinoUserInterfaceLevelNames = [...]string{"base", "base", "elevated"}

// Name is the name of the level in Dart: base or elevated.
//
// Flutter: cupertino/interface_level.dart CupertinoUserInterfaceLevelData
func (l CupertinoUserInterfaceLevelData) Name() string {
	if l < 0 || int(l) >= len(cupertinoUserInterfaceLevelNames) {
		return fmt.Sprintf("CupertinoUserInterfaceLevelData(%d)", int(l))
	}
	return cupertinoUserInterfaceLevelNames[l]
}

// CupertinoUserInterfaceLevel establishes a subtree in which CupertinoDynamicColors
// resolve to the colors of the given interface level. CupertinoApp and the routes of
// sheets and popups introduce it.
//
// Flutter: cupertino/interface_level.dart CupertinoUserInterfaceLevel
type CupertinoUserInterfaceLevel struct {
	widgets.InheritedWidget
	data CupertinoUserInterfaceLevelData
}

// NewCupertinoUserInterfaceLevel creates the widget that gives the level to the widgets
// below it.
//
// Flutter: cupertino/interface_level.dart CupertinoUserInterfaceLevel.new
func NewCupertinoUserInterfaceLevel(data CupertinoUserInterfaceLevelData, child widgets.Widget, opts ...widgets.WidgetOpts) *CupertinoUserInterfaceLevel {
	if data == CupertinoUserInterfaceLevelDataUnspecified {
		data = CupertinoUserInterfaceLevelBase
	}
	return &CupertinoUserInterfaceLevel{widgets.NewInheritedWidgetBase(child, opts...), data}
}

// Data is the level that the widget gives.
//
// Flutter: cupertino/interface_level.dart CupertinoUserInterfaceLevel._data
func (w *CupertinoUserInterfaceLevel) Data() CupertinoUserInterfaceLevelData { return w.data }

// UpdateShouldNotify is whether the level changed.
//
// Flutter: cupertino/interface_level.dart CupertinoUserInterfaceLevel.updateShouldNotify
func (w *CupertinoUserInterfaceLevel) UpdateShouldNotify(old widgets.Widget) bool {
	return old.(*CupertinoUserInterfaceLevel).data != w.data
}

// CupertinoUserInterfaceLevelOf is the level of the closest CupertinoUserInterfaceLevel
// above the context. The context depends on it. It panics with the tag
// CupertinoUserInterfaceLevelMissing when there is none.
//
// Flutter: cupertino/interface_level.dart CupertinoUserInterfaceLevel.of
func CupertinoUserInterfaceLevelOf(bc widgets.BuildContext) CupertinoUserInterfaceLevelData {
	if w, ok := widgets.DependOn[*CupertinoUserInterfaceLevel](bc); ok {
		return w.data
	}
	panic(errors.New(tagCupertinoUserInterfaceLevelMissing,
		"CupertinoUserInterfaceLevelOf() called with a context that does not contain a CupertinoUserInterfaceLevel; "+
			"no CupertinoUserInterfaceLevel ancestor could be found starting from the context that was passed; "+
			"this can happen because you do not have a WidgetsApp or MaterialApp widget (those widgets introduce a "+
			"CupertinoUserInterfaceLevel), or it can happen if the context you use comes from a widget above those widgets",
		"context", bc.(widgets.Element).String()))
}

// CupertinoUserInterfaceLevelMaybeOf is the level of the closest CupertinoUserInterfaceLevel
// above the context, or false when there is none. The context depends on it.
//
// Flutter: cupertino/interface_level.dart CupertinoUserInterfaceLevel.maybeOf
func CupertinoUserInterfaceLevelMaybeOf(bc widgets.BuildContext) (CupertinoUserInterfaceLevelData, bool) {
	w, ok := widgets.DependOn[*CupertinoUserInterfaceLevel](bc)
	if !ok {
		return CupertinoUserInterfaceLevelDataUnspecified, false
	}
	return w.data, true
}

// DebugFillProperties adds the level.
//
// Flutter: cupertino/interface_level.dart CupertinoUserInterfaceLevel.debugFillProperties
func (w *CupertinoUserInterfaceLevel) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewEnumProperty("user interface level", foundation.Some(w.data)))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoUserInterfaceLevelMissing = "CupertinoUserInterfaceLevelMissing"
)
