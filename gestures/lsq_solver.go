package gestures

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
)

// lsqVector is a view on a row of an lsqMatrix, or a vector of its own.
type lsqVector []float64

func (v lsqVector) dot(a lsqVector) float64 {
	result := 0.0
	for i := range v {
		result += v[i] * a[i]
	}
	return result
}

func (v lsqVector) norm() float64 { return math.Sqrt(v.dot(v)) }

type lsqMatrix struct {
	columns  int
	elements []float64
}

func newLSQMatrix(rows, cols int) *lsqMatrix {
	return &lsqMatrix{columns: cols, elements: make([]float64, rows*cols)}
}

func (m *lsqMatrix) get(row, col int) float64 { return m.elements[row*m.columns+col] }

func (m *lsqMatrix) set(row, col int, value float64) { m.elements[row*m.columns+col] = value }

func (m *lsqMatrix) getRow(row int) lsqVector {
	return lsqVector(m.elements[row*m.columns : (row+1)*m.columns])
}

// PolynomialFit is a polynomial that fits some points: the coefficients, from
// the constant term up, and how well it fits (1 is a perfect fit).
//
// Flutter: gestures/lsq_solver.dart PolynomialFit
type PolynomialFit struct {
	// Coefficients are the coefficients of the polynomial, lowest degree
	// first.
	//
	// Flutter: gestures/lsq_solver.dart PolynomialFit.coefficients
	Coefficients []float64
	// Confidence is the coefficient of determination of the fit.
	//
	// Flutter: gestures/lsq_solver.dart PolynomialFit.confidence
	Confidence float64
}

// NewPolynomialFit creates a fit of the degree, with zero coefficients.
//
// Flutter: gestures/lsq_solver.dart PolynomialFit.new
func NewPolynomialFit(degree int) *PolynomialFit {
	return &PolynomialFit{Coefficients: make([]float64, degree+1)}
}

// String describes the fit.
//
// Flutter: gestures/lsq_solver.dart PolynomialFit.toString
func (f *PolynomialFit) String() string {
	parts := make([]string, len(f.Coefficients))
	for i, c := range f.Coefficients {
		parts[i] = toStringAsPrecision3(c)
	}
	return fmt.Sprintf("PolynomialFit([%s], confidence: %s)", strings.Join(parts, ", "), dartfmt.ToStringAsFixed(f.Confidence, 3))
}

// toStringAsPrecision3 formats x with three significant digits as Dart's
// toStringAsPrecision(3) does: exponent form when the exponent is below -6 or
// at least 3.
func toStringAsPrecision3(x float64) string {
	switch {
	case math.IsNaN(x):
		return "NaN"
	case math.IsInf(x, 1):
		return "Infinity"
	case math.IsInf(x, -1):
		return "-Infinity"
	case x == 0:
		if math.Signbit(x) {
			return "-0.00"
		}
		return "0.00"
	}
	e := strconv.FormatFloat(x, 'e', 2, 64)
	i := strings.IndexByte(e, 'e')
	exp, _ := strconv.Atoi(e[i+1:])
	if exp < -6 || exp >= 3 {
		mant := e[:i]
		sign := "+"
		if exp < 0 {
			sign = "-"
			exp = -exp
		}
		return mant + "e" + sign + strconv.Itoa(exp)
	}
	return strconv.FormatFloat(x, 'f', 2-exp, 64)
}

// LeastSquaresSolver fits a polynomial to points by the method of least
// squares, weighting each point.
//
// Flutter: gestures/lsq_solver.dart LeastSquaresSolver
type LeastSquaresSolver struct {
	x, y, w []float64
}

// NewLeastSquaresSolver creates a solver for the points (x[i], y[i]) with the
// weights w[i]. The three slices must have the same length, or a debug build
// panics with the tag LeastSquaresLengthMismatch.
//
// Flutter: gestures/lsq_solver.dart LeastSquaresSolver.new
func NewLeastSquaresSolver(x, y, w []float64) *LeastSquaresSolver {
	if foundation.DebugMode && (len(x) != len(y) || len(y) != len(w)) {
		panic(errors.New(tagLeastSquaresLengthMismatch, "x, y and w must have the same length",
			"x", len(x), "y", len(y), "w", len(w)))
	}
	return &LeastSquaresSolver{x: x, y: y, w: w}
}

