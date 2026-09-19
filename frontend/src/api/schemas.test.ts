import { describe, expect, it } from 'vitest'

import {
  MAX_MAGNITUDE,
  MAX_POWER_EXPONENT,
  MAX_SIGNIFICANT_DIGITS,
  calculateResponseSchema,
  describeNumber,
  errorResponseSchema,
  validateOperand,
} from './schemas'

describe('describeNumber', () => {
  // Digit counting matches the backend's, so the frontend reports the same
  // limits the API would enforce.
  it.each([
    ['12.5', { digits: 3, magnitude: 1, isWhole: false }],
    ['100', { digits: 3, magnitude: 2, isWhole: true }],
    ['0.004', { digits: 1, magnitude: -3, isWhole: false }],
    ['1.2e10', { digits: 2, magnitude: 10, isWhole: true }],
    ['-250', { digits: 3, magnitude: 2, isWhole: true }],
    ['0', { digits: 0, magnitude: 0, isWhole: true }],
    ['0.0', { digits: 0, magnitude: 0, isWhole: true }],
    // Trailing zeros count as digits but do not make a value fractional.
    ['2.0', { digits: 2, magnitude: 0, isWhole: true }],
    ['1.20', { digits: 3, magnitude: 0, isWhole: false }],
    ['1e-1000', { digits: 1, magnitude: -1000, isWhole: false }],
  ])('describes %s', (input, expected) => {
    expect(describeNumber(input)).toEqual(expected)
  })
})

describe('validateOperand', () => {
  it.each(['0', '1', '-5', '12.5', '.5', '0.004', '1.2e10', '-1.5E-3', '  7  '])(
    'accepts %s',
    (input) => {
      expect(validateOperand(input)).toBeNull()
    },
  )

  it('requires a value', () => {
    expect(validateOperand('')).toBe('Enter a number')
    expect(validateOperand('   ')).toBe('Enter a number')
  })

  it.each(['abc', '1.2.3', '1,5', '--1', '5px', '1e', 'Infinity', 'NaN'])(
    'rejects %s',
    (input) => {
      expect(validateOperand(input)).toBe(
        'Enter a valid number, for example 12.5 or 1.2e10',
      )
    },
  )

  it('rejects too many significant digits', () => {
    const tooLong = '9'.repeat(MAX_SIGNIFICANT_DIGITS + 1)
    expect(validateOperand(tooLong)).toBe(
      `Use at most ${MAX_SIGNIFICANT_DIGITS} significant digits`,
    )
    expect(validateOperand('9'.repeat(MAX_SIGNIFICANT_DIGITS))).toBeNull()
  })

  it('rejects numbers outside the accepted magnitude', () => {
    const message = `Number must be between 1e-${MAX_MAGNITUDE} and 1e${MAX_MAGNITUDE}`
    expect(validateOperand(`1e${MAX_MAGNITUDE + 1}`)).toBe(message)
    expect(validateOperand(`1e-${MAX_MAGNITUDE + 1}`)).toBe(message)
    expect(validateOperand(`1e${MAX_MAGNITUDE}`)).toBeNull()
  })

  // The input that motivated the bounds: tiny to type, expensive to compute.
  it('rejects the resource-exhaustion input', () => {
    expect(validateOperand('1e1000000')).not.toBeNull()
  })

  describe('whole numbers', () => {
    it.each(['2', '2.0', '100', '-7', '1e3', '0'])('accepts %s', (input) => {
      expect(validateOperand(input, true)).toBeNull()
    })

    it.each(['1.5', '1.20', '0.5', '1e-3'])('rejects %s', (input) => {
      expect(validateOperand(input, true)).toBe('Exponent must be a whole number')
    })
  })
})

describe('response schemas', () => {
  it('accepts a well-formed success response', () => {
    const parsed = calculateResponseSchema.safeParse({
      operation: 'add',
      operands: ['0.1', '0.2'],
      result: '0.3',
    })
    expect(parsed.success).toBe(true)
  })

  // A number here would mean the backend stopped sending exact decimals.
  it('rejects a numeric result', () => {
    const parsed = calculateResponseSchema.safeParse({
      operation: 'add',
      operands: ['0.1', '0.2'],
      result: 0.3,
    })
    expect(parsed.success).toBe(false)
  })

  it('accepts an error response without a field', () => {
    const parsed = errorResponseSchema.safeParse({
      error: { code: 'DIVISION_BY_ZERO', message: 'division by zero is undefined' },
    })
    expect(parsed.success).toBe(true)
  })

  it('rejects an error response missing a code', () => {
    const parsed = errorResponseSchema.safeParse({ error: { message: 'nope' } })
    expect(parsed.success).toBe(false)
  })
})

// The backend caps the power exponent by value at MaxPowerExponent, which is
// tighter than MAX_MAGNITUDE. Without mirroring it, the form submits requests
// the API rejects — the exact drift the duplicated contract risks.
describe('power exponent bound', () => {
  it.each(['1000', '-1000', '999', '0', '2'])('accepts %s', (input) => {
    expect(validateOperand(input, true)).toBeNull()
  })

  it.each(['1001', '-1001', '5000', '1e4', '1e400', '1e1000'])('rejects %s', (input) => {
    expect(validateOperand(input, true)).toBe(
      `Exponent must be between -${MAX_POWER_EXPONENT} and ${MAX_POWER_EXPONENT}`,
    )
  })
})

// describeNumber previously stripped trailing zeros with an unanchored /0+$/,
// which backtracks quadratically on a long interior zero run: 50k characters
// froze the tab for over two seconds.
describe('describeNumber performance', () => {
  it('handles a long interior zero run in linear time', () => {
    const pathological = '1' + '0'.repeat(50_000) + '1'

    const start = performance.now()
    const shape = describeNumber(pathological)
    const elapsed = performance.now() - start

    expect(shape.digits).toBe(50_002)
    expect(elapsed).toBeLessThan(100)
  })
})
