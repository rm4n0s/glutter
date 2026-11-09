//go:build darwin && !ios && glutter_display

package metal

import (
	"bytes"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest/gotest"
)

// The tests in this file need a Metal device; they are skipped, with the reason, on a Mac without one (a virtual
// machine without GPU passthrough). They run with MTL_DEBUG_LAYER=1 (SPEC §29.5, §29.13): a call that breaks a rule
// of Metal then fails the process, so a green run means no validation message.

func TestBufferRoundTrip(t *testing.T) {
	d := openTestDevice(t, Options{})
	a, err := d.NewBuffer(256, explicit.BufferCopySource|explicit.BufferCopyDestination)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Destroy()
	b, err := d.NewBuffer(256, explicit.BufferCopySource|explicit.BufferCopyDestination)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Destroy()
	want := make([]byte, 256)
	for i := range want {
		want[i] = byte(i * 3)
	}
	d.WriteBuffer(a, 0, want)
	enc, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	enc.CopyBuffer(a, 0, b, 0, 256)
	var got []byte
	enc.ReadBuffer(b, 0, 256, func(data []byte, err error) {
		if err != nil {
			t.Error(err)
		}
		got = data
	})
	if err := enc.Submit(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got[:16], want[:16])
	}
}

func TestScenes(t *testing.T) {
	d := openTestDevice(t, Options{})
	devicetest.Run(gotest.Wrap(t), &devicetest.Env{Dev: d, Pixels: true, Settle: d.Settle})
}

// TestScenesOnTheManagedPath runs the scenes with the device told that it has no unified memory: the staging
// buffers are managed and their written ranges are given to Metal with didModifyRange:, the readback buffers are
// managed and synchronized, as on a Mac with a discrete GPU (SPEC §29.5). Apple silicon accepts the managed storage
// mode for buffers, so the calls are the ones that an Intel Mac makes, but only its own hardware shows the rest.
func TestScenesOnTheManagedPath(t *testing.T) {
	d := openTestDevice(t, Options{})
	d.unif = false
	devicetest.Run(gotest.Wrap(t), &devicetest.Env{Dev: d, Pixels: true, Settle: d.Settle})
}
