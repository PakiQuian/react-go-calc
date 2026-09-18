package calculator

import (
	"errors"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

// dec parses a decimal literal for use in tests, failing loudly on bad input.
func dec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("dec(%q): %v", s, err)
	}
	return d
}

func decs(t *testing.T, values ...string) []decimal.Decimal {
	t.Helper()
	out := make([]decimal.Decimal, 0, len(values))
	for _, v := range values {
		out = append(out, dec(t, v))
	}
	return out
}

func TestCalculateResults(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		operands  []string
		want      string
	}{
		// Exactness: these are the cases binary floating point gets wrong.
		// float64 gives 0.30000000000000004 and 3.3000000000000003.
		{"add is exact for decimal fractions", "add", []string{"0.1", "0.2"}, "0.3"},
		{"multiply is exact for decimal fractions", "multiply", []string{"1.1", "3"}, "3.3"},
		{"subtract is exact for decimal fractions", "subtract", []string{"0.3", "0.1"}, "0.2"},

		{"add negatives", "add", []string{"-5", "3"}, "-2"},
		{"subtract to negative", "subtract", []string{"3", "5"}, "-2"},
		{"multiply by zero", "multiply", []string{"12345.6789", "0"}, "0"},
		{"add very large and very small", "add", []string{"1e100", "1e-100"},
			"1" + strings.Repeat("0", 100) + "." + strings.Repeat("0", 99) + "1"},

		{"divide exact", "divide", []string{"10", "4"}, "2.5"},
		{"divide repeating", "divide", []string{"1", "3"}, "0.3333333333333333"},
		{"divide repeating rounds up", "divide", []string{"2", "3"}, "0.6666666666666667"},
		{"divide negative", "divide", []string{"-10", "4"}, "-2.5"},
		{"divide small values", "divide", []string{"1e-20", "4"}, "0.0000000000000000000025"},

		// percentage(a, b) is "what percentage a is of b" — see CONTEXT.md.
		{"percentage basic", "percentage", []string{"1", "10"}, "10"},
		{"percentage fractional", "percentage", []string{"25", "200"}, "12.5"},
		{"percentage repeating", "percentage", []string{"1", "3"}, "33.33333333333333"},
		{"percentage above one hundred", "percentage", []string{"30", "20"}, "150"},
		{"percentage of negative total", "percentage", []string{"1", "-10"}, "-10"},

		{"power positive exponent", "power", []string{"2", "10"}, "1024"},
		{"power zero exponent", "power", []string{"7", "0"}, "1"},
		{"power of zero", "power", []string{"0", "0"}, "1"},
		{"power negative exponent", "power", []string{"2", "-3"}, "0.125"},
		{"power negative base", "power", []string{"-2", "3"}, "-8"},
		{"power decimal base", "power", []string{"1.5", "2"}, "2.25"},

		{"sqrt perfect square", "sqrt", []string{"4"}, "2"},
		{"sqrt of zero", "sqrt", []string{"0"}, "0"},
		{"sqrt of one", "sqrt", []string{"1"}, "1"},
		{"sqrt irrational", "sqrt", []string{"2"}, "1.414213562373095"},
		{"sqrt large perfect square", "sqrt", []string{"152399025"}, "12345"},
		{"sqrt of a decimal", "sqrt", []string{"0.25"}, "0.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Calculate(tt.operation, decs(t, tt.operands...))
			if err != nil {
				t.Fatalf("Calculate(%q, %v) returned error: %v", tt.operation, tt.operands, err)
			}
			if got.String() != tt.want {
				t.Errorf("Calculate(%q, %v) = %s, want %s",
					tt.operation, tt.operands, got.String(), tt.want)
			}
		})
	}
}

