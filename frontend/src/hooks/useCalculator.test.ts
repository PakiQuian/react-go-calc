import { act, renderHook, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { useCalculator } from './useCalculator'
import type { CalculateFn, CalculateOutcome } from '../api/client'

const request = { operation: 'add' as const, operands: ['1', '2'] }

/** A client whose resolution the test controls, for ordering assertions. */
function deferredClient() {
  const resolvers: ((outcome: CalculateOutcome) => void)[] = []
  const fn: CalculateFn = () =>
    new Promise<CalculateOutcome>((resolve) => {
      resolvers.push(resolve)
    })
  return { fn, resolvers }
}

describe('useCalculator', () => {
  it('starts idle', () => {
    const { result } = renderHook(() => useCalculator(vi.fn()))
    expect(result.current.state).toEqual({ status: 'idle' })
  })

  it('moves through loading to success', async () => {
    const { fn, resolvers } = deferredClient()
    const { result } = renderHook(() => useCalculator(fn))

    act(() => {
      void result.current.submit(request, '1 + 2')
    })
    expect(result.current.state).toEqual({ status: 'loading' })

    await act(async () => {
      resolvers[0]({ ok: true, result: '3' })
    })

    expect(result.current.state).toEqual({
      status: 'success',
      result: '3',
      expression: '1 + 2',
    })
  })

  it('records a rejection with its code', async () => {
    const calculate: CalculateFn = vi.fn().mockResolvedValue({
      ok: false,
      code: 'DIVISION_BY_ZERO',
      message: 'division by zero is undefined',
      field: 'operands[1]',
    })
    const { result } = renderHook(() => useCalculator(calculate))

    await act(async () => {
      await result.current.submit({ operation: 'divide', operands: ['1', '0'] }, '1 ÷ 0')
    })

    expect(result.current.state).toEqual({
      status: 'error',
      code: 'DIVISION_BY_ZERO',
      message: 'division by zero is undefined',
      field: 'operands[1]',
    })
  })

  it('returns to idle on reset', async () => {
    const calculate: CalculateFn = vi.fn().mockResolvedValue({ ok: true, result: '3' })
    const { result } = renderHook(() => useCalculator(calculate))

    await act(async () => {
      await result.current.submit(request, '1 + 2')
    })
    act(() => result.current.reset())

    expect(result.current.state).toEqual({ status: 'idle' })
  })

  // Without the request-id guard, a slow first response would overwrite the
  // result of a second, newer request.
  it('ignores a stale response that arrives after a newer one', async () => {
    const { fn, resolvers } = deferredClient()
    const { result } = renderHook(() => useCalculator(fn))

    act(() => {
      void result.current.submit(request, 'first')
    })
    act(() => {
      void result.current.submit(request, 'second')
    })

    await act(async () => {
      resolvers[1]({ ok: true, result: 'second result' })
      resolvers[0]({ ok: true, result: 'stale result' })
    })

    await waitFor(() => {
      expect(result.current.state).toMatchObject({
        status: 'success',
        result: 'second result',
      })
    })
  })

  it('ignores a response that arrives after a reset', async () => {
    const { fn, resolvers } = deferredClient()
    const { result } = renderHook(() => useCalculator(fn))

    act(() => {
      void result.current.submit(request, '1 + 2')
    })
    act(() => result.current.reset())

    await act(async () => {
      resolvers[0]({ ok: true, result: '3' })
    })

    expect(result.current.state).toEqual({ status: 'idle' })
  })
})
