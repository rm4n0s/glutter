// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

// AspectSet is the set of aspects of a model that a dependent depends on.
//
// Flutter: widgets/inherited_model.dart InheritedModel.updateShouldNotifyDependent
type AspectSet struct{ set orderedSet[any] }

// Contains is whether the aspect is in the set.
func (s *AspectSet) Contains(aspect any) bool { return s.set.has(aspect) }

// Len is the number of aspects.
func (s *AspectSet) Len() int { return s.set.len() }

// Items are the aspects, in the order they were added.
func (s *AspectSet) Items() []any { return append([]any(nil), s.set.items...) }

// InheritedModel is an inherited widget whose dependents depend on parts of it,
// the aspects: a dependent is rebuilt only when an aspect it asked for changed.
// Aspects are comparable values of any type. Embed it, implement
// UpdateShouldNotifyDependent, and optionally IsSupportedAspect.
//
// Flutter: widgets/inherited_model.dart InheritedModel
type InheritedModel struct{ InheritedWidget }

// NewInheritedModelBase creates the embedded base of an inherited model.
//
// Flutter: widgets/inherited_model.dart InheritedModel.new
func NewInheritedModelBase(child Widget, opts ...WidgetOpts) InheritedModel {
	return InheritedModel{NewInheritedWidgetBase(child, opts...)}
}

func (*InheritedModel) isInheritedModel() {}

// IsSupportedAspect is whether the model has the aspect; it is true by default.
// When it is false, the lookup goes on to the next model of the same type above.
//
// Flutter: widgets/inherited_model.dart InheritedModel.isSupportedAspect
func (*InheritedModel) IsSupportedAspect(any) bool { return true }

// modelWidget is any widget with the InheritedModel base.
type modelWidget interface {
	InheritedWidgetInterface
	// UpdateShouldNotifyDependent is whether a dependent on the given aspects
	// is rebuilt after the widget was replaced.
	//
	// Flutter: widgets/inherited_model.dart InheritedModel.updateShouldNotifyDependent
	UpdateShouldNotifyDependent(oldWidget Widget, dependencies *AspectSet) bool
	IsSupportedAspect(aspect any) bool
	isInheritedModel()
}

// DependOnModel returns the closest model of type T above the context that
// supports the aspect, and makes the context depend on that aspect of every
// model of type T from the closest up to that one. With a nil aspect it is
// DependOn.
//
// Flutter: widgets/inherited_model.dart InheritedModel.inheritFrom
func DependOnModel[T InheritedWidgetInterface](bc BuildContext, aspect any) (T, bool) {
	var zero T
	if aspect == nil {
		return DependOn[T](bc)
	}
	var models []InheritedElementSelf
	findModels[T](bc, aspect, &models)
	if len(models) == 0 {
		return zero, false
	}
	last := models[len(models)-1]
	for _, model := range models {
		value := bc.DependOnInheritedElement(model, aspect).(T)
		if model == last {
			return value, true
		}
	}
	return zero, false
}

// findModels collects the models of type T above the context, ending with the
// one that supports the aspect.
func findModels[T InheritedWidgetInterface](bc BuildContext, aspect any, results *[]InheritedElementSelf) {
	model := GetElementForInheritedWidget[T](bc)
	if model == nil {
		return
	}
	*results = append(*results, model)
	if model.AsElement().widget.(modelWidget).IsSupportedAspect(aspect) {
		return
	}
	var modelParent Element
	model.VisitAncestorElements(func(ancestor Element) bool {
		modelParent = ancestor
		return false
	})
	if modelParent == nil {
		return
	}
	findModels[T](modelParent, aspect, results)
}

// InheritedModelElement is the element of an InheritedModel: it keeps the
// aspects each dependent depends on.
//
// Flutter: widgets/inherited_model.dart InheritedModelElement
type InheritedModelElement struct {
	InheritedElement
}

// NewInheritedModelElement creates the element of an inherited model.
//
// Flutter: widgets/inherited_model.dart InheritedModelElement.new
func NewInheritedModelElement(w modelWidget) *InheritedModelElement {
	e := &InheritedModelElement{}
	InitInheritedElement(&e.InheritedElement, e, w)
	return e
}

// UpdateDependencies adds the aspect to the aspects of the dependent; a
// dependent with no aspect depends on the whole model.
//
// Flutter: widgets/inherited_model.dart InheritedModelElement.updateDependencies
func (e *InheritedModelElement) UpdateDependencies(dependent Element, aspect any) {
	deps, _ := e.GetDependencies(dependent).(*AspectSet)
	if deps != nil && deps.Len() == 0 {
		return
	}
	if aspect == nil {
		e.SetDependencies(dependent, &AspectSet{})
		return
	}
	if deps == nil {
		deps = &AspectSet{}
	}
	deps.set.add(aspect)
	e.SetDependencies(dependent, deps)
}

// NotifyDependent rebuilds the dependent if it depends on the whole model or
// on an aspect that the widget says changed.
//
// Flutter: widgets/inherited_model.dart InheritedModelElement.notifyDependent
func (e *InheritedModelElement) NotifyDependent(oldWidget Widget, dependent Element) {
	deps, _ := e.GetDependencies(dependent).(*AspectSet)
	if deps == nil {
		return
	}
	if deps.Len() == 0 || e.widget.(modelWidget).UpdateShouldNotifyDependent(oldWidget, deps) {
		dependent.DidChangeDependencies()
	}
}
