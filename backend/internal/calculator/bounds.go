package calculator

import "github.com/shopspring/decimal"

// The decimal type is arbitrary-precision: it has no inherent ceiling, so these
// limits are a deliberate part of the contract rather than a property of a
// number type. Without them a 45-byte request can allocate megabytes and occupy
// a core for seconds. See docs/adr/0001-decimal-arithmetic.md.
const (
	// MaxSignificantDigits caps the coefficient of any operand.
	MaxSignificantDigits = 100

	// MaxMagnitude caps an operand's size: values must lie within
	// ±10^MaxMagnitude, and non-zero values must be no smaller than
	// 10^-MaxMagnitude.
	MaxMagnitude = 1000

	// MaxPowerExponent caps the exponent accepted by power.
	MaxPowerExponent = 1000

	// MaxPowerResultDigits caps the estimated size of a power result. Operand
	// bounds cannot catch this case: power(10, 1000000) has tiny operands and a
	// result of a million digits.
	//
	// The limit is set so that squaring the largest accepted operand is allowed,
	// since multiplying it by itself already produces a result of about this
	// size. A tighter cap would reject power(x, 2) while permitting the
	// identical multiply(x, x).
	MaxPowerResultDigits = 2100

	// ResultPrecision is the number of significant digits retained for results
	// that cannot be represented exactly — division, percentage and square
	// root. Exact operations are never rounded.
	ResultPrecision = 16

	// maxDivisionPlaces bounds the working precision of a single division, so
	// that operands at opposite ends of the magnitude range cannot force an
	// unbounded computation.
	maxDivisionPlaces = 4096
)

// magnitude returns the power of ten of a value's leading digit: 0 for 1.0,
// -1 for 0.5, 2 for 999, 400 for 1e400. Zero has no magnitude and returns 0.
func magnitude(d decimal.Decimal) int {
	if d.IsZero() {
		return 0
	}
	return int(d.Exponent()) + d.NumDigits() - 1
}

// checkOperand rejects operands outside the accepted range.
func checkOperand(d decimal.Decimal, index int) *Error {
	if digits := d.NumDigits(); digits > MaxSignificantDigits {
		return newError(CodeOperandOutOfRange, operandField(index),
			"operand has %d significant digits, the maximum is %d",
			digits, MaxSignificantDigits)
	}
	if d.IsZero() {
		return nil
	}
	if mag := magnitude(d); mag > MaxMagnitude || mag < -MaxMagnitude {
		return newError(CodeOperandOutOfRange, operandField(index),
			"operand magnitude 1e%d is outside the accepted range 1e-%d to 1e%d",
			mag, MaxMagnitude, MaxMagnitude)
	}
	return nil
}

// roundSignificant rounds to a number of significant digits rather than decimal
// places. decimal.Round counts places, which silently flattens small values to
// zero: rounding 1e-500 to 16 places gives 0.
func roundSignificant(d decimal.Decimal, digits int) decimal.Decimal {
	if d.IsZero() {
		return d
	}
	places := digits - magnitude(d) - 1
	if places < 0 {
		places = 0
	}
	return d.Round(int32(places))
}

// divideSignificant divides to a fixed number of significant digits.
//
// decimal.DivRound takes decimal places, so the working precision has to be
// derived from the magnitude of the quotient; dividing 1e-500 by 3 at 16 places
// would otherwise return zero. The caller must have excluded a zero divisor:
// the library panics rather than returning an error.
func divideSignificant(a, b decimal.Decimal, digits int) decimal.Decimal {
	if a.IsZero() {
		return decimal.Zero
	}
	places := digits - (magnitude(a) - magnitude(b)) + 5
	if places < digits {
		places = digits
	}
	if places > maxDivisionPlaces {
		places = maxDivisionPlaces
	}
	return roundSignificant(a.DivRound(b, int32(places)), digits)
}