func TestCalculateRejections(t *testing.T) {
	hundredAndOneDigits := strings.Repeat("9", 101)

	tests := []struct {
		name      string
		operation string
		operands  []string
		wantCode  ErrorCode
		wantField string
	}{
		{"unknown operation", "modulo", []string{"1", "2"},
			CodeUnknownOperation, "operation"},
		{"empty operation name", "", []string{"1", "2"},
			CodeUnknownOperation, "operation"},

		{"too few operands", "add", []string{"1"},
			CodeInvalidOperandCount, "operands"},
		{"too many operands", "add", []string{"1", "2", "3"},
			CodeInvalidOperandCount, "operands"},
		{"unary operation given two operands", "sqrt", []string{"4", "9"},
			CodeInvalidOperandCount, "operands"},
		{"binary operation given none", "multiply", nil,
			CodeInvalidOperandCount, "operands"},

		{"division by zero", "divide", []string{"1", "0"},
			CodeDivisionByZero, "operands[1]"},
		{"division of zero by zero", "divide", []string{"0", "0"},
			CodeDivisionByZero, "operands[1]"},
		{"percentage of zero total", "percentage", []string{"5", "0"},
			CodeDivisionByZero, "operands[1]"},
		{"zero to a negative power", "power", []string{"0", "-1"},
			CodeDivisionByZero, "operands[0]"},

		{"square root of a negative", "sqrt", []string{"-4"},
			CodeNegativeSqrt, "operands[0]"},

		{"too many significant digits", "add", []string{hundredAndOneDigits, "1"},
			CodeOperandOutOfRange, "operands[0]"},
		{"magnitude too large", "add", []string{"1", "1e1001"},
			CodeOperandOutOfRange, "operands[1]"},
		{"magnitude too small", "add", []string{"1e-1001", "1"},
			CodeOperandOutOfRange, "operands[0]"},

		{"fractional exponent", "power", []string{"2", "0.5"},
			CodeOperandOutOfRange, "operands[1]"},
		{"exponent beyond the limit", "power", []string{"2", "1001"},
			CodeOperandOutOfRange, "operands[1]"},
		// 1e400 is a legal operand but overflows int64; it must be rejected by
		// the bound rather than silently read as an exponent of zero.
		{"exponent that overflows int64", "power", []string{"2", "1e400"},
			CodeOperandOutOfRange, "operands[1]"},

		{"result too large", "power", []string{"99999999999999999999", "1000"},
			CodeResultTooLarge, "operands"},
		{"result too large from a big base", "power", []string{"1e400", "100"},
			CodeResultTooLarge, "operands"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Calculate(tt.operation, decs(t, tt.operands...))
			if err == nil {
				t.Fatalf("Calculate(%q, %v) succeeded, want %s",
					tt.operation, tt.operands, tt.wantCode)
			}

			var calcErr *Error
			if !errors.As(err, &calcErr) {
				t.Fatalf("error is %T, want *calculator.Error", err)
			}
			if calcErr.Code != tt.wantCode {
				t.Errorf("code = %s, want %s", calcErr.Code, tt.wantCode)
			}
			if calcErr.Field != tt.wantField {
				t.Errorf("field = %q, want %q", calcErr.Field, tt.wantField)
			}
			if calcErr.Message == "" {
				t.Error("message is empty")
			}
		})
	}
}

// Validation is ordered so that one bad request always produces one predictable
// error, rather than depending on which check happens to run first.
func TestValidationOrder(t *testing.T) {
	t.Run("unknown operation beats operand count", func(t *testing.T) {
		_, err := Calculate("modulo", nil)
		assertCode(t, err, CodeUnknownOperation)
	})

	t.Run("unknown operation beats out-of-range operands", func(t *testing.T) {
		_, err := Calculate("modulo", decs(t, "1e5000"))
		assertCode(t, err, CodeUnknownOperation)
	})

	t.Run("operand count beats operand range", func(t *testing.T) {
		_, err := Calculate("add", decs(t, "1e1001"))
		assertCode(t, err, CodeInvalidOperandCount)
	})

	t.Run("operand range beats arithmetic errors", func(t *testing.T) {
		// Division by zero would be reported were the operands in range.
		_, err := Calculate("divide", decs(t, "1e1001", "0"))
		assertCode(t, err, CodeOperandOutOfRange)
	})
}

// The bounds exist to stop a tiny request from consuming unbounded resources.
// These inputs are the ones that motivated them.
func TestBoundsRejectResourceExhaustion(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		operands  []string
	}{
		{"huge exponent on an operand", "add", []string{"1e1000000", "1"}},
		{"huge negative exponent", "multiply", []string{"1e-1000000", "1"}},
		{"power producing millions of digits", "power", []string{"10", "1000000"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Calculate(tt.operation, decs(t, tt.operands...)); err == nil {
				t.Fatalf("Calculate(%q, %v) succeeded, want rejection",
					tt.operation, tt.operands)
			}
		})
	}
}

// Every accepted operand must be computable quickly. This is the guarantee the
// bounds are meant to provide, so it is asserted rather than assumed.
func TestOperationsAreFastAtTheBounds(t *testing.T) {
	largest := strings.Repeat("9", MaxSignificantDigits) + "e901"    // magnitude 1000
	smallest := strings.Repeat("9", MaxSignificantDigits) + "e-1099" // magnitude -1000

	cases := []struct {
		operation string
		operands  []string
	}{
		{"add", []string{largest, smallest}},
		{"subtract", []string{largest, smallest}},
		{"multiply", []string{largest, largest}},
		{"divide", []string{largest, smallest}},
		{"divide", []string{smallest, largest}},
		{"percentage", []string{smallest, largest}},
		{"sqrt", []string{largest}},
		{"sqrt", []string{smallest}},
		{"power", []string{largest, "1"}},
	}

	for _, tc := range cases {
		t.Run(tc.operation, func(t *testing.T) {
			if _, err := Calculate(tc.operation, decs(t, tc.operands...)); err != nil {
				t.Fatalf("Calculate(%q, %v) = %v, want success",
					tc.operation, tc.operands, err)
			}
		})
	}
}

