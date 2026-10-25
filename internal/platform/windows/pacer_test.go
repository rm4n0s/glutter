package windows

import (
	"testing"
	"time"
)

func TestIntervalFor(t *testing.T) {
	tests := []struct {
		rate float64
		want time.Duration
	}{
		{60, time.Second / 60}, {144, time.Second / 144}, {0, time.Second / 60}, {1, time.Second / 60},
		{-5, time.Second / 60}, {5000, time.Second / 60}, {59.94, 16683350 * time.Nanosecond},
	}
	for _, tt := range tests {
		if got := intervalFor(tt.rate); got != tt.want {
			t.Errorf("intervalFor(%v) = %v, want %v", tt.rate, got, tt.want)
		}
	}
}

// flush is one DwmFlush that the pacer is told about.
type flush struct {
	took   time.Duration
	failed bool
}

func TestPacerFollowsDwmFlushUntilItDoesNotPace(t *testing.T) {
	const interval = time.Second / 60
	normal := flush{took: interval - time.Millisecond}
	atOnce := flush{took: 20 * time.Microsecond}
	tests := []struct {
		name      string
		flushes   []flush
		wantTimer bool
	}{
		{"normal flushes keep DwmFlush", []flush{normal, normal, normal}, false},
		{"one that returns at once is tolerated", []flush{atOnce, normal, atOnce, normal}, false},
		{"two in a row that return at once give it up", []flush{atOnce, atOnce}, true},
		{"two at once after a normal one", []flush{normal, atOnce, atOnce}, true},
		{"a failure gives it up", []flush{normal, {failed: true}}, true},
		{"a failure at the first call", []flush{{failed: true}}, true},
		{"a flush of a quarter of the interval is not too quick", []flush{{took: interval / 4}, {took: interval / 4}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPacer(60)
			for _, f := range tt.flushes {
				p.flushed(f.took, f.failed)
			}
			if p.usesTimer() != tt.wantTimer {
				t.Errorf("usesTimer = %v, want %v", p.usesTimer(), tt.wantTimer)
			}
		})
	}
}

func TestPacerActions(t *testing.T) {
	p := newPacer(100) // 10 ms
	if act, _ := p.action(false, 0); act != pacerIdle {
		t.Errorf("no frame wanted: action = %v, want idle (a wanted frame while none is wanted must not tick)", act)
	}
	if act, _ := p.action(true, 0); act != pacerFlush {
		t.Errorf("action = %v, want flush", act)
	}
	p.flushed(0, true) // DwmFlush does not pace (Wine)
	if act, _ := p.action(false, 0); act != pacerIdle {
		t.Errorf("in timer mode with no frame wanted: action = %v, want idle", act)
	}
	tests := []struct {
		since time.Duration
		want  time.Duration
	}{
		{0, 10 * time.Millisecond}, {4 * time.Millisecond, 6 * time.Millisecond}, {10 * time.Millisecond, 0}, {50 * time.Millisecond, 0},
	}
	for _, tt := range tests {
		act, wait := p.action(true, tt.since)
		if act != pacerSleep || wait != tt.want {
			t.Errorf("since %v: %v, %v; want sleep, %v", tt.since, act, wait, tt.want)
		}
	}
	p.flushed(time.Hour, false) // no longer looked at
	p.setRefreshRate(0)         // 0 is 60 Hz
	if _, wait := p.action(true, 0); wait != time.Second/60 {
		t.Errorf("after setRefreshRate(0): wait = %v, want 1/60 s", wait)
	}
}
