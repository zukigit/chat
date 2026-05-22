import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { getToken, removeToken } from '../auth'
import { hasPrivateKey } from '../crypto'

const E2EE_PATHS = ['/key-setup', '/pin-entry']

export default function PrivateRoute() {
  const location = useLocation()
  const token = getToken()

  if (!token) {
    removeToken()
    return <Navigate to="/login" replace />
  }

  const isE2eePath = E2EE_PATHS.includes(location.pathname)

  if (!isE2eePath && !hasPrivateKey()) {
    return <Navigate to="/pin-entry" replace />
  }

  return <Outlet />
}
