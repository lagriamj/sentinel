import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { StatusBadge } from './StatusBadge'

describe('StatusBadge', () => {
  it('shows the status name', () => {
    render(<StatusBadge status="running" />)
    expect(screen.getByText('Running')).toBeInTheDocument()
  })
})
