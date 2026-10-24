package main

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/perfstats"
)

func TestSummarizeDurations(t *testing.T) {
	ms := time.Millisecond
	tests := []struct {
		name string
		in   []time.Duration
		want durStats
	}{
		{"none", nil, durStats{}},
		{"one", []time.Duration{7 * ms}, durStats{N: 1, Min: 7 * ms, Median: 7 * ms, Max: 7 * ms}},
		{"five starts, the first the coldest", []time.Duration{900 * ms, 410 * ms, 400 * ms, 430 * ms, 420 * ms}, durStats{N: 5, Min: 400 * ms, Median: 420 * ms, Max: 900 * ms}},
		{"even count takes the upper middle", []time.Duration{1 * ms, 2 * ms, 3 * ms, 4 * ms}, durStats{N: 4, Min: 1 * ms, Median: 3 * ms, Max: 4 * ms}},
	}
	for _, tt := range tests {
		if got := summarizeDurations(tt.in); got != tt.want {
			t.Errorf("%s: summarizeDurations = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestCPUPercent(t *testing.T) {
	tests := []struct {
		name    string
		t0, t1  uint64
		elapsed time.Duration
		want    float64
	}{
		{"idle", 1834, 1834, 10 * time.Second, 0},
		{"one tick a second is 1%", 1000, 1010, 10 * time.Second, 1},
		{"a full core", 0, 1000, 10 * time.Second, 100},
		{"two cores", 0, 2000, 10 * time.Second, 200},
		{"a window of 10.5 s", 100, 121, 10500 * time.Millisecond, 2},
		{"no time", 1, 5, 0, 0},
		{"a counter that went back (a new process)", 50, 10, time.Second, 0},
	}
	for _, tt := range tests {
		if got := cpuPercent(tt.t0, tt.t1, tt.elapsed); math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("%s: cpuPercent = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestSummarizeFrames(t *testing.T) {
	ms := time.Millisecond
	tests := []struct {
		name string
		in   []perfstats.Frames
		want frameSummary
	}{
		{"none", nil, frameSummary{}},
		{
			name: "three seconds, one with slow frames",
			in: []perfstats.Frames{
				{FPS: 60, BuildAvg: 2 * ms, BuildMax: 5 * ms, RasterAvg: 1 * ms, RasterMax: 2 * ms, GoHeapMB: 10, GoSysMB: 30},
				{FPS: 40, BuildAvg: 6 * ms, BuildMax: 25 * ms, RasterAvg: 3 * ms, RasterMax: 9 * ms, Over16: 4, GoHeapMB: 14, GoSysMB: 34},
				{FPS: 60, BuildAvg: 2 * ms, BuildMax: 4 * ms, RasterAvg: 1 * ms, RasterMax: 2 * ms, GoHeapMB: 12, GoSysMB: 35},
			},
			want: frameSummary{
				Seconds: 3, Frames: 160, MeanFPS: 160.0 / 3, MinFPS: 40, MaxFPS: 60,
				// build: (2*60 + 6*40 + 2*60) / 160 = 3.0 ms; raster: (60 + 120 + 60) / 160 = 1.5 ms
				BuildAvg: 3 * ms, RasterAvg: 1500 * time.Microsecond, BuildMax: 25 * ms, RasterMax: 9 * ms, WorstSecond: 9 * ms,
				Over16: 4, Over16Percent: 2.5, HeapMaxMB: 14, HeapLastMB: 12, SysLastMB: 35,
			},
		},
		{
			name: "a second with no frames drawn (fps 0) does not divide by zero",
			in:   []perfstats.Frames{{FPS: 0, GoHeapMB: 3, GoSysMB: 9}},
			want: frameSummary{Seconds: 1, HeapMaxMB: 3, HeapLastMB: 3, SysLastMB: 9},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := summarizeFrames(tt.in); got != tt.want {
				t.Errorf("summarizeFrames =\n %+v\nwant\n %+v", got, tt.want)
			}
		})
	}
}
