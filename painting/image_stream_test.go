package painting

import (
	"image/color"
	"testing"
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func testImage(w, h int) *ui.Image { return ui.NewImageFromRGBA(make([]byte, w*h*4), w, h) }

// imageRecorder is a listener that records what it gets.
type imageRecorder struct {
	images []*ImageInfo
	syncs  []bool
	errs   []error
	chunks []ImageChunkEvent
}

func (r *imageRecorder) listener() *ImageStreamListener {
	return NewImageStreamListener(func(i *ImageInfo, sync bool) {
		r.images = append(r.images, i)
		r.syncs = append(r.syncs, sync)
	}, ImageStreamListenerOpts{
		OnError: func(err error) { r.errs = append(r.errs, err) },
		OnChunk: func(e ImageChunkEvent) { r.chunks = append(r.chunks, e) },
	})
}

func TestImageInfo(t *testing.T) {
	img := testImage(3, 5)
	info := NewImageInfo(img, ImageInfoOpts{Scale: foundation.Some(2.0), DebugLabel: "x"})
	if info.SizeBytes() != 60 {
		t.Errorf("SizeBytes = %d, want 60", info.SizeBytes())
	}
	clone := info.Clone()
	if !clone.IsCloneOf(info) || clone.Image == info.Image {
		t.Error("a clone shares the pixels with its own handle")
	}
	if NewImageInfo(testImage(3, 5)).IsCloneOf(info) {
		t.Error("another image is not a clone")
	}
	if got := info.String(); len(got) < 2 || got[:2] != "x " {
		t.Errorf("String = %q", got)
	}
	if NewImageInfo(img).Scale != 1 {
		t.Error("the default scale is 1")
	}
	if !info.Equal(info) || info.Equal(NewImageInfo(img)) || info.Equal(nil) {
		t.Error("Equal compares image, scale and label")
	}
	info.Dispose()
	if !img.IsDisposed() {
		t.Error("Dispose releases the image")
	}
}

func TestImageStreamListeners(t *testing.T) {
	pt := newPaintingTester(t)
	tests := []struct {
		name        string
		listenFirst bool
		wantSync    bool
	}{
		{"listener added before the completer is set", true, false},
		{"listener added after the image arrived", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := NewImageStream()
			c := NewImageStreamCompleter(pt.PaintingBinding)
			rec := &imageRecorder{}
			l := rec.listener()
			if tt.listenFirst {
				stream.AddListener(l)
				if stream.Key() != any(stream) {
					t.Error("before the completer the key is the stream")
				}
			}
			c.SetImage(NewImageInfo(testImage(2, 2)))
			stream.SetCompleter(c)
			if !tt.listenFirst {
				stream.AddListener(l)
			}
			if stream.Key() != any(c) {
				t.Error("the key is the completer once set")
			}
			if len(rec.images) != 1 || rec.syncs[0] != tt.wantSync {
				t.Fatalf("images %d syncs %v, want one image with sync %v", len(rec.images), rec.syncs, tt.wantSync)
			}
			// A later image goes to the listener asynchronously.
			c.SetImage(NewImageInfo(testImage(3, 3)))
			if len(rec.images) != 2 || rec.syncs[1] {
				t.Errorf("second image: syncs %v", rec.syncs)
			}
			stream.RemoveListener(l)
			if c.Disposed() == false {
				t.Error("removing the last listener disposes the completer")
			}
		})
	}
}

func TestImageStreamRemoveBeforeCompleter(t *testing.T) {
	pt := newPaintingTester(t)
	stream := NewImageStream()
	rec := &imageRecorder{}
	l := rec.listener()
	stream.AddListener(l)
	stream.RemoveListener(l)
	c := NewImageStreamCompleter(pt.PaintingBinding)
	stream.SetCompleter(c)
	if c.HasListeners() {
		t.Error("a removed listener must not reach the completer")
	}
}

