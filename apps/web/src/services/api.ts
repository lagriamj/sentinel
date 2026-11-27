import { getAccessToken, setAccessToken } from '../auth/token'

export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

type ErrorEnvelope = {
  error?: {
    code?: string
    message?: string
  }
}

type SessionResponse = {
  access_token: string
}

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  return request(path, init, true)
}

export async function refreshAccessToken(): Promise<boolean> {
  const response = await fetch('/api/v1/auth/refresh', {
    method: 'POST',
    credentials: 'include',
    headers: { Accept: 'application/json' },
  })
  if (!response.ok) {
    setAccessToken('')
    return false
  }
  const payload = (await response.json()) as SessionResponse
  if (!payload.access_token) {
    setAccessToken('')
    return false
  }
  setAccessToken(payload.access_token)
  return true
}

async function request<T>(path: string, init: RequestInit | undefined, allowRefresh: boolean): Promise<T> {
  const headers = new Headers(init?.headers)
  headers.set('Accept', 'application/json')
  if (init?.body !== undefined && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  const token = getAccessToken()
  if (token !== '' && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const response = await fetch(path, { ...init, headers, credentials: 'include' })
  if (response.status === 401 && allowRefresh && !path.startsWith('/api/v1/auth/')) {
    const refreshed = await refreshAccessToken()
    if (refreshed) {
      return request(path, init, false)
    }
  }
  if (response.status === 204) {
    return undefined as T
  }

  const payload: unknown = await readJson(response)
  if (!response.ok) {
    throw toApiError(response.status, payload)
  }
  return payload as T
}

async function readJson(response: Response): Promise<unknown> {
  const text = await response.text()
  if (text === '') {
    return null
  }
  try {
    return JSON.parse(text) as unknown
  } catch {
    return null
  }
}

function toApiError(status: number, payload: unknown): ApiError {
  if (isErrorEnvelope(payload)) {
    return new ApiError(
      status,
      payload.error?.code || 'request_failed',
      payload.error?.message || 'Request failed',
    )
  }
  return new ApiError(status, 'request_failed', 'Request failed')
}

function isErrorEnvelope(payload: unknown): payload is ErrorEnvelope {
  return typeof payload === 'object' && payload !== null && 'error' in payload
}
