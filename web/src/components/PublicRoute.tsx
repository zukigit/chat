import { Navigate, useLocation } from "react-router-dom";
import { authService } from "../services/authService";

interface PublicRouteProps {
  children: React.ReactNode;
}

/**
 * Route guard for public pages (login/signup).
 * Redirects authenticated users to the home/dashboard page.
 */
export function PublicRoute({ children }: PublicRouteProps) {
  const location = useLocation();

  if (authService.isAuthenticated()) {
    // Redirect to the page they came from, or home by default
    const from = (location.state as { from?: Location })?.from?.pathname || "/";
    return <Navigate to={from} replace />;
  }

  return <>{children}</>;
}
