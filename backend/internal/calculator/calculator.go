// Package calculator implements the arithmetic this service exposes. It knows
// nothing about HTTP: operations take and return decimals, and rejections are
// *Error values carrying a code that the transport layer maps onto a status.
package calculator

import (
	"sort"

	"github.com/shopspring/decimal"
)

// operation is one entry in the dispatch table. Arity is data rather than a
// per-handler check, which is what lets validation reject an unknown operation
// before any operand is examined. See
// docs/adr/0002-single-calculate-endpoint.md.
type operation struct {
	arity int
	apply func(operands []decimal.Decimal) (decimal.Decimal, error)
}

var operations = map[string]operation{
	"add":        {arity: 2, apply: add},
	"subtract":   {arity: 2, apply: subtract},
	"multiply":   {arity: 2, apply: multiply},
	"divide":     {arity: 2, apply: divide},
	"power":      {arity: 2, apply: power},
	"percentage": {arity: 2, apply: percentage},
	"sqrt":       {arity: 1, apply: sqrt},
}

// Operations returns the supported operation names in a stable order.
func Operations() []string {
	names := make([]string, 0, len(operations))
	for name := range operations {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Arity returns how many operands an operation requires.
func Arity(name string) (int, bool) {
	op, ok := operations[name]
	return op.arity, ok
}

// Calculate applies a named operation to its operands.
//
// Validation runs in a fixed order, so a given bad request always produces the
// same error: the operation must exist, then the operand count must match its
// arity, then every operand must be within range. Only then does arithmetic run.
func Calculate(name string, operands []decimal.Decimal) (decimal.Decimal, error) {
	op, ok := operations[name]
	if !ok {
		return decimal.Zero, newError(CodeUnknownOperation, "operation",
			"unknown operation %q", name)
	}

	if len(operands) != op.arity {
		return decimal.Zero, newError(CodeInvalidOperandCount, "operands",
			"operation %q requires %d operand(s), got %d",
			name, op.arity, len(operands))
	}

	for i, operand := range operands {
		if err := checkOperand(operand, i); err != nil {
			return decimal.Zero, err
		}
	}

	return op.apply(operands)
}

// Addition, subtraction and multiplication are exact and are never rounded.

func add(o []decimal.Decimal) (decimal.Decimal, error) { return o[0].Add(o[1]), nil }

func subtract(o []decimal.Decimal) (decimal.Decimal, error) { return o[0].Sub(o[1]), nil }

func multiply(o []decimal.Decimal) (decimal.Decimal, error) { return o[0].Mul(o[1]), nil }

func divide(o []decimal.Decimal) (decimal.Decimal, error) {
	dividend, divisor := o[0], o[1]
	if divisor.IsZero() {
		return decimal.Zero, newError(CodeDivisionByZero, operandField(1),
			"division by zero is undefined")
	}
	return divideSignificant(dividend, divisor, ResultPrecision), nil
}

// percentage answers "what percentage is a of b", with b as the total. See
// CONTEXT.md: the other common reading, "a percent of b", is a different
// function and the two disagree on every input.
func percentage(o []decimal.Decimal) (decimal.Decimal, error) {
	part, total := o[0], o[1]
	if total.IsZero() {
		return decimal.Zero, newError(CodeDivisionByZero, operandField(1),
			"a percentage of zero is undefined")
	}
	// Multiplying first keeps the operation to a single rounding step.
	return divideSignificant(part.Mul(decimal.NewFromInt(100)), total, ResultPrecision), nil
}

func sqrt(o []decimal.Decimal) (decimal.Decimal, error) {
	if o[0].IsNegative() {
		return decimal.Zero, newError(CodeNegativeSqrt, operandField(0),
			"the square root of a negative number is not a real number")
	}
	return sqrtNewton(o[0], ResultPrecision), nil
}

// power accepts integer exponents only. Fractional exponents would route through
// the same inaccurate library path that sqrt avoids, so they are rejected rather
// than approximated. See docs/adr/0004-newton-raphson-sqrt.md.
func power(o []decimal.Decimal) (decimal.Decimal, error) {
	base, exponent := o[0], o[1]

	if !exponent.IsInteger() {
		return decimal.Zero, newError(CodeOperandOutOfRange, operandField(1),
			"exponent must be a whole number; fractional exponents are not supported")
	}
	// Bound the exponent before reading it: IntPart silently returns 0 on int64
	// overflow, so 1e400 would otherwise be read as an exponent of zero.
	if exponent.Abs().GreaterThan(decimal.NewFromInt(MaxPowerExponent)) {
		return decimal.Zero, newError(CodeOperandOutOfRange, operandField(1),
			"exponent magnitude exceeds the maximum of %d", MaxPowerExponent)
	}
	exp := exponent.IntPart()

	// The decimal library panics rather than returning an error on division by
	// zero, and a negative exponent divides by the base.
	if base.IsZero() && exp < 0 {
		return decimal.Zero, newError(CodeDivisionByZero, operandField(0),
			"zero raised to a negative exponent is undefined")
	}

	// The library treats 0^0 as undefined and returns an error. This service
	// follows the convention used by IEEE 754 pow, Go's math.Pow and every
	// pocket calculator, and answers 1.
	if base.IsZero() && exp == 0 {
		return decimal.NewFromInt(1), nil
	}

	if digits := estimatedPowerDigits(base, exp); digits > MaxPowerResultDigits {
		return decimal.Zero, newError(CodeResultTooLarge, "operands",
			"the result would have roughly %d digits, the maximum is %d",
			digits, MaxPowerResultDigits)
	}

	result, err := base.PowInt32(int32(exp))
	if err != nil {
		return decimal.Zero, newError(CodeResultTooLarge, "operands",
			"the result could not be computed: %v", err)
	}
	if exp < 0 {
		// A negative exponent divides, so the result is generally inexact.
		return roundSignificant(result, ResultPrecision), nil
	}
	return result, nil
}

// estimatedPowerDigits approximates the size of base^exp without computing it.
// The span of a decimal is its coefficient length plus its exponent offset, and
// raising to a power multiplies that span.
func estimatedPowerDigits(base decimal.Decimal, exp int64) int {
	span := base.NumDigits() + abs(int(base.Exponent()))
	return span * abs(int(exp))
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
