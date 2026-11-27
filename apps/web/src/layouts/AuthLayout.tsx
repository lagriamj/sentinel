import { Outlet } from 'react-router-dom'

export function AuthLayout() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4 py-16">
      <div className="w-full max-w-sm">
        <div className="mb-8 flex items-center gap-3">
          <span className="flex size-9 items-center justify-center rounded-lg border bg-card text-sm font-medium">S</span>
          <div>
            <p className="text-sm font-medium leading-none">Sentinel</p>
            <p className="mt-1 text-xs text-muted-foreground">Scanner</p>
          </div>
        </div>
        <Outlet />
      </div>
    </div>
  )
}
