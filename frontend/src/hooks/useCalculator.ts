import { useCallback, useRef, useState } from 'react'

import { calculate as defaultCalculate } from '../api/client'
import type { CalculateFn, CalculateRequest } from '../api/client'

export type CalculatorState =
  | { status: 'idle' }
  | { status: 'loading' }
  | { status: 'success'; result: string; expression: string }
  | { status: 'error'; code: string; message: string; field?: string }

export interface UseCalculator {
  state: CalculatorState
  submit: (request: CalculateRequest, expression: string) => Promise<void>
  reset: () => void
}

/**
 * Owns the request lifecycle.
 *
 * The client is a parameter so tests can pass a fake, which is why no network
 * mocking library is needed: there is exactly one call to intercept, and
 * injecting it is simpler than running a service worker.
 */
export function useCalculator(calculate: CalculateFn = defaultCalculate): UseCalculator {
  const [state, setState] = useState<CalculatorState>({ status: 'idle' })

  // Guards against a slow earlier response overwriting a newer one.
  const latestRequest = useRef(0)

  const submit = useCallback(
    async (request: CalculateRequest, expression: string) => {
      const requestId = ++latestRequest.current
      setState({ status: 'loading' })

      const outcome = await calculate(request)
      if (requestId !== latestRequest.current) return

      setState(
        outcome.ok
          ? { status: 'success', result: outcome.result, expression }
          : {
              status: 'error',
              code: outcome.code,
              message: outcome.message,
              field: outcome.field,
            },
      )
    },
    [calculate],
  )

  const reset = useCallback(() => {
    // Invalidate any in-flight request so its result cannot land after a reset.
    latestRequest.current += 1
    setState({ status: 'idle' })
  }, [])

  return { state, submit, reset }
}
