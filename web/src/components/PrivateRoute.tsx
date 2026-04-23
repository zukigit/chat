import { Navigate, Outlet } from 'react-router-dom'
import { getToken, removeToken } from '../auth'

export default function PrivateRoute() {
  const token = getToken()
  if (!token) {
    removeToken()
    return <Navigate to="/login" replace />
  }
  return <Outlet />
}