// Solve fits a polynomial of the degree. It returns nil if there are fewer
// points than the degree, or if the points do not determine a polynomial.
//
// Flutter: gestures/lsq_solver.dart LeastSquaresSolver.solve
func (s *LeastSquaresSolver) Solve(degree int) *PolynomialFit {
	if degree > len(s.x) {
		// Not enough data to fit a curve.
		return nil
	}
	result := NewPolynomialFit(degree)
	x, y, w := s.x, s.y, s.w

	// Shorthands for the purpose of notation equivalence to original C++ code.
	m := len(x)
	n := degree + 1

	// Expand the X vector to a matrix A, pre-multiplied by the weights.
	a := newLSQMatrix(n, m)
	for h := 0; h < m; h++ {
		a.set(0, h, w[h])
		for i := 1; i < n; i++ {
			a.set(i, h, a.get(i-1, h)*x[h])
		}
	}

	// Apply the Gram-Schmidt process to A to obtain its QR decomposition.

	// Orthonormal basis, column-major order.
	q := newLSQMatrix(n, m)
	// Upper triangular matrix, row-major order.
	r := newLSQMatrix(n, n)
	for j := 0; j < n; j++ {
		for h := 0; h < m; h++ {
			q.set(j, h, a.get(j, h))
		}
		for i := 0; i < j; i++ {
			dot := q.getRow(j).dot(q.getRow(i))
			for h := 0; h < m; h++ {
				q.set(j, h, q.get(j, h)-dot*q.get(i, h))
			}
		}
		norm := q.getRow(j).norm()
		if norm < foundation.PrecisionErrorTolerance {
			// Vectors are linearly dependent or zero so no solution.
			return nil
		}
		inverseNorm := 1.0 / norm
		for h := 0; h < m; h++ {
			q.set(j, h, q.get(j, h)*inverseNorm)
		}
		for i := 0; i < n; i++ {
			if i < j {
				r.set(j, i, 0)
			} else {
				r.set(j, i, q.getRow(j).dot(a.getRow(i)))
			}
		}
	}

	// Solve R B = Qt W Y to find B. This is easy because R is upper triangular.
	// We just work from bottom-right to top-left calculating B's coefficients.
	wy := make(lsqVector, m)
	for h := 0; h < m; h++ {
		wy[h] = y[h] * w[h]
	}
	for i := n - 1; i >= 0; i-- {
		result.Coefficients[i] = q.getRow(i).dot(wy)
		for j := n - 1; j > i; j-- {
			result.Coefficients[i] -= r.get(i, j) * result.Coefficients[j]
		}
		result.Coefficients[i] /= r.get(i, i)
	}

	// Calculate the coefficient of determination (confidence) as:
	//   1 - (sumSquaredError / sumSquaredTotal)
	// ...where sumSquaredError is the residual sum of squares (variance of the
	// error), and sumSquaredTotal is the total sum of squares (variance of the
	// data) where each has been weighted.
	yMean := 0.0
	for h := 0; h < m; h++ {
		yMean += y[h]
	}
	yMean /= float64(m)

	sumSquaredError := 0.0
	sumSquaredTotal := 0.0
	for h := 0; h < m; h++ {
		term := 1.0
		err := y[h] - result.Coefficients[0]
		for i := 1; i < n; i++ {
			term *= x[h]
			err -= term * result.Coefficients[i]
		}
		sumSquaredError += w[h] * w[h] * err * err
		v := y[h] - yMean
		sumSquaredTotal += w[h] * w[h] * v * v
	}

	if sumSquaredTotal <= foundation.PrecisionErrorTolerance {
		result.Confidence = 1.0
	} else {
		result.Confidence = 1.0 - (sumSquaredError / sumSquaredTotal)
	}
	return result
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagLeastSquaresLengthMismatch = "LeastSquaresLengthMismatch"
)
