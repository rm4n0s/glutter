// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"maps"
	"reflect"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
)

// ProxyWidget is a widget that has exactly one child and builds nothing itself:
// it exists to give its element a job. InheritedWidget and ParentDataWidget are
// proxy widgets. A proxy widget of your own makes its element with
// CreateElement (see ElementCreator) and embeds ProxyElement in it.
//
// Flutter: widgets/framework.dart ProxyWidget
type ProxyWidget struct {
	WidgetBase
	child Widget
}

// NewProxyWidgetBase creates the embedded base of a proxy widget.
//
// Flutter: widgets/framework.dart ProxyWidget.new
func NewProxyWidgetBase(child Widget, opts ...WidgetOpts) ProxyWidget {
	return ProxyWidget{NewWidgetBase(opts...), child}
}

// Child is the widget below this one in the tree.
//
// Flutter: widgets/framework.dart ProxyWidget.child
func (p *ProxyWidget) Child() Widget { return p.child }

func (*ProxyWidget) isProxy() {}

// proxyWidget is any widget with the ProxyWidget base.
type proxyWidget interface {
	Widget
	Child() Widget
	isProxy()
}

// ProxyElementSelf is what a proxy element needs from the outer type.
type ProxyElementSelf interface {
	ComponentElementSelf
	// NotifyClients tells the dependents of the widget about the change, when the
	// widget changed in a way that matters to them (see Updated).
	//
	// Flutter: widgets/framework.dart ProxyElement.notifyClients
	NotifyClients(oldWidget Widget)
	// Updated is called when the widget changes, before the element rebuilds. It
	// calls NotifyClients by default.
	//
	// Flutter: widgets/framework.dart ProxyElement.updated
	Updated(oldWidget Widget)
}

// ProxyElement is the element of a ProxyWidget: its child is the widget's
// child. Embed it, call InitProxyElement and implement NotifyClients.
//
// Flutter: widgets/framework.dart ProxyElement
type ProxyElement struct {
	ComponentElement
	pself ProxyElementSelf
}

// InitProxyElement sets up a proxy element; self is the outer element.
//
// Flutter: widgets/framework.dart ProxyElement.new
func InitProxyElement(p *ProxyElement, self ProxyElementSelf, widget proxyWidget) {
	InitComponentElement(&p.ComponentElement, self, widget)
	p.pself = self
}

// Build is the child of the widget.
//
// Flutter: widgets/framework.dart ProxyElement.build
func (p *ProxyElement) Build() Widget { return p.widget.(proxyWidget).Child() }

// Update replaces the widget, tells the clients and rebuilds.
//
// Flutter: widgets/framework.dart ProxyElement.update
func (p *ProxyElement) Update(newWidget Widget) {
	oldWidget := p.widget
	p.ComponentElement.Update(newWidget)
	p.pself.Updated(oldWidget)
	p.Rebuild(RebuildOpts{Force: true})
}

// Updated calls NotifyClients.
//
// Flutter: widgets/framework.dart ProxyElement.updated
func (p *ProxyElement) Updated(oldWidget Widget) { p.pself.NotifyClients(oldWidget) }

// InheritedWidget is a widget that passes information down the tree: the
// widgets below it can look it up with DependOn[T](bc) and are rebuilt when it
// changes. Embed it in a widget type and implement UpdateShouldNotify:
//
//	type AppConfig struct {
//		widgets.InheritedWidget
//		color Color
//	}
//
//	func (c *AppConfig) UpdateShouldNotify(old widgets.Widget) bool {
//		return old.(*AppConfig).color != c.color
//	}
//
// Flutter: widgets/framework.dart InheritedWidget
type InheritedWidget struct{ ProxyWidget }

// NewInheritedWidgetBase creates the embedded base of an inherited widget.
//
// Flutter: widgets/framework.dart InheritedWidget.new
func NewInheritedWidgetBase(child Widget, opts ...WidgetOpts) InheritedWidget {
	return InheritedWidget{NewProxyWidgetBase(child, opts...)}
}

func (*InheritedWidget) isInherited() {}

// InheritedWidgetInterface is what an inherited widget implements: the embedded
// InheritedWidget and UpdateShouldNotify. It is the constraint of DependOn.
//
// Flutter: widgets/framework.dart InheritedWidget
type InheritedWidgetInterface interface {
	proxyWidget
	// UpdateShouldNotify is whether the widgets that depend on this one are
	// rebuilt after the widget is replaced by this one. oldWidget has the same
	// type.
	//
	// Flutter: widgets/framework.dart InheritedWidget.updateShouldNotify
	UpdateShouldNotify(oldWidget Widget) bool
	isInherited()
}

