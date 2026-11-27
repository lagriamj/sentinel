import { setAccessToken } from '../auth/token'
import { api } from './api'

export type Account = {
  id: string
  email: string
}

type SessionResponse = {
  access_token: string
  user: Account
}

export function registerAccount(email: string, password: string): Promise<Account> {
  return openSession('/api/v1/auth/register', email, password)
}

export function loginAccount(email: string, password: string): Promise<Account> {
  return openSession('/api/v1/auth/login', email, password)
}

export function fetchSession(): Promise<Account> {
  return api<Account>('/api/v1/auth/session')
}

export async function logoutAccount(): Promise<void> {
  try {
    await api<void>('/api/v1/auth/logout', { method: 'POST' })
  } finally {
    setAccessToken('')
  }
}

async function openSession(path: string, email: string, password: string): Promise<Account> {
  const session = await api<SessionResponse>(path, {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
  setAccessToken(session.access_token)
  return session.user
}
