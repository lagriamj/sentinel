import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ScanJobDetailPage } from './ScanJobDetailPage'
import { renderPage } from '../../test/render'
import type { ScanJob } from '../../types/scan-job'

const job: ScanJob = {
  id: '11111111-1111-4111-8111-111111111111',
  name: 'Homepage',
  status: 'pending',
  target: {
    id: '22222222-2222-4222-8222-222222222222',
    url: 'https://example.com',
    scheme: 'https',
    host: 'example.com',
    port: 443,
    authorized: true,
  },
  created_at: '2026-11-27T08:00:00Z',
  updated_at: '2026-11-27T08:05:00Z',
}

function renderDetail() {
  renderPage(`/scan-jobs/${job.id}`, '/scan-jobs/:id', <ScanJobDetailPage />)
}

describe('ScanJobDetailPage', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
  })

  it('shows the target and both actions for a pending job', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(job), { status: 200 })))
    renderDetail()
    expect(await screen.findByRole('heading', { name: 'Homepage' })).toBeInTheDocument()
    expect(screen.getByText('example.com:443')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Cancel job' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete job' })).toBeInTheDocument()
  })

  it('asks for confirmation before cancelling', async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify(job), { status: 200 })))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    renderDetail()
    await user.click(await screen.findByRole('button', { name: 'Cancel job' }))
    await user.click(screen.getByRole('button', { name: 'Confirm cancel' }))
    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        `/api/v1/scan-jobs/${job.id}/cancel`,
        expect.objectContaining({ method: 'POST' }),
      )
    })
  })

  it('hides cancel for a completed job', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ ...job, status: 'completed', finished_at: '2026-11-27T09:00:00Z' }), {
          status: 200,
        }),
      ),
    )
    renderDetail()
    expect(await screen.findByRole('heading', { name: 'Homepage' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Cancel job' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete job' })).toBeInTheDocument()
  })

  it('shows a missing job', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: { code: 'not_found', message: 'scan job not found' } }), { status: 404 }),
      ),
    )
    renderDetail()
    expect(await screen.findByRole('alert')).toHaveTextContent('scan job not found')
  })
})
