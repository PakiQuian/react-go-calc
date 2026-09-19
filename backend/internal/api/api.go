// Package api exposes the calculator over HTTP. It owns everything the domain
// package deliberately does not know about: request decoding, the mapping from
// rejection codes to status codes, and the JSON error envelope.
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/PakiQuian/react-go-calc/backend/internal/calculator"
)

// maxRequestBytes bounds the request body. A calculation request is a few dozen
// bytes; anything larger is a mistake or an attack. nginx applies its own limit
// in front of this one, but a service defends itself rather than relying on the
// proxy it happens to be deployed behind.
const maxRequestBytes = 1 << 10

// codeRequestTooLarge is a transport-level rejection with no domain equivalent,
// so it is declared here rather than in the calculator package.
const codeRequestTooLarge = "REQUEST_TOO_LARGE"

// codeInternalError is returned only if a handler panics. Reaching it is a bug.
const codeInternalError = "INTERNAL_ERROR"

// operand wraps decimal.Decimal only to reject JSON null.
//
// decimal.Decimal's own UnmarshalJSON accepts null as a no-op, leaving the
// zero value, so {"operands":["1",null]} would decode as [1, 0] and be
// answered rather than rejected — and divide(1, null) would report
// DIVISION_BY_ZERO, blaming the mathematics for a malformed request.
type operand struct {
	decimal.Decimal
}

func (o *operand) UnmarshalJSON(data []byte) error {
	if string(bytes.TrimSpace(data)) == "null" {
		return errors.New("operand must be a number, not null")
	}
	return o.Decimal.UnmarshalJSON(data)
}

type calculateRequest struct {
	Operation string    `json:"operation"`
	Operands  []operand `json:"operands"`
}

// decimals unwraps the operands for the calculator, which has no reason to
// know about the transport layer's null handling.
func (r calculateRequest) decimals() []decimal.Decimal {
	out := make([]decimal.Decimal, len(r.Operands))
	for i, o := range r.Operands {
		out[i] = o.Decimal
	}
	return out
}