func TestCompleterDisposal(t *testing.T) {
	pt := newPaintingTester(t)
	t.Run("keepAlive", func(t *testing.T) {
		c := NewImageStreamCompleter(pt.PaintingBinding)
		l := (&imageRecorder{}).listener()
		c.AddListener(l)
		h := c.KeepAlive()
		c.RemoveListener(l)
		if c.Disposed() {
			t.Fatal("a handle keeps the completer alive")
		}
		h.Dispose()
		if !c.Disposed() {
			t.Error("disposing the last handle disposes the completer")
		}
	})
	t.Run("current image is disposed", func(t *testing.T) {
		c := NewImageStreamCompleter(pt.PaintingBinding)
		l := (&imageRecorder{}).listener()
		c.AddListener(l)
		info := NewImageInfo(testImage(1, 1))
		c.SetImage(info)
		cur := c.CurrentImage()
		c.RemoveListener(l)
		if !cur.Image.IsDisposed() || c.CurrentImage() != nil {
			t.Error("the completer disposes its image")
		}
	})
	t.Run("last listener callbacks", func(t *testing.T) {
		c := NewImageStreamCompleter(pt.PaintingBinding)
		l := (&imageRecorder{}).listener()
		c.AddListener(l)
		calls, removed := 0, 0
		c.AddOnLastListenerRemovedCallback(func() { calls++ })
		remove := c.AddOnLastListenerRemovedCallback(func() { removed++ })
		remove()
		c.RemoveListener(l)
		if calls != 1 || removed != 0 {
			t.Errorf("calls %d removed %d", calls, removed)
		}
	})
	t.Run("on disposed", func(t *testing.T) {
		c := NewImageStreamCompleter(pt.PaintingBinding)
		n := 0
		c.SetOnDisposed(func() { n++ })
		l := (&imageRecorder{}).listener()
		c.AddListener(l)
		c.RemoveListener(l)
		c.MaybeDispose()
		if n != 1 {
			t.Errorf("onDisposed called %d times", n)
		}
	})
}

func TestCompleterErrors(t *testing.T) {
	pt := newPaintingTester(t)
	boom := errors.New("Boom", "boom")
	t.Run("error listener", func(t *testing.T) {
		c := NewImageStreamCompleter(pt.PaintingBinding)
		rec := &imageRecorder{}
		c.AddListener(rec.listener())
		c.ReportError(ImageErrorDetails{Err: boom})
		if len(rec.errs) != 1 || rec.errs[0] != boom {
			t.Errorf("errs = %v", rec.errs)
		}
		// A listener added later gets the error right away.
		late := &imageRecorder{}
		c.AddListener(late.listener())
		if len(late.errs) != 1 {
			t.Errorf("late errs = %v", late.errs)
		}
	})
	t.Run("unhandled goes to the binding", func(t *testing.T) {
		before := len(pt.errs)
		c := NewImageStreamCompleter(pt.PaintingBinding)
		c.AddListener(NewImageStreamListener(func(*ImageInfo, bool) {}))
		c.ReportError(ImageErrorDetails{Err: boom, Silent: true})
		if len(pt.errs) != before+1 || pt.errs[before].Library != "image resource service" {
			t.Errorf("reported = %v", pt.errs)
		}
	})
	t.Run("a listener that opts out suppresses the report", func(t *testing.T) {
		before := len(pt.errs)
		c := NewImageStreamCompleter(pt.PaintingBinding)
		c.AddListener(NewImageStreamListener(func(*ImageInfo, bool) {}, ImageStreamListenerOpts{ReportErrors: foundation.Some(false)}))
		c.ReportError(ImageErrorDetails{Err: boom})
		if len(pt.errs) != before {
			t.Error("the error was reported although the listener opted out")
		}
	})
	t.Run("ephemeral listener", func(t *testing.T) {
		c := NewImageStreamCompleter(pt.PaintingBinding)
		var got []error
		c.AddEphemeralErrorListener(func(err error) { got = append(got, err) })
		c.ReportError(ImageErrorDetails{Err: boom})
		c.ReportError(ImageErrorDetails{Err: boom})
		if len(got) != 1 {
			t.Errorf("an ephemeral listener is called once, got %d", len(got))
		}
		// With an error already there it is called at once, and with an image it
		// is dropped.
		var again []error
		c.AddEphemeralErrorListener(func(err error) { again = append(again, err) })
		if len(again) != 1 {
			t.Errorf("immediate call: %v", again)
		}
		c2 := NewImageStreamCompleter(pt.PaintingBinding)
		var dropped []error
		c2.AddEphemeralErrorListener(func(err error) { dropped = append(dropped, err) })
		c2.AddListener((&imageRecorder{}).listener())
		c2.SetImage(NewImageInfo(testImage(1, 1)))
		c2.ReportError(ImageErrorDetails{Err: boom})
		if len(dropped) != 0 {
			t.Error("an image drops the ephemeral listeners")
		}
	})
	t.Run("chunk events", func(t *testing.T) {
		c := NewImageStreamCompleter(pt.PaintingBinding)
		rec := &imageRecorder{}
		c.AddListener(rec.listener())
		c.ReportImageChunkEvent(NewImageChunkEvent(10, foundation.Some(100)))
		if len(rec.chunks) != 1 || rec.chunks[0].CumulativeBytesLoaded != 10 || rec.chunks[0].ExpectedTotalBytes.Value() != 100 {
			t.Errorf("chunks = %v", rec.chunks)
		}
	})
}

