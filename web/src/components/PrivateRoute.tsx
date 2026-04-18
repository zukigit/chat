import { Navigate, Outlet } from 'react-router-dom'
import { getToken } from '../auth'

export default function PrivateRoute() {
  return getToken() ? <Outlet /> : <Navigate to="/login" replace />
}
