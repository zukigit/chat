import { useEffect, useState } from 'react'
import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { getToken, removeToken } from '../auth'
import { hasPrivateKey } from '../crypto'

export default function PrivateRoute() {
  const token = getToken()
  if (!token) {
    removeToken()
    return <Navigate to="/login" replace />
  }

  return <PrivateRouteGuard />
}

function PrivateRouteGuard() {
  const [unlocked, setUnlocked] = useState<boolean | null>(null)
  const location = useLocation()

  const e2eePaths = ['/key-setup', '/pin-entry']
  const isE2eePath = e2eePaths.includes(location.pathname)

  useEffect(() => {
    if (isE2eePath) {
      setUnlocked(true)
      return
    }
    setUnlocked(hasPrivateKey())
  }, [isE2eePath])

  if (unlocked === null) {
    return (
      <div className="auth-page">
        <div className="auth-card">
          <div className="auth-header">
            <h1 className="auth-title">Loading...</h1>
          </div>
          <div className="auth-spinner" style={{ width: 32, height: 32, borderWidth: 3 }} />
        </div>
      </div>
    )
  }

  if (!unlocked && !isE2eePath) {
    return <Navigate to="/pin-entry" replace />
  }

  return <Outlet />
}
