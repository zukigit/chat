import { Navigate, useLocation } from "react-router-dom";
import { authService } from "../services/authService";

interface ProtectedRouteProps {
  children: React.ReactNode;
}

/**
 * Route guard that redirects unauthenticated users to login.
 * Preserves the attempted URL so users can be redirected back after login.
 */
export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const location = useLocation();

  if (!authService.isAuthenticated()) {
    // Redirect to login, preserving the attempted URL
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  return <>{children}</>;
}