func TestOperationsAndArity(t *testing.T) {
	want := []string{"add", "divide", "multiply", "percentage", "power", "sqrt", "subtract"}
	got := Operations()

	if len(got) != len(want) {
		t.Fatalf("Operations() returned %d names, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Operations()[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	if arity, ok := Arity("sqrt"); !ok || arity != 1 {
		t.Errorf("Arity(sqrt) = %d, %v; want 1, true", arity, ok)
	}
	if arity, ok := Arity("add"); !ok || arity != 2 {
		t.Errorf("Arity(add) = %d, %v; want 2, true", arity, ok)
	}
	if _, ok := Arity("modulo"); ok {
		t.Error("Arity(modulo) reported a known operation")
	}
}

func assertCode(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %s, got nil", want)
	}
	var calcErr *Error
	if !errors.As(err, &calcErr) {
		t.Fatalf("error is %T, want *calculator.Error", err)
	}
	if calcErr.Code != want {
		t.Errorf("code = %s, want %s", calcErr.Code, want)
	}
}

// decimal.DivRound counts decimal places, not significant digits, so dividing
// values near the bottom of the accepted magnitude range at a fixed precision
// would silently return zero. See divideSignificant.
func TestDivisionKeepsSignificanceAtExtremeMagnitudes(t *testing.T) {
	tests := []struct {
		name     string
		operands []string
		want     string
	}{
		{"tiny dividend", []string{"1e-500", "3"}, "3.333333333333333e-501"},
		{"tiny over tiny", []string{"1e-900", "1e-800"}, "1e-100"},
		{"huge over tiny", []string{"1e900", "1e-900"}, "1e1800"},
		{"percentage of a tiny total", []string{"1e-900", "1e-898"}, "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operation := "divide"
			if tt.name == "percentage of a tiny total" {
				operation = "percentage"
			}
			got, err := Calculate(operation, decs(t, tt.operands...))
			if err != nil {
				t.Fatalf("Calculate(%q, %v): %v", operation, tt.operands, err)
			}
			if got.IsZero() {
				t.Fatalf("Calculate(%q, %v) collapsed to zero", operation, tt.operands)
			}
			if want := dec(t, tt.want); !got.Equal(want) {
				t.Errorf("Calculate(%q, %v) = %s, want %s",
					operation, tt.operands, got.String(), want.String())
			}
		})
	}
}

// Newton-Raphson is hand-written, so its accuracy is asserted directly rather
// than assumed: squaring the result must reproduce the operand.
func TestSqrtAccuracy(t *testing.T) {
	t.Run("exact for large perfect powers of ten", func(t *testing.T) {
		got, err := Calculate("sqrt", decs(t, "1e400"))
		if err != nil {
			t.Fatalf("sqrt(1e400): %v", err)
		}
		// This is the case where the library implementation returns a value
		// wrong from the 27th digit. See docs/adr/0004-newton-raphson-sqrt.md.
		if want := dec(t, "1e200"); !got.Equal(want) {
			t.Errorf("sqrt(1e400) = %s, want %s", got.String(), want.String())
		}
	})

	t.Run("squaring the root reproduces the operand", func(t *testing.T) {
		for _, input := range []string{"2", "3", "10", "0.5", "1e-900", "1e900", "152399025"} {
			operand := dec(t, input)
			root, err := Calculate("sqrt", []decimal.Decimal{operand})
			if err != nil {
				t.Fatalf("sqrt(%s): %v", input, err)
			}

			// Compare relative error, since the root is rounded to
			// ResultPrecision significant digits.
			squared := root.Mul(root)
			relative := squared.Sub(operand).Abs().DivRound(operand.Abs(), 40)
			tolerance := decimal.New(1, -(ResultPrecision - 2))
			if relative.GreaterThan(tolerance) {
				t.Errorf("sqrt(%s) = %s; squared gives relative error %s, tolerance %s",
					input, root.String(), relative.String(), tolerance.String())
			}
		}
	})
}

// The Error method is what appears in server logs, so it carries the code.
func TestErrorMessage(t *testing.T) {
	_, err := Calculate("divide", decs(t, "1", "0"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if got, want := err.Error(), "DIVISION_BY_ZERO: division by zero is undefined"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
