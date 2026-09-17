import { calculateResponseSchema, errorResponseSchema } from './schemas'
import type { Operation } from './operations'

/**
 * The API client.
 *
 * Plain fetch rather than axios: the only feature axios would add here is
 * throwing on 4xx, and this API deliberately returns structured error bodies on
 * 400 and 422 that the UI needs to read. Those are data, not exceptions.
 */

/** Same origin in every environment — nginx proxies /api in production and
 * Vite proxies it in development, so there is no base URL to configure. */
const ENDPOINT = '/api/v1/calculate'

const REQUEST_TIMEOUT_MS = 10_000

export interface CalculateRequest {
  operation: Operation
  /** Sent as strings so exact decimals are never routed through a JS number. */
  operands: string[]
}

export type CalculateOutcome =
  | { ok: true; result: string }
  | { ok: false; code: string; message: string; field?: string }

export type CalculateFn = (request: CalculateRequest) => Promise<CalculateOutcome>

export const calculate: CalculateFn = async (request) => {
  let response: Response
  try {
    response = await fetch(ENDPOINT, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
      signal: AbortSignal.timeout(REQUEST_TIMEOUT_MS),
    })
  } catch {
    return {
      ok: false,
      code: 'NETWORK_ERROR',
      message: 'Could not reach the calculator service. Check that it is running.',
    }
  }

  let body: unknown
  try {
    body = await response.json()
  } catch {
    return {
      ok: false,
      code: 'INVALID_RESPONSE',
      message: `The server returned an unreadable response (HTTP ${response.status}).`,
    }
  }

  if (!response.ok) {
    const parsed = errorResponseSchema.safeParse(body)
    if (!parsed.success) {
      return {
        ok: false,
        code: 'INVALID_RESPONSE',
        message: `The server rejected the request but its response could not be read (HTTP ${response.status}).`,
      }
    }
    return { ok: false, ...parsed.data.error }
  }

  // Parsing the success response too: a contract mismatch should be one clear
  // error here, not `undefined` surfacing inside a component later.
  const parsed = calculateResponseSchema.safeParse(body)
  if (!parsed.success) {
    return {
      ok: false,
      code: 'INVALID_RESPONSE',
      message: 'The server response did not match the expected format.',
    }
  }

  return { ok: true, result: parsed.data.result }
}
