// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package gluttertest

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/widgets"
)

// Finder finds the elements of the tree under test that match some criterion.
// Make one with ByText, ByKey, ByType and the others of this file, combine them
// with Descendant and Ancestor, and give it to the methods of the Tester
// (Tap, Widget, Center) or to the matchers (FindsOneWidget). A finder looks at
// the tree each time it is used, so it can be made before the widget exists.
//
// Flutter: flutter_test/lib/src/finders.dart Finder
type Finder interface {
	// Describe says what the finder looks for, for the messages of a failure.
	//
	// Flutter: flutter_test/lib/src/finders.dart Finder.description
	Describe() string
	// Find returns the matching elements of the tree under test, in depth first
	// order (a parent before its children, siblings in order).
	//
	// Flutter: flutter_test/lib/src/finders.dart Finder.evaluate
	Find(t *Tester) []widgets.Element
}

// OffstageWidget is implemented by a widget that hides its subtree from the
// finders, such as Offstage (step 7.6): the finders skip the subtree unless they
// are made with IncludeOffstage.
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.skipOffstage
type OffstageWidget interface {
	widgets.Widget
	// IsOffstage is whether the subtree is hidden.
	IsOffstage() bool
}

// TextWidget is implemented by a widget that shows text: Text, RichText and
// EditableText (steps 7.7 and 11.4). ByText and ByTextContaining match their
// PlainText.
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.text
type TextWidget interface {
	widgets.Widget
	// PlainText is the text the widget shows, without styles.
	PlainText() string
}

// IconWidget is implemented by the Icon widget (step 7.7): ByIcon matches its
// IconValue with the icon it is given.
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.byIcon
type IconWidget interface {
	widgets.Widget
	// IconValue is the icon the widget shows (an IconData).
	IconValue() any
}

// TooltipWidget is implemented by the Tooltip widget (step 9.x): ByTooltip
// matches its message.
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.byTooltip
type TooltipWidget interface {
	widgets.Widget
	// TooltipMessage is the message the widget shows.
	TooltipMessage() string
}

// FindOpts are the optional parameters of the finders that look at every
// element.
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders
type FindOpts struct {
	// IncludeOffstage also looks inside the widgets that are offstage; Flutter's
	// skipOffstage: false.
	IncludeOffstage bool
	// FindRichText makes ByText and ByTextContaining also match RichText widgets,
	// not only Text; Flutter's findRichText: true.
	FindRichText bool
}

func findRichText(opts []FindOpts) bool { return len(opts) > 0 && opts[0].FindRichText }

// richTextWidget is implemented by RichText, which the text finders skip unless
// FindRichText is set.
type richTextWidget interface{ IsRichText() bool }

func textWidgetOf(e widgets.Element, withRich bool) (TextWidget, bool) {
	w, ok := e.Widget().(TextWidget)
	if !ok {
		return nil, false
	}
	if r, isRich := w.(richTextWidget); isRich && r.IsRichText() && !withRich {
		return nil, false
	}
	return w, true
}

func includeOffstage(opts []FindOpts) bool { return len(opts) > 0 && opts[0].IncludeOffstage }

// elementFinder is a Finder that tests every element of the tree.
type elementFinder struct {
	desc            string
	match           func(e widgets.Element) bool
	includeOffstage bool
}

func (f *elementFinder) Describe() string { return f.desc }

func (f *elementFinder) Find(t *Tester) []widgets.Element {
	var found []widgets.Element
	root := t.Binding.RootElement()
	if root == nil {
		return nil
	}
	walk(root, f.includeOffstage, func(e widgets.Element) {
		if f.match(e) {
			found = append(found, e)
		}
	})
	return found
}

// walk visits the elements below e, e included, in depth first order. Unless
// includeOffstage it visits only the on stage children (DebugVisitOnstageChildren),
// as Flutter's finders do.
func walk(e widgets.Element, includeOffstage bool, visit func(widgets.Element)) {
	visit(e)
	if includeOffstage {
		e.VisitChildren(func(c widgets.Element) { walk(c, includeOffstage, visit) })
		return
	}
	if isOffstage(e) {
		return
	}
	e.DebugVisitOnstageChildren(func(c widgets.Element) { walk(c, includeOffstage, visit) })
}

func isOffstage(e widgets.Element) bool {
	o, ok := e.Widget().(OffstageWidget)
	return ok && o.IsOffstage()
}

