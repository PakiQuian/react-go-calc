# Calculator

A full-stack calculator: a React frontend over a Go REST service that performs
all arithmetic with exact decimals.

```
$ curl -s -X POST localhost:3000/api/v1/calculate \
    -H 'Content-Type: application/json' \
    -d '{"operation":"add","operands":["0.1","0.2"]}'

{"operation":"add","operands":["0.1","0.2"],"result":"0.3"}
```

That `0.3` is the reason most of the design decisions here went the way they
did. In IEEE 754 binary floating point — Go's `float64`, JavaScript's `number` —
the same sum is `0.30000000000000004`.

---

## Quick start

```bash
docker compose up --build
```

Then open <http://localhost:3000>.

Two containers start: nginx serving the built frontend and reverse-proxying
`/api`, and the Go service, which publishes no port to the host and is reachable
only through the proxy.

```bash
docker compose down     # stop
```

## Running without Docker

**Backend** — needs Go 1.27 or newer.

```bash
cd backend
go run ./cmd/api          # listens on :8080, or $PORT
```

**Frontend** — needs Node 24 or newer.

```bash
cd frontend
npm install
npm run dev               # http://localhost:5173
```

Vite proxies `/api` to `localhost:8080`, which is the same arrangement nginx
provides in production. The frontend therefore calls a relative path in every
environment: there is no API base URL to configure, no `.env` file, and no CORS
handling anywhere in the codebase.

---

## API

Base path `/api/v1`. Two endpoints.

### `POST /api/v1/calculate`

```json
{
  "operation": "divide",
  "operands": ["10", "4"]
}
```

`operation` is one of `add`, `subtract`, `multiply`, `divide`, `power`,
`percentage`, `sqrt`. `operands` is an array whose length must match that
operation's arity — one for `sqrt`, two for the rest.

