package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// discardLogger keeps test output readable; the logging middleware is exercised
// either way.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func post(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	return do(t, http.MethodPost, "/api/v1/calculate", body, "application/json")
}

func do(t *testing.T, method, path, body, contentType string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	recorder := httptest.NewRecorder()
	NewRouter(discardLogger()).ServeHTTP(recorder, req)
	return recorder
}

func errorFrom(t *testing.T, recorder *httptest.ResponseRecorder) errorBody {
	t.Helper()
	var payload errorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response body is not a valid error envelope: %v\nbody: %s",
			err, recorder.Body.String())
	}
	return payload.Error
}

func TestCalculateSuccess(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"add", `{"operation":"add","operands":["0.1","0.2"]}`, "0.3"},
		{"subtract", `{"operation":"subtract","operands":["5","8"]}`, "-3"},
		{"multiply", `{"operation":"multiply","operands":["1.1","3"]}`, "3.3"},
		{"divide", `{"operation":"divide","operands":["10","4"]}`, "2.5"},
		{"power", `{"operation":"power","operands":["2","10"]}`, "1024"},
		{"percentage", `{"operation":"percentage","operands":["1","10"]}`, "10"},
		{"sqrt", `{"operation":"sqrt","operands":["4"]}`, "2"},

		// The API accepts bare JSON numbers as well as strings: decimal parses
		// both, and the canonical string form is documented in the README.
		{"numeric operands", `{"operation":"add","operands":[1,2]}`, "3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := post(t, tt.body)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body: %s", recorder.Code, recorder.Body)
			}

			var payload calculateResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decoding response: %v", err)
			}
			if got := payload.Result.String(); got != tt.want {
				t.Errorf("result = %s, want %s", got, tt.want)
			}
			if ct := recorder.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
		})
	}
}

// Results cross the wire as JSON strings, not JSON numbers. A number would be
// parsed into a float64 by the browser, discarding the precision the decimal
// arithmetic exists to preserve. See ADR 0001.
func TestResultIsAJSONString(t *testing.T) {
	recorder := post(t, `{"operation":"sqrt","operands":["2"]}`)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	result := string(raw["result"])
	if !strings.HasPrefix(result, `"`) {
		t.Errorf("result is %s, want a quoted JSON string", result)
	}
	if result != `"1.414213562373095"` {
		t.Errorf("result = %s, want \"1.414213562373095\"", result)
	}

	operands := string(raw["operands"])
	if operands != `["2"]` {
		t.Errorf("operands = %s, want [\"2\"]", operands)
	}
}

func TestCalculateRejections(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		// Faults in the request: 400.
		{"unknown operation", `{"operation":"modulo","operands":["1","2"]}`,
			http.StatusBadRequest, "UNKNOWN_OPERATION"},
		{"missing operation", `{"operands":["1","2"]}`,
			http.StatusBadRequest, "UNKNOWN_OPERATION"},
		{"too few operands", `{"operation":"add","operands":["1"]}`,
			http.StatusBadRequest, "INVALID_OPERAND_COUNT"},
		{"too many operands for sqrt", `{"operation":"sqrt","operands":["4","9"]}`,
			http.StatusBadRequest, "INVALID_OPERAND_COUNT"},
		{"missing operands", `{"operation":"add"}`,
			http.StatusBadRequest, "INVALID_OPERAND_COUNT"},
		{"unparseable operand", `{"operation":"add","operands":["abc","1"]}`,
			http.StatusBadRequest, "MALFORMED_OPERAND"},
		{"operand of the wrong type", `{"operation":"add","operands":[true,"1"]}`,
			http.StatusBadRequest, "MALFORMED_OPERAND"},
		{"malformed JSON", `{"operation":`,
			http.StatusBadRequest, "MALFORMED_OPERAND"},
		{"empty body", ``,
			http.StatusBadRequest, "MALFORMED_OPERAND"},
		{"unknown field", `{"operation":"add","operands":["1","2"],"precision":4}`,
			http.StatusBadRequest, "MALFORMED_OPERAND"},
		{"operand out of range", `{"operation":"add","operands":["1e1001","1"]}`,
			http.StatusBadRequest, "OPERAND_OUT_OF_RANGE"},
		{"fractional exponent", `{"operation":"power","operands":["2","0.5"]}`,
			http.StatusBadRequest, "OPERAND_OUT_OF_RANGE"},

		// Well-formed requests with no answer: 422.
		{"division by zero", `{"operation":"divide","operands":["1","0"]}`,
			http.StatusUnprocessableEntity, "DIVISION_BY_ZERO"},
		{"percentage of zero", `{"operation":"percentage","operands":["1","0"]}`,
			http.StatusUnprocessableEntity, "DIVISION_BY_ZERO"},
		{"square root of a negative", `{"operation":"sqrt","operands":["-4"]}`,
			http.StatusUnprocessableEntity, "NEGATIVE_SQRT"},
		{"result too large", `{"operation":"power","operands":["1e400","100"]}`,
			http.StatusUnprocessableEntity, "RESULT_TOO_LARGE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := post(t, tt.body)

			if recorder.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s",
					recorder.Code, tt.wantStatus, recorder.Body)
			}

			body := errorFrom(t, recorder)
			if body.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", body.Code, tt.wantCode)
			}
			if body.Message == "" {
				t.Error("message is empty")
			}
		})
	}
}

