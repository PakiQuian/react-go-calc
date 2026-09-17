import { useEffect, useState } from 'react'

import type { CalculatorState } from '../hooks/useCalculator'
import styles from './ResultPanel.module.css'

interface Props {
  state: CalculatorState
}

export function ResultPanel({ state }: Props) {
  if (state.status === 'error') {
    return (
      <div className={styles.panel}>
        <p className={styles.error} role="alert">
          {state.message}
        </p>
        <p className={styles.code}>{state.code}</p>
      </div>
    )
  }

  if (state.status !== 'success') {
    return (
      <div className={styles.panel}>
        <p className={styles.placeholder}>Enter values and press Calculate.</p>
      </div>
    )
  }

  return (
    <div className={styles.panel}>
      <p className={styles.expression}>{state.expression}</p>
      {/* Results are shown in full. A square root can run to twenty digits and
          truncating it would defeat the exact-decimal arithmetic behind it. */}
      <output className={styles.result} aria-live="polite">
        {state.result}
      </output>
      <CopyButton value={state.result} />
    </div>
  )
}

function CopyButton({ value }: { value: string }) {
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!copied) return
    const timer = setTimeout(() => setCopied(false), 2000)
    return () => clearTimeout(timer)
  }, [copied])

  // Copies the complete value regardless of how it is displayed.
  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
    } catch {
      setCopied(false)
    }
  }

  return (
    <button type="button" className={styles.copy} onClick={handleCopy}>
      {copied ? 'Copied' : 'Copy result'}
    </button>
  )
}
