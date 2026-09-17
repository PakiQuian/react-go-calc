import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { calculate } from './client'

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('calculate', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  const fetchMock = () => vi.mocked(fetch)

  it('posts operands as strings to the relative API path', async () => {
    fetchMock().mockResolvedValue(
      jsonResponse({ operation: 'add', operands: ['0.1', '0.2'], result: '0.3' }),
    )

    const outcome = await calculate({ operation: 'add', operands: ['0.1', '0.2'] })

    expect(outcome).toEqual({ ok: true, result: '0.3' })

    const [url, init] = fetchMock().mock.calls[0]
    // Relative, so the same code works behind Vite's proxy and behind nginx.
    expect(url).toBe('/api/v1/calculate')
    expect(init?.method).toBe('POST')
    expect(JSON.parse(String(init?.body))).toEqual({
      operation: 'add',
      operands: ['0.1', '0.2'],
    })
  })

  it('returns the structured error body for a rejected request', async () => {
    fetchMock().mockResolvedValue(
      jsonResponse(
        {
          error: {
            code: 'DIVISION_BY_ZERO',
            message: 'division by zero is undefined',
            field: 'operands[1]',
          },
        },
        422,
      ),
    )

    const outcome = await calculate({ operation: 'divide', operands: ['1', '0'] })

    expect(outcome).toEqual({
      ok: false,
      code: 'DIVISION_BY_ZERO',
      message: 'division by zero is undefined',
      field: 'operands[1]',
    })
  })

  it('reports a network failure without throwing', async () => {
    fetchMock().mockRejectedValue(new TypeError('Failed to fetch'))

    const outcome = await calculate({ operation: 'add', operands: ['1', '2'] })

    expect(outcome).toMatchObject({ ok: false, code: 'NETWORK_ERROR' })
  })

  it('reports a response that is not JSON', async () => {
    fetchMock().mockResolvedValue(new Response('<html>502</html>', { status: 502 }))

    const outcome = await calculate({ operation: 'add', operands: ['1', '2'] })

    expect(outcome).toMatchObject({ ok: false, code: 'INVALID_RESPONSE' })
  })

  // Parsing the success response is the job Zod does that the backend cannot:
  // a contract mismatch becomes one clear error instead of `undefined` turning
  // up inside a component.
  it('reports a success response that does not match the contract', async () => {
    fetchMock().mockResolvedValue(
      jsonResponse({ operation: 'add', operands: ['1', '2'], result: 3 }),
    )

    const outcome = await calculate({ operation: 'add', operands: ['1', '2'] })

    expect(outcome).toMatchObject({ ok: false, code: 'INVALID_RESPONSE' })
  })

  it('reports an error response that does not match the contract', async () => {
    fetchMock().mockResolvedValue(jsonResponse({ oops: true }, 400))

    const outcome = await calculate({ operation: 'add', operands: ['1', '2'] })

    expect(outcome).toMatchObject({ ok: false, code: 'INVALID_RESPONSE' })
  })
})
