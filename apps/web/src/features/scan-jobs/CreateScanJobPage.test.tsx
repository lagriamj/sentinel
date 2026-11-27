import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { CreateScanJobPage } from './CreateScanJobPage'
import { renderPage } from '../../test/render'

const project = {
  id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
  name: 'Web',
  description: '',
  role: 'owner',
  created_at: '2026-11-27T08:00:00Z',
  updated_at: '2026-11-27T08:00:00Z',
}

const target = {
  id: '22222222-2222-4222-8222-222222222222',
  project_id: project.id,
  url: 'https://example.com',
  scheme: 'https',
  host: 'example.com',
  port: 443,
  authorized: true,
  created_at: '2026-11-27T08:00:00Z',
  updated_at: '2026-11-27T08:00:00Z',
}

describe('CreateScanJobPage', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
  })

  it('asks for a project before calling the API', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: [], page: 1, page_size: 20, total: 0 }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    renderPage('/scan-jobs/new', '/scan-jobs/new', <CreateScanJobPage />)

    expect(await screen.findByRole('heading', { name: 'No projects' })).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalledWith('/api/v1/scan-jobs', expect.anything())
  })

  it('posts a job for a registered target', async () => {
    const fetchMock = vi.fn().mockImplementation((input: RequestInfo) => {
      const url = String(input)
      if (url.includes('/targets')) {
        return Promise.resolve(new Response(JSON.stringify({ data: [target], page: 1, page_size: 100, total: 1 }), { status: 200 }))
      }
      if (url.includes('/projects')) {
        return Promise.resolve(new Response(JSON.stringify({ data: [project], page: 1, page_size: 20, total: 1 }), { status: 200 }))
      }
      return Promise.resolve(
        new Response(
          JSON.stringify({
            id: '11111111-1111-4111-8111-111111111111',
            name: 'Homepage',
            status: 'pending',
            target,
            created_at: '2026-11-27T08:00:00Z',
            updated_at: '2026-11-27T08:00:00Z',
          }),
          { status: 201 },
        ),
      )
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    renderPage('/scan-jobs/new', '/scan-jobs/new', <CreateScanJobPage />)

    await user.type(await screen.findByLabelText('Name'), ' Homepage ')
    await user.click(screen.getByRole('button', { name: 'Save job' }))

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/scan-jobs',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          name: 'Homepage',
          target_id: target.id,
        }),
      }),
    )
  })
})
