import { z } from 'zod'

/**
 * Zod is used for two distinct jobs.
 *
 * Validating form input is a convenience: it gives instant feedback without a
 * round trip. It is not a security boundary — the backend validates everything
 * again, because anyone can call the API directly.
 *
 * Parsing the response is the job with no backend equivalent. It makes a
 * contract mismatch surface as one clear error instead of `undefined` appearing
 * somewhere deep in a component.
 */

/** Limits mirroring the backend. See backend/internal/calculator/bounds.go. */
export const MAX_SIGNIFICANT_DIGITS = 100
export const MAX_MAGNITUDE = 1000
export const MAX_POWER_EXPONENT = 1000

const NUMBER_PATTERN = /^[+-]?(\d+(\.\d*)?|\.\d+)([eE][+-]?\d+)?$/

export interface NumberShape {
  /**
   * Digit count, matching how the backend counts a decimal coefficient:
   * trailing zeros are included, so "1.20" has three.
   */
  digits: number
  /** Power of ten of the leading digit: 1 for 12.5, -3 for 0.004. */
  magnitude: number
  /** True when no significant digit falls below the decimal point. */
  isWhole: boolean
}

/**
 * Describes a number without converting it to a JS number, which would lose
 * precision on exactly the values this application exists to handle.
 */
export function describeNumber(input: string): NumberShape {
  const [mantissa, exponentPart] = input.replace(/^[+-]/, '').split(/[eE]/)
  const [whole = '', fraction = ''] = mantissa.split('.')

  const allDigits = `${whole}${fraction}`
  const withoutLeadingZeros = allDigits.replace(/^0+/, '')

  if (withoutLeadingZeros === '') {
    return { digits: 0, magnitude: 0, isWhole: true } // the value is zero
  }

  // Leading zeros in "0.004" push the first significant digit to the right.
  const leadingZeros = allDigits.length - withoutLeadingZeros.length
  const exponent = exponentPart ? Number(exponentPart) : 0
  const magnitude = whole.length - 1 - leadingZeros + exponent

  // Trailing zeros count toward the digit bound but not toward wholeness:
  // "2.0" is an integer, "1.20" is not.
  let significant = withoutLeadingZeros.length
  while (significant > 0 && withoutLeadingZeros[significant - 1] === '0') significant--

  return {
    digits: withoutLeadingZeros.length,
    magnitude,
    isWhole: magnitude >= significant - 1,
  }
}

export const operandSchema = z
  .string()
  .trim()
  .min(1, 'Enter a number')
  .regex(NUMBER_PATTERN, 'Enter a valid number, for example 12.5 or 1.2e10')
  .refine(
    (value) => describeNumber(value).digits <= MAX_SIGNIFICANT_DIGITS,
    `Use at most ${MAX_SIGNIFICANT_DIGITS} significant digits`,
  )
  .refine((value) => {
    const { digits, magnitude } = describeNumber(value)
    return digits === 0 || Math.abs(magnitude) <= MAX_MAGNITUDE
  }, `Number must be between 1e-${MAX_MAGNITUDE} and 1e${MAX_MAGNITUDE}`)

/**
 * The exponent of `power`: a whole number, and additionally bounded by value
 * rather than by magnitude. The backend caps it at MaxPowerExponent, which is
 * a tighter limit than MAX_MAGNITUDE — without mirroring it here, the form
 * would happily submit power(2, 5000) for the API to reject.
 */
export const wholeNumberSchema = operandSchema
  .refine((value) => describeNumber(value).isWhole, 'Exponent must be a whole number')
  .refine(
    // Safe to use Number here: anything that survives the bounds above and is
    // still too large lands on Infinity, which fails the comparison.
    (value) => Math.abs(Number(value)) <= MAX_POWER_EXPONENT,
    `Exponent must be between -${MAX_POWER_EXPONENT} and ${MAX_POWER_EXPONENT}`,
  )

export const calculateResponseSchema = z.object({
  operation: z.string(),
  operands: z.array(z.string()),
  result: z.string(),
})

export const errorResponseSchema = z.object({
  error: z.object({
    code: z.string(),
    message: z.string(),
    field: z.string().optional(),
  }),
})

export type CalculateResponse = z.infer<typeof calculateResponseSchema>
export type ErrorResponse = z.infer<typeof errorResponseSchema>

/** Returns the first validation message, or null when the value is acceptable. */
export function validateOperand(value: string, mustBeWhole = false): string | null {
  const schema = mustBeWhole ? wholeNumberSchema : operandSchema
  const parsed = schema.safeParse(value)
  return parsed.success ? null : (parsed.error.issues[0]?.message ?? 'Invalid number')
}
