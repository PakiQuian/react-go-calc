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
| Backend | `internal/calculator` | 104 | **94.1%** |
| Backend | `internal/api` | 54 | **97.6%** |
| Backend | `cmd/api` | — | 0.0% |
| Frontend | `src/` | 97 | **98.4%** |

**255 tests.** Frontend branch coverage is 94.8%, function coverage 97.5%.

## Backend, by function

```
cmd/api/main.go:32:                                  main                   0.0%
cmd/api/main.go:41:                                  run                    0.0%
cmd/api/main.go:85:                                  port                   0.0%
internal/api/api.go:43:                              UnmarshalJSON          100.0%
internal/api/api.go:57:                              decimals               100.0%
internal/api/api.go:82:                              NewRouter              100.0%
internal/api/api.go:97:                              handleHealth           100.0%
internal/api/api.go:111:                             handleNotFound         100.0%
internal/api/api.go:127:                             handleCalculate        94.9%
internal/api/api.go:195:                             statusFor              100.0%
internal/api/api.go:207:                             decodeError            100.0%
internal/api/api.go:249:                             isJSONContentType      100.0%
internal/api/api.go:254:                             writeError             100.0%
internal/api/api.go:258:                             writeJSON              100.0%
internal/api/middleware.go:18:                       WriteHeader            100.0%
internal/api/middleware.go:24:                       Write                  66.7%
internal/api/middleware.go:31:                       logRequests            100.0%
internal/api/middleware.go:54:                       recoverPanics          100.0%
internal/calculator/bounds.go:50:                    magnitude              66.7%
internal/calculator/bounds.go:58:                    checkOperand           100.0%
internal/calculator/bounds.go:99:                    roundSignificant       85.7%
internal/calculator/bounds.go:117:                   divideSignificant      87.5%
internal/calculator/calculator.go:32:                Operations             100.0%
internal/calculator/calculator.go:42:                Arity                  100.0%
internal/calculator/calculator.go:52:                Calculate              100.0%
internal/calculator/calculator.go:76:                add                    100.0%
internal/calculator/calculator.go:78:                subtract               100.0%
internal/calculator/calculator.go:80:                multiply               100.0%
internal/calculator/calculator.go:82:                divide                 100.0%
internal/calculator/calculator.go:94:                percentage             100.0%
internal/calculator/calculator.go:104:               sqrt                   100.0%
internal/calculator/calculator.go:115:               power                  88.9%
internal/calculator/calculator.go:179:               estimatedPowerDigits   100.0%
internal/calculator/calculator.go:184:               abs                    100.0%
internal/calculator/errors.go:31:                    Error                  100.0%
internal/calculator/errors.go:33:                    newError               100.0%
internal/calculator/errors.go:37:                    operandField           100.0%
internal/calculator/sqrt.go:23:                      sqrtNewton             95.7%
internal/calculator/sqrt.go:59:                      initialGuess           100.0%
total:                                               (statements)           82.5%
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

**`divideSignificant`, `roundSignificant`, `magnitude`.** An earlier version of this
file claimed these branches were "unreachable in practice". A review disproved
two of them: `divideSignificant`'s zero-dividend guard is reached by
`divide(0, 5)` — the guard is on the *divisor*, so a zero dividend is an
ordinary request — and `roundSignificant`'s zero guard is reached through
`power` with a negative exponent. Both now have tests. What remains uncovered
is `magnitude`'s zero guard, whose every caller pre-checks `IsZero`, and
`divideSignificant`'s `maxDivisionPlaces` clamp, which needs a magnitude spread
beyond ±2023 while the operand bounds cap it at ±2000.

**`sqrtNewton`.** The uncovered branch is the `work > maxDivisionPlaces` clamp.
`work` is `26 - magnitude/2`, which stays within ±526 for any in-range operand,
so the clamp cannot fire. An earlier version of this file described it as "an
error return from the decimal library"; `sqrt.go` contains no error return and
calls no fallible function.

**`handleCalculate`, `decodeError`.** The remaining uncovered path is the
`INTERNAL_ERROR` fallback for an error type `Calculate` never returns, which
exists so an unexpected failure cannot escape as a 200. An earlier version of
this file claimed the `decodeError` branches were covered by the other cases; in
fact two tests were passing through the catch-all and asserting only the code,
which is identical across all four branches. Those tests now pin the message
and field, and the inputs that reach each branch are covered.

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