type calculateResponse struct {
	Operation string            `json:"operation"`
	Operands  []decimal.Decimal `json:"operands"`
	Result    decimal.Decimal   `json:"result"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

// NewRouter builds the HTTP handler for the service.
func NewRouter(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	// Go 1.22 method patterns answer a wrong method with 405 automatically, so
	// no router library is needed for two routes. See ADR 0003.
	mux.HandleFunc("POST /api/v1/calculate", handleCalculate)
	mux.HandleFunc("GET /api/v1/health", handleHealth)

	// Unmatched paths would otherwise get net/http's plain-text 404, which is
	// jarring for a client that expects JSON everywhere.
	mux.HandleFunc("/", handleNotFound)

	return recoverPanics(logger)(logRequests(logger)(mux))
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// routeMethods records which methods each route accepts.
//
// net/http answers an unmatched method with 405 on its own, but only when no
// other pattern matches. The "/" catch-all registered for JSON 404s matches
// everything, so that behaviour has to be reproduced here.
var routeMethods = map[string][]string{
	"/api/v1/calculate": {http.MethodPost},
	"/api/v1/health":    {http.MethodGet},
}

func handleNotFound(w http.ResponseWriter, r *http.Request) {
	if methods, ok := routeMethods[r.URL.Path]; ok {
		w.Header().Set("Allow", strings.Join(methods, ", "))
		writeError(w, http.StatusMethodNotAllowed, errorBody{
			Code:    "METHOD_NOT_ALLOWED",
			Message: r.Method + " is not allowed on " + r.URL.Path,
		})
		return
	}

	writeError(w, http.StatusNotFound, errorBody{
		Code:    "NOT_FOUND",
		Message: "no route matches " + r.Method + " " + r.URL.Path,
	})
}

func handleCalculate(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != "" && !isJSONContentType(ct) {
		writeError(w, http.StatusUnsupportedMediaType, errorBody{
			Code:    "UNSUPPORTED_MEDIA_TYPE",
			Message: "Content-Type must be application/json",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)

	var req calculateRequest
	decoder := json.NewDecoder(r.Body)
	// Reject unrecognised fields so a client misspelling "operands" is told,
	// rather than silently receiving an operand-count error.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		status, body := decodeError(err)
		writeError(w, status, body)
		return
	}

	// Decode stops at the end of the first JSON value. Anything after it is a
	// client mistake, and reporting it is consistent with DisallowUnknownFields
	// above rather than silently absorbing half the request.
	if decoder.More() {
		writeError(w, http.StatusBadRequest, errorBody{
			Code:    string(calculator.CodeMalformedOperand),
			Message: "request body contains data after the JSON object",
		})
		return
	}

	operands := req.decimals()
	result, err := calculator.Calculate(req.Operation, operands)
	if err != nil {
		var calcErr *calculator.Error
		if errors.As(err, &calcErr) {
			writeError(w, statusFor(calcErr.Code), errorBody{
				Code:    string(calcErr.Code),
				Message: calcErr.Message,
				Field:   calcErr.Field,
			})
			return
		}
		// Calculate only ever returns *calculator.Error; this branch exists so
		// an unexpected error cannot leak as a 200.
		writeError(w, http.StatusInternalServerError, errorBody{
			Code:    codeInternalError,
			Message: "the calculation failed unexpectedly",
		})
		return
	}

	writeJSON(w, http.StatusOK, calculateResponse{
		Operation: req.Operation,
		Operands:  operands,
		Result:    result,
	})
}

// statusFor maps a rejection onto a status code.
//
// Faults in the request are 400. Division by zero, the square root of a
// negative and an oversized result are 422: the request was well formed and
// within bounds, and the service still cannot answer — which is precisely what
// "unprocessable content" means.
func statusFor(code calculator.ErrorCode) int {
	switch code {
	case calculator.CodeDivisionByZero,
		calculator.CodeNegativeSqrt,
		calculator.CodeResultTooLarge:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusBadRequest
	}
}

// decodeError turns a JSON decoding failure into a client-facing rejection.
func decodeError(err error) (int, errorBody) {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return http.StatusRequestEntityTooLarge, errorBody{
			Code:    codeRequestTooLarge,
			Message: "request body exceeds the maximum size",
		}
	}

	malformed := func(message, field string) (int, errorBody) {
		return http.StatusBadRequest, errorBody{
			Code:    string(calculator.CodeMalformedOperand),
			Message: message,
			Field:   field,
		}
	}

	// A truncated body yields io.ErrUnexpectedEOF, which is neither a
	// SyntaxError nor io.EOF, so without this it falls through to the operand
	// branch below and gets blamed on a field the request does not contain.
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) || errors.Is(err, io.ErrUnexpectedEOF) {
		return malformed("request body is not valid JSON", "")
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return malformed("field "+typeErr.Field+" has the wrong type", typeErr.Field)
	}

	if errors.Is(err, io.EOF) {
		return malformed("request body is empty", "")
	}

	// decimal reports unparseable operands as a plain error, and the decoder
	// reports unexpected fields the same way. Both are client mistakes.
	if strings.Contains(err.Error(), "unknown field") {
		return malformed(err.Error(), "")
	}
	return malformed("operand is not a valid number: "+err.Error(), "operands")
}

func isJSONContentType(contentType string) bool {
	mediaType, _, _ := strings.Cut(contentType, ";")
	return strings.EqualFold(strings.TrimSpace(mediaType), "application/json")
}

func writeError(w http.ResponseWriter, status int, body errorBody) {
	writeJSON(w, status, errorResponse{Error: body})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// The status and headers are already committed, so a failure here can only
	// be logged by the caller's middleware, not corrected.
	_ = json.NewEncoder(w).Encode(payload)
}
