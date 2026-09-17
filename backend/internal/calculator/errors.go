package calculator

import "fmt"

// ErrorCode identifies why a calculation was rejected. Codes are part of the
// public API contract: clients switch on them rather than parsing messages.
//
// MalformedOperand is not produced here — it originates in the transport layer,
// where operand text is decoded, and is declared alongside the others so the
// full set lives in one place.
type ErrorCode string

const (
	CodeUnknownOperation    ErrorCode = "UNKNOWN_OPERATION"
	CodeInvalidOperandCount ErrorCode = "INVALID_OPERAND_COUNT"
	CodeMalformedOperand    ErrorCode = "MALFORMED_OPERAND"
	CodeOperandOutOfRange   ErrorCode = "OPERAND_OUT_OF_RANGE"
	CodeDivisionByZero      ErrorCode = "DIVISION_BY_ZERO"
	CodeNegativeSqrt        ErrorCode = "NEGATIVE_SQRT"
	CodeResultTooLarge      ErrorCode = "RESULT_TOO_LARGE"
)

// Error is a rejected calculation. Field, when set, names the offending input
// in the request body, e.g. "operands[1]".
type Error struct {
	Code    ErrorCode
	Message string
	Field   string
}

func (e *Error) Error() string { return string(e.Code) + ": " + e.Message }

func newError(code ErrorCode, field, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...), Field: field}
}

func operandField(index int) string { return fmt.Sprintf("operands[%d]", index) }
