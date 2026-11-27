import { screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ScanJobListPage } from './ScanJobListPage'
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
  updated_at: '2026-11-27T08:00:00Z',
}

describe('ScanJobListPage', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
  })

  it('shows an empty state', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ data: [], page: 1, page_size: 20, total: 0 }), { status: 200 }),
      ),
    )
    renderPage('/scan-jobs', '/scan-jobs', <ScanJobListPage />)
    expect(await screen.findByRole('heading', { name: 'No scan jobs' })).toBeInTheDocument()
  })

  it('shows a job returned by the API', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ data: [job], page: 1, page_size: 20, total: 1 }), { status: 200 }),
      ),
    )
    renderPage('/scan-jobs', '/scan-jobs', <ScanJobListPage />)
    expect(await screen.findByRole('link', { name: 'Homepage' })).toHaveAttribute(
      'href',
      '/scan-jobs/11111111-1111-4111-8111-111111111111',
    )
    expect(screen.getAllByText('Pending').length).toBeGreaterThan(0)
  })

  it('shows the API error', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: { code: 'internal_error', message: 'internal server error' } }), {
          status: 500,
        }),
      ),
    )
    renderPage('/scan-jobs', '/scan-jobs', <ScanJobListPage />)
    expect(await screen.findByRole('alert')).toHaveTextContent('internal server error')
  })
})
