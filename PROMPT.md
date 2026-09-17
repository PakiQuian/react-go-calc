# Prompts

A record of the prompts used to build this project, lightly edited for readability.
The work was done with Claude Code (Opus 5), used as a design partner first and a
code generator second — most of the value came from being interrogated about
decisions before any code existed.

---

## 1. Framing the task

> I have a take-home assignment to complete. Read TASK.md and don't do anything
> yet — just tell me what it asks for.

Deliberately read-only. Starting with "build me a calculator" would have produced
a plausible app built on unexamined defaults.

---

## 2. Choosing the stack

> Let's start thinking about how we'll do this. First, let's look at the stack
> we're going to use, including libraries — for example Zod for model validation.

Run through a structured interview ("grilling") rather than a single answer: the
assistant maps the decisions as a tree, asks every question whose prerequisites
are already settled, gives a recommendation for each, and waits. Each round of
answers unblocks the next set of questions.

This surfaced decisions that would otherwise have been made silently — API shape,
numeric representation, whether to use an HTTP framework at all.

---

## 3. Decisions taken in that interview

Answers given, with the reasoning that came out of them:

- **Repo layout** — `backend/` and `frontend/` side by side. Go and Node don't
  share a dependency graph, so a workspace tool would add ceremony for nothing.

- **API shape** — a single `POST /api/v1/calculate` taking an operation name and
  an array of operands, rather than one endpoint per operation. One validation
  path, one handler table, one test table. Validation runs in two stages:
  the operation must belong to a closed enum (otherwise 400), and only then is
  operand count checked against that specific operation's arity — `sqrt` takes
  one, `add` takes two.

- **HTTP layer** — Go's standard library, no framework. Go 1.22 added method and
  path matching to `net/http`, which covers everything a router would give us at
  two routes.

- **Frontend** — Vite + React + TypeScript. No routing or SSR requirement, so
  Next.js would add a server to containerize for no benefit.

- **UI shape** — a form (two operands + operation select) rather than a classic
  keypad. A keypad implies local expression state, which fights a REST API that
  does one operation per round trip; the form puts the API and its validation in
  the foreground.

- **Go tests** — standard library `testing`, table-driven. The idiom a Go
  reviewer expects.

---

## 4. Understanding the numeric model

> Explain the difference between float64 and decimal so I understand it. Which
> one is the Java BigDecimal equivalent?

Asked because the recommendation (decimal for the four basic operations,
float64 for square root and exponentiation) needed to be understood before it
could be defended in the README.

Short version: `float64` is IEEE 754 binary — it stores numbers in base 2, where
`0.1` has no exact representation, so `0.1 + 0.2` returns `0.30000000000000004`.
Decimal stores an integer coefficient with a base-10 exponent, making the same
sum exact. `shopspring/decimal` is the Go equivalent of `java.math.BigDecimal`;
`float64` is `double`. Square root and fractional powers stay in float64 because
`sqrt(2)` is irrational and has no exact decimal form at any precision.

---

## 5. Questioning the framework choice

> You're recommending not using a toolkit? Why — too much overhead for a single
> endpoint?

Worth pushing back on rather than accepting. The answer held up: at two routes
with no path parameters, a router's features (route groups, radix-tree matching,
struct-tag binding) earn nothing, and the validation rules here aren't
expressible in struct tags anyway.

---

## 6. Pushing back on the numeric plan

> Decimal everywhere makes sense — in Java I'd have used BigDecimal, since
> double and float are for incidental numbers, but here the numbers are the
> core of the project, like when handling money.

> So for square root and division we use float64? What does float64 give us in
> those cases — more precision, since they're irrational?

The second question exposed an error in the recommendation. The assumption had
been that irrational results require a binary float. Checking the library
instead of assuming showed that `decimal.PowWithPrecision` accepts non-integer
exponents, so `sqrt(x)` is `x^0.5` at any precision — and that `float64`, capped
at 15-17 significant digits, is in fact *less* precise. The hybrid design was
dropped for decimal throughout, which was both simpler and more accurate.

Recorded as [ADR 0001](docs/adr/0001-decimal-arithmetic.md).

---

## 7. Where validation belongs

> I thought Zod was only for the frontend — doesn't Go have its own validation
> mechanism?

It does, and both layers exist for different reasons. Backend validation is a
security boundary: the server cannot trust any client, and anyone can call the
endpoint directly. Frontend validation is a user-experience affordance, giving
instant feedback without a round trip, and is trivially bypassed. Remove the
frontend layer and the app still behaves correctly; remove the backend layer and
the service is broken.

Zod's second job has no Go counterpart: parsing the *response*, so a contract
mismatch surfaces as a clear error rather than as `undefined` inside a component.

---

## 8. Bounding an unbounded type

> Isn't there an integer max in Go? If the number is bigger than that we won't
> be able to work with it.

True of `int64` and `float64`, but not of `decimal`, which is an arbitrary-
precision coefficient with an exponent — bounded by memory, not by a type. That
is the feature, and it is also the exposure: `{"operands":["1e1000000","1"]}` is
a 45-byte request that takes 35 ms of CPU and produces a million-digit result,
because addition has to align exponents.

Limits were therefore added deliberately — at most 100 significant digits and a
magnitude within 10±¹⁰⁰⁰ — rather than inherited from a type.

---

## 9. Checking whether the limits were actually safe

The bounds above were then benchmarked at their own worst case, rather than
assumed to be sufficient. Two problems surfaced:

- `sqrt` of the largest legal operand took **48 seconds**.
- `power` with an exponent of 1000 produced a **1.1 MB** result from tiny
  operands, showing that bounding the input does not bound the output.

Investigating the first revealed that the library's `PowWithPrecision` is not
only slow at large magnitudes but inaccurate: `sqrt(1e400)`, whose exact answer
is `1e200`, came back wrong from the 27th digit. Newton-Raphson, about twelve
lines, is exact and converges in four iterations on the same input.

Recorded as [ADR 0004](docs/adr/0004-newton-raphson-sqrt.md). This sequence —
propose a bound, measure it at its limit, discover the real problem was
underneath it — is the part of the process most worth reading.

---

## 10. Working method

Two habits did most of the work here, and both are visible in the git history:

**Nothing was accepted without measurement.** Every claim about performance or
precision in the ADRs comes from a scratch program run during the session, not
from recollection. Three recommendations were overturned this way, including one
where Go's compile-time constant folding made a floating-point bug appear not to
exist.

**Decisions were recorded before the code they govern.** `CONTEXT.md` and the
four ADRs were committed before the first line of Go, so the reasoning is not a
reconstruction written afterwards.
