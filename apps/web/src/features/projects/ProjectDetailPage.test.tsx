import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ProjectDetailPage } from './ProjectDetailPage'
import { renderPage } from '../../test/render'

const projectID = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'

describe('ProjectDetailPage', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
  })

  it('requires authorization before registering a target', async () => {
    const fetchMock = vi.fn().mockImplementation((input: RequestInfo) => {
      const url = String(input)
      if (url.endsWith('/members')) {
        return Promise.resolve(new Response(JSON.stringify({ data: [] }), { status: 200 }))
      }
      if (url.includes('/targets')) {
        return Promise.resolve(new Response(JSON.stringify({ data: [], page: 1, page_size: 100, total: 0 }), { status: 200 }))
      }
      return Promise.resolve(
        new Response(
          JSON.stringify({
            id: projectID,
            name: 'Web',
            description: '',
            role: 'owner',
            created_at: '2026-11-27T08:00:00Z',
            updated_at: '2026-11-27T08:00:00Z',
          }),
          { status: 200 },
        ),
      )
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    renderPage(`/projects/${projectID}`, '/projects/:id', <ProjectDetailPage />)

    await user.type(await screen.findByLabelText('Target URL'), 'https://example.com')
    await user.click(screen.getByRole('button', { name: 'Register target' }))

    expect(screen.getByRole('alert')).toHaveTextContent('Authorization confirmation is required.')
    expect(fetchMock).not.toHaveBeenCalledWith(`/api/v1/projects/${projectID}/targets`, expect.objectContaining({ method: 'POST' }))
  })
})
