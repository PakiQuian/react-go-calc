# Context

Glossary for the calculator service. Domain language only — no implementation
details.

## Operation

A named arithmetic function the service can perform. The set of operations is
**closed**: a request naming anything outside it is rejected before any other
check runs.

Each Operation has a fixed **arity** — the number of Operands it requires.
`add`, `subtract`, `multiply`, `divide` and `percentage` are binary; `sqrt` is
unary; `power` is binary.

## Operand

An input number to an Operation. Operands are ordered: for non-commutative
Operations (`subtract`, `divide`, `power`) position carries meaning.

## Calculation

One Operation applied to its Operands. A Calculation either produces a **Result**
or fails with a **Rejection**. It has no memory of any previous Calculation —
there is no running total and no session.

## Rejection

A Calculation that could not be performed, distinguished by cause:

- **Unknown operation** — the named Operation is not in the closed set.
- **Wrong arity** — the Operand count does not match the Operation's arity.
- **Malformed operand** — a value that is not a number.
- **Undefined result** — the Operation is defined but has no answer for these
  Operands: division by zero, square root of a negative number.

The first three are faults in the *request*. The last is a fault in the
*mathematics* — the request was well-formed and the service still cannot answer.

## Percentage

Ambiguous in everyday use, so fixed here: `percentage(a, b)` means **what
percentage `a` is of `b`**, with `b` as the total — `a / b * 100`. So
`percentage(1, 10)` is `10`.

It does not mean "a percent of b". Both readings are common in calculators and
they disagree on every input; choosing one is a domain decision, not an
implementation detail.

Because the definition divides by the total, `percentage(a, 0)` is an **undefined
result**, exactly as `divide(a, 0)` is.