// ByText finds the widgets that show exactly this text: those that implement
// TextWidget (Text, and RichText with FindRichText).
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.text
func ByText(text string, opts ...FindOpts) Finder {
	return &elementFinder{
		desc: fmt.Sprintf("text %q", text),
		match: func(e widgets.Element) bool {
			w, ok := textWidgetOf(e, findRichText(opts))
			return ok && w.PlainText() == text
		},
		includeOffstage: includeOffstage(opts),
	}
}

// ByTextContaining finds the widgets that show a text with this substring.
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.textContaining
func ByTextContaining(substring string, opts ...FindOpts) Finder {
	return &elementFinder{
		desc: fmt.Sprintf("text containing %q", substring),
		match: func(e widgets.Element) bool {
			w, ok := textWidgetOf(e, findRichText(opts))
			return ok && strings.Contains(w.PlainText(), substring)
		},
		includeOffstage: includeOffstage(opts),
	}
}

// ByKey finds the widgets that have the key.
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.byKey
func ByKey(key foundation.Key, opts ...FindOpts) Finder {
	return &elementFinder{
		desc:            "key " + key.String(),
		match:           func(e widgets.Element) bool { return e.Widget().Key() == key },
		includeOffstage: includeOffstage(opts),
	}
}

// ByType finds the widgets of the type T, such as ByType[*Button]().
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.byType
func ByType[T widgets.Widget](opts ...FindOpts) Finder {
	return &elementFinder{
		desc:            "type " + reflect.TypeFor[T]().String(),
		match:           func(e widgets.Element) bool { _, ok := e.Widget().(T); return ok },
		includeOffstage: includeOffstage(opts),
	}
}

// ByWidget finds the elements of this very widget (the same pointer).
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.byWidget
func ByWidget(w widgets.Widget, opts ...FindOpts) Finder {
	return &elementFinder{
		desc:            "widget " + widgets.WidgetToStringShort(w),
		match:           func(e widgets.Element) bool { return e.Widget() == w },
		includeOffstage: includeOffstage(opts),
	}
}

// ByPredicate finds the widgets for which the function is true. The description
// is used in the messages of a failure.
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.byWidgetPredicate
func ByPredicate(description string, fn func(w widgets.Widget) bool, opts ...FindOpts) Finder {
	return &elementFinder{
		desc:            description,
		match:           func(e widgets.Element) bool { return fn(e.Widget()) },
		includeOffstage: includeOffstage(opts),
	}
}

// ByElementPredicate finds the elements for which the function is true.
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.byElementPredicate
func ByElementPredicate(description string, fn func(e widgets.Element) bool, opts ...FindOpts) Finder {
	return &elementFinder{desc: description, match: fn, includeOffstage: includeOffstage(opts)}
}

// ByIcon finds the widgets that implement IconWidget and show this icon.
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.byIcon
func ByIcon(icon any, opts ...FindOpts) Finder {
	return &elementFinder{
		desc: fmt.Sprintf("icon %v", icon),
		match: func(e widgets.Element) bool {
			w, ok := e.Widget().(IconWidget)
			return ok && w.IconValue() == icon
		},
		includeOffstage: includeOffstage(opts),
	}
}

// ByTooltip finds the widgets that implement TooltipWidget and have this
// message.
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.byTooltip
func ByTooltip(message string, opts ...FindOpts) Finder {
	return &elementFinder{
		desc: fmt.Sprintf("tooltip %q", message),
		match: func(e widgets.Element) bool {
			w, ok := e.Widget().(TooltipWidget)
			return ok && w.TooltipMessage() == message
		},
		includeOffstage: includeOffstage(opts),
	}
}

// BySemanticsLabel finds the elements whose render object has a semantics node
// with exactly this label. Ask for semantics with Tester.EnsureSemantics and
// pump a frame first. It reads RenderObject.DebugSemantics, so it finds nothing
// in a release or profile build (Flutter's tests run in debug mode).
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.bySemanticsLabel
func BySemanticsLabel(label string, opts ...FindOpts) Finder {
	return bySemantics(fmt.Sprintf("semantics label %q", label), func(l string) bool { return l == label }, opts)
}

// BySemanticsLabelMatching is BySemanticsLabel with a regular expression.
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.bySemanticsLabel
func BySemanticsLabelMatching(pattern *regexp.Regexp, opts ...FindOpts) Finder {
	return bySemantics("semantics label matching "+pattern.String(), pattern.MatchString, opts)
}