// ParentDataWidgetOf is the embedded base of a widget that writes parent data
// of type T to the render objects of the widgets below it, such as Flutter's
// Positioned or Flexible. Embed it and implement ApplyParentData:
//
//	type Tag struct {
//		widgets.ParentDataWidgetOf[*rendering.BoxParentData]
//	}
//
//	func (t *Tag) ApplyParentData(renderObject rendering.RenderObject) { ... }
//
// Optional methods are DebugCanApplyOutOfTurn() bool,
// DebugTypicalAncestorWidgetClass() string and
// DebugTypicalAncestorWidgetDescription() string.
//
// Flutter: widgets/framework.dart ParentDataWidget
type ParentDataWidgetOf[T rendering.ParentData] struct{ ProxyWidget }

// NewParentDataWidgetBase creates the embedded base of a parent data widget.
//
// Flutter: widgets/framework.dart ParentDataWidget.new
func NewParentDataWidgetBase[T rendering.ParentData](child Widget, opts ...WidgetOpts) ParentDataWidgetOf[T] {
	return ParentDataWidgetOf[T]{NewProxyWidgetBase(child, opts...)}
}

func (*ParentDataWidgetOf[T]) isParentDataWidget() {}

// DebugIsValidRenderObject is whether the parent data of the render object is a
// T, so that ApplyParentData can write to it.
//
// Flutter: widgets/framework.dart ParentDataWidget.debugIsValidRenderObject
func (*ParentDataWidgetOf[T]) DebugIsValidRenderObject(renderObject rendering.RenderObject) bool {
	_, ok := renderObject.ParentData().(T)
	return ok
}

// DebugParentDataType is the name of T.
//
// Flutter: widgets/framework.dart ParentDataElement.debugParentDataType
func (*ParentDataWidgetOf[T]) DebugParentDataType() string {
	return foundation.RuntimeTypeName(*new(T))
}

// parentDataWidget is what the framework needs of a parent data widget.
type parentDataWidget interface {
	proxyWidget
	// ApplyParentData writes the data to the parent data of the render object.
	//
	// Flutter: widgets/framework.dart ParentDataWidget.applyParentData
	ApplyParentData(renderObject rendering.RenderObject)
	DebugIsValidRenderObject(renderObject rendering.RenderObject) bool
	DebugParentDataType() string
	isParentDataWidget()
}

// ParentDataElement is the element of a ParentDataWidget: when the widget
// changes it writes the parent data again to the render objects below.
//
// Flutter: widgets/framework.dart ParentDataElement
type ParentDataElement struct {
	ProxyElement
}

// NewParentDataElement creates the element of a parent data widget.
//
// Flutter: widgets/framework.dart ParentDataElement.new
func NewParentDataElement(w parentDataWidget) *ParentDataElement {
	e := &ParentDataElement{}
	InitProxyElement(&e.ProxyElement, e, w)
	return e
}

// DebugParentDataType is the name of the parent data type of the widget.
//
// Flutter: widgets/framework.dart ParentDataElement.debugParentDataType
func (e *ParentDataElement) DebugParentDataType() string {
	return e.widget.(parentDataWidget).DebugParentDataType()
}

func (e *ParentDataElement) applyParentData(w parentDataWidget) {
	var applyToChild func(child Element)
	applyToChild = func(child Element) {
		if ro, ok := child.(renderObjectElementLike); ok {
			ro.updateParentData(w)
		} else if next := child.RenderObjectAttachingChild(); next != nil {
			applyToChild(next)
		}
	}
	if child := e.RenderObjectAttachingChild(); child != nil {
		applyToChild(child)
	}
}

// ApplyWidgetOutOfTurn applies the parent data of a new widget to the render
// objects below, without rebuilding. The widget must say it can be applied
// that way (DebugCanApplyOutOfTurn) and have the same child.
//
// Flutter: widgets/framework.dart ParentDataElement.applyWidgetOutOfTurn
func (e *ParentDataElement) ApplyWidgetOutOfTurn(newWidget Widget) {
	nw := newWidget.(parentDataWidget)
	if foundation.DebugMode {
		if c, ok := nw.(interface{ DebugCanApplyOutOfTurn() bool }); !ok || !c.DebugCanApplyOutOfTurn() {
			panic(errors.New("ParentDataOutOfTurnNotAllowed",
				"the widget cannot apply its parent data out of turn; it must say so with DebugCanApplyOutOfTurn",
				"widget", foundation.RuntimeTypeName(newWidget)))
		}
		if nw.Child() != e.widget.(proxyWidget).Child() {
			panic(errors.New("ParentDataOutOfTurnChildChanged",
				"a parent data widget applied out of turn must have the same child as the one it replaces",
				"widget", foundation.RuntimeTypeName(newWidget)))
		}
	}
	e.applyParentData(nw)
}

// NotifyClients writes the parent data to the render objects below.
//
// Flutter: widgets/framework.dart ParentDataElement.notifyClients
func (e *ParentDataElement) NotifyClients(Widget) { e.applyParentData(e.widget.(parentDataWidget)) }

