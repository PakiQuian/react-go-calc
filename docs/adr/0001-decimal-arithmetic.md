---
status: accepted
---

# Arbitrary-precision decimal for all arithmetic

Arithmetic is not incidental to this service — it is the entire product, so the
numeric type is the most consequential decision in the codebase. All operations
use `shopspring/decimal` rather than `float64`. Operands are decoded straight
into `decimal.Decimal`, whose `UnmarshalJSON` parses the raw token text, so no
value passes through a binary float at any point — including bare JSON numbers,
which the API also accepts.

## Considered options

**`float64` throughout.** Rejected. IEEE 754 is base 2, and decimal fractions
such as `0.1` have no exact base-2 representation, so errors are introduced at
parse time and compound through the calculation:

```
0.1 + 0.2  =  0.30000000000000004
1.1 * 3    =  3.3000000000000003
```

For a calculator these are visible in the response body. For a payments company
they are the canonical example of how not to represent quantities.

**Decimal for the four basic operations, `float64` for `sqrt` and `power`.**
Rejected after checking the library rather than assuming. This was the initial
plan, on the belief that irrational results require a binary float. They do not:
`decimal.PowWithPrecision` accepts non-integer exponents, so `sqrt(x)` is
`x^0.5` computed to any requested precision. `float64` is in fact *less* precise,
capped at roughly 15-17 significant digits:

```
sqrt(2)  float64   :  1.4142135623730951
sqrt(2)  decimal   :  1.41421356237309504880168872420969807856967187537695
```

Rejecting the hybrid removed a type boundary, a conversion step and a paragraph
of explanation, while increasing precision. It is strictly simpler and strictly
better.

## Consequences

**Undefined results surface as errors, not as poison values.** `float64` returns
`NaN` or `+Inf`, which propagate silently and then fail at the edge, since
`encoding/json` refuses to marshal either one. Decimal has the opposite failure
mode: it *panics* on division by zero rather than returning anything. So each
such case is guarded explicitly before the library is called, and the guard
produces the "undefined result" rejection, answered with HTTP 422. Panic
recovery middleware is the backstop, and its firing means a guard is missing.

**Division requires an explicit precision.** `1/3` does not terminate in base 10
any more than `0.1` terminates in base 2, so division and non-integer powers take
a precision argument. This is an advantage rather than a cost: the loss of
precision is declared in one place instead of occurring silently.

**Precision is a floor, not a digit count.** `PowWithPrecision(0.5, 16)` returns
26 digits, not 16. Results are therefore rounded explicitly before serialization
rather than relying on that parameter to shape the output.

**Performance is roughly two orders of magnitude worse than `float64`,** and
every operation allocates. Irrelevant here: one arithmetic operation per HTTP
request, where the round trip dominates by several orders of magnitude.