func TestOneFrameImageStreamCompleter(t *testing.T) {
	pt := newPaintingTester(t)
	t.Run("image", func(t *testing.T) {
		ch := make(chan ImageResult, 1)
		c := NewOneFrameImageStreamCompleter(pt.PaintingBinding, ch, nil)
		rec := &imageRecorder{}
		c.AddListener(rec.listener())
		ch <- ImageResult{Info: NewImageInfo(testImage(2, 2))}
		pt.waitFor(t, "the image", func() bool { return len(rec.images) == 1 })
		if rec.syncs[0] {
			t.Error("the image arrives asynchronously")
		}
	})
	t.Run("error", func(t *testing.T) {
		ch := make(chan ImageResult, 1)
		c := NewOneFrameImageStreamCompleter(pt.PaintingBinding, ch, nil)
		rec := &imageRecorder{}
		c.AddListener(rec.listener())
		ch <- ImageResult{Err: errors.New("Nope", "nope")}
		pt.waitFor(t, "the error", func() bool { return len(rec.errs) == 1 })
	})
	t.Run("disposed before the image", func(t *testing.T) {
		ch := make(chan ImageResult, 1)
		c := NewOneFrameImageStreamCompleter(pt.PaintingBinding, ch, nil)
		l := (&imageRecorder{}).listener()
		c.AddListener(l)
		c.RemoveListener(l)
		info := NewImageInfo(testImage(2, 2))
		ch <- ImageResult{Info: info}
		time.Sleep(10 * time.Millisecond)
		pt.host.runPosted()
	})
}

func TestMultiFrameImageStreamCompleterSingleFrame(t *testing.T) {
	pt := newPaintingTester(t)
	ch := make(chan CodecResult, 1)
	chunks := make(chan ImageChunkEvent, 1)
	c := NewMultiFrameImageStreamCompleter(pt.PaintingBinding, MultiFrameImageStreamCompleterOpts{
		Codec: ch, Scale: 2, DebugLabel: "png", ChunkEvents: chunks,
	})
	rec := &imageRecorder{}
	c.AddListener(rec.listener())
	codec, err := ui.InstantiateImageCodec(pt.Context(), pngBytes(t, 3, 2, color.RGBA{255, 0, 0, 255}))
	if err != nil {
		t.Fatal(err)
	}
	chunks <- ImageChunkEvent{CumulativeBytesLoaded: 5}
	ch <- CodecResult{Codec: codec}
	pt.waitFor(t, "the image", func() bool { return len(rec.images) == 1 })
	pt.waitFor(t, "the chunk", func() bool { return len(rec.chunks) == 1 })
	if got := rec.images[0]; got.Scale != 2 || got.DebugLabel != "png" || got.Image.Width() != 3 || got.Image.Height() != 2 {
		t.Errorf("image = %v", got)
	}
	// A listener that comes later gets the same image at once.
	late := &imageRecorder{}
	c.AddListener(late.listener())
	if len(late.images) != 1 || !late.syncs[0] {
		t.Errorf("late listener: %d images, syncs %v", len(late.images), late.syncs)
	}
}

