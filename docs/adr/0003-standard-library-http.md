---
status: accepted
---

# `net/http` from the standard library, with no router or web framework

The HTTP layer uses `net/http` directly. No `gin`, `echo`, `chi` or `fiber`, and
no validation library such as `go-playground/validator`.

## Considered options

**A router (`chi`, `gin`).** Rejected for this service's size. Go 1.22 added
method and path matching to the standard `ServeMux`, so
`mux.HandleFunc("POST /api/v1/calculate", h)` and path wildcards now come from
the standard library. What a router still adds beyond that is route groups,
struct-tag request binding and radix-tree matching — all of which pay off at
dozens or hundreds of routes. This service has two: one calculation endpoint and
one health check. Middleware is a `func(http.Handler) http.Handler`, and
composing the two we need takes about five lines.

**A validation library.** Rejected because the central rule does not fit the
model. The required operand count depends on *which operation the request
names*, and struct tags describe fields in isolation; expressing "this field's
length depends on that field's value" means writing a custom validator and
wrapping it in the library's ceremony. Plain Go is shorter and states the rule
directly.

## Consequences

**No dependency needs justifying in review.** The only third-party package in the
backend is `shopspring/decimal`, which exists for a reason recorded in ADR 0001.

**Adding many more routes would change this calculus,** and that is the signal to
revisit. The decision is about scale, not about frameworks being undesirable.
