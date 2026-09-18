# Coverage report

Generated from the committed test suites. To reproduce:

```bash
cd backend  && go test ./... -coverprofile=cover.out && go tool cover -func=cover.out
cd frontend && npm run test:coverage
```

For an annotated, line-by-line HTML view:

```bash
cd backend  && go tool cover -html=cover.out       # opens in a browser
cd frontend && npm run test:coverage               # writes coverage/index.html
```

HTML output is gitignored: it is generated, it goes stale on the next commit,
and the two commands above regenerate it in seconds.

## Summary

| Layer | Package | Tests | Statements |
|---|---|---|---|
| Backend | `internal/calculator` | 79 | **94.6%** |
| Backend | `internal/api` | 37 | **95.7%** |
| Backend | `cmd/api` | — | 0.0% |
| Frontend | `src/` | 85 | **98.3%** |

**201 tests.** Frontend branch coverage is 96.3%, function coverage 97.5%.

## Backend, by function

```
internal/calculator/calculator.go   Operations             100.0%
internal/calculator/calculator.go   Arity                  100.0%
internal/calculator/calculator.go   Calculate              100.0%
internal/calculator/calculator.go   add                    100.0%
internal/calculator/calculator.go   subtract               100.0%
internal/calculator/calculator.go   multiply               100.0%
internal/calculator/calculator.go   divide                 100.0%
internal/calculator/calculator.go   percentage             100.0%
internal/calculator/calculator.go   sqrt                   100.0%
internal/calculator/calculator.go   power                   95.5%
internal/calculator/calculator.go   estimatedPowerDigits   100.0%
internal/calculator/sqrt.go         sqrtNewton              95.7%
internal/calculator/sqrt.go         initialGuess           100.0%
internal/calculator/bounds.go       checkOperand           100.0%
internal/calculator/bounds.go       magnitude               66.7%
internal/calculator/bounds.go       roundSignificant        83.3%
internal/calculator/bounds.go       divideSignificant       75.0%
internal/calculator/errors.go       Error                  100.0%
internal/calculator/errors.go       newError               100.0%
internal/calculator/errors.go       operandField           100.0%

internal/api/api.go                 NewRouter              100.0%
internal/api/api.go                 handleHealth           100.0%
internal/api/api.go                 handleNotFound         100.0%
internal/api/api.go                 handleCalculate         94.3%
internal/api/api.go                 statusFor              100.0%
internal/api/api.go                 decodeError             87.5%
internal/api/api.go                 isJSONContentType      100.0%
internal/api/api.go                 writeError             100.0%
internal/api/api.go                 writeJSON              100.0%
internal/api/middleware.go          logRequests            100.0%
internal/api/middleware.go          recoverPanics          100.0%
internal/api/middleware.go          WriteHeader            100.0%
internal/api/middleware.go          Write                   66.7%

cmd/api/main.go                     main                     0.0%
cmd/api/main.go                     run                      0.0%
cmd/api/main.go                     port                     0.0%
```

## Frontend, by file

```
File                      % Stmts   % Branch   % Funcs   % Lines
------------------------  --------  ---------  --------  --------
All files                   98.33      96.29      97.50     99.08
 src/App.tsx               100.00     100.00     100.00    100.00
 src/api/client.ts         100.00     100.00     100.00    100.00
 src/api/operations.ts     100.00     100.00     100.00    100.00
 src/api/schemas.ts        100.00      93.33     100.00    100.00
 src/components/
   CalculatorForm.tsx      100.00      95.00     100.00    100.00
   ResultPanel.tsx          88.88     100.00      83.33     93.33
 src/hooks/
   useCalculator.ts        100.00     100.00     100.00    100.00
```

## What is not covered, and why

**`cmd/api` — 0%.** Server wiring: constructing `http.Server` with timeouts,
installing a signal handler, calling `ListenAndServe`. Testing it would mean
asserting that the standard library works. It is instead verified by running it:
graceful shutdown was confirmed against the real container, where SIGTERM
produced `shutdown signal received, draining connections` followed by
`shutdown complete`.

**`ResultPanel` — 88.9%.** The uncovered line is the `catch` in the
clipboard handler, reached only when `navigator.clipboard.writeText` rejects —
which happens on an insecure origin or when the user denies permission. The
failure path does nothing but leave the button label unchanged.

**`divideSignificant`, `roundSignificant`, `magnitude` — 66-83%.** Defensive
clamps: the branch that caps working precision at `maxDivisionPlaces`, and the
zero-value guards. They are reachable only from operand combinations the bounds
check already rejects, so they are unreachable in practice and kept because the
functions should be correct in isolation.

**`power` — 95.5%, `sqrtNewton` — 95.7%.** In both cases the uncovered branch is
an error return from the decimal library that the preceding guards make
unreachable — `power` checks for a zero base with a negative exponent before
calling `PowInt32`, and `sqrtNewton` bounds its iteration count against a
convergence failure that cannot occur for in-range operands.

**`handleCalculate` — 94.3%, `decodeError` — 87.5%.** The uncovered paths are
the `INTERNAL_ERROR` fallback for an error type `Calculate` never returns, and
one JSON decoder error variant that the other cases already cover. Both exist so
an unexpected failure cannot escape as a 200.

## Notes on what the tests actually assert

Coverage counts lines, not value, so a few of the more load-bearing tests:

- **Exactness.** `0.1 + 0.2 = 0.3` and `1.1 × 3 = 3.3` — the cases `float64`
  gets wrong — are asserted on the result string, not on an approximate compare.
- **Square root accuracy.** `sqrt(1e400)` must equal exactly `1e200`, which is
  the case where the decimal library's own implementation is wrong from the 27th
  digit. Squaring the root reproduces the operand for seven other inputs.
- **Resource bounds.** The inputs that motivated the limits — `1e1000000` as an
  operand, `power(10, 1000000)` — are asserted to be rejected, and every
  operation is asserted to succeed quickly at the exact edge of the bounds.
- **Validation order.** An unknown operation beats an operand-count error, which
  beats an out-of-range operand, which beats division by zero. One bad request
  always produces one predictable error.
- **Wire format.** The result is asserted to be a JSON *string*, not a number,
  because a number would be flattened to a float64 by the browser.
