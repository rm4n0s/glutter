package gestures

import (
	"math"
	"testing"
)

func TestLeastSquaresSolver(t *testing.T) {
	x := []float64{0, 1, 2, 3, 4, 5, 6, 7, 8}
	ones := []float64{1, 1, 1, 1, 1, 1, 1, 1, 1}
	slope := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}
	for _, tc := range []struct {
		name   string
		y      []float64
		degree int
		want   []float64
	}{
		{"linear polynomial to line", ones, 1, []float64{1, 0}},
		{"linear polynomial to sloped line", slope, 1, []float64{1, 1}},
		{"quadratic polynomial to line", ones, 2, []float64{1, 0, 0}},
		{"quadratic polynomial to sloped line", slope, 2, []float64{1, 1, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fit := NewLeastSquaresSolver(x, tc.y, ones).Solve(tc.degree)
			if fit == nil {
				t.Fatal("no fit")
			}
			if len(fit.Coefficients) != len(tc.want) {
				t.Fatalf("got %d coefficients, want %d", len(fit.Coefficients), len(tc.want))
			}
			for i, w := range tc.want {
				if math.Abs(fit.Coefficients[i]-w) >= 1e-6 {
					t.Errorf("coefficient %d = %v, want %v", i, fit.Coefficients[i], w)
				}
			}
			if math.Abs(fit.Confidence-1) >= 1e-6 {
				t.Errorf("confidence = %v, want 1", fit.Confidence)
			}
		})
	}
}

func TestPolynomialFitString(t *testing.T) {
	fit := NewPolynomialFit(2)
	fit.Coefficients[0], fit.Coefficients[1], fit.Coefficients[2] = 123.45, 54.321, 1.3579
	fit.Confidence = 0.9876
	if got, want := fit.String(), "PolynomialFit([123, 54.3, 1.36], confidence: 0.988)"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
