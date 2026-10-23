package widgets

import (
	"context"
	"fmt"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// ConnectionState is the state of connection to an asynchronous computation.
//
// Flutter: widgets/async.dart ConnectionState
type ConnectionState int

// The values of ConnectionState follow Flutter's declaration order.
const (
	// ConnectionStateUnspecified is the zero value; it is not a real state.
	//
	// Flutter: no counterpart
	ConnectionStateUnspecified ConnectionState = iota
	// ConnectionStateNone is not currently connected to any asynchronous
	// computation.
	//
	// Flutter: widgets/async.dart ConnectionState.none
	ConnectionStateNone
	// ConnectionStateWaiting is connected to an asynchronous computation and
	// awaiting interaction.
	//
	// Flutter: widgets/async.dart ConnectionState.waiting
	ConnectionStateWaiting
	// ConnectionStateActive is connected to an active asynchronous computation.
	//
	// Flutter: widgets/async.dart ConnectionState.active
	ConnectionStateActive
	// ConnectionStateDone is connected to a terminated asynchronous computation.
	//
	// Flutter: widgets/async.dart ConnectionState.done
	ConnectionStateDone
)

// String is the name of the state as Flutter writes it.
//
// Flutter: widgets/async.dart ConnectionState
func (s ConnectionState) String() string {
	switch s {
	case ConnectionStateNone:
		return "ConnectionState.none"
	case ConnectionStateWaiting:
		return "ConnectionState.waiting"
	case ConnectionStateActive:
		return "ConnectionState.active"
	case ConnectionStateDone:
		return "ConnectionState.done"
	}
	return "ConnectionState.unspecified"
}

// AsyncSnapshot is the immutable most recent interaction with an asynchronous
// computation: its connection state and the latest data or error.
//
// Flutter: widgets/async.dart AsyncSnapshot
type AsyncSnapshot[T any] struct {
	connectionState ConnectionState
	data            T
	hasData         bool
	err             error
}

// NewAsyncSnapshotNothing creates a snapshot in ConnectionStateNone with no data
// or error.
//
// Flutter: widgets/async.dart AsyncSnapshot.nothing
func NewAsyncSnapshotNothing[T any]() AsyncSnapshot[T] {
	return AsyncSnapshot[T]{connectionState: ConnectionStateNone}
}

// NewAsyncSnapshotWaiting creates a snapshot in ConnectionStateWaiting with no
// data or error.
//
// Flutter: widgets/async.dart AsyncSnapshot.waiting
func NewAsyncSnapshotWaiting[T any]() AsyncSnapshot[T] {
	return AsyncSnapshot[T]{connectionState: ConnectionStateWaiting}
}

// NewAsyncSnapshotWithData creates a snapshot in the state with the data.
//
// Flutter: widgets/async.dart AsyncSnapshot.withData
func NewAsyncSnapshotWithData[T any](state ConnectionState, data T) AsyncSnapshot[T] {
	return AsyncSnapshot[T]{connectionState: state, data: data, hasData: true}
}

// NewAsyncSnapshotWithError creates a snapshot in the state with the error.
//
// Flutter: widgets/async.dart AsyncSnapshot.withError
func NewAsyncSnapshotWithError[T any](state ConnectionState, err error) AsyncSnapshot[T] {
	return AsyncSnapshot[T]{connectionState: state, err: err}
}

// ConnectionState is the current state of the connection to the asynchronous
// computation.
//
// Flutter: widgets/async.dart AsyncSnapshot.connectionState
func (s AsyncSnapshot[T]) ConnectionState() ConnectionState { return s.connectionState }

// Data is the latest data received by the computation, or the zero value when
// there is none (see HasData).
//
// Flutter: widgets/async.dart AsyncSnapshot.data
func (s AsyncSnapshot[T]) Data() T { return s.data }

// HasData is whether the snapshot has data.
//
// Flutter: widgets/async.dart AsyncSnapshot.hasData
func (s AsyncSnapshot[T]) HasData() bool { return s.hasData }

// Error is the latest error received by the computation, or nil.
//
// Flutter: widgets/async.dart AsyncSnapshot.error
func (s AsyncSnapshot[T]) Error() error { return s.err }

// HasError is whether the snapshot has an error.
//
// Flutter: widgets/async.dart AsyncSnapshot.hasError
func (s AsyncSnapshot[T]) HasError() bool { return s.err != nil }

// RequireData is the latest data, or panics with the error of the snapshot, or
// with the tag AsyncSnapshotEmpty when it has neither.
//
// Flutter: widgets/async.dart AsyncSnapshot.requireData
func (s AsyncSnapshot[T]) RequireData() T {
	if s.hasData {
		return s.data
	}
	if s.err != nil {
		panic(s.err)
	}
	panic(errors.New(tagAsyncSnapshotEmpty, "the snapshot has neither data nor error"))
}

// InState is a copy of the snapshot in another connection state.
//
// Flutter: widgets/async.dart AsyncSnapshot.inState
func (s AsyncSnapshot[T]) InState(state ConnectionState) AsyncSnapshot[T] {
	s.connectionState = state
	return s
}

// String describes the snapshot.
//
// Flutter: widgets/async.dart AsyncSnapshot.toString
func (s AsyncSnapshot[T]) String() string {
	var data any
	if s.hasData {
		data = s.data
	}
	return fmt.Sprintf("AsyncSnapshot(%s, %v, %v)", s.connectionState, data, s.err)
}

// AsyncWidgetBuilder builds a widget for the snapshot of an asynchronous
// computation.
//
// Flutter: widgets/async.dart AsyncWidgetBuilder
type AsyncWidgetBuilder[T any] func(bc BuildContext, snapshot AsyncSnapshot[T]) Widget

// subscription runs one connection to an asynchronous source: the goroutine
// stops when the connection ends or the state is disposed.
type subscription struct{ cancel context.CancelFunc }

// StreamBuilderBaseOpts are the optional parameters of NewStreamBuilderBase.
//
// Flutter: widgets/async.dart StreamBuilderBase.new
type StreamBuilderBaseOpts[T, S any] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// AfterConnected transforms the summary when the stream is connected; the
	// default keeps it.
	AfterConnected func(current S) S
	// AfterDone transforms the summary when the stream is closed; the default
	// keeps it.
	AfterDone func(current S) S
	// AfterDisconnected transforms the summary when the widget stops listening to
	// the stream; the default keeps it.
	AfterDisconnected func(current S) S
}

