package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
)

// MetaDataOpts are the optional parameters of NewMetaData.
//
// Flutter: widgets/basic.dart MetaData.new
type MetaDataOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// MetaData is the meta data to hold in the render tree.
	MetaData any
	// Behavior is how to behave during hit testing; the default is
	// HitTestBehaviorDeferToChild.
	Behavior rendering.HitTestBehavior
}

// MetaData holds meta data in the render tree: a hit test finds it, and the code
// that walks the hit test path reads it. A DragTarget is found this way.
//
// Flutter: widgets/basic.dart MetaData
type MetaData struct {
	SingleChildRenderObjectWidget
	metaData any
	behavior rendering.HitTestBehavior
}

// NewMetaData creates a widget that holds the meta data.
//
// Flutter: widgets/basic.dart MetaData.new
func NewMetaData(child Widget, opts ...MetaDataOpts) *MetaData {
	o := firstOpts(opts)
	if o.Behavior == rendering.HitTestBehaviorUnspecified {
		o.Behavior = rendering.HitTestBehaviorDeferToChild
	}
	return &MetaData{NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), o.MetaData, o.Behavior}
}

// MetaData is the meta data that the widget holds.
//
// Flutter: widgets/basic.dart MetaData.metaData
func (w *MetaData) MetaData() any { return w.metaData }

// Behavior is how the widget behaves during hit testing.
//
// Flutter: widgets/basic.dart MetaData.behavior
func (w *MetaData) Behavior() rendering.HitTestBehavior { return w.behavior }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart MetaData.createRenderObject
func (w *MetaData) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderMetaData(nil, rendering.RenderMetaDataOpts{MetaData: w.metaData, Behavior: w.behavior})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart MetaData.updateRenderObject
func (w *MetaData) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderMetaData)
	r.MetaData = w.metaData
	r.Behavior = w.behavior
}
