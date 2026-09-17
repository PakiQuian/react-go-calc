---
status: accepted
---

# Square root by Newton-Raphson, not by `PowWithPrecision`

`sqrt` is implemented with Newton-Raphson iteration in `internal/calculator`
rather than by calling `decimal.PowWithPrecision(0.5, n)`. The library function
is both far slower and measurably less accurate at large magnitudes, and this is
the kind of hand-written numeric code a later reader is likely to delete as
unnecessary. It is not.

## Why the obvious call was rejected

`PowWithPrecision` computes `x^y` through logarithm and exponential series.
Those converge poorly as the magnitude of `x` grows. Measured on the operand
bounds this service accepts:

| Operand | `PowWithPrecision` | Newton-Raphson |
|---|---|---|
| 100 digits, exponent 0 | 213 ms | below timer resolution |
| 1 digit, exponent 400 | 348 ms | below timer resolution |
| 100 digits, exponent 1000 | 33 s | 4 iterations |

Thirty-three seconds of CPU for a single legal request is a denial-of-service
vector reachable with a forty-byte request body.

Accuracy is the more serious problem. For `sqrt(1e400)`, where the exact answer
is `1e200`:

```
Newton-Raphson : 1000000000000000000000000000...000      (exact)
PowWithPrecision: 99999999999999999999999999812715847...  (wrong from digit 27)
```

Silently returning wrong digits defeats the entire purpose of ADR 0001, which
exists so that results are exact.

## Consequences

**The operand bounds can stay generous.** Had the library implementation been
kept, `sqrt` would have needed a magnitude limit roughly thirty times tighter
than every other operation — an arbitrary-looking restriction that existed only
to hide a slow function. Newton-Raphson is fast across the entire accepted range,
so one uniform bound covers all seven operations.

**`power` accepts integer exponents only.** Non-integer exponents would route
through the same `PowWithPrecision` path, with the same two problems, so they are
rejected rather than approximated. `sqrt` covers the common fractional case
exactly, and implementing general fractional powers correctly is far outside the
scope of this assignment.

**Twelve lines of numeric code now need tests of their own,** which is a fair
price and arguably a benefit: convergence, the negative-input case, zero, perfect
squares and irrational results are all directly testable, and
`internal/calculator` gains logic worth testing rather than a call-through.