// StreamBuilderBase is a widget that builds itself based on the latest snapshot
// of interaction with a stream, in a summary of type S.
//
// Deviation: a Go channel carries no errors, so there is no afterError (SPEC
// §12.10).
//
// Flutter: widgets/async.dart StreamBuilderBase
type StreamBuilderBase[T, S any] struct {
	StatefulWidget
	stream  <-chan T
	initial func() S
	after   StreamBuilderBaseOpts[T, S]
	data    func(current S, data T) S
	build   func(bc BuildContext, summary S) Widget
}

// NewStreamBuilderBase creates a widget that keeps a summary of the values of
// the stream: initial makes the first one, afterData folds each value into it,
// and build builds the widget for it. The stream may be nil.
//
// Flutter: widgets/async.dart StreamBuilderBase.new
func NewStreamBuilderBase[T, S any](stream <-chan T, initial func() S, afterData func(current S, data T) S,
	build func(bc BuildContext, summary S) Widget, opts ...StreamBuilderBaseOpts[T, S]) *StreamBuilderBase[T, S] {
	o := firstOpts(opts)
	return &StreamBuilderBase[T, S]{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), stream, initial, o, afterData, build}
}

// Stream is the channel to listen to; nil for none.
//
// Flutter: widgets/async.dart StreamBuilderBase.stream
func (w *StreamBuilderBase[T, S]) Stream() <-chan T { return w.stream }

func (w *StreamBuilderBase[T, S]) base() *StreamBuilderBase[T, S] { return w }

// CreateState creates the state.
//
// Flutter: widgets/async.dart StreamBuilderBase.createState
func (w *StreamBuilderBase[T, S]) CreateState() State { return &streamBuilderBaseState[T, S]{} }

// streamBuilderBaseWidget is a StreamBuilderBase or a widget that embeds it.
type streamBuilderBaseWidget[T, S any] interface {
	Widget
	base() *StreamBuilderBase[T, S]
}

type streamBuilderBaseState[T, S any] struct {
	StateOf[streamBuilderBaseWidget[T, S]]
	sub     *subscription
	summary S
}

func (s *streamBuilderBaseState[T, S]) InitState() {
	s.summary = s.Widget().base().initial()
	s.subscribe()
}

