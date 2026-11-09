package metal

import (
	"reflect"
	"testing"
)

func TestDirtyRangesMerge(t *testing.T) {
	tests := []struct {
		name string
		adds []span
		want []span
	}{
		{"nothing", nil, nil},
		{"one", []span{{10, 5}}, []span{{10, 5}}},
		{"empty and negative are ignored", []span{{10, 0}, {-1, 5}, {4, -2}}, nil},
		{"apart stay apart, sorted", []span{{100, 10}, {0, 10}}, []span{{0, 10}, {100, 10}}},
		{"touching merge", []span{{0, 10}, {10, 10}}, []span{{0, 20}}},
		{"overlapping merge", []span{{0, 10}, {5, 10}}, []span{{0, 15}}},
		{"contained", []span{{0, 100}, {10, 5}}, []span{{0, 100}}},
		{"a chain in reverse order", []span{{30, 10}, {20, 10}, {10, 10}, {0, 10}}, []span{{0, 40}}},
		{"two groups", []span{{0, 4}, {2, 4}, {50, 4}, {54, 4}}, []span{{0, 6}, {50, 8}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d dirtyRanges
			for _, s := range tt.adds {
				d.add(s.off, s.n)
			}
			if got := d.take(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("take = %v, want %v", got, tt.want)
			}
			if !d.empty() {
				t.Error("take left ranges behind")
			}
		})
	}
}