Operands are sent as **strings**, and results come back as strings. Bare JSON
numbers are accepted too, but strings are canonical, for the reason in
[Design decisions](#numbers-cross-the-wire-as-strings) below.

```json
{
  "operation": "divide",
  "operands": ["10", "4"],
  "result": "2.5"
}
```

### `GET /api/v1/health`

```json
{ "status": "ok" }
```

### Operations

| Operation | Operands | Meaning | Example |
|---|---|---|---|
| `add` | 2 | `a + b` | `["0.1","0.2"]` → `0.3` |
| `subtract` | 2 | `a − b` | `["5","8"]` → `-3` |
| `multiply` | 2 | `a × b` | `["1.1","3"]` → `3.3` |
| `divide` | 2 | `a ÷ b` | `["10","4"]` → `2.5` |
| `power` | 2 | `a^b`, integer `b` only | `["2","10"]` → `1024` |
| `percentage` | 2 | what percentage `a` is of `b` | `["1","10"]` → `10` |
| `sqrt` | 1 | `√a` | `["2"]` → `1.414213562373095` |

`percentage` is genuinely ambiguous in everyday use — it can mean "what
percentage `a` is of `b`" or "`a` percent of `b`", and the two disagree on every
input. This service means the first: `a / b * 100`, with `b` as the total.

### Errors

Every failure returns the same envelope:

```json
{
  "error": {
    "code": "DIVISION_BY_ZERO",
    "message": "division by zero is undefined",
    "field": "operands[1]"
  }
}
```

| Code | Status | Cause |
|---|---|---|
| `UNKNOWN_OPERATION` | 400 | operation is not one of the seven |
| `INVALID_OPERAND_COUNT` | 400 | operand count does not match the arity |
| `MALFORMED_OPERAND` | 400 | unparseable number, bad JSON, unknown field |
| `OPERAND_OUT_OF_RANGE` | 400 | outside the accepted bounds (see below) |
| `REQUEST_TOO_LARGE` | 413 | request body over 1 KB |
| `UNSUPPORTED_MEDIA_TYPE` | 415 | `Content-Type` is not `application/json` |
| `DIVISION_BY_ZERO` | 422 | `divide(a, 0)`, `percentage(a, 0)`, `0^-n` |
| `NEGATIVE_SQRT` | 422 | square root of a negative number |
| `RESULT_TOO_LARGE` | 422 | the answer exists but is too large to return |

**400 versus 422** is a deliberate split. The 400s are faults in the *request*:
something about it is wrong. The 422s are not — the request was well formed and
within bounds, and the service still cannot answer, either because mathematics
has no answer or because the answer will not fit. That is precisely what
"unprocessable content" means.

### Examples

```bash
# Exact decimal arithmetic
curl -X POST localhost:3000/api/v1/calculate \
  -H 'Content-Type: application/json' \
  -d '{"operation":"multiply","operands":["1.1","3"]}'
# {"operation":"multiply","operands":["1.1","3"],"result":"3.3"}

# Square root, exact to 16 significant digits
curl -X POST localhost:3000/api/v1/calculate \
  -H 'Content-Type: application/json' \
  -d '{"operation":"sqrt","operands":["2"]}'
# {"operation":"sqrt","operands":["2"],"result":"1.414213562373095"}

# Division by zero: 422, not 400
curl -i -X POST localhost:3000/api/v1/calculate \
  -H 'Content-Type: application/json' \
  -d '{"operation":"divide","operands":["1","0"]}'
# HTTP/1.1 422 Unprocessable Content
# {"error":{"code":"DIVISION_BY_ZERO","message":"division by zero is undefined","field":"operands[1]"}}

# Wrong operand count for a unary operation
curl -X POST localhost:3000/api/v1/calculate \
  -H 'Content-Type: application/json' \
  -d '{"operation":"sqrt","operands":["4","9"]}'
# {"error":{"code":"INVALID_OPERAND_COUNT","message":"operation \"sqrt\" requires 1 operand(s), got 2","field":"operands"}}

# Rejected before it can consume resources
curl -X POST localhost:3000/api/v1/calculate \
  -H 'Content-Type: application/json' \
  -d '{"operation":"add","operands":["1e1000000","1"]}'
# {"error":{"code":"OPERAND_OUT_OF_RANGE","message":"operand magnitude 1e1000000 is outside the accepted range 1e-1000 to 1e1000","field":"operands[0]"}}
```

---

## Tests

```bash
cd backend  && go test ./...                              # 116 tests
cd frontend && npm test                                   # 85 tests

cd backend  && go test ./... -coverprofile=cover.out && go tool cover -func=cover.out
cd frontend && npm run test:coverage
```

201 tests in total.

| Package | Tests | Statements |
|---|---|---|
| `backend/internal/calculator` | 79 | 94.6% |
| `backend/internal/api` | 37 | 95.7% |
| `frontend/src` | 85 | 98.3% |

**[`COVERAGE.md`](COVERAGE.md)** has the full report: per-function figures for
the backend, per-file for the frontend, and an account of every uncovered branch
and why it is uncovered.

`backend/cmd/api` is not covered: it is server wiring, and testing it would mean
asserting that the standard library works. It is verified by running it instead
— graceful shutdown was confirmed against the real container.

---

## Design decisions

The full reasoning for the four decisions that were hard to reverse is in
[`docs/adr/`](docs/adr/). The domain vocabulary — what an operand is, what the
distinct kinds of rejection are — is in [`CONTEXT.md`](CONTEXT.md). The prompts
that produced all of it, including the three recommendations that turned out to
be wrong, are in [`PROMPT.md`](PROMPT.md).

### Exact decimals everywhere

`float64` stores numbers in base 2, where `0.1` has no exact representation, so
errors appear at parse time and compound. For a calculator they are visible in
the response body. `shopspring/decimal` — the Go equivalent of Java's
`BigDecimal` — stores an integer coefficient with a base-10 exponent, making the
same arithmetic exact.

The initial plan was a hybrid: decimals for the four basic operations, `float64`
for `sqrt` and `power`, on the assumption that irrational results need a binary
float. They don't, and `float64` is *less* precise at 15-17 significant digits.
Dropping the hybrid removed a type boundary and increased accuracy.
([ADR 0001](docs/adr/0001-decimal-arithmetic.md))

### One endpoint, not seven

`POST /calculate` names the operation in the body rather than exposing `/add`,
`/subtract` and so on. Seven routes would mean seven near-identical handlers
differing only in which function they call. Here, operations are a table where
arity is data, which is what makes the two-stage validation meaningful: the
operation must be in the closed set before its operand count is checked, so one
bad request always produces one predictable error.
([ADR 0002](docs/adr/0002-single-calculate-endpoint.md))

### No HTTP framework

Go 1.22 added method and path matching to the standard `ServeMux`. What a router
adds beyond that — route groups, struct-tag binding, radix-tree matching — pays
off at dozens of routes. This service has two. The one validation rule that
matters, "the required operand count depends on which operation was named",
cannot be expressed in struct tags anyway.
([ADR 0003](docs/adr/0003-standard-library-http.md))

### Square root is hand-written

`decimal.PowWithPrecision(0.5, n)` is the obvious way to compute a square root,
and it is both slow and wrong at large magnitudes. Measured against the operand
bounds this service accepts:

| Operand | Library | Newton-Raphson |
|---|---|---|
| 100 digits, exponent 0 | 213 ms | below timer resolution |
| 100 digits, exponent 1000 | 33 s | 4 iterations |

Worse than slow, it is inaccurate. For `sqrt(1e400)`, where the exact answer is
`1e200`:

```
Newton-Raphson   : 1000000000000000000000000000...000      (exact)
PowWithPrecision : 99999999999999999999999999812715847...  (wrong from digit 27)
```

Twelve lines of Newton-Raphson are exact and fast across the whole range.
Because it is fast, the operand bounds could stay generous instead of being
tightened thirty-fold to hide a slow function.
([ADR 0004](docs/adr/0004-newton-raphson-sqrt.md))

### Numbers cross the wire as strings

If a 20-digit result were returned as a JSON *number*, the browser's
`JSON.parse` would convert it to a float64 and discard every digit past the 17th
— one line before it reached the user. Strings preserve the value end to end,
and it is also the decimal library's default marshalling, so it costs no
configuration.

### Operands are bounded

Arbitrary precision has no natural ceiling; `decimal` is bounded by memory, not
by a type. `{"operands":["1e1000000","1"]}` is a 45-byte request that takes 35 ms
of CPU and produces a million-digit result, because addition has to align
exponents. So operands are capped at **100 significant digits** and a magnitude
within **1e±1000**, with a 1 KB body limit in front.

Bounding the input is not sufficient, though: `power(10, 1000000)` has tiny
operands and a million-digit *result*. `power` therefore estimates its result
size before computing anything, and `RESULT_TOO_LARGE` is a distinct kind of
rejection because of it.

### Validation lives in both layers, for different reasons

The backend validates because it is a security boundary — anyone can call the
API directly, and the React app is irrelevant to them. The frontend validates
because it is a user-experience affordance: instant feedback with no round trip.
Remove the frontend layer and the application still behaves correctly; remove
the backend layer and the service is broken.

Zod also parses *responses*, which has no backend equivalent. That turns a
contract mismatch into one clear error instead of `undefined` surfacing inside a
component.

### A form, not a keypad

A keypad implies local expression state and immediate feedback, which fights a
REST API that performs one operation per round trip. The form makes the API and
its validation the visible subject, and it is directly testable. The operand
fields are driven by the same arity the backend enforces, so choosing `sqrt`
hides the second field: the UI cannot construct a request the API would reject.

---

## Assumptions

- **Each calculation is independent.** No running total, no memory, no session.
  The service is stateless and needs no database.
- **`percentage(a, b)` means "what percentage `a` is of `b`".** Both readings are
  common; this one is fixed in `CONTEXT.md` and in the field labels.
- **`0^0` is `1`,** following IEEE 754 `pow`, Go's `math.Pow` and every pocket
  calculator. The decimal library treats it as undefined.
- **Inexact results carry 16 significant digits.** Division, percentage and
  square root are rounded; addition, subtraction and multiplication are exact
  and never rounded.
- **`power` takes whole-number exponents only.** Fractional exponents would route
  through the same inaccurate code path that `sqrt` avoids, so they are rejected
  rather than approximated.
- **No authentication.** Nothing in the assignment implies users or access
  control.

## Out of scope

Deliberately not built, so their absence is a decision rather than an oversight:

- **Calculation history**, memory buttons, keyboard shortcuts, dark-mode toggle.
  The assignment asks to prioritise correctness and clarity over extra features.
  (The UI does follow the OS light/dark preference, which is a few CSS lines.)
- **An expression parser** (`1 + 2 * 3`). Operator precedence and parsing are a
  substantial piece of work that would dominate the codebase.
- **OpenAPI and generated types.** For one endpoint, codegen costs more
  toolchain than the duplication it removes.
- **CI.** The two test commands above are what CI would run.
- **A per-request precision parameter.** 16 significant digits, fixed and
  documented, until something asks for otherwise.

---

## Layout

```
backend/
  cmd/api/              server wiring: timeouts, graceful shutdown
  internal/calculator/  the arithmetic — no HTTP anywhere in this package
  internal/api/         handler, validation, error-to-status mapping
frontend/
  src/api/              operation table, Zod schemas, fetch client
  src/hooks/            useCalculator — the request lifecycle
  src/components/       the form and the result panel
docker/                 Dockerfile.api, Dockerfile.web, nginx.conf
docs/adr/               the four decisions that were hard to reverse
CONTEXT.md              domain glossary
PROMPT.md               the prompts behind the design
```

The separation that matters is `internal/calculator` knowing nothing about HTTP.
It takes decimals and returns a result or a domain error, so its tests are plain
table-driven tests with no `httptest` in sight, and the mapping from error to
status code lives in exactly one place.
