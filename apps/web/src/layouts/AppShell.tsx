import { useQuery } from '@tanstack/react-query'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { useSession } from '../features/auth/session'
import { fetchHealth } from '../services/scan-jobs'

const linkClass = ({ isActive }: { isActive: boolean }) =>
  [
    'block rounded-md px-3 py-2 text-sm',
    isActive
      ? 'bg-sidebar-accent text-sidebar-accent-foreground'
      : 'text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
  ].join(' ')

export function AppShell() {
  const navigate = useNavigate()
  const { account, signOut } = useSession()
  const health = useQuery({
    queryKey: ['health'],
    queryFn: fetchHealth,
    refetchInterval: 30_000,
  })

  const healthLabel = health.isPending ? 'Checking API' : health.isError ? 'API unavailable' : 'API connected'

  async function onSignOut() {
    await signOut()
    void navigate('/login')
  }

  return (
    <div className="min-h-screen bg-background md:grid md:grid-cols-[240px_1fr]">
      <aside className="border-b border-sidebar-border bg-sidebar text-sidebar-foreground md:border-b-0 md:border-r">
        <div className="flex items-center justify-between gap-4 px-4 py-4 md:block">
          <div>
            <p className="text-xs font-medium tracking-wide text-muted-foreground">Sentinel</p>
            <p className="text-sm">Scanner</p>
          </div>
          <nav className="flex gap-2 md:mt-6 md:block md:space-y-1">
            {account ? (
              <>
                <NavLink to="/projects" end className={linkClass}>
                  Projects
                </NavLink>
                <NavLink to="/scan-jobs" end className={linkClass}>
                  Scan jobs
                </NavLink>
              </>
            ) : (
              <NavLink to="/login" className={linkClass}>
                Sign in
              </NavLink>
            )}
          </nav>
        </div>
        <div className="space-y-3 px-4 pb-4">
          {account ? (
            <div className="space-y-2">
              <p className="truncate text-xs text-muted-foreground">{account.email}</p>
              <Button type="button" variant="outline" size="sm" onClick={() => void onSignOut()}>
                Sign out
              </Button>
            </div>
          ) : null}
          <p className="text-xs text-muted-foreground">{healthLabel}</p>
        </div>
      </aside>
      <div className="min-w-0">
        <div className="border-b px-4 py-2 text-xs text-muted-foreground md:px-8">
          Local operator console. Saved jobs remain pending.
        </div>
        <main className="px-4 py-6 md:px-8">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
