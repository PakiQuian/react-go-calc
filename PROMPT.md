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
