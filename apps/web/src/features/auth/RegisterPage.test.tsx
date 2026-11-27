import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { RegisterPage } from './RegisterPage'
import { SessionProvider } from './session'
import { renderPage } from '../../test/render'

describe('RegisterPage', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
  })

  it('asks for a longer password before calling the API', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 401 }))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    renderPage(
      '/register',
      '/register',
      <SessionProvider>
        <RegisterPage />
      </SessionProvider>,
    )

    await user.type(await screen.findByLabelText('Email'), 'ada@example.com')
    await user.type(screen.getByLabelText('Password'), 'short')
    await user.click(screen.getByRole('button', { name: 'Create account' }))

    expect(screen.getByRole('alert')).toHaveTextContent('Password must be 12 characters or more.')
    expect(fetchMock).not.toHaveBeenCalledWith('/api/v1/auth/register', expect.anything())
  })
})
