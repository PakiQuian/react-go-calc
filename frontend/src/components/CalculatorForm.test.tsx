import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { CalculatorForm } from './CalculatorForm'
import type { CalculateFn } from '../api/client'

function renderForm(calculate: CalculateFn = vi.fn()) {
  const user = userEvent.setup()
  render(<CalculatorForm calculate={calculate} />)
  return { user, calculate }
}

describe('CalculatorForm', () => {
  it('sends the operation and operands as strings', async () => {
    const calculate: CalculateFn = vi.fn().mockResolvedValue({ ok: true, result: '0.3' })
    const { user } = renderForm(calculate)

    await user.type(screen.getByLabelText('First number'), '0.1')
    await user.type(screen.getByLabelText('Second number'), '0.2')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))

    await waitFor(() => {
      expect(calculate).toHaveBeenCalledWith({
        operation: 'add',
        operands: ['0.1', '0.2'],
      })
    })
    expect(await screen.findByText('0.3')).toBeInTheDocument()
  })

  // The UI must not be able to build a request the API would reject, so the
  // second field is driven by the same arity the backend enforces.
  it('shows one operand field for a unary operation', async () => {
    const { user } = renderForm()

    expect(screen.getByLabelText('Second number')).toBeInTheDocument()

    await user.selectOptions(screen.getByLabelText('Operation'), 'sqrt')

    expect(screen.getByLabelText('Number')).toBeInTheDocument()
    expect(screen.queryByLabelText('Second number')).not.toBeInTheDocument()
  })

  it('relabels the fields for percentage', async () => {
    const { user } = renderForm()

    await user.selectOptions(screen.getByLabelText('Operation'), 'percentage')

    expect(screen.getByLabelText('Part')).toBeInTheDocument()
    expect(screen.getByLabelText('Total')).toBeInTheDocument()
  })

  it('rejects invalid input without calling the API', async () => {
    const calculate: CalculateFn = vi.fn()
    const { user } = renderForm(calculate)

    await user.type(screen.getByLabelText('First number'), 'abc')
    await user.type(screen.getByLabelText('Second number'), '2')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Enter a valid number')
    expect(calculate).not.toHaveBeenCalled()
  })

  it('requires both operands', async () => {
    const calculate: CalculateFn = vi.fn()
    const { user } = renderForm(calculate)

    await user.type(screen.getByLabelText('First number'), '1')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))

    expect(await screen.findByText('Enter a number')).toBeInTheDocument()
    expect(calculate).not.toHaveBeenCalled()
  })

  it('rejects a fractional exponent for power before sending it', async () => {
    const calculate: CalculateFn = vi.fn()
    const { user } = renderForm(calculate)

    await user.selectOptions(screen.getByLabelText('Operation'), 'power')
    await user.type(screen.getByLabelText('Base'), '2')
    await user.type(screen.getByLabelText('Exponent (whole number)'), '0.5')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))

    expect(await screen.findByText('Exponent must be a whole number')).toBeInTheDocument()
    expect(calculate).not.toHaveBeenCalled()
  })

  it('displays a rejection from the server', async () => {
    const calculate: CalculateFn = vi.fn().mockResolvedValue({
      ok: false,
      code: 'DIVISION_BY_ZERO',
      message: 'division by zero is undefined',
      field: 'operands[1]',
    })
    const { user } = renderForm(calculate)

    await user.selectOptions(screen.getByLabelText('Operation'), 'divide')
    await user.type(screen.getByLabelText('Dividend'), '1')
    await user.type(screen.getByLabelText('Divisor'), '0')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'division by zero is undefined',
    )
    expect(screen.getByText('DIVISION_BY_ZERO')).toBeInTheDocument()
  })

  // Truncating would defeat the exact-decimal arithmetic behind the result.
  it('shows a long result in full and copies all of it', async () => {
    const longResult = '1.414213562373095'
    const calculate: CalculateFn = vi
      .fn()
      .mockResolvedValue({ ok: true, result: longResult })

    const { user } = renderForm(calculate)

    // userEvent.setup installs its own clipboard stub, so spy after rendering.
    const writeText = vi
      .spyOn(navigator.clipboard, 'writeText')
      .mockResolvedValue(undefined)

    await user.selectOptions(screen.getByLabelText('Operation'), 'sqrt')
    await user.type(screen.getByLabelText('Number'), '2')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))

    expect(await screen.findByText(longResult)).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Copy result' }))
    expect(writeText).toHaveBeenCalledWith(longResult)

    writeText.mockRestore()
  })

  it('clears the previous result when the operation changes', async () => {
    const calculate: CalculateFn = vi.fn().mockResolvedValue({ ok: true, result: '3' })
    const { user } = renderForm(calculate)

    await user.type(screen.getByLabelText('First number'), '1')
    await user.type(screen.getByLabelText('Second number'), '2')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))
    expect(await screen.findByText('3')).toBeInTheDocument()

    await user.selectOptions(screen.getByLabelText('Operation'), 'multiply')

    expect(screen.queryByText('3')).not.toBeInTheDocument()
    expect(screen.getByText('Enter values and press Calculate.')).toBeInTheDocument()
  })
})
