import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { refreshAccessToken } from '../../services/api'
import { fetchSession, logoutAccount, type Account } from '../../services/auth'

type SessionValue = {
  ready: boolean
  account: Account | null
  setAccount: (account: Account | null) => void
  signOut: () => Promise<void>
}

const SessionContext = createContext<SessionValue | null>(null)

export function SessionProvider({ children }: { children: ReactNode }) {
  const [account, setAccount] = useState<Account | null>(null)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    let active = true
    refreshAccessToken()
      .then((refreshed) => (refreshed ? fetchSession() : null))
      .then((next) => {
        if (active) {
          setAccount(next)
        }
      })
      .catch(() => {
        if (active) {
          setAccount(null)
        }
      })
      .finally(() => {
        if (active) {
          setReady(true)
        }
      })
    return () => {
      active = false
    }
  }, [])

  async function signOut() {
    await logoutAccount()
    setAccount(null)
  }

  return <SessionContext.Provider value={{ ready, account, setAccount, signOut }}>{children}</SessionContext.Provider>
}

export function useSession(): SessionValue {
  const value = useContext(SessionContext)
  if (!value) {
    throw new Error('session is unavailable')
  }
  return value
}
