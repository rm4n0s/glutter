// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"reflect"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// DoesDependOnInheritedElement is whether this element depends on the inherited
// element.
//
// Flutter: widgets/framework.dart Element.doesDependOnInheritedElement
func (b *ElementBase) DoesDependOnInheritedElement(ancestor Element) bool {
	_, ok := b.dependencies[ancestor]
	return ok
}

// DependOnInheritedElement registers this element as a dependent of the
// inherited element and returns its widget.
//
// Flutter: widgets/framework.dart Element.dependOnInheritedElement
func (b *ElementBase) DependOnInheritedElement(ancestor InheritedElementSelf, aspect any) Widget {
	if b.dependencies == nil {
		b.dependencies = map[Element]struct{}{}
	}
	b.dependencies[ancestor] = struct{}{}
	ancestor.UpdateDependencies(b.self, aspect)
	return ancestor.AsElement().widget
}

func (b *ElementBase) dependOnInheritedType(t reflect.Type, aspect any) Widget {
	b.debugCheckStateIsActiveForAncestorLookup()
	if ancestor, ok := b.inheritedElements[t]; ok {
		return b.self.DependOnInheritedElement(ancestor.(InheritedElementSelf), aspect)
	}
	b.hadUnsatisfiedDependencies = true
	return nil
}

func (b *ElementBase) inheritedElementOfType(t reflect.Type) Element {
	b.debugCheckStateIsActiveForAncestorLookup()
	return b.inheritedElements[t]
}

// DebugDependencies are the inherited elements this element depends on.
//
// Flutter: widgets/framework.dart Element._dependencies
func (b *ElementBase) DebugDependencies() []Element {
	out := make([]Element, 0, len(b.dependencies))
	for d := range b.dependencies {
		out = append(out, d)
	}
	sortElementsByShortName(out)
	return out
}

// DependOnInheritedElement checks, in debug builds, that InitState has run, then
// registers the dependency. It panics with the tag
// DependOnInheritedBeforeInitState. (Flutter also rejects a dependency after
// dispose; here the element is already defunct then, and the lookup panics with
// AncestorLookupOnInactiveElement.)
//
// Flutter: widgets/framework.dart StatefulElement.dependOnInheritedElement
func (e *StatefulElement) DependOnInheritedElement(ancestor InheritedElementSelf, aspect any) Widget {
	if foundation.DebugMode && e.state != nil {
		target := foundation.RuntimeTypeName(ancestor.AsElement().widget)
		switch e.state.stateInternals().data().lifecycle {
		case StateLifecycleCreated:
			panic(errors.New("DependOnInheritedBeforeInitState",
				"DependOn or DependOnInheritedElement was called before InitState completed; when an inherited widget changes, its dependent widgets are rebuilt, but if the dependent widget's reference to the inherited widget is in a constructor or an InitState method, then the rebuilt dependent widget will not reflect the changes; typically references to inherited widgets should occur in Build methods, or initialization based on them can be placed in DidChangeDependencies, which is called after InitState and whenever the dependencies change thereafter",
				"inherited", target, "state", foundation.RuntimeTypeName(e.state)))
		}
	}
	return e.ComponentElement.DependOnInheritedElement(ancestor, aspect)
}
