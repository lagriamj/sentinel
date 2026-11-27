import { Navigate, Outlet } from 'react-router-dom'
import { LoadingState } from '../../components/LoadingState'
import { useSession } from './session'

export function RequireSession() {
  const { ready, account } = useSession()
  if (!ready) {
    return <LoadingState label="Checking session" />
  }
  if (!account) {
    return <Navigate to="/login" replace />
  }
  return <Outlet />
}
