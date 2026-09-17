---
status: accepted
---

# A single `/calculate` endpoint rather than one route per operation

All arithmetic goes through `POST /api/v1/calculate`, whose body names an
operation and supplies an array of operands, rather than exposing `/add`,
`/subtract`, `/multiply` and so on. Validation runs in two ordered stages: the
operation name must belong to a closed enum, and only then is the operand count
checked against *that operation's* arity.

## Considered options

**One endpoint per operation.** Rejected. Seven routes, seven handlers and seven
near-identical validation blocks that differ only in which function they call at
the end. Adding an operation would mean touching routing, handlers and tests
instead of adding one row to a table.

**An expression endpoint** taking `"1 + 2 * 3"`. Rejected. Operator precedence
and parsing are a substantial piece of work that the assignment never asked for,
and they would dominate the codebase without demonstrating anything about API
design or testing.

## Consequences

**Operations are data, not code.** Dispatch is a `map[string]operation` where
each entry carries its arity, its bounds and its function. Arity checking is
therefore a table lookup rather than a per-handler `if`, which is what makes the
two-stage validation order meaningful: an unknown operation never reaches arity
checking, so there is no ambiguity about which error a bad request produces.

**Unary and binary operations share one shape.** `sqrt` takes one operand and
the rest take two, expressed as a number in the table rather than as a different
request type or a nullable second field.

**The frontend consumes the same table.** The form hides its second input for
`sqrt`, driven by the same arity rule, so the UI cannot construct a request the
API will reject.

**One URL for every operation** is slightly less RESTful in the strictest
reading, since the operation is in the body rather than the path. The trade is
deliberate: `calculate` is the resource, and the alternative is seven copies of
one handler.
