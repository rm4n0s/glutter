package explicit

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func TestIsSwapchainOutOfDate(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"the error of an out-of-date swapchain", errors.New(SwapchainOutOfDate, "configure again"), true},
		{"another tagged error", errors.New("GPUOutOfMemory", "no memory"), false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSwapchainOutOfDate(tt.err); got != tt.want {
				t.Errorf("IsSwapchainOutOfDate = %v, want %v", got, tt.want)
			}
		})
	}
}
