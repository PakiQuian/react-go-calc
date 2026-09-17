package calculator

import (
	"math"

	"github.com/shopspring/decimal"
)

var two = decimal.NewFromInt(2)

// sqrtNewton computes a square root by Newton-Raphson iteration:
//
//	x[n+1] = (x[n] + d/x[n]) / 2
//
// Each iteration doubles the number of correct digits, so convergence takes a
// handful of steps across the entire accepted operand range.
//
// PowWithPrecision(0.5, n) from the decimal library is not used. It routes
// through logarithm and exponential series that are both slow and inaccurate at
// large magnitudes: sqrt(1e400) comes back wrong from the 27th digit, and the
// largest accepted operand takes over thirty seconds. See
// docs/adr/0004-newton-raphson-sqrt.md for the measurements.
func sqrtNewton(d decimal.Decimal, digits int) decimal.Decimal {
	if d.IsZero() {
		return decimal.Zero
	}

	// A square root has roughly half the magnitude of its operand, which sets
	// both the working precision and the initial guess.
	resultMag := magnitude(d) / 2

	work := int32(digits - resultMag + 10)
	if work < int32(digits) {
		work = int32(digits)
	}
	if work > maxDivisionPlaces {
		work = maxDivisionPlaces
	}

	x := initialGuess(d, resultMag)

	// The iteration is self-correcting, so a poor guess costs iterations rather
	// than accuracy. The cap only bounds pathological input.
	epsilon := decimal.New(1, -work)
	for i := 0; i < 200; i++ {
		next := x.Add(d.DivRound(x, work)).DivRound(two, work)
		if next.Sub(x).Abs().LessThanOrEqual(epsilon) {
			x = next
			break
		}
		x = next
	}
	return roundSignificant(x, digits)
}

// initialGuess seeds the iteration. float64 covers the common range directly;
// outside it, 10^(magnitude/2) is close enough that convergence still takes only
// a few iterations.
func initialGuess(d decimal.Decimal, resultMag int) decimal.Decimal {
	if f, _ := d.Float64(); f > 0 && !math.IsInf(f, 0) {
		if g := math.Sqrt(f); g > 0 && !math.IsInf(g, 0) {
			return decimal.NewFromFloat(g)
		}
	}
	return decimal.New(1, int32(resultMag))
}
