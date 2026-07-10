import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";

interface Props {
  adminOnly?: boolean;
}

export function ProtectedRoute({ adminOnly = false }: Props) {
  const { user, token } = useAuth();

  if (!token || !user) return <Navigate to="/login" replace />;
  if (adminOnly && !user.is_admin) return <Navigate to="/offers" replace />;

  return <Outlet />;
}
