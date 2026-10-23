package webgpu

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func TestLossState(t *testing.T) {
	tests := []struct {
		name        string
		steps       func(s *lossState) []bool // the answers of the calls, in order
		want        []bool
		wantErrTag  string
		wantIsLost  bool
		wantClosedN int
	}{
		{"a loss is news once", func(s *lossState) []bool {
			return []bool{s.markLost("unknown", "the GPU reset"), s.markLost("unknown", "again")}
		}, []bool{true, false}, "GPUDeviceLost", true, 0},
		{"a device that was destroyed by its owner reports a loss that is not one", func(s *lossState) []bool {
			return []bool{s.markClosed(), s.markLost("destroyed", "Device was destroyed.")}
		}, []bool{true, false}, "", false, 1},
		{"closing twice is not news", func(s *lossState) []bool {
			return []bool{s.markClosed(), s.markClosed()}
		}, []bool{true, false}, "", false, 1},
		{"a lost device can still be closed", func(s *lossState) []bool {
			return []bool{s.markLost("unknown", "x"), s.markClosed()}
		}, []bool{true, true}, "GPUDeviceLost", true, 1},
		{"a device that nothing happened to", func(s *lossState) []bool { return nil }, nil, "", false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s lossState
			got := tt.steps(&s)
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("answers %v, want %v", got, tt.want)
					break
				}
			}
			err := s.err()
			if tt.wantErrTag == "" && err != nil {
				t.Errorf("err = %v, want none", err)
			}
			if tt.wantErrTag != "" {
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != tt.wantErrTag || !appErr.HasRoute("lossState.err.GPUDeviceLost") {
					t.Errorf("err = %v, want %s via lossState.err", err, tt.wantErrTag)
				}
			}
			if s.isLost() != tt.wantIsLost {
				t.Errorf("isLost = %v, want %v", s.isLost(), tt.wantIsLost)
			}
		})
	}
}

func TestLossErrorCarriesWhy(t *testing.T) {
	var s lossState
	s.markLost("destroyed", "Device was destroyed.")
	appErr, ok := errors.FromError(s.err())
	if !ok {
		t.Fatal("not an *errors.Error")
	}
	if appErr.Metadata["reason"] != "destroyed" || appErr.Metadata["message"] != "Device was destroyed." {
		t.Errorf("metadata = %v", appErr.Metadata)
	}
}