func TestMultiFrameImageStreamCompleterError(t *testing.T) {
	pt := newPaintingTester(t)
	ch := make(chan CodecResult, 1)
	c := NewMultiFrameImageStreamCompleter(pt.PaintingBinding, MultiFrameImageStreamCompleterOpts{Codec: ch, Scale: 1})
	rec := &imageRecorder{}
	c.AddListener(rec.listener())
	codec, err := ui.InstantiateImageCodec(pt.Context(), []byte("not an image"))
	ch <- CodecResult{Codec: codec, Err: err}
	pt.waitFor(t, "the error", func() bool { return len(rec.errs) == 1 })
	appErr, ok := errors.FromError(rec.errs[0])
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", rec.errs[0])
	}
	if !appErr.HasRoute("ImageDecodeFailed") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestMultiFrameImageStreamCompleterAnimation(t *testing.T) {
	pt := newPaintingTester(t)
	// Three frames of 200 ms; loop count 1 means one repetition after the first
	// play in Go's gif package, which ui reports as RepetitionCount 1.
	data := gifBytes(t, 20, 0, color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}, color.RGBA{0, 0, 255, 255})
	codec, err := ui.InstantiateImageCodec(pt.Context(), data)
	if err != nil {
		t.Fatal(err)
	}
	if codec.FrameCount() != 3 {
		t.Fatalf("frames = %d", codec.FrameCount())
	}
	ch := make(chan CodecResult, 1)
	ch <- CodecResult{Codec: codec}
	c := NewMultiFrameImageStreamCompleter(pt.PaintingBinding, MultiFrameImageStreamCompleterOpts{Codec: ch, Scale: 1})
	rec := &imageRecorder{}
	l := rec.listener()
	c.AddListener(l)
	pt.waitFor(t, "the first frame callback", func() bool { return pt.sched.TransientCallbackCount() == 1 })

	pt.frame(0)
	pt.host.runPosted()
	if len(rec.images) != 1 {
		t.Fatalf("after the first frame: %d images", len(rec.images))
	}
	// Half way through the first frame nothing changes: a timer is set.
	pt.clock.Advance(100 * time.Millisecond)
	pt.frame(100 * time.Millisecond)
	pt.host.runPosted()
	if len(rec.images) != 1 {
		t.Fatalf("before the frame duration: %d images", len(rec.images))
	}
	pt.clock.Advance(100 * time.Millisecond)
	pt.host.runPosted()
	pt.frame(200 * time.Millisecond)
	pt.host.runPosted()
	if len(rec.images) != 2 {
		t.Fatalf("after 200 ms: %d images", len(rec.images))
	}
	pt.frame(400 * time.Millisecond)
	pt.host.runPosted()
	if len(rec.images) != 3 {
		t.Fatalf("after 400 ms: %d images", len(rec.images))
	}
	// Without listeners the animation stops and the timer is cancelled.
	c.RemoveListener(l)
	if !c.Disposed() {
		t.Error("the completer is disposed without listeners")
	}
	pt.clock.Advance(time.Second)
	pt.frame(2 * time.Second)
	pt.host.runPosted()
	if len(rec.images) != 3 {
		t.Errorf("no frames after the last listener left, got %d", len(rec.images))
	}
}