func (s *streamBuilderBaseState[T, S]) DidUpdateWidget(old streamBuilderBaseWidget[T, S]) {
	if old.base().stream != s.Widget().base().stream {
		if s.sub != nil {
			s.unsubscribe()
			if f := s.Widget().base().after.AfterDisconnected; f != nil {
				s.summary = f(s.summary)
			}
		}
		s.subscribe()
	}
}

func (s *streamBuilderBaseState[T, S]) Dispose() { s.unsubscribe() }

func (s *streamBuilderBaseState[T, S]) subscribe() {
	w := s.Widget().base()
	if w.stream == nil {
		return
	}
	ctx, cancel := context.WithCancel(s.Lifetime())
	sub := &subscription{cancel}
	s.sub = sub
	ch := w.stream
	s.Go(func(context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-ch:
				if !ok {
					s.SetState(func() {
						if s.sub == sub {
							if f := s.Widget().base().after.AfterDone; f != nil {
								s.summary = f(s.summary)
							}
						}
					})
					return
				}
				s.SetState(func() {
					if s.sub == sub {
						s.summary = s.Widget().base().data(s.summary, v)
					}
				})
			}
		}
	})
	if f := w.after.AfterConnected; f != nil {
		s.summary = f(s.summary)
	}
}

func (s *streamBuilderBaseState[T, S]) unsubscribe() {
	if s.sub != nil {
		s.sub.cancel()
		s.sub = nil
	}
}

func (s *streamBuilderBaseState[T, S]) Build(bc BuildContext) Widget {
	return s.Widget().base().build(bc, s.summary)
}

// StreamBuilderOpts are the optional parameters of NewStreamBuilder.
//
// Flutter: widgets/async.dart StreamBuilder.new
type StreamBuilderOpts[T any] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// InitialData is the data that will be used to create the initial snapshot.
	InitialData foundation.Opt[T]
}

// StreamBuilder is a widget that builds itself based on the latest snapshot of
// interaction with a channel.
//
// Flutter: widgets/async.dart StreamBuilder
type StreamBuilder[T any] struct {
	StreamBuilderBase[T, AsyncSnapshot[T]]
	builder     AsyncWidgetBuilder[T]
	initialData foundation.Opt[T]
}

// NewStreamBuilder creates a widget that builds with the latest value received
// from the channel. The channel may be nil. The builder sees the connection
// state: waiting after the channel is connected, active after each value, done
// once the channel is closed.
//
// Flutter: widgets/async.dart StreamBuilder.new
func NewStreamBuilder[T any](stream <-chan T, builder AsyncWidgetBuilder[T], opts ...StreamBuilderOpts[T]) *StreamBuilder[T] {
	o := firstOpts(opts)
	w := &StreamBuilder[T]{builder: builder, initialData: o.InitialData}
	w.StreamBuilderBase = *NewStreamBuilderBase(stream,
		func() AsyncSnapshot[T] {
			if d, ok := w.initialData.Get(); ok {
				return NewAsyncSnapshotWithData(ConnectionStateNone, d)
			}
			return NewAsyncSnapshotNothing[T]()
		},
		func(_ AsyncSnapshot[T], data T) AsyncSnapshot[T] {
			return NewAsyncSnapshotWithData(ConnectionStateActive, data)
		},
		func(bc BuildContext, summary AsyncSnapshot[T]) Widget { return w.builder(bc, summary) },
		StreamBuilderBaseOpts[T, AsyncSnapshot[T]]{
			Key:               o.Key,
			AfterConnected:    func(c AsyncSnapshot[T]) AsyncSnapshot[T] { return c.InState(ConnectionStateWaiting) },
			AfterDone:         func(c AsyncSnapshot[T]) AsyncSnapshot[T] { return c.InState(ConnectionStateDone) },
			AfterDisconnected: func(c AsyncSnapshot[T]) AsyncSnapshot[T] { return c.InState(ConnectionStateNone) },
		})
	return w
}

// Builder is the function that builds the widget for a snapshot.
//
// Flutter: widgets/async.dart StreamBuilder.builder
func (w *StreamBuilder[T]) Builder() AsyncWidgetBuilder[T] { return w.builder }

// InitialData is the data of the initial snapshot, if any.
//
// Flutter: widgets/async.dart StreamBuilder.initialData
func (w *StreamBuilder[T]) InitialData() foundation.Opt[T] { return w.initialData }

