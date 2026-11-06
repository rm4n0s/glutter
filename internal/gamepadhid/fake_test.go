package gamepadhid

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/padstd"
)

// fakeIO is the IO of a driver under test: the reports it reads are scripted, the reports it writes and sends are kept, and the
// time moves only when a read waits in vain.
type fakeIO struct {
	now      time.Duration
	reads    [][]byte
	features map[byte][]byte
	// writes are the reports written at once, sends the ones that were queued, and featuresSent the feature reports sent.
	writes, sends, featuresSent [][]byte
	// onWrite and onSend run after a report is written or queued, so that a scripted pad can answer.
	onWrite func(data []byte)
	onSend  func(data []byte)
	readErr error
	// merges are the functions that the reports queued with SendMerging came with.
	merges []func([]byte) bool
}

func (f *fakeIO) Read(buf []byte, timeout time.Duration) (int, error) {
	if f.readErr != nil {
		return 0, f.readErr
	}
	if len(f.reads) == 0 {
		f.now += timeout
		return 0, nil
	}
	n := copy(buf, f.reads[0])
	f.reads = f.reads[1:]
	return n, nil
}

func (f *fakeIO) Write(data []byte) (int, error) {
	f.writes = append(f.writes, append([]byte(nil), data...))
	if f.onWrite != nil {
		f.onWrite(data)
	}
	return len(data), nil
}

func (f *fakeIO) Send(data []byte) error { return f.SendMerging(data, nil) }

func (f *fakeIO) SendMerging(data []byte, merge func([]byte) bool) error {
	f.sends = append(f.sends, append([]byte(nil), data...))
	f.merges = append(f.merges, merge)
	if f.onSend != nil {
		f.onSend(data)
	}
	return nil
}

func (f *fakeIO) Feature(id byte, buf []byte) (int, error) {
	clear(buf)
	buf[0] = id
	v, ok := f.features[id]
	if !ok {
		return -1, errors.New(tagFakeFeatureMissing, "no such feature report")
	}
	return copy(buf, v), nil
}

func (f *fakeIO) SendFeature(data []byte) (int, error) {
	f.featuresSent = append(f.featuresSent, append([]byte(nil), data...))
	return len(data), nil
}

func (f *fakeIO) Now() time.Duration { return f.now }

// lastSend returns the report that was queued last.
func (f *fakeIO) lastSend(t *testing.T) []byte {
	t.Helper()
	if len(f.sends) == 0 {
		t.Fatal("no report was sent")
	}
	return f.sends[len(f.sends)-1]
}

// recSink keeps what a driver reports.
type recSink struct {
	states    []padstd.Snapshot
	touches   []platform.PadTouch
	sensors   []platform.PadSensor
	batteries []platform.PadBattery
	lost      int
}

func (r *recSink) State(now time.Duration, s *padstd.Snapshot) { r.states = append(r.states, *s) }
func (r *recSink) Touch(now time.Duration, t platform.PadTouch) {
	t.Time = now
	r.touches = append(r.touches, t)
}
func (r *recSink) Sensor(now time.Duration, s platform.PadSensor) {
	s.Time = now
	r.sensors = append(r.sensors, s)
}
func (r *recSink) Battery(b platform.PadBattery) { r.batteries = append(r.batteries, b) }
func (r *recSink) Lost()                         { r.lost++ }

func (r *recSink) lastState(t *testing.T) padstd.Snapshot {
	t.Helper()
	if len(r.states) == 0 {
		t.Fatal("the driver reported no state")
	}
	return r.states[len(r.states)-1]
}

// slowCRC32 is the CRC-32 of zlib (the polynomial 0xEDB88320, reflected, start and end value 0xFFFFFFFF) made a bit at a time, so
// that the tests do not use the code that the driver uses: the CRC of the output reports of the PlayStation pads covers the HIDP
// header and the report.
func slowCRC32(data ...[]byte) uint32 {
	crc := ^uint32(0)
	for _, d := range data {
		for _, b := range d {
			crc ^= uint32(b)
			for i := 0; i < 8; i++ {
				if crc&1 != 0 {
					crc = crc>>1 ^ 0xEDB88320
				} else {
					crc >>= 1
				}
			}
		}
	}
	return ^crc
}

// withCRC puts the CRC of a report that the pad sends (header 0xA1) in its last four bytes.
func withCRC(header byte, r []byte) []byte {
	n := len(r) - 4
	c := slowCRC32([]byte{header}, r[:n])
	r[n], r[n+1], r[n+2], r[n+3] = byte(c), byte(c>>8), byte(c>>16), byte(c>>24)
	return r
}

func mustEqual(t *testing.T, what string, got, want []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		t.Errorf("%s:\n got %x\nwant %x", what, got, want)
	}
}

// fakeTransport is the transport of a Device under test.
type fakeTransport struct {
	mu       sync.Mutex
	in       chan []byte
	closed   chan struct{}
	once     sync.Once
	written  [][]byte
	features map[byte][]byte
	// failRead makes the next read fail, which is how a pad goes.
	failRead chan error
	onWrite  func(data []byte, t *fakeTransport)
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{in: make(chan []byte, 64), closed: make(chan struct{}), failRead: make(chan error, 1), features: map[byte][]byte{}}
}

func (f *fakeTransport) Read(buf []byte, timeout time.Duration) (int, error) {
	select {
	case r := <-f.in:
		return copy(buf, r), nil
	case err := <-f.failRead:
		return 0, err
	case <-f.closed:
		return 0, errors.New(tagFakeTransportClosed, "closed")
	case <-time.After(timeout):
		return 0, nil
	}
}

func (f *fakeTransport) Write(data []byte) (int, error) {
	f.mu.Lock()
	f.written = append(f.written, append([]byte(nil), data...))
	cb := f.onWrite
	f.mu.Unlock()
	if cb != nil {
		cb(data, f)
	}
	return len(data), nil
}

func (f *fakeTransport) SendFeature(data []byte) (int, error) { return len(data), nil }

func (f *fakeTransport) GetFeature(data []byte) (int, error) {
	v, ok := f.features[data[0]]
	if !ok {
		return -1, errors.New(tagFakeFeatureMissing, "no such feature report")
	}
	return copy(data, v), nil
}

func (f *fakeTransport) Close() error {
	f.once.Do(func() { close(f.closed) })
	return nil
}

func (f *fakeTransport) writes() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]byte(nil), f.written...)
}

// The tags of the errors of the doubles of this file.
const (
	tagFakeFeatureMissing  = "FakeFeatureMissing"
	tagFakeTransportClosed = "FakeTransportClosed"
)