func bySemantics(desc string, matches func(label string) bool, opts []FindOpts) Finder {
	return &elementFinder{
		desc: desc,
		match: func(e widgets.Element) bool {
			// An element that does not own a render object itself (the render
			// object of a component element is the one of its descendant) is
			// skipped, so that the match is the element that made the node.
			if _, ok := e.(widgets.RenderObjectElementSelf); !ok {
				return false
			}
			ro := e.RenderObject()
			if ro == nil {
				return false
			}
			node := ro.AsRenderObject().DebugSemantics()
			return node != nil && matches(node.Label())
		},
		includeOffstage: includeOffstage(opts),
	}
}

// Everything finds every element of the tree.
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.everything
func Everything(opts ...FindOpts) Finder {
	return &elementFinder{desc: "every element", match: func(widgets.Element) bool { return true }, includeOffstage: includeOffstage(opts)}
}

// subsetFinder is a Finder made from the result of another.
type subsetFinder struct {
	of     Finder
	desc   string
	choose func(found []widgets.Element) []widgets.Element
}

func (f *subsetFinder) Describe() string { return f.desc }
func (f *subsetFinder) Find(t *Tester) []widgets.Element {
	return f.choose(f.of.Find(t))
}

// First is the first match of the finder, in depth first order.
//
// Flutter: flutter_test/lib/src/finders.dart FinderBase.first
func First(f Finder) Finder { return At(f, 0) }

// Last is the last match of the finder.
//
// Flutter: flutter_test/lib/src/finders.dart FinderBase.last
func Last(f Finder) Finder {
	return &subsetFinder{of: f, desc: "the last of " + f.Describe(), choose: func(found []widgets.Element) []widgets.Element {
		if len(found) == 0 {
			return nil
		}
		return found[len(found)-1:]
	}}
}

// At is the match of the finder at the index, or nothing when there are not so
// many.
//
// Flutter: flutter_test/lib/src/finders.dart FinderBase.at
func At(f Finder, index int) Finder {
	return &subsetFinder{of: f, desc: fmt.Sprintf("match %d of %s", index, f.Describe()), choose: func(found []widgets.Element) []widgets.Element {
		if index < 0 || index >= len(found) {
			return nil
		}
		return found[index : index+1]
	}}
}

// DescendantOpts are the optional parameters of Descendant and Ancestor.
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.descendant
type DescendantOpts struct {
	// MatchRoot also counts the element of the first finder itself.
	MatchRoot bool
}

type relationFinder struct {
	desc     string
	of       Finder
	matching Finder
	root     bool
	// below is true for Descendant, false for Ancestor.
	below bool
}

func (f *relationFinder) Describe() string { return f.desc }

func (f *relationFinder) Find(t *Tester) []widgets.Element {
	matches := map[widgets.Element]bool{}
	for _, e := range f.matching.Find(t) {
		matches[e] = true
	}
	var result []widgets.Element
	seen := map[widgets.Element]bool{}
	add := func(e widgets.Element) {
		if matches[e] && !seen[e] {
			seen[e] = true
			result = append(result, e)
		}
	}
	for _, base := range f.of.Find(t) {
		if f.below {
			// Every element under the base, in depth first order.
			var visit func(e widgets.Element)
			visit = func(e widgets.Element) {
				add(e)
				e.VisitChildren(visit)
			}
			if f.root {
				visit(base)
			} else {
				base.VisitChildren(visit)
			}
			continue
		}
		if f.root {
			add(base)
		}
		base.VisitAncestorElements(func(a widgets.Element) bool { add(a); return true })
	}
	return result
}

// Descendant finds the elements of the second finder that are below an element
// of the first (or are one, with MatchRoot).
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.descendant
func Descendant(of, matching Finder, opts ...DescendantOpts) Finder {
	var o DescendantOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	return &relationFinder{desc: fmt.Sprintf("%s descending from %s", matching.Describe(), of.Describe()),
		of: of, matching: matching, root: o.MatchRoot, below: true}
}

// Ancestor finds the elements of the second finder that are above an element of
// the first (or are one, with MatchRoot), the closest first.
//
// Flutter: flutter_test/lib/src/finders.dart CommonFinders.ancestor
func Ancestor(of, matching Finder, opts ...DescendantOpts) Finder {
	var o DescendantOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	return &relationFinder{desc: fmt.Sprintf("%s that is an ancestor of %s", matching.Describe(), of.Describe()),
		of: of, matching: matching, root: o.MatchRoot}
}
