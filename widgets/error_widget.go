package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
)

// ErrorWidget is the widget that replaces a widget that failed to build: a box
// with the message of the error. In debug builds the box is dark red with the
// message in yellow monospace text; in release builds it is gray.
// ComponentElement.PerformRebuild puts one in place of a widget whose Build or
// update panicked, built by BuildOwner.ErrorWidgetBuilder (SPEC §14.2).
//
// Flutter: widgets/framework.dart ErrorWidget
type ErrorWidget struct {
	LeafRenderObjectWidget
	message string
	err     error
}

// NewErrorWidget creates the widget for an error; the message is the text of
// the error. It has a unique key.
//
// Flutter: widgets/framework.dart ErrorWidget.new
func NewErrorWidget(err error) *ErrorWidget {
	msg := "Error"
	if err != nil {
		msg = err.Error()
	}
	return &ErrorWidget{
		LeafRenderObjectWidget: NewLeafRenderObjectWidgetBase(WidgetOpts{Key: foundation.NewUniqueKey()}),
		message:                msg, err: err,
	}
}

// NewErrorWidgetWithDetails creates the widget for a message and, optionally,
// the error behind it.
//
// Flutter: widgets/framework.dart ErrorWidget.withDetails
func NewErrorWidgetWithDetails(message string, err error) *ErrorWidget {
	return &ErrorWidget{
		LeafRenderObjectWidget: NewLeafRenderObjectWidgetBase(WidgetOpts{Key: foundation.NewUniqueKey()}),
		message:                message, err: err,
	}
}

// Message is the message the widget shows.
//
// Flutter: widgets/framework.dart ErrorWidget.message
func (w *ErrorWidget) Message() string { return w.message }

// Err is the error behind the message, or nil.
//
// Flutter: widgets/framework.dart ErrorWidget._flutterError
func (w *ErrorWidget) Err() error { return w.err }

// CreateRenderObject creates the red box. The text engine is the one of the
// binding that the context belongs to; a tree without a binding makes a box
// without text.
//
// Flutter: widgets/framework.dart ErrorWidget.createRenderObject
func (w *ErrorWidget) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	var o rendering.RenderErrorBoxOpts
	if b := MaybeWidgetsBindingOf(bc); b != nil {
		o.TextEngine = b.PlatformDispatcher().TextEngine()
	}
	return rendering.NewRenderErrorBox(w.message, o)
}

// DebugFillProperties adds the message.
//
// Flutter: widgets/framework.dart ErrorWidget.debugFillProperties
func (w *ErrorWidget) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewStringProperty("message", foundation.Some(w.message), foundation.StringPropertyOpts{Unquoted: true}))
}