// InheritedElementSelf is what an inherited element needs from the outer type;
// InheritedElement implements all of it, so an element that embeds it overrides
// only the hooks it changes (InheritedModel and InheritedNotifier do).
type InheritedElementSelf interface {
	ProxyElementSelf
	// GetDependencies is the information the dependent keeps about what it
	// depends on (an aspect, for models); nil for none.
	//
	// Flutter: widgets/framework.dart InheritedElement.getDependencies
	GetDependencies(dependent Element) any
	// SetDependencies sets that information.
	//
	// Flutter: widgets/framework.dart InheritedElement.setDependencies
	SetDependencies(dependent Element, value any)
	// UpdateDependencies is called when the dependent starts to depend on this
	// element, with the aspect it asked for.
	//
	// Flutter: widgets/framework.dart InheritedElement.updateDependencies
	UpdateDependencies(dependent Element, aspect any)
	// NotifyDependent tells one dependent that the widget changed.
	//
	// Flutter: widgets/framework.dart InheritedElement.notifyDependent
	NotifyDependent(oldWidget Widget, dependent Element)
	// RemoveDependent forgets the dependent.
	//
	// Flutter: widgets/framework.dart InheritedElement.removeDependent
	RemoveDependent(dependent Element)
	asInheritedElement() *InheritedElement
}

// InheritedElement is the element of an InheritedWidget. It keeps the
// elements that depend on its widget.
//
// Flutter: widgets/framework.dart InheritedElement
type InheritedElement struct {
	ProxyElement
	iself      InheritedElementSelf
	dependents orderedMap[Element, any]
}

// NewInheritedElement creates the element of an inherited widget.
//
// Flutter: widgets/framework.dart InheritedElement.new
func NewInheritedElement(w InheritedWidgetInterface) *InheritedElement {
	e := &InheritedElement{}
	InitInheritedElement(e, e, w)
	return e
}

// InitInheritedElement sets up an inherited element; self is the outer element.
//
// Flutter: widgets/framework.dart InheritedElement.new
func InitInheritedElement(e *InheritedElement, self InheritedElementSelf, w InheritedWidgetInterface) {
	InitProxyElement(&e.ProxyElement, self, w)
	e.iself = self
}

func (e *InheritedElement) asInheritedElement() *InheritedElement { return e }

// updateInheritance adds this element to the inherited elements above, under
// the type of its widget.
//
// Flutter: widgets/framework.dart InheritedElement._updateInheritance
func (e *InheritedElement) updateInheritance() {
	incoming := e.parent
	next := map[reflect.Type]Element{}
	if incoming != nil {
		maps.Copy(next, incoming.AsElement().inheritedElements)
	}
	next[reflect.TypeOf(e.widget)] = e.self
	e.inheritedElements = next
}

// GetDependencies is what the dependent keeps about its dependency.
//
// Flutter: widgets/framework.dart InheritedElement.getDependencies
func (e *InheritedElement) GetDependencies(dependent Element) any {
	v, _ := e.dependents.get(dependent)
	return v
}

// SetDependencies sets what the dependent keeps about its dependency.
//
// Flutter: widgets/framework.dart InheritedElement.setDependencies
func (e *InheritedElement) SetDependencies(dependent Element, value any) {
	e.dependents.put(dependent, value)
}

// UpdateDependencies records a dependent with no aspect.
//
// Flutter: widgets/framework.dart InheritedElement.updateDependencies
func (e *InheritedElement) UpdateDependencies(dependent Element, _ any) {
	e.iself.SetDependencies(dependent, nil)
}

// NotifyDependent calls DidChangeDependencies on the dependent.
//
// Flutter: widgets/framework.dart InheritedElement.notifyDependent
func (e *InheritedElement) NotifyDependent(_ Widget, dependent Element) {
	dependent.DidChangeDependencies()
}

// RemoveDependent forgets the dependent.
//
// Flutter: widgets/framework.dart InheritedElement.removeDependent
func (e *InheritedElement) RemoveDependent(dependent Element) { e.dependents.remove(dependent) }

// Updated tells the clients only if the widget says they should hear about it.
//
// Flutter: widgets/framework.dart InheritedElement.updated
func (e *InheritedElement) Updated(oldWidget Widget) {
	if e.widget.(InheritedWidgetInterface).UpdateShouldNotify(oldWidget) {
		e.ProxyElement.Updated(oldWidget)
	}
}

// NotifyClients tells every dependent that the widget changed.
//
// Flutter: widgets/framework.dart InheritedElement.notifyClients
func (e *InheritedElement) NotifyClients(oldWidget Widget) {
	if foundation.DebugMode && e.owner.debugCurrentBuildTarget == nil {
		panic(errors.New("NotifyClientsOutsideBuild",
			"NotifyClients was called at an inappropriate time; it may only be called while the widgets are being built",
			"widget", foundation.RuntimeTypeName(e.widget)))
	}
	for _, dependent := range slices.Clone(e.dependents.keys) {
		e.iself.NotifyDependent(oldWidget, dependent)
	}
}
