import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'

import { OPERATIONS, OPERATION_SPECS } from '../api/operations'
import type { Operation } from '../api/operations'
import { validateOperand } from '../api/schemas'
import { useCalculator } from '../hooks/useCalculator'
import type { CalculateFn } from '../api/client'
import { ResultPanel } from './ResultPanel'
import styles from './CalculatorForm.module.css'

interface Props {
  /** Injected in tests; defaults to the real API client. */
  calculate?: CalculateFn
}

export function CalculatorForm({ calculate }: Props) {
  const [operation, setOperation] = useState<Operation>('add')
  const [operands, setOperands] = useState<string[]>(['', ''])
  const [fieldErrors, setFieldErrors] = useState<(string | null)[]>([null, null])

  const { state, submit, reset } = useCalculator(calculate)
  const spec = OPERATION_SPECS[operation]

  // Only the fields this operation actually uses. Changing to `sqrt` hides the
  // second input rather than sending an operand the API would reject.
  const activeOperands = useMemo(
    () => operands.slice(0, spec.arity),
    [operands, spec.arity],
  )

  function handleOperationChange(next: Operation) {
    setOperation(next)
    setFieldErrors([null, null])
    reset()
  }

  function handleOperandChange(index: number, value: string) {
    setOperands((current) => current.map((v, i) => (i === index ? value : v)))
    setFieldErrors((current) => current.map((e, i) => (i === index ? null : e)))
    reset()
  }

  function handleSubmit(event: FormEvent) {
    event.preventDefault()

    // `power` is the only operation with a stricter rule than "is a number".
    const errors = activeOperands.map((value, index) =>
      validateOperand(value, operation === 'power' && index === 1),
    )
    setFieldErrors(errors)
    if (errors.some((error) => error !== null)) return

    const trimmed = activeOperands.map((value) => value.trim())
    void submit({ operation, operands: trimmed }, spec.describe(trimmed))
  }

  const isLoading = state.status === 'loading'

  return (
    <div className={styles.card}>
      <form className={styles.form} onSubmit={handleSubmit} noValidate>
        <div className={styles.field}>
          <label className={styles.label} htmlFor="operation">
            Operation
          </label>
          <select
            id="operation"
            className={styles.select}
            value={operation}
            onChange={(event) => handleOperationChange(event.target.value as Operation)}
          >
            {OPERATIONS.map((name) => (
              <option key={name} value={name}>
                {OPERATION_SPECS[name].label}
              </option>
            ))}
          </select>
        </div>

        <div className={styles.operands}>
          {spec.operandLabels.map((label, index) => {
            const inputId = `operand-${index}`
            const error = fieldErrors[index]
            const errorId = `${inputId}-error`

            return (
              <div key={inputId} className={styles.field}>
                <label className={styles.label} htmlFor={inputId}>
                  {label}
                </label>
                <input
                  id={inputId}
                  className={error ? `${styles.input} ${styles.inputError}` : styles.input}
                  value={operands[index] ?? ''}
                  onChange={(event) => handleOperandChange(index, event.target.value)}
                  inputMode="decimal"
                  autoComplete="off"
                  placeholder="0"
                  aria-invalid={error ? true : undefined}
                  aria-describedby={error ? errorId : undefined}
                />
                {error && (
                  <p id={errorId} className={styles.fieldError} role="alert">
                    {error}
                  </p>
                )}
              </div>
            )
          })}
        </div>

        <button type="submit" className={styles.submit} disabled={isLoading}>
          {isLoading ? 'Calculating…' : 'Calculate'}
        </button>
      </form>

      <ResultPanel state={state} />
    </div>
  )
}