func TestRequestBodyTooLarge(t *testing.T) {
	// Well past the 1 KB limit, but still valid JSON.
	padding := strings.Repeat("9", 2048)
	recorder := post(t, `{"operation":"add","operands":["`+padding+`","1"]}`)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413; body: %s", recorder.Code, recorder.Body)
	}
	if code := errorFrom(t, recorder).Code; code != codeRequestTooLarge {
		t.Errorf("code = %q, want %q", code, codeRequestTooLarge)
	}
}

func TestContentTypeHandling(t *testing.T) {
	body := `{"operation":"add","operands":["1","2"]}`

	t.Run("rejects a non-JSON content type", func(t *testing.T) {
		recorder := do(t, http.MethodPost, "/api/v1/calculate", body, "text/plain")
		if recorder.Code != http.StatusUnsupportedMediaType {
			t.Errorf("status = %d, want 415", recorder.Code)
		}
	})

	t.Run("accepts a charset parameter", func(t *testing.T) {
		recorder := do(t, http.MethodPost, "/api/v1/calculate", body,
			"application/json; charset=utf-8")
		if recorder.Code != http.StatusOK {
			t.Errorf("status = %d, want 200; body: %s", recorder.Code, recorder.Body)
		}
	})

	t.Run("accepts a missing content type", func(t *testing.T) {
		recorder := do(t, http.MethodPost, "/api/v1/calculate", body, "")
		if recorder.Code != http.StatusOK {
			t.Errorf("status = %d, want 200; body: %s", recorder.Code, recorder.Body)
		}
	})
}

func TestRouting(t *testing.T) {
	t.Run("health", func(t *testing.T) {
		recorder := do(t, http.MethodGet, "/api/v1/health", "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", recorder.Code)
		}

		var payload map[string]string
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
		if payload["status"] != "ok" {
			t.Errorf("status field = %q, want ok", payload["status"])
		}
	})

	t.Run("wrong method is 405 with an Allow header", func(t *testing.T) {
		recorder := do(t, http.MethodGet, "/api/v1/calculate", "", "")
		if recorder.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", recorder.Code)
		}
		if allow := recorder.Header().Get("Allow"); allow != "POST" {
			t.Errorf("Allow = %q, want POST", allow)
		}
		if code := errorFrom(t, recorder).Code; code != "METHOD_NOT_ALLOWED" {
			t.Errorf("code = %q, want METHOD_NOT_ALLOWED", code)
		}
	})

	t.Run("unknown path is a JSON 404", func(t *testing.T) {
		recorder := do(t, http.MethodGet, "/api/v1/nonsense", "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", recorder.Code)
		}
		if code := errorFrom(t, recorder).Code; code != "NOT_FOUND" {
			t.Errorf("code = %q, want NOT_FOUND", code)
		}
	})
}

// The decimal library panics on division by zero, so the calculator guards
// every such path. This middleware is the backstop for the case a guard is
// missed, and is tested directly since no real route can reach it.
func TestPanicRecovery(t *testing.T) {
	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("decimal division by 0")
	})
	handler := recoverPanics(discardLogger())(panicking)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/calculate", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if code := errorFrom(t, recorder).Code; code != codeInternalError {
		t.Errorf("code = %q, want %q", code, codeInternalError)
	}
	if strings.Contains(recorder.Body.String(), "division by 0") {
		t.Error("panic detail leaked into the response body")
	}
}
