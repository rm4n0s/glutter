//go:build darwin && !ios

package a11y

import (
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// nsRect, nsRange and rect are the AppKit structures that the properties of an element take.
type (
	nsRect  = struct{ X, Y, W, H float64 }
	nsRange struct{ Location, Length uint64 }
	rect    = struct{ X, Y, W, H float64 }
)

// elementClass is the class of the elements: an NSAccessibilityElement that does what the assistive technologies ask
// (the press, the increment, the set value of a text field), registered once for the process.
var elementClass = sync.OnceValues(func() (*objcx.RegisteredClass, error) {
	action := func(name string, a ui.SemanticsAction) objcx.Method {
		return objcx.Method{Name: name, Fn: func(self objc.ID, _ objc.SEL) (ok bool) {
			if e, found := objcx.LookupAs[*element](objcx.HandleOf(self)); found {
				e.b.guard(func() { ok = e.b.perform(e.ref, a, nil) })
			}
			return ok
		}}
	}
	return objcx.RegisterClass(objcx.ClassDef{
		Name: "AccessibilityElement", Super: "NSAccessibilityElement",
		Methods: []objcx.Method{
			action("accessibilityPerformPress", ui.SemanticsActionTap),
			action("accessibilityPerformIncrement", ui.SemanticsActionIncrease),
			action("accessibilityPerformDecrement", ui.SemanticsActionDecrease),
			action("accessibilityPerformShowMenu", ui.SemanticsActionLongPress),
			action("accessibilityPerformCancel", ui.SemanticsActionDismiss),
			// The actions that the element lists are those of its node, not every one that the methods above could do.
			{Name: "accessibilityActionNames", Fn: func(self objc.ID, _ objc.SEL) objc.ID {
				e, found := objcx.LookupAs[*element](objcx.HandleOf(self))
				if !found {
					return 0
				}
				n := e.b.model.Node(e.ref)
				if n == nil {
					return 0
				}
				names := ActionNames(n)
				objs := make([]objc.ID, len(names))
				for i, s := range names {
					objs[i] = str(s)
				}
				arr := objcx.NewArray(objs...)
				for _, o := range objs {
					objcx.Release(o)
				}
				return objcx.Autorelease(arr)
			}},
			// Another name of the same actions, for the clients that ask by name.
			{Name: "accessibilityPerformAction:", Fn: func(self objc.ID, _ objc.SEL, name objc.ID) (ok bool) {
				if e, found := objcx.LookupAs[*element](objcx.HandleOf(self)); found {
					e.b.guard(func() {
						n := e.b.model.Node(e.ref)
						if n == nil {
							return
						}
						if a, supported := ActionFor(n, objcx.String(name)); supported {
							ok = e.b.perform(e.ref, a, nil)
						}
					})
				}
				return ok
			}},
			// A client that sets the value of a text field asks for the set text action; the bridge's own sets are
			// passed to the superclass.
			{Name: "setAccessibilityValue:", Fn: func(self objc.ID, _ objc.SEL, v objc.ID) {
				e, found := objcx.LookupAs[*element](objcx.HandleOf(self))
				if found && !e.internal {
					e.b.guard(func() {
						if n := e.b.model.Node(e.ref); n != nil && n.Flags.IsTextField && v != 0 {
							e.b.perform(e.ref, ui.SemanticsActionSetText, objcx.String(v))
						}
					})
					return
				}
				objc.SendSuper[struct{}](self, objcx.Sel("setAccessibilityValue:"), v)
			}},
			{Name: "setAccessibilitySelectedTextRange:", Fn: func(self objc.ID, _ objc.SEL, loc, length uint64) {
				e, found := objcx.LookupAs[*element](objcx.HandleOf(self))
				if found && !e.internal {
					e.b.guard(func() {
						if n := e.b.model.Node(e.ref); n != nil && n.Flags.IsTextField {
							e.b.perform(e.ref, ui.SemanticsActionSetSelection, SelectionArguments(n.Value, int(loc), int(length)))
						}
					})
					return
				}
				objc.SendSuper[struct{}](self, objcx.Sel("setAccessibilitySelectedTextRange:"), nsRange{Location: loc, Length: length})
			}},
		},
	})
})
