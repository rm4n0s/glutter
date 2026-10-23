package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// firstNonUniqueKey is the first key that two of the widgets have, or nil.
func firstNonUniqueKey(widgets []Widget) foundation.Key {
	seen := map[foundation.Key]struct{}{}
	for _, w := range widgets {
		k := w.Key()
		if k == nil {
			continue
		}
		if _, ok := seen[k]; ok {
			return k
		}
		seen[k] = struct{}{}
	}
	return nil
}

// DebugChildrenHaveDuplicateKeys panics, in debug builds, with the tag
// ChildrenHaveDuplicateKeys when two of the children of the parent have the
// same key. It always returns false, so it can be used in a condition.
//
// Flutter: widgets/debug.dart debugChildrenHaveDuplicateKeys
func DebugChildrenHaveDuplicateKeys(parent Widget, children []Widget) bool {
	if foundation.DebugMode {
		if k := firstNonUniqueKey(children); k != nil {
			panic(errors.New(tagChildrenHaveDuplicateKeys,
				"Duplicate keys found.\nIf multiple keyed widgets exist as children of another widget, they must have unique keys.\n"+
					WidgetToStringShort(parent)+" has multiple children with key "+k.String()+".",
				"parent", WidgetToStringShort(parent), "key", k.String()))
		}
	}
	return false
}

// DebugItemsHaveDuplicateKeys panics, in debug builds, with the tag
// ItemsHaveDuplicateKeys when two of the widgets have the same key. It always
// returns false.
//
// Flutter: widgets/debug.dart debugItemsHaveDuplicateKeys
func DebugItemsHaveDuplicateKeys(items []Widget) bool {
	if foundation.DebugMode {
		if k := firstNonUniqueKey(items); k != nil {
			panic(errors.New(tagItemsHaveDuplicateKeys, "Duplicate key found.", "key", k.String()))
		}
	}
	return false
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagChildrenHaveDuplicateKeys = "ChildrenHaveDuplicateKeys"
	tagItemsHaveDuplicateKeys    = "ItemsHaveDuplicateKeys"
)
