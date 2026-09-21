# Context

Glossary for the calculator service. Domain language only — no implementation
details.

## Operation

A named arithmetic function the service can perform. The set of operations is
**closed**: a request naming anything outside it is rejected before any other
check runs.

Each Operation has a fixed **arity** — the number of Operands it requires:

| Operation | Arity |
|---|---|
| `add`, `subtract`, `multiply`, `divide` | 2 |
| `power`, `percentage` | 2 |
| `sqrt` | 1 |

`power` accepts **integer exponents only**. A non-integer exponent is rejected
rather than approximated; `sqrt` covers the common fractional case exactly.

## Operand

An input number to an Operation. Operands are ordered: for non-commutative
Operations (`subtract`, `divide`, `power`, `percentage`) position carries
meaning.

Operands are exact decimal quantities, not binary floating-point approximations.
They are **bounded** — at most 100 significant digits and a magnitude between
10⁻¹⁰⁰⁰ and 10¹⁰⁰⁰. The underlying representation has no inherent ceiling, so
the boundary is a deliberate part of the contract rather than a property of a
number type.

How an Operand is *written* is bounded too, separately from what it is worth.
Zero has no magnitude, so a magnitude limit alone says nothing about `0e-2147483647`
— a value of zero, written with an extreme exponent. Such an Operand is rejected:
the cost of handling a number follows the notation it arrives in, not only the
quantity it denotes.

## Calculation

One Operation applied to its Operands. A Calculation either produces a **Result**
or fails with a **Rejection**. It has no memory of any previous Calculation —
there is no running total and no session.

## Rejection

A Calculation that could not be performed, distinguished by cause:

- **Unknown operation** — the named Operation is not in the closed set.
- **Wrong arity** — the Operand count does not match the Operation's arity.
- **Malformed operand** — a value that is not a number.
- **Operand out of range** — a well-formed number outside the bounds above.
- **Undefined result** — the Operation is defined but has no answer for these
  Operands: division by zero, square root of a negative number.
- **Unrepresentable result** — the answer exists mathematically but is too large
  to return. `power` can turn small Operands into a result of millions of
  digits, so the size of the Result is constrained independently of the size of
  the Operands.

The first four are faults in the *request*. The last two are not: the request was
well-formed and within bounds, and the service still declines to answer — in one
case because mathematics has no answer, in the other because the answer will not
fit.

## Percentage

Ambiguous in everyday use, so fixed here: `percentage(a, b)` means **what
percentage `a` is of `b`**, with `b` as the total — `a / b * 100`. So
`percentage(1, 10)` is `10`.

It does not mean "a percent of b". Both readings are common in calculators and
they disagree on every input; choosing one is a domain decision, not an
implementation detail.

Because the definition divides by the total, `percentage(a, 0)` is an **undefined
result**, exactly as `divide(a, 0)` is.
