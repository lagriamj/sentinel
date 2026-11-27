import { describe, expect, it, vi } from 'vitest'
import { ApiError, api } from './api'

describe('api', () => {
  it('returns parsed JSON for a successful response', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ status: 'ok' }), { status: 200 })),
    )

    await expect(api<{ status: string }>('/api/v1/health')).resolves.toEqual({ status: 'ok' })
    vi.unstubAllGlobals()
  })

  it('returns undefined for an empty success response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 204 })))

    await expect(api<void>('/api/v1/scan-jobs/1', { method: 'DELETE' })).resolves.toBeUndefined()
    vi.unstubAllGlobals()
  })

  it('uses the API error envelope', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: { code: 'validation_error', message: 'name is required' } }), {
          status: 400,
        }),
      ),
    )

    await expect(api('/api/v1/scan-jobs', { method: 'POST', body: '{}' })).rejects.toMatchObject({
      name: 'ApiError',
      status: 400,
      code: 'validation_error',
      message: 'name is required',
    } satisfies Partial<ApiError>)
    vi.unstubAllGlobals()
  })

  it('hides a body that is not the error envelope', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('password=secret', { status: 500 })))

    await expect(api('/api/v1/health')).rejects.toMatchObject({
      message: 'Request failed',
      status: 500,
    })
    vi.unstubAllGlobals()
  })
})
