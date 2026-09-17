import { describe, expect, it } from 'vitest'

import { OPERATIONS, OPERATION_SPECS, isOperation } from './operations'

describe('the operation table', () => {
  // This table is the frontend half of a contract defined twice. If it drifts
  // from the backend's dispatch map, the UI starts building requests the API
  // rejects — so its internal consistency is asserted here, and the backend
  // remains the authority at runtime.
  it('describes every operation', () => {
    expect(Object.keys(OPERATION_SPECS).sort()).toEqual([...OPERATIONS].sort())
  })

  it.each(OPERATIONS)('%s has labels matching its arity', (operation) => {
    const spec = OPERATION_SPECS[operation]
    expect(spec.operandLabels).toHaveLength(spec.arity)
    expect(spec.label).not.toBe('')
  })

  it.each(OPERATIONS)('%s describes a calculation in words', (operation) => {
    const spec = OPERATION_SPECS[operation]
    const operands = Array.from({ length: spec.arity }, (_, i) => String(i + 1))

    const described = spec.describe(operands)
    expect(described).toContain('1')
    for (const operand of operands) {
      expect(described).toContain(operand)
    }
  })

  it('matches the arity the backend enforces', () => {
    expect(OPERATION_SPECS.sqrt.arity).toBe(1)
    for (const operation of OPERATIONS.filter((name) => name !== 'sqrt')) {
      expect(OPERATION_SPECS[operation].arity).toBe(2)
    }
  })

  // percentage(a, b) is "what percentage a is of b" — the labels have to say
  // which operand is the total, because the other reading is equally common.
  it('labels percentage operands unambiguously', () => {
    expect(OPERATION_SPECS.percentage.operandLabels).toEqual(['Part', 'Total'])
  })

  it('recognises known operation names', () => {
    expect(isOperation('add')).toBe(true)
    expect(isOperation('sqrt')).toBe(true)
    expect(isOperation('modulo')).toBe(false)
    expect(isOperation('')).toBe(false)
  })
})
