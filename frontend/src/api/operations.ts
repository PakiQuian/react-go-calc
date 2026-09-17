/**
 * The operation table, mirroring the backend's dispatch map.
 *
 * The contract is defined twice rather than generated, which is the right
 * trade at this size: codegen for one endpoint costs more toolchain than it
 * saves. The arity here is what drives the form — the second operand field
 * disappears for unary operations, so the UI cannot construct a request the
 * API would reject.
 */

export const OPERATIONS = [
  'add',
  'subtract',
  'multiply',
  'divide',
  'power',
  'percentage',
  'sqrt',
] as const

export type Operation = (typeof OPERATIONS)[number]

export interface OperationSpec {
  /** Shown in the operation selector. */
  label: string
  /** Number of operands the API requires. */
  arity: 1 | 2
  /** Field labels, one per operand. */
  operandLabels: string[]
  /** Rendered between the operands in the result summary. */
  describe: (operands: string[]) => string
}

export const OPERATION_SPECS: Record<Operation, OperationSpec> = {
  add: {
    label: 'Add',
    arity: 2,
    operandLabels: ['First number', 'Second number'],
    describe: ([a, b]) => `${a} + ${b}`,
  },
  subtract: {
    label: 'Subtract',
    arity: 2,
    operandLabels: ['First number', 'Second number'],
    describe: ([a, b]) => `${a} − ${b}`,
  },
  multiply: {
    label: 'Multiply',
    arity: 2,
    operandLabels: ['First number', 'Second number'],
    describe: ([a, b]) => `${a} × ${b}`,
  },
  divide: {
    label: 'Divide',
    arity: 2,
    operandLabels: ['Dividend', 'Divisor'],
    describe: ([a, b]) => `${a} ÷ ${b}`,
  },
  power: {
    label: 'Power',
    arity: 2,
    operandLabels: ['Base', 'Exponent (whole number)'],
    describe: ([a, b]) => `${a} ^ ${b}`,
  },
  // percentage(a, b) answers "what percentage a is of b". The other common
  // reading, "a percent of b", is a different function — see CONTEXT.md.
  percentage: {
    label: 'Percentage',
    arity: 2,
    operandLabels: ['Part', 'Total'],
    describe: ([a, b]) => `${a} as a percentage of ${b}`,
  },
  sqrt: {
    label: 'Square root',
    arity: 1,
    operandLabels: ['Number'],
    describe: ([a]) => `√${a}`,
  },
}

export function isOperation(value: string): value is Operation {
  return (OPERATIONS as readonly string[]).includes(value)
}