// FutureBuilderOpts are the optional parameters of NewFutureBuilder.
//
// Flutter: widgets/async.dart FutureBuilder.new
type FutureBuilderOpts[T any] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// InitialData is the data that will be used to create the initial snapshot.
	InitialData foundation.Opt[T]
	// Identity says which computation the function is: when it changes, compared
	// with ==, the old one is forgotten and the function is started again. Flutter
	// starts again when the Future object changes; function values cannot be
	// compared in Go, so give the value that the result depends on. The default
	// (nil) starts the function once, and again only when the function becomes
	// nil and then non-nil.
	Identity any
}

// FutureBuilder is a widget that builds itself based on the latest snapshot of
// interaction with a function that runs in a goroutine.
//
// Deviation: the function is started with State.Go, and Identity stands for the
// identity of Flutter's Future (SPEC §12.10).
//
// Flutter: widgets/async.dart FutureBuilder
type FutureBuilder[T any] struct {
	StatefulWidget
	future      func(ctx context.Context) (T, error)
	builder     AsyncWidgetBuilder[T]
	initialData foundation.Opt[T]
	identity    any
}

// NewFutureBuilder creates a widget that runs the function, with a context that
// is cancelled when the widget leaves the tree or the function is replaced, and
// builds with its result. The function may be nil, for no computation.
//
// Flutter: widgets/async.dart FutureBuilder.new
func NewFutureBuilder[T any](future func(ctx context.Context) (T, error), builder AsyncWidgetBuilder[T], opts ...FutureBuilderOpts[T]) *FutureBuilder[T] {
	o := firstOpts(opts)
	return &FutureBuilder[T]{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), future, builder, o.InitialData, o.Identity}
}

// Future is the function that computes the data; nil for none.
//
// Flutter: widgets/async.dart FutureBuilder.future
func (w *FutureBuilder[T]) Future() func(ctx context.Context) (T, error) { return w.future }

// Builder is the function that builds the widget for a snapshot.
//
// Flutter: widgets/async.dart FutureBuilder.builder
func (w *FutureBuilder[T]) Builder() AsyncWidgetBuilder[T] { return w.builder }

// InitialData is the data of the initial snapshot, if any.
//
// Flutter: widgets/async.dart FutureBuilder.initialData
func (w *FutureBuilder[T]) InitialData() foundation.Opt[T] { return w.initialData }

// CreateState creates the state.
//
// Flutter: widgets/async.dart FutureBuilder.createState
func (w *FutureBuilder[T]) CreateState() State { return &futureBuilderState[T]{} }

type futureBuilderState[T any] struct {
	StateOf[*FutureBuilder[T]]
	sub      *subscription
	snapshot AsyncSnapshot[T]
}

func (s *futureBuilderState[T]) InitState() {
	if d, ok := s.Widget().initialData.Get(); ok {
		s.snapshot = NewAsyncSnapshotWithData(ConnectionStateNone, d)
	} else {
		s.snapshot = NewAsyncSnapshotNothing[T]()
	}
	s.subscribe()
}

func (s *futureBuilderState[T]) DidUpdateWidget(old *FutureBuilder[T]) {
	w := s.Widget()
	if (old.future == nil) == (w.future == nil) && old.identity == w.identity {
		return
	}
	if s.sub != nil {
		s.unsubscribe()
		s.snapshot = s.snapshot.InState(ConnectionStateNone)
	}
	s.subscribe()
}

func (s *futureBuilderState[T]) Dispose() { s.unsubscribe() }

func (s *futureBuilderState[T]) Build(bc BuildContext) Widget {
	return s.Widget().builder(bc, s.snapshot)
}

func (s *futureBuilderState[T]) subscribe() {
	f := s.Widget().future
	if f == nil {
		return
	}
	ctx, cancel := context.WithCancel(s.Lifetime())
	sub := &subscription{cancel}
	s.sub = sub
	s.Go(func(context.Context) {
		data, err := f(ctx)
		s.SetState(func() {
			if s.sub != sub {
				return
			}
			if err != nil {
				s.snapshot = NewAsyncSnapshotWithError[T](ConnectionStateDone, err)
			} else {
				s.snapshot = NewAsyncSnapshotWithData(ConnectionStateDone, data)
			}
		})
	})
	if s.snapshot.connectionState != ConnectionStateDone {
		s.snapshot = s.snapshot.InState(ConnectionStateWaiting)
	}
}

func (s *futureBuilderState[T]) unsubscribe() {
	if s.sub != nil {
		s.sub.cancel()
		s.sub = nil
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagAsyncSnapshotEmpty = "AsyncSnapshotEmpty"
)
